---
title: "FluxbaseManagement"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.management](../index.md)/[FluxbaseManagement](index.md)

# FluxbaseManagement

[jvm]\
class [FluxbaseManagement](index.md)(http: [FluxbaseHttpClient](../../io.github.nimbleflux.fluxbase.core/-fluxbase-http-client/index.md))

Management module — port of `FluxbaseManagement` from `sdk/src/management.ts`.

Aggregate of three sub-managers: client keys, webhooks, invitations. Unlike most modules which wrap in `{data, error}`, the TS management managers throw on error. The Kotlin port wraps them in [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md) for consistency.

Usage:

```kotlin
val (key, _) = client.management.clientKeys.create(mapOf("name" to "mobile-app", "scopes" to listOf("read")))
val (hooks, _) = client.management.webhooks.list()
```

## Constructors

| | |
|---|---|
| [FluxbaseManagement](-fluxbase-management.md) | [jvm]<br>constructor(http: [FluxbaseHttpClient](../../io.github.nimbleflux.fluxbase.core/-fluxbase-http-client/index.md)) |

## Properties

| Name | Summary |
|---|---|
| [clientKeys](client-keys.md) | [jvm]<br>val [clientKeys](client-keys.md): [ClientKeysManager](../-client-keys-manager/index.md) |
| [invitations](invitations.md) | [jvm]<br>val [invitations](invitations.md): [InvitationsManager](../-invitations-manager/index.md) |
| [webhooks](webhooks.md) | [jvm]<br>val [webhooks](webhooks.md): [WebhooksManager](../-webhooks-manager/index.md) |
