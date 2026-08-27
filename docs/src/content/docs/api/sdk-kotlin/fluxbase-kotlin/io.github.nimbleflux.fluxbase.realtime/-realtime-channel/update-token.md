---
title: "updateToken"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../../)/[io.github.nimbleflux.fluxbase.realtime](../../)/[RealtimeChannel](../)/[updateToken](./)

# updateToken

[jvm]\
suspend fun [updateToken](./)(newToken: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html))

Update the auth token (e.g. after a JWT refresh). If the socket is open, pushes an `access_token` message so the server refreshes auth context without a reconnect; otherwise the new token is used on the next connect.

Port of `updateToken()` in `realtime.ts:1037-1090`. Note the TS version additionally waits for an ack and reconnects on a 5s timeout; this Kotlin port propagates the token immediately and relies on [onError](../on-error/) / reconnect handling if the server rejects it.
