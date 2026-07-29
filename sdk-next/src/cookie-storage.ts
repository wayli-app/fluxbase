/**
 * Cookie-backed `StorageAdapter` for Next.js SSR.
 *
 * Bridges the core SDK's injectable storage seam to Next.js's `cookies()` API
 * (from `next/headers`) so the auth session lives in an httpOnly cookie.
 *
 * @example
 * ```ts
 * // app/actions.ts or a server component
 * import { cookies } from 'next/headers'
 * import { createClient } from '@nimbleflux/fluxbase-sdk'
 * import { createCookieStorage } from '@nimbleflux/fluxbase-sdk-next'
 *
 * const cookieStore = await cookies()
 * const client = createClient({
 *   url: process.env.FLUXBASE_URL!,
 *   auth: { storage: createCookieStorage(cookieStore) },
 * })
 * ```
 */

import type { StorageAdapter } from "@nimbleflux/fluxbase-sdk";

/**
 * The subset of Next.js's `ReadonlyRequestCookies` / `ResponseCookies` API
 * this adapter depends on. Kept structural to avoid importing `next` at
 * runtime from this framework-agnostic shim.
 */
export interface NextCookies {
  get(name: string): { value?: string } | undefined;
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
  delete(name: string): void;
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
 * Build a `StorageAdapter` backed by Next.js's `cookies()`.
 */
export function createCookieStorage(
  cookies: NextCookies,
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
      return cookies.get(key)?.value ?? null;
    },
    setItem(key: string, value: string): void {
      cookies.set(key, value, cookieOpts);
    },
    removeItem(key: string): void {
      cookies.delete(key);
    },
  };
}
