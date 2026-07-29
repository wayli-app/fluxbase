# Fluxbase SvelteKit Example

A minimal SvelteKit app showing the three core flows: auth, reactive queries,
and realtime subscriptions.

> This is a reference for how the SDK is wired up, not a runnable project on
> its own (it omits the SvelteKit scaffolding boilerplate). Copy the relevant
> files into a fresh `npm create svelte@latest` app.

## Setup

### 1. `src/hooks.server.ts` — SSR auth via httpOnly cookies

```ts
import { createClient } from '@nimbleflux/fluxbase-sdk'
import { createCookieStorage } from '@nimbleflux/fluxbase-sdk-svelte'
import type { Handle } from '@sveltejs/kit'
import { FLUXBASE_URL } from '$env/static/private'

export const handle: Handle = async ({ event, resolve }) => {
  event.locals.client = createClient({
    url: FLUXBASE_URL,
    auth: { storage: createCookieStorage(event.cookies) },
  })
  return resolve(event)
}
```

### 2. `src/routes/+layout.svelte` — provide the client

```svelte
<script lang="ts">
  import { setFluxbaseClient } from '@nimbleflux/fluxbase-sdk-svelte'
  import type { LayoutData } from './$types'

  export let data

  // `data.client` comes from the +layout.server.ts load function
  setFluxbaseClient(data.client)
</script>

<slot />
```

### 3. `src/routes/+page.svelte` — auth + queries + realtime

```svelte
<script lang="ts">
  import { session, table, tableInserts, signIn, signOut } from '@nimbleflux/fluxbase-sdk-svelte'

  let email = ''
  let password = ''

  const $session = session()
  const products = table('products', (q) => q.select('*').eq('active', true), {
    queryKey: ['products', 'active'],
  })
  const latestInsert = tableInserts('products')
</script>

{#if !$session.data}
  <form on:submit|preventDefault={() => signIn().mutate({ email, password })}>
    <input bind:value={email} type="email" placeholder="Email" />
    <input bind:value={password} type="password" placeholder="Password" />
    <button>Sign in</button>
  </form>
{:else}
  <p>Signed in as {$session.data?.user.email}</p>
  <button on:click={() => signOut().mutate()}>Sign out</button>

  <h2>Products</h2>
  {#if $products.isLoading}Loading…{/if}
  <ul>
    {#each $products.data ?? [] as product (product.id)}
      <li>{product.name}</li>
    {/each}
  </ul>
  {#if $latestInsert}
    <p>🔔 New product added: {$latestInsert.new_record.name}</p>
  {/if}
{/if}
```

## What's happening

- **`hooks.server.ts`** builds the client per-request with cookie-backed
  storage, so the JWT lives in an httpOnly cookie and never reaches client JS.
- **`+layout.svelte`** stashes that client (plus a per-request `QueryClient`)
  in Svelte context.
- **`+page.svelte`** uses `session()`, `table()`, and `tableInserts()` — all
  reactive stores. Realtime auto-subscribes on mount and cleans up on destroy.
