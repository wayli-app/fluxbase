//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.core](../index.md)/[FluxbaseHttpClient](index.md)/[patch](patch.md)

# patch

[jvm]\
inline suspend fun &lt;[T](patch.md)&gt; [patch](patch.md)(path: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), body: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)? = null, headers: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt; = emptyMap()): [T](patch.md)

PATCH [body](patch.md) to [path](patch.md), deserialize the JSON response to [T](patch.md).
