# OAuth Providers

Fluxbase supports OAuth 2.0 authentication with multiple providers.

## Supported Providers

Google, GitHub, Microsoft/Azure AD, Apple, Facebook, Twitter, LinkedIn, GitLab, Bitbucket, Custom OAuth 2.0

## Implementation

**Initiate OAuth flow:**

```typescript
const { url } = await fluxbase.auth.signInWithOAuth({
  provider: 'google',
  options: {
    redirectTo: `${window.location.origin}/auth/callback`,
  },
});

window.location.href = url;
```

**Handle callback:**

```typescript
// In your /auth/callback route
const code = searchParams.get('code');
const state = searchParams.get('state');

const { user, session } = await fluxbase.auth.exchangeCodeForSession({
  code,
  state,
});

// Store session and redirect to dashboard
```

## Provider Configuration

| Provider | Setup Steps | Redirect URI | Required Scopes |
|----------|-------------|--------------|-----------------|
| **Google** | [Cloud Console](https://console.cloud.google.com/) → Credentials → OAuth 2.0 Client ID | `http://your-domain.com/api/v1/auth/callback/google` | openid, email, profile |
| **GitHub** | [Developer Settings](https://github.com/settings/developers) → New OAuth App | `http://your-domain.com/api/v1/auth/callback/github` | user:email |
| **Microsoft** | [Azure Portal](https://portal.azure.com/) → Azure AD → App registrations | `http://your-domain.com/api/v1/auth/callback/microsoft` | openid, email, profile |

**Configuration example (Google):**

```yaml
auth:
  oauth:
    google:
      client_id: "YOUR_GOOGLE_CLIENT_ID"
      client_secret: "YOUR_GOOGLE_CLIENT_SECRET"
      redirect_url: "http://localhost:8080/api/v1/auth/callback/google"
      scopes: ["openid", "email", "profile"]
```

**Custom OAuth provider:**

```yaml
auth:
  oauth:
    custom:
      my_provider:
        name: "My Custom Provider"
        authorization_url: "https://provider.com/oauth/authorize"
        token_url: "https://provider.com/oauth/token"
        userinfo_url: "https://provider.com/oauth/userinfo"
        client_id: "YOUR_CLIENT_ID"
        client_secret: "YOUR_CLIENT_SECRET"
        redirect_url: "http://localhost:8080/api/v1/auth/callback/my_provider"
        scopes: ["openid", "email", "profile"]
```

## Advanced Features

**Link OAuth to existing account:**

```typescript
const { url } = await fluxbase.auth.linkIdentity({ provider: 'google' });
window.location.href = url;
```

**Unlink OAuth provider:**

```typescript
await fluxbase.auth.unlinkIdentity({ provider: 'google' });
```

## Troubleshooting

| Issue | Solution |
|-------|----------|
| **Redirect URI mismatch** | Ensure redirect URI in Fluxbase config exactly matches provider registration (include protocol, no trailing slashes unless required) |
| **Invalid state parameter** | Enable cookies (state stored in cookie), verify cross-site cookie settings, check state parameter passed correctly |
| **Users can't sign in** | Check provider returns email, verify scopes include necessary permissions, check Fluxbase logs, confirm credentials correct |
| **Dev vs prod URLs** | Use environment variables for redirect URLs based on `NODE_ENV` |

## Best Practices

| Practice | Description |
|----------|-------------|
| **Use HTTPS in production** | OAuth requires secure connections |
| **Validate state parameter** | Fluxbase does this automatically to prevent CSRF |
| **Protect client secrets** | Keep them server-side only, never expose in client code |
| **Use PKCE** | Fluxbase implements PKCE for enhanced security |
| **Limit scopes** | Only request necessary permissions |
| **Monitor OAuth usage** | Watch for unusual patterns and failed attempts |
