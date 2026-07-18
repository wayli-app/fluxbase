package ai

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// panickingNode is a Node whose Run panics. Used to verify the routerNode's
// goroutine recovery converts panics into graph errors instead of crashing
// the server.
type panickingNode struct {
	name string
}

func (n *panickingNode) Name() string { return n.name }

func (n *panickingNode) Run(_ context.Context, _ *State) error {
	panic("simulated nil-deref — production stack trace had this in invoke.go:296")
}

// errorNode returns a clean error. Used alongside panickingNode to verify
// recovery doesn't suppress normal errors.
type errorNode struct {
	name string
	err  error
}

func (n *errorNode) Name() string { return n.name }

func (n *errorNode) Run(_ context.Context, _ *State) error { return n.err }

// safeRecordingNode records that it ran. Used to verify sibling agents
// still execute after a panic in one agent.
type safeRecordingNode struct {
	name string
	log  *[]string
	mu   *sync.Mutex
}

func (n *safeRecordingNode) Name() string { return n.name }

func (n *safeRecordingNode) Run(_ context.Context, _ *State) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	*n.log = append(*n.log, n.name)
	return nil
}

// TestRouterNode_AgentPanic_RecoveredAndSurfaced is the regression test
// for the production server crash at invoke.go:296 (nil proc deref) and
// the broader "any agent panic kills the server" class of bugs.
//
// Before Layer 2's recovery, a panic in any specialist goroutine
// propagated up and crashed the process. After recovery, the panic is
// converted to an error that the supervisor graph surfaces to the chat
// handler, which falls back to the legacy ReAct loop.
//
// This test pins the recovery behavior so a future refactor that removes
// the defer/recover (or moves the goroutine launch) can't silently
// re-introduce the crash.
func TestRouterNode_AgentPanic_RecoveredAndSurfaced(t *testing.T) {
	deps := &AgentDeps{Chatbot: &Chatbot{Name: "test"}}
	router := &routerNode{
		deps: deps,
		agents: map[string]Node{
			"panic": &panickingNode{name: "panic"},
		},
	}

	state := NewState()
	state.Set(SupervisorPlanKey, &SupervisorPlan{Route: []string{"panic"}})

	err := router.Run(context.Background(), state)
	require.Error(t, err, "router must surface recovered panic as an error")
	assert.Contains(t, err.Error(), "panicked",
		"error message must indicate a panic so it's diagnosable in logs")
	assert.Contains(t, err.Error(), "panic",
		"error message must name the agent that panicked")
}

// TestRouterNode_PanicInOneAgent_OthersStillRun verifies sibling agents
// execute even when one agent in the route panics. Important for
// multi-agent routes — partial results are better than no results.
func TestRouterNode_PanicInOneAgent_OthersStillRun(t *testing.T) {
	var log []string
	var mu sync.Mutex

	deps := &AgentDeps{Chatbot: &Chatbot{Name: "test"}}
	router := &routerNode{
		deps: deps,
		agents: map[string]Node{
			"panic": &panickingNode{name: "panic"},
			"safe1": &safeRecordingNode{name: "safe1", log: &log, mu: &mu},
			"safe2": &safeRecordingNode{name: "safe2", log: &log, mu: &mu},
		},
	}

	state := NewState()
	state.Set(SupervisorPlanKey, &SupervisorPlan{Route: []string{"safe1", "panic", "safe2"}})

	err := router.Run(context.Background(), state)
	require.Error(t, err, "router must report the panic from one agent")
	// Sibling agents must still have run — partial results matter.
	assert.Contains(t, log, "safe1", "safe1 must run before the panic")
	assert.Contains(t, log, "safe2", "safe2 must run after the panic (recovery doesn't stop siblings)")
}

// TestRouterNode_NormalError_NotWrappedAsPanic verifies the recovery path
// only fires on actual panics. Normal errors should pass through cleanly
// without the "panicked" wrapper text.
func TestRouterNode_NormalError_NotWrappedAsPanic(t *testing.T) {
	deps := &AgentDeps{Chatbot: &Chatbot{Name: "test"}}
	myErr := errors.New("provider rate limited")
	router := &routerNode{
		deps: deps,
		agents: map[string]Node{
			"sql": &errorNode{name: "sql", err: myErr},
		},
	}

	state := NewState()
	state.Set(SupervisorPlanKey, &SupervisorPlan{Route: []string{"sql"}})

	err := router.Run(context.Background(), state)
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "panicked",
		"normal errors must not be wrapped as panics")
	assert.Contains(t, err.Error(), "provider rate limited",
		"original error message must be preserved")
}
