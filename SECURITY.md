# Security Policy

## Supported Versions

Currently supported versions of Fluxbase:

| Version | Supported          |
| ------- | ------------------ |
| 1.0.x   | :white_check_mark: |

## Reporting a Vulnerability

We take security vulnerabilities seriously. If you discover a security issue in Fluxbase, please report it responsibly.

### How to Report

**Please DO NOT report security vulnerabilities through public GitHub issues.**

Instead, please send an email to the maintainers with:
- Description of the vulnerability
- Steps to reproduce the issue
- Potential impact
- Suggested fix (if any)

### What to Expect

- **Initial Response**: We aim to acknowledge your report within 48 hours
- **Status Updates**: We will keep you informed about the progress of fixing the vulnerability
- **Disclosure Timeline**: We aim to release a fix within 90 days of the initial report
- **Credit**: We will credit you in the security advisory (unless you prefer to remain anonymous)

## Security Features

Fluxbase includes several built-in security features:

### Authentication & Authorization
- JWT-based authentication with secure token handling
- OAuth 2.0 provider support (Google, GitHub, etc.)
- Two-Factor Authentication (TOTP)
- Row-Level Security (RLS) for data access control
- Role-based access control (RBAC)

### API Security
- Rate limiting on all endpoints
- CORS configuration
- Input validation and sanitization
- SQL injection protection via prepared statements
- XSS prevention through proper output encoding

### Infrastructure Security
- Automated security scanning (Trivy, gosec, golangci-lint)
- Dependency vulnerability monitoring (nancy)
- Regular security audits via GitHub Security tab
- Secure default configurations

### Data Protection
- Password hashing with bcrypt
- Encrypted sensitive data storage
- Secure session management
- PostgreSQL RLS for data isolation

## Security Best Practices

When deploying Fluxbase:

### Configuration Security

1. **JWT Secret**: Must be changed from default and at least 32 characters long
   - The server will refuse to start with the default insecure JWT secret
   - Generate a secure secret: `openssl rand -base64 32`

2. **Setup Token**: Must be changed from default before production deployment
   - The server validates against known insecure defaults
   - Use a cryptographically secure random token

3. **Environment Variables**: Never commit sensitive credentials to version control
   - Use `.env` files for development
   - Use proper secrets management in production (K8s Secrets, AWS Secrets Manager, etc.)

4. **Database**: Use strong passwords and enable SSL/TLS for database connections
   - Change default postgres password
   - Enable `ssl_mode=require` in production

### API & Authentication Security

5. **Rate Limiting**: All sensitive endpoints have built-in rate limiting
   - Login: 10 attempts per 15 minutes
   - 2FA verification: 5 attempts per 5 minutes
   - Password reset: 5 requests per 15 minutes
   - Admin setup: 5 attempts per 15 minutes

6. **Token Storage**: Access tokens use cookies with security attributes
   - `SameSite=Strict` for CSRF protection
   - `Secure` flag in production (HTTPS only)

7. **Client keys**: Rotate client keys and tokens regularly

### Operational Security

8. **Updates**: Keep Fluxbase and all dependencies up to date
9. **Monitoring**: Enable audit logging and monitor for suspicious activity
10. **Network**: Deploy behind a reverse proxy with TLS/SSL termination
11. **Backups**: Implement regular encrypted backups

## Known Security Considerations

### Development Mode
- Development mode includes additional debugging features that should **never** be enabled in production
- Ensure `FLUXBASE_ENV=production` in production deployments
- Debug mode logs may contain sensitive information

### Database Access
- RLS policies are enforced at the database level
- Ensure PostgreSQL RLS is properly configured before production use
- Review RLS policies regularly
- Service role tokens bypass RLS - use with caution

### Rate Limiting
- Default rate limits are configured conservatively
- Adjust rate limits based on your specific use case
- Consider implementing additional DDoS protection at the infrastructure level

### Cookie Security
- Access tokens are stored in cookies with `SameSite=Strict` and `Secure` flags
- For maximum security in sensitive applications, consider implementing server-side session management
- Refresh tokens are stored in localStorage (standard practice, but consider the trade-offs)

### JWT Security
- JWTs are signed using HMAC-SHA256 algorithm
- Algorithm verification is enforced to prevent algorithm confusion attacks
- Token revocation is supported via token blacklist

## Security Scanning

This project uses automated security scanning:

- **Trivy**: Filesystem and dependency vulnerability scanning
- **gosec**: Go security-focused static analysis
- **golangci-lint**: Multiple security linters enabled
- **nancy**: Dependency vulnerability checks

Security scans run:
- On every pull request
- On every push to main/develop branches
- Daily at 2 AM UTC via scheduled workflow

Results are available in the GitHub Security tab.

## Compliance

Fluxbase is designed to help you meet common compliance requirements:

- **GDPR**: RLS and data isolation features support data privacy requirements
- **SOC 2**: Audit logging and access controls support compliance efforts
- **HIPAA**: Encryption and access control features can support healthcare compliance (additional configuration required)

**Note**: Fluxbase provides security features, but achieving compliance requires proper configuration and operational practices.

## Contact

For security-related questions or concerns, please reach out to the maintainers.

---

**Last Updated**: 2025-11-03
