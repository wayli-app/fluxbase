#!/bin/bash

echo "=== Fluxbase Storage E2E Test ==="
echo ""

API="http://localhost:8080/api/storage"
BUCKET="test-bucket-$$"  # Use PID to make unique
PASSED=0
FAILED=0

# Test 1: List buckets
echo "Test 1: List buckets"
curl -s "$API/buckets" | grep -q "buckets" && echo "✓ PASS" && ((PASSED++)) || { echo "✗ FAIL"; ((FAILED++)); }

# Test 2: Create bucket  
echo "Test 2: Create bucket"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$API/buckets/$BUCKET")
[ "$HTTP_CODE" = "201" ] && echo "✓ PASS (201)" && ((PASSED++)) || { echo "✗ FAIL ($HTTP_CODE)"; ((FAILED++)); }

# Test 3: Upload file
echo "Test 3: Upload file"
echo "Test content" > /tmp/test.txt
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$API/$BUCKET/test.txt" -F "file=@/tmp/test.txt")
[ "$HTTP_CODE" = "201" ] && echo "✓ PASS (201)" && ((PASSED++)) || { echo "✗ FAIL ($HTTP_CODE)"; ((FAILED++)); }

# Test 4: Download file
echo "Test 4: Download file"
curl -s "$API/$BUCKET/test.txt" -o /tmp/downloaded.txt
diff /tmp/test.txt /tmp/downloaded.txt > /dev/null && echo "✓ PASS" && ((PASSED++)) || { echo "✗ FAIL"; ((FAILED++)); }

# Test 5: List files
echo "Test 5: List files"
curl -s "$API/$BUCKET" | grep -q "test.txt" && echo "✓ PASS" && ((PASSED++)) || { echo "✗ FAIL"; ((FAILED++)); }

# Test 6: Delete file
echo "Test 6: Delete file"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$API/$BUCKET/test.txt")
[ "$HTTP_CODE" = "204" ] || [ "$HTTP_CODE" = "200" ] && echo "✓ PASS ($HTTP_CODE)" && ((PASSED++)) || { echo "✗ FAIL ($HTTP_CODE)"; ((FAILED++)); }

# Test 7: Delete bucket
echo "Test 7: Delete bucket"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$API/buckets/$BUCKET")
[ "$HTTP_CODE" = "204" ] || [ "$HTTP_CODE" = "200" ] && echo "✓ PASS ($HTTP_CODE)" && ((PASSED++)) || { echo "✗ FAIL ($HTTP_CODE)"; ((FAILED++)); }

echo ""
echo "=== Results ==="
echo "Passed: $PASSED/7"
echo "Failed: $FAILED/7"
[ $FAILED -eq 0 ] && echo "✓ ALL TESTS PASSED!" && exit 0 || { echo "✗ SOME TESTS FAILED"; exit 1; }
