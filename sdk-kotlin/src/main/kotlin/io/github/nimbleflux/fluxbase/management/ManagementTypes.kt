package io.github.nimbleflux.fluxbase.management

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonElement

// ---- Client Keys ----

@Serializable
data class ClientKey(
    val id: String,
    val name: String,
    val description: String? = null,
    @SerialName("key_prefix") val keyPrefix: String = "",
    val scopes: List<String> = emptyList(),
    @SerialName("rate_limit_per_minute") val rateLimitPerMinute: Int = 0,
    @SerialName("created_at") val createdAt: String = "",
    @SerialName("expires_at") val expiresAt: String? = null,
    @SerialName("revoked_at") val revokedAt: String? = null,
    @SerialName("last_used_at") val lastUsedAt: String? = null,
    @SerialName("user_id") val userId: String = "",
)

@Serializable
data class CreateClientKeyResponse(
    @SerialName("client_key") val clientKey: ClientKey,
    val key: String,
)

@Serializable
data class ListClientKeysResponse(
    @SerialName("client_keys") val clientKeys: List<ClientKey> = emptyList(),
    val total: Int = 0,
)

// ---- Webhooks ----

@Serializable
data class Webhook(
    val id: String,
    val url: String,
    val events: List<String> = emptyList(),
    val secret: String? = null,
    val description: String? = null,
    @SerialName("is_active") val isActive: Boolean = true,
    @SerialName("created_at") val createdAt: String = "",
    @SerialName("user_id") val userId: String = "",
)

@Serializable
data class ListWebhooksResponse(
    val webhooks: List<Webhook> = emptyList(),
    val total: Int = 0,
)

// ---- Invitations ----

@Serializable
data class Invitation(
    val id: String,
    val email: String,
    val role: String,
    val token: String? = null,
    @SerialName("invited_by") val invitedBy: String = "",
    @SerialName("accepted_at") val acceptedAt: String? = null,
    @SerialName("expires_at") val expiresAt: String = "",
    @SerialName("created_at") val createdAt: String = "",
)

@Serializable
data class ValidateInvitationResponse(
    val valid: Boolean,
    val invitation: Invitation? = null,
    val error: String? = null,
)
