---
title: "update"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../../)/[io.github.nimbleflux.fluxbase.tenant](../../)/[FluxbaseTenant](../)/[update](./)

# update

[jvm]\
suspend fun [update](./)(id: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), options: [UpdateTenantOptions](../../-update-tenant-options/)): [FluxbaseResponse](../../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[Tenant](../../-tenant/)&gt;

Update a tenant. PATCHes `/api/v1/admin/tenants/{id}`.
