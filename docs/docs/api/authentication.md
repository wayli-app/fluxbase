# Authentication API

Fluxbase provides a complete authentication system with email/password, magic links, and JWT tokens. All authentication endpoints are under `/api/auth`.

## Endpoints Overview

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| POST | `/api/auth/signup` | Register a new user | No |
| POST | `/api/auth/signin` | Sign in with email and password | No |
| POST | `/api/auth/signout` | Sign out and invalidate session | Yes |
| POST | `/api/auth/refresh` | Refresh access token | No |
| GET | `/api/auth/user` | Get current user profile | Yes |
| PATCH | `/api/auth/user` | Update user profile | Yes |
| POST | `/api/auth/magiclink` | Send magic link email | No |
| POST | `/api/auth/magiclink/verify` | Verify magic link token | No |

## Sign Up

Register a new user account.

**Endpoint:** `POST /api/auth/signup`

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "securepassword123",
  "metadata": {
    "name": "John Doe",
    "company": "Acme Inc"
  }
}
```

**Response (201 Created):**
```json
{
  "user": {
    "id": "123e4567-e89b-12d3-a456-426614174000",
    "email": "user@example.com",
    "email_verified": false,
    "role": "user",
    "metadata": {
      "name": "John Doe",
      "company": "Acme Inc"
    },
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T10:30:00Z"
  },
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 900
}
```

**Error Responses:**
- `400 Bad Request`: Invalid request body or validation error
- `409 Conflict`: Email already exists

**Notes:**
- Password must meet minimum length requirement (default: 8 characters)
- Access token expires in 15 minutes (default)
- Refresh token expires in 7 days (default)
- Metadata field is optional and can contain any JSON data

---

## Sign In

Authenticate with email and password.

**Endpoint:** `POST /api/auth/signin`

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "securepassword123"
}
```

**Response (200 OK):**
```json
{
  "user": {
    "id": "123e4567-e89b-12d3-a456-426614174000",
    "email": "user@example.com",
    "email_verified": false,
    "role": "user",
    "metadata": {
      "name": "John Doe"
    },
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T10:30:00Z"
  },
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 900
}
```

**Error Responses:**
- `400 Bad Request`: Missing email or password
- `401 Unauthorized`: Invalid email or password

---

## Sign Out

Sign out and invalidate the current session.

**Endpoint:** `POST /api/auth/signout`

**Headers:**
```
Authorization: Bearer <access_token>
```

**Response (200 OK):**
```json
{
  "message": "Successfully signed out"
}
```

**Error Responses:**
- `400 Bad Request`: Missing Authorization header
- `500 Internal Server Error`: Failed to sign out

**Notes:**
- This endpoint invalidates the session in the database
- The access token will no longer be valid for subsequent requests
- Client should discard both access and refresh tokens

---

## Refresh Token

Get a new access token using a refresh token.

**Endpoint:** `POST /api/auth/refresh`

**Request Body:**
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Response (200 OK):**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 900
}
```

**Error Responses:**
- `400 Bad Request`: Missing refresh token
- `401 Unauthorized`: Invalid or expired refresh token

**Notes:**
- The refresh token remains the same (not rotated)
- Only the access token is refreshed
- Session must still exist in the database

---

## Get User

Get the current user's profile.

**Endpoint:** `GET /api/auth/user`

**Headers:**
```
Authorization: Bearer <access_token>
```

**Response (200 OK):**
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "email": "user@example.com",
  "email_verified": false,
  "role": "user",
  "metadata": {
    "name": "John Doe",
    "company": "Acme Inc"
  },
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

**Error Responses:**
- `401 Unauthorized`: Missing or invalid access token
- `401 Unauthorized`: Session not found or expired

---

## Update User

Update the current user's profile.

**Endpoint:** `PATCH /api/auth/user`

**Headers:**
```
Authorization: Bearer <access_token>
```

**Request Body:**
```json
{
  "email": "newemail@example.com",
  "password": "newsecurepassword123",
  "metadata": {
    "name": "Jane Doe",
    "company": "New Company"
  }
}
```

**Response (200 OK):**
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "email": "newemail@example.com",
  "email_verified": false,
  "role": "user",
  "metadata": {
    "name": "Jane Doe",
    "company": "New Company"
  },
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T11:45:00Z"
}
```

**Error Responses:**
- `400 Bad Request`: Invalid request body or validation error
- `401 Unauthorized`: Not authenticated

**Notes:**
- All fields are optional - only provided fields will be updated
- Password changes require the new password to meet validation rules
- Email changes will reset email_verified to false

---

## Send Magic Link

Request a passwordless magic link sign-in email.

**Endpoint:** `POST /api/auth/magiclink`

**Request Body:**
```json
{
  "email": "user@example.com"
}
```

**Response (200 OK):**
```json
{
  "message": "Magic link sent to your email"
}
```

**Error Responses:**
- `400 Bad Request`: Missing or invalid email
- `400 Bad Request`: Magic link authentication is disabled

**Notes:**
- Magic link expires in 15 minutes (default)
- If the email doesn't exist, a new user account will be created upon verification
- Email must be enabled in configuration (`FLUXBASE_EMAIL_ENABLED=true`)

---

## Verify Magic Link

Verify a magic link token and sign in.

**Endpoint:** `POST /api/auth/magiclink/verify`

**Request Body:**
```json
{
  "token": "abc123def456..."
}
```

**Response (200 OK):**
```json
{
  "user": {
    "id": "123e4567-e89b-12d3-a456-426614174000",
    "email": "user@example.com",
    "email_verified": true,
    "role": "user",
    "metadata": null,
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T10:30:00Z"
  },
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 900
}
```

**Error Responses:**
- `400 Bad Request`: Missing or invalid token
- `400 Bad Request`: Token expired or already used

**Notes:**
- Creates a new user account if email doesn't exist
- Marks email as verified automatically
- Token can only be used once

---

## JWT Token Structure

Access and refresh tokens are JWTs with the following claims:

```json
{
  "user_id": "123e4567-e89b-12d3-a456-426614174000",
  "email": "user@example.com",
  "role": "user",
  "type": "access",
  "exp": 1705318200,
  "iat": 1705317300
}
```

**Claims:**
- `user_id`: User's unique identifier
- `email`: User's email address
- `role`: User's role (e.g., "user", "admin")
- `type`: Token type ("access" or "refresh")
- `exp`: Expiration timestamp (Unix)
- `iat`: Issued at timestamp (Unix)

---

## Configuration

Authentication behavior can be configured via environment variables:

```bash
# JWT Configuration
FLUXBASE_AUTH_JWT_SECRET=your-secret-key-change-in-production
FLUXBASE_AUTH_JWT_EXPIRY=15m
FLUXBASE_AUTH_REFRESH_EXPIRY=168h

# Password Requirements
FLUXBASE_AUTH_PASSWORD_MIN_LENGTH=8
FLUXBASE_AUTH_BCRYPT_COST=10

# Feature Toggles
FLUXBASE_AUTH_ENABLE_SIGNUP=true
FLUXBASE_AUTH_ENABLE_MAGIC_LINK=true

# Base URL (for magic link emails)
FLUXBASE_BASE_URL=http://localhost:8080

# Email Configuration (required for magic links)
FLUXBASE_EMAIL_ENABLED=true
FLUXBASE_EMAIL_PROVIDER=smtp
FLUXBASE_EMAIL_FROM_ADDRESS=noreply@example.com
```

See [Configuration Guide](/docs/configuration) for complete details.

---

## Example: Complete Auth Flow

```bash
# 1. Sign up
curl -X POST http://localhost:8080/api/auth/signup \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "securepassword123"
  }'

# Response includes access_token and refresh_token
# {
#   "user": {...},
#   "access_token": "eyJ...",
#   "refresh_token": "eyJ...",
#   "expires_in": 900
# }

# 2. Get user profile
curl http://localhost:8080/api/auth/user \
  -H "Authorization: Bearer eyJ..."

# 3. Update profile
curl -X PATCH http://localhost:8080/api/auth/user \
  -H "Authorization: Bearer eyJ..." \
  -H "Content-Type: application/json" \
  -d '{
    "metadata": {
      "name": "John Doe"
    }
  }'

# 4. Refresh token (when access token expires)
curl -X POST http://localhost:8080/api/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "eyJ..."
  }'

# 5. Sign out
curl -X POST http://localhost:8080/api/auth/signout \
  -H "Authorization: Bearer eyJ..."
```

---

## Security Best Practices

1. **Always use HTTPS in production** - JWT tokens should never be transmitted over unencrypted connections
2. **Store tokens securely** - Use httpOnly cookies or secure storage, never localStorage
3. **Implement token rotation** - Consider rotating refresh tokens on each use
4. **Set appropriate expiration times** - Short-lived access tokens (15min), longer refresh tokens (7d)
5. **Validate on every request** - Always check session existence, not just JWT validity
6. **Use strong JWT secrets** - Generate a cryptographically secure random string (32+ characters)
7. **Monitor for suspicious activity** - Track failed login attempts, unusual access patterns
8. **Implement rate limiting** - Prevent brute force attacks on authentication endpoints

---

## Next Steps

- [Row Level Security (RLS)](/docs/security/rls) - Secure your data with PostgreSQL RLS
- [API Keys](/docs/security/api-keys) - Authenticate server-to-server requests
- [OAuth2 Integration](/docs/authentication/oauth) - Social login with Google, GitHub, etc.
