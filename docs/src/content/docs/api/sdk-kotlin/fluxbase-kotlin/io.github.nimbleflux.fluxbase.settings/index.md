---
title: "Package-level declarations"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../)/[io.github.nimbleflux.fluxbase.settings](./)

# Package-level declarations

## Types

| Name | Summary |
|---|---|
| [FluxbaseSettings](-fluxbase-settings/) | [jvm]<br>class [FluxbaseSettings](-fluxbase-settings/)(http: [FluxbaseHttpClient](../iogithubnimblefluxfluxbasecore/-fluxbase-http-client/))<br>Settings module — port of the public `SettingsClient` from `sdk/src/settings.ts`. |
| [UserSecretMetadata](-user-secret-metadata/) | [jvm]<br>@Serializable<br>data class [UserSecretMetadata](-user-secret-metadata/)(val key: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), val description: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, val createdAt: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;&quot;, val updatedAt: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;&quot;)<br>A secret metadata entry (for user secrets stored encrypted on the server). Values are never returned — only metadata. |
| [UserSetting](-user-setting/) | [jvm]<br>@Serializable<br>data class [UserSetting](-user-setting/)(val id: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), val key: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), val value: JsonElement, val description: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, val userId: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;&quot;, val createdAt: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;&quot;, val updatedAt: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;&quot;)<br>A non-encrypted user setting. Port of `UserSetting` from `sdk/src/types.ts:1539`. The [value](-user-setting/value/) is an arbitrary JSON object (the server stores it as JSONB). |
