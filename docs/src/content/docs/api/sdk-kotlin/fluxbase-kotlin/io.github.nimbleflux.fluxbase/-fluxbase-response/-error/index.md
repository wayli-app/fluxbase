---
title: "Error"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../../index.md)/[io.github.nimbleflux.fluxbase](../../index.md)/[FluxbaseResponse](../index.md)/[Error](index.md)

# Error

[jvm]\
data class [Error](index.md)(val error: [FluxbaseError](../../-fluxbase-error/index.md)) : [FluxbaseResponse](../index.md)&lt;[Nothing](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-nothing/index.html)&gt; 

Failed result.

## Constructors

| | |
|---|---|
| [Error](-error.md) | [jvm]<br>constructor(error: [FluxbaseError](../../-fluxbase-error/index.md)) |

## Properties

| Name | Summary |
|---|---|
| [data](data.md) | [jvm]<br>open override val [data](data.md): [Nothing](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-nothing/index.html)?<br>The data payload on success, or null on error. In the TS SDK this is the `data` field. |
| [error](error.md) | [jvm]<br>open override val [error](error.md): [FluxbaseError](../../-fluxbase-error/index.md)<br>The error on failure. Null on success. |

## Functions

| Name | Summary |
|---|---|
| [component2](../component2.md) | [jvm]<br>open operator fun [component2](../component2.md)(): [FluxbaseError](../../-fluxbase-error/index.md)? |
| [getOrNull](../../get-or-null.md) | [jvm]<br>fun &lt;[T](../../get-or-null.md)&gt; [FluxbaseResponse](../index.md)&lt;[T](../../get-or-null.md)&gt;.[getOrNull](../../get-or-null.md)(): [T](../../get-or-null.md)?<br>Returns the data on success, or null on error. Equivalent to `result.data` in the TS SDK. |
| [getOrThrow](../../get-or-throw.md) | [jvm]<br>fun &lt;[T](../../get-or-throw.md)&gt; [FluxbaseResponse](../index.md)&lt;[T](../../get-or-throw.md)&gt;.[getOrThrow](../../get-or-throw.md)(): [T](../../get-or-throw.md)<br>Returns the data on success, or throws the error on failure. |
