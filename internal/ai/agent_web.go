package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nimbleflux/fluxbase/internal/ai/integrations"
)

// WebAgent is the specialist that answers current-info / lookup questions
// via Tavily web search. Routes through the integrations storage to
// resolve the configured Tavily credentials at turn start, then runs an
// internal tool-calling loop with two tools: web_search and fetch_url.
//
// Opt-in: only runs when (a) the chatbot has @fluxbase:web-search enabled
// and (b) a default web_search integration exists at the tenant/instance
// level. If either is missing, the supervisor silently excludes "web"
// from the route whitelist.
type WebAgent struct {
	deps *AgentDeps
}

// NewWebAgent constructs a WebAgent bound to the given deps.
func NewWebAgent(deps *AgentDeps) *WebAgent { return &WebAgent{deps: deps} }

// Name implements Node.
func (a *WebAgent) Name() string { return "web" }

// Run executes the web agent's search-then-answer loop.
//
// Resolution flow:
//  1. Resolve default web_search integration from deps.Integrations
//  2. Construct TavilyClient from integration.Config.api_key + chatbot-level defaults
//  3. Build messages with focused web-research prompt
//  4. Internal tool loop bounded by chatbot.MaxToolIterations:
//     - LLM call (non-streaming) → may emit reasoning + tool_call
//     - For each tool_call: execute via Tavily, return short summary
//     - When LLM emits no tool_call: that's the final answer
//  5. Stream final answer through Sender.SendContent
//  6. Append to state.AgentOutputs + state.FinalResponse
func (a *WebAgent) Run(ctx context.Context, state *State) error {
	chatbot := a.deps.Chatbot
	provider := a.deps.Provider
	if provider == nil {
		return fmt.Errorf("web agent: no provider")
	}
	if a.deps.Integrations == nil {
		return fmt.Errorf("web agent: no integrations storage configured")
	}

	// Resolve default web_search integration. Returns (nil, nil) when
	// none configured — caller (supervisor) shouldn't have routed to us.
	integration, err := a.deps.Integrations.GetDefaultIntegration(ctx, integrations.IntegrationTypeWebSearch)
	if err != nil {
		return fmt.Errorf("web agent: resolve integration: %w", err)
	}
	if integration == nil {
		return fmt.Errorf("web agent: no web_search integration configured")
	}

	apiKey := integration.Config["api_key"]
	if apiKey == "" {
		return fmt.Errorf("web agent: integration %q has empty api_key", integration.Name)
	}
	baseURL := integration.Config["base_url"]

	// Construct Tavily client
	client := integrations.NewTavilyClient(apiKey, baseURL, nil)

	if a.deps.Sender != nil {
		a.deps.Sender.SendAgentTransition(ctx, a.deps.ConversationID, AgentTransition{To: "web"})
	}

	// Build messages
	systemPrompt := BuildWebAgentPrompt(chatbot)
	dynamicContext := BuildDynamicContextForAgent(chatbot, a.deps.UserID, "web", a.deps.PageProfile)
	if lang := state.UserLanguage(); lang != "" {
		dynamicContext += fmt.Sprintf("\nUser's language: %s\n", lang)
	}
	if len(chatbot.WebSearchDomains) > 0 {
		dynamicContext += fmt.Sprintf("\nRestrict web search to these domains: %s\n",
			strings.Join(chatbot.WebSearchDomains, ", "))
	}

	userMsg := state.UserMessage()
	messages := []Message{
		{Role: RoleSystem, Content: systemPrompt},
		{Role: RoleSystem, Content: dynamicContext},
	}
	messages = append(messages, tailMessages(state.ConversationHistory(), 4)...)
	messages = append(messages, Message{Role: RoleUser, Content: userMsg})

	tools := []Tool{WebSearchTool, FetchURLTool}

	maxIters := chatbot.MaxToolIterations
	if maxIters <= 0 {
		maxIters = 5
	}

	model := chatbot.Model
	if override, ok := chatbot.SupervisorAgentModels["web"]; ok && override != "" {
		model = override
	}

	var finalContent string
	for iter := 0; iter < maxIters; iter++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		req := &ChatRequest{
			Messages:    messages,
			Model:       model,
			MaxTokens:   chatbot.MaxTokens,
			Temperature: chatbot.Temperature,
			Tools:       tools,
			Stream:      false,
		}

		resp, err := provider.Chat(ctx, req)
		if err != nil {
			return fmt.Errorf("web agent: provider call failed: %w", err)
		}
		if resp == nil || len(resp.Choices) == 0 {
			return fmt.Errorf("web agent: empty provider response")
		}
		if resp.Usage != nil {
			state.AddUsage(*resp.Usage)
		}

		msg := resp.Choices[0].Message

		// Emit pre-tool reasoning as a thought-process chunk.
		if msg.Content != "" && a.deps.Sender != nil {
			a.deps.Sender.SendAgentThought(ctx, a.deps.ConversationID, AgentThought{
				Agent: "web",
				Kind:  "reasoning",
				Delta: msg.Content,
			})
		}

		if len(msg.ToolCalls) == 0 {
			finalContent = msg.Content
			break
		}

		messages = append(messages, msg)
		for _, tc := range msg.ToolCalls {
			// Emit tool_call thought before executing.
			if a.deps.Sender != nil {
				a.deps.Sender.SendAgentThought(ctx, a.deps.ConversationID, AgentThought{
					Agent:    "web",
					Kind:     "tool_call",
					ToolName: tc.Function.Name,
					ToolArgs: json.RawMessage(tc.Function.Arguments),
				})
			}
			resultStr := a.executeTool(ctx, tc, client, chatbot)
			// Short tool_result summary in the thought stream.
			if a.deps.Sender != nil {
				summary := resultStr
				if len(summary) > 200 {
					summary = summary[:200] + "..."
				}
				a.deps.Sender.SendAgentThought(ctx, a.deps.ConversationID, AgentThought{
					Agent: "web",
					Kind:  "tool_result",
					Delta: summary,
				})
			}
			messages = append(messages, Message{
				Role:       RoleTool,
				Content:    resultStr,
				ToolCallID: tc.ID,
				Name:       tc.Function.Name,
			})
		}
	}

	if finalContent == "" {
		finalContent = "I wasn't able to find current information on that. Please try rephrasing or check back later."
	}

	state.AppendAgentOutput("web", finalContent)
	state.SetFinalResponse(finalContent)
	return nil
}

// executeTool dispatches one tool call to the Tavily client.
func (a *WebAgent) executeTool(ctx context.Context, tc ToolCall, client *integrations.TavilyClient, chatbot *Chatbot) string {
	switch tc.Function.Name {
	case "web_search":
		return a.executeWebSearch(ctx, tc, client, chatbot)
	case "fetch_url":
		return a.executeFetchURL(ctx, tc, client)
	default:
		return fmt.Sprintf("Error: unknown tool %q", tc.Function.Name)
	}
}

// executeWebSearch runs a single Tavily /search call.
func (a *WebAgent) executeWebSearch(ctx context.Context, tc ToolCall, client *integrations.TavilyClient, chatbot *Chatbot) string {
	var args struct {
		Query          string   `json:"query"`
		MaxResults     int      `json:"max_results,omitempty"`
		SearchDepth    string   `json:"search_depth,omitempty"`
		IncludeDomains []string `json:"include_domains,omitempty"`
		ExcludeDomains []string `json:"exclude_domains,omitempty"`
	}
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
		return fmt.Sprintf("Error: failed to parse web_search arguments: %v", err)
	}
	if args.Query == "" {
		return "Error: web_search requires a non-empty 'query'"
	}
	if args.MaxResults <= 0 {
		args.MaxResults = 5
	}
	if args.SearchDepth == "" {
		args.SearchDepth = "basic"
	}

	// Apply chatbot-level domain allowlist if the LLM didn't pass its own.
	// The intersection of chatbot-level + LLM-level is the strictest.
	if len(chatbot.WebSearchDomains) > 0 && len(args.IncludeDomains) == 0 {
		args.IncludeDomains = chatbot.WebSearchDomains
	}

	if a.deps.Sender != nil {
		a.deps.Sender.SendProgress(ctx, a.deps.ConversationID, "searching",
			fmt.Sprintf("Searching the web: %s", args.Query))
	}

	result, err := client.Search(ctx, integrations.SearchOptions{
		Query:          args.Query,
		MaxResults:     args.MaxResults,
		SearchDepth:    args.SearchDepth,
		IncludeDomains: args.IncludeDomains,
		ExcludeDomains: args.ExcludeDomains,
		IncludeAnswer:  true,
	})
	if err != nil {
		return fmt.Sprintf("Error running web_search: %v", err)
	}

	// Format for LLM consumption. Synthesized answer first (if present),
	// then result list with title/url/content.
	var sb strings.Builder
	if result.Answer != "" {
		fmt.Fprintf(&sb, "Tavily synthesized answer: %s\n\n", result.Answer)
	}
	fmt.Fprintf(&sb, "Top %d web results:\n", len(result.Results))
	for i, r := range result.Results {
		fmt.Fprintf(&sb, "\n%d. %s\n   URL: %s\n   %s\n",
			i+1, r.Title, r.URL, r.Content)
	}
	return sb.String()
}

// executeFetchURL runs a single Tavily /extract call.
func (a *WebAgent) executeFetchURL(ctx context.Context, tc ToolCall, client *integrations.TavilyClient) string {
	var args struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
		return fmt.Sprintf("Error: failed to parse fetch_url arguments: %v", err)
	}
	if args.URL == "" {
		return "Error: fetch_url requires a non-empty 'url'"
	}

	if a.deps.Sender != nil {
		a.deps.Sender.SendProgress(ctx, a.deps.ConversationID, "fetching",
			fmt.Sprintf("Fetching page: %s", args.URL))
	}

	result, err := client.Extract(ctx, integrations.ExtractOptions{URLs: []string{args.URL}})
	if err != nil {
		return fmt.Sprintf("Error running fetch_url: %v", err)
	}
	if len(result.Results) == 0 {
		return "Error: Tavily returned no results for this URL"
	}
	r := result.Results[0]
	if r.Failed {
		return fmt.Sprintf("Error extracting %s: %s", r.URL, r.Error)
	}
	// ponytail: cap content at 8KB so we don't blow the LLM context window.
	// Real-world pages can be 50KB+; the LLM only needs the relevant section.
	content := r.RawContent
	const maxLen = 8 * 1024
	if len(content) > maxLen {
		content = content[:maxLen] + "\n\n[... content truncated at 8KB ...]"
	}
	return fmt.Sprintf("URL: %s\n\n%s", r.URL, content)
}

// WebSearchTool is the function-calling tool definition for web_search.
var WebSearchTool = Tool{
	Type: "function",
	Function: ToolFunction{
		Name:        "web_search",
		Description: "Search the web for current information using Tavily. Use for questions about recent events, current prices/hours, news, documentation, or anything that needs up-to-date info from the internet.\n\nWHEN TO USE:\n- User asks about current events, news, or anything time-sensitive\n- User asks \"what is the latest X\" or \"how do I do X today\"\n- User asks about specific websites, products, or services\n- KB doesn't have the answer and SQL data won't help\n\nWHEN NOT TO USE:\n- The user asks about historical facts that haven't changed\n- The chatbot's knowledge base already has the answer\n- The user is asking about data in the application's own database\n\nReturns: synthesized answer (when available) + top web results with title, URL, and content snippet.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "The search query. Be specific — include relevant keywords, dates, or names.",
				},
				"max_results": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum results to return (default 5, max 10).",
				},
				"search_depth": map[string]interface{}{
					"type":        "string",
					"description": `"basic" (default, faster) or "advanced" (slower, more thorough). Use "advanced" for complex research questions.`,
				},
			},
			"required": []string{"query"},
		},
	},
}

// FetchURLTool is the function-calling tool definition for fetch_url.
var FetchURLTool = Tool{
	Type: "function",
	Function: ToolFunction{
		Name:        "fetch_url",
		Description: "Fetch the full content of a specific URL as clean markdown. Use after web_search when the top result needs more detail to answer the user's question.\n\nWHEN TO USE:\n- web_search returned a promising URL but the snippet is too short\n- User provided a specific URL directly\n- You need exact content from a page (e.g., current hours, menu, prices)\n\nReturns: the page's content as markdown (capped at 8KB to fit context window).",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"url": map[string]interface{}{
					"type":        "string",
					"description": "The fully-qualified URL to fetch (e.g., https://example.com/page).",
				},
			},
			"required": []string{"url"},
		},
	},
}
