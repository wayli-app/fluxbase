---
editUrl: false
next: false
prev: false
title: "bundleCode"
---

> **bundleCode**(`options`): `Promise`\<[`BundleResult`](/api/sdk/interfaces/bundleresult/)\>

Bundle code using esbuild (client-side)

Transforms and bundles TypeScript/JavaScript code into a single file
that can be executed by the Fluxbase Deno runtime.

Requires esbuild as a peer dependency.

## Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `options` | [`BundleOptions`](/api/sdk/interfaces/bundleoptions/) | Bundle options including source code |

## Returns

`Promise`\<[`BundleResult`](/api/sdk/interfaces/bundleresult/)\>

Promise resolving to bundled code

## Throws

Error if esbuild is not available

## Example

```typescript
import { bundleCode } from '@fluxbase/sdk'

const bundled = await bundleCode({
  code: `
    import { helper } from './utils'
    export default async function handler(req) {
      return helper(req.payload)
    }
  `,
  minify: true,
})

// Use bundled code in sync
await client.admin.functions.sync({
  namespace: 'default',
  functions: [{
    name: 'my-function',
    code: bundled.code,
    is_pre_bundled: true,
  }]
})
```
