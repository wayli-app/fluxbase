import { useState, useMemo, useCallback, useRef } from 'react'
import { RefreshCw, Play, Download, Wifi, WifiOff, Radio } from 'lucide-react'
import type { LogQueryOptionsAPI } from '@/lib/api'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Skeleton } from '@/components/ui/skeleton'
import {
  LOG_CATEGORY_CONFIG,
  DEFAULT_PAGE_SIZE,
  TIME_RANGE_PRESETS,
} from '../constants'
import { useLogStream } from '../hooks/use-log-stream'
import { useLogs, useLogStats } from '../hooks/use-logs'
import {
  defaultLogFilters,
  type LogEntry,
  type LogFilters,
  type LogCategory,
  type LogLevel,
} from '../types'
import { LogDetailSheet } from './log-detail-sheet'
import { LogFiltersToolbar } from './log-filters'
import { LogLevelBadge } from './log-level-badge'

export function LogViewer() {
  // Mode state
  const [isLiveMode, setIsLiveMode] = useState(true)

  // Filters state
  const [filters, setFilters] = useState<LogFilters>(defaultLogFilters)

  // Active time preset (in minutes)
  const [activeTimePreset, setActiveTimePreset] = useState<number | null>(null)

  // Pagination state (for historical mode)
  const [page, setPage] = useState(0)
  const pageSize = DEFAULT_PAGE_SIZE

  // Selected log for detail view
  const [selectedLog, setSelectedLog] = useState<LogEntry | null>(null)
  const [showDetail, setShowDetail] = useState(false)

  // Scroll ref for auto-scroll
  const scrollRef = useRef<HTMLDivElement>(null)

  // Build query options for historical mode
  const queryOptions: LogQueryOptionsAPI = useMemo(
    () => ({
      category: filters.category !== 'all' ? filters.category : undefined,
      levels: filters.levels.length > 0 ? filters.levels : undefined,
      search: filters.search || undefined,
      start_time: filters.timeRange.start?.toISOString(),
      end_time: filters.timeRange.end?.toISOString(),
      limit: pageSize,
      offset: page * pageSize,
      sort_asc: false,
    }),
    [filters, page, pageSize]
  )

  // Historical data fetch
  const { data, isLoading, refetch, isFetching } = useLogs(
    queryOptions,
    !isLiveMode
  )

  // Live stream
  const {
    logs: streamLogs,
    connected,
    clearLogs,
    error: streamError,
  } = useLogStream({
    enabled: isLiveMode,
    category:
      filters.category !== 'all'
        ? (filters.category as LogCategory)
        : undefined,
    levels:
      filters.levels.length > 0 ? (filters.levels as LogLevel[]) : undefined,
  })

  // Stats
  const { data: stats } = useLogStats()

  // Logs to display (memoized to prevent useCallback dependency issues)
  const logs = useMemo(
    () => (isLiveMode ? streamLogs : (data?.entries as LogEntry[]) || []),
    [isLiveMode, streamLogs, data?.entries]
  )
  const totalCount = isLiveMode ? streamLogs.length : data?.total_count || 0

  // Handle filter changes with auto-pause when time range is selected
  const handleFiltersChange = useCallback(
    (newFilters: LogFilters) => {
      // Clear live logs when category or levels change
      if (
        newFilters.category !== filters.category ||
        JSON.stringify(newFilters.levels) !== JSON.stringify(filters.levels)
      ) {
        clearLogs()
      }
      setFilters(newFilters)
      setPage(0)
      // Auto-pause when time range is selected (historical mode supports it, live mode doesn't)
      if (newFilters.timeRange.start || newFilters.timeRange.end) {
        setIsLiveMode(false)
      }
    },
    [filters.category, filters.levels, clearLogs]
  )

  // Handle time preset change
  const handleTimePresetChange = useCallback((minutes: number | null) => {
    setActiveTimePreset(minutes)
    setPage(0)
    if (minutes !== null) {
      setIsLiveMode(false)
    }
  }, [])

  // Handle return to live mode
  const handleReturnToLive = useCallback(() => {
    setIsLiveMode(true)
    setActiveTimePreset(null)
    setFilters((prev) => ({
      ...prev,
      timeRange: { start: null, end: null },
    }))
    setPage(0)
    clearLogs()
  }, [clearLogs])

  // Get label for active time preset
  const activeTimePresetLabel = useMemo(() => {
    if (!activeTimePreset) return null
    const preset = TIME_RANGE_PRESETS.find(
      (p) => p.minutes === activeTimePreset
    )
    return preset?.label || null
  }, [activeTimePreset])

  // Open log detail
  const openLogDetail = (log: LogEntry) => {
    setSelectedLog(log)
    setShowDetail(true)
  }

  // Export logs as JSON
  const exportLogs = useCallback(() => {
    const exportData = {
      exported_at: new Date().toISOString(),
      filters: {
        category: filters.category,
        levels: filters.levels,
        search: filters.search,
        time_range: filters.timeRange,
      },
      mode: isLiveMode ? 'live' : 'historical',
      count: logs.length,
      logs: logs,
    }

    const blob = new Blob([JSON.stringify(exportData, null, 2)], {
      type: 'application/json',
    })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `logs-${new Date().toISOString().slice(0, 19).replace(/:/g, '-')}.json`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  }, [logs, filters, isLiveMode])

  // Format timestamp for display
  const formatTime = (timestamp: string) => {
    const date = new Date(timestamp)
    const time = date.toLocaleTimeString('en-US', {
      hour12: false,
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    })
    const ms = date.getMilliseconds().toString().padStart(3, '0')
    return `${time}.${ms}`
  }

  // Format log message with category-specific enrichment
  const formatLogMessage = (log: LogEntry) => {
    const ip = log.ip_address?.replace(/\/32$/, '')

    // HTTP logs: show method, path, status
    if (log.category === 'http' && log.fields) {
      const method = log.fields.method as string | undefined
      const path = log.fields.path as string | undefined
      const statusCode = log.fields.status_code as number | undefined

      if (method && path) {
        const parts = [`${method} ${path}`]
        if (statusCode) parts.push(`→ ${statusCode}`)
        if (ip) parts.push(`[${ip}]`)
        return parts.join(' ')
      }
    }

    // Security logs: show event type and email
    if (log.category === 'security' && log.fields) {
      const eventType = (log.fields.security_event || log.fields.event_type) as
        | string
        | undefined
      const email = log.fields.email as string | undefined
      const details = log.fields.details as Record<string, unknown> | undefined
      const reason = details?.reason as string | undefined

      if (eventType) {
        const parts = [eventType]
        if (email) parts.push(`- ${email}`)
        if (reason) parts.push(`(${reason})`)
        if (ip) parts.push(`[${ip}]`)
        return parts.join(' ')
      }
    }

    // AI logs: show tool and operation details
    if (log.category === 'ai' && log.fields) {
      const tool = log.fields.tool as string | undefined
      const success = log.fields.success as boolean | undefined
      const rowsReturned = log.fields.rows_returned as number | undefined
      const durationMs = log.fields.duration_ms as number | undefined

      if (tool) {
        const parts = [tool]
        if (success !== undefined) parts.push(success ? '✓' : '✗')
        if (rowsReturned !== undefined) parts.push(`${rowsReturned} rows`)
        if (durationMs !== undefined) parts.push(`${durationMs}ms`)
        return parts.join(' ')
      }
    }

    // Execution logs: show execution type and details
    if (log.category === 'execution' && log.fields) {
      const execType = log.fields.execution_type as string | undefined
      const name = log.fields.name as string | undefined

      if (execType || name) {
        const parts: string[] = []
        if (execType) parts.push(`[${execType}]`)
        if (name) parts.push(name)
        parts.push(log.message)
        return parts.join(' ')
      }
    }

    return log.message
  }

  return (
    <div className='flex h-full flex-col gap-4'>
      {/* Stats Bar */}
      <Card className='!gap-0 !py-0'>
        <CardContent className='px-4 py-2'>
          <div className='flex items-center gap-6 text-sm'>
            <div className='flex items-center gap-1.5'>
              <span className='text-muted-foreground text-xs'>Total:</span>
              <span className='font-semibold tabular-nums'>
                {stats?.total_entries?.toLocaleString() || 0}
              </span>
            </div>
            {stats?.entries_by_level && (
              <>
                <div className='flex items-center gap-1.5'>
                  <span className='h-2 w-2 rounded-full bg-red-500' />
                  <span className='text-muted-foreground text-xs'>Errors:</span>
                  <span className='font-semibold text-red-500 tabular-nums'>
                    {(
                      (stats.entries_by_level['error'] || 0) +
                      (stats.entries_by_level['fatal'] || 0) +
                      (stats.entries_by_level['panic'] || 0)
                    ).toLocaleString()}
                  </span>
                </div>
                <div className='flex items-center gap-1.5'>
                  <span className='h-2 w-2 rounded-full bg-yellow-500' />
                  <span className='text-muted-foreground text-xs'>
                    Warnings:
                  </span>
                  <span className='font-semibold text-yellow-500 tabular-nums'>
                    {(stats.entries_by_level['warn'] || 0).toLocaleString()}
                  </span>
                </div>
              </>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Toolbar */}
      <div className='flex flex-wrap items-center justify-between gap-4'>
        <LogFiltersToolbar
          filters={filters}
          onFiltersChange={handleFiltersChange}
          activeTimePreset={activeTimePreset ?? undefined}
          onTimePresetChange={handleTimePresetChange}
        />

        <div className='flex items-center gap-3'>
          {/* Live/Historical Mode Display */}
          {isLiveMode ? (
            <>
              {/* Live Mode Indicator */}
              <Badge
                variant='default'
                className='gap-1 bg-green-600 hover:bg-green-600'
              >
                <Radio className='h-3 w-3 animate-pulse' />
                Live
              </Badge>

              {/* Connection Status */}
              <Badge
                variant={connected ? 'outline' : 'secondary'}
                className='gap-1'
              >
                {connected ? (
                  <>
                    <Wifi className='h-3 w-3' /> Connected
                  </>
                ) : (
                  <>
                    <WifiOff className='h-3 w-3' /> Connecting...
                  </>
                )}
              </Badge>
            </>
          ) : (
            <>
              {/* Historical Mode - Show time range being viewed */}
              <Badge variant='secondary' className='gap-1'>
                Viewing: {activeTimePresetLabel || 'Historical'}
              </Badge>

              {/* Return to Live Button */}
              <Button
                variant='default'
                size='sm'
                onClick={handleReturnToLive}
                className='gap-1 bg-green-600 hover:bg-green-700'
              >
                <Play className='h-3 w-3' />
                Return to Live
              </Button>

              {/* Refresh */}
              <Button
                variant='outline'
                size='sm'
                onClick={() => refetch()}
                disabled={isFetching}
              >
                <RefreshCw
                  className={`mr-1 h-3 w-3 ${isFetching ? 'animate-spin' : ''}`}
                />
                Refresh
              </Button>
            </>
          )}

          {/* Export */}
          <Button
            variant='outline'
            size='sm'
            onClick={exportLogs}
            disabled={logs.length === 0}
          >
            <Download className='mr-1 h-3 w-3' />
            Export JSON
          </Button>
        </div>
      </div>

      {/* Stream Error */}
      {streamError && isLiveMode && (
        <div className='bg-destructive/10 text-destructive rounded p-2 text-sm'>
          {streamError.message}
        </div>
      )}

      {/* Log Entries */}
      <ScrollArea ref={scrollRef} className='min-h-0 flex-1 rounded-md border'>
        <div className='p-1'>
          {isLoading && !isLiveMode ? (
            <div className='space-y-1'>
              {Array.from({ length: 10 }).map((_, i) => (
                <Skeleton key={i} className='h-8 w-full' />
              ))}
            </div>
          ) : logs.length === 0 ? (
            <div className='text-muted-foreground py-12 text-center'>
              {isLiveMode
                ? 'Waiting for logs...'
                : 'No logs found matching your filters'}
            </div>
          ) : (
            <div className='space-y-0.5'>
              {logs.map((log) => {
                const categoryConfig =
                  LOG_CATEGORY_CONFIG[log.category as LogCategory]
                const CategoryIcon = categoryConfig?.icon

                return (
                  <div
                    key={log.id}
                    onClick={() => openLogDetail(log)}
                    className='hover:bg-muted group flex cursor-pointer items-center gap-2 rounded px-2 py-1.5'
                  >
                    <span className='text-muted-foreground w-[85px] shrink-0 font-mono text-[10px] tabular-nums'>
                      {formatTime(log.timestamp)}
                    </span>
                    <LogLevelBadge level={log.level} />
                    <Badge
                      variant='outline'
                      className='h-4 shrink-0 gap-0.5 px-1 text-[10px]'
                    >
                      {CategoryIcon && <CategoryIcon className='h-2.5 w-2.5' />}
                      {log.category}
                    </Badge>
                    {log.component && (
                      <span className='text-muted-foreground shrink-0 font-mono text-[10px]'>
                        [{log.component}]
                      </span>
                    )}
                    <span className='group-hover:text-foreground flex-1 truncate font-mono text-sm'>
                      {formatLogMessage(log)}
                    </span>
                  </div>
                )
              })}
            </div>
          )}
        </div>
      </ScrollArea>

      {/* Pagination (historical mode) */}
      {!isLiveMode && data && (
        <div className='flex items-center justify-between text-sm'>
          <span className='text-muted-foreground'>
            Showing {logs.length} of {totalCount.toLocaleString()} logs
          </span>
          <div className='flex items-center gap-2'>
            <Button
              variant='outline'
              size='sm'
              onClick={() => setPage((p) => Math.max(0, p - 1))}
              disabled={page === 0}
            >
              Previous
            </Button>
            <span className='text-muted-foreground px-2'>
              Page {page + 1} of {Math.ceil(totalCount / pageSize) || 1}
            </span>
            <Button
              variant='outline'
              size='sm'
              onClick={() => setPage((p) => p + 1)}
              disabled={!data?.has_more}
            >
              Next
            </Button>
          </div>
        </div>
      )}

      {/* Log Detail Sheet */}
      <LogDetailSheet
        log={selectedLog}
        open={showDetail}
        onOpenChange={setShowDetail}
      />
    </div>
  )
}
