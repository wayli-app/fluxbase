---
title: Security Headers
sidebar_position: 3
---

# Security Headers

HTTP security headers are an essential part of web application security. They instruct browsers how to handle your application's content and help protect against common web vulnerabilities. Fluxbase automatically sets secure headers on all responses.

## Configured Security Headers

Fluxbase sets the following security headers by default:

| Header                    | Value                           | Purpose                       |
| ------------------------- | ------------------------------- | ----------------------------- |
| Content-Security-Policy   | Restrictive policy              | Prevents XSS attacks          |
| X-Frame-Options           | DENY                            | Prevents clickjacking         |
| X-Content-Type-Options    | nosniff                         | Prevents MIME sniffing        |
| X-XSS-Protection          | 1; mode=block                   | Legacy XSS protection         |
| Strict-Transport-Security | max-age=31536000                | Forces HTTPS                  |
| Referrer-Policy           | strict-origin-when-cross-origin | Controls referrer information |
| Permissions-Policy        | Restrictive policy              | Controls browser features     |

---

## Content Security Policy (CSP)

CSP is the most powerful security header, preventing XSS attacks by controlling which resources can be loaded.

### Default CSP

```
Content-Security-Policy:
  default-src 'self';
  script-src 'self' 'unsafe-inline' 'unsafe-eval';
  style-src 'self' 'unsafe-inline';
  img-src 'self' data: blob:;
  font-src 'self' data:;
  connect-src 'self' ws: wss:;
  frame-ancestors 'none'
```

### CSP Directives Explained

- **default-src 'self'**: Only load resources from same origin
- **script-src**: JavaScript sources (includes 'unsafe-inline' and 'unsafe-eval' for Admin UI)
- **style-src**: CSS sources (includes 'unsafe-inline' for Admin UI)
- **img-src**: Image sources (includes data: URLs and blob:)
- **font-src**: Font sources
- **connect-src**: AJAX, WebSocket, and EventSource connections (includes ws: and wss: for realtime)
- **frame-ancestors 'none'**: Prevents page from being embedded in frames

### Custom CSP Configuration

**Via `fluxbase.yaml`:**

```yaml
security:
  headers:
    content_security_policy: >
      default-src 'self';
      script-src 'self' https://cdn.example.com;
      style-src 'self' https://fonts.googleapis.com;
      img-src 'self' https: data:;
      font-src 'self' https://fonts.gstatic.com;
      connect-src 'self' wss://realtime.example.com;
      frame-ancestors 'none'
```

**Via Environment Variable:**

```bash
FLUXBASE_SECURITY_HEADERS_CSP="default-src 'self'; script-src 'self'"
```

### CSP for Single-Page Applications

If you're hosting a React/Vue/Angular app, you may need to relax CSP:

```yaml
security:
  headers:
    content_security_policy: >
      default-src 'self';
      script-src 'self' 'unsafe-inline' 'unsafe-eval';
      style-src 'self' 'unsafe-inline';
      img-src 'self' https: data: blob:;
      connect-src 'self' ws: wss: https:;
      frame-ancestors 'none'
```

⚠️ **Warning**: `'unsafe-inline'` and `'unsafe-eval'` reduce security. Use nonces or hashes for production:

```html
<!-- Use nonce for inline scripts -->
<script nonce="random-nonce-here">
  console.log("This script is allowed");
</script>
```

### Testing CSP

Use browser DevTools Console to see CSP violations:

```
Refused to load the script 'https://evil.com/script.js' because it violates
the following Content Security Policy directive: "script-src 'self'"
```

**CSP Report URI** (optional):

```yaml
security:
  headers:
    content_security_policy: >
      default-src 'self';
      report-uri /api/v1/csp-report;
      report-to csp-endpoint
```

---

## X-Frame-Options

Prevents your site from being embedded in an iframe, protecting against clickjacking attacks.

### Default Value

```
X-Frame-Options: DENY
```

### Configuration Options

```yaml
security:
  headers:
    x_frame_options: "DENY"  # Never allow framing
    # OR
    x_frame_options: "SAMEORIGIN"  # Allow framing from same origin
    # OR
    x_frame_options: "ALLOW-FROM https://trusted.com"  # Allow specific origin
```

### Use Cases

- **DENY**: Most secure, use for most applications
- **SAMEORIGIN**: Use if you need to iframe your own content
- **ALLOW-FROM**: Use for specific trusted partners (deprecated, use CSP `frame-ancestors` instead)

### Modern Alternative: CSP frame-ancestors

```yaml
security:
  headers:
    content_security_policy: "frame-ancestors 'none'"  # Equivalent to DENY
    # OR
    content_security_policy: "frame-ancestors 'self'"  # Equivalent to SAMEORIGIN
    # OR
    content_security_policy: "frame-ancestors https://trusted.com"  # Allow specific origin
```

---

## X-Content-Type-Options

Prevents browsers from MIME-sniffing responses, forcing them to respect the `Content-Type` header.

### Default Value

```
X-Content-Type-Options: nosniff
```

### Why It Matters

Without this header, browsers might execute JavaScript disguised as images:

```html
<!-- Attacker uploads "image.jpg" that's actually JavaScript -->
<img src="/uploads/image.jpg" />
<!-- Browser might execute it as JS without nosniff -->
```

With `nosniff`, the browser will only execute files with `Content-Type: application/javascript`.

### Configuration

```yaml
security:
  headers:
    x_content_type_options: "nosniff" # Always use this
```

---

## X-XSS-Protection

Legacy header for older browsers to enable XSS filtering. Modern browsers rely on CSP instead.

### Default Value

```
X-XSS-Protection: 1; mode=block
```

### Configuration Options

```yaml
security:
  headers:
    x_xss_protection: "1; mode=block"  # Enable XSS filter and block
    # OR
    x_xss_protection: "1"  # Enable XSS filter (sanitize)
    # OR
    x_xss_protection: "0"  # Disable XSS filter
```

### Modern Approach

Instead of relying on X-XSS-Protection, use a strong Content Security Policy:

```yaml
security:
  headers:
    content_security_policy: "default-src 'self'; script-src 'self'"
    x_xss_protection: "0" # Disable legacy protection, rely on CSP
```

---

## Strict-Transport-Security (HSTS)

Forces browsers to only connect via HTTPS, preventing protocol downgrade attacks.

### Default Value

```
Strict-Transport-Security: max-age=31536000; includeSubDomains
```

### Configuration

```yaml
security:
  headers:
    strict_transport_security: "max-age=31536000; includeSubDomains; preload"
```

### Parameters

- **max-age**: Duration in seconds (31536000 = 1 year)
- **includeSubDomains**: Apply to all subdomains
- **preload**: Eligible for HSTS preload list

### HSTS Preload List

Submit your domain to the [HSTS Preload List](https://hstspreload.org/) to be hardcoded into browsers:

```yaml
security:
  headers:
    strict_transport_security: "max-age=63072000; includeSubDomains; preload"
```

**Requirements:**

1. Valid TLS certificate
2. Redirect all HTTP to HTTPS
3. Serve HSTS header on base domain
4. Set `max-age` to at least 1 year
5. Include `includeSubDomains`
6. Include `preload` directive

⚠️ **Warning**: Once preloaded, removal takes months. Test thoroughly first!

### HTTPS-Only

HSTS header is only sent on HTTPS connections:

```yaml
server:
  tls:
    enabled: true
    cert_file: /path/to/cert.pem
    key_file: /path/to/key.pem

security:
  headers:
    strict_transport_security: "max-age=31536000; includeSubDomains"
```

---

## Referrer-Policy

Controls how much referrer information is sent with requests.

### Default Value

```
Referrer-Policy: strict-origin-when-cross-origin
```

### Configuration Options

```yaml
security:
  headers:
    referrer_policy: "no-referrer"  # Never send referrer
    # OR
    referrer_policy: "no-referrer-when-downgrade"  # Don't send on HTTPS→HTTP
    # OR
    referrer_policy: "same-origin"  # Only send to same origin
    # OR
    referrer_policy: "origin"  # Only send origin (not full URL)
    # OR
    referrer_policy: "strict-origin"  # Origin only, not on HTTPS→HTTP
    # OR
    referrer_policy: "origin-when-cross-origin"  # Full URL same-origin, origin cross-origin
    # OR
    referrer_policy: "strict-origin-when-cross-origin"  # Balanced approach (default)
    # OR
    referrer_policy: "unsafe-url"  # Always send full URL (not recommended)
```

### Policy Comparison

| Policy                          | Same-Origin    | Cross-Origin HTTPS | Cross-Origin HTTP |
| ------------------------------- | -------------- | ------------------ | ----------------- |
| no-referrer                     | ❌             | ❌                 | ❌                |
| same-origin                     | ✅ Full URL    | ❌                 | ❌                |
| origin                          | ✅ Origin only | ✅ Origin only     | ✅ Origin only    |
| strict-origin                   | ✅ Origin only | ✅ Origin only     | ❌                |
| strict-origin-when-cross-origin | ✅ Full URL    | ✅ Origin only     | ❌                |

### Use Cases

**Maximum Privacy:**

```yaml
referrer_policy: "no-referrer"
```

**Analytics-Friendly:**

```yaml
referrer_policy: "strict-origin-when-cross-origin" # Default
```

**Internal Links Only:**

```yaml
referrer_policy: "same-origin"
```

---

## Permissions-Policy

Controls which browser features and APIs can be used (formerly Feature-Policy).

### Default Value

```
Permissions-Policy: geolocation=(), microphone=(), camera=()
```

### Configuration

```yaml
security:
  headers:
    permissions_policy: >
      geolocation=(),
      microphone=(),
      camera=(),
      payment=(),
      usb=(),
      magnetometer=(),
      gyroscope=(),
      accelerometer=()
```

### Available Features

Common features you can control:

- **geolocation**: GPS location
- **camera**: Camera access
- **microphone**: Microphone access
- **payment**: Payment Request API
- **usb**: WebUSB API
- **bluetooth**: Web Bluetooth API
- **midi**: Web MIDI API
- **fullscreen**: Fullscreen API
- **picture-in-picture**: Picture-in-Picture API
- **display-capture**: Screen capture
- **autoplay**: Media autoplay

### Policy Syntax

```yaml
# Deny all origins (most secure)
permissions_policy: "geolocation=()"

# Allow same origin only
permissions_policy: "geolocation=(self)"

# Allow specific origins
permissions_policy: "geolocation=(self 'https://trusted.com')"

# Allow all origins (not recommended)
permissions_policy: "geolocation=*"
```

### Example: Allow Specific Features

```yaml
security:
  headers:
    permissions_policy: >
      geolocation=(self),
      camera=(self),
      microphone=(self),
      payment=(self 'https://payment-provider.com'),
      usb=(),
      bluetooth=()
```

---

## Complete Configuration Example

### Production Configuration

```yaml
# fluxbase.yaml
server:
  port: 443
  tls:
    enabled: true
    cert_file: /etc/letsencrypt/live/example.com/fullchain.pem
    key_file: /etc/letsencrypt/live/example.com/privkey.pem

security:
  headers:
    # Content Security Policy
    content_security_policy: >
      default-src 'self';
      script-src 'self';
      style-src 'self' https://fonts.googleapis.com;
      img-src 'self' https: data: blob:;
      font-src 'self' data: https://fonts.gstatic.com;
      connect-src 'self' wss://example.com;
      frame-ancestors 'none';
      base-uri 'self';
      form-action 'self'

    # Clickjacking protection
    x_frame_options: "DENY"

    # MIME sniffing protection
    x_content_type_options: "nosniff"

    # XSS protection (legacy)
    x_xss_protection: "1; mode=block"

    # Force HTTPS (1 year, include subdomains, preload)
    strict_transport_security: "max-age=31536000; includeSubDomains; preload"

    # Referrer policy (balanced)
    referrer_policy: "strict-origin-when-cross-origin"

    # Permissions policy (restrict sensitive features)
    permissions_policy: >
      geolocation=(),
      microphone=(),
      camera=(),
      payment=(),
      usb=(),
      bluetooth=()
```

### Development Configuration

```yaml
# fluxbase.yaml
security:
  headers:
    # Relaxed CSP for development
    content_security_policy: >
      default-src 'self';
      script-src 'self' 'unsafe-inline' 'unsafe-eval';
      style-src 'self' 'unsafe-inline';
      img-src 'self' https: data: blob:;
      connect-src 'self' ws: wss: http: https:;
      frame-ancestors 'self'

    x_frame_options: "SAMEORIGIN"
    x_content_type_options: "nosniff"
    x_xss_protection: "1; mode=block"

    # No HSTS in development (HTTP allowed)
    strict_transport_security: ""

    referrer_policy: "no-referrer-when-downgrade"
    permissions_policy: "" # Allow all in development
```

---

## Testing Security Headers

### Manual Testing with cURL

```bash
# Check all headers
curl -I https://yourapp.com/

# Check specific header
curl -I https://yourapp.com/ | grep -i "content-security-policy"
```

### Online Testing Tools

1. **Security Headers** (https://securityheaders.com/)

   - Comprehensive security header analysis
   - Letter grade rating
   - Recommendations for improvement

2. **Mozilla Observatory** (https://observatory.mozilla.org/)

   - Security and privacy analysis
   - Detailed scoring
   - Specific recommendations

3. **SSL Labs** (https://www.ssllabs.com/ssltest/)
   - TLS/SSL configuration testing
   - HSTS validation
   - Certificate chain analysis

### Automated Testing

```typescript
import { describe, it, expect } from "vitest";

describe("Security Headers", () => {
  it("should set Content-Security-Policy", async () => {
    const response = await fetch("https://yourapp.com/");
    expect(response.headers.get("content-security-policy")).toContain(
      "default-src 'self'"
    );
  });

  it("should set X-Frame-Options", async () => {
    const response = await fetch("https://yourapp.com/");
    expect(response.headers.get("x-frame-options")).toBe("DENY");
  });

  it("should set HSTS on HTTPS", async () => {
    const response = await fetch("https://yourapp.com/");
    expect(response.headers.get("strict-transport-security")).toContain(
      "max-age="
    );
  });
});
```

---

## Troubleshooting

### Issue: CSP Blocks Legitimate Resources

**Symptom**: Resources failing to load, console errors

**Solution**: Add specific origins to CSP:

```yaml
security:
  headers:
    content_security_policy: >
      default-src 'self';
      script-src 'self' https://cdn.example.com;
      style-src 'self' https://fonts.googleapis.com
```

### Issue: Admin UI Not Working

**Symptom**: React/Vue app broken, CSP violations

**Solution**: Use relaxed CSP for Admin UI:

```yaml
security:
  headers:
    # Admin UI needs 'unsafe-inline' and 'unsafe-eval'
    content_security_policy: >
      default-src 'self';
      script-src 'self' 'unsafe-inline' 'unsafe-eval';
      style-src 'self' 'unsafe-inline'
```

Or use route-specific headers:

```go
// Apply relaxed headers only to Admin UI routes
app.Use("/admin", AdminUISecurityHeaders())
```

### Issue: Embedded Content Not Loading

**Symptom**: iframes, embedded videos failing

**Solution**: Update CSP frame-src:

```yaml
security:
  headers:
    content_security_policy: >
      default-src 'self';
      frame-src https://www.youtube.com https://player.vimeo.com
```

### Issue: WebSocket Connections Failing

**Symptom**: Realtime features not working

**Solution**: Add ws: and wss: to connect-src:

```yaml
security:
  headers:
    content_security_policy: >
      default-src 'self';
      connect-src 'self' ws: wss:
```

---

## Best Practices

### 1. Start Strict, Relax as Needed

```yaml
# Start with most restrictive policy
content_security_policy: "default-src 'self'"

# Add specific exceptions as needed
content_security_policy: >
  default-src 'self';
  img-src 'self' https://cdn.example.com
```

### 2. Use CSP Report-Only Mode for Testing

```yaml
# Test CSP without breaking functionality
Content-Security-Policy-Report-Only: default-src 'self'
```

### 3. Avoid 'unsafe-inline' and 'unsafe-eval'

Use nonces or hashes instead:

```html
<!-- Generate random nonce per request -->
<script nonce="2726c7f26c">
  // Inline script allowed
</script>
```

```yaml
content_security_policy: "script-src 'nonce-2726c7f26c'"
```

### 4. Monitor CSP Violations

Set up reporting:

```yaml
content_security_policy: >
  default-src 'self';
  report-uri /api/v1/csp-report
```

```go
// Log CSP violations
app.Post("/api/v1/csp-report", func(c *fiber.Ctx) error {
    var report map[string]interface{}
    c.BodyParser(&report)
    log.Warn().Interface("csp_violation", report).Msg("CSP violation reported")
    return c.SendStatus(204)
})
```

### 5. Test on All Browsers

Different browsers have different CSP support:

- Test on Chrome, Firefox, Safari, Edge
- Check mobile browsers (iOS Safari, Chrome Mobile)
- Verify old browser fallbacks

### 6. Document Custom Headers

```yaml
# Document why each exception is needed
content_security_policy: >
  default-src 'self';
  script-src 'self' https://cdn.example.com;  # Third-party analytics
  style-src 'self' 'unsafe-inline';  # Required for Admin UI
  img-src 'self' https:;  # User-uploaded images from CDN
```

---

## Security Headers Checklist

- [ ] Content-Security-Policy configured
- [ ] X-Frame-Options set to DENY or SAMEORIGIN
- [ ] X-Content-Type-Options set to nosniff
- [ ] HSTS enabled with appropriate max-age
- [ ] Referrer-Policy configured
- [ ] Permissions-Policy restricts unnecessary features
- [ ] Tested on securityheaders.com (A+ rating)
- [ ] Tested on Mozilla Observatory (A+ rating)
- [ ] CSP violations monitored
- [ ] Headers documented and reviewed

---

## Further Reading

- [Security Overview](./overview.md)
- [CSRF Protection](./csrf-protection.md)
- [Best Practices](./best-practices.md)
- [OWASP Secure Headers Project](https://owasp.org/www-project-secure-headers/)
- [MDN: CSP](https://developer.mozilla.org/en-US/docs/Web/HTTP/CSP)
- [Content Security Policy Reference](https://content-security-policy.com/)

---

## Summary

Security headers are a critical defense layer:

- ✅ **Content Security Policy** - Prevents XSS attacks
- ✅ **X-Frame-Options** - Prevents clickjacking
- ✅ **X-Content-Type-Options** - Prevents MIME sniffing
- ✅ **HSTS** - Forces HTTPS
- ✅ **Referrer-Policy** - Controls referrer information
- ✅ **Permissions-Policy** - Restricts browser features

Fluxbase sets secure defaults, but customize them for your specific needs. Test thoroughly and monitor for violations.
