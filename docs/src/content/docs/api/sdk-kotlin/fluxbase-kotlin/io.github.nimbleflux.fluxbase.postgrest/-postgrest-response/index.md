---
title: "PostgrestResponse"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.postgrest](../index.md)/[PostgrestResponse](index.md)

# PostgrestResponse

[jvm]\
data class [PostgrestResponse](index.md)&lt;[T](index.md)&gt;(val data: [T](index.md)?, val error: [FluxbaseError](../../io.github.nimbleflux.fluxbase/-fluxbase-error/index.md)?, val count: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html)?, val status: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html), val statusText: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html))

Result of a PostgREST query. Port of `PostgrestResponse<T>` from `sdk/src/types.ts:257`.

## Constructors

| | |
|---|---|
| [PostgrestResponse](-postgrest-response.md) | [jvm]<br>constructor(data: [T](index.md)?, error: [FluxbaseError](../../io.github.nimbleflux.fluxbase/-fluxbase-error/index.md)?, count: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html)?, status: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html), statusText: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)) |

## Properties

| Name | Summary |
|---|---|
| [count](count.md) | [jvm]<br>val [count](count.md): [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html)? |
| [data](data.md) | [jvm]<br>val [data](data.md): [T](index.md)? |
| [error](error.md) | [jvm]<br>val [error](error.md): [FluxbaseError](../../io.github.nimbleflux.fluxbase/-fluxbase-error/index.md)? |
| [status](status.md) | [jvm]<br>val [status](status.md): [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html) |
| [statusText](status-text.md) | [jvm]<br>val [statusText](status-text.md): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) |
