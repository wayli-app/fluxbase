---
title: CLI Command Reference
description: Complete reference for all Fluxbase CLI commands
---

## Command Overview

```
fluxbase [command] [subcommand] [flags]
```

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

```bash
fluxbase functions logs my-function
fluxbase functions logs my-function --tail 50
```

### `fluxbase functions sync`

```bash
fluxbase functions sync --dir ./functions
fluxbase functions sync --dir ./functions --namespace production --dry-run
```

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

```bash
fluxbase jobs stats
```

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

## RPC Commands

Invoke stored procedures.

```bash
# List procedures
fluxbase rpc list

# Invoke procedure
fluxbase rpc invoke default/calculate_totals
fluxbase rpc invoke default/process --params '{"id": 123}'
```

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
