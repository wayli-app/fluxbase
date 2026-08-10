package io.github.nimbleflux.fluxbase.branching

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonElement

@Serializable
data class Branch(
    val id: String,
    val name: String,
    val slug: String,
    @SerialName("database_name") val databaseName: String,
    val status: String,
    val type: String = "preview",
    @SerialName("parent_branch_id") val parentBranchId: String? = null,
    @SerialName("data_clone_mode") val dataCloneMode: String = "schema_only",
    @SerialName("github_pr_number") val githubPrNumber: Int? = null,
    @SerialName("github_pr_url") val githubPrUrl: String? = null,
    @SerialName("github_repo") val githubRepo: String? = null,
    @SerialName("error_message") val errorMessage: String? = null,
    @SerialName("created_by") val createdBy: String? = null,
    @SerialName("created_at") val createdAt: String = "",
    @SerialName("updated_at") val updatedAt: String = "",
    @SerialName("expires_at") val expiresAt: String? = null,
)

@Serializable
data class ListBranchesResponse(
    val branches: List<Branch> = emptyList(),
    val total: Int = 0,
    val limit: Int = 0,
    val offset: Int = 0,
)

@Serializable
data class BranchActivity(
    val id: String,
    @SerialName("branch_id") val branchId: String,
    val action: String,
    val status: String,
    val details: JsonElement? = null,
    @SerialName("executed_by") val executedBy: String? = null,
    @SerialName("created_at") val createdAt: String = "",
)

data class CreateBranchOptions(
    val parentBranchId: String? = null,
    val dataCloneMode: String = "schema_only",
    val type: String = "preview",
    val githubPrNumber: Int? = null,
    val githubPrUrl: String? = null,
    val githubRepo: String? = null,
    val expiresIn: String? = null,
)

data class ListBranchesOptions(
    val status: String? = null,
    val type: String? = null,
    val githubRepo: String? = null,
    val mine: Boolean? = null,
    val limit: Int? = null,
    val offset: Int? = null,
)
