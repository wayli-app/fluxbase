---
title: "Success"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../../)/[io.github.nimbleflux.fluxbase](../../)/[FluxbaseResponse](../)/[Success](./)

# Success

[jvm]\
data class [Success](./)&lt;[T](./)&gt;(val data: [T](./)) : [FluxbaseResponse](../)&lt;[T](./)&gt; 

Successful result.

## Constructors

| | |
|---|---|
| [Success](-success/) | [jvm]<br>constructor(data: [T](./)) |

## Properties

| Name | Summary |
|---|---|
| [data](data/) | [jvm]<br>open override val [data](data/): [T](./)<br>The data payload on success, or null on error. In the TS SDK this is the `data` field. |
| [error](error/) | [jvm]<br>open override val [error](error/): [FluxbaseError](../../-fluxbase-error/)?<br>The error on failure. Null on success. |

## Functions

| Name | Summary |
|---|---|
| [component2](../component2/) | [jvm]<br>open operator fun [component2](../component2/)(): [FluxbaseError](../../-fluxbase-error/)? |
| [getOrNull](../../get-or-null/) | [jvm]<br>fun &lt;[T](../../get-or-null/)&gt; [FluxbaseResponse](../)&lt;[T](../../get-or-null/)&gt;.[getOrNull](../../get-or-null/)(): [T](../../get-or-null/)?<br>Returns the data on success, or null on error. Equivalent to `result.data` in the TS SDK. |
| [getOrThrow](../../get-or-throw/) | [jvm]<br>fun &lt;[T](../../get-or-throw/)&gt; [FluxbaseResponse](../)&lt;[T](../../get-or-throw/)&gt;.[getOrThrow](../../get-or-throw/)(): [T](../../get-or-throw/)<br>Returns the data on success, or throws the error on failure. |
