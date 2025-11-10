# Webhooks

Webhooks allow you to receive real-time notifications when events occur in your Fluxbase database. Instead of continuously polling for changes, webhooks push data to your application instantly.

## Overview

Fluxbase webhooks trigger HTTP requests to your specified endpoint whenever database events occur, such as:

- **INSERT**: A new row is added to a table
- **UPDATE**: An existing row is modified
- **DELETE**: A row is removed from a table

---

## Using the SDK (Recommended)

### Installation

```bash
npm install @fluxbase/sdk
```

### Quick Start

```typescript
import { FluxbaseClient } from "@fluxbase/sdk";

// Initialize client (requires authentication)
const client = new FluxbaseClient({
  url: "http://localhost:8080",
  apiKey: process.env.FLUXBASE_API_KEY, // Or use service key for backend
});

// Create a webhook
const webhook = await client.webhooks.create({
  name: "User Events",
  description: "Notify external system of user changes",
  url: "https://example.com/webhooks/fluxbase",
  secret: "your-webhook-secret",
  enabled: true,
  events: [
    {
      table: "users",
      operations: ["INSERT", "UPDATE"],
    },
  ],
  max_retries: 3,
  timeout_seconds: 30,
});

console.log("Webhook created:", webhook.id);

// List all webhooks
const webhooks = await client.webhooks.list();

// Get webhook details
const details = await client.webhooks.get(webhook.id);

// Update webhook
await client.webhooks.update(webhook.id, {
  enabled: false,
  events: [
    {
      table: "users",
      operations: ["INSERT", "UPDATE", "DELETE"],
    },
  ],
});

// Delete webhook
await client.webhooks.delete(webhook.id);
```

---

### Managing Webhooks

#### Create a Webhook

```typescript
const webhook = await client.webhooks.create({
  name: "Order Notifications",
  description: "Send notifications when orders are created or updated",
  url: "https://api.myapp.com/webhooks/orders",
  secret: "secure-random-secret-string",
  enabled: true,
  events: [
    {
      table: "orders",
      operations: ["INSERT", "UPDATE"],
    },
    {
      table: "order_items",
      operations: ["INSERT"],
    },
  ],
  max_retries: 3,
  timeout_seconds: 30,
  retry_backoff_seconds: 5,
});

console.log("Webhook ID:", webhook.id);
console.log("Webhook URL:", webhook.url);
```

**Configuration Options:**

| Field                   | Type     | Description                                       |
| ----------------------- | -------- | ------------------------------------------------- |
| `name`                  | string   | Descriptive name for the webhook                  |
| `description`           | string   | Optional details about the webhook's purpose      |
| `url`                   | string   | The endpoint that will receive webhook events     |
| `secret`                | string   | Optional webhook secret for verifying requests    |
| `enabled`               | boolean  | Whether the webhook is active                     |
| `events`                | array    | List of event configurations                      |
| `max_retries`           | number   | Retry attempts for failed deliveries (default: 3) |
| `timeout_seconds`       | number   | Request timeout in seconds (default: 30)          |
| `retry_backoff_seconds` | number   | Seconds between retries (default: 5)              |

**Event Configuration:**

```typescript
{
  table: "table_name",           // Table to monitor
  operations: ["INSERT", "UPDATE", "DELETE"]  // Events to trigger on
}
```

#### List Webhooks

```typescript
const webhooks = await client.webhooks.list();

webhooks.forEach((webhook) => {
  console.log(`${webhook.name}: ${webhook.enabled ? "Enabled" : "Disabled"}`);
  console.log(`  URL: ${webhook.url}`);
  console.log(`  Events: ${webhook.events.length} configured`);
});
```

#### Get Webhook Details

```typescript
const webhook = await client.webhooks.get("webhook-id");

console.log("Name:", webhook.name);
console.log("Enabled:", webhook.enabled);
console.log("Events:", webhook.events);
console.log("Max Retries:", webhook.max_retries);
console.log("Created:", webhook.created_at);
```

#### Update Webhook

```typescript
// Enable/disable webhook
await client.webhooks.update(webhookId, {
  enabled: false,
});

// Update events
await client.webhooks.update(webhookId, {
  events: [
    {
      table: "users",
      operations: ["INSERT", "UPDATE", "DELETE"],
    },
  ],
});

// Update URL and secret
await client.webhooks.update(webhookId, {
  url: "https://new-endpoint.com/webhooks",
  secret: "new-secret-key",
});
```

#### Delete Webhook

```typescript
await client.webhooks.delete(webhookId);
console.log("Webhook deleted");
```

#### View Delivery History

```typescript
const deliveries = await client.webhooks.getDeliveries(webhookId, {
  limit: 50,
  offset: 0,
});

deliveries.forEach((delivery) => {
  console.log(`${delivery.created_at}: ${delivery.status}`);
  console.log(`  Response: ${delivery.response_status}`);
  console.log(`  Attempts: ${delivery.attempts}`);
  if (delivery.error) {
    console.log(`  Error: ${delivery.error}`);
  }
});
```

---

### Verifying Webhook Signatures

The SDK provides utilities to verify webhook signatures in your webhook receiver.

```typescript
import { FluxbaseClient } from "@fluxbase/sdk";
import express from "express";

const app = express();
app.use(express.json());

const WEBHOOK_SECRET = process.env.FLUXBASE_WEBHOOK_SECRET!;

app.post("/webhooks/fluxbase", (req, res) => {
  const signature = req.headers["x-fluxbase-signature"] as string;
  const payload = req.body;

  // Verify signature using SDK utility
  const isValid = FluxbaseClient.verifyWebhookSignature(
    payload,
    signature,
    WEBHOOK_SECRET
  );

  if (!isValid) {
    return res.status(401).send("Invalid signature");
  }

  // Process the webhook event
  const { event, table, record } = payload;
  console.log(`Received ${event} event for ${table}`);

  // Handle event...

  res.status(200).send("OK");
});

app.listen(3000);
```

---

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

| Field        | Type         | Description                                     |
| ------------ | ------------ | ----------------------------------------------- |
| `event`      | string       | The operation type: "INSERT", "UPDATE", "DELETE" |
| `table`      | string       | Name of the table where the event occurred      |
| `schema`     | string       | Database schema (usually "public")              |
| `record`     | object       | The new/current state of the record             |
| `old_record` | object\|null | Previous state (only for UPDATE and DELETE)     |
| `timestamp`  | string       | ISO 8601 timestamp when the event occurred      |
| `webhook_id` | string       | UUID of the webhook configuration               |

### Event-Specific Payloads

#### INSERT Event

```json
{
  "event": "INSERT",
  "record": {
    /* new record data */
  },
  "old_record": null
}
```

#### UPDATE Event

```json
{
  "event": "UPDATE",
  "record": {
    /* updated record data */
  },
  "old_record": {
    /* previous record data */
  }
}
```

#### DELETE Event

```json
{
  "event": "DELETE",
  "record": {
    /* deleted record data */
  },
  "old_record": {
    /* same as record */
  }
}
```

---

## Receiving Webhooks

### Example: Express.js Handler

```javascript
const express = require("express");
const { FluxbaseClient } = require("@fluxbase/sdk");

const app = express();
app.use(express.json());

const WEBHOOK_SECRET = process.env.FLUXBASE_WEBHOOK_SECRET;

app.post("/webhooks/fluxbase", (req, res) => {
  // Verify webhook signature using SDK
  const signature = req.headers["x-fluxbase-signature"];
  const isValid = FluxbaseClient.verifyWebhookSignature(
    req.body,
    signature,
    WEBHOOK_SECRET
  );

  if (!isValid) {
    return res.status(401).send("Invalid signature");
  }

  // Process the webhook event
  const { event, table, record, old_record } = req.body;

  console.log(`Received ${event} event for ${table}`);
  console.log("Record:", record);

  // Handle the event
  switch (event) {
    case "INSERT":
      handleInsert(table, record);
      break;
    case "UPDATE":
      handleUpdate(table, record, old_record);
      break;
    case "DELETE":
      handleDelete(table, record);
      break;
  }

  // Acknowledge receipt
  res.status(200).send("OK");
});

function handleInsert(table, record) {
  console.log("New record created:", record.id);
}

function handleUpdate(table, record, oldRecord) {
  console.log("Record updated:", record.id);
}

function handleDelete(table, record) {
  console.log("Record deleted:", record.id);
}

app.listen(3000);
```

### Example: Next.js API Route

```typescript
// pages/api/webhooks/fluxbase.ts
import { NextApiRequest, NextApiResponse } from "next";
import { FluxbaseClient } from "@fluxbase/sdk";

export default async function handler(
  req: NextApiRequest,
  res: NextApiResponse
) {
  if (req.method !== "POST") {
    return res.status(405).json({ error: "Method not allowed" });
  }

  // Verify webhook signature using SDK
  const signature = req.headers["x-fluxbase-signature"] as string;
  const secret = process.env.FLUXBASE_WEBHOOK_SECRET!;

  const isValid = FluxbaseClient.verifyWebhookSignature(
    req.body,
    signature,
    secret
  );

  if (!isValid) {
    return res.status(401).json({ error: "Invalid signature" });
  }

  const { event, table, record, old_record } = req.body;

  // Process the event
  try {
    switch (event) {
      case "INSERT":
        await handleNewUser(record);
        break;
      case "UPDATE":
        await handleUserUpdate(record, old_record);
        break;
      case "DELETE":
        await handleUserDeletion(record);
        break;
    }

    res.status(200).json({ received: true });
  } catch (error) {
    console.error("Webhook processing error:", error);
    res.status(500).json({ error: "Processing failed" });
  }
}

async function handleNewUser(record: any) {
  // Send welcome email, create profile, etc.
  console.log("New user created:", record.email);
}

async function handleUserUpdate(record: any, oldRecord: any) {
  // Check what changed
  if (record.email !== oldRecord.email) {
    console.log("Email changed:", oldRecord.email, "->", record.email);
  }
}

async function handleUserDeletion(record: any) {
  // Clean up related data
  console.log("User deleted:", record.id);
}
```

---

## Best Practices

### 1. Respond Quickly

Your webhook endpoint should respond within the timeout period (default 30s). Process events asynchronously if needed:

```javascript
app.post("/webhooks", async (req, res) => {
  // Acknowledge receipt immediately
  res.status(200).send("OK");

  // Process asynchronously
  processWebhookEvent(req.body).catch((err) => {
    console.error("Async processing error:", err);
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
    console.log("Event already processed");
    return;
  }

  // Process the event
  await db.insert({ external_id: record.id, ...record });
}
```

### 3. Secure Your Endpoint

- Always verify webhook signatures in production using SDK utilities
- Use HTTPS for webhook URLs
- Consider IP allowlisting if available
- Monitor for unusual traffic patterns

### 4. Log Everything

Keep detailed logs of webhook deliveries for debugging:

```javascript
app.post("/webhooks", (req, res) => {
  const logEntry = {
    timestamp: new Date(),
    event: req.body.event,
    table: req.body.table,
    record_id: req.body.record?.id,
    webhook_id: req.body.webhook_id,
  };

  logger.info("Webhook received", logEntry);

  // Process...
});
```

### 5. Monitor Delivery Failures

Use the SDK to monitor webhook delivery failures:

```typescript
// Check for failed deliveries
const deliveries = await client.webhooks.getDeliveries(webhookId);
const failed = deliveries.filter((d) => d.status === "failed");

if (failed.length > 0) {
  console.warn(`${failed.length} failed deliveries for webhook ${webhookId}`);
  // Set up alerts, investigate issues
}
```

---

## Common Use Cases

### 1. Send Notifications

Trigger emails or push notifications when events occur:

```javascript
async function handleNewOrder(record) {
  await sendEmail({
    to: record.customer_email,
    subject: "Order Confirmation",
    body: `Your order #${record.id} has been received!`,
  });
}
```

### 2. Sync Data

Keep external systems in sync with your database:

```javascript
async function syncToSalesforce(record, oldRecord) {
  if (record.status !== oldRecord.status) {
    await salesforce.updateDeal(record.id, {
      stage: record.status,
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
    timestamp: new Date(),
  });
}
```

### 4. Trigger Workflows

Start automated workflows based on database changes:

```javascript
async function handleUserSignup(record) {
  await workflow.start("user_onboarding", {
    userId: record.id,
    email: record.email,
  });
}
```

---

## Testing Webhooks

### Using the Dashboard

1. Navigate to **Webhooks** in your dashboard
2. Click on a webhook to view details
3. Click **Test** to send a sample payload to your endpoint
4. View delivery logs to see the response

### Using the SDK

```typescript
// Trigger a test webhook event
await client.webhooks.test(webhookId, {
  event: "INSERT",
  table: "test_table",
  record: {
    id: "test-id",
    name: "Test Record",
  },
});
```

### Using webhook.site

For quick testing without setting up a server:

1. Go to [webhook.site](https://webhook.site)
2. Copy the unique URL
3. Create a webhook using the SDK with this URL
4. Trigger events in your database
5. View the received payloads on webhook.site

---

## Monitoring & Troubleshooting

### Monitoring Deliveries

Use the SDK to monitor webhook deliveries:

```typescript
const deliveries = await client.webhooks.getDeliveries(webhookId, {
  limit: 100,
});

// Check delivery status
deliveries.forEach((delivery) => {
  console.log(`${delivery.created_at}: ${delivery.status}`);
  console.log(`  Response: ${delivery.response_status}`);
  console.log(`  Attempts: ${delivery.attempts}`);
  console.log(`  Duration: ${delivery.response_time_ms}ms`);
});
```

### Retry Behavior

Failed deliveries are automatically retried based on your configuration:

- **Retry Count**: Configurable (default: 3 attempts)
- **Backoff Strategy**: Exponential backoff between retries
- **Timeout**: Configurable request timeout (default: 30s)

### Common Issues

**Webhook Not Triggering:**

- Verify the table name is correct
- Check that the operations (INSERT/UPDATE/DELETE) are configured
- Ensure the webhook is enabled using `client.webhooks.get(id)`
- Check if the event actually occurred in the database

**Delivery Failures:**

- Verify your endpoint is publicly accessible
- Check your server logs for errors
- Ensure your endpoint responds within the timeout period
- Verify SSL certificates if using HTTPS

**Signature Verification Failing:**

- Ensure you're using the correct webhook secret
- Use the SDK's `verifyWebhookSignature()` utility
- Verify you're signing the raw request body (before parsing JSON)

---

## Advanced: REST API Reference

For direct HTTP access or custom integrations, Fluxbase provides a complete REST API for webhook management.

### Create Webhook

```
POST /api/v1/webhooks
```

**Headers:**

```
Authorization: Bearer {access_token}
Content-Type: application/json
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

**Response (201 Created):**

```json
{
  "id": "webhook-uuid",
  "name": "User Events",
  "url": "https://example.com/webhooks/fluxbase",
  "enabled": true,
  "events": [...],
  "created_at": "2025-11-02T10:00:00Z"
}
```

---

### List Webhooks

```
GET /api/v1/webhooks
```

**Headers:**

```
Authorization: Bearer {access_token}
```

**Response (200 OK):**

```json
{
  "webhooks": [
    {
      "id": "webhook-uuid",
      "name": "User Events",
      "url": "https://example.com/webhooks/fluxbase",
      "enabled": true,
      "created_at": "2025-11-02T10:00:00Z"
    }
  ]
}
```

---

### Get Webhook

```
GET /api/v1/webhooks/{webhook_id}
```

**Headers:**

```
Authorization: Bearer {access_token}
```

**Response (200 OK):**

```json
{
  "id": "webhook-uuid",
  "name": "User Events",
  "description": "Notify external system of user changes",
  "url": "https://example.com/webhooks/fluxbase",
  "secret_configured": true,
  "enabled": true,
  "events": [...],
  "max_retries": 3,
  "timeout_seconds": 30,
  "retry_backoff_seconds": 5,
  "created_at": "2025-11-02T10:00:00Z",
  "updated_at": "2025-11-02T10:00:00Z"
}
```

---

### Update Webhook

```
PUT /api/v1/webhooks/{webhook_id}
```

**Headers:**

```
Authorization: Bearer {access_token}
Content-Type: application/json
```

**Request Body:**

```json
{
  "enabled": false,
  "events": [
    {
      "table": "users",
      "operations": ["INSERT", "UPDATE", "DELETE"]
    }
  ]
}
```

**Response (200 OK):**

```json
{
  "id": "webhook-uuid",
  "enabled": false,
  ...
}
```

---

### Delete Webhook

```
DELETE /api/v1/webhooks/{webhook_id}
```

**Headers:**

```
Authorization: Bearer {access_token}
```

**Response (204 No Content)**

---

### List Deliveries

```
GET /api/v1/webhooks/{webhook_id}/deliveries?limit=50&offset=0
```

**Headers:**

```
Authorization: Bearer {access_token}
```

**Response (200 OK):**

```json
{
  "deliveries": [
    {
      "id": "delivery-uuid",
      "webhook_id": "webhook-uuid",
      "event": "INSERT",
      "table": "users",
      "status": "success",
      "response_status": 200,
      "response_time_ms": 145,
      "attempts": 1,
      "created_at": "2025-11-02T10:30:00Z"
    }
  ],
  "total": 150,
  "limit": 50,
  "offset": 0
}
```

---

## Learn More

- [Authentication](/docs/guides/authentication) - Authenticate to manage webhooks
- [Row Level Security](/docs/guides/row-level-security) - Secure your data
- [Realtime](/docs/guides/realtime) - Alternative to webhooks for client apps
