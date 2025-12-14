---
editUrl: false
next: false
prev: false
title: "loadImportMap"
---

> **loadImportMap**(`denoJsonPath`): `Promise`\<`Record`\<`string`, `string`\> \| `null`\>

Load import map from a deno.json file

## Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `denoJsonPath` | `string` | Path to deno.json file |

## Returns

`Promise`\<`Record`\<`string`, `string`\> \| `null`\>

Import map object or null if not found

## Example

```typescript
const importMap = await loadImportMap('./deno.json')
const bundled = await bundleCode({
  code: myCode,
  importMap,
})
```
