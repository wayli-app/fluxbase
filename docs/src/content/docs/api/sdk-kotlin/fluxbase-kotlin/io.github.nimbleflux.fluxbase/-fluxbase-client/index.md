---
title: "FluxbaseClient"
editUrl: false
next: false
prev: false
---

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
| [branching](branching.md) | [jvm]<br>val [branching](branching.md): [FluxbaseBranching](../../io.github.nimbleflux.fluxbase.branching/-fluxbase-branching/index.md)<br>Database branching (admin). |
| [functions](functions.md) | [jvm]<br>val [functions](functions.md): [FluxbaseFunctions](../../io.github.nimbleflux.fluxbase.functions/-fluxbase-functions/index.md)<br>Edge Functions module. |
| [graphql](graphql.md) | [jvm]<br>val [graphql](graphql.md): [FluxbaseGraphQL](../../io.github.nimbleflux.fluxbase.graphql/-fluxbase-graph-q-l/index.md)<br>GraphQL module. |
| [http](http.md) | [jvm]<br>val [http](http.md): [FluxbaseHttpClient](../../io.github.nimbleflux.fluxbase.core/-fluxbase-http-client/index.md)<br>The shared HTTP client used by all modules. |
| [jobs](jobs.md) | [jvm]<br>val [jobs](jobs.md): [FluxbaseJobs](../../io.github.nimbleflux.fluxbase.jobs/-fluxbase-jobs/index.md)<br>Background Jobs module. |
| [management](management.md) | [jvm]<br>val [management](management.md): [FluxbaseManagement](../../io.github.nimbleflux.fluxbase.management/-fluxbase-management/index.md)<br>Client keys, webhooks, invitations. |
| [rpc](rpc.md) | [jvm]<br>val [rpc](rpc.md): [FluxbaseRpc](../../io.github.nimbleflux.fluxbase.rpc/-fluxbase-rpc/index.md)<br>RPC module — namespaced stored procedures. |
| [secrets](secrets.md) | [jvm]<br>val [secrets](secrets.md): [FluxbaseSecrets](../../io.github.nimbleflux.fluxbase.secrets/-fluxbase-secrets/index.md)<br>Encrypted Secrets module. |
| [settings](settings.md) | [jvm]<br>val [settings](settings.md): [FluxbaseSettings](../../io.github.nimbleflux.fluxbase.settings/-fluxbase-settings/index.md)<br>App/system settings + user secrets. |
| [storage](storage.md) | [jvm]<br>val [storage](storage.md): [FluxbaseStorage](../../io.github.nimbleflux.fluxbase.storage/-fluxbase-storage/index.md)<br>Storage module — file upload/download/list. |
| [tenant](tenant.md) | [jvm]<br>val [tenant](tenant.md): [FluxbaseTenant](../../io.github.nimbleflux.fluxbase.tenant/-fluxbase-tenant/index.md)<br>Multi-tenancy management (admin). |
| [vector](vector.md) | [jvm]<br>val [vector](vector.md): [FluxbaseVector](../../io.github.nimbleflux.fluxbase.vector/-fluxbase-vector/index.md)<br>Vector embedding + similarity search. |

## Functions

| Name | Summary |
|---|---|
| [channel](channel.md) | [jvm]<br>fun [channel](channel.md)(name: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), transport: [WebSocketTransport](../../io.github.nimbleflux.fluxbase.realtime/-web-socket-transport/index.md)? = null): [RealtimeChannel](../../io.github.nimbleflux.fluxbase.realtime/-realtime-channel/index.md)<br>Create a realtime channel for postgres_changes/broadcast/presence subscriptions. Port of `channel()` in `client.ts:654-674`. |
| [from](../from.md) | [jvm]<br>inline fun &lt;[T](../from.md)&gt; [FluxbaseClient](index.md).[from](../from.md)(table: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), schema: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null): [QueryBuilder](../../io.github.nimbleflux.fluxbase.postgrest/-query-builder/index.md)&lt;[T](../from.md)&gt;<br>Start a PostgREST query against [table](../from.md). Uses a reified type parameter so the kotlinx.serialization serializer is resolved at compile time. |
| [setTenant](set-tenant.md) | [jvm]<br>fun [setTenant](set-tenant.md)(tenantId: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html))<br>Set the active tenant for multi-tenancy. Sets the `X-FB-Tenant` header. Port of `setTenant()` in `client.ts:567-574`. |
