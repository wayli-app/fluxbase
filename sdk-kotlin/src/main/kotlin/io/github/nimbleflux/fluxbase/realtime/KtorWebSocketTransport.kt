package io.github.nimbleflux.fluxbase.realtime

import io.ktor.client.HttpClient
import io.ktor.client.engine.okhttp.OkHttp
import io.ktor.client.plugins.websocket.ClientWebSocketSession
import io.ktor.client.plugins.websocket.WebSockets
import io.ktor.client.plugins.websocket.webSocketSession
import io.ktor.client.request.url
import io.ktor.websocket.Frame
import io.ktor.websocket.readText
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.channels.Channel
import kotlinx.coroutines.channels.consumeEach
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.receiveAsFlow
import kotlinx.coroutines.launch

/**
 * Ktor-backed [WebSocketTransport] — the production WebSocket implementation
 * for JVM/Android.
 *
 * Connects to Fluxbase's `/realtime` endpoint via Ktor's WebSocket client
 * (OkHttp engine). The connection is established eagerly on an internal
 * [Dispatchers.IO] scope when [connect] is called; inbound text frames are
 * bridged onto the returned [Flow], which [RealtimeChannel] collects.
 * Outbound messages are sent via [send] once the session handshake completes
 * (observable via [isConnected]).
 *
 * The protocol itself (subscribe/heartbeat/postgres_changes/broadcast) lives
 * in [RealtimeChannel]; this class is purely the transport seam — the same SPI
 * tests fake with [FakeWebSocketTransport].
 */
class KtorWebSocketTransport(trustAllCertificates: Boolean = false) : WebSocketTransport {
    private val scope = CoroutineScope(Dispatchers.IO + SupervisorJob())
    private val client = HttpClient(OkHttp) {
        install(WebSockets)
        if (trustAllCertificates) {
            engine {
                config {
                    sslSocketFactory(
                        io.github.nimbleflux.fluxbase.core.TrustAllCertificates.socketFactory,
                        io.github.nimbleflux.fluxbase.core.TrustAllCertificates.trustManager,
                    )
                    hostnameVerifier(io.github.nimbleflux.fluxbase.core.TrustAllCertificates.hostnameVerifier)
                }
            }
        }
    }

    private var session: ClientWebSocketSession? = null
    private var connectionJob: Job? = null
    private var incomingChannel: Channel<String>? = null

    @Volatile
    override var isConnected: Boolean = false
        private set

    override fun connect(url: String): Flow<String> {
        // Reset any prior connection before opening a new one.
        teardown()

        val channel = Channel<String>(Channel.BUFFERED)
        incomingChannel = channel
        val target = url

        connectionJob = scope.launch {
            try {
                val s = client.webSocketSession { url(target) }
                session = s
                isConnected = true
                try {
                    // Drain server frames until the session closes or is cancelled.
                    s.incoming.consumeEach { frame ->
                        if (frame is Frame.Text) {
                            channel.send(frame.readText())
                        }
                    }
                } finally {
                    isConnected = false
                }
            } catch (_: CancellationException) {
                isConnected = false
            } catch (cause: Throwable) {
                isConnected = false
                // Surface transport-level failures to the flow consumer.
                channel.close(cause)
            } finally {
                channel.close()
            }
        }
        return channel.receiveAsFlow()
    }

    override suspend fun send(text: String) {
        val s = session ?: error("KtorWebSocketTransport is not connected; call connect() first.")
        s.send(Frame.Text(text))
    }

    override fun close() {
        teardown()
    }

    /**
     * Cancel the in-flight connection job (which also tears down the WS session
     * opened inside it) and close the inbound channel. Safe to call repeatedly.
     */
    private fun teardown() {
        connectionJob?.cancel()
        connectionJob = null
        incomingChannel?.close()
        incomingChannel = null
        session = null
        isConnected = false
    }
}
