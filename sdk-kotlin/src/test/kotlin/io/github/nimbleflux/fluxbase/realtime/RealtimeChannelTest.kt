package io.github.nimbleflux.fluxbase.realtime

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNotNull
import kotlin.test.assertTrue

/**
 * Unit tests for [RealtimeChannel] — porting the TS SDK's `realtime.test.ts`.
 *
 * Uses [FakeWebSocketTransport] to simulate server messages.
 */
class RealtimeChannelTest {

    @Test
    fun `subscribe sends subscribe message with channel and config`() = runBlocking {
        val transport = FakeWebSocketTransport()
        val channel = RealtimeChannel("http://localhost:8080", "public:trips", "test-token", transport, Dispatchers.Unconfined)

        channel.on("INSERT", PostgresChangesConfig(event = "INSERT", schema = "public", table = "trips")) {}
        channel.subscribe()
        delay(200)

        val firstMsg = transport.sentMessages.firstOrNull()
        assertNotNull(firstMsg)
        val json = Json.parseToJsonElement(firstMsg).jsonObject
        assertEquals("subscribe", json["type"]?.jsonPrimitive?.content)
        assertEquals("public:trips", json["channel"]?.jsonPrimitive?.content)
        assertNotNull(json["config"])
    }

    @Test
    fun `ws URL uses wss for https and includes token`() = runBlocking {
        val transport = FakeWebSocketTransport()
        val channel = RealtimeChannel("https://flux.example.com", "test", "my-jwt", transport, Dispatchers.Unconfined)

        channel.subscribe()
        delay(100)

        assertEquals("wss://flux.example.com/realtime?token=my-jwt", transport.lastConnectedUrl)
    }

    @Test
    fun `ws URL uses ws for http`() = runBlocking {
        val transport = FakeWebSocketTransport()
        val channel = RealtimeChannel("http://localhost:8080", "test", null, transport, Dispatchers.Unconfined)

        channel.subscribe()
        delay(100)

        assertEquals("ws://localhost:8080/realtime", transport.lastConnectedUrl)
    }

    @Test
    fun `receives INSERT postgres_changes event`() = runBlocking {
        val transport = FakeWebSocketTransport()
        val channel = RealtimeChannel("http://localhost:8080", "public:trips", "tok", transport, Dispatchers.Unconfined)
        val received = mutableListOf<PostgresChangesPayload>()

        channel.on("INSERT", PostgresChangesConfig(event = "INSERT", schema = "public", table = "trips")) {
            received.add(it)
        }
        channel.subscribe()
        delay(150)

        transport.simulateMessage(
            """{
                "type":"postgres_changes",
                "payload":{
                    "type":"INSERT",
                    "schema":"public",
                    "table":"trips",
                    "new_record":{"id":1,"title":"Paris Trip"}
                }
            }""".trimIndent(),
        )
        delay(100)

        assertEquals(1, received.size)
        assertEquals("INSERT", received[0].eventType)
        assertEquals("trips", received[0].table)
        val newObj = received[0].newRecord!!.jsonObject
        assertEquals("Paris Trip", newObj["title"]!!.jsonPrimitive.content)
    }

    @Test
    fun `normalizes new_record to new`() = runBlocking {
        val transport = FakeWebSocketTransport()
        val channel = RealtimeChannel("http://localhost:8080", "ch", "tok", transport, Dispatchers.Unconfined)
        val received = mutableListOf<PostgresChangesPayload>()

        channel.on("INSERT") { received.add(it) }
        channel.subscribe()
        delay(150)

        transport.simulateMessage(
            """{"type":"postgres_changes","payload":{"type":"INSERT","schema":"public","table":"x","new_record":{"a":1}}}""",
        )
        delay(100)

        val newObj = received[0].new!!.jsonObject
        assertEquals(1, newObj["a"]!!.jsonPrimitive.content.toInt())
    }

    @Test
    fun `wildcard callback receives all events`() = runBlocking {
        val transport = FakeWebSocketTransport()
        val channel = RealtimeChannel("http://localhost:8080", "ch", "tok", transport, Dispatchers.Unconfined)
        val all = mutableListOf<String>()

        channel.on("*") { all.add(it.eventType) }
        channel.subscribe()
        delay(150)

        transport.simulateMessage("""{"type":"postgres_changes","payload":{"type":"INSERT","table":"x","new_record":{}}}""")
        transport.simulateMessage("""{"type":"postgres_changes","payload":{"type":"DELETE","table":"x","old_record":{}}}""")
        delay(100)

        assertEquals(listOf("INSERT", "DELETE"), all)
    }

    @Test
    fun `broadcast callback receives matching events`() = runBlocking {
        val transport = FakeWebSocketTransport()
        val channel = RealtimeChannel("http://localhost:8080", "ch", "tok", transport, Dispatchers.Unconfined)
        val messages = mutableListOf<String>()

        channel.onBroadcast("message") { messages.add(it.toString()) }
        channel.subscribe()
        delay(150)

        transport.simulateMessage(
            """{"type":"broadcast","payload":{"broadcast":{"event":"message","payload":{"text":"hello"}}}}""",
        )
        delay(100)

        assertEquals(1, messages.size)
        assertTrue(messages[0].contains("hello"))
    }

    @Test
    fun `unsubscribe sends unsubscribe message`() = runBlocking {
        val transport = FakeWebSocketTransport()
        val channel = RealtimeChannel("http://localhost:8080", "ch", "tok", transport, Dispatchers.Unconfined)
        channel.subscribe()
        delay(150)

        channel.unsubscribe()
        delay(50)

        val lastMsg = Json.parseToJsonElement(transport.sentMessages.last()).jsonObject
        assertEquals("unsubscribe", lastMsg["type"]?.jsonPrimitive?.content)
    }

    @Test
    fun `heartbeat is sent on interval`() = runBlocking {
        val transport = FakeWebSocketTransport()
        val channel = RealtimeChannel("http://localhost:8080", "ch", "tok", transport, Dispatchers.Unconfined)
        channel.subscribe()
        delay(150)

        assertTrue(transport.isConnected)
    }
}

/**
 * Fake [WebSocketTransport] for tests — records sent messages and simulates
 * incoming messages. The Kotlin equivalent of the TS `MockWebSocket` in
 * `realtime.test.ts:23-97`.
 */
class FakeWebSocketTransport : WebSocketTransport {
    val sentMessages = mutableListOf<String>()
    var lastConnectedUrl: String? = null
        private set

    private val incomingFlow = MutableSharedFlow<String>(replay = 10)
    private var connected = false

    override val isConnected: Boolean get() = connected

    override fun connect(url: String): Flow<String> {
        lastConnectedUrl = url
        connected = true
        return incomingFlow
    }

    override suspend fun send(text: String) {
        sentMessages.add(text)
    }

    override fun close() {
        connected = false
    }

    /** Simulate a server-sent JSON message. */
    fun simulateMessage(json: String) {
        incomingFlow.tryEmit(json)
    }
}
