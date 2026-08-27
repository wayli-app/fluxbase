---
title: "download"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../../)/[io.github.nimbleflux.fluxbase.storage](../../)/[StorageBucket](../)/[download](./)

# download

[jvm]\
suspend fun [download](./)(path: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[ByteArray](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-byte-array/index.html)&gt;

Download a file as raw bytes. GETs `/api/v1/storage/{bucket}/{path}`. Port of `download()` in `storage.ts:368`.

Uses the binary-safe [FluxbaseHttpClient.getBytes](../../../iogithubnimblefluxfluxbasecore/-fluxbase-http-client/get-bytes/) path, so non-text payloads (images, archives) survive intact — the response body never passes through a charset decode.
