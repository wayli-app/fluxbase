# @nimbleflux/fluxbase-sdk-svelte

Svelte stores and SvelteKit SSR helpers for the [Fluxbase](https://github.com/nimbleflux/fluxbase) SDK, built on [TanStack Svelte Query](https://tanstack.com/query/latest/docs/framework/svelte/overview).

## Install

```bash
bun add @nimbleflux/fluxbase-sdk-svelte
# or: npm install / pnpm install / yarn add
```

Peer dependencies: `@nimbleflux/fluxbase-sdk`, `@tanstack/svelte-query`, `svelte` (4 or 5).

## Quick Start

### 1. Set up the client + provider (`+layout.svelte`)

```svelte
<script lang="ts">
  import { setFluxbaseClient } from '@nimbleflux/fluxbase-sdk-svelte'
  import { createClient } from '@nimbleflux/fluxbase-sdk'

  const client = createClient({ url: 'http://localhost:8080' })
  setFluxbaseClient(client)
</script>

<slot />
```

`setFluxbaseClient` stashes the client and a **per-request `QueryClient`** in Svelte context. The per-request client is required for correct SSR (a module-level singleton would leak state between users on the server).

### 2. Use reactive stores

```svelte
<script lang="ts">
  import { session, table, signIn } from '@nimbleflux/fluxbase-sdk-svelte'

  const $session = session()
  const products = table('products', (q) => q.eq('active', true), {
    queryKey: ['products', 'active'],
  })
</script>

{#if !$session.data}
  <button on:click={() => signIn().mutate({ email, password })}>Sign in</button>
{:else}
  {#each $products.data ?? [] as p}{p.name}{/each}
{/if}
```

## SSR Auth with httpOnly Cookies

For server-rendered apps, back the session with an httpOnly cookie so the JWT
never reaches client-side JavaScript. Pass a `StorageAdapter` to the core SDK's
`auth.storage` option:

```ts
// hooks.server.ts
import { createClient } from '@nimbleflux/fluxbase-sdk'
import { createCookieStorage } from '@nimbleflux/fluxbase-sdk-svelte'
import type { Handle } from '@sveltejs/kit'

export const handle: Handle = async ({ event, resolve }) => {
  const client = createClient({
    url: FLUXBASE_URL,
    auth: {
      storage: createCookieStorage(event.cookies), // httpOnly + sameSite=lax
    },
  })
  event.locals.client = client
  return resolve(event)
}
```

The cookie adapter handles session persistence; the core SDK's existing
401-token-refresh logic runs against it transparently.

## Available Stores

| Area | Stores |
|------|--------|
| Auth | `session`, `user`, `signIn`, `signUp`, `signOut`, `updateUser` |
| Data | `table`, `fluxbaseQuery`, `insert`, `update`, `upsert`, `remove` |
| Realtime | `realtimeChannel`, `tableInserts`, `tableUpdates`, `tableDeletes` |
| Storage | `storageList`, `storageBuckets`, `storageUpload`, `storageDownload`, `storageRemove`, `storagePublicUrl` |
| Functions | `invokeFunction`, `functions` |
| Jobs | `jobs`, `jobStatus`, `submitJob`, `cancelJob` |
| Branching | `branches`, `createBranch`, `deleteBranch` |
| RPC | `rpcList`, `invokeRPC` |
| Vector | `vectorEmbed`, `vectorSearch` |
| Secrets | `secrets`, `createSecret`, `updateSecret`, `deleteSecret` |
| GraphQL | `graphqlQuery`, `graphqlMutation` |
| SAML | `samlProviders`, `signInWithSAML` |
| Config | `captchaConfig`, `authConfig` |
| Admin | `users`, `webhooks`, `appSettings`, `systemSettings` |

Query stores are read with `$` (e.g. `$products.data`). Mutation stores are
read with `$` then called (e.g. `signIn().mutate(...)` or `get(signIn()).mutateAsync(...)`).

## License

MIT
