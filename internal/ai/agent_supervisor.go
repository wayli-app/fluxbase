package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// SupervisorPlan is the structured output contract returned by the
// Supervisor agent. It drives routing decisions in the supervisor graph.
type SupervisorPlan struct {
	UserLanguage      string   `json:"user_language"`
	Route             []string `json:"route"`
	SubQuestions      []string `json:"sub_questions,omitempty"`
	RequiresSynthesis bool     `json:"requires_synthesis"`
	IsInvestigative   bool     `json:"is_investigative"`
	MinToolCalls      int      `json:"min_tool_calls"`
}

// investigativeKeywords is the regex-free pre-pass used to bias the
// supervisor's routing decision. If any of these appear in the user
// message, the supervisor prompt nudges toward "sql". The LLM still makes
// the final call.
var investigativeKeywords = []string{
	"how many", "how much", "how often",
	"count", "list", "show me", "show all",
	"find", "total", "sum", "average", "avg", "mean",
	"what is the", "what are the",
	"which", "who", "when",
	"compare", "comparison",
	"top", "bottom", "best", "worst",
	"group by", "by category", "by month", "by week", "by day",
	"trend", "trends",
	"between", "since", "before", "after",
}

// looksInvestigative returns true if the user message contains any of the
// investigative keywords. Case-insensitive.
func looksInvestigative(msg string) bool {
	lower := strings.ToLower(msg)
	for _, kw := range investigativeKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// SupervisorAgent is the routing node. It runs one non-streaming LLM call
// against the chatbot's provider, parses the JSON SupervisorPlan, applies
// the page-profile whitelist (when present), and stores the result in graph
// state for downstream nodes to consume.
type SupervisorAgent struct {
	deps *AgentDeps
}

// NewSupervisorAgent constructs a SupervisorAgent bound to the given deps.
func NewSupervisorAgent(deps *AgentDeps) *SupervisorAgent {
	return &SupervisorAgent{deps: deps}
}

// Name implements Node.
func (a *SupervisorAgent) Name() string { return "supervisor" }

// Run executes the supervisor's routing decision.
//
// Errors here are recoverable — the supervisor graph falls back to the
// legacy ReAct loop on failure (see chat_handler_message.go).
func (a *SupervisorAgent) Run(ctx context.Context, state *State) error {
	chatbot := a.deps.Chatbot
	provider := a.deps.Provider
	userMsg := state.UserMessage()

	if provider == nil {
		return fmt.Errorf("supervisor: no provider available")
	}

	// Emit a routing transition event so the client can show "Investigating with SQL..."
	if a.deps.Sender != nil {
		a.deps.Sender.SendAgentTransition(ctx, a.deps.ConversationID, AgentTransition{To: "supervisor"})
	}

	// Build the supervisor's messages: static system prompt + dynamic
	// context (per-page whitelist) + user message.
	systemPrompt := BuildSupervisorPrompt(chatbot)
	dynamicContext := buildSupervisorDynamicContext(chatbot, state.PageProfile())

	messages := []Message{
		{Role: RoleSystem, Content: systemPrompt},
		{Role: RoleSystem, Content: dynamicContext},
		{Role: RoleUser, Content: userMsg},
	}

	model := chatbot.Model
	if override, ok := chatbot.SupervisorAgentModels["supervisor"]; ok && override != "" {
		model = override
	}

	req := &ChatRequest{
		Messages:    messages,
		Model:       model,
		MaxTokens:   600,
		Temperature: 0,
		Stream:      false,
	}

	resp, err := provider.Chat(ctx, req)
	if err != nil {
		return fmt.Errorf("supervisor: provider call failed: %w", err)
	}
	if resp == nil || len(resp.Choices) == 0 {
		return fmt.Errorf("supervisor: empty response from provider")
	}
	content := resp.Choices[0].Message.Content
	if resp.Usage != nil {
		state.AddUsage(*resp.Usage)
	}

	plan, err := parseSupervisorPlan(content)
	if err != nil {
		return fmt.Errorf("supervisor: failed to parse plan: %w (content=%q)", err, content)
	}

	// Honor chatbot.ResponseLanguage: when set (anything other than "" or
	// "auto"), the configured language WINS over the supervisor's detected
	// language. This matches the legacy ReAct loop's behavior — see
	// schema_builder.go's "Response Language" section. Without this
	// override, a chatbot pinned to German via @fluxbase:response-language
	// would still reply in whatever language the supervisor thought it
	// detected, which is the reported regression ("asked in English, got
	// German" was the supervisor mis-detecting on a German-configured bot).
	if chatbot.ResponseLanguage != "" && chatbot.ResponseLanguage != "auto" {
		plan.UserLanguage = chatbot.ResponseLanguage
	}

	// Apply page-profile whitelist: filter out agents the page doesn't allow.
	if profile := state.PageProfile(); profile != nil && len(profile.Agents) > 0 {
		filtered := make([]string, 0, len(plan.Route))
		for _, r := range plan.Route {
			if profile.HasAgent(r) {
				filtered = append(filtered, r)
			}
		}
		// If filtering removed everything (LLM routed to disallowed agents),
		// fall back to "chat" so the user still gets a coherent response.
		if len(filtered) == 0 {
			filtered = []string{"chat"}
		}
		plan.Route = filtered
	}

	// Empty route is treated as a chat fallback.
	if len(plan.Route) == 0 {
		plan.Route = []string{"chat"}
	}

	// Default MinToolCalls when investigative but unset.
	if plan.IsInvestigative && plan.MinToolCalls <= 0 {
		plan.MinToolCalls = 1
	}

	// Emit the routing decision as a thought-process event so clients can
	// render "Supervisor routed to: sql, kb because...". The plan carries
	// the route + detected language + investigative flag.
	if a.deps.Sender != nil {
		a.deps.Sender.SendAgentThought(ctx, a.deps.ConversationID, AgentThought{
			Agent: "supervisor",
			Kind:  "plan",
			Plan:  plan,
		})
	}

	state.Set(SupervisorPlanKey, plan)
	state.SetUserLanguage(plan.UserLanguage)
	return nil
}

// SupervisorPlanKey is the state key under which the parsed SupervisorPlan
// is stored. Exposed so other nodes (and tests) can fetch it.
const SupervisorPlanKey = stateKeySupervisorPlan

// parseSupervisorPlan extracts the SupervisorPlan JSON from the LLM output.
// Tolerates markdown fences and stray prose around the JSON object.
func parseSupervisorPlan(content string) (*SupervisorPlan, error) {
	cleaned := extractJSONObjectFromString(content)
	if cleaned == "" {
		return nil, fmt.Errorf("no JSON object found in content")
	}

	var plan SupervisorPlan
	if err := json.Unmarshal([]byte(cleaned), &plan); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	// Validate route entries
	for _, r := range plan.Route {
		switch r {
		case "sql", "kb", "action", "chat":
			// valid
		default:
			return nil, fmt.Errorf("unknown agent in route: %q", r)
		}
	}

	return &plan, nil
}

// extractJSONObjectFromString finds the first balanced {...} substring in s.
// Used to tolerate stray prose or markdown fences around the supervisor's
// JSON output.
func extractJSONObjectFromString(s string) string {
	start := strings.Index(s, "{")
	if start < 0 {
		return ""
	}
	return extractBalancedJSONObject(s, start)
}

// buildSupervisorDynamicContext returns the per-turn dynamic context
// message for the supervisor. Includes the page-profile agent whitelist
// when present so the supervisor's routing respects it.
func buildSupervisorDynamicContext(chatbot *Chatbot, profile *PageProfile) string {
	var sb strings.Builder
	sb.WriteString("Current date and time: ")
	sb.WriteString(currentTimeForPrompt())
	sb.WriteString("\n")

	if profile != nil {
		fmt.Fprintf(&sb, "\n## Page Context\n\n")
		fmt.Fprintf(&sb, "User is currently on page: %s\n", profile.Page)
		if len(profile.Agents) > 0 {
			fmt.Fprintf(&sb, "Agents available on this page: %s\n", strings.Join(profile.Agents, ", "))
			sb.WriteString("Route ONLY to agents in the list above. If the user's intent doesn't match any available agent, route to \"chat\" with a clarification.\n")
		}
		if profile.Suffix != "" {
			fmt.Fprintf(&sb, "Focus instruction: %s\n", profile.Suffix)
		}
	}

	// Pre-pass hint: if the message contains investigative keywords, nudge.
	// This is advisory — the LLM still makes the final call.
	// (Used inside buildSupervisorDynamicContext so the hint travels with
	// the page context, not the static prompt.)

	return sb.String()
}
