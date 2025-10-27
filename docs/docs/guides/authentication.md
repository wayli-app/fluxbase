# Authentication

Fluxbase provides a complete authentication system with JWT tokens, password management, and session handling.

## Overview

The authentication system includes:

- **User Registration** - Create new user accounts with email and password
- **Login/Logout** - Secure authentication with JWT tokens
- **Token Refresh** - Automatic token renewal without re-authentication
- **Password Management** - Secure password hashing with bcrypt
- **Email Verification** - Verify user email addresses
- **Session Management** - Track active user sessions

## Quick Start

### User Registration

```bash
curl -X POST http://localhost:8080/api/v1/auth/signup \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePassword123"
  }'
```

Response:

```json
{
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "email_verified": false,
    "role": "authenticated",
    "created_at": "2024-10-26T10:00:00Z",
    "updated_at": "2024-10-26T10:00:00Z"
  },
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_at": "2024-10-26T10:15:00Z"
}
```

### User Login

```bash
curl -X POST http://localhost:8080/api/v1/auth/signin \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePassword123"
  }'
```

Response:

```json
{
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "email_verified": false,
    "role": "authenticated"
  },
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_at": "2024-10-26T10:15:00Z"
}
```

### Making Authenticated Requests

Include the access token in the Authorization header:

```bash
curl http://localhost:8080/api/v1/auth/user \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

### Token Refresh

When the access token expires, use the refresh token to get a new one:

```bash
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }'
```

Response:

```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_at": "2024-10-26T10:30:00Z"
}
```

### Logout

```bash
curl -X POST http://localhost:8080/api/v1/auth/signout \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

## API Reference

### POST /api/v1/auth/signup

Register a new user account.

**Request Body:**

```json
{
  "email": "user@example.com",
  "password": "SecurePassword123",
  "role": "authenticated" // optional
}
```

**Response (201 Created):**

```json
{
  "user": {
    "id": "uuid",
    "email": "user@example.com",
    "email_verified": false,
    "role": "authenticated",
    "created_at": "timestamp",
    "updated_at": "timestamp"
  },
  "access_token": "jwt_token",
  "refresh_token": "jwt_token",
  "expires_at": "timestamp"
}
```

**Errors:**

- `400 Bad Request` - Invalid email or weak password
- `409 Conflict` - Email already registered

---

### POST /api/v1/auth/signin

Authenticate with email and password.

**Request Body:**

```json
{
  "email": "user@example.com",
  "password": "SecurePassword123"
}
```

**Response (200 OK):**

```json
{
  "user": {...},
  "access_token": "jwt_token",
  "refresh_token": "jwt_token",
  "expires_at": "timestamp"
}
```

**Errors:**

- `400 Bad Request` - Missing email or password
- `401 Unauthorized` - Invalid credentials

---

### POST /api/v1/auth/signout

Invalidate the current session.

**Headers:**

```
Authorization: Bearer {access_token}
```

**Response (204 No Content)**

**Errors:**

- `401 Unauthorized` - Invalid or missing token

---

### POST /api/v1/auth/refresh

Get a new access token using a refresh token.

**Request Body:**

```json
{
  "refresh_token": "jwt_token"
}
```

**Response (200 OK):**

```json
{
  "access_token": "new_jwt_token",
  "expires_at": "timestamp"
}
```

**Errors:**

- `400 Bad Request` - Missing refresh token
- `401 Unauthorized` - Invalid or expired refresh token

---

### GET /api/v1/auth/user

Get the current authenticated user's profile.

**Headers:**

```
Authorization: Bearer {access_token}
```

**Response (200 OK):**

```json
{
  "id": "uuid",
  "email": "user@example.com",
  "email_verified": true,
  "role": "authenticated",
  "metadata": {},
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

**Errors:**

- `401 Unauthorized` - Invalid or missing token

---

### PATCH /api/v1/auth/user

Update the current user's profile.

**Headers:**

```
Authorization: Bearer {access_token}
```

**Request Body:**

```json
{
  "email": "newemail@example.com", // optional
  "metadata": {
    // optional
    "display_name": "John Doe"
  }
}
```

**Response (200 OK):**

```json
{
  "id": "uuid",
  "email": "newemail@example.com",
  "email_verified": false, // Reset if email changed
  "role": "authenticated",
  "metadata": {
    "display_name": "John Doe"
  },
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

---

### POST /api/v1/auth/magic-link

Request a magic link for passwordless authentication (Coming Soon).

---

### GET /api/v1/auth/verify

Verify email address via verification link (Coming Soon).

## Configuration

Authentication settings can be configured via environment variables or config file:

```yaml
auth:
  jwt_secret: "your-secret-key" # Secret for signing JWT tokens
  jwt_expiry: 15m # Access token expiration
  refresh_expiry: 168h # Refresh token expiration (7 days)
  password_min_length: 8 # Minimum password length
  bcrypt_cost: 12 # Bcrypt cost factor
  enable_signup: true # Allow new user registration
  enable_magic_link: false # Enable magic link auth
```

### Password Requirements

By default, passwords must:

- Be at least 8 characters long
- Not exceed 72 characters (bcrypt limit)

You can configure additional requirements:

- Require uppercase letters
- Require lowercase letters
- Require digits
- Require special characters

## JWT Token Structure

Access tokens contain the following claims:

```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "role": "authenticated",
  "session_id": "session-uuid",
  "token_type": "access",
  "iss": "fluxbase",
  "sub": "user-id",
  "iat": 1698307200,
  "exp": 1698308100,
  "nbf": 1698307200,
  "jti": "token-uuid"
}
```

## Security Best Practices

### Password Storage

- Passwords are hashed using bcrypt with configurable cost factor (default: 12)
- Password hashes are never exposed in API responses
- Supports automatic hash upgrades when cost factor changes

### Token Security

- JWT tokens are signed with HMAC-SHA256
- Access tokens are short-lived (default: 15 minutes)
- Refresh tokens are long-lived (default: 7 days)
- Both token types must be validated on each request

### Session Management

- Each login creates a new session with unique session ID
- Sessions can be invalidated on logout
- Concurrent sessions are supported
- Session tracking helps identify active logins

## Database Schema

### Users Table

```sql
CREATE TABLE auth.users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    email_verified BOOLEAN DEFAULT false,
    role TEXT DEFAULT 'authenticated',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_users_email ON auth.users(email);
```

### Sessions Table

```sql
CREATE TABLE auth.sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    access_token TEXT NOT NULL,
    refresh_token TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(access_token),
    UNIQUE(refresh_token)
);

CREATE INDEX idx_sessions_user_id ON auth.sessions(user_id);
CREATE INDEX idx_sessions_access_token ON auth.sessions(access_token);
```

## Client SDKs

### JavaScript/TypeScript

```typescript
import { FluxbaseClient } from "@fluxbase/client";

const client = new FluxbaseClient({
  url: "http://localhost:8080",
});

// Sign up
const { user, session } = await client.auth.signUp({
  email: "user@example.com",
  password: "SecurePassword123",
});

// Sign in
const { user, session } = await client.auth.signIn({
  email: "user@example.com",
  password: "SecurePassword123",
});

// Get current user
const user = await client.auth.user();

// Sign out
await client.auth.signOut();

// Automatic token refresh
client.auth.onAuthStateChange((event, session) => {
  console.log(event, session);
});
```

### Python

```python
from fluxbase import FluxbaseClient

client = FluxbaseClient(url="http://localhost:8080")

# Sign up
response = client.auth.sign_up(
    email="user@example.com",
    password="SecurePassword123"
)

# Sign in
response = client.auth.sign_in(
    email="user@example.com",
    password="SecurePassword123"
)

# Get current user
user = client.auth.user()

# Sign out
client.auth.sign_out()
```

## Troubleshooting

### "Invalid or expired token"

- Access tokens expire after 15 minutes by default
- Use the refresh token to get a new access token
- Check that you're sending the token in the Authorization header

### "Email already registered"

- The email is already in use
- Try logging in instead of signing up
- Use password reset if you forgot your password

### "Weak password"

- Password must meet minimum length requirement (default: 8 characters)
- Check for additional password requirements in your configuration

### "Session expired"

- Both access and refresh tokens have expired
- User must log in again
- Consider increasing refresh token expiration time

## Next Steps

- [Row Level Security](row-level-security) - Secure your data with RLS policies
- [REST API](the REST API) - Make authenticated requests to your data
- [Realtime](realtime features) - Subscribe to real-time database changes

## OAuth / Social Login (SSO)

Fluxbase supports OAuth authentication with multiple providers, allowing users to sign in with their existing accounts.

### Supported OAuth Providers

- **Google** - Sign in with Google
- **GitHub** - Sign in with GitHub
- **Microsoft** - Sign in with Microsoft/Azure AD
- **Apple** - Sign in with Apple
- **Facebook** - Sign in with Facebook
- **Twitter** - Sign in with Twitter
- **LinkedIn** - Sign in with LinkedIn
- **GitLab** - Sign in with GitLab
- **Bitbucket** - Sign in with Bitbucket

### OAuth Flow

#### 1. Initiate OAuth Flow

```bash
curl http://localhost:8080/api/v1/auth/oauth/google
```

This will return an authorization URL:

```json
{
  "url": "https://accounts.google.com/o/oauth2/auth?client_id=...&redirect_uri=...&state=..."
}
```

#### 2. User Authorizes

Redirect the user to the authorization URL. After authorization, they'll be redirected back to your app with a code.

#### 3. Complete OAuth Flow

```bash
curl -X POST http://localhost:8080/api/v1/auth/oauth/callback \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "google",
    "code": "authorization_code",
    "state": "state_token"
  }'
```

Response:

```json
{
  "user": {
    "id": "uuid",
    "email": "user@gmail.com",
    "email_verified": true,
    "role": "authenticated",
    "metadata": {
      "provider": "google",
      "provider_id": "google_user_id"
    }
  },
  "access_token": "jwt_token",
  "refresh_token": "jwt_token",
  "expires_at": "timestamp"
}
```

### Configuration

Configure OAuth providers in your environment or config file:

```yaml
auth:
  oauth:
    google:
      client_id: "your-google-client-id"
      client_secret: "your-google-client-secret"
      redirect_url: "http://localhost:8080/auth/callback"
      scopes:
        - "email"
        - "profile"

    github:
      client_id: "your-github-client-id"
      client_secret: "your-github-client-secret"
      redirect_url: "http://localhost:8080/auth/callback"
      scopes:
        - "user:email"

    microsoft:
      client_id: "your-microsoft-client-id"
      client_secret: "your-microsoft-client-secret"
      redirect_url: "http://localhost:8080/auth/callback"
      scopes:
        - "openid"
        - "email"
        - "profile"
```

### Client Implementation

**JavaScript/TypeScript:**

```typescript
// Initiate OAuth flow
const { url } = await client.auth.signInWithOAuth({
  provider: "google",
  options: {
    redirectTo: "http://localhost:3000/auth/callback",
  },
});

// Redirect user to OAuth provider
window.location.href = url;

// Handle callback (in your callback page)
const { user, session } = await client.auth.exchangeCodeForSession({
  code: searchParams.get("code"),
  state: searchParams.get("state"),
});
```

**Python:**

```python
# Initiate OAuth flow
response = client.auth.sign_in_with_oauth(
    provider="google",
    redirect_to="http://localhost:3000/auth/callback"
)
# Redirect user to response['url']

# Handle callback
session = client.auth.exchange_code_for_session(
    code=request.args.get('code'),
    state=request.args.get('state')
)
```

---

## Magic Link (Passwordless)

Magic links provide a passwordless authentication method. Users receive an email with a one-time login link.

### Send Magic Link

```bash
curl -X POST http://localhost:8080/api/v1/auth/magic-link \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com"
  }'
```

Response:

```json
{
  "message": "Magic link sent to user@example.com"
}
```

### Verify Magic Link

When the user clicks the link, they'll be redirected to:

```
http://localhost:8080/api/v1/auth/verify?token=MAGIC_LINK_TOKEN
```

Or you can verify programmatically:

```bash
curl -X POST http://localhost:8080/api/v1/auth/verify \
  -H "Content-Type: application/json" \
  -d '{
    "token": "MAGIC_LINK_TOKEN"
  }'
```

Response:

```json
{
  "user": {
    "id": "uuid",
    "email": "user@example.com",
    "email_verified": true
  },
  "access_token": "jwt_token",
  "refresh_token": "jwt_token",
  "expires_at": "timestamp"
}
```

### Configuration

```yaml
auth:
  magic_link:
    enabled: true
    expiry: 15m # Magic link expires after 15 minutes
    email:
      from: "noreply@yourapp.com"
      subject: "Your login link"
      template: |
        Hi,

        Click the link below to log in:
        {{.Link}}

        This link expires in 15 minutes.
```

### Client Implementation

**JavaScript/TypeScript:**

```typescript
// Request magic link
await client.auth.signInWithMagicLink({
  email: "user@example.com",
  options: {
    redirectTo: "http://localhost:3000/auth/callback",
  },
});

// User clicks link in email, gets redirected to callback
// Extract token from URL and verify
const { user, session } = await client.auth.verifyMagicLink({
  token: searchParams.get("token"),
});
```

**Python:**

```python
# Request magic link
client.auth.sign_in_with_magic_link(
    email="user@example.com",
    redirect_to="http://localhost:3000/auth/callback"
)

# Verify magic link
session = client.auth.verify_magic_link(
    token=request.args.get('token')
)
```

### Security Considerations

**Magic Links:**

- Links expire after a configurable time (default: 15 minutes)
- One-time use only - cannot be reused
- Sent to verified email address
- Include CSRF protection via state parameter

**OAuth:**

- State parameter prevents CSRF attacks
- Tokens are validated before exchanging for session
- Provider tokens are not stored
- User data is fetched fresh on each OAuth login

---

## Multi-Factor Authentication (Coming Soon)

Support for TOTP-based 2FA will be added in a future release.
