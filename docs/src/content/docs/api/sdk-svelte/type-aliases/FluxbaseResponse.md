---
editUrl: false
next: false
prev: false
title: "FluxbaseResponse"
---

> **FluxbaseResponse**\<`T`\> = \{ `data`: `T`; `error`: `null`; \} \| \{ `data`: `null`; `error`: `Error`; \}

Base Fluxbase response type (Supabase-compatible)
Returns either `{ data, error: null }` on success or `{ data: null, error }` on failure

## Type Parameters

| Type Parameter |
| ------ |
| `T` |
