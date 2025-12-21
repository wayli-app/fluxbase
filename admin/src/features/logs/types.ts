// Log types matching backend storage/log_types.go

export type LogCategory =
  | 'system'
  | 'http'
  | 'security'
  | 'execution'
  | 'ai'
  | 'custom'

export type LogLevel =
  | 'trace'
  | 'debug'
  | 'info'
  | 'warn'
  | 'error'
  | 'fatal'
  | 'panic'

export interface LogEntry {
  id: string
  timestamp: string
  category: LogCategory
  level: LogLevel
  message: string
  custom_category?: string
  request_id?: string
  trace_id?: string
  component?: string
  user_id?: string
  ip_address?: string
  fields?: Record<string, unknown>
  execution_id?: string
  execution_type?: 'function' | 'job' | 'rpc'
  line_number?: number
}

export interface LogQueryOptions {
  category?: LogCategory
  custom_category?: string
  levels?: LogLevel[]
  component?: string
  request_id?: string
  trace_id?: string
  user_id?: string
  execution_id?: string
  search?: string
  start_time?: string
  end_time?: string
  limit?: number
  offset?: number
  sort_asc?: boolean
}

export interface LogQueryResult {
  entries: LogEntry[]
  total_count: number
  has_more: boolean
}

export interface LogStats {
  total_entries: number
  entries_by_category: Record<string, number>
  entries_by_level: Record<string, number>
  oldest_entry?: string
  newest_entry?: string
}

export interface LogFilters {
  category: LogCategory | 'all'
  levels: LogLevel[]
  component: string
  search: string
  timeRange: {
    start: Date | null
    end: Date | null
  }
  hideStaticAssets: boolean
}

// Default filter state
export const defaultLogFilters: LogFilters = {
  category: 'all',
  levels: [],
  component: '',
  search: '',
  timeRange: {
    start: null,
    end: null,
  },
  hideStaticAssets: false,
}
