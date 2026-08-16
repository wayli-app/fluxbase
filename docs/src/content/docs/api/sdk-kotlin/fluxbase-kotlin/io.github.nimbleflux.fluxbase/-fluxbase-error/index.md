---
title: "FluxbaseError"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase](../index.md)/[FluxbaseError](index.md)

# FluxbaseError

[jvm]\
data class [FluxbaseError](index.md)(val status: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html)? = null, val code: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, val details: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)? = null, val message: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)) : [RuntimeException](https://docs.oracle.com/javase/8/docs/api/java/lang/RuntimeException.html)

An error from a Fluxbase API call. Port of `FluxbaseError` from `sdk/src/types.ts:235`.

## Constructors

| | |
|---|---|
| [FluxbaseError](-fluxbase-error.md) | [jvm]<br>constructor(status: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html)? = null, code: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, details: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)? = null, message: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)) |

## Properties

| Name | Summary |
|---|---|
| [cause](../../io.github.nimbleflux.fluxbase.core/-fluxbase-exception/index.md#-654012527%2FProperties%2F-1216412040) | [jvm]<br>open val [cause](../../io.github.nimbleflux.fluxbase.core/-fluxbase-exception/index.md#-654012527%2FProperties%2F-1216412040): [Throwable](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-throwable/index.html)? |
| [code](code.md) | [jvm]<br>val [code](code.md): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null |
| [details](details.md) | [jvm]<br>val [details](details.md): [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)? = null |
| [message](message.md) | [jvm]<br>open override val [message](message.md): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) |
| [status](status.md) | [jvm]<br>val [status](status.md): [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html)? = null |

## Functions

| Name | Summary |
|---|---|
| [addSuppressed](../../io.github.nimbleflux.fluxbase.core/-fluxbase-exception/index.md#282858770%2FFunctions%2F-1216412040) | [jvm]<br>fun [addSuppressed](../../io.github.nimbleflux.fluxbase.core/-fluxbase-exception/index.md#282858770%2FFunctions%2F-1216412040)(p0: [Throwable](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-throwable/index.html)) |
| [fillInStackTrace](../../io.github.nimbleflux.fluxbase.core/-fluxbase-exception/index.md#-1102069925%2FFunctions%2F-1216412040) | [jvm]<br>open fun [fillInStackTrace](../../io.github.nimbleflux.fluxbase.core/-fluxbase-exception/index.md#-1102069925%2FFunctions%2F-1216412040)(): [Throwable](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-throwable/index.html) |
| [getLocalizedMessage](../../io.github.nimbleflux.fluxbase.core/-fluxbase-exception/index.md#1043865560%2FFunctions%2F-1216412040) | [jvm]<br>open fun [getLocalizedMessage](../../io.github.nimbleflux.fluxbase.core/-fluxbase-exception/index.md#1043865560%2FFunctions%2F-1216412040)(): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) |
| [getStackTrace](../../io.github.nimbleflux.fluxbase.core/-fluxbase-exception/index.md#2050903719%2FFunctions%2F-1216412040) | [jvm]<br>open fun [getStackTrace](../../io.github.nimbleflux.fluxbase.core/-fluxbase-exception/index.md#2050903719%2FFunctions%2F-1216412040)(): [Array](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-array/index.html)&lt;[StackTraceElement](https://docs.oracle.com/javase/8/docs/api/java/lang/StackTraceElement.html)&gt; |
| [getSuppressed](../../io.github.nimbleflux.fluxbase.core/-fluxbase-exception/index.md#672492560%2FFunctions%2F-1216412040) | [jvm]<br>fun [getSuppressed](../../io.github.nimbleflux.fluxbase.core/-fluxbase-exception/index.md#672492560%2FFunctions%2F-1216412040)(): [Array](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-array/index.html)&lt;[Throwable](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-throwable/index.html)&gt; |
| [initCause](../../io.github.nimbleflux.fluxbase.core/-fluxbase-exception/index.md#-418225042%2FFunctions%2F-1216412040) | [jvm]<br>open fun [initCause](../../io.github.nimbleflux.fluxbase.core/-fluxbase-exception/index.md#-418225042%2FFunctions%2F-1216412040)(p0: [Throwable](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-throwable/index.html)): [Throwable](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-throwable/index.html) |
| [printStackTrace](../../io.github.nimbleflux.fluxbase.core/-fluxbase-exception/index.md#-1769529168%2FFunctions%2F-1216412040) | [jvm]<br>open fun [printStackTrace](../../io.github.nimbleflux.fluxbase.core/-fluxbase-exception/index.md#-1769529168%2FFunctions%2F-1216412040)()<br>open fun [printStackTrace](../../io.github.nimbleflux.fluxbase.core/-fluxbase-exception/index.md#1841853697%2FFunctions%2F-1216412040)(p0: [PrintStream](https://docs.oracle.com/javase/8/docs/api/java/io/PrintStream.html))<br>open fun [printStackTrace](../../io.github.nimbleflux.fluxbase.core/-fluxbase-exception/index.md#1175535278%2FFunctions%2F-1216412040)(p0: [PrintWriter](https://docs.oracle.com/javase/8/docs/api/java/io/PrintWriter.html)) |
| [setStackTrace](../../io.github.nimbleflux.fluxbase.core/-fluxbase-exception/index.md#2135801318%2FFunctions%2F-1216412040) | [jvm]<br>open fun [setStackTrace](../../io.github.nimbleflux.fluxbase.core/-fluxbase-exception/index.md#2135801318%2FFunctions%2F-1216412040)(p0: [Array](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-array/index.html)&lt;[StackTraceElement](https://docs.oracle.com/javase/8/docs/api/java/lang/StackTraceElement.html)&gt;) |
