//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.storage](../index.md)/[StorageBucket](index.md)/[upload](upload.md)

# upload

[jvm]\
suspend fun [upload](upload.md)(path: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), data: [ByteArray](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-byte-array/index.html), contentType: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;application/octet-stream&quot;, upsert: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = false, metadata: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt;? = null): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[UploadResult](../-upload-result/index.md)&gt;

Upload a file (simple upload via POST multipart). Port of `upload()` in `storage.ts:41`.

#### Parameters

jvm

| | |
|---|---|
| path | the object path within the bucket. |
| data | the file content as a byte array. |
| contentType | MIME type (default application/octet-stream). |
| upsert | overwrite if the file already exists. |
