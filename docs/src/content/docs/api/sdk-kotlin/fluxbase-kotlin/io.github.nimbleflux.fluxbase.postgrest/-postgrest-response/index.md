---
title: "PostgrestResponse"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../)/[io.github.nimbleflux.fluxbase.postgrest](../)/[PostgrestResponse](./)

# PostgrestResponse

[jvm]\
data class [PostgrestResponse](./)&lt;[T](./)&gt;(val data: [T](./)?, val error: [FluxbaseError](../../iogithubnimblefluxfluxbase/-fluxbase-error/)?, val count: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html)?, val status: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html), val statusText: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html))

Result of a PostgREST query. Port of `PostgrestResponse<T>` from `sdk/src/types.ts:257`.

## Constructors

| | |
|---|---|
| [PostgrestResponse](-postgrest-response/) | [jvm]<br>constructor(data: [T](./)?, error: [FluxbaseError](../../iogithubnimblefluxfluxbase/-fluxbase-error/)?, count: [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html)?, status: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html), statusText: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)) |

## Properties

| Name | Summary |
|---|---|
| [count](count/) | [jvm]<br>val [count](count/): [Long](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-long/index.html)? |
| [data](data/) | [jvm]<br>val [data](data/): [T](./)? |
| [error](error/) | [jvm]<br>val [error](error/): [FluxbaseError](../../iogithubnimblefluxfluxbase/-fluxbase-error/)? |
| [status](status/) | [jvm]<br>val [status](status/): [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html) |
| [statusText](status-text/) | [jvm]<br>val [statusText](status-text/): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) |
