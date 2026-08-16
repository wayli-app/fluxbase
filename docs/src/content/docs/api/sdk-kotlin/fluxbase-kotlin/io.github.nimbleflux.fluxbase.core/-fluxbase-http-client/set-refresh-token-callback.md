---
title: "setRefreshTokenCallback"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.core](../index.md)/[FluxbaseHttpClient](index.md)/[setRefreshTokenCallback](set-refresh-token-callback.md)

# setRefreshTokenCallback

[jvm]\
fun [setRefreshTokenCallback](set-refresh-token-callback.md)(callback: suspend () -&gt; [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)?)

Register the callback that refreshes the access token. On a 401 the client invokes this once (deduped across concurrent requests), applies the returned token via [setAuthToken](set-auth-token.md), and retries the original request a single time. Mirrors TS `setRefreshTokenCallback` in `fetch.ts`.
