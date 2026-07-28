package ai

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/ai/eval"
)

// TestEvalRoutingCorpus_Deterministic runs the wayli routing corpus against
// the supervisor in deterministic mode: a fake provider returns each case's
// canned plan, and we verify the parser + page-profile filtering apply it
// correctly. This is the regression guard for routing plumbing — prompt
// changes are verified by the live mode (manual).
//
// This test pins the contract: if a refactor breaks how plans flow through
// page-profile whitelists or state, this fails before it reaches production.
func TestEvalRoutingCorpus_Deterministic(t *testing.T) {
	corpusPath := filepath.Join("eval", "testdata", "wayli_routing.eval.json")
	corpus, err := eval.LoadCorpus(corpusPath)
	require.NoError(t, err, "failed to load corpus at %s", corpusPath)
	require.NotEmpty(t, corpus.Cases, "corpus has no cases")

	// Build a chatbot whose page-profiles mirror wayli's, so page-context
	// whitelisting is exercised the same way production configures it.
	chatbotFactory := func(c eval.Case) *Chatbot {
		cb := &Chatbot{
			Name:             "wayli-assistant-eval",
			Model:            "gpt-4",
			WebSearchEnabled: true,
			PageProfiles: PageProfiles{
				"default": {Page: "default", Agents: []string{"sql", "kb", "action", "chat", "web"}},
				"plan":    {Page: "plan", Agents: []string{"sql", "action", "web"}},
			},
		}
		return cb
	}

	results := make([]eval.Result, 0, len(corpus.Cases))
	for _, c := range corpus.Cases {
		c := c
		t.Run(c.ID, func(t *testing.T) {
			require.NotEmpty(t, c.DeterministicPlan,
				"deterministic mode requires 'deterministic_plan' for case %q", c.ID)

			provider := newFakeProvider("eval")
			provider.QueueResponse(newSimpleResponse(c.DeterministicPlan))

			chatbot := chatbotFactory(c)
			deps := &AgentDeps{Chatbot: chatbot, Provider: provider}
			supervisor := NewSupervisorAgent(deps)

			state := NewState()
			state.SetUserMessage(c.Message)
			if c.PageContext != "" {
				if profile := chatbot.PageProfiles.Resolve(c.PageContext); profile != nil {
					state.SetPageProfile(profile)
				}
			}

			err := supervisor.Run(context.Background(), state)
			require.NoError(t, err, "supervisor Run failed for case %q", c.ID)

			// Extract the route the supervisor stored in state.
			route := routeFromState(t, state)

			passed, reason := eval.CheckRoute(c, route)
			results = append(results, eval.Result{Case: c, Passed: passed, Reason: reason, Route: route})
			if !passed {
				t.Errorf("case %q [%s]: %s\n  route=%v", c.ID, c.Category, reason, route)
			}
		})
	}

	t.Logf("\n%s", eval.SummaryReport(results))
	if eval.CountPassed(results) != len(results) {
		t.Fatalf("%d/%d eval cases failed", len(results)-eval.CountPassed(results), len(results))
	}
}

// routeFromState reads the supervisor's stored plan and returns its Route.
func routeFromState(t *testing.T, state *State) []string {
	t.Helper()
	planVal, ok := state.Get(SupervisorPlanKey)
	require.True(t, ok, "no supervisor plan stored in state")
	plan, ok := planVal.(*SupervisorPlan)
	require.True(t, ok, "plan in state is not a *SupervisorPlan")
	if plan == nil {
		return nil
	}
	return plan.Route
}
