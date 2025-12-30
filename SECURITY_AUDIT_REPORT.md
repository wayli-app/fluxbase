# Fluxbase Security Audit Report

**Date:** December 30, 2025
**Auditor:** Claude Code
**Scope:** Authentication, data access, sensitive data exposure, authorization

---

## Executive Summary

The Fluxbase codebase demonstrates **solid security practices overall** with defense-in-depth implementation. The authentication, authorization, and data access controls are well-implemented with appropriate safeguards. However, there are a few areas that require attention for production deployments.

### Risk Rating Overview

| Category | Rating | Notes |
|----------|--------|-------|
| Authentication | ✅ Strong | JWT with proper validation, bcrypt passwords, MFA support |
| Authorization | ✅ Strong | RLS, scope-based access, role enforcement |
| SQL Injection | ✅ Strong | Parameterized queries, identifier validation |
| Secrets Management | ✅ Strong | AES-256-GCM encryption, never exposed via API |
| Rate Limiting | ⚠️ Moderate | In-memory per-instance (see recommendations) |
| Session Management | ✅ Strong | Tokens hashed before storage |
| SAML/OAuth | ✅ Strong | Proper validation, replay protection |

---

## Detailed Findings

### 1. Authentication Security

#### ✅ JWT Implementation (`internal/auth/jwt.go`)

**Strengths:**
- Uses HS256 signing with proper signature verification
- Validates token type (access vs refresh vs service_role)
- Strict issuer validation for service role tokens (only accepts "fluxbase")
- Token blacklisting/revocation support via `TokenBlacklistService`
- Role validation restricts to valid roles: `anon`, `authenticated`, `service_role`

**Code Reference:** `internal/auth/jwt.go:43-87`

#### ✅ Password Security (`internal/auth/password.go`)

**Strengths:**
- Bcrypt with cost factor 12 (appropriate for 2025)
- Minimum password length: 12 characters
- Maximum password length: 72 bytes (bcrypt limit enforcement)
- Requires uppercase, lowercase, and digit characters
- Password strength validation before hashing

**Code Reference:** `internal/auth/password.go:15-60`

#### ✅ Session Security (`internal/auth/session.go`)

**Strengths:**
- **Tokens are hashed with SHA-256 before storage** - This is excellent security practice
- Expired session validation
- Session cleanup functionality
- Refresh tokens properly rotated

**Code Reference:** `internal/auth/session.go:45-65`

#### ✅ Account Protection (`internal/auth/service.go`)

**Strengths:**
- Account lockout after 5 failed login attempts
- Email verification flow
- `app_metadata` stripped on signup to prevent privilege escalation
- Token blacklisting on logout

**Code Reference:** `internal/auth/service.go:182-220`

---

### 2. Authorization & Access Control

#### ✅ Row Level Security (`internal/middleware/rls.go`)

**Strengths:**
- Uses PostgreSQL `SET LOCAL ROLE` per transaction
- Only allows three validated database roles: `authenticated`, `anon`, `service_role`
- JWT claims passed as session variables (`app.user_id`, `app.rls_role`)
- RLS enforced at database layer via PostgreSQL policies
- RLS violation logging for audit trail

**Code Reference:** `internal/middleware/rls.go:30-85`

#### ✅ Scope-Based Authorization

**Strengths:**
- Scopes defined for all major operations (read/write for functions, storage, etc.)
- Middleware enforces scope requirements per endpoint
- Client keys have configurable scope restrictions

**Code Reference:** `internal/auth/scopes.go`, `internal/middleware/`

---

### 3. SQL Injection Prevention

#### ✅ Query Builder (`internal/api/query_builder.go`)

**Strengths:**
- All values use parameterized queries (`$1`, `$2`, etc.)
- Identifier validation via regex: `^[a-zA-Z_][a-zA-Z0-9_]*$`
- `quoteIdentifier()` function properly escapes double quotes for PostgreSQL
- Aggregations and filters use parameterized values

**Regex Validation:** Only allows alphanumeric characters and underscores, must start with letter or underscore. This prevents SQL injection in identifiers.

**Code Reference:** `internal/api/query_parser.go:14-31`

---

### 4. Secrets Management

#### ✅ Encryption (`internal/crypto/encrypt.go`)

**Strengths:**
- **AES-256-GCM** (authenticated encryption)
- 12-byte random nonce from `crypto/rand`
- Key length validation (exactly 32 bytes)
- HKDF for user-specific key derivation

**Code Reference:** `internal/crypto/encrypt.go:26-55`

#### ✅ Secret Storage (`internal/secrets/storage.go`)

**Strengths:**
- Encrypted values tagged with `json:"-"` (never exposed via API)
- Separate `SecretSummary` type for list responses (excludes encrypted value)
- Version history for audit trail
- Expiration support
- Namespace-scoped access

**Code Reference:** `internal/secrets/storage.go:15-28`

---

### 5. SAML Security

#### ✅ SAML Implementation (`internal/auth/saml.go`)

**Strengths:**
- **Assertion replay prevention** via database tracking
- **RelayState validation** with whitelisted hosts (blocks open redirects)
- **HTTPS enforcement** for metadata URLs (unless explicitly disabled)
- **Audience validation** - checks assertion is for this SP
- **IdP-initiated SSO controlled by flag** (default: disabled)
- **XSS sanitization** of SAML attribute values
- Proper time condition validation (NotBefore, NotOnOrAfter)

**Code Reference:** `internal/auth/saml.go:434-527` (ParseAssertion), `internal/auth/saml.go:1082-1123` (ValidateRelayState)

---

### 6. OAuth Security

#### ✅ OAuth Implementation (`internal/auth/oauth.go`)

**Strengths:**
- 32-byte random state for CSRF protection
- State validation includes time-based expiry (10 minutes)
- Uses `crypto/rand` for secure random generation
- State tokens deleted after single use (prevents replay)

**Code Reference:** `internal/auth/oauth.go:210-255`

---

### 7. WebSocket/Realtime Security

#### ✅ Realtime Handler (`internal/realtime/handler.go`)

**Strengths:**
- JWT validation required for all subscriptions
- Admin-only access for `subscribe_all_logs`
- Execution ownership verification for log subscriptions
- Same "not found" error for unauthorized access (prevents enumeration)

**Code Reference:** `internal/realtime/handler.go:246-254`, `internal/realtime/handler.go:689-724`

---

### 8. Storage Security

#### ✅ Signed URLs (`internal/api/storage_signed.go`)

**Strengths:**
- Rate limiting: 100 requests per minute per IP
- Token validation with expiry
- HTTP method validation (token only valid for specified method)
- HMAC-based signature verification

**Code Reference:** `internal/api/storage_signed.go:16-58`, `internal/api/storage_signed.go:112-187`

---

### 9. Edge Functions Security

#### ✅ Functions Handler (`internal/functions/handler.go`)

**Strengths:**
- Authentication middleware enforced
- Scope-based authorization (`execute:functions`, `read:functions`)
- Per-function rate limiting (per minute/hour/day)
- Sandboxed Deno runtime with permission controls
- Impersonation token rate limiting (5 per 5 min per IP)
- Admin-only for management endpoints (list, get, delete)

**Code Reference:** `internal/functions/handler.go:245-285`

---

### 10. Security Headers

#### ✅ Security Headers Middleware (`internal/middleware/security_headers.go`)

**Strengths:**
- **CSP:** `default-src 'self'` with strict script/style policies (API endpoints)
- **X-Frame-Options:** `DENY`
- **X-Content-Type-Options:** `nosniff`
- **HSTS:** 1 year with `includeSubDomains`
- **Referrer-Policy:** `strict-origin-when-cross-origin`
- **Permissions-Policy:** Disables geolocation, microphone, camera
- **Server header removed** to prevent information disclosure

**Code Reference:** `internal/middleware/security_headers.go:25-44`

---

## Areas Requiring Attention

### ⚠️ 1. Rate Limiting Is Per-Instance

**Location:** `internal/middleware/rate_limiter.go:23-32`

**Issue:** Rate limiting uses in-memory storage, which means:
- Each server instance maintains independent counters
- Attackers can bypass rate limits by targeting different instances

**Current Mitigation:** The code includes a documented security warning about this limitation.

**Recommendation:**
- For production multi-instance deployments, use a reverse proxy (nginx, Traefik) with centralized rate limiting
- Or implement Redis-backed rate limiting for distributed environments

---

### ⚠️ 2. CSRF Protection Is Per-Instance

**Location:** `internal/middleware/csrf.go`

**Issue:** CSRF token storage is in-memory per-instance.

**Recommendation:** For multi-instance deployments, consider Redis-backed CSRF token storage or use the existing cookie-based approach with proper SameSite settings (which is already implemented).

---

### ⚠️ 3. SQL Handler Role Setting

**Location:** `internal/api/sql_handler.go:257`

**Issue:** Uses `fmt.Sprintf("SET LOCAL ROLE %q", dbRole)` - the `%q` uses Go-style quotes, not PostgreSQL-style quotes.

**Mitigation:** The `isKnownDatabaseRole()` function limits valid roles to only `authenticated`, `anon`, `service_role`, which prevents injection. The whitelist approach is safe.

**Recommendation:** Consider using PostgreSQL-style quoting (`pq.QuoteIdentifier`) for consistency, though the current whitelist approach is secure.

---

### ⚠️ 4. service_role Tokens Bypass Rate Limiting for Migrations

**Location:** `internal/middleware/rate_limiter.go:340-342`

**Issue:** service_role JWT tokens bypass rate limiting entirely for the migrations API.

**Mitigation:** This is intentional behavior for trusted service keys.

**Recommendation:** Ensure service_role keys are properly secured and rotated. Consider adding audit logging for service_role API usage.

---

## Security Best Practices Confirmed

1. **Defense in Depth:** Multiple layers of security (JWT + RLS + application-level checks)
2. **Secure Defaults:** Most security features enabled by default
3. **Password Hashing:** Bcrypt with appropriate cost factor
4. **Token Security:** SHA-256 hashing of session tokens before storage
5. **Parameterized Queries:** Consistent use throughout the codebase
6. **Identifier Validation:** Strict regex whitelist for SQL identifiers
7. **Encryption at Rest:** AES-256-GCM for secrets
8. **XSS Prevention:** SAML attribute sanitization, CSP headers
9. **Open Redirect Prevention:** RelayState validation with host whitelist
10. **Replay Prevention:** SAML assertion ID tracking
11. **Rate Limiting:** Comprehensive rate limiting for auth endpoints
12. **Audit Logging:** Impersonation sessions logged

---

## Production Deployment Checklist

1. ✅ Use HTTPS in production (HSTS is enabled)
2. ⚠️ Configure centralized rate limiting if using multiple instances
3. ✅ Set strong JWT secret (minimum 32 bytes, use `crypto/rand`)
4. ✅ Set strong encryption key for secrets (exactly 32 bytes)
5. ✅ Review SAML `AllowIDPInitiated` setting (default: disabled)
6. ✅ Review `AllowInsecureMetadataURL` setting (default: disabled)
7. ✅ Configure allowed redirect hosts for SAML
8. ⚠️ Monitor rate limiting metrics
9. ✅ Rotate service_role keys periodically

---

## Conclusion

The Fluxbase codebase demonstrates **strong security practices** with appropriate authentication, authorization, and data protection mechanisms. The main considerations for production deployments relate to the per-instance nature of rate limiting and CSRF storage, which should be addressed with reverse proxy configurations or Redis-backed storage in horizontally-scaled environments.

The codebase follows security best practices including:
- Proper password hashing with bcrypt
- Secure token management with SHA-256 hashing
- Parameterized SQL queries
- Row-level security at the database layer
- Comprehensive input validation
- Encrypted secrets storage

No critical vulnerabilities were identified during this audit.
