---
editUrl: false
next: false
prev: false
title: "setFluxbaseClient"
---

> **setFluxbaseClient**(`client`, `queryClient?`): `void`

Stash the Fluxbase client and a fresh per-request QueryClient in Svelte
context. Call this once in your root `+layout.svelte` (and in
`hooks.server.ts` if you build a server-side client there).

## Parameters

| Parameter | Type |
| ------ | ------ |
| `client` | [`FluxbaseClient`](/api/sdk-svelte/interfaces/fluxbaseclient/) |
| `queryClient?` | `QueryClient` |

## Returns

`void`

## Example

```svelte
<!-- +layout.svelte -->
<script lang="ts">
  import { setFluxbaseClient } from '@nimbleflux/fluxbase-sdk-svelte'
  import { createClient } from '@nimbleflux/fluxbase-sdk'

  const client = createClient({ url: $env...PUBLIC.FLUXBASE_URL })
  setFluxbaseClient(client)
</script>
<slot />
```
