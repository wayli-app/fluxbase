package io.github.nimbleflux.fluxbase.tenant

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonElement

@Serializable
data class Tenant(
    val id: String,
    val slug: String,
    val name: String,
    @SerialName("db_name") val dbName: String? = null,
    val status: String = "active",
    @SerialName("is_default") val isDefault: Boolean = false,
    val metadata: JsonElement? = null,
    @SerialName("created_at") val createdAt: String = "",
    @SerialName("updated_at") val updatedAt: String? = null,
)

@Serializable
data class TenantAdminAssignment(
    val id: String,
    @SerialName("tenant_id") val tenantId: String,
    @SerialName("user_id") val userId: String,
    @SerialName("created_at") val createdAt: String = "",
)

data class CreateTenantOptions(
    val slug: String,
    val name: String,
    val metadata: Map<String, Any?>? = null,
)

data class UpdateTenantOptions(
    val name: String? = null,
    val metadata: Map<String, Any?>? = null,
)
