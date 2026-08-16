.PHONY: help dev dev-full ensure-embed-placeholder ensure-embedded-sdk ensure-sdks build build-lite build-full clean fmt lint test migrate-up migrate-down migrate-create db-reset db-reset-full db-grants deps setup-dev install-hooks uninstall-hooks docs docs-build docs-check-links version docker-build docker-push release cli cli-install cli-completions viz-deps viz-deps-svg viz-internal viz-callgraph viz-callgraph-svg viz-uml viz-uml-api viz-uml-auth viz-module-deps viz-all test-cleanup test-cli

# Variables
BINARY_NAME=fluxbase-server
CLI_BINARY_NAME=fluxbase
MAIN_PATH=cmd/fluxbase/main.go
CLI_MAIN_PATH=cli/main.go

# Go build tags. Default enables Tesseract OCR (needs libtesseract + leptonica).
# Override to suit your host:
#   make build BUILD_TAGS=""            -> lite: no C media deps, OCR/vips stubs
#   make build BUILD_TAGS="ocr vips"    -> full: OCR + libvips image transforms
# See: docs/src/content/docs/getting-started/building-from-source.md
BUILD_TAGS ?= ocr

# Version variables
VERSION := $(shell cat VERSION)
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -s -w -X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildDate=$(BUILD_DATE)

# Docker variables
DOCKER_REGISTRY ?= ghcr.io
DOCKER_ORG ?= nimbleflux
DOCKER_IMAGE := $(DOCKER_REGISTRY)/$(DOCKER_ORG)/fluxbase

# macOS Homebrew keg detection. Leptonica/tesseract/vips are keg-only, so their
# headers/libs/pkgconfig files are not in the default search path. Two mechanisms
# are needed, because the bindings discover them differently:
#   * govips (the `vips` tag) uses pkg-config, so PKG_CONFIG_PATH suffices.
#   * gosseract (the `ocr` tag) does NOT use pkg-config — it hardcodes
#     -I/usr/local/include and -L/usr/local/lib (Intel Homebrew) via #cgo. On
#     Apple Silicon the kegs live under $(BREW_PREFIX)/opt/<pkg>, so we point
#     CGO at those dirs directly via CGO_CPPFLAGS/CGO_CXXFLAGS/CGO_LDFLAGS.
# Without this, `make build` fails with: 'leptonica/allheaders.h' file not found.
# Extra/unused -I and -L dirs are ignored by the compiler/linker, so this is
# harmless when a formula or build tag isn't in use.
UNAME_S := $(shell uname -s 2>/dev/null)
ifeq ($(UNAME_S),Darwin)
BREW_PREFIX := $(shell brew --prefix 2>/dev/null)
ifdef BREW_PREFIX
export PKG_CONFIG_PATH := $(BREW_PREFIX)/opt/leptonica/lib/pkgconfig:$(BREW_PREFIX)/opt/tesseract/lib/pkgconfig:$(BREW_PREFIX)/opt/vips/lib/pkgconfig:$(PKG_CONFIG_PATH)
BREW_CGO_INCLUDES := -I$(BREW_PREFIX)/opt/leptonica/include -I$(BREW_PREFIX)/opt/tesseract/include -I$(BREW_PREFIX)/opt/vips/include
BREW_CGO_LIBDIRS  := -L$(BREW_PREFIX)/opt/leptonica/lib -L$(BREW_PREFIX)/opt/tesseract/lib -L$(BREW_PREFIX)/opt/vips/lib
export CGO_CPPFLAGS := $(BREW_CGO_INCLUDES) $(CGO_CPPFLAGS)
export CGO_CXXFLAGS := $(BREW_CGO_INCLUDES) $(CGO_CXXFLAGS)
export CGO_LDFLAGS  := $(BREW_CGO_LIBDIRS) $(CGO_LDFLAGS)
endif
endif

# Database connection variables
# These can be overridden via environment variables or make command line
# Supports multiple environments:
# - CI (GitHub Actions): Uses localhost (default)
# - Devcontainer: Uses postgres service name
# - Local dev: Configurable via .env or command line
DATABASE_HOST ?= $(FLUXBASE_DATABASE_HOST)
DATABASE_PORT ?= $(FLUXBASE_DATABASE_PORT)
DATABASE_USER ?= $(FLUXBASE_DATABASE_USER)
DATABASE_ADMIN_USER ?= $(FLUXBASE_DATABASE_ADMIN_USER)
DATABASE_ADMIN_PASSWORD ?= $(FLUXBASE_DATABASE_ADMIN_PASSWORD)
DATABASE_NAME ?= $(FLUXBASE_DATABASE_DATABASE)
DATABASE_SSL_MODE ?= $(FLUXBASE_DATABASE_SSL_MODE)

# Fallback to sensible defaults if env vars are not set
ifeq ($(DATABASE_HOST),)
	DATABASE_HOST := localhost
endif
ifeq ($(DATABASE_USER),)
	DATABASE_USER := fluxbase_app
endif
ifeq ($(DATABASE_PORT),)
	DATABASE_PORT := 5432
endif
ifeq ($(DATABASE_ADMIN_USER),)
	DATABASE_ADMIN_USER := postgres
endif
ifeq ($(DATABASE_ADMIN_PASSWORD),)
	DATABASE_ADMIN_PASSWORD := postgres
endif
ifeq ($(DATABASE_NAME),)
	DATABASE_NAME := fluxbase
endif
ifeq ($(DATABASE_SSL_MODE),)
	DATABASE_SSL_MODE := disable
endif

# Docker container name for postgres (for docker exec commands in db-reset)
# In devcontainer: fluxbase-postgres-dev (default)
# In CI: Not used (CI uses direct psql commands)
# In local dev with custom compose: Set via POSTGRES_CONTAINER env var
# Examples:
#   make db-reset                                          # Uses fluxbase-postgres-dev
#   POSTGRES_CONTAINER=my-postgres make db-reset           # Uses custom container
#   POSTGRES_CONTAINER= fluxbase-postgres-dev make db-reset  # Explicit (same as default)
POSTGRES_CONTAINER ?= fluxbase-postgres-dev

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
	@echo "  make dev            # Fast dev: backend + frontend (skips admin build)"
	@echo "  make build          # Build production binary with embedded UI"
	@echo "  make test-all       # Run ALL tests (backend + SDK + React + integration)"
	@echo ""
	@echo "${GREEN}All Commands:${NC}"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  ${GREEN}%-20s${NC} %s\\n", $$1, $$2}'

dev: ## Fast dev: backend + frontend (skips admin build, uses Vite HMR at :5050)
	@echo "${YELLOW}Starting Fluxbase development environment (fast mode)...${NC}"
	@lsof -ti:8080 | xargs -r kill -9 2>/dev/null || true
	@lsof -ti:5050 | xargs -r kill -9 2>/dev/null || true
	@if [ ! -d "sdk/node_modules" ]; then \
		echo "${YELLOW}Installing SDK dependencies...${NC}"; \
		cd sdk && unset NODE_OPTIONS && bun install; \
	fi
	@$(MAKE) ensure-embedded-sdk
	@$(MAKE) ensure-sdks
	@if [ ! -d "admin/node_modules" ]; then \
		echo "${YELLOW}Installing admin UI dependencies...${NC}"; \
		cd admin && unset NODE_OPTIONS && bun install; \
	fi
	@$(MAKE) ensure-embed-placeholder
	@echo "${GREEN}Backend:${NC}     http://localhost:8080"
	@echo "${GREEN}Frontend:${NC}    http://localhost:5050/admin/"
	@echo "${GREEN}Admin Login:${NC} http://localhost:5050/admin/login"
	@echo ""
	@echo "${YELLOW}Press Ctrl+C to stop both servers${NC}"
	@echo ""
	@bash -c 'trap "kill 0" EXIT; ./run-server.sh & SERVER_PID=$$!; cd admin && unset NODE_OPTIONS && bun run dev & VITE_PID=$$!; wait -n 2>/dev/null || while kill -0 $$SERVER_PID 2>/dev/null && kill -0 $$VITE_PID 2>/dev/null; do sleep 1; done'

ensure-embed-placeholder:
	@if [ ! -f "internal/adminui/dist/index.html" ]; then \
		echo "${YELLOW}Creating placeholder admin UI for embed...${NC}"; \
		mkdir -p internal/adminui/dist/assets; \
		echo '<!DOCTYPE html><html><head><meta charset="utf-8"><title>Fluxbase (dev mode)</title></head><body style="font-family:system-ui;padding:2rem;text-align:center"><h1>Dev Mode</h1><p>Use <a href="http://localhost:5050/admin/">localhost:5050</a> for the admin UI with HMR.</p></body></html>' > internal/adminui/dist/index.html; \
		echo '/* placeholder */' > internal/adminui/dist/assets/placeholder.css; \
	fi

ensure-embedded-sdk:
	@SDK_STALE=0; \
	if [ ! -f "internal/jobs/embedded_sdk.js" ]; then \
		SDK_STALE=1; \
	else \
		NEWER=$$(find sdk/src -newer internal/jobs/embedded_sdk.js -type f 2>/dev/null | head -1); \
		if [ -n "$$NEWER" ]; then \
			SDK_STALE=1; \
		fi; \
	fi; \
	if [ "$$SDK_STALE" = "1" ]; then \
		echo "${YELLOW}Generating embedded SDK for job runtime...${NC}"; \
		cd sdk && unset NODE_OPTIONS && bun run generate:embedded-sdk; \
	else \
		echo "${GREEN}Embedded SDK up to date, skipping generation${NC}"; \
	fi

ensure-sdks:
	@for pkg in sdk sdk-react; do \
		if [ ! -d "$$pkg/node_modules" ]; then \
			echo "${YELLOW}Installing $$pkg dependencies...${NC}"; \
			( cd $$pkg && unset NODE_OPTIONS && bun install ); \
		fi; \
	done
	@if [ ! -f "sdk/dist/index.js" ] || [ -n "$$(find sdk/src -newer sdk/dist/index.js -type f 2>/dev/null | head -1)" ]; then \
		echo "${YELLOW}Building TypeScript SDK (tsup)...${NC}"; \
		cd sdk && unset NODE_OPTIONS && bun run build; \
	else \
		echo "${GREEN}TypeScript SDK up to date, skipping build${NC}"; \
	fi
	@if [ ! -f "sdk-react/dist/index.mjs" ] || [ -n "$$(find sdk-react/src -newer sdk-react/dist/index.mjs -type f 2>/dev/null | head -1)" ]; then \
		echo "${YELLOW}Building React SDK (tsup)...${NC}"; \
		cd sdk-react && unset NODE_OPTIONS && bun run build; \
	else \
		echo "${GREEN}React SDK up to date, skipping build${NC}"; \
	fi

dev-full: ## Full build + run (builds admin UI with type-check, slower)
	@echo "${YELLOW}Starting Fluxbase with full admin UI build...${NC}"
	@lsof -ti:8080 | xargs -r kill -9 2>/dev/null || true
	@lsof -ti:5050 | xargs -r kill -9 2>/dev/null || true
	@if [ ! -d "sdk/node_modules" ]; then \
		echo "${YELLOW}Installing SDK dependencies...${NC}"; \
		cd sdk && unset NODE_OPTIONS && bun install; \
	fi
	@echo "${YELLOW}Generating embedded SDK for job runtime...${NC}"
	@cd sdk && unset NODE_OPTIONS && bun run generate:embedded-sdk
	@if [ ! -d "admin/node_modules" ]; then \
		echo "${YELLOW}Installing admin UI dependencies...${NC}"; \
		cd admin && unset NODE_OPTIONS && bun install; \
	fi
	@$(MAKE) ensure-sdks
	@echo "${YELLOW}Building admin UI...${NC}"
	@cd admin && unset NODE_OPTIONS && bun run build
	@rm -rf internal/adminui/dist
	@cp -r admin/dist internal/adminui/dist
	@echo "${GREEN}Backend:${NC}     http://localhost:8080"
	@echo "${GREEN}Frontend:${NC}    http://localhost:5050/admin/"
	@echo ""
	@echo "${YELLOW}Press Ctrl+C to stop both servers${NC}"
	@echo ""
	@bash -c 'trap "kill 0" EXIT; ./run-server.sh & SERVER_PID=$$!; cd admin && unset NODE_OPTIONS && bun run dev & VITE_PID=$$!; wait -n 2>/dev/null || while kill -0 $$SERVER_PID 2>/dev/null && kill -0 $$VITE_PID 2>/dev/null; do sleep 1; done'

version: ## Show version information
	@echo "${GREEN}Version:${NC}    $(VERSION)"
	@echo "${GREEN}Commit:${NC}     $(COMMIT)"
	@echo "${GREEN}Build Date:${NC} $(BUILD_DATE)"

build: ## Build production binary with embedded admin UI
	@echo "${YELLOW}Generating embedded SDK for job runtime...${NC}"
	@cd sdk && unset NODE_OPTIONS && bun run generate:embedded-sdk
	@$(MAKE) ensure-sdks
	@echo "${YELLOW}Building admin UI...${NC}"
	@cd admin && unset NODE_OPTIONS && bun run build
	@rm -rf internal/adminui/dist
	@cp -r admin/dist internal/adminui/dist
	@echo "${YELLOW}Building ${BINARY_NAME} v$(VERSION)...${NC}"
	@mkdir -p build/
	@go build -tags "$(BUILD_TAGS)" -ldflags="$(LDFLAGS)" -o build/${BINARY_NAME} ${MAIN_PATH}
	@echo "${GREEN}Build complete: ${BINARY_NAME} v$(VERSION) (tags: $(BUILD_TAGS))${NC}"

build-lite: ## Build WITHOUT OCR/vips (no C media deps required; features self-disable at runtime)
	@$(MAKE) build BUILD_TAGS=""

build-full: ## Build with OCR AND image transforms (tags: ocr vips) — needs tesseract/leptonica/libvips
	@$(MAKE) build BUILD_TAGS="ocr vips"

clean: ## Clean build artifacts
	@echo "${YELLOW}Cleaning...${NC}"
	@rm -f build/${BINARY_NAME}
	@rm -f coverage.out coverage.html
	@rm -rf internal/adminui/dist
	@mkdir -p internal/adminui/dist/assets
	@echo '<!DOCTYPE html><html><head><meta charset="utf-8"><title>Fluxbase (placeholder)</title></head><body style="font-family:system-ui;padding:2rem;text-align:center"><p>Run make dev or make build</p></body></html>' > internal/adminui/dist/index.html
	@echo '/* placeholder */' > internal/adminui/dist/assets/placeholder.css
	@echo "${GREEN}Clean complete!${NC}"

fmt: ## Format Go code with gofumpt (stricter than gofmt)
	@echo "${YELLOW}Formatting Go code...${NC}"
	@command -v gofumpt >/dev/null 2>&1 || go install mvdan.cc/gofumpt@latest
	@gofumpt -w .
	@echo "${GREEN}Formatting complete!${NC}"

lint: ## Run golangci-lint with all enabled linters
	@echo "${YELLOW}Running golangci-lint...${NC}"
	@command -v golangci-lint >/dev/null 2>&1 || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $$(go env GOPATH)/bin v2.10.0
	@golangci-lint run --timeout 10m ./...
	@echo "${GREEN}Linting complete!${NC}"

test: ## Run all tests with race detector (short mode - skips slow tests, excludes e2e)
	@FLUXBASE_LOG_LEVEL=info ./scripts/test-runner.sh go test -timeout 2m -v -race -short -cover $(shell go list ./... | grep -v '/test/e2e')

test-cleanup: ## Clean up test resources (tables, secrets, keys, buckets) after running tests
	@echo "${YELLOW}Cleaning up test resources...${NC}"
	@go run test/cleanup/cmd/main.go
	@echo "${GREEN}Test resource cleanup complete${NC}"

test-coverage: ## Run ALL tests (unit + e2e) with combined coverage (requires postgres, mailhog, minio - may take 20+ minutes)
	@echo "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
	@echo "${BLUE}║           COVERAGE REPORT (Unit + E2E)                     ║${NC}"
	@echo "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
	@echo ""
	@echo "${YELLOW}[1/5] Running Go unit tests with coverage for code metrics (~30-60 seconds)...${NC}"
	@echo "${BLUE}Note: Integration tests with service goroutines are skipped for accurate coverage metrics${NC}"
	@echo ""
	@FLUXBASE_LOG_LEVEL=info FLUXBASE_PARALLEL_TEST=true NO_COLOR=1 go test -v -timeout 30m -short -coverprofile=coverage.out -covermode=set $(shell go list ./... | grep -v "^github.com/nimbleflux/fluxbase/test/e2e$$") 2>&1 | tee /tmp/go-test-output.txt
	@echo ""
	@echo "${BLUE}Full test output written to: ${YELLOW}/tmp/go-test-output.txt${NC}"
	@echo ""
	@echo "${YELLOW}[2/5] Checking coverage thresholds...${NC}"
	@-go-test-coverage --config=.testcoverage.yml || echo "${YELLOW}Coverage threshold not met (informational only)${NC}"
	@echo ""
	@echo "${YELLOW}[3/5] Generating Go coverage report...${NC}"
	@go tool cover -html=coverage.out -o coverage.html
	@go tool cover -func=coverage.out | grep total | awk '{print "  ${GREEN}Go Coverage (Unit Tests Only): " $$3 "${NC}"}'
	@echo ""
	@echo "${YELLOW}[4/5] Running SDK tests with coverage...${NC}"
	@cd sdk && unset NODE_OPTIONS && npx vitest --coverage --run 2>&1 | tail -20 || true
	@echo ""
	@echo "${YELLOW}[5/5] Cleaning up test resources...${NC}"
	@$(MAKE) test-cleanup
	@echo ""
	@echo "${GREEN}Coverage reports generated (unit tests only):${NC}"
	@echo "  - coverage.out           (Go profile - unit tests)"
	@echo "  - coverage.html          (Go HTML report)"
	@echo "  - sdk/coverage/          (SDK coverage)"
	@echo "  - /tmp/go-test-output.txt (Full test log)"
	@echo ""
	@echo "${YELLOW}Note: Integration tests excluded from coverage to avoid service goroutine interference.${NC}"
	@echo "${YELLOW}      Run 'make test-full' for integration tests with coverage.${NC}"

test-coverage-check: ## Check coverage thresholds without running tests (requires coverage.out)
	@go-test-coverage --config=.testcoverage.yml

test-coverage-unit: ## Run unit tests only with coverage (excludes e2e, faster for development)
	@echo "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
	@echo "${BLUE}║           UNIT TEST COVERAGE (excludes e2e)               ║${NC}"
	@echo "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
	@echo ""
	@echo "${YELLOW}[1/3] Running Go unit tests with coverage (~30-60 seconds)...${NC}"
	@echo "${BLUE}Watch for: '=== RUN TestName' lines showing test progress${NC}"
	@echo ""
	@FLUXBASE_LOG_LEVEL=info go test -v -short -timeout 5m -coverprofile=coverage.out -covermode=atomic $(shell go list ./... | grep -v '/test/e2e' | grep -v '/test$$') 2>&1 | tee /tmp/go-test-unit-output.txt
	@echo ""
	@echo "${BLUE}Unit test output written to: ${YELLOW}/tmp/go-test-unit-output.txt${NC}"
	@echo ""
	@echo "${YELLOW}[2/3] Generating Go coverage report...${NC}"
	@go tool cover -html=coverage.out -o coverage.html
	@go tool cover -func=coverage.out | grep total | awk '{print "  ${GREEN}Go Unit Coverage: " $$3 "${NC}"}'
	@echo ""
	@echo "${YELLOW}[3/3] Running SDK tests with coverage...${NC}"
	@cd sdk && unset NODE_OPTIONS && npx vitest --coverage --run 2>&1 | tail -20 || true
	@echo ""
	@echo "${GREEN}Unit test coverage reports generated:${NC}"
	@echo "  - coverage.out              (Go profile - unit only)"
	@echo "  - coverage.html             (Go HTML report)"
	@echo "  - sdk/coverage/             (SDK coverage)"
	@echo "  - /tmp/go-test-unit-output.txt (Unit test log)"

test-coverage-full: test-coverage ## Alias for test-coverage (now includes e2e by default)

test-fast: ## Run all tests without race detector (faster, excludes e2e)
	@FLUXBASE_LOG_LEVEL=info ./scripts/test-runner.sh go test -timeout 1m -v -short -cover $(shell go list ./... | grep -v '/test/e2e')

test-setup-db: ## Apply bootstrap + declarative schemas to match CI pipeline setup
	@echo "${YELLOW}Applying database schema for tests (matching CI pipeline)...${NC}"
	@echo "${BLUE}Database:${NC} $(DATABASE_ADMIN_USER)@$(DATABASE_HOST):$(DATABASE_PORT)/$(DATABASE_NAME)"
	@# 1. Apply bootstrap SQL (creates schemas, extensions, roles, default privileges)
	@# Substitute {{APP_USER}} with $(DATABASE_USER) (Go runtime does this via SubstituteAppUser)
	@sed "s/{{APP_USER}}/$(DATABASE_USER)/g" internal/database/bootstrap/bootstrap.sql | PGPASSWORD=$(DATABASE_ADMIN_PASSWORD) psql -h $(DATABASE_HOST) -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -v ON_ERROR_STOP=1
	@# 2. Apply each declarative schema in dependency order (all use CREATE IF NOT EXISTS = idempotent)
	@# Note: We do NOT use ON_ERROR_STOP because tables may reference FKs not yet created;
	@# cross-schema FKs are added later by post-schema-fks.sql. Errors are tolerated via || true.
	@for schema in platform auth storage jobs functions realtime ai rpc app branching logging mcp; do \
		echo "Applying schema: $$schema"; \
		sed "s/{{APP_USER}}/$(DATABASE_USER)/g" internal/database/schema/schemas/$$schema.sql | PGPASSWORD=$(DATABASE_ADMIN_PASSWORD) psql -h $(DATABASE_HOST) -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) || true; \
	done
	@# 3. Apply cross-schema foreign keys (idempotent DO blocks)
	@echo "Applying cross-schema foreign keys..."
	@PGPASSWORD=$(DATABASE_ADMIN_PASSWORD) psql -h $(DATABASE_HOST) -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -v ON_ERROR_STOP=1 -f internal/database/schema/schemas/post-schema-fks.sql || true
	@# 4. Apply cross-schema policies (safe on fresh database)
	@echo "Applying cross-schema policies..."
	@PGPASSWORD=$(DATABASE_ADMIN_PASSWORD) psql -h $(DATABASE_HOST) -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -v ON_ERROR_STOP=1 -f internal/database/schema/schemas/post-schema.sql || true
	@# 5. Grant role memberships for SET ROLE support
	@PGPASSWORD=$(DATABASE_ADMIN_PASSWORD) psql -h $(DATABASE_HOST) -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "GRANT anon, authenticated, service_role, tenant_service TO fluxbase_app, fluxbase_rls_test;" || true
	@# 6. Grant admin privileges
	@PGPASSWORD=$(DATABASE_ADMIN_PASSWORD) psql -h $(DATABASE_HOST) -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "GRANT ALL ON DATABASE $(DATABASE_NAME) TO fluxbase_app;" || true
	@PGPASSWORD=$(DATABASE_ADMIN_PASSWORD) psql -h $(DATABASE_HOST) -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "GRANT ALL ON SCHEMA public TO fluxbase_app;" || true
	@echo "${GREEN}Database schema applied successfully!${NC}"

test-full: ## Run ALL tests including e2e with race detector (may take 5-10 minutes)
	@./scripts/test-runner.sh go test -timeout 15m -v -race -cover -tags=integration ./...

test-e2e: ## Run e2e tests only (self-sufficient: resets and bootstraps fluxbase_go_e2e DB). Use RUN= to filter tests.
	@./scripts/test-runner.sh go test -v -race -parallel=1 -timeout=10m -tags=integration ./test/e2e/... $(if $(RUN),-run $(RUN),)

test-e2e-fast: ## Run e2e tests without race detector (faster for dev iteration). Use RUN= to filter tests.
	@./scripts/test-runner.sh go test -v -parallel=1 -timeout=5m -tags=integration ./test/e2e/... $(if $(RUN),-run $(RUN),)

test-auth: ## Run authentication tests only
	@./scripts/test-runner.sh go test -v -race -timeout=5m -tags=integration ./test/e2e/ -run TestAuth

test-rls: ## Run RLS security tests only
	@./scripts/test-runner.sh go test -v -race -timeout=5m -tags=integration ./test/e2e/ -run TestRLS

test-rest: ## Run REST API tests only
	@./scripts/test-runner.sh go test -v -race -timeout=5m -tags=integration ./test/e2e/ -run TestREST

test-storage: ## Run storage tests only
	@./scripts/test-runner.sh go test -v -race -timeout=5m -tags=integration ./test/e2e/ -run TestStorage

test-cli: ## Run CLI tests (unit + mock server, no external dependencies)
	@./scripts/test-runner.sh go test -v -race -timeout=2m ./cli/...

test-sdk: ## Run SDK tests (TypeScript)
	@echo "${YELLOW}Running SDK tests...${NC}"
	@cd sdk && unset NODE_OPTIONS && bun test -- src/admin.test.ts src/auth.test.ts src/management.test.ts src/ddl.test.ts src/impersonation.test.ts src/settings.test.ts src/oauth.test.ts
	@echo "${GREEN}SDK tests complete!${NC}"

test-sdk-react: ## Build React SDK (includes type checking)
	@echo "${YELLOW}Building React SDK...${NC}"
	@cd sdk-react && unset NODE_OPTIONS && bun run build
	@echo "${GREEN}React SDK build complete!${NC}"

test-integration: ## Run admin integration tests (requires running server)
	@echo "${YELLOW}Running admin integration tests...${NC}"
	@if ! curl -s http://localhost:8080/health > /dev/null; then \
		echo "${RED}Error: Fluxbase server not running on localhost:8080${NC}"; \
		echo "${YELLOW}Start server with: make dev${NC}"; \
		exit 1; \
	fi
	@cd examples/admin-setup && unset NODE_OPTIONS && bun test
	@echo "${GREEN}Integration tests complete!${NC}"

test-e2e-ui: ## Run Playwright E2E tests. Resets DB, starts server, runs all tests.
	@echo "${YELLOW}Running Playwright E2E tests (clean database)...${NC}"
	@cd admin && bunx playwright test

test-e2e-ui-headed: ## Run Playwright E2E tests with visible browser
	@cd admin && bunx playwright test --headed

test-e2e-ui-debug: ## Run Playwright E2E tests in debug mode
	@cd admin && bunx playwright test --debug

test-e2e-ui-dev: ## Run Playwright E2E tests reusing existing server (for dev iteration)
	@./scripts/start-e2e-ui.sh --ensure
	@echo "${YELLOW}Running Playwright E2E tests (reusing server)...${NC}"
	@cd admin && PLAYWRIGHT_REUSE_SERVER=true bunx playwright test

test-e2e-ui-server: ## Start E2E test servers (Go :8082 + Vite :5050). Ctrl+C to stop.
	@./scripts/start-e2e-ui.sh

test-e2e-ui-restart: ## Restart E2E test servers (kills existing, starts fresh)
	@./scripts/start-e2e-ui.sh --restart

test-e2e-ui-setup: ## Reset the Playwright test database (drop schemas, server re-applies on startup)
	@echo "${YELLOW}Resetting Playwright test database...${NC}"
	@PGPASSWORD=$(DATABASE_ADMIN_PASSWORD) psql -h $(DATABASE_HOST) -U $(DATABASE_ADMIN_USER) -d fluxbase_playwright -c "DO \$\$ DECLARE r RECORD; BEGIN FOR r IN SELECT nspname FROM pg_namespace WHERE nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast') LOOP EXECUTE 'DROP SCHEMA IF EXISTS ' || quote_ident(r.nspname) || ' CASCADE'; END LOOP; END \$\$;"
	@PGPASSWORD=$(DATABASE_ADMIN_PASSWORD) psql -h $(DATABASE_HOST) -U $(DATABASE_ADMIN_USER) -d fluxbase_playwright -c "CREATE SCHEMA IF NOT EXISTS public; GRANT ALL ON SCHEMA public TO public;"
	@echo "${GREEN}Playwright test database reset! Start server to re-apply schemas.${NC}"

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
	@cd admin && unset NODE_OPTIONS && bun install
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

migrate-up: ## Run user-provided migrations (internal schema is auto-applied)
	@echo "${YELLOW}Note: Internal Fluxbase schema is applied automatically on server startup.${NC}"
	@echo "${YELLOW}This target is for user-provided migrations only.${NC}"
	@echo "${YELLOW}Set USER_MIGRATIONS_PATH in your config to use this feature.${NC}"

migrate-down: ## Rollback last user migration
	@echo "${YELLOW}Note: Internal Fluxbase schema is managed declaratively.${NC}"
	@echo "${YELLOW}This target is for user-provided migrations only.${NC}"

migrate-create: ## Create new user migration (usage: make migrate-create name=add_users_table)
	@if [ -z "$(name)" ]; then \
		echo "${YELLOW}Error: Provide migration name${NC}"; \
		echo "Usage: make migrate-create name=add_users_table"; \
		exit 1; \
	fi
	@echo "${YELLOW}Note: Create user migrations in your own directory.${NC}"
	@echo "${YELLOW}Set USER_MIGRATIONS_PATH in your config.${NC}"

db-reset: ## Reset database (preserves public, auth.users, platform.users, setup_completed). Use db-reset-full for full reset.
	@echo "${YELLOW}Resetting database (preserving public schema, user data, setup_completed)...${NC}"
	@echo "${BLUE}Database:${NC} $(DATABASE_ADMIN_USER)@$(DATABASE_HOST):$(DATABASE_PORT)/$(DATABASE_NAME)"
	@echo "${BLUE}PostgreSQL container:${NC} $(POSTGRES_CONTAINER)"
	@# Backup user data and settings before dropping schemas
	@echo "${YELLOW}Backing up user data...${NC}"
	@PGPASSWORD=$(DATABASE_ADMIN_PASSWORD) psql -h $(DATABASE_HOST) -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "DROP TABLE IF EXISTS _fluxbase_auth_users_backup; CREATE TABLE _fluxbase_auth_users_backup AS SELECT * FROM auth.users;" 2>/dev/null || true
	@PGPASSWORD=$(DATABASE_ADMIN_PASSWORD) psql -h $(DATABASE_HOST) -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "DROP TABLE IF EXISTS _fluxbase_platform_users_backup; CREATE TABLE _fluxbase_platform_users_backup AS SELECT * FROM platform.users;" 2>/dev/null || true
	@PGPASSWORD=$(DATABASE_ADMIN_PASSWORD) psql -h $(DATABASE_HOST) -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "DROP TABLE IF EXISTS _fluxbase_setup_backup; CREATE TABLE _fluxbase_setup_backup AS SELECT * FROM app.settings WHERE key = 'setup_completed';" 2>/dev/null || true
	@# Drop all schemas (including auth) for clean migration
	@PGPASSWORD=$(DATABASE_ADMIN_PASSWORD) psql -h $(DATABASE_HOST) -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "DROP SCHEMA IF EXISTS app CASCADE;" || true
	@PGPASSWORD=$(DATABASE_ADMIN_PASSWORD) psql -h $(DATABASE_HOST) -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "DROP SCHEMA IF EXISTS auth CASCADE;" || true
	@PGPASSWORD=$(DATABASE_ADMIN_PASSWORD) psql -h $(DATABASE_HOST) -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "DROP SCHEMA IF EXISTS storage CASCADE;" || true
	@PGPASSWORD=$(DATABASE_ADMIN_PASSWORD) psql -h $(DATABASE_HOST) -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "DROP SCHEMA IF EXISTS functions CASCADE;" || true
	@PGPASSWORD=$(DATABASE_ADMIN_PASSWORD) psql -h $(DATABASE_HOST) -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "DROP SCHEMA IF EXISTS jobs CASCADE;" || true
	@PGPASSWORD=$(DATABASE_ADMIN_PASSWORD) psql -h $(DATABASE_HOST) -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "DROP SCHEMA IF EXISTS realtime CASCADE;" || true
	@PGPASSWORD=$(DATABASE_ADMIN_PASSWORD) psql -h $(DATABASE_HOST) -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "DROP SCHEMA IF EXISTS ai CASCADE;" || true
	@PGPASSWORD=$(DATABASE_ADMIN_PASSWORD) psql -h $(DATABASE_HOST) -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "DROP SCHEMA IF EXISTS rpc CASCADE;" || true
	@PGPASSWORD=$(DATABASE_ADMIN_PASSWORD) psql -h $(DATABASE_HOST) -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "DROP SCHEMA IF EXISTS branching CASCADE;" || true
	@echo "${YELLOW}Ensuring test users exist with correct permissions...${NC}"
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "DO \$$\$$ BEGIN IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'fluxbase_app') THEN CREATE USER fluxbase_app WITH PASSWORD 'fluxbase_app_password' LOGIN CREATEDB BYPASSRLS; END IF; END \$$\$$;" || true
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "DO \$$\$$ BEGIN IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'fluxbase_rls_test') THEN CREATE USER fluxbase_rls_test WITH PASSWORD 'fluxbase_rls_test_password' LOGIN; END IF; END \$$\$$;" || true
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "ALTER USER postgres WITH BYPASSRLS;" || true
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "ALTER USER postgres SET search_path TO public;" || true
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "ALTER USER fluxbase_app WITH BYPASSRLS;" || true
	@echo "${YELLOW}Running bootstrap...${NC}"
	@echo "${BLUE}Database:${NC} $(DATABASE_ADMIN_USER)@$(DATABASE_HOST):$(DATABASE_PORT)/$(DATABASE_NAME)"
	@PGPASSWORD=$(DATABASE_ADMIN_PASSWORD) psql -h $(DATABASE_HOST) -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -f internal/database/bootstrap/bootstrap.sql
	@echo "${YELLOW}Granting permissions to test users (fluxbase_app, fluxbase_rls_test)...${NC}"
	@$(MAKE) db-grants
	@# Restore user data from backups
	@echo "${YELLOW}Restoring user data from backups...${NC}"
	@PGPASSWORD=$(DATABASE_ADMIN_PASSWORD) psql -h $(DATABASE_HOST) -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "INSERT INTO auth.users SELECT * FROM _fluxbase_auth_users_backup ON CONFLICT (id) DO NOTHING; DROP TABLE IF EXISTS _fluxbase_auth_users_backup;" 2>/dev/null || true
	@PGPASSWORD=$(DATABASE_ADMIN_PASSWORD) psql -h $(DATABASE_HOST) -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "INSERT INTO platform.users SELECT * FROM _fluxbase_platform_users_backup ON CONFLICT (id) DO NOTHING; DROP TABLE IF EXISTS _fluxbase_platform_users_backup;" 2>/dev/null || true
	@PGPASSWORD=$(DATABASE_ADMIN_PASSWORD) psql -h $(DATABASE_HOST) -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "INSERT INTO app.settings SELECT * FROM _fluxbase_setup_backup ON CONFLICT (key) WHERE user_id IS NULL DO UPDATE SET value = EXCLUDED.value, updated_at = NOW(); DROP TABLE IF EXISTS _fluxbase_setup_backup;" 2>/dev/null || true
	@echo "${GREEN}Database reset complete!${NC}"
	@echo "${BLUE}Note: Migrations granted all permissions to the user running them ($(DATABASE_ADMIN_USER)).${NC}"
	@echo "${BLUE}Additional permissions granted to fluxbase_app and fluxbase_rls_test for testing.${NC}"

# db-grants: Grant fluxbase_app and fluxbase_rls_test the role memberships, schema USAGE,
# table/sequence/function privileges, and SET ROLE memberships they need.
# Required because bootstrap.sql runs as the admin user via the Go server, so
# `GRANT ... TO CURRENT_USER` lines never reach the app user. Both db-reset and
# db-reset-full invoke this target so the database is left in a usable state.
db-grants:
	@echo "${YELLOW}Applying schema and table grants...${NC}"
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "GRANT CREATE ON DATABASE $(DATABASE_NAME) TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "GRANT USAGE, CREATE ON SCHEMA app TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "GRANT USAGE, CREATE ON SCHEMA auth TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "GRANT USAGE, CREATE ON SCHEMA platform TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "GRANT USAGE, CREATE ON SCHEMA functions TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "GRANT USAGE, CREATE ON SCHEMA jobs TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "GRANT USAGE, CREATE ON SCHEMA storage TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "GRANT USAGE, CREATE ON SCHEMA realtime TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "GRANT USAGE, CREATE ON SCHEMA ai TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "GRANT USAGE, CREATE ON SCHEMA rpc TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "GRANT USAGE, CREATE ON SCHEMA mcp TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "GRANT USAGE, CREATE ON SCHEMA branching TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "GRANT ALL ON ALL TABLES IN SCHEMA app TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "GRANT ALL ON ALL SEQUENCES IN SCHEMA app TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "GRANT ALL ON ALL TABLES IN SCHEMA auth TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "GRANT ALL ON ALL SEQUENCES IN SCHEMA auth TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "GRANT ALL ON ALL TABLES IN SCHEMA platform TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "GRANT ALL ON ALL SEQUENCES IN SCHEMA platform TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "GRANT ALL ON ALL TABLES IN SCHEMA functions TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "GRANT ALL ON ALL SEQUENCES IN SCHEMA functions TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "GRANT ALL ON ALL TABLES IN SCHEMA jobs TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "GRANT ALL ON ALL SEQUENCES IN SCHEMA jobs TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "GRANT ALL ON ALL TABLES IN SCHEMA storage TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "GRANT ALL ON ALL SEQUENCES IN SCHEMA storage TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "GRANT ALL ON ALL TABLES IN SCHEMA realtime TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "GRANT ALL ON ALL SEQUENCES IN SCHEMA realtime TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "GRANT ALL ON ALL TABLES IN SCHEMA ai TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "GRANT ALL ON ALL SEQUENCES IN SCHEMA ai TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "GRANT ALL ON ALL TABLES IN SCHEMA rpc TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "GRANT ALL ON ALL SEQUENCES IN SCHEMA rpc TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "GRANT ALL ON ALL TABLES IN SCHEMA mcp TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "GRANT ALL ON ALL SEQUENCES IN SCHEMA mcp TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "GRANT ALL ON ALL TABLES IN SCHEMA branching TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "GRANT ALL ON ALL SEQUENCES IN SCHEMA branching TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA auth TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA storage TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA rpc TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA mcp TO fluxbase_app, fluxbase_rls_test;" || true
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO fluxbase_app, fluxbase_rls_test;" || true
	@echo "${YELLOW}Granting role memberships for SET ROLE support...${NC}"
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "DO \$$\$$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_auth_members WHERE roleid = (SELECT oid FROM pg_roles WHERE rolname = 'anon') AND member = (SELECT oid FROM pg_roles WHERE rolname = 'fluxbase_app')) THEN GRANT anon TO fluxbase_app; END IF; IF NOT EXISTS (SELECT 1 FROM pg_auth_members WHERE roleid = (SELECT oid FROM pg_roles WHERE rolname = 'authenticated') AND member = (SELECT oid FROM pg_roles WHERE rolname = 'fluxbase_app')) THEN GRANT authenticated TO fluxbase_app; END IF; IF NOT EXISTS (SELECT 1 FROM pg_auth_members WHERE roleid = (SELECT oid FROM pg_roles WHERE rolname = 'service_role') AND member = (SELECT oid FROM pg_roles WHERE rolname = 'fluxbase_app')) THEN GRANT service_role TO fluxbase_app; END IF; IF NOT EXISTS (SELECT 1 FROM pg_auth_members WHERE roleid = (SELECT oid FROM pg_roles WHERE rolname = 'tenant_service') AND member = (SELECT oid FROM pg_roles WHERE rolname = 'fluxbase_app')) THEN GRANT tenant_service TO fluxbase_app; END IF; IF NOT EXISTS (SELECT 1 FROM pg_auth_members WHERE roleid = (SELECT oid FROM pg_roles WHERE rolname = 'tenant_migration_role') AND member = (SELECT oid FROM pg_roles WHERE rolname = 'fluxbase_app')) THEN GRANT tenant_migration_role TO fluxbase_app; END IF; IF NOT EXISTS (SELECT 1 FROM pg_auth_members WHERE roleid = (SELECT oid FROM pg_roles WHERE rolname = 'anon') AND member = (SELECT oid FROM pg_roles WHERE rolname = 'fluxbase_rls_test')) THEN GRANT anon TO fluxbase_rls_test; END IF; IF NOT EXISTS (SELECT 1 FROM pg_auth_members WHERE roleid = (SELECT oid FROM pg_roles WHERE rolname = 'authenticated') AND member = (SELECT oid FROM pg_roles WHERE rolname = 'fluxbase_rls_test')) THEN GRANT authenticated TO fluxbase_rls_test; END IF; IF NOT EXISTS (SELECT 1 FROM pg_auth_members WHERE roleid = (SELECT oid FROM pg_roles WHERE rolname = 'service_role') AND member = (SELECT oid FROM pg_roles WHERE rolname = 'fluxbase_rls_test')) THEN GRANT service_role TO fluxbase_rls_test; END IF; IF NOT EXISTS (SELECT 1 FROM pg_auth_members WHERE roleid = (SELECT oid FROM pg_roles WHERE rolname = 'tenant_service') AND member = (SELECT oid FROM pg_roles WHERE rolname = 'fluxbase_rls_test')) THEN GRANT tenant_service TO fluxbase_rls_test; END IF; IF NOT EXISTS (SELECT 1 FROM pg_auth_members WHERE roleid = (SELECT oid FROM pg_roles WHERE rolname = 'tenant_migration_role') AND member = (SELECT oid FROM pg_roles WHERE rolname = 'fluxbase_rls_test')) THEN GRANT tenant_migration_role TO fluxbase_rls_test; END IF; END \$$\$$;" || true

db-reset-full: ## Full database reset (drops ALL schemas). Bootstrap and schema applied on server startup. WARNING: Destroys all data!
	@echo "${RED}WARNING: Full database reset - this will destroy ALL data!${NC}"
	@echo "${BLUE}Database:${NC} $(DATABASE_ADMIN_USER)@$(DATABASE_HOST):$(DATABASE_PORT)/$(DATABASE_NAME)"
	@echo "${BLUE}PostgreSQL container:${NC} $(POSTGRES_CONTAINER)"
	@echo "${YELLOW}Dropping all non-system schemas...${NC}"
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "DO \$$\$$ DECLARE r RECORD; BEGIN FOR r IN SELECT nspname FROM pg_namespace WHERE nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast') LOOP EXECUTE 'DROP SCHEMA IF EXISTS ' || quote_ident(r.nspname) || ' CASCADE'; END LOOP; END \$$\$$;" || true
	@echo "${YELLOW}Recreating public schema...${NC}"
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "CREATE SCHEMA IF NOT EXISTS public;" || true
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "GRANT ALL ON SCHEMA public TO postgres;" || true
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "GRANT ALL ON SCHEMA public TO public;" || true
	@echo "${YELLOW}Ensuring test users exist...${NC}"
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "DO \$$\$$ BEGIN IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'fluxbase_app') THEN CREATE USER fluxbase_app WITH PASSWORD 'fluxbase_app_password' LOGIN CREATEDB BYPASSRLS; END IF; END \$$\$$;" || true
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "DO \$$\$$ BEGIN IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'fluxbase_rls_test') THEN CREATE USER fluxbase_rls_test WITH PASSWORD 'fluxbase_rls_test_password' LOGIN; END IF; END \$$\$$;" || true
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "ALTER USER postgres WITH BYPASSRLS;" || true
	@docker exec $(POSTGRES_CONTAINER) psql -U $(DATABASE_ADMIN_USER) -d $(DATABASE_NAME) -c "ALTER USER fluxbase_app WITH BYPASSRLS;" || true
	@# Grant app user permissions before bootstrap runs. The server's bootstrap runs
	@# as the admin user, so without these grants fluxbase_app cannot SET LOCAL ROLE
	@# service_role or read from platform/storage/auth schemas on first startup.
	@$(MAKE) db-grants
	@echo "${GREEN}Full database reset complete! Run 'make dev' to apply bootstrap and declarative schema.${NC}"

docs: ## Serve Starlight documentation at http://localhost:4321
	@echo "${YELLOW}Starting Starlight documentation server...${NC}"
	@if [ ! -d "docs/node_modules" ]; then \
		echo "${YELLOW}Installing documentation dependencies...${NC}"; \
		cd docs && unset NODE_OPTIONS && bun install; \
	fi
	@echo ""
	@echo "${GREEN}📚 Documentation will be available at:${NC}"
	@echo "  ${GREEN}http://localhost:4321${NC}"
	@echo ""
	@echo "${YELLOW}Press Ctrl+C to stop the server${NC}"
	@echo ""
	@cd docs && unset NODE_OPTIONS && bun run dev -- --host 0.0.0.0

docs-build: ## Build static documentation site for production
	@echo "${YELLOW}Building documentation site...${NC}"
	@if [ ! -d "docs/node_modules" ]; then \
		echo "${YELLOW}Installing documentation dependencies...${NC}"; \
		cd docs && unset NODE_OPTIONS && bun install; \
	fi
	@cd docs && unset NODE_OPTIONS && bun run build
	@echo "${GREEN}Documentation built successfully!${NC}"
	@echo "${YELLOW}Output:${NC} docs/dist/"
	@echo "${YELLOW}To preview locally:${NC} cd docs && bun run preview"

sdk-kotlin-docs: ## Regenerate Kotlin SDK API docs (Dokka GFM → docs/src/content/docs/api/sdk-kotlin/)
	@echo "${YELLOW}Generating Kotlin SDK API documentation...${NC}"
	@cd sdk-kotlin && ./gradlew dokkaGfm
	@echo "${GREEN}Kotlin SDK docs generated!${NC}"
	@echo "${YELLOW}Output:${NC} docs/src/content/docs/api/sdk-kotlin/"
	@echo "${YELLOW}Commit the generated files to update the docs site.${NC}"

docs-check-links: docs-build ## Check documentation for broken links
	@echo "${YELLOW}Checking documentation for broken links...${NC}"
	@which lychee > /dev/null 2>&1 || { \
		echo "${RED}Error: lychee is not installed${NC}"; \
		echo "${YELLOW}Install with: cargo install lychee${NC}"; \
		echo "${YELLOW}Or on macOS: brew install lychee${NC}"; \
		echo "${YELLOW}Or download from: https://github.com/lycheeverse/lychee/releases${NC}"; \
		exit 1; \
	}
	@lychee --config .lychee.toml docs/dist
	@echo "${GREEN}Link check complete!${NC}"

docker-build-docs: ## Build documentation Docker image
	@echo "${YELLOW}Building documentation Docker image...${NC}"
	@docker build \
		-t $(DOCKER_IMAGE)-docs:$(VERSION) \
		-t $(DOCKER_IMAGE)-docs:latest \
		-f Dockerfile.docs .
	@echo "${GREEN}Documentation Docker image built!${NC}"
	@echo "${YELLOW}To run locally:${NC} docker run -p 8080:8080 $(DOCKER_IMAGE)-docs:latest"
	@echo "${YELLOW}Access at:${NC} http://localhost:8080"

# Image flavor selection.
#   FLAVOR=full (default): debian-slim runtime with OCR + ffmpeg + libvips image
#                          transforms. Produces the default (unsuffixed) image.
#   FLAVOR=lite:           distroless runtime, pure-Go static binary. No OCR,
#                          no ffmpeg, no image transforms. Produces a -lite tag.
# FLAVOR selects the go-builder config (CGO + build tags); the matching runtime
# stage is selected via --target, so both must agree (enforced below).
FLAVOR ?= full

# Derive the tag suffix and target stage from FLAVOR.
ifeq ($(FLAVOR),full)
	FLAVOR_SUFFIX :=
	FLAVOR_TARGET := runtime-full
else ifeq ($(FLAVOR),lite)
	FLAVOR_SUFFIX := -lite
	FLAVOR_TARGET := runtime-lite
else
$(error Unsupported FLAVOR "$(FLAVOR)" — expected "full" or "lite")
endif

docker-build: ## Build Docker image (FLAVOR=full|lite, default full)
	@echo "${YELLOW}Building Docker image $(DOCKER_IMAGE):$(VERSION)$(FLAVOR_SUFFIX) [flavor=$(FLAVOR)]...${NC}"
	@docker build \
		--build-arg FLAVOR=$(FLAVOR) \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		--target $(FLAVOR_TARGET) \
		-t $(DOCKER_IMAGE):$(VERSION)$(FLAVOR_SUFFIX) \
		-t $(DOCKER_IMAGE):latest$(FLAVOR_SUFFIX) \
		-f Dockerfile .
	@echo "${GREEN}Docker image built: $(DOCKER_IMAGE):$(VERSION)$(FLAVOR_SUFFIX)${NC}"

docker-build-production: ## Build production Docker image with admin UI (FLAVOR=full|lite)
	@echo "${YELLOW}Building production Docker image with admin UI [flavor=$(FLAVOR)]...${NC}"
	@docker build \
		--build-arg FLAVOR=$(FLAVOR) \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		--target $(FLAVOR_TARGET) \
		-t $(DOCKER_IMAGE):$(VERSION)$(FLAVOR_SUFFIX) \
		-t $(DOCKER_IMAGE):latest$(FLAVOR_SUFFIX) \
		-f Dockerfile .
	@echo "${GREEN}Production Docker image built: $(DOCKER_IMAGE):$(VERSION)$(FLAVOR_SUFFIX)${NC}"

docker-build-full: ## Build the full Docker image (OCR + ffmpeg + libvips)
	@$(MAKE) docker-build-production FLAVOR=full

docker-build-lite: ## Build the lite Docker image (distroless, no media deps)
	@$(MAKE) docker-build-production FLAVOR=lite

docker-push: docker-build-production ## Push Docker image to registry (FLAVOR=full|lite)
	@echo "${YELLOW}Pushing Docker images [flavor=$(FLAVOR)]...${NC}"
	@docker push $(DOCKER_IMAGE):$(VERSION)$(FLAVOR_SUFFIX)
	@docker push $(DOCKER_IMAGE):latest$(FLAVOR_SUFFIX)
	@echo "${GREEN}Docker images pushed!${NC}"

bump-patch: ## Bump patch version (0.1.0 -> 0.1.1)
	@echo "${YELLOW}Bumping patch version...${NC}"
	@NEW_VERSION=$$(echo $(VERSION) | awk -F. '{$$3 = $$3 + 1;} 1' | sed 's/ /./g'); \
	echo $$NEW_VERSION > VERSION; \
	sed -i '' 's/$${FLUXBASE_VERSION:-$(VERSION)}/$${FLUXBASE_VERSION:-'"$$NEW_VERSION"'}/g' deploy/docker-compose.minimal.yaml; \
	echo "${GREEN}Version bumped to $$NEW_VERSION${NC}"

bump-minor: ## Bump minor version (0.1.0 -> 0.2.0)
	@echo "${YELLOW}Bumping minor version...${NC}"
	@NEW_VERSION=$$(echo $(VERSION) | awk -F. '{$$2 = $$2 + 1; $$3 = 0;} 1' | sed 's/ /./g'); \
	echo $$NEW_VERSION > VERSION; \
	sed -i '' 's/$${FLUXBASE_VERSION:-$(VERSION)}/$${FLUXBASE_VERSION:-'"$$NEW_VERSION"'}/g' deploy/docker-compose.minimal.yaml; \
	echo "${GREEN}Version bumped to $$NEW_VERSION${NC}"

bump-major: ## Bump major version (0.1.0 -> 1.0.0)
	@echo "${YELLOW}Bumping major version...${NC}"
	@NEW_VERSION=$$(echo $(VERSION) | awk -F. '{$$1 = $$1 + 1; $$2 = 0; $$3 = 0;} 1' | sed 's/ /./g'); \
	echo $$NEW_VERSION > VERSION; \
	sed -i '' 's/$${FLUXBASE_VERSION:-$(VERSION)}/$${FLUXBASE_VERSION:-'"$$NEW_VERSION"'}/g' deploy/docker-compose.minimal.yaml; \
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
	@go build -ldflags="-X github.com/nimbleflux/fluxbase/cli/cmd.Version=$(VERSION) -X github.com/nimbleflux/fluxbase/cli/cmd.Commit=$(COMMIT) -X github.com/nimbleflux/fluxbase/cli/cmd.BuildDate=$(BUILD_DATE)" -o build/${CLI_BINARY_NAME} ${CLI_MAIN_PATH}
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
	@GOOS=darwin GOARCH=amd64 go build -ldflags="-X github.com/nimbleflux/fluxbase/cli/cmd.Version=$(VERSION)" -o build/dist/fluxbase-darwin-amd64 ${CLI_MAIN_PATH}
	@GOOS=darwin GOARCH=arm64 go build -ldflags="-X github.com/nimbleflux/fluxbase/cli/cmd.Version=$(VERSION)" -o build/dist/fluxbase-darwin-arm64 ${CLI_MAIN_PATH}
	@GOOS=linux GOARCH=amd64 go build -ldflags="-X github.com/nimbleflux/fluxbase/cli/cmd.Version=$(VERSION)" -o build/dist/fluxbase-linux-amd64 ${CLI_MAIN_PATH}
	@GOOS=linux GOARCH=arm64 go build -ldflags="-X github.com/nimbleflux/fluxbase/cli/cmd.Version=$(VERSION)" -o build/dist/fluxbase-linux-arm64 ${CLI_MAIN_PATH}
	@GOOS=windows GOARCH=amd64 go build -ldflags="-X github.com/nimbleflux/fluxbase/cli/cmd.Version=$(VERSION)" -o build/dist/fluxbase-windows-amd64.exe ${CLI_MAIN_PATH}
	@echo "${GREEN}Cross-compilation complete! Binaries in build/dist/${NC}"

# ═══════════════════════════════════════════════════════════════════════════════
# CODEBASE VISUALIZATION
# ═══════════════════════════════════════════════════════════════════════════════
#
# Recommended visualizations (clean, curated, easy to read):
#   viz-architecture  - Simplified layered architecture diagram
#   viz-workflows     - Flow diagrams for common operations
#   viz-core-modules  - Dependency graph of core modules only
#
# Advanced visualizations (auto-generated, may be complex):
#   viz-deps          - Full external package dependencies
#   viz-internal-*    - Various layouts of complete internal dependencies
#   viz-uml           - PlantUML class diagrams
#
# Run 'make viz' to generate recommended visualizations.
# Run 'make viz-all' to generate everything.

viz: viz-architecture viz-workflows viz-core-modules ## Generate recommended visualizations (clean & readable)
	@echo ""
	@echo "${GREEN}✓ Recommended visualizations generated!${NC}"
	@echo ""
	@echo "${BLUE}Architecture:${NC}"
	@echo "  - architecture-simplified.svg  - Layered architecture overview"
	@echo ""
	@echo "${BLUE}Workflows:${NC}"
	@echo "  - flow-rest-api.svg           - REST API request flow"
	@echo "  - flow-authentication.svg     - Authentication flow"
	@echo "  - flow-edge-functions.svg     - Edge functions execution"
	@echo "  - flow-background-jobs.svg    - Background jobs processing"
	@echo "  - flow-file-storage.svg       - File storage operations"
	@echo "  - flow-realtime.svg           - Realtime WebSocket flow"
	@echo ""
	@echo "${BLUE}Core Dependencies:${NC}"
	@echo "  - core-modules.svg            - API, Auth, Database dependencies"
	@echo ""
	@echo "${YELLOW}Open these files in your browser or IDE to view!${NC}"

viz-architecture: ## Generate simplified layered architecture diagram
	@echo "${YELLOW}Generating layered architecture diagram...${NC}"
	@./scripts/viz-architecture.sh
	@echo "${GREEN}Architecture diagram: build/viz/architecture-simplified.svg${NC}"

viz-workflows: ## Generate workflow diagrams (REST, auth, functions, jobs, storage)
	@echo "${YELLOW}Generating workflow diagrams...${NC}"
	@./scripts/viz-workflows.sh
	@echo "${GREEN}Workflow diagrams: build/viz/flow-*.svg${NC}"

viz-deps: ## Generate package dependency graph (PNG)
	@echo "${YELLOW}Generating package dependency graph...${NC}"
	@mkdir -p build/viz
	@godepgraph -s github.com/nimbleflux/fluxbase/... 2>/dev/null | dot -Tpng -o build/viz/deps.png
	@echo "${GREEN}Dependency graph: build/viz/deps.png${NC}"

viz-deps-svg: ## Generate package dependency graph (SVG, interactive)
	@echo "${YELLOW}Generating package dependency graph (SVG)...${NC}"
	@mkdir -p build/viz
	@godepgraph -s github.com/nimbleflux/fluxbase/... 2>/dev/null | dot -Tsvg -o build/viz/deps.svg
	@echo "${GREEN}Dependency graph: build/viz/deps.svg${NC}"

viz-internal-detailed: ## [ADVANCED] Complete internal dependency graph (complex, may be hard to read)
	@echo "${YELLOW}Generating complete internal dependencies (force-directed)...${NC}"
	@echo "${RED}Warning: This generates a complex graph with many nodes. Try 'make viz-architecture' for a cleaner view.${NC}"
	@mkdir -p build/viz
	@goda graph -cluster ./internal/... 2>/dev/null | \
		sed 's|github.com/nimbleflux/fluxbase/internal/||g' | \
		sfdp -Tsvg -Gsize=30,30 -Goverlap=scale -Gsplines=true -Gsep=1.5 -Gnodesep=2 -o build/viz/internal-deps-complete.svg
	@echo "${GREEN}Complete internal dependencies: build/viz/internal-deps-complete.svg${NC}"

viz-internal-hierarchical: ## [ADVANCED] Complete internal graph with hierarchical layout
	@echo "${YELLOW}Generating complete internal dependencies (hierarchical)...${NC}"
	@mkdir -p build/viz
	@goda graph -cluster ./internal/... 2>/dev/null | \
		sed 's|github.com/nimbleflux/fluxbase/internal/||g' | \
		dot -Tsvg -Grankdir=TB -Gnodesep=1.5 -Granksep=2 -o build/viz/internal-deps-hierarchical.svg
	@echo "${GREEN}Hierarchical dependencies: build/viz/internal-deps-hierarchical.svg${NC}"

viz-core-modules: ## Generate dependency graph for core modules only (api, auth, database)
	@echo "${YELLOW}Generating core module dependencies...${NC}"
	@mkdir -p build/viz
	@goda graph "reach(./internal/api/... + ./internal/auth/... + ./internal/database/..., ./internal/...)" 2>/dev/null | \
		sed 's|github.com/nimbleflux/fluxbase/internal/||g' | \
		dot -Tsvg -Grankdir=TB -Gnodesep=1.5 -Granksep=2 -o build/viz/core-modules.svg
	@echo "${GREEN}Core module dependencies: build/viz/core-modules.svg${NC}"

viz-callgraph: ## Generate call graph (opens in browser)
	@echo "${YELLOW}Generating call graph visualization...${NC}"
	@echo "${BLUE}This will open a browser window. Press Ctrl+C to stop.${NC}"
	@go-callvis -group pkg,type -focus github.com/nimbleflux/fluxbase ./cmd/fluxbase

viz-callgraph-svg: ## Generate call graph as SVG file
	@echo "${YELLOW}Generating call graph SVG...${NC}"
	@mkdir -p build/viz
	@go-callvis -group pkg,type -format svg -file build/viz/callgraph -focus github.com/nimbleflux/fluxbase ./cmd/fluxbase 2>/dev/null || true
	@echo "${GREEN}Call graph: build/viz/callgraph.svg${NC}"

viz-uml: ## Generate UML class diagrams (PlantUML format)
	@echo "${YELLOW}Generating UML diagrams...${NC}"
	@mkdir -p build/viz
	@goplantuml -recursive -show-aggregations -show-compositions -show-implementations ./internal > build/viz/internal.puml
	@echo "${GREEN}UML diagram: build/viz/internal.puml${NC}"
	@echo "${YELLOW}View at: https://www.plantuml.com/plantuml/uml/ or use PlantUML extension${NC}"

viz-uml-api: ## Generate UML diagram for API package only
	@echo "${YELLOW}Generating API UML diagram...${NC}"
	@mkdir -p build/viz
	@goplantuml -show-aggregations -show-compositions -show-implementations ./internal/api > build/viz/api.puml
	@echo "${GREEN}API UML diagram: build/viz/api.puml${NC}"

viz-uml-auth: ## Generate UML diagram for Auth package only
	@echo "${YELLOW}Generating Auth UML diagram...${NC}"
	@mkdir -p build/viz
	@goplantuml -show-aggregations -show-compositions -show-implementations ./internal/auth > build/viz/auth.puml
	@echo "${GREEN}Auth UML diagram: build/viz/auth.puml${NC}"

viz-module-deps: ## Show module-level dependencies
	@echo "${YELLOW}Analyzing module dependencies...${NC}"
	@echo ""
	@echo "${BLUE}Direct dependencies:${NC}"
	@go list -m -f '{{if not .Indirect}}{{.Path}}{{end}}' all | grep -v "^$$" | sort
	@echo ""
	@echo "${BLUE}Run 'go mod graph' for full dependency tree${NC}"

viz-all: viz viz-deps-svg viz-internal-detailed viz-uml ## Generate all visualizations (recommended + advanced)
	@echo ""
	@echo "${GREEN}✓ All visualizations generated in build/viz/${NC}"
	@echo ""
	@echo "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
	@echo "${BLUE}Recommended Visualizations (clean & readable):${NC}"
	@echo "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
	@echo ""
	@echo "  ${GREEN}architecture-simplified.svg${NC}   - Layered architecture overview"
	@echo "  ${GREEN}flow-*.svg${NC}                    - Workflow diagrams (REST, auth, functions, jobs, storage, realtime)"
	@echo "  ${GREEN}core-modules.svg${NC}              - Core module dependencies"
	@echo ""
	@echo "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
	@echo "${BLUE}Advanced Visualizations (detailed, complex):${NC}"
	@echo "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
	@echo ""
	@echo "  deps.svg                     - Full package dependency graph"
	@echo "  internal-deps-complete.svg   - Complete internal dependencies (complex)"
	@echo "  internal.puml                - UML class diagrams (PlantUML)"
	@echo ""
	@ls -lh build/viz/ | grep -E '\.(svg|puml)$$' | tail -20
