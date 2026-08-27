---
title: "FluxbaseBranching"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../)/[io.github.nimbleflux.fluxbase.branching](../)/[FluxbaseBranching](./)

# FluxbaseBranching

[jvm]\
class [FluxbaseBranching](./)(http: [FluxbaseHttpClient](../../iogithubnimblefluxfluxbasecore/-fluxbase-http-client/))

Database branching module — port of `FluxbaseBranching` from `sdk/src/branching.ts`. All endpoints under `/api/v1/admin/branches`.

Usage:

```kotlin
val (branch, _) = client.branching.create("feature-x", CreateBranchOptions(dataCloneMode = "full_clone"))
val ready = client.branching.waitForReady(branch!!.id)
```

## Constructors

| | |
|---|---|
| [FluxbaseBranching](-fluxbase-branching/) | [jvm]<br>constructor(http: [FluxbaseHttpClient](../../iogithubnimblefluxfluxbasecore/-fluxbase-http-client/)) |

## Functions

| Name | Summary |
|---|---|
| [create](create/) | [jvm]<br>suspend fun [create](create/)(name: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), options: [CreateBranchOptions](../-create-branch-options/) = CreateBranchOptions()): [FluxbaseResponse](../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[Branch](../-branch/)&gt;<br>Create a new branch. POSTs `/api/v1/admin/branches`. |
| [delete](delete/) | [jvm]<br>suspend fun [delete](delete/)(idOrSlug: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[Unit](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-unit/index.html)&gt;<br>Delete a branch. DELETEs `/api/v1/admin/branches/{idOrSlug}`. |
| [exists](exists/) | [jvm]<br>suspend fun [exists](exists/)(idOrSlug: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html)<br>Check if a branch exists (returns boolean). |
| [get](get/) | [jvm]<br>suspend fun [get](get/)(idOrSlug: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[Branch](../-branch/)&gt;<br>Get a branch by ID or slug. |
| [list](list/) | [jvm]<br>suspend fun [list](list/)(options: [ListBranchesOptions](../-list-branches-options/) = ListBranchesOptions()): [FluxbaseResponse](../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[ListBranchesResponse](../-list-branches-response/)&gt;<br>List branches. GETs `/api/v1/admin/branches`. |
| [reset](reset/) | [jvm]<br>suspend fun [reset](reset/)(idOrSlug: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[Branch](../-branch/)&gt;<br>Reset a branch to its parent state. POSTs `.../reset`. |
