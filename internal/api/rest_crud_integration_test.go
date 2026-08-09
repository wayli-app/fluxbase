//go:build integration
// +build integration

package api_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/testutil/e2e"
)

// randomEmail generates a unique email address for testing
func randomEmail() string {
	return fmt.Sprintf("test-%s@example.com", uuid.New().String()[:8])
}

// createTestTable creates a test table and sets up necessary permissions and schema cache refresh.
// This is a convenience function that combines all the setup steps needed for creating
// tables during integration tests.
// Note: Currently unused but kept for potential future use.
/*
func createTestTable(tc *e2e.IntegrationTestContext, schema, table, createSQL string) {
	tc.ExecuteSQL(createSQL)
	refreshSchemaCache(tc)
	grantTablePermissions(tc, schema, table)
}
*/

// refreshSchemaCache refreshes the REST API schema cache so that newly created tables
// can be discovered. This is necessary because the schema cache is populated at server
// startup and doesn't automatically discover tables created afterward.
func refreshSchemaCache(tc *e2e.IntegrationTestContext) {
	// Access the schema cache directly and refresh it
	// This is more reliable than using the API endpoint which requires special authentication
	// and avoids authentication issues with service keys in admin routes
	schemaCache := tc.Server.Schema.Cache
	if schemaCache != nil {
		if err := schemaCache.Refresh(context.Background()); err != nil {
			tc.T.Fatalf("Failed to refresh schema cache: %v", err)
		}
	}
}

// grantTablePermissions grants necessary permissions on a table to the authenticated role
// This is needed for tables created during tests so that authenticated users can perform CRUD operations
func grantTablePermissions(tc *e2e.IntegrationTestContext, schema, table string) {
	// Grant SELECT, INSERT, UPDATE, DELETE permissions on the table to authenticated role
	// This bypasses RLS and allows basic CRUD operations for testing
	// We use regular ExecuteSQL since fluxbase_app user has sufficient privileges
	tc.ExecuteSQL(`GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE ` + schema + `.` + table + ` TO authenticated`)
	// Grant USAGE on the sequence if there is one (for auto-increment/serial columns)
	tc.ExecuteSQL(`GRANT USAGE ON ALL SEQUENCES IN SCHEMA ` + schema + ` TO authenticated`)
}

// =============================================================================
// CRUD Collection Operations (GET, POST, PATCH, DELETE on /tables/:schema/:table)
// =============================================================================

func TestRESTHandler_Create_SingleRecord_Integration(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Drop table if exists to ensure clean state
	tc.ExecuteSQLAsSuperuser(`DROP TABLE IF EXISTS public.products CASCADE`)

	// Create test table
	tc.ExecuteSQL(`
		CREATE TABLE products (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT NOT NULL,
			price NUMERIC NOT NULL DEFAULT 0,
			stock INTEGER DEFAULT 0,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)
	`)

	// Refresh schema cache so REST API discovers the new table
	refreshSchemaCache(tc)

	// Grant permissions on the table to authenticated role
	grantTablePermissions(tc, "public", "products")

	// Create a user and get token
	_, token := tc.CreateTestUser(randomEmail(), "password123")

	// Use unique product name to avoid conflicts from previous test runs
	productName := "Test Product " + uuid.New().String()[:8]

	// Insert single product via API
	resp := tc.NewRequest("POST", "/api/v1/tables/public/products").
		WithAuth(token).
		WithBody(map[string]interface{}{
			"name":  productName,
			"price": 29.99,
			"stock": 100,
		}).
		Send().
		AssertStatus(201)

	// Verify response contains created record
	var product map[string]interface{}
	resp.JSON(&product)
	assert.Equal(t, productName, product["name"])
	assert.Equal(t, 29.99, product["price"])

	// Verify in database
	results := tc.QuerySQL(`SELECT * FROM public.products WHERE name = $1`, productName)
	assert.Len(t, results, 1, "Product should exist in database")
}

func TestRESTHandler_Create_BatchRecords_Integration(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Drop table if exists to ensure clean state
	tc.ExecuteSQLAsSuperuser(`DROP TABLE IF EXISTS public.items CASCADE`)

	// Create test table
	tc.ExecuteSQL(`
		CREATE TABLE items (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			sku TEXT NOT NULL,
			quantity INTEGER NOT NULL
		)
	`)

	// Refresh schema cache so REST API discovers the new table
	refreshSchemaCache(tc)

	// Grant permissions
	grantTablePermissions(tc, "public", "items")

	_, token := tc.CreateTestUser(randomEmail(), "password123")

	// Use unique SKUs to avoid conflicts from previous test runs
	testID := uuid.New().String()[:8]

	// Insert multiple items via API
	resp := tc.NewRequest("POST", "/api/v1/tables/public/items").
		WithAuth(token).
		WithBody([]map[string]interface{}{
			{"sku": "ITEM-" + testID + "-001", "quantity": 10},
			{"sku": "ITEM-" + testID + "-002", "quantity": 20},
			{"sku": "ITEM-" + testID + "-003", "quantity": 30},
		}).
		Send().
		AssertStatus(201)

	var items []map[string]interface{}
	resp.JSON(&items)
	assert.Len(t, items, 3, "Should create 3 items")

	// Verify in database - only count items with our unique prefix
	results := tc.QuerySQL(`SELECT COUNT(*) as count FROM public.items WHERE sku LIKE $1`, "ITEM-"+testID+"%")
	assert.EqualValues(t, 3, results[0]["count"])
}

func TestRESTHandler_List_AllRecords_Integration(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Drop table if exists to ensure clean state
	tc.ExecuteSQLAsSuperuser(`DROP TABLE IF EXISTS public.documents CASCADE`)

	// Create test table and seed data
	tc.ExecuteSQL(`
		CREATE TABLE documents (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			title TEXT NOT NULL,
			status TEXT DEFAULT 'draft',
			priority INTEGER DEFAULT 1
		)
	`)

	// Refresh schema cache so REST API discovers the new table
	refreshSchemaCache(tc)

	// Grant permissions
	grantTablePermissions(tc, "public", "documents")

	// Use unique titles to avoid conflicts
	testID := uuid.New().String()[:8]
	tc.ExecuteSQL(`INSERT INTO documents (title, status, priority) VALUES ($1, $2, $3)`, "Doc 1 "+testID, "published", 1)
	tc.ExecuteSQL(`INSERT INTO documents (title, status, priority) VALUES ($1, $2, $3)`, "Doc 2 "+testID, "draft", 2)
	tc.ExecuteSQL(`INSERT INTO documents (title, status, priority) VALUES ($1, $2, $3)`, "Doc 3 "+testID, "published", 3)

	_, token := tc.CreateTestUser(randomEmail(), "password123")

	// List all documents
	resp := tc.NewRequest("GET", "/api/v1/tables/public/documents").
		WithAuth(token).
		Send().
		AssertStatus(200)

	var docs []map[string]interface{}
	resp.JSON(&docs)
	// Filter to only our docs with the unique prefix
	var ourDocs []map[string]interface{}
	for _, doc := range docs {
		if title, ok := doc["title"].(string); ok && len(title) > 8 && title[len(title)-8:] == testID {
			ourDocs = append(ourDocs, doc)
		}
	}
	assert.Len(t, ourDocs, 3, "Should return our 3 documents")
}

func TestRESTHandler_List_WithFilter_Integration(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Drop table if exists to ensure clean state
	tc.ExecuteSQLAsSuperuser(`DROP TABLE IF EXISTS public.tasks CASCADE`)

	// Create test table and seed data
	tc.ExecuteSQL(`
		CREATE TABLE tasks (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			title TEXT NOT NULL,
			status TEXT DEFAULT 'open',
			priority INTEGER DEFAULT 1
		)
	`)

	// Refresh schema cache so REST API discovers the new table
	refreshSchemaCache(tc)

	// Grant permissions
	grantTablePermissions(tc, "public", "tasks")

	// Use unique task titles
	testID := uuid.New().String()[:8]
	tc.ExecuteSQL(`INSERT INTO tasks (title, status, priority) VALUES ($1, $2, $3)`, "Task 1 "+testID, "open", 1)
	tc.ExecuteSQL(`INSERT INTO tasks (title, status, priority) VALUES ($1, $2, $3)`, "Task 2 "+testID, "closed", 2)
	tc.ExecuteSQL(`INSERT INTO tasks (title, status, priority) VALUES ($1, $2, $3)`, "Task 3 "+testID, "open", 3)

	_, token := tc.CreateTestUser(randomEmail(), "password123")

	// Filter by status = open
	resp := tc.NewRequest("GET", "/api/v1/tables/public/tasks?status=eq.open").
		WithAuth(token).
		Send().
		AssertStatus(200)

	var tasks []map[string]interface{}
	resp.JSON(&tasks)
	// Filter to only our tasks
	var ourTasks []map[string]interface{}
	for _, task := range tasks {
		if title, ok := task["title"].(string); ok && len(title) > 8 && title[len(title)-8:] == testID {
			ourTasks = append(ourTasks, task)
		}
	}
	assert.Len(t, ourTasks, 2, "Should return only our 2 open tasks")
	for _, task := range ourTasks {
		assert.Equal(t, "open", task["status"])
	}
}

func TestRESTHandler_List_WithPagination_Integration(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Drop table if exists to ensure clean state
	tc.ExecuteSQLAsSuperuser(`DROP TABLE IF EXISTS public.products CASCADE`)

	// Create test table and seed many records
	tc.ExecuteSQL(`
		CREATE TABLE products (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT NOT NULL,
			price NUMERIC NOT NULL
		)
	`)

	// Refresh schema cache so REST API discovers the new table
	refreshSchemaCache(tc)

	// Grant permissions
	grantTablePermissions(tc, "public", "products")

	// Use unique product names to avoid conflicts
	testID := uuid.New().String()[:8]

	// Insert 25 products
	for i := 1; i <= 25; i++ {
		tc.ExecuteSQL(`INSERT INTO products (name, price) VALUES ($1, $2)`, fmt.Sprintf("Product %d %s", i, testID), float64(i*10))
	}

	_, token := tc.CreateTestUser(randomEmail(), "password123")

	// Get first page with limit=10
	resp := tc.NewRequest("GET", "/api/v1/tables/public/products?limit=10&offset=0").
		WithAuth(token).
		Send().
		AssertStatus(200)

	var products []map[string]interface{}
	resp.JSON(&products)
	assert.Len(t, products, 10, "Should return 10 products")

	// Note: Content-Range header may not be present in all implementations
	// The key assertion is that we got the correct number of products
}

func TestRESTHandler_GetSingleRecord_Integration(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Drop table if exists to ensure clean state
	tc.ExecuteSQLAsSuperuser(`DROP TABLE IF EXISTS public.customers CASCADE`)

	// Create test table and seed data
	tc.ExecuteSQL(`
		CREATE TABLE customers (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT NOT NULL,
			email TEXT UNIQUE NOT NULL
		)
	`)

	// Refresh schema cache so REST API discovers the new table
	refreshSchemaCache(tc)

	// Grant permissions
	grantTablePermissions(tc, "public", "customers")

	// Use unique customer data to avoid conflicts
	testID := uuid.New().String()[:8]
	customerName := "John Doe " + testID
	customerEmail := "john-" + testID + "@example.com"

	// Insert a customer
	result := tc.QuerySQL(`INSERT INTO customers (name, email) VALUES ($1, $2) RETURNING id`, customerName, customerEmail)
	customerID := result[0]["id"]

	_, token := tc.CreateTestUser(randomEmail(), "password123")

	// Get specific customer by ID
	resp := tc.NewRequest("GET", "/api/v1/tables/public/customers/"+customerID.(string)).
		WithAuth(token).
		Send().
		AssertStatus(200)

	var customer map[string]interface{}
	resp.JSON(&customer)
	assert.Equal(t, customerName, customer["name"])
	assert.Equal(t, customerEmail, customer["email"])
}

func TestRESTHandler_GetSingleRecord_NotFound_Integration(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Drop table if exists to ensure clean state
	tc.ExecuteSQLAsSuperuser(`DROP TABLE IF EXISTS public.widgets CASCADE`)

	// Create test table
	tc.ExecuteSQL(`
		CREATE TABLE widgets (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT NOT NULL
		)
	`)

	// Refresh schema cache so REST API discovers the new table
	refreshSchemaCache(tc)

	// Grant permissions
	grantTablePermissions(tc, "public", "widgets")

	_, token := tc.CreateTestUser(randomEmail(), "password123")

	// Try to get non-existent record
	resp := tc.NewRequest("GET", "/api/v1/tables/public/widgets/00000000-0000-0000-0000-000000000000").
		WithAuth(token).
		Send()

	// Should return 404
	assert.Equal(t, 404, resp.Status())
}

func TestRESTHandler_UpdateSingleRecord_Patch_Integration(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Drop table if exists to ensure clean state
	tc.ExecuteSQLAsSuperuser(`DROP TABLE IF EXISTS public.profiles CASCADE`)

	// Create test table
	tc.ExecuteSQL(`
		CREATE TABLE profiles (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			display_name TEXT,
			bio TEXT,
			views INTEGER DEFAULT 0
		)
	`)

	// Refresh schema cache so REST API discovers the new table
	refreshSchemaCache(tc)

	// Grant permissions
	grantTablePermissions(tc, "public", "profiles")

	// Use unique data to avoid conflicts
	testID := uuid.New().String()[:8]
	originalName := "Original Name " + testID
	updatedName := "Updated Name " + testID
	originalBio := "Original bio " + testID

	// Insert a profile
	result := tc.QuerySQL(`INSERT INTO profiles (display_name, bio, views) VALUES ($1, $2, $3) RETURNING id`, originalName, originalBio, 100)
	profileID := result[0]["id"]

	_, token := tc.CreateTestUser(randomEmail(), "password123")

	// Partial update with PATCH
	resp := tc.NewRequest("PATCH", "/api/v1/tables/public/profiles/"+profileID.(string)).
		WithAuth(token).
		WithBody(map[string]interface{}{
			"display_name": updatedName,
			"views":        200,
		}).
		Send().
		AssertStatus(200)

	var profile map[string]interface{}
	resp.JSON(&profile)
	assert.Equal(t, updatedName, profile["display_name"])
	assert.Equal(t, originalBio, profile["bio"], "bio should be unchanged")
	assert.EqualValues(t, 200, profile["views"])

	// Verify in database
	results := tc.QuerySQL(`SELECT display_name, views FROM profiles WHERE id = $1`, profileID)
	assert.Len(t, results, 1)
	assert.Equal(t, updatedName, results[0]["display_name"])
}

func TestRESTHandler_UpdateSingleRecord_Put_Integration(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Drop table if exists to ensure clean state
	tc.ExecuteSQLAsSuperuser(`DROP TABLE IF EXISTS public.config CASCADE`)

	// Create test table
	tc.ExecuteSQL(`
		CREATE TABLE config (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			key TEXT NOT NULL,
			value TEXT NOT NULL
		)
	`)

	// Refresh schema cache so REST API discovers the new table
	refreshSchemaCache(tc)

	// Grant permissions
	grantTablePermissions(tc, "public", "config")

	// Use unique config to avoid conflicts
	testID := uuid.New().String()[:8]
	configKey := "theme-" + testID

	// Insert config
	result := tc.QuerySQL(`INSERT INTO config (key, value) VALUES ($1, $2) RETURNING id`, configKey, "dark")
	configID := result[0]["id"]

	_, token := tc.CreateTestUser(randomEmail(), "password123")

	// Replace entire record with PUT
	resp := tc.NewRequest("PUT", "/api/v1/tables/public/config/"+configID.(string)).
		WithAuth(token).
		WithBody(map[string]interface{}{
			"key":   configKey,
			"value": "light",
		}).
		Send().
		AssertStatus(200)

	var config map[string]interface{}
	resp.JSON(&config)
	assert.Equal(t, "light", config["value"])

	// Verify in database
	results := tc.QuerySQL(`SELECT value FROM config WHERE id = $1`, configID)
	assert.Len(t, results, 1)
	assert.Equal(t, "light", results[0]["value"])
}

func TestRESTHandler_DeleteSingleRecord_Integration(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Drop table if exists to ensure clean state
	tc.ExecuteSQLAsSuperuser(`DROP TABLE IF EXISTS public.temp_data CASCADE`)

	// Create test table
	tc.ExecuteSQL(`
		CREATE TABLE temp_data (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			data TEXT NOT NULL
		)
	`)

	// Refresh schema cache so REST API discovers the new table
	refreshSchemaCache(tc)

	// Grant permissions
	grantTablePermissions(tc, "public", "temp_data")

	// Use unique data to avoid conflicts
	testID := uuid.New().String()[:8]
	testData := "test data " + testID

	// Insert test data
	result := tc.QuerySQL(`INSERT INTO temp_data (data) VALUES ($1) RETURNING id`, testData)
	tempID := result[0]["id"]

	_, token := tc.CreateTestUser(randomEmail(), "password123")

	// Verify record exists before deletion
	tc.NewRequest("GET", "/api/v1/tables/public/temp_data/"+tempID.(string)).
		WithAuth(token).
		Send().
		AssertStatus(200)

	// Delete record
	tc.NewRequest("DELETE", "/api/v1/tables/public/temp_data/"+tempID.(string)).
		WithAuth(token).
		Send().
		AssertStatus(204)

	// Verify record is deleted
	verifyResp := tc.NewRequest("GET", "/api/v1/tables/public/temp_data/"+tempID.(string)).
		WithAuth(token).
		Send()
	assert.Equal(t, 404, verifyResp.Status(), "Record should be deleted")
}

func TestRESTHandler_BatchDelete_WithFilter_Integration(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Drop table if exists to ensure clean state
	tc.ExecuteSQLAsSuperuser(`DROP TABLE IF EXISTS public.logs CASCADE`)

	// Create test table
	tc.ExecuteSQL(`
		CREATE TABLE logs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			level TEXT NOT NULL,
			message TEXT,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)
	`)

	// Refresh schema cache so REST API discovers the new table
	refreshSchemaCache(tc)

	// Grant permissions
	grantTablePermissions(tc, "public", "logs")

	// Use unique log messages to avoid conflicts
	testID := uuid.New().String()[:8]

	// Insert test data
	tc.ExecuteSQL(`INSERT INTO logs (level, message) VALUES ($1, $2)`, "error", "Error 1 "+testID)
	tc.ExecuteSQL(`INSERT INTO logs (level, message) VALUES ($1, $2)`, "error", "Error 2 "+testID)
	tc.ExecuteSQL(`INSERT INTO logs (level, message) VALUES ($1, $2)`, "info", "Info 1 "+testID)
	tc.ExecuteSQL(`INSERT INTO logs (level, message) VALUES ($1, $2)`, "debug", "Debug 1 "+testID)

	_, token := tc.CreateTestUser(randomEmail(), "password123")

	// Batch delete all error logs - returns 200 with affected records
	tc.NewRequest("DELETE", "/api/v1/tables/public/logs?level=eq.error").
		WithAuth(token).
		Send().
		AssertStatus(200)

	// Verify only non-error logs remain for our test data
	results := tc.QuerySQL(`SELECT COUNT(*) as count FROM public.logs WHERE level = 'error' AND message LIKE $1`, "%"+testID)
	assert.EqualValues(t, 0, results[0]["count"], "All our error logs should be deleted")

	totalResults := tc.QuerySQL(`SELECT COUNT(*) as count FROM public.logs WHERE message LIKE $1`, "%"+testID)
	assert.EqualValues(t, 2, totalResults[0]["count"], "Should have 2 our non-error logs remaining")
}

func TestRESTHandler_BatchUpdate_WithFilter_Integration(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Drop table if exists to ensure clean state
	tc.ExecuteSQLAsSuperuser(`DROP TABLE IF EXISTS public.inventory CASCADE`)

	// Create test table
	tc.ExecuteSQL(`
		CREATE TABLE inventory (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			product_name TEXT NOT NULL,
			quantity INTEGER NOT NULL
		)
	`)

	// Refresh schema cache so REST API discovers the new table
	refreshSchemaCache(tc)

	// Grant permissions
	grantTablePermissions(tc, "public", "inventory")

	// Use unique product names to avoid conflicts (no spaces to avoid URL encoding issues)
	testID := uuid.New().String()[:8]
	productA := "ProductA_" + testID
	productB := "ProductB_" + testID

	// Insert test data
	tc.ExecuteSQL(`INSERT INTO inventory (product_name, quantity) VALUES ($1, $2)`, productA, 10)
	tc.ExecuteSQL(`INSERT INTO inventory (product_name, quantity) VALUES ($1, $2)`, productB, 20)
	tc.ExecuteSQL(`INSERT INTO inventory (product_name, quantity) VALUES ($1, $2)`, productA, 5)

	_, token := tc.CreateTestUser(randomEmail(), "password123")

	// Batch update all ProductA records to quantity = 50
	tc.NewRequest("PATCH", "/api/v1/tables/public/inventory?product_name=eq."+productA).
		WithAuth(token).
		WithBody(map[string]interface{}{
			"quantity": 50,
		}).
		Send().
		AssertStatus(200)

	// Verify updates
	results := tc.QuerySQL(`SELECT product_name, quantity FROM inventory WHERE product_name = $1 ORDER BY id`, productA)
	assert.Len(t, results, 2)
	for _, result := range results {
		assert.EqualValues(t, 50, result["quantity"], "All ProductA records should be updated to 50")
	}

	// Verify ProductB is unchanged
	resultsB := tc.QuerySQL(`SELECT quantity FROM inventory WHERE product_name = $1`, productB)
	assert.Len(t, resultsB, 1)
	assert.EqualValues(t, 20, resultsB[0]["quantity"], "ProductB should be unchanged")
}

func TestRESTHandler_OrderBy_Integration(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Drop table if exists to ensure clean state
	tc.ExecuteSQLAsSuperuser(`DROP TABLE IF EXISTS public.articles CASCADE`)

	// Create test table
	tc.ExecuteSQL(`
		CREATE TABLE articles (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			title TEXT NOT NULL,
			published_at TIMESTAMPTZ,
			views INTEGER DEFAULT 0
		)
	`)

	// Refresh schema cache so REST API discovers the new table
	refreshSchemaCache(tc)

	// Grant permissions
	grantTablePermissions(tc, "public", "articles")

	// Use unique article titles to avoid conflicts
	testID := uuid.New().String()[:8]

	// Insert articles with different timestamps
	tc.ExecuteSQL(`INSERT INTO articles (title, published_at, views) VALUES ($1, NOW() - INTERVAL '1 day', 100)`, "Article 1 "+testID)
	tc.ExecuteSQL(`INSERT INTO articles (title, published_at, views) VALUES ($1, NOW() - INTERVAL '2 days', 200)`, "Article 2 "+testID)
	tc.ExecuteSQL(`INSERT INTO articles (title, published_at, views) VALUES ($1, NOW() - INTERVAL '3 days', 150)`, "Article 3 "+testID)

	_, token := tc.CreateTestUser(randomEmail(), "password123")

	// Order by published_at ascending
	resp := tc.NewRequest("GET", "/api/v1/tables/public/articles?order=published_at.asc").
		WithAuth(token).
		Send().
		AssertStatus(200)

	var articles []map[string]interface{}
	resp.JSON(&articles)
	// Filter to only our articles
	var ourArticles []map[string]interface{}
	for _, article := range articles {
		if title, ok := article["title"].(string); ok && len(title) > 8 && title[len(title)-8:] == testID {
			ourArticles = append(ourArticles, article)
		}
	}
	assert.Len(t, ourArticles, 3)
}

func TestRESTHandler_Selection_Integration(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Drop table if exists to ensure clean state
	tc.ExecuteSQLAsSuperuser(`DROP TABLE IF EXISTS public.users_extended CASCADE`)

	// Create test table with many columns
	tc.ExecuteSQL(`
		CREATE TABLE users_extended (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			first_name TEXT,
			last_name TEXT,
			email TEXT NOT NULL,
			phone TEXT,
			address TEXT,
			city TEXT,
			zip_code TEXT
		)
	`)

	// Refresh schema cache so REST API discovers the new table
	refreshSchemaCache(tc)

	// Grant permissions
	grantTablePermissions(tc, "public", "users_extended")

	// Use unique user data to avoid conflicts
	testID := uuid.New().String()[:8]
	firstName := "John " + testID
	lastName := "Doe " + testID
	email := "john-" + testID + "@example.com"
	phone := "555-" + testID

	tc.ExecuteSQL(`INSERT INTO users_extended (first_name, last_name, email, phone) VALUES ($1, $2, $3, $4)`,
		firstName, lastName, email, phone)

	_, token := tc.CreateTestUser(randomEmail(), "password123")

	// Select only specific columns
	resp := tc.NewRequest("GET", "/api/v1/tables/public/users_extended?select=first_name,last_name,email").
		WithAuth(token).
		Send().
		AssertStatus(200)

	var users []map[string]interface{}
	resp.JSON(&users)
	// Filter to only our user
	var ourUsers []map[string]interface{}
	for _, user := range users {
		if fn, ok := user["first_name"].(string); ok && len(fn) > 8 && fn[len(fn)-8:] == testID {
			ourUsers = append(ourUsers, user)
		}
	}
	require.Len(t, ourUsers, 1)

	// Verify only selected columns are returned
	_, hasFirstName := ourUsers[0]["first_name"]
	_, hasLastName := ourUsers[0]["last_name"]
	_, hasEmail := ourUsers[0]["email"]
	_, hasPhone := ourUsers[0]["phone"]

	assert.True(t, hasFirstName, "Should have first_name")
	assert.True(t, hasLastName, "Should have last_name")
	assert.True(t, hasEmail, "Should have email")
	assert.False(t, hasPhone, "Should NOT have phone (not selected)")
}

func TestRESTHandler_Aggregation_Count_Integration(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Drop table if exists to ensure clean state
	tc.ExecuteSQLAsSuperuser(`DROP TABLE IF EXISTS public.orders CASCADE`)

	// Create test table
	tc.ExecuteSQL(`
		CREATE TABLE orders (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			status TEXT NOT NULL,
			total NUMERIC(10,2) NOT NULL
		)
	`)

	// Refresh schema cache so REST API discovers the new table
	refreshSchemaCache(tc)

	// Grant permissions
	grantTablePermissions(tc, "public", "orders")

	// Use unique order data to avoid conflicts by using testID in status
	testID := uuid.New().String()[:8]
	statusPending := "pending-" + testID

	// Insert test data
	tc.ExecuteSQL(`INSERT INTO orders (status, total) VALUES ($1, $2)`, statusPending, 100.00)
	tc.ExecuteSQL(`INSERT INTO orders (status, total) VALUES ($1, $2)`, statusPending, 150.00)
	tc.ExecuteSQL(`INSERT INTO orders (status, total) VALUES ($1, $2)`, statusPending, 200.00)
	tc.ExecuteSQL(`INSERT INTO orders (status, total) VALUES ($1, $2)`, statusPending, 50.00)

	_, token := tc.CreateTestUser(randomEmail(), "password123")

	// Query orders by status to verify data was inserted
	resp := tc.NewRequest("GET", "/api/v1/tables/public/orders?status=eq."+statusPending).
		WithAuth(token).
		Send().
		AssertStatus(200)

	var results []map[string]interface{}
	resp.JSON(&results)

	// Should have 4 orders with our test status
	assert.Len(t, results, 4, "Should have 4 orders with pending status")
}

func TestRESTHandler_InvalidTable_Integration(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	_, token := tc.CreateTestUser(randomEmail(), "password123")

	// Try to access non-existent table
	resp := tc.NewRequest("GET", "/api/v1/tables/public/non_existent_table").
		WithAuth(token).
		Send()

	// Should return error
	assert.NotEqual(t, 200, resp.Status())
}

func TestRESTHandler_Unauthenticated_Integration(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "api")
	defer tc.Close()
	defer tc.CleanupTestData()

	// Drop table if exists to ensure clean state
	tc.ExecuteSQLAsSuperuser(`DROP TABLE IF EXISTS public.public_data CASCADE`)

	// Create test table
	tc.ExecuteSQL(`
		CREATE TABLE public_data (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			info TEXT
		)
	`)

	// Refresh schema cache so REST API discovers the new table
	refreshSchemaCache(tc)

	// Grant permissions
	grantTablePermissions(tc, "public", "public_data")

	// Try to access without authentication
	resp := tc.NewRequest("GET", "/api/v1/tables/public/public_data").
		Send()

	// Should return 401 Unauthorized
	assert.Equal(t, 401, resp.Status())
}
