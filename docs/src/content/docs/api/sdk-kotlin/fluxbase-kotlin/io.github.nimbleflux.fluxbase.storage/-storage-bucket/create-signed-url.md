---
title: "createSignedUrl"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../../)/[io.github.nimbleflux.fluxbase.storage](../../)/[StorageBucket](../)/[createSignedUrl](./)

# createSignedUrl

[jvm]\
suspend fun [createSignedUrl](./)(path: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), expiresIn: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html) = 3600): [FluxbaseResponse](../../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[SignedUrlResult](../../-signed-url-result/)&gt;

Create a signed URL for temporary access. POSTs `/api/v1/storage/{bucket}/sign/{path}`. Port of `createSignedUrl()` in `storage.ts:1261`.
