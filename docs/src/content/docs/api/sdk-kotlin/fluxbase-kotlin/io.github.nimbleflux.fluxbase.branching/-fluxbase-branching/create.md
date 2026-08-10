---
title: "create"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.branching](../index.md)/[FluxbaseBranching](index.md)/[create](create.md)

# create

[jvm]\
suspend fun [create](create.md)(name: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), options: [CreateBranchOptions](../-create-branch-options/index.md) = CreateBranchOptions()): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[Branch](../-branch/index.md)&gt;

Create a new branch. POSTs `/api/v1/admin/branches`.
