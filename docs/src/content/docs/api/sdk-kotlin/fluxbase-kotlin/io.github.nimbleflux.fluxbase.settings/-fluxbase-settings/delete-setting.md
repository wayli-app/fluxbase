---
title: "deleteSetting"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.settings](../index.md)/[FluxbaseSettings](index.md)/[deleteSetting](delete-setting.md)

# deleteSetting

[jvm]\
suspend fun [deleteSetting](delete-setting.md)(key: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[Unit](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-unit/index.html)&gt;

Delete one of the current user's own settings, reverting to the system default (if any). DELETEs `/api/v1/settings/user/{key}`. Port of `deleteSetting()` in `settings.ts:1733`.
