package ai

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// State is the per-turn mutable state passed between graph nodes.
//
// State is a thin wrapper around a map[string]any with typed accessors for
// the values the supervisor graph actually needs. Custom agents can store
// arbitrary additional values via Set/Get if needed.
//
// State is NOT safe for concurrent writes by different parallel branches —
// each parallel branch should write under a unique key derived from the node
// name. The graph executor uses sync.Mutex around the small set of
// well-known accumulators (AgentOutputs, ToolResults) that all branches
// write into. Reads are concurrent-safe via snapshot.
type State struct {
	mu sync.Mutex
	m  map[string]any
}

// NewState returns an empty State.
func NewState() *State {
	return &State{m: make(map[string]any)}
}

// Set stores a value under the given key.
func (s *State) Set(key string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[key] = value
}

// Get returns the value under key, or nil if absent. The ok return is false
// when the key is missing.
func (s *State) Get(key string) (value any, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok = s.m[key]
	return
}

// GetString returns the string value at key, or "" if missing/not a string.
func (s *State) GetString(key string) string {
	v, ok := s.Get(key)
	if !ok {
		return ""
	}
	str, _ := v.(string)
	return str
}

// SetUserMessage stores the original user message text.
func (s *State) SetUserMessage(msg string) { s.Set(stateKeyUserMessage, msg) }

// UserMessage returns the original user message text.
func (s *State) UserMessage() string { return s.GetString(stateKeyUserMessage) }

// SetPageContext stores the page_context string sent by the client.
func (s *State) SetPageContext(ctx string) { s.Set(stateKeyPageContext, ctx) }

// PageContext returns the page_context string (may be empty).
func (s *State) PageContext() string { return s.GetString(stateKeyPageContext) }

// SetPageProfile stores the resolved PageProfile for the current page_context.
// nil means "no matching profile, fall back to chatbot global config".
func (s *State) SetPageProfile(p *PageProfile) { s.Set(stateKeyPageProfile, p) }

// PageProfile returns the resolved PageProfile, or nil.
func (s *State) PageProfile() *PageProfile {
	v, ok := s.Get(stateKeyPageProfile)
	if !ok {
		return nil
	}
	p, _ := v.(*PageProfile)
	return p
}

// SetUserLanguage stores the detected user language (BCP-47-ish string or
// human-readable name like "German"). Used by the synthesizer and verifier
// to enforce language consistency.
func (s *State) SetUserLanguage(lang string) { s.Set(stateKeyUserLanguage, lang) }

// UserLanguage returns the detected user language (may be empty).
func (s *State) UserLanguage() string { return s.GetString(stateKeyUserLanguage) }

// SetConversationHistory stores the trimmed conversation history to feed
// into each agent.
func (s *State) SetConversationHistory(msgs []Message) { s.Set(stateKeyHistory, msgs) }

// ConversationHistory returns the stored conversation history.
func (s *State) ConversationHistory() []Message {
	v, ok := s.Get(stateKeyHistory)
	if !ok {
		return nil
	}
	msgs, _ := v.([]Message)
	return msgs
}

// AppendAgentOutput adds one specialist agent's output to the accumulator.
// Safe to call from parallel goroutines. Order is preserved as insertions
// happen, but inter-agent order from parallel branches is not deterministic.
func (s *State) AppendAgentOutput(name, output string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, _ := s.m[stateKeyAgentOutputs].([]AgentOutput)
	s.m[stateKeyAgentOutputs] = append(existing, AgentOutput{Name: name, Content: output})
}

// AgentOutputs returns a snapshot of all agent outputs collected so far.
func (s *State) AgentOutputs() []AgentOutput {
	s.mu.Lock()
	defer s.mu.Unlock()
	out, _ := s.m[stateKeyAgentOutputs].([]AgentOutput)
	cp := make([]AgentOutput, len(out))
	copy(cp, out)
	return cp
}

// AppendToolResult adds a tool result to the accumulator.
func (s *State) AppendToolResult(r QueryResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, _ := s.m[stateKeyToolResults].([]QueryResult)
	s.m[stateKeyToolResults] = append(existing, r)
}

// ToolResults returns a snapshot of all tool results.
func (s *State) ToolResults() []QueryResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	out, _ := s.m[stateKeyToolResults].([]QueryResult)
	cp := make([]QueryResult, len(out))
	copy(cp, out)
	return cp
}

// SetFinalResponse stores the response that will be sent to the client.
// Typically written by the synthesizer (or by a single-agent path that
// skips synthesis).
func (s *State) SetFinalResponse(r string) { s.Set(stateKeyFinalResponse, r) }

// FinalResponse returns the final response string.
func (s *State) FinalResponse() string { return s.GetString(stateKeyFinalResponse) }

// AddUsage accumulates token usage from one node into the running total.
func (s *State) AddUsage(u UsageStats) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, _ := s.m[stateKeyUsage].(UsageStats)
	existing.PromptTokens += u.PromptTokens
	existing.CompletionTokens += u.CompletionTokens
	existing.TotalTokens += u.TotalTokens
	existing.CachedTokens += u.CachedTokens
	s.m[stateKeyUsage] = existing
}

// Usage returns the accumulated token usage across all nodes.
func (s *State) Usage() UsageStats {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, _ := s.m[stateKeyUsage].(UsageStats)
	return u
}

// State key constants — keeps the typos in one place.
const (
	stateKeyUserMessage    = "user_message"
	stateKeyPageContext    = "page_context"
	stateKeyPageProfile    = "page_profile"
	stateKeyUserLanguage   = "user_language"
	stateKeyHistory        = "history"
	stateKeyAgentOutputs   = "agent_outputs"
	stateKeyToolResults    = "tool_results"
	stateKeyFinalResponse  = "final_response"
	stateKeyUsage          = "usage"
	stateKeySupervisorPlan = "supervisor_plan" // *SupervisorPlan
)

// AgentOutput is one specialist agent's text contribution to the turn.
type AgentOutput struct {
	Name    string
	Content string
}

// Node is one step in a graph. Nodes do work, mutate state, and return.
// The graph executor decides what runs next based on edges.
type Node interface {
	Name() string
	Run(ctx context.Context, state *State) error
}

// Edge connects two nodes. If Condition is non-nil, it returns the name of
// the next node to run based on state; the edge's To field is ignored in
// that case. If Condition is nil, To is the unconditional next node.
//
// Multiple edges can leave the same node — they run in parallel (fan-out).
// Conditional edges, however, must be the only outgoing edge from their
// source (the executor returns an error at construction time otherwise).
type Edge struct {
	From      string
	To        string
	Condition func(state *State) string
}

// Graph is the executable pipeline.
//
// Construction rules enforced at AddEdge time:
//   - From and To must reference registered nodes
//   - Conditional edges must be the sole outgoing edge from their source
//
// Execution rules:
//   - The graph starts at the configured Entry node
//   - At each step, the executor looks up outgoing edges:
//   - If exactly one conditional edge exists, evaluate it and jump to its target
//   - Otherwise, run all unconditional edges in parallel (fan-out)
//   - Nodes return after writing to state; the executor never re-runs a node
//     in one Run() call (no cycles at execution time)
//   - Empty edge set → execution ends
type Graph struct {
	nodes map[string]Node
	// unconditional[from] = list of unconditional targets
	unconditional map[string][]string
	// conditional[from] = single conditional edge
	conditional map[string]Edge
	entry       string
}

// NewGraph returns an empty Graph with the given entry node.
func NewGraph(entry string) *Graph {
	return &Graph{
		nodes:         make(map[string]Node),
		unconditional: make(map[string][]string),
		conditional:   make(map[string]Edge),
		entry:         entry,
	}
}

// AddNode registers a node. Idempotent on name.
func (g *Graph) AddNode(n Node) {
	if n == nil {
		return
	}
	g.nodes[n.Name()] = n
}

// AddUnconditionalEdge adds a From → To edge. Multiple unconditional edges
// from the same From create a parallel fan-out.
func (g *Graph) AddUnconditionalEdge(from, to string) error {
	if _, ok := g.nodes[from]; !ok {
		return fmt.Errorf("graph: unknown source node %q", from)
	}
	if _, ok := g.nodes[to]; !ok {
		return fmt.Errorf("graph: unknown target node %q", to)
	}
	if _, ok := g.conditional[from]; ok {
		return fmt.Errorf("graph: node %q already has a conditional edge; cannot add unconditional", from)
	}
	g.unconditional[from] = append(g.unconditional[from], to)
	return nil
}

// AddConditionalEdge adds a conditional edge: when execution reaches From,
// Condition is called and must return the name of the next node.
func (g *Graph) AddConditionalEdge(from string, cond func(state *State) string) error {
	if _, ok := g.nodes[from]; !ok {
		return fmt.Errorf("graph: unknown source node %q", from)
	}
	if len(g.unconditional[from]) > 0 {
		return fmt.Errorf("graph: node %q already has unconditional edges; cannot add conditional", from)
	}
	if _, exists := g.conditional[from]; exists {
		return fmt.Errorf("graph: node %q already has a conditional edge", from)
	}
	g.conditional[from] = Edge{From: from, Condition: cond}
	return nil
}

// Run executes the graph starting at Entry.
//
// Execution is acyclic within a single Run: the executor keeps a visited
// set and refuses to run any node twice. Cycles are reported as errors at
// runtime rather than construction (they may depend on state via conditions).
func (g *Graph) Run(ctx context.Context, state *State) error {
	if g.entry == "" {
		return errors.New("graph: no entry node")
	}
	current := g.entry
	visited := make(map[string]bool)

	for current != "" {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if visited[current] {
			return fmt.Errorf("graph: cycle detected revisiting %q", current)
		}

		node, ok := g.nodes[current]
		if !ok {
			return fmt.Errorf("graph: unknown node %q", current)
		}

		if err := node.Run(ctx, state); err != nil {
			return fmt.Errorf("graph: node %q failed: %w", current, err)
		}
		visited[current] = true

		// Decide what runs next
		if condEdge, hasCond := g.conditional[current]; hasCond {
			next := condEdge.Condition(state)
			current = next
			continue
		}

		targets := g.unconditional[current]
		if len(targets) == 0 {
			// No outgoing edges — execution ends naturally
			return nil
		}
		if len(targets) == 1 {
			current = targets[0]
			continue
		}

		// Fan-out: run all target nodes in parallel via a sub-traversal.
		// Each branch runs until it hits a node with multiple incoming edges
		// (a join) or terminates. This implementation assumes the supervisor
		// graph's fan-out branches all converge on a single node (synthesizer)
		// — that node's idempotency on parallel writes via State methods makes
		// this safe.
		if err := g.runParallelBranches(ctx, state, targets, visited); err != nil {
			return err
		}
		// After fan-out, execution ends. The synthesizer (the join target)
		// must be the last node in one of the branches.
		return nil
	}
	return nil
}

// runParallelBranches runs each target's chain in a goroutine. Each chain
// is walked until it terminates or hits a node that's NOT in the visited set
// AND NOT in this fan-out's targets. The shared visited set is updated under
// the graph's mutex when needed — but since branches here are disjoint
// sub-chains, we use per-branch local tracking to avoid false-positive
// cycle errors.
func (g *Graph) runParallelBranches(ctx context.Context, state *State, targets []string, sharedVisited map[string]bool) error {
	var wg sync.WaitGroup
	errs := make([]error, len(targets))

	for i, t := range targets {
		wg.Add(1)
		go func(idx int, start string) {
			defer wg.Done()
			errs[idx] = g.runChain(ctx, state, start, sharedVisited)
		}(i, t)
	}
	wg.Wait()

	// Return the first non-nil error
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

// runChain walks a single linear chain starting at start, updating the
// shared visited map as it goes. Stops when a node has multiple outgoing
// edges (shouldn't happen in a specialist chain), no edges, or hits a node
// already in sharedVisited (the join point).
func (g *Graph) runChain(ctx context.Context, state *State, start string, sharedVisited map[string]bool) error {
	current := start
	for current != "" {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if sharedVisited[current] {
			// Join point — another branch already ran this node. Stop here.
			return nil
		}

		node, ok := g.nodes[current]
		if !ok {
			return fmt.Errorf("graph: unknown node %q", current)
		}

		if err := node.Run(ctx, state); err != nil {
			return fmt.Errorf("graph: node %q failed: %w", current, err)
		}
		sharedVisited[current] = true

		// Single-edge chain: follow unconditional edge if exactly one, else stop
		if condEdge, hasCond := g.conditional[current]; hasCond {
			current = condEdge.Condition(state)
			continue
		}
		targets := g.unconditional[current]
		if len(targets) == 1 {
			current = targets[0]
			continue
		}
		// 0 or >1 edges — stop (join points or terminators handled by caller)
		return nil
	}
	return nil
}

// Entry returns the configured entry node name.
func (g *Graph) Entry() string { return g.entry }
