import { useEffect, useState, useRef, useCallback, useMemo } from 'react'
import { getAccessToken } from '@/lib/auth'
import type { LogEntry, LogCategory, LogLevel } from '../types'
import { STREAM_MAX_ENTRIES } from '../constants'

interface UseLogStreamOptions {
  /** Enable streaming (default: true) */
  enabled?: boolean
  /** Optional filter by category */
  category?: LogCategory
  /** Optional filter by levels */
  levels?: LogLevel[]
  /** Callback when a new log entry arrives */
  onNewLog?: (log: LogEntry) => void
}

interface UseLogStreamResult {
  /** Streamed log entries (newest first) */
  logs: LogEntry[]
  /** Whether WebSocket is connected */
  connected: boolean
  /** Clear all logs */
  clearLogs: () => void
  /** Connection error if any */
  error: Error | null
}

/**
 * Hook for streaming all logs via WebSocket (admin only)
 */
export function useLogStream(
  options: UseLogStreamOptions = {}
): UseLogStreamResult {
  const { enabled = true, category, levels, onNewLog } = options
  const [logs, setLogs] = useState<LogEntry[]>([])
  const [connected, setConnected] = useState(false)
  const [error, setError] = useState<Error | null>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const onNewLogRef = useRef(onNewLog)
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  // Keep callback ref up to date
  useEffect(() => {
    onNewLogRef.current = onNewLog
  }, [onNewLog])

  // Add a new log entry
  const addLog = useCallback(
    (log: LogEntry) => {
      // Apply category filter
      if (category && log.category !== category) return
      // Apply level filter
      if (levels?.length && !levels.includes(log.level)) return

      setLogs((prev) => {
        // Avoid duplicates by ID
        if (prev.some((l) => l.id === log.id)) return prev
        // Add new log at the beginning, limit total
        const updated = [log, ...prev].slice(0, STREAM_MAX_ENTRIES)
        return updated
      })

      onNewLogRef.current?.(log)
    },
    [category, levels]
  )

  // Clear all logs
  const clearLogs = useCallback(() => {
    setLogs([])
  }, [])

  // Get token for WebSocket connection (computed once per render)
  const token = useMemo(() => getAccessToken(), [])

  // Derive token error state instead of setting it in useEffect
  const tokenError = useMemo(
    () => (enabled && !token ? new Error('No auth token available') : null),
    [enabled, token]
  )

  // Connect to WebSocket
  useEffect(() => {
    if (!enabled || !token) {
      return
    }

    // Build WebSocket URL
    const baseUrl =
      import.meta.env.VITE_API_URL || window.location.origin.replace(/^http/, 'ws')
    const wsUrl = `${baseUrl.replace(/^http/, 'ws')}/realtime?token=${encodeURIComponent(token)}`

    let ws: WebSocket

    const connect = () => {
      try {
        ws = new WebSocket(wsUrl)
        wsRef.current = ws

        ws.onopen = () => {
          setConnected(true)
          setError(null)

          // Subscribe to all logs
          const subscribeMsg = {
            type: 'subscribe_all_logs',
            payload: {
              category: category || undefined,
              levels: levels?.length ? levels : undefined,
            },
          }
          ws.send(JSON.stringify(subscribeMsg))
        }

        ws.onmessage = (event) => {
          try {
            const msg = JSON.parse(event.data)

            // Handle different message types
            if (msg.type === 'log_entry' && msg.payload) {
              const log = msg.payload as LogEntry
              addLog(log)
            } else if (msg.type === 'heartbeat') {
              // Ignore heartbeat
            } else if (msg.type === 'error') {
              setError(new Error(msg.error))
            }
          } catch {
            // Ignore malformed messages
          }
        }

        ws.onerror = () => {
          setError(new Error('WebSocket connection error'))
        }

        ws.onclose = (event) => {
          setConnected(false)
          wsRef.current = null

          // Reconnect if not intentionally closed
          if (enabled && event.code !== 1000) {
            reconnectTimeoutRef.current = setTimeout(() => {
              connect()
            }, 2000)
          }
        }
      } catch (e) {
        setError(e as Error)
      }
    }

    connect()

    return () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current)
      }
      if (wsRef.current) {
        wsRef.current.close(1000)
        wsRef.current = null
      }
      setConnected(false)
    }
  }, [enabled, token, category, levels, addLog])

  return {
    logs,
    connected,
    clearLogs,
    error: tokenError || error,
  }
}
