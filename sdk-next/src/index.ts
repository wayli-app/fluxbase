/**
 * Fluxbase Next.js SDK
 *
 * Server/client adapters and SSR cookie storage for Fluxbase, built on the
 * core `@nimbleflux/fluxbase-sdk`.
 *
 * This is a scaffold: it provides the SSR-auth foundation (cookie storage,
 * server client factory, client provider). Full React Query hooks (like the
 * React SDK) are a follow-on.
 */

export {
  createCookieStorage,
  type CookieStorageOptions,
  type NextCookies,
} from "./cookie-storage";

export {
  createServerClient,
  type CreateServerClientOptions,
} from "./create-server-client";

export {
  FluxbaseProvider,
  useFluxbaseClient,
  type FluxbaseProviderProps,
} from "./provider";

export type { FluxbaseClient, StorageAdapter } from "@nimbleflux/fluxbase-sdk";
