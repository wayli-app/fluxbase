/**
 * Cookie-backed `StorageAdapter` for Vue / Nuxt SSR.
 *
 * Bridges the core SDK's injectable storage seam to Nuxt's `useCookie` / the
 * h3 event's cookie API so the auth session lives in an httpOnly cookie.
 *
 * @example
 * ```ts
 * // server middleware or plugin
 * import { createClient } from '@nimbleflux/fluxbase-sdk'
 * import { createCookieStorage } from '@nimbleflux/fluxbase-sdk-vue'
 *
 * const client = createClient({
 *   url: process.env.FLUXBASE_URL!,
 *   auth: { storage: createCookieStorage(event) },
 * })
 * ```
 */

import type { StorageAdapter } from "@nimbleflux/fluxbase-sdk";

/**
 * Minimal interface over an h3 `H3Event` (or any cookie bag) that this adapter
 * needs. Kept structural so it works against Nuxt's event without importing
 * h3 at runtime.
 */
export interface NuxtCookieEvent {
  /** Read a cookie value. */
  getCookie(name: string): string | undefined;
  /** Set a cookie with options. */
  setCookie(
    name: string,
    value: string,
    opts?: {
      path?: string;
      httpOnly?: boolean;
      sameSite?: "lax" | "strict" | "none";
      secure?: boolean;
      maxAge?: number;
    },
  ): void;
  /** Delete a cookie. */
  deleteCookie(name: string, opts?: { path?: string }): void;
}

export interface CookieStorageOptions {
  /** Cookie path. @default "/" */
  path?: string;
  /** httpOnly so the cookie is not readable from client JS. @default true */
  httpOnly?: boolean;
  /** SameSite attribute. @default "lax" */
  sameSite?: "lax" | "strict" | "none";
  /** Require HTTPS. @default process.env.NODE_ENV === 'production' */
  secure?: boolean;
  /** Max age in seconds. @default undefined (session cookie) */
  maxAge?: number;
}

/**
 * Build a `StorageAdapter` backed by a Nuxt/h3 cookie event.
 */
export function createCookieStorage(
  event: NuxtCookieEvent,
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
      return event.getCookie(key) ?? null;
    },
    setItem(key: string, value: string): void {
      event.setCookie(key, value, cookieOpts);
    },
    removeItem(key: string): void {
      event.deleteCookie(key, { path });
    },
  };
}
