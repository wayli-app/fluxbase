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

---

## Using the SDK (Recommended)

### Installation

**TypeScript/JavaScript:**

```bash
npm install @fluxbase/sdk
```

**Python:**

```bash
pip install fluxbase
```

### Quick Start

#### TypeScript/JavaScript

```typescript
import { FluxbaseClient } from "@fluxbase/sdk";

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
const user = await client.auth.getCurrentUser();

// Sign out
await client.auth.signOut();
```

#### Python

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

#### React

```tsx
import { useState } from "react";
import { FluxbaseClient } from "@fluxbase/sdk";

const client = new FluxbaseClient({ url: "http://localhost:8080" });

function AuthForm() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [user, setUser] = useState(null);

  const handleSignUp = async (e) => {
    e.preventDefault();
    const { user } = await client.auth.signUp({ email, password });
    setUser(user);
  };

  const handleSignIn = async (e) => {
    e.preventDefault();
    const { user } = await client.auth.signIn({ email, password });
    setUser(user);
  };

  const handleSignOut = async () => {
    await client.auth.signOut();
    setUser(null);
  };

  if (user) {
    return (
      <div>
        <p>Welcome, {user.email}!</p>
        <button onClick={handleSignOut}>Sign Out</button>
      </div>
    );
  }

  return (
    <form>
      <input
        type="email"
        value={email}
        onChange={(e) => setEmail(e.target.value)}
        placeholder="Email"
      />
      <input
        type="password"
        value={password}
        onChange={(e) => setPassword(e.target.value)}
        placeholder="Password"
      />
      <button onClick={handleSignUp}>Sign Up</button>
      <button onClick={handleSignIn}>Sign In</button>
    </form>
  );
}
```

---

### Core Authentication

#### Sign Up (Create Account)

Create a new user account with email and password.

**TypeScript:**

```typescript
const { user, session } = await client.auth.signUp({
  email: "user@example.com",
  password: "SecurePassword123",
});

console.log("User created:", user.id);
console.log("Access token:", session.access_token);
```

**Python:**

```python
response = client.auth.sign_up(
    email="user@example.com",
    password="SecurePassword123"
)

print(f"User created: {response['user']['id']}")
print(f"Access token: {response['access_token']}")
```

**React:**

```tsx
function SignUpForm() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");

  const handleSubmit = async (e) => {
    e.preventDefault();
    try {
      const { user } = await client.auth.signUp({ email, password });
      console.log("Welcome,", user.email);
    } catch (err) {
      setError(err.message);
    }
  };

  return (
    <form onSubmit={handleSubmit}>
      {error && <p style={{ color: "red" }}>{error}</p>}
      <input
        type="email"
        value={email}
        onChange={(e) => setEmail(e.target.value)}
        placeholder="Email"
        required
      />
      <input
        type="password"
        value={password}
        onChange={(e) => setPassword(e.target.value)}
        placeholder="Password"
        required
      />
      <button type="submit">Sign Up</button>
    </form>
  );
}
```

#### Sign In (Login)

Authenticate with email and password.

**TypeScript:**

```typescript
const { user, session } = await client.auth.signIn({
  email: "user@example.com",
  password: "SecurePassword123",
});

console.log("Logged in as:", user.email);
console.log("Session expires:", session.expires_at);
```

**Python:**

```python
response = client.auth.sign_in(
    email="user@example.com",
    password="SecurePassword123"
)

print(f"Logged in as: {response['user']['email']}")
```

**React:**

```tsx
function SignInForm() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");

  const handleSubmit = async (e) => {
    e.preventDefault();
    try {
      const { user } = await client.auth.signIn({ email, password });
      window.location.href = "/dashboard";
    } catch (err) {
      alert("Invalid credentials");
    }
  };

  return (
    <form onSubmit={handleSubmit}>
      <input
        type="email"
        value={email}
        onChange={(e) => setEmail(e.target.value)}
        placeholder="Email"
        required
      />
      <input
        type="password"
        value={password}
        onChange={(e) => setPassword(e.target.value)}
        placeholder="Password"
        required
      />
      <button type="submit">Sign In</button>
    </form>
  );
}
```

#### Sign Out (Logout)

Invalidate the current session.

**TypeScript:**

```typescript
await client.auth.signOut();
console.log("Signed out successfully");
```

**Python:**

```python
client.auth.sign_out()
print("Signed out successfully")
```

#### Get Current User

Retrieve the currently authenticated user's profile.

**TypeScript:**

```typescript
const user = await client.auth.getCurrentUser();
console.log("Current user:", user.email);
console.log("User ID:", user.id);
console.log("Role:", user.role);
```

**Python:**

```python
user = client.auth.user()
print(f"Current user: {user['email']}")
print(f"User ID: {user['id']}")
```

**React Hook:**

```tsx
import { useState, useEffect } from "react";

function useCurrentUser() {
  const [user, setUser] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    client.auth
      .getCurrentUser()
      .then(setUser)
      .catch(() => setUser(null))
      .finally(() => setLoading(false));
  }, []);

  return { user, loading };
}

// Usage
function Profile() {
  const { user, loading } = useCurrentUser();

  if (loading) return <p>Loading...</p>;
  if (!user) return <p>Not logged in</p>;

  return (
    <div>
      <h2>{user.email}</h2>
      <p>Role: {user.role}</p>
    </div>
  );
}
```

#### Update User Profile

Update the current user's email or metadata.

**TypeScript:**

```typescript
const updatedUser = await client.auth.updateUser({
  email: "newemail@example.com",
  metadata: {
    display_name: "John Doe",
    avatar_url: "https://example.com/avatar.jpg",
  },
});

console.log("Updated user:", updatedUser.email);
```

**Python:**

```python
updated_user = client.auth.update_user(
    email="newemail@example.com",
    metadata={
        "display_name": "John Doe",
        "avatar_url": "https://example.com/avatar.jpg"
    }
)

print(f"Updated user: {updated_user['email']}")
```

**React:**

```tsx
function UpdateProfileForm() {
  const [displayName, setDisplayName] = useState("");

  const handleSubmit = async (e) => {
    e.preventDefault();
    await client.auth.updateUser({
      metadata: { display_name: displayName },
    });
    alert("Profile updated!");
  };

  return (
    <form onSubmit={handleSubmit}>
      <input
        type="text"
        value={displayName}
        onChange={(e) => setDisplayName(e.target.value)}
        placeholder="Display Name"
      />
      <button type="submit">Update Profile</button>
    </form>
  );
}
```

#### Token Refresh

The SDK automatically handles token refresh. Access tokens are refreshed automatically when they expire using the refresh token.

**TypeScript (Automatic):**

```typescript
// The SDK automatically refreshes tokens when needed
const client = new FluxbaseClient({ url: "http://localhost:8080" });

// Your access token will be refreshed automatically on API calls
const user = await client.auth.getCurrentUser(); // Auto-refreshes if needed
```

**TypeScript (Manual):**

```typescript
// If you need to manually refresh the token
const { access_token, expires_at } = await client.auth.refreshSession();

console.log("New access token:", access_token);
console.log("Expires at:", expires_at);
```

**Python:**

```python
# The SDK automatically refreshes tokens
# To manually refresh:
session = client.auth.refresh_session()
print(f"New access token: {session['access_token']}")
```

---

### Password Management

#### Password Reset Flow

Fluxbase provides a secure password reset flow that sends a time-limited token to the user's email.

**How It Works:**

1. **User requests password reset** - User provides their email address
2. **Email sent** - If the email exists, a reset token is sent (system doesn't reveal if email exists)
3. **Token verification** - (Optional) Verify the token is valid before showing password form
4. **Password reset** - User provides the token and new password

**TypeScript:**

```typescript
// Step 1: Request password reset
await client.auth.sendPasswordReset("user@example.com");

// Step 2: Verify reset token (optional)
const { valid, message } = await client.auth.verifyResetToken(token);

if (valid) {
  // Step 3: Reset password with token
  await client.auth.resetPassword(token, "NewSecurePassword123");
}
```

**Python:**

```python
# Step 1: Request password reset
client.auth.send_password_reset("user@example.com")

# Step 2: Verify token (optional)
result = client.auth.verify_reset_token(token)

if result['valid']:
    # Step 3: Reset password
    client.auth.reset_password(token, "NewSecurePassword123")
```

**React Complete Example:**

```tsx
import { useState } from "react";
import { FluxbaseClient } from "@fluxbase/sdk";

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

**Security Considerations:**

- **No user enumeration** - API doesn't reveal if email exists in database
- **Time-limited tokens** - Reset tokens expire after configured time (default: 1 hour)
- **One-time use** - Tokens can only be used once
- **Secure token generation** - Uses cryptographically secure random token
- **Password requirements** - New password must meet minimum requirements

---

### Passwordless Authentication

#### Magic Link

Magic links provide a passwordless authentication method. Users receive an email with a one-time login link.

**TypeScript:**

```typescript
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

**Python:**

```python
# Request magic link
client.auth.send_magic_link(
    email="user@example.com",
    redirect_to="http://localhost:3000/auth/callback"
)

# Verify magic link (in callback handler)
token = request.args.get('token')
session = client.auth.verify_magic_link(token)
```

**React Complete Example:**

```tsx
import { useState, useEffect } from "react";
import { useSearchParams, useNavigate } from "react-router-dom";
import { FluxbaseClient } from "@fluxbase/sdk";

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

**Security Considerations:**

- Links expire after a configurable time (default: 15 minutes)
- One-time use only - cannot be reused
- Sent to verified email address
- Include CSRF protection via state parameter

#### Anonymous Authentication

Anonymous authentication allows users to access your application without creating an account. This is useful for:

- Guest checkout in e-commerce
- Trial/demo access
- Gaming sessions
- Shopping carts before checkout
- Converting anonymous users to registered users later

**TypeScript:**

```typescript
// Sign in anonymously
const session = await client.auth.signInAnonymously();
console.log("Anonymous user ID:", session.user.id);

// Later, convert to registered user
await client.auth.signUp({
  email: "user@example.com",
  password: "SecurePassword123",
});
// Cart items will be preserved with the same user_id
```

**Python:**

```python
# Sign in anonymously
session = client.auth.sign_in_anonymously()
print(f"Anonymous user ID: {session['user']['id']}")

# Convert to registered user
client.auth.sign_up(
    email="user@example.com",
    password="SecurePassword123"
)
```

**React Guest Checkout Example:**

```tsx
import { useState, useEffect } from "react";
import { FluxbaseClient } from "@fluxbase/sdk";

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

**Security Considerations:**

- **Limited permissions** - Anonymous users should have restricted RLS policies
- **Time-limited sessions** - Anonymous sessions expire after configured time
- **Data cleanup** - Old anonymous user data should be cleaned up periodically
- **Rate limiting** - Apply stricter rate limits to anonymous users

---

### Social Login (OAuth)

Fluxbase supports OAuth authentication with multiple providers, allowing users to sign in with their existing accounts.

#### Supported OAuth Providers

- **Google** - Sign in with Google
- **GitHub** - Sign in with GitHub
- **Microsoft** - Sign in with Microsoft/Azure AD
- **Apple** - Sign in with Apple
- **Facebook** - Sign in with Facebook
- **Twitter** - Sign in with Twitter
- **LinkedIn** - Sign in with LinkedIn
- **GitLab** - Sign in with GitLab
- **Bitbucket** - Sign in with Bitbucket

#### OAuth Flow

**TypeScript:**

```typescript
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

**Python:**

```python
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

**React Complete Example:**

```tsx
import { useEffect } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { FluxbaseClient } from "@fluxbase/sdk";

const client = new FluxbaseClient({ url: "http://localhost:8080" });

// Login page
function LoginPage() {
  const handleGoogleLogin = async () => {
    await client.auth.signInWithOAuth("google", {
      redirect_to: window.location.origin + "/auth/callback",
    });
  };

  const handleGitHubLogin = async () => {
    await client.auth.signInWithOAuth("github", {
      redirect_to: window.location.origin + "/auth/callback",
    });
  };

  return (
    <div>
      <button onClick={handleGoogleLogin}>Sign in with Google</button>
      <button onClick={handleGitHubLogin}>Sign in with GitHub</button>
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

**Security Considerations:**

- State parameter prevents CSRF attacks
- Tokens are validated before exchanging for session
- Provider tokens are not stored
- User data is fetched fresh on each OAuth login

---

### API Keys & Service Authentication

Fluxbase supports three types of authentication for programmatic access:

1. **JWT Tokens** - User sessions (recommended for web/mobile apps)
2. **User API Keys** - Long-lived keys tied to user accounts
3. **Service Keys** - Elevated privileges for backend services

#### Using API Keys

User API keys are tied to specific user accounts and inherit the user's permissions and RLS policies.

**TypeScript:**

```typescript
// Initialize client with API key
const client = new FluxbaseClient({
  url: "http://localhost:8080",
  apiKey: "fbk_live_abc123...",
});

// Now all requests use the API key
const data = await client.from("users").select("*");
```

**Python:**

```python
# Initialize client with API key
client = FluxbaseClient(
    url="http://localhost:8080",
    api_key="fbk_live_abc123..."
)

# All requests use the API key
data = client.table("users").select("*").execute()
```

#### Using Service Keys (Backend Only)

Service role keys provide **elevated privileges** that bypass user-level RLS policies.

**⚠️ Security Warning**: Service keys bypass RLS and have full database access. Store them securely and never expose them to clients.

**TypeScript (Backend Only):**

```typescript
// ONLY use in backend/server code - NEVER in frontend!
const client = new FluxbaseClient({
  url: "http://localhost:8080",
  serviceKey: process.env.FLUXBASE_SERVICE_KEY, // Use environment variable
});

// Service key bypasses RLS - full database access
const data = await client.from("users").select("*");
```

**Python (Backend Only):**

```python
import os

# ONLY use in backend code
client = FluxbaseClient(
    url="http://localhost:8080",
    service_key=os.environ['FLUXBASE_SERVICE_KEY']  # From environment
)

# Full database access
data = client.table("users").select("*").execute()
```

#### Authentication Comparison

| Feature             | JWT Token            | User API Key        | Service Key          |
| ------------------- | -------------------- | ------------------- | -------------------- |
| **Created by**      | User login           | Authenticated user  | Database admin       |
| **Lifespan**        | 15 min (default)     | Until revoked       | Until revoked        |
| **RLS Enforcement** | ✅ Yes               | ✅ Yes              | ❌ No (bypasses RLS) |
| **Use Case**        | Web/mobile apps      | Programmatic access | Backend services     |
| **Privileges**      | User's permissions   | User's permissions  | Full database access |
| **Rotation**        | Automatic            | Manual              | Manual               |
| **Client-safe**     | ✅ Yes (short-lived) | ⚠️ Depends          | ❌ Never expose      |

#### Best Practices

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
- ❌ Never expose service keys in client code
- ❌ Never commit to version control
- ❌ Don't use in frontend applications

---

## Advanced: REST API Reference

For direct HTTP access or custom integrations, Fluxbase provides a complete REST API.

### Core Authentication Endpoints

#### POST /api/v1/auth/signup

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

#### POST /api/v1/auth/signin

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

#### POST /api/v1/auth/signout

Invalidate the current session.

**Headers:**

```
Authorization: Bearer {access_token}
```

**Response (204 No Content)**

**Errors:**

- `401 Unauthorized` - Invalid or missing token

---

#### POST /api/v1/auth/refresh

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

#### GET /api/v1/auth/user

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

#### PATCH /api/v1/auth/user

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

### Password Management Endpoints

#### POST /api/v1/auth/password/reset

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

#### POST /api/v1/auth/password/reset/verify

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

#### POST /api/v1/auth/password/reset/confirm

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

### Passwordless Authentication Endpoints

#### POST /api/v1/auth/magiclink

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

#### POST /api/v1/auth/magiclink/verify

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

#### POST /api/v1/auth/signin/anonymous

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

### OAuth Endpoints

#### GET /api/v1/auth/oauth/providers

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

#### GET /api/v1/auth/oauth/:provider/authorize

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

#### POST /api/v1/auth/oauth/callback

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

---

### API Key Management Endpoints

#### POST /api/v1/api-keys

Create a new API key.

**Headers:**

```
Authorization: Bearer {your_jwt_token}
```

**Request Body:**

```json
{
  "name": "My Application Key",
  "description": "API key for my mobile app",
  "scopes": ["read:data", "write:data"],
  "rate_limit_per_minute": 100
}
```

**Response (201 Created):**

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

---

#### GET /api/v1/api-keys

List your API keys.

**Headers:**

```
Authorization: Bearer {your_jwt_token}
```

**Response (200 OK):**

```json
{
  "keys": [
    {
      "id": "uuid",
      "name": "My Application Key",
      "key_prefix": "fbk_live",
      "scopes": ["read:data", "write:data"],
      "last_used_at": "timestamp",
      "created_at": "timestamp"
    }
  ]
}
```

---

#### PATCH /api/v1/api-keys/:id

Update an API key.

**Headers:**

```
Authorization: Bearer {your_jwt_token}
```

**Request Body:**

```json
{
  "name": "Updated name",
  "scopes": ["read:data"]
}
```

**Response (200 OK):**

```json
{
  "id": "uuid",
  "name": "Updated name",
  "scopes": ["read:data"],
  ...
}
```

---

#### DELETE /api/v1/api-keys/:id

Revoke an API key.

**Headers:**

```
Authorization: Bearer {your_jwt_token}
```

**Response (204 No Content)**

---

## Reference

### JWT Token Structure

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

### Database Schema

#### Users Table

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

#### Sessions Table

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

### Security Best Practices

#### Password Storage

- Passwords are hashed using bcrypt with configurable cost factor (default: 12)
- Password hashes are never exposed in API responses
- Supports automatic hash upgrades when cost factor changes

#### Token Security

- JWT tokens are signed with HMAC-SHA256
- Access tokens are short-lived (default: 15 minutes)
- Refresh tokens are long-lived (default: 7 days)
- Both token types must be validated on each request

#### Session Management

- Each login creates a new session with unique session ID
- Sessions can be invalidated on logout
- Concurrent sessions are supported
- Session tracking helps identify active logins

### Troubleshooting

#### "Invalid or expired token"

- Access tokens expire after 15 minutes by default
- Use the refresh token to get a new access token
- Check that you're sending the token in the Authorization header

#### "Email already registered"

- The email is already in use
- Try logging in instead of signing up
- Use password reset if you forgot your password

#### "Weak password"

- Password must meet minimum length requirement (default: 8 characters)
- Check for additional password requirements in your configuration

#### "Session expired"

- Both access and refresh tokens have expired
- User must log in again
- Consider increasing refresh token expiration time

### Multi-Factor Authentication

Support for TOTP-based 2FA will be added in a future release.

---

## Next Steps

- [Row Level Security](row-level-security) - Secure your data with RLS policies
- [Database Operations](/docs/guides/typescript-sdk/database) - Query your data with the SDK
- [Realtime](realtime) - Subscribe to real-time database changes
