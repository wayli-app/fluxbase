---
title: Management SDK
sidebar_position: 4
---

# Management SDK

The Management SDK provides tools for managing API keys, webhooks, and invitations in your Fluxbase instance. These features allow you to:

- **API Keys**: Create and manage API keys for service-to-service authentication
- **Webhooks**: Set up event-driven integrations with external services
- **Invitations**: Invite new users to join your dashboard

## Installation

The management module is included with the Fluxbase SDK:

```bash
npm install @fluxbase/sdk
```

## Quick Start

```typescript
import { createClient } from "@fluxbase/sdk";

const client = createClient({
  url: "http://localhost:8080",
});

// Authenticate first
await client.auth.login({
  email: "user@example.com",
  password: "password",
});

// Create an API key
const { api_key, key } = await client.management.apiKeys.create({
  name: "Production Service",
  scopes: ["read:users", "write:users"],
  rate_limit_per_minute: 100,
});

// Create a webhook
const webhook = await client.management.webhooks.create({
  url: "https://myapp.com/webhook",
  events: ["user.created", "user.updated"],
});

// Create an invitation (admin only)
await client.admin.login({
  email: "admin@example.com",
  password: "admin-password",
});

const invitation = await client.management.invitations.create({
  email: "newuser@example.com",
  role: "dashboard_user",
});
```

---

## API Keys Management

API keys provide a secure way for external services to authenticate with your Fluxbase instance without using user credentials.

### Create API Key

Create a new API key with specific scopes and rate limits.

```typescript
const { api_key, key } = await client.management.apiKeys.create({
  name: "Production Service",
  description: "API key for production microservice",
  scopes: ["read:users", "write:users", "read:products"],
  rate_limit_per_minute: 100,
  expires_at: "2025-12-31T23:59:59Z", // Optional expiration
});

// ⚠️ IMPORTANT: Store the full key securely - it won't be shown again
console.log("API Key:", key); // fb_live_abc123def456...
console.log("Key Prefix:", api_key.key_prefix); // fb_live_abc
```

**Parameters:**

- `name` (required): Human-readable name for the API key
- `description` (optional): Detailed description
- `scopes` (required): Array of permission scopes
- `rate_limit_per_minute` (required): Maximum requests per minute
- `expires_at` (optional): ISO 8601 expiration date

**Returns:** Object containing:

- `api_key`: API key metadata
- `key`: Full API key value (only returned once)

### List API Keys

Retrieve all API keys for the authenticated user.

```typescript
const { api_keys, total } = await client.management.apiKeys.list();

api_keys.forEach((key) => {
  console.log(`${key.name}: ${key.key_prefix}...`);
  console.log(`  Scopes: ${key.scopes.join(", ")}`);
  console.log(`  Rate Limit: ${key.rate_limit_per_minute}/min`);
  console.log(`  Last Used: ${key.last_used_at || "Never"}`);

  if (key.revoked_at) {
    console.log(`  Status: REVOKED`);
  } else if (key.expires_at && new Date(key.expires_at) < new Date()) {
    console.log(`  Status: EXPIRED`);
  } else {
    console.log(`  Status: ACTIVE`);
  }
});
```

### Get API Key

Retrieve details for a specific API key.

```typescript
const apiKey = await client.management.apiKeys.get("key-uuid");

console.log("Name:", apiKey.name);
console.log("Scopes:", apiKey.scopes);
console.log("Created:", apiKey.created_at);
console.log("Last Used:", apiKey.last_used_at);
```

### Update API Key

Update API key properties (name, description, scopes, rate limit).

```typescript
const updated = await client.management.apiKeys.update("key-uuid", {
  name: "Updated Service Name",
  description: "New description",
  scopes: ["read:users", "write:users", "read:orders"],
  rate_limit_per_minute: 200,
});

console.log("Updated:", updated.name);
```

**Note:** Updating scopes immediately affects API key permissions. Update rate limits to handle increased/decreased traffic.

### Revoke API Key

Revoke an API key to prevent further use while keeping it for audit logs.

```typescript
await client.management.apiKeys.revoke("key-uuid");
console.log("API key revoked successfully");
```

**Difference between Revoke and Delete:**

- **Revoke**: Disables the key but keeps it in the database for audit trails
- **Delete**: Permanently removes the key from the system

### Delete API Key

Permanently delete an API key.

```typescript
await client.management.apiKeys.delete("key-uuid");
console.log("API key deleted successfully");
```

⚠️ **Warning:** This action cannot be undone. Consider revoking instead of deleting for audit purposes.

---

## Webhooks Management

Webhooks allow you to receive real-time notifications when events occur in your Fluxbase instance.

### Create Webhook

Set up a new webhook endpoint to receive event notifications.

```typescript
const webhook = await client.management.webhooks.create({
  url: "https://myapp.com/webhook",
  events: [
    "user.created",
    "user.updated",
    "user.deleted",
    "auth.login",
    "auth.logout",
  ],
  description: "User and auth events webhook",
  secret: "my-webhook-secret-key", // Used to sign webhook payloads
});

console.log("Webhook created:", webhook.id);
console.log("URL:", webhook.url);
console.log("Events:", webhook.events);
```

**Parameters:**

- `url` (required): HTTPS endpoint to receive webhook POST requests
- `events` (required): Array of event types to subscribe to
- `description` (optional): Human-readable description
- `secret` (optional): Secret key for HMAC signature verification

**Common Event Types:**

- `user.created` - New user registered
- `user.updated` - User profile updated
- `user.deleted` - User account deleted
- `auth.login` - User logged in
- `auth.logout` - User logged out
- `password.reset` - Password reset initiated
- `email.verified` - Email address verified

### List Webhooks

Retrieve all webhooks for the authenticated user.

```typescript
const { webhooks, total } = await client.management.webhooks.list();

webhooks.forEach((webhook) => {
  console.log(`${webhook.url} (${webhook.is_active ? "active" : "inactive"})`);
  console.log(`  Events: ${webhook.events.join(", ")}`);
  console.log(`  Created: ${webhook.created_at}`);
});
```

### Get Webhook

Retrieve details for a specific webhook.

```typescript
const webhook = await client.management.webhooks.get("webhook-uuid");

console.log("URL:", webhook.url);
console.log("Events:", webhook.events);
console.log("Active:", webhook.is_active);
console.log("Description:", webhook.description);
```

### Update Webhook

Modify webhook configuration.

```typescript
const updated = await client.management.webhooks.update("webhook-uuid", {
  url: "https://myapp.com/new-webhook-endpoint",
  events: ["user.created", "user.deleted"], // Changed event list
  is_active: false, // Temporarily disable
});

console.log("Webhook updated");
```

**Common Use Cases:**

- Update the webhook URL when your endpoint changes
- Add/remove event subscriptions
- Temporarily disable webhooks during maintenance

### Delete Webhook

Permanently remove a webhook.

```typescript
await client.management.webhooks.delete("webhook-uuid");
console.log("Webhook deleted");
```

### Test Webhook

Send a test event to verify your webhook endpoint is working correctly.

```typescript
const result = await client.management.webhooks.test("webhook-uuid");

if (result.success) {
  console.log("✅ Webhook test successful");
  console.log("Status Code:", result.status_code);
  console.log("Response:", result.response_body);
} else {
  console.error("❌ Webhook test failed");
  console.error("Error:", result.error);
}
```

**Test Payload Structure:**

```json
{
  "event": "webhook.test",
  "timestamp": "2024-01-26T10:00:00Z",
  "data": {
    "test": true,
    "webhook_id": "webhook-uuid"
  }
}
```

### List Webhook Deliveries

View the delivery history for a webhook, including successes and failures.

```typescript
const { deliveries } = await client.management.webhooks.listDeliveries(
  "webhook-uuid",
  100,
);

deliveries.forEach((delivery) => {
  console.log(`Event: ${delivery.event}`);
  console.log(`Status: ${delivery.status_code}`);
  console.log(`Created: ${delivery.created_at}`);
  console.log(`Delivered: ${delivery.delivered_at || "Pending"}`);

  if (delivery.error) {
    console.error(`Error: ${delivery.error}`);
  }

  console.log("---");
});
```

**Parameters:**

- `webhookId`: Webhook UUID
- `limit`: Maximum deliveries to return (default: 50, max: 1000)

### Webhook Payload Format

When an event occurs, Fluxbase sends a POST request to your webhook URL with the following structure:

```typescript
interface WebhookPayload {
  event: string; // Event type (e.g., "user.created")
  timestamp: string; // ISO 8601 timestamp
  data: {
    // Event-specific data
    user_id?: string;
    email?: string;
    // ... more fields
  };
  signature: string; // HMAC-SHA256 signature (if secret provided)
}
```

### Verifying Webhook Signatures

If you provided a `secret` when creating the webhook, verify incoming webhook payloads:

```typescript
import crypto from "crypto";

function verifyWebhookSignature(
  payload: string,
  signature: string,
  secret: string,
): boolean {
  const expectedSignature = crypto
    .createHmac("sha256", secret)
    .update(payload)
    .digest("hex");

  return crypto.timingSafeEqual(
    Buffer.from(signature),
    Buffer.from(expectedSignature),
  );
}

// In your webhook endpoint:
app.post("/webhook", (req, res) => {
  const signature = req.headers["x-fluxbase-signature"];
  const payload = JSON.stringify(req.body);

  if (!verifyWebhookSignature(payload, signature, WEBHOOK_SECRET)) {
    return res.status(401).json({ error: "Invalid signature" });
  }

  // Process the webhook
  console.log("Event:", req.body.event);
  console.log("Data:", req.body.data);

  res.json({ received: true });
});
```

---

## Invitations Management

Invitations allow administrators to invite new users to join the dashboard. The invitation system supports role-based access and automatic email notifications.

### Create Invitation

Create a new invitation for a user (admin only).

```typescript
// Must be authenticated as admin
await client.admin.login({
  email: "admin@example.com",
  password: "admin-password",
});

const invitation = await client.management.invitations.create({
  email: "newuser@example.com",
  role: "dashboard_user", // or 'dashboard_admin'
  expiry_duration: 604800, // 7 days in seconds (default)
});

console.log("Invitation created");
console.log("Invite Link:", invitation.invite_link);
console.log("Email Sent:", invitation.email_sent);

// Share the invite link with the user
// They'll use this link to set up their account
```

**Parameters:**

- `email` (required): Email address to invite
- `role` (required): Either `'dashboard_user'` or `'dashboard_admin'`
- `expiry_duration` (optional): Duration in seconds (default: 604800 = 7 days)

**Roles:**

- `dashboard_user`: Can access the dashboard with limited permissions
- `dashboard_admin`: Full admin access to instance management

### List Invitations

Retrieve all invitations (admin only).

```typescript
// List only pending invitations
const { invitations } = await client.management.invitations.list({
  include_accepted: false,
  include_expired: false,
});

invitations.forEach((invite) => {
  console.log(`${invite.email} - ${invite.role}`);
  console.log(`  Expires: ${invite.expires_at}`);
  console.log(`  Status: ${invite.accepted_at ? "Accepted" : "Pending"}`);
});

// List all invitations including accepted and expired
const all = await client.management.invitations.list({
  include_accepted: true,
  include_expired: true,
});

console.log(`Total invitations: ${all.invitations.length}`);
```

**Filter Options:**

- `include_accepted`: Include invitations that have been accepted (default: false)
- `include_expired`: Include expired invitations (default: false)

### Validate Invitation

Check if an invitation token is valid (public endpoint - no authentication required).

```typescript
const result = await client.management.invitations.validate("invitation-token");

if (result.valid) {
  console.log("✅ Valid invitation");
  console.log("Email:", result.invitation?.email);
  console.log("Role:", result.invitation?.role);
  console.log("Expires:", result.invitation?.expires_at);
} else {
  console.error("❌ Invalid invitation:", result.error);
  // Possible errors:
  // - "Invitation has expired"
  // - "Invitation has already been accepted"
  // - "Invitation not found"
}
```

### Accept Invitation

Accept an invitation and create a new user account (public endpoint).

```typescript
const response = await client.management.invitations.accept(
  "invitation-token",
  {
    password: "SecurePassword123!",
    name: "John Doe",
  },
);

console.log("✅ Account created successfully");
console.log("User:", response.user.email);
console.log("Name:", response.user.name);
console.log("Role:", response.user.role);

// Store authentication tokens
localStorage.setItem("access_token", response.access_token);
localStorage.setItem("refresh_token", response.refresh_token);

// User is now logged in and can access the dashboard
```

**Parameters:**

- `token`: Invitation token from the invite link
- `password`: New user's password (minimum 12 characters)
- `name`: User's display name (minimum 2 characters)

**Returns:** Authentication response with:

- `user`: Created user details
- `access_token`: JWT access token
- `refresh_token`: JWT refresh token
- `expires_in`: Token expiration time in seconds

### Revoke Invitation

Cancel an invitation (admin only).

```typescript
await client.management.invitations.revoke("invitation-token");
console.log("Invitation revoked");
```

**Use Cases:**

- User no longer needs access
- Invitation was sent to wrong email
- Security concerns with pending invitations

---

## Complete Examples

### API Key Management Dashboard

```typescript
import { createClient } from "@fluxbase/sdk";

const client = createClient({ url: "http://localhost:8080" });

async function setupAPIKeyDashboard() {
  // Authenticate
  await client.auth.login({
    email: "user@example.com",
    password: "password",
  });

  // List existing keys
  const { api_keys } = await client.management.apiKeys.list();

  console.log("=== API Keys ===");
  api_keys.forEach((key) => {
    const status = key.revoked_at
      ? "🔴 REVOKED"
      : key.expires_at && new Date(key.expires_at) < new Date()
        ? "🟡 EXPIRED"
        : "🟢 ACTIVE";

    console.log(`${status} ${key.name}`);
    console.log(`  Prefix: ${key.key_prefix}`);
    console.log(`  Rate Limit: ${key.rate_limit_per_minute}/min`);
    console.log(`  Last Used: ${key.last_used_at || "Never"}`);
  });

  // Create a new key
  const { api_key, key } = await client.management.apiKeys.create({
    name: "New Integration",
    scopes: ["read:users"],
    rate_limit_per_minute: 60,
  });

  console.log("\n✅ New API Key Created");
  console.log("Key:", key);
  console.log("Save this key securely - it won't be shown again!");

  // Update an existing key
  const updated = await client.management.apiKeys.update(api_keys[0].id, {
    rate_limit_per_minute: 120,
  });

  console.log(
    `\n✅ Updated ${updated.name} rate limit to ${updated.rate_limit_per_minute}/min`,
  );
}

setupAPIKeyDashboard().catch(console.error);
```

### Webhook Event Handler

```typescript
import express from "express";
import crypto from "crypto";
import { createClient } from "@fluxbase/sdk";

const app = express();
const client = createClient({ url: "http://localhost:8080" });
const WEBHOOK_SECRET = "your-webhook-secret";

// Verify webhook signature
function verifySignature(payload: string, signature: string): boolean {
  const expectedSignature = crypto
    .createHmac("sha256", WEBHOOK_SECRET)
    .update(payload)
    .digest("hex");

  return crypto.timingSafeEqual(
    Buffer.from(signature),
    Buffer.from(expectedSignature),
  );
}

// Webhook endpoint
app.post("/webhook", express.json(), (req, res) => {
  const signature = req.headers["x-fluxbase-signature"] as string;
  const payload = JSON.stringify(req.body);

  // Verify signature
  if (!verifySignature(payload, signature)) {
    return res.status(401).json({ error: "Invalid signature" });
  }

  // Process event
  const { event, timestamp, data } = req.body;

  console.log(`Received event: ${event} at ${timestamp}`);

  switch (event) {
    case "user.created":
      console.log("New user registered:", data.email);
      // Send welcome email, create customer profile, etc.
      break;

    case "user.deleted":
      console.log("User deleted:", data.user_id);
      // Clean up related data, cancel subscriptions, etc.
      break;

    case "auth.login":
      console.log("User logged in:", data.email);
      // Track login analytics, send notification, etc.
      break;

    default:
      console.log("Unknown event:", event);
  }

  res.json({ received: true });
});

// Set up webhook
async function setupWebhook() {
  await client.auth.login({
    email: "user@example.com",
    password: "password",
  });

  const webhook = await client.management.webhooks.create({
    url: "https://myapp.com/webhook",
    events: ["user.created", "user.deleted", "auth.login"],
    secret: WEBHOOK_SECRET,
  });

  console.log("Webhook created:", webhook.id);

  // Test the webhook
  const testResult = await client.management.webhooks.test(webhook.id);

  if (testResult.success) {
    console.log("✅ Webhook test successful");
  } else {
    console.error("❌ Webhook test failed:", testResult.error);
  }
}

app.listen(3000, () => {
  console.log("Server running on port 3000");
  setupWebhook().catch(console.error);
});
```

### Invitation Management System

```typescript
import { createClient } from "@fluxbase/sdk";

const client = createClient({ url: "http://localhost:8080" });

async function manageInvitations() {
  // Admin login
  await client.admin.login({
    email: "admin@example.com",
    password: "admin-password",
  });

  // Bulk invite multiple users
  const usersToInvite = [
    { email: "user1@example.com", role: "dashboard_user" as const },
    { email: "user2@example.com", role: "dashboard_user" as const },
    { email: "admin2@example.com", role: "dashboard_admin" as const },
  ];

  console.log("Creating invitations...");

  for (const user of usersToInvite) {
    const invitation = await client.management.invitations.create({
      email: user.email,
      role: user.role,
      expiry_duration: 604800, // 7 days
    });

    console.log(`✅ Invited ${user.email}`);
    console.log(`   Link: ${invitation.invite_link}`);

    // In production, send this link via email
  }

  // Check pending invitations
  const { invitations } = await client.management.invitations.list({
    include_accepted: false,
    include_expired: false,
  });

  console.log(`\n${invitations.length} pending invitations:`);
  invitations.forEach((invite) => {
    const expiresIn = Math.floor(
      (new Date(invite.expires_at).getTime() - Date.now()) /
        (1000 * 60 * 60 * 24),
    );
    console.log(`- ${invite.email} (expires in ${expiresIn} days)`);
  });

  // Revoke old invitations
  const oldInvitations = invitations.filter((invite) => {
    const daysOld = Math.floor(
      (Date.now() - new Date(invite.created_at).getTime()) /
        (1000 * 60 * 60 * 24),
    );
    return daysOld > 5;
  });

  if (oldInvitations.length > 0) {
    console.log(`\nRevoking ${oldInvitations.length} old invitations...`);
    for (const invite of oldInvitations) {
      await client.management.invitations.revoke(invite.token!);
      console.log(`✅ Revoked invitation for ${invite.email}`);
    }
  }
}

manageInvitations().catch(console.error);
```

---

## Error Handling

All management methods may throw errors. Always wrap calls in try-catch blocks:

```typescript
try {
  const { api_key, key } = await client.management.apiKeys.create({
    name: "New Key",
    scopes: ["read:users"],
    rate_limit_per_minute: 100,
  });

  console.log("API key created:", key);
} catch (error) {
  if (error.message.includes("unauthorized")) {
    console.error("You must be logged in to create API keys");
  } else if (error.message.includes("rate_limit")) {
    console.error("Invalid rate limit value");
  } else {
    console.error("Failed to create API key:", error.message);
  }
}
```

**Common Error Scenarios:**

1. **Authentication Required**

   ```typescript
   // Error: User not authenticated
   // Solution: Log in before calling management methods
   await client.auth.login({ email, password });
   ```

2. **Admin Permission Required**

   ```typescript
   // Error: Admin role required
   // Solution: Use admin credentials for invitation management
   await client.admin.login({ email: adminEmail, password: adminPassword });
   ```

3. **Invalid Invitation Token**

   ```typescript
   const result = await client.management.invitations.validate("invalid-token");
   if (!result.valid) {
     console.error(result.error); // "Invitation not found"
   }
   ```

4. **Webhook Delivery Failure**
   ```typescript
   const result = await client.management.webhooks.test("webhook-id");
   if (!result.success) {
     console.error("Webhook failed:", result.error);
     // Common issues: invalid URL, timeout, SSL certificate errors
   }
   ```

---

## Best Practices

### API Keys

1. **Store keys securely**: Never commit API keys to version control
2. **Rotate regularly**: Create new keys and revoke old ones periodically
3. **Use specific scopes**: Grant minimal permissions needed
4. **Monitor usage**: Check `last_used_at` to identify unused keys
5. **Set expiration**: Use `expires_at` for temporary integrations

### Webhooks

1. **Validate signatures**: Always verify webhook signatures to prevent spoofing
2. **Handle retries**: Implement idempotent handlers (same event may be delivered multiple times)
3. **Respond quickly**: Return 200 OK within 5 seconds to avoid timeouts
4. **Process async**: Queue events for background processing
5. **Monitor deliveries**: Check delivery history for failures

### Invitations

1. **Use appropriate roles**: Grant minimal necessary permissions
2. **Set reasonable expiry**: Default 7 days is usually sufficient
3. **Track invitations**: Regularly clean up expired/unused invitations
4. **Secure invite links**: Treat invitation tokens as sensitive data
5. **Revoke when needed**: Cancel invitations if user no longer needs access

---

## TypeScript Types

All management types are fully typed for excellent IDE support:

```typescript
import type {
  // API Keys
  APIKey,
  CreateAPIKeyRequest,
  CreateAPIKeyResponse,

  // Webhooks
  Webhook,
  CreateWebhookRequest,
  WebhookDelivery,
  TestWebhookResponse,

  // Invitations
  Invitation,
  CreateInvitationRequest,
  ValidateInvitationResponse,
  AcceptInvitationResponse,
} from "@fluxbase/sdk";
```

---

## Next Steps

- [Admin SDK](./admin.md) - Instance administration
- [OAuth](./oauth.md) - OAuth authentication
- [Advanced Features](./advanced-features.md) - Advanced SDK features
