# Fluxbase

[![CI](https://github.com/nimbleflux/fluxbase/actions/workflows/ci.yml/badge.svg)](https://github.com/nimbleflux/fluxbase/actions/workflows/ci.yml)

> **Beta Software**: Fluxbase is currently in beta. While we're working hard to stabilize the API and features, you may encounter breaking changes between versions. We welcome feedback and contributions!

A lightweight, single-binary Backend-as-a-Service (BaaS) alternative to Supabase. Fluxbase provides essential backend services including auto-generated REST APIs, authentication, realtime subscriptions, file storage, and edge functions - all in a single Go binary with PostgreSQL as the only dependency.

## Features

### Core Services

- **PostgREST-compatible REST API**: Auto-generates CRUD endpoints from your PostgreSQL schema
- **GraphQL API**: Full GraphQL support with configurable depth/complexity limits
- **Authentication**: Email/password, magic links, OAuth2 (Google, GitHub, Microsoft, etc.), OIDC, SAML SSO, MFA/TOTP
- **Realtime Subscriptions**: WebSocket-based live data updates using PostgreSQL LISTEN/NOTIFY
- **Storage**: File upload/download with access policies (local filesystem or S3), image transformations
- **Edge Functions**: JavaScript/TypeScript function execution with Deno runtime
- **Background Jobs**: Long-running tasks with progress tracking, retry logic, cron scheduling
- **RPC/Procedures**: SQL-based serverless procedures with scheduling and RBAC
- **Webhooks**: Event-driven webhook delivery for database changes with retries and HMAC signing
- **Vector Search**: pgvector-powered semantic search with automatic embeddings
- **AI/RAG**: Knowledge bases, chatbots, and entity extraction
- **MCP Server**: Model Context Protocol for AI assistant integration
- **Multi-Tenancy**: Tenant isolation with RLS, separate databases, and per-tenant config
- **Database Branching**: Isolated dev/test environments with GitHub PR integration

### Key Highlights

- Single binary or container deployment
- PostgreSQL as the only external dependency
- Automatic REST endpoint generation
- Row Level Security (RLS) support
- TypeScript SDK + React hooks
- Built-in observability (Prometheus metrics, OpenTelemetry tracing)
- Horizontal scaling with leader election

## Quick Start

### Prerequisites

- [Go](https://go.dev/) 1.25+
- [PostgreSQL](https://www.postgresql.org/) 16+ (with [pgvector](https://github.com/pgvector/pgvector))
- [Node.js](https://nodejs.org/) 20+ / [Bun](https://bun.sh/) (for admin UI and SDKs)
- [Deno](https://deno.land/) (for edge functions runtime)

### Build & Run

```bash
# Clone the repository
git clone https://github.com/nimbleflux/fluxbase.git
cd fluxbase

# Build the binary
make build

# Or run in development mode (backend + admin UI)
make dev
```

For Docker-based deployment, see the [deployment guide](https://fluxbase.eu/deployment/overview/).

### SDKs

```bash
# TypeScript SDK
cd sdk && bun install && bun run build

# React SDK
cd sdk-react && bun install && bun run build
```

For more information, see the [quick start guide](https://fluxbase.eu/getting-started/quick-start/).

## Development

```bash
make setup-dev    # Install dependencies + git hooks
make test         # Unit tests
make test-full    # All tests including E2E
make lint-go      # Go linting
make cli-install  # Build and install CLI
```

See [CLAUDE.md](CLAUDE.md) for a detailed codebase guide and architecture overview.

## Support

- GitHub Issues: [github.com/nimbleflux/fluxbase/issues](https://github.com/nimbleflux/fluxbase/issues)
- Documentation: [fluxbase.eu](https://fluxbase.eu)
- Discord: [discord.gg/BXPRHkQzkA](https://discord.gg/BXPRHkQzkA)

## License

This project is licensed under the terms described in [docs/src/content/docs/intro.md](docs/src/content/docs/intro.md).
