//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.core](../index.md)/[FluxbaseHttpClient](index.md)/[get](get.md)

# get

[jvm]\
inline suspend fun &lt;[T](get.md)&gt; [get](get.md)(path: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), headers: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt; = emptyMap()): [T](get.md)

GET [path](get.md), deserialize the JSON body to [T](get.md).
