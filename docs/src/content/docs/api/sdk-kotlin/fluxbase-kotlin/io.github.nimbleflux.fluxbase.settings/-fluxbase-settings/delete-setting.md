---
title: "deleteSetting"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../../)/[io.github.nimbleflux.fluxbase.settings](../../)/[FluxbaseSettings](../)/[deleteSetting](./)

# deleteSetting

[jvm]\
suspend fun [deleteSetting](./)(key: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[Unit](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-unit/index.html)&gt;

Delete one of the current user's own settings, reverting to the system default (if any). DELETEs `/api/v1/settings/user/{key}`. Port of `deleteSetting()` in `settings.ts:1733`.
