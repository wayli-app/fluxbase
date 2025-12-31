---
editUrl: false
next: false
prev: false
title: "isObject"
---

> **isObject**(`value`): `value is Record<string, unknown>`

Type guard to check if a value is a non-null object
Useful for narrowing unknown types from API responses

## Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `value` | `unknown` | The value to check |

## Returns

`value is Record<string, unknown>`

true if value is a non-null object
