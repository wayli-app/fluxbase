---
title: CLI Workflows
description: Common workflows and best practices for the Fluxbase CLI
---

This guide covers common workflows for developing and deploying with the Fluxbase CLI.

## Project Structure

Organize your Fluxbase project with this recommended structure:

```
my-project/
├── fluxbase/
│   ├── functions/
│   │   ├── _shared/
│   │   │   ├── utils.ts
│   │   │   └── auth.ts
│   │   ├── api-handler.ts
│   │   └── webhook.ts
│   ├── jobs/
│   │   ├── _shared/
│   │   │   └── helpers.ts
│   │   ├── process-data.ts
│   │   └── send-emails.ts
│   ├── rpc/
│   │   ├── calculate_totals.sql
│   │   └── cleanup_old_records.sql
│   ├── migrations/
│   │   ├── 001_create_users.up.sql
│   │   ├── 001_create_users.down.sql
│   │   ├── 002_add_profiles.up.sql
│   │   └── 002_add_profiles.down.sql
│   └── chatbots/
│       └── support-bot.yaml
├── .fluxbase/
│   └── config.yaml
└── package.json
```

## Local Development Workflow

### Initial Setup

1. **Install the CLI** (see [Installation](/cli/installation/))

2. **Authenticate with your server:**

```bash
# Development server
fluxbase auth login --profile dev --server http://localhost:8080

# Production server
fluxbase auth login --profile prod --server https://api.example.com
```

3. **Verify authentication:**

```bash
fluxbase auth status
fluxbase auth whoami
```

### Development Cycle

**Deploy changes with sync:**

```bash
# Preview what would change
fluxbase sync --dry-run

# Deploy all resources
fluxbase sync

# Deploy specific resource types
fluxbase functions sync --dir ./fluxbase/functions
fluxbase jobs sync --dir ./fluxbase/jobs
```

**Test functions locally:**

```bash
# Invoke a function
fluxbase functions invoke my-function --data '{"test": true}'

# View logs
fluxbase functions logs my-function --follow
```

**Debug issues:**

```bash
# Enable debug output
fluxbase --debug functions invoke my-function

# Stream all logs
fluxbase logs tail --level error

# View execution logs
fluxbase logs execution <execution-id>
```

## Multi-Environment Management

### Setting Up Profiles

Create profiles for each environment:

```bash
# Local development
fluxbase auth login \
  --profile dev \
  --server http://localhost:8080 \
  --email admin@localhost.local \
  --password admin

# Staging
fluxbase auth login \
  --profile staging \
  --server https://staging.example.com \
  --token "$STAGING_TOKEN"

# Production
fluxbase auth login \
  --profile prod \
  --server https://api.example.com \
  --token "$PROD_TOKEN" \
  --use-keychain
```

### Switching Environments

```bash
# Switch default profile
fluxbase auth switch prod

# Use specific profile for a command
fluxbase --profile staging functions list

# Check which profile is active
fluxbase auth status
```

### Environment-Specific Namespaces

Use namespaces to isolate resources within an environment:

```bash
# Deploy to staging namespace
fluxbase sync --namespace staging

# Deploy to production namespace
fluxbase sync --namespace production
```

## CI/CD Integration

### Environment Variables

Configure authentication using environment variables:

```bash
export FLUXBASE_SERVER="https://api.example.com"
export FLUXBASE_TOKEN="your-api-token"
```

### GitHub Actions

```yaml
name: Deploy to Fluxbase

on:
  push:
    branches: [main]
    paths:
      - 'fluxbase/**'

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install Fluxbase CLI
        run: |
          curl -fsSL https://raw.githubusercontent.com/fluxbase-eu/fluxbase/main/install-cli.sh | bash
          echo "/usr/local/bin" >> $GITHUB_PATH

      - name: Deploy to production
        env:
          FLUXBASE_SERVER: ${{ secrets.FLUXBASE_SERVER }}
          FLUXBASE_TOKEN: ${{ secrets.FLUXBASE_TOKEN }}
        run: |
          fluxbase sync --namespace production
```

### GitLab CI

```yaml
stages:
  - deploy

deploy:
  stage: deploy
  image: ubuntu:latest
  before_script:
    - apt-get update && apt-get install -y curl
    - curl -fsSL https://raw.githubusercontent.com/fluxbase-eu/fluxbase/main/install-cli.sh | bash
  script:
    - fluxbase sync --namespace production
  variables:
    FLUXBASE_SERVER: $FLUXBASE_SERVER
    FLUXBASE_TOKEN: $FLUXBASE_TOKEN
  only:
    - main
  changes:
    - fluxbase/**
```

### Deployment Previews

For pull request previews, deploy to a unique namespace:

```yaml
# GitHub Actions example
- name: Deploy preview
  if: github.event_name == 'pull_request'
  run: |
    fluxbase sync --namespace "pr-${{ github.event.number }}"
```

## Common Tasks

### Deploy a Function End-to-End

1. **Create the function file:**

```typescript
// fluxbase/functions/hello.ts
export default async function handler(req: Request): Promise<Response> {
  const { name } = await req.json();
  return new Response(JSON.stringify({ message: `Hello, ${name}!` }));
}
```

2. **Set required secrets:**

```bash
fluxbase secrets set API_KEY "your-api-key"
```

3. **Deploy:**

```bash
fluxbase functions sync
```

4. **Test:**

```bash
fluxbase functions invoke hello --data '{"name": "World"}'
```

5. **Monitor:**

```bash
fluxbase functions logs hello --follow
```

### Run a One-Off Job

```bash
# Submit job with payload
fluxbase jobs submit process-data --payload '{"batch_id": 123}'

# Check status
fluxbase jobs status <job-id>

# View logs
fluxbase logs execution <job-id>
```

### Manage Secrets Across Environments

```bash
# Set secrets for each environment
fluxbase --profile dev secrets set API_KEY "dev-key"
fluxbase --profile staging secrets set API_KEY "staging-key"
fluxbase --profile prod secrets set API_KEY "prod-key"

# Namespace-scoped secrets
fluxbase secrets set DB_PASSWORD "secret" --scope namespace --namespace production
```

### Debug Failing Functions

1. **Check recent logs:**

```bash
fluxbase logs list --category execution --level error --since 1h
```

2. **Get execution details:**

```bash
fluxbase logs execution <execution-id>
```

3. **Test with debug output:**

```bash
fluxbase --debug functions invoke my-function --data '{"test": true}'
```

4. **Tail logs in real-time:**

```bash
fluxbase logs tail --category execution --component my-function
```

### Database Operations

```bash
# List tables
fluxbase tables list

# Query data
fluxbase tables query users --select "id,email" --where "role=eq.admin" --limit 10

# Export as JSON
fluxbase tables query users -o json > users.json

# Run migrations
fluxbase migrations apply-pending
```

## Best Practices

### Use Dry Run Before Deploying

Always preview changes before applying:

```bash
fluxbase sync --dry-run
```

### Store Credentials Securely

- Use `--use-keychain` for local development
- Use environment variables in CI/CD
- Never commit tokens to version control

### Organize with Namespaces

- Use namespaces to separate concerns (e.g., `default`, `internal`, `webhooks`)
- Use environment-specific namespaces in production

### Version Control Your Resources

- Keep all Fluxbase resources in version control
- Use meaningful migration names (e.g., `003_add_user_roles`)
- Review sync diffs in pull requests

### Monitor Deployments

```bash
# After deploying, verify resources
fluxbase functions list
fluxbase jobs list

# Watch for errors
fluxbase logs tail --level error
```
