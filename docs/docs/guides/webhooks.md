# Webhooks

Webhooks allow you to receive real-time notifications when events occur in your Fluxbase database. Instead of continuously polling for changes, webhooks push data to your application instantly.

## Overview

Fluxbase webhooks trigger HTTP requests to your specified endpoint whenever database events occur, such as:

- **INSERT**: A new row is added to a table
- **UPDATE**: An existing row is modified
- **DELETE**: A row is removed from a table

## Setting Up Webhooks

### 1. Create a Webhook

Navigate to the **Webhooks** page in your Fluxbase dashboard and click **Create Webhook**.

Configure the following:

- **Name**: A descriptive name for your webhook
- **Description**: Optional details about the webhook's purpose
- **URL**: The endpoint that will receive webhook events
- **Secret**: Optional webhook secret for verifying request authenticity
- **Events**: Select which tables and operations to monitor

### 2. Configure Events

For each webhook, you can configure multiple event listeners:

```typescript
{
  table: "users",           // Table to monitor
  operations: ["INSERT", "UPDATE", "DELETE"]  // Events to trigger on
}
```

#### Event Configuration Options

| Field | Type | Description |
|-------|------|-------------|
| `table` | string | The table name to monitor (e.g., "users", "products") |
| `operations` | string[] | Array of operations: "INSERT", "UPDATE", "DELETE" |

### 3. Advanced Settings

- **Max Retries**: Number of retry attempts for failed deliveries (default: 3)
- **Timeout**: Request timeout in seconds (default: 30)
- **Retry Backoff**: Seconds to wait between retries (default: exponential backoff)

## Webhook Payload

When an event occurs, Fluxbase sends a POST request to your webhook URL with the following payload:

```json
{
  "event": "INSERT",
  "table": "users",
  "schema": "public",
  "record": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "created_at": "2025-11-02T10:30:00Z"
  },
  "old_record": null,
  "timestamp": "2025-11-02T10:30:00.123Z",
  "webhook_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7"
}
```

### Payload Fields

| Field | Type | Description |
|-------|------|-------------|
| `event` | string | The operation type: "INSERT", "UPDATE", or "DELETE" |
| `table` | string | Name of the table where the event occurred |
| `schema` | string | Database schema (usually "public") |
| `record` | object | The new/current state of the record |
| `old_record` | object\|null | Previous state (only for UPDATE and DELETE) |
| `timestamp` | string | ISO 8601 timestamp when the event occurred |
| `webhook_id` | string | UUID of the webhook configuration |

### Event-Specific Payloads

#### INSERT Event
```json
{
  "event": "INSERT",
  "record": { /* new record data */ },
  "old_record": null
}
```

#### UPDATE Event
```json
{
  "event": "UPDATE",
  "record": { /* updated record data */ },
  "old_record": { /* previous record data */ }
}
```

#### DELETE Event
```json
{
  "event": "DELETE",
  "record": { /* deleted record data */ },
  "old_record": { /* same as record */ }
}
```

## Receiving Webhooks

### Example: Express.js Handler

```javascript
const express = require('express');
const crypto = require('crypto');

const app = express();
app.use(express.json());

// Webhook secret from Fluxbase dashboard
const WEBHOOK_SECRET = process.env.FLUXBASE_WEBHOOK_SECRET;

app.post('/webhooks/fluxbase', (req, res) => {
  // Verify webhook signature
  const signature = req.headers['x-fluxbase-signature'];
  const payload = JSON.stringify(req.body);

  if (WEBHOOK_SECRET) {
    const expectedSignature = crypto
      .createHmac('sha256', WEBHOOK_SECRET)
      .update(payload)
      .digest('hex');

    if (signature !== expectedSignature) {
      return res.status(401).send('Invalid signature');
    }
  }

  // Process the webhook event
  const { event, table, record } = req.body;

  console.log(`Received ${event} event for ${table}`);
  console.log('Record:', record);

  // Handle the event
  switch (event) {
    case 'INSERT':
      handleInsert(table, record);
      break;
    case 'UPDATE':
      handleUpdate(table, record, req.body.old_record);
      break;
    case 'DELETE':
      handleDelete(table, record);
      break;
  }

  // Acknowledge receipt
  res.status(200).send('OK');
});

app.listen(3000);
```

### Example: Next.js API Route

```typescript
// pages/api/webhooks/fluxbase.ts
import { NextApiRequest, NextApiResponse } from 'next';
import crypto from 'crypto';

export default async function handler(
  req: NextApiRequest,
  res: NextApiResponse
) {
  if (req.method !== 'POST') {
    return res.status(405).json({ error: 'Method not allowed' });
  }

  // Verify webhook signature
  const signature = req.headers['x-fluxbase-signature'] as string;
  const secret = process.env.FLUXBASE_WEBHOOK_SECRET!;

  const expectedSignature = crypto
    .createHmac('sha256', secret)
    .update(JSON.stringify(req.body))
    .digest('hex');

  if (signature !== expectedSignature) {
    return res.status(401).json({ error: 'Invalid signature' });
  }

  const { event, table, record, old_record } = req.body;

  // Process the event
  try {
    switch (event) {
      case 'INSERT':
        await handleNewUser(record);
        break;
      case 'UPDATE':
        await handleUserUpdate(record, old_record);
        break;
      case 'DELETE':
        await handleUserDeletion(record);
        break;
    }

    res.status(200).json({ received: true });
  } catch (error) {
    console.error('Webhook processing error:', error);
    res.status(500).json({ error: 'Processing failed' });
  }
}

async function handleNewUser(record: any) {
  // Send welcome email, create profile, etc.
  console.log('New user created:', record.email);
}

async function handleUserUpdate(record: any, oldRecord: any) {
  // Check what changed
  if (record.email !== oldRecord.email) {
    console.log('Email changed:', oldRecord.email, '->', record.email);
  }
}

async function handleUserDeletion(record: any) {
  // Clean up related data
  console.log('User deleted:', record.id);
}
```

## Verifying Webhook Signatures

Fluxbase signs all webhook requests using HMAC SHA-256. Always verify signatures in production to ensure requests are authentic.

### Signature Header

Webhook requests include an `X-Fluxbase-Signature` header containing the HMAC signature of the request body.

### Verification Steps

1. Get the signature from the `X-Fluxbase-Signature` header
2. Compute the expected signature using your webhook secret
3. Compare the signatures using a constant-time comparison

```javascript
const crypto = require('crypto');

function verifyWebhookSignature(payload, signature, secret) {
  const expectedSignature = crypto
    .createHmac('sha256', secret)
    .update(JSON.stringify(payload))
    .digest('hex');

  return crypto.timingSafeEqual(
    Buffer.from(signature),
    Buffer.from(expectedSignature)
  );
}
```

## Testing Webhooks

### Using the Dashboard

1. Navigate to **Webhooks** in your dashboard
2. Click on a webhook to view details
3. Click **Test** to send a sample payload to your endpoint
4. View delivery logs to see the response

### Using webhook.site

For quick testing without setting up a server:

1. Go to [webhook.site](https://webhook.site)
2. Copy the unique URL
3. Use this URL when creating a webhook in Fluxbase
4. Trigger events in your database
5. View the received payloads on webhook.site

## Monitoring Webhook Deliveries

The Webhooks page shows delivery history for each webhook:

- **Status**: Success (2xx), Failed (4xx/5xx), or Pending
- **Attempts**: Number of delivery attempts
- **Response Time**: How long the endpoint took to respond
- **Response Body**: The response from your endpoint
- **Timestamp**: When the delivery was attempted

### Retry Behavior

Failed deliveries are automatically retried based on your configuration:

- **Retry Count**: Configurable (default: 3 attempts)
- **Backoff Strategy**: Exponential backoff between retries
- **Timeout**: Configurable request timeout (default: 30s)

## Best Practices

### 1. Respond Quickly

Your webhook endpoint should respond within the timeout period (default 30s). Process events asynchronously if needed:

```javascript
app.post('/webhooks', async (req, res) => {
  // Acknowledge receipt immediately
  res.status(200).send('OK');

  // Process asynchronously
  processWebhookEvent(req.body).catch(err => {
    console.error('Async processing error:', err);
  });
});
```

### 2. Handle Duplicate Events

Webhooks may be delivered more than once. Make your processing idempotent:

```javascript
async function handleInsert(record) {
  // Use record ID to ensure idempotency
  const existing = await db.findOne({ external_id: record.id });

  if (existing) {
    console.log('Event already processed');
    return;
  }

  // Process the event
  await db.insert({ external_id: record.id, ...record });
}
```

### 3. Secure Your Endpoint

- Always verify webhook signatures in production
- Use HTTPS for webhook URLs
- Consider IP allowlisting if available
- Monitor for unusual traffic patterns

### 4. Log Everything

Keep detailed logs of webhook deliveries for debugging:

```javascript
app.post('/webhooks', (req, res) => {
  const logEntry = {
    timestamp: new Date(),
    event: req.body.event,
    table: req.body.table,
    record_id: req.body.record?.id,
    webhook_id: req.body.webhook_id
  };

  logger.info('Webhook received', logEntry);

  // Process...
});
```

### 5. Monitor Delivery Failures

Set up alerts for webhook delivery failures:

- Check delivery logs regularly
- Monitor retry exhaustion
- Investigate 4xx errors (client-side issues)
- Monitor 5xx errors (server-side issues)

## Common Use Cases

### 1. Send Notifications

Trigger emails or push notifications when events occur:

```javascript
async function handleNewOrder(record) {
  await sendEmail({
    to: record.customer_email,
    subject: 'Order Confirmation',
    body: `Your order #${record.id} has been received!`
  });
}
```

### 2. Sync Data

Keep external systems in sync with your database:

```javascript
async function syncToSalesforce(record, oldRecord) {
  if (record.status !== oldRecord.status) {
    await salesforce.updateDeal(record.id, {
      stage: record.status
    });
  }
}
```

### 3. Analytics & Logging

Stream events to analytics platforms:

```javascript
async function trackEvent(event, record) {
  await analytics.track({
    event: `${event}_${record.table}`,
    properties: record,
    timestamp: new Date()
  });
}
```

### 4. Trigger Workflows

Start automated workflows based on database changes:

```javascript
async function handleUserSignup(record) {
  await workflow.start('user_onboarding', {
    userId: record.id,
    email: record.email
  });
}
```

## Troubleshooting

### Webhook Not Triggering

- Verify the table name is correct
- Check that the operations (INSERT/UPDATE/DELETE) are configured
- Ensure the webhook is enabled
- Check if the event actually occurred in the database

### Delivery Failures

- Verify your endpoint is publicly accessible
- Check your server logs for errors
- Ensure your endpoint responds within the timeout period
- Verify SSL certificates if using HTTPS

### Signature Verification Failing

- Ensure you're using the correct webhook secret
- Verify you're signing the raw request body (before parsing JSON)
- Use the exact payload format Fluxbase sends
- Use constant-time comparison to prevent timing attacks

## API Reference

### Create Webhook

```
POST /api/v1/webhooks
```

**Request Body:**
```json
{
  "name": "User Events",
  "description": "Notify external system of user changes",
  "url": "https://example.com/webhooks/fluxbase",
  "secret": "your-webhook-secret",
  "enabled": true,
  "events": [
    {
      "table": "users",
      "operations": ["INSERT", "UPDATE"]
    }
  ],
  "max_retries": 3,
  "timeout_seconds": 30,
  "retry_backoff_seconds": 5
}
```

### Update Webhook

```
PUT /api/v1/webhooks/{webhook_id}
```

### Delete Webhook

```
DELETE /api/v1/webhooks/{webhook_id}
```

### List Deliveries

```
GET /api/v1/webhooks/{webhook_id}/deliveries?limit=50
```

## Learn More

- [REST API Documentation](/docs/api/rest)
- [Row Level Security](/docs/guides/row-level-security)
- [Authentication](/docs/guides/authentication)
