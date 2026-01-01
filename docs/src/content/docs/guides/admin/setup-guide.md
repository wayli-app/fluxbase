---
title: "Initial Setup Guide"
description: "Complete guide to setting up your Fluxbase admin account"
---

This guide walks you through the initial setup process for a new Fluxbase installation, including generating secrets and creating your first admin account.

## Prerequisites

Before starting, ensure you have:

- Fluxbase running (via Docker, Kubernetes, or binary)
- Access to the server's environment configuration
- A web browser

## Step 1: Generate Secrets

Fluxbase requires several secrets for secure operation. The easiest way to generate them is using the provided script:

```bash
cd deploy
./generate-keys.sh
```

The script generates:

| Secret                          | Purpose                                                           |
| ------------------------------- | ----------------------------------------------------------------- |
| `FLUXBASE_AUTH_JWT_SECRET`      | Signs authentication tokens                                       |
| `FLUXBASE_ENCRYPTION_KEY`       | Encrypts secrets and OAuth tokens (must be exactly 32 characters) |
| `FLUXBASE_SECURITY_SETUP_TOKEN` | One-time token to access the setup page                           |
| `POSTGRES_PASSWORD`             | Database password                                                 |

:::caution[Setup Token]
You need the setup token **once** when registering the first dashboard user.
:::

### Manual Generation

If you prefer to generate secrets manually:

```bash
# JWT Secret (base64, 32+ bytes)
openssl rand -base64 32

# Encryption Key (exactly 32 characters for AES-256)
openssl rand -base64 32 | head -c 32

# Setup Token
openssl rand -base64 32
```

## Step 2: Configure Environment

Set the secrets as environment variables or in your configuration:

### Docker Compose

The `generate-keys.sh` script creates a `.env` file automatically:

```bash
# .env file (generated)
POSTGRES_PASSWORD=<generated>
FLUXBASE_AUTH_JWT_SECRET=<generated>
FLUXBASE_ENCRYPTION_KEY=<generated>
FLUXBASE_SECURITY_SETUP_TOKEN=<generated>
```

### Kubernetes

Use the generated `fluxbase-secrets.yaml` or create a Secret:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: fluxbase-secrets
type: Opaque
stringData:
  jwt-secret: "<your-jwt-secret>"
  encryption-key: "<your-32-char-key>"
  setup-token: "<your-setup-token>"
```

### Environment Variables

```bash
export FLUXBASE_AUTH_JWT_SECRET="your-jwt-secret"
export FLUXBASE_ENCRYPTION_KEY="your-32-char-encryption-key"
export FLUXBASE_SECURITY_SETUP_TOKEN="your-setup-token"
```

## Step 3: Access the Setup Page

1. Start Fluxbase if not already running:

   ```bash
   docker compose -f docker-compose.minimal.yaml up -d
   ```

2. Open your browser and navigate to:

   ```
   http://localhost:8080/admin/setup
   ```

   Replace `localhost:8080` with your server's address if different.

:::note
The setup page is only accessible when no admin users exist. After creating an admin account, this page will return a 403 Forbidden error.
:::

## Step 4: Create Your Admin Account

On the setup page, you'll see a form with the following fields:

### Setup Token

Enter the `FLUXBASE_SECURITY_SETUP_TOKEN` value from your configuration. This verifies you have authorized access to create the admin account.

If you see "Admin setup is disabled", ensure the `FLUXBASE_SECURITY_SETUP_TOKEN` environment variable is set and the server was restarted.

### Account Details

| Field                | Requirements                         |
| -------------------- | ------------------------------------ |
| **Full Name**        | Minimum 2 characters                 |
| **Email**            | Valid email address (used for login) |
| **Password**         | Minimum 12 characters                |
| **Confirm Password** | Must match password                  |

### Password Requirements

Your admin password must be at least 12 characters long. For best security:

- Use a mix of uppercase, lowercase, numbers, and symbols
- Don't reuse passwords from other services
- Consider using a password manager

## Step 5: Complete Setup

Click **"Complete Setup"** to create your account. On success:

1. Your admin account is created in the `dashboard_auth.users` table
2. A session is created and stored securely
3. The system is marked as "setup complete"
4. You're automatically redirected to the dashboard

## After Setup

### Access the Dashboard

Navigate to `http://localhost:8080/admin` and log in with your email and password.

### Explore Key Features

- **Tables** - Browse and edit your database
- **Users** - Manage application users
- **Functions** - Deploy and monitor edge functions
- **Storage** - Manage file uploads
- **Settings** - Configure your instance

### Create Additional Admin Users

After initial setup, you can invite additional admins through:

1. **Dashboard**: Settings > Team > Invite User
2. **CLI**: `fluxbase admin create-user --email admin@example.com`

### Enable Two-Factor Authentication

For enhanced security, enable 2FA for your admin account:

1. Go to Settings > Security
2. Click "Enable 2FA"
3. Scan the QR code with your authenticator app
4. Enter the verification code

## Troubleshooting

### "Invalid setup token"

- Verify the token matches exactly (no extra spaces)
- Check that `FLUXBASE_SECURITY_SETUP_TOKEN` is set in your environment
- Restart Fluxbase after changing environment variables

### "Setup has already been completed"

The setup page only works once. To reset:

1. Connect to your database
2. Delete the system setting: `DELETE FROM app.settings WHERE key = 'setup_complete';`
3. Delete existing dashboard users: `TRUNCATE dashboard_auth.users CASCADE;`

:::danger
Only do this in development. In production, use the normal login flow or password reset.
:::

### "Admin setup is disabled"

The `FLUXBASE_SECURITY_SETUP_TOKEN` environment variable is not set. Add it to your configuration and restart Fluxbase.

### Setup page shows 404

Ensure Fluxbase is running and the admin UI is enabled. Check the logs:

```bash
docker logs fluxbase
```

## Security Notes

- The setup token should be treated like a password - keep it secret
- After setup, the token is no longer needed and can be rotated
- Consider removing `FLUXBASE_SECURITY_SETUP_TOKEN` from production after initial setup
- Admin credentials are separate from regular user accounts
- All admin actions are logged for audit purposes

## Next Steps

- [Quick Start](/getting-started/quick-start/) - Build your first API
- [Authentication Guide](/guides/authentication/) - Set up user authentication
- [Configuration Reference](/reference/configuration/) - All configuration options
