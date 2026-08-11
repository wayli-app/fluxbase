package io.github.nimbleflux.fluxbase.realtime

import kotlinx.coroutines.test.runTest
import kotlin.test.Test
import kotlin.test.assertFalse
import kotlin.test.assertTrue

/**
 * Lifecycle tests for [KtorWebSocketTransport].
 *
 * The transport is a thin I/O wrapper around Ktor's WebSocket client; the
 * realtime *protocol* (subscribe/heartbeat/postgres_changes/broadcast/ack/error)
 * is exercised exhaustively in [RealtimeChannelTest] via [FakeWebSocketTransport].
 *
 * A full end-to-end round-trip against a live server belongs in the
 * `integrationTest` source set (see build.gradle.kts), not the fast unit suite —
 * a real Netty server + OkHttp pool here would keep the forked test JVM alive
 * after the method returns. These tests cover the deterministic lifecycle guards
 * that don't need a server.
 */
class KtorWebSocketTransportTest {

    @Test
    fun `starts disconnected`() {
        val transport = KtorWebSocketTransport()
        assertFalse(transport.isConnected, "a fresh transport must report disconnected")
    }

    @Test
    fun `close marks the transport disconnected`() = runTest {
        val transport = KtorWebSocketTransport()
        // close() is safe to call without a prior connect() (idempotent teardown).
        transport.close()
        assertFalse(transport.isConnected, "close() must leave isConnected false")
    }

    @Test
    fun `send before connect is a programmer error`() = runTest {
        val transport = KtorWebSocketTransport()
        val thrown = kotlin.test.assertFailsWith<IllegalStateException> {
            transport.send("oops")
        }
        assertTrue(thrown.message!!.contains("not connected"), "error must explain connect() is required")
    }
}
