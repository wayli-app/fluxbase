---
title: Vue SDK
description: Vue/Nuxt composables and SSR cookie storage for Fluxbase.
---

# Vue SDK

The Fluxbase Vue SDK (`@nimbleflux/fluxbase-sdk-vue`) provides composables and
SSR cookie-based auth for [Vue 3](https://vuejs.org/) /
[Nuxt](https://nuxt.com/), built on the core
[`@nimbleflux/fluxbase-sdk`](../sdk/getting-started/).

> **Scaffold status:** provides the SSR-auth foundation (cookie storage,
> provide/inject composable). Full reactive composables (like the React/Svelte
> hooks) are a follow-on.

## Installation

```bash
bun add @nimbleflux/fluxbase-sdk-vue
```

Peer dependencies: `@nimbleflux/fluxbase-sdk`, `vue`.

## Provide the client

```ts
// main.ts
import { createApp } from 'vue'
import { createClient } from '@nimbleflux/fluxbase-sdk'
import { FLUXBASE_CLIENT_KEY } from '@nimbleflux/fluxbase-sdk-vue'
import App from './App.vue'

const app = createApp(App)
app.provide(
  FLUXBASE_CLIENT_KEY,
  createClient({ url: 'http://localhost:8080' }),
)
app.mount('#app')
```

Use it in a component:

```vue
<script setup lang="ts">
import { useFluxbaseClient } from '@nimbleflux/fluxbase-sdk-vue'

const client = useFluxbaseClient()
const { data } = await client.from('products').select('*').execute()
</script>
```

## SSR auth with httpOnly cookies (Nuxt)

Back the auth session with an httpOnly cookie via the h3 event's cookie API:

```ts
// server plugin / middleware
import { createClient } from '@nimbleflux/fluxbase-sdk'
import { createCookieStorage } from '@nimbleflux/fluxbase-sdk-vue'

export default defineEventHandler((event) => {
  const client = createClient({
    url: useRuntimeConfig().fluxbaseUrl,
    auth: { storage: createCookieStorage(event) }, // httpOnly + sameSite=lax
  })
  event.context.fluxbase = client
})
```

## How the cookie adapter works

The adapter uses the core SDK's [injectable `StorageAdapter`](../sdk/getting-started/#ssr--custom-storage-adapter)
seam to persist the auth session in an httpOnly cookie.
