---
title: "validate"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.management](../index.md)/[InvitationsManager](index.md)/[validate](validate.md)

# validate

[jvm]\
suspend fun [validate](validate.md)(token: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[ValidateInvitationResponse](../-validate-invitation-response/index.md)&gt;

Validate an invitation token (public). GETs `/api/v1/invitations/{token}/validate`.
