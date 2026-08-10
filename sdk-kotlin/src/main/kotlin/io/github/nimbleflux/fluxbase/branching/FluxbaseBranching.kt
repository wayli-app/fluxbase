package io.github.nimbleflux.fluxbase.branching

import io.github.nimbleflux.fluxbase.FluxbaseResponse
import io.github.nimbleflux.fluxbase.core.FluxbaseHttpClient
import io.github.nimbleflux.fluxbase.fluxbaseResponse
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.put

/**
 * Database branching module — port of `FluxbaseBranching` from
 * `sdk/src/branching.ts`. All endpoints under `/api/v1/admin/branches`.
 *
 * Usage:
 * ```
 * val (branch, _) = client.branching.create("feature-x", CreateBranchOptions(dataCloneMode = "full_clone"))
 * val ready = client.branching.waitForReady(branch!!.id)
 * ```
 */
class FluxbaseBranching(@PublishedApi internal val http: FluxbaseHttpClient) {

    /** List branches. GETs `/api/v1/admin/branches`. */
    suspend fun list(options: ListBranchesOptions = ListBranchesOptions()): FluxbaseResponse<ListBranchesResponse> = fluxbaseResponse {
        val params = mutableListOf<String>()
        options.status?.let { params.add("status=${encode(it)}") }
        options.type?.let { params.add("type=${encode(it)}") }
        options.githubRepo?.let { params.add("github_repo=${encode(it)}") }
        options.mine?.let { params.add("mine=$it") }
        options.limit?.let { params.add("limit=$it") }
        options.offset?.let { params.add("offset=$it") }
        val query = if (params.isEmpty()) "" else "?" + params.joinToString("&")
        http.get("/api/v1/admin/branches$query")
    }

    /** Get a branch by ID or slug. */
    suspend fun get(idOrSlug: String): FluxbaseResponse<Branch> = fluxbaseResponse {
        http.get("/api/v1/admin/branches/${encode(idOrSlug)}")
    }

    /** Create a new branch. POSTs `/api/v1/admin/branches`. */
    suspend fun create(name: String, options: CreateBranchOptions = CreateBranchOptions()): FluxbaseResponse<Branch> = fluxbaseResponse {
        val body = buildJsonObject {
            put("name", name)
            options.parentBranchId?.let { put("parent_branch_id", it) }
            put("data_clone_mode", options.dataCloneMode)
            put("type", options.type)
            options.githubPrNumber?.let { put("github_pr_number", it) }
            options.githubPrUrl?.let { put("github_pr_url", it) }
            options.githubRepo?.let { put("github_repo", it) }
            options.expiresIn?.let { put("expires_in", it) }
        }
        http.post("/api/v1/admin/branches", body)
    }

    /** Delete a branch. DELETEs `/api/v1/admin/branches/{idOrSlug}`. */
    suspend fun delete(idOrSlug: String): FluxbaseResponse<Unit> = fluxbaseResponse {
        http.delete("/api/v1/admin/branches/${encode(idOrSlug)}")
        Unit
    }

    /** Reset a branch to its parent state. POSTs `.../reset`. */
    suspend fun reset(idOrSlug: String): FluxbaseResponse<Branch> = fluxbaseResponse {
        http.post("/api/v1/admin/branches/${encode(idOrSlug)}/reset", buildJsonObject {})
    }

    /** Check if a branch exists (returns boolean). */
    suspend fun exists(idOrSlug: String): Boolean {
        val result = get(idOrSlug)
        return result.error == null && result.data != null
    }

    @PublishedApi
    internal fun encode(s: String): String = java.net.URLEncoder.encode(s, "UTF-8")
}
