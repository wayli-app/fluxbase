#!/bin/bash
set -e

# Simple Storage E2E Test (no auth required for now)
echo "=== Simple Storage E2E Test ==="

API_BASE="http://localhost:8080/api/storage"
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

PASSED=0
FAILED=0

pass() { echo -e "${GREEN}✓ $1${NC}"; ((PASSED++)); }
fail() { echo -e "${RED}✗ $1${NC}"; ((FAILED++)); }

# Test 1: List buckets
echo "Test 1: List buckets"
RESPONSE=$(curl -s "$API_BASE/buckets")
if echo "$RESPONSE" | grep -q "buckets"; then
    pass "Listed buckets"
else
    fail "Failed to list buckets: $RESPONSE"
fi

# Test 2: Create bucket
echo "Test 2: Create bucket"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_BASE/buckets/e2e-test-bucket")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
if [ "$HTTP_CODE" = "201" ] || [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "409" ]; then
    pass "Created bucket (HTTP $HTTP_CODE)"
else
    fail "Failed to create bucket (HTTP $HTTP_CODE)"
fi

# Test 3: Upload file
echo "Test 3: Upload file"
TEST_FILE="/tmp/test-upload.txt"
echo "Hello Fluxbase Storage!" > "$TEST_FILE"

RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_BASE/e2e-test-bucket/test.txt" \
    -F "file=@$TEST_FILE")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
if [ "$HTTP_CODE" = "201" ] || [ "$HTTP_CODE" = "200" ]; then
    pass "Uploaded file (HTTP $HTTP_CODE)"
else
    fail "Failed to upload (HTTP $HTTP_CODE): $(echo "$RESPONSE" | head -n-1)"
fi

# Test 4: Download file
echo "Test 4: Download file"
RESPONSE=$(curl -s -w "\n%{http_code}" -X GET "$API_BASE/e2e-test-bucket/test.txt" \
    -o "/tmp/downloaded.txt")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
if [ "$HTTP_CODE" = "200" ]; then
    ORIG=$(cat "$TEST_FILE")
    DOWN=$(cat "/tmp/downloaded.txt")
    if [ "$ORIG" = "$DOWN" ]; then
        pass "Downloaded file matches"
    else
        fail "File content mismatch"
    fi
else
    fail "Failed to download (HTTP $HTTP_CODE)"
fi

# Test 5: List files
echo "Test 5: List files in bucket"
RESPONSE=$(curl -s "$API_BASE/e2e-test-bucket")
if echo "$RESPONSE" | grep -q "test.txt"; then
    pass "Listed files in bucket"
else
    fail "File not found in bucket listing"
fi

# Test 6: Delete file
echo "Test 6: Delete file"
RESPONSE=$(curl -s -w "\n%{http_code}" -X DELETE "$API_BASE/e2e-test-bucket/test.txt")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
if [ "$HTTP_CODE" = "204" ] || [ "$HTTP_CODE" = "200" ]; then
    pass "Deleted file (HTTP $HTTP_CODE)"
else
    fail "Failed to delete file (HTTP $HTTP_CODE)"
fi

# Test 7: Delete bucket
echo "Test 7: Delete bucket"
RESPONSE=$(curl -s -w "\n%{http_code}" -X DELETE "$API_BASE/buckets/e2e-test-bucket")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
if [ "$HTTP_CODE" = "204" ] || [ "$HTTP_CODE" = "200" ]; then
    pass "Deleted bucket (HTTP $HTTP_CODE)"
else
    fail "Failed to delete bucket (HTTP $HTTP_CODE)"
fi

# Summary
echo ""
echo "=== Test Summary ==="
echo -e "${GREEN}Passed: $PASSED${NC}"
if [ $FAILED -gt 0 ]; then
    echo -e "${RED}Failed: $FAILED${NC}"
    exit 1
else
    echo -e "${GREEN}All tests passed!${NC}"
fi
