package io.github.nimbleflux.fluxbase.core

import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.boolean
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import kotlin.test.Test
import kotlin.test.assertEquals

/**
 * RPC request bodies are built as pre-assembled [JsonObject]s (see
 * [io.github.nimbleflux.fluxbase.rpc.FluxbaseRpc]). The transport must pass
 * them through untouched: re-wrapping leaves via `toString()` turned booleans
 * into "false" strings and quoted string values, which servers reject
 * ("missing required parameter: label" — the body bind failed and the whole
 * request was treated as bodyless).
 */
class KtorHttpTransportJsonTest {

    private fun transport() = KtorHttpTransport(
        baseUrl = "https://example.com",
    )

    @Test
    fun `rpc-style body survives serialization unchanged`() {
        val body = transport().encodeToJsonElement(
            JsonObject(
                mapOf(
                    "params" to JsonObject(
                        mapOf(
                            "label" to Json.parseToJsonElement("\"Android\""),
                            "token_hash" to Json.parseToJsonElement("\"abc123\""),
                        ),
                    ),
                    "async" to Json.parseToJsonElement("false"),
                ),
            ),
        )

        val obj = body.jsonObject
        assertEquals("Android", obj["params"]!!.jsonObject["label"]!!.jsonPrimitive.content)
        assertEquals("abc123", obj["params"]!!.jsonObject["token_hash"]!!.jsonPrimitive.content)
        assertEquals(false, obj["async"]!!.jsonPrimitive.boolean)
    }

    @Test
    fun `plain maps and primitives still encode`() {
        val body = transport().encodeToJsonElement(
            mapOf("name" to "Trip", "days" to 30, "enabled" to true),
        )

        val obj = body.jsonObject
        assertEquals("Trip", obj["name"]!!.jsonPrimitive.content)
        assertEquals(30, obj["days"]!!.jsonPrimitive.content.toInt())
        assertEquals(true, obj["enabled"]!!.jsonPrimitive.boolean)
    }
}
