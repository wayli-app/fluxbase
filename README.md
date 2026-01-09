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

- Single binary deployment (~50MB)
- PostgreSQL as the only external dependency
- Automatic REST endpoint generation
- Row Level Security (RLS) support
- TypeScript SDK

## Quick Start

For more information about Fluxbase, look into [the docs](https://fluxbase.eu/getting-started/quick-start/).

## Support

For issues, questions, and discussions:

- GitHub Issues: [github.com/fluxbase-eu/fluxbase/issues](https://github.com/fluxbase-eu/fluxbase/issues)
- Documentation: [docs.fluxbase.eu](https://docs.fluxbase.eu)
