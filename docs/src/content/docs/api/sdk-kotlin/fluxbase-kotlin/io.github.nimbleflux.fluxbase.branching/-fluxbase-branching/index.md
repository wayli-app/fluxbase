//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.branching](../index.md)/[FluxbaseBranching](index.md)

# FluxbaseBranching

[jvm]\
class [FluxbaseBranching](index.md)(http: [FluxbaseHttpClient](../../io.github.nimbleflux.fluxbase.core/-fluxbase-http-client/index.md))

Database branching module — port of `FluxbaseBranching` from `sdk/src/branching.ts`. All endpoints under `/api/v1/admin/branches`.

Usage:

```kotlin
val (branch, _) = client.branching.create("feature-x", CreateBranchOptions(dataCloneMode = "full_clone"))
val ready = client.branching.waitForReady(branch!!.id)
```

## Constructors

| | |
|---|---|
| [FluxbaseBranching](-fluxbase-branching.md) | [jvm]<br>constructor(http: [FluxbaseHttpClient](../../io.github.nimbleflux.fluxbase.core/-fluxbase-http-client/index.md)) |

## Functions

| Name | Summary |
|---|---|
| [create](create.md) | [jvm]<br>suspend fun [create](create.md)(name: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), options: [CreateBranchOptions](../-create-branch-options/index.md) = CreateBranchOptions()): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[Branch](../-branch/index.md)&gt;<br>Create a new branch. POSTs `/api/v1/admin/branches`. |
| [delete](delete.md) | [jvm]<br>suspend fun [delete](delete.md)(idOrSlug: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[Unit](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-unit/index.html)&gt;<br>Delete a branch. DELETEs `/api/v1/admin/branches/{idOrSlug}`. |
| [exists](exists.md) | [jvm]<br>suspend fun [exists](exists.md)(idOrSlug: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html)<br>Check if a branch exists (returns boolean). |
| [get](get.md) | [jvm]<br>suspend fun [get](get.md)(idOrSlug: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[Branch](../-branch/index.md)&gt;<br>Get a branch by ID or slug. |
| [list](list.md) | [jvm]<br>suspend fun [list](list.md)(options: [ListBranchesOptions](../-list-branches-options/index.md) = ListBranchesOptions()): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[ListBranchesResponse](../-list-branches-response/index.md)&gt;<br>List branches. GETs `/api/v1/admin/branches`. |
| [reset](reset.md) | [jvm]<br>suspend fun [reset](reset.md)(idOrSlug: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[Branch](../-branch/index.md)&gt;<br>Reset a branch to its parent state. POSTs `.../reset`. |
