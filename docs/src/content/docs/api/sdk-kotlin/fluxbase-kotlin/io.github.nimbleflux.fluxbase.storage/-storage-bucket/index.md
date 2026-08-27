---
title: "StorageBucket"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../)/[io.github.nimbleflux.fluxbase.storage](../)/[StorageBucket](./)

# StorageBucket

[jvm]\
class [StorageBucket](./)(http: [FluxbaseHttpClient](../../iogithubnimblefluxfluxbasecore/-fluxbase-http-client/), bucket: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html))

Operations on a single storage bucket. Port of `StorageBucket` from `sdk/src/storage.ts`.

## Constructors

| | |
|---|---|
| [StorageBucket](-storage-bucket/) | [jvm]<br>constructor(http: [FluxbaseHttpClient](../../iogithubnimblefluxfluxbasecore/-fluxbase-http-client/), bucket: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)) |

## Functions

| Name | Summary |
|---|---|
| [copy](copy/) | [jvm]<br>suspend fun [copy](copy/)(fromPath: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), toPath: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[Unit](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-unit/index.html)&gt;<br>Copy a file. POSTs `/api/v1/storage/{bucket}/copy`. Port of `copy()` in `storage.ts`. |
| [createSignedUrl](create-signed-url/) | [jvm]<br>suspend fun [createSignedUrl](create-signed-url/)(path: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), expiresIn: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html) = 3600): [FluxbaseResponse](../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[SignedUrlResult](../-signed-url-result/)&gt;<br>Create a signed URL for temporary access. POSTs `/api/v1/storage/{bucket}/sign/{path}`. Port of `createSignedUrl()` in `storage.ts:1261`. |
| [download](download/) | [jvm]<br>suspend fun [download](download/)(path: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[ByteArray](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-byte-array/index.html)&gt;<br>Download a file as raw bytes. GETs `/api/v1/storage/{bucket}/{path}`. Port of `download()` in `storage.ts:368`. |
| [getPublicUrl](get-public-url/) | [jvm]<br>fun [getPublicUrl](get-public-url/)(path: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)<br>Get the public URL for a file (if the bucket is public). Port of `getPublicUrl()` in `storage.ts`. |
| [list](list/) | [jvm]<br>suspend fun [list](list/)(prefix: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, limit: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html)? = null, offset: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html)? = null): [FluxbaseResponse](../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[FileObject](../-file-object/)&gt;&gt;<br>List files in the bucket (or a prefix). GETs `/api/v1/storage/{bucket}`. Port of `list()` in `storage.ts:1040`. |
| [move](move/) | [jvm]<br>suspend fun [move](move/)(fromPath: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), toPath: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[Unit](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-unit/index.html)&gt;<br>Move/rename a file. POSTs `/api/v1/storage/{bucket}/move`. Port of `move()` in `storage.ts`. |
| [remove](remove/) | [jvm]<br>suspend fun [remove](remove/)(path: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[Unit](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-unit/index.html)&gt;<br>Delete files. DELETEs `/api/v1/storage/{bucket}/{path}`. Port of `remove()` in `storage.ts:1099`. |
| [upload](upload/) | [jvm]<br>suspend fun [upload](upload/)(path: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), data: [ByteArray](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-byte-array/index.html), contentType: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;application/octet-stream&quot;, upsert: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = false, metadata: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt;? = null): [FluxbaseResponse](../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[UploadResult](../-upload-result/)&gt;<br>Upload a file (simple upload via POST multipart). Port of `upload()` in `storage.ts:41`. |
