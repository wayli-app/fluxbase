---
title: "RealtimeChannel"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../)/[io.github.nimbleflux.fluxbase.realtime](../)/[RealtimeChannel](./)

# RealtimeChannel

class [RealtimeChannel](./)(baseUrl: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), channelName: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), token: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)?, transport: [WebSocketTransport](../-web-socket-transport/), coroutineDispatcher: [CoroutineContext](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.coroutines/-coroutine-context/index.html) = Dispatchers.IO)

A realtime channel — one WebSocket connection to `/realtime?token=<jwt>` that subscribes to postgres_changes, broadcast, and/or presence events.

Port of `RealtimeChannel` from `sdk/src/realtime.ts`.

The channel manages its own WebSocket lifecycle:

- 
   On [subscribe](subscribe/): connects, sends `{type:"subscribe", channel, config}` per     postgres_changes config, starts a 30s heartbeat.
- 
   On `postgres_changes` messages: normalizes `new_record`→`new` /     `old_record`→`old` (matching the Supabase shape) and dispatches to callbacks.
- 
   On `broadcast` messages: dispatches to event-specific callbacks.
- 
   Reconnects with exponential backoff (capped at 10 attempts).

#### Parameters

jvm

| | |
|---|---|
| baseUrl | the Fluxbase base URL (http(s)→ws(s)). |
| channelName | the channel identifier. |
| token | the JWT (sent as `?token=` query param, not subprotocol). |
| transport | the [WebSocketTransport](../-web-socket-transport/) SPI (tests inject a fake). |

## Constructors

| | |
|---|---|
| [RealtimeChannel](-realtime-channel/) | [jvm]<br>constructor(baseUrl: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), channelName: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), token: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)?, transport: [WebSocketTransport](../-web-socket-transport/), coroutineDispatcher: [CoroutineContext](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.coroutines/-coroutine-context/index.html) = Dispatchers.IO) |

## Properties

| Name | Summary |
|---|---|
| [isSubscribed](is-subscribed/) | [jvm]<br>val [isSubscribed](is-subscribed/): [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html)<br>Whether this channel is actively subscribed. |
| [onError](on-error/) | [jvm]<br>var [onError](on-error/): (JsonObject) -&gt; [Unit](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-unit/index.html)?<br>Optional callback invoked when the server sends an `error` message. Set this to surface subscription/server errors to application code. |

## Functions

| Name | Summary |
|---|---|
| [broadcast](broadcast/) | [jvm]<br>suspend fun [broadcast](broadcast/)(event: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), payload: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?&gt;)<br>Broadcast a message on this channel. |
| [on](on/) | [jvm]<br>fun [on](on/)(event: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), config: [PostgresChangesConfig](../-postgres-changes-config/)? = null, callback: [ChangeEventCallback](../-change-event-callback/)): [RealtimeChannel](./)<br>Register a callback for a postgres_changes event type. Port of `channel.on("INSERT"|"UPDATE"|"DELETE"|"*", cb)` in `realtime.ts`. |
| [onBroadcast](on-broadcast/) | [jvm]<br>fun [onBroadcast](on-broadcast/)(event: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), callback: [BroadcastCallback](../-broadcast-callback/)): [RealtimeChannel](./)<br>Register a callback for broadcast events. Port of `channel.on("broadcast", {event}, cb)` in `realtime.ts`. |
| [subscribe](subscribe/) | [jvm]<br>suspend fun [subscribe](subscribe/)()<br>Subscribe to the channel — connects the WebSocket and sends subscribe messages. Port of `subscribe()` in `realtime.ts:540-617`. |
| [unsubscribe](unsubscribe/) | [jvm]<br>suspend fun [unsubscribe](unsubscribe/)()<br>Unsubscribe and close the connection. Port of `unsubscribe()` in `realtime.ts`. |
| [updateToken](update-token/) | [jvm]<br>suspend fun [updateToken](update-token/)(newToken: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html))<br>Update the auth token (e.g. after a JWT refresh). If the socket is open, pushes an `access_token` message so the server refreshes auth context without a reconnect; otherwise the new token is used on the next connect. |
