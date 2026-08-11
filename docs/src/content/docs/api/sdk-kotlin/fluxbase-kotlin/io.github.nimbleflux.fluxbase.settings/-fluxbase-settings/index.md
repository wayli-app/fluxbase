---
title: "FluxbaseSettings"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.settings](../index.md)/[FluxbaseSettings](index.md)

# FluxbaseSettings

[jvm]\
class [FluxbaseSettings](index.md)(http: [FluxbaseHttpClient](../../io.github.nimbleflux.fluxbase.core/-fluxbase-http-client/index.md))

Settings module — port of the public `SettingsClient` from `sdk/src/settings.ts`.

Provides read access to app/system settings (RLS-respecting), write/list/delete for the current user's own settings, and CRUD for encrypted user secrets. The TS SDK splits admin-side settings into multiple managers; this public client covers what app clients need: reading any visible setting, managing their own settings, and managing their own encrypted secrets.

Usage:

```kotlin
val (config, _) = client.settings.get("wayli.public_trips_require_auth")
client.settings.setSetting("wayli.pexels_rate_limit", mapOf("limit" to 100))
client.settings.setUserSecret("owntracks_api_key", "key-value", "OwnTracks key")
```

## Constructors

| | |
|---|---|
| [FluxbaseSettings](-fluxbase-settings.md) | [jvm]<br>constructor(http: [FluxbaseHttpClient](../../io.github.nimbleflux.fluxbase.core/-fluxbase-http-client/index.md)) |

## Functions

| Name | Summary |
|---|---|
| [deleteSetting](delete-setting.md) | [jvm]<br>suspend fun [deleteSetting](delete-setting.md)(key: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[Unit](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-unit/index.html)&gt;<br>Delete one of the current user's own settings, reverting to the system default (if any). DELETEs `/api/v1/settings/user/{key}`. Port of `deleteSetting()` in `settings.ts:1733`. |
| [deleteUserSecret](delete-user-secret.md) | [jvm]<br>suspend fun [deleteUserSecret](delete-user-secret.md)(key: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[Unit](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-unit/index.html)&gt;<br>Delete a user secret. DELETEs `/api/v1/settings/secret/{key}`. |
| [get](get.md) | [jvm]<br>suspend fun [get](get.md)(key: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;JsonElement&gt;<br>Get a single setting value by key. GETs `/api/v1/settings/{key}`. |
| [getMany](get-many.md) | [jvm]<br>suspend fun [getMany](get-many.md)(keys: [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt;, prefix: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;JsonElement&gt;<br>Get multiple settings. POSTs `/api/v1/settings/batch`. |
| [listSettings](list-settings.md) | [jvm]<br>suspend fun [listSettings](list-settings.md)(): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[UserSetting](../-user-setting/index.md)&gt;&gt;<br>List the current user's own (non-encrypted) settings. System defaults are not included. GETs `/api/v1/settings/user/list`. Port of `listSettings()` in `settings.ts:1715`. |
| [listUserSecrets](list-user-secrets.md) | [jvm]<br>suspend fun [listUserSecrets](list-user-secrets.md)(): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[UserSecretMetadata](../-user-secret-metadata/index.md)&gt;&gt;<br>List user secret metadata. GETs `/api/v1/settings/secret`. |
| [setSetting](set-setting.md) | [jvm]<br>suspend fun [setSetting](set-setting.md)(key: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), value: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?&gt;, description: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[UserSetting](../-user-setting/index.md)&gt;<br>Set (create or update) one of the current user's own settings. PUTs `/api/v1/settings/user/{key}` with `{ value, description }`. Port of `setSetting()` in `settings.ts:1687`. |
| [setUserSecret](set-user-secret.md) | [jvm]<br>suspend fun [setUserSecret](set-user-secret.md)(key: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), value: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), description: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[Unit](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-unit/index.html)&gt;<br>Set an encrypted user secret. PUTs `/api/v1/settings/secret/{key}`. |
