---
title: "Package-level declarations"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../index.md)/[io.github.nimbleflux.fluxbase.settings](index.md)

# Package-level declarations

## Types

| Name | Summary |
|---|---|
| [FluxbaseSettings](-fluxbase-settings/index.md) | [jvm]<br>class [FluxbaseSettings](-fluxbase-settings/index.md)(http: [FluxbaseHttpClient](../io.github.nimbleflux.fluxbase.core/-fluxbase-http-client/index.md))<br>Settings module — port of the public `SettingsClient` from `sdk/src/settings.ts`. |
| [UserSecretMetadata](-user-secret-metadata/index.md) | [jvm]<br>@Serializable<br>data class [UserSecretMetadata](-user-secret-metadata/index.md)(val key: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), val description: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, val createdAt: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;&quot;, val updatedAt: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;&quot;)<br>A secret metadata entry (for user secrets stored encrypted on the server). Values are never returned — only metadata. |
| [UserSetting](-user-setting/index.md) | [jvm]<br>@Serializable<br>data class [UserSetting](-user-setting/index.md)(val id: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), val key: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), val value: JsonElement, val description: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, val userId: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;&quot;, val createdAt: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;&quot;, val updatedAt: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;&quot;)<br>A non-encrypted user setting. Port of `UserSetting` from `sdk/src/types.ts:1539`. The [value](-user-setting/value.md) is an arbitrary JSON object (the server stores it as JSONB). |
