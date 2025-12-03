---
editUrl: false
next: false
prev: false
title: "WebhooksManager"
---

Webhooks management client

Provides methods for managing webhooks to receive real-time event notifications.
Webhooks allow your application to be notified when events occur in Fluxbase.

## Example

```typescript
const client = createClient({ url: 'http://localhost:8080' })
await client.auth.login({ email: 'user@example.com', password: 'password' })

// Create a webhook
const webhook = await client.management.webhooks.create({
  url: 'https://myapp.com/webhook',
  events: ['user.created', 'user.updated'],
  secret: 'my-webhook-secret'
})

// Test the webhook
const result = await client.management.webhooks.test(webhook.id)
```

## Constructors

### new WebhooksManager()

> **new WebhooksManager**(`fetch`): [`WebhooksManager`](/api/sdk/classes/webhooksmanager/)

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `fetch` | [`FluxbaseFetch`](/api/sdk/classes/fluxbasefetch/) |

#### Returns

[`WebhooksManager`](/api/sdk/classes/webhooksmanager/)

## Methods

### create()

> **create**(`request`): `Promise`\<[`Webhook`](/api/sdk/interfaces/webhook/)\>

Create a new webhook

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `request` | [`CreateWebhookRequest`](/api/sdk/interfaces/createwebhookrequest/) | Webhook configuration |

#### Returns

`Promise`\<[`Webhook`](/api/sdk/interfaces/webhook/)\>

Created webhook

#### Example

```typescript
const webhook = await client.management.webhooks.create({
  url: 'https://myapp.com/webhook',
  events: ['user.created', 'user.updated', 'user.deleted'],
  description: 'User events webhook',
  secret: 'my-webhook-secret'
})
```

***

### delete()

> **delete**(`webhookId`): `Promise`\<[`DeleteWebhookResponse`](/api/sdk/interfaces/deletewebhookresponse/)\>

Delete a webhook

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `webhookId` | `string` | Webhook ID |

#### Returns

`Promise`\<[`DeleteWebhookResponse`](/api/sdk/interfaces/deletewebhookresponse/)\>

Deletion confirmation

#### Example

```typescript
await client.management.webhooks.delete('webhook-uuid')
console.log('Webhook deleted')
```

***

### get()

> **get**(`webhookId`): `Promise`\<[`Webhook`](/api/sdk/interfaces/webhook/)\>

Get a specific webhook by ID

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `webhookId` | `string` | Webhook ID |

#### Returns

`Promise`\<[`Webhook`](/api/sdk/interfaces/webhook/)\>

Webhook details

#### Example

```typescript
const webhook = await client.management.webhooks.get('webhook-uuid')
console.log('Events:', webhook.events)
```

***

### list()

> **list**(): `Promise`\<[`ListWebhooksResponse`](/api/sdk/interfaces/listwebhooksresponse/)\>

List all webhooks for the authenticated user

#### Returns

`Promise`\<[`ListWebhooksResponse`](/api/sdk/interfaces/listwebhooksresponse/)\>

List of webhooks

#### Example

```typescript
const { webhooks, total } = await client.management.webhooks.list()

webhooks.forEach(webhook => {
  console.log(`${webhook.url}: ${webhook.is_active ? 'active' : 'inactive'}`)
})
```

***

### listDeliveries()

> **listDeliveries**(`webhookId`, `limit`): `Promise`\<[`ListWebhookDeliveriesResponse`](/api/sdk/interfaces/listwebhookdeliveriesresponse/)\>

List webhook delivery history

#### Parameters

| Parameter | Type | Default value | Description |
| ------ | ------ | ------ | ------ |
| `webhookId` | `string` | `undefined` | Webhook ID |
| `limit` | `number` | `50` | Maximum number of deliveries to return (default: 50) |

#### Returns

`Promise`\<[`ListWebhookDeliveriesResponse`](/api/sdk/interfaces/listwebhookdeliveriesresponse/)\>

List of webhook deliveries

#### Example

```typescript
const { deliveries } = await client.management.webhooks.listDeliveries('webhook-uuid', 100)

deliveries.forEach(delivery => {
  console.log(`Event: ${delivery.event}, Status: ${delivery.status_code}`)
})
```

***

### test()

> **test**(`webhookId`): `Promise`\<[`TestWebhookResponse`](/api/sdk/interfaces/testwebhookresponse/)\>

Test a webhook by sending a test payload

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `webhookId` | `string` | Webhook ID |

#### Returns

`Promise`\<[`TestWebhookResponse`](/api/sdk/interfaces/testwebhookresponse/)\>

Test result with status and response

#### Example

```typescript
const result = await client.management.webhooks.test('webhook-uuid')

if (result.success) {
  console.log('Webhook test successful')
} else {
  console.error('Webhook test failed:', result.error)
}
```

***

### update()

> **update**(`webhookId`, `updates`): `Promise`\<[`Webhook`](/api/sdk/interfaces/webhook/)\>

Update a webhook

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `webhookId` | `string` | Webhook ID |
| `updates` | [`UpdateWebhookRequest`](/api/sdk/interfaces/updatewebhookrequest/) | Fields to update |

#### Returns

`Promise`\<[`Webhook`](/api/sdk/interfaces/webhook/)\>

Updated webhook

#### Example

```typescript
const updated = await client.management.webhooks.update('webhook-uuid', {
  events: ['user.created', 'user.deleted'],
  is_active: false
})
```
