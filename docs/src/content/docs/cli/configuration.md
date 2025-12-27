---
title: CLI Configuration
description: Configure the Fluxbase CLI
---

The Fluxbase CLI stores its configuration in a YAML file.

## Configuration File Location

| Platform | Path |
|----------|------|
| macOS / Linux | `~/.fluxbase/config.yaml` |
| Windows | `%USERPROFILE%\.fluxbase\config.yaml` |

You can override this with the `--config` flag or `FLUXBASE_CONFIG` environment variable.

## Configuration File Structure

```yaml
version: "1"
current_profile: "dev"

profiles:
  dev:
    name: "dev"
    server: "http://localhost:8080"
    credential_store: "file"
    credentials:
      access_token: "eyJ..."
      refresh_token: "eyJ..."
      expires_at: 1234567890
    user:
      id: "uuid"
      email: "user@example.com"
      role: "admin"
    default_namespace: "default"
    output_format: "table"

  prod:
    name: "prod"
    server: "https://api.example.com"
    credential_store: "keychain"
    default_namespace: "production"

defaults:
  output: "table"
  no_headers: false
  quiet: false
  namespace: "default"
```

## Configuration Options

### Profiles

Each profile contains:

| Field | Description |
|-------|-------------|
| `name` | Profile identifier |
| `server` | Fluxbase server URL |
| `credential_store` | Where credentials are stored: `file` or `keychain` |
| `credentials` | Authentication tokens (when using file storage) |
| `user` | Cached user information |
| `default_namespace` | Default namespace for functions/jobs |
| `output_format` | Default output format for this profile |

### Defaults

Global defaults that apply to all profiles:

| Field | Description |
|-------|-------------|
| `output` | Default output format: `table`, `json`, `yaml` |
| `no_headers` | Hide table headers by default |
| `quiet` | Enable quiet mode by default |
| `namespace` | Default namespace |

## Environment Variables

Environment variables override configuration file settings:

| Variable | Description |
|----------|-------------|
| `FLUXBASE_SERVER` | Server URL (overrides profile) |
| `FLUXBASE_TOKEN` | API token (overrides credentials) |
| `FLUXBASE_PROFILE` | Profile to use (overrides `current_profile`) |
| `FLUXBASE_CONFIG` | Path to config file (overrides default location) |
| `FLUXBASE_DEBUG` | Set to `true` to enable debug output |

### CI/CD Example

```bash
export FLUXBASE_SERVER="https://api.example.com"
export FLUXBASE_TOKEN="your-api-token"

# Commands will use these credentials
fluxbase sync --namespace production
```

## Credential Storage

### File Storage (Default)

Credentials are stored in the configuration file with `0600` permissions (owner read/write only).

### System Keychain

For enhanced security, use the system keychain:

```bash
fluxbase auth login --use-keychain
```

This stores credentials in:
- **macOS**: Keychain Access
- **Windows**: Windows Credential Manager
- **Linux**: Secret Service (requires gnome-keyring or similar)

When using keychain storage, only minimal metadata is stored in the config file.

## Managing Configuration

### Initialize Configuration

```bash
fluxbase config init
```

### View Configuration

```bash
# View full configuration (credentials masked)
fluxbase config view

# View as JSON
fluxbase config view -o json
```

### Set Configuration Values

```bash
# Set default output format
fluxbase config set defaults.output json

# Set default namespace
fluxbase config set defaults.namespace production

# Switch current profile
fluxbase config set current_profile prod
```

### Get Configuration Values

```bash
fluxbase config get defaults.output
fluxbase config get current_profile
```

## Profile Management

### List Profiles

```bash
fluxbase config profiles
```

### Add a Profile

```bash
# Add empty profile
fluxbase config profiles add staging

# Then configure it
fluxbase auth login --profile staging --server https://staging.example.com
```

### Remove a Profile

```bash
fluxbase config profiles remove staging
```

### Switch Profiles

```bash
fluxbase auth switch prod
```

## Security Best Practices

1. **Use keychain storage** for production credentials
2. **Use environment variables** in CI/CD pipelines
3. **Create separate profiles** for different environments
4. **Never commit** the config file to version control
5. **Use API tokens** instead of passwords when possible

## Troubleshooting

### Reset Configuration

```bash
# macOS / Linux
rm -rf ~/.fluxbase
fluxbase auth login

# Windows (PowerShell)
Remove-Item -Recurse -Force "$env:USERPROFILE\.fluxbase"
fluxbase auth login
```

### Debug Mode

Enable verbose output to diagnose issues:

```bash
# Using flag
fluxbase --debug functions list

# Using environment variable
export FLUXBASE_DEBUG=true
fluxbase functions list
```

Debug mode shows:
- HTTP request/response details
- Authentication flow
- Configuration loading

### Check Credential Status

```bash
# Show all profiles and their status
fluxbase auth status

# Show current user info
fluxbase auth whoami
```

### Token Expiration

If you see authentication errors, your tokens may have expired:

```bash
# Re-authenticate
fluxbase auth login

# Or use a fresh API token
fluxbase auth login --server URL --token NEW_TOKEN
```

### Configuration Not Loading

1. Check the config file exists:
   ```bash
   cat ~/.fluxbase/config.yaml
   ```

2. Verify file permissions (should be readable by your user):
   ```bash
   ls -la ~/.fluxbase/config.yaml
   ```

3. Try initializing a fresh config:
   ```bash
   fluxbase config init
   ```
