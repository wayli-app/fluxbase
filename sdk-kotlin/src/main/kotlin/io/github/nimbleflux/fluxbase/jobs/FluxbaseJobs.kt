package io.github.nimbleflux.fluxbase.jobs

import io.github.nimbleflux.fluxbase.FluxbaseResponse
import io.github.nimbleflux.fluxbase.core.FluxbaseHttpClient
import io.github.nimbleflux.fluxbase.fluxbaseResponse
import kotlinx.serialization.json.JsonElement

/**
 * Background Jobs module — port of `FluxbaseJobs` from `sdk/src/jobs.ts`.
 *
 * Submit, track, and cancel background jobs. Fluxbase's jobs surface has no
 * Supabase equivalent — this is a Fluxbase-only feature.
 *
 * NOTE: The TS SDK has a bug on `jobs.ts:64` — it calls the unregistered path
 * `/rest/v1/user_profiles` instead of `/api/v1/tables/user_profiles`. The
 * `getCurrentUserRole()` auto-population of `onBehalfOf` for service-role
 * clients is not ported here (it requires app-specific user_profiles knowledge);
 * callers pass `onBehalfOf` explicitly if needed.
 *
 * Usage:
 * ```
 * val (job, error) = client.jobs.submit("trip-detection", mapOf("date" to "2024-01-01"), SubmitJobOptions(namespace = "wayli"))
 * val (status, _) = client.jobs.get(job!!.id)
 * ```
 */
class FluxbaseJobs(private val http: FluxbaseHttpClient) {

    /**
     * Submit a new job for execution. POSTs to `/api/v1/jobs/submit`.
     * Port of `submit()` in `jobs.ts:130`.
     */
    suspend fun submit(
        jobName: String,
        payload: Any? = null,
        options: SubmitJobOptions = SubmitJobOptions(),
    ): FluxbaseResponse<Job> = fluxbaseResponse {
        val body = mutableMapOf<String, Any?>(
            "job_name" to jobName,
            "payload" to payload,
        )
        options.priority?.let { body["priority"] = it }
        options.namespace?.let { body["namespace"] = it }
        options.scheduled?.let { body["scheduled"] = it }
        options.onBehalfOf?.let {
            body["on_behalf_of"] = mapOf(
                "user_id" to it.userId,
                "user_email" to it.userEmail,
                "user_role" to it.userRole,
            )
        }
        http.post("/api/v1/jobs/submit", body)
    }

    /** Get a job by ID. GETs `/api/v1/jobs/{id}`. */
    suspend fun get(jobId: String): FluxbaseResponse<Job> = fluxbaseResponse {
        http.get("/api/v1/jobs/$jobId")
    }

    /** List jobs. GETs `/api/v1/jobs` with optional filters. */
    suspend fun list(
        status: String? = null,
        namespace: String? = null,
        limit: Int? = null,
        offset: Int? = null,
    ): FluxbaseResponse<List<Job>> = fluxbaseResponse {
        val params = mutableListOf<String>()
        status?.let { params.add("status=${encode(it)}") }
        namespace?.let { params.add("namespace=${encode(it)}") }
        limit?.let { params.add("limit=$it") }
        offset?.let { params.add("offset=$it") }
        val query = if (params.isEmpty()) "" else "?" + params.joinToString("&")
        http.get("/api/v1/jobs$query")
    }

    /** Cancel a running job. POSTs `/api/v1/jobs/{id}/cancel`. */
    suspend fun cancel(jobId: String): FluxbaseResponse<Job> = fluxbaseResponse {
        http.post("/api/v1/jobs/$jobId/cancel")
    }

    /** Retry a failed job. POSTs `/api/v1/jobs/{id}/retry`. */
    suspend fun retry(jobId: String): FluxbaseResponse<Job> = fluxbaseResponse {
        http.post("/api/v1/jobs/$jobId/retry")
    }

    /** Get job execution logs. GETs `/api/v1/jobs/{id}/logs`. */
    suspend fun getLogs(jobId: String, afterLine: Int? = null): FluxbaseResponse<List<ExecutionLog>> = fluxbaseResponse {
        val query = afterLine?.let { "?after_line=$it" } ?: ""
        http.get("/api/v1/jobs/$jobId/logs$query")
    }

    private fun encode(s: String): String = java.net.URLEncoder.encode(s, "UTF-8")
}
