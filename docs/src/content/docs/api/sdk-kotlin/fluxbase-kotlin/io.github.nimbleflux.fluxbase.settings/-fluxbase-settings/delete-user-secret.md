---
title: "deleteUserSecret"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.settings](../index.md)/[FluxbaseSettings](index.md)/[deleteUserSecret](delete-user-secret.md)

# deleteUserSecret

[jvm]\
suspend fun [deleteUserSecret](delete-user-secret.md)(key: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[Unit](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-unit/index.html)&gt;

Delete a user secret. DELETEs `/api/v1/settings/secret/{key}`.
