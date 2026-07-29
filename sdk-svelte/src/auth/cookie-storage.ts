/**
 * Cookie-backed `StorageAdapter` for SvelteKit SSR.
 *
 * This bridges the core SDK's injectable storage seam (Phase A) to SvelteKit's
 * `cookies()` API so the auth session lives in an httpOnly cookie on the
 * server instead of `localStorage`. Pass the result to `createClient`'s
 * `auth.storage` option.
 *
 * @example
 * ```ts
 * // hooks.server.ts
 * import { createClient } from '@nimbleflux/fluxbase-sdk'
 * import { createCookieStorage } from '@nimbleflux/fluxbase-sdk-svelte'
 * import type { Handle } from '@sveltejs/kit'
 *
 * export const handle: Handle = async ({ event, resolve }) => {
 *   const client = createClient({
 *     url: FLUXBASE_URL,
 *     auth: { storage: createCookieStorage(event.cookies) },
 *   })
 *   event.locals.client = client
 *   return resolve(event)
 * }
 * ```
 */

import type { StorageAdapter } from "@nimbleflux/fluxbase-sdk";

/**
 * The subset of SvelteKit's `Cookies` API this adapter depends on.
 * Kept structural so it works against the real type without importing kit
 * at runtime (the adapter ships in a browser/edge bundle too).
 */
export interface SvelteKitCookies {
  get(name: string): string | undefined;
  set(
    name: string,
    value: string,
    opts?: {
      path?: string;
      httpOnly?: boolean;
      sameSite?: "lax" | "strict" | "none";
      secure?: boolean;
      maxAge?: number;
      expires?: Date;
    },
  ): void;
  delete(name: string, opts?: { path?: string }): void;
}

export interface CookieStorageOptions {
  /** Cookie path. @default "/" */
  path?: string;
  /** Mark the cookie httpOnly so it is not readable from client JS. @default true */
  httpOnly?: boolean;
  /** SameSite attribute. @default "lax" */
  sameSite?: "lax" | "strict" | "none";
  /** Require HTTPS. Defaults to true in production. @default process.env.NODE_ENV === 'production' */
  secure?: boolean;
  /** Max age in seconds. @default undefined (session cookie) */
  maxAge?: number;
}

/**
 * Build a `StorageAdapter` backed by SvelteKit's `cookies()`.
 *
 * Every key the core SDK stores (e.g. `fluxbase.auth.session`) becomes its
 * own cookie. Values larger than a cookie can hold (~4KB) should be avoided;
 * the Fluxbase session JSON is well under that limit.
 */
export function createCookieStorage(
  cookies: SvelteKitCookies,
  options: CookieStorageOptions = {},
): StorageAdapter {
  const {
    path = "/",
    httpOnly = true,
    sameSite = "lax",
    secure = typeof process !== "undefined"
      ? process.env?.NODE_ENV === "production"
      : false,
    maxAge,
  } = options;

  const cookieOpts = { path, httpOnly, sameSite, secure, maxAge };

  return {
    getItem(key: string): string | null {
      const value = cookies.get(key);
      return value ?? null;
    },
    setItem(key: string, value: string): void {
      cookies.set(key, value, cookieOpts);
    },
    removeItem(key: string): void {
      cookies.delete(key, { path });
    },
  };
}
