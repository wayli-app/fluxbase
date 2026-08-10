package io.github.nimbleflux.fluxbase.settings

import io.github.nimbleflux.fluxbase.FluxbaseResponse
import io.github.nimbleflux.fluxbase.core.FluxbaseHttpClient
import io.github.nimbleflux.fluxbase.fluxbaseResponse
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonElement

/**
 * A secret metadata entry (for user secrets stored encrypted on the server).
 * Values are never returned — only metadata.
 */
@Serializable
data class UserSecretMetadata(
    val key: String,
    val description: String? = null,
    @SerialName("created_at") val createdAt: String = "",
    @SerialName("updated_at") val updatedAt: String = "",
)

/**
 * Settings module — port of the public `SettingsClient` from `sdk/src/settings.ts`.
 *
 * Provides read access to app/system settings (RLS-respecting) and CRUD for
 * encrypted user secrets. The TS SDK splits this into multiple admin managers;
 * this public client covers what app clients need: reading settings and managing
 * their own encrypted secrets.
 *
 * Usage:
 * ```
 * val (config, _) = client.settings.get("wayli.public_trips_require_auth")
 * client.settings.setUserSecret("owntracks_api_key", "key-value", "OwnTracks key")
 * ```
 */
class FluxbaseSettings(private val http: FluxbaseHttpClient) {

    /** Get a single setting value by key. GETs `/api/v1/settings/{key}`. */
    suspend fun get(key: String): FluxbaseResponse<JsonElement> = fluxbaseResponse {
        http.get("/api/v1/settings/${encode(key)}")
    }

    /** Get multiple settings. POSTs `/api/v1/settings/batch`. */
    suspend fun getMany(keys: List<String>, prefix: String? = null): FluxbaseResponse<JsonElement> = fluxbaseResponse {
        val body = mutableMapOf<String, Any?>("keys" to keys)
        prefix?.let { body["prefix"] = it }
        http.post("/api/v1/settings/batch", body)
    }

    // ---- User secrets (encrypted) — matches fluxbase.settings.setSecret in TS ----

    /** Set an encrypted user secret. PUTs `/api/v1/settings/secret/{key}`. */
    suspend fun setUserSecret(key: String, value: String, description: String? = null): FluxbaseResponse<Unit> = fluxbaseResponse {
        val body = mutableMapOf<String, Any?>("value" to value)
        description?.let { body["description"] = it }
        http.put<Unit>("/api/v1/settings/secret/${encode(key)}", body)
        Unit
    }

    /** List user secret metadata. GETs `/api/v1/settings/secret`. */
    suspend fun listUserSecrets(): FluxbaseResponse<List<UserSecretMetadata>> = fluxbaseResponse {
        http.get("/api/v1/settings/secret")
    }

    /** Delete a user secret. DELETEs `/api/v1/settings/secret/{key}`. */
    suspend fun deleteUserSecret(key: String): FluxbaseResponse<Unit> = fluxbaseResponse {
        http.delete("/api/v1/settings/secret/${encode(key)}")
        Unit
    }

    private fun encode(s: String): String = java.net.URLEncoder.encode(s, "UTF-8")
}
