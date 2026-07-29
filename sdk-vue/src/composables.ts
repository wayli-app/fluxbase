/**
 * Vue composables for the Fluxbase SDK.
 *
 * Provide the client once (in `app.provide`) and inject it in setup functions.
 *
 * @example
 * ```ts
 * // main.ts
 * import { createApp } from 'vue'
 * import { createClient } from '@nimbleflux/fluxbase-sdk'
 * import { FLUXBASE_CLIENT_KEY } from '@nimbleflux/fluxbase-sdk-vue'
 *
 * const app = createApp(App)
 * app.provide(FLUXBASE_CLIENT_KEY, createClient({ url: 'http://localhost:8080' }))
 * ```
 *
 * @example
 * ```vue
 * <script setup lang="ts">
 * import { useFluxbaseClient } from '@nimbleflux/fluxbase-sdk-vue'
 * const client = useFluxbaseClient()
 * const { data } = await client.from('products').select('*').execute()
 * </script>
 * ```
 */

import { inject, type InjectionKey } from "vue";
import type { FluxbaseClient } from "@nimbleflux/fluxbase-sdk";

/** The provide/inject key for the Fluxbase client. */
export const FLUXBASE_CLIENT_KEY: InjectionKey<FluxbaseClient> =
  Symbol("fluxbase-client");

/**
 * Inject the Fluxbase client provided via `app.provide(FLUXBASE_CLIENT_KEY, ...)`.
 * Throws if no client was provided.
 */
export function useFluxbaseClient(): FluxbaseClient {
  const client = inject(FLUXBASE_CLIENT_KEY);
  if (!client) {
    throw new Error(
      "useFluxbaseClient() requires a client provided via " +
        "app.provide(FLUXBASE_CLIENT_KEY, createClient(...)).",
    );
  }
  return client;
}
