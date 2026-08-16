---
title: "updateToken"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.realtime](../index.md)/[RealtimeChannel](index.md)/[updateToken](update-token.md)

# updateToken

[jvm]\
suspend fun [updateToken](update-token.md)(newToken: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html))

Update the auth token (e.g. after a JWT refresh). If the socket is open, pushes an `access_token` message so the server refreshes auth context without a reconnect; otherwise the new token is used on the next connect.

Port of `updateToken()` in `realtime.ts:1037-1090`. Note the TS version additionally waits for an ack and reconnects on a 5s timeout; this Kotlin port propagates the token immediately and relies on [onError](on-error.md) / reconnect handling if the server rejects it.
