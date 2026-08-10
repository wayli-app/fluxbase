package io.github.nimbleflux.fluxbase.realtime

import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.flow

/**
 * Ktor-backed [WebSocketTransport] — the production WebSocket implementation
 * for JVM/Android.
 *
 * Uses Ktor's WebSocket client to connect to Fluxbase's `/realtime` endpoint.
 * The actual Ktor WS session is established on [connect] and messages are
 * received as a Flow of JSON text.
 *
 * TODO: implement the full Ktor WebSocket session (connect, send, receive,
 * close) once the integration test infrastructure is in place. For now this
 * is a placeholder that throws on connect — tests use [FakeWebSocketTransport].
 */
class KtorWebSocketTransport : WebSocketTransport {
    override var isConnected: Boolean = false
        private set

    override fun connect(url: String): Flow<String> = flow {
        throw NotImplementedError("KtorWebSocketTransport is not yet implemented. Use FakeWebSocketTransport in tests.")
    }

    override suspend fun send(text: String) {
        throw NotImplementedError("KtorWebSocketTransport is not yet implemented.")
    }

    override fun close() {
        isConnected = false
    }
}
