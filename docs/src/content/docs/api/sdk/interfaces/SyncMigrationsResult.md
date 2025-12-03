---
editUrl: false
next: false
prev: false
title: "SyncMigrationsResult"
---

Result of a migration sync operation

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| `details` | `object` | Detailed results |
| `details.applied` | `string`[] | - |
| `details.created` | `string`[] | - |
| `details.errors` | `string`[] | - |
| `details.skipped` | `string`[] | - |
| `details.unchanged` | `string`[] | - |
| `details.updated` | `string`[] | - |
| `dry_run` | `boolean` | Whether this was a dry run |
| `message` | `string` | Status message |
| `namespace` | `string` | Namespace that was synced |
| `summary` | `object` | Summary counts |
| `summary.applied` | `number` | - |
| `summary.created` | `number` | - |
| `summary.errors` | `number` | - |
| `summary.skipped` | `number` | - |
| `summary.unchanged` | `number` | - |
| `summary.updated` | `number` | - |
| `warnings?` | `string`[] | Warning messages |
