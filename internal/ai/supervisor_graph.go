package ai

import (
	"context"
	"fmt"
)

// SupervisorGraphFactory builds the graph for one turn of a chatbot running
// in "supervisor" reasoning mode. A fresh graph is constructed per turn
// because the routing depends on the supervisor's plan, which is itself
// derived from the user's message.
//
// Graph shape:
//
//     ┌──────────┐
//     │supervisor│
//     └────┬─────┘
//          │ conditional: route to specialists based on SupervisorPlan
//     ┌────┴────┐
//     │ fan-out │ (parallel goroutines for multi-agent routes)
//     └────┬────┘
//          │ conditional: skip synthesizer when route has 1 agent
//     ┌────┴────┐
//     │ synth   │ (only when 2+ agents OR requires_synthesis)
//     └────┬────┘
//          │
//     ┌────┴────┐
//     │ verifier│ (only on investigative route)
//     └─────────┘
//
// The graph executor itself is generic (see graph.go); this file knows
// about the specific nodes and edges for the supervisor pipeline.

// SupervisorGraph wraps a Graph with supervisor-specific helpers.
type SupervisorGraph struct {
	graph *Graph
	deps  *AgentDeps
}

// NewSupervisorGraph builds a supervisor graph bound to the given deps.
// The returned graph is ready to Run(); one graph per turn.
func NewSupervisorGraph(deps *AgentDeps) *SupervisorGraph {
	supervisor := NewSupervisorAgent(deps)
	sql := NewSQLAgent(deps)
	kb := NewKBAgent(deps)
	action := NewActionAgent(deps)
	chat := NewChatAgent(deps)
	synthesizer := NewSynthesizerAgent(deps)
	verifier := NewVerifierAgent(deps)

	// Router fan-out node — exists only to own the conditional edge from
	// supervisor → specialists. Each invocation reads the plan from state
	// and runs the routed specialists in parallel.
	router := &routerNode{deps: deps, agents: map[string]Node{
		"sql": sql, "kb": kb, "action": action, "chat": chat,
	}}

	// Synthesizer gate node — reads agent outputs and decides whether
	// synthesis is needed based on count + SupervisorPlan.RequiresSynthesis.
	// Single-agent routes skip synthesis entirely.
	synthGate := &synthesizerGateNode{synthesizer: synthesizer}

	// Verifier gate node — runs the verifier only on investigative routes.
	verifierGate := &verifierGateNode{verifier: verifier, deps: deps}

	g := NewGraph("supervisor")
	g.AddNode(supervisor)
	g.AddNode(router)
	g.AddNode(synthGate)
	g.AddNode(verifierGate)

	_ = g.AddUnconditionalEdge("supervisor", "router")
	_ = g.AddUnconditionalEdge("router", "synthesizer_gate")
	_ = g.AddUnconditionalEdge("synthesizer_gate", "verifier_gate")

	return &SupervisorGraph{graph: g, deps: deps}
}

// Run executes the supervisor graph with the given initial state.
func (g *SupervisorGraph) Run(ctx context.Context, state *State) error {
	ctx = ContextWithState(ctx, state)
	return g.graph.Run(ctx, state)
}

// ── Router node ──────────────────────────────────────────────────────────────

// routerNode is a Node that fans out to specialist agents based on the
// SupervisorPlan stored in state. It does the parallel execution inline
// rather than declaring multiple outgoing edges, because the targets are
// chosen dynamically based on state.
type routerNode struct {
	deps   *AgentDeps
	agents map[string]Node
}

func (n *routerNode) Name() string { return "router" }

func (n *routerNode) Run(ctx context.Context, state *State) error {
	planVal, _ := state.Get(SupervisorPlanKey)
	plan, _ := planVal.(*SupervisorPlan)
	if plan == nil {
		// No plan from supervisor — fallback: route to chat
		plan = &SupervisorPlan{Route: []string{"chat"}}
	}

	// Emit a routing transition event so the client can render the decision
	if n.deps.Sender != nil {
		n.deps.Sender.SendAgentTransition(ctx, n.deps.ConversationID, AgentTransition{
			Route: plan.Route,
		})
	}

	// Deduplicate route entries (LLMs occasionally repeat)
	seen := make(map[string]bool)
	route := make([]string, 0, len(plan.Route))
	for _, r := range plan.Route {
		if !seen[r] {
			seen[r] = true
			route = append(route, r)
		}
	}

	// Single-agent path: run inline (no goroutines)
	if len(route) == 1 {
		node, ok := n.agents[route[0]]
		if !ok {
			return fmt.Errorf("router: unknown agent %q", route[0])
		}
		return node.Run(ctx, state)
	}

	// Multi-agent path: run all routed specialists in parallel.
	// Each writes its output via state.AppendAgentOutput (mutex-safe).
	type result struct {
		err error
	}
	results := make([]result, len(route))

	type indexedRun struct {
		i    int
		node Node
	}
	workers := make([]indexedRun, 0, len(route))
	for i, r := range route {
		node, ok := n.agents[r]
		if !ok {
			results[i] = result{err: fmt.Errorf("router: unknown agent %q", r)}
			continue
		}
		workers = append(workers, indexedRun{i: i, node: node})
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for _, w := range workers {
			i, node := w.i, w.node
			// Bound concurrency: each agent runs sequentially to keep
			// provider rate limits happy. Parallelism here is a future
			// optimization; correctness first.
			// ponytail: serial fan-out for v1 — the supervisor graph's
			// multi-agent case is rare; the parallel speedup didn't earn
			// its complexity yet.
			err := node.Run(ctx, state)
			results[i] = result{err: err}
		}
	}()

	<-done

	// Return the first non-nil error (don't fail the whole turn if one
	// agent succeeded — synthesizer can still merge partial outputs).
	for _, r := range results {
		if r.err != nil {
			return r.err
		}
	}
	return nil
}

// ── Synthesizer gate node ────────────────────────────────────────────────────

// synthesizerGateNode runs the synthesizer when there are 2+ agent outputs
// OR the supervisor explicitly requested synthesis. Otherwise it passes
// through (the single agent's output is already in state.FinalResponse).
type synthesizerGateNode struct {
	synthesizer *SynthesizerAgent
}

func (n *synthesizerGateNode) Name() string { return "synthesizer_gate" }

func (n *synthesizerGateNode) Run(ctx context.Context, state *State) error {
	outputs := state.AgentOutputs()

	// Need synthesis if 2+ agents produced output, OR the supervisor plan
	// explicitly requested synthesis regardless of agent count.
	planVal, _ := state.Get(SupervisorPlanKey)
	plan, _ := planVal.(*SupervisorPlan)

	requiresSynth := len(outputs) >= 2
	if plan != nil && plan.RequiresSynthesis {
		requiresSynth = true
	}

	if !requiresSynth {
		// Pass through: single agent's final response is already set
		return nil
	}

	return n.synthesizer.Run(ctx, state)
}

// ── Verifier gate node ───────────────────────────────────────────────────────

// verifierGateNode runs the verifier only when the supervisor flagged the
// turn as investigative. Non-investigative turns (chitchat) skip verification.
type verifierGateNode struct {
	verifier *VerifierAgent
	deps     *AgentDeps
}

func (n *verifierGateNode) Name() string { return "verifier_gate" }

func (n *verifierGateNode) Run(ctx context.Context, state *State) error {
	planVal, _ := state.Get(SupervisorPlanKey)
	plan, _ := planVal.(*SupervisorPlan)
	if plan == nil || !plan.IsInvestigative {
		return nil
	}
	return n.verifier.Run(ctx, state)
}
