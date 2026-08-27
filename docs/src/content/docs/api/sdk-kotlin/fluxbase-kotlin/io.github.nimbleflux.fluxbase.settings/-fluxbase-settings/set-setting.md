---
title: "setSetting"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../../)/[io.github.nimbleflux.fluxbase.settings](../../)/[FluxbaseSettings](../)/[setSetting](./)

# setSetting

[jvm]\
suspend fun [setSetting](./)(key: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), value: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?&gt;, description: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null): [FluxbaseResponse](../../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[UserSetting](../../-user-setting/)&gt;

Set (create or update) one of the current user's own settings. PUTs `/api/v1/settings/user/{key}` with `{ value, description }`. Port of `setSetting()` in `settings.ts:1687`.

#### Parameters

jvm

| | |
|---|---|
| value | a JSON object (mirrors the TS `Record<string, unknown>` value). |
| description | optional human-readable note. |
