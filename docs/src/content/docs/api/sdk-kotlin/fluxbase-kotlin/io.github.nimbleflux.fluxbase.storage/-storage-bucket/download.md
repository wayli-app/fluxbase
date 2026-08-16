---
title: "download"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.storage](../index.md)/[StorageBucket](index.md)/[download](download.md)

# download

[jvm]\
suspend fun [download](download.md)(path: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[ByteArray](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-byte-array/index.html)&gt;

Download a file as raw bytes. GETs `/api/v1/storage/{bucket}/{path}`. Port of `download()` in `storage.ts:368`.

Uses the binary-safe [FluxbaseHttpClient.getBytes](../../io.github.nimbleflux.fluxbase.core/-fluxbase-http-client/get-bytes.md) path, so non-text payloads (images, archives) survive intact — the response body never passes through a charset decode.
