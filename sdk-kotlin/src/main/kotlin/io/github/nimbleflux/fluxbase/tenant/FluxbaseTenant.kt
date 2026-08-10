package io.github.nimbleflux.fluxbase.tenant

import io.github.nimbleflux.fluxbase.FluxbaseResponse
import io.github.nimbleflux.fluxbase.core.FluxbaseHttpClient
import io.github.nimbleflux.fluxbase.fluxbaseResponse
import kotlinx.serialization.json.JsonElement

/**
 * Multi-tenancy module — port of `FluxbaseTenant` from `sdk/src/tenant.ts`.
 * All endpoints under `/api/v1/admin/tenants`.
 *
 * Usage:
 * ```
 * val (tenant, _) = client.tenant.create(CreateTenantOptions(slug = "acme", name = "Acme Inc"))
 * client.setTenant(tenant!!.id) // scope subsequent requests to this tenant
 * ```
 */
class FluxbaseTenant(@PublishedApi internal val http: FluxbaseHttpClient) {

    /** List all tenants (instance admin only). */
    suspend fun list(): FluxbaseResponse<List<Tenant>> = fluxbaseResponse {
        http.get("/api/v1/admin/tenants")
    }

    /** List tenants the current user belongs to. */
    suspend fun listMine(): FluxbaseResponse<List<Tenant>> = fluxbaseResponse {
        http.get("/api/v1/admin/tenants/mine")
    }

    /** Get a tenant by ID. */
    suspend fun get(id: String): FluxbaseResponse<Tenant> = fluxbaseResponse {
        http.get("/api/v1/admin/tenants/$id")
    }

    /** Create a tenant. POSTs `/api/v1/admin/tenants`. */
    suspend fun create(options: CreateTenantOptions): FluxbaseResponse<Tenant> = fluxbaseResponse {
        val body = mutableMapOf<String, Any?>("slug" to options.slug, "name" to options.name)
        options.metadata?.let { body["metadata"] = it }
        http.post("/api/v1/admin/tenants", body)
    }

    /** Update a tenant. PATCHes `/api/v1/admin/tenants/{id}`. */
    suspend fun update(id: String, options: UpdateTenantOptions): FluxbaseResponse<Tenant> = fluxbaseResponse {
        val body = mutableMapOf<String, Any?>()
        options.name?.let { body["name"] = it }
        options.metadata?.let { body["metadata"] = it }
        http.patch("/api/v1/admin/tenants/$id", body)
    }

    /** Delete a tenant. DELETEs `/api/v1/admin/tenants/{id}`. */
    suspend fun delete(id: String): FluxbaseResponse<Unit> = fluxbaseResponse {
        http.delete("/api/v1/admin/tenants/$id")
        Unit
    }

    /** Run pending migrations for a tenant. POSTs `.../migrate`. */
    suspend fun migrate(id: String): FluxbaseResponse<JsonElement> = fluxbaseResponse {
        http.post("/api/v1/admin/tenants/$id/migrate", mapOf<String, Any?>())
    }
}
