---
title: "FluxbaseManagement"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../)/[io.github.nimbleflux.fluxbase.management](../)/[FluxbaseManagement](./)

# FluxbaseManagement

[jvm]\
class [FluxbaseManagement](./)(http: [FluxbaseHttpClient](../../iogithubnimblefluxfluxbasecore/-fluxbase-http-client/))

Management module — port of `FluxbaseManagement` from `sdk/src/management.ts`.

Aggregate of three sub-managers: client keys, webhooks, invitations. Unlike most modules which wrap in `{data, error}`, the TS management managers throw on error. The Kotlin port wraps them in [FluxbaseResponse](../../iogithubnimblefluxfluxbase/-fluxbase-response/) for consistency.

Usage:

```kotlin
val (key, _) = client.management.clientKeys.create(mapOf("name" to "mobile-app", "scopes" to listOf("read")))
val (hooks, _) = client.management.webhooks.list()
```

## Constructors

| | |
|---|---|
| [FluxbaseManagement](-fluxbase-management/) | [jvm]<br>constructor(http: [FluxbaseHttpClient](../../iogithubnimblefluxfluxbasecore/-fluxbase-http-client/)) |

## Properties

| Name | Summary |
|---|---|
| [clientKeys](client-keys/) | [jvm]<br>val [clientKeys](client-keys/): [ClientKeysManager](../-client-keys-manager/) |
| [invitations](invitations/) | [jvm]<br>val [invitations](invitations/): [InvitationsManager](../-invitations-manager/) |
| [webhooks](webhooks/) | [jvm]<br>val [webhooks](webhooks/): [WebhooksManager](../-webhooks-manager/) |
