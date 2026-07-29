# @nimbleflux/fluxbase-sdk-next

Next.js server/client adapters and SSR cookie storage for the
[Fluxbase](https://github.com/nimbleflux/fluxbase) SDK.

> **Scaffold status:** provides the SSR-auth foundation (cookie storage, server
> client factory, client provider). Full React Query hooks (like the React SDK)
> are a follow-on.

## Install

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
import { createServerClient } from '@nimbleflux/fluxbase-sdk-next'

export async function getClient() {
  return createServerClient(process.env.FLUXBASE_URL!, {
    cookies: await cookies(), // from 'next/headers'
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
  auth: { storage: createCookieStorage(cookieStore) },
})
```

### Client Components

```tsx
'use client'
import { FluxbaseProvider, useFluxbaseClient } from '@nimbleflux/fluxbase-sdk-next'
import { createClient } from '@nimbleflux/fluxbase-sdk'

export function Providers({ children }) {
  const client = createClient({ url: process.env.NEXT_PUBLIC_FLUXBASE_URL! })
  return <FluxbaseProvider client={client}>{children}</FluxbaseProvider>
}

export function MyComponent() {
  const client = useFluxbaseClient()
  // ...
}
```

## License

MIT
