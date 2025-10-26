/**
 * Realtime subscription hooks for Fluxbase SDK
 */

import { useEffect, useRef } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { useFluxbaseClient } from './context'
import type { RealtimeCallback, RealtimeChangePayload } from '@fluxbase/sdk'

export interface UseRealtimeOptions {
  /**
   * The channel name (e.g., 'table:public.products')
   */
  channel: string

  /**
   * Event type to listen for ('INSERT', 'UPDATE', 'DELETE', or '*' for all)
   */
  event?: 'INSERT' | 'UPDATE' | 'DELETE' | '*'

  /**
   * Callback function when an event is received
   */
  callback?: RealtimeCallback

  /**
   * Whether to automatically invalidate queries for the table
   * Default: true
   */
  autoInvalidate?: boolean

  /**
   * Custom query key to invalidate (if autoInvalidate is true)
   * Default: ['fluxbase', 'table', tableName]
   */
  invalidateKey?: unknown[]

  /**
   * Whether the subscription is enabled
   * Default: true
   */
  enabled?: boolean
}

/**
 * Hook to subscribe to realtime changes for a channel
 */
export function useRealtime(options: UseRealtimeOptions) {
  const client = useFluxbaseClient()
  const queryClient = useQueryClient()
  const channelRef = useRef<ReturnType<typeof client.realtime.channel> | null>(null)

  const {
    channel: channelName,
    event = '*',
    callback,
    autoInvalidate = true,
    invalidateKey,
    enabled = true,
  } = options

  useEffect(() => {
    if (!enabled) {
      return
    }

    // Create channel and subscribe
    const channel = client.realtime.channel(channelName)
    channelRef.current = channel

    const handleChange = (payload: RealtimeChangePayload) => {
      // Call user callback
      if (callback) {
        callback(payload)
      }

      // Auto-invalidate queries if enabled
      if (autoInvalidate) {
        // Extract table name from channel (e.g., 'table:public.products' -> 'public.products')
        const tableName = channelName.replace(/^table:/, '')

        const key = invalidateKey || ['fluxbase', 'table', tableName]
        queryClient.invalidateQueries({ queryKey: key })
      }
    }

    channel.on(event, handleChange).subscribe()

    return () => {
      channel.unsubscribe()
      channelRef.current = null
    }
  }, [client, channelName, event, callback, autoInvalidate, invalidateKey, queryClient, enabled])

  return {
    channel: channelRef.current,
  }
}

/**
 * Hook to subscribe to a table's changes
 * @param table - Table name (with optional schema, e.g., 'public.products')
 * @param options - Subscription options
 */
export function useTableSubscription(
  table: string,
  options?: Omit<UseRealtimeOptions, 'channel'>
) {
  return useRealtime({
    ...options,
    channel: `table:${table}`,
  })
}

/**
 * Hook to subscribe to INSERT events on a table
 */
export function useTableInserts(
  table: string,
  callback: (payload: RealtimeChangePayload) => void,
  options?: Omit<UseRealtimeOptions, 'channel' | 'event' | 'callback'>
) {
  return useRealtime({
    ...options,
    channel: `table:${table}`,
    event: 'INSERT',
    callback,
  })
}

/**
 * Hook to subscribe to UPDATE events on a table
 */
export function useTableUpdates(
  table: string,
  callback: (payload: RealtimeChangePayload) => void,
  options?: Omit<UseRealtimeOptions, 'channel' | 'event' | 'callback'>
) {
  return useRealtime({
    ...options,
    channel: `table:${table}`,
    event: 'UPDATE',
    callback,
  })
}

/**
 * Hook to subscribe to DELETE events on a table
 */
export function useTableDeletes(
  table: string,
  callback: (payload: RealtimeChangePayload) => void,
  options?: Omit<UseRealtimeOptions, 'channel' | 'event' | 'callback'>
) {
  return useRealtime({
    ...options,
    channel: `table:${table}`,
    event: 'DELETE',
    callback,
  })
}
