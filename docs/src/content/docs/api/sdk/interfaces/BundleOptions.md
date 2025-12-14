---
editUrl: false
next: false
prev: false
title: "BundleOptions"
---

Options for bundling code

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| `baseDir?` | `string` | Base directory for resolving relative imports (resolveDir in esbuild) |
| `code` | `string` | Entry point code |
| `define?` | `Record`\<`string`, `string`\> | Custom define values for esbuild (e.g., { 'process.env.NODE_ENV': '"production"' }) |
| `external?` | `string`[] | External modules to exclude from bundle |
| `importMap?` | `Record`\<`string`, `string`\> | Import map from deno.json (maps aliases to npm: or file paths) |
| `minify?` | `boolean` | Minify output |
| `nodePaths?` | `string`[] | Additional paths to search for node_modules (useful when importing from parent directories) |
| `sourcemap?` | `boolean` | Source map generation |
