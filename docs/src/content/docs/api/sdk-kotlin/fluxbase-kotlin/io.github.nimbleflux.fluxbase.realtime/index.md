---
title: "Package-level declarations"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../)/[io.github.nimbleflux.fluxbase.realtime](./)

# Package-level declarations

## Types

| Name | Summary |
|---|---|
| [BroadcastCallback](-broadcast-callback/) | [jvm]<br>typealias [BroadcastCallback](-broadcast-callback/) = (JsonElement) -&gt; [Unit](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-unit/index.html)<br>Callback type for broadcast events. |
| [ChangeEventCallback](-change-event-callback/) | [jvm]<br>typealias [ChangeEventCallback](-change-event-callback/) = ([PostgresChangesPayload](-postgres-changes-payload/)) -&gt; [Unit](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-unit/index.html)<br>Callback type for postgres_changes events. |
| [ChangeEventType](-change-event-type/) | [jvm]<br>enum [ChangeEventType](-change-event-type/) : [Enum](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-enum/index.html)&lt;[ChangeEventType](-change-event-type/)&gt; <br>The type of a postgres_changes event. Port of the TS `eventType` values. |
| [KtorWebSocketTransport](-ktor-web-socket-transport/) | [jvm]<br>class [KtorWebSocketTransport](-ktor-web-socket-transport/) : [WebSocketTransport](-web-socket-transport/)<br>Ktor-backed [WebSocketTransport](-web-socket-transport/) — the production WebSocket implementation for JVM/Android. |
| [PostgresChangesConfig](-postgres-changes-config/) | [jvm]<br>data class [PostgresChangesConfig](-postgres-changes-config/)(val event: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;*&quot;, val schema: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;public&quot;, val table: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, val filter: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null)<br>Subscription config for a postgres_changes channel. Port of `PostgresChangesConfig` from `sdk/src/types.ts:397`. |
| [PostgresChangesPayload](-postgres-changes-payload/) | [jvm]<br>@Serializable<br>data class [PostgresChangesPayload](-postgres-changes-payload/)(val eventType: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), val schema: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, val table: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, val commitTimestamp: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, val newRecord: JsonElement? = null, val oldRecord: JsonElement? = null, val errors: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null)<br>A normalized postgres_changes payload — the Supabase-compatible shape the TS SDK produces by converting the server's `new_record`/`old_record` to `new`/`old`. Port of `RealtimePostgresChangesPayload` from `sdk/src/types.ts:408`. |
| [RealtimeChannel](-realtime-channel/) | [jvm]<br>class [RealtimeChannel](-realtime-channel/)(baseUrl: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), channelName: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), token: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)?, transport: [WebSocketTransport](-web-socket-transport/), coroutineDispatcher: [CoroutineContext](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.coroutines/-coroutine-context/index.html) = Dispatchers.IO)<br>A realtime channel — one WebSocket connection to `/realtime?token=<jwt>` that subscribes to postgres_changes, broadcast, and/or presence events. |
| [WebSocketTransport](-web-socket-transport/) | [jvm]<br>interface [WebSocketTransport](-web-socket-transport/)<br>SPI for WebSocket I/O — the seam where tests inject a fake transport (see FakeWebSocketTransport) instead of a real WS connection. |
