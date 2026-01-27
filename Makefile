.PHONY: help dev build clean test migrate-up migrate-down migrate-create db-reset db-reset-full deps setup-dev install-hooks uninstall-hooks docs docs-build version docker-build docker-push release cli cli-install cli-completions

# Variables
BINARY_NAME=fluxbase-server
CLI_BINARY_NAME=fluxbase
MAIN_PATH=cmd/fluxbase/main.go
CLI_MAIN_PATH=cli/main.go

# Version variables
VERSION := $(shell cat VERSION)
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -s -w -X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildDate=$(BUILD_DATE)

# Docker variables
DOCKER_REGISTRY ?= ghcr.io
DOCKER_ORG ?= fluxbase-eu
DOCKER_IMAGE := $(DOCKER_REGISTRY)/$(DOCKER_ORG)/fluxbase

# Colors for output
GREEN=\033[0;32m
YELLOW=\033[1;33m
BLUE=\033[0;34m
NC=\033[0m # No Color

# Default target
.DEFAULT_GOAL := help

help: ## Show available commands
	@echo "╔════════════════════════════════════════════════════════════╗"
	@echo "║                     FLUXBASE COMMANDS                      ║"
	@echo "╚════════════════════════════════════════════════════════════╝"
	@echo ""
	@echo "${GREEN}Quick Start:${NC}"
	@echo "  make dev            # Build & run backend + frontend (all-in-one)"
	@echo "  make build          # Build production binary with embedded UI"
	@echo "  make test-all       # Run ALL tests (backend + SDK + React + integration)"
	@echo ""
	@echo "${GREEN}All Commands:${NC}"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  ${GREEN}%-20s${NC} %s\\n", $$1, $$2}'

dev: ## Build and run backend + frontend dev server (all-in-one)
	@echo "${YELLOW}Starting Fluxbase development environment...${NC}"
	@lsof -ti:8080 | xargs -r kill -9 2>/dev/null || true
	@lsof -ti:5050 | xargs -r kill -9 2>/dev/null || true
	@if [ ! -d "sdk/node_modules" ]; then \
		echo "${YELLOW}Installing SDK dependencies...${NC}"; \
		cd sdk && unset NODE_OPTIONS && npm install; \
	fi
	@echo "${YELLOW}Generating embedded SDK for job runtime...${NC}"
	@cd sdk && unset NODE_OPTIONS && npm run generate:embedded-sdk
	@if [ ! -d "admin/node_modules" ]; then \
		echo "${YELLOW}Installing admin UI dependencies...${NC}"; \
		cd admin && unset NODE_OPTIONS && npm install; \
	fi
	@echo "${YELLOW}Building admin UI...${NC}"
	@cd admin && unset NODE_OPTIONS && npm run build
	@rm -rf internal/adminui/dist
	@cp -r admin/dist internal/adminui/dist
	@echo "${GREEN}Backend:${NC}     http://localhost:8080"
	@echo "${GREEN}Frontend:${NC}    http://localhost:5050/admin/"
	@echo "${GREEN}Admin Login:${NC} http://localhost:5050/admin/login"
	@echo ""
	@echo "${YELLOW}Press Ctrl+C to stop both servers${NC}"
	@echo ""
	@bash -c 'trap "kill 0" EXIT; ./run-server.sh & SERVER_PID=$$!; cd admin && unset NODE_OPTIONS && npm run dev & NPM_PID=$$!; wait -n 2>/dev/null || while kill -0 $$SERVER_PID 2>/dev/null && kill -0 $$NPM_PID 2>/dev/null; do sleep 1; done'

version: ## Show version information
	@echo "${GREEN}Version:${NC}    $(VERSION)"
	@echo "${GREEN}Commit:${NC}     $(COMMIT)"
	@echo "${GREEN}Build Date:${NC} $(BUILD_DATE)"

build: ## Build production binary with embedded admin UI
	@echo "${YELLOW}Generating embedded SDK for job runtime...${NC}"
	@cd sdk && unset NODE_OPTIONS && npm run generate:embedded-sdk
	@echo "${YELLOW}Building admin UI...${NC}"
	@cd admin && unset NODE_OPTIONS && npm run build
	@rm -rf internal/adminui/dist
	@cp -r admin/dist internal/adminui/dist
	@echo "${YELLOW}Building ${BINARY_NAME} v$(VERSION)...${NC}"
	@mkdir -p build/
	@go build -tags "ocr" -ldflags="$(LDFLAGS)" -o build/${BINARY_NAME} ${MAIN_PATH}
	@echo "${GREEN}Build complete: ${BINARY_NAME} v$(VERSION)${NC}"

clean: ## Clean build artifacts
	@echo "${YELLOW}Cleaning...${NC}"
	@rm -f build/${BINARY_NAME}
	@rm -f coverage.out coverage.html
	@rm -rf internal/adminui/dist
	@echo "${GREEN}Clean complete!${NC}"

test: ## Run all tests with race detector (short mode - skips slow tests, excludes e2e)
	@./scripts/test-runner.sh go test -timeout 2m -v -race -short -cover $(shell go list ./... | grep -v '/test/e2e')

test-coverage: ## Run tests and generate coverage report with enforcement (Go + SDK)
	@echo "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
	@echo "${BLUE}║                 COVERAGE REPORT                            ║${NC}"
	@echo "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
	@echo ""
	@echo "${YELLOW}[1/4] Running Go unit tests with coverage...${NC}"
	@go test -short -timeout 5m -coverprofile=coverage.out -covermode=atomic $(shell go list ./... | grep -v '/test/e2e' | grep -v '/test$$')
	@echo ""
	@echo "${YELLOW}[2/4] Enforcing coverage thresholds...${NC}"
	@go-test-coverage --config=.testcoverage.yml
	@echo ""
	@echo "${YELLOW}[3/4] Generating Go coverage report...${NC}"
	@go tool cover -html=coverage.out -o coverage.html
	@go tool cover -func=coverage.out | grep total | awk '{print "  ${GREEN}Go Coverage: " $$3 "${NC}"}'
	@echo ""
	@echo "${YELLOW}[4/4] Running SDK tests with coverage...${NC}"
	@cd sdk && unset NODE_OPTIONS && npx vitest --coverage --run 2>&1 | tail -20 || true
	@echo ""
	@echo "${GREEN}Coverage reports generated:${NC}"
	@echo "  - coverage.out     (Go profile)"
	@echo "  - coverage.html    (Go HTML report)"
	@echo "  - sdk/coverage/    (SDK coverage)"

test-coverage-check: ## Check coverage thresholds without running tests (requires coverage.out)
	@go-test-coverage --config=.testcoverage.yml

test-fast: ## Run all tests without race detector (faster, excludes e2e)
	@./scripts/test-runner.sh go test -timeout 1m -v -short -cover $(shell go list ./... | grep -v '/test/e2e')

test-full: ## Run ALL tests including e2e with race detector (may take 5-10 minutes)
	@./scripts/test-runner.sh go test -timeout 15m -v -race -cover ./...

test-e2e: ## Run e2e tests only (requires postgres, mailhog, minio services). Use RUN= to filter tests.
	@./scripts/test-runner.sh go test -v -race -parallel=1 -timeout=5m ./test/e2e/... $(if $(RUN),-run $(RUN),)

test-e2e-fast: ## Run e2e tests without race detector (faster for dev iteration). Use RUN= to filter tests.
	@./scripts/test-runner.sh go test -v -parallel=1 -timeout=3m ./test/e2e/... $(if $(RUN),-run $(RUN),)

test-auth: ## Run authentication tests only
	@./scripts/test-runner.sh go test -v -race -timeout=5m ./test/e2e/ -run TestAuth

test-rls: ## Run RLS security tests only
	@./scripts/test-runner.sh go test -v -race -timeout=5m ./test/e2e/ -run TestRLS

test-rest: ## Run REST API tests only
	@./scripts/test-runner.sh go test -v -race -timeout=5m ./test/e2e/ -run TestREST

test-storage: ## Run storage tests only
	@./scripts/test-runner.sh go test -v -race -timeout=5m ./test/e2e/ -run TestStorage

test-sdk: ## Run SDK tests (TypeScript)
	@echo "${YELLOW}Running SDK tests...${NC}"
	@cd sdk && unset NODE_OPTIONS && npm test -- src/admin.test.ts src/auth.test.ts src/management.test.ts src/ddl.test.ts src/impersonation.test.ts src/settings.test.ts src/oauth.test.ts
	@echo "${GREEN}SDK tests complete!${NC}"

test-sdk-react: ## Build React SDK (includes type checking)
	@echo "${YELLOW}Building React SDK...${NC}"
	@cd sdk-react && unset NODE_OPTIONS && npm run build
	@echo "${GREEN}React SDK build complete!${NC}"

test-integration: ## Run admin integration tests (requires running server)
	@echo "${YELLOW}Running admin integration tests...${NC}"
	@if ! curl -s http://localhost:8080/health > /dev/null; then \
		echo "${RED}Error: Fluxbase server not running on localhost:8080${NC}"; \
		echo "${YELLOW}Start server with: make dev${NC}"; \
		exit 1; \
	fi
	@cd examples/admin-setup && unset NODE_OPTIONS && npm test
	@echo "${GREEN}Integration tests complete!${NC}"

test-all: ## Run ALL tests (backend + SDK + React + integration)
	@echo "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
	@echo "${BLUE}║              FLUXBASE - COMPLETE TEST SUITE                ║${NC}"
	@echo "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
	@echo ""
	@echo "${YELLOW}[1/4] Running Backend Tests (Go)...${NC}"
	@$(MAKE) test
	@echo ""
	@echo "${YELLOW}[2/4] Running Core SDK Tests (TypeScript)...${NC}"
	@$(MAKE) test-sdk
	@echo ""
	@echo "${YELLOW}[3/4] Building React SDK...${NC}"
	@$(MAKE) test-sdk-react
	@echo ""
	@echo "${YELLOW}[4/4] Running Admin Integration Tests...${NC}"
	@$(MAKE) test-integration || true
	@echo ""
	@echo "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
	@echo "${BLUE}║                      TEST SUMMARY                          ║${NC}"
	@echo "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
	@echo "${GREEN}✓ All test suites complete!${NC}"

deps: ## Install Go dependencies
	@echo "${YELLOW}Installing dependencies...${NC}"
	@go mod download
	@go mod tidy
	@echo "${GREEN}Dependencies installed!${NC}"

setup-dev: ## Set up development environment (first-time setup)
	@echo "${YELLOW}Setting up development environment...${NC}"
	@go mod download
	@go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@go install github.com/vladopajic/go-test-coverage/v2@latest
	@cd admin && unset NODE_OPTIONS && npm install
	@cp .env.example .env 2>/dev/null || echo ".env already exists"
	@$(MAKE) install-hooks
	@echo "${GREEN}Development environment ready!${NC}"
	@echo "${YELLOW}Next steps:${NC}"
	@echo "  1. Configure your database in .env"
	@echo "  2. Run: make migrate-up"
	@echo "  3. Run: make dev"

install-hooks: ## Install git pre-commit hooks
	@echo "${YELLOW}Installing git pre-commit hooks...${NC}"
	@cp scripts/pre-commit .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "${GREEN}✓ Pre-commit hook installed${NC}"
	@echo "${YELLOW}The hook will run go fmt and TypeScript type checking before commits${NC}"
	@echo "${YELLOW}To skip: git commit --no-verify${NC}"

uninstall-hooks: ## Uninstall git pre-commit hooks
	@echo "${YELLOW}Uninstalling git pre-commit hooks...${NC}"
	@rm -f .git/hooks/pre-commit
	@echo "${GREEN}✓ Pre-commit hook uninstalled${NC}"

migrate-up: ## Run database migrations
	@echo "${YELLOW}Running migrations...${NC}"
	@migrate -path internal/database/migrations -database "postgresql://postgres:postgres@localhost:5432/fluxbase?sslmode=disable" up
	@echo "${GREEN}Migrations complete!${NC}"

migrate-down: ## Rollback last migration
	@echo "${YELLOW}Rolling back migration...${NC}"
	@migrate -path internal/database/migrations -database "postgresql://postgres:postgres@localhost:5432/fluxbase?sslmode=disable" down 1
	@echo "${GREEN}Rollback complete!${NC}"

migrate-create: ## Create new migration (usage: make migrate-create name=add_users_table)
	@if [ -z "$(name)" ]; then \
		echo "${YELLOW}Error: Provide migration name${NC}"; \
		echo "Usage: make migrate-create name=add_users_table"; \
		exit 1; \
	fi
	@echo "${YELLOW}Creating migration: $(name)...${NC}"
	@migrate create -ext sql -dir internal/database/migrations -seq $(name)
	@echo "${GREEN}Migration files created!${NC}"

db-reset: ## Reset database (preserves public, auth.users, dashboard.users, setup_completed). Use db-reset-full for full reset.
	@echo "${YELLOW}Resetting database (preserving public schema, user data, setup_completed)...${NC}"
	@# Backup user data and settings before dropping schemas
	@echo "${YELLOW}Backing up user data...${NC}"
	@PGPASSWORD=postgres psql -h postgres -U postgres -d fluxbase_dev -c "DROP TABLE IF EXISTS _fluxbase_auth_users_backup; CREATE TABLE _fluxbase_auth_users_backup AS SELECT * FROM auth.users;" 2>/dev/null || true
	@PGPASSWORD=postgres psql -h postgres -U postgres -d fluxbase_dev -c "DROP TABLE IF EXISTS _fluxbase_dashboard_users_backup; CREATE TABLE _fluxbase_dashboard_users_backup AS SELECT * FROM dashboard.users;" 2>/dev/null || true
	@PGPASSWORD=postgres psql -h postgres -U postgres -d fluxbase_dev -c "DROP TABLE IF EXISTS _fluxbase_setup_backup; CREATE TABLE _fluxbase_setup_backup AS SELECT * FROM app.settings WHERE key = 'setup_completed';" 2>/dev/null || true
	@# Drop all schemas (including auth) for clean migration
	@PGPASSWORD=postgres psql -h postgres -U postgres -d fluxbase_dev -c "DROP SCHEMA IF EXISTS app CASCADE;" || true
	@PGPASSWORD=postgres psql -h postgres -U postgres -d fluxbase_dev -c "DROP SCHEMA IF EXISTS auth CASCADE;" || true
	@PGPASSWORD=postgres psql -h postgres -U postgres -d fluxbase_dev -c "DROP SCHEMA IF EXISTS dashboard CASCADE;" || true
	@PGPASSWORD=postgres psql -h postgres -U postgres -d fluxbase_dev -c "DROP SCHEMA IF EXISTS storage CASCADE;" || true
	@PGPASSWORD=postgres psql -h postgres -U postgres -d fluxbase_dev -c "DROP SCHEMA IF EXISTS functions CASCADE;" || true
	@PGPASSWORD=postgres psql -h postgres -U postgres -d fluxbase_dev -c "DROP SCHEMA IF EXISTS jobs CASCADE;" || true
	@PGPASSWORD=postgres psql -h postgres -U postgres -d fluxbase_dev -c "DROP SCHEMA IF EXISTS realtime CASCADE;" || true
	@PGPASSWORD=postgres psql -h postgres -U postgres -d fluxbase_dev -c "DROP SCHEMA IF EXISTS ai CASCADE;" || true
	@PGPASSWORD=postgres psql -h postgres -U postgres -d fluxbase_dev -c "DROP SCHEMA IF EXISTS rpc CASCADE;" || true
	@PGPASSWORD=postgres psql -h postgres -U postgres -d fluxbase_dev -c "DROP SCHEMA IF EXISTS branching CASCADE;" || true
	@PGPASSWORD=postgres psql -h postgres -U postgres -d fluxbase_dev -c "DROP SCHEMA IF EXISTS migrations CASCADE;" || true
	@PGPASSWORD=postgres psql -h postgres -U postgres -d fluxbase_dev -c "CREATE SCHEMA IF NOT EXISTS migrations;" || true
	@echo "${YELLOW}Ensuring test users exist with correct permissions...${NC}"
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "DO \$$\$$ BEGIN IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'fluxbase_app') THEN CREATE USER fluxbase_app WITH PASSWORD 'fluxbase_app_password' LOGIN CREATEDB BYPASSRLS; END IF; END \$$\$$;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "DO \$$\$$ BEGIN IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'fluxbase_rls_test') THEN CREATE USER fluxbase_rls_test WITH PASSWORD 'fluxbase_rls_test_password' LOGIN; END IF; END \$$\$$;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "ALTER USER postgres WITH BYPASSRLS;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "ALTER USER postgres SET search_path TO public;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "ALTER USER fluxbase_app WITH BYPASSRLS;" || true
	@echo "${YELLOW}Running migrations...${NC}"
	@migrate -path internal/database/migrations -database 'postgresql://postgres:postgres@postgres:5432/fluxbase_dev?sslmode=disable&x-migrations-table="migrations"."fluxbase"&x-migrations-table-quoted=1' up
	@echo "${YELLOW}Granting permissions to test users (fluxbase_app, fluxbase_rls_test)...${NC}"
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT CREATE ON DATABASE fluxbase_dev TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT USAGE, CREATE ON SCHEMA app TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT USAGE, CREATE ON SCHEMA auth TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT USAGE, CREATE ON SCHEMA dashboard TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT USAGE, CREATE ON SCHEMA functions TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT USAGE, CREATE ON SCHEMA jobs TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT USAGE, CREATE ON SCHEMA storage TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT USAGE, CREATE ON SCHEMA realtime TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT USAGE, CREATE ON SCHEMA migrations TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT USAGE, CREATE ON SCHEMA ai TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT USAGE, CREATE ON SCHEMA rpc TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT USAGE, CREATE ON SCHEMA branching TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT ALL ON ALL TABLES IN SCHEMA app TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT ALL ON ALL SEQUENCES IN SCHEMA app TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT ALL ON ALL TABLES IN SCHEMA auth TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT ALL ON ALL SEQUENCES IN SCHEMA auth TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT ALL ON ALL TABLES IN SCHEMA dashboard TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT ALL ON ALL SEQUENCES IN SCHEMA dashboard TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT ALL ON ALL TABLES IN SCHEMA functions TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT ALL ON ALL SEQUENCES IN SCHEMA functions TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT ALL ON ALL TABLES IN SCHEMA jobs TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT ALL ON ALL SEQUENCES IN SCHEMA jobs TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT ALL ON ALL TABLES IN SCHEMA storage TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT ALL ON ALL SEQUENCES IN SCHEMA storage TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT ALL ON ALL TABLES IN SCHEMA realtime TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT ALL ON ALL SEQUENCES IN SCHEMA realtime TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT ALL ON ALL TABLES IN SCHEMA migrations TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT ALL ON ALL SEQUENCES IN SCHEMA migrations TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT ALL ON ALL TABLES IN SCHEMA ai TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT ALL ON ALL SEQUENCES IN SCHEMA ai TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT ALL ON ALL TABLES IN SCHEMA branching TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT ALL ON ALL SEQUENCES IN SCHEMA branching TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA auth TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA storage TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO fluxbase_app, fluxbase_rls_test;" || true
	@echo "${YELLOW}Granting role memberships for SET ROLE support...${NC}"
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "DO \$$\$$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_auth_members WHERE roleid = (SELECT oid FROM pg_roles WHERE rolname = 'anon') AND member = (SELECT oid FROM pg_roles WHERE rolname = 'fluxbase_app')) THEN GRANT anon TO fluxbase_app; END IF; IF NOT EXISTS (SELECT 1 FROM pg_auth_members WHERE roleid = (SELECT oid FROM pg_roles WHERE rolname = 'authenticated') AND member = (SELECT oid FROM pg_roles WHERE rolname = 'fluxbase_app')) THEN GRANT authenticated TO fluxbase_app; END IF; IF NOT EXISTS (SELECT 1 FROM pg_auth_members WHERE roleid = (SELECT oid FROM pg_roles WHERE rolname = 'service_role') AND member = (SELECT oid FROM pg_roles WHERE rolname = 'fluxbase_app')) THEN GRANT service_role TO fluxbase_app; END IF; IF NOT EXISTS (SELECT 1 FROM pg_auth_members WHERE roleid = (SELECT oid FROM pg_roles WHERE rolname = 'anon') AND member = (SELECT oid FROM pg_roles WHERE rolname = 'fluxbase_rls_test')) THEN GRANT anon TO fluxbase_rls_test; END IF; IF NOT EXISTS (SELECT 1 FROM pg_auth_members WHERE roleid = (SELECT oid FROM pg_roles WHERE rolname = 'authenticated') AND member = (SELECT oid FROM pg_roles WHERE rolname = 'fluxbase_rls_test')) THEN GRANT authenticated TO fluxbase_rls_test; END IF; IF NOT EXISTS (SELECT 1 FROM pg_auth_members WHERE roleid = (SELECT oid FROM pg_roles WHERE rolname = 'service_role') AND member = (SELECT oid FROM pg_roles WHERE rolname = 'fluxbase_rls_test')) THEN GRANT service_role TO fluxbase_rls_test; END IF; END \$$\$$;" || true
	@# Restore user data from backups
	@echo "${YELLOW}Restoring user data from backups...${NC}"
	@PGPASSWORD=postgres psql -h postgres -U postgres -d fluxbase_dev -c "INSERT INTO auth.users SELECT * FROM _fluxbase_auth_users_backup ON CONFLICT (id) DO NOTHING; DROP TABLE IF EXISTS _fluxbase_auth_users_backup;" 2>/dev/null || true
	@PGPASSWORD=postgres psql -h postgres -U postgres -d fluxbase_dev -c "INSERT INTO dashboard.users SELECT * FROM _fluxbase_dashboard_users_backup ON CONFLICT (id) DO NOTHING; DROP TABLE IF EXISTS _fluxbase_dashboard_users_backup;" 2>/dev/null || true
	@PGPASSWORD=postgres psql -h postgres -U postgres -d fluxbase_dev -c "INSERT INTO app.settings SELECT * FROM _fluxbase_setup_backup ON CONFLICT (key) WHERE user_id IS NULL DO UPDATE SET value = EXCLUDED.value, updated_at = NOW(); DROP TABLE IF EXISTS _fluxbase_setup_backup;" 2>/dev/null || true
	@echo "${GREEN}Database reset complete!${NC}"
	@echo "${BLUE}Note: Migrations granted all permissions to the user running them (postgres).${NC}"
	@echo "${BLUE}Additional permissions granted to fluxbase_app and fluxbase_rls_test for testing.${NC}"

db-reset-full: ## Full database reset (drops ALL schemas including public, auth, migrations). WARNING: Destroys all data!
	@echo "${RED}WARNING: Full database reset - this will destroy ALL data including users and migrations!${NC}"
	@echo "${YELLOW}Dropping ALL schemas...${NC}"
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "DROP SCHEMA IF EXISTS app CASCADE;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "DROP SCHEMA IF EXISTS auth CASCADE;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "DROP SCHEMA IF EXISTS dashboard CASCADE;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "DROP SCHEMA IF EXISTS storage CASCADE;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "DROP SCHEMA IF EXISTS functions CASCADE;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "DROP SCHEMA IF EXISTS jobs CASCADE;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "DROP SCHEMA IF EXISTS realtime CASCADE;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "DROP SCHEMA IF EXISTS ai CASCADE;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "DROP SCHEMA IF EXISTS rpc CASCADE;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "DROP SCHEMA IF EXISTS branching CASCADE;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "DROP SCHEMA IF EXISTS migrations CASCADE;" || true
	@echo "${YELLOW}Dropping and recreating public schema...${NC}"
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "DROP SCHEMA IF EXISTS public CASCADE;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "CREATE SCHEMA IF NOT EXISTS public;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "CREATE SCHEMA IF NOT EXISTS migrations;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT ALL ON SCHEMA public TO postgres;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT ALL ON SCHEMA public TO public;" || true
	@echo "${YELLOW}Ensuring test users exist with correct permissions...${NC}"
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "DO \$$\$$ BEGIN IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'fluxbase_app') THEN CREATE USER fluxbase_app WITH PASSWORD 'fluxbase_app_password' LOGIN CREATEDB BYPASSRLS; END IF; END \$$\$$;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "DO \$$\$$ BEGIN IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'fluxbase_rls_test') THEN CREATE USER fluxbase_rls_test WITH PASSWORD 'fluxbase_rls_test_password' LOGIN; END IF; END \$$\$$;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "ALTER USER postgres WITH BYPASSRLS;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "ALTER USER postgres SET search_path TO public;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "ALTER USER fluxbase_app WITH BYPASSRLS;" || true
	@echo "${YELLOW}Running migrations...${NC}"
	@migrate -path internal/database/migrations -database 'postgresql://postgres:postgres@postgres:5432/fluxbase_dev?sslmode=disable&x-migrations-table="migrations"."fluxbase"&x-migrations-table-quoted=1' up
	@echo "${YELLOW}Granting permissions to test users (fluxbase_app, fluxbase_rls_test)...${NC}"
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT CREATE ON DATABASE fluxbase_dev TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT USAGE, CREATE ON SCHEMA app TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT USAGE, CREATE ON SCHEMA auth TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT USAGE, CREATE ON SCHEMA dashboard TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT USAGE, CREATE ON SCHEMA functions TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT USAGE, CREATE ON SCHEMA jobs TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT USAGE, CREATE ON SCHEMA storage TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT USAGE, CREATE ON SCHEMA realtime TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT USAGE, CREATE ON SCHEMA migrations TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT USAGE, CREATE ON SCHEMA ai TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT USAGE, CREATE ON SCHEMA rpc TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT USAGE, CREATE ON SCHEMA branching TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT ALL ON ALL TABLES IN SCHEMA app TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT ALL ON ALL SEQUENCES IN SCHEMA app TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT ALL ON ALL TABLES IN SCHEMA auth TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT ALL ON ALL SEQUENCES IN SCHEMA auth TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT ALL ON ALL TABLES IN SCHEMA dashboard TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT ALL ON ALL SEQUENCES IN SCHEMA dashboard TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT ALL ON ALL TABLES IN SCHEMA functions TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT ALL ON ALL SEQUENCES IN SCHEMA functions TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT ALL ON ALL TABLES IN SCHEMA jobs TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT ALL ON ALL SEQUENCES IN SCHEMA jobs TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT ALL ON ALL TABLES IN SCHEMA storage TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT ALL ON ALL SEQUENCES IN SCHEMA storage TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT ALL ON ALL TABLES IN SCHEMA realtime TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT ALL ON ALL SEQUENCES IN SCHEMA realtime TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT ALL ON ALL TABLES IN SCHEMA migrations TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT ALL ON ALL SEQUENCES IN SCHEMA migrations TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT ALL ON ALL TABLES IN SCHEMA ai TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT ALL ON ALL SEQUENCES IN SCHEMA ai TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT ALL ON ALL TABLES IN SCHEMA branching TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT ALL ON ALL SEQUENCES IN SCHEMA branching TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA auth TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA storage TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO fluxbase_app, fluxbase_rls_test;" || true
	@echo "${YELLOW}Granting role memberships for SET ROLE support...${NC}"
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "DO \$$\$$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_auth_members WHERE roleid = (SELECT oid FROM pg_roles WHERE rolname = 'anon') AND member = (SELECT oid FROM pg_roles WHERE rolname = 'fluxbase_app')) THEN GRANT anon TO fluxbase_app; END IF; IF NOT EXISTS (SELECT 1 FROM pg_auth_members WHERE roleid = (SELECT oid FROM pg_roles WHERE rolname = 'authenticated') AND member = (SELECT oid FROM pg_roles WHERE rolname = 'fluxbase_app')) THEN GRANT authenticated TO fluxbase_app; END IF; IF NOT EXISTS (SELECT 1 FROM pg_auth_members WHERE roleid = (SELECT oid FROM pg_roles WHERE rolname = 'service_role') AND member = (SELECT oid FROM pg_roles WHERE rolname = 'fluxbase_app')) THEN GRANT service_role TO fluxbase_app; END IF; IF NOT EXISTS (SELECT 1 FROM pg_auth_members WHERE roleid = (SELECT oid FROM pg_roles WHERE rolname = 'anon') AND member = (SELECT oid FROM pg_roles WHERE rolname = 'fluxbase_rls_test')) THEN GRANT anon TO fluxbase_rls_test; END IF; IF NOT EXISTS (SELECT 1 FROM pg_auth_members WHERE roleid = (SELECT oid FROM pg_roles WHERE rolname = 'authenticated') AND member = (SELECT oid FROM pg_roles WHERE rolname = 'fluxbase_rls_test')) THEN GRANT authenticated TO fluxbase_rls_test; END IF; IF NOT EXISTS (SELECT 1 FROM pg_auth_members WHERE roleid = (SELECT oid FROM pg_roles WHERE rolname = 'service_role') AND member = (SELECT oid FROM pg_roles WHERE rolname = 'fluxbase_rls_test')) THEN GRANT service_role TO fluxbase_rls_test; END IF; END \$$\$$;" || true
	@echo "${GREEN}Full database reset complete!${NC}"

docs: ## Serve Starlight documentation at http://localhost:4321
	@echo "${YELLOW}Starting Starlight documentation server...${NC}"
	@if [ ! -d "docs/node_modules" ]; then \
		echo "${YELLOW}Installing documentation dependencies...${NC}"; \
		cd docs && unset NODE_OPTIONS && npm install; \
	fi
	@echo ""
	@echo "${GREEN}📚 Documentation will be available at:${NC}"
	@echo "  ${GREEN}http://localhost:4321${NC}"
	@echo ""
	@echo "${YELLOW}Press Ctrl+C to stop the server${NC}"
	@echo ""
	@cd docs && unset NODE_OPTIONS && npm run dev -- --host 0.0.0.0

docs-build: ## Build static documentation site for production
	@echo "${YELLOW}Building documentation site...${NC}"
	@if [ ! -d "docs/node_modules" ]; then \
		echo "${YELLOW}Installing documentation dependencies...${NC}"; \
		cd docs && unset NODE_OPTIONS && npm install; \
	fi
	@cd docs && unset NODE_OPTIONS && npm run build
	@echo "${GREEN}Documentation built successfully!${NC}"
	@echo "${YELLOW}Output:${NC} docs/dist/"
	@echo "${YELLOW}To preview locally:${NC} cd docs && npm run preview"

docker-build-docs: ## Build documentation Docker image
	@echo "${YELLOW}Building documentation Docker image...${NC}"
	@docker build \
		-t $(DOCKER_IMAGE)-docs:$(VERSION) \
		-t $(DOCKER_IMAGE)-docs:latest \
		-f Dockerfile.docs .
	@echo "${GREEN}Documentation Docker image built!${NC}"
	@echo "${YELLOW}To run locally:${NC} docker run -p 8080:8080 $(DOCKER_IMAGE)-docs:latest"
	@echo "${YELLOW}Access at:${NC} http://localhost:8080"

docker-build: ## Build Docker image
	@echo "${YELLOW}Building Docker image $(DOCKER_IMAGE):$(VERSION)...${NC}"
	@docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		-t $(DOCKER_IMAGE):$(VERSION) \
		-t $(DOCKER_IMAGE):latest \
		-f Dockerfile .
	@echo "${GREEN}Docker image built: $(DOCKER_IMAGE):$(VERSION)${NC}"

docker-build-production: ## Build production Docker image with admin UI
	@echo "${YELLOW}Building production Docker image with admin UI...${NC}"
	@docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		-t $(DOCKER_IMAGE):$(VERSION) \
		-t $(DOCKER_IMAGE):latest \
		-f Dockerfile .
	@echo "${GREEN}Production Docker image built: $(DOCKER_IMAGE):$(VERSION)${NC}"

docker-push: docker-build-production ## Push Docker image to registry
	@echo "${YELLOW}Pushing Docker images...${NC}"
	@docker push $(DOCKER_IMAGE):$(VERSION)
	@docker push $(DOCKER_IMAGE):latest
	@echo "${GREEN}Docker images pushed!${NC}"

bump-patch: ## Bump patch version (0.1.0 -> 0.1.1)
	@echo "${YELLOW}Bumping patch version...${NC}"
	@NEW_VERSION=$$(echo $(VERSION) | awk -F. '{$$3 = $$3 + 1;} 1' | sed 's/ /./g'); \
	echo $$NEW_VERSION > VERSION; \
	echo "${GREEN}Version bumped to $$NEW_VERSION${NC}"

bump-minor: ## Bump minor version (0.1.0 -> 0.2.0)
	@echo "${YELLOW}Bumping minor version...${NC}"
	@NEW_VERSION=$$(echo $(VERSION) | awk -F. '{$$2 = $$2 + 1; $$3 = 0;} 1' | sed 's/ /./g'); \
	echo $$NEW_VERSION > VERSION; \
	echo "${GREEN}Version bumped to $$NEW_VERSION${NC}"

bump-major: ## Bump major version (0.1.0 -> 1.0.0)
	@echo "${YELLOW}Bumping major version...${NC}"
	@NEW_VERSION=$$(echo $(VERSION) | awk -F. '{$$1 = $$1 + 1; $$2 = 0; $$3 = 0;} 1' | sed 's/ /./g'); \
	echo $$NEW_VERSION > VERSION; \
	echo "${GREEN}Version bumped to $$NEW_VERSION${NC}"

release-tag: ## Create and push git tag for current version
	@echo "${YELLOW}Creating release tag v$(VERSION)...${NC}"
	@git tag -a v$(VERSION) -m "Release v$(VERSION)"
	@git push origin v$(VERSION)
	@echo "${GREEN}Tag v$(VERSION) created and pushed${NC}"

release: ## Create a new release (test, build, tag, push)
	@echo "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
	@echo "${BLUE}║               Creating Release v$(VERSION)                     ║${NC}"
	@echo "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
	@echo ""
	@$(MAKE) test
	@$(MAKE) build
	@$(MAKE) docker-build-production
	@$(MAKE) docker-push
	@$(MAKE) release-tag
	@echo ""
	@echo "${GREEN}✓ Release v$(VERSION) complete!${NC}"
	@echo "${YELLOW}Next: Create GitHub release with binaries${NC}"

# ═══════════════════════════════════════════════════════════════════════════════
# CLI COMMANDS
# ═══════════════════════════════════════════════════════════════════════════════

cli: ## Build the Fluxbase CLI tool
	@echo "${YELLOW}Building ${CLI_BINARY_NAME} v$(VERSION)...${NC}"
	@mkdir -p build/
	@go build -ldflags="-X github.com/fluxbase-eu/fluxbase/cli/cmd.Version=$(VERSION) -X github.com/fluxbase-eu/fluxbase/cli/cmd.Commit=$(COMMIT) -X github.com/fluxbase-eu/fluxbase/cli/cmd.BuildDate=$(BUILD_DATE)" -o build/${CLI_BINARY_NAME} ${CLI_MAIN_PATH}
	@echo "${GREEN}CLI build complete: build/${CLI_BINARY_NAME}${NC}"

cli-install: cli ## Build and install CLI to /usr/local/bin
	@echo "${YELLOW}Installing ${CLI_BINARY_NAME} to /usr/local/bin...${NC}"
	@sudo cp build/${CLI_BINARY_NAME} /usr/local/bin/fluxbase
	@echo "${GREEN}CLI installed! Run 'fluxbase --help' to get started.${NC}"

cli-completions: cli ## Generate shell completion scripts
	@echo "${YELLOW}Generating shell completions...${NC}"
	@mkdir -p build/completions
	@./build/${CLI_BINARY_NAME} completion bash > build/completions/fluxbase.bash
	@./build/${CLI_BINARY_NAME} completion zsh > build/completions/_fluxbase
	@./build/${CLI_BINARY_NAME} completion fish > build/completions/fluxbase.fish
	@./build/${CLI_BINARY_NAME} completion powershell > build/completions/fluxbase.ps1
	@echo "${GREEN}Completions generated in build/completions/${NC}"

cli-cross-compile: ## Cross-compile CLI for multiple platforms
	@echo "${YELLOW}Cross-compiling CLI for multiple platforms...${NC}"
	@mkdir -p build/dist
	@GOOS=darwin GOARCH=amd64 go build -ldflags="-X github.com/fluxbase-eu/fluxbase/cli/cmd.Version=$(VERSION)" -o build/dist/fluxbase-darwin-amd64 ${CLI_MAIN_PATH}
	@GOOS=darwin GOARCH=arm64 go build -ldflags="-X github.com/fluxbase-eu/fluxbase/cli/cmd.Version=$(VERSION)" -o build/dist/fluxbase-darwin-arm64 ${CLI_MAIN_PATH}
	@GOOS=linux GOARCH=amd64 go build -ldflags="-X github.com/fluxbase-eu/fluxbase/cli/cmd.Version=$(VERSION)" -o build/dist/fluxbase-linux-amd64 ${CLI_MAIN_PATH}
	@GOOS=linux GOARCH=arm64 go build -ldflags="-X github.com/fluxbase-eu/fluxbase/cli/cmd.Version=$(VERSION)" -o build/dist/fluxbase-linux-arm64 ${CLI_MAIN_PATH}
	@GOOS=windows GOARCH=amd64 go build -ldflags="-X github.com/fluxbase-eu/fluxbase/cli/cmd.Version=$(VERSION)" -o build/dist/fluxbase-windows-amd64.exe ${CLI_MAIN_PATH}
	@echo "${GREEN}Cross-compilation complete! Binaries in build/dist/${NC}"
