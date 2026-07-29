"use client";

/**
 * Client-side provider for using a Fluxbase client in React Client Components.
 *
 * Wrap your app (or a subtree) with `<FluxbaseProvider client={...}>` and read
 * the client in child components via `useFluxbaseClient()`.
 *
 * For Server Components, use `createServerClient` instead.
 *
 * @example
 * ```tsx
 * 'use client'
 * import { FluxbaseProvider } from '@nimbleflux/fluxbase-sdk-next'
 * import { createClient } from '@nimbleflux/fluxbase-sdk'
 *
 * export function Providers({ children }) {
 *   const client = createClient({ url: process.env.NEXT_PUBLIC_FLUXBASE_URL! })
 *   return <FluxbaseProvider client={client}>{children}</FluxbaseProvider>
 * }
 * ```
 */

import { createContext, useContext, type ReactNode } from "react";
import type { FluxbaseClient } from "@nimbleflux/fluxbase-sdk";

const FluxbaseContext = createContext<FluxbaseClient | null>(null);

export interface FluxbaseProviderProps {
  client: FluxbaseClient;
  children: ReactNode;
}

export function FluxbaseProvider({ client, children }: FluxbaseProviderProps) {
  return (
    <FluxbaseContext.Provider value={client}>
      {children}
    </FluxbaseContext.Provider>
  );
}

export function useFluxbaseClient(): FluxbaseClient {
  const client = useContext(FluxbaseContext);
  if (!client) {
    throw new Error(
      "useFluxbaseClient must be used within a <FluxbaseProvider>.",
    );
  }
  return client;
}
