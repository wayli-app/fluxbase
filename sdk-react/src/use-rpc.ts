/**
 * React hooks for RPC (Remote Procedure Calls)
 * Call PostgreSQL functions with React Query integration
 */

import { useQuery, useMutation, useQueryClient, type UseQueryOptions, type UseMutationOptions } from '@tanstack/react-query'
import { useFluxbaseClient } from './context'
import type { PostgrestResponse } from '@fluxbase/sdk'

/**
 * Hook to call a PostgreSQL function and cache the result
 *
 * @example
 * ```tsx
 * const { data, isLoading, error } = useRPC(
 *   'calculate_total',
 *   { order_id: 123 },
 *   { enabled: !!orderId }
 * )
 * ```
 */
export function useRPC<TData = unknown, TParams extends Record<string, unknown> = Record<string, unknown>>(
  functionName: string,
  params?: TParams,
  options?: Omit<UseQueryOptions<TData, Error>, 'queryKey' | 'queryFn'>
) {
  const client = useFluxbaseClient()

  return useQuery<TData, Error>({
    queryKey: ['rpc', functionName, params],
    queryFn: async () => {
      const { data, error } = await client.rpc<TData>(functionName, params)
      if (error) {
        throw new Error(error.message)
      }
      return data as TData
    },
    ...options,
  })
}

/**
 * Hook to create a mutation for calling PostgreSQL functions
 * Useful for functions that modify data
 *
 * @example
 * ```tsx
 * const createOrder = useRPCMutation('create_order')
 *
 * const handleSubmit = async () => {
 *   await createOrder.mutateAsync({
 *     user_id: 123,
 *     items: [{ product_id: 1, quantity: 2 }]
 *   })
 * }
 * ```
 */
export function useRPCMutation<TData = unknown, TParams extends Record<string, unknown> = Record<string, unknown>>(
  functionName: string,
  options?: Omit<UseMutationOptions<TData, Error, TParams>, 'mutationFn'>
) {
  const client = useFluxbaseClient()

  return useMutation<TData, Error, TParams>({
    mutationFn: async (params: TParams) => {
      const { data, error } = await client.rpc<TData>(functionName, params)
      if (error) {
        throw new Error(error.message)
      }
      return data as TData
    },
    ...options,
  })
}

/**
 * Hook to call multiple RPC functions in parallel
 *
 * @example
 * ```tsx
 * const { data, isLoading } = useRPCBatch([
 *   { name: 'get_user_stats', params: { user_id: 123 } },
 *   { name: 'get_recent_orders', params: { limit: 10 } },
 * ])
 * ```
 */
export function useRPCBatch<TData = unknown>(
  calls: Array<{ name: string; params?: Record<string, unknown> }>,
  options?: Omit<UseQueryOptions<TData[], Error, TData[], readonly unknown[]>, 'queryKey' | 'queryFn'>
) {
  const client = useFluxbaseClient()

  return useQuery({
    queryKey: ['rpc-batch', calls] as const,
    queryFn: async () => {
      const results = await Promise.all(
        calls.map(async ({ name, params }) => {
          const { data, error } = await client.rpc<TData>(name, params)
          if (error) {
            throw new Error(`${name}: ${error.message}`)
          }
          return data
        })
      )
      return results as TData[]
    },
    ...options,
  })
}
