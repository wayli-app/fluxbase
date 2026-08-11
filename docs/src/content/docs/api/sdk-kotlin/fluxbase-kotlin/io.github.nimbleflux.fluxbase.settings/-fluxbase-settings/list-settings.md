---
title: "listSettings"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.settings](../index.md)/[FluxbaseSettings](index.md)/[listSettings](list-settings.md)

# listSettings

[jvm]\
suspend fun [listSettings](list-settings.md)(): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[UserSetting](../-user-setting/index.md)&gt;&gt;

List the current user's own (non-encrypted) settings. System defaults are not included. GETs `/api/v1/settings/user/list`. Port of `listSettings()` in `settings.ts:1715`.
