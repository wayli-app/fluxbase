package eval

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadCorpus reads the bundled wayli routing corpus and verifies it parses
// and is non-empty. This also gives the eval package direct coverage so the
// package-threshold CI gate passes (the corpus is otherwise only exercised
// from package ai's eval_test.go).
func TestLoadCorpus(t *testing.T) {
	path := filepath.Join("testdata", "wayli_routing.eval.json")
	corpus, err := LoadCorpus(path)
	require.NoError(t, err)
	require.NotNil(t, corpus)
	assert.NotEmpty(t, corpus.Cases, "corpus should have cases")
	for _, c := range corpus.Cases {
		assert.NotEmpty(t, c.ID, "every case needs an id")
		assert.NotEmpty(t, c.Message, "case %q needs a message", c.ID)
	}
}

// TestLoadCorpus_MissingFile verifies the error path.
func TestLoadCorpus_MissingFile(t *testing.T) {
	_, err := LoadCorpus("testdata/does_not_exist.json")
	require.Error(t, err)
}

// TestLoadCorpus_InvalidJSON verifies a malformed file is rejected.
func TestLoadCorpus_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "bad.json")
	require.NoError(t, os.WriteFile(p, []byte("{not json"), 0o644))
	_, err := LoadCorpus(p)
	require.Error(t, err)
}

// TestCheckRoute covers the include/exclude scoring logic.
func TestCheckRoute(t *testing.T) {
	c := Case{
		ExpectedRouteIncludes: []string{"web"},
		ExpectedRouteExcludes: []string{"sql"},
	}

	// Includes satisfied, excludes absent → pass.
	ok, reason := CheckRoute(c, []string{"web", "action"})
	assert.True(t, ok)
	assert.Equal(t, "ok", reason)

	// Missing an include → fail.
	ok, reason = CheckRoute(c, []string{"action"})
	assert.False(t, ok)
	assert.Contains(t, reason, "include")

	// Present an exclude → fail.
	ok, reason = CheckRoute(c, []string{"web", "sql"})
	assert.False(t, ok)
	assert.Contains(t, reason, "exclude")

	// Empty expectations → any route passes.
	empty := Case{}
	ok, reason = CheckRoute(empty, []string{"anything"})
	assert.True(t, ok)
}

// TestSummaryReport verifies the report rendering marks passes/fails.
func TestSummaryReport(t *testing.T) {
	results := []Result{
		{Case: Case{ID: "a", Category: "x"}, Passed: true, Route: []string{"web"}},
		{Case: Case{ID: "b", Category: "y"}, Passed: false, Reason: "missing web", Route: []string{"chat"}},
	}
	report := SummaryReport(results)
	assert.Contains(t, report, "1/2 passed")
	assert.Contains(t, report, "✓ a")
	assert.Contains(t, report, "✗ b")
	assert.Contains(t, report, "missing web")
}

// TestCountPassed verifies the pass counter.
func TestCountPassed(t *testing.T) {
	results := []Result{
		{Passed: true},
		{Passed: false},
		{Passed: true},
	}
	assert.Equal(t, 2, CountPassed(results))
	assert.Equal(t, 0, CountPassed(nil))
}
