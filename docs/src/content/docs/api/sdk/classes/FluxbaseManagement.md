---
editUrl: false
next: false
prev: false
title: "FluxbaseManagement"
---

Management client for client keys, webhooks, and invitations

## Constructors

### new FluxbaseManagement()

> **new FluxbaseManagement**(`fetch`): [`FluxbaseManagement`](/api/sdk/classes/fluxbasemanagement/)

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `fetch` | [`FluxbaseFetch`](/api/sdk/classes/fluxbasefetch/) |

#### Returns

[`FluxbaseManagement`](/api/sdk/classes/fluxbasemanagement/)

## Properties

| Property | Modifier | Type | Description |
| ------ | ------ | ------ | ------ |
| ~~`apiKeys`~~ | `public` | [`ClientKeysManager`](/api/sdk/classes/clientkeysmanager/) | :::caution[Deprecated] Use clientKeys instead ::: |
| `clientKeys` | `public` | [`ClientKeysManager`](/api/sdk/classes/clientkeysmanager/) | Client Keys management |
| `invitations` | `public` | [`InvitationsManager`](/api/sdk/classes/invitationsmanager/) | Invitations management |
| `webhooks` | `public` | [`WebhooksManager`](/api/sdk/classes/webhooksmanager/) | Webhooks management |
