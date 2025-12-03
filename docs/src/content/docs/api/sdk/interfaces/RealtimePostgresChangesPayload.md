---
editUrl: false
next: false
prev: false
title: "RealtimePostgresChangesPayload"
---

Realtime postgres_changes payload structure
Compatible with Supabase realtime payloads

## Type Parameters

| Type Parameter | Default type |
| ------ | ------ |
| `T` | `any` |

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| `commit_timestamp` | `string` | Commit timestamp (Supabase-compatible field name) |
| `errors` | `null` \| `string` | Error message if any |
| `eventType` | `"DELETE"` \| `"INSERT"` \| `"UPDATE"` \| `"*"` | Event type (Supabase-compatible field name) |
| `new` | `T` | New record data (Supabase-compatible field name) |
| `old` | `T` | Old record data (Supabase-compatible field name) |
| `schema` | `string` | Database schema |
| `table` | `string` | Table name |
