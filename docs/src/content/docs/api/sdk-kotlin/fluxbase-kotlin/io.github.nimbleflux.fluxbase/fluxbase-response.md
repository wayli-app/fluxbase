---
title: "fluxbaseResponse"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../)/[io.github.nimbleflux.fluxbase](../)/[fluxbaseResponse](./)

# fluxbaseResponse

[jvm]\
suspend fun &lt;[T](./)&gt; [fluxbaseResponse](./)(block: suspend () -&gt; [T](./)): [FluxbaseResponse](../-fluxbase-response/)&lt;[T](./)&gt;

Wraps a suspending block, catching exceptions and converting them to [FluxbaseResponse.Error](../-fluxbase-response/-error/). The TS SDK uses `wrapAsync` for the same purpose (`sdk/src/utils/error-handling.ts`).
