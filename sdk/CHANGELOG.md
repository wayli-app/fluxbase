# Changelog

All notable changes to @fluxbase/sdk will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0] - 2025-10-26 (Evening)

### Added

- **Aggregation Methods** - Full support for database aggregations
  - `.count(column)` - Count rows or specific column values
  - `.sum(column)` - Sum numeric values
  - `.avg(column)` - Calculate average
  - `.min(column)` - Find minimum value
  - `.max(column)` - Find maximum value
  - `.groupBy(columns)` - Group results by one or more columns
  - Full integration with filters and ordering
- **Batch Operation Aliases** - Convenience methods for clarity
  - `.insertMany(rows)` - Batch insert multiple rows
  - `.updateMany(data)` - Batch update matching rows
  - `.deleteMany()` - Batch delete matching rows
- **Enhanced Documentation**
  - Added comprehensive aggregation examples
  - Added batch operation examples
  - Updated Quick Start guide
  - Added E2E test examples

### Changed

- Query builder now supports `group_by` parameter for aggregations
- Improved README with new features prominently displayed

## [0.1.0] - 2025-10-26

### Added

- Initial release of Fluxbase TypeScript/JavaScript SDK
- **Authentication Module**
  - Sign up, sign in, sign out
  - JWT-based authentication with automatic token refresh
  - Session persistence to localStorage
  - User profile management (get current user, update user)
  - OAuth2 provider support
  - Magic link authentication
- **Database Query Builder**
  - PostgREST-compatible query syntax
  - Comprehensive filtering operators (eq, neq, gt, gte, lt, lte, like, ilike, is, in, contains, textSearch)
  - Ordering with nulls handling
  - Pagination (limit, offset, range)
  - Single row queries
  - Insert, upsert, update, delete operations
  - Batch operations support
- **Realtime Subscriptions**
  - WebSocket-based subscriptions to database changes
  - Support for INSERT, UPDATE, DELETE events
  - Automatic reconnection with exponential backoff
  - Heartbeat/ping-pong mechanism
  - Channel-based routing (table:schema.table_name)
- **Storage Client**
  - File upload with metadata support
  - File download with streaming
  - List files with prefix filtering and pagination
  - Delete files
  - Signed URL generation for temporary access
  - Public URL generation
  - Copy and move operations
  - Multi-bucket support
- **RPC (Remote Procedure Calls)**
  - Call PostgreSQL functions directly
  - Named and positional parameter support
- **TypeScript Support**
  - Full type safety with generics
  - Auto-completion for all methods
  - Type inference for query results
- **Error Handling**
  - Consistent error response format
  - Detailed error messages
- **Cross-platform**
  - Browser support (ES6+, Fetch API, WebSockets)
  - Node.js support (18+)
  - SSR compatible

### Dependencies

- `cross-fetch`: ^4.0.0 (for Node.js compatibility)

[0.1.0]: https://github.com/fluxbase-eu/fluxbase/releases/tag/sdk-v0.1.0
