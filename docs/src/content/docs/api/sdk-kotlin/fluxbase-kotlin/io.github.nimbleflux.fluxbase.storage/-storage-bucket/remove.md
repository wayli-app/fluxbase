//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.storage](../index.md)/[StorageBucket](index.md)/[remove](remove.md)

# remove

[jvm]\
suspend fun [remove](remove.md)(path: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[Unit](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-unit/index.html)&gt;

Delete files. DELETEs `/api/v1/storage/{bucket}/{path}`. Port of `remove()` in `storage.ts:1099`.
