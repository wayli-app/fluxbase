---
editUrl: false
next: false
prev: false
title: "FunctionSpec"
---

Function specification for bulk sync operations

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| `allow_env?` | `boolean` | - |
| `allow_net?` | `boolean` | - |
| `allow_read?` | `boolean` | - |
| `allow_unauthenticated?` | `boolean` | - |
| `allow_write?` | `boolean` | - |
| `code` | `string` | - |
| `cron_schedule?` | `string` | - |
| `description?` | `string` | - |
| `enabled?` | `boolean` | - |
| `is_pre_bundled?` | `boolean` | If true, code is already bundled and server will skip bundling |
| `is_public?` | `boolean` | - |
| `memory_limit_mb?` | `number` | - |
| `name` | `string` | - |
| `nodePaths?` | `string`[] | Additional paths to search for node_modules during bundling (used by syncWithBundling) |
| `original_code?` | `string` | Original source code (for debugging when pre-bundled) |
| `sourceDir?` | `string` | Source directory for resolving relative imports during bundling (used by syncWithBundling) |
| `timeout_seconds?` | `number` | - |
