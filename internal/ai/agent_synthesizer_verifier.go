package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
)

// SynthesizerAgent merges outputs from multiple specialist agents into one
// coherent answer in the user's detected language. Skipped when only one
// agent ran (the single agent's output is the final answer).
type SynthesizerAgent struct {
	deps *AgentDeps
}

// NewSynthesizerAgent constructs a SynthesizerAgent bound to the given deps.
func NewSynthesizerAgent(deps *AgentDeps) *SynthesizerAgent {
	return &SynthesizerAgent{deps: deps}
}

// Name implements Node.
func (a *SynthesizerAgent) Name() string { return "synthesizer" }

// Run merges specialist outputs into one final answer.
func (a *SynthesizerAgent) Run(ctx context.Context, state *State) error {
	chatbot := a.deps.Chatbot
	provider := a.deps.Provider
	if provider == nil {
		return fmt.Errorf("synthesizer: no provider")
	}

	outputs := state.AgentOutputs()
	if len(outputs) == 0 {
		// Nothing to synthesize — final response stays whatever was last set
		return nil
	}
	if len(outputs) == 1 {
		// Single agent → its output IS the final response. Skip synthesis.
		state.SetFinalResponse(outputs[0].Content)
		return nil
	}

	if a.deps.Sender != nil {
		a.deps.Sender.SendAgentTransition(ctx, a.deps.ConversationID, AgentTransition{To: "synthesizer"})
	}

	systemPrompt := BuildSynthesizerPrompt(chatbot)
	dynamicContext := fmt.Sprintf("Current date and time: %s\n", currentTimeForPrompt())
	if lang := state.UserLanguage(); lang != "" {
		dynamicContext += fmt.Sprintf("\nUser's detected language: %s\n", lang)
		dynamicContext += "Write the synthesized answer in this language.\n"
	}

	// Build the user message: original question + each agent's output
	var userMsg strings.Builder
	fmt.Fprintf(&userMsg, "Original user question:\n%s\n\n", state.UserMessage())
	userMsg.WriteString("Specialist agent outputs:\n\n")
	for _, o := range outputs {
		fmt.Fprintf(&userMsg, "## %s agent\n%s\n\n", o.Name, o.Content)
	}
	userMsg.WriteString("Synthesize the above into a single coherent answer.")

	messages := []Message{
		{Role: RoleSystem, Content: systemPrompt},
		{Role: RoleSystem, Content: dynamicContext},
		{Role: RoleUser, Content: userMsg.String()},
	}

	model := chatbot.Model
	if override, ok := chatbot.SupervisorAgentModels["synthesizer"]; ok && override != "" {
		model = override
	}

	req := &ChatRequest{
		Messages:    messages,
		Model:       model,
		MaxTokens:   chatbot.MaxTokens,
		Temperature: chatbot.Temperature,
		Stream:      false,
	}

	resp, err := provider.Chat(ctx, req)
	if err != nil {
		return fmt.Errorf("synthesizer: provider call failed: %w", err)
	}
	if resp == nil || len(resp.Choices) == 0 {
		return fmt.Errorf("synthesizer: empty provider response")
	}
	if resp.Usage != nil {
		state.AddUsage(*resp.Usage)
	}

	state.SetFinalResponse(resp.Choices[0].Message.Content)
	return nil
}

// VerifierAgent checks the final answer against (1) the user's language
// and (2) tool-result grounding. Runs only on the investigative path.
// One retry cap — verifier never blocks the response permanently.
type VerifierAgent struct {
	deps *AgentDeps
}

// NewVerifierAgent constructs a VerifierAgent bound to the given deps.
func NewVerifierAgent(deps *AgentDeps) *VerifierAgent {
	return &VerifierAgent{deps: deps}
}

// Name implements Node.
func (a *VerifierAgent) Name() string { return "verifier" }

// Run performs the verification checks. Stores the report in state but
// does NOT mutate the final response directly — the caller (supervisor
// graph) decides whether to retry based on the report.
func (a *VerifierAgent) Run(ctx context.Context, state *State) error {
	report := &VerifyReport{}

	userMsg := state.UserMessage()
	response := state.FinalResponse()

	// Always: rule-based language script check
	report.LanguageOK = checkLanguageScriptMatch(userMsg, response)

	// Conditional LLM grounding check
	plan, _ := state.Get(SupervisorPlanKey)
	supervisorPlan, _ := plan.(*SupervisorPlan)
	toolResults := state.ToolResults()

	if supervisorPlan != nil && supervisorPlan.IsInvestigative && report.LanguageOK && len(toolResults) > 0 && a.deps.Provider != nil {
		if a.deps.Sender != nil {
			a.deps.Sender.SendAgentTransition(ctx, a.deps.ConversationID, AgentTransition{To: "verifier"})
		}
		llmReport, err := a.groundingCheck(ctx, userMsg, response, toolResults, supervisorPlan)
		if err == nil && llmReport != nil {
			report.GroundingOK = llmReport.OK
			report.Issues = append(report.Issues, llmReport.Issues...)
		}
	} else {
		report.GroundingOK = true
	}

	state.Set(VerifyReportKey, report)
	return nil
}

// VerifyReport is the verifier's output, stored in state for the supervisor
// graph to act on.
type VerifyReport struct {
	LanguageOK  bool
	GroundingOK bool
	Issues      []string
}

// VerifyReportKey is the state key for the verifier's report.
const VerifyReportKey = "verify_report"

// groundingCheck runs the LLM grounding check and returns its verdict.
func (a *VerifierAgent) groundingCheck(ctx context.Context, userMsg, response string, toolResults []QueryResult, plan *SupervisorPlan) (*verifierLLMReport, error) {
	chatbot := a.deps.Chatbot
	provider := a.deps.Provider

	systemPrompt := BuildVerifierPrompt()

	// Format tool results compactly for the verifier
	var toolDump strings.Builder
	for i, r := range toolResults {
		fmt.Fprintf(&toolDump, "### Query %d\nSQL: %s\nSummary: %s\nRows returned: %d\n", i+1, r.Query, r.Summary, r.RowCount)
		if len(r.Data) > 0 {
			sample := r.Data
			if len(sample) > 3 {
				sample = sample[:3]
			}
			sampleJSON, _ := json.Marshal(sample)
			fmt.Fprintf(&toolDump, "Sample rows: %s\n\n", string(sampleJSON))
		} else {
			toolDump.WriteString("\n")
		}
	}

	userContent := fmt.Sprintf(
		"User question:\n%s\n\nAnswer to verify:\n%s\n\nTool results:\n%s",
		userMsg, response, toolDump.String(),
	)

	messages := []Message{
		{Role: RoleSystem, Content: systemPrompt},
		{Role: RoleUser, Content: userContent},
	}

	model := chatbot.Model
	if override, ok := chatbot.SupervisorAgentModels["verifier"]; ok && override != "" {
		model = override
	}

	req := &ChatRequest{
		Messages:    messages,
		Model:       model,
		MaxTokens:   300,
		Temperature: 0,
		Stream:      false,
	}

	resp, err := provider.Chat(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp == nil || len(resp.Choices) == 0 {
		return nil, fmt.Errorf("empty verifier response")
	}
	if resp.Usage != nil {
		s := getStateFromContext(ctx)
		if s != nil {
			s.AddUsage(*resp.Usage)
		}
	}

	content := resp.Choices[0].Message.Content
	cleaned := extractJSONObjectFromString(content)
	if cleaned == "" {
		// ponytail: if LLM didn't return JSON, assume grounding OK
		// (don't fail the response on a verifier formatting bug)
		return &verifierLLMReport{OK: true}, nil
	}
	var out verifierLLMReport
	if err := json.Unmarshal([]byte(cleaned), &out); err != nil {
		return &verifierLLMReport{OK: true}, nil
	}
	return &out, nil
}

type verifierLLMReport struct {
	OK     bool     `json:"ok"`
	Issues []string `json:"issues"`
}

// checkLanguageScriptMatch returns true if userMsg and response are written
// in the same dominant Unicode script. Catches the common failure mode
// where the model replies in English to a non-English question.
//
// Heuristic, not a hard guarantee — mixed-script messages (e.g., a German
// message with English brand names) are matched permissively.
func checkLanguageScriptMatch(userMsg, response string) bool {
	userScript := dominantScript(userMsg)
	respScript := dominantScript(response)

	// If either is empty/mixed, treat as a match (be permissive)
	if userScript == "" || respScript == "" || userScript == "mixed" || respScript == "mixed" {
		return true
	}
	return userScript == respScript
}

// dominantScript returns the dominant Unicode script of s by counting
// letters in each script. Returns "mixed" if no script has >60% of letters,
// "" if there are no letters at all.
func dominantScript(s string) string {
	counts := map[string]int{}
	total := 0
	for _, r := range s {
		if !unicode.IsLetter(r) {
			continue
		}
		total++
		for name, table := range scriptTables {
			if table(r) {
				counts[name]++
				break
			}
		}
	}
	if total == 0 {
		return ""
	}
	var dominant string
	max := 0
	for name, c := range counts {
		if c > max {
			max = c
			dominant = name
		}
	}
	if max*100/total < 60 {
		return "mixed"
	}
	return dominant
}

// scriptTables is the per-script classifier set. Order matters: more
// specific checks first (e.g., Hiragana before "any Han").
var scriptTables = map[string]func(rune) bool{
	"latin":      func(r rune) bool { return unicode.In(r, unicode.Latin) },
	"cyrillic":   func(r rune) bool { return unicode.In(r, unicode.Cyrillic) },
	"greek":      func(r rune) bool { return unicode.In(r, unicode.Greek) },
	"arabic":     func(r rune) bool { return unicode.In(r, unicode.Arabic) },
	"hebrew":     func(r rune) bool { return unicode.In(r, unicode.Hebrew) },
	"han":        func(r rune) bool { return unicode.In(r, unicode.Han) },
	"hiragana":   func(r rune) bool { return unicode.In(r, unicode.Hiragana) },
	"katakana":   func(r rune) bool { return unicode.In(r, unicode.Katakana) },
	"hangul":     func(r rune) bool { return unicode.In(r, unicode.Hangul) },
	"devanagari": func(r rune) bool { return unicode.In(r, unicode.Devanagari) },
	"thai":       func(r rune) bool { return unicode.In(r, unicode.Thai) },
}

// getStateFromContext pulls the state back out of context for usage
// accumulation in nested helper functions. Returns nil if not present.
// ponytail: thread-through-context saves us threading *State through every
// helper signature. If usage accounting gets complex, upgrade to explicit
// parameter.
func getStateFromContext(ctx context.Context) *State {
	if s, ok := ctx.Value(stateContextKey{}).(*State); ok {
		return s
	}
	return nil
}

// stateContextKey is the context key type for *State values.
type stateContextKey struct{}

// ContextWithState returns ctx with state attached for downstream helpers.
func ContextWithState(ctx context.Context, state *State) context.Context {
	return context.WithValue(ctx, stateContextKey{}, state)
}
