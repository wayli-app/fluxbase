---
editUrl: false
next: false
prev: false
title: "SyncFunctionsOptions"
---

Options for syncing functions

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| `functions` | [`FunctionSpec`](/api/sdk/interfaces/functionspec/)[] | Functions to sync |
| `namespace?` | `string` | Namespace to sync functions to (defaults to "default") |
| `options?` | `object` | Options for sync operation |
| `options.delete_missing?` | `boolean` | Delete functions in namespace that are not in the sync payload |
| `options.dry_run?` | `boolean` | Preview changes without applying them |
