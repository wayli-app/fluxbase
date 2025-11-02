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

1. **Environment Variables**: Never commit sensitive credentials to version control
2. **Database**: Use strong passwords and enable SSL/TLS for database connections
3. **API Keys**: Rotate API keys and tokens regularly
4. **Updates**: Keep Fluxbase and all dependencies up to date
5. **Monitoring**: Enable audit logging and monitor for suspicious activity
6. **Network**: Deploy behind a reverse proxy with TLS/SSL termination
7. **Backups**: Implement regular encrypted backups

## Known Security Considerations

### Development Mode
- Development mode includes additional debugging features that should **never** be enabled in production
- Ensure `FLUXBASE_ENV=production` in production deployments

### Database Access
- RLS policies are enforced at the database level
- Ensure PostgreSQL RLS is properly configured before production use
- Review RLS policies regularly

### Rate Limiting
- Default rate limits are configured conservatively
- Adjust rate limits based on your specific use case
- Consider implementing additional DDoS protection at the infrastructure level

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
