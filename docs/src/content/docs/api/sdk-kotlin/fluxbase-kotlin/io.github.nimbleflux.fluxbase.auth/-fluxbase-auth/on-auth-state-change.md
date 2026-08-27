---
title: "onAuthStateChange"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../../)/[io.github.nimbleflux.fluxbase.auth](../../)/[FluxbaseAuth](../)/[onAuthStateChange](./)

# onAuthStateChange

[jvm]\
fun [onAuthStateChange](./)(callback: ([AuthState](../../-auth-state/)) -&gt; [Unit](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-unit/index.html)): () -&gt; [Unit](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-unit/index.html)

Register a callback for auth state changes. Returns a function to unsubscribe.

Kotlin-native equivalent of the TS `onAuthStateChange(callback)`. For coroutine-native consumption, wrap this in a callbackFlow.
