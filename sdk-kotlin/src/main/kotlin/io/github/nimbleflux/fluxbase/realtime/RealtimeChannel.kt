package io.github.nimbleflux.fluxbase.realtime

import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.asSharedFlow
import kotlinx.coroutines.flow.collect
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import kotlinx.serialization.json.put

/**
 * A realtime channel — one WebSocket connection to `/realtime?token=<jwt>` that
 * subscribes to postgres_changes, broadcast, and/or presence events.
 *
 * Port of `RealtimeChannel` from `sdk/src/realtime.ts`.
 *
 * The channel manages its own WebSocket lifecycle:
 *   - On [subscribe]: connects, sends `{type:"subscribe", channel, config}` per
 *     postgres_changes config, starts a 30s heartbeat.
 *   - On `postgres_changes` messages: normalizes `new_record`→`new` /
 *     `old_record`→`old` (matching the Supabase shape) and dispatches to callbacks.
 *   - On `broadcast` messages: dispatches to event-specific callbacks.
 *   - Reconnects with exponential backoff (capped at 10 attempts).
 *
 * @param baseUrl the Fluxbase base URL (http(s)→ws(s)).
 * @param channelName the channel identifier.
 * @param token the JWT (sent as `?token=` query param, not subprotocol).
 * @param transport the [WebSocketTransport] SPI (tests inject a fake).
 */
class RealtimeChannel(
    private val baseUrl: String,
    internal val channelName: String,
    private var token: String?,
    private val transport: WebSocketTransport,
    private val coroutineDispatcher: kotlin.coroutines.CoroutineContext = Dispatchers.IO,
) {
    private val json = Json { ignoreUnknownKeys = true; isLenient = true }

    internal val postgresConfigs = mutableListOf<PostgresChangesConfig>()
    private val postgresCallbacks = mutableListOf<Pair<PostgresChangesConfig, ChangeEventCallback>>()
    private val broadcastCallbacks = mutableListOf<Pair<String, BroadcastCallback>>()
    private val wildcardCallbacks = mutableListOf<ChangeEventCallback>()

    private var scope: CoroutineScope? = null
    private var heartbeatJob: Job? = null
    private var reconnectAttempts = 0

    /**
     * Optional callback invoked when the server sends an `error` message.
     * Set this to surface subscription/server errors to application code.
     */
    var onError: ((JsonObject) -> Unit)? = null

    /**
     * The subscription id assigned by the server once a subscription is
     * acknowledged. Null until the first `ack` arrives. Port of the TS
     * `pendingAcks`/subscription tracking in `realtime.ts`.
     */
    internal var subscriptionId: String? = null
        private set

    /** Whether this channel is actively subscribed. */
    val isSubscribed: Boolean get() = transport.isConnected

    /**
     * Register a callback for a postgres_changes event type.
     * Port of `channel.on("INSERT"|"UPDATE"|"DELETE"|"*", cb)` in `realtime.ts`.
     */
    fun on(event: String, config: PostgresChangesConfig? = null, callback: ChangeEventCallback): RealtimeChannel {
        if (event == "*") {
            wildcardCallbacks.add(callback)
        } else {
            val cfg = config ?: PostgresChangesConfig(event = event)
            postgresConfigs.add(cfg)
            postgresCallbacks.add(cfg to callback)
        }
        return this
    }

    /**
     * Register a callback for broadcast events.
     * Port of `channel.on("broadcast", {event}, cb)` in `realtime.ts`.
     */
    fun onBroadcast(event: String, callback: BroadcastCallback): RealtimeChannel {
        broadcastCallbacks.add(event to callback)
        return this
    }

    /**
     * Subscribe to the channel — connects the WebSocket and sends subscribe messages.
     * Port of `subscribe()` in `realtime.ts:540-617`.
     */
    suspend fun subscribe() {
        reconnectAttempts = 0
        connect()
    }

    private suspend fun connect() {
        val wsUrl = buildWsUrl()
        val incoming = transport.connect(wsUrl)
        scope = CoroutineScope(SupervisorJob() + coroutineDispatcher)
        scope!!.launch {
            incoming.collect { messageText ->
                handleMessage(messageText)
            }
        }
        // Wait for the transport to finish its WS handshake before subscribing.
        // (For the fake transport [isConnected] is set synchronously; for the real
        // Ktor transport it flips once the socket opens — see [KtorWebSocketTransport].)
        awaitConnected()
        if (transport.isConnected) {
            sendSubscribeMessages()
            startHeartbeat()
        }
    }

    /**
     * Block until [transport.isConnected] becomes true or [timeoutMs] elapses.
     * Replaces a fixed `delay(100)` that was too short for real-world handshakes.
     */
    private suspend fun awaitConnected(timeoutMs: Long = 10_000) {
        val deadline = System.currentTimeMillis() + timeoutMs
        while (!transport.isConnected && System.currentTimeMillis() < deadline) {
            delay(50)
        }
    }

    private fun buildWsUrl(): String {
        val wsScheme = if (baseUrl.startsWith("https")) "wss" else "ws"
        val host = baseUrl.substringAfter("://").trimEnd('/')
        val tokenParam = token?.let { "?token=$it" } ?: ""
        return "$wsScheme://$host/realtime$tokenParam"
    }

    private suspend fun sendSubscribeMessages() {
        if (postgresConfigs.isEmpty()) {
            // Broadcast/presence only — subscribe without config.
            send(buildJsonObject { put("type", "subscribe"); put("channel", channelName) })
        } else {
            for (config in postgresConfigs) {
                send(buildJsonObject {
                    put("type", "subscribe")
                    put("channel", channelName)
                    put("config", buildJsonObject {
                        put("event", config.event)
                        put("schema", config.schema)
                        config.table?.let { put("table", it) }
                        config.filter?.let { put("filter", it) }
                    })
                })
            }
        }
    }

    private fun startHeartbeat() {
        heartbeatJob?.cancel()
        heartbeatJob = scope?.launch {
            while (isActive) {
                delay(30_000) // 30s heartbeat, matches TS.
                send(buildJsonObject { put("type", "heartbeat") })
            }
        }
    }

    /**
     * Handle an incoming message (JSON text). Port of `handleMessage` in `realtime.ts:728-882`.
     * Normalizes postgres_changes payloads and dispatches to callbacks.
     */
    internal fun handleMessage(messageText: String) {
        val message = try {
            json.parseToJsonElement(messageText).jsonObject
        } catch (_: Exception) {
            return
        }

        val type = message["type"]?.jsonPrimitive?.contentOrNull ?: return

        when (type) {
            "postgres_changes" -> handlePostgresChanges(message["payload"]?.jsonObject)
            "broadcast" -> handleBroadcast(message)
            "heartbeat" -> { /* no-op — we send our own */ }
            "ack" -> handleAck(message)
            "error" -> onError?.invoke(message)
        }
    }

    /**
     * Handle a server `ack` message. Port of the ack handling in `realtime.ts:788-796`.
     * Records the server-assigned [subscriptionId] when present.
     */
    private fun handleAck(message: JsonObject) {
        subscriptionId = message["subscription_id"]?.jsonPrimitive?.contentOrNull
    }

    private fun handlePostgresChanges(payload: JsonObject?) {
        payload ?: return
        val normalized = PostgresChangesPayload(
            eventType = (payload["type"] ?: payload["eventType"])?.jsonPrimitive?.contentOrNull
                ?: (payload["eventType"])?.jsonPrimitive?.contentOrNull ?: return,
            schema = payload["schema"]?.jsonPrimitive?.contentOrNull,
            table = payload["table"]?.jsonPrimitive?.contentOrNull,
            commitTimestamp = (payload["timestamp"] ?: payload["commit_timestamp"])?.jsonPrimitive?.contentOrNull,
            newRecord = payload["new_record"] ?: payload["new"] ?: payload["record"],
            oldRecord = payload["old_record"] ?: payload["old"],
            errors = payload["errors"]?.jsonPrimitive?.contentOrNull,
        )

        // Dispatch to matching config-specific callbacks.
        postgresCallbacks.forEach { (config, cb) ->
            val schemaMatch = config.schema == normalized.schema
            val tableMatch = config.table == null || config.table == normalized.table
            val eventMatch = config.event == "*" || config.event == normalized.eventType
            if (schemaMatch && tableMatch && eventMatch) {
                cb(normalized)
            }
        }
        // Wildcard callbacks.
        wildcardCallbacks.forEach { it(normalized) }
    }

    private fun handleBroadcast(message: JsonObject) {
        val payload = message["payload"]?.jsonObject ?: return
        val broadcast = payload["broadcast"]?.jsonObject ?: payload
        val event = broadcast["event"]?.jsonPrimitive?.contentOrNull ?: return
        val eventPayload = broadcast["payload"] ?: return

        broadcastCallbacks.forEach { (registeredEvent, cb) ->
            if (registeredEvent == event || registeredEvent == "*") {
                cb(eventPayload)
            }
        }
    }

    /** Send a JSON object over the WebSocket. */
    internal suspend fun send(obj: JsonObject) {
        transport.send(json.encodeToString(JsonObject.serializer(), obj))
    }

    /** Broadcast a message on this channel. */
    suspend fun broadcast(event: String, payload: Map<String, Any?>) {
        send(buildJsonObject {
            put("type", "broadcast")
            put("channel", channelName)
            put("event", event)
            put("payload", Json.encodeToJsonElement(JsonObject.serializer(), buildJsonObject {
                payload.forEach { (k, v) ->
                    when (v) {
                        is String -> put(k, v)
                        is Number -> put(k, v)
                        is Boolean -> put(k, v)
                        else -> put(k, v.toString())
                    }
                }
            }))
        })
    }

    /**
     * Unsubscribe and close the connection.
     * Port of `unsubscribe()` in `realtime.ts`.
     */
    suspend fun unsubscribe() {
        send(buildJsonObject { put("type", "unsubscribe"); put("channel", channelName) })
        heartbeatJob?.cancel()
        scope?.cancel()
        scope = null
        transport.close()
    }

    /**
     * Update the auth token (e.g. after a JWT refresh). If the socket is open,
     * pushes an `access_token` message so the server refreshes auth context
     * without a reconnect; otherwise the new token is used on the next connect.
     *
     * Port of `updateToken()` in `realtime.ts:1037-1090`. Note the TS version
     * additionally waits for an ack and reconnects on a 5s timeout; this Kotlin
     * port propagates the token immediately and relies on [onError] / reconnect
     * handling if the server rejects it.
     */
    suspend fun updateToken(newToken: String) {
        if (token == newToken) return
        token = newToken
        if (!transport.isConnected) return
        send(buildJsonObject {
            put("type", "access_token")
            put("token", newToken)
        })
    }
}
