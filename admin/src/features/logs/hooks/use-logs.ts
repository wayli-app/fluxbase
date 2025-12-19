import { useQuery } from '@tanstack/react-query'
import {
  logsApi,
  type LogQueryOptionsAPI,
  type LogQueryResultAPI,
  type LogStatsAPI,
} from '@/lib/api'

/**
 * Hook for fetching logs with filters and pagination
 */
export function useLogs(
  options: LogQueryOptionsAPI,
  enabled = true,
  refetchInterval?: number | false
) {
  return useQuery<LogQueryResultAPI>({
    queryKey: ['logs', options],
    queryFn: () =>
      logsApi.query({
        ...options,
        levels: options.levels?.length ? options.levels : undefined,
      }),
    staleTime: 10000, // 10 seconds
    refetchOnWindowFocus: false,
    enabled,
    refetchInterval,
  })
}

/**
 * Hook for fetching log statistics
 */
export function useLogStats() {
  return useQuery<LogStatsAPI>({
    queryKey: ['log-stats'],
    queryFn: logsApi.getStats,
    staleTime: 30000, // 30 seconds
    refetchOnWindowFocus: false,
  })
}
