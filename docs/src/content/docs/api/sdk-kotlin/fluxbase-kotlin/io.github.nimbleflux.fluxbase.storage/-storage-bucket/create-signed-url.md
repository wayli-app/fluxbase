//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.storage](../index.md)/[StorageBucket](index.md)/[createSignedUrl](create-signed-url.md)

# createSignedUrl

[jvm]\
suspend fun [createSignedUrl](create-signed-url.md)(path: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), expiresIn: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html) = 3600): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[SignedUrlResult](../-signed-url-result/index.md)&gt;

Create a signed URL for temporary access. POSTs `/api/v1/storage/{bucket}/sign/{path}`. Port of `createSignedUrl()` in `storage.ts:1261`.
