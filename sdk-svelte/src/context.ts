/**
 * Svelte context for the Fluxbase client and per-request QueryClient.
 *
 * In SvelteKit, a module-level QueryClient would leak state between users on
 * the server, so the provider stashes a fresh QueryClient in context per
 * request. Components retrieve both the client and the query client via
 * `getClient()` / `getQueryClient()`.
 */

import { getContext, setContext } from "svelte";
import { QueryClient } from "@tanstack/svelte-query";
import type { FluxbaseClient } from "@nimbleflux/fluxbase-sdk";

const CLIENT_KEY = Symbol("fluxbase-client");
const QUERY_CLIENT_KEY = Symbol("fluxbase-query-client");

/**
 * Stash the Fluxbase client and a fresh per-request QueryClient in Svelte
 * context. Call this once in your root `+layout.svelte` (and in
 * `hooks.server.ts` if you build a server-side client there).
 *
 * @example
 * ```svelte
 * <!-- +layout.svelte -->
 * <script lang="ts">
 *   import { setFluxbaseClient } from '@nimbleflux/fluxbase-sdk-svelte'
 *   import { createClient } from '@nimbleflux/fluxbase-sdk'
 *
 *   const client = createClient({ url: $env...PUBLIC.FLUXBASE_URL })
 *   setFluxbaseClient(client)
 * </script>
 * <slot />
 * ```
 */
export function setFluxbaseClient(
  client: FluxbaseClient,
  queryClient?: QueryClient,
): void {
  setContext(CLIENT_KEY, client);
  setContext(
    QUERY_CLIENT_KEY,
    queryClient ??
      new QueryClient({
        defaultOptions: {
          queries: {
            staleTime: 1000 * 60, // 1 minute
            refetchOnWindowFocus: false,
          },
        },
      }),
  );
}

/**
 * Retrieve the Fluxbase client from Svelte context.
 * Throws if called outside of a component tree that ran `setFluxbaseClient`.
 */
export function getClient(): FluxbaseClient {
  const client = getContext<FluxbaseClient | undefined>(CLIENT_KEY);
  if (!client) {
    throw new Error(
      "getClient() must be used within a component tree that called setFluxbaseClient()",
    );
  }
  return client;
}

/**
 * Retrieve the per-request QueryClient from Svelte context.
 * Throws if called outside of a component tree that ran `setFluxbaseClient`.
 */
export function getQueryClient(): QueryClient {
  const qc = getContext<QueryClient | undefined>(QUERY_CLIENT_KEY);
  if (!qc) {
    throw new Error(
      "getQueryClient() must be used within a component tree that called setFluxbaseClient()",
    );
  }
  return qc;
}
