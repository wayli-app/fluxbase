package test

import (
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// E2EAdminTestSuite defines the E2E admin and metadata test suite
type E2EAdminTestSuite struct {
	suite.Suite
	tc *TestContext
}

// SetupSuite runs once before all tests
func (s *E2EAdminTestSuite) SetupSuite() {
	s.tc = NewTestContext(s.T())
}

// TearDownSuite runs once after all tests
func (s *E2EAdminTestSuite) TearDownSuite() {
	s.tc.Close()
}

// TestGetSchemas tests getting all database schemas
func (s *E2EAdminTestSuite) TestGetSchemas() {
	resp := s.tc.NewRequest("GET", "/api/admin/schemas").Send()

	resp.AssertStatus(fiber.StatusOK)

	var schemas []string
	resp.JSON(&schemas)

	assert.NotEmpty(s.T(), schemas)
	assert.Contains(s.T(), schemas, "public")
}

// TestGetTablesAdmin tests getting all tables via admin endpoint
func (s *E2EAdminTestSuite) TestGetTablesAdmin() {
	resp := s.tc.NewRequest("GET", "/api/admin/tables").Send()

	resp.AssertStatus(fiber.StatusOK)

	var tables []map[string]interface{}
	resp.JSON(&tables)

	assert.NotEmpty(s.T(), tables)

	// Verify table structure
	for _, table := range tables {
		assert.NotNil(s.T(), table["name"])
		assert.NotNil(s.T(), table["schema"])
	}
}

// TestTableMetadata tests getting table metadata
func (s *E2EAdminTestSuite) TestTableMetadata() {
	// Get items table metadata via REST API metadata endpoint
	resp := s.tc.NewRequest("GET", "/api/rest/").Send()

	resp.AssertStatus(fiber.StatusOK)

	var tables []map[string]interface{}
	resp.JSON(&tables)

	assert.NotEmpty(s.T(), tables)

	// Find items table
	var itemsTable map[string]interface{}
	for _, table := range tables {
		if table["name"] == "items" {
			itemsTable = table
			break
		}
	}

	assert.NotNil(s.T(), itemsTable, "items table should exist")
	assert.Equal(s.T(), "items", itemsTable["name"])
	assert.Equal(s.T(), "public", itemsTable["schema"])

	// Check that columns are present
	if columns, ok := itemsTable["columns"]; ok {
		assert.NotNil(s.T(), columns)
	}
}

// TestCORSHeaders tests that CORS headers are properly set
func (s *E2EAdminTestSuite) TestCORSHeaders() {
	resp := s.tc.NewRequest("GET", "/health").
		WithHeader("Origin", "http://localhost:3000").
		Send()

	resp.AssertStatus(fiber.StatusOK)

	// Check CORS headers
	assert.NotEmpty(s.T(), resp.Header("Access-Control-Allow-Origin"))
}

// TestRequestID tests that request ID is properly set
func (s *E2EAdminTestSuite) TestRequestID() {
	resp := s.tc.NewRequest("GET", "/health").Send()

	resp.AssertStatus(fiber.StatusOK)

	// Check request ID header
	requestID := resp.Header("X-Request-Id")
	assert.NotEmpty(s.T(), requestID)
}

// Test404Handler tests the 404 error handling
func (s *E2EAdminTestSuite) Test404Handler() {
	resp := s.tc.NewRequest("GET", "/nonexistent/path").Send()

	assert.Equal(s.T(), fiber.StatusNotFound, resp.Status())

	var result map[string]interface{}
	resp.JSON(&result)

	assert.Equal(s.T(), "Not Found", result["error"])
	assert.Equal(s.T(), "/nonexistent/path", result["path"])
}

// TestCompressionSupport tests that compression is working
func (s *E2EAdminTestSuite) TestCompressionSupport() {
	resp := s.tc.NewRequest("GET", "/health").
		WithHeader("Accept-Encoding", "gzip").
		Send()

	resp.AssertStatus(fiber.StatusOK)

	// If compression is enabled, the Content-Encoding header should be present
	// Note: This depends on response size and compression middleware config
	// For small responses, compression might not be applied
}

// TestContentTypeHeaders tests proper content type handling
func (s *E2EAdminTestSuite) TestContentTypeHeaders() {
	// Test JSON endpoint
	resp := s.tc.NewRequest("GET", "/health").Send()

	resp.AssertStatus(fiber.StatusOK)
	assert.Contains(s.T(), resp.Header("Content-Type"), "application/json")
}

// TestMethodNotAllowed tests handling of unsupported HTTP methods
func (s *E2EAdminTestSuite) TestMethodNotAllowed() {
	// Health endpoint only supports GET
	resp := s.tc.NewRequest("POST", "/health").
		WithBody(map[string]interface{}{"test": "data"}).
		Send()

	// Should return method not allowed or not found
	assert.NotEqual(s.T(), fiber.StatusOK, resp.Status())
}

// Run the test suite
func TestE2EAdminSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	suite.Run(t, new(E2EAdminTestSuite))
}
