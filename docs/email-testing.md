# Email Testing with MailHog

This guide explains how to test email functionality in Fluxbase using MailHog.

## What is MailHog?

[MailHog](https://github.com/mailhog/MailHog) is a fake SMTP server that captures emails for testing. It provides:

- An SMTP server on port 1025
- A web UI on port 8025 to view captured emails
- An API to programmatically access emails

## Setup

MailHog is automatically included in the devcontainer. It's already running and accessible at:

- **SMTP Server**: `mailhog:1025` (from within containers) or `localhost:1025` (from host)
- **Web UI**: http://localhost:8025
- **API**: http://localhost:8025/api

##Configuration

To use MailHog for testing, configure your email settings:

### Environment Variables

```bash
FLUXBASE_EMAIL_ENABLED=true
FLUXBASE_EMAIL_PROVIDER=smtp
FLUXBASE_EMAIL_SMTP_HOST=mailhog  # or localhost if running from host
FLUXBASE_EMAIL_SMTP_PORT=1025
FLUXBASE_EMAIL_SMTP_TLS=false
FLUXBASE_EMAIL_FROM_ADDRESS=test@fluxbase.eu
FLUXBASE_EMAIL_FROM_NAME=Fluxbase Test
```

### YAML Configuration

```yaml
email:
  enabled: true
  provider: smtp
  smtp_host: mailhog
  smtp_port: 1025
  smtp_tls: false
  from_address: test@fluxbase.eu
  from_name: Fluxbase Test
```

## Running Tests

### Unit Tests Only

```bash
make test-unit
```

### Integration Tests with MailHog

```bash
# Run all email integration tests
MAILHOG_HOST=mailhog make test-email

# Or directly with go test
MAILHOG_HOST=mailhog go test -v -race -tags=integration ./internal/email/...
```

### All Tests

```bash
make test
```

## Viewing Emails

### Web UI

Open http://localhost:8025 in your browser to see all captured emails with:

- Full email content (HTML and plain text)
- Headers
- Attachments
- Source view

### API

```bash
# Get all messages
curl http://localhost:8025/api/v2/messages

# Get a specific message
curl http://localhost:8025/api/v2/messages/{message_id}

# Delete all messages
curl -X DELETE http://localhost:8025/api/v1/messages
```

## Manual Testing

You can send test emails using the SMTP service:

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/wayli-app/fluxbase/internal/config"
    "github.com/wayli-app/fluxbase/internal/email"
)

func main() {
    cfg := &config.EmailConfig{
        Enabled:      true,
        Provider:     "smtp",
        SMTPHost:     "localhost",
        SMTPPort:     1025,
        SMTPTLS:      false,
        FromAddress:  "test@fluxbase.eu",
        FromName:     "Fluxbase Test",
        MagicLinkExpiry: 15 * time.Minute,
    }

    service := email.NewSMTPService(cfg)

    // Send a magic link
    ctx := context.Background()
    err := service.SendMagicLink(
        ctx,
        "user@example.com",
        "test-token-123",
        "https://example.com/auth/verify?token=test-token-123",
    )
    if err != nil {
        log.Fatal(err)
    }

    log.Println("Email sent! Check http://localhost:8025")
}
```

## Integration Test Features

The email integration tests verify:

1. **Magic Link Emails**: Sends and validates magic link emails with proper formatting
2. **Verification Emails**: Sends and validates email verification links
3. **Custom Emails**: Sends generic emails with custom subject and body
4. **Multiple Recipients**: Tests sending multiple emails concurrently
5. **Email Content**: Validates HTML templates, headers, and metadata
6. **MailHog Connectivity**: Ensures MailHog is accessible and responding

## Troubleshooting

### MailHog Not Accessible

```bash
# Check if MailHog is running
curl http://localhost:8025/api/v2/messages

# Check from within devcontainer
curl http://mailhog:8025/api/v2/messages
```

### Tests Failing

1. Ensure MailHog is running (it should be automatic in devcontainer)
2. Check that `MAILHOG_HOST` environment variable is set correctly
3. Verify no firewall is blocking ports 1025 or 8025
4. Check MailHog logs for errors

### Clear All Emails

```bash
curl -X DELETE http://localhost:8025/api/v1/messages
```

## Production Configuration

In production, replace MailHog with a real SMTP provider:

### Gmail

```bash
FLUXBASE_EMAIL_SMTP_HOST=smtp.gmail.com
FLUXBASE_EMAIL_SMTP_PORT=587
FLUXBASE_EMAIL_SMTP_USERNAME=your-email@gmail.com
FLUXBASE_EMAIL_SMTP_PASSWORD=your-app-password
FLUXBASE_EMAIL_SMTP_TLS=true
```

### SendGrid

```bash
FLUXBASE_EMAIL_PROVIDER=sendgrid
FLUXBASE_EMAIL_SENDGRID_API_KEY=your-api-key
```

### AWS SES

```bash
FLUXBASE_EMAIL_PROVIDER=ses
FLUXBASE_EMAIL_SES_ACCESS_KEY=your-access-key
FLUXBASE_EMAIL_SES_SECRET_KEY=your-secret-key
FLUXBASE_EMAIL_SES_REGION=us-east-1
```

## Best Practices

1. **Always use MailHog for development** - Never send test emails to real addresses
2. **Clear emails between tests** - Use the DELETE API to reset state
3. **Validate email content** - Check both HTML and headers in tests
4. **Test error cases** - Verify behavior when SMTP server is unavailable
5. **Use integration tags** - Keep integration tests separate with build tags

## Resources

- [MailHog GitHub](https://github.com/mailhog/MailHog)
- [MailHog API Documentation](https://github.com/mailhog/MailHog/blob/master/docs/APIv2.md)
- [Go SMTP Package](https://pkg.go.dev/net/smtp)
