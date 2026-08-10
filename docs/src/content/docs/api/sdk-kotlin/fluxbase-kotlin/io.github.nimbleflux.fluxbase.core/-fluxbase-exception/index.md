//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.core](../index.md)/[FluxbaseException](index.md)

# FluxbaseException

[jvm]\
class [FluxbaseException](index.md)(val status: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html), val code: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, val details: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)? = null, message: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)) : [RuntimeException](https://docs.oracle.com/javase/8/docs/api/java/lang/RuntimeException.html)

A Fluxbase API error. Mirrors `FluxbaseError` from `sdk/src/types.ts:235`.

Thrown by the transport when a response is not ok (status >= 400). Higher layers (query builder, auth) catch this and convert it into `FluxbaseResponse.Error`.

## Constructors

| | |
|---|---|
| [FluxbaseException](-fluxbase-exception.md) | [jvm]<br>constructor(status: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html), code: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, details: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)? = null, message: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)) |

## Properties

| Name | Summary |
|---|---|
| [cause](index.md#-654012527%2FProperties%2F-1216412040) | [jvm]<br>open val [cause](index.md#-654012527%2FProperties%2F-1216412040): [Throwable](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-throwable/index.html)? |
| [code](code.md) | [jvm]<br>val [code](code.md): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null |
| [details](details.md) | [jvm]<br>val [details](details.md): [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)? = null |
| [message](index.md#1824300659%2FProperties%2F-1216412040) | [jvm]<br>open val [message](index.md#1824300659%2FProperties%2F-1216412040): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? |
| [status](status.md) | [jvm]<br>val [status](status.md): [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html) |

## Functions

| Name | Summary |
|---|---|
| [addSuppressed](index.md#282858770%2FFunctions%2F-1216412040) | [jvm]<br>fun [addSuppressed](index.md#282858770%2FFunctions%2F-1216412040)(p0: [Throwable](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-throwable/index.html)) |
| [fillInStackTrace](index.md#-1102069925%2FFunctions%2F-1216412040) | [jvm]<br>open fun [fillInStackTrace](index.md#-1102069925%2FFunctions%2F-1216412040)(): [Throwable](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-throwable/index.html) |
| [getLocalizedMessage](index.md#1043865560%2FFunctions%2F-1216412040) | [jvm]<br>open fun [getLocalizedMessage](index.md#1043865560%2FFunctions%2F-1216412040)(): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) |
| [getStackTrace](index.md#2050903719%2FFunctions%2F-1216412040) | [jvm]<br>open fun [getStackTrace](index.md#2050903719%2FFunctions%2F-1216412040)(): [Array](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-array/index.html)&lt;[StackTraceElement](https://docs.oracle.com/javase/8/docs/api/java/lang/StackTraceElement.html)&gt; |
| [getSuppressed](index.md#672492560%2FFunctions%2F-1216412040) | [jvm]<br>fun [getSuppressed](index.md#672492560%2FFunctions%2F-1216412040)(): [Array](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-array/index.html)&lt;[Throwable](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-throwable/index.html)&gt; |
| [initCause](index.md#-418225042%2FFunctions%2F-1216412040) | [jvm]<br>open fun [initCause](index.md#-418225042%2FFunctions%2F-1216412040)(p0: [Throwable](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-throwable/index.html)): [Throwable](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-throwable/index.html) |
| [printStackTrace](index.md#-1769529168%2FFunctions%2F-1216412040) | [jvm]<br>open fun [printStackTrace](index.md#-1769529168%2FFunctions%2F-1216412040)()<br>open fun [printStackTrace](index.md#1841853697%2FFunctions%2F-1216412040)(p0: [PrintStream](https://docs.oracle.com/javase/8/docs/api/java/io/PrintStream.html))<br>open fun [printStackTrace](index.md#1175535278%2FFunctions%2F-1216412040)(p0: [PrintWriter](https://docs.oracle.com/javase/8/docs/api/java/io/PrintWriter.html)) |
| [setStackTrace](index.md#2135801318%2FFunctions%2F-1216412040) | [jvm]<br>open fun [setStackTrace](index.md#2135801318%2FFunctions%2F-1216412040)(p0: [Array](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-array/index.html)&lt;[StackTraceElement](https://docs.oracle.com/javase/8/docs/api/java/lang/StackTraceElement.html)&gt;) |
