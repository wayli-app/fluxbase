---
title: "InvitationsManager"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.management](../index.md)/[InvitationsManager](index.md)

# InvitationsManager

[jvm]\
class [InvitationsManager](index.md)(http: [FluxbaseHttpClient](../../io.github.nimbleflux.fluxbase.core/-fluxbase-http-client/index.md))

Invitations. Public methods: validate, accept. Admin methods: create, list, revoke.

## Constructors

| | |
|---|---|
| [InvitationsManager](-invitations-manager.md) | [jvm]<br>constructor(http: [FluxbaseHttpClient](../../io.github.nimbleflux.fluxbase.core/-fluxbase-http-client/index.md)) |

## Functions

| Name | Summary |
|---|---|
| [accept](accept.md) | [jvm]<br>suspend fun [accept](accept.md)(token: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), password: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), name: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[Unit](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-unit/index.html)&gt;<br>Accept an invitation (public). POSTs `/api/v1/invitations/{token}/accept`. |
| [validate](validate.md) | [jvm]<br>suspend fun [validate](validate.md)(token: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[ValidateInvitationResponse](../-validate-invitation-response/index.md)&gt;<br>Validate an invitation token (public). GETs `/api/v1/invitations/{token}/validate`. |
