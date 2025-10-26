.PHONY: help build run test clean migrate-up migrate-down docker-build docker-run dev admin-dev admin-install

# Variables
BINARY_NAME=fluxbase
MAIN_PATH=cmd/fluxbase/main.go
DOCKER_IMAGE=fluxbase:latest
GO_VERSION=1.22

# Colors for output
RED=\033[0;31m
GREEN=\033[0;32m
YELLOW=\033[1;33m
NC=\033[0m # No Color

help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  ${GREEN}%-20s${NC} %s\n", $$1, $$2}'

build: build-admin ## Build the binary with embedded admin UI
	@echo "${YELLOW}Building ${BINARY_NAME}...${NC}"
	@go build -ldflags="-s -w" -o ${BINARY_NAME} ${MAIN_PATH}
	@echo "${GREEN}Build complete!${NC}"

build-admin: ## Build admin UI for production
	@echo "${YELLOW}Building admin UI...${NC}"
	@cd admin && npm run build
	@echo "${YELLOW}Copying admin UI to embed location...${NC}"
	@rm -rf internal/adminui/dist
	@cp -r admin/dist internal/adminui/dist
	@echo "${GREEN}Admin UI ready for embedding!${NC}"

admin-dev: ## Run admin UI in development mode
	@if [ ! -d "admin/node_modules" ]; then \
		echo "${YELLOW}Node modules not found. Installing dependencies...${NC}"; \
		cd admin && npm install; \
	fi
	@echo "${YELLOW}Starting admin UI development server...${NC}"
	@echo "${GREEN}Admin UI will be available at http://localhost:5173${NC}"
	@echo "${YELLOW}Make sure the backend is running on http://localhost:8080${NC}"
	@echo "${YELLOW}Press Ctrl+C to stop the server${NC}"
	@cd admin && ./dev.sh

admin-install: ## Install admin UI dependencies
	@echo "${YELLOW}Installing admin UI dependencies...${NC}"
	@cd admin && npm install
	@echo "${GREEN}Admin UI dependencies installed!${NC}"

run: ## Run the application
	@echo "${YELLOW}Starting ${BINARY_NAME}...${NC}"
	@go run ${MAIN_PATH}

dev: ## Run in development mode with hot reload (requires air)
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "${RED}Air is not installed. Install it with: go install github.com/cosmtrek/air@latest${NC}"; \
		exit 1; \
	fi

test: ## Run all tests (unit + integration + e2e)
	@echo "${YELLOW}Running all tests...${NC}"
	@make test-unit
	@make test-e2e
	@echo "${GREEN}All tests complete!${NC}"

test-quick: ## Run all tests without database setup
	@echo "${YELLOW}Running tests...${NC}"
	@go test -v -race -cover ./...
	@echo "${GREEN}Tests complete!${NC}"

test-coverage: ## Run tests with coverage report
	@echo "${YELLOW}Running tests with coverage...${NC}"
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "${GREEN}Coverage report generated: coverage.html${NC}"

clean: ## Clean build artifacts
	@echo "${YELLOW}Cleaning...${NC}"
	@rm -f ${BINARY_NAME}
	@rm -f coverage.out coverage.html
	@echo "${GREEN}Clean complete!${NC}"

deps: ## Download dependencies
	@echo "${YELLOW}Downloading dependencies...${NC}"
	@go mod download
	@go mod tidy
	@echo "${GREEN}Dependencies ready!${NC}"

migrate-up: ## Run database migrations
	@echo "${YELLOW}Running migrations...${NC}"
	@migrate -path internal/database/migrations -database "postgresql://postgres:postgres@localhost:5432/fluxbase?sslmode=disable" up
	@echo "${GREEN}Migrations complete!${NC}"

migrate-down: ## Rollback database migrations
	@echo "${YELLOW}Rolling back migrations...${NC}"
	@migrate -path internal/database/migrations -database "postgresql://postgres:postgres@localhost:5432/fluxbase?sslmode=disable" down
	@echo "${GREEN}Rollback complete!${NC}"

migrate-create: ## Create a new migration (usage: make migrate-create name=migration_name)
	@if [ -z "$(name)" ]; then \
		echo "${RED}Error: Please provide a migration name. Usage: make migrate-create name=migration_name${NC}"; \
		exit 1; \
	fi
	@echo "${YELLOW}Creating migration: $(name)...${NC}"
	@migrate create -ext sql -dir internal/database/migrations -seq $(name)
	@echo "${GREEN}Migration created!${NC}"

db-setup: ## Setup PostgreSQL database
	@echo "${YELLOW}Setting up database...${NC}"
	@createdb -h localhost -U postgres fluxbase 2>/dev/null || echo "Database already exists"
	@psql -h localhost -U postgres -d fluxbase < example/create_tables.sql
	@echo "${GREEN}Database setup complete!${NC}"

docker-build: ## Build Docker image
	@echo "${YELLOW}Building Docker image...${NC}"
	@docker build -t ${DOCKER_IMAGE} .
	@echo "${GREEN}Docker build complete!${NC}"

docker-run: ## Run Docker container
	@echo "${YELLOW}Starting Docker container...${NC}"
	@docker run -d \
		--name fluxbase \
		-p 8080:8080 \
		-e FLUXBASE_DATABASE_HOST=host.docker.internal \
		-v $(PWD)/config:/root/config \
		-v $(PWD)/storage:/root/storage \
		${DOCKER_IMAGE}
	@echo "${GREEN}Container started!${NC}"

docker-stop: ## Stop Docker container
	@echo "${YELLOW}Stopping Docker container...${NC}"
	@docker stop fluxbase && docker rm fluxbase
	@echo "${GREEN}Container stopped!${NC}"

lint: ## Run linters
	@echo "${YELLOW}Running linters...${NC}"
	@golangci-lint run ./...
	@echo "${GREEN}Linting complete!${NC}"

fmt: ## Format code
	@echo "${YELLOW}Formatting code...${NC}"
	@go fmt ./...
	@echo "${GREEN}Formatting complete!${NC}"

vet: ## Run go vet
	@echo "${YELLOW}Running go vet...${NC}"
	@go vet ./...
	@echo "${GREEN}Vet complete!${NC}"

install: build ## Build and install the binary
	@echo "${YELLOW}Installing ${BINARY_NAME}...${NC}"
	@go install ${MAIN_PATH}
	@echo "${GREEN}Installation complete!${NC}"

all: clean deps fmt vet test build ## Run all checks and build

setup-dev: ## Set up development environment
	@echo "${YELLOW}Setting up development environment...${NC}"
	@go mod download
	@go install github.com/cosmtrek/air@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@cp .env.example .env 2>/dev/null || true
	@echo "${GREEN}Development environment ready!${NC}"

test-unit: ## Run unit tests only
	@echo "${YELLOW}Running unit tests...${NC}"
	@go test -v -short -race -cover ./internal/...
	@echo "${GREEN}Unit tests complete!${NC}"

test-integration: ## Run integration tests (requires database and MailHog)
	@echo "${YELLOW}Running integration tests...${NC}"
	@go test -v -race -cover -tags=integration ./...
	@echo "${GREEN}Integration tests complete!${NC}"

test-email: ## Run email integration tests with MailHog
	@echo "${YELLOW}Running email integration tests...${NC}"
	@echo "${YELLOW}Make sure MailHog is running (make mailhog-start)${NC}"
	@MAILHOG_HOST=localhost go test -v -race -tags=integration ./internal/email/...
	@echo "${GREEN}Email tests complete!${NC}"

test-e2e: ## Run end-to-end tests (requires database)
	@echo "${YELLOW}Setting up test database...${NC}"
	@./test/scripts/setup_test_db.sh
	@echo "${YELLOW}Running end-to-end tests...${NC}"
	@go test -v -race -cover ./test/... -run E2E
	@echo "${GREEN}End-to-end tests complete!${NC}"

test-e2e-quick: ## Run E2E tests without database setup (faster)
	@echo "${YELLOW}Running end-to-end tests (quick mode)...${NC}"
	@go test -v -race -cover ./test/... -run E2E
	@echo "${GREEN}End-to-end tests complete!${NC}"

test-load: ## Run load tests with k6
	@echo "${YELLOW}Running load tests...${NC}"
	@if command -v k6 > /dev/null; then \
		k6 run test/k6/load-test.js; \
	else \
		echo "${RED}k6 is not installed. Install it from https://k6.io/docs/getting-started/installation${NC}"; \
		exit 1; \
	fi
	@echo "${GREEN}Load tests complete!${NC}"

docs-install: ## Install documentation dependencies
	@echo "${YELLOW}Installing documentation dependencies...${NC}"
	@cd docs && npm install
	@echo "${GREEN}Documentation dependencies installed!${NC}"

docs-server: ## Start documentation server (recommended)
	@make docs-dev

docs-dev: ## Run documentation in development mode
	@if [ ! -d "docs/node_modules" ]; then \
		echo "${YELLOW}Node modules not found. Installing dependencies...${NC}"; \
		cd docs && npm install; \
	fi
	@echo "${YELLOW}Starting documentation server...${NC}"
	@echo "${GREEN}Documentation will be available at http://localhost:3000${NC}"
	@echo "${YELLOW}Press Ctrl+C to stop the server${NC}"
	@cd docs && NODE_OPTIONS= npm start -- --host 0.0.0.0

docs-stop: ## Stop documentation server
	@echo "${YELLOW}Stopping documentation server...${NC}"
	@pkill -f "docusaurus start" || echo "No documentation server running"
	@echo "${GREEN}Documentation server stopped!${NC}"

docs-build: ## Build documentation for production
	@echo "${YELLOW}Building documentation...${NC}"
	@cd docs && npm run build
	@echo "${GREEN}Documentation built in docs/build/${NC}"

docker-dev: ## Run development environment with Docker Compose
	@echo "${YELLOW}Starting development environment...${NC}"
	@docker-compose -f .devcontainer/docker-compose.yml up -d
	@echo "${GREEN}Development environment started!${NC}"

docker-dev-stop: ## Stop development environment
	@echo "${YELLOW}Stopping development environment...${NC}"
	@docker-compose -f .devcontainer/docker-compose.yml down
	@echo "${GREEN}Development environment stopped!${NC}"

docker-dev-logs: ## Show development environment logs
	@docker-compose -f .devcontainer/docker-compose.yml logs -f

mailhog-start: ## Start MailHog for email testing
	@echo "${YELLOW}Starting MailHog...${NC}"
	@docker-compose up -d mailhog
	@echo "${GREEN}MailHog started!${NC}"
	@echo "${GREEN}SMTP server: localhost:1025${NC}"
	@echo "${GREEN}Web UI: http://localhost:8025${NC}"

mailhog-stop: ## Stop MailHog
	@echo "${YELLOW}Stopping MailHog...${NC}"
	@docker-compose stop mailhog
	@echo "${GREEN}MailHog stopped!${NC}"

mailhog-logs: ## Show MailHog logs
	@docker-compose logs -f mailhog

release: ## Create a new release (requires Release Please)
	@echo "${YELLOW}Creating release...${NC}"
	@if [ -z "$(version)" ]; then \
		echo "${RED}Error: Please provide a version. Usage: make release version=1.0.0${NC}"; \
		exit 1; \
	fi
	@git tag v$(version)
	@git push origin v$(version)
	@echo "${GREEN}Release v$(version) created!${NC}"

ci-local: ## Run CI pipeline locally
	@echo "${YELLOW}Running CI pipeline locally...${NC}"
	@make fmt
	@make vet
	@make lint
	@make test
	@make build
	@echo "${GREEN}CI pipeline complete!${NC}"

.DEFAULT_GOAL := help