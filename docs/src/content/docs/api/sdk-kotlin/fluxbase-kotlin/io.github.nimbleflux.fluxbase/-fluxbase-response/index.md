---
title: "FluxbaseResponse"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase](../index.md)/[FluxbaseResponse](index.md)

# FluxbaseResponse

sealed interface [FluxbaseResponse](index.md)&lt;out [T](index.md)&gt;

The result type for all Fluxbase SDK operations — the Kotlin equivalent of the TS SDK's `{ data: T; error: null } | { data: null; error: Error }` union (`sdk/src/types.ts:3904`).

Instead of throwing, SDK methods return a [FluxbaseResponse](index.md) so callers can destructure the result:

```kotlin
val (session, error) = client.auth.signInWithPassword(...)
if (error != null) { ... }
```

This matches the TS pattern `const { data, error } = await client.auth.signIn(...)`.

#### Inheritors

| |
|---|
| [Success](-success/index.md) |
| [Error](-error/index.md) |

## Types

| Name | Summary |
|---|---|
| [Error](-error/index.md) | [jvm]<br>data class [Error](-error/index.md)(val error: [FluxbaseError](../-fluxbase-error/index.md)) : [FluxbaseResponse](index.md)&lt;[Nothing](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-nothing/index.html)&gt; <br>Failed result. |
| [Success](-success/index.md) | [jvm]<br>data class [Success](-success/index.md)&lt;[T](-success/index.md)&gt;(val data: [T](-success/index.md)) : [FluxbaseResponse](index.md)&lt;[T](-success/index.md)&gt; <br>Successful result. |

## Properties

| Name | Summary |
|---|---|
| [data](data.md) | [jvm]<br>abstract val [data](data.md): [T](index.md)?<br>The data payload on success, or null on error. In the TS SDK this is the `data` field. |
| [error](error.md) | [jvm]<br>abstract val [error](error.md): [FluxbaseError](../-fluxbase-error/index.md)?<br>The error on failure. Null on success. |

## Functions

| Name | Summary |
|---|---|
| [component1](component1.md) | [jvm]<br>open operator fun [component1](component1.md)(): [T](index.md)?<br>Destructuring support: `val (data, error) = result`. Matches the TS `const { data, error } = await ...` pattern. These must be members (not extensions) for Kotlin to recognize them in destructuring declarations. |
| [component2](component2.md) | [jvm]<br>open operator fun [component2](component2.md)(): [FluxbaseError](../-fluxbase-error/index.md)? |
| [getOrNull](../get-or-null.md) | [jvm]<br>fun &lt;[T](../get-or-null.md)&gt; [FluxbaseResponse](index.md)&lt;[T](../get-or-null.md)&gt;.[getOrNull](../get-or-null.md)(): [T](../get-or-null.md)?<br>Returns the data on success, or null on error. Equivalent to `result.data` in the TS SDK. |
| [getOrThrow](../get-or-throw.md) | [jvm]<br>fun &lt;[T](../get-or-throw.md)&gt; [FluxbaseResponse](index.md)&lt;[T](../get-or-throw.md)&gt;.[getOrThrow](../get-or-throw.md)(): [T](../get-or-throw.md)<br>Returns the data on success, or throws the error on failure. |
