#!/bin/bash
# Test script to verify devcontainer setup

echo "🧪 Testing Fluxbase DevContainer Setup"
echo "======================================="
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

PASSED=0
FAILED=0

# Test function
test_command() {
  local name=$1
  local command=$2

  echo -n "Testing $name... "
  if eval "$command" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ PASS${NC}"
    ((PASSED++))
  else
    echo -e "${RED}✗ FAIL${NC}"
    ((FAILED++))
  fi
}

# Test Go
echo "📦 Testing Go Environment"
echo "-------------------------"
test_command "Go version" "go version"
test_command "Go modules" "go env GOMODCACHE"
test_command "gopls" "which gopls"
test_command "dlv" "which dlv"
test_command "golangci-lint" "which golangci-lint"
test_command "air" "which air"
test_command "migrate" "which migrate"
echo ""

# Test Node
echo "📦 Testing Node.js Environment"
echo "------------------------------"
test_command "Node.js" "node --version"
test_command "npm" "npm --version"
test_command "TypeScript" "which tsc"
test_command "Prettier" "which prettier"
test_command "ESLint" "which eslint"
echo ""

# Test Database Tools
echo "🗄️  Testing Database Tools"
echo "-------------------------"
test_command "psql" "which psql"
test_command "PostgreSQL connection" "pg_isready -h postgres -U postgres"
test_command "Redis CLI" "which redis-cli"
test_command "Redis connection" "redis-cli -h redis ping"
echo ""

# Test Development Tools
echo "🔧 Testing Development Tools"
echo "----------------------------"
test_command "git" "git --version"
test_command "gh (GitHub CLI)" "which gh"
test_command "docker" "which docker"
test_command "make" "which make"
test_command "jq" "which jq"
test_command "httpie" "which http"
echo ""

# Test Project Structure
echo "📁 Testing Project Structure"
echo "----------------------------"
test_command "Workspace mounted" "test -d /workspace"
test_command "go.mod exists" "test -f /workspace/go.mod"
test_command "Makefile exists" "test -f /workspace/Makefile"
test_command "TODO.md exists" "test -f /workspace/TODO.md"
test_command ".env file" "test -f /workspace/.env"
test_command "Storage directory" "test -d /workspace/storage"
echo ""

# Test Database
echo "🗄️  Testing Database Setup"
echo "-------------------------"
test_command "fluxbase_dev database" "psql -h postgres -U postgres -lqt | cut -d \| -f 1 | grep -qw fluxbase_dev"
test_command "fluxbase_test database" "psql -h postgres -U postgres -lqt | cut -d \| -f 1 | grep -qw fluxbase_test"
test_command "uuid-ossp extension" "psql -h postgres -U postgres -d fluxbase_dev -tAc \"SELECT 1 FROM pg_extension WHERE extname='uuid-ossp'\""
echo ""

# Test Go Project
echo "🚀 Testing Go Project"
echo "--------------------"
cd /workspace
test_command "Go mod download" "go mod download"
test_command "Go build" "go build -o /tmp/fluxbase-test cmd/fluxbase/main.go && rm /tmp/fluxbase-test"
echo ""

# Summary
echo ""
echo "======================================="
echo "📊 Test Summary"
echo "======================================="
echo -e "Passed: ${GREEN}$PASSED${NC}"
echo -e "Failed: ${RED}$FAILED${NC}"
echo ""

if [ $FAILED -eq 0 ]; then
  echo -e "${GREEN}✨ All tests passed! DevContainer is ready for development.${NC}"
  echo ""
  echo "Next steps:"
  echo "  1. Run 'make dev' to start the development server"
  echo "  2. Run 'make test' to run the test suite"
  echo "  3. Check TODO.md for the implementation plan"
  echo ""
  exit 0
else
  echo -e "${RED}❌ Some tests failed. Please check the output above.${NC}"
  echo ""
  echo "Troubleshooting:"
  echo "  1. Try rebuilding: F1 → 'Dev Containers: Rebuild Container'"
  echo "  2. Check Docker Desktop is running"
  echo "  3. Check logs: docker-compose logs"
  echo ""
  exit 1
fi
