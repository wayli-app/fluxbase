---
title: "FluxbaseResponse"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../)/[io.github.nimbleflux.fluxbase](../)/[FluxbaseResponse](./)

# FluxbaseResponse

sealed interface [FluxbaseResponse](./)&lt;out [T](./)&gt;

The result type for all Fluxbase SDK operations — the Kotlin equivalent of the TS SDK's `{ data: T; error: null } | { data: null; error: Error }` union (`sdk/src/types.ts:3904`).

Instead of throwing, SDK methods return a [FluxbaseResponse](./) so callers can destructure the result:

```kotlin
val (session, error) = client.auth.signInWithPassword(...)
if (error != null) { ... }
```

This matches the TS pattern `const { data, error } = await client.auth.signIn(...)`.

#### Inheritors

| |
|---|
| [Success](-success/) |
| [Error](-error/) |

## Types

| Name | Summary |
|---|---|
| [Error](-error/) | [jvm]<br>data class [Error](-error/)(val error: [FluxbaseError](../-fluxbase-error/)) : [FluxbaseResponse](./)&lt;[Nothing](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-nothing/index.html)&gt; <br>Failed result. |
| [Success](-success/) | [jvm]<br>data class [Success](-success/)&lt;[T](-success/)&gt;(val data: [T](-success/)) : [FluxbaseResponse](./)&lt;[T](-success/)&gt; <br>Successful result. |

## Properties

| Name | Summary |
|---|---|
| [data](data/) | [jvm]<br>abstract val [data](data/): [T](./)?<br>The data payload on success, or null on error. In the TS SDK this is the `data` field. |
| [error](error/) | [jvm]<br>abstract val [error](error/): [FluxbaseError](../-fluxbase-error/)?<br>The error on failure. Null on success. |

## Functions

| Name | Summary |
|---|---|
| [component1](component1/) | [jvm]<br>open operator fun [component1](component1/)(): [T](./)?<br>Destructuring support: `val (data, error) = result`. Matches the TS `const { data, error } = await ...` pattern. These must be members (not extensions) for Kotlin to recognize them in destructuring declarations. |
| [component2](component2/) | [jvm]<br>open operator fun [component2](component2/)(): [FluxbaseError](../-fluxbase-error/)? |
| [getOrNull](../get-or-null/) | [jvm]<br>fun &lt;[T](../get-or-null/)&gt; [FluxbaseResponse](./)&lt;[T](../get-or-null/)&gt;.[getOrNull](../get-or-null/)(): [T](../get-or-null/)?<br>Returns the data on success, or null on error. Equivalent to `result.data` in the TS SDK. |
| [getOrThrow](../get-or-throw/) | [jvm]<br>fun &lt;[T](../get-or-throw/)&gt; [FluxbaseResponse](./)&lt;[T](../get-or-throw/)&gt;.[getOrThrow](../get-or-throw/)(): [T](../get-or-throw/)<br>Returns the data on success, or throws the error on failure. |
