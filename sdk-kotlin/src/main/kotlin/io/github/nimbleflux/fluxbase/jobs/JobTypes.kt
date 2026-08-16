package io.github.nimbleflux.fluxbase.jobs

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonElement

/**
 * A background job. Port of `Job` from `sdk/src/types.ts:2675`.
 */
@Serializable
data class Job(
    val id: String,
    val namespace: String? = null,
    @SerialName("job_name") val jobName: String,
    val status: String = "pending",
    val payload: JsonElement? = null,
    val result: JsonElement? = null,
    val error: String? = null,
    val priority: Int = 5,
    @SerialName("progress_percent") val progressPercent: Int? = null,
    @SerialName("progress_message") val progressMessage: String? = null,
    @SerialName("retry_count") val retryCount: Int = 0,
    @SerialName("created_by") val createdBy: String? = null,
    @SerialName("created_at") val createdAt: String = "",
    @SerialName("updated_at") val updatedAt: String = "",
)

/** Submit a job on behalf of a user (service_role only). */
@Serializable
data class OnBehalfOf(
    @SerialName("user_id") val userId: String,
    @SerialName("user_email") val userEmail: String? = null,
    @SerialName("user_role") val userRole: String? = null,
)

/** Options for [FluxbaseJobs.submit]. */
data class SubmitJobOptions(
    val priority: Int? = null,
    val namespace: String? = null,
    val scheduled: String? = null,
    val onBehalfOf: OnBehalfOf? = null,
)

/** Log entry from a job execution. */
@Serializable
data class ExecutionLog(
    val line: String,
    @SerialName("timestamp") val timestamp: String? = null,
    val level: String? = null,
)
