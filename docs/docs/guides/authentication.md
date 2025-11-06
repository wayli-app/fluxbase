# Authentication

Fluxbase provides a complete authentication system with JWT tokens, password management, and session handling.

## Overview

The authentication system includes:

- **User Registration** - Create new user accounts with email and password
- **Login/Logout** - Secure authentication with JWT tokens
- **Token Refresh** - Automatic token renewal without re-authentication
- **Password Reset** - Secure password reset flow with email verification
- **Magic Link (Passwordless)** - Sign in via email link without password
- **Anonymous Authentication** - Guest access without account creation
- **OAuth / Social Login** - Sign in with Google, GitHub, Microsoft, and more
- **Two-Factor Authentication (2FA)** - TOTP-based multi-factor authentication
- **Password Management** - Secure password hashing with bcrypt
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

### POST /api/v1/auth/password/reset

Request a password reset email.

**Request Body:**

```json
{
  "email": "user@example.com"
}
```

**Response (200 OK):**

```json
{
  "message": "If an account with that email exists, a password reset link has been sent"
}
```

---

### POST /api/v1/auth/password/reset/verify

Verify a password reset token before allowing password reset.

**Request Body:**

```json
{
  "token": "reset_token"
}
```

**Response (200 OK):**

```json
{
  "valid": true,
  "message": "Token is valid"
}
```

**Errors:**

- `400 Bad Request` - Invalid or expired token

---

### POST /api/v1/auth/password/reset/confirm

Complete the password reset with a valid token.

**Request Body:**

```json
{
  "token": "reset_token",
  "new_password": "NewSecurePassword123"
}
```

**Response (200 OK):**

```json
{
  "message": "Password has been successfully reset"
}
```

**Errors:**

- `400 Bad Request` - Invalid token or weak password

---

### POST /api/v1/auth/magiclink

Request a magic link for passwordless authentication.

**Request Body:**

```json
{
  "email": "user@example.com",
  "redirect_to": "https://app.example.com/dashboard" // optional
}
```

**Response (200 OK):**

```json
{
  "message": "Magic link sent to your email"
}
```

---

### POST /api/v1/auth/magiclink/verify

Verify a magic link token and sign in.

**Request Body:**

```json
{
  "token": "magic_link_token"
}
```

**Response (200 OK):**

```json
{
  "user": {...},
  "access_token": "jwt_token",
  "refresh_token": "jwt_token",
  "expires_in": 3600
}
```

**Errors:**

- `400 Bad Request` - Invalid or expired token

---

### POST /api/v1/auth/signin/anonymous

Sign in anonymously without credentials.

**Request Body:** (empty)

**Response (200 OK):**

```json
{
  "user": {
    "id": "anon-uuid",
    "email": "anonymous@fluxbase.local",
    "role": "anonymous",
    ...
  },
  "access_token": "jwt_token",
  "refresh_token": "jwt_token",
  "expires_in": 3600
}
```

---

### GET /api/v1/auth/oauth/providers

Get list of enabled OAuth providers.

**Response (200 OK):**

```json
{
  "providers": [
    {
      "id": "google",
      "name": "Google",
      "enabled": true
    },
    {
      "id": "github",
      "name": "GitHub",
      "enabled": true
    }
  ]
}
```

---

### GET /api/v1/auth/oauth/:provider/authorize

Get OAuth authorization URL for a specific provider.

**Query Parameters:**

- `redirect_to` (optional) - URL to redirect after OAuth completion
- `scopes` (optional) - Comma-separated list of scopes

**Response (200 OK):**

```json
{
  "url": "https://accounts.google.com/o/oauth2/v2/auth?...",
  "provider": "google"
}
```

---

### POST /api/v1/auth/oauth/callback

Exchange OAuth authorization code for session.

**Request Body:**

```json
{
  "code": "authorization_code"
}
```

**Response (200 OK):**

```json
{
  "user": {...},
  "access_token": "jwt_token",
  "refresh_token": "jwt_token",
  "expires_in": 3600
}
```

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
const session = await client.auth.signUp({
  email: "user@example.com",
  password: "SecurePassword123",
});

// Sign in
const session = await client.auth.signIn({
  email: "user@example.com",
  password: "SecurePassword123",
});

// Get current user
const user = await client.auth.getCurrentUser();

// Sign out
await client.auth.signOut();

// Password Reset Flow
// 1. Request password reset
await client.auth.sendPasswordReset("user@example.com");

// 2. Verify reset token (optional)
const { valid } = await client.auth.verifyResetToken("reset-token");

// 3. Reset password with token
await client.auth.resetPassword("reset-token", "NewPassword123");

// Magic Link (Passwordless)
// 1. Send magic link
await client.auth.sendMagicLink("user@example.com", {
  redirect_to: "https://app.example.com/dashboard",
});

// 2. Verify magic link (after user clicks email link)
const session = await client.auth.verifyMagicLink("magic-link-token");

// Anonymous Authentication
const anonSession = await client.auth.signInAnonymously();

// OAuth Authentication
// 1. Get list of available providers
const { providers } = await client.auth.getOAuthProviders();

// 2. Get OAuth URL (manual approach)
const { url } = await client.auth.getOAuthUrl("google", {
  redirect_to: "https://app.example.com/auth/callback",
  scopes: ["email", "profile"],
});

// 3. Or use convenience method to redirect automatically
await client.auth.signInWithOAuth("google", {
  redirect_to: "https://app.example.com/auth/callback",
});

// 4. In your OAuth callback handler, exchange code for session
const session = await client.auth.exchangeCodeForSession("auth-code");
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

---

## Password Reset Flow

Fluxbase provides a secure password reset flow that sends a time-limited token to the user's email.

### How It Works

1. **User requests password reset** - User provides their email address
2. **Email sent** - If the email exists, a reset token is sent (system doesn't reveal if email exists)
3. **Token verification** - (Optional) Verify the token is valid before showing password form
4. **Password reset** - User provides the token and new password

### Send Password Reset Email

**API:**

```bash
curl -X POST http://localhost:8080/api/v1/auth/password/reset \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com"
  }'
```

**SDK (TypeScript):**

```typescript
await client.auth.sendPasswordReset("user@example.com");
```

**Response:**

```json
{
  "message": "If an account with that email exists, a password reset link has been sent"
}
```

### Verify Reset Token (Optional)

Before showing the password reset form, you can verify the token is valid:

**SDK (TypeScript):**

```typescript
const { valid, message } = await client.auth.verifyResetToken(token);

if (valid) {
  // Show password reset form
} else {
  // Show error: token expired or invalid
}
```

### Complete Password Reset

**SDK (TypeScript):**

```typescript
try {
  await client.auth.resetPassword(token, "NewSecurePassword123");
  // Password reset successful, redirect to login
} catch (error) {
  // Handle error: invalid token or weak password
}
```

### Example: React Password Reset Flow

```tsx
import { useState } from "react";
import { FluxbaseClient } from "@fluxbase/client";

const client = new FluxbaseClient({ url: "http://localhost:8080" });

// Step 1: Request password reset
function RequestResetForm() {
  const [email, setEmail] = useState("");
  const [sent, setSent] = useState(false);

  const handleSubmit = async (e) => {
    e.preventDefault();
    await client.auth.sendPasswordReset(email);
    setSent(true);
  };

  if (sent) {
    return <p>Check your email for a password reset link.</p>;
  }

  return (
    <form onSubmit={handleSubmit}>
      <input
        type="email"
        value={email}
        onChange={(e) => setEmail(e.target.value)}
        placeholder="Your email"
        required
      />
      <button type="submit">Send Reset Link</button>
    </form>
  );
}

// Step 2: Reset password with token
function ResetPasswordForm({ token }: { token: string }) {
  const [password, setPassword] = useState("");
  const [confirmed, setConfirmed] = useState(false);

  const handleSubmit = async (e) => {
    e.preventDefault();
    try {
      await client.auth.resetPassword(token, password);
      setConfirmed(true);
      // Redirect to login after 2 seconds
      setTimeout(() => (window.location.href = "/login"), 2000);
    } catch (error) {
      alert("Failed to reset password. Please try again.");
    }
  };

  if (confirmed) {
    return <p>Password reset successful! Redirecting to login...</p>;
  }

  return (
    <form onSubmit={handleSubmit}>
      <input
        type="password"
        value={password}
        onChange={(e) => setPassword(e.target.value)}
        placeholder="New password"
        required
      />
      <button type="submit">Reset Password</button>
    </form>
  );
}
```

### Configuration

Configure password reset settings:

```yaml
auth:
  password_reset:
    enabled: true
    token_expiry: 1h # Reset token expires after 1 hour
    email:
      from: "noreply@yourapp.com"
      subject: "Reset your password"
      template: |
        Hi,

        Click the link below to reset your password:
        {{.ResetLink}}

        This link expires in 1 hour.
        If you didn't request this, please ignore this email.
```

### Security Considerations

- **No user enumeration** - API doesn't reveal if email exists in database
- **Time-limited tokens** - Reset tokens expire after configured time (default: 1 hour)
- **One-time use** - Tokens can only be used once
- **Secure token generation** - Uses cryptographically secure random token
- **Password requirements** - New password must meet minimum requirements

---

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
import { FluxbaseClient } from "@fluxbase/client";

const client = new FluxbaseClient({ url: "http://localhost:8080" });

// Method 1: Automatic redirect (convenience method)
await client.auth.signInWithOAuth("google", {
  redirect_to: "http://localhost:3000/auth/callback",
  scopes: ["email", "profile"],
});
// User is automatically redirected to Google OAuth page

// Method 2: Manual control over redirect
const { url } = await client.auth.getOAuthUrl("github", {
  redirect_to: "http://localhost:3000/auth/callback",
  scopes: ["read:user", "user:email"],
});
window.location.href = url; // Manually redirect

// Handle OAuth callback (in your /auth/callback page)
// Extract code from URL query parameters
const urlParams = new URLSearchParams(window.location.search);
const code = urlParams.get("code");

if (code) {
  try {
    const session = await client.auth.exchangeCodeForSession(code);
    console.log("Logged in as:", session.user.email);
    // Redirect to app
    window.location.href = "/dashboard";
  } catch (error) {
    console.error("OAuth failed:", error);
  }
}
```

**React Example:**

```tsx
import { useEffect } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { FluxbaseClient } from "@fluxbase/client";

const client = new FluxbaseClient({ url: "http://localhost:8080" });

// Login page
function LoginPage() {
  const handleGoogleLogin = async () => {
    await client.auth.signInWithOAuth("google", {
      redirect_to: window.location.origin + "/auth/callback",
    });
  };

  return (
    <div>
      <button onClick={handleGoogleLogin}>Sign in with Google</button>
    </div>
  );
}

// OAuth callback handler
function OAuthCallback() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();

  useEffect(() => {
    const code = searchParams.get("code");
    if (code) {
      client.auth
        .exchangeCodeForSession(code)
        .then(() => navigate("/dashboard"))
        .catch((error) => {
          console.error("OAuth failed:", error);
          navigate("/login");
        });
    }
  }, [searchParams, navigate]);

  return <div>Completing sign in...</div>;
}
```

**Python:**

```python
from fluxbase import FluxbaseClient

client = FluxbaseClient(url="http://localhost:8080")

# Initiate OAuth flow
oauth_url_response = client.auth.get_oauth_url(
    provider="google",
    redirect_to="http://localhost:3000/auth/callback"
)
# Redirect user to oauth_url_response['url']

# Handle callback
session = client.auth.exchange_code_for_session(
    code=request.args.get('code')
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
import { FluxbaseClient } from "@fluxbase/client";

const client = new FluxbaseClient({ url: "http://localhost:8080" });

// Step 1: Request magic link
await client.auth.sendMagicLink("user@example.com", {
  redirect_to: "http://localhost:3000/auth/callback",
});
// User receives email with magic link

// Step 2: Verify magic link (in your callback handler)
// Extract token from URL query parameters
const urlParams = new URLSearchParams(window.location.search);
const token = urlParams.get("token");

if (token) {
  try {
    const session = await client.auth.verifyMagicLink(token);
    console.log("Logged in as:", session.user.email);
    window.location.href = "/dashboard";
  } catch (error) {
    console.error("Magic link verification failed:", error);
  }
}
```

**React Example:**

```tsx
import { useState, useEffect } from "react";
import { useSearchParams, useNavigate } from "react-router-dom";
import { FluxbaseClient } from "@fluxbase/client";

const client = new FluxbaseClient({ url: "http://localhost:8080" });

// Login page - request magic link
function MagicLinkLogin() {
  const [email, setEmail] = useState("");
  const [sent, setSent] = useState(false);

  const handleSubmit = async (e) => {
    e.preventDefault();
    await client.auth.sendMagicLink(email, {
      redirect_to: window.location.origin + "/auth/magic-link",
    });
    setSent(true);
  };

  if (sent) {
    return (
      <div>
        <h2>Check your email</h2>
        <p>We sent a magic link to {email}.</p>
        <p>Click the link to sign in.</p>
      </div>
    );
  }

  return (
    <form onSubmit={handleSubmit}>
      <input
        type="email"
        value={email}
        onChange={(e) => setEmail(e.target.value)}
        placeholder="Your email"
        required
      />
      <button type="submit">Send Magic Link</button>
    </form>
  );
}

// Magic link callback handler
function MagicLinkCallback() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const [error, setError] = useState(null);

  useEffect(() => {
    const token = searchParams.get("token");
    if (token) {
      client.auth
        .verifyMagicLink(token)
        .then(() => {
          navigate("/dashboard");
        })
        .catch((err) => {
          setError("Magic link is invalid or expired");
        });
    }
  }, [searchParams, navigate]);

  if (error) {
    return <div>Error: {error}</div>;
  }

  return <div>Verifying magic link...</div>;
}
```

**Python:**

```python
from fluxbase import FluxbaseClient

client = FluxbaseClient(url="http://localhost:8080")

# Request magic link
client.auth.send_magic_link(
    email="user@example.com",
    redirect_to="http://localhost:3000/auth/callback"
)

# Verify magic link (in callback handler)
token = request.args.get('token')
session = client.auth.verify_magic_link(token)
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

## Anonymous Authentication

Anonymous authentication allows users to access your application without creating an account. This is useful for:

- Guest checkout in e-commerce
- Trial/demo access
- Gaming sessions
- Shopping carts before checkout
- Converting anonymous users to registered users later

### How It Works

1. **Create anonymous session** - User gets a temporary session without providing credentials
2. **Use the app** - User can interact with the app as an anonymous user
3. **Convert to permanent** (optional) - User can later register to keep their data

### Sign In Anonymously

**API:**

```bash
curl -X POST http://localhost:8080/api/v1/auth/signin/anonymous
```

**SDK (TypeScript):**

```typescript
const session = await client.auth.signInAnonymously();
console.log("Anonymous user ID:", session.user.id);
```

**Response:**

```json
{
  "user": {
    "id": "anon-uuid",
    "email": "anonymous@fluxbase.local",
    "role": "anonymous",
    "created_at": "2024-10-26T10:00:00Z"
  },
  "access_token": "jwt_token",
  "refresh_token": "jwt_token",
  "expires_in": 3600
}
```

### React Example: Guest Checkout

```tsx
import { useState, useEffect } from "react";
import { FluxbaseClient } from "@fluxbase/client";

const client = new FluxbaseClient({ url: "http://localhost:8080" });

function GuestCheckout() {
  const [cart, setCart] = useState([]);
  const [session, setSession] = useState(null);

  useEffect(() => {
    // Sign in anonymously when component mounts
    const initAnonymousSession = async () => {
      const anonSession = await client.auth.signInAnonymously();
      setSession(anonSession);
    };

    // Check if user is already signed in
    const existingSession = client.auth.getSession();
    if (!existingSession) {
      initAnonymousSession();
    } else {
      setSession(existingSession);
    }
  }, []);

  const addToCart = async (item) => {
    // Anonymous users can add items to cart
    const { data } = await client
      .from("cart_items")
      .insert({ user_id: session.user.id, ...item });

    setCart([...cart, data]);
  };

  const convertToRegisteredUser = async (email, password) => {
    // Convert anonymous user to registered user
    await client.auth.signUp({ email, password });
    // Cart items will be preserved with the same user_id
  };

  return (
    <div>
      <h2>Shopping Cart</h2>
      {session?.user.role === "anonymous" && <p>Sign up to save your cart!</p>}
      {/* Cart UI */}
    </div>
  );
}
```

### Converting Anonymous Users

Anonymous users can be converted to permanent users by signing up with email/password or OAuth:

```typescript
// User is currently anonymous
const anonSession = await client.auth.signInAnonymously();

// Later, user decides to create an account
// Their anonymous data can be migrated using RLS policies
await client.auth.signUp({
  email: "user@example.com",
  password: "SecurePassword123",
});
```

### Configuration

Configure anonymous authentication:

```yaml
auth:
  anonymous:
    enabled: true
    session_expiry: 24h # How long anonymous sessions last
    auto_cleanup: 30d # Delete anonymous users after 30 days of inactivity
```

### Security Considerations

- **Limited permissions** - Anonymous users should have restricted RLS policies
- **Time-limited sessions** - Anonymous sessions expire after configured time
- **Data cleanup** - Old anonymous user data should be cleaned up periodically
- **Rate limiting** - Apply stricter rate limits to anonymous users

### RLS Policies for Anonymous Users

Example policy to allow anonymous users to manage their own cart:

```sql
-- Allow anonymous users to insert their own cart items
CREATE POLICY "Anonymous users can manage their cart"
ON cart_items
FOR ALL
USING (current_setting('app.user_id', true)::uuid = user_id)
WITH CHECK (current_setting('app.user_id', true)::uuid = user_id);

-- Allow authenticated users more privileges
CREATE POLICY "Authenticated users can save orders"
ON orders
FOR INSERT
TO authenticated
WITH CHECK (current_setting('app.user_id', true)::uuid = user_id);
```

---

## API Keys & Service Keys

Fluxbase supports three types of authentication for programmatic access:

### 1. User API Keys

User API keys are tied to specific user accounts and inherit the user's permissions and RLS policies.

**Creating an API Key:**

```bash
curl -X POST http://localhost:8080/api/v1/api-keys \
  -H "Authorization: Bearer {your_jwt_token}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My Application Key",
    "description": "API key for my mobile app",
    "scopes": ["read:data", "write:data"],
    "rate_limit_per_minute": 100
  }'
```

**Response:**

```json
{
  "id": "uuid",
  "name": "My Application Key",
  "key": "fbk_live_abc123...", // Only shown once!
  "key_prefix": "fbk_live",
  "scopes": ["read:data", "write:data"],
  "rate_limit_per_minute": 100,
  "created_at": "2024-10-26T10:00:00Z"
}
```

**Using an API Key:**

```bash
# Via X-API-Key header (recommended)
curl http://localhost:8080/api/v1/tables/users \
  -H "X-API-Key: fbk_live_abc123..."

# Via query parameter (not recommended for production)
curl "http://localhost:8080/api/v1/tables/users?apikey=fbk_live_abc123..."
```

### 2. Service Role Keys

Service role keys provide **elevated privileges** that bypass user-level RLS policies. They are intended for:

- Backend services
- Cron jobs
- Admin scripts
- CI/CD pipelines
- Server-to-server communication

**⚠️ Security Warning**: Service keys bypass RLS and have full database access. Store them securely and never expose them to clients.

**Creating a Service Key:**

Service keys are created directly in the database by an admin:

```sql
-- Generate a service key
INSERT INTO auth.service_keys (name, description, key_hash, key_prefix, scopes, enabled)
VALUES (
  'Backend Service',
  'Key for backend API service',
  -- Hash your key with bcrypt (cost 12)
  crypt('sk_live_your_random_key_here', gen_salt('bf', 12)),
  'sk_live_',  -- First 8 chars of the key
  ARRAY[]::TEXT[],  -- Empty array = full access
  true
);
```

**Key Format**: `sk_{environment}_{random}` (e.g., `sk_live_abc123xyz...`)

**Using a Service Key:**

```bash
# Via X-Service-Key header (recommended)
curl http://localhost:8080/api/v1/tables/users \
  -H "X-Service-Key: sk_live_abc123..."

# Via Authorization header
curl http://localhost:8080/api/v1/tables/users \
  -H "Authorization: ServiceKey sk_live_abc123..."
```

### 3. JWT Tokens (User Sessions)

JWT tokens are obtained through user login and represent authenticated user sessions.

```bash
# Standard user authentication
curl http://localhost:8080/api/v1/tables/users \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

### Authentication Priority

When multiple authentication methods are provided, Fluxbase checks them in this order:

1. **Service Key** (highest privilege)
2. **JWT Token**
3. **API Key**

If a service key is provided, other auth methods are ignored.

### Comparison

| Feature             | JWT Token            | User API Key        | Service Key          |
| ------------------- | -------------------- | ------------------- | -------------------- |
| **Created by**      | User login           | Authenticated user  | Database admin       |
| **Lifespan**        | 15 min (default)     | Until revoked       | Until revoked        |
| **RLS Enforcement** | ✅ Yes               | ✅ Yes              | ❌ No (bypasses RLS) |
| **Use Case**        | Web/mobile apps      | Programmatic access | Backend services     |
| **Privileges**      | User's permissions   | User's permissions  | Full database access |
| **Rotation**        | Automatic            | Manual              | Manual               |
| **Client-safe**     | ✅ Yes (short-lived) | ⚠️ Depends          | ❌ Never expose      |

### SDK Usage

**TypeScript:**

```typescript
import { FluxbaseClient } from "@fluxbase/client";

// Using JWT (after login)
const client = new FluxbaseClient({
  url: "http://localhost:8080",
});
await client.auth.signIn({ email, password });

// Using API Key
const client = new FluxbaseClient({
  url: "http://localhost:8080",
  apiKey: "fbk_live_abc123...",
});

// Using Service Key (backend only!)
const client = new FluxbaseClient({
  url: "http://localhost:8080",
  serviceKey: "sk_live_abc123...",
});
```

### Best Practices

**API Keys:**

- ✅ Use for mobile apps and SPAs where you need persistent auth
- ✅ Set appropriate scopes to limit permissions
- ✅ Rotate keys regularly
- ✅ Use separate keys for dev/staging/production
- ❌ Don't commit keys to version control
- ❌ Don't log keys in plaintext

**Service Keys:**

- ✅ Use only in backend services (never in clients)
- ✅ Store in secure secrets management (e.g., AWS Secrets Manager, HashiCorp Vault)
- ✅ Use environment variables, never hardcode
- ✅ Monitor usage via `last_used_at` timestamp
- ✅ Set expiration dates where possible
- ❌ Never expose service keys in client code
- ❌ Never commit to version control
- ❌ Don't use in frontend applications

**Example: Environment Variables**

```bash
# .env (never commit this file!)
FLUXBASE_URL=http://localhost:8080
FLUXBASE_SERVICE_KEY=sk_live_abc123xyz...

# In your backend code:
const client = new FluxbaseClient({
  url: process.env.FLUXBASE_URL,
  serviceKey: process.env.FLUXBASE_SERVICE_KEY,
});
```

### Managing API Keys

**List your API keys:**

```bash
curl http://localhost:8080/api/v1/api-keys \
  -H "Authorization: Bearer {your_jwt_token}"
```

**Revoke an API key:**

```bash
curl -X DELETE http://localhost:8080/api/v1/api-keys/{key_id} \
  -H "Authorization: Bearer {your_jwt_token}"
```

**Update API key:**

```bash
curl -X PATCH http://localhost:8080/api/v1/api-keys/{key_id} \
  -H "Authorization: Bearer {your_jwt_token}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Updated name",
    "scopes": ["read:data"]
  }'
```

---

## Multi-Factor Authentication (Coming Soon)

Support for TOTP-based 2FA will be added in a future release.
