#!/bin/bash
set -e

echo "🧪 Testing Realtime Dashboard API"

BASE_URL="http://localhost:8080"

echo ""
echo "1️⃣ Testing /api/v1/realtime/stats endpoint (empty state)..."
STATS=$(curl -s "$BASE_URL/api/v1/realtime/stats")
echo "Stats Response: $STATS"

# Verify response structure
echo "$STATS" | jq -e '.total_connections != null' > /dev/null || { echo "❌ Missing total_connections"; exit 1; }
echo "$STATS" | jq -e '.total_channels != null' > /dev/null || { echo "❌ Missing total_channels"; exit 1; }
echo "$STATS" | jq -e '.connections != null' > /dev/null || { echo "❌ Missing connections array"; exit 1; }
echo "$STATS" | jq -e '.channels != null' > /dev/null || { echo "❌ Missing channels array"; exit 1; }
echo "✅ Stats endpoint structure valid"

echo ""
echo "2️⃣ Testing /api/v1/realtime/broadcast endpoint..."
BROADCAST_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/realtime/broadcast" \
  -H "Content-Type: application/json" \
  -d '{"channel":"test-channel","message":{"type":"test","data":"hello"}}')
echo "Broadcast Response: $BROADCAST_RESPONSE"

# Verify broadcast response
echo "$BROADCAST_RESPONSE" | jq -e '.success == true' > /dev/null || { echo "❌ Broadcast failed"; exit 1; }
echo "$BROADCAST_RESPONSE" | jq -e '.channel == "test-channel"' > /dev/null || { echo "❌ Wrong channel"; exit 1; }
echo "✅ Broadcast endpoint working"

echo ""
echo "3️⃣ Testing broadcast with missing channel (should fail)..."
ERROR_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/realtime/broadcast" \
  -H "Content-Type: application/json" \
  -d '{"message":{"type":"test"}}')
echo "Error Response: $ERROR_RESPONSE"
echo "$ERROR_RESPONSE" | jq -e '.error != null' > /dev/null || { echo "❌ Should return error"; exit 1; }
echo "✅ Error handling working"

echo ""
echo "4️⃣ Checking Admin UI is accessible..."
curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/admin" | grep -q "200" || { echo "❌ Admin UI not accessible"; exit 1; }
echo "✅ Admin UI accessible"

echo ""
echo "🎉 All Realtime Dashboard API tests passed!"
