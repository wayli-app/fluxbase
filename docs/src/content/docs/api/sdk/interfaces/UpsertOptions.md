---
editUrl: false
next: false
prev: false
title: "UpsertOptions"
---

Options for upsert operations (Supabase-compatible)

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| `defaultToNull?` | `boolean` | If true, missing columns default to null instead of using existing values **Default** `false` |
| `ignoreDuplicates?` | `boolean` | If true, duplicate rows are ignored (not upserted) **Default** `false` |
| `onConflict?` | `string` | Comma-separated columns to use for conflict resolution **Examples** `'email'` `'user_id,tenant_id'` |
