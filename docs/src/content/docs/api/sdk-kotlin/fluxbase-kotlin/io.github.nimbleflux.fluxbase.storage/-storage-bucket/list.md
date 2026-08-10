//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.storage](../index.md)/[StorageBucket](index.md)/[list](list.md)

# list

[jvm]\
suspend fun [list](list.md)(prefix: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, limit: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html)? = null, offset: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html)? = null): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[FileObject](../-file-object/index.md)&gt;&gt;

List files in the bucket (or a prefix). GETs `/api/v1/storage/{bucket}`. Port of `list()` in `storage.ts:1040`.
