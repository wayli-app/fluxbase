#!/bin/bash

# Test All - Run all test suites (backend, SDK, React SDK, integration)
# Usage: ./test-all.sh

set -e  # Exit on first error

echo "╔════════════════════════════════════════════════════════════╗"
echo "║              FLUXBASE - COMPLETE TEST SUITE                ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo ""

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

FAILED=0

# 1. Backend Tests
echo "${YELLOW}[1/4] Running Backend Tests (Go)...${NC}"
if make test; then
    echo "${GREEN}✓ Backend tests passed${NC}"
    echo ""
else
    echo "${RED}✗ Backend tests failed${NC}"
    FAILED=1
fi

# 2. Core SDK Tests
echo "${YELLOW}[2/4] Running Core SDK Tests (TypeScript)...${NC}"
if cd sdk && npm test; then
    echo "${GREEN}✓ Core SDK tests passed${NC}"
    echo ""
    cd ..
else
    echo "${RED}✗ Core SDK tests failed${NC}"
    FAILED=1
    cd ..
fi

# 3. React SDK Build Test
echo "${YELLOW}[3/4] Building React SDK...${NC}"
if cd sdk-react && npm run build; then
    echo "${GREEN}✓ React SDK build passed${NC}"
    echo ""
    cd ..
else
    echo "${RED}✗ React SDK build failed${NC}"
    FAILED=1
    cd ..
fi

# 4. Admin Integration Tests
echo "${YELLOW}[4/4] Running Admin Integration Tests...${NC}"
if cd examples/admin-setup && npm test 2>&1; then
    echo "${GREEN}✓ Admin integration tests passed${NC}"
    echo ""
    cd ../..
else
    echo "${RED}✗ Admin integration tests failed${NC}"
    FAILED=1
    cd ../..
fi

# Summary
echo "╔════════════════════════════════════════════════════════════╗"
echo "║                      TEST SUMMARY                          ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo ""

if [ $FAILED -eq 0 ]; then
    echo "${GREEN}✓ All test suites passed!${NC}"
    echo ""
    exit 0
else
    echo "${RED}✗ Some test suites failed${NC}"
    echo ""
    exit 1
fi
