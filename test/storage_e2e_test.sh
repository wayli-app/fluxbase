#!/bin/bash
set -e

# E2E Test for Storage Service with MinIO
# This test verifies the complete storage workflow with S3-compatible backend

echo "=== Storage Service E2E Tests ==="
echo ""

# Configuration
API_BASE="http://localhost:8080"
STORAGE_API="$API_BASE/api/storage"
AUTH_API="$API_BASE/api/auth"
MINIO_ENDPOINT="${FLUXBASE_STORAGE_S3_ENDPOINT:-http://minio:9000}"
MINIO_ACCESS_KEY="${FLUXBASE_STORAGE_S3_ACCESS_KEY:-minioadmin}"
MINIO_SECRET_KEY="${FLUXBASE_STORAGE_S3_SECRET_KEY:-minioadmin}"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Helper functions
pass() {
    echo -e "${GREEN}✓ $1${NC}"
    ((TESTS_PASSED++))
}

fail() {
    echo -e "${RED}✗ $1${NC}"
    ((TESTS_FAILED++))
}

info() {
    echo -e "${BLUE}ℹ $1${NC}"
}

# Cleanup function
cleanup() {
    info "Cleaning up test resources..."

    # Delete test bucket if it exists
    if [ -n "$TOKEN" ]; then
        curl -s -X DELETE "$STORAGE_API/test-e2e-bucket" \
            -H "Authorization: Bearer $TOKEN" \
            > /dev/null 2>&1 || true
    fi
}

trap cleanup EXIT

# Check MinIO availability
info "Checking MinIO availability at $MINIO_ENDPOINT..."
if ! curl -s -f "$MINIO_ENDPOINT/minio/health/live" > /dev/null; then
    fail "MinIO is not accessible at $MINIO_ENDPOINT"
    exit 1
fi
pass "MinIO is accessible"

# Wait for Fluxbase to be ready
info "Waiting for Fluxbase API to be ready..."
MAX_RETRIES=30
RETRY_COUNT=0
while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    if curl -s -f "$API_BASE/health" > /dev/null 2>&1; then
        break
    fi
    RETRY_COUNT=$((RETRY_COUNT + 1))
    sleep 1
done

if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
    fail "Fluxbase API did not become ready in time"
    exit 1
fi
pass "Fluxbase API is ready"

# Test 1: User signup and authentication
info "Test 1: User authentication"
SIGNUP_RESPONSE=$(curl -s -X POST "$AUTH_API/signup" \
    -H "Content-Type: application/json" \
    -d '{
        "email": "storage-test@example.com",
        "password": "Test1234!",
        "data": {"name": "Storage Test User"}
    }')

TOKEN=$(echo "$SIGNUP_RESPONSE" | grep -o '"access_token":"[^"]*' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
    # Try signing in if user already exists
    SIGNIN_RESPONSE=$(curl -s -X POST "$AUTH_API/signin" \
        -H "Content-Type: application/json" \
        -d '{
            "email": "storage-test@example.com",
            "password": "Test1234!"
        }')
    TOKEN=$(echo "$SIGNIN_RESPONSE" | grep -o '"access_token":"[^"]*' | cut -d'"' -f4)
fi

if [ -n "$TOKEN" ]; then
    pass "User authenticated successfully"
else
    fail "Failed to authenticate user"
    exit 1
fi

# Test 2: List buckets (should be empty initially)
info "Test 2: List buckets"
BUCKETS_RESPONSE=$(curl -s -X GET "$STORAGE_API/buckets" \
    -H "Authorization: Bearer $TOKEN")

if echo "$BUCKETS_RESPONSE" | grep -q '\[\]' || echo "$BUCKETS_RESPONSE" | grep -q '"buckets"'; then
    pass "Listed buckets successfully"
else
    fail "Failed to list buckets: $BUCKETS_RESPONSE"
fi

# Test 3: Create a bucket
info "Test 3: Create bucket"
CREATE_BUCKET_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$STORAGE_API/test-e2e-bucket" \
    -H "Authorization: Bearer $TOKEN")

HTTP_CODE=$(echo "$CREATE_BUCKET_RESPONSE" | tail -n1)
RESPONSE_BODY=$(echo "$CREATE_BUCKET_RESPONSE" | head -n-1)

if [ "$HTTP_CODE" = "201" ] || [ "$HTTP_CODE" = "200" ]; then
    pass "Created bucket 'test-e2e-bucket'"
elif [ "$HTTP_CODE" = "409" ]; then
    pass "Bucket 'test-e2e-bucket' already exists (409)"
else
    fail "Failed to create bucket (HTTP $HTTP_CODE): $RESPONSE_BODY"
fi

# Test 4: Upload a file
info "Test 4: Upload file"
TEST_FILE="/tmp/test-file.txt"
echo "Hello from Fluxbase Storage E2E Test!" > "$TEST_FILE"

UPLOAD_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$STORAGE_API/test-e2e-bucket/test-file.txt" \
    -H "Authorization: Bearer $TOKEN" \
    -F "file=@$TEST_FILE" \
    -F "x-meta-description=E2E test file" \
    -F "x-meta-author=fluxbase-test")

HTTP_CODE=$(echo "$UPLOAD_RESPONSE" | tail -n1)
RESPONSE_BODY=$(echo "$UPLOAD_RESPONSE" | head -n-1)

if [ "$HTTP_CODE" = "201" ] || [ "$HTTP_CODE" = "200" ]; then
    pass "Uploaded file successfully"
else
    fail "Failed to upload file (HTTP $HTTP_CODE): $RESPONSE_BODY"
fi

# Test 5: Get object metadata
info "Test 5: Get object metadata"
METADATA_RESPONSE=$(curl -s -w "\n%{http_code}" -X HEAD "$STORAGE_API/test-e2e-bucket/test-file.txt" \
    -H "Authorization: Bearer $TOKEN")

HTTP_CODE=$(echo "$METADATA_RESPONSE" | tail -n1)

if [ "$HTTP_CODE" = "200" ]; then
    pass "Retrieved object metadata"
else
    fail "Failed to get metadata (HTTP $HTTP_CODE)"
fi

# Test 6: Download the file
info "Test 6: Download file"
DOWNLOAD_RESPONSE=$(curl -s -w "\n%{http_code}" -X GET "$STORAGE_API/test-e2e-bucket/test-file.txt" \
    -H "Authorization: Bearer $TOKEN" \
    -o /tmp/downloaded-file.txt)

HTTP_CODE=$(echo "$DOWNLOAD_RESPONSE" | tail -n1)

if [ "$HTTP_CODE" = "200" ]; then
    ORIGINAL_CONTENT=$(cat "$TEST_FILE")
    DOWNLOADED_CONTENT=$(cat /tmp/downloaded-file.txt)

    if [ "$ORIGINAL_CONTENT" = "$DOWNLOADED_CONTENT" ]; then
        pass "Downloaded file matches original"
    else
        fail "Downloaded file content mismatch"
    fi
else
    fail "Failed to download file (HTTP $HTTP_CODE)"
fi

# Test 7: List objects in bucket
info "Test 7: List objects in bucket"
LIST_RESPONSE=$(curl -s -w "\n%{http_code}" -X GET "$STORAGE_API/test-e2e-bucket" \
    -H "Authorization: Bearer $TOKEN")

HTTP_CODE=$(echo "$LIST_RESPONSE" | tail -n1)
RESPONSE_BODY=$(echo "$LIST_RESPONSE" | head -n-1)

if [ "$HTTP_CODE" = "200" ] && echo "$RESPONSE_BODY" | grep -q "test-file.txt"; then
    pass "Listed objects in bucket"
else
    fail "Failed to list objects (HTTP $HTTP_CODE): $RESPONSE_BODY"
fi

# Test 8: Upload file with nested path
info "Test 8: Upload file with nested path"
NESTED_UPLOAD_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$STORAGE_API/test-e2e-bucket/documents/reports/report.pdf" \
    -H "Authorization: Bearer $TOKEN" \
    -F "file=@$TEST_FILE")

HTTP_CODE=$(echo "$NESTED_UPLOAD_RESPONSE" | tail -n1)

if [ "$HTTP_CODE" = "201" ] || [ "$HTTP_CODE" = "200" ]; then
    pass "Uploaded file with nested path"
else
    fail "Failed to upload nested file (HTTP $HTTP_CODE)"
fi

# Test 9: List with prefix
info "Test 9: List objects with prefix"
PREFIX_LIST_RESPONSE=$(curl -s -w "\n%{http_code}" -X GET "$STORAGE_API/test-e2e-bucket?prefix=documents/" \
    -H "Authorization: Bearer $TOKEN")

HTTP_CODE=$(echo "$PREFIX_LIST_RESPONSE" | tail -n1)
RESPONSE_BODY=$(echo "$PREFIX_LIST_RESPONSE" | head -n-1)

if [ "$HTTP_CODE" = "200" ] && echo "$RESPONSE_BODY" | grep -q "documents/reports/report.pdf"; then
    pass "Listed objects with prefix filter"
else
    fail "Failed to list with prefix (HTTP $HTTP_CODE)"
fi

# Test 10: Generate signed URL
info "Test 10: Generate signed URL"
SIGNED_URL_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$STORAGE_API/test-e2e-bucket/test-file.txt/signed-url" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"method": "GET", "expires_in": 3600}')

HTTP_CODE=$(echo "$SIGNED_URL_RESPONSE" | tail -n1)
RESPONSE_BODY=$(echo "$SIGNED_URL_RESPONSE" | head -n-1)

if [ "$HTTP_CODE" = "200" ] && echo "$RESPONSE_BODY" | grep -q "url"; then
    pass "Generated signed URL"
elif [ "$HTTP_CODE" = "501" ]; then
    pass "Signed URLs not implemented for current storage provider (expected)"
else
    fail "Unexpected response for signed URL (HTTP $HTTP_CODE)"
fi

# Test 11: Delete object
info "Test 11: Delete object"
DELETE_RESPONSE=$(curl -s -w "\n%{http_code}" -X DELETE "$STORAGE_API/test-e2e-bucket/test-file.txt" \
    -H "Authorization: Bearer $TOKEN")

HTTP_CODE=$(echo "$DELETE_RESPONSE" | tail -n1)

if [ "$HTTP_CODE" = "204" ] || [ "$HTTP_CODE" = "200" ]; then
    pass "Deleted object successfully"
else
    fail "Failed to delete object (HTTP $HTTP_CODE)"
fi

# Test 12: Verify object is deleted
info "Test 12: Verify object is deleted"
VERIFY_DELETE_RESPONSE=$(curl -s -w "\n%{http_code}" -X GET "$STORAGE_API/test-e2e-bucket/test-file.txt" \
    -H "Authorization: Bearer $TOKEN" \
    -o /dev/null)

HTTP_CODE=$(echo "$VERIFY_DELETE_RESPONSE" | tail -n1)

if [ "$HTTP_CODE" = "404" ]; then
    pass "Verified object was deleted"
else
    fail "Object still exists after deletion (HTTP $HTTP_CODE)"
fi

# Test 13: Delete bucket with contents
info "Test 13: Delete bucket (should fail if not empty)"
DELETE_BUCKET_RESPONSE=$(curl -s -w "\n%{http_code}" -X DELETE "$STORAGE_API/test-e2e-bucket" \
    -H "Authorization: Bearer $TOKEN")

HTTP_CODE=$(echo "$DELETE_BUCKET_RESPONSE" | tail -n1)

if [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "409" ]; then
    pass "Cannot delete non-empty bucket (expected)"

    # Clean up remaining objects
    info "Cleaning up remaining objects..."
    curl -s -X DELETE "$STORAGE_API/test-e2e-bucket/documents/reports/report.pdf" \
        -H "Authorization: Bearer $TOKEN" > /dev/null || true

    # Try deleting bucket again
    DELETE_BUCKET_RESPONSE=$(curl -s -w "\n%{http_code}" -X DELETE "$STORAGE_API/test-e2e-bucket" \
        -H "Authorization: Bearer $TOKEN")
    HTTP_CODE=$(echo "$DELETE_BUCKET_RESPONSE" | tail -n1)
fi

if [ "$HTTP_CODE" = "204" ] || [ "$HTTP_CODE" = "200" ]; then
    pass "Deleted empty bucket successfully"
elif [ "$HTTP_CODE" = "404" ]; then
    pass "Bucket already deleted"
else
    fail "Failed to delete bucket (HTTP $HTTP_CODE)"
fi

# Cleanup temp files
rm -f "$TEST_FILE" /tmp/downloaded-file.txt

# Summary
echo ""
echo "=== Test Summary ==="
echo -e "${GREEN}Passed: $TESTS_PASSED${NC}"
if [ $TESTS_FAILED -gt 0 ]; then
    echo -e "${RED}Failed: $TESTS_FAILED${NC}"
    exit 1
else
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
fi
