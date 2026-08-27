---
title: "between"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../../)/[io.github.nimbleflux.fluxbase.postgrest](../../)/[QueryBuilder](../)/[between](./)

# between

[jvm]\
fun [between](./)(column: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), min: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?, max: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)?): [QueryBuilder](../)&lt;[T](../)&gt;

Filter values between [min](./) and [max](./) (inclusive). Adds gte + lte filters.
