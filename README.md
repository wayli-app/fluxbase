# Fluxbase

[![CI](https://github.com/fluxbase-eu/fluxbase/actions/workflows/ci.yml/badge.svg)](https://github.com/fluxbase-eu/fluxbase/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/fluxbase-eu/fluxbase/branch/main/graph/badge.svg)](https://codecov.io/gh/fluxbase-eu/fluxbase)

> **Beta Software**: Fluxbase is currently in beta. While we're working hard to stabilize the API and features, you may encounter breaking changes between versions. We welcome feedback and contributions!

A lightweight, single-binary Backend-as-a-Service (BaaS) alternative to Supabase. Fluxbase provides essential backend services including auto-generated REST APIs, authentication, realtime subscriptions, file storage, and edge functions - all in a single Go binary with PostgreSQL as the only dependency.

## Features

### Core Services

- **PostgREST-compatible REST API**: Auto-generates CRUD endpoints from your PostgreSQL schema
- **Authentication**: Email/password, magic links, JWT tokens with session management
- **Realtime Subscriptions**: WebSocket-based live data updates using PostgreSQL LISTEN/NOTIFY
- **Storage**: File upload/download with access policies (local filesystem or S3)
- **Edge Functions**: JavaScript/TypeScript function execution with Deno runtime
- **Background Jobs**: Long-running tasks with progress tracking, retry logic, and real-time updates
- **Schema Introspection**: Automatic API generation from database tables

### Key Highlights

- Single binary deployment (~80MB)
- PostgreSQL as the only external dependency
- Automatic REST endpoint generation
- Row Level Security (RLS) support
- TypeScript and Go SDKs (coming soon)
- High performance (1000+ concurrent connections)

## Quick Start

> 📖 **See [GETTING_STARTED.md](GETTING_STARTED.md) for complete setup instructions**

### Try it Now: Docker Compose (2 minutes)

Get Fluxbase running instantly with Docker:

```bash
# Clone the repository
git clone https://github.com/fluxbase-eu/fluxbase.git
cd fluxbase/deploy

# Start all services (PostgreSQL + Fluxbase + MinIO)
docker-compose up -d

# Check health
curl http://localhost:8080/health

# Access admin dashboard
open http://localhost:8080
```

That's it! Fluxbase is now running at http://localhost:8080

**Default credentials:**

- Database: `postgres:postgres`
- MinIO: `minioadmin:minioadmin`

### Recommended: DevContainer (5 minutes)

The fastest way to get started:

1. Install [VS Code](https://code.visualstudio.com/) + [Docker Desktop](https://www.docker.com/products/docker-desktop)
2. Open project in VS Code
3. Click "Reopen in Container"
4. Wait for setup (~5-10 minutes first time)
5. Run `make dev`

All tools pre-installed! See [.devcontainer/QUICK_START.md](.devcontainer/QUICK_START.md)

### Alternative: Local Setup

#### Prerequisites

- Go 1.25+
- PostgreSQL 15+

#### Installation

```bash
# Clone the repository
git clone https://github.com/fluxbase-eu/fluxbase.git
cd fluxbase

# Install dependencies
go mod download

# Setup database
createdb fluxbase
make migrate-up

# Create config
cp .env.example .env

# Run development server
make dev
```

### Configuration

Create a `fluxbase.yaml` file (or use environment variables):

```yaml
server:
  address: ":8080"

database:
  host: "localhost"
  port: 5432
  user: "postgres"
  password: "postgres"
  database: "fluxbase"

auth:
  jwt_secret: "your-secret-key-change-in-production"

storage:
  provider: "local"
  local_path: "./storage"

realtime:
  enabled: true

debug: true
```

### Environment Variables

All configuration options can be set via environment variables with the `FLUXBASE_` prefix:

```bash
export FLUXBASE_DATABASE_HOST=localhost
export FLUXBASE_DATABASE_PORT=5432
export FLUXBASE_DATABASE_USER=postgres
export FLUXBASE_DATABASE_PASSWORD=postgres
export FLUXBASE_AUTH_JWT_SECRET=your-secret-key
export FLUXBASE_SERVER_BODY_LIMIT=2147483648  # 2GB (default) - max size for HTTP request bodies including file uploads
```

## CLI Installation

The Fluxbase CLI provides command-line access to manage your Fluxbase platform.

### Install Script (Recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/fluxbase-eu/fluxbase/main/install-cli.sh | bash
```

The script automatically detects your OS and architecture, downloads the appropriate binary, and installs it to `/usr/local/bin`.

### Verify Installation

```bash
fluxbase version
```

For manual installation, platform-specific instructions, and shell completion setup, see the [CLI Documentation](https://fluxbase.eu/cli/installation).

## API Usage

### Auto-Generated REST Endpoints

Fluxbase automatically generates REST endpoints for all your database tables:

```bash
# Get all records from a table
curl http://localhost:8080/api/v1/tables/users

# Get a specific record
curl http://localhost:8080/api/v1/tables/users/123

# Create a new record
curl -X POST http://localhost:8080/api/v1/tables/users \
  -H "Content-Type: application/json" \
  -d '{"name": "John Doe", "email": "john@example.com"}'

# Update a record
curl -X PATCH http://localhost:8080/api/v1/tables/users/123 \
  -H "Content-Type: application/json" \
  -d '{"name": "Jane Doe"}'

# Delete a record
curl -X DELETE http://localhost:8080/api/v1/tables/users/123
```

### Query Parameters (PostgREST-compatible)

```bash
# Filtering
curl "http://localhost:8080/api/v1/tables/users?age=gt.18&name=like.*john*"

# Selecting specific columns
curl "http://localhost:8080/api/v1/tables/users?select=id,name,email"

# Ordering
curl "http://localhost:8080/api/v1/tables/users?order=created_at.desc"

# Pagination
curl "http://localhost:8080/api/v1/tables/users?limit=10&offset=20"

# Complex filters
curl "http://localhost:8080/api/v1/tables/posts?or=(status.eq.draft,status.eq.published)&author_id=eq.1"
```

## Database Schema

Fluxbase uses special schemas for its internal operations:

- `auth.*` - Authentication and user management
- `storage.*` - File storage metadata
- `realtime.*` - Realtime subscription configuration
- `functions.*` - Edge functions registry

Your application tables should be created in the `public` schema or custom schemas.

### Example Application Schema

```sql
-- Create your application tables
CREATE TABLE posts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL,
    content TEXT,
    author_id UUID REFERENCES auth.users(id),
    published BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Enable Row Level Security
ALTER TABLE posts ENABLE ROW LEVEL SECURITY;

-- Create RLS policies
CREATE POLICY "Public posts are viewable by everyone"
    ON posts FOR SELECT
    USING (published = true);

CREATE POLICY "Users can manage their own posts"
    ON posts FOR ALL
    USING (author_id = current_user_id());
```

## Architecture

### Project Structure

```
fluxbase/
├── cmd/fluxbase/          # Main application entry point
├── internal/
│   ├── api/               # HTTP server and REST handlers
│   ├── auth/              # Authentication service
│   ├── config/            # Configuration management
│   ├── database/          # Database connection and introspection
│   ├── realtime/          # WebSocket and realtime features
│   ├── storage/           # File storage service
│   ├── functions/         # Edge functions runtime
│   └── middleware/        # HTTP middlewares
├── pkg/
│   ├── client/            # Go client SDK
│   └── types/             # Shared types
├── migrations/            # Database migrations
└── config/                # Configuration files
```

### Technology Stack

- **Language**: Go 1.22+
- **Web Framework**: Fiber v2 (FastHTTP-based)
- **Database**: PostgreSQL with pgx/v5 driver
- **Migrations**: golang-migrate
- **Configuration**: Viper
- **Logging**: Zerolog
- **Authentication**: JWT (golang-jwt)
- **WebSockets**: Gorilla WebSocket

## Example Applications

Fluxbase includes **3 complete, production-ready example applications** to help you get started:

| Example                                    | Tech Stack           | Features           | Difficulty   |
| ------------------------------------------ | -------------------- | ------------------ | ------------ |
| [Todo App](./examples/todo-app/)           | React + TypeScript   | CRUD, RLS, Auth    | Beginner     |
| [Blog Platform](./examples/blog-platform/) | Next.js + TypeScript | SSR, Auth, Storage | Intermediate |
| [Chat Application](./examples/chat-app/)   | React + TypeScript   | Realtime, Presence | Intermediate |

Each example includes:

- ✅ Complete source code
- ✅ Setup instructions
- ✅ Deployment guides
- ✅ Best practices

**Quick start**:

```bash
cd examples/todo-app
npm install
cp .env.example .env.local
# Edit .env.local with your Fluxbase URL
npm run dev
```

See [examples/README.md](./examples/README.md) for detailed information.

## Development

### Documentation Server

View and edit documentation locally with Docusaurus:

```bash
# Start documentation server
make docs-server

# Or use the shorthand
make docs-dev

# Stop the server
make docs-stop
```

Documentation will be available at http://localhost:3000

### Running Tests

```bash
# All tests
make test

# Unit tests only
make test-unit

# Integration tests (requires MailHog)
make test-integration

# Email tests with MailHog
make test-email

# E2E tests
make test-e2e
```

### Building for Production

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o fluxbase cmd/fluxbase/main.go

# macOS
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o fluxbase cmd/fluxbase/main.go
```

### Docker Deployment

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -ldflags="-s -w" -o fluxbase cmd/fluxbase/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/fluxbase .
COPY --from=builder /app/config ./config
EXPOSE 8080
CMD ["./fluxbase"]
```

## Roadmap

### Upcoming Features

- [ ] TypeScript SDK (PostgREST-compatible)
- [ ] Go SDK improvements
- [ ] SDK documentation
- [ ] Vector/AI support (pgvector integration)
- [ ] Enhanced monitoring and observability
- [ ] Performance benchmarks and optimization

## Contributing

Contributions are welcome! Please read our contributing guidelines and submit pull requests to our repository.

## License

MIT License - see LICENSE file for details.

## Comparison with Supabase

| Feature        | Fluxbase                | Supabase                       |
| -------------- | ----------------------- | ------------------------------ |
| Deployment     | Single binary           | Multiple services              |
| Dependencies   | PostgreSQL only         | PostgreSQL + multiple services |
| Size           | ~80MB                   | 2+ GB                          |
| REST API       | ✅ PostgREST-compatible | ✅ PostgREST                   |
| Authentication | ✅ Built-in             | ✅ GoTrue                      |
| Chatbots       | ✅ Built-in             | ❌                             |
| Realtime       | ✅ Built-in             | ✅ Realtime                    |
| Storage        | ✅ Built-in             | ✅ Storage API                 |
| Edge Functions | ✅ Deno                 | ✅ Deno                        |
| Vector/AI      | ✅                      | ✅                             |
| Admin UI       | ✅ Built-in             | ✅                             |

## Documentation

### Getting Started

- **[GETTING_STARTED.md](GETTING_STARTED.md)** - Complete setup guide with DevContainer and local options
- **[examples/](examples/)** - 3 production-ready example applications (Todo, Blog, Chat)

### Production

- **[PRODUCTION_RUNBOOK.md](PRODUCTION_RUNBOOK.md)** - Production deployment, configuration, monitoring, and operations
- **[VERSIONING.md](VERSIONING.md)** - Version management, build automation, and release process

### GitHub Setup

- **[.github/SETUP_GUIDE.md](.github/SETUP_GUIDE.md)** - Complete GitHub repository configuration
- **[.github/SECRETS.md](.github/SECRETS.md)** - GitHub secrets and variables reference
- **[.github/QUICK_REFERENCE.md](.github/QUICK_REFERENCE.md)** - Quick reference card for GitHub setup

### Monitoring

- **[deploy/MONITORING.md](deploy/MONITORING.md)** - Prometheus, Grafana, and observability setup

### Additional Resources

- **[docs/](docs/)** - Full Docusaurus documentation (run `make docs` to serve locally)
- **[.docs/archive/](.docs/archive/)** - Historical project tracking documents

## Support

For issues, questions, and discussions:

- GitHub Issues: [github.com/fluxbase-eu/fluxbase/issues](https://github.com/fluxbase-eu/fluxbase/issues)
- Documentation: [docs.fluxbase.eu](https://docs.fluxbase.eu)
