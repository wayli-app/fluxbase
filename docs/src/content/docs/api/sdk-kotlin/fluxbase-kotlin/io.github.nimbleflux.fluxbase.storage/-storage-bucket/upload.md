---
title: "upload"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../../)/[io.github.nimbleflux.fluxbase.storage](../../)/[StorageBucket](../)/[upload](./)

# upload

[jvm]\
suspend fun [upload](./)(path: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), data: [ByteArray](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-byte-array/index.html), contentType: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;application/octet-stream&quot;, upsert: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = false, metadata: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt;? = null): [FluxbaseResponse](../../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[UploadResult](../../-upload-result/)&gt;

Upload a file (simple upload via POST multipart). Port of `upload()` in `storage.ts:41`.

#### Parameters

jvm

| | |
|---|---|
| path | the object path within the bucket. |
| data | the file content as a byte array. |
| contentType | MIME type (default application/octet-stream). |
| upsert | overwrite if the file already exists. |
