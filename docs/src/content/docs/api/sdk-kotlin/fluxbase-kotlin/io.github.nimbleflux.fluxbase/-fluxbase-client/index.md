//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase](../index.md)/[FluxbaseClient](index.md)

# FluxbaseClient

[jvm]\
class [FluxbaseClient](index.md)

The top-level Fluxbase client — port of `FluxbaseClient` from `sdk/src/client.ts`.

Wires together the HTTP layer ([http](http.md)) and all sub-modules ([auth](auth.md), and later postgrest, realtime, storage, functions, jobs, etc.). Construct via the companion [create](-companion/create.md) factory (or the top-level [createFluxbaseClient](../create-fluxbase-client.md) function), which resolves the URL and key from arguments or environment variables.

Usage:

```kotlin
val client = FluxbaseClient.create("https://flux.example.com", anonKey)
val (session, error) = client.auth.signInWithPassword("user@example.com", "pw")
```

## Types

| Name | Summary |
|---|---|
| [Companion](-companion/index.md) | [jvm]<br>object [Companion](-companion/index.md) |

## Properties

| Name | Summary |
|---|---|
| [auth](auth.md) | [jvm]<br>val [auth](auth.md): [FluxbaseAuth](../../io.github.nimbleflux.fluxbase.auth/-fluxbase-auth/index.md)<br>The auth module. |
| [http](http.md) | [jvm]<br>val [http](http.md): [FluxbaseHttpClient](../../io.github.nimbleflux.fluxbase.core/-fluxbase-http-client/index.md)<br>The shared HTTP client used by all modules. |

## Functions

| Name | Summary |
|---|---|
| [setTenant](set-tenant.md) | [jvm]<br>fun [setTenant](set-tenant.md)(tenantId: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html))<br>Set the active tenant for multi-tenancy. Sets the `X-FB-Tenant` header. Port of `setTenant()` in `client.ts:567-574`. |
