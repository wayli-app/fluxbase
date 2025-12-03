---
editUrl: false
next: false
prev: false
title: "useTableUpdates"
---

> **useTableUpdates**(`table`, `callback`, `options`?): `object`

Hook to subscribe to UPDATE events on a table

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
