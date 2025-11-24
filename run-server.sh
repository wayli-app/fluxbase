#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Trap SIGINT (Ctrl+C) and SIGTERM
cleanup() {
    echo ""
    echo -e "${YELLOW}Shutting down server...${NC}"
    if [ ! -z "$SERVER_PID" ]; then
        kill $SERVER_PID 2>/dev/null
        wait $SERVER_PID 2>/dev/null
    fi
    echo -e "${GREEN}Server stopped${NC}"
    exit 0
}

trap cleanup SIGINT SIGTERM

# Add Deno to PATH (required for edge functions)
export PATH="/home/vscode/.deno/bin:$PATH"

echo -e "${YELLOW}Starting fluxbase...${NC}"
echo -e "${GREEN}Server will be available at:${NC}"
echo -e "  ${GREEN}API:${NC}  http://localhost:8080/api/v1/"
echo -e "  ${GREEN}Admin UI:${NC}  http://localhost:8080/admin"
echo -e "  ${GREEN}Health:${NC}  http://localhost:8080/health"
echo -e "${YELLOW}Press Ctrl+C to stop${NC}"
echo ""

# Run the server and capture its PID
go run cmd/fluxbase/main.go &
SERVER_PID=$!

# Wait for the server process
wait $SERVER_PID
