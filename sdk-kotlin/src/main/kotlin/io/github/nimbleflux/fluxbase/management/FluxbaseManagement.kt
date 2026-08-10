package io.github.nimbleflux.fluxbase.management

import io.github.nimbleflux.fluxbase.FluxbaseResponse
import io.github.nimbleflux.fluxbase.core.FluxbaseHttpClient
import io.github.nimbleflux.fluxbase.fluxbaseResponse

/**
 * Management module — port of `FluxbaseManagement` from `sdk/src/management.ts`.
 *
 * Aggregate of three sub-managers: client keys, webhooks, invitations.
 * Unlike most modules which wrap in `{data, error}`, the TS management managers
 * throw on error. The Kotlin port wraps them in [FluxbaseResponse] for consistency.
 *
 * Usage:
 * ```
 * val (key, _) = client.management.clientKeys.create(mapOf("name" to "mobile-app", "scopes" to listOf("read")))
 * val (hooks, _) = client.management.webhooks.list()
 * ```
 */
class FluxbaseManagement(http: FluxbaseHttpClient) {
    val clientKeys = ClientKeysManager(http)
    val webhooks = WebhooksManager(http)
    val invitations = InvitationsManager(http)
}

/** Client API keys. Endpoints under `/api/v1/client-keys`. */
class ClientKeysManager(@PublishedApi internal val http: FluxbaseHttpClient) {

    suspend fun create(
        name: String,
        scopes: List<String>,
        description: String? = null,
        rateLimitPerMinute: Int? = null,
        expiresAt: String? = null,
    ): FluxbaseResponse<CreateClientKeyResponse> = fluxbaseResponse {
        val body = mutableMapOf<String, Any?>("name" to name, "scopes" to scopes)
        description?.let { body["description"] = it }
        rateLimitPerMinute?.let { body["rate_limit_per_minute"] = it }
        expiresAt?.let { body["expires_at"] = it }
        http.post("/api/v1/client-keys", body)
    }

    suspend fun list(): FluxbaseResponse<ListClientKeysResponse> = fluxbaseResponse {
        http.get("/api/v1/client-keys")
    }

    suspend fun revoke(keyId: String): FluxbaseResponse<Unit> = fluxbaseResponse {
        http.post<Unit>("/api/v1/client-keys/$keyId/revoke")
        Unit
    }

    suspend fun delete(keyId: String): FluxbaseResponse<Unit> = fluxbaseResponse {
        http.delete("/api/v1/client-keys/$keyId")
        Unit
    }
}

/** Webhooks. Endpoints under `/api/v1/webhooks`. */
class WebhooksManager(@PublishedApi internal val http: FluxbaseHttpClient) {

    suspend fun create(
        url: String,
        events: List<String>,
        description: String? = null,
        secret: String? = null,
    ): FluxbaseResponse<Webhook> = fluxbaseResponse {
        val body = mutableMapOf<String, Any?>("url" to url, "events" to events)
        description?.let { body["description"] = it }
        secret?.let { body["secret"] = it }
        http.post("/api/v1/webhooks", body)
    }

    suspend fun list(): FluxbaseResponse<ListWebhooksResponse> = fluxbaseResponse {
        http.get("/api/v1/webhooks")
    }

    suspend fun delete(webhookId: String): FluxbaseResponse<Unit> = fluxbaseResponse {
        http.delete("/api/v1/webhooks/$webhookId")
        Unit
    }

    suspend fun test(webhookId: String): FluxbaseResponse<Unit> = fluxbaseResponse {
        http.post<Unit>("/api/v1/webhooks/$webhookId/test")
        Unit
    }
}

/** Invitations. Public methods: validate, accept. Admin methods: create, list, revoke. */
class InvitationsManager(@PublishedApi internal val http: FluxbaseHttpClient) {

    /** Validate an invitation token (public). GETs `/api/v1/invitations/{token}/validate`. */
    suspend fun validate(token: String): FluxbaseResponse<ValidateInvitationResponse> = fluxbaseResponse {
        http.get("/api/v1/invitations/$token/validate")
    }

    /** Accept an invitation (public). POSTs `/api/v1/invitations/{token}/accept`. */
    suspend fun accept(token: String, password: String, name: String): FluxbaseResponse<Unit> = fluxbaseResponse {
        http.post<Unit>("/api/v1/invitations/$token/accept", mapOf("password" to password, "name" to name))
        Unit
    }
}
