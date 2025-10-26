# Getting Started with Fluxbase Development

Welcome to Fluxbase! This guide will get you up and running in minutes.

## 🎯 What is Fluxbase?

Fluxbase is a lightweight, single-binary Backend-as-a-Service (BaaS) alternative to Supabase. It provides:

- Auto-generated REST APIs from PostgreSQL schemas
- JWT authentication
- Realtime WebSocket subscriptions
- File storage (local or S3)
- Edge functions
- All in a single ~50MB Go binary!

## ⚡ Quick Start (Recommended: DevContainer)

### Prerequisites

- [VS Code](https://code.visualstudio.com/)
- [Docker Desktop](https://www.docker.com/products/docker-desktop)
- [Dev Containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)

### 3 Steps to Start

1. **Open in VS Code**

   ```bash
   code /Users/bart/Dev/fluxbase
   ```

2. **Reopen in Container**

   - Click "Reopen in Container" when prompted
   - Or: `F1` → "Dev Containers: Reopen in Container"
   - First build: ~5-10 minutes
   - Subsequent starts: ~30 seconds

3. **Start Developing**

   ```bash
   make dev  # Starts with hot-reload
   ```

4. **Verify**
   ```bash
   curl http://localhost:8080/health
   # Should return: {"status":"ok"}
   ```

**That's it!** You're ready to code. See [DevContainer Quick Start](.devcontainer/QUICK_START.md) for more.

## 🖥️ Local Development (Without DevContainer)

### Prerequisites

- Go 1.22+
- PostgreSQL 14+
- Node.js 20+ (for SDK development)
- Make

### Setup

1. **Clone Repository**

   ```bash
   git clone https://github.com/wayli-app/fluxbase.git
   cd fluxbase
   ```

2. **Install Dependencies**

   ```bash
   go mod download
   ```

3. **Setup PostgreSQL**

   ```bash
   # Create database
   createdb fluxbase

   # Run migrations
   make migrate-up
   ```

4. **Configure Environment**

   ```bash
   cp .env.example .env
   # Edit .env with your settings
   ```

5. **Run Development Server**

   ```bash
   make dev  # With hot-reload
   # Or
   go run cmd/fluxbase/main.go
   ```

6. **Verify**
   ```bash
   curl http://localhost:8080/health
   ```

## 🧪 Testing

```bash
# Run all tests
make test

# Run specific test types
make test-unit          # Unit tests
make test-integration   # Integration tests (requires DB)
make test-load          # k6 load tests

# With coverage
make test-coverage
```

## 📚 Documentation

### For Developers

- **[DevContainer Quick Start](.devcontainer/QUICK_START.md)** - Fast reference
- **[DevContainer Full Docs](.devcontainer/README.md)** - Complete guide
- **[Claude Instructions](.claude/instructions.md)** - Development guidelines
- **[TODO List](TODO.md)** - What needs to be built
- **[Implementation Plan](IMPLEMENTATION_PLAN.md)** - 6-week sprint plan

### For Users (Coming Soon)

- **[Documentation Site](docs/)** - Built with Docusaurus
- **API Reference** - REST API documentation
- **SDK Guides** - TypeScript and Go client libraries

## 🏗️ Project Structure

```
fluxbase/
├── cmd/fluxbase/          # Main application
├── internal/              # Private application code
│   ├── api/              # ✅ REST API (complete)
│   ├── auth/             # 🚧 Authentication (next sprint)
│   ├── config/           # ✅ Configuration (complete)
│   ├── database/         # ✅ Database layer (complete)
│   ├── realtime/         # 🚧 WebSocket (sprint 3)
│   ├── storage/          # 🚧 File storage (sprint 4)
│   └── functions/        # 🚧 Edge functions (sprint 9)
├── pkg/                  # Public libraries
├── test/                 # Integration tests
├── docs/                 # Documentation site
├── .devcontainer/        # DevContainer setup
└── migrations/           # Database migrations
```

## 🎯 Current Status

### ✅ Complete (100%)

- Core REST API engine
- PostgREST-compatible query syntax
- PostgreSQL schema introspection
- Dynamic endpoint generation
- Configuration management
- CI/CD pipeline
- Testing framework
- Documentation site
- DevContainer

### 🚧 Next Sprint: Authentication (Week 1)

- JWT token utilities
- User registration/login
- Session management
- Auth middleware
- Protected endpoints

See [TODO.md](TODO.md) for the complete task list.

## 🚀 Development Workflow

### Daily Workflow

1. Open project in DevContainer (or start services locally)
2. Pull latest changes: `git pull`
3. Run tests: `make test`
4. Start development server: `make dev`
5. Make changes (auto-reloads)
6. Run tests again
7. Commit and push

### Making Changes

1. Check [TODO.md](TODO.md) for current sprint tasks
2. Create a feature branch
3. Implement feature
4. Write tests (aim for 80% coverage)
5. Update documentation
6. Run `make ci-local` (fmt, vet, lint, test, build)
7. Commit with clear message
8. Push and create PR

### Using Make Commands

```bash
make help          # See all commands
make dev           # Start with hot-reload
make test          # Run tests
make build         # Build binary
make lint          # Run linters
make docs-dev      # Start docs server
```

## 🛠️ Available Tools

### In DevContainer

- **Claude Code** - AI-powered development
- **Go Tools** - gopls, dlv, golangci-lint, air
- **Database** - PostgreSQL 16, pgAdmin, SQLTools
- **Testing** - k6 load testing
- **API Tools** - Thunder Client, REST Client
- **Git Tools** - GitLens, Git Graph

### Services

- **PostgreSQL**: localhost:5432
- **Redis**: localhost:6379
- **pgAdmin**: http://localhost:5050
- **MailHog**: http://localhost:8025

## 📖 Learning Resources

### Understanding the Codebase

1. Start with [.claude/project.md](.claude/project.md) - Quick overview
2. Read [.claude/instructions.md](.claude/instructions.md) - Development guide
3. Check [internal/api/rest_handler.go](internal/api/rest_handler.go) - See how REST API works
4. Review [internal/database/schema_inspector.go](internal/database/schema_inspector.go) - Schema introspection

### API Examples

```bash
# List all tables
curl http://localhost:8080/api/rest/

# Query with filters
curl "http://localhost:8080/api/rest/posts?published=eq.true&limit=10"

# Create a record
curl -X POST http://localhost:8080/api/rest/posts \
  -H "Content-Type: application/json" \
  -d '{"title":"Hello","content":"World"}'
```

## 🎯 What to Build Next

According to [IMPLEMENTATION_PLAN.md](IMPLEMENTATION_PLAN.md):

### Sprint 1: Authentication (Week 1) - START HERE

- [ ] JWT token utilities (4h)
- [ ] User registration endpoint (4h)
- [ ] Login endpoint (3h)
- [ ] Auth middleware (4h)
- [ ] Session management (4h)

**Goal**: Secure all APIs with JWT authentication

### Future Sprints

- Sprint 2: Enhanced REST API (Week 2)
- Sprint 3: Realtime Engine (Week 3)
- Sprint 4: Storage Service (Week 4)
- Sprint 5: TypeScript SDK (Week 5)
- Sprint 6: Admin UI (Week 6)

## 💡 Pro Tips

1. **Use DevContainer**: Everything is pre-configured!
2. **Use Claude Code**: AI assistant in VS Code
3. **Read TODO.md**: Always know what to work on next
4. **Write Tests First**: TDD makes development faster
5. **Run `make help`**: See all available commands
6. **Check Examples**: Look at test files for usage examples
7. **Use Hot Reload**: `make dev` auto-reloads on changes

## 🐛 Troubleshooting

### Container Won't Start

```bash
# Rebuild
F1 → "Dev Containers: Rebuild Container"

# Check Docker
docker ps
docker-compose logs
```

### Database Connection Issues

```bash
# Test connection
pg_isready -h localhost -U postgres

# Or in DevContainer
pg_isready -h postgres -U postgres
```

### Tests Failing

```bash
# Clean and rebuild
make clean
make deps
make test
```

### Build Issues

```bash
# Update dependencies
go mod tidy
go mod download

# Clear cache
go clean -cache -modcache -i -r
```

## 📞 Getting Help

1. **Documentation**: Check `.devcontainer/README.md`
2. **Claude Code**: Ask the AI assistant in VS Code
3. **TODO**: Check [TODO.md](TODO.md) for context
4. **Issues**: Open a GitHub issue
5. **Code**: Read existing code for examples

## ✅ Verification Checklist

Before starting development, verify:

```bash
# In DevContainer
bash .devcontainer/test-setup.sh

# Or manually
go version          # Should be 1.22+
make test           # All tests pass
make build          # Binary builds
curl localhost:8080/health  # Server responds
```

All green? You're ready! 🎉

## 🎉 Next Steps

1. ✅ Set up environment (you just did this!)
2. 📖 Read [TODO.md](TODO.md) - Understand what needs to be built
3. 🏃 Start Sprint 1 - Begin with JWT authentication
4. 💻 Use Claude Code - AI-powered development
5. 🧪 Write tests - Maintain 80% coverage
6. 📝 Update docs - Keep documentation current

**Ready to build Fluxbase!** 🚀

---

**Quick Links**:

- [DevContainer Quick Start](.devcontainer/QUICK_START.md)
- [TODO List](TODO.md)
- [Implementation Plan](IMPLEMENTATION_PLAN.md)
- [Development Guidelines](.claude/instructions.md)
