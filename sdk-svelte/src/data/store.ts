/**
 * Database query stores for the Fluxbase Svelte SDK.
 *
 * Mirrors `sdk-react/src/use-query.ts`: `fluxbaseQuery` / `table` for reads,
 * and `insert` / `update` / `upsert` / `delete` mutations that invalidate the
 * affected table cache on success.
 *
 * Each factory passes its `queryClient` explicitly to `createQuery` /
 * `createMutation` (their optional 2nd arg) so the stores work both inside a
 * Svelte component tree (via context) and standalone in tests/SSR.
 */

import { createQuery, createMutation } from "@tanstack/svelte-query";
import type { QueryClient } from "@tanstack/svelte-query";
import type { FluxbaseClient, QueryBuilder } from "@nimbleflux/fluxbase-sdk";
import type { StoreDeps } from "../auth/store";

export interface FluxbaseQueryOptions<T>
  extends Omit<
    import("@tanstack/svelte-query").CreateQueryOptions<T[], Error>,
    "queryKey" | "queryFn"
  > {
  /** Stable query key. Strongly recommended for reliable caching. */
  queryKey?: unknown[];
}

/**
 * Execute a Fluxbase query reactively. Always pass a stable `queryKey`.
 */
export function createFluxbaseQuery<T = any>(
  { client, queryClient }: StoreDeps,
  buildQuery: (client: FluxbaseClient) => QueryBuilder<T>,
  options?: FluxbaseQueryOptions<T>,
) {
  const { queryKey: customKey, ...queryOptions } = options || {};
  if (!customKey) {
    console.warn(
      "[fluxbaseQuery] No queryKey provided — caching may be unstable. Pass a stable queryKey.",
    );
  }
  const tenantId = client.getTenantId();
  const queryKey = customKey
    ? ["fluxbase", tenantId ?? null, ...customKey]
    : ["fluxbase", tenantId ?? null, "query", "unstable"];

  return createQuery(
    () => ({
      queryKey,
      queryFn: async () => {
        const query = buildQuery(client);
        const { data, error } = await query.execute();
        if (error) throw error;
        return (Array.isArray(data) ? data : data ? [data] : []) as T[];
      },
      ...queryOptions,
    }),
    () => queryClient,
  );
}

/**
 * Read a table reactively.
 *
 * @example
 * ```svelte
 * <script lang="ts">
 *   import { table } from '@nimbleflux/fluxbase-sdk-svelte'
 *   const products = table('products', (q) => q.eq('active', true), { queryKey: ['products','active'] })
 * </script>
 * {#each $products.data ?? [] as p}{p.name}{/each}
 * ```
 */
export function createTableQuery<T = any>(
  deps: StoreDeps,
  tableName: string,
  buildQuery?: (query: QueryBuilder<T>) => QueryBuilder<T>,
  options?: FluxbaseQueryOptions<T>,
) {
  if (buildQuery && !options?.queryKey) {
    console.warn(
      `[table] Using buildQuery without a custom queryKey for "${tableName}" may cause cache misses.`,
    );
  }
  return createFluxbaseQuery<T>(
    deps,
    (client) => {
      const query = client.from<T>(tableName);
      return buildQuery ? buildQuery(query) : query;
    },
    { ...options, queryKey: options?.queryKey || ["table", tableName] },
  );
}

/** Insert rows, then invalidate the table cache. */
export function createInsertMutation<T = any>(
  { client, queryClient }: StoreDeps,
  tableName: string,
) {
  return createMutation(
    () => ({
      mutationFn: async (data: Partial<T> | Partial<T>[]) => {
        const query = client.from<T>(tableName);
        const { data: result, error } = await query.insert(data as Partial<T>);
        if (error) throw error;
        return result;
      },
      onSuccess: () => {
        const tenantId = client.getTenantId();
        queryClient.invalidateQueries({
          queryKey: ["fluxbase", tenantId ?? null, "table", tableName],
        });
      },
    }),
    () => queryClient,
  );
}

/** Update rows, then invalidate the table cache. */
export function createUpdateMutation<T = any>(
  { client, queryClient }: StoreDeps,
  tableName: string,
) {
  return createMutation(
    () => ({
      mutationFn: async (params: {
        data: Partial<T>;
        buildQuery: (query: QueryBuilder<T>) => QueryBuilder<T>;
      }) => {
        const query = client.from<T>(tableName);
        const built = params.buildQuery(query);
        const { data: result, error } = await built.update(params.data);
        if (error) throw error;
        return result;
      },
      onSuccess: () => {
        const tenantId = client.getTenantId();
        queryClient.invalidateQueries({
          queryKey: ["fluxbase", tenantId ?? null, "table", tableName],
        });
      },
    }),
    () => queryClient,
  );
}

/** Upsert rows, then invalidate the table cache. */
export function createUpsertMutation<T = any>(
  { client, queryClient }: StoreDeps,
  tableName: string,
) {
  return createMutation(
    () => ({
      mutationFn: async (data: Partial<T> | Partial<T>[]) => {
        const query = client.from<T>(tableName);
        const { data: result, error } = await query.upsert(data as Partial<T>);
        if (error) throw error;
        return result;
      },
      onSuccess: () => {
        const tenantId = client.getTenantId();
        queryClient.invalidateQueries({
          queryKey: ["fluxbase", tenantId ?? null, "table", tableName],
        });
      },
    }),
    () => queryClient,
  );
}

/** Delete rows, then invalidate the table cache. */
export function createDeleteMutation<T = any>(
  { client, queryClient }: StoreDeps,
  tableName: string,
) {
  return createMutation(
    () => ({
      mutationFn: async (
        buildQuery: (query: QueryBuilder<T>) => QueryBuilder<T>,
      ) => {
        const query = client.from<T>(tableName);
        const built = buildQuery(query);
        const { error } = await built.delete();
        if (error) throw error;
      },
      onSuccess: () => {
        const tenantId = client.getTenantId();
        queryClient.invalidateQueries({
          queryKey: ["fluxbase", tenantId ?? null, "table", tableName],
        });
      },
    }),
    () => queryClient,
  );
}
