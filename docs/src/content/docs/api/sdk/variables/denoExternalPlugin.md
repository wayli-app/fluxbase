---
editUrl: false
next: false
prev: false
title: "denoExternalPlugin"
---

> `const` **denoExternalPlugin**: `object`

esbuild plugin that marks Deno-specific imports as external
Use this when bundling functions/jobs with esbuild to handle npm:, https://, and jsr: imports

## Type declaration

| Name | Type | Default value |
| ------ | ------ | ------ |
| `name` | `string` | "deno-external" |
| `setup()` | `void` | - |

## Example

```typescript
import { denoExternalPlugin } from '@fluxbase/sdk'
import * as esbuild from 'esbuild'

const result = await esbuild.build({
  entryPoints: ['./my-function.ts'],
  bundle: true,
  plugins: [denoExternalPlugin],
  // ... other options
})
```
