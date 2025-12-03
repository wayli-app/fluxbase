---
editUrl: false
next: false
prev: false
title: "useTableDeletes"
---

> **useTableDeletes**(`table`, `callback`, `options`?): `object`

Hook to subscribe to DELETE events on a table

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
