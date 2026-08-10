//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.postgrest](../index.md)/[QueryBuilder](index.md)/[notBetween](not-between.md)

# notBetween

[jvm]\
fun [notBetween](not-between.md)(column: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), min: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?, max: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?): [QueryBuilder](index.md)&lt;[T](index.md)&gt;

Filter values NOT between [min](not-between.md) and [max](not-between.md). Adds lt + gt filters.
