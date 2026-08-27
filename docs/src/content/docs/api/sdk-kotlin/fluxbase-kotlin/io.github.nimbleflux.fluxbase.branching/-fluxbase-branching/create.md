---
title: "create"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../../)/[io.github.nimbleflux.fluxbase.branching](../../)/[FluxbaseBranching](../)/[create](./)

# create

[jvm]\
suspend fun [create](./)(name: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), options: [CreateBranchOptions](../../-create-branch-options/) = CreateBranchOptions()): [FluxbaseResponse](../../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[Branch](../../-branch/)&gt;

Create a new branch. POSTs `/api/v1/admin/branches`.
