---
editUrl: false
next: false
prev: false
title: "useTableSubscription"
---

> **useTableSubscription**(`table`, `options`?): `object`

Hook to subscribe to a table's changes

## Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `table` | `string` | Table name (with optional schema, e.g., 'public.products') |
| `options`? | `Omit`\<`UseRealtimeOptions`, `"channel"`\> | Subscription options |

## Returns

`object`

| Name | Type | Default value |
| ------ | ------ | ------ |
| `channel` | `null` \| `RealtimeChannel` | channelRef.current |
