---
title: "Webhooks"
---

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
  apiKey: process.env.FLUXBASE_CLIENT_KEY, // Or use service key for backend
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

```javascript
const express = require("express");
const { FluxbaseClient } = require("@fluxbase/sdk");

const app = express();
app.use(express.json());

app.post("/webhooks/fluxbase", (req, res) => {
  // Verify signature
  const signature = req.headers["x-fluxbase-signature"];
  const isValid = FluxbaseClient.verifyWebhookSignature(
    req.body,
    signature,
    process.env.FLUXBASE_WEBHOOK_SECRET
  );

  if (!isValid) {
    return res.status(401).send("Invalid signature");
  }

  // Process event
  const { event, table, record, old_record } = req.body;

  switch (event) {
    case "INSERT":
      console.log("New record:", record.id);
      break;
    case "UPDATE":
      console.log("Updated:", record.id);
      break;
    case "DELETE":
      console.log("Deleted:", record.id);
      break;
  }

  res.status(200).send("OK");
});

app.listen(3000);
```

---

## Best Practices

| Practice | Description |
|----------|-------------|
| **Respond quickly** | Return 200 status immediately, process asynchronously if needed (30s timeout) |
| **Handle duplicates** | Use record IDs to ensure idempotent processing |
| **Verify signatures** | Always verify webhook signatures using SDK utilities |
| **Use HTTPS** | Secure webhook URLs with HTTPS in production |
| **Log deliveries** | Keep detailed logs of events for debugging |
| **Monitor failures** | Use `client.webhooks.getDeliveries()` to track failed deliveries |

**Example: Async processing**

```javascript
app.post("/webhooks", async (req, res) => {
  res.status(200).send("OK"); // Respond immediately
  processWebhookEvent(req.body).catch(console.error); // Process async
});
```

---

## Common Use Cases

- **Send notifications**: Trigger emails/push notifications on events
- **Sync data**: Keep external systems (Salesforce, CRM) in sync
- **Analytics**: Stream events to analytics platforms
- **Trigger workflows**: Start automated workflows based on database changes

---

## Testing Webhooks

```typescript
// Test via SDK
await client.webhooks.test(webhookId, {
  event: "INSERT",
  table: "test_table",
  record: { id: "test-id", name: "Test Record" }
});

// Or use dashboard: Webhooks → Select webhook → Test
```

---

## Monitoring & Troubleshooting

**Monitor deliveries:**

```typescript
const deliveries = await client.webhooks.getDeliveries(webhookId, { limit: 100 });
```

**Retry behavior:** Failed deliveries automatically retry (default: 3 attempts, exponential backoff, 30s timeout)

**Common issues:**

| Issue | Solution |
|-------|----------|
| Webhook not triggering | Verify table name, operations configured, webhook enabled |
| Delivery failures | Check endpoint is public, responds within timeout, valid SSL |
| Signature failing | Use correct secret, SDK's `verifyWebhookSignature()`, raw body |

---

## Learn More

- [Authentication](/docs/guides/authentication) - Authenticate to manage webhooks
- [Row Level Security](/docs/guides/row-level-security) - Secure your data
- [Realtime](/docs/guides/realtime) - Alternative to webhooks for client apps
