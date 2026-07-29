/**
 * Server-side Fluxbase client factory for Next.js.
 *
 * Builds a client per request, backed by an httpOnly cookie so the session is
 * read from the incoming request and any token refresh is written back to the
 * response cookies. Use in Server Components, Route Handlers, and Server
 * Actions.
 *
 * @example
 * ```ts
 * // app/lib/fluxbase.ts
 * import { createServerClient } from '@nimbleflux/fluxbase-sdk-next'
 *
 * export function getClient() {
 *   return createServerClient(process.env.FLUXBASE_URL!)
 * }
 * ```
 */

import { createClient, type FluxbaseClient } from "@nimbleflux/fluxbase-sdk";
import { createCookieStorage, type NextCookies } from "./cookie-storage";

export interface CreateServerClientOptions {
  /** The Fluxbase anon/client key. Falls back to FLUXBASE_ANON_KEY env var. */
  key?: string;
  /** Override the per-request cookie store (defaults to next/headers cookies()). */
  cookies?: NextCookies;
}

/**
 * Create a Fluxbase client for use in Next.js server code. Reads and writes the
 * auth session via httpOnly cookies.
 *
 * Pass `cookies()` from `next/headers` explicitly, or omit it and call this
 * inside a request scope where `next/headers` is available.
 */
export function createServerClient(
  url: string,
  options: CreateServerClientOptions = {},
): FluxbaseClient {
  let cookies: NextCookies | undefined = options.cookies;

  if (!cookies) {
    // Lazily import next/headers so this module is testable without Next.
    try {
      // eslint-disable-next-line @typescript-eslint/no-var-requires
      const { cookies: nextCookies } = require("next/headers");
      // In Next 15 `cookies()` returns a promise; in 14 it's sync. Support both.
      const result = typeof nextCookies === "function" ? nextCookies() : nextCookies;
      cookies = result as unknown as NextCookies;
    } catch {
      throw new Error(
        "createServerClient could not read cookies(). Pass a `cookies` option " +
          "explicitly, or ensure this is called within a Next.js request scope.",
      );
    }
  }

  const key =
    options.key ??
    (typeof process !== "undefined" ? process.env?.FLUXBASE_ANON_KEY : undefined);

  if (!key) {
    throw new Error(
      "createServerClient requires a Fluxbase anon key. Pass `options.key` or set FLUXBASE_ANON_KEY.",
    );
  }

  return createClient(
    url,
    key,
    { auth: { storage: createCookieStorage(cookies) } },
  );
}
