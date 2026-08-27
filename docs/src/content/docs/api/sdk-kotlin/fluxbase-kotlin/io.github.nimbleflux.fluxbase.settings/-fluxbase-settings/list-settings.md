---
title: "listSettings"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../../)/[io.github.nimbleflux.fluxbase.settings](../../)/[FluxbaseSettings](../)/[listSettings](./)

# listSettings

[jvm]\
suspend fun [listSettings](./)(): [FluxbaseResponse](../../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[UserSetting](../../-user-setting/)&gt;&gt;

List the current user's own (non-encrypted) settings. System defaults are not included. GETs `/api/v1/settings/user/list`. Port of `listSettings()` in `settings.ts:1715`.
