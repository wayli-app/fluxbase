# Fluxbase DevContainer

This devcontainer provides a complete development environment for Fluxbase with all necessary tools pre-installed.

## What's Included

### Languages & Runtimes

- Go 1.25
- Node.js 20
- PostgreSQL client
- Redis client

### Development Tools

- **Go Tools**: gopls, dlv, golangci-lint, air, migrate, swag, mockery, staticcheck
- **Node Tools**: TypeScript, ESLint, Prettier, tsx, nodemon
- **Testing**: gotestsum, ginkgo
- **Database**: psql, pgAdmin 4, SQLTools
- **Utilities**: git, gh, docker, make, httpie, jq, tree

### VS Code Extensions

#### Essential

- **Claude Code** (saoudrizwan.claude-dev) - AI assistant
- **Go** (golang.go) - Go language support
- **SQLTools** - Database management

#### Development

- Docker, Kubernetes support
- Makefile tools
- Git tools (GitLens, Git Graph)
- API testing (Thunder Client, REST Client)
- Code quality (spell checker, TODO tree)

#### Languages

- TypeScript/JavaScript with ESLint & Prettier
- Markdown with preview
- YAML, TOML support
- Shell scripting support

### Services

All services are pre-configured and ready to use:

- **PostgreSQL 16**: Main database
- **Redis 7**: Caching and sessions
- **pgAdmin 4**: Database management UI
- **MailHog**: Email testing

## Getting Started

### Open in DevContainer

1. Install prerequisites:
   - [VS Code](https://code.visualstudio.com/)
   - [Docker Desktop](https://www.docker.com/products/docker-desktop)
   - [Dev Containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)

2. Open the project:

   ```bash
   code /Users/bart/Dev/fluxbase
   ```

3. VS Code will prompt: "Reopen in Container" - Click it!
   - Or press `F1` → "Dev Containers: Reopen in Container"

4. Wait for the container to build (first time takes ~5-10 minutes)

5. Once ready, you'll see a success message in the terminal

### Quick Commands

```bash
# Start development server with hot-reload
make dev

# Run tests
make test

# Run specific test types
make test-unit
make test-integration

# Build the binary
make build

# View all commands
make help

# Start documentation server
make docs-dev
```

## Environment

### Database Connections

Two PostgreSQL databases are pre-configured:

1. **fluxbase_dev** - Development database
   - Host: `postgres`
   - Port: `5432`
   - User: `postgres`
   - Password: `postgres`

2. **fluxbase_test** - Test database
   - Host: `postgres`
   - Port: `5432`
   - User: `postgres`
   - Password: `postgres`

### SQLTools Configuration

Database connections are automatically configured in SQLTools. Access them via:

- Click the database icon in the left sidebar
- Or press `Ctrl+Shift+P` → "SQLTools: Connect"

### Service URLs

When services are running:

- Fluxbase API: http://localhost:8080
- pgAdmin: http://localhost:5050
- MailHog UI: http://localhost:8025
- Documentation: http://localhost:3000 (when `make docs-dev` is running)

## Files Structure

```
.devcontainer/
├── devcontainer.json      # Container configuration
├── docker-compose.yml     # Services definition
├── Dockerfile             # Development image
├── init-db.sql           # Database initialization
├── post-create.sh        # Setup script
└── README.md             # This file
```

## Customization

### Add VS Code Extensions

Edit `.devcontainer/devcontainer.json`:

```json
{
  "customizations": {
    "vscode": {
      "extensions": ["your-extension-id"]
    }
  }
}
```

### Add System Packages

Edit `.devcontainer/Dockerfile`:

```dockerfile
RUN apt-get update && apt-get install -y \
    your-package-name \
    && rm -rf /var/lib/apt/lists/*
```

### Modify Services

Edit `.devcontainer/docker-compose.yml` to add or configure services.

## Troubleshooting

### Container Won't Build

1. Check Docker is running
2. Try rebuilding: `F1` → "Dev Containers: Rebuild Container"
3. Check Docker logs for errors

### Database Connection Issues

```bash
# Check if PostgreSQL is running
pg_isready -h postgres -U postgres

# View PostgreSQL logs
docker logs fluxbase-postgres-dev
```

### Port Conflicts

If ports are already in use, edit `.devcontainer/devcontainer.json` and change the `forwardPorts` array.

### Slow Performance

1. Increase Docker resources (CPU/Memory)
2. Check Docker Desktop settings
3. Use volumes for Go modules (already configured)

## Features

### Hot Reload

The devcontainer includes Air for hot-reload:

```bash
make dev  # Automatically reloads on file changes
```

### Pre-configured Linting

```bash
make lint  # Run golangci-lint
make fmt   # Format code
make vet   # Run go vet
```

### Database Migrations

```bash
make migrate-up    # Apply migrations
make migrate-down  # Rollback migrations
```

## VS Code Tips

### Keyboard Shortcuts

- `Ctrl+\`` - Toggle terminal
- `F5` - Start debugging
- `Ctrl+Shift+P` - Command palette
- `Ctrl+K Ctrl+O` - Open folder

### Recommended Workflow

1. Use Claude Code for AI assistance
2. Use SQLTools to explore database
3. Use Thunder Client for API testing
4. Use GitLens for git history
5. Use TODO Tree to track tasks

## Updating the Container

When the Dockerfile or configuration changes:

1. `F1` → "Dev Containers: Rebuild Container"
2. Wait for rebuild to complete
3. The container will restart with new configuration

## Performance Tips

1. Use volumes for node_modules and Go modules (already configured)
2. Close unused services in docker-compose.yml
3. Use `make build` instead of `go build` for optimized builds
4. Run tests in parallel: `go test -parallel 4 ./...`

## Support

- Check `.claude/instructions.md` for development guidelines
- Read `TODO.md` for the implementation plan
- View `IMPLEMENTATION_PLAN.md` for detailed sprints
- Open an issue on GitHub for bugs

## Next Steps

1. Check that everything works: `make test`
2. Read the development guidelines: `.claude/instructions.md`
3. Explore the codebase and documentation
4. Use Claude Code for AI-powered development assistance

Happy coding! 🚀
