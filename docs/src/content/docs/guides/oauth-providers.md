---
title: "OAuth Providers"
---

Fluxbase supports OAuth 2.0 and OpenID Connect (OIDC) authentication with multiple providers. Integrate with identity providers like Google, GitHub, Microsoft, Apple, and any OIDC-compliant provider.

## Overview

OAuth authentication enables:

- **Social login** - Users sign in with existing accounts (Google, GitHub, etc.)
- **Enterprise SSO** - Integrate with corporate identity providers
- **Custom OIDC** - Connect any OIDC-compliant provider (Keycloak, Auth0, Authelia)
- **Automatic provisioning** - Create users on first login

## Supported Providers

Fluxbase includes built-in support for well-known providers and custom OIDC providers:

| Provider | Type | Auto-Discovery |
|----------|------|----------------|
| Google | Well-known | Yes |
| GitHub | Well-known | Yes |
| Microsoft/Azure AD | Well-known | Yes |
| Apple | Well-known | Yes |
| Facebook | Well-known | Yes |
| Twitter | Well-known | Yes |
| LinkedIn | Well-known | Yes |
| GitLab | Well-known | Yes |
| Bitbucket | Well-known | Yes |
| Keycloak | Custom OIDC | Requires `issuer_url` |
| Auth0 | Custom OIDC | Requires `issuer_url` |
| Authelia | Custom OIDC | Requires `issuer_url` |
| Any OIDC Provider | Custom OIDC | Requires `issuer_url` |

## Configuration

### YAML Configuration

Configure OAuth providers in your `fluxbase.yaml`:

```yaml
auth:
  oauth_providers:
    # Well-known providers (issuer URL auto-detected)
    - name: google
      enabled: true
      client_id: "YOUR_GOOGLE_CLIENT_ID.apps.googleusercontent.com"
      client_secret: "YOUR_CLIENT_SECRET"
      scopes: [openid, email, profile]
      display_name: "Google"

    - name: github
      enabled: true
      client_id: "YOUR_GITHUB_CLIENT_ID"
      client_secret: "YOUR_CLIENT_SECRET"
      scopes: [user:email]
      display_name: "GitHub"

    - name: microsoft
      enabled: true
      client_id: "YOUR_AZURE_AD_CLIENT_ID"
      client_secret: "YOUR_CLIENT_SECRET"
      scopes: [openid, email, profile]
      display_name: "Microsoft"

    - name: apple
      enabled: true
      client_id: "com.yourapp.service"
      client_secret: "YOUR_APPLE_SECRET"
      scopes: [openid, email, name]
      display_name: "Apple"

    # Custom OIDC providers (issuer_url required)
    - name: keycloak
      enabled: true
      issuer_url: "https://auth.example.com/realms/main"
      client_id: "fluxbase-client"
      client_secret: "YOUR_CLIENT_SECRET"
      scopes: [openid, email, profile]
      display_name: "Corporate SSO"

    - name: auth0
      enabled: true
      issuer_url: "https://your-tenant.auth0.com"
      client_id: "YOUR_AUTH0_CLIENT_ID"
      client_secret: "YOUR_CLIENT_SECRET"
      scopes: [openid, email, profile]
      display_name: "Auth0"
```

### Configuration Options

| Option | Description | Required |
|--------|-------------|----------|
| `name` | Provider identifier (lowercase, e.g., "google", "keycloak") | Yes |
| `enabled` | Enable this provider | Yes |
| `client_id` | OAuth client ID from the provider | Yes |
| `client_secret` | OAuth client secret | Yes (except Apple) |
| `issuer_url` | OIDC issuer URL for discovery | Required for custom providers |
| `scopes` | OAuth scopes to request | No (defaults provided) |
| `display_name` | Human-friendly name for UI | No |

**Additional options (via Admin API or database):**

| Option | Description | Default |
|--------|-------------|---------|
| `authorization_url` | Custom authorization endpoint | Auto-discovered |
| `token_url` | Custom token endpoint | Auto-discovered |
| `user_info_url` | Custom userinfo endpoint | Auto-discovered |
| `allow_dashboard_login` | Allow for admin dashboard SSO | false |
| `allow_app_login` | Allow for app user authentication | true |

### Environment Variables

For deployment, use environment variables with the pattern `FLUXBASE_AUTH_OAUTH_PROVIDERS_*`:

```bash
# First provider (index 0)
FLUXBASE_AUTH_OAUTH_PROVIDERS_0_NAME=google
FLUXBASE_AUTH_OAUTH_PROVIDERS_0_ENABLED=true
FLUXBASE_AUTH_OAUTH_PROVIDERS_0_CLIENT_ID=your-google-client-id
FLUXBASE_AUTH_OAUTH_PROVIDERS_0_CLIENT_SECRET=your-google-secret
FLUXBASE_AUTH_OAUTH_PROVIDERS_0_SCOPES=openid,email,profile
FLUXBASE_AUTH_OAUTH_PROVIDERS_0_DISPLAY_NAME=Google

# Second provider (index 1)
FLUXBASE_AUTH_OAUTH_PROVIDERS_1_NAME=github
FLUXBASE_AUTH_OAUTH_PROVIDERS_1_ENABLED=true
FLUXBASE_AUTH_OAUTH_PROVIDERS_1_CLIENT_ID=your-github-client-id
FLUXBASE_AUTH_OAUTH_PROVIDERS_1_CLIENT_SECRET=your-github-secret
FLUXBASE_AUTH_OAUTH_PROVIDERS_1_SCOPES=user:email

# Custom OIDC provider (index 2)
FLUXBASE_AUTH_OAUTH_PROVIDERS_2_NAME=keycloak
FLUXBASE_AUTH_OAUTH_PROVIDERS_2_ENABLED=true
FLUXBASE_AUTH_OAUTH_PROVIDERS_2_ISSUER_URL=https://auth.example.com/realms/main
FLUXBASE_AUTH_OAUTH_PROVIDERS_2_CLIENT_ID=fluxbase-client
FLUXBASE_AUTH_OAUTH_PROVIDERS_2_CLIENT_SECRET=your-keycloak-secret
```

## Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/auth/oauth/providers` | GET | List available OAuth providers (public) |
| `/api/v1/auth/oauth/:provider/authorize` | GET | Initiate OAuth flow (redirects to provider) |
| `/api/v1/auth/oauth/:provider/callback` | GET | OAuth callback handler |
| `/api/v1/admin/oauth/providers` | GET | List all providers (admin) |
| `/api/v1/admin/oauth/providers` | POST | Create new provider (admin) |
| `/api/v1/admin/oauth/providers/:id` | PATCH | Update provider (admin) |
| `/api/v1/admin/oauth/providers/:id` | DELETE | Delete provider (admin) |

## Setup Guide

### Step 1: Configure Fluxbase

Add your OAuth provider configuration:

```yaml
auth:
  oauth_providers:
    - name: google
      enabled: true
      client_id: "YOUR_CLIENT_ID"
      client_secret: "YOUR_CLIENT_SECRET"
      scopes: [openid, email, profile]
```

### Step 2: Configure Your Provider

Register your application with the OAuth provider and set the redirect URI:

```
https://your-domain.com/api/v1/auth/oauth/{provider}/callback
```

For example:
- Google: `https://your-domain.com/api/v1/auth/oauth/google/callback`
- GitHub: `https://your-domain.com/api/v1/auth/oauth/github/callback`

### Step 3: Test Login

```bash
# Open in browser to initiate OAuth flow
open "https://your-domain.com/api/v1/auth/oauth/google/authorize?redirect_to=https://your-domain.com/dashboard"
```

## Provider-Specific Setup

### Google

1. Go to [Google Cloud Console](https://console.cloud.google.com/) → **APIs & Services** → **Credentials**
2. Click **Create Credentials** → **OAuth 2.0 Client ID**
3. Select **Web application**
4. Add authorized redirect URI: `https://your-domain.com/api/v1/auth/oauth/google/callback`
5. Copy the Client ID and Client Secret

```yaml
auth:
  oauth_providers:
    - name: google
      enabled: true
      client_id: "YOUR_CLIENT_ID.apps.googleusercontent.com"
      client_secret: "YOUR_CLIENT_SECRET"
      scopes: [openid, email, profile]
```

### GitHub

1. Go to [GitHub Developer Settings](https://github.com/settings/developers) → **OAuth Apps** → **New OAuth App**
2. Set **Authorization callback URL**: `https://your-domain.com/api/v1/auth/oauth/github/callback`
3. Copy the Client ID and generate a Client Secret

```yaml
auth:
  oauth_providers:
    - name: github
      enabled: true
      client_id: "YOUR_CLIENT_ID"
      client_secret: "YOUR_CLIENT_SECRET"
      scopes: [user:email]
```

### Microsoft/Azure AD

1. Go to [Azure Portal](https://portal.azure.com/) → **Azure Active Directory** → **App registrations**
2. Click **New registration**
3. Add redirect URI: `https://your-domain.com/api/v1/auth/oauth/microsoft/callback`
4. Go to **Certificates & secrets** → **New client secret**

```yaml
auth:
  oauth_providers:
    - name: microsoft
      enabled: true
      client_id: "YOUR_APPLICATION_CLIENT_ID"
      client_secret: "YOUR_CLIENT_SECRET"
      scopes: [openid, email, profile]
```

### Apple

1. Go to [Apple Developer Portal](https://developer.apple.com/) → **Certificates, Identifiers & Profiles**
2. Create a **Services ID** with Sign in with Apple enabled
3. Configure the return URL: `https://your-domain.com/api/v1/auth/oauth/apple/callback`
4. Create a **Key** for Sign in with Apple

```yaml
auth:
  oauth_providers:
    - name: apple
      enabled: true
      client_id: "com.yourapp.service"
      client_secret: "YOUR_APPLE_SECRET"  # Generated JWT
      scopes: [openid, email, name]
```

### Custom OIDC Providers

For Keycloak, Auth0, Authelia, or any OIDC-compliant provider:

**Keycloak:**

```yaml
auth:
  oauth_providers:
    - name: keycloak
      enabled: true
      issuer_url: "https://keycloak.example.com/realms/myrealm"
      client_id: "fluxbase"
      client_secret: "YOUR_CLIENT_SECRET"
      scopes: [openid, email, profile]
      display_name: "Corporate SSO"
```

**Auth0:**

```yaml
auth:
  oauth_providers:
    - name: auth0
      enabled: true
      issuer_url: "https://your-tenant.auth0.com"
      client_id: "YOUR_AUTH0_CLIENT_ID"
      client_secret: "YOUR_AUTH0_CLIENT_SECRET"
      scopes: [openid, email, profile]
      display_name: "Auth0"
```

**Authelia:**

```yaml
auth:
  oauth_providers:
    - name: authelia
      enabled: true
      issuer_url: "https://auth.example.com"
      client_id: "fluxbase"
      client_secret: "YOUR_AUTHELIA_SECRET"
      scopes: [openid, email, profile]
      display_name: "Authelia"
```

## SDK Usage

### TypeScript SDK

**Initiate OAuth flow:**

```typescript
import { FluxbaseClient } from '@fluxbase/sdk'

const client = new FluxbaseClient({ url: 'https://api.example.com' })

// Get available OAuth providers
const { data: providers } = await client.auth.getOAuthProviders()
// [{ name: 'google', display_name: 'Google' }, ...]

// Get OAuth authorization URL
const { data } = await client.auth.signInWithOAuth({
  provider: 'google',
  options: {
    redirectTo: `${window.location.origin}/auth/callback`,
  },
})

// Redirect to provider
window.location.href = data.url
```

**Handle callback:**

```typescript
// In your /auth/callback route
const code = searchParams.get('code')
const state = searchParams.get('state')

const { user, session } = await client.auth.exchangeCodeForSession({
  code,
  state,
})

// User is now authenticated
console.log('Logged in as:', user.email)
```

### React SDK

```tsx
import {
  useOAuthProviders,
  useSignInWithOAuth,
  useSession
} from '@fluxbase/sdk-react'

function OAuthLoginButtons() {
  const { data: providers, isLoading } = useOAuthProviders()
  const signIn = useSignInWithOAuth()

  if (isLoading) return <div>Loading...</div>

  return (
    <div>
      {providers?.map(provider => (
        <button
          key={provider.name}
          onClick={() => signIn.mutate({
            provider: provider.name,
            redirectTo: '/dashboard'
          })}
        >
          Sign in with {provider.display_name || provider.name}
        </button>
      ))}
    </div>
  )
}
```

**Handling callback:**

```tsx
// pages/auth/callback.tsx
import { useEffect } from 'react'
import { useSession } from '@fluxbase/sdk-react'
import { useNavigate } from 'react-router-dom'

function OAuthCallback() {
  const { data: session, isLoading } = useSession()
  const navigate = useNavigate()

  useEffect(() => {
    if (!isLoading && session) {
      navigate('/dashboard')
    }
  }, [session, isLoading])

  return <div>Completing sign in...</div>
}
```

## REST API

### List Providers (Public)

```bash
GET /api/v1/auth/oauth/providers
```

Response:
```json
{
  "providers": [
    {
      "name": "google",
      "display_name": "Google",
      "enabled": true
    },
    {
      "name": "github",
      "display_name": "GitHub",
      "enabled": true
    }
  ]
}
```

### Initiate OAuth Login

```bash
GET /api/v1/auth/oauth/:provider/authorize?redirect_to=https://myapp.example.com/dashboard
```

Redirects to provider's authorization page.

### List All Providers (Admin)

```bash
GET /api/v1/admin/oauth/providers
Authorization: Bearer <admin-token>
```

Response:
```json
{
  "providers": [
    {
      "id": "uuid",
      "provider_name": "google",
      "display_name": "Google",
      "client_id": "xxx.apps.googleusercontent.com",
      "enabled": true,
      "scopes": ["openid", "email", "profile"],
      "allow_dashboard_login": false,
      "allow_app_login": true,
      "created_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

### Create Provider (Admin)

```bash
POST /api/v1/admin/oauth/providers
Authorization: Bearer <admin-token>
Content-Type: application/json

{
  "provider_name": "okta",
  "display_name": "Okta SSO",
  "enabled": true,
  "client_id": "YOUR_OKTA_CLIENT_ID",
  "client_secret": "YOUR_OKTA_SECRET",
  "is_custom": true,
  "authorization_url": "https://your-org.okta.com/oauth2/v1/authorize",
  "token_url": "https://your-org.okta.com/oauth2/v1/token",
  "user_info_url": "https://your-org.okta.com/oauth2/v1/userinfo",
  "scopes": ["openid", "email", "profile"],
  "allow_dashboard_login": true,
  "allow_app_login": true
}
```

### Update Provider (Admin)

```bash
PATCH /api/v1/admin/oauth/providers/:id
Authorization: Bearer <admin-token>
Content-Type: application/json

{
  "enabled": false,
  "display_name": "Updated Name"
}
```

### Delete Provider (Admin)

```bash
DELETE /api/v1/admin/oauth/providers/:id
Authorization: Bearer <admin-token>
```

## Security

### Token Encryption

OAuth tokens are encrypted at rest using AES-256-GCM. Configure an encryption key:

```bash
# Must be exactly 32 bytes
FLUXBASE_ENCRYPTION_KEY=your-32-byte-encryption-key-here
```

### CSRF Protection

Fluxbase automatically generates and validates state tokens to prevent CSRF attacks:

- State tokens expire after 10 minutes
- Tokens are cryptographically random
- Validation is enforced on callback

### Best Practices

| Practice | Description |
|----------|-------------|
| **Use HTTPS in production** | OAuth requires secure connections |
| **Protect client secrets** | Store in environment variables, never in code |
| **Use PKCE** | Fluxbase supports PKCE for enhanced security |
| **Limit scopes** | Only request necessary permissions |
| **Rotate secrets** | Periodically rotate client secrets |
| **Monitor OAuth usage** | Watch for unusual patterns and failed attempts |
| **Validate redirect URIs** | Use exact match, avoid wildcards |

## Role-Based Access Control (RBAC)

Control which users can authenticate based on claims in their OAuth ID token (roles, groups, permissions, etc.).

### Claims-Based Access Control

Restrict OAuth authentication based on ID token claims:

```yaml
auth:
  oauth_providers:
    - name: google
      enabled: true
      client_id: "YOUR_CLIENT_ID"
      client_secret: "YOUR_CLIENT_SECRET"
      scopes: [openid, email, profile]
      allow_dashboard_login: true

      # RBAC configuration
      required_claims:
        roles:
          - "admin"           # User must have at least ONE of these role values
          - "editor"
        department:
          - "IT"
          - "Engineering"

      denied_claims:
        status:
          - "suspended"       # Reject users with ANY of these status values
          - "inactive"
```

### RBAC Rules

Two types of claim validation rules:

| Rule | Logic | Example Use Case |
| --- | --- | --- |
| `required_claims` | User must have at least ONE matching value per claim | Require admin OR editor role |
| `denied_claims` | Reject if ANY value matches | Block suspended or inactive users |

**Execution order**: Denied claims are checked first (highest priority), then required claims.

### Claim Value Types

OAuth claims can be strings or arrays. The validation handles both:

```json
{
  "roles": "admin",           // Single string value
  "groups": ["admins", "IT"], // Array of strings
  "level": 5                  // Number (converted to string)
}
```

### Example Configurations

**Dashboard admin access (Google Workspace)**

Only users with admin role can access the dashboard:

```yaml
auth:
  oauth_providers:
    - name: google
      enabled: true
      client_id: "YOUR_CLIENT_ID.apps.googleusercontent.com"
      client_secret: "YOUR_CLIENT_SECRET"
      scopes: [openid, email, profile]
      allow_dashboard_login: true
      allow_app_login: false
      required_claims:
        hd: ["company.com"]  # Google Workspace domain
        role: ["admin", "superuser"]
```

**Azure AD group-based access**

Restrict access based on Azure AD group membership:

```yaml
auth:
  oauth_providers:
    - name: microsoft
      enabled: true
      client_id: "YOUR_AZURE_CLIENT_ID"
      client_secret: "YOUR_CLIENT_SECRET"
      scopes: [openid, email, profile]
      allow_dashboard_login: true
      required_claims:
        groups:
          - "FluxbaseAdmins"       # Azure AD group name
          - "ApplicationAdmins"
      denied_claims:
        groups:
          - "Contractors"
```

**Auth0 with custom claims**

Use Auth0 app_metadata or custom claims:

```yaml
auth:
  oauth_providers:
    - name: auth0
      enabled: true
      issuer_url: "https://your-tenant.auth0.com"
      client_id: "YOUR_CLIENT_ID"
      client_secret: "YOUR_CLIENT_SECRET"
      scopes: [openid, email, profile]
      allow_dashboard_login: true
      required_claims:
        "https://myapp.com/roles":
          - "admin"
        "https://myapp.com/subscription":
          - "premium"
          - "enterprise"
```

**Keycloak realm roles**

Filter by Keycloak realm or client roles:

```yaml
auth:
  oauth_providers:
    - name: keycloak
      enabled: true
      issuer_url: "https://auth.company.com/realms/main"
      client_id: "fluxbase-client"
      client_secret: "YOUR_CLIENT_SECRET"
      scopes: [openid, email, profile, roles]
      allow_dashboard_login: true
      required_claims:
        realm_access.roles:
          - "fluxbase-admin"
      denied_claims:
        account_status:
          - "locked"
          - "suspended"
```

### Error Messages

When claim validation fails, users see clear error messages:

- `"Access denied: claim 'roles' has restricted value 'contractor'"` - User has denied claim value
- `"Access denied: missing required claim 'roles'"` - Required claim not in ID token
- `"Access denied: claim 'roles' must have one of: [admin, editor]"` - User doesn't have any allowed values

### ID Token vs Userinfo

Fluxbase validates claims from the OAuth ID token (JWT). The ID token is returned during the token exchange and contains user identity and claims.

For custom claims:
1. Configure your IdP to include claims in ID token (not just userinfo endpoint)
2. Request appropriate scopes to receive the claims
3. Some providers require custom scopes for custom claims

### Troubleshooting RBAC

**"Claims not being found"**

Check the ID token structure from your provider:

```bash
# Use jwt.io to decode your ID token and inspect claims
# Or check browser dev tools Network tab during OAuth callback
```

**"Azure AD groups not working"**

Azure AD requires specific configuration to include groups in ID token:

1. In Azure Portal, go to **App registrations** → Your app
2. Go to **Token configuration**
3. Click **Add groups claim**
4. Select **Security groups** and **Group ID** (or **sAMAccountName** for names)
5. Check **ID** under token type

**"Keycloak roles not in token"**

Enable role claims in Keycloak:

1. Go to **Clients** → Your client → **Mappers**
2. Add **realm roles** mapper
3. Set **Token Claim Name** to match your config (e.g., `realm_access.roles`)
4. Enable **Add to ID token**

**"Auth0 custom claims not appearing"**

Auth0 custom claims must use namespaced format:

```javascript
// Auth0 Rule/Action to add custom claims
function(user, context, callback) {
  const namespace = 'https://myapp.com/';
  context.idToken[namespace + 'roles'] = user.app_metadata.roles;
  callback(null, user, context);
}
```

**"Claim values are case-sensitive"**

Claim values are matched exactly:
- `"Admin"` ≠ `"admin"`
- `"IT-Team"` ≠ `"IT Team"`

Check exact values from your IdP.

## Dashboard SSO (Admin Login)

OAuth providers can be used for dashboard admin authentication, enabling SSO-only mode.

### Enable Dashboard SSO

Configure a provider for dashboard login:

```yaml
auth:
  oauth_providers:
    - name: google
      enabled: true
      client_id: "YOUR_CLIENT_ID"
      client_secret: "YOUR_CLIENT_SECRET"
      scopes: [openid, email, profile]
```

Then via Admin API, enable dashboard login:

```bash
PATCH /api/v1/admin/oauth/providers/:id
{
  "allow_dashboard_login": true
}
```

### Disable Password Login

Once SSO is configured for dashboard login:

1. Go to **Authentication** → **Auth Settings** in the dashboard
2. Enable **Disable Password Login**
3. Save settings

When enabled:
- Login page shows only SSO buttons
- Password form is hidden
- Backend rejects password attempts

### CLI SSO Login

With password login disabled, use SSO for CLI authentication:

```bash
# SSO login (opens browser)
fluxbase auth login --server https://api.example.com --sso

# Or use an API token
fluxbase auth login --server https://api.example.com --token your-api-token
```

### Emergency Recovery

If locked out due to misconfigured SSO:

```bash
FLUXBASE_DASHBOARD_FORCE_PASSWORD_LOGIN=true
```

This temporarily re-enables password login.

## Linking Accounts

### Link OAuth to Existing Account

```typescript
// User must be authenticated
const { url } = await client.auth.linkIdentity({ provider: 'github' })
window.location.href = url
```

### Unlink OAuth Provider

```typescript
await client.auth.unlinkIdentity({ provider: 'github' })
```

## Troubleshooting

| Issue | Solution |
|-------|----------|
| **Redirect URI mismatch** | Ensure redirect URI in Fluxbase config exactly matches provider registration (include protocol, no trailing slashes) |
| **Invalid state parameter** | Enable cookies (state stored in cookie), verify cross-site cookie settings |
| **Client secret invalid** | Verify secret hasn't expired, regenerate if needed |
| **Users can't sign in** | Check provider returns email, verify scopes include necessary permissions |
| **Custom provider not working** | Verify `issuer_url` is accessible, check OIDC discovery endpoint |
| **Token encryption errors** | Ensure `FLUXBASE_ENCRYPTION_KEY` is exactly 32 bytes |
| **CORS errors** | Configure allowed origins in Fluxbase CORS settings |
| **Dev vs prod URLs** | Use environment variables for redirect URLs |

### Debug Logging

Enable debug logging to troubleshoot OAuth:

```yaml
debug: true
```

Check logs for:
- OAuth request/response details
- Token exchange errors
- User info extraction

## Next Steps

- [Authentication](/docs/guides/authentication) - Authentication overview
- [SAML SSO](/docs/guides/saml-sso) - Enterprise SAML authentication
- [Row-Level Security](/docs/guides/row-level-security) - Data access control
