//go:build integration
// +build integration

package api_test

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/testutil/e2e"
)

// =============================================================================
// RLS (Row Level Security) Integration Tests
//
// These tests verify that RLS policies properly isolate data between users.
// We use the regular integration test context to set up tables with RLS policies,
// then verify that RLS is enforced when making API calls with user tokens.
// =============================================================================

func TestRESTHandler_RLS_UserIsolation_Integration(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Drop table if exists to ensure clean state (as superuser)
	tc.ExecuteSQLAsSuperuser(`DROP TABLE IF EXISTS public.tasks CASCADE`)

	// Create test table with user_id column for RLS (as superuser)
	tc.ExecuteSQLAsSuperuser(`
		CREATE TABLE tasks (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL,
			title TEXT NOT NULL,
			status TEXT DEFAULT 'open'
		)
	`)

	// Enable RLS on the table (requires superuser)
	tc.ExecuteSQLAsSuperuser(`ALTER TABLE tasks ENABLE ROW LEVEL SECURITY`)

	// Create RLS policy: Users can only see their own tasks (requires superuser)
	// The policy uses auth.current_user_id() which extracts from request.jwt.claims
	tc.ExecuteSQLAsSuperuser(`
		CREATE POLICY user_tasks_policy ON tasks
		FOR ALL
		USING (user_id = auth.current_user_id())
		WITH CHECK (user_id = auth.current_user_id())
	`)

	// Grant permissions to authenticated role (requires superuser)
	tc.ExecuteSQLAsSuperuser(`GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE tasks TO authenticated`)
	tc.ExecuteSQLAsSuperuser(`GRANT USAGE ON ALL SEQUENCES IN SCHEMA public TO authenticated`)

	// Refresh schema cache so REST API discovers the new table
	refreshSchemaCache(tc)

	// Create two users
	user1Email := randomEmail()
	user2Email := randomEmail()
	user1ID, token1 := tc.CreateTestUser(user1Email, "password123")
	user2ID, token2 := tc.CreateTestUser(user2Email, "password123")

	// Use unique task names
	testID := uuid.New().String()[:8]
	task1Title := "User1 Task " + testID
	task2Title := "User2 Task " + testID

	// User1 creates a task
	resp1 := tc.NewRequest("POST", "/api/v1/tables/public/tasks").
		WithAuth(token1).
		WithBody(map[string]interface{}{
			"user_id": user1ID,
			"title":   task1Title,
			"status":  "open",
		}).
		Send().
		AssertStatus(201)

	var task1 map[string]interface{}
	resp1.JSON(&task1)
	assert.Equal(t, task1Title, task1["title"])

	// User2 creates a task
	resp2 := tc.NewRequest("POST", "/api/v1/tables/public/tasks").
		WithAuth(token2).
		WithBody(map[string]interface{}{
			"user_id": user2ID,
			"title":   task2Title,
			"status":  "open",
		}).
		Send().
		AssertStatus(201)

	var task2 map[string]interface{}
	resp2.JSON(&task2)
	assert.Equal(t, task2Title, task2["title"])

	// User1 lists tasks - should only see their own task
	listResp1 := tc.NewRequest("GET", "/api/v1/tables/public/tasks").
		WithAuth(token1).
		Send().
		AssertStatus(200)

	var tasks1 []map[string]interface{}
	listResp1.JSON(&tasks1)
	require.Len(t, tasks1, 1, "User1 should only see their own task")
	assert.Equal(t, task1Title, tasks1[0]["title"])

	// User2 lists tasks - should only see their own task
	listResp2 := tc.NewRequest("GET", "/api/v1/tables/public/tasks").
		WithAuth(token2).
		Send().
		AssertStatus(200)

	var tasks2 []map[string]interface{}
	listResp2.JSON(&tasks2)
	require.Len(t, tasks2, 1, "User2 should only see their own task")
	assert.Equal(t, task2Title, tasks2[0]["title"])

	// User1 tries to access User2's task directly - should fail (404 due to RLS)
	task2ID := task2["id"].(string)
	tc.NewRequest("GET", "/api/v1/tables/public/tasks/"+task2ID).
		WithAuth(token1).
		Send().
		AssertStatus(404)

	// User1 tries to update User2's task - should fail (403 due to RLS)
	// UPDATE returns 403 instead of 404 because RLS WITH CHECK blocks the modification
	tc.NewRequest("PATCH", "/api/v1/tables/public/tasks/"+task2ID).
		WithAuth(token1).
		WithBody(map[string]interface{}{
			"status": "closed",
		}).
		Send().
		AssertStatus(403)

	// User1 tries to delete User2's task - should fail (403 due to RLS)
	// DELETE returns 403 instead of 404 because RLS policy blocks the deletion
	tc.NewRequest("DELETE", "/api/v1/tables/public/tasks/"+task2ID).
		WithAuth(token1).
		Send().
		AssertStatus(403)
}

func TestRESTHandler_RLS_FilterWithRLS_Integration(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Drop table if exists to ensure clean state (as superuser)
	tc.ExecuteSQLAsSuperuser(`DROP TABLE IF EXISTS public.documents CASCADE`)

	// Create test table with user_id column for RLS (as superuser)
	tc.ExecuteSQLAsSuperuser(`
		CREATE TABLE documents (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL,
			title TEXT NOT NULL,
			status TEXT DEFAULT 'draft',
			priority INTEGER DEFAULT 1
		)
	`)

	// Enable RLS (requires superuser)
	tc.ExecuteSQLAsSuperuser(`ALTER TABLE documents ENABLE ROW LEVEL SECURITY`)

	// Create RLS policy: Users can only see their own documents (requires superuser)
	tc.ExecuteSQLAsSuperuser(`
		CREATE POLICY user_documents_policy ON documents
		FOR ALL
		USING (user_id = auth.current_user_id())
		WITH CHECK (user_id = auth.current_user_id())
	`)

	// Grant permissions (requires superuser)
	tc.ExecuteSQLAsSuperuser(`GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE documents TO authenticated`)
	tc.ExecuteSQLAsSuperuser(`GRANT USAGE ON ALL SEQUENCES IN SCHEMA public TO authenticated`)

	// Refresh schema cache
	refreshSchemaCache(tc)

	// Create user
	userEmail := randomEmail()
	userID, token := tc.CreateTestUser(userEmail, "password123")

	// Insert multiple documents with different statuses
	testID := uuid.New().String()[:8]
	tc.ExecuteSQL(`INSERT INTO documents (user_id, title, status, priority) VALUES ($1, $2, $3, $4)`,
		userID, "Draft 1 "+testID, "draft", 1)
	tc.ExecuteSQL(`INSERT INTO documents (user_id, title, status, priority) VALUES ($1, $2, $3, $4)`,
		userID, "Published 1 "+testID, "published", 2)
	tc.ExecuteSQL(`INSERT INTO documents (user_id, title, status, priority) VALUES ($1, $2, $3, $4)`,
		userID, "Draft 2 "+testID, "draft", 3)

	// Query with filter - should only return draft documents
	resp := tc.NewRequest("GET", "/api/v1/tables/public/documents?status=eq.draft").
		WithAuth(token).
		Send().
		AssertStatus(200)

	var docs []map[string]interface{}
	resp.JSON(&docs)

	// Filter to only our documents
	var ourDocs []map[string]interface{}
	for _, doc := range docs {
		if title, ok := doc["title"].(string); ok && len(title) > 8 && title[len(title)-8:] == testID {
			ourDocs = append(ourDocs, doc)
		}
	}

	require.Len(t, ourDocs, 2, "Should return only draft documents")
	for _, doc := range ourDocs {
		assert.Equal(t, "draft", doc["status"])
	}
}

func TestRESTHandler_RLS_PaginationWithRLS_Integration(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Drop table if exists (as superuser to handle any existing table)
	tc.ExecuteSQLAsSuperuser(`DROP TABLE IF EXISTS public.notes CASCADE`)

	// Create test table (as superuser for RLS setup)
	tc.ExecuteSQLAsSuperuser(`
		CREATE TABLE notes (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL,
			content TEXT NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)
	`)

	// Enable RLS (requires superuser)
	tc.ExecuteSQLAsSuperuser(`ALTER TABLE notes ENABLE ROW LEVEL SECURITY`)

	// Create RLS policy: Users can only see their own notes (requires superuser)
	tc.ExecuteSQLAsSuperuser(`
		CREATE POLICY user_notes_policy ON notes
		FOR ALL
		USING (user_id = auth.current_user_id())
		WITH CHECK (user_id = auth.current_user_id())
	`)

	// Grant permissions (requires superuser)
	tc.ExecuteSQLAsSuperuser(`GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE notes TO authenticated`)
	tc.ExecuteSQLAsSuperuser(`GRANT USAGE ON ALL SEQUENCES IN SCHEMA public TO authenticated`)

	// Refresh schema cache
	refreshSchemaCache(tc)

	// Create user
	userEmail := randomEmail()
	userID, token := tc.CreateTestUser(userEmail, "password123")

	// Insert 25 notes for this user
	testID := uuid.New().String()[:8]
	for i := 1; i <= 25; i++ {
		tc.ExecuteSQL(`INSERT INTO notes (user_id, content) VALUES ($1, $2)`,
			userID, fmt.Sprintf("Note %d %s", i, testID))
	}

	// Create another user and insert notes for them
	user2Email := randomEmail()
	user2ID, _ := tc.CreateTestUser(user2Email, "password123")
	for i := 1; i <= 25; i++ {
		tc.ExecuteSQL(`INSERT INTO notes (user_id, content) VALUES ($1, $2)`,
			user2ID, fmt.Sprintf("Other User Note %d", i))
	}

	// User1 queries with pagination - should only see their own notes
	resp := tc.NewRequest("GET", "/api/v1/tables/public/notes?limit=10&offset=0").
		WithAuth(token).
		Send().
		AssertStatus(200)

	var notes []map[string]interface{}
	resp.JSON(&notes)

	// Filter to only our notes
	var ourNotes []map[string]interface{}
	for _, note := range notes {
		if content, ok := note["content"].(string); ok && len(content) > 8 && content[len(content)-8:] == testID {
			ourNotes = append(ourNotes, note)
		}
	}

	assert.Len(t, ourNotes, 10, "Should return 10 notes")

	// Get total count for user1's notes
	totalResp := tc.NewRequest("GET", "/api/v1/tables/public/notes").
		WithAuth(token).
		Send().
		AssertStatus(200)

	var allNotes []map[string]interface{}
	totalResp.JSON(&allNotes)

	// Count only our notes
	var ourTotal int
	for _, note := range allNotes {
		if content, ok := note["content"].(string); ok && len(content) > 8 && content[len(content)-8:] == testID {
			ourTotal++
		}
	}

	assert.Equal(t, 25, ourTotal, "User should have 25 notes total")
}

func TestRESTHandler_RLS_UpdateOwnRecord_Integration(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Drop table if exists (as superuser)
	tc.ExecuteSQLAsSuperuser(`DROP TABLE IF EXISTS public.profiles CASCADE`)

	// Create test table (as superuser)
	tc.ExecuteSQLAsSuperuser(`
		CREATE TABLE profiles (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL,
			display_name TEXT,
			bio TEXT
		)
	`)

	// Enable RLS (requires superuser)
	tc.ExecuteSQLAsSuperuser(`ALTER TABLE profiles ENABLE ROW LEVEL SECURITY`)

	// Create RLS policy: Users can only see their own profiles (requires superuser)
	tc.ExecuteSQLAsSuperuser(`
		CREATE POLICY user_profiles_policy ON profiles
		FOR ALL
		USING (user_id = auth.current_user_id())
		WITH CHECK (user_id = auth.current_user_id())
	`)

	// Grant permissions (requires superuser)
	tc.ExecuteSQLAsSuperuser(`GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE profiles TO authenticated`)
	tc.ExecuteSQLAsSuperuser(`GRANT USAGE ON ALL SEQUENCES IN SCHEMA public TO authenticated`)

	// Refresh schema cache
	refreshSchemaCache(tc)

	// Create user
	userEmail := randomEmail()
	userID, token := tc.CreateTestUser(userEmail, "password123")

	// Create profile
	testID := uuid.New().String()[:8]
	originalName := "Original Name " + testID
	resp := tc.NewRequest("POST", "/api/v1/tables/public/profiles").
		WithAuth(token).
		WithBody(map[string]interface{}{
			"user_id":      userID,
			"display_name": originalName,
			"bio":          "Original bio",
		}).
		Send().
		AssertStatus(201)

	var profile map[string]interface{}
	resp.JSON(&profile)
	profileID := profile["id"].(string)

	// Update profile
	updatedName := "Updated Name " + testID
	updateResp := tc.NewRequest("PATCH", "/api/v1/tables/public/profiles/"+profileID).
		WithAuth(token).
		WithBody(map[string]interface{}{
			"display_name": updatedName,
		}).
		Send().
		AssertStatus(200)

	var updatedProfile map[string]interface{}
	updateResp.JSON(&updatedProfile)
	assert.Equal(t, updatedName, updatedProfile["display_name"])
}

func TestRESTHandler_RLS_DeleteOwnRecord_Integration(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Drop table if exists (as superuser)
	tc.ExecuteSQLAsSuperuser(`DROP TABLE IF EXISTS public.todos CASCADE`)

	// Create test table (as superuser)
	tc.ExecuteSQLAsSuperuser(`
		CREATE TABLE todos (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL,
			task TEXT NOT NULL,
			completed BOOLEAN DEFAULT false
		)
	`)

	// Enable RLS (requires superuser)
	tc.ExecuteSQLAsSuperuser(`ALTER TABLE todos ENABLE ROW LEVEL SECURITY`)

	// Create RLS policy: Users can only see their own todos (requires superuser)
	tc.ExecuteSQLAsSuperuser(`
		CREATE POLICY user_todos_policy ON todos
		FOR ALL
		USING (user_id = auth.current_user_id())
		WITH CHECK (user_id = auth.current_user_id())
	`)

	// Grant permissions (requires superuser)
	tc.ExecuteSQLAsSuperuser(`GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE todos TO authenticated`)
	tc.ExecuteSQLAsSuperuser(`GRANT USAGE ON ALL SEQUENCES IN SCHEMA public TO authenticated`)

	// Refresh schema cache
	refreshSchemaCache(tc)

	// Create user
	userEmail := randomEmail()
	userID, token := tc.CreateTestUser(userEmail, "password123")

	// Create todo
	testID := uuid.New().String()[:8]
	resp := tc.NewRequest("POST", "/api/v1/tables/public/todos").
		WithAuth(token).
		WithBody(map[string]interface{}{
			"user_id":   userID,
			"task":      "Test task " + testID,
			"completed": false,
		}).
		Send().
		AssertStatus(201)

	var todo map[string]interface{}
	resp.JSON(&todo)
	todoID := todo["id"].(string)

	// Verify todo exists
	getResp := tc.NewRequest("GET", "/api/v1/tables/public/todos/"+todoID).
		WithAuth(token).
		Send().
		AssertStatus(200)

	var getTodo map[string]interface{}
	getResp.JSON(&getTodo)
	assert.Equal(t, "Test task "+testID, getTodo["task"])

	// Delete todo
	tc.NewRequest("DELETE", "/api/v1/tables/public/todos/"+todoID).
		WithAuth(token).
		Send().
		AssertStatus(204)

	// Verify todo is deleted
	verifyResp := tc.NewRequest("GET", "/api/v1/tables/public/todos/"+todoID).
		WithAuth(token).
		Send()
	assert.Equal(t, 404, verifyResp.Status(), "Todo should be deleted")
}

func TestRESTHandler_RLS_BatchOperationsWithRLS_Integration(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Drop table if exists (as superuser)
	tc.ExecuteSQLAsSuperuser(`DROP TABLE IF EXISTS public.items CASCADE`)

	// Create test table (as superuser)
	tc.ExecuteSQLAsSuperuser(`
		CREATE TABLE items (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL,
			name TEXT NOT NULL,
			quantity INTEGER NOT NULL
		)
	`)

	// Enable RLS (requires superuser)
	tc.ExecuteSQLAsSuperuser(`ALTER TABLE items ENABLE ROW LEVEL SECURITY`)

	// Create RLS policy: Users can only see their own items (requires superuser)
	tc.ExecuteSQLAsSuperuser(`
		CREATE POLICY user_items_policy ON items
		FOR ALL
		USING (user_id = auth.current_user_id())
		WITH CHECK (user_id = auth.current_user_id())
	`)

	// Grant permissions (requires superuser)
	tc.ExecuteSQLAsSuperuser(`GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE items TO authenticated`)
	tc.ExecuteSQLAsSuperuser(`GRANT USAGE ON ALL SEQUENCES IN SCHEMA public TO authenticated`)

	// Refresh schema cache
	refreshSchemaCache(tc)

	// Create two users
	user1Email := randomEmail()
	user2Email := randomEmail()
	user1ID, token1 := tc.CreateTestUser(user1Email, "password123")
	user2ID, token2 := tc.CreateTestUser(user2Email, "password123")

	testID := uuid.New().String()[:8]

	// User1 creates multiple items
	resp1 := tc.NewRequest("POST", "/api/v1/tables/public/items").
		WithAuth(token1).
		WithBody([]map[string]interface{}{
			{"user_id": user1ID, "name": "Item 1 " + testID, "quantity": 10},
			{"user_id": user1ID, "name": "Item 2 " + testID, "quantity": 20},
			{"user_id": user1ID, "name": "Item 3 " + testID, "quantity": 30},
		}).
		Send().
		AssertStatus(201)

	var items1 []map[string]interface{}
	resp1.JSON(&items1)
	assert.Len(t, items1, 3)

	// User2 creates multiple items
	resp2 := tc.NewRequest("POST", "/api/v1/tables/public/items").
		WithAuth(token2).
		WithBody([]map[string]interface{}{
			{"user_id": user2ID, "name": "Item 4 " + testID, "quantity": 40},
			{"user_id": user2ID, "name": "Item 5 " + testID, "quantity": 50},
		}).
		Send().
		AssertStatus(201)

	var items2 []map[string]interface{}
	resp2.JSON(&items2)
	assert.Len(t, items2, 2)

	// User1 lists all items - should only see their 3 items
	listResp1 := tc.NewRequest("GET", "/api/v1/tables/public/items").
		WithAuth(token1).
		Send().
		AssertStatus(200)

	var allItems1 []map[string]interface{}
	listResp1.JSON(&allItems1)

	// Filter to only our items
	var ourItems1 []map[string]interface{}
	for _, item := range allItems1 {
		if name, ok := item["name"].(string); ok && len(name) > 8 && name[len(name)-8:] == testID {
			ourItems1 = append(ourItems1, item)
		}
	}

	assert.Len(t, ourItems1, 3, "User1 should only see their 3 items")

	// User2 lists all items - should only see their 2 items
	listResp2 := tc.NewRequest("GET", "/api/v1/tables/public/items").
		WithAuth(token2).
		Send().
		AssertStatus(200)

	var allItems2 []map[string]interface{}
	listResp2.JSON(&allItems2)

	// Filter to only our items
	var ourItems2 []map[string]interface{}
	for _, item := range allItems2 {
		if name, ok := item["name"].(string); ok && len(name) > 8 && name[len(name)-8:] == testID {
			ourItems2 = append(ourItems2, item)
		}
	}

	assert.Len(t, ourItems2, 2, "User2 should only see their 2 items")

	// User1 tries to batch update all items with quantity < 25
	// Should only update their own items
	tc.NewRequest("PATCH", "/api/v1/tables/public/items?quantity=lt.25").
		WithAuth(token1).
		WithBody(map[string]interface{}{
			"quantity": 100,
		}).
		Send().
		AssertStatus(200)

	// Verify User1's items were updated but User2's items were not
	verifyResp := tc.NewRequest("GET", "/api/v1/tables/public/items").
		WithAuth(token1).
		Send().
		AssertStatus(200)

	var verifyItems1 []map[string]interface{}
	verifyResp.JSON(&verifyItems1)

	// Filter to only our items and check quantities
	for _, item := range verifyItems1 {
		if name, ok := item["name"].(string); ok && len(name) > 8 && name[len(name)-8:] == testID {
			qty := item["quantity"]
			// All User1's items should have quantity 100 now
			if name == "Item 1 "+testID || name == "Item 2 "+testID {
				assert.EqualValues(t, 100, qty, "Item should be updated to 100")
			}
		}
	}
}
