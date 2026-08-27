---
title: "updateUser"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../../)/[io.github.nimbleflux.fluxbase.auth](../../)/[FluxbaseAuth](../)/[updateUser](./)

# updateUser

[jvm]\
suspend fun [updateUser](./)(attributes: [Map](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)&gt;): [FluxbaseResponse](../../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[User](../../-user/)&gt;

Update the user's attributes. PATCHes `/api/v1/auth/user`. Port of `updateUser()` in `auth.ts`.
