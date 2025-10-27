#!/bin/bash
set -e

echo "🧪 Testing Functions/RPC Filtering"

BASE_URL="http://localhost:8080"

echo ""
echo "1️⃣ Fetching RPC function list..."
FUNCTIONS=$(curl -s "$BASE_URL/api/v1/rpc/")
FUNCTION_COUNT=$(echo "$FUNCTIONS" | jq 'length')
echo "Total functions: $FUNCTION_COUNT"

echo ""
echo "2️⃣ Checking for internal functions (should be 0)..."
INTERNAL_COUNT=$(echo "$FUNCTIONS" | jq '[.[] | select(.name | contains("realtime") or contains("notify") or contains("update_updated") or contains("gin_") or contains("gtrgm_"))] | length')
echo "Internal functions found: $INTERNAL_COUNT"

if [ "$INTERNAL_COUNT" -ne 0 ]; then
  echo "❌ Internal functions should be filtered out!"
  echo "$FUNCTIONS" | jq '[.[] | select(.name | contains("realtime") or contains("notify") or contains("update_updated"))]'
  exit 1
fi
echo "✅ No internal functions found"

echo ""
echo "3️⃣ Checking for user-facing utility functions..."
UUID_COUNT=$(echo "$FUNCTIONS" | jq '[.[] | select(.name | contains("uuid_generate"))] | length')
echo "UUID generation functions: $UUID_COUNT"

if [ "$UUID_COUNT" -eq 0 ]; then
  echo "❌ UUID functions should be available!"
  exit 1
fi
echo "✅ UUID functions available"

echo ""
echo "4️⃣ Testing a UUID generation function..."
UUID_RESULT=$(curl -s -X POST "$BASE_URL/api/v1/rpc/uuid_generate_v4" -H "Content-Type: application/json" -d '{}')
echo "$UUID_RESULT" | jq -e '.result != null' > /dev/null || { echo "❌ UUID generation failed"; exit 1; }
echo "✅ UUID generation working"

echo ""
echo "5️⃣ Verifying enable_realtime is not accessible..."
REALTIME_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/api/v1/rpc/enable_realtime" -H "Content-Type: application/json" -d '{"table_name":"test","schema_name":"public"}')
if [ "$REALTIME_STATUS" -eq 404 ]; then
  echo "✅ enable_realtime correctly returns 404"
else
  echo "❌ enable_realtime should return 404, got $REALTIME_STATUS"
  exit 1
fi

echo ""
echo "6️⃣ Verifying disable_realtime is not accessible..."
DISABLE_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/api/v1/rpc/disable_realtime" -H "Content-Type: application/json" -d '{"table_name":"test","schema_name":"public"}')
if [ "$DISABLE_STATUS" -eq 404 ]; then
  echo "✅ disable_realtime correctly returns 404"
else
  echo "❌ disable_realtime should return 404, got $DISABLE_STATUS"
  exit 1
fi

echo ""
echo "🎉 All Functions/RPC filtering tests passed!"
echo "📊 Summary: $FUNCTION_COUNT user-facing functions exposed (down from 132 total)"
