---
editUrl: false
next: false
prev: false
title: "ExecutionLogsChannel"
---

Specialized channel for execution log subscriptions
Provides a cleaner API than the generic RealtimeChannel

## Constructors

### new ExecutionLogsChannel()

> **new ExecutionLogsChannel**(`url`, `executionId`, `type`, `token`, `tokenRefreshCallback`): [`ExecutionLogsChannel`](/api/sdk/classes/executionlogschannel/)

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `url` | `string` |
| `executionId` | `string` |
| `type` | [`ExecutionType`](/api/sdk/type-aliases/executiontype/) |
| `token` | `null` \| `string` |
| `tokenRefreshCallback` | `null` \| () => `Promise`\<`null` \| `string`\> |

#### Returns

[`ExecutionLogsChannel`](/api/sdk/classes/executionlogschannel/)

## Methods

### onLog()

> **onLog**(`callback`): `this`

Register a callback for log events

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `callback` | [`ExecutionLogCallback`](/api/sdk/type-aliases/executionlogcallback/) | Function to call when log entries are received |

#### Returns

`this`

This channel for chaining

#### Example

```typescript
channel.onLog((log) => {
  console.log(`[${log.level}] Line ${log.line_number}: ${log.message}`)
})
```

***

### subscribe()

> **subscribe**(`callback`?): `this`

Subscribe to execution logs

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `callback`? | (`status`, `err`?) => `void` | Optional status callback |

#### Returns

`this`

Promise that resolves when subscribed

#### Example

```typescript
await channel.subscribe()
```

***

### unsubscribe()

> **unsubscribe**(): `Promise`\<`"error"` \| `"ok"` \| `"timed out"`\>

Unsubscribe from execution logs

#### Returns

`Promise`\<`"error"` \| `"ok"` \| `"timed out"`\>

Promise resolving to status

#### Example

```typescript
await channel.unsubscribe()
```
