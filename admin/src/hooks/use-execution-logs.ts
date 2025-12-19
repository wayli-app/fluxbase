/**
 * Hook for fetching and subscribing to real-time execution logs
 * Works with edge functions, jobs, and RPC executions
 */
import { useEffect, useState, useRef, useCallback } from 'react'
import type { ExecutionLogEvent, ExecutionType } from '@fluxbase/sdk'
import { fluxbaseClient } from '@/lib/fluxbase-client'
import { logsApi } from '@/lib/api'

export type { ExecutionType }

// Log level type compatible with both SDK and API types
// SDK uses: 'debug' | 'info' | 'warn' | 'error'
// API uses: 'debug' | 'info' | 'warning' | 'error' | 'fatal'
export type ExecutionLogLevel =
  | 'debug'
  | 'info'
  | 'warn'
  | 'warning'
  | 'error'
  | 'fatal'

export interface ExecutionLog {
  id: string
  execution_id: string
  line_number: number
  level: ExecutionLogLevel
  message: string
  created_at: string
  fields?: Record<string, unknown>
}

interface UseExecutionLogsOptions {
  /** The execution ID to fetch logs for */
  executionId: string | null
  /** The type of execution (function, job, rpc) */
  executionType: ExecutionType
  /** Only subscribe when enabled (e.g., when dialog is open) */
  enabled?: boolean
  /** Callback when a new log entry arrives */
  onNewLog?: (log: ExecutionLog) => void
}

interface UseExecutionLogsResult {
  /** The list of log entries */
  logs: ExecutionLog[]
  /** Whether initial logs are being loaded */
  loading: boolean
  /** Any error that occurred */
  error: Error | null
  /** Manually refetch logs */
  refetch: () => Promise<void>
}

/**
 * Hook for fetching and subscribing to execution logs in real-time.
 *
 * @example
 * ```tsx
 * const { logs, loading } = useExecutionLogs({
 *   executionId: selectedJobId,
 *   executionType: 'job',
 *   enabled: showJobDetails,
 *   onNewLog: (log) => console.log('New log:', log),
 * })
 * ```
 */
export function useExecutionLogs({
  executionId,
  executionType,
  enabled = true,
  onNewLog,
}: UseExecutionLogsOptions): UseExecutionLogsResult {
  const [logs, setLogs] = useState<ExecutionLog[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<Error | null>(null)
  const channelRef = useRef<ReturnType<
    typeof fluxbaseClient.realtime.executionLogs
  > | null>(null)
  const onNewLogRef = useRef(onNewLog)

  // Keep callback ref up to date
  useEffect(() => {
    onNewLogRef.current = onNewLog
  }, [onNewLog])

  // Fetch initial logs from the API
  const fetchLogs = useCallback(async () => {
    if (!executionId) return

    setLoading(true)
    setError(null)

    try {
      const data = await logsApi.getExecutionLogs(executionId)
      const entries = (data.entries || []).map((entry) => ({
        id: entry.id,
        execution_id: entry.execution_id || executionId,
        line_number: entry.line_number || 0,
        level: entry.level as ExecutionLog['level'],
        message: entry.message,
        created_at: entry.timestamp,
        fields: entry.fields,
      }))

      // Sort by line number
      entries.sort(
        (a: ExecutionLog, b: ExecutionLog) => a.line_number - b.line_number
      )
      setLogs(entries)
    } catch (err) {
      setError(err as Error)
    } finally {
      setLoading(false)
    }
  }, [executionId])

  // Subscribe to real-time updates
  useEffect(() => {
    if (!executionId || !enabled) {
      return
    }

    // Fetch initial logs
    fetchLogs()

    // Subscribe to real-time log updates via SDK
    const channel = fluxbaseClient.realtime
      .executionLogs(executionId, executionType)
      .onLog((event: ExecutionLogEvent) => {
        const newLog: ExecutionLog = {
          id: `${event.execution_id}-${event.line_number}`,
          execution_id: event.execution_id,
          line_number: event.line_number,
          level: event.level,
          message: event.message,
          created_at: event.timestamp,
        }

        setLogs((prev) => {
          // Avoid duplicates by line_number
          if (prev.some((l) => l.line_number === newLog.line_number)) {
            return prev
          }
          // Insert in sorted order
          const updated = [...prev, newLog]
          updated.sort((a, b) => a.line_number - b.line_number)
          return updated
        })

        // Notify callback
        onNewLogRef.current?.(newLog)
      })
      .subscribe()

    channelRef.current = channel

    return () => {
      channel.unsubscribe()
      channelRef.current = null
    }
  }, [executionId, executionType, enabled, fetchLogs])

  // Clear logs when execution changes
  useEffect(() => {
    if (!executionId) {
      setLogs([])
      setError(null)
    }
  }, [executionId])

  return {
    logs,
    loading,
    error,
    refetch: fetchLogs,
  }
}
