---
title: "FluxbaseError"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../)/[io.github.nimbleflux.fluxbase](../)/[FluxbaseError](./)

# FluxbaseError

[jvm]\
data class [FluxbaseError](./)(val status: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html)? = null, val code: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, val details: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)? = null, val message: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)) : [RuntimeException](https://docs.oracle.com/javase/8/docs/api/java/lang/RuntimeException.html)

An error from a Fluxbase API call. Port of `FluxbaseError` from `sdk/src/types.ts:235`.

## Constructors

| | |
|---|---|
| [FluxbaseError](-fluxbase-error/) | [jvm]<br>constructor(status: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html)? = null, code: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, details: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)? = null, message: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)) |

## Properties

| Name | Summary |
|---|---|
| [cause](../../iogithubnimblefluxfluxbasecore/-fluxbase-exception/#-654012527%2FProperties%2F-1216412040) | [jvm]<br>open val [cause](../../iogithubnimblefluxfluxbasecore/-fluxbase-exception/#-654012527%2FProperties%2F-1216412040): [Throwable](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-throwable/index.html)? |
| [code](code/) | [jvm]<br>val [code](code/): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null |
| [details](details/) | [jvm]<br>val [details](details/): [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)? = null |
| [message](message/) | [jvm]<br>open override val [message](message/): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) |
| [status](status/) | [jvm]<br>val [status](status/): [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html)? = null |

## Functions

| Name | Summary |
|---|---|
| [addSuppressed](../../iogithubnimblefluxfluxbasecore/-fluxbase-exception/#282858770%2FFunctions%2F-1216412040) | [jvm]<br>fun [addSuppressed](../../iogithubnimblefluxfluxbasecore/-fluxbase-exception/#282858770%2FFunctions%2F-1216412040)(p0: [Throwable](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-throwable/index.html)) |
| [fillInStackTrace](../../iogithubnimblefluxfluxbasecore/-fluxbase-exception/#-1102069925%2FFunctions%2F-1216412040) | [jvm]<br>open fun [fillInStackTrace](../../iogithubnimblefluxfluxbasecore/-fluxbase-exception/#-1102069925%2FFunctions%2F-1216412040)(): [Throwable](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-throwable/index.html) |
| [getLocalizedMessage](../../iogithubnimblefluxfluxbasecore/-fluxbase-exception/#1043865560%2FFunctions%2F-1216412040) | [jvm]<br>open fun [getLocalizedMessage](../../iogithubnimblefluxfluxbasecore/-fluxbase-exception/#1043865560%2FFunctions%2F-1216412040)(): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) |
| [getStackTrace](../../iogithubnimblefluxfluxbasecore/-fluxbase-exception/#2050903719%2FFunctions%2F-1216412040) | [jvm]<br>open fun [getStackTrace](../../iogithubnimblefluxfluxbasecore/-fluxbase-exception/#2050903719%2FFunctions%2F-1216412040)(): [Array](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-array/index.html)&lt;[StackTraceElement](https://docs.oracle.com/javase/8/docs/api/java/lang/StackTraceElement.html)&gt; |
| [getSuppressed](../../iogithubnimblefluxfluxbasecore/-fluxbase-exception/#672492560%2FFunctions%2F-1216412040) | [jvm]<br>fun [getSuppressed](../../iogithubnimblefluxfluxbasecore/-fluxbase-exception/#672492560%2FFunctions%2F-1216412040)(): [Array](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-array/index.html)&lt;[Throwable](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-throwable/index.html)&gt; |
| [initCause](../../iogithubnimblefluxfluxbasecore/-fluxbase-exception/#-418225042%2FFunctions%2F-1216412040) | [jvm]<br>open fun [initCause](../../iogithubnimblefluxfluxbasecore/-fluxbase-exception/#-418225042%2FFunctions%2F-1216412040)(p0: [Throwable](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-throwable/index.html)): [Throwable](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-throwable/index.html) |
| [printStackTrace](../../iogithubnimblefluxfluxbasecore/-fluxbase-exception/#-1769529168%2FFunctions%2F-1216412040) | [jvm]<br>open fun [printStackTrace](../../iogithubnimblefluxfluxbasecore/-fluxbase-exception/#-1769529168%2FFunctions%2F-1216412040)()<br>open fun [printStackTrace](../../iogithubnimblefluxfluxbasecore/-fluxbase-exception/#1841853697%2FFunctions%2F-1216412040)(p0: [PrintStream](https://docs.oracle.com/javase/8/docs/api/java/io/PrintStream.html))<br>open fun [printStackTrace](../../iogithubnimblefluxfluxbasecore/-fluxbase-exception/#1175535278%2FFunctions%2F-1216412040)(p0: [PrintWriter](https://docs.oracle.com/javase/8/docs/api/java/io/PrintWriter.html)) |
| [setStackTrace](../../iogithubnimblefluxfluxbasecore/-fluxbase-exception/#2135801318%2FFunctions%2F-1216412040) | [jvm]<br>open fun [setStackTrace](../../iogithubnimblefluxfluxbasecore/-fluxbase-exception/#2135801318%2FFunctions%2F-1216412040)(p0: [Array](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-array/index.html)&lt;[StackTraceElement](https://docs.oracle.com/javase/8/docs/api/java/lang/StackTraceElement.html)&gt;) |
