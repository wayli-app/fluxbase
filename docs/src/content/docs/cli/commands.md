---
title: CLI Command Reference
description: Complete reference for all Fluxbase CLI commands
---

This page documents all Fluxbase CLI commands, their subcommands, flags, and usage examples.

## Command Overview

```
fluxbase [command] [subcommand] [flags]
```

### Global Flags

These flags work with all commands:

| Flag           | Short | Description                                           |
| -------------- | ----- | ----------------------------------------------------- |
| `--config`     |       | Config file path (default: `~/.fluxbase/config.yaml`) |
| `--profile`    | `-p`  | Profile to use                                        |
| `--output`     | `-o`  | Output format: `table`, `json`, `yaml`                |
| `--no-headers` |       | Hide table headers                                    |
| `--quiet`      | `-q`  | Minimal output                                        |
| `--debug`      |       | Enable debug output                                   |

---

## Authentication Commands

### `fluxbase auth login`

Authenticate with a Fluxbase server.

```bash
# Interactive login
fluxbase auth login

# Non-interactive with credentials
fluxbase auth login --server URL --email EMAIL --password PASSWORD

# With API token
fluxbase auth login --server URL --token TOKEN

# Save to named profile
fluxbase auth login --profile prod --server URL
```

**Flags:**

- `--server` - Fluxbase server URL
- `--email` - Email address
- `--password` - Password
- `--token` - API token (alternative to email/password)
- `--profile` - Profile name (default: "default")
- `--use-keychain` - Store credentials in system keychain

### `fluxbase auth logout`

Clear stored credentials.

```bash
fluxbase auth logout
fluxbase auth logout --profile prod
```

### `fluxbase auth status`

Show authentication status for all profiles.

```bash
fluxbase auth status
```

### `fluxbase auth switch`

Switch the active profile.

```bash
fluxbase auth switch prod
```

### `fluxbase auth whoami`

Display current user information.

```bash
fluxbase auth whoami
```

---

## Functions Commands

Manage edge functions.

### `fluxbase functions list`

```bash
fluxbase functions list
fluxbase functions list --namespace production
```

### `fluxbase functions get`

```bash
fluxbase functions get my-function
```

### `fluxbase functions create`

```bash
fluxbase functions create my-function --code ./function.ts
fluxbase functions create my-function --code ./function.ts --timeout 60 --memory 256
```

**Flags:**

- `--code` - Path to function code file (required)
- `--description` - Function description
- `--timeout` - Execution timeout in seconds (default: 30)
- `--memory` - Memory limit in MB (default: 128)

### `fluxbase functions update`

```bash
fluxbase functions update my-function --code ./function.ts
fluxbase functions update my-function --timeout 120
```

### `fluxbase functions delete`

```bash
fluxbase functions delete my-function
```

### `fluxbase functions invoke`

```bash
fluxbase functions invoke my-function
fluxbase functions invoke my-function --data '{"key": "value"}'
fluxbase functions invoke my-function --file ./payload.json
```

### `fluxbase functions logs`

View execution logs for a function.

```bash
fluxbase functions logs my-function
fluxbase functions logs my-function --tail 50
fluxbase functions logs my-function --follow
```

**Flags:**

- `--tail` - Number of lines to show (default: 20)
- `--follow`, `-f` - Stream new log entries in real-time

### `fluxbase functions sync`

Sync all functions from a local directory to the server.

```bash
fluxbase functions sync --dir ./functions
fluxbase functions sync --dir ./functions --namespace production --dry-run
```

**Flags:**

- `--dir` - Directory containing function files (default: `./functions`)
- `--namespace` - Target namespace (default: `default`)
- `--dry-run` - Preview changes without applying

**Shared Modules:**

Place shared code in a `_shared/` subdirectory:

```
functions/
├── _shared/
│   └── utils.ts
├── api-handler.ts
└── webhook.ts
```

Functions can import from shared modules:

```typescript
import { helper } from "./_shared/utils.ts";
```

If Deno is installed locally, functions with imports are automatically bundled before upload.

---

## Jobs Commands

Manage background jobs.

### `fluxbase jobs list`

```bash
fluxbase jobs list
```

### `fluxbase jobs submit`

```bash
fluxbase jobs submit my-job
fluxbase jobs submit my-job --payload '{"data": "value"}'
fluxbase jobs submit my-job --priority 10
```

### `fluxbase jobs status`

```bash
fluxbase jobs status abc123
```

### `fluxbase jobs cancel`

```bash
fluxbase jobs cancel abc123
```

### `fluxbase jobs retry`

```bash
fluxbase jobs retry abc123
```

### `fluxbase jobs logs`

```bash
fluxbase jobs logs abc123
```

### `fluxbase jobs stats`

Show job queue statistics.

```bash
fluxbase jobs stats
```

### `fluxbase jobs sync`

Sync job functions from a local directory.

```bash
fluxbase jobs sync --dir ./jobs
fluxbase jobs sync --dir ./jobs --namespace production --dry-run
```

**Flags:**

- `--dir` - Directory containing job files (default: `./jobs`)
- `--namespace` - Target namespace (default: `default`)
- `--dry-run` - Preview changes without applying

Like functions, jobs support a `_shared/` directory for shared modules and JSON/GeoJSON data files.

---

## Storage Commands

Manage file storage.

### Bucket Commands

```bash
# List buckets
fluxbase storage buckets list

# Create bucket
fluxbase storage buckets create my-bucket
fluxbase storage buckets create my-bucket --public

# Delete bucket
fluxbase storage buckets delete my-bucket
```

### Object Commands

```bash
# List objects
fluxbase storage objects list my-bucket
fluxbase storage objects list my-bucket --prefix images/

# Upload file
fluxbase storage objects upload my-bucket path/to/file.jpg ./local-file.jpg

# Download file
fluxbase storage objects download my-bucket path/to/file.jpg ./local-file.jpg

# Delete object
fluxbase storage objects delete my-bucket path/to/file.jpg

# Get signed URL
fluxbase storage objects url my-bucket path/to/file.jpg --expires 7200
```

---

## Chatbot Commands

Manage AI chatbots.

```bash
# List chatbots
fluxbase chatbots list

# Get chatbot details
fluxbase chatbots get abc123

# Create chatbot
fluxbase chatbots create support-bot --system-prompt "You are helpful"

# Update chatbot
fluxbase chatbots update abc123 --model gpt-4

# Delete chatbot
fluxbase chatbots delete abc123

# Interactive chat
fluxbase chatbots chat abc123
```

---

## Knowledge Base Commands

Manage knowledge bases for RAG.

```bash
# List knowledge bases
fluxbase kb list

# Create knowledge base
fluxbase kb create docs --description "Product documentation"

# Upload document
fluxbase kb upload abc123 ./manual.pdf

# List documents
fluxbase kb documents abc123

# Search knowledge base
fluxbase kb search abc123 "how to reset password"

# Delete knowledge base
fluxbase kb delete abc123
```

---

## Table Commands

Query and manage database tables.

```bash
# List tables
fluxbase tables list

# Describe table
fluxbase tables describe users

# Query table
fluxbase tables query users
fluxbase tables query users --select "id,email" --where "role=eq.admin" --limit 10

# Insert record
fluxbase tables insert users --data '{"email": "user@example.com"}'

# Update records
fluxbase tables update users --where "id=eq.123" --data '{"name": "New Name"}'

# Delete records
fluxbase tables delete users --where "id=eq.123"
```

---

## GraphQL Commands

Execute GraphQL queries and mutations against the auto-generated GraphQL API.

### `fluxbase graphql query`

Execute a GraphQL query.

```bash
# Simple query
fluxbase graphql query '{ users { id email created_at } }'

# Query with filtering
fluxbase graphql query '{ users(where: {role: {_eq: "admin"}}) { id email } }'

# Query with ordering and pagination
fluxbase graphql query '{ users(limit: 10, order_by: {created_at: desc}) { id email } }'

# Query from file
fluxbase graphql query --file ./get-users.graphql

# Query with variables
fluxbase graphql query 'query GetUser($id: ID!) { user(id: $id) { id email } }' --var 'id=abc-123'

# Multiple variables
fluxbase graphql query 'query($limit: Int, $offset: Int) { users(limit: $limit, offset: $offset) { id } }' \
  --var 'limit=10' --var 'offset=20'

# Output as JSON
fluxbase graphql query '{ users { id } }' -o json
```

**Flags:**

- `--file`, `-f` - File containing the GraphQL query
- `--var` - Variables in format `name=value` (can be repeated)
- `--pretty` - Pretty print JSON output (default: true)

### `fluxbase graphql mutation`

Execute a GraphQL mutation.

```bash
# Insert a record
fluxbase graphql mutation 'mutation {
  insert_users(objects: [{email: "new@example.com", name: "New User"}]) {
    returning { id email }
  }
}'

# Update records
fluxbase graphql mutation 'mutation {
  update_users(where: {id: {_eq: "user-id"}}, _set: {name: "Updated Name"}) {
    affected_rows
    returning { id name }
  }
}'

# Delete records
fluxbase graphql mutation 'mutation {
  delete_users(where: {id: {_eq: "user-id"}}) {
    affected_rows
  }
}'

# Mutation with variables
fluxbase graphql mutation 'mutation CreateUser($email: String!, $name: String!) {
  insert_users(objects: [{email: $email, name: $name}]) {
    returning { id }
  }
}' --var 'email=test@example.com' --var 'name=Test User'

# Mutation from file
fluxbase graphql mutation --file ./create-user.graphql --var 'email=user@example.com'
```

**Flags:**

- `--file`, `-f` - File containing the GraphQL mutation
- `--var` - Variables in format `name=value` (can be repeated)
- `--pretty` - Pretty print JSON output (default: true)

### `fluxbase graphql introspect`

Fetch and display the GraphQL schema via introspection.

```bash
# Full introspection query
fluxbase graphql introspect

# List only type names
fluxbase graphql introspect --types

# Output as JSON
fluxbase graphql introspect -o json
```

**Flags:**

- `--types` - List only type names (simplified output)

**Note:** Introspection must be enabled on the server. It's enabled by default in development but should be disabled in production for security.

---

## RPC Commands

Manage and invoke stored procedures.

### `fluxbase rpc list`

List all RPC procedures.

```bash
fluxbase rpc list
fluxbase rpc list --namespace production
```

### `fluxbase rpc get`

Get details of a specific procedure.

```bash
fluxbase rpc get default/calculate_totals
```

### `fluxbase rpc invoke`

Invoke a stored procedure.

```bash
fluxbase rpc invoke default/calculate_totals
fluxbase rpc invoke default/process --params '{"id": 123}'
fluxbase rpc invoke default/batch_update --file ./params.json --async
```

**Flags:**

- `--params` - JSON parameters to pass
- `--file` - Load parameters from file
- `--async` - Run asynchronously (returns immediately)

### `fluxbase rpc sync`

Sync RPC procedures from SQL files in a directory.

```bash
fluxbase rpc sync --dir ./rpc
fluxbase rpc sync --dir ./rpc --namespace production --dry-run
```

**Flags:**

- `--dir` - Directory containing `.sql` files (default: `./rpc`)
- `--namespace` - Target namespace (default: `default`)
- `--dry-run` - Preview changes without applying
- `--delete-missing` - Delete procedures not in local directory

---

## Webhook Commands

Manage webhooks.

```bash
# List webhooks
fluxbase webhooks list

# Create webhook
fluxbase webhooks create --url https://example.com/webhook --events "INSERT,UPDATE"

# Test webhook
fluxbase webhooks test abc123

# View deliveries
fluxbase webhooks deliveries abc123

# Delete webhook
fluxbase webhooks delete abc123
```

---

## API Key Commands

Manage API keys.

```bash
# List API keys
fluxbase apikeys list

# Create API key
fluxbase apikeys create --name "Production" --scopes "read:tables,write:tables"

# Revoke API key
fluxbase apikeys revoke abc123

# Delete API key
fluxbase apikeys delete abc123
```

---

## Migration Commands

Manage database migrations.

```bash
# List migrations
fluxbase migrations list

# Apply specific migration
fluxbase migrations apply 001_create_users

# Rollback migration
fluxbase migrations rollback 001_create_users

# Apply all pending
fluxbase migrations apply-pending

# Sync from directory
fluxbase migrations sync --dir ./migrations
```

---

## Extension Commands

Manage PostgreSQL extensions.

```bash
# List extensions
fluxbase extensions list

# Enable extension
fluxbase extensions enable pgvector

# Disable extension
fluxbase extensions disable pgvector
```

---

## Realtime Commands

Manage realtime connections.

```bash
# Show stats
fluxbase realtime stats

# Broadcast message
fluxbase realtime broadcast my-channel --message '{"type": "notification"}'
```

---

## Settings Commands

Manage system settings.

```bash
# List settings
fluxbase settings list

# Get setting
fluxbase settings get auth.signup_enabled

# Set setting
fluxbase settings set auth.signup_enabled true
```

---

## Settings Secrets Commands

Manage encrypted application settings secrets. These are separate from the function secrets (`fluxbase secrets`) and are used for storing sensitive application configuration such as API keys and credentials.

Settings secrets support two scopes:

- **System secrets** - Global application secrets (admin only)
- **User secrets** - Per-user secrets encrypted with user-specific keys

### `fluxbase settings secrets list`

List all secrets (values are never shown).

```bash
# List system secrets (admin)
fluxbase settings secrets list

# List user's own secrets
fluxbase settings secrets list --user
```

**Flags:**

- `--user` - List user-specific secrets instead of system secrets

### `fluxbase settings secrets set`

Create or update a secret.

```bash
# Set a system secret (admin only)
fluxbase settings secrets set stripe_api_key "sk-live-xxx"
fluxbase settings secrets set openai_key "sk-proj-xxx" --description "OpenAI API key"

# Set a user-specific secret
fluxbase settings secrets set my_api_key "user-key-xxx" --user
fluxbase settings secrets set my_api_key "user-key-xxx" --user --description "My personal API key"
```

**Flags:**

- `--user` - Create/update a user-specific secret instead of a system secret
- `--description` - Description of the secret

User secrets are encrypted with a user-derived key, ensuring that even admins cannot decrypt other users' secrets.

### `fluxbase settings secrets get`

Get metadata for a secret (the value is never returned).

```bash
# Get system secret metadata
fluxbase settings secrets get stripe_api_key

# Get user secret metadata
fluxbase settings secrets get my_api_key --user
```

**Flags:**

- `--user` - Get a user-specific secret instead of a system secret

### `fluxbase settings secrets delete`

Delete a secret permanently.

```bash
# Delete system secret
fluxbase settings secrets delete stripe_api_key

# Delete user secret
fluxbase settings secrets delete my_api_key --user
```

**Flags:**

- `--user` - Delete a user-specific secret instead of a system secret

### Comparison: Settings Secrets vs Legacy Secrets

| Feature             | `fluxbase settings secrets` (Recommended) | `fluxbase secrets` (Legacy)         |
| ------------------- | ----------------------------------------- | ----------------------------------- |
| Storage             | `app.settings` table                      | `functions.secrets` table           |
| Scopes              | System, user                              | Global, namespace                   |
| User-specific       | Yes (with HKDF encryption)                | No                                  |
| Version history     | No                                        | Yes                                 |
| Access in functions | `secrets.get()`, `secrets.getRequired()`  | `Deno.env.get("FLUXBASE_SECRET_*")` |
| Fallback            | User → System automatic fallback          | Namespace → Global                  |

---

## Config Commands

Manage CLI configuration.

```bash
# Initialize config
fluxbase config init

# View config
fluxbase config view

# Set config value
fluxbase config set defaults.output json

# List profiles
fluxbase config profiles

# Add profile
fluxbase config profiles add staging

# Remove profile
fluxbase config profiles remove staging
```

---

## Secrets Commands (Legacy)

:::note[Recommended: Use Settings Secrets]
For new projects, use `fluxbase settings secrets` instead. Settings secrets provide user-specific encryption and integrate with the `secrets` object in functions. See [Settings Secrets Commands](#settings-secrets-commands) above.
:::

The legacy `fluxbase secrets` commands manage namespace-scoped secrets stored in the `functions.secrets` table.

### `fluxbase secrets list`

List all secrets (values are never shown).

```bash
fluxbase secrets list
fluxbase secrets list --scope global
fluxbase secrets list --namespace my-namespace
```

**Flags:**

- `--scope` - Filter by scope (`global` or `namespace`)
- `--namespace` - Filter by namespace

### `fluxbase secrets set`

Create or update a secret.

```bash
fluxbase secrets set API_KEY "my-secret-key"
fluxbase secrets set DATABASE_URL "postgres://..." --scope namespace --namespace my-ns
fluxbase secrets set TEMP_KEY "value" --expires 30d
```

**Flags:**

- `--scope` - Secret scope: `global` (default) or `namespace`
- `--namespace` - Namespace for namespace-scoped secrets
- `--description` - Description of the secret
- `--expires` - Expiration duration (e.g., `30d`, `1y`, `24h`)

Legacy secrets are available in functions as `FLUXBASE_SECRET_<NAME>` environment variables via `Deno.env.get()`.

### `fluxbase secrets get`

Get metadata for a secret (the value is never returned).

```bash
fluxbase secrets get API_KEY
fluxbase secrets get DATABASE_URL --namespace my-namespace
```

### `fluxbase secrets delete`

Delete a secret permanently.

```bash
fluxbase secrets delete API_KEY
fluxbase secrets delete DATABASE_URL --namespace my-namespace
```

### `fluxbase secrets history`

Show version history for a secret.

```bash
fluxbase secrets history API_KEY
fluxbase secrets history DATABASE_URL --namespace my-namespace
```

### `fluxbase secrets rollback`

Rollback a secret to a previous version.

```bash
fluxbase secrets rollback API_KEY 2
fluxbase secrets rollback DATABASE_URL 1 --namespace my-namespace
```

---

## Logs Commands

Query and stream logs from the central logging system.

### `fluxbase logs list`

List logs with filters.

```bash
fluxbase logs list
fluxbase logs list --category system --level error
fluxbase logs list --since 1h --search "database"
fluxbase logs list --category execution --limit 50
fluxbase logs list --user-id abc123 -o json
```

**Flags:**

- `--category` - Filter by category: `system`, `http`, `security`, `execution`, `ai`, `custom`
- `--custom-category` - Filter by custom category name (requires `--category=custom`)
- `--level` - Filter by level: `debug`, `info`, `warn`, `error`
- `--component` - Filter by component name
- `--request-id` - Filter by request ID
- `--user-id` - Filter by user ID
- `--search` - Full-text search in message
- `--since` - Show logs since time (e.g., `1h`, `30m`, `2024-01-15T10:00:00Z`)
- `--until` - Show logs until time
- `--limit` - Maximum entries to return (default: 100)
- `--asc` - Sort ascending (oldest first)

### `fluxbase logs tail`

Tail logs in real-time.

```bash
fluxbase logs tail
fluxbase logs tail --category security
fluxbase logs tail --level error
fluxbase logs tail --category system --component auth
```

**Flags:**

- `--category` - Filter by category
- `--level` - Filter by level
- `--component` - Filter by component
- `--lines` - Number of initial lines to show (default: 20)

### `fluxbase logs stats`

Show log statistics.

```bash
fluxbase logs stats
fluxbase logs stats -o json
```

### `fluxbase logs execution`

View logs for a specific function, job, or RPC execution.

```bash
fluxbase logs execution abc123-def456
fluxbase logs execution abc123-def456 -o json
fluxbase logs execution abc123-def456 --follow
fluxbase logs execution abc123-def456 --tail 50
```

**Flags:**

- `--follow`, `-f` - Stream new log entries in real-time
- `--tail` - Show only last N lines

---

## Sync Command

Unified sync for all resource types.

### `fluxbase sync`

Sync all Fluxbase resources from a directory structure.

```bash
fluxbase sync                           # Auto-detect from ./fluxbase/ or current dir
fluxbase sync --dir ./src               # Specify root directory
fluxbase sync --namespace production    # Apply namespace to all
fluxbase sync --dry-run                 # Preview all changes
```

**Flags:**

- `--dir` - Root directory (default: `./fluxbase` or current directory)
- `--namespace` - Target namespace for all resources (default: `default`)
- `--dry-run` - Preview changes without applying

The sync command automatically detects and syncs these subdirectories:

```
fluxbase/
├── rpc/           # SQL files for stored procedures
├── migrations/    # Database migrations (.up.sql, .down.sql)
├── functions/     # Edge functions (.ts, .js)
├── jobs/          # Background jobs (.ts, .js)
└── chatbots/      # Chatbot configurations (.yaml)
```

Resources are synced in dependency order: RPC → Migrations → Functions → Jobs → Chatbots
