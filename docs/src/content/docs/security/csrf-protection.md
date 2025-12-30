---
title: CSRF Protection
---

Cross-Site Request Forgery (CSRF) is an attack that tricks users into performing unwanted actions on a web application where they're authenticated. Fluxbase provides built-in CSRF protection to prevent these attacks.

## What is CSRF?

CSRF attacks exploit the trust a web application has in a user's browser. If a user is logged into your application, an attacker can trick their browser into making requests to your application without their knowledge.

### Example Attack Scenario

1. User logs into `https://yourapp.com`
2. User visits malicious site `https://evil.com`
3. Malicious site contains:
   ```html
   <img src="https://yourapp.com/api/v1/tables/users?delete=all" />
   ```
4. Browser automatically includes authentication cookies
5. Unintended action is performed

---

## How Fluxbase Prevents CSRF

Fluxbase implements the **Double-Submit Cookie** pattern:

1. Server generates a random CSRF token
2. Token is stored in:
   - HTTP-only cookie (not accessible to JavaScript)
   - Response header/body (for client to read)
3. Client includes token in subsequent requests
4. Server validates both tokens match

### Request Flow

```
Client                          Server
  |                               |
  |-- GET /api/v1/users --------->|
  |                               | Generate CSRF token
  |<------- Set-Cookie ----------| Set csrf_token cookie
  |                               |
  |-- POST /api/v1/users -------->|
  |    X-CSRF-Token: abc123       | Validate:
  |    Cookie: csrf_token=abc123  | - Cookie matches header
  |                               | - Token exists in storage
  |<------- 200 OK ---------------|
```

---

## Configuration

### Enable CSRF Protection

CSRF protection is **enabled by default** for state-changing methods (POST, PUT, PATCH, DELETE).

**Configuration via `fluxbase.yaml`:**

```yaml
security:
  csrf:
    enabled: true
    token_length: 32
    token_lookup: "header:X-CSRF-Token"
    cookie_name: "csrf_token"
    cookie_secure: true # Set to true in production
    cookie_http_only: true
    cookie_same_site: "Strict"
    expiration: "24h"
```

**Configuration via Environment Variables:**

```bash
FLUXBASE_SECURITY_CSRF_ENABLED=true
FLUXBASE_SECURITY_CSRF_TOKEN_LENGTH=32
FLUXBASE_SECURITY_CSRF_COOKIE_NAME=csrf_token
FLUXBASE_SECURITY_CSRF_COOKIE_SECURE=true
FLUXBASE_SECURITY_CSRF_COOKIE_SAME_SITE=Strict
```

### Configuration Options

| Option             | Default               | Description                                  |
| ------------------ | --------------------- | -------------------------------------------- |
| `enabled`          | `true`                | Enable/disable CSRF protection               |
| `token_length`     | `32`                  | Length of CSRF token in bytes                |
| `token_lookup`     | `header:X-CSRF-Token` | Where to find the token in requests          |
| `cookie_name`      | `csrf_token`          | Name of the CSRF cookie                      |
| `cookie_secure`    | `false`               | Mark cookie as HTTPS-only                    |
| `cookie_http_only` | `true`                | Prevent JavaScript access to cookie          |
| `cookie_same_site` | `Strict`              | SameSite attribute (`Strict`, `Lax`, `None`) |
| `expiration`       | `24h`                 | How long tokens are valid                    |

### SameSite Cookie Attributes

- **Strict**: Cookie only sent for same-site requests (most secure)
- **Lax**: Cookie sent for same-site requests and top-level navigation
- **None**: Cookie sent for all requests (requires `Secure` flag)

```yaml
# Recommended for most applications
cookie_same_site: "Strict"

# For cross-site authentication flows
cookie_same_site: "Lax"

# For third-party integrations (requires HTTPS)
cookie_same_site: "None"
cookie_secure: true
```

---

## Client Implementation

### Vanilla JavaScript/TypeScript

```typescript
// 1. Get CSRF token from cookie
function getCsrfToken(): string | null {
  const match = document.cookie.match(/csrf_token=([^;]+)/);
  return match ? match[1] : null;
}

// 2. Include token in requests
async function makeRequest(url: string, data: any) {
  const csrfToken = getCsrfToken();

  const response = await fetch(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "X-CSRF-Token": csrfToken || "", // Include CSRF token
      Authorization: `Bearer ${accessToken}`,
    },
    body: JSON.stringify(data),
    credentials: "include", // Include cookies
  });

  if (!response.ok) {
    throw new Error("Request failed");
  }

  return response.json();
}

// Example usage
makeRequest("/api/v1/tables/users", {
  name: "John Doe",
  email: "john@example.com",
});
```

### Fluxbase SDK (Automatic)

The Fluxbase SDK handles CSRF tokens automatically:

```typescript
import { createClient } from "@fluxbase/sdk";

const client = createClient("http://localhost:8080", "your-anon-key");

// CSRF token is automatically included
await client
  .from("users")
  .insert({
    name: "John Doe",
    email: "john@example.com",
  })
  .execute();
```

### React Example

```tsx
import { createClient } from "@fluxbase/sdk";

const client = createClient("http://localhost:8080", "your-anon-key");

function UserForm() {
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    // CSRF token is automatically included by the SDK
    const { error } = await client.from("users").insert({
      name: "John Doe",
      email: "john@example.com",
    });

    if (error) {
      console.error("Request failed:", error.message);
    }
  };

  return (
    <form onSubmit={handleSubmit}>
      {/* Form fields */}
      <button type="submit">Submit</button>
    </form>
  );
}
```

### Vue.js Example

```vue
<template>
  <form @submit.prevent="handleSubmit">
    <!-- Form fields -->
    <button type="submit">Submit</button>
  </form>
</template>

<script>
import { createClient } from "@fluxbase/sdk";

const client = createClient("http://localhost:8080", "your-anon-key");

export default {
  methods: {
    async handleSubmit() {
      // CSRF token is automatically included by the SDK
      const { error } = await client.from("users").insert({
        name: "John Doe",
        email: "john@example.com",
      });

      if (error) {
        console.error("Request failed:", error.message);
      }
    },
  },
};
</script>
```

### Axios Interceptor

```typescript
import axios from "axios";

// Create Axios instance
const api = axios.create({
  baseURL: "http://localhost:8080",
  withCredentials: true, // Include cookies
});

// Add CSRF token to all requests
api.interceptors.request.use((config) => {
  const match = document.cookie.match(/csrf_token=([^;]+)/);
  const csrfToken = match ? match[1] : null;

  if (csrfToken && config.headers) {
    config.headers["X-CSRF-Token"] = csrfToken;
  }

  return config;
});

// Usage
api.post("/api/v1/tables/users", {
  name: "John Doe",
  email: "john@example.com",
});
```

---

## Server-Side Implementation

### Custom Middleware (Go)

If you're building custom endpoints, use the CSRF middleware:

```go
package main

import (
    "github.com/gofiber/fiber/v2"
    "github.com/fluxbase-eu/fluxbase/internal/middleware"
)

func main() {
    app := fiber.New()

    // Apply CSRF middleware
    app.Use(middleware.CSRF(middleware.CSRFConfig{
        TokenLength:    32,
        TokenLookup:    "header:X-CSRF-Token",
        CookieName:     "csrf_token",
        CookieSecure:   true,
        CookieHTTPOnly: true,
        CookieSameSite: "Strict",
    }))

    // Your routes
    app.Post("/api/users", createUser)

    app.Listen(":8080")
}
```

### Excluded Paths

Some paths are automatically excluded from CSRF protection:

- **Safe methods**: GET, HEAD, OPTIONS
- **WebSocket endpoint**: `/realtime`
- **Health checks**: `/health`, `/ready`
- **Metrics**: `/metrics`
- **API requests with API key authentication**

---

## Testing CSRF Protection

### Manual Testing

**1. Test without CSRF token:**

```bash
# This should fail with 403 Forbidden
curl -X POST http://localhost:8080/api/v1/tables/users \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{"name":"John Doe"}'
```

**2. Test with valid CSRF token:**

```bash
# First, get the CSRF token (from browser cookies or initial request)
CSRF_TOKEN="your-csrf-token-here"

# This should succeed
curl -X POST http://localhost:8080/api/v1/tables/users \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "X-CSRF-Token: $CSRF_TOKEN" \
  -b "csrf_token=$CSRF_TOKEN" \
  -d '{"name":"John Doe"}'
```

### Automated Testing

```typescript
import { describe, it, expect } from "vitest";
import { createClient } from "@fluxbase/sdk";

describe("CSRF Protection", () => {
  it("should reject requests without CSRF token", async () => {
    const client = createClient("http://localhost:8080", "your-anon-key");

    await client.auth.signIn({
      email: "user@example.com",
      password: "password",
    });

    try {
      // Manually make request without CSRF token
      await fetch("http://localhost:8080/api/v1/tables/users", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${client.getAuthToken()}`,
        },
        body: JSON.stringify({ name: "John" }),
      });

      // Should not reach here
      expect(true).toBe(false);
    } catch (error) {
      expect(error.response.status).toBe(403);
    }
  });

  it("should accept requests with valid CSRF token", async () => {
    const client = createClient("http://localhost:8080", "your-anon-key");

    await client.auth.signIn({
      email: "user@example.com",
      password: "password",
    });

    // SDK automatically handles CSRF
    const { data, error } = await client
      .from("users")
      .insert({
        name: "John Doe",
      })
      .execute();

    expect(error).toBeNull();
    expect(data).toBeDefined();
  });
});
```

---

## Common Issues

### Issue: "CSRF token validation failed"

**Cause**: CSRF token missing or invalid

**Solutions**:

1. Ensure cookie is being sent:

   ```typescript
   fetch(url, {
     credentials: "include", // Include cookies
   });
   ```

2. Check cookie domain matches:

   ```yaml
   security:
     csrf:
       cookie_domain: ".yourdomain.com" # Allows subdomain.yourdomain.com
   ```

3. Verify token is included in header:
   ```typescript
   headers: {
     'X-CSRF-Token': csrfToken
   }
   ```

### Issue: "CSRF token expired"

**Cause**: Token older than configured expiration

**Solution**: The SDK handles token refresh automatically. If you encounter this issue, any SDK call will refresh the token:

```typescript
// Any SDK request will refresh the CSRF token
await client.admin.getHealth();
```

### Issue: CSRF with CORS

**Cause**: Cross-origin requests with credentials

**Solution**: Configure CORS properly:

```yaml
server:
  cors:
    allowed_origins:
      - "https://yourdomain.com"
    allow_credentials: true # Required for cookies
```

```typescript
// Client must include credentials
fetch(url, {
  credentials: "include", // Send cookies cross-origin
  headers: {
    "X-CSRF-Token": csrfToken,
  },
});
```

### Issue: CSRF in mobile apps

**Cause**: Mobile apps don't use cookies like browsers

**Solution**: Use API key authentication instead:

```typescript
// Mobile app using API key (no CSRF needed)
const client = createClient("https://api.yourdomain.com", "your-anon-key");
```

---

## Disable CSRF (Not Recommended)

For development or specific use cases, you can disable CSRF:

```yaml
# fluxbase.yaml
security:
  csrf:
    enabled: false
```

⚠️ **Warning**: Only disable CSRF if:

- You're in development
- Using API key authentication exclusively
- Using a different CSRF protection mechanism
- Building a non-browser client (mobile app, CLI)

---

## Best Practices

### 1. Always Use HTTPS in Production

```yaml
server:
  tls:
    enabled: true
    cert_file: /path/to/cert.pem
    key_file: /path/to/key.pem

security:
  csrf:
    cookie_secure: true # Requires HTTPS
```

### 2. Use Strict SameSite Cookies

```yaml
security:
  csrf:
    cookie_same_site: "Strict" # Most secure
```

### 3. Set Reasonable Expiration

```yaml
security:
  csrf:
    expiration: "24h" # Balance security and UX
```

### 4. Rotate Tokens After Sensitive Actions

```typescript
// After password change, the SDK automatically handles token refresh
await client.auth.changePassword(oldPassword, newPassword);

// The next request will use a fresh CSRF token
```

### 5. Monitor CSRF Failures

```go
// Log CSRF failures for security monitoring
app.Use(func(c *fiber.Ctx) error {
    err := c.Next()
    if err != nil && err.Error() == "CSRF token validation failed" {
        log.Warn().
            Str("ip", c.IP()).
            Str("path", c.Path()).
            Msg("CSRF validation failed")
    }
    return err
})
```

---

## Security Considerations

### CSRF vs XSS

CSRF protection doesn't prevent XSS attacks. Always:

- Implement Content Security Policy
- Sanitize user input
- Use secure templating
- Enable security headers

### CSRF vs client keys

API key authentication bypasses CSRF protection:

- Client keys are not stored in cookies
- Intended for server-to-server communication
- Still need proper authentication and authorization

### Token Storage

Never store CSRF tokens in:

- ❌ LocalStorage (vulnerable to XSS)
- ❌ SessionStorage (vulnerable to XSS)
- ✅ HTTP-only cookies (safe from JavaScript)

---

## Further Reading

- [OWASP CSRF Prevention Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html)
- [Security Overview](./overview.md)
- [Security Headers](./security-headers.md)
- [Best Practices](./best-practices.md)

---

## Summary

Fluxbase provides robust CSRF protection out of the box:

- ✅ **Double-submit cookie pattern**
- ✅ **Automatic token generation**
- ✅ **HTTP-only cookies**
- ✅ **SameSite attribute support**
- ✅ **Configurable expiration**
- ✅ **SDK handles tokens automatically**

Enable CSRF protection in production and follow best practices to protect your users from CSRF attacks.
