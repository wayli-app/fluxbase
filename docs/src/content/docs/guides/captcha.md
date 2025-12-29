---
title: "CAPTCHA Protection"
---

Fluxbase supports CAPTCHA verification to protect authentication endpoints from bots and automated abuse. Multiple providers are supported including hCaptcha, reCAPTCHA v3, Cloudflare Turnstile, and the self-hosted Cap provider.

## Overview

CAPTCHA protection can be enabled on specific authentication endpoints:

- **signup** - New user registration
- **login** - User authentication
- **password_reset** - Password reset requests
- **magic_link** - Magic link authentication

When enabled, clients must include a valid CAPTCHA token with their authentication requests.

## Supported Providers

| Provider | Type | Self-Hosted | Best For |
|----------|------|-------------|----------|
| [hCaptcha](https://www.hcaptcha.com/) | Visual challenge | No | Privacy-focused applications |
| [reCAPTCHA v3](https://www.google.com/recaptcha/) | Invisible (score-based) | No | Seamless user experience |
| [Cloudflare Turnstile](https://www.cloudflare.com/products/turnstile/) | Invisible | No | Cloudflare users, free tier |
| [Cap](https://capjs.js.org/) | Proof-of-work | Yes | Self-hosted, privacy-first |

## Configuration

### YAML Configuration

```yaml
security:
  captcha:
    enabled: true
    provider: hcaptcha  # hcaptcha, recaptcha_v3, turnstile, cap
    site_key: "your-site-key"
    secret_key: "your-secret-key"
    score_threshold: 0.5  # reCAPTCHA v3 only (0.0-1.0)
    endpoints:
      - signup
      - login
      - password_reset
      - magic_link
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `FLUXBASE_SECURITY_CAPTCHA_ENABLED` | Enable CAPTCHA verification (`true`/`false`) |
| `FLUXBASE_SECURITY_CAPTCHA_PROVIDER` | Provider name |
| `FLUXBASE_SECURITY_CAPTCHA_SITE_KEY` | Public site key |
| `FLUXBASE_SECURITY_CAPTCHA_SECRET_KEY` | Secret key for verification |
| `FLUXBASE_SECURITY_CAPTCHA_SCORE_THRESHOLD` | Score threshold (reCAPTCHA v3 only) |
| `FLUXBASE_SECURITY_CAPTCHA_ENDPOINTS` | Comma-separated list of endpoints |

### Cap Provider Configuration

For the self-hosted Cap provider, use different configuration options:

```yaml
security:
  captcha:
    enabled: true
    provider: cap
    cap_server_url: "http://localhost:3000"  # Your Cap server URL
    cap_api_key: "your-api-key"
    endpoints:
      - signup
      - login
```

| Variable | Description |
|----------|-------------|
| `FLUXBASE_SECURITY_CAPTCHA_CAP_SERVER_URL` | URL of your Cap server |
| `FLUXBASE_SECURITY_CAPTCHA_CAP_API_KEY` | Cap API key |

## Provider Setup

### hCaptcha

1. Sign up at [hcaptcha.com](https://www.hcaptcha.com/)
2. Add your domain to get your site key and secret key
3. Configure Fluxbase:

```yaml
security:
  captcha:
    enabled: true
    provider: hcaptcha
    site_key: "10000000-ffff-ffff-ffff-000000000001"  # Test key
    secret_key: "0x0000000000000000000000000000000000000000"  # Test key
```

### reCAPTCHA v3

1. Register at [Google reCAPTCHA](https://www.google.com/recaptcha/admin)
2. Select reCAPTCHA v3 and add your domains
3. Configure with your keys:

```yaml
security:
  captcha:
    enabled: true
    provider: recaptcha_v3
    site_key: "your-recaptcha-site-key"
    secret_key: "your-recaptcha-secret-key"
    score_threshold: 0.5  # Reject scores below this (0.0 = bot, 1.0 = human)
```

### Cloudflare Turnstile

1. Access [Cloudflare Turnstile](https://dash.cloudflare.com/?to=/:account/turnstile) dashboard
2. Create a widget for your domain
3. Configure Fluxbase:

```yaml
security:
  captcha:
    enabled: true
    provider: turnstile
    site_key: "your-turnstile-site-key"
    secret_key: "your-turnstile-secret-key"
```

### Cap (Self-Hosted)

Cap is a proof-of-work CAPTCHA that runs entirely on your infrastructure.

1. Run the Cap server:

```bash
docker run -p 3000:3000 ghcr.io/tiagozip/cap:latest
```

2. Configure Fluxbase:

```yaml
security:
  captcha:
    enabled: true
    provider: cap
    cap_server_url: "http://localhost:3000"
    cap_api_key: "your-api-key"
```

## Frontend Integration

### Getting CAPTCHA Configuration

First, fetch the CAPTCHA configuration from your Fluxbase server:

```typescript
const response = await fetch('http://localhost:8080/api/v1/auth/captcha/config')
const config = await response.json()

// {
//   "enabled": true,
//   "provider": "hcaptcha",
//   "site_key": "your-site-key",
//   "endpoints": ["signup", "login"]
// }
```

### hCaptcha Widget

```html
<script src="https://js.hcaptcha.com/1/api.js" async defer></script>

<form id="signup-form">
  <input type="email" name="email" required />
  <input type="password" name="password" required />
  <div class="h-captcha" data-sitekey="YOUR_SITE_KEY"></div>
  <button type="submit">Sign Up</button>
</form>

<script>
document.getElementById('signup-form').onsubmit = async (e) => {
  e.preventDefault();
  const token = hcaptcha.getResponse();

  await fetch('/api/v1/auth/signup', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      email: e.target.email.value,
      password: e.target.password.value,
      captcha_token: token
    })
  });
};
</script>
```

### reCAPTCHA v3 Widget

```html
<script src="https://www.google.com/recaptcha/api.js?render=YOUR_SITE_KEY"></script>

<script>
async function signUp(email, password) {
  const token = await grecaptcha.execute('YOUR_SITE_KEY', { action: 'signup' });

  await fetch('/api/v1/auth/signup', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      email,
      password,
      captcha_token: token
    })
  });
}
</script>
```

### Cloudflare Turnstile Widget

```html
<script src="https://challenges.cloudflare.com/turnstile/v0/api.js" async defer></script>

<form id="signup-form">
  <input type="email" name="email" required />
  <input type="password" name="password" required />
  <div class="cf-turnstile" data-sitekey="YOUR_SITE_KEY"></div>
  <button type="submit">Sign Up</button>
</form>

<script>
document.getElementById('signup-form').onsubmit = async (e) => {
  e.preventDefault();
  const token = document.querySelector('[name="cf-turnstile-response"]').value;

  await fetch('/api/v1/auth/signup', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      email: e.target.email.value,
      password: e.target.password.value,
      captcha_token: token
    })
  });
};
</script>
```

### Cap Widget

```html
<!-- Load Cap widget from your self-hosted server -->
<script src="http://localhost:3000/widget.js"></script>

<form id="signup-form">
  <input type="email" name="email" required />
  <input type="password" name="password" required />
  <cap-widget data-cap-url="http://localhost:3000"></cap-widget>
  <input type="hidden" name="captcha_token" id="captcha_token" />
  <button type="submit">Sign Up</button>
</form>

<script>
// Cap widget will populate the token when solved
window.onCapComplete = (token) => {
  document.getElementById('captcha_token').value = token;
};
</script>
```

## SDK Usage

### TypeScript SDK

```typescript
import { FluxbaseClient } from '@fluxbase/sdk'

const client = new FluxbaseClient({ url: 'http://localhost:8080' })

// Get CAPTCHA configuration
const { data: config } = await client.auth.getCaptchaConfig()

if (config?.enabled) {
  console.log('CAPTCHA provider:', config.provider)
  console.log('Site key:', config.site_key)
  console.log('Protected endpoints:', config.endpoints)
}

// Sign up with CAPTCHA token
const { data, error } = await client.auth.signUp({
  email: 'user@example.com',
  password: 'SecurePassword123',
  captchaToken: 'token-from-widget'
})

// Sign in with CAPTCHA token
const { data: session, error } = await client.auth.signIn({
  email: 'user@example.com',
  password: 'SecurePassword123',
  captchaToken: 'token-from-widget'
})

// Request password reset with CAPTCHA
await client.auth.resetPassword({
  email: 'user@example.com',
  captchaToken: 'token-from-widget'
})
```

### React SDK

```tsx
import {
  useCaptchaConfig,
  useCaptcha,
  useSignUp,
  isCaptchaRequiredForEndpoint
} from '@fluxbase/sdk-react'

function SignUpForm() {
  const { data: config } = useCaptchaConfig()
  const captcha = useCaptcha(config?.provider)
  const signUp = useSignUp()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    let captchaToken: string | undefined

    // Check if CAPTCHA is required for signup
    if (isCaptchaRequiredForEndpoint(config, 'signup')) {
      captchaToken = await captcha.execute()
    }

    await signUp.mutateAsync({
      email,
      password,
      captchaToken
    })
  }

  return (
    <form onSubmit={handleSubmit}>
      <input type="email" name="email" required />
      <input type="password" name="password" required />

      {config?.enabled && config.provider && (
        <CaptchaWidget
          provider={config.provider}
          siteKey={config.site_key}
          onVerify={captcha.setToken}
        />
      )}

      <button type="submit" disabled={signUp.isPending}>
        Sign Up
      </button>
    </form>
  )
}
```

### useCaptcha Hook

The `useCaptcha` hook provides a unified interface for all CAPTCHA providers:

```tsx
const captcha = useCaptcha(provider)

// Properties
captcha.token      // Current token (string | null)
captcha.isReady    // Widget loaded and ready (boolean)
captcha.isLoading  // Widget is loading (boolean)
captcha.error      // Any error during loading (Error | null)

// Methods
captcha.execute()  // Execute CAPTCHA and get token (Promise<string>)
captcha.reset()    // Reset the widget
captcha.setToken() // Manually set token (for widget callbacks)
```

## REST API

### Get CAPTCHA Configuration

```bash
GET /api/v1/auth/captcha/config
```

Returns the public CAPTCHA configuration:

```json
{
  "enabled": true,
  "provider": "hcaptcha",
  "site_key": "10000000-ffff-ffff-ffff-000000000001",
  "endpoints": ["signup", "login", "password_reset", "magic_link"]
}
```

### Protected Endpoints

When CAPTCHA is enabled for an endpoint, include the token in your request:

```bash
# Sign up with CAPTCHA
curl -X POST http://localhost:8080/api/v1/auth/signup \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePassword123",
    "captcha_token": "10000000-aaaa-bbbb-cccc-000000000001"
  }'

# Sign in with CAPTCHA
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePassword123",
    "captcha_token": "10000000-aaaa-bbbb-cccc-000000000001"
  }'
```

### Error Responses

| Status | Error | Description |
|--------|-------|-------------|
| 400 | `captcha_required` | CAPTCHA token is required but missing |
| 400 | `captcha_invalid` | CAPTCHA verification failed |
| 400 | `captcha_expired` | CAPTCHA token has expired |
| 400 | `captcha_score_too_low` | reCAPTCHA v3 score below threshold |

## Troubleshooting

### CAPTCHA verification always fails

1. **Check your keys** - Ensure site key and secret key match and are for the correct environment (test vs production)
2. **Domain mismatch** - Verify your domain is registered with the CAPTCHA provider
3. **Clock skew** - Ensure server time is synchronized (tokens expire)

### reCAPTCHA v3 scores are too low

- Adjust `score_threshold` lower (e.g., 0.3 instead of 0.5)
- reCAPTCHA v3 learns over time; scores improve with traffic
- Consider using action names that match your endpoint (`signup`, `login`)

### Cap widget not loading

- Verify Cap server is running and accessible
- Check browser console for CORS errors
- Ensure `cap_server_url` matches your Cap server exactly

### CAPTCHA not showing on frontend

```typescript
// Always check if CAPTCHA is required before rendering
const { data: config } = await client.auth.getCaptchaConfig()

if (config?.enabled && config.endpoints?.includes('signup')) {
  // Render CAPTCHA widget
}
```

## Security Best Practices

1. **Never expose secret keys** - Secret keys should only be on the server
2. **Use HTTPS** - CAPTCHA tokens should be transmitted over HTTPS
3. **Combine with rate limiting** - CAPTCHA doesn't replace rate limiting
4. **Monitor verification failures** - High failure rates may indicate attacks
5. **Token single-use** - Each token should only be used once

## Next Steps

- [Authentication](/docs/guides/authentication) - Authentication overview
- [Rate Limiting](/docs/guides/rate-limiting) - Rate limiting configuration
- [Row-Level Security](/docs/guides/row-level-security) - Data access control
