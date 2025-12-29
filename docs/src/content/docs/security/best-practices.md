---
title: Security Best Practices
---

This guide provides comprehensive security best practices for deploying and maintaining a secure Fluxbase instance.

## Table of Contents

- [Authentication & Authorization](#authentication--authorization)
- [SAML SSO Security](#saml-sso-security)
- [GraphQL API Security](#graphql-api-security)
- [Database Security](#database-security)
- [Network Security](#network-security)
- [Secrets Management](#secrets-management)
- [Input Validation](#input-validation)
- [Error Handling](#error-handling)
- [Logging & Monitoring](#logging--monitoring)
- [Deployment Security](#deployment-security)
- [Incident Response](#incident-response)

---

## Authentication & Authorization

### 1. Enforce Strong Password Policies

```yaml
# fluxbase.yaml
auth:
  password_min_length: 12
  password_require_uppercase: true
  password_require_lowercase: true
  password_require_number: true
  password_require_special: true
  password_max_age_days: 90 # Force password rotation
```

**Client-side validation:**

```typescript
function validatePassword(password: string): string[] {
  const errors: string[] = [];

  if (password.length < 12) {
    errors.push("Password must be at least 12 characters");
  }
  if (!/[A-Z]/.test(password)) {
    errors.push("Password must contain an uppercase letter");
  }
  if (!/[a-z]/.test(password)) {
    errors.push("Password must contain a lowercase letter");
  }
  if (!/[0-9]/.test(password)) {
    errors.push("Password must contain a number");
  }
  if (!/[!@#$%^&*]/.test(password)) {
    errors.push("Password must contain a special character");
  }

  return errors;
}
```

### 2. Implement Multi-Factor Authentication

```typescript
// Enable 2FA for user
const { qr_code, secret } = await client.auth.setup2FA();

// Display QR code to user
showQRCode(qr_code);

// Verify and enable
await client.auth.enable2FA({ code: userEnteredCode });
```

**Enforce 2FA for sensitive operations:**

```yaml
auth:
  require_2fa_for_admins: true
  require_2fa_for_sensitive_ops: true
```

### 3. Use Short-Lived Tokens

```yaml
auth:
  access_token_expiry: "15m" # Short-lived access tokens
  refresh_token_expiry: "7d" # Longer refresh tokens
  refresh_token_rotation: true # Rotate on each use
```

### 4. Implement Token Blacklisting

```typescript
// Logout revokes tokens
await client.auth.signOut(); // Token added to blacklist

// Verify token isn't blacklisted in middleware
const isBlacklisted = await checkTokenBlacklist(token);
if (isBlacklisted) {
  throw new Error("Token has been revoked");
}
```

### 5. Limit Failed Login Attempts

```yaml
auth:
  max_login_attempts: 5
  lockout_duration: "15m"
  lockout_type: "ip_and_email" # Lock both IP and email
```

### 6. Implement Row Level Security

```sql
-- Enable RLS on ALL user tables
ALTER TABLE public.user_data ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.user_data FORCE ROW LEVEL SECURITY;

-- Create isolation policy
CREATE POLICY user_isolation ON public.user_data
  FOR ALL
  USING (user_id = auth.uid())
  WITH CHECK (user_id = auth.uid());
```

[Learn more about RLS →](../guides/row-level-security.md)

---

## SAML SSO Security

When configuring SAML SSO providers, follow these security best practices to prevent common attacks.

### 1. Disable IdP-Initiated SSO

IdP-initiated SSO is vulnerable to assertion replay attacks. Only enable it if your IdP requires it.

```yaml
# fluxbase.yaml
auth:
  saml_providers:
    - name: okta
      enabled: true
      allow_idp_initiated: false  # Recommended - prevents replay attacks
```

### 2. Validate Audience Restrictions

Fluxbase automatically validates that SAML assertions are intended for your service provider by checking the Audience element. This prevents attackers from replaying assertions meant for other applications.

### 3. Configure RelayState Redirect Whitelist

Prevent open redirect attacks by configuring allowed redirect hosts for SAML authentication.

```yaml
auth:
  saml_providers:
    - name: okta
      enabled: true
      # Only allow redirects to your application domains
      allowed_redirect_hosts:
        - "app.example.com"
        - "dashboard.example.com"
```

Without this configuration, only relative URLs (same-origin) are allowed for redirects.

### 4. Require HTTPS for Metadata URLs

Always use HTTPS for IdP metadata URLs to prevent man-in-the-middle attacks.

```yaml
auth:
  saml_providers:
    - name: okta
      enabled: true
      idp_metadata_url: "https://company.okta.com/app/xxx/sso/saml/metadata"
      allow_insecure_metadata_url: false  # Default - requires HTTPS
```

### 5. User Attribute Sanitization

Fluxbase automatically sanitizes user attributes (like display names) from SAML assertions to prevent XSS attacks from malicious IdP responses.

---

## GraphQL API Security

Fluxbase implements several security measures for the GraphQL API to prevent abuse and resource exhaustion.

### 1. Query Depth Limiting

Deeply nested queries can cause exponential resource consumption. Configure maximum query depth.

```yaml
# fluxbase.yaml
graphql:
  enabled: true
  max_depth: 10  # Maximum nesting depth
```

**Example rejected query (depth > 5):**

```graphql
{
  users {           # depth 1
    posts {         # depth 2
      comments {    # depth 3
        author {    # depth 4
          posts {   # depth 5
            title   # depth 6 - REJECTED
          }
        }
      }
    }
  }
}
```

### 2. Query Complexity Analysis

Fluxbase calculates a complexity score for each query based on fields and list traversals.

```yaml
graphql:
  max_complexity: 1000  # Maximum complexity score
```

The complexity score accounts for:
- Base cost per field (1 point)
- Higher cost for list fields (10 points)
- Multiplied cost for nested lists

### 3. Disable Introspection in Production

GraphQL introspection exposes your entire schema. Disable it in production.

```yaml
graphql:
  introspection: false  # Disable schema introspection
```

### 4. Row Level Security (RLS) Enforcement

The GraphQL endpoint enforces PostgreSQL Row Level Security for all operations:

**How it works:**
- GraphQL resolvers execute queries with `SET LOCAL ROLE` set to the appropriate database role
- JWT claims are passed via `request.jwt.claims` for use in RLS policies
- All queries, mutations, and foreign key traversals respect RLS policies
- Anonymous requests use the `anon` role
- Authenticated users use the `authenticated` role
- Service keys use the `service_role` (bypasses RLS)

**Role Mapping:**
| Application Role | Database Role |
|-----------------|---------------|
| `service_role`, `dashboard_admin` | `service_role` (bypasses RLS) |
| `anon`, empty | `anon` |
| All others (`user`, `admin`, etc.) | `authenticated` |

**Configure RLS on your tables:**

```sql
-- Enable RLS on tables accessed via GraphQL
ALTER TABLE public.posts ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.posts FORCE ROW LEVEL SECURITY;

-- Create appropriate policies
CREATE POLICY posts_select ON public.posts
  FOR SELECT
  USING (user_id = auth.uid() OR is_public = true);

-- Use JWT claims for fine-grained access
CREATE POLICY admin_full_access ON public.posts
  FOR ALL
  USING (
    current_setting('request.jwt.claims', true)::jsonb->>'role' = 'admin'
  );
```

**Important:** Foreign key traversals also respect RLS. A user cannot access related records through GraphQL joins if they don't have permission to view those records directly.

---

## Database Security

### 1. Use Principle of Least Privilege

```sql
-- Create application user with minimal permissions
CREATE USER fluxbase_app WITH PASSWORD 'secure_password';

-- Grant only necessary permissions
GRANT CONNECT ON DATABASE fluxbase TO fluxbase_app;
GRANT USAGE ON SCHEMA public TO fluxbase_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO fluxbase_app;

-- Revoke dangerous permissions
REVOKE CREATE ON SCHEMA public FROM fluxbase_app;
REVOKE ALL ON SCHEMA pg_catalog FROM fluxbase_app;

-- For read-only operations
CREATE USER readonly_user WITH PASSWORD 'secure_password';
GRANT CONNECT ON DATABASE fluxbase TO readonly_user;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO readonly_user;
```

### 2. Enable Encrypted Connections

```yaml
# fluxbase.yaml
database:
  url: "postgres://user:pass@host:5432/db?sslmode=require"
  max_connections: 50
  ssl_mode: "require" # or "verify-full" for production
```

**PostgreSQL SSL Configuration:**

```ini
# postgresql.conf
ssl = on
ssl_cert_file = '/path/to/server.crt'
ssl_key_file = '/path/to/server.key'
ssl_ca_file = '/path/to/root.crt'
```

### 3. Regular Backups with Encryption

```bash
#!/bin/bash
# backup.sh

# Backup with encryption
pg_dump -U postgres fluxbase | \
  gpg --encrypt --recipient admin@example.com > \
  /backups/fluxbase-$(date +%Y%m%d).sql.gpg

# Rotate old backups
find /backups -name "fluxbase-*.sql.gpg" -mtime +30 -delete
```

**Automated backups:**

```bash
# crontab
0 2 * * * /usr/local/bin/backup.sh
```

### 4. Audit Database Access

```sql
-- Enable audit logging
CREATE EXTENSION IF NOT EXISTS pgaudit;

-- Configure audit settings
ALTER SYSTEM SET pgaudit.log = 'write, ddl';
ALTER SYSTEM SET pgaudit.log_catalog = off;
ALTER SYSTEM SET pgaudit.log_parameter = on;

-- Reload configuration
SELECT pg_reload_conf();
```

### 5. Protect Against SQL Injection

```typescript
// ✅ GOOD: Parameterized queries
const { data } = await client
  .from("users")
  .select("*")
  .eq("email", userEmail) // Safely parameterized
  .execute();

// ❌ BAD: String concatenation
const query = `SELECT * FROM users WHERE email = '${userEmail}'`;
// NEVER DO THIS!
```

---

## Network Security

### 1. Always Use HTTPS in Production

```yaml
# fluxbase.yaml
server:
  port: 443
  tls:
    enabled: true
    cert_file: /etc/letsencrypt/live/example.com/fullchain.pem
    key_file: /etc/letsencrypt/live/example.com/privkey.pem
    min_version: "1.2" # TLS 1.2 minimum
```

**Automatic certificate renewal with Let's Encrypt:**

```bash
# Install certbot
apt-get install certbot

# Get certificate
certbot certonly --standalone -d example.com

# Auto-renewal (crontab)
0 0 1 * * certbot renew --quiet && systemctl reload fluxbase
```

### 2. Configure Firewall Rules

```bash
# UFW (Ubuntu)
ufw default deny incoming
ufw default allow outgoing
ufw allow 22/tcp    # SSH
ufw allow 443/tcp   # HTTPS
ufw allow 80/tcp    # HTTP (for Let's Encrypt)
ufw enable

# Restrict PostgreSQL access
ufw allow from 10.0.0.0/8 to any port 5432
```

**iptables alternative:**

```bash
# Block all incoming except SSH and HTTPS
iptables -P INPUT DROP
iptables -A INPUT -p tcp --dport 22 -j ACCEPT
iptables -A INPUT -p tcp --dport 443 -j ACCEPT
iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT
```

### 3. Enable Rate Limiting

```yaml
# fluxbase.yaml
rate_limiting:
  enabled: true
  per_minute: 60
  per_hour: 1000

  # Stricter limits for sensitive endpoints
  endpoints:
    - path: "/api/v1/auth/login"
      per_minute: 5
      per_hour: 20

    - path: "/api/v1/auth/signup"
      per_minute: 3
      per_hour: 10

    - path: "/api/v1/auth/password/reset"
      per_minute: 2
      per_hour: 5
```

[Learn more about Rate Limiting →](../guides/rate-limiting.md)

### 4. Configure CORS Properly

```yaml
# fluxbase.yaml
server:
  cors:
    # ✅ GOOD: Specific origins
    allowed_origins:
      - "https://yourdomain.com"
      - "https://www.yourdomain.com"

    # ❌ BAD: Wildcard allows any origin
    # allowed_origins: ["*"]

    allowed_methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
    allowed_headers: ["Content-Type", "Authorization", "X-CSRF-Token"]
    allow_credentials: true # Required for cookies
    max_age: 3600
```

### 5. Implement Security Headers

```yaml
security:
  headers:
    content_security_policy: "default-src 'self'"
    x_frame_options: "DENY"
    strict_transport_security: "max-age=31536000; includeSubDomains"
```

[Learn more about Security Headers →](./security-headers.md)

---

## Secrets Management

### 1. Never Commit Secrets to Git

```text
# .gitignore
.env
.env.local
.env.*.local
fluxbase.yaml
*.pem
*.key
secrets/
```

**Check for committed secrets:**

```bash
# Use git-secrets
git secrets --scan

# Use truffleHog
trufflehog filesystem .
```

### 2. Use Environment Variables

```yaml
# fluxbase.yaml - reference environment variables
database:
  url: ${DATABASE_URL}

auth:
  jwt_secret: ${JWT_SECRET}

email:
  smtp_password: ${SMTP_PASSWORD}
```

```bash
# .env (never commit!)
DATABASE_URL=postgres://user:pass@host/db
JWT_SECRET=your-super-secret-key-min-32-chars
SMTP_PASSWORD=smtp-password-here
```

### 3. Use Secrets Management Services

**Docker Secrets:**

```bash
# Create secret
echo "my-jwt-secret" | docker secret create jwt_secret -

# Use in docker-compose.yml
services:
  fluxbase:
    secrets:
      - jwt_secret
    environment:
      JWT_SECRET_FILE: /run/secrets/jwt_secret
```

**Kubernetes Secrets:**

```bash
# Create secret
kubectl create secret generic fluxbase-secrets \
  --from-literal=jwt-secret=my-jwt-secret \
  --from-literal=database-url=postgres://...

# Use in deployment
env:
  - name: JWT_SECRET
    valueFrom:
      secretKeyRef:
        name: fluxbase-secrets
        key: jwt-secret
```

**AWS Secrets Manager:**

```typescript
import { SecretsManager } from "@aws-sdk/client-secrets-manager";

const client = new SecretsManager({ region: "us-east-1" });

async function getSecret(secretName: string): Promise<string> {
  const response = await client.getSecretValue({ SecretId: secretName });
  return response.SecretString || "";
}

// Use in application
const jwtSecret = await getSecret("fluxbase/jwt-secret");
```

### 4. Rotate Secrets Regularly

```bash
#!/bin/bash
# rotate-secrets.sh

# Generate new JWT secret
NEW_SECRET=$(openssl rand -base64 32)

# Update in secrets manager
kubectl patch secret fluxbase-secrets \
  -p="{\"data\":{\"jwt-secret\":\"$(echo -n $NEW_SECRET | base64)\"}}"

# Rolling restart
kubectl rollout restart deployment/fluxbase
```

### 5. Limit Secret Access

```bash
# Kubernetes RBAC
apiVersion: v1
kind: ServiceAccount
metadata:
  name: fluxbase
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: secret-reader
rules:
- apiGroups: [""]
  resources: ["secrets"]
  resourceNames: ["fluxbase-secrets"]
  verbs: ["get"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: read-secrets
subjects:
- kind: ServiceAccount
  name: fluxbase
roleRef:
  kind: Role
  name: secret-reader
  apiGroup: rbac.authorization.k8s.io
```

---

## Input Validation

### 1. Validate All User Input

```typescript
import validator from "validator";

interface CreateUserInput {
  email: string;
  name: string;
  age?: number;
  website?: string;
}

function validateUserInput(input: CreateUserInput): string[] {
  const errors: string[] = [];

  // Email validation
  if (!validator.isEmail(input.email)) {
    errors.push("Invalid email address");
  }

  // Name validation
  if (!input.name || input.name.length < 2 || input.name.length > 100) {
    errors.push("Name must be between 2 and 100 characters");
  }

  // Age validation
  if (input.age !== undefined) {
    if (!Number.isInteger(input.age) || input.age < 0 || input.age > 150) {
      errors.push("Age must be between 0 and 150");
    }
  }

  // URL validation
  if (input.website && !validator.isURL(input.website)) {
    errors.push("Invalid website URL");
  }

  return errors;
}
```

### 2. Sanitize User Input

```typescript
import DOMPurify from "isomorphic-dompurify";

// Sanitize HTML
const safeHTML = DOMPurify.sanitize(userInput);

// Escape for SQL (use parameterized queries instead!)
import escape from "pg-escape";

// Validate and sanitize file uploads
function validateUpload(file: File): boolean {
  const allowedTypes = ["image/jpeg", "image/png", "image/gif"];
  const maxSize = 5 * 1024 * 1024; // 5MB

  if (!allowedTypes.includes(file.type)) {
    throw new Error("Invalid file type");
  }

  if (file.size > maxSize) {
    throw new Error("File too large");
  }

  return true;
}
```

### 3. Use Type Validation Libraries

```typescript
import { z } from "zod";

// Define schema
const userSchema = z.object({
  email: z.string().email(),
  name: z.string().min(2).max(100),
  age: z.number().int().min(0).max(150).optional(),
  website: z.string().url().optional(),
});

// Validate input
try {
  const validatedData = userSchema.parse(userInput);
  // Use validatedData safely
} catch (error) {
  if (error instanceof z.ZodError) {
    console.error("Validation errors:", error.errors);
  }
}
```

### 4. Implement Rate Limiting for Forms

```typescript
// Client-side debouncing
import { debounce } from "lodash";

const debouncedSubmit = debounce(
  async (data) => {
    await submitForm(data);
  },
  1000,
  { leading: true, trailing: false }
);
```

---

## Error Handling

### 1. Don't Leak Sensitive Information

```typescript
// ✅ GOOD: Generic error messages
try {
  await client.auth.signIn({ email, password });
} catch (error) {
  throw new Error("Invalid email or password");
}

// ❌ BAD: Reveals whether user exists
try {
  const user = await findUser(email);
  if (!user) {
    throw new Error("User not found");
  }
  if (!verifyPassword(password, user.password_hash)) {
    throw new Error("Incorrect password");
  }
} catch (error) {
  // Reveals too much information
  throw error;
}
```

### 2. Log Errors Securely

```typescript
// ✅ GOOD: Log without sensitive data
logger.error("Authentication failed", {
  ip: req.ip,
  user_agent: req.headers["user-agent"],
  timestamp: new Date().toISOString(),
});

// ❌ BAD: Logs sensitive data
logger.error("Authentication failed", {
  email: req.body.email,
  password: req.body.password, // NEVER LOG PASSWORDS!
  token: req.headers.authorization,
});
```

### 3. Implement Error Boundaries

```typescript
// Global error handler
app.use((err: Error, req, res, next) => {
  // Log full error internally
  logger.error("Unhandled error", { error: err, stack: err.stack });

  // Return generic error to client
  res.status(500).json({
    error: "Internal server error",
    message:
      process.env.NODE_ENV === "development"
        ? err.message
        : "An unexpected error occurred",
  });
});
```

---

## Logging & Monitoring

### 1. Enable Audit Logging

```yaml
# fluxbase.yaml
logging:
  level: "info"
  audit_enabled: true
  audit_log_file: "/var/log/fluxbase/audit.log"
  audit_events:
    - "auth.login"
    - "auth.logout"
    - "auth.signup"
    - "auth.password_reset"
    - "admin.user_create"
    - "admin.user_delete"
    - "admin.role_change"
```

### 2. Monitor Security Events

```typescript
// Set up alerts for suspicious activity
const alerts = {
  failed_logins: {
    threshold: 10,
    window: "5m",
    action: "notify_admin",
  },
  unusual_api_activity: {
    threshold: 1000,
    window: "1m",
    action: "rate_limit",
  },
  admin_actions: {
    threshold: 1,
    window: "0s",
    action: "log_and_notify",
  },
};
```

### 3. Use Structured Logging

```typescript
import winston from "winston";

const logger = winston.createLogger({
  format: winston.format.json(),
  transports: [
    new winston.transports.File({ filename: "error.log", level: "error" }),
    new winston.transports.File({ filename: "combined.log" }),
  ],
});

logger.info("User logged in", {
  user_id: user.id,
  ip: req.ip,
  user_agent: req.headers["user-agent"],
  timestamp: new Date().toISOString(),
});
```

### 4. Monitor Performance Metrics

```yaml
# Enable Prometheus metrics
monitoring:
  enabled: true
  port: 9090
  path: "/metrics"

# Monitor key metrics
metrics:
  - request_duration
  - request_count
  - error_rate
  - active_connections
  - database_query_time
  - cache_hit_rate
```

---

## Deployment Security

### 1. Use Container Security Scanning

```bash
# Scan Docker images
docker scan fluxbase/fluxbase:latest

# Use Trivy
trivy image fluxbase/fluxbase:latest

# Use Snyk
snyk container test fluxbase/fluxbase:latest
```

### 2. Run as Non-Root User

```dockerfile
# Dockerfile
FROM node:25-alpine

# Create non-root user
RUN addgroup -g 1001 -S fluxbase && \
    adduser -S fluxbase -u 1001

# Switch to non-root user
USER fluxbase

# Run application
CMD ["node", "server.js"]
```

### 3. Use Read-Only File System

```yaml
# docker-compose.yml
services:
  fluxbase:
    image: ghcr.io/fluxbase-eu/fluxbase:latest:latest
    read_only: true
    tmpfs:
      - /tmp
      - /var/run
```

### 4. Limit Container Resources

```yaml
# docker-compose.yml
services:
  fluxbase:
    deploy:
      resources:
        limits:
          cpus: "2"
          memory: 2G
        reservations:
          cpus: "1"
          memory: 1G
```

### 5. Enable Security Contexts (Kubernetes)

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      securityContext:
        runAsNonRoot: true
        runAsUser: 1001
        fsGroup: 1001
      containers:
        - name: fluxbase
          securityContext:
            allowPrivilegeEscalation: false
            readOnlyRootFilesystem: true
            capabilities:
              drop:
                - ALL
```

---

## Incident Response

### 1. Create Incident Response Plan

```markdown
# Incident Response Plan

## Detection

- Monitor logs for suspicious activity
- Set up automated alerts
- Regular security audits

## Containment

1. Isolate affected systems
2. Block malicious IPs
3. Revoke compromised credentials
4. Enable additional logging

## Eradication

1. Identify root cause
2. Patch vulnerabilities
3. Remove malicious code
4. Update security rules

## Recovery

1. Restore from clean backups
2. Verify system integrity
3. Monitor for persistence
4. Gradually restore services

## Post-Incident

1. Document incident
2. Update security measures
3. Train team members
4. Conduct retrospective
```

### 2. Prepare Recovery Procedures

```bash
#!/bin/bash
# disaster-recovery.sh

# 1. Stop services
systemctl stop fluxbase

# 2. Restore database from backup
pg_restore -U postgres -d fluxbase /backups/latest.dump

# 3. Verify integrity
psql -U postgres -d fluxbase -c "SELECT COUNT(*) FROM auth.users"

# 4. Start services
systemctl start fluxbase

# 5. Verify functionality
curl https://api.example.com/health
```

### 3. Maintain Security Contacts

```yaml
# contacts.yaml
security_team:
  - name: "Security Lead"
    email: "security@example.com"
    phone: "+1-555-0100"

  - name: "Infrastructure Lead"
    email: "infra@example.com"
    phone: "+1-555-0101"

external_contacts:
  - name: "Security Researcher"
    email: "researcher@example.com"

  - name: "Hosting Provider"
    email: "support@hosting.com"
    phone: "+1-555-0200"
```

---

## Security Checklist

### Pre-Production

- [ ] Strong password policies enforced
- [ ] 2FA enabled for admin accounts
- [ ] HTTPS/TLS configured with valid certificates
- [ ] Database connections encrypted
- [ ] RLS policies reviewed and tested
- [ ] Rate limiting configured
- [ ] Security headers configured
- [ ] CORS properly configured
- [ ] Secrets stored securely (not in code)
- [ ] Input validation implemented
- [ ] Error handling doesn't leak information
- [ ] Audit logging enabled
- [ ] Backup strategy implemented
- [ ] Firewall rules configured
- [ ] Container security scanning passed
- [ ] Dependency vulnerabilities resolved
- [ ] Penetration testing completed

### Post-Production

- [ ] Monitor logs daily
- [ ] Review audit logs weekly
- [ ] Update dependencies monthly
- [ ] Rotate secrets quarterly
- [ ] Conduct security audits annually
- [ ] Test backups monthly
- [ ] Review access controls quarterly
- [ ] Update incident response plan annually
- [ ] Train team on security quarterly

---

## Further Reading

- [Security Overview](./overview.md)
- [CSRF Protection](./csrf-protection.md)
- [Security Headers](./security-headers.md)
- [Row Level Security](../guides/row-level-security.md)
- [Rate Limiting](../guides/rate-limiting.md)
- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [CWE Top 25](https://cwe.mitre.org/top25/)

---

## Summary

Security is a continuous process, not a one-time task:

- ✅ **Authentication**: Strong passwords, 2FA, short-lived tokens
- ✅ **Authorization**: RLS, RBAC, least privilege
- ✅ **Network**: HTTPS, firewall, rate limiting
- ✅ **Secrets**: Environment variables, secrets management
- ✅ **Validation**: Input validation, sanitization, type checking
- ✅ **Errors**: Generic messages, secure logging
- ✅ **Monitoring**: Audit logs, alerts, metrics
- ✅ **Deployment**: Container security, non-root user
- ✅ **Response**: Incident plan, recovery procedures

Follow these best practices and stay vigilant to maintain a secure Fluxbase instance.
