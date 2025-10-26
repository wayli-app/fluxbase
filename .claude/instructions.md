# Claude Development Instructions for Fluxbase

## Project Overview

You are working on **Fluxbase**, a lightweight Backend-as-a-Service (BaaS) alternative to Supabase. The project provides a single Go binary that includes:
- Auto-generated REST APIs from PostgreSQL schemas
- Authentication system
- Realtime subscriptions via WebSockets
- File storage service
- Edge functions runtime
- Admin UI

## Current State

### ✅ Completed Components
- Core REST API engine with PostgREST compatibility
- PostgreSQL connection pooling and schema introspection
- Dynamic endpoint generation from database tables
- Query parser for filtering, ordering, and pagination
- CI/CD pipeline with GitHub Actions
- Documentation site with Docusaurus
- DevContainer setup for development
- Comprehensive test framework

### 🚧 Work in Progress
Check `TODO.md` for the current task list and priorities.

## Development Guidelines

### Code Style
1. **Go Code**:
   - Follow standard Go idioms and conventions
   - Use meaningful variable and function names
   - Keep functions small and focused
   - Add comments for exported functions
   - Use structured logging with zerolog
   - Handle errors explicitly

2. **File Organization**:
   - `/cmd/fluxbase/` - Main application entry point
   - `/internal/` - Private application code
   - `/pkg/` - Public libraries that can be imported
   - `/test/` - Integration and load tests
   - `/docs/` - Documentation site
   - `/migrations/` - Database migrations

3. **Testing**:
   - Write unit tests for new functions
   - Add integration tests for API endpoints
   - Use table-driven tests where appropriate
   - Mock external dependencies
   - Aim for 80% code coverage

### Architecture Principles

1. **Single Binary**: Everything must compile into one executable
2. **PostgreSQL Only**: Don't add dependencies on other databases
3. **Modular Design**: Keep components loosely coupled
4. **Performance First**: Optimize for speed and low memory usage
5. **Developer Experience**: APIs should be intuitive and well-documented

### Working with the Codebase

1. **Before Making Changes**:
   - Read `TODO.md` to understand current priorities
   - Check existing code patterns in similar files
   - Review test coverage for the area you're modifying

2. **When Adding Features**:
   - Update TODO.md with progress
   - Add configuration options to `internal/config/`
   - Create database migrations if needed
   - Add REST endpoints to `internal/api/`
   - Write tests for new functionality
   - Update documentation

3. **Database Migrations**:
   - Create new migration files in `internal/database/migrations/`
   - Use format: `XXX_description.up.sql` and `XXX_description.down.sql`
   - Test both up and down migrations
   - Keep migrations idempotent when possible

4. **API Development**:
   - Follow PostgREST conventions for query parameters
   - Return appropriate HTTP status codes
   - Include helpful error messages
   - Support JSON request/response bodies
   - Add OpenAPI documentation comments

### Environment Setup

1. **Local Development**:
   ```bash
   # Copy environment file
   cp .env.example .env

   # Install development tools
   make setup-dev

   # Run with hot-reload
   make dev
   ```

2. **Testing**:
   ```bash
   # Run all tests
   make test

   # Run specific test types
   make test-unit
   make test-integration
   make test-load
   ```

3. **Using DevContainer**:
   - Open project in VS Code
   - Reopen in Container when prompted
   - All tools are pre-installed

### Common Tasks

1. **Adding a New Service**:
   - Create service interface in `/internal/servicename/`
   - Implement service logic
   - Add service to Server struct in `/internal/api/server.go`
   - Create routes and handlers
   - Write unit and integration tests

2. **Adding Configuration Options**:
   - Add fields to config struct in `/internal/config/config.go`
   - Set defaults in `setDefaults()`
   - Add validation in `Validate()`
   - Document in `.env.example`
   - Update documentation

3. **Creating API Endpoints**:
   - Add handler function to appropriate service
   - Register route in `setupRoutes()`
   - Implement request validation
   - Add error handling
   - Write integration tests

### Error Handling

1. Always check and handle errors explicitly
2. Use structured errors with context
3. Return appropriate HTTP status codes
4. Log errors with appropriate severity
5. Don't expose internal errors to clients

### Security Considerations

1. Validate all user input
2. Use prepared statements for SQL queries
3. Implement rate limiting for sensitive endpoints
4. Hash passwords with bcrypt
5. Use secure random tokens
6. Follow OWASP best practices

### Performance Guidelines

1. Use connection pooling efficiently
2. Implement caching where appropriate
3. Paginate large result sets
4. Use database indexes effectively
5. Profile code to identify bottlenecks
6. Minimize allocations in hot paths

### Git Workflow

1. Work on feature branches
2. Keep commits focused and atomic
3. Write clear commit messages
4. Update TODO.md with progress
5. Ensure tests pass before committing
6. Run `make fmt` before committing

## Important Files

- `TODO.md` - Current task list and progress
- `.env.example` - Environment variable documentation
- `Makefile` - Common development commands
- `go.mod` - Go dependencies
- `docker-compose.yml` - Local development services
- `.github/workflows/` - CI/CD pipelines

## Debugging Tips

1. Enable debug mode: `FLUXBASE_DEBUG=true`
2. Check logs for detailed error messages
3. Use `make docker-dev-logs` to see container logs
4. Connect to PostgreSQL: `psql -h localhost -U postgres -d fluxbase`
5. Use pgAdmin at http://localhost:5050

## Resources

- [PostgREST Documentation](https://postgrest.org/)
- [Fiber Framework](https://gofiber.io/)
- [pgx PostgreSQL Driver](https://github.com/jackc/pgx)
- [JWT Authentication](https://jwt.io/)
- [WebSocket Protocol](https://developer.mozilla.org/en-US/docs/Web/API/WebSockets_API)

## Questions to Ask Yourself

Before implementing a feature:
1. Does this maintain backward compatibility?
2. Will this work in a single binary?
3. Is this performant at scale?
4. Is the API intuitive for developers?
5. Have I added appropriate tests?
6. Is this documented clearly?

## Need Help?

1. Check existing code for patterns
2. Review test files for examples
3. Look at TODO.md for context
4. Check documentation in `/docs/`
5. Review similar projects like Supabase/PostgREST

Remember: The goal is to create a **simple, fast, and developer-friendly** alternative to Supabase that can be deployed as a single binary.