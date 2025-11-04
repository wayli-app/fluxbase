# Email Services

Fluxbase includes a built-in email system for sending authentication emails (magic links, password resets, email verification) and custom transactional emails.

## Overview

The email system supports multiple providers:

- **SMTP** - Standard SMTP servers (Gmail, Outlook, custom servers)
- **SendGrid** - SendGrid API integration
- **Mailgun** - Mailgun API integration
- **AWS SES** - Amazon Simple Email Service

All providers support:
- Magic link authentication
- Email verification
- Password reset emails
- Custom HTML email templates
- Fallback to default templates

## Quick Start

### 1. Choose a Provider

Select an email provider based on your needs:

| Provider | Best For | Pricing | Setup Difficulty |
|----------|----------|---------|------------------|
| **SMTP** | Development, custom servers | Free (using your own server) | Easy |
| **SendGrid** | Production, high volume | Free tier: 100 emails/day | Easy |
| **Mailgun** | Production, flexibility | Free tier: 5,000 emails/month | Easy |
| **AWS SES** | AWS infrastructure, scalability | Pay-as-you-go | Medium |

### 2. Configure Email Provider

#### Option A: Environment Variables

```bash
# Basic settings (all providers)
FLUXBASE_EMAIL_ENABLED=true
FLUXBASE_EMAIL_PROVIDER=smtp  # smtp, sendgrid, mailgun, ses
FLUXBASE_EMAIL_FROM_ADDRESS=noreply@yourapp.com
FLUXBASE_EMAIL_FROM_NAME="Your App Name"

# SMTP Settings
FLUXBASE_EMAIL_SMTP_HOST=smtp.gmail.com
FLUXBASE_EMAIL_SMTP_PORT=587
FLUXBASE_EMAIL_SMTP_USERNAME=your-email@gmail.com
FLUXBASE_EMAIL_SMTP_PASSWORD=your-app-password
FLUXBASE_EMAIL_SMTP_TLS=true
```

#### Option B: Configuration File

Create `fluxbase.yaml`:

```yaml
email:
  enabled: true
  provider: smtp
  from_address: noreply@yourapp.com
  from_name: "Your App Name"

  # SMTP configuration
  smtp_host: smtp.gmail.com
  smtp_port: 587
  smtp_username: your-email@gmail.com
  smtp_password: your-app-password
  smtp_tls: true
```

### 3. Test Email Configuration

Start Fluxbase and trigger a password reset to test:

```bash
curl -X POST http://localhost:8080/api/v1/auth/password/reset \
  -H "Content-Type: application/json" \
  -d '{"email": "test@example.com"}'
```

Check your email inbox for the password reset email.

---

## Provider Configuration

### SMTP

Standard SMTP configuration for any email server.

**Configuration:**

```yaml
email:
  enabled: true
  provider: smtp
  from_address: noreply@yourapp.com
  from_name: "Your App Name"
  reply_to_address: support@yourapp.com  # Optional

  smtp_host: smtp.example.com
  smtp_port: 587
  smtp_username: your-username
  smtp_password: your-password
  smtp_tls: true  # Use STARTTLS
```

**Environment Variables:**

```bash
FLUXBASE_EMAIL_ENABLED=true
FLUXBASE_EMAIL_PROVIDER=smtp
FLUXBASE_EMAIL_FROM_ADDRESS=noreply@yourapp.com
FLUXBASE_EMAIL_FROM_NAME="Your App Name"
FLUXBASE_EMAIL_SMTP_HOST=smtp.example.com
FLUXBASE_EMAIL_SMTP_PORT=587
FLUXBASE_EMAIL_SMTP_USERNAME=your-username
FLUXBASE_EMAIL_SMTP_PASSWORD=your-password
FLUXBASE_EMAIL_SMTP_TLS=true
```

**Common SMTP Providers:**

#### Gmail

```yaml
email:
  smtp_host: smtp.gmail.com
  smtp_port: 587
  smtp_username: your-email@gmail.com
  smtp_password: your-app-password  # Generate app password in Google Account settings
  smtp_tls: true
```

**Important:** Gmail requires an App Password. Generate one at: https://myaccount.google.com/apppasswords

#### Outlook/Office 365

```yaml
email:
  smtp_host: smtp.office365.com
  smtp_port: 587
  smtp_username: your-email@outlook.com
  smtp_password: your-password
  smtp_tls: true
```

#### Custom SMTP Server

```yaml
email:
  smtp_host: mail.yourserver.com
  smtp_port: 587  # or 465 for SSL, 25 for unencrypted
  smtp_username: user@yourserver.com
  smtp_password: your-password
  smtp_tls: true
```

---

### SendGrid

SendGrid provides reliable email delivery with good deliverability rates.

**Configuration:**

```yaml
email:
  enabled: true
  provider: sendgrid
  from_address: noreply@yourapp.com
  from_name: "Your App Name"

  sendgrid_api_key: SG.xxxxxxxxxxxxxxxxxxxxx
```

**Environment Variables:**

```bash
FLUXBASE_EMAIL_ENABLED=true
FLUXBASE_EMAIL_PROVIDER=sendgrid
FLUXBASE_EMAIL_FROM_ADDRESS=noreply@yourapp.com
FLUXBASE_EMAIL_FROM_NAME="Your App Name"
FLUXBASE_EMAIL_SENDGRID_API_KEY=SG.xxxxxxxxxxxxxxxxxxxxx
```

**Setup Steps:**

1. Sign up at https://sendgrid.com
2. Verify your sender identity (single email or domain)
3. Create an API key:
   - Go to Settings → API Keys
   - Click "Create API Key"
   - Choose "Full Access" or "Restricted Access" (Mail Send permission required)
   - Copy the API key (shown only once)
4. Add API key to your Fluxbase configuration

**Domain Verification:**

For production, verify your domain to improve deliverability:
- Go to Settings → Sender Authentication → Domain Authentication
- Follow the DNS setup instructions
- Add the provided DNS records to your domain

**Free Tier:** 100 emails/day

---

### Mailgun

Mailgun offers flexible email sending with good developer tools.

**Configuration:**

```yaml
email:
  enabled: true
  provider: mailgun
  from_address: noreply@yourapp.com
  from_name: "Your App Name"

  mailgun_api_key: key-xxxxxxxxxxxxxxxxxxxxx
  mailgun_domain: mg.yourapp.com
```

**Environment Variables:**

```bash
FLUXBASE_EMAIL_ENABLED=true
FLUXBASE_EMAIL_PROVIDER=mailgun
FLUXBASE_EMAIL_FROM_ADDRESS=noreply@yourapp.com
FLUXBASE_EMAIL_FROM_NAME="Your App Name"
FLUXBASE_EMAIL_MAILGUN_API_KEY=key-xxxxxxxxxxxxxxxxxxxxx
FLUXBASE_EMAIL_MAILGUN_DOMAIN=mg.yourapp.com
```

**Setup Steps:**

1. Sign up at https://www.mailgun.com
2. Add and verify your domain:
   - Go to Sending → Domains
   - Click "Add New Domain"
   - Add the provided DNS records to your domain
   - Wait for verification (usually 5-10 minutes)
3. Get your API key:
   - Go to Settings → API Keys
   - Copy the "Private API key"
4. Add credentials to your Fluxbase configuration

**Sandbox Domain:**

For testing, Mailgun provides a sandbox domain. Only works for authorized recipients:
```yaml
email:
  mailgun_domain: sandboxXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX.mailgun.org
```

**Free Tier:** 5,000 emails/month for first 3 months, then pay-as-you-go

---

### AWS SES (Amazon Simple Email Service)

AWS SES is cost-effective and integrates well with AWS infrastructure.

**Configuration:**

```yaml
email:
  enabled: true
  provider: ses
  from_address: noreply@yourapp.com
  from_name: "Your App Name"

  ses_access_key: AKIAXXXXXXXXXXXXXXXX
  ses_secret_key: XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
  ses_region: us-east-1
```

**Environment Variables:**

```bash
FLUXBASE_EMAIL_ENABLED=true
FLUXBASE_EMAIL_PROVIDER=ses
FLUXBASE_EMAIL_FROM_ADDRESS=noreply@yourapp.com
FLUXBASE_EMAIL_FROM_NAME="Your App Name"
FLUXBASE_EMAIL_SES_ACCESS_KEY=AKIAXXXXXXXXXXXXXXXX
FLUXBASE_EMAIL_SES_SECRET_KEY=XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
FLUXBASE_EMAIL_SES_REGION=us-east-1
```

**Setup Steps:**

1. Sign up for AWS and go to SES console: https://console.aws.amazon.com/ses
2. Choose your region (us-east-1, eu-west-1, etc.)
3. Verify your email address or domain:
   - For testing: Verify a single email address
   - For production: Verify your domain
4. Request production access:
   - By default, SES is in sandbox mode (can only send to verified addresses)
   - Go to "Account Dashboard" → "Request Production Access"
   - Fill out the form explaining your use case
   - Approval usually takes 24-48 hours
5. Create IAM user with SES permissions:
   - Go to IAM → Users → Create User
   - Attach policy: `AmazonSESFullAccess` (or create custom policy with only `ses:SendEmail`)
   - Create access key and copy credentials
6. Add credentials to Fluxbase configuration

**Regions:**

Common SES regions:
- `us-east-1` - US East (N. Virginia)
- `us-west-2` - US West (Oregon)
- `eu-west-1` - EU (Ireland)
- `ap-southeast-1` - Asia Pacific (Singapore)

**Pricing:** $0.10 per 1,000 emails (very cost-effective for high volume)

---

## Email Templates

Fluxbase uses HTML templates for authentication emails. You can customize these templates or use the defaults.

### Default Templates

Fluxbase includes professional default templates for:
- Magic link authentication
- Email verification
- Password reset

### Custom Templates

Override default templates by specifying custom template paths:

```yaml
email:
  magic_link_template: /path/to/magic-link.html
  verification_template: /path/to/verification.html
  password_reset_template: /path/to/password-reset.html
```

**Template Variables:**

Your custom templates have access to these variables:

- `{{.Link}}` - The full URL for the action (magic link, verification, reset)
- `{{.Token}}` - The token (if you want to build your own link)

**Example Custom Template:**

Create `templates/password-reset.html`:

```html
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Reset Your Password</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            background-color: #f4f4f4;
            margin: 0;
            padding: 20px;
        }
        .container {
            max-width: 600px;
            margin: 0 auto;
            background: white;
            padding: 40px;
            border-radius: 8px;
        }
        .button {
            display: inline-block;
            padding: 12px 24px;
            background-color: #007bff;
            color: white;
            text-decoration: none;
            border-radius: 4px;
            margin: 20px 0;
        }
        .footer {
            margin-top: 30px;
            font-size: 12px;
            color: #666;
        }
    </style>
</head>
<body>
    <div class="container">
        <h2>Reset Your Password</h2>
        <p>You requested to reset your password. Click the button below to set a new password:</p>
        <a href="{{.Link}}" class="button">Reset Password</a>
        <p>This link will expire in 1 hour.</p>
        <p>If you didn't request this, please ignore this email.</p>
        <div class="footer">
            <p>This is an automated email from Your App Name.</p>
        </div>
    </div>
</body>
</html>
```

Then configure:

```yaml
email:
  password_reset_template: ./templates/password-reset.html
```

---

## Development Setup

### MailHog (Local Testing)

For local development, use MailHog to capture emails without sending them:

**1. Start MailHog with Docker:**

```bash
docker run -d -p 1025:1025 -p 8025:8025 mailhog/mailhog
```

**2. Configure Fluxbase:**

```yaml
email:
  enabled: true
  provider: smtp
  from_address: dev@localhost
  smtp_host: localhost
  smtp_port: 1025
  smtp_tls: false
```

**3. View emails:**

Open http://localhost:8025 to see captured emails.

### Docker Compose

Add MailHog to your `docker-compose.yml`:

```yaml
version: '3.8'
services:
  fluxbase:
    image: fluxbase:latest
    environment:
      FLUXBASE_EMAIL_ENABLED: "true"
      FLUXBASE_EMAIL_PROVIDER: smtp
      FLUXBASE_EMAIL_FROM_ADDRESS: dev@localhost
      FLUXBASE_EMAIL_SMTP_HOST: mailhog
      FLUXBASE_EMAIL_SMTP_PORT: "1025"
      FLUXBASE_EMAIL_SMTP_TLS: "false"
    depends_on:
      - mailhog

  mailhog:
    image: mailhog/mailhog:latest
    ports:
      - "1025:1025"  # SMTP
      - "8025:8025"  # Web UI
```

---

## Troubleshooting

### Emails Not Sending

**1. Check if email is enabled:**

```bash
# View logs
docker logs fluxbase

# Look for: "Email service initialized with provider: smtp"
```

**2. Verify configuration:**

```bash
# Test SMTP connection
telnet smtp.gmail.com 587

# Should connect successfully
```

**3. Check authentication:**

```yaml
# Make sure credentials are correct
smtp_username: your-email@gmail.com
smtp_password: your-password
```

### Gmail "Less Secure App" Error

Gmail blocks "less secure apps" by default. Use an App Password instead:

1. Enable 2FA on your Google Account
2. Go to https://myaccount.google.com/apppasswords
3. Generate an app password for "Mail"
4. Use the generated password in your configuration

### SendGrid Authentication Error

**Error:** `401 Unauthorized`

**Solution:** Verify your API key has "Mail Send" permission:
1. Go to SendGrid → Settings → API Keys
2. Click on your API key
3. Ensure "Mail Send" permission is enabled

### Mailgun Domain Not Verified

**Error:** `Domain not verified`

**Solution:** Add Mailgun's DNS records to your domain:
1. Go to Mailgun → Sending → Domains → [Your Domain]
2. Copy the TXT and MX records
3. Add them to your DNS provider (Cloudflare, GoDaddy, etc.)
4. Wait 5-10 minutes for propagation
5. Click "Verify DNS Settings"

### AWS SES Sandbox Limitation

**Error:** `Email address is not verified`

**Solution:**
- In sandbox mode, you can only send to verified email addresses
- Verify recipient emails in SES console, or
- Request production access (recommended for production)

### SMTP Timeout

**Error:** `dial tcp: i/o timeout`

**Solutions:**
1. Check firewall rules allow outbound SMTP (port 587 or 465)
2. Verify SMTP host and port are correct
3. Try different ports: 587 (TLS), 465 (SSL), 25 (unencrypted)
4. Check if your hosting provider blocks SMTP (some do)

---

## Best Practices

### 1. Use Environment Variables for Secrets

Never commit API keys or passwords to version control:

```bash
# ✅ Good
FLUXBASE_EMAIL_SMTP_PASSWORD=your-password

# ❌ Bad - Don't put in config file
smtp_password: my-actual-password
```

### 2. Verify Your Domain

For production:
- Always verify your sending domain
- Improves deliverability significantly
- Reduces chance of emails going to spam

### 3. Use Different Providers for Dev/Prod

```yaml
# Development
email:
  provider: smtp
  smtp_host: localhost
  smtp_port: 1025  # MailHog

# Production
email:
  provider: sendgrid
  sendgrid_api_key: ${SENDGRID_API_KEY}
```

### 4. Monitor Email Delivery

- Set up email delivery webhooks (available in SendGrid, Mailgun, SES)
- Monitor bounce rates and spam complaints
- Keep bounce rate < 5%

### 5. Implement Rate Limiting

Avoid hitting provider rate limits:

| Provider | Rate Limit |
|----------|------------|
| Gmail | 500/day (free), 2,000/day (paid) |
| SendGrid | 100/day (free), unlimited (paid) |
| Mailgun | 5,000/month (free), unlimited (paid) |
| AWS SES | 1 email/sec (sandbox), higher in production |

### 6. Handle Failures Gracefully

```go
// Application code should handle email failures
if err := emailService.Send(ctx, to, subject, body); err != nil {
    log.Error().Err(err).Msg("Failed to send email")
    // Don't block user registration/login if email fails
    // Store for retry later
}
```

---

## Security Considerations

### 1. Protect Email Credentials

- Store API keys in environment variables or secrets management
- Use IAM roles (AWS) or service accounts where possible
- Rotate credentials regularly

### 2. Validate Email Addresses

- Use proper email validation
- Implement rate limiting on email endpoints
- Prevent email enumeration attacks

### 3. SPF, DKIM, DMARC

Configure email authentication records:

**SPF (Sender Policy Framework):**
```
v=spf1 include:sendgrid.net ~all
```

**DKIM (DomainKeys Identified Mail):**
Configured automatically by your email provider

**DMARC (Domain-based Message Authentication):**
```
v=DMARC1; p=quarantine; rua=mailto:dmarc@yourapp.com
```

Add these DNS records to improve deliverability and prevent spoofing.

### 4. Content Security

- Sanitize user-provided content in emails
- Don't expose sensitive information in email
- Use HTTPS links only
- Include unsubscribe links for marketing emails

---

## Next Steps

- [Authentication](authentication) - Learn about magic links and password resets
- [Configuration Reference](../reference/configuration) - Complete configuration options
- [Production Deployment](../deployment/overview) - Deploy to production
