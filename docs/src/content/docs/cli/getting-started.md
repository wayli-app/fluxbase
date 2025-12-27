---
title: CLI Getting Started
description: Get started with the Fluxbase CLI
---

This guide walks you through authenticating and using the Fluxbase CLI. Make sure you've [installed the CLI](/cli/installation/) first.

## Authentication

Before using the CLI, you need to authenticate with your Fluxbase server.

### Interactive Login

The simplest way to login:

```bash
fluxbase auth login
```

You'll be prompted for:
- **Server URL**: Your Fluxbase server (e.g., `https://api.example.com`)
- **Email**: Your account email
- **Password**: Your account password

### Two-Factor Authentication (2FA)

If your account has 2FA enabled, you'll be prompted for a verification code after entering your password:

```
Enter 2FA code: 123456
```

The CLI supports both TOTP (authenticator app) and email-based verification codes.

### Non-interactive Login

For scripts and CI/CD:

```bash
# With email/password
fluxbase auth login \
  --server https://api.example.com \
  --email user@example.com \
  --password "your-password"

# With API token
fluxbase auth login \
  --server https://api.example.com \
  --token "your-api-token"
```

### Using Environment Variables

You can also use environment variables:

```bash
export FLUXBASE_SERVER="https://api.example.com"
export FLUXBASE_TOKEN="your-api-token"

# Now commands will use these credentials
fluxbase functions list
```

## Check Authentication Status

```bash
fluxbase auth status
```

This shows all configured profiles and their authentication status.

## Profile Management

The CLI supports multiple profiles for different environments:

```bash
# Login to different environments with named profiles
fluxbase auth login --profile dev --server http://localhost:8080
fluxbase auth login --profile staging --server https://staging.example.com
fluxbase auth login --profile prod --server https://api.example.com

# Switch between profiles
fluxbase auth switch prod

# Use a specific profile for a command
fluxbase --profile staging functions list
```

## Quick Examples

### List Functions

```bash
fluxbase functions list
```

### Deploy a Function

```bash
fluxbase functions create my-function --code ./function.ts
```

### Submit a Job

```bash
fluxbase jobs submit process-data --payload '{"file": "data.csv"}'
```

### Upload a File

```bash
fluxbase storage objects upload my-bucket images/photo.jpg ./photo.jpg
```

### Query a Table

```bash
fluxbase tables query users --where "role=eq.admin" --limit 10
```

## Output Formats

The CLI supports multiple output formats:

```bash
# Table format (default)
fluxbase functions list

# JSON format
fluxbase functions list -o json

# YAML format
fluxbase functions list -o yaml

# Quiet mode (minimal output)
fluxbase functions list -q
```

## Global Flags

These flags work with all commands:

| Flag | Short | Description |
|------|-------|-------------|
| `--config` | | Config file path (default: `~/.fluxbase/config.yaml`) |
| `--profile` | `-p` | Profile to use |
| `--output` | `-o` | Output format: `table`, `json`, `yaml` |
| `--no-headers` | | Hide table headers |
| `--quiet` | `-q` | Minimal output |
| `--debug` | | Enable debug output |

## What's Next?

Choose your path based on what you want to do:

### Deploy Functions and Jobs

```bash
# Create a function
echo 'export default (req) => new Response("Hello!")' > hello.ts
fluxbase functions create hello --code ./hello.ts

# Test it
fluxbase functions invoke hello
```

See [Command Reference - Functions](/cli/commands/#functions-commands) for more.

### Work with Data

```bash
# List your tables
fluxbase tables list

# Query data
fluxbase tables query users --select "id,email" --limit 10

# Insert a record
fluxbase tables insert users --data '{"email": "new@example.com"}'
```

See [Command Reference - Tables](/cli/commands/#table-commands) for more.

### Set Up CI/CD

Configure automated deployments using environment variables and the `sync` command.

See [Workflows](/cli/workflows/) for GitHub Actions and GitLab CI examples.

### Manage Multiple Environments

Set up profiles for dev, staging, and production servers.

See [Configuration](/cli/configuration/) for profile management.

## Further Reading

- [Command Reference](/cli/commands/) - Full command documentation
- [Configuration](/cli/configuration/) - Configuration file details
- [Workflows](/cli/workflows/) - CI/CD and development workflows
