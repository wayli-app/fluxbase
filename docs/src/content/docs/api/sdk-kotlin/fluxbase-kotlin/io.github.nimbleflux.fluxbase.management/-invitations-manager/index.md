---
title: "InvitationsManager"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../)/[io.github.nimbleflux.fluxbase.management](../)/[InvitationsManager](./)

# InvitationsManager

[jvm]\
class [InvitationsManager](./)(http: [FluxbaseHttpClient](../../iogithubnimblefluxfluxbasecore/-fluxbase-http-client/))

Invitations. Public methods: validate, accept. Admin methods: create, list, revoke.

## Constructors

| | |
|---|---|
| [InvitationsManager](-invitations-manager/) | [jvm]<br>constructor(http: [FluxbaseHttpClient](../../iogithubnimblefluxfluxbasecore/-fluxbase-http-client/)) |

## Functions

| Name | Summary |
|---|---|
| [accept](accept/) | [jvm]<br>suspend fun [accept](accept/)(token: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), password: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), name: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[Unit](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-unit/index.html)&gt;<br>Accept an invitation (public). POSTs `/api/v1/invitations/{token}/accept`. |
| [validate](validate/) | [jvm]<br>suspend fun [validate](validate/)(token: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[ValidateInvitationResponse](../-validate-invitation-response/)&gt;<br>Validate an invitation token (public). GETs `/api/v1/invitations/{token}/validate`. |
