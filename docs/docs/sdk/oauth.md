---
sidebar_position: 5
---

# OAuth Provider Configuration SDK

The OAuth SDK provides comprehensive tools for managing OAuth providers and authentication settings in Fluxbase. Configure third-party authentication providers (GitHub, Google, GitLab, etc.) and customize authentication behavior including password requirements, session management, and signup controls.

## Overview

The OAuth SDK consists of two main managers:

1. **OAuthProviderManager**: Manage OAuth provider configurations
2. **AuthSettingsManager**: Configure global authentication settings

## Getting Started

```typescript
import { createClient } from '@fluxbase/sdk'

const client = createClient(
  'http://localhost:8080',
  'your-service-role-key'
)

// Authenticate as admin
await client.admin.login({
  email: 'admin@example.com',
  password: 'your-password'
})

// Access OAuth configuration
const oauth = client.admin.oauth
```

## OAuth Provider Manager

### List OAuth Providers

List all configured OAuth providers:

```typescript
const providers = await client.admin.oauth.providers.listProviders()

providers.forEach(provider => {
  console.log(`${provider.display_name}:`, provider.enabled ? 'enabled' : 'disabled')
  console.log('  Provider:', provider.provider_name)
  console.log('  Redirect URL:', provider.redirect_url)
  console.log('  Scopes:', provider.scopes.join(', '))
})
```

### Get Specific Provider

Retrieve detailed information about a specific provider:

```typescript
const provider = await client.admin.oauth.providers.getProvider('provider-uuid')

console.log('Provider:', provider.display_name)
console.log('Status:', provider.enabled ? 'Enabled' : 'Disabled')
console.log('Client ID:', provider.client_id)
console.log('Scopes:', provider.scopes)
console.log('Is Custom:', provider.is_custom)
```

**Note:** Client secrets are never returned in API responses for security reasons.

### Create Built-in OAuth Provider

Configure a built-in OAuth provider like GitHub, Google, or GitLab:

```typescript
const result = await client.admin.oauth.providers.createProvider({
  provider_name: 'github',
  display_name: 'GitHub',
  enabled: true,
  client_id: process.env.GITHUB_CLIENT_ID!,
  client_secret: process.env.GITHUB_CLIENT_SECRET!,
  redirect_url: 'https://yourapp.com/auth/callback',
  scopes: ['user:email', 'read:user'],
  is_custom: false
})

console.log('Provider created:', result.id)
```

#### Built-in Provider Names

Supported built-in providers:
- `github`
- `google`
- `gitlab`
- `microsoft`
- `discord`
- `slack`
- `facebook`
- `twitter`

### Create Custom OAuth2 Provider

Configure a custom OAuth2 provider for enterprise SSO:

```typescript
await client.admin.oauth.providers.createProvider({
  provider_name: 'custom_sso',
  display_name: 'Custom SSO',
  enabled: true,
  client_id: 'your-client-id',
  client_secret: 'your-client-secret',
  redirect_url: 'https://yourapp.com/auth/callback',
  scopes: ['openid', 'profile', 'email'],
  is_custom: true,
  authorization_url: 'https://sso.example.com/oauth/authorize',
  token_url: 'https://sso.example.com/oauth/token',
  user_info_url: 'https://sso.example.com/oauth/userinfo'
})
```

### Update OAuth Provider

Update an existing OAuth provider configuration:

```typescript
// Disable a provider
await client.admin.oauth.providers.updateProvider('provider-id', {
  enabled: false
})

// Update scopes
await client.admin.oauth.providers.updateProvider('provider-id', {
  scopes: ['user:email', 'read:user', 'read:org']
})

// Update redirect URL
await client.admin.oauth.providers.updateProvider('provider-id', {
  redirect_url: 'https://newdomain.com/auth/callback'
})

// Rotate credentials
await client.admin.oauth.providers.updateProvider('provider-id', {
  client_id: 'new-client-id',
  client_secret: 'new-client-secret'
})
```

### Delete OAuth Provider

Permanently delete an OAuth provider:

```typescript
await client.admin.oauth.providers.deleteProvider('provider-id')
console.log('Provider deleted')
```

### Enable/Disable Provider

Convenience methods for toggling provider status:

```typescript
// Enable a provider
await client.admin.oauth.providers.enableProvider('provider-id')

// Disable a provider
await client.admin.oauth.providers.disableProvider('provider-id')
```

## Authentication Settings Manager

### Get Authentication Settings

Retrieve current authentication configuration:

```typescript
const settings = await client.admin.oauth.authSettings.get()

console.log('Signup enabled:', settings.enable_signup)
console.log('Email verification:', settings.require_email_verification)
console.log('Magic link auth:', settings.enable_magic_link)
console.log('Password min length:', settings.password_min_length)
console.log('Session timeout:', settings.session_timeout_minutes, 'minutes')
```

### Update Authentication Settings

Update one or more authentication settings:

```typescript
await client.admin.oauth.authSettings.update({
  // Signup control
  enable_signup: true,

  // Email verification
  require_email_verification: true,
  enable_magic_link: true,

  // Password requirements
  password_min_length: 16,
  password_require_uppercase: true,
  password_require_lowercase: true,
  password_require_number: true,
  password_require_special: true,

  // Session management
  session_timeout_minutes: 240,
  max_sessions_per_user: 5
})
```

## Common Use Cases

### Use Case 1: Configure GitHub OAuth

Set up GitHub authentication for your application:

```typescript
// Step 1: Create GitHub OAuth provider
const github = await client.admin.oauth.providers.createProvider({
  provider_name: 'github',
  display_name: 'GitHub',
  enabled: true,
  client_id: process.env.GITHUB_CLIENT_ID!,
  client_secret: process.env.GITHUB_CLIENT_SECRET!,
  redirect_url: 'https://yourapp.com/auth/callback/github',
  scopes: ['user:email', 'read:user'],
  is_custom: false
})

console.log('GitHub OAuth configured:', github.id)

// Step 2: Users can now sign in with GitHub
// In your frontend:
// window.location.href = 'http://localhost:8080/auth/github'
```

### Use Case 2: Configure Multiple OAuth Providers

Set up multiple authentication providers:

```typescript
const providers = [
  {
    provider_name: 'github',
    display_name: 'GitHub',
    client_id: process.env.GITHUB_CLIENT_ID!,
    client_secret: process.env.GITHUB_CLIENT_SECRET!,
    scopes: ['user:email', 'read:user']
  },
  {
    provider_name: 'google',
    display_name: 'Google',
    client_id: process.env.GOOGLE_CLIENT_ID!,
    client_secret: process.env.GOOGLE_CLIENT_SECRET!,
    scopes: ['openid', 'email', 'profile']
  },
  {
    provider_name: 'gitlab',
    display_name: 'GitLab',
    client_id: process.env.GITLAB_CLIENT_ID!,
    client_secret: process.env.GITLAB_CLIENT_SECRET!,
    scopes: ['read_user', 'email']
  }
]

// Create all providers
for (const config of providers) {
  await client.admin.oauth.providers.createProvider({
    ...config,
    enabled: true,
    redirect_url: `https://yourapp.com/auth/callback/${config.provider_name}`,
    is_custom: false
  })
  console.log(`${config.display_name} configured`)
}
```

### Use Case 3: Enterprise SSO Integration

Configure custom OAuth2 provider for enterprise single sign-on:

```typescript
await client.admin.oauth.providers.createProvider({
  provider_name: 'okta_sso',
  display_name: 'Okta SSO',
  enabled: true,
  client_id: process.env.OKTA_CLIENT_ID!,
  client_secret: process.env.OKTA_CLIENT_SECRET!,
  redirect_url: 'https://yourapp.com/auth/callback/okta',
  scopes: ['openid', 'profile', 'email', 'groups'],
  is_custom: true,
  authorization_url: 'https://your-org.okta.com/oauth2/v1/authorize',
  token_url: 'https://your-org.okta.com/oauth2/v1/token',
  user_info_url: 'https://your-org.okta.com/oauth2/v1/userinfo'
})

console.log('Enterprise SSO configured')
```

### Use Case 4: Strengthen Password Requirements

Enforce strong password policies:

```typescript
await client.admin.oauth.authSettings.update({
  password_min_length: 16,
  password_require_uppercase: true,
  password_require_lowercase: true,
  password_require_number: true,
  password_require_special: true
})

console.log('Password policy strengthened')

// Now users must create passwords with:
// - At least 16 characters
// - At least one uppercase letter
// - At least one lowercase letter
// - At least one number
// - At least one special character
```

### Use Case 5: Session Management Configuration

Configure session timeouts and limits:

```typescript
await client.admin.oauth.authSettings.update({
  session_timeout_minutes: 120, // 2 hours
  max_sessions_per_user: 3 // Limit to 3 concurrent sessions
})

console.log('Session management configured')
```

### Use Case 6: Disable Signup for Invite-Only App

Make your application invite-only:

```typescript
// Disable public signup
await client.admin.oauth.authSettings.update({
  enable_signup: false
})

console.log('Public signup disabled')

// Users can now only join via admin invitation
const invitation = await client.admin.management.invitations.create({
  email: 'newuser@example.com',
  role: 'user',
  expires_in_hours: 72
})

console.log('Invitation created:', invitation.invite_url)
```

### Use Case 7: Development vs Production Settings

Configure different settings for development and production:

```typescript
const isDevelopment = process.env.NODE_ENV === 'development'

await client.admin.oauth.authSettings.update({
  // Relaxed requirements for development
  password_min_length: isDevelopment ? 8 : 16,
  require_email_verification: !isDevelopment,

  // Longer sessions in development
  session_timeout_minutes: isDevelopment ? 480 : 120,

  // More sessions allowed in development
  max_sessions_per_user: isDevelopment ? 10 : 3
})

console.log(`Settings configured for ${isDevelopment ? 'development' : 'production'}`)
```

### Use Case 8: Rotate OAuth Credentials

Regularly rotate OAuth credentials for security:

```typescript
async function rotateOAuthCredentials(providerId: string) {
  // Get current provider
  const provider = await client.admin.oauth.providers.getProvider(providerId)

  console.log(`Rotating credentials for ${provider.display_name}`)

  // Update with new credentials
  await client.admin.oauth.providers.updateProvider(providerId, {
    client_id: process.env.NEW_CLIENT_ID!,
    client_secret: process.env.NEW_CLIENT_SECRET!
  })

  console.log('Credentials rotated successfully')
}

// Rotate GitHub credentials
await rotateOAuthCredentials('github-provider-id')
```

### Use Case 9: Enable Magic Link Authentication

Configure passwordless magic link authentication:

```typescript
await client.admin.oauth.authSettings.update({
  enable_magic_link: true,
  require_email_verification: true
})

console.log('Magic link authentication enabled')

// Users can now sign in with email-only authentication
// Backend will send a magic link to their email
```

### Use Case 10: Audit OAuth Configuration

Review and audit OAuth provider configuration:

```typescript
const providers = await client.admin.oauth.providers.listProviders()
const settings = await client.admin.oauth.authSettings.get()

console.log('OAuth Configuration Audit')
console.log('========================\n')

console.log('Providers:')
providers.forEach(provider => {
  console.log(`- ${provider.display_name}:`, provider.enabled ? '✅ Enabled' : '❌ Disabled')
  console.log(`  Type: ${provider.is_custom ? 'Custom' : 'Built-in'}`)
  console.log(`  Client ID: ${provider.client_id}`)
  console.log(`  Scopes: ${provider.scopes.join(', ')}`)
})

console.log('\nAuthentication Settings:')
console.log(`- Signup: ${settings.enable_signup ? 'Enabled' : 'Disabled'}`)
console.log(`- Email verification: ${settings.require_email_verification ? 'Required' : 'Optional'}`)
console.log(`- Magic link: ${settings.enable_magic_link ? 'Enabled' : 'Disabled'}`)
console.log(`- Password min length: ${settings.password_min_length}`)
console.log(`- Password requirements:`)
console.log(`  - Uppercase: ${settings.password_require_uppercase ? 'Yes' : 'No'}`)
console.log(`  - Lowercase: ${settings.password_require_lowercase ? 'Yes' : 'No'}`)
console.log(`  - Number: ${settings.password_require_number ? 'Yes' : 'No'}`)
console.log(`  - Special: ${settings.password_require_special ? 'Yes' : 'No'}`)
console.log(`- Session timeout: ${settings.session_timeout_minutes} minutes`)
console.log(`- Max sessions per user: ${settings.max_sessions_per_user}`)
```

## Error Handling

Handle common errors when working with OAuth configuration:

```typescript
import { FluxbaseError } from '@fluxbase/sdk'

try {
  await client.admin.oauth.providers.createProvider({
    provider_name: 'github',
    display_name: 'GitHub',
    enabled: true,
    client_id: 'invalid-id',
    client_secret: 'invalid-secret',
    redirect_url: 'https://yourapp.com/callback',
    scopes: ['user:email'],
    is_custom: false
  })
} catch (error) {
  if (error instanceof FluxbaseError) {
    if (error.status === 409) {
      console.error('Provider already exists')
    } else if (error.status === 400) {
      console.error('Invalid provider configuration:', error.message)
    } else if (error.status === 401) {
      console.error('Not authenticated as admin')
    } else {
      console.error('Failed to create provider:', error.message)
    }
  } else {
    console.error('Unexpected error:', error)
  }
}
```

## Type Definitions

### OAuthProvider

```typescript
interface OAuthProvider {
  id: string
  provider_name: string
  display_name: string
  enabled: boolean
  client_id: string
  // client_secret is never returned
  redirect_url: string
  scopes: string[]
  is_custom: boolean
  authorization_url?: string
  token_url?: string
  user_info_url?: string
  created_at: string
  updated_at: string
}
```

### CreateOAuthProviderRequest

```typescript
interface CreateOAuthProviderRequest {
  provider_name: string
  display_name: string
  enabled: boolean
  client_id: string
  client_secret: string
  redirect_url: string
  scopes: string[]
  is_custom: boolean
  authorization_url?: string
  token_url?: string
  user_info_url?: string
}
```

### UpdateOAuthProviderRequest

```typescript
interface UpdateOAuthProviderRequest {
  display_name?: string
  enabled?: boolean
  client_id?: string
  client_secret?: string
  redirect_url?: string
  scopes?: string[]
  authorization_url?: string
  token_url?: string
  user_info_url?: string
}
```

### AuthSettings

```typescript
interface AuthSettings {
  enable_signup: boolean
  require_email_verification: boolean
  enable_magic_link: boolean
  password_min_length: number
  password_require_uppercase: boolean
  password_require_lowercase: boolean
  password_require_number: boolean
  password_require_special: boolean
  session_timeout_minutes: number
  max_sessions_per_user: number
}
```

### UpdateAuthSettingsRequest

```typescript
interface UpdateAuthSettingsRequest {
  enable_signup?: boolean
  require_email_verification?: boolean
  enable_magic_link?: boolean
  password_min_length?: number
  password_require_uppercase?: boolean
  password_require_lowercase?: boolean
  password_require_number?: boolean
  password_require_special?: boolean
  session_timeout_minutes?: number
  max_sessions_per_user?: number
}
```

## Best Practices

### 1. Secure Credential Storage

Never hardcode OAuth credentials:

```typescript
// ❌ Bad - hardcoded credentials
const provider = await client.admin.oauth.providers.createProvider({
  client_id: 'abc123',
  client_secret: 'secret123',
  // ...
})

// ✅ Good - use environment variables
const provider = await client.admin.oauth.providers.createProvider({
  client_id: process.env.GITHUB_CLIENT_ID!,
  client_secret: process.env.GITHUB_CLIENT_SECRET!,
  // ...
})
```

### 2. Regular Credential Rotation

Rotate OAuth credentials periodically:

```typescript
// Rotate credentials every 90 days
async function scheduleCredentialRotation(providerId: string) {
  setInterval(async () => {
    await client.admin.oauth.providers.updateProvider(providerId, {
      client_id: await getNewClientId(),
      client_secret: await getNewClientSecret()
    })
    console.log('OAuth credentials rotated')
  }, 90 * 24 * 60 * 60 * 1000) // 90 days
}
```

### 3. Validate Redirect URLs

Always use HTTPS redirect URLs in production:

```typescript
const redirectUrl = process.env.NODE_ENV === 'production'
  ? 'https://yourapp.com/auth/callback'
  : 'http://localhost:3000/auth/callback'

await client.admin.oauth.providers.createProvider({
  // ...
  redirect_url: redirectUrl
})
```

### 4. Enforce Strong Password Policies

Set strong password requirements in production:

```typescript
await client.admin.oauth.authSettings.update({
  password_min_length: 16,
  password_require_uppercase: true,
  password_require_lowercase: true,
  password_require_number: true,
  password_require_special: true
})
```

### 5. Limit Session Count

Prevent session accumulation:

```typescript
await client.admin.oauth.authSettings.update({
  max_sessions_per_user: 3
})
```

### 6. Provider-Specific Scopes

Request only necessary OAuth scopes:

```typescript
// ✅ Good - minimal scopes
const githubScopes = ['user:email', 'read:user']

// ❌ Bad - excessive scopes
const excessiveScopes = ['user', 'repo', 'admin:org', 'delete_repo']
```

## Related Resources

- [Admin SDK](/docs/sdk/admin) - Admin authentication and user management
- [Management SDK](/docs/sdk/management) - API keys, webhooks, and invitations
- [Settings SDK](/docs/sdk/settings) - Application configuration
- [Advanced Features](/docs/sdk/advanced-features) - Complete feature overview
- [Authentication Guide](/docs/guides/authentication) - End-user authentication
- [Security Best Practices](/docs/security/best-practices) - Security guidelines
