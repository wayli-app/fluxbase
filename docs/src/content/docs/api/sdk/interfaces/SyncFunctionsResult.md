---
editUrl: false
next: false
prev: false
title: "SyncFunctionsResult"
---

Result of a function sync operation

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| `details` | `object` | Detailed results |
| `details.created` | `string`[] | - |
| `details.deleted` | `string`[] | - |
| `details.unchanged` | `string`[] | - |
| `details.updated` | `string`[] | - |
| `dry_run` | `boolean` | Whether this was a dry run |
| `errors` | [`SyncError`](/api/sdk/interfaces/syncerror/)[] | Errors encountered |
| `message` | `string` | Status message |
| `namespace` | `string` | Namespace that was synced |
| `summary` | `object` | Summary counts |
| `summary.created` | `number` | - |
| `summary.deleted` | `number` | - |
| `summary.errors` | `number` | - |
| `summary.unchanged` | `number` | - |
| `summary.updated` | `number` | - |
