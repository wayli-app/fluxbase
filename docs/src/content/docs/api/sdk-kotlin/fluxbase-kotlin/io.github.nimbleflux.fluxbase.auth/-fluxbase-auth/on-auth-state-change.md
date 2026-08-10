---
title: "onAuthStateChange"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.auth](../index.md)/[FluxbaseAuth](index.md)/[onAuthStateChange](on-auth-state-change.md)

# onAuthStateChange

[jvm]\
fun [onAuthStateChange](on-auth-state-change.md)(callback: ([AuthState](../-auth-state/index.md)) -&gt; [Unit](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-unit/index.html)): () -&gt; [Unit](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-unit/index.html)

Register a callback for auth state changes. Returns a function to unsubscribe.

Kotlin-native equivalent of the TS `onAuthStateChange(callback)`. For coroutine-native consumption, wrap this in a callbackFlow.
