---
title: "remove"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../../)/[io.github.nimbleflux.fluxbase.storage](../../)/[StorageBucket](../)/[remove](./)

# remove

[jvm]\
suspend fun [remove](./)(path: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[Unit](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-unit/index.html)&gt;

Delete files. DELETEs `/api/v1/storage/{bucket}/{path}`. Port of `remove()` in `storage.ts:1099`.
