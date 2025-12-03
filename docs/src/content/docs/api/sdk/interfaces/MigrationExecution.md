---
editUrl: false
next: false
prev: false
title: "MigrationExecution"
---

Migration execution record (audit log)

## Properties

| Property | Type |
| ------ | ------ |
| `action` | `"apply"` \| `"rollback"` |
| `duration_ms?` | `number` |
| `error_message?` | `string` |
| `executed_at` | `string` |
| `executed_by?` | `string` |
| `id` | `string` |
| `logs?` | `string` |
| `migration_id` | `string` |
| `status` | `"success"` \| `"failed"` |
