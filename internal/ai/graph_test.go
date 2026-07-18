package ai

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordingNode records the order in which its Run method was called.
type recordingNode struct {
	name   string
	log    *[]string
	mu     *sync.Mutex
	runErr error
}

func newRecordingNode(name string, log *[]string, mu *sync.Mutex) *recordingNode {
	return &recordingNode{name: name, log: log, mu: mu}
}

func (n *recordingNode) Name() string { return n.name }

func (n *recordingNode) Run(ctx context.Context, state *State) error {
	n.mu.Lock()
	*n.log = append(*n.log, n.name)
	n.mu.Unlock()
	if n.runErr != nil {
		return n.runErr
	}
	return nil
}

func TestGraph_LinearExecution_RunsAllNodes(t *testing.T) {
	var log []string
	var mu sync.Mutex

	g := NewGraph("a")
	g.AddNode(newRecordingNode("a", &log, &mu))
	g.AddNode(newRecordingNode("b", &log, &mu))
	g.AddNode(newRecordingNode("c", &log, &mu))
	require.NoError(t, g.AddUnconditionalEdge("a", "b"))
	require.NoError(t, g.AddUnconditionalEdge("b", "c"))

	err := g.Run(context.Background(), NewState())
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, log)
}

func TestGraph_UnknownEntry_ReturnsError(t *testing.T) {
	g := NewGraph("nonexistent")
	err := g.Run(context.Background(), NewState())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown node")
}

func TestGraph_NodeFailure_PropagatesError(t *testing.T) {
	var log []string
	var mu sync.Mutex

	g := NewGraph("a")
	a := newRecordingNode("a", &log, &mu)
	a.runErr = errors.New("boom")
	g.AddNode(a)
	g.AddNode(newRecordingNode("b", &log, &mu))
	require.NoError(t, g.AddUnconditionalEdge("a", "b"))

	err := g.Run(context.Background(), NewState())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "node \"a\" failed")
	assert.Contains(t, err.Error(), "boom")
	// b should not have run
	assert.Equal(t, []string{"a"}, log)
}

func TestGraph_ConditionalEdge_RoutesBasedOnState(t *testing.T) {
	var log []string
	var mu sync.Mutex

	g := NewGraph("start")
	g.AddNode(newRecordingNode("start", &log, &mu))
	g.AddNode(newRecordingNode("path_a", &log, &mu))
	g.AddNode(newRecordingNode("path_b", &log, &mu))

	// Conditional: if state has key "go_a" → path_a, else path_b
	require.NoError(t, g.AddConditionalEdge("start", func(s *State) string {
		if _, ok := s.Get("go_a"); ok {
			return "path_a"
		}
		return "path_b"
	}))

	t.Run("routes to path_a when key set", func(t *testing.T) {
		log = nil
		s := NewState()
		s.Set("go_a", true)
		require.NoError(t, g.Run(context.Background(), s))
		assert.Equal(t, []string{"start", "path_a"}, log)
	})

	t.Run("routes to path_b when key absent", func(t *testing.T) {
		log = nil
		require.NoError(t, g.Run(context.Background(), NewState()))
		assert.Equal(t, []string{"start", "path_b"}, log)
	})
}

func TestGraph_AddUnconditionalEdge_RejectsUnknownNodes(t *testing.T) {
	g := NewGraph("a")
	g.AddNode(&recordingNode{name: "a"})
	err := g.AddUnconditionalEdge("a", "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown target node")
}

func TestGraph_AddConditionalEdge_RejectsMixedEdges(t *testing.T) {
	g := NewGraph("a")
	g.AddNode(&recordingNode{name: "a"})
	g.AddNode(&recordingNode{name: "b"})

	require.NoError(t, g.AddUnconditionalEdge("a", "b"))
	err := g.AddConditionalEdge("a", func(s *State) string { return "b" })
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already has unconditional")
}

func TestState_AppendAgentOutput_IsMutexSafe(t *testing.T) {
	s := NewState()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s.AppendAgentOutput("agent", "output")
		}(i)
	}
	wg.Wait()
	outputs := s.AgentOutputs()
	assert.Len(t, outputs, 100)
}

func TestState_TypedAccessors_RoundTrip(t *testing.T) {
	s := NewState()
	s.SetUserMessage("hello")
	s.SetUserLanguage("German")
	s.SetPageContext("orders")
	s.SetPageProfile(&PageProfile{Page: "orders"})
	s.SetFinalResponse("answer")

	assert.Equal(t, "hello", s.UserMessage())
	assert.Equal(t, "German", s.UserLanguage())
	assert.Equal(t, "orders", s.PageContext())
	require.NotNil(t, s.PageProfile())
	assert.Equal(t, "orders", s.PageProfile().Page)
	assert.Equal(t, "answer", s.FinalResponse())
}

func TestState_AddUsage_Accumulates(t *testing.T) {
	s := NewState()
	s.AddUsage(UsageStats{PromptTokens: 10, CompletionTokens: 5, CachedTokens: 2})
	s.AddUsage(UsageStats{PromptTokens: 20, CompletionTokens: 7, CachedTokens: 3})

	u := s.Usage()
	assert.Equal(t, 30, u.PromptTokens)
	assert.Equal(t, 12, u.CompletionTokens)
	assert.Equal(t, 5, u.CachedTokens)
}
