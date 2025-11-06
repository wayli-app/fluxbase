# Fluxbase Project Context

## Quick Summary

**Fluxbase** is a lightweight, single-binary Backend-as-a-Service (BaaS) that provides REST APIs, authentication, realtime subscriptions, file storage, and edge functions - all with PostgreSQL as the only dependency. Think of it as a simpler, self-hostable alternative to Supabase.

## Key Design Decisions

1. **Single Binary Architecture**: Everything compiles into one ~80MB Go executable for easy deployment
2. **PostgreSQL Only**: No Redis, no RabbitMQ, just PostgreSQL for all persistence
3. **PostgREST Compatible**: Maintains API compatibility with Supabase where practical
4. **Schema-Driven**: Database tables automatically become REST endpoints
5. **Performance Focused**: Written in Go, targets <1ms response times, 1000+ concurrent connections

## Project Structure

```
fluxbase/
├── cmd/fluxbase/          # Main entry point
├── internal/              # Private application code
│   ├── api/              # HTTP server & REST handlers
│   ├── auth/             # Authentication (NOT YET IMPLEMENTED)
│   ├── config/           # Configuration management
│   ├── database/         # DB connection & introspection
│   ├── realtime/         # WebSocket server (NOT YET IMPLEMENTED)
│   ├── storage/          # File storage (NOT YET IMPLEMENTED)
│   └── functions/        # Edge functions (NOT YET IMPLEMENTED)
├── pkg/                  # Public Go packages
├── docs/                 # Docusaurus documentation site
├── test/                 # Integration & load tests
└── migrations/           # Database migrations
```

## Current Implementation Status

### ✅ What's Working

- **REST API Generator**: Automatically creates CRUD endpoints from PostgreSQL tables
- **Query Parser**: Full PostgREST-compatible filtering, ordering, pagination
- **Schema Introspection**: Discovers tables, columns, relationships
- **Configuration**: YAML and environment variable support
- **Database Layer**: Connection pooling, migrations, health checks
- **CI/CD**: GitHub Actions, semantic versioning, Docker builds
- **Testing**: Unit, integration, and load testing frameworks
- **Documentation**: Docusaurus site with guides and API docs
- **DevContainer**: Full development environment with all tools

### 🚧 What's Not Yet Implemented

- **Authentication**: JWT, user management, magic links
- **Realtime**: WebSocket server, subscriptions, presence
- **Storage**: File upload/download, buckets, S3 integration
- **Edge Functions**: JavaScript runtime, HTTP/cron triggers
- **Client SDKs**: TypeScript and Go libraries
- **Admin UI**: Web-based database management interface

## Technology Stack

- **Language**: Go 1.22+
- **Web Framework**: Fiber v2 (FastHTTP-based)
- **Database**: PostgreSQL 14+ with pgx/v5
- **Migrations**: golang-migrate
- **Config**: Viper
- **Logging**: Zerolog
- **Testing**: testify, k6
- **Docs**: Docusaurus 3
- **CI/CD**: GitHub Actions

## API Examples

The REST API is auto-generated from your database schema:

```bash
# Create a record
POST /api/rest/posts
{"title": "Hello", "content": "World"}

# Query with filters
GET /api/rest/posts?published=eq.true&order=created_at.desc&limit=10

# Update a record
PATCH /api/rest/posts/123
{"published": true}

# Delete a record
DELETE /api/rest/posts/123
```

## Development Commands

```bash
make dev           # Run with hot-reload
make test          # Run all tests
make build         # Build binary
make docker-dev    # Start full dev environment
make docs-dev      # Start documentation server
```

## Environment Variables

All configuration can be set via environment variables with the `FLUXBASE_` prefix:

```bash
FLUXBASE_DATABASE_HOST=localhost
FLUXBASE_DATABASE_PORT=5432
FLUXBASE_AUTH_JWT_SECRET=secret
FLUXBASE_STORAGE_PROVIDER=s3
```

## Design Philosophy

1. **Simplicity > Features**: Better to do fewer things well
2. **Developer Experience**: APIs should be intuitive and predictable
3. **Performance**: Every millisecond counts
4. **Self-Contained**: Minimize external dependencies
5. **Production-Ready**: Built for real-world use, not just demos

## Common Pitfalls to Avoid

1. Don't add new external dependencies without strong justification
2. Don't break PostgREST API compatibility unnecessarily
3. Don't implement features that can't work in a single binary
4. Don't sacrifice performance for convenience
5. Don't forget to update tests and documentation

## Next Priority

According to TODO.md, the next major milestone is implementing the **Authentication System**:

1. JWT token generation and validation
2. User registration and login endpoints
3. Password hashing with bcrypt
4. Session management
5. Magic link authentication

## Quick Context for Claude

When you (Claude) work on this project:

1. Check `TODO.md` for current tasks
2. Follow patterns in existing code
3. Maintain PostgREST compatibility
4. Write tests for new features
5. Keep the single-binary philosophy
6. Update documentation as you go

The goal is to build a **production-ready**, **developer-friendly**, **high-performance** backend that can replace Supabase for most use cases while being **100x easier to deploy and operate**.
