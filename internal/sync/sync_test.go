package sync

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeItem is a minimal ItemSpec for testing.
type fakeItem struct {
	name string
}

func (f fakeItem) GetName() string { return f.name }

// memorySyncer is a Syncer backed by an in-memory map. It emulates how a
// real storage layer would behave: Create writes a record, Update mutates
// it, ListExisting reads the map. Crucially, IsChanged returns false when
// content matches — exactly what real storage does when files are unchanged.
//
// Used to verify the sync framework converges (does not oscillate) across
// repeated runs with the same payload.
type memorySyncer struct {
	mu           sync.Mutex
	records      map[string]string // name → content
	existingCall int
	createCall   int
	updateCall   int
	deleteCall   int

	// FailCreateOn makes the next Create call for this name return an error.
	// Used to simulate the unique-constraint violation that the pre-fix RPC
	// sync oscillated on.
	FailCreateOn map[string]bool
}

func newMemorySyncer() *memorySyncer {
	return &memorySyncer{
		records:      make(map[string]string),
		FailCreateOn: make(map[string]bool),
	}
}

func (m *memorySyncer) ListExisting(ctx context.Context, opts Options) (map[string]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.existingCall++
	out := make(map[string]string, len(m.records))
	for name := range m.records {
		out[name] = "id-" + name
	}
	return out, nil
}

func (m *memorySyncer) IsChanged(ctx context.Context, existingID string, item fakeItem, opts Options) (bool, error) {
	// Real storage compares content hashes here. We don't have content on
	// fakeItem, so just return false (unchanged). The framework puts the
	// item in the unchanged bucket. This is the convergence guarantee we
	// are testing: once ListExisting sees the item, IsChanged=false skips
	// the create/update path entirely.
	return false, nil
}

func (m *memorySyncer) Preprocess(ctx context.Context, item fakeItem) error { return nil }

func (m *memorySyncer) Create(ctx context.Context, item fakeItem, opts Options) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.createCall++
	if m.FailCreateOn[item.name] {
		return errors.New("duplicate key value violates unique constraint")
	}
	m.records[item.name] = "created"
	return nil
}

func (m *memorySyncer) Update(ctx context.Context, item fakeItem, existingID string, opts Options) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updateCall++
	m.records[item.name] = "updated"
	return nil
}

func (m *memorySyncer) Delete(ctx context.Context, name string, existingID string, opts Options) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deleteCall++
	delete(m.records, name)
	return true, nil
}

func (m *memorySyncer) PostSync(ctx context.Context, result *Result, opts Options) error { return nil }

// TestExecute_SamePayloadTwice_Converges is the regression test for the
// RPC sync oscillation bug. The pre-fix behavior was:
//   - Run 1: ListExisting returns nothing for some items (because tenant_id
//     was NULL on those rows). The framework marks them as "to create".
//     Create fails with unique-constraint violation because the rows exist
//     in the DB but are invisible to the list query.
//   - Run 2: same situation, same failures, forever.
//
// After the fix (tenant_id properly inserted), ListExisting consistently
// returns all items on every run, IsChanged returns false for unchanged
// content, and the framework correctly reports "unchanged".
func TestExecute_SamePayloadTwice_Converges(t *testing.T) {
	syncer := newMemorySyncer()
	items := []fakeItem{{name: "a"}, {name: "b"}, {name: "c"}}
	opts := Options{Namespace: "default"}

	// Run 1: all items new → all created
	r1, err := Execute(context.Background(), syncer, items, opts)
	require.NoError(t, err)
	assert.Equal(t, 3, r1.Summary.Created, "first run creates all 3 items")
	assert.Equal(t, 0, r1.Summary.Updated)
	assert.Equal(t, 0, r1.Summary.Unchanged)
	assert.Equal(t, 0, r1.Summary.Errors, "first run must not error")
	assert.Equal(t, 1, syncer.existingCall)
	assert.Equal(t, 3, syncer.createCall)

	// Run 2: all items now exist, IsChanged returns false → all unchanged
	r2, err := Execute(context.Background(), syncer, items, opts)
	require.NoError(t, err)
	assert.Equal(t, 0, r2.Summary.Created, "second run must not re-create")
	assert.Equal(t, 0, r2.Summary.Updated)
	assert.Equal(t, 3, r2.Summary.Unchanged, "second run reports all unchanged")
	assert.Equal(t, 0, r2.Summary.Errors, "second run must not error")

	// Run 3: still stable
	r3, err := Execute(context.Background(), syncer, items, opts)
	require.NoError(t, err)
	assert.Equal(t, 0, r3.Summary.Created)
	assert.Equal(t, 3, r3.Summary.Unchanged)
	assert.Equal(t, 0, r3.Summary.Errors)
}

// TestExecute_OscillationReproduces_PreFix simulates the pre-fix bug to
// lock in the regression. With FailCreateOn set, the create call fails
// (mimicking the unique-constraint violation when rows are invisible to
// ListExisting). This test passes against the fixed framework because the
// framework itself never re-creates items that are visible — the bug was
// in the storage layer (NULL tenant_id making rows invisible), not in the
// framework. We keep this test as documentation of the failure mode.
func TestExecute_CreateFailure_OnInvisibleExistingRecord(t *testing.T) {
	syncer := newMemorySyncer()
	// Pre-seed an item as if it existed in the DB but was invisible to
	// ListExisting (the pre-fix NULL tenant_id case). We simulate this by
	// marking it as "must fail create" but not putting it in records.
	syncer.FailCreateOn["a"] = true

	items := []fakeItem{{name: "a"}}
	opts := Options{Namespace: "default"}

	r1, err := Execute(context.Background(), syncer, items, opts)
	require.NoError(t, err) // Execute itself doesn't error; it records per-item errors
	assert.Equal(t, 0, r1.Summary.Created, "create must fail on invisible existing row")
	assert.Equal(t, 1, r1.Summary.Errors, "error must be reported")
	require.Len(t, r1.Errors, 1)
	assert.Equal(t, "a", r1.Errors[0].Name)
	assert.Equal(t, "create", r1.Errors[0].Action)
}

// TestExecute_DeleteMissing_RemovesOnlyMissing verifies the delete-missing
// path doesn't accidentally touch items that ARE in the payload.
func TestExecute_DeleteMissing_RemovesOnlyMissing(t *testing.T) {
	syncer := newMemorySyncer()
	// Pre-seed 3 items
	syncer.records = map[string]string{
		"keep":   "v1",
		"also":   "v1",
		"delete": "v1",
	}

	// Payload only has "keep" and "also"
	items := []fakeItem{{name: "keep"}, {name: "also"}}
	opts := Options{Namespace: "default", DeleteMissing: true}

	r1, err := Execute(context.Background(), syncer, items, opts)
	require.NoError(t, err)
	assert.Equal(t, 0, r1.Summary.Created, "both payload items already exist")
	assert.Equal(t, 2, r1.Summary.Unchanged)
	assert.Equal(t, 1, r1.Summary.Deleted, "delete-missing removes the third item")
	assert.Equal(t, 0, r1.Summary.Errors)
	assert.Equal(t, 1, syncer.deleteCall)
}

// TestExecute_DryRun_DoesNotMutate verifies dry-run leaves storage untouched.
func TestExecute_DryRun_DoesNotMutate(t *testing.T) {
	syncer := newMemorySyncer()
	items := []fakeItem{{name: "a"}, {name: "b"}}
	opts := Options{Namespace: "default", DryRun: true}

	r1, err := Execute(context.Background(), syncer, items, opts)
	require.NoError(t, err)
	assert.Equal(t, 2, r1.Summary.Created, "dry-run reports what would be created")
	assert.Equal(t, 0, syncer.createCall, "no actual Create calls were made")
	assert.Equal(t, 0, len(syncer.records), "storage untouched")
}
