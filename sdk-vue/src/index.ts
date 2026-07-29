/**
 * Fluxbase Vue SDK
 *
 * Vue/Nuxt composables and SSR cookie storage for Fluxbase, built on the core
 * `@nimbleflux/fluxbase-sdk`.
 *
 * This is a scaffold: it provides the SSR-auth foundation (cookie storage,
 * provide/inject composable). Full reactive composables (like the React/Svelte
 * hooks) are a follow-on.
 */

export {
  createCookieStorage,
  type CookieStorageOptions,
  type NuxtCookieEvent,
} from "./cookie-storage";

export {
  FLUXBASE_CLIENT_KEY,
  useFluxbaseClient,
} from "./composables";

export type { FluxbaseClient, StorageAdapter } from "@nimbleflux/fluxbase-sdk";
