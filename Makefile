.PHONY: help dev build clean test migrate-up migrate-down migrate-create db-reset deps setup-dev docs docs-build version docker-build docker-push release

# Variables
BINARY_NAME=fluxbase
MAIN_PATH=cmd/fluxbase/main.go

# Version variables
VERSION := $(shell cat VERSION)
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -s -w -X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildDate=$(BUILD_DATE)

# Docker variables
DOCKER_REGISTRY ?= ghcr.io
DOCKER_ORG ?= wayli-app
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
	@lsof -ti:5173 | xargs -r kill -9 2>/dev/null || true
	@if [ ! -d "admin/node_modules" ]; then \
		echo "${YELLOW}Installing admin UI dependencies...${NC}"; \
		cd admin && npm install; \
	fi
	@echo "${YELLOW}Building admin UI...${NC}"
	@cd admin && npm run build
	@rm -rf internal/adminui/dist
	@cp -r admin/dist internal/adminui/dist
	@echo "${GREEN}Backend:${NC}     http://localhost:8080"
	@echo "${GREEN}Frontend:${NC}    http://localhost:5173/admin/"
	@echo "${GREEN}Admin Login:${NC} http://localhost:5173/admin/login"
	@echo ""
	@echo "${YELLOW}Press Ctrl+C to stop both servers${NC}"
	@echo ""
	@./run-server.sh & \
	cd admin && npm run dev

version: ## Show version information
	@echo "${GREEN}Version:${NC}    $(VERSION)"
	@echo "${GREEN}Commit:${NC}     $(COMMIT)"
	@echo "${GREEN}Build Date:${NC} $(BUILD_DATE)"

build: ## Build production binary with embedded admin UI
	@echo "${YELLOW}Building admin UI...${NC}"
	@cd admin && npm run build
	@rm -rf internal/adminui/dist
	@cp -r admin/dist internal/adminui/dist
	@echo "${YELLOW}Building ${BINARY_NAME} v$(VERSION)...${NC}"
	@mkdir -p build/
	@go build -ldflags="$(LDFLAGS)" -o build/${BINARY_NAME} ${MAIN_PATH}
	@echo "${GREEN}Build complete: ${BINARY_NAME} v$(VERSION)${NC}"

clean: ## Clean build artifacts
	@echo "${YELLOW}Cleaning...${NC}"
	@rm -f build/${BINARY_NAME}
	@rm -f coverage.out coverage.html
	@rm -rf internal/adminui/dist
	@echo "${GREEN}Clean complete!${NC}"

test: ## Run all tests with race detector (short mode - skips slow tests, excludes e2e)
	@echo "${YELLOW}Running tests with race detector (short mode)...${NC}"
	@go test -timeout 2m -v -race -short -cover $(shell go list ./... | grep -v '/test/e2e')
	@echo "${GREEN}Tests complete!${NC}"

test-fast: ## Run all tests without race detector (faster, excludes e2e)
	@echo "${YELLOW}Running tests (fast mode)...${NC}"
	@go test -timeout 1m -v -short -cover $(shell go list ./... | grep -v '/test/e2e')
	@echo "${GREEN}Tests complete!${NC}"

test-full: ## Run ALL tests including e2e with race detector (may take 5-10 minutes)
	@echo "${YELLOW}Running full test suite with race detector...${NC}"
	@go test -timeout 15m -v -race -cover ./...
	@echo "${GREEN}Full test suite complete!${NC}"

test-e2e: ## Run e2e tests only (requires postgres, mailhog, minio services)
	@echo "${YELLOW}Running e2e tests...${NC}"
	@go test -v -race -timeout=5m ./test/e2e/...
	@echo "${GREEN}E2E tests complete!${NC}"

test-sdk: ## Run SDK tests (TypeScript)
	@echo "${YELLOW}Running SDK tests...${NC}"
	@cd sdk && npm test -- src/admin.test.ts src/auth.test.ts src/management.test.ts src/ddl.test.ts src/impersonation.test.ts src/settings.test.ts src/oauth.test.ts
	@echo "${GREEN}SDK tests complete!${NC}"

test-sdk-react: ## Build React SDK (includes type checking)
	@echo "${YELLOW}Building React SDK...${NC}"
	@cd sdk-react && npm run build
	@echo "${GREEN}React SDK build complete!${NC}"

test-integration: ## Run admin integration tests (requires running server)
	@echo "${YELLOW}Running admin integration tests...${NC}"
	@if ! curl -s http://localhost:8080/health > /dev/null; then \
		echo "${RED}Error: Fluxbase server not running on localhost:8080${NC}"; \
		echo "${YELLOW}Start server with: make dev${NC}"; \
		exit 1; \
	fi
	@cd examples/admin-setup && npm test
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
	@cd admin && npm install
	@cp .env.example .env 2>/dev/null || echo ".env already exists"
	@echo "${GREEN}Development environment ready!${NC}"
	@echo "${YELLOW}Next steps:${NC}"
	@echo "  1. Configure your database in .env"
	@echo "  2. Run: make migrate-up"
	@echo "  3. Run: make dev"

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

db-reset: ## Reset database (drop all schemas and run migrations)
	@echo "${YELLOW}Resetting database...${NC}"
	@echo "${YELLOW}Dropping all schemas except public...${NC}"
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "DROP SCHEMA IF EXISTS auth CASCADE;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "DROP SCHEMA IF EXISTS dashboard CASCADE;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "DROP SCHEMA IF EXISTS storage CASCADE;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "DROP SCHEMA IF EXISTS functions CASCADE;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "DROP SCHEMA IF EXISTS realtime CASCADE;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "DROP SCHEMA IF EXISTS _fluxbase CASCADE;" || true
	@echo "${YELLOW}Creating _fluxbase schema for migration tracking...${NC}"
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "CREATE SCHEMA IF NOT EXISTS _fluxbase;" || true
	@echo "${YELLOW}Running migrations...${NC}"
	@migrate -path internal/database/migrations -database 'postgresql://postgres:postgres@postgres:5432/fluxbase_dev?sslmode=disable&x-migrations-table="_fluxbase"."schema_migrations"&x-migrations-table-quoted=1' up
	@echo "${YELLOW}Granting permissions to fluxbase_app user...${NC}"
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "ALTER USER fluxbase_app WITH BYPASSRLS;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "ALTER USER fluxbase_app SET search_path TO public;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT USAGE, CREATE ON SCHEMA public TO fluxbase_app;" || true
	@docker exec fluxbase-postgres-dev psql -U postgres -d fluxbase_dev -c "GRANT ALL ON ALL TABLES IN SCHEMA _fluxbase TO fluxbase_app;" || true
	@echo "${GREEN}Database reset complete!${NC}"

docs: ## Serve Docusaurus documentation at http://localhost:3000
	@echo "${YELLOW}Starting Docusaurus documentation server...${NC}"
	@if [ ! -d "docs/node_modules" ]; then \
		echo "${YELLOW}Installing documentation dependencies...${NC}"; \
		cd docs && npm install; \
	fi
	@echo ""
	@echo "${GREEN}📚 Documentation will be available at:${NC}"
	@echo "  ${GREEN}http://localhost:3000${NC}"
	@echo ""
	@echo "${GREEN}New Pages Added:${NC}"
	@echo "  • API Cookbook (60+ examples)"
	@echo "  • Supabase Migration Guide"
	@echo "  • Advanced Guides (RLS, Performance, Scaling)"
	@echo "  • Example Applications (Todo, Blog, Chat)"
	@echo ""
	@echo "${YELLOW}Press Ctrl+C to stop the server${NC}"
	@echo ""
	@cd docs && npm start -- --host 0.0.0.0

docs-build: ## Build static documentation site for production
	@echo "${YELLOW}Building documentation site...${NC}"
	@if [ ! -d "docs/node_modules" ]; then \
		echo "${YELLOW}Installing documentation dependencies...${NC}"; \
		cd docs && npm install; \
	fi
	@cd docs && npm run build
	@echo "${GREEN}Documentation built successfully!${NC}"
	@echo "${YELLOW}Output:${NC} docs/build/"
	@echo "${YELLOW}To serve locally:${NC} cd docs && npm run serve"

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
