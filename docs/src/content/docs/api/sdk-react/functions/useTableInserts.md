---
editUrl: false
next: false
prev: false
title: "useTableInserts"
---

> **useTableInserts**(`table`, `callback`, `options`?): `object`

Hook to subscribe to INSERT events on a table

## Parameters

| Parameter | Type |
| ------ | ------ |
| `table` | `string` |
| `callback` | (`payload`) => `void` |
| `options`? | `Omit`\<`UseRealtimeOptions`, `"channel"` \| `"event"` \| `"callback"`\> |

## Returns

`object`

| Name | Type | Default value |
| ------ | ------ | ------ |
| `channel` | `null` \| `RealtimeChannel` | channelRef.current |
