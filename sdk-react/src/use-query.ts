/**
 * Database query hooks for Fluxbase SDK
 */

import { useQuery, useMutation, useQueryClient, type UseQueryOptions } from '@tanstack/react-query'
import { useFluxbaseClient } from './context'
import type { QueryBuilder } from '@fluxbase/sdk'

export interface UseFluxbaseQueryOptions<T> extends Omit<UseQueryOptions<T[], Error>, 'queryKey' | 'queryFn'> {
  /**
   * Custom query key. If not provided, will use table name and filters.
   */
  queryKey?: unknown[]
}

/**
 * Hook to execute a database query
 * @param buildQuery - Function that builds and returns the query
 * @param options - React Query options
 */
export function useFluxbaseQuery<T = any>(
  buildQuery: (client: ReturnType<typeof useFluxbaseClient>) => QueryBuilder<T>,
  options?: UseFluxbaseQueryOptions<T>
) {
  const client = useFluxbaseClient()

  // Build a stable query key
  const queryKey = options?.queryKey || ['fluxbase', 'query', buildQuery.toString()]

  return useQuery({
    queryKey,
    queryFn: async () => {
      const query = buildQuery(client)
      const { data, error } = await query.execute()

      if (error) {
        throw error
      }

      return (Array.isArray(data) ? data : data ? [data] : []) as T[]
    },
    ...options,
  })
}

/**
 * Hook for table queries with a simpler API
 * @param table - Table name
 * @param buildQuery - Function to build the query
 */
export function useTable<T = any>(
  table: string,
  buildQuery?: (query: QueryBuilder<T>) => QueryBuilder<T>,
  options?: UseFluxbaseQueryOptions<T>
) {
  const client = useFluxbaseClient()

  return useFluxbaseQuery(
    (client) => {
      const query = client.from<T>(table)
      return buildQuery ? buildQuery(query) : query
    },
    {
      ...options,
      queryKey: options?.queryKey || ['fluxbase', 'table', table, buildQuery?.toString()],
    }
  )
}

/**
 * Hook to insert data into a table
 */
export function useInsert<T = any>(table: string) {
  const client = useFluxbaseClient()
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (data: Partial<T> | Partial<T>[]) => {
      const query = client.from<T>(table)
      const { data: result, error } = await query.insert(data as Partial<T>)

      if (error) {
        throw error
      }

      return result
    },
    onSuccess: () => {
      // Invalidate all queries for this table
      queryClient.invalidateQueries({ queryKey: ['fluxbase', 'table', table] })
    },
  })
}

/**
 * Hook to update data in a table
 */
export function useUpdate<T = any>(table: string) {
  const client = useFluxbaseClient()
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (params: { data: Partial<T>; buildQuery: (query: QueryBuilder<T>) => QueryBuilder<T> }) => {
      const query = client.from<T>(table)
      const builtQuery = params.buildQuery(query)
      const { data: result, error } = await builtQuery.update(params.data)

      if (error) {
        throw error
      }

      return result
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['fluxbase', 'table', table] })
    },
  })
}

/**
 * Hook to upsert data into a table
 */
export function useUpsert<T = any>(table: string) {
  const client = useFluxbaseClient()
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (data: Partial<T> | Partial<T>[]) => {
      const query = client.from<T>(table)
      const { data: result, error } = await query.upsert(data as Partial<T>)

      if (error) {
        throw error
      }

      return result
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['fluxbase', 'table', table] })
    },
  })
}

/**
 * Hook to delete data from a table
 */
export function useDelete<T = any>(table: string) {
  const client = useFluxbaseClient()
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (buildQuery: (query: QueryBuilder<T>) => QueryBuilder<T>) => {
      const query = client.from<T>(table)
      const builtQuery = buildQuery(query)
      const { error } = await builtQuery.delete()

      if (error) {
        throw error
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['fluxbase', 'table', table] })
    },
  })
}
