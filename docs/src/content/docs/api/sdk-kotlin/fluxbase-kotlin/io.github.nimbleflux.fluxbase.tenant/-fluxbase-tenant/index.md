//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.tenant](../index.md)/[FluxbaseTenant](index.md)

# FluxbaseTenant

[jvm]\
class [FluxbaseTenant](index.md)(http: [FluxbaseHttpClient](../../io.github.nimbleflux.fluxbase.core/-fluxbase-http-client/index.md))

Multi-tenancy module — port of `FluxbaseTenant` from `sdk/src/tenant.ts`. All endpoints under `/api/v1/admin/tenants`.

Usage:

```kotlin
val (tenant, _) = client.tenant.create(CreateTenantOptions(slug = "acme", name = "Acme Inc"))
client.setTenant(tenant!!.id) // scope subsequent requests to this tenant
```

## Constructors

| | |
|---|---|
| [FluxbaseTenant](-fluxbase-tenant.md) | [jvm]<br>constructor(http: [FluxbaseHttpClient](../../io.github.nimbleflux.fluxbase.core/-fluxbase-http-client/index.md)) |

## Functions

| Name | Summary |
|---|---|
| [create](create.md) | [jvm]<br>suspend fun [create](create.md)(options: [CreateTenantOptions](../-create-tenant-options/index.md)): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[Tenant](../-tenant/index.md)&gt;<br>Create a tenant. POSTs `/api/v1/admin/tenants`. |
| [delete](delete.md) | [jvm]<br>suspend fun [delete](delete.md)(id: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[Unit](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-unit/index.html)&gt;<br>Delete a tenant. DELETEs `/api/v1/admin/tenants/{id}`. |
| [get](get.md) | [jvm]<br>suspend fun [get](get.md)(id: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[Tenant](../-tenant/index.md)&gt;<br>Get a tenant by ID. |
| [list](list.md) | [jvm]<br>suspend fun [list](list.md)(): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[Tenant](../-tenant/index.md)&gt;&gt;<br>List all tenants (instance admin only). |
| [listMine](list-mine.md) | [jvm]<br>suspend fun [listMine](list-mine.md)(): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[Tenant](../-tenant/index.md)&gt;&gt;<br>List tenants the current user belongs to. |
| [migrate](migrate.md) | [jvm]<br>suspend fun [migrate](migrate.md)(id: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;JsonElement&gt;<br>Run pending migrations for a tenant. POSTs `.../migrate`. |
| [update](update.md) | [jvm]<br>suspend fun [update](update.md)(id: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), options: [UpdateTenantOptions](../-update-tenant-options/index.md)): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[Tenant](../-tenant/index.md)&gt;<br>Update a tenant. PATCHes `/api/v1/admin/tenants/{id}`. |
