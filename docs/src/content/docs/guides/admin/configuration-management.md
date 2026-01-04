---
title: Configuration Management
---

Fluxbase settings can be configured through environment variables or the admin UI. Understanding how these interact is crucial for proper system administration.

## Configuration Precedence

When a setting is configured through both methods, **environment variables always take precedence**:

```
Environment Variables > UI Settings > Default Values
```

This ensures infrastructure-as-code practices and prevents accidental changes to critical settings in production.

## Visual Indicators

The admin UI provides clear feedback about configuration sources:

- **Lock Badge** (🔒 Environment Variable) - Setting is controlled by an environment variable
- **Disabled Input/Switch** - Setting cannot be changed through the UI
- **Env Var Name** - Shows which environment variable controls the setting (e.g., `FLUXBASE_AUTH_SIGNUP_ENABLED`)

When you try to update an overridden setting, you'll receive an error: _"This setting is controlled by an environment variable and cannot be changed"_

## Setting Categories

### ENV-ONLY Settings

These **must** be configured via environment variables (cannot be set in UI):

- Database connection (host, port, credentials)
- SMTP credentials (password, client keys)
- JWT secrets
- Storage provider credentials

**Why**: Security best practices - credentials should not be stored in the database.

### HYBRID Settings

Can be configured via **either** environment variables **or** UI:

- Feature flags (`enable_realtime`, `enable_storage`, `enable_functions`)
- Email service (`enabled`, `provider`)
- Authentication (`enable_signup`, `enable_magic_link`, `password_min_length`)

**Use case**: Lock in production via env vars, allow changes in development via UI.

### DB-ONLY Settings

Only configurable through the UI (no env var option):

- OAuth provider configurations
- Email templates
- Webhook configurations

## Environment Variable Format

Fluxbase uses a consistent naming convention:

```
FLUXBASE_<CATEGORY>_<SETTING>
```

Examples:

- `app.auth.enable_signup` → `FLUXBASE_AUTH_SIGNUP_ENABLED`
- `app.realtime.enabled` → `FLUXBASE_REALTIME_ENABLED`
- `app.email.enabled` → `FLUXBASE_EMAIL_ENABLED`
- `app.email.provider` → `FLUXBASE_EMAIL_PROVIDER`

**Rule**: Remove `app.` prefix, convert to uppercase, replace dots with underscores. Feature enabled flags follow the pattern `app.<feature>.enabled` → `FLUXBASE_<FEATURE>_ENABLED`.

## Common Scenarios

### Lock Settings in Production

Set environment variables to enforce configuration:

```bash
# docker-compose.yml or .env
FLUXBASE_FEATURES_REALTIME_ENABLED=true
FLUXBASE_AUTH_SIGNUP_ENABLED=false
FLUXBASE_EMAIL_PROVIDER=sendgrid
```

Admin UI will show these as locked and display the controlling env var.

### Allow Runtime Changes in Development

Don't set env vars for settings you want to control via UI:

```bash
# Only set credentials, leave feature flags unset
FLUXBASE_EMAIL_SMTP_HOST=mailhog
FLUXBASE_EMAIL_SMTP_PORT=1025
# FLUXBASE_AUTH_SIGNUP_ENABLED not set → editable in UI
```

### Mixed Approach

Lock critical settings, allow non-critical ones:

```bash
# Lock security settings
FLUXBASE_AUTH_REQUIRE_EMAIL_VERIFICATION=true

# Allow feature toggles in UI
# (no env vars for features)
```

## Troubleshooting

### "Why can't I change this setting?"

1. Check for lock badge (🔒) next to the setting
2. Note the displayed environment variable name
3. Check your `.env` file or docker compose configuration
4. Remove or comment out the env var to enable UI control

### Check Override Status

Settings with environment variables will show:

- Lock badge in UI
- Env var name below the input
- Disabled state (grayed out)

### Remove an Override

To allow UI control of a setting:

1. Remove the environment variable from your configuration
2. Restart the Fluxbase server
3. The setting will become editable in the admin UI

## Best Practices

1. **Production**: Use env vars for all settings to ensure consistency
2. **Development**: Use UI for quick testing, env vars for team-shared config
3. **Security**: Always use env vars for credentials (SMTP passwords, client keys, JWT secrets)
4. **Documentation**: Document which env vars your deployment uses

## See Also

- [Email Services Configuration](/docs/guides/email-services)
- [Deployment Guide](/docs/deployment/overview)
- [Configuration Reference](/docs/reference/configuration)
