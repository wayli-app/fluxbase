---
editUrl: false
next: false
prev: false
title: "useWebhooks"
---

> **useWebhooks**(`options`): `UseWebhooksReturn`

Hook for managing webhooks

## Parameters

| Parameter | Type |
| ------ | ------ |
| `options` | `UseWebhooksOptions` |

## Returns

`UseWebhooksReturn`

## Example

```tsx
function WebhooksManager() {
  const { webhooks, isLoading, createWebhook, deleteWebhook } = useWebhooks({
    autoFetch: true
  })

  const handleCreate = async () => {
    await createWebhook({
      url: 'https://example.com/webhook',
      events: ['user.created', 'user.updated'],
      enabled: true
    })
  }

  return <div>...</div>
}
```
