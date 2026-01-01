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
- TypeScript SDK

## Quick Start

> 📖 **See [GETTING_STARTED.md](GETTING_STARTED.md) for complete setup instructions**

### Try it Now: Docker Compose (2 minutes)

Get Fluxbase running instantly with Docker:

```bash
# Clone the repository
git clone https://github.com/fluxbase-eu/fluxbase.git
cd fluxbase/deploy

# Start all services (PostgreSQL + Fluxbase + MinIO)
docker compose up -d

# Check health
curl http://localhost:8080/health

# Access admin dashboard
open http://localhost:8080
```

That's it! Fluxbase is now running at http://localhost:8080

**Default credentials:**

- Database: `postgres:postgres`

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
