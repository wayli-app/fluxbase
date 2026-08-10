---
title: "Success"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../../index.md)/[io.github.nimbleflux.fluxbase](../../index.md)/[FluxbaseResponse](../index.md)/[Success](index.md)

# Success

[jvm]\
data class [Success](index.md)&lt;[T](index.md)&gt;(val data: [T](index.md)) : [FluxbaseResponse](../index.md)&lt;[T](index.md)&gt; 

Successful result.

## Constructors

| | |
|---|---|
| [Success](-success.md) | [jvm]<br>constructor(data: [T](index.md)) |

## Properties

| Name | Summary |
|---|---|
| [data](data.md) | [jvm]<br>open override val [data](data.md): [T](index.md)<br>The data payload on success, or null on error. In the TS SDK this is the `data` field. |
| [error](error.md) | [jvm]<br>open override val [error](error.md): [FluxbaseError](../../-fluxbase-error/index.md)?<br>The error on failure. Null on success. |

## Functions

| Name | Summary |
|---|---|
| [component2](../component2.md) | [jvm]<br>open operator fun [component2](../component2.md)(): [FluxbaseError](../../-fluxbase-error/index.md)? |
| [getOrNull](../../get-or-null.md) | [jvm]<br>fun &lt;[T](../../get-or-null.md)&gt; [FluxbaseResponse](../index.md)&lt;[T](../../get-or-null.md)&gt;.[getOrNull](../../get-or-null.md)(): [T](../../get-or-null.md)?<br>Returns the data on success, or null on error. Equivalent to `result.data` in the TS SDK. |
| [getOrThrow](../../get-or-throw.md) | [jvm]<br>fun &lt;[T](../../get-or-throw.md)&gt; [FluxbaseResponse](../index.md)&lt;[T](../../get-or-throw.md)&gt;.[getOrThrow](../../get-or-throw.md)(): [T](../../get-or-throw.md)<br>Returns the data on success, or throws the error on failure. |
