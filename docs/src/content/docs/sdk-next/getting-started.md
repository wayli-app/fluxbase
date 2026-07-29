---
title: Next.js SDK
description: Server/client adapters and SSR cookie storage for Fluxbase in Next.js.
---

# Next.js SDK

The Fluxbase Next.js SDK (`@nimbleflux/fluxbase-sdk-next`) provides server-side
client creation, SSR cookie-based auth, and a client-side provider for
[Next.js](https://nextjs.org/), built on the core
[`@nimbleflux/fluxbase-sdk`](../sdk/getting-started/).

> **Scaffold status:** provides the SSR-auth foundation (cookie storage, server
> client factory, client provider). Full React Query hooks (like the React SDK)
> are a follow-on.

## Installation

```bash
bun add @nimbleflux/fluxbase-sdk-next
```

Peer dependencies: `@nimbleflux/fluxbase-sdk`, `next`, `react`.

## SSR auth with httpOnly cookies

Back the auth session with an httpOnly cookie so the JWT never reaches
client-side JavaScript.

### Server Components / Route Handlers / Server Actions

```ts
// app/lib/fluxbase.ts
import { cookies } from 'next/headers'
import { createServerClient } from '@nimbleflux/fluxbase-sdk-next'

export async function getClient() {
  return createServerClient(process.env.FLUXBASE_URL!, {
    cookies: await cookies(),
  })
}
```

Or wire the cookie adapter directly:

```ts
import { createClient } from '@nimbleflux/fluxbase-sdk'
import { createCookieStorage } from '@nimbleflux/fluxbase-sdk-next'
import { cookies } from 'next/headers'

const cookieStore = await cookies()
const client = createClient({
  url: process.env.FLUXBASE_URL!,
  key: process.env.FLUXBASE_ANON_KEY!,
  auth: { storage: createCookieStorage(cookieStore) },
})
```

## Client Components

```tsx
'use client'
import { FluxbaseProvider, useFluxbaseClient } from '@nimbleflux/fluxbase-sdk-next'
import { createClient } from '@nimbleflux/fluxbase-sdk'

export function Providers({ children }) {
  const client = createClient({ url: process.env.NEXT_PUBLIC_FLUXBASE_URL! })
  return <FluxbaseProvider client={client}>{children}</FluxbaseProvider>
}
```

## How the cookie adapter works

The adapter uses the core SDK's [injectable `StorageAdapter`](../sdk/getting-started/#ssr--custom-storage-adapter)
seam to persist the auth session in an httpOnly cookie. The core SDK's existing
401 token-refresh logic runs against it transparently.
