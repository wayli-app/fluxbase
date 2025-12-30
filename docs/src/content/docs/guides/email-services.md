---
title: "Email Services"
---

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

**Choose a provider:**

| Provider     | Best For                    |
| ------------ | --------------------------- |
| **SMTP**     | Development, custom servers |
| **SendGrid** | Production, high volume     |
| **Mailgun**  | Production, flexibility     |
| **AWS SES**  | AWS infrastructure          |

**Configure (environment variables):**

```bash
FLUXBASE_EMAIL_ENABLED=true
FLUXBASE_EMAIL_PROVIDER=smtp  # smtp, sendgrid, mailgun, ses
FLUXBASE_EMAIL_FROM_ADDRESS=noreply@yourapp.com
FLUXBASE_EMAIL_FROM_NAME="Your App Name"

# SMTP example
FLUXBASE_EMAIL_SMTP_HOST=smtp.gmail.com
FLUXBASE_EMAIL_SMTP_PORT=587
FLUXBASE_EMAIL_SMTP_USERNAME=your-email@gmail.com
FLUXBASE_EMAIL_SMTP_PASSWORD=your-app-password
FLUXBASE_EMAIL_SMTP_TLS=true
```

:::note[Configuration Management]
Email settings can be controlled via **environment variables** or the **admin UI**. When set via environment variables, UI settings become read-only. See [Configuration Management](/docs/guides/admin/configuration-management) for details.
:::

---

## Provider Configuration

### SMTP

```bash
FLUXBASE_EMAIL_PROVIDER=smtp
FLUXBASE_EMAIL_SMTP_HOST=smtp.gmail.com
FLUXBASE_EMAIL_SMTP_PORT=587
FLUXBASE_EMAIL_SMTP_USERNAME=your-email@gmail.com
FLUXBASE_EMAIL_SMTP_PASSWORD=your-app-password
FLUXBASE_EMAIL_SMTP_TLS=true
```

**Common SMTP hosts:**

| Provider | Host                | Port | Notes                                                              |
| -------- | ------------------- | ---- | ------------------------------------------------------------------ |
| Gmail    | smtp.gmail.com      | 587  | Requires [App Password](https://myaccount.google.com/apppasswords) |
| Outlook  | smtp.office365.com  | 587  | Use account password                                               |
| Custom   | mail.yourserver.com | 587  | TLS recommended                                                    |

---

### SendGrid

```bash
FLUXBASE_EMAIL_PROVIDER=sendgrid
FLUXBASE_EMAIL_SENDGRID_API_KEY=SG.xxxxxxxxxxxxxxxxxxxxx
```

**Setup:**

1. Sign up at [sendgrid.com](https://sendgrid.com)
2. Settings → client keys → Create API Key (Mail Send permission)
3. Verify domain: Settings → Sender Authentication → Add DNS records

---

### Mailgun

```bash
FLUXBASE_EMAIL_PROVIDER=mailgun
FLUXBASE_EMAIL_MAILGUN_API_KEY=key-xxxxxxxxxxxxxxxxxxxxx
FLUXBASE_EMAIL_MAILGUN_DOMAIN=mg.yourapp.com
```

**Setup:**

1. Sign up at [mailgun.com](https://www.mailgun.com)
2. Add domain: Sending → Domains → Add DNS records
3. Get API key: Settings → client keys → Copy Private API key

---

### AWS SES

```bash
FLUXBASE_EMAIL_PROVIDER=ses
FLUXBASE_EMAIL_SES_ACCESS_KEY=AKIAXXXXXXXXXXXXXXXX
FLUXBASE_EMAIL_SES_SECRET_KEY=XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
FLUXBASE_EMAIL_SES_REGION=us-east-1
```

**Setup:**

1. Go to [SES console](https://console.aws.amazon.com/ses)
2. Verify email/domain
3. Request production access (sandbox mode only sends to verified addresses)
4. Create IAM user with `AmazonSESFullAccess` policy

---

## Email Templates

Fluxbase includes default HTML templates for magic links, email verification, and password resets.

**Custom templates:**

```yaml
email:
  magic_link_template: /path/to/magic-link.html
  verification_template: /path/to/verification.html
  password_reset_template: /path/to/password-reset.html
```

**Template variables:**

- `{{.Link}}` - Full action URL
- `{{.Token}}` - Token only

---

## Troubleshooting

| Issue                           | Solution                                                                                                                             |
| ------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| **Emails not sending**          | Check logs for "Email service initialized", verify SMTP connection with `telnet smtp.gmail.com 587`, confirm credentials are correct |
| **Gmail "Less secure app"**     | Use [App Password](https://myaccount.google.com/apppasswords): Enable 2FA → Generate app password → Use in config                    |
| **SendGrid 401**                | Verify API key has "Mail Send" permission: Settings → client keys → Check permissions                                                   |
| **Mailgun domain not verified** | Add DNS records: Copy TXT/MX records → Add to DNS provider → Wait 5-10 min → Verify                                                  |
| **AWS SES sandbox**             | In sandbox, only sends to verified addresses. Request production access or verify recipients                                         |
| **SMTP timeout**                | Check firewall allows port 587/465, verify host/port, try different ports, check if hosting provider blocks SMTP                     |

---

## Best Practices

| Practice                      | Description                                                                              |
| ----------------------------- | ---------------------------------------------------------------------------------------- |
| **Use environment variables** | Never commit client keys/passwords to version control. Use `FLUXBASE_EMAIL_*` env vars      |
| **Verify domain**             | Always verify sending domain in production to improve deliverability and avoid spam      |
| **Separate dev/prod**         | Use SMTP/MailHog for development, SendGrid/Mailgun/SES for production                    |
| **Monitor delivery**          | Set up webhooks to track bounces/complaints. Keep bounce rate < 5%                       |
| **Respect rate limits**       | Gmail: 500/day, SendGrid: 100/day (free), Mailgun: 5k/month (free), SES: 1/sec (sandbox) |
| **Handle failures**           | Don't block user flows if email fails. Log errors and retry later                        |
| **Protect credentials**       | Store in env vars/secrets manager, use IAM roles where possible, rotate regularly        |
| **Email authentication**      | Configure SPF, DKIM, DMARC DNS records to prevent spoofing and improve deliverability    |
| **Content security**          | Sanitize user content, use HTTPS links, include unsubscribe for marketing emails         |
