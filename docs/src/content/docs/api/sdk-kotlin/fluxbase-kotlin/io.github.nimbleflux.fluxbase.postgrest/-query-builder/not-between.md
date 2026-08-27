---
title: "notBetween"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../../)/[io.github.nimbleflux.fluxbase.postgrest](../../)/[QueryBuilder](../)/[notBetween](./)

# notBetween

[jvm]\
fun [notBetween](./)(column: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), min: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?, max: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?): [QueryBuilder](../)&lt;[T](../)&gt;

Filter values NOT between [min](./) and [max](./). Adds lt + gt filters.
