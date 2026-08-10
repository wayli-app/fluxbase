package io.github.nimbleflux.fluxbase.secrets

import io.github.nimbleflux.fluxbase.FluxbaseResponse
import io.github.nimbleflux.fluxbase.core.FluxbaseHttpClient
import io.github.nimbleflux.fluxbase.fluxbaseResponse
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * A secret summary (metadata only — values are never returned by the API).
 * Port of `SecretSummary` from `sdk/src/secrets.ts`.
 */
@Serializable
data class SecretSummary(
    val id: String,
    val name: String,
    val scope: String = "global",
    val namespace: String? = null,
    val description: String? = null,
    @SerialName("current_version") val currentVersion: Int = 1,
    @SerialName("created_at") val createdAt: String = "",
    @SerialName("updated_at") val updatedAt: String = "",
)

/**
 * Encrypted Secrets module — port of `SecretsManager` from `sdk/src/secrets.ts`.
 *
 * Fluxbase's secrets are encrypted at rest with `FLUXBASE_ENCRYPTION_KEY` and
 * injected into function/job runtime environments. Values are never returned
 * by the API — only metadata.
 *
 * Usage:
 * ```
 * client.secrets.set("my-api-key", "secret-value", scope = "namespace", namespace = "wayli")
 * val (secrets, _) = client.secrets.list(namespace = "wayli")
 * ```
 */
class FluxbaseSecrets(private val http: FluxbaseHttpClient) {

    /** Create a secret. POSTs `/api/v1/secrets`. */
    suspend fun set(
        name: String,
        value: String,
        scope: String = "global",
        namespace: String? = null,
        description: String? = null,
    ): FluxbaseResponse<SecretSummary> = fluxbaseResponse {
        val body = mutableMapOf<String, Any?>(
            "name" to name,
            "value" to value,
            "scope" to scope,
        )
        namespace?.let { body["namespace"] = it }
        description?.let { body["description"] = it }
        http.post("/api/v1/secrets", body)
    }

    /** Get a secret's metadata by name. GETs `/api/v1/secrets/by-name/{name}`. */
    suspend fun get(name: String, namespace: String? = null): FluxbaseResponse<SecretSummary> = fluxbaseResponse {
        val query = namespace?.let { "?namespace=${encode(it)}" } ?: ""
        http.get("/api/v1/secrets/by-name/$name$query")
    }

    /** List all secrets (metadata only). GETs `/api/v1/secrets`. */
    suspend fun list(scope: String? = null, namespace: String? = null): FluxbaseResponse<List<SecretSummary>> = fluxbaseResponse {
        val params = mutableListOf<String>()
        scope?.let { params.add("scope=${encode(it)}") }
        namespace?.let { params.add("namespace=${encode(it)}") }
        val query = if (params.isEmpty()) "" else "?" + params.joinToString("&")
        http.get("/api/v1/secrets$query")
    }

    /** Delete a secret by name. DELETEs `/api/v1/secrets/by-name/{name}`. */
    suspend fun delete(name: String, namespace: String? = null): FluxbaseResponse<Unit> = fluxbaseResponse {
        val query = namespace?.let { "?namespace=${encode(it)}" } ?: ""
        http.delete("/api/v1/secrets/by-name/$name$query")
        Unit
    }

    /** Update a secret's value (creates a new version). PUTs `/api/v1/secrets/by-name/{name}`. */
    suspend fun update(
        name: String,
        value: String,
        namespace: String? = null,
        description: String? = null,
    ): FluxbaseResponse<SecretSummary> = fluxbaseResponse {
        val body = mutableMapOf<String, Any?>("value" to value)
        description?.let { body["description"] = it }
        val query = namespace?.let { "?namespace=${encode(it)}" } ?: ""
        http.put("/api/v1/secrets/by-name/$name$query", body)
    }

    private fun encode(s: String): String = java.net.URLEncoder.encode(s, "UTF-8")
}
