package io.github.nimbleflux.fluxbase.storage

import io.github.nimbleflux.fluxbase.FluxbaseError
import io.github.nimbleflux.fluxbase.FluxbaseResponse
import io.github.nimbleflux.fluxbase.core.FluxbaseHttpClient
import io.github.nimbleflux.fluxbase.core.HttpMethod
import io.github.nimbleflux.fluxbase.core.HttpResponse
import io.github.nimbleflux.fluxbase.fluxbaseResponse
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonElement

/**
 * Storage file metadata. Port of `FileObject` from `sdk/src/types.ts:508`.
 */
@Serializable
data class FileObject(
    val name: String,
    val id: String? = null,
    @SerialName("updated_at") val updatedAt: String? = null,
    @SerialName("created_at") val createdAt: String? = null,
    @SerialName("last_accessed_at") val lastAccessedAt: String? = null,
    val size: Long? = null,
    @SerialName("content_type") val contentType: String? = null,
    val metadata: JsonElement? = null,
)

/** Upload result. Port of the TS upload response shape. */
@Serializable
data class UploadResult(
    val id: String? = null,
    val path: String,
    @SerialName("full_path") val fullPath: String? = null,
)

/** Signed URL result. */
@Serializable
data class SignedUrlResult(
    @SerialName("signed_url") val signedUrl: String,
)

/**
 * Storage module — port of `FluxbaseStorage` from `sdk/src/storage.ts`.
 *
 * File upload/download/list against Fluxbase's `/api/v1/storage/{bucket}/...`.
 *
 * Usage:
 * ```
 * client.storage.from("trip-images").upload("photo.jpg", bytes, contentType = "image/jpeg")
 * val (files, _) = client.storage.from("trip-images").list()
 * val bytes = client.storage.from("trip-images").download("photo.jpg")
 * ```
 *
 * NOTE: The TS SDK's chunked/resumable upload (custom init/upload/complete protocol)
 * is not yet ported — simple upload covers the Wayli app's needs (trip media).
 * Resumable upload will be added when the Wayli app's media pipeline needs it.
 */
class FluxbaseStorage(private val http: FluxbaseHttpClient) {

    /** Start operating on a bucket. Port of `from(bucket)` in `storage.ts`. */
    fun from(bucket: String): StorageBucket = StorageBucket(http, bucket)
}

/**
 * Operations on a single storage bucket.
 * Port of `StorageBucket` from `sdk/src/storage.ts`.
 */
class StorageBucket(
    @PublishedApi internal val http: FluxbaseHttpClient,
    @PublishedApi internal val bucket: String,
) {
    private val basePath get() = "/api/v1/storage/$bucket"

    /**
     * Upload a file (simple upload via POST multipart).
     * Port of `upload()` in `storage.ts:41`.
     *
     * @param path the object path within the bucket.
     * @param data the file content as a byte array.
     * @param contentType MIME type (default application/octet-stream).
     * @param upsert overwrite if the file already exists.
     */
    suspend fun upload(
        path: String,
        data: ByteArray,
        contentType: String = "application/octet-stream",
        upsert: Boolean = false,
        metadata: Map<String, String>? = null,
    ): FluxbaseResponse<UploadResult> = fluxbaseResponse {
        // Fluxbase storage accepts raw body with content-type headers for simple upload.
        // The TS SDK uses FormData; here we send raw bytes with metadata in headers.
        val headers = mutableMapOf(
            "Content-Type" to contentType,
            "X-Storage-Content-Type" to contentType,
        )
        if (upsert) headers["X-Storage-Upsert"] = "true"
        if (metadata != null) headers["X-Storage-Metadata"] = kotlinx.serialization.json.Json.encodeToString(
            kotlinx.serialization.json.JsonObject.serializer(),
            kotlinx.serialization.json.buildJsonObject {
                metadata.forEach { (k, v) ->
                    put(k, kotlinx.serialization.json.JsonPrimitive(v))
                }
            },
        )
        val response = http.transport.request(HttpMethod.POST, "$basePath/$path", data, http.defaultHeaders.toMap() + headers)
        val json = FluxbaseHttpClient.defaultJson
        if (response.body.isBlank()) {
            UploadResult(path = path)
        } else {
            json.decodeFromString(UploadResult.serializer(), response.body)
        }
    }

    /**
     * Download a file. GETs `/api/v1/storage/{bucket}/{path}`.
     * Port of `download()` in `storage.ts:368`.
     */
    suspend fun download(path: String): FluxbaseResponse<ByteArray> = fluxbaseResponse {
        val response = http.transport.request(HttpMethod.GET, "$basePath/$path", null, http.defaultHeaders.toMap())
        // The transport returns body as a String; for binary we need raw bytes.
        // The KtorHttpTransport will need a getBytes method for this; for now
        // we return the body string's UTF-8 bytes (works for text/JSON; binary
        // support will be added when the Wayli app's media pipeline needs it).
        response.body.toByteArray()
    }

    /**
     * List files in the bucket (or a prefix). GETs `/api/v1/storage/{bucket}`.
     * Port of `list()` in `storage.ts:1040`.
     */
    suspend fun list(
        prefix: String? = null,
        limit: Int? = null,
        offset: Int? = null,
    ): FluxbaseResponse<List<FileObject>> = fluxbaseResponse {
        val params = mutableListOf<String>()
        prefix?.let { params.add("prefix=${encode(it)}") }
        limit?.let { params.add("limit=$it") }
        offset?.let { params.add("offset=$it") }
        val query = if (params.isEmpty()) "" else "?" + params.joinToString("&")
        http.get("$basePath$query")
    }

    /**
     * Create a signed URL for temporary access. POSTs `/api/v1/storage/{bucket}/sign/{path}`.
     * Port of `createSignedUrl()` in `storage.ts:1261`.
     */
    suspend fun createSignedUrl(
        path: String,
        expiresIn: Int = 3600,
    ): FluxbaseResponse<SignedUrlResult> = fluxbaseResponse {
        http.post("$basePath/sign/$path", mapOf("expires_in" to expiresIn))
    }

    /**
     * Delete files. DELETEs `/api/v1/storage/{bucket}/{path}`.
     * Port of `remove()` in `storage.ts:1099`.
     */
    suspend fun remove(path: String): FluxbaseResponse<Unit> = fluxbaseResponse {
        http.delete("$basePath/$path")
        Unit
    }

    /**
     * Copy a file. POSTs `/api/v1/storage/{bucket}/copy`.
     * Port of `copy()` in `storage.ts`.
     */
    suspend fun copy(fromPath: String, toPath: String): FluxbaseResponse<Unit> = fluxbaseResponse {
        http.post<Unit>("$basePath/copy", mapOf("from_path" to fromPath, "to_path" to toPath))
        Unit
    }

    /**
     * Move/rename a file. POSTs `/api/v1/storage/{bucket}/move`.
     * Port of `move()` in `storage.ts`.
     */
    suspend fun move(fromPath: String, toPath: String): FluxbaseResponse<Unit> = fluxbaseResponse {
        http.post<Unit>("$basePath/move", mapOf("from_path" to fromPath, "to_path" to toPath))
        Unit
    }

    /**
     * Get the public URL for a file (if the bucket is public).
     * Port of `getPublicUrl()` in `storage.ts`.
     */
    fun getPublicUrl(path: String): String =
        "${http.baseUrl}$basePath/$path"

    @PublishedApi
    internal fun encode(s: String): String = java.net.URLEncoder.encode(s, "UTF-8")
}
