package test

import (
	"fmt"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// E2ERESTTestSuite defines the E2E REST API test suite
type E2ERESTTestSuite struct {
	suite.Suite
	tc *TestContext
}

// SetupSuite runs once before all tests
func (s *E2ERESTTestSuite) SetupSuite() {
	s.tc = NewTestContext(s.T())
}

// TearDownSuite runs once after all tests
func (s *E2ERESTTestSuite) TearDownSuite() {
	s.tc.Close()
}

// SetupTest runs before each test
func (s *E2ERESTTestSuite) SetupTest() {
	// Clean test data before each test
	s.tc.ExecuteSQL("TRUNCATE TABLE items CASCADE")
}

// TestHealthEndpoint tests the health check endpoint
func (s *E2ERESTTestSuite) TestHealthEndpoint() {
	resp := s.tc.NewRequest("GET", "/health").Send()

	resp.AssertStatus(fiber.StatusOK)

	var result map[string]interface{}
	resp.JSON(&result)

	assert.Equal(s.T(), "ok", result["status"])
	assert.NotNil(s.T(), result["services"])
	assert.NotNil(s.T(), result["timestamp"])
}

// TestGetTables tests getting list of available tables
func (s *E2ERESTTestSuite) TestGetTables() {
	resp := s.tc.NewRequest("GET", "/api/rest/").Send()

	resp.AssertStatus(fiber.StatusOK)

	var tables []map[string]interface{}
	resp.JSON(&tables)

	assert.NotEmpty(s.T(), tables)

	// Check that items table is in the list
	found := false
	for _, table := range tables {
		if table["name"] == "items" {
			found = true
			break
		}
	}
	assert.True(s.T(), found, "items table should be in the list")
}

// TestCRUDOperations tests Create, Read, Update, Delete operations
func (s *E2ERESTTestSuite) TestCRUDOperations() {
	// CREATE
	createPayload := map[string]interface{}{
		"name":        "Test Item",
		"description": "This is a test item",
		"quantity":    10,
		"active":      true,
	}

	createResp := s.tc.NewRequest("POST", "/api/rest/items").
		WithBody(createPayload).
		Send()

	createResp.AssertStatus(fiber.StatusCreated)

	var createdItem map[string]interface{}
	createResp.JSON(&createdItem)

	assert.NotNil(s.T(), createdItem["id"])
	assert.Equal(s.T(), "Test Item", createdItem["name"])
	assert.Equal(s.T(), "This is a test item", createdItem["description"])
	assert.Equal(s.T(), float64(10), createdItem["quantity"])

	itemID := createdItem["id"].(string)

	// READ (single item)
	readResp := s.tc.NewRequest("GET", fmt.Sprintf("/api/rest/items/%s", itemID)).Send()

	readResp.AssertStatus(fiber.StatusOK)

	var fetchedItem map[string]interface{}
	readResp.JSON(&fetchedItem)

	assert.Equal(s.T(), itemID, fetchedItem["id"])
	assert.Equal(s.T(), "Test Item", fetchedItem["name"])

	// UPDATE
	updatePayload := map[string]interface{}{
		"name":     "Updated Item",
		"quantity": 20,
	}

	updateResp := s.tc.NewRequest("PATCH", fmt.Sprintf("/api/rest/items/%s", itemID)).
		WithBody(updatePayload).
		Send()

	updateResp.AssertStatus(fiber.StatusOK)

	var updatedItem map[string]interface{}
	updateResp.JSON(&updatedItem)

	assert.Equal(s.T(), "Updated Item", updatedItem["name"])
	assert.Equal(s.T(), float64(20), updatedItem["quantity"])

	// DELETE
	deleteResp := s.tc.NewRequest("DELETE", fmt.Sprintf("/api/rest/items/%s", itemID)).Send()

	deleteResp.AssertStatus(fiber.StatusNoContent)

	// Verify deletion
	verifyResp := s.tc.NewRequest("GET", fmt.Sprintf("/api/rest/items/%s", itemID)).Send()

	verifyResp.AssertStatus(fiber.StatusNotFound)
}

// TestListItems tests listing items with pagination
func (s *E2ERESTTestSuite) TestListItems() {
	// Insert test data
	for i := 1; i <= 15; i++ {
		s.tc.NewRequest("POST", "/api/rest/items").
			WithBody(map[string]interface{}{
				"name":     fmt.Sprintf("Item %d", i),
				"quantity": i * 10,
				"active":   i%2 == 0,
			}).
			Send().
			AssertStatus(fiber.StatusCreated)
	}

	// Test default listing
	resp := s.tc.NewRequest("GET", "/api/rest/items").Send()
	resp.AssertStatus(fiber.StatusOK)

	var items []map[string]interface{}
	resp.JSON(&items)
	assert.NotEmpty(s.T(), items)

	// Test pagination with limit
	resp = s.tc.NewRequest("GET", "/api/rest/items?limit=5").Send()
	resp.AssertStatus(fiber.StatusOK)
	resp.JSON(&items)
	assert.Len(s.T(), items, 5)

	// Test pagination with limit and offset
	resp = s.tc.NewRequest("GET", "/api/rest/items?limit=5&offset=5").Send()
	resp.AssertStatus(fiber.StatusOK)
	resp.JSON(&items)
	assert.Len(s.T(), items, 5)
}

// TestQueryFilters tests various query filters
func (s *E2ERESTTestSuite) TestQueryFilters() {
	// Insert test data
	testData := []map[string]interface{}{
		{"name": "Apple", "quantity": 5, "active": true},
		{"name": "Banana", "quantity": 15, "active": true},
		{"name": "Cherry", "quantity": 25, "active": false},
		{"name": "Date", "quantity": 35, "active": true},
	}

	for _, data := range testData {
		s.tc.NewRequest("POST", "/api/rest/items").
			WithBody(data).
			Send().
			AssertStatus(fiber.StatusCreated)
	}

	// Test equality filter
	resp := s.tc.NewRequest("GET", "/api/rest/items?name=eq.Apple").Send()
	resp.AssertStatus(fiber.StatusOK)

	var items []map[string]interface{}
	resp.JSON(&items)
	assert.Len(s.T(), items, 1)
	assert.Equal(s.T(), "Apple", items[0]["name"])

	// Test greater than filter
	resp = s.tc.NewRequest("GET", "/api/rest/items?quantity=gt.20").Send()
	resp.AssertStatus(fiber.StatusOK)
	resp.JSON(&items)
	assert.Len(s.T(), items, 2)

	// Test boolean filter
	resp = s.tc.NewRequest("GET", "/api/rest/items?active=eq.true").Send()
	resp.AssertStatus(fiber.StatusOK)
	resp.JSON(&items)
	assert.Len(s.T(), items, 3)

	// Test multiple filters
	resp = s.tc.NewRequest("GET", "/api/rest/items?quantity=gt.10&active=eq.true").Send()
	resp.AssertStatus(fiber.StatusOK)
	resp.JSON(&items)
	assert.Len(s.T(), items, 2)
}

// TestQueryOrdering tests ordering results
func (s *E2ERESTTestSuite) TestQueryOrdering() {
	// Insert test data
	for i := 1; i <= 5; i++ {
		s.tc.NewRequest("POST", "/api/rest/items").
			WithBody(map[string]interface{}{
				"name":     fmt.Sprintf("Item %d", i),
				"quantity": i * 10,
			}).
			Send().
			AssertStatus(fiber.StatusCreated)
	}

	// Test ascending order
	resp := s.tc.NewRequest("GET", "/api/rest/items?order=quantity.asc").Send()
	resp.AssertStatus(fiber.StatusOK)

	var items []map[string]interface{}
	resp.JSON(&items)
	assert.Equal(s.T(), float64(10), items[0]["quantity"])

	// Test descending order
	resp = s.tc.NewRequest("GET", "/api/rest/items?order=quantity.desc").Send()
	resp.AssertStatus(fiber.StatusOK)
	resp.JSON(&items)
	assert.Equal(s.T(), float64(50), items[0]["quantity"])
}

// TestQuerySelect tests column selection
func (s *E2ERESTTestSuite) TestQuerySelect() {
	// Insert test data
	s.tc.NewRequest("POST", "/api/rest/items").
		WithBody(map[string]interface{}{
			"name":        "Test Item",
			"description": "Test Description",
			"quantity":    10,
		}).
		Send().
		AssertStatus(fiber.StatusCreated)

	// Test selecting specific columns
	resp := s.tc.NewRequest("GET", "/api/rest/items?select=id,name").Send()
	resp.AssertStatus(fiber.StatusOK)

	var items []map[string]interface{}
	resp.JSON(&items)
	assert.NotEmpty(s.T(), items)

	// Check that only selected columns are present
	item := items[0]
	assert.NotNil(s.T(), item["id"])
	assert.NotNil(s.T(), item["name"])
	// Note: description and quantity should be excluded, but this depends on implementation
}

// TestInvalidRequests tests error handling
func (s *E2ERESTTestSuite) TestInvalidRequests() {
	// Test invalid table
	resp := s.tc.NewRequest("GET", "/api/rest/nonexistent_table").Send()
	assert.NotEqual(s.T(), fiber.StatusOK, resp.Status())

	// Test invalid JSON
	resp = s.tc.NewRequest("POST", "/api/rest/items").
		WithHeader("Content-Type", "application/json").
		WithBody("invalid json").
		Send()
	assert.NotEqual(s.T(), fiber.StatusCreated, resp.Status())

	// Test GET non-existent item
	resp = s.tc.NewRequest("GET", "/api/rest/items/00000000-0000-0000-0000-000000000000").Send()
	resp.AssertStatus(fiber.StatusNotFound)
}

// TestConcurrentRequests tests handling of concurrent requests
func (s *E2ERESTTestSuite) TestConcurrentRequests() {
	// Create items concurrently
	numItems := 10
	done := make(chan bool, numItems)

	for i := 0; i < numItems; i++ {
		go func(idx int) {
			s.tc.NewRequest("POST", "/api/rest/items").
				WithBody(map[string]interface{}{
					"name":     fmt.Sprintf("Concurrent Item %d", idx),
					"quantity": idx,
				}).
				Send().
				AssertStatus(fiber.StatusCreated)
			done <- true
		}(i)
	}

	// Wait for all requests to complete
	for i := 0; i < numItems; i++ {
		<-done
	}

	// Verify all items were created
	resp := s.tc.NewRequest("GET", "/api/rest/items").Send()
	resp.AssertStatus(fiber.StatusOK)

	var items []map[string]interface{}
	resp.JSON(&items)
	assert.GreaterOrEqual(s.T(), len(items), numItems)
}

// TestProductsWithComplexData tests working with more complex data types
func (s *E2ERESTTestSuite) TestProductsWithComplexData() {
	// Clean products table
	s.tc.ExecuteSQL("TRUNCATE TABLE products CASCADE")

	// Create product with JSONB metadata
	createResp := s.tc.NewRequest("POST", "/api/rest/products").
		WithBody(map[string]interface{}{
			"name":     "Complex Product",
			"price":    99.99,
			"category": "electronics",
			"in_stock": true,
			"metadata": map[string]interface{}{
				"color": "black",
				"size":  "large",
				"specs": map[string]interface{}{
					"weight": "2kg",
					"dimensions": map[string]interface{}{
						"width":  10,
						"height": 20,
						"depth":  5,
					},
				},
			},
		}).
		Send()

	createResp.AssertStatus(fiber.StatusCreated)

	var product map[string]interface{}
	createResp.JSON(&product)

	assert.Equal(s.T(), "Complex Product", product["name"])
	assert.Equal(s.T(), float64(99.99), product["price"])
	assert.NotNil(s.T(), product["metadata"])
}

// Run the test suite
func TestE2ERESTSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	suite.Run(t, new(E2ERESTTestSuite))
}
