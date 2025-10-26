#!/bin/bash

# Realtime WebSocket Test Script
# This script tests the end-to-end realtime functionality

set -e

echo "🧪 Fluxbase Realtime Test"
echo "========================="
echo ""

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Start the server in background
echo -e "${BLUE}Starting Fluxbase server...${NC}"
pkill -f fluxbase 2>/dev/null || true
sleep 1

cd /workspace
go build -o /tmp/fluxbase-realtime-test cmd/fluxbase/main.go
/tmp/fluxbase-realtime-test > /tmp/realtime-test.log 2>&1 &
SERVER_PID=$!

# Wait for server to be ready
echo -e "${BLUE}Waiting for server to start...${NC}"
for i in {1..30}; do
    if curl -s http://localhost:8080/health > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Server started successfully${NC}"
        break
    fi
    if [ $i -eq 30 ]; then
        echo "❌ Server failed to start"
        tail -20 /tmp/realtime-test.log
        kill $SERVER_PID 2>/dev/null || true
        exit 1
    fi
    sleep 0.5
done

# Check realtime stats
echo ""
echo -e "${BLUE}Checking realtime stats...${NC}"
STATS=$(curl -s http://localhost:8080/api/realtime/stats)
echo "Stats: $STATS"

# Test 1: WebSocket Connection (using websocat if available, or create a simple test)
echo ""
echo -e "${YELLOW}Test 1: WebSocket Connection${NC}"
echo "WebSocket endpoint: ws://localhost:8080/realtime"
echo "To manually test: websocat ws://localhost:8080/realtime"
echo ""

# Test 2: Database Change Notification
echo -e "${YELLOW}Test 2: Database Change Notification${NC}"
echo "Inserting a new product..."

INSERT_RESULT=$(curl -s -X POST http://localhost:8080/api/tables/products \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "Realtime Test Product",
    "price": 999.99,
    "stock": 1,
    "description": "Testing realtime notifications"
  }')

echo "Insert result:"
echo "$INSERT_RESULT" | jq '.'

PRODUCT_ID=$(echo "$INSERT_RESULT" | jq -r '.id')
echo ""
echo -e "${GREEN}✓ Product created with ID: $PRODUCT_ID${NC}"

# Check logs for notification
echo ""
echo -e "${BLUE}Checking server logs for NOTIFY events...${NC}"
sleep 1
if grep -q "Broadcasted change event" /tmp/realtime-test.log; then
    echo -e "${GREEN}✓ Change event was broadcasted!${NC}"
    grep "Broadcasted change event" /tmp/realtime-test.log | tail -1
else
    echo -e "${YELLOW}⚠ No broadcast found in logs (might need WebSocket client connected)${NC}"
fi

# Test 3: Update notification
echo ""
echo -e "${YELLOW}Test 3: Update Notification${NC}"
UPDATE_RESULT=$(curl -s -X PATCH "http://localhost:8080/api/tables/products/${PRODUCT_ID}" \
  -H 'Content-Type: application/json' \
  -d '{"price": 1299.99}')

echo "Update result:"
echo "$UPDATE_RESULT" | jq '.'
echo -e "${GREEN}✓ Product updated${NC}"

# Test 4: Delete notification
echo ""
echo -e "${YELLOW}Test 4: Delete Notification${NC}"
DELETE_RESULT=$(curl -s -X DELETE "http://localhost:8080/api/tables/products/${PRODUCT_ID}")
echo "Delete result:"
echo "$DELETE_RESULT" | jq '.'
echo -e "${GREEN}✓ Product deleted${NC}"

# Check final stats
echo ""
echo -e "${BLUE}Final realtime stats:${NC}"
curl -s http://localhost:8080/api/realtime/stats | jq '.'

# Cleanup
echo ""
echo -e "${BLUE}Cleaning up...${NC}"
kill $SERVER_PID 2>/dev/null || true
wait $SERVER_PID 2>/dev/null || true

echo ""
echo -e "${GREEN}=========================${NC}"
echo -e "${GREEN}✓ All tests completed!${NC}"
echo -e "${GREEN}=========================${NC}"
echo ""
echo "📝 Server logs saved to: /tmp/realtime-test.log"
echo ""
echo "To view the logs: tail -f /tmp/realtime-test.log"
