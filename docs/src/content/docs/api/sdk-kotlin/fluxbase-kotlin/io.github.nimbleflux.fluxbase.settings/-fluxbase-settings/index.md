//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.settings](../index.md)/[FluxbaseSettings](index.md)

# FluxbaseSettings

[jvm]\
class [FluxbaseSettings](index.md)(http: [FluxbaseHttpClient](../../io.github.nimbleflux.fluxbase.core/-fluxbase-http-client/index.md))

Settings module — port of the public `SettingsClient` from `sdk/src/settings.ts`.

Provides read access to app/system settings (RLS-respecting) and CRUD for encrypted user secrets. The TS SDK splits this into multiple admin managers; this public client covers what app clients need: reading settings and managing their own encrypted secrets.

Usage:

```kotlin
val (config, _) = client.settings.get("wayli.public_trips_require_auth")
client.settings.setUserSecret("owntracks_api_key", "key-value", "OwnTracks key")
```

## Constructors

| | |
|---|---|
| [FluxbaseSettings](-fluxbase-settings.md) | [jvm]<br>constructor(http: [FluxbaseHttpClient](../../io.github.nimbleflux.fluxbase.core/-fluxbase-http-client/index.md)) |

## Functions

| Name | Summary |
|---|---|
| [deleteUserSecret](delete-user-secret.md) | [jvm]<br>suspend fun [deleteUserSecret](delete-user-secret.md)(key: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[Unit](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-unit/index.html)&gt;<br>Delete a user secret. DELETEs `/api/v1/settings/secret/{key}`. |
| [get](get.md) | [jvm]<br>suspend fun [get](get.md)(key: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;JsonElement&gt;<br>Get a single setting value by key. GETs `/api/v1/settings/{key}`. |
| [getMany](get-many.md) | [jvm]<br>suspend fun [getMany](get-many.md)(keys: [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt;, prefix: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;JsonElement&gt;<br>Get multiple settings. POSTs `/api/v1/settings/batch`. |
| [listUserSecrets](list-user-secrets.md) | [jvm]<br>suspend fun [listUserSecrets](list-user-secrets.md)(): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[UserSecretMetadata](../-user-secret-metadata/index.md)&gt;&gt;<br>List user secret metadata. GETs `/api/v1/settings/secret`. |
| [setUserSecret](set-user-secret.md) | [jvm]<br>suspend fun [setUserSecret](set-user-secret.md)(key: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), value: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), description: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[Unit](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-unit/index.html)&gt;<br>Set an encrypted user secret. PUTs `/api/v1/settings/secret/{key}`. |
