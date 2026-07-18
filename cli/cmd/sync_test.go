package cmd

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSyncRPCFromDir_SendsAllFilesAsProcedures verifies the CLI reads
// every .sql file in the directory and sends each one to the sync API
// with name + code. Regression guard for the chatbot sync tenant_id bug
// (Bug 1) — the CLI side of the contract is "send name + code for each
// file"; the server side is responsible for tenant_id assignment.
func TestSyncRPCFromDir_SendsAllFilesAsProcedures(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.sql"), []byte("SELECT 1;"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b.sql"), []byte("SELECT 2;"), 0o644))
	// Non-SQL files are ignored
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("docs"), 0o644))

	var capturedBody map[string]interface{}
	_, _, cleanup := setupTestEnvWithHandler(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "/api/v1/admin/rpc/sync")
		_ = json.NewDecoder(r.Body).Decode(&capturedBody)
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"summary": map[string]interface{}{
				"created": 2, "updated": 0, "deleted": 0, "unchanged": 0, "errors": 0,
			},
			"details": map[string]interface{}{
				"created":   []string{"a", "b"},
				"updated":   []string{},
				"deleted":   []string{},
				"unchanged": []string{},
			},
		})
	})
	defer cleanup()

	require.NoError(t, syncRPCFromDir(context.Background(), dir, "test", false, false))

	// Verify payload structure
	require.Equal(t, "test", capturedBody["namespace"])
	procs, ok := capturedBody["procedures"].([]interface{})
	require.True(t, ok)
	require.Len(t, procs, 2)

	names := map[string]bool{}
	for _, p := range procs {
		m := p.(map[string]interface{})
		names[m["name"].(string)] = true
		assert.NotEmpty(t, m["code"])
	}
	assert.True(t, names["a"])
	assert.True(t, names["b"])
}

// TestSyncChatbotsFromDir_FlatFile_SendsNameAndCode verifies the chatbot
// sync reads flat .ts files and sends them to the API. Locks in the CLI
// side of the tenant_id fix: the CLI sends name+code, the server attaches
// tenant_id from auth context.
func TestSyncChatbotsFromDir_FlatFile_SendsNameAndCode(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "support-bot.ts"),
		[]byte("export default `You are helpful.`;"), 0o644))

	var capturedBody map[string]interface{}
	_, _, cleanup := setupTestEnvWithHandler(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "/api/v1/admin/ai/chatbots/sync")
		_ = json.NewDecoder(r.Body).Decode(&capturedBody)
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"summary": map[string]interface{}{
				"created": 1, "updated": 0, "deleted": 0, "unchanged": 0, "errors": 0,
			},
			"details": map[string]interface{}{
				"created": []string{"support-bot"},
			},
		})
	})
	defer cleanup()

	require.NoError(t, syncChatbotsFromDir(context.Background(), dir, "test", false, false))

	require.Equal(t, "test", capturedBody["namespace"])
	cbs, ok := capturedBody["chatbots"].([]interface{})
	require.True(t, ok)
	require.Len(t, cbs, 1)
	cb := cbs[0].(map[string]interface{})
	assert.Equal(t, "support-bot", cb["name"])
	assert.NotEmpty(t, cb["code"])
}

// TestSyncRPCFromDir_Convergence_RepeatedRunsDoNotOscillate is the
// regression test for Bug 2 (RPC sync oscillation). It runs the CLI sync
// path twice against a stateful fake server that emulates the
// post-fix storage behavior: items created in run 1 are visible in
// ListExisting on run 2, so IsChanged=false → Unchanged.
//
// Pre-fix, the server's ListProceduresForSync filtered out rows with NULL
// tenant_id, making previously-created procedures invisible. The CLI saw
// "not in existing" and tried to Create them again, hitting the unique
// constraint. This test passes against the fixed server because items
// stay visible across runs.
func TestSyncRPCFromDir_Convergence_RepeatedRunsDoNotOscillate(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.sql"), []byte("SELECT 1;"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b.sql"), []byte("SELECT 2;"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "c.sql"), []byte("SELECT 3;"), 0o644))

	// Stateful fake server: tracks which procedure names have been "created".
	// Each call: list existing, then accept the upsert/sync.
	var serverStateMu sync.Mutex
	serverState := map[string]bool{}

	var allSummaries []map[string]int
	_, _, cleanup := setupTestEnvWithHandler(t, func(w http.ResponseWriter, r *http.Request) {
		// The CLI sends ALL files in one POST. The server (this fake)
		// decides per-file what to do based on prior state.
		var body struct {
			Namespace  string `json:"namespace"`
			Procedures []struct {
				Name string `json:"name"`
				Code string `json:"code"`
			} `json:"procedures"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)

		serverStateMu.Lock()
		defer serverStateMu.Unlock()

		created, unchanged := 0, 0
		var createdNames, unchangedNames []string
		for _, p := range body.Procedures {
			if serverState[p.Name] {
				// Already exists → unchanged (post-fix visibility)
				unchanged++
				unchangedNames = append(unchangedNames, p.Name)
			} else {
				// New → create succeeds (post-fix: tenant_id properly set)
				serverState[p.Name] = true
				created++
				createdNames = append(createdNames, p.Name)
			}
		}

		summary := map[string]int{
			"created":   created,
			"updated":   0,
			"deleted":   0,
			"unchanged": unchanged,
			"errors":    0,
		}
		allSummaries = append(allSummaries, summary)

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"summary": summary,
			"details": map[string]interface{}{
				"created":   createdNames,
				"unchanged": unchangedNames,
			},
		})
	})
	defer cleanup()

	ctx := context.Background()
	// Run 1: all new → 3 created, 0 unchanged, 0 errors
	require.NoError(t, syncRPCFromDir(ctx, dir, "test", false, false))
	require.NotEmpty(t, allSummaries)
	last := allSummaries[len(allSummaries)-1]
	assert.Equal(t, 3, last["created"], "run 1: 3 procedures created")
	assert.Equal(t, 0, last["unchanged"])
	assert.Equal(t, 0, last["errors"], "run 1 must have zero errors")

	// Run 2: same files → 0 created, 3 unchanged, 0 errors (convergence)
	require.NoError(t, syncRPCFromDir(ctx, dir, "test", false, false))
	last = allSummaries[len(allSummaries)-1]
	assert.Equal(t, 0, last["created"], "run 2: must not re-create (would oscillate pre-fix)")
	assert.Equal(t, 3, last["unchanged"], "run 2: all 3 are unchanged")
	assert.Equal(t, 0, last["errors"], "run 2: must have zero errors")

	// Run 3: still stable
	require.NoError(t, syncRPCFromDir(ctx, dir, "test", false, false))
	last = allSummaries[len(allSummaries)-1]
	assert.Equal(t, 0, last["created"])
	assert.Equal(t, 3, last["unchanged"])
	assert.Equal(t, 0, last["errors"])
}

// TestPrintSyncSummary_DebugPrintsPerFileDecisions verifies Bug 3 fix:
// when --debug is active, printSyncSummary emits one DEBUG line per file
// per decision (created/updated/deleted/unchanged).
func TestPrintSyncSummary_DebugPrintsPerFileDecisions(t *testing.T) {
	prevDebug := debug
	debug = true
	defer func() { debug = prevDebug }()

	// printSyncSummary writes to stdout via fmt.Printf, so capture stdout.
	origStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w
	defer func() { os.Stdout = origStdout }()

	result := map[string]interface{}{
		"summary": map[string]interface{}{
			"created": 2, "updated": 1, "deleted": 1, "unchanged": 3, "errors": 0,
		},
		"details": map[string]interface{}{
			"created":   []interface{}{"a", "b"},
			"updated":   []interface{}{"c"},
			"deleted":   []interface{}{"d"},
			"unchanged": []interface{}{"e", "f", "g"},
		},
	}

	printSyncSummary(result, "procedures")
	_ = w.Close()

	out, err := io.ReadAll(r)
	require.NoError(t, err)
	output := string(out)

	// ponytail: assert.Contains keeps the test resilient to format tweaks
	assert.Contains(t, output, "DEBUG: a → decision: create")
	assert.Contains(t, output, "DEBUG: b → decision: create")
	assert.Contains(t, output, "DEBUG: c → decision: update")
	assert.Contains(t, output, "DEBUG: d → decision: delete")
	assert.Contains(t, output, "DEBUG: e → decision: unchanged")
}

// TestPrintSyncSummary_NoDebug_OmitsPerFileDecisions verifies the debug
// output is suppressed when --debug is not set.
func TestPrintSyncSummary_NoDebug_OmitsPerFileDecisions(t *testing.T) {
	prevDebug := debug
	debug = false
	defer func() { debug = prevDebug }()

	origStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w
	defer func() { os.Stdout = origStdout }()

	result := map[string]interface{}{
		"summary": map[string]interface{}{
			"created": 1, "updated": 0, "deleted": 0, "unchanged": 0, "errors": 0,
		},
		"details": map[string]interface{}{
			"created": []interface{}{"a"},
		},
	}

	printSyncSummary(result, "procedures")
	_ = w.Close()

	out, err := io.ReadAll(r)
	require.NoError(t, err)
	output := string(out)
	assert.NotContains(t, output, "DEBUG")
}
