---
editUrl: false
next: false
prev: false
title: "FluxbaseRealtime"
---

## Constructors

### new FluxbaseRealtime()

> **new FluxbaseRealtime**(`url`, `token`): [`FluxbaseRealtime`](/api/sdk/classes/fluxbaserealtime/)

#### Parameters

| Parameter | Type | Default value |
| ------ | ------ | ------ |
| `url` | `string` | `undefined` |
| `token` | `null` \| `string` | `null` |

#### Returns

[`FluxbaseRealtime`](/api/sdk/classes/fluxbaserealtime/)

## Methods

### channel()

> **channel**(`channelName`, `config`?): [`RealtimeChannel`](/api/sdk/classes/realtimechannel/)

Create or get a channel with optional configuration

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `channelName` | `string` | Channel name (e.g., 'table:public.products') |
| `config`? | [`RealtimeChannelConfig`](/api/sdk/interfaces/realtimechannelconfig/) | Optional channel configuration |

#### Returns

[`RealtimeChannel`](/api/sdk/classes/realtimechannel/)

RealtimeChannel instance

#### Example

```typescript
const channel = realtime.channel('room-1', {
  broadcast: { self: true, ack: true },
  presence: { key: 'user-123' }
})
```

***

### executionLogs()

> **executionLogs**(`executionId`, `type`): [`ExecutionLogsChannel`](/api/sdk/classes/executionlogschannel/)

Create an execution log subscription channel

This provides a cleaner API for subscribing to execution logs
(functions, jobs, or RPC procedures).

#### Parameters

| Parameter | Type | Default value | Description |
| ------ | ------ | ------ | ------ |
| `executionId` | `string` | `undefined` | The execution ID to subscribe to |
| `type` | [`ExecutionType`](/api/sdk/type-aliases/executiontype/) | `"function"` | The type of execution ('function', 'job', 'rpc') |

#### Returns

[`ExecutionLogsChannel`](/api/sdk/classes/executionlogschannel/)

ExecutionLogsChannel instance with fluent API

#### Example

```typescript
const channel = client.realtime.executionLogs('exec-123', 'function')
  .onLog((log) => {
    console.log(`[${log.level}] ${log.message}`)
  })
  .subscribe()
```

***

### removeAllChannels()

> **removeAllChannels**(): `void`

Remove all channels

#### Returns

`void`

***

### removeChannel()

> **removeChannel**(`channel`): `Promise`\<`"error"` \| `"ok"`\>

Remove a specific channel

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `channel` | [`RealtimeChannel`](/api/sdk/classes/realtimechannel/) | The channel to remove |

#### Returns

`Promise`\<`"error"` \| `"ok"`\>

Promise resolving to status

#### Example

```typescript
const channel = realtime.channel('room-1')
await realtime.removeChannel(channel)
```

***

### setAuth()

> **setAuth**(`token`): `void`

Update auth token for all channels
Updates both the stored token for new channels and propagates
the token to all existing connected channels.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `token` | `null` \| `string` | The new auth token |

#### Returns

`void`
