# Fluxbase DevContainer Quick Start

## 🚀 Get Started in 3 Steps

### 1. Open in Container

```
VS Code → Reopen in Container
```

Wait 5-10 minutes for first build (subsequent starts: ~30 seconds)

### 2. Verify Setup

```bash
bash .devcontainer/test-setup.sh
```

### 3. Start Developing

```bash
make dev  # Start with hot-reload
```

## 📝 Common Commands

```bash
# Development
make dev              # Start with hot-reload (backend + admin UI)
make build            # Build production binary

# Testing
make test             # Unit tests (~2min)
make test-e2e         # End-to-end tests
make test-full        # All tests including E2E (10min+)
make test-sdk         # TypeScript SDK tests

# Code Quality
make fmt              # Format Go code
make lint-go          # Go linting with golangci-lint
make lint-typescript  # TypeScript linting (admin UI + SDKs)

# Database
make db-reset         # Reset database (preserve user data)
make db-reset-full    # Full reset (destroys all data)

# Documentation
make docs             # Start docs dev server
make docs-build       # Build static docs

# Docker
make docker-build     # Build Docker image

# All Commands
make help             # Show all commands
```

## 🌐 Service URLs

| Service       | URL                   | Credentials               |
| ------------- | --------------------- | ------------------------- |
| Fluxbase API  | http://localhost:8080 | -                         |
| MailHog       | http://localhost:8025 | -                         |
| Documentation | http://localhost:4321 | -                         |

## 🗄️ Database

```bash
# Quick connect
psql -h postgres -U postgres -d fluxbase_dev

# Or use SQLTools in VS Code sidebar
```

**Databases**:

- `fluxbase_dev` - Development
- `fluxbase_test` - Testing

**Credentials**:

- Host: `postgres`
- User: `postgres`
- Password: `postgres`

## 🛠️ Installed Tools

### Go

- gopls, dlv, golangci-lint, air, migrate, swag, mockery, staticcheck

### Node.js

- typescript, eslint, prettier, tsx, nodemon

### Testing

- gotestsum, ginkgo

### Database

- psql, redis-cli, pgAdmin

### Utilities

- git, gh, docker, make, jq, httpie, tree

## 🎨 VS Code Extensions

### Essential

- **Claude Code** - AI assistant
- **Go** - Full Go support
- **SQLTools** - Database management

### Useful Shortcuts

- `Ctrl+` ` - Toggle terminal
- `F5` - Start debugging
- `Ctrl+Shift+P` - Command palette
- `Ctrl+P` - Quick file open

## 📋 Project Structure

```
fluxbase/
├── cmd/fluxbase/       # Main entry point
├── internal/           # Private app code
│   ├── ai/            # AI features (chatbots, KBs, vector search)
│   ├── api/           # REST API, GraphQL, handlers
│   ├── auth/          # Authentication (JWT, OAuth, SAML, MFA)
│   ├── branching/     # Database branching
│   ├── config/        # Configuration
│   ├── database/      # DB layer, migrations
│   ├── functions/     # Edge functions (Deno runtime)
│   ├── jobs/          # Background jobs
│   ├── realtime/      # WebSocket subscriptions
│   └── storage/       # File storage, logging
├── admin/             # Admin UI (React 19, Vite)
├── sdk/               # TypeScript SDK
├── sdk-react/         # React SDK (hooks)
├── cli/               # CLI tool
├── docs/              # Documentation
├── test/              # E2E tests
├── deploy/            # Docker, Helm, deploy configs
└── Makefile           # Build & dev commands
```

## 💡 Pro Tips

1. **Use Claude Code**: AI-powered development - just ask!
2. **SQLTools**: Database icon in sidebar for queries
3. **Thunder Client**: Test APIs right in VS Code
4. **GitLens**: See git blame inline
5. **TODO Tree**: Track tasks from code comments
6. **Hot Reload**: Changes apply automatically with `make dev`

## 🐛 Quick Troubleshooting

### Container Issues

```bash
# Rebuild
F1 → "Dev Containers: Rebuild Container"

# Check logs
docker compose logs -f
```

### Database Issues

```bash
# Test connection
pg_isready -h postgres -U postgres

# View logs
docker logs fluxbase-postgres-dev
```

### Go Issues

```bash
# Reinstall dependencies
go mod download
go mod tidy

# Rebuild
go build cmd/fluxbase/main.go
```

## 📚 Documentation

- **This Guide**: Quick start reference
- **Full Docs**: `.devcontainer/README.md`
- **Developer Guide**: `docs/src/content/docs/guides/developer-guide.md`
- **Feature Guides**: `docs/src/content/docs/guides/`
- **SDK Docs**: `docs/src/content/docs/sdk/`

## ✅ Health Check

Run this to verify everything:

```bash
bash .devcontainer/test-setup.sh
```

Should show all green checkmarks ✓

## 🎉 You're Ready!

Start building:

```bash
make dev
```

Open http://localhost:8080/health - should return `{"status":"ok"}`

You're all set! Check out the documentation in `docs/` to learn more.

---

**Need Help?** Use Claude Code or check `.devcontainer/README.md`
