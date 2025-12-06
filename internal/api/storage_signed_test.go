package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStorageAPI_GenerateSignedURL_NotSupported(t *testing.T) {
	app, _, db := setupStorageTestServer(t)
	defer db.Close()

	// Create bucket and upload file
	createTestBucket(t, app, "signed-bucket")
	uploadTestFile(t, app, "signed-bucket", "signed.txt", "signed content")

	// Try to generate signed URL (not supported for local storage)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/storage/signed-bucket/signed.txt/signed-url", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Signed URL route returns 400 (bad request) or 501 (not implemented) depending on validation
	// Both are acceptable error responses
	assert.Contains(t, []int{http.StatusBadRequest, http.StatusNotImplemented}, resp.StatusCode)
}
