package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// KBAgent is the specialist that answers conceptual questions using
// knowledge-base retrieval. It wraps the existing RAGService — no RAG
// logic is duplicated.
type KBAgent struct {
	deps *AgentDeps
}

// NewKBAgent constructs a KBAgent bound to the given deps.
func NewKBAgent(deps *AgentDeps) *KBAgent { return &KBAgent{deps: deps} }

// Name implements Node.
func (a *KBAgent) Name() string { return "kb" }

// Run executes the KB agent's retrieval-then-answer flow.
func (a *KBAgent) Run(ctx context.Context, state *State) error {
	chatbot := a.deps.Chatbot
	provider := a.deps.Provider
	if provider == nil {
		return fmt.Errorf("kb agent: no provider")
	}

	if a.deps.Sender != nil {
		a.deps.Sender.SendAgentTransition(ctx, a.deps.ConversationID, AgentTransition{To: "kb"})
	}

	userMsg := state.UserMessage()

	// Build prompt + dynamic context
	systemPrompt := BuildKBAgentPrompt(chatbot)
	dynamicContext := BuildDynamicContextForAgent(chatbot, a.deps.UserID, "kb", a.deps.PageProfile)

	// Retrieve RAG context. KB list may be overridden by the page profile.
	kbs := a.deps.PageProfile.ResolvedKBs(chatbot.KnowledgeBases)
	if a.deps.RAGService != nil && len(kbs) > 0 {
		opts := RetrieveContextOptions{
			ChatbotID: chatbot.ID,
			Query:     userMsg,
			UserID:    a.deps.UserID,
		}
		if chatbot.RAGMaxChunks > 0 {
			opts.MaxChunks = chatbot.RAGMaxChunks
		}
		if chatbot.RAGSimilarityThreshold > 0 {
			opts.Threshold = chatbot.RAGSimilarityThreshold
		}
		if chatbot.RAGGraphBoostWeight > 0 {
			opts.GraphBoostWeight = chatbot.RAGGraphBoostWeight
		}
		section, err := a.deps.RAGService.RetrieveContext(ctx, opts)
		if err == nil && section != nil && section.FormattedContext != "" {
			dynamicContext += "\n\n## Retrieved Context\n\n" + section.FormattedContext
		}
	}

	messages := []Message{
		{Role: RoleSystem, Content: systemPrompt},
		{Role: RoleSystem, Content: dynamicContext},
	}
	messages = append(messages, tailMessages(state.ConversationHistory(), 4)...)
	messages = append(messages, Message{Role: RoleUser, Content: userMsg})

	model := chatbot.Model
	if override, ok := chatbot.SupervisorAgentModels["kb"]; ok && override != "" {
		model = override
	}

	// KB agent does NOT use streaming here (simpler). The synthesizer is
	// responsible for the final streamed output to the client when
	// synthesis is engaged. If this is the only agent on the route and
	// synthesis was skipped, the chat_handler_message.go path emits the
	// final response in one shot.
	req := &ChatRequest{
		Messages:    messages,
		Model:       model,
		MaxTokens:   chatbot.MaxTokens,
		Temperature: chatbot.Temperature,
		Stream:      false,
	}

	resp, err := provider.Chat(ctx, req)
	if err != nil {
		return fmt.Errorf("kb agent: provider call failed: %w", err)
	}
	if resp == nil || len(resp.Choices) == 0 {
		return fmt.Errorf("kb agent: empty provider response")
	}
	if resp.Usage != nil {
		state.AddUsage(*resp.Usage)
	}

	content := resp.Choices[0].Message.Content
	state.AppendAgentOutput("kb", content)
	state.SetFinalResponse(content)
	return nil
}

// ActionAgent is the specialist that executes mutations and RPC procedures.
type ActionAgent struct {
	deps *AgentDeps
}

// NewActionAgent constructs an ActionAgent bound to the given deps.
func NewActionAgent(deps *AgentDeps) *ActionAgent { return &ActionAgent{deps: deps} }

// Name implements Node.
func (a *ActionAgent) Name() string { return "action" }

// Run executes the action agent's tool-calling loop.
func (a *ActionAgent) Run(ctx context.Context, state *State) error {
	chatbot := a.deps.Chatbot
	provider := a.deps.Provider
	if provider == nil {
		return fmt.Errorf("action agent: no provider")
	}
	if a.deps.MCPExecutor == nil {
		return fmt.Errorf("action agent: MCP executor not configured")
	}

	if a.deps.Sender != nil {
		a.deps.Sender.SendAgentTransition(ctx, a.deps.ConversationID, AgentTransition{To: "action"})
	}

	systemPrompt := BuildActionAgentPrompt(chatbot)
	dynamicContext := BuildDynamicContextForAgent(chatbot, a.deps.UserID, "action", a.deps.PageProfile)
	allowedTables := a.deps.PageProfile.ResolvedTables(chatbot.AllowedTables)
	if len(allowedTables) > 0 {
		dynamicContext += fmt.Sprintf("\nTables in scope: %s\n", strings.Join(allowedTables, ", "))
	}

	userMsg := state.UserMessage()
	messages := []Message{
		{Role: RoleSystem, Content: systemPrompt},
		{Role: RoleSystem, Content: dynamicContext},
	}
	messages = append(messages, tailMessages(state.ConversationHistory(), 4)...)
	messages = append(messages, Message{Role: RoleUser, Content: userMsg})

	// Get available MCP tools, filtered to action-oriented ones
	tools := a.buildToolList(chatbot)
	if len(tools) == 0 {
		// No action tools configured — fall back to a chat-style response
		noToolsMsg := "I don't have any action tools configured for this chatbot."
		state.AppendAgentOutput("action", noToolsMsg)
		state.SetFinalResponse(noToolsMsg)
		return nil
	}

	maxIters := chatbot.MaxToolIterations
	if maxIters <= 0 {
		maxIters = 5
	}

	model := chatbot.Model
	if override, ok := chatbot.SupervisorAgentModels["action"]; ok && override != "" {
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
			return fmt.Errorf("action agent: provider call failed: %w", err)
		}
		if resp == nil || len(resp.Choices) == 0 {
			return fmt.Errorf("action agent: empty provider response")
		}
		if resp.Usage != nil {
			state.AddUsage(*resp.Usage)
		}

		msg := resp.Choices[0].Message

		// Emit pre-tool reasoning as a thought-process chunk.
		if msg.Content != "" && a.deps.Sender != nil {
			a.deps.Sender.SendAgentThought(ctx, a.deps.ConversationID, AgentThought{
				Agent: "action",
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
			// Emit tool_call thought before executing so the client can
			// render the action flow ("calling invoke_function with...").
			if a.deps.Sender != nil {
				a.deps.Sender.SendAgentThought(ctx, a.deps.ConversationID, AgentThought{
					Agent:    "action",
					Kind:     "tool_call",
					ToolName: tc.Function.Name,
					ToolArgs: json.RawMessage(tc.Function.Arguments),
				})
			}
			resultStr := a.executeActionTool(ctx, tc, chatbot)
			// Short tool_result summary in the thought stream.
			if a.deps.Sender != nil {
				summary := resultStr
				if len(summary) > 200 {
					summary = summary[:200] + "..."
				}
				a.deps.Sender.SendAgentThought(ctx, a.deps.ConversationID, AgentThought{
					Agent: "action",
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
		finalContent = "Action completed."
	}

	state.AppendAgentOutput("action", finalContent)
	state.SetFinalResponse(finalContent)
	return nil
}

// buildToolList returns the action-oriented MCP tools available to this chatbot.
func (a *ActionAgent) buildToolList(chatbot *Chatbot) []Tool {
	if !chatbot.HasMCPTools() {
		return nil
	}
	mcpDefs := a.deps.MCPExecutor.GetAvailableTools(chatbot)
	var out []Tool
	for _, def := range mcpDefs {
		// Look up the tool's category to decide if it's an action tool.
		info, ok := MCPToolInfoMap[def.Name]
		if !ok || info.Category != MCPToolCategoryExecution {
			continue
		}
		out = append(out, Tool{
			Type:     "function",
			Function: ToolFunction(def),
		})
	}
	return out
}

// executeActionTool dispatches one MCP tool call.
//
// chatCtx is required — MCP tools (ChatbotAuthContext) read user/role from
// it for RLS-scoped operations. The supervisor path passes deps.ChatCtx
// (which is always populated when running under the WS handler). The
// nil-safe fallback in ChatbotAuthContext covers tests that don't construct
// a full ChatContext, but production paths must populate it.
func (a *ActionAgent) executeActionTool(ctx context.Context, tc ToolCall, chatbot *Chatbot) string {
	var args map[string]any
	if err := parseJSONArgs(tc.Function.Arguments, &args); err != nil {
		return fmt.Sprintf("Error: failed to parse tool arguments: %v", err)
	}
	if a.deps.Sender != nil {
		a.deps.Sender.SendProgress(ctx, a.deps.ConversationID, "executing", fmt.Sprintf("Executing %s...", tc.Function.Name))
	}
	result, err := a.deps.MCPExecutor.ExecuteTool(ctx, tc.Function.Name, args, a.deps.ChatCtx, chatbot)
	if err != nil {
		return fmt.Sprintf("Error executing %s: %v", tc.Function.Name, err)
	}
	return result.Content
}

// ChatAgent is the specialist for chitchat, clarifications, and follow-ups.
type ChatAgent struct {
	deps *AgentDeps
}

// NewChatAgent constructs a ChatAgent bound to the given deps.
func NewChatAgent(deps *AgentDeps) *ChatAgent { return &ChatAgent{deps: deps} }

// Name implements Node.
func (a *ChatAgent) Name() string { return "chat" }

// Run generates a brief conversational response.
func (a *ChatAgent) Run(ctx context.Context, state *State) error {
	chatbot := a.deps.Chatbot
	provider := a.deps.Provider
	if provider == nil {
		return fmt.Errorf("chat agent: no provider")
	}

	if a.deps.Sender != nil {
		a.deps.Sender.SendAgentTransition(ctx, a.deps.ConversationID, AgentTransition{To: "chat"})
	}

	systemPrompt := BuildChatAgentPrompt(chatbot)
	dynamicContext := BuildDynamicContextForAgent(chatbot, a.deps.UserID, "chat", a.deps.PageProfile)
	// Include the user's detected language (from supervisor) so the chat
	// agent replies in the right language even on cheap models.
	if lang := state.UserLanguage(); lang != "" {
		dynamicContext += fmt.Sprintf("\nUser's language: %s\n", lang)
	}

	userMsg := state.UserMessage()
	messages := []Message{
		{Role: RoleSystem, Content: systemPrompt},
		{Role: RoleSystem, Content: dynamicContext},
	}
	messages = append(messages, tailMessages(state.ConversationHistory(), 4)...)
	messages = append(messages, Message{Role: RoleUser, Content: userMsg})

	model := chatbot.Model
	if override, ok := chatbot.SupervisorAgentModels["chat"]; ok && override != "" {
		model = override
	}

	req := &ChatRequest{
		Messages:    messages,
		Model:       model,
		MaxTokens:   512, // chat responses should be short
		Temperature: chatbot.Temperature,
		Stream:      false,
	}

	resp, err := provider.Chat(ctx, req)
	if err != nil {
		return fmt.Errorf("chat agent: provider call failed: %w", err)
	}
	if resp == nil || len(resp.Choices) == 0 {
		return fmt.Errorf("chat agent: empty provider response")
	}
	if resp.Usage != nil {
		state.AddUsage(*resp.Usage)
	}

	content := resp.Choices[0].Message.Content
	state.AppendAgentOutput("chat", content)
	state.SetFinalResponse(content)
	return nil
}

// parseJSONArgs unmarshals argsJSON into out. Returns error on failure.
func parseJSONArgs(argsJSON string, out any) error {
	return json.Unmarshal([]byte(argsJSON), out)
}
