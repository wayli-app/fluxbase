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
| `--tenant`     | `-t`  | Tenant slug (overrides profile default)               |

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

# SSO login (opens browser)
fluxbase auth login --server URL --sso

# Save to named profile
fluxbase auth login --profile prod --server URL
```

**Flags:**

- `--server` - Fluxbase server URL
- `--email` - Email address
- `--password` - Password
- `--token` - API token (alternative to email/password)
- `--sso` - Login via SSO (opens browser for OAuth/SAML authentication)
- `--profile` - Profile name (default: "default")
- `--use-keychain` - Store credentials in system keychain

**Note:** When password login is disabled on the server, the CLI automatically detects this and initiates SSO login.

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
fluxbase functions invoke my-function --async
```

**Flags:**

- `--data` - JSON payload to send
- `--file` - Load payload from file
- `--async` - Run asynchronously (returns immediately)

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
- `--keep` - Keep functions not present in directory

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
fluxbase jobs submit my-job --file ./payload.json
fluxbase jobs submit my-job --priority 10
fluxbase jobs submit my-job --schedule "0 * * * *"
```

**Flags:**

- `--payload` - JSON payload to send
- `--file` - Load payload from file
- `--priority` - Job priority (higher = more important)
- `--schedule` - Cron schedule for recurring jobs

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
- `--keep` - Keep jobs not present in directory

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
fluxbase storage buckets create my-bucket --max-size 10737418240  # 10GB limit

# Delete bucket
fluxbase storage buckets delete my-bucket
```

### `fluxbase storage buckets create`

**Flags:**

- `--public` - Make bucket publicly accessible
- `--max-size` - Maximum bucket size in bytes

### Object Commands

```bash
# List objects
fluxbase storage objects list my-bucket
fluxbase storage objects list my-bucket --prefix images/

# Upload file
fluxbase storage objects upload my-bucket path/to/file.jpg ./local-file.jpg
fluxbase storage objects upload my-bucket path/to/file.jpg ./local-file.jpg --content-type image/jpeg

# Download file
fluxbase storage objects download my-bucket path/to/file.jpg ./local-file.jpg

# Delete object
fluxbase storage objects delete my-bucket path/to/file.jpg

# Get signed URL
fluxbase storage objects url my-bucket path/to/file.jpg --expires 7200
```

### `fluxbase storage objects upload`

**Flags:**

- `--content-type` - MIME type for the uploaded file

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

# Sync chatbots from directory
fluxbase chatbots sync --dir ./chatbots
```

### `fluxbase chatbots create`

**Flags:**

- `--system-prompt` - System prompt for the chatbot
- `--model` - AI model to use (e.g., `gpt-4`, `gpt-3.5-turbo`)
- `--temperature` - Response randomness (0.0-2.0)
- `--max-tokens` - Maximum response length
- `--knowledge-base` - Knowledge base ID to attach

### `fluxbase chatbots update`

**Flags:**

- `--system-prompt` - System prompt for the chatbot
- `--model` - AI model to use
- `--temperature` - Response randomness (0.0-2.0)
- `--max-tokens` - Maximum response length

### `fluxbase chatbots sync`

Sync chatbots from a local directory.

```bash
fluxbase chatbots sync --dir ./chatbots
fluxbase chatbots sync --dir ./chatbots --namespace production --dry-run
```

**Flags:**

- `--dir` - Directory containing chatbot files (default: `./chatbots`)
- `--namespace` - Target namespace (default: `default`)
- `--dry-run` - Preview changes without applying
- `--delete-missing` - Delete chatbots not in local directory

---

## Knowledge Base Commands

Manage knowledge bases for RAG (Retrieval-Augmented Generation). Knowledge bases store documents that are chunked, embedded, and indexed for semantic search.

### `fluxbase kb list`

List all knowledge bases.

```bash
fluxbase kb list
fluxbase kb list --namespace production
fluxbase kb list -o json
```

**Flags:**

- `--namespace` - Filter by namespace

### `fluxbase kb get`

Get details of a specific knowledge base.

```bash
fluxbase kb get abc123
```

### `fluxbase kb create`

Create a new knowledge base.

```bash
fluxbase kb create docs --description "Product documentation"
fluxbase kb create docs --embedding-model text-embedding-ada-002 --chunk-size 512
```

**Flags:**

- `--description` - Knowledge base description
- `--embedding-model` - Embedding model to use
- `--chunk-size` - Document chunk size (default: 512)
- `--namespace` - Target namespace (default: `default`)

### `fluxbase kb update`

Update an existing knowledge base.

```bash
fluxbase kb update abc123 --description "Updated description"
fluxbase kb update abc123 --entity-extraction=false
```

**Flags:**

- `--name` - New name
- `--description` - New description
- `--chunk-size` - Chunk size in tokens
- `--chunk-overlap` - Chunk overlap in tokens
- `--embedding-model` - Embedding model
- `--chunk-strategy` - `recursive` | `sentence` | `paragraph` | `fixed`
- `--visibility` - `private` | `shared` | `public`
- `--enabled` - Enable/disable the KB
- `--entity-extraction` - Enable/disable rule-based entity extraction (default: true)

### `fluxbase kb delete`

Delete a knowledge base and all its documents.

```bash
fluxbase kb delete abc123
```

### `fluxbase kb upload`

Upload a file (PDF, DOCX, TXT, MD, images with OCR enabled on the server) as a document to a knowledge base.

```bash
fluxbase kb upload abc123 ./manual.pdf
fluxbase kb upload abc123 ./guide.md --title "User Guide"
fluxbase kb upload abc123 ./report.pdf --tag finance --tag q4
```

**Flags:**

- `--title` - Document title (defaults to filename)
- `--tag` - Tag to attach (can be repeated: `--tag a --tag b`)

### `fluxbase kb documents`

List documents in a knowledge base (alias: `kb docs`).

```bash
fluxbase kb documents abc123
fluxbase kb documents abc123 --limit 100
```

**Flags:**

- `--limit` - Maximum number of documents to return (default: 50)

### `fluxbase kb graph`

Show the knowledge graph for a knowledge base, including entities and their relationships.

```bash
fluxbase kb graph abc123
```

### `fluxbase kb entities`

List entities extracted from the knowledge base.

```bash
fluxbase kb entities abc123
fluxbase kb entities abc123 --type person
fluxbase kb entities abc123 --search "John"
```

**Flags:**

- `--type` - Filter by entity type
- `--search` - Search entities by name

### `fluxbase kb chatbots`

List all chatbots using a knowledge base.

```bash
fluxbase kb chatbots abc123
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
fluxbase tables query users --order-by "created_at.desc" --offset 20 --limit 10

# Insert record
fluxbase tables insert users --data '{"email": "user@example.com"}'

# Update records
fluxbase tables update users --where "id=eq.123" --data '{"name": "New Name"}'

# Delete records
fluxbase tables delete users --where "id=eq.123"
```

### `fluxbase tables query`

**Flags:**

- `--select` - Comma-separated columns to return (default: all)
- `--where` - Filter conditions (PostgREST syntax)
- `--order-by` - Order by column (e.g., `created_at.desc`, `name.asc`)
- `--limit` - Maximum rows to return (default: 100)
- `--offset` - Number of rows to skip (for pagination)

---

## Type Generation Commands

Generate TypeScript type definitions from your database schema. The generated types can be used with the Fluxbase TypeScript SDK for type-safe database queries.

### `fluxbase types generate`

Generate TypeScript type definitions from your database schema.

```bash
# Generate types for the public schema and write to types.ts
fluxbase types generate --output types.ts

# Generate types for multiple schemas
fluxbase types generate --schemas public,auth --output types.ts

# Generate types including RPC function signatures
fluxbase types generate --include-functions --output types.ts

# Generate types without views
fluxbase types generate --include-views=false --output types.ts

# Output to stdout (for piping)
fluxbase types generate

# Generate with helper functions
fluxbase types generate --format full --output types.ts
```

**Flags:**

- `--schemas` - Schemas to include (default: `public`, comma-separated)
- `--include-functions` - Include RPC function types (default: true)
- `--include-views` - Include view types (default: true)
- `--output`, `-o` - Output file path (default: stdout)
- `--format` - Output format: `types` (interfaces only) or `full` (with helpers)

### `fluxbase types list`

List all available database schemas that can be used for type generation.

```bash
fluxbase types list
```

---

## GraphQL Commands

Execute GraphQL queries and mutations against the auto-generated GraphQL API.

### `fluxbase graphql query`

Execute a GraphQL query.

```bash
# Simple query
fluxbase graphql query '{ users { id email createdAt } }'

# Query with filtering
fluxbase graphql query '{ users(filter: {role_eq: "admin"}) { id email } }'

# Query with ordering and pagination
fluxbase graphql query '{ users(limit: 10, orderBy: [{createdAt: DESC}]) { id email } }'

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
  insertUser(data: {email: "new@example.com", name: "New User"}) {
    id email
  }
}'

# Update a record
fluxbase graphql mutation 'mutation {
  updateUser(id: "user-id", data: {name: "Updated Name"}) {
    id name
  }
}'

# Delete a record
fluxbase graphql mutation 'mutation {
  deleteUser(id: "user-id") { id }
}'

# Bulk update
fluxbase graphql mutation 'mutation {
  updateManyUser(filter: {status_eq: "active"}, data: {status: "archived"})
}'

# Bulk delete
fluxbase graphql mutation 'mutation {
  deleteManyUser(filter: {status_eq: "inactive"})
}'

# Mutation with variables
fluxbase graphql mutation 'mutation CreateUser($data: UserInput!) {
  insertUser(data: $data) { id }
}' --var 'data={"email":"test@example.com","name":"Test User"}'

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
fluxbase rpc get default/calculate_totals --namespace production
```

**Flags:**

- `--namespace` - Namespace (default: `default`)

### `fluxbase rpc invoke`

Invoke a stored procedure.

```bash
fluxbase rpc invoke default/calculate_totals
fluxbase rpc invoke default/process --params '{"id": 123}'
fluxbase rpc invoke default/batch_update --file ./params.json --async
fluxbase rpc invoke default/process --namespace production
```

**Flags:**

- `--namespace` - Namespace (default: `default`)
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
- `--keep` - Keep procedures not in local directory
- `--delete-missing` - Delete procedures not in local directory

---

## Webhook Commands

Manage webhooks for database events.

```bash
# List webhooks
fluxbase webhooks list

# Get webhook details
fluxbase webhooks get abc123

# Create webhook
fluxbase webhooks create --url https://example.com/webhook --events "INSERT,UPDATE"

# Update webhook
fluxbase webhooks update abc123 --url https://new-url.com/webhook
fluxbase webhooks update abc123 --events "INSERT,UPDATE,DELETE"
fluxbase webhooks update abc123 --enabled=false

# Test webhook
fluxbase webhooks test abc123

# View deliveries
fluxbase webhooks deliveries abc123

# Delete webhook
fluxbase webhooks delete abc123
```

### `fluxbase webhooks create`

Create a new webhook.

```bash
fluxbase webhooks create --url https://example.com/webhook --events "INSERT,UPDATE"
fluxbase webhooks create --url https://example.com/webhook --events "*" --secret "my-secret"
```

**Flags:**

- `--url` - Webhook URL (required)
- `--events` - Comma-separated events (e.g., `INSERT,UPDATE,DELETE` or `*` for all)
- `--secret` - Secret for webhook signature verification

### `fluxbase webhooks update`

Update a webhook.

```bash
fluxbase webhooks update abc123 --url https://new-url.com/webhook
fluxbase webhooks update abc123 --events "INSERT,UPDATE,DELETE"
fluxbase webhooks update abc123 --enabled=false
```

**Flags:**

- `--url` - New webhook URL
- `--events` - New comma-separated events
- `--enabled` - Enable or disable the webhook

---

## Client Key Commands

Manage client keys for API authentication.

```bash
# List client keys
fluxbase clientkeys list

# Create client key
fluxbase clientkeys create --name "Production" --scopes "read:tables,write:tables"

# Get client key details
fluxbase clientkeys get abc123

# Revoke client key
fluxbase clientkeys revoke abc123

# Delete client key
fluxbase clientkeys delete abc123
```

### `fluxbase clientkeys create`

**Flags:**

- `--name` - Client key name (required)
- `--scopes` - Comma-separated scopes (e.g., `read:tables,write:tables`)
- `--rate-limit` - Rate limit per minute (e.g., `100`)
- `--expires` - Expiration duration (e.g., `30d`, `1y`)

---

## Migration Commands

Manage database migrations.

```bash
# List migrations
fluxbase migrations list

# Get migration details
fluxbase migrations get 001_create_users

# Create migration
fluxbase migrations create add_users_table --up-sql "CREATE TABLE users..." --down-sql "DROP TABLE users"

# Apply specific migration
fluxbase migrations apply 001_create_users

# Rollback migration
fluxbase migrations rollback 001_create_users

# Apply all pending
fluxbase migrations apply-pending

# Sync from directory
fluxbase migrations sync --dir ./migrations
```

### `fluxbase migrations list`

List all migrations.

```bash
fluxbase migrations list
fluxbase migrations list --namespace production
```

**Flags:**

- `--namespace` - Filter by namespace

### `fluxbase migrations get`

Get migration details.

```bash
fluxbase migrations get 001_create_users
```

### `fluxbase migrations create`

Create a new migration.

```bash
fluxbase migrations create add_users_table --up-sql "CREATE TABLE users (id SERIAL PRIMARY KEY);"
fluxbase migrations create add_users_table --up-sql "CREATE TABLE..." --down-sql "DROP TABLE..."
```

**Flags:**

- `--up-sql` - SQL for up migration
- `--down-sql` - SQL for down migration
- `--namespace` - Target namespace (default: `default`)

### `fluxbase migrations sync`

Sync migrations from a directory.

```bash
fluxbase migrations sync --dir ./migrations
fluxbase migrations sync --dir ./migrations --namespace production --no-apply
```

**Flags:**

- `--dir` - Directory containing migration files (default: `./migrations`)
- `--namespace` - Target namespace (default: `default`)
- `--no-apply` - Sync without auto-applying pending migrations
- `--dry-run` - Preview changes without applying

---

## Extension Commands

Manage PostgreSQL extensions.

```bash
# List extensions
fluxbase extensions list

# Get extension status
fluxbase extensions status pgvector

# Enable extension
fluxbase extensions enable pgvector

# Disable extension
fluxbase extensions disable pgvector
```

### `fluxbase extensions status`

Get the status of a specific extension.

```bash
fluxbase extensions status pgvector
```

### `fluxbase extensions enable`

Enable a PostgreSQL extension.

```bash
fluxbase extensions enable pgvector
fluxbase extensions enable pgvector --schema vector_schema
```

**Flags:**

- `--schema` - Schema to install the extension in (default: extension default)

---

## Realtime Commands

Manage realtime connections.

```bash
# Show stats
fluxbase realtime stats

# Broadcast message
fluxbase realtime broadcast my-channel --message '{"type": "notification"}'
fluxbase realtime broadcast my-channel --message '{"data": "value"}' --event custom-event
```

### `fluxbase realtime broadcast`

**Flags:**

- `--message` - JSON message to broadcast (required)
- `--event` - Custom event name (default: `broadcast`)

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

Manage encrypted application settings secrets. These are separate from the function secrets (`fluxbase secrets`) and are used for storing sensitive application configuration such as client keys and credentials.

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

## Service Key Commands

Manage service keys for server-to-server authentication with elevated permissions. Service keys are used for automated workflows, CI/CD pipelines, and backend services that need to access the Fluxbase API.

:::caution
Service keys have elevated permissions. Store them securely and never expose them in client-side code.
:::

### `fluxbase servicekeys list`

List all service keys.

```bash
fluxbase servicekeys list
fluxbase servicekeys list -o json
```

### `fluxbase servicekeys create`

Create a new service key.

```bash
fluxbase servicekeys create --name "Migrations Key" --scopes "migrations:*"
fluxbase servicekeys create --name "Production" --rate-limit-per-hour 100
fluxbase servicekeys create --name "CI/CD" --scopes "*" --expires 2025-12-31T23:59:59Z
```

**Flags:**

- `--name` - Service key name (required)
- `--description` - Service key description
- `--scopes` - Comma-separated scopes (default: `*` for all)
- `--rate-limit-per-minute` - Requests per minute (0 = no limit)
- `--rate-limit-per-hour` - Requests per hour (0 = no limit)
- `--expires` - Expiration time (e.g., `2025-12-31T23:59:59Z`)

:::note
The full key is only shown once at creation. Save it immediately!
:::

### `fluxbase servicekeys get`

Get details of a specific service key.

```bash
fluxbase servicekeys get abc123
```

### `fluxbase servicekeys update`

Update a service key's properties.

```bash
fluxbase servicekeys update abc123 --name "New Name"
fluxbase servicekeys update abc123 --rate-limit-per-hour 200
fluxbase servicekeys update abc123 --enabled=false
```

**Flags:**

- `--name` - New service key name
- `--description` - New service key description
- `--scopes` - New comma-separated scopes
- `--rate-limit-per-minute` - Requests per minute (0 = no limit)
- `--rate-limit-per-hour` - Requests per hour (0 = no limit)
- `--enabled` - Enable or disable the key (default: true)

### `fluxbase servicekeys disable`

Disable a service key (keeps the record but prevents use).

```bash
fluxbase servicekeys disable abc123
```

### `fluxbase servicekeys enable`

Enable a previously disabled service key.

```bash
fluxbase servicekeys enable abc123
```

### `fluxbase servicekeys delete`

Delete a service key permanently.

```bash
fluxbase servicekeys delete abc123
```

### `fluxbase servicekeys revoke`

Emergency revoke a service key immediately. This action is irreversible.

```bash
fluxbase servicekeys revoke abc123 --reason "Key compromised"
fluxbase servicekeys revoke abc123 --reason "Employee departure"
```

**Flags:**

- `--reason` - Reason for revocation (required)

The key will be permanently disabled and marked as revoked with an audit trail.

### `fluxbase servicekeys deprecate`

Mark a service key as deprecated with a grace period. The key continues working during the grace period, allowing time for migration to a new key.

```bash
fluxbase servicekeys deprecate abc123 --grace-period 24h
fluxbase servicekeys deprecate abc123 --grace-period 7d --reason "Scheduled rotation"
```

**Flags:**

- `--grace-period` - Grace period before key stops working (default: `24h`, e.g., `24h`, `7d`)
- `--reason` - Reason for deprecation

### `fluxbase servicekeys rotate`

Create a new service key as a replacement for an existing one. This deprecates the old key with a grace period and creates a new key with the same configuration.

```bash
fluxbase servicekeys rotate abc123 --grace-period 24h
fluxbase servicekeys rotate abc123 --grace-period 7d
```

**Flags:**

- `--grace-period` - Grace period for old key (default: `24h`, e.g., `24h`, `7d`)

The output shows the new key (save it immediately!) and when the old key will stop working.

### `fluxbase servicekeys revocations`

View the revocation audit log for a service key.

```bash
fluxbase servicekeys revocations abc123
```

Shows all revocation events including emergency revocations, rotations, and expirations.

---

## Config Commands

Manage CLI configuration.

```bash
# Initialize config
fluxbase config init

# View config
fluxbase config view

# Get config value
fluxbase config get defaults.output

# Set config value
fluxbase config set defaults.output json

# List profiles
fluxbase config profiles

# Add profile
fluxbase config profiles add staging

# Remove profile
fluxbase config profiles remove staging
```

### `fluxbase config get`

Get a specific config value.

```bash
fluxbase config get defaults.output
fluxbase config get current_profile
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

## MCP Commands

Manage custom MCP (Model Context Protocol) tools for AI assistant integration. Custom MCP tools allow you to extend the Fluxbase MCP server with your own tools that can be used by AI assistants.

### `fluxbase mcp tools list`

List all custom MCP tools.

```bash
fluxbase mcp tools list
fluxbase mcp tools list --namespace production
fluxbase mcp tools list -o json
```

**Flags:**

- `--namespace` - Filter by namespace

### `fluxbase mcp tools get`

Get details of a specific custom MCP tool.

```bash
fluxbase mcp tools get weather_forecast
fluxbase mcp tools get weather_forecast -o json
```

### `fluxbase mcp tools create`

Create a new custom MCP tool.

```bash
fluxbase mcp tools create weather_forecast --code ./weather.ts
fluxbase mcp tools create weather_forecast --code ./weather.ts --description "Get weather forecast"
fluxbase mcp tools create weather_forecast --code ./weather.ts --timeout 60 --memory 256
```

**Flags:**

- `--code` - Path to TypeScript code file (required)
- `--namespace` - Namespace (default: `default`)
- `--description` - Tool description
- `--timeout` - Execution timeout in seconds (default: 30)
- `--memory` - Memory limit in MB (default: 128)
- `--allow-net` - Allow network access (default: true)
- `--allow-env` - Allow environment variable access
- `--allow-read` - Allow file read access
- `--allow-write` - Allow file write access

### `fluxbase mcp tools update`

Update an existing custom MCP tool.

```bash
fluxbase mcp tools update weather_forecast --code ./weather.ts
fluxbase mcp tools update weather_forecast --timeout 60
```

**Flags:**

- `--code` - Path to TypeScript code file
- `--namespace` - Namespace (default: `default`)
- `--description` - Tool description
- `--timeout` - Execution timeout in seconds
- `--memory` - Memory limit in MB

### `fluxbase mcp tools delete`

Delete a custom MCP tool.

```bash
fluxbase mcp tools delete weather_forecast
```

### `fluxbase mcp tools sync`

Sync custom MCP tools from a directory to the server.

```bash
fluxbase mcp tools sync                                # Auto-detect directory
fluxbase mcp tools sync --dir ./mcp-tools
fluxbase mcp tools sync --dir ./mcp-tools --namespace production
fluxbase mcp tools sync --dry-run
```

**Flags:**

- `--dir` - Directory containing tool files (auto-detects `./fluxbase/mcp-tools/` or `./mcp-tools/`)
- `--namespace` - Target namespace (default: `default`)
- `--dry-run` - Preview changes without applying

Each `.ts` file in the directory will be synced as a custom tool. Tool name defaults to filename. You can use annotations in your code:

```typescript
// @fluxbase:name my_tool
// @fluxbase:namespace production
// @fluxbase:description Get weather forecast
// @fluxbase:timeout 30
// @fluxbase:memory 128
// @fluxbase:allow-net
```

### `fluxbase mcp tools test`

Test a custom MCP tool by invoking it with sample arguments.

```bash
fluxbase mcp tools test weather_forecast --args '{"location": "New York"}'
```

**Flags:**

- `--args` - JSON arguments to pass to the tool (default: `{}`)
- `--namespace` - Namespace (default: `default`)

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
fluxbase sync --keep                    # Keep resources not in directory
fluxbase sync --analyze                 # Analyze bundle sizes
fluxbase sync --analyze --verbose       # Detailed analysis
```

**Flags:**

- `--dir` - Root directory (default: `./fluxbase` or current directory)
- `--namespace` - Target namespace for all resources (default: `default`)
- `--dry-run` - Preview changes without applying
- `--keep` - Keep items not present in directory
- `--analyze` - Analyze bundle sizes (shows breakdown of what's in each bundle)
- `--verbose` - Show detailed analysis (with `--analyze`)

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

---

## Branch Commands

Manage database branches for isolated development and testing environments. See the [Database Branching Guide](/guides/branching/) for full documentation.

### `fluxbase branch list`

List all database branches.

```bash
fluxbase branch list
fluxbase branch list --type preview
fluxbase branch list --mine
fluxbase branch list -o json
```

**Flags:**

- `--type` - Filter by branch type (`main`, `preview`, `persistent`)
- `--mine`, `-m` - Show only branches created by you

### `fluxbase branch get`

Get details of a specific branch.

```bash
fluxbase branch get my-feature
fluxbase branch get pr-123
fluxbase branch get 550e8400-e29b-41d4-a716-446655440000
```

### `fluxbase branch create`

Create a new database branch.

```bash
# Basic branch
fluxbase branch create my-feature

# With full data clone
fluxbase branch create staging --clone-data full_clone

# Persistent branch (not auto-deleted)
fluxbase branch create staging --type persistent

# Branch with expiration
fluxbase branch create temp-test --expires-in 24h

# Branch linked to GitHub PR
fluxbase branch create pr-123 --pr 123 --repo owner/repo

# Branch from another branch
fluxbase branch create feature-b --from feature-a
```

**Flags:**

- `--clone-data` - Data clone mode: `schema_only` (default), `full_clone`, `seed_data`
- `--type` - Branch type: `preview` (default), `persistent`
- `--expires-in` - Auto-delete after duration (e.g., `24h`, `7d`)
- `--from` - Parent branch to clone from (default: `main`)
- `--pr` - GitHub PR number to associate
- `--repo` - GitHub repository (e.g., `owner/repo`)
- `--seeds-dir` - Custom directory containing seed SQL files (only with `--clone-data seed_data`)

After creation, the command shows how to connect:

```
Branch 'my-feature' created successfully!

Slug:     my-feature
Database: branch_my_feature
Status:   ready

To use this branch:
  Header:  X-Fluxbase-Branch: my-feature
  Query:   ?branch=my-feature
  SDK:     { branch: 'my-feature' }
```

### `fluxbase branch delete`

Delete a database branch and its associated database.

```bash
fluxbase branch delete my-feature
fluxbase branch delete pr-123 --force
```

**Flags:**

- `--force`, `-f` - Skip confirmation prompt

:::caution
This action is irreversible. All data in the branch will be permanently deleted.
:::

### `fluxbase branch reset`

Reset a branch to its parent state, recreating the database.

```bash
fluxbase branch reset my-feature
fluxbase branch reset pr-123 --force
```

**Flags:**

- `--force`, `-f` - Skip confirmation prompt

This drops the branch database and recreates it from the parent branch. All changes are lost.

### `fluxbase branch status`

Show the current status of a branch.

```bash
fluxbase branch status my-feature
```

Output shows the branch name, slug, and current status (`creating`, `ready`, `migrating`, `error`, `deleting`).

### `fluxbase branch activity`

Show the activity log for a branch.

```bash
fluxbase branch activity my-feature
fluxbase branch activity pr-123 --limit 20
```

**Flags:**

- `--limit`, `-n` - Maximum number of entries to show (default: 50)

### `fluxbase branch stats`

Show connection pool statistics for all branches.

```bash
fluxbase branch stats
```

Useful for debugging and monitoring database connections across branches.

### `fluxbase branch use`

Set the default branch for all subsequent CLI commands. This saves the branch to your profile config.

```bash
fluxbase branch use my-feature
fluxbase branch use pr-123
fluxbase branch use main  # Reset to main branch
```

After setting a default branch, all CLI commands will automatically use that branch without needing to specify it each time.

### `fluxbase branch current`

Show the current default branch set for CLI commands.

```bash
fluxbase branch current
```

---

## Admin Commands

Manage admin users, invitations, and sessions for the Fluxbase dashboard. Admin users have access to the admin dashboard for managing database, users, functions, and other platform features.

### Admin User Commands

#### `fluxbase admin users list`

List all admin/dashboard users.

```bash
fluxbase admin users list
fluxbase admin users list -o json
```

#### `fluxbase admin users get`

Get details of a specific admin user.

```bash
fluxbase admin users get 550e8400-e29b-41d4-a716-446655440000
```

#### `fluxbase admin users invite`

Invite a new admin user via email.

```bash
fluxbase admin users invite --email admin@example.com
fluxbase admin users invite --email admin@example.com --role instance_admin
```

**Flags:**

- `--email` - Email address to invite (required)
- `--role` - Role for the new user: `tenant_admin` (default) or `instance_admin`

#### `fluxbase admin users delete`

Delete an admin user.

```bash
fluxbase admin users delete 550e8400-e29b-41d4-a716-446655440000
fluxbase admin users delete 550e8400-e29b-41d4-a716-446655440000 --force
```

**Flags:**

- `--force`, `-f` - Skip confirmation prompt

### Admin Invitation Commands

#### `fluxbase admin invitations list`

List pending and accepted admin invitations.

```bash
fluxbase admin invitations list
fluxbase admin invitations list --include-accepted
fluxbase admin invitations list --include-expired
```

**Flags:**

- `--include-accepted` - Include accepted invitations
- `--include-expired` - Include expired invitations

#### `fluxbase admin invitations revoke`

Revoke a pending admin invitation.

```bash
fluxbase admin invitations revoke abc123def456
fluxbase admin invitations revoke abc123def456 --force
```

**Flags:**

- `--force`, `-f` - Skip confirmation prompt

### Admin Session Commands

#### `fluxbase admin sessions list`

List all active admin sessions.

```bash
fluxbase admin sessions list
fluxbase admin sessions list -o json
```

#### `fluxbase admin sessions revoke`

Revoke a specific admin session.

```bash
fluxbase admin sessions revoke 550e8400-e29b-41d4-a716-446655440000
```

**Flags:**

- `--force`, `-f` - Skip confirmation prompt

#### `fluxbase admin sessions revoke-all`

Revoke all sessions for a specific admin user.

```bash
fluxbase admin sessions revoke-all 550e8400-e29b-41d4-a716-446655440000
fluxbase admin sessions revoke-all 550e8400-e29b-41d4-a716-446655440000 --force
```

**Flags:**

- `--force`, `-f` - Skip confirmation prompt

### Admin Password Reset

#### `fluxbase admin password-reset`

Send a password reset email to an admin user.

```bash
fluxbase admin password-reset --email admin@example.com
```

**Flags:**

- `--email` - Email address of the admin user (required)

---

## User Commands

Manage application users (end users of your application). For admin/dashboard users, use `fluxbase admin users` instead.

### `fluxbase users list`

List all application users.

```bash
fluxbase users list
fluxbase users list -o json
fluxbase users list --search john
```

**Flags:**

- `--search` - Search users by email

### `fluxbase users get`

Get details of a specific application user.

```bash
fluxbase users get 550e8400-e29b-41d4-a716-446655440000
```

### `fluxbase users invite`

Invite a new application user via email.

```bash
fluxbase users invite --email user@example.com
```

**Flags:**

- `--email` - Email address to invite (required)

### `fluxbase users delete`

Delete an application user.

```bash
fluxbase users delete 550e8400-e29b-41d4-a716-446655440000
fluxbase users delete 550e8400-e29b-41d4-a716-446655440000 --force
```

**Flags:**

- `--force`, `-f` - Skip confirmation prompt

---

## Version Command

### `fluxbase version`

Show CLI version information.

```bash
fluxbase version
```

---

## Completion Command

### `fluxbase completion`

Generate shell completion scripts for bash, zsh, fish, and powershell.

```bash
# Bash
fluxbase completion bash > /etc/bash_completion.d/fluxbase

# Zsh
fluxbase completion zsh > "${fpath[1]}/_fluxbase"

# Fish
fluxbase completion fish > ~/.config/fish/completions/fluxbase.fish

# PowerShell
fluxbase completion powershell > ~/.config/powershell/completions/fluxbase.ps1
```

After installation, restart your shell or source the completion file to enable autocompletion.

---

## AI Providers Commands

Manage AI providers (OpenAI, Azure, Ollama) used by chatbots, embeddings, and vector search. The command group is `providers` (alias `provider`).

:::note
The command is `fluxbase providers`, **not** `fluxbase ai providers`. (Some older examples referenced an `ai` parent that does not exist.)
:::

### `fluxbase providers list`

List configured AI providers.

```bash
fluxbase providers list
fluxbase providers list -o json
```

### `fluxbase providers get [id]`

Get a provider by ID.

### `fluxbase providers create [name]`

Create a new provider.

```bash
fluxbase providers create openai \
  --type openai \
  --display-name "OpenAI" \
  --api-key sk-... \
  --model gpt-4o \
  --default
```

**Flags:** `--type` (`openai`, `azure`, `ollama`), `--display-name`, `--api-key`, `--base-url` (Ollama/custom), `--model`, `--default` (set as default), `--enabled` (default `true`).

### `fluxbase providers update [id]`

Update a provider. **Flags:** `--display-name`, `--api-key`, `--base-url`, `--model`, `--embedding-model`, `--enabled`, `--set-enabled`.

### `fluxbase providers delete [id]`

Delete a provider (aliases: `rm`, `remove`).

### `fluxbase providers set-default [id]`

Set a provider as the default for chat/completions.

### `fluxbase providers set-embedding [id]` / `clear-embedding [id]`

Set or clear the provider used for embeddings.

### `fluxbase providers metrics`

Show AI usage metrics (token spend, request counts).

## Internal Schema Commands

Manage Fluxbase's internal database schemas (auth, storage, jobs, functions, realtime, ai, rpc, platform, etc.) declaratively. Powered by pgschema diff/apply.

### `fluxbase internal-schema dump`

Dump the current internal schema to SQL.

### `fluxbase internal-schema plan`

Show the planned changes between the desired and live internal schema.

### `fluxbase internal-schema apply`

Apply planned changes. **Flags:** `--auto-approve` (skip confirmation), `--allow-destructive` (permit `DROP`/`ALTER`).

### `fluxbase internal-schema validate`

Validate the internal schema against the live database. **Flag:** `--fail-on-drift` (exit non-zero if drift is detected — useful in CI).

### `fluxbase internal-schema status`

Show the current internal-schema status.

### `fluxbase internal-schema migrate`

One-time migration from imperative internal migrations to the declarative schema. **Flag:** `--keep-old-migrations` (retain the old migration records).

**Persistent flag:** `--file` — path to a schema file (for backward compatibility).

## Schema Commands

Manage your application's own database schema (e.g. the `public` schema) declaratively from a desired-state SQL file. This is the app-developer counterpart to `internal-schema` (which manages Fluxbase's internal schemas) and an opt-in alternative to imperative `migrations sync`. Fluxbase stores the synced content and reconciles the live database via pgschema diff/apply on every sync. A given `(namespace, schema)` should use one mode, not both. Requires `database.declarative_app_schema.enabled=true` on the server for startup auto-apply; `sync`/`plan`/`apply` work on demand regardless. See the [Database Migrations guide](/guides/database-migrations/) for the full workflow.

### `fluxbase schema sync`

Read a desired-state schema file (`<dir>/<schema>.sql`) and sync it to Fluxbase. Computes a fingerprint, stores the content, and applies the diff unless `--no-apply` is set. Re-syncing unchanged content is a no-op; drift introduced outside Fluxbase is reconciled on every sync.

```bash
fluxbase schema sync --dir fluxbase/schema --namespace wayli
fluxbase schema sync --dir fluxbase/schema --namespace wayli --no-apply
```

**Flags:** `--dir` (directory containing the schema file, default `./schema`), `--no-apply` (store content only, do not apply), `--allow-destructive` (permit destructive changes during apply).

### `fluxbase schema status`

Show the status of declarative app schema(s). With `--namespace`, shows a single schema's stored fingerprint and last-applied state; without it, lists all stored app schemas.

```bash
fluxbase schema status
fluxbase schema status --namespace wayli
```

### `fluxbase schema plan`

Compare the stored schema content to the live database and show the pending changes. Does not modify the database.

```bash
fluxbase schema plan --namespace wayli
```

### `fluxbase schema validate`

Validate that the live database matches the stored schema content. Useful for CI/CD pipelines. **Flag:** `--fail-on-drift` (exit non-zero if drift is detected).

```bash
fluxbase schema validate --namespace wayli --fail-on-drift
```

### `fluxbase schema apply`

Apply the already-stored schema content for a namespace (reconcile drift).

```bash
fluxbase schema apply --namespace wayli
```

**Persistent flags:** `--namespace` (app namespace, e.g. `wayli`), `--schema` (PostgreSQL schema name, default `public`).

## Command Aliases

Many commands have shorter aliases for convenience:

| Command       | Aliases                             |
| ------------- | ----------------------------------- |
| `admin`       | `adm`                               |
| `branch`      | `branches`, `br`                    |
| `chatbots`    | `chatbot`, `cb`                     |
| `clientkeys`  | `clientkey`, `keys`                 |
| `extensions`  | `extension`, `ext`                  |
| `functions`   | `fn`, `function`                    |
| `graphql`     | `gql`                               |
| `jobs`        | `job`                               |
| `kb`          | `knowledge-bases`, `knowledge-base` |
| `logs`        | `log`                               |
| `migrations`  | `migration`, `migrate`              |
| `realtime`    | `rt`                                |
| `secrets`     | `secret`                            |
| `servicekeys` | `servicekey`, `sk`                  |
| `tables`      | `table`, `db`                       |
| `users`       | `user`                              |
| `webhooks`    | `webhook`, `wh`                     |

Examples:

```bash
fluxbase fn list          # Same as fluxbase functions list
fluxbase br create test   # Same as fluxbase branch create test
fluxbase rt stats         # Same as fluxbase realtime stats
```
