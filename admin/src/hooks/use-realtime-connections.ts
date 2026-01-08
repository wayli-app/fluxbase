/**
 * Hook for subscribing to real-time connection events
 * Provides live updates when WebSocket connections are established or terminated
 */
import { useEffect, useState, useRef } from 'react'
import { fluxbaseClient } from '@/lib/fluxbase-client'

export interface RealtimeConnection {
  id: string
  user_id: string | null
  email: string | null
  display_name?: string | null
  remote_addr: string
  connected_at: string
}

interface ConnectionEvent {
  type: 'connected' | 'disconnected'
  id: string
  user_id: string | null
  email: string | null
  display_name?: string | null
  remote_addr: string
  connected_at: string
  timestamp: string
}

interface UseRealtimeConnectionsOptions {
  /** Initial connections from the stats API */
  initialConnections: RealtimeConnection[]
  /** Only subscribe when enabled (e.g., when page is visible) */
  enabled?: boolean
  /** Callback when a connection is added */
  onConnectionAdded?: (connection: RealtimeConnection) => void
  /** Callback when a connection is removed */
  onConnectionRemoved?: (connectionId: string) => void
  /** Callback when WebSocket subscription is established */
  onSubscribed?: () => void
}

interface UseRealtimeConnectionsResult {
  /** Current list of active connections */
  connections: RealtimeConnection[]
  /** Total number of connections */
  totalConnections: number
  /** Whether the WebSocket subscription is active */
  isSubscribed: boolean
  /** Any error that occurred during subscription */
  error: Error | null
}

/**
 * Hook for subscribing to real-time connection events.
 * Maintains a live list of active WebSocket connections.
 *
 * @example
 * ```tsx
 * const { connections, totalConnections, isSubscribed } = useRealtimeConnections({
 *   initialConnections: statsData.connections,
 *   enabled: true,
 *   onConnectionAdded: (conn) => console.log('New connection:', conn.id),
 *   onConnectionRemoved: (id) => console.log('Disconnected:', id),
 * })
 * ```
 */
export function useRealtimeConnections({
  initialConnections,
  enabled = true,
  onConnectionAdded,
  onConnectionRemoved,
  onSubscribed,
}: UseRealtimeConnectionsOptions): UseRealtimeConnectionsResult {
  const [connections, setConnections] =
    useState<RealtimeConnection[]>(initialConnections)
  const [isSubscribed, setIsSubscribed] = useState(false)
  const [error, setError] = useState<Error | null>(null)
  const channelRef = useRef<ReturnType<
    typeof fluxbaseClient.realtime.channel
  > | null>(null)
  const onConnectionAddedRef = useRef(onConnectionAdded)
  const onConnectionRemovedRef = useRef(onConnectionRemoved)
  const onSubscribedRef = useRef(onSubscribed)

  // Keep callback refs up to date
  useEffect(() => {
    onConnectionAddedRef.current = onConnectionAdded
  }, [onConnectionAdded])

  useEffect(() => {
    onConnectionRemovedRef.current = onConnectionRemoved
  }, [onConnectionRemoved])

  useEffect(() => {
    onSubscribedRef.current = onSubscribed
  }, [onSubscribed])

  // Update connections when initialConnections prop changes
  useEffect(() => {
    setConnections(initialConnections)
  }, [initialConnections])

  // Subscribe to admin connection events
  useEffect(() => {
    if (!enabled) {
      setIsSubscribed(false)
      return
    }

    try {
      // Subscribe to the admin connections channel
      const channel = fluxbaseClient.realtime
        .channel('realtime:admin:connections')
        .on('broadcast', { event: 'connected' }, (payload) => {
          // Payload structure: { event: 'connected', payload: ConnectionEvent }
          const event = payload.payload as ConnectionEvent

          const newConnection: RealtimeConnection = {
            id: event.id,
            user_id: event.user_id,
            email: event.email,
            display_name: event.display_name,
            remote_addr: event.remote_addr,
            connected_at: event.connected_at,
          }

          setConnections((prev) => {
            // Avoid duplicates
            if (prev.some((c) => c.id === newConnection.id)) {
              return prev
            }
            return [...prev, newConnection]
          })

          // Notify callback
          onConnectionAddedRef.current?.(newConnection)
        })
        .on('broadcast', { event: 'disconnected' }, (payload) => {
          // Payload structure: { event: 'disconnected', payload: ConnectionEvent }
          const event = payload.payload as ConnectionEvent
          const connectionId = event.id

          setConnections((prev) => prev.filter((c) => c.id !== connectionId))

          // Notify callback
          onConnectionRemovedRef.current?.(connectionId)
        })
        .subscribe((status) => {
          // Only call onSubscribed when actually subscribed
          if (status === 'SUBSCRIBED') {
            setIsSubscribed(true)
            setError(null)

            // Notify that subscription is ready
            // This allows the parent to fetch initial data after WebSocket connects
            onSubscribedRef.current?.()
          } else if (status === 'CHANNEL_ERROR' || status === 'TIMED_OUT') {
            setError(new Error(`Subscription ${status}`))
            setIsSubscribed(false)
          }
        })

      channelRef.current = channel

      return () => {
        channel.unsubscribe()
        channelRef.current = null
        setIsSubscribed(false)
      }
    } catch (err) {
      setError(err as Error)
      setIsSubscribed(false)
      return undefined
    }
  }, [enabled])

  return {
    connections,
    totalConnections: connections.length,
    isSubscribed,
    error,
  }
}
