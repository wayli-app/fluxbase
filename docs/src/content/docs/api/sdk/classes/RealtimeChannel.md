---
editUrl: false
next: false
prev: false
title: "RealtimeChannel"
---

## Constructors

### new RealtimeChannel()

> **new RealtimeChannel**(`url`, `channelName`, `token`, `config`): [`RealtimeChannel`](/api/sdk/classes/realtimechannel/)

#### Parameters

| Parameter | Type | Default value |
| ------ | ------ | ------ |
| `url` | `string` | `undefined` |
| `channelName` | `string` | `undefined` |
| `token` | `null` \| `string` | `null` |
| `config` | [`RealtimeChannelConfig`](/api/sdk/interfaces/realtimechannelconfig/) | `{}` |

#### Returns

[`RealtimeChannel`](/api/sdk/classes/realtimechannel/)

## Methods

### off()

> **off**(`event`, `callback`): `this`

Remove a callback

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `event` | `"DELETE"` \| `"INSERT"` \| `"UPDATE"` \| `"*"` |
| `callback` | [`RealtimeCallback`](/api/sdk/type-aliases/realtimecallback/) |

#### Returns

`this`

***

### on()

#### on(event, config, callback)

> **on**(`event`, `config`, `callback`): `this`

Listen to postgres_changes with optional row-level filtering

##### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `event` | `"postgres_changes"` | 'postgres_changes' |
| `config` | [`PostgresChangesConfig`](/api/sdk/interfaces/postgreschangesconfig/) | Configuration including optional filter |
| `callback` | [`RealtimeCallback`](/api/sdk/type-aliases/realtimecallback/) | Function to call when changes occur |

##### Returns

`this`

This channel for chaining

##### Example

```typescript
channel.on('postgres_changes', {
  event: '*',
  schema: 'public',
  table: 'jobs',
  filter: 'created_by=eq.user123'
}, (payload) => {
  console.log('Job updated:', payload)
})
```

#### on(event, callback)

> **on**(`event`, `callback`): `this`

Listen to a specific event type (backwards compatibility)

##### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `event` | `"DELETE"` \| `"INSERT"` \| `"UPDATE"` \| `"*"` | The event type (INSERT, UPDATE, DELETE, or '*' for all) |
| `callback` | [`RealtimeCallback`](/api/sdk/type-aliases/realtimecallback/) | The callback function |

##### Returns

`this`

This channel for chaining

##### Example

```typescript
channel.on('INSERT', (payload) => {
  console.log('New record inserted:', payload.new_record)
})
```

#### on(event, config, callback)

> **on**(`event`, `config`, `callback`): `this`

Listen to broadcast messages

##### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `event` | `"broadcast"` | 'broadcast' |
| `config` | `object` | Configuration with event name |
| `config.event` | `string` | - |
| `callback` | [`BroadcastCallback`](/api/sdk/type-aliases/broadcastcallback/) | Function to call when broadcast received |

##### Returns

`this`

This channel for chaining

##### Example

```typescript
channel.on('broadcast', { event: 'cursor-pos' }, (payload) => {
  console.log('Cursor moved:', payload)
})
```

#### on(event, config, callback)

> **on**(`event`, `config`, `callback`): `this`

Listen to presence events

##### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `event` | `"presence"` | 'presence' |
| `config` | `object` | Configuration with event type (sync, join, leave) |
| `config.event` | `"sync"` \| `"join"` \| `"leave"` | - |
| `callback` | [`PresenceCallback`](/api/sdk/type-aliases/presencecallback/) | Function to call when presence changes |

##### Returns

`this`

This channel for chaining

##### Example

```typescript
channel.on('presence', { event: 'sync' }, (payload) => {
  console.log('Presence synced:', payload)
})
```

***

### presenceState()

> **presenceState**(): `Record`\<`string`, [`PresenceState`](/api/sdk/interfaces/presencestate/)[]\>

Get current presence state for all users on this channel

#### Returns

`Record`\<`string`, [`PresenceState`](/api/sdk/interfaces/presencestate/)[]\>

Current presence state

#### Example

```typescript
const state = channel.presenceState()
console.log('Online users:', Object.keys(state).length)
```

***

### send()

> **send**(`message`): `Promise`\<`"error"` \| `"ok"`\>

Send a broadcast message to all subscribers on this channel

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `message` | [`BroadcastMessage`](/api/sdk/interfaces/broadcastmessage/) | Broadcast message with type, event, and payload |

#### Returns

`Promise`\<`"error"` \| `"ok"`\>

Promise resolving to status

#### Example

```typescript
await channel.send({
  type: 'broadcast',
  event: 'cursor-pos',
  payload: { x: 100, y: 200 }
})
```

***

### subscribe()

> **subscribe**(`callback`?, `_timeout`?): `this`

Subscribe to the channel

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `callback`? | (`status`, `err`?) => `void` | Optional status callback (Supabase-compatible) |
| `_timeout`? | `number` | Optional timeout in milliseconds (currently unused) |

#### Returns

`this`

***

### track()

> **track**(`state`): `Promise`\<`"error"` \| `"ok"`\>

Track user presence on this channel

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `state` | [`PresenceState`](/api/sdk/interfaces/presencestate/) | Presence state to track |

#### Returns

`Promise`\<`"error"` \| `"ok"`\>

Promise resolving to status

#### Example

```typescript
await channel.track({
  user_id: 123,
  status: 'online'
})
```

***

### unsubscribe()

> **unsubscribe**(`timeout`?): `Promise`\<`"error"` \| `"ok"` \| `"timed out"`\>

Unsubscribe from the channel

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `timeout`? | `number` | Optional timeout in milliseconds |

#### Returns

`Promise`\<`"error"` \| `"ok"` \| `"timed out"`\>

Promise resolving to status string (Supabase-compatible)

***

### untrack()

> **untrack**(): `Promise`\<`"error"` \| `"ok"`\>

Stop tracking presence on this channel

#### Returns

`Promise`\<`"error"` \| `"ok"`\>

Promise resolving to status

#### Example

```typescript
await channel.untrack()
```
