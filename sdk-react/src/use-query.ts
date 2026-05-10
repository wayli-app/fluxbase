/**
 * Database query hooks for Fluxbase SDK
 */

import {
  useQuery,
  useMutation,
  useQueryClient,
  type UseQueryOptions,
} from "@tanstack/react-query";
import { useFluxbaseClient } from "./context";
import type { QueryBuilder } from "@nimbleflux/fluxbase-sdk";

export interface UseFluxbaseQueryOptions<T> extends Omit<
  UseQueryOptions<T[], Error>,
  "queryKey" | "queryFn"
> {
  /**
   * Custom query key. If not provided, will use table name and filters.
   */
  queryKey?: unknown[];
}

/**
 * Hook to execute a database query
 * @param buildQuery - Function that builds and returns the query
 * @param options - React Query options
 *
 * IMPORTANT: You must provide a stable `queryKey` in options for proper caching.
 * Without a custom queryKey, each render may create a new cache entry.
 *
 * @example
 * ```tsx
 * // Always provide a queryKey for stable caching
 * useFluxbaseQuery(
 *   (client) => client.from('users').select('*'),
 *   { queryKey: ['users', 'all'] }
 * )
 * ```
 */
export function useFluxbaseQuery<T = any>(
  buildQuery: (client: ReturnType<typeof useFluxbaseClient>) => QueryBuilder<T>,
  options?: UseFluxbaseQueryOptions<T>,
) {
  const client = useFluxbaseClient();
  const tenantId = client.getTenantId();

  const { queryKey: customKey, ...queryOptions } = options || {};

  // Require queryKey for stable caching - function.toString() is not reliable
  // as it can vary between renders for inline functions
  if (!customKey) {
    console.warn(
      "[useFluxbaseQuery] No queryKey provided. This may cause cache misses. " +
        "Please provide a stable queryKey in options.",
    );
  }

  const queryKey = customKey
    ? ["fluxbase", tenantId ?? null, ...customKey]
    : ["fluxbase", tenantId ?? null, "query", "unstable"];

  return useQuery({
    queryKey,
    queryFn: async () => {
      const query = buildQuery(client);
      const { data, error } = await query.execute();

      if (error) {
        throw error;
      }

      return (Array.isArray(data) ? data : data ? [data] : []) as T[];
    },
    ...queryOptions,
  });
}

/**
 * Hook for table queries with a simpler API
 * @param table - Table name
 * @param buildQuery - Optional function to build the query (e.g., add filters)
 * @param options - Query options including a stable queryKey
 *
 * NOTE: When using buildQuery with filters, provide a custom queryKey that includes
 * the filter values to ensure proper caching.
 *
 * @example
 * ```tsx
 * // Simple query - queryKey is auto-generated from table name
 * useTable('users')
 *
 * // With filters - provide queryKey including filter values
 * useTable('users',
 *   (q) => q.eq('status', 'active'),
 *   { queryKey: ['users', 'active'] }
 * )
 * ```
 */
export function useTable<T = any>(
  table: string,
  buildQuery?: (query: QueryBuilder<T>) => QueryBuilder<T>,
  options?: UseFluxbaseQueryOptions<T>,
) {
  // Generate a stable base queryKey from table name
  // When buildQuery is provided without a custom queryKey, warn about potential cache issues
  if (buildQuery && !options?.queryKey) {
    console.warn(
      `[useTable] Using buildQuery without a custom queryKey for table "${table}". ` +
        "This may cause cache misses. Provide a queryKey that includes your filter values.",
    );
  }

  return useFluxbaseQuery(
    (client) => {
      const query = client.from<T>(table);
      return buildQuery ? buildQuery(query) : query;
    },
    {
      ...options,
      queryKey: options?.queryKey || ["table", table],
    },
  );
}

/**
 * Hook to insert data into a table
 */
export function useInsert<T = any>(table: string) {
  const client = useFluxbaseClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: Partial<T> | Partial<T>[]) => {
      const query = client.from<T>(table);
      const { data: result, error } = await query.insert(data as Partial<T>);

      if (error) {
        throw error;
      }

      return result;
    },
    onSuccess: () => {
      const tenantId = client.getTenantId();
      queryClient.invalidateQueries({
        queryKey: ["fluxbase", tenantId ?? null, "table", table],
      });
    },
  });
}

/**
 * Hook to update data in a table
 */
export function useUpdate<T = any>(table: string) {
  const client = useFluxbaseClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (params: {
      data: Partial<T>;
      buildQuery: (query: QueryBuilder<T>) => QueryBuilder<T>;
    }) => {
      const query = client.from<T>(table);
      const builtQuery = params.buildQuery(query);
      const { data: result, error } = await builtQuery.update(params.data);

      if (error) {
        throw error;
      }

      return result;
    },
    onSuccess: () => {
      const tenantId = client.getTenantId();
      queryClient.invalidateQueries({
        queryKey: ["fluxbase", tenantId ?? null, "table", table],
      });
    },
  });
}

/**
 * Hook to upsert data into a table
 */
export function useUpsert<T = any>(table: string) {
  const client = useFluxbaseClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: Partial<T> | Partial<T>[]) => {
      const query = client.from<T>(table);
      const { data: result, error } = await query.upsert(data as Partial<T>);

      if (error) {
        throw error;
      }

      return result;
    },
    onSuccess: () => {
      const tenantId = client.getTenantId();
      queryClient.invalidateQueries({
        queryKey: ["fluxbase", tenantId ?? null, "table", table],
      });
    },
  });
}

/**
 * Hook to delete data from a table
 */
export function useDelete<T = any>(table: string) {
  const client = useFluxbaseClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (
      buildQuery: (query: QueryBuilder<T>) => QueryBuilder<T>,
    ) => {
      const query = client.from<T>(table);
      const builtQuery = buildQuery(query);
      const { error } = await builtQuery.delete();

      if (error) {
        throw error;
      }
    },
    onSuccess: () => {
      const tenantId = client.getTenantId();
      queryClient.invalidateQueries({
        queryKey: ["fluxbase", tenantId ?? null, "table", table],
      });
    },
  });
}
