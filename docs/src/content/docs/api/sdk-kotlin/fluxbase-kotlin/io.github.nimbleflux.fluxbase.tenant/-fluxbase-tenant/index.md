---
title: "FluxbaseTenant"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../)/[io.github.nimbleflux.fluxbase.tenant](../)/[FluxbaseTenant](./)

# FluxbaseTenant

[jvm]\
class [FluxbaseTenant](./)(http: [FluxbaseHttpClient](../../iogithubnimblefluxfluxbasecore/-fluxbase-http-client/))

Multi-tenancy module — port of `FluxbaseTenant` from `sdk/src/tenant.ts`. All endpoints under `/api/v1/admin/tenants`.

Usage:

```kotlin
val (tenant, _) = client.tenant.create(CreateTenantOptions(slug = "acme", name = "Acme Inc"))
client.setTenant(tenant!!.id) // scope subsequent requests to this tenant
```

## Constructors

| | |
|---|---|
| [FluxbaseTenant](-fluxbase-tenant/) | [jvm]<br>constructor(http: [FluxbaseHttpClient](../../iogithubnimblefluxfluxbasecore/-fluxbase-http-client/)) |

## Functions

| Name | Summary |
|---|---|
| [create](create/) | [jvm]<br>suspend fun [create](create/)(options: [CreateTenantOptions](../-create-tenant-options/)): [FluxbaseResponse](../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[Tenant](../-tenant/)&gt;<br>Create a tenant. POSTs `/api/v1/admin/tenants`. |
| [delete](delete/) | [jvm]<br>suspend fun [delete](delete/)(id: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[Unit](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-unit/index.html)&gt;<br>Delete a tenant. DELETEs `/api/v1/admin/tenants/{id}`. |
| [get](get/) | [jvm]<br>suspend fun [get](get/)(id: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[Tenant](../-tenant/)&gt;<br>Get a tenant by ID. |
| [list](list/) | [jvm]<br>suspend fun [list](list/)(): [FluxbaseResponse](../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[Tenant](../-tenant/)&gt;&gt;<br>List all tenants (instance admin only). |
| [listMine](list-mine/) | [jvm]<br>suspend fun [listMine](list-mine/)(): [FluxbaseResponse](../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[Tenant](../-tenant/)&gt;&gt;<br>List tenants the current user belongs to. |
| [migrate](migrate/) | [jvm]<br>suspend fun [migrate](migrate/)(id: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;JsonElement&gt;<br>Run pending migrations for a tenant. POSTs `.../migrate`. |
| [update](update/) | [jvm]<br>suspend fun [update](update/)(id: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), options: [UpdateTenantOptions](../-update-tenant-options/)): [FluxbaseResponse](../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[Tenant](../-tenant/)&gt;<br>Update a tenant. PATCHes `/api/v1/admin/tenants/{id}`. |
