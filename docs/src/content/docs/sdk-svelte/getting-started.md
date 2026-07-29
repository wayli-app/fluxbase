---
title: Svelte SDK
description: Reactive Svelte stores and SvelteKit SSR helpers for Fluxbase, built on TanStack Svelte Query.
---

# Svelte SDK

The Fluxbase Svelte SDK (`@nimbleflux/fluxbase-sdk-svelte`) provides idiomatic
Svelte stores and SvelteKit SSR helpers on top of the core
[`@nimbleflux/fluxbase-sdk`](../sdk/getting-started/), built on
[TanStack Svelte Query](https://tanstack.com/query/latest/docs/framework/svelte/overview).

## Installation

```bash
bun add @nimbleflux/fluxbase-sdk-svelte
```

Peer dependencies: `@nimbleflux/fluxbase-sdk`, `@tanstack/svelte-query`, `svelte` (4 or 5).

## Set up the provider

In your root `+layout.svelte`, create a Fluxbase client and provide it along
with a **per-request `QueryClient`** via `setFluxbaseClient`:

```svelte
<script lang="ts">
  import { setFluxbaseClient } from '@nimbleflux/fluxbase-sdk-svelte'
  import { createClient } from '@nimbleflux/fluxbase-sdk'

  const client = createClient({ url: 'http://localhost:8080' })
  setFluxbaseClient(client)
</script>

<slot />
```

A per-request `QueryClient` is mandatory for correct SSR — a module-level
singleton would leak query state between users on the server.

## SSR auth with httpOnly cookies

For server-rendered apps, back the auth session with an httpOnly cookie so the
JWT never reaches client-side JavaScript. The SDK ships a
`createCookieStorage` adapter that bridges SvelteKit's `cookies()` API to the
core SDK's [injectable `StorageAdapter`](../sdk/getting-started/#ssr--custom-storage-adapter):

```ts
// src/hooks.server.ts
import { createClient } from '@nimbleflux/fluxbase-sdk'
import { createCookieStorage } from '@nimbleflux/fluxbase-sdk-svelte'
import type { Handle } from '@sveltejs/kit'

export const handle: Handle = async ({ event, resolve }) => {
  event.locals.client = createClient({
    url: FLUXBASE_URL,
    auth: {
      storage: createCookieStorage(event.cookies), // httpOnly + sameSite=lax
    },
  })
  return resolve(event)
}
```

The cookie adapter handles persistence; the core SDK's existing 401 token
refresh runs against it transparently.

## Use reactive stores

Query stores are read with `$` (e.g. `$products.data`). Mutation stores are
read with `$` then invoked (e.g. `$signIn.mutate(...)`).

```svelte
<script lang="ts">
  import { session, table, signIn, signOut, tableInserts } from '@nimbleflux/fluxbase-sdk-svelte'

  const $session = session()
  const products = table('products', (q) => q.select('*').eq('active', true), {
    queryKey: ['products', 'active'],
  })
  const latestInsert = tableInserts('products')
</script>

{#if !$session.data}
  <button on:click={() => signIn().mutate({ email, password })}>Sign in</button>
{:else}
  <p>Signed in as {$session.data?.user.email}</p>
  <button on:click={() => signOut().mutate()}>Sign out</button>

  <ul>
    {#each $products.data ?? [] as product (product.id)}
      <li>{product.name}</li>
    {/each}
  </ul>
  {#if $latestInsert}
    <p>🔔 New product: {$latestInsert.new_record.name}</p>
  {/if}
{/if}
```

Realtime stores (`tableInserts`, `tableUpdates`, `tableDeletes`,
`realtimeChannel`) subscribe on mount and clean up on destroy automatically.

## What's available

The Svelte SDK mirrors the [React SDK](../sdk-react/getting-started/) surface:
auth, database queries, realtime, storage, functions, jobs, branching, RPC,
vector, secrets, GraphQL, SAML, captcha, auth config, and admin operations.
