---
title: "FluxbaseClient"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../)/[io.github.nimbleflux.fluxbase](../)/[FluxbaseClient](./)

# FluxbaseClient

[jvm]\
class [FluxbaseClient](./)

The top-level Fluxbase client — port of `FluxbaseClient` from `sdk/src/client.ts`.

Wires together the HTTP layer ([http](http/)) and all sub-modules ([auth](auth/), and later postgrest, realtime, storage, functions, jobs, etc.). Construct via the companion [create](-companion/create/) factory (or the top-level [createFluxbaseClient](../create-fluxbase-client/) function), which resolves the URL and key from arguments or environment variables.

Usage:

```kotlin
val client = FluxbaseClient.create("https://flux.example.com", anonKey)
val (session, error) = client.auth.signInWithPassword("user@example.com", "pw")
```

## Types

| Name | Summary |
|---|---|
| [Companion](-companion/) | [jvm]<br>object [Companion](-companion/) |

## Properties

| Name | Summary |
|---|---|
| [auth](auth/) | [jvm]<br>val [auth](auth/): [FluxbaseAuth](../../iogithubnimblefluxfluxbaseauth/-fluxbase-auth/)<br>The auth module. |
| [branching](branching/) | [jvm]<br>val [branching](branching/): [FluxbaseBranching](../../iogithubnimblefluxfluxbasebranching/-fluxbase-branching/)<br>Database branching (admin). |
| [functions](functions/) | [jvm]<br>val [functions](functions/): [FluxbaseFunctions](../../iogithubnimblefluxfluxbasefunctions/-fluxbase-functions/)<br>Edge Functions module. |
| [graphql](graphql/) | [jvm]<br>val [graphql](graphql/): [FluxbaseGraphQL](../../iogithubnimblefluxfluxbasegraphql/-fluxbase-graph-q-l/)<br>GraphQL module. |
| [http](http/) | [jvm]<br>val [http](http/): [FluxbaseHttpClient](../../iogithubnimblefluxfluxbasecore/-fluxbase-http-client/)<br>The shared HTTP client used by all modules. |
| [jobs](jobs/) | [jvm]<br>val [jobs](jobs/): [FluxbaseJobs](../../iogithubnimblefluxfluxbasejobs/-fluxbase-jobs/)<br>Background Jobs module. |
| [management](management/) | [jvm]<br>val [management](management/): [FluxbaseManagement](../../iogithubnimblefluxfluxbasemanagement/-fluxbase-management/)<br>Client keys, webhooks, invitations. |
| [rpc](rpc/) | [jvm]<br>val [rpc](rpc/): [FluxbaseRpc](../../iogithubnimblefluxfluxbaserpc/-fluxbase-rpc/)<br>RPC module — namespaced stored procedures. |
| [secrets](secrets/) | [jvm]<br>val [secrets](secrets/): [FluxbaseSecrets](../../iogithubnimblefluxfluxbasesecrets/-fluxbase-secrets/)<br>Encrypted Secrets module. |
| [settings](settings/) | [jvm]<br>val [settings](settings/): [FluxbaseSettings](../../iogithubnimblefluxfluxbasesettings/-fluxbase-settings/)<br>App/system settings + user secrets. |
| [storage](storage/) | [jvm]<br>val [storage](storage/): [FluxbaseStorage](../../iogithubnimblefluxfluxbasestorage/-fluxbase-storage/)<br>Storage module — file upload/download/list. |
| [tenant](tenant/) | [jvm]<br>val [tenant](tenant/): [FluxbaseTenant](../../iogithubnimblefluxfluxbasetenant/-fluxbase-tenant/)<br>Multi-tenancy management (admin). |
| [vector](vector/) | [jvm]<br>val [vector](vector/): [FluxbaseVector](../../iogithubnimblefluxfluxbasevector/-fluxbase-vector/)<br>Vector embedding + similarity search. |

## Functions

| Name | Summary |
|---|---|
| [channel](channel/) | [jvm]<br>fun [channel](channel/)(name: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), transport: [WebSocketTransport](../../iogithubnimblefluxfluxbaserealtime/-web-socket-transport/)? = null): [RealtimeChannel](../../iogithubnimblefluxfluxbaserealtime/-realtime-channel/)<br>Create a realtime channel for postgres_changes/broadcast/presence subscriptions. Port of `channel()` in `client.ts:654-674`. |
| [from](../from/) | [jvm]<br>inline fun &lt;[T](../from/)&gt; [FluxbaseClient](./).[from](../from/)(table: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), schema: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null): [QueryBuilder](../../iogithubnimblefluxfluxbasepostgrest/-query-builder/)&lt;[T](../from/)&gt;<br>Start a PostgREST query against [table](../from/). Uses a reified type parameter so the kotlinx.serialization serializer is resolved at compile time. |
| [setTenant](set-tenant/) | [jvm]<br>fun [setTenant](set-tenant/)(tenantId: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html))<br>Set the active tenant for multi-tenancy. Sets the `X-FB-Tenant` header. Port of `setTenant()` in `client.ts:567-574`. |
