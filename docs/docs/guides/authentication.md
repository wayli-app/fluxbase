# Authentication

Fluxbase provides JWT-based authentication with support for email/password, magic links, OAuth, anonymous auth, and two-factor authentication.

## Features

- Email/password authentication
- Magic links (passwordless)
- OAuth providers (Google, GitHub, Microsoft, etc.)
- Anonymous authentication
- Two-factor authentication (TOTP)
- Session management
- Password reset flows

## Configuration

Configure authentication in your config file or via environment variables:

```yaml
auth:
  jwt_secret: "your-secret-key"
  jwt_expiry: 15m
  refresh_expiry: 168h # 7 days
  password_min_length: 8
  bcrypt_cost: 12
  enable_signup: true
  enable_magic_link: false
```

### Password Requirements

Default requirements:
- Minimum 8 characters
- Maximum 72 characters (bcrypt limit)

Optional requirements (configurable):
- Uppercase letters
- Lowercase letters
- Digits
- Special characters

## Installation

```bash
npm install @fluxbase/sdk
```

## Quick Start

```typescript
import { FluxbaseClient } from '@fluxbase/sdk'

const client = new FluxbaseClient({
  url: 'http://localhost:8080'
})

// Sign up
const { user, session } = await client.auth.signUp({
  email: 'user@example.com',
  password: 'SecurePassword123'
})

// Sign in
const { user, session } = await client.auth.signIn({
  email: 'user@example.com',
  password: 'SecurePassword123'
})

// Get current user
const user = await client.auth.getCurrentUser()

// Sign out
await client.auth.signOut()
```

## Core Authentication

### Sign Up

```typescript
const { user, session } = await client.auth.signUp({
  email: 'user@example.com',
  password: 'SecurePassword123',
  metadata: { name: 'John Doe' } // optional
})
```

### Sign In

```typescript
const { user, session } = await client.auth.signIn({
  email: 'user@example.com',
  password: 'SecurePassword123'
})
```

### Sign Out

```typescript
await client.auth.signOut()
```

### Get Current User

```typescript
const user = await client.auth.getCurrentUser()
if (user) {
  console.log('Logged in as:', user.email)
}
```

### Get Session

```typescript
const session = await client.auth.getSession()
if (session) {
  console.log('Token expires at:', session.expires_at)
}
```

## Password Reset

### Request Reset

```typescript
await client.auth.resetPassword({
  email: 'user@example.com'
})
// Reset email sent to user
```

### Confirm Reset

Users receive a reset token via email:

```typescript
await client.auth.confirmPasswordReset({
  token: 'reset-token-from-email',
  password: 'NewSecurePassword123'
})
```

## Magic Links

Enable magic links in configuration, then:

```typescript
// Request magic link
await client.auth.sendMagicLink({
  email: 'user@example.com'
})

// User clicks link, automatically signed in
```

## OAuth / Social Login

### Available Providers

- Google
- GitHub
- Microsoft
- GitLab
- Bitbucket
- Facebook
- Twitter/X
- Discord
- Slack

### Configuration

```yaml
oauth:
  google:
    client_id: "your-client-id"
    client_secret: "your-client-secret"
    redirect_url: "http://localhost:8080/api/v1/auth/callback/google"
  github:
    client_id: "your-client-id"
    client_secret: "your-client-secret"
    redirect_url: "http://localhost:8080/api/v1/auth/callback/github"
```

### Usage

```typescript
// Get authorization URL
const { url } = await client.auth.getOAuthUrl({
  provider: 'google',
  redirectTo: 'http://localhost:3000/dashboard'
})

// Redirect user to authorization URL
window.location.href = url

// Handle callback (automatic)
// User is redirected back with session tokens
```

## Anonymous Authentication

Allow guest access without account creation:

```typescript
const { user, session } = await client.auth.signInAnonymously()

// User has limited permissions (configure via RLS)
console.log('Anonymous user:', user.id)

// Convert to permanent account later
await client.auth.convertAnonymousUser({
  email: 'user@example.com',
  password: 'SecurePassword123'
})
```

## Two-Factor Authentication

### Enable 2FA

```typescript
// Generate TOTP secret
const { secret, qr_code } = await client.auth.enable2FA()

// Display QR code to user for scanning with authenticator app
console.log('Scan this QR code:', qr_code)

// Verify setup with code from authenticator
await client.auth.verify2FA({
  code: '123456' // from authenticator app
})
```

### Disable 2FA

```typescript
await client.auth.disable2FA({
  code: '123456' // verification code
})
```

### Sign In with 2FA

```typescript
// Initial sign in
const { requires_2fa } = await client.auth.signIn({
  email: 'user@example.com',
  password: 'SecurePassword123'
})

if (requires_2fa) {
  // Prompt user for 2FA code
  const { user, session } = await client.auth.verify2FACode({
    code: '123456'
  })
}
```

## Session Management

### List Active Sessions

```typescript
const sessions = await client.auth.listSessions()

sessions.forEach(session => {
  console.log('Session:', session.id)
  console.log('Created:', session.created_at)
  console.log('Last active:', session.last_active_at)
  console.log('IP:', session.ip_address)
  console.log('User agent:', session.user_agent)
})
```

### Revoke Session

```typescript
// Revoke specific session
await client.auth.revokeSession(session_id)

// Revoke all other sessions (keep current)
await client.auth.revokeAllSessions({ except_current: true })
```

## Token Refresh

Tokens are automatically refreshed by the SDK. For manual refresh:

```typescript
const { session } = await client.auth.refreshSession()
console.log('New token:', session.access_token)
```

## Auth State Changes

Listen to authentication state changes:

```typescript
const subscription = client.auth.onAuthStateChange((event, session) => {
  console.log('Auth event:', event)
  // Events: SIGNED_IN, SIGNED_OUT, TOKEN_REFRESHED, USER_UPDATED

  if (event === 'SIGNED_IN') {
    console.log('User signed in:', session.user)
  } else if (event === 'SIGNED_OUT') {
    console.log('User signed out')
  }
})

// Unsubscribe when done
subscription.unsubscribe()
```

## User Metadata

### Update User Metadata

```typescript
await client.auth.updateUser({
  metadata: {
    name: 'John Doe',
    avatar_url: 'https://example.com/avatar.jpg'
  }
})
```

### Update Email

```typescript
await client.auth.updateEmail({
  email: 'newemail@example.com',
  password: 'CurrentPassword123' // confirmation required
})
// Verification email sent
```

### Update Password

```typescript
await client.auth.updatePassword({
  current_password: 'OldPassword123',
  new_password: 'NewPassword123'
})
```

## API Keys

Generate API keys for server-to-server authentication:

```typescript
// Create API key
const { key, id } = await client.auth.createApiKey({
  name: 'Production API',
  expires_in: 86400 * 365 // 1 year in seconds
})

// List API keys
const keys = await client.auth.listApiKeys()

// Revoke API key
await client.auth.revokeApiKey(key_id)
```

Use API keys in requests:

```typescript
const client = new FluxbaseClient({
  url: 'http://localhost:8080',
  apiKey: 'your-api-key'
})
```

## Service Keys (Admin)

Service keys bypass Row-Level Security and should only be used in backend services.

```typescript
const adminClient = new FluxbaseClient({
  url: 'http://localhost:8080',
  serviceKey: process.env.FLUXBASE_SERVICE_KEY
})

// This bypasses RLS
const allUsers = await adminClient.from('users').select('*')
```

**Security best practices:**
- Store in secure secrets management
- Use environment variables
- Never expose in client code
- Never commit to version control

## REST API

For direct HTTP access without the SDK, see the [API Reference](/docs/api/authentication).

## Reference

### JWT Token Structure

Access tokens contain:

```json
{
  "user_id": "uuid",
  "email": "user@example.com",
  "role": "authenticated",
  "session_id": "uuid",
  "token_type": "access",
  "iss": "fluxbase",
  "sub": "user-id",
  "iat": 1698307200,
  "exp": 1698308100
}
```

### User Roles

- `anonymous` - Guest users (limited access)
- `authenticated` - Logged-in users
- `service_role` - Admin/backend services (bypass RLS)

Configure role-based access with Row-Level Security policies.

## Next Steps

- [Row-Level Security](/docs/guides/row-level-security) - Secure data with RLS policies
- [OAuth Providers](/docs/guides/oauth-providers) - Configure social login
- [Email Services](/docs/guides/email-services) - Set up email for password reset
