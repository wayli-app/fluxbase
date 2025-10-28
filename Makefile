.PHONY: help dev build clean test migrate-up migrate-down migrate-create deps setup-dev

# Variables
BINARY_NAME=fluxbase
MAIN_PATH=cmd/fluxbase/main.go

# Colors for output
GREEN=\033[0;32m
YELLOW=\033[1;33m
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
	@echo "${GREEN}Backend:${NC}     http://localhost:8080"
	@echo "${GREEN}Frontend:${NC}    http://localhost:5173/admin/"
	@echo "${GREEN}Admin Login:${NC} http://localhost:5173/admin/login"
	@echo ""
	@echo "${YELLOW}Press Ctrl+C to stop both servers${NC}"
	@echo ""
	@./run-server.sh & \
	cd admin && npm run dev

build: ## Build production binary with embedded admin UI
	@echo "${YELLOW}Building admin UI...${NC}"
	@cd admin && npm run build
	@rm -rf internal/adminui/dist
	@cp -r admin/dist internal/adminui/dist
	@echo "${YELLOW}Building ${BINARY_NAME}...${NC}"
	@go build -ldflags="-s -w" -o ${BINARY_NAME} ${MAIN_PATH}
	@echo "${GREEN}Build complete: ${BINARY_NAME}${NC}"

clean: ## Clean build artifacts
	@echo "${YELLOW}Cleaning...${NC}"
	@rm -f ${BINARY_NAME}
	@rm -f coverage.out coverage.html
	@rm -rf internal/adminui/dist
	@echo "${GREEN}Clean complete!${NC}"

test: ## Run all tests
	@echo "${YELLOW}Running tests...${NC}"
	@go test -v -race -cover ./...
	@echo "${GREEN}Tests complete!${NC}"

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
