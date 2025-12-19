import {
  AlertCircle,
  AlertTriangle,
  Bug,
  Info,
  MessageCircle,
  Skull,
  Zap,
  Server,
  Globe,
  Shield,
  Play,
  Bot,
  Tag,
} from 'lucide-react'
import type { LogCategory, LogLevel } from './types'

export const LOG_LEVEL_CONFIG: Record<
  LogLevel,
  {
    color: string
    bgColor: string
    textColor: string
    icon: typeof Info
    label: string
  }
> = {
  trace: {
    color: 'bg-gray-500',
    bgColor: 'bg-gray-500/10',
    textColor: 'text-gray-500',
    icon: MessageCircle,
    label: 'Trace',
  },
  debug: {
    color: 'bg-slate-500',
    bgColor: 'bg-slate-500/10',
    textColor: 'text-slate-500',
    icon: Bug,
    label: 'Debug',
  },
  info: {
    color: 'bg-green-500',
    bgColor: 'bg-green-500/10',
    textColor: 'text-green-500',
    icon: Info,
    label: 'Info',
  },
  warn: {
    color: 'bg-yellow-500',
    bgColor: 'bg-yellow-500/10',
    textColor: 'text-yellow-500',
    icon: AlertTriangle,
    label: 'Warning',
  },
  error: {
    color: 'bg-red-500',
    bgColor: 'bg-red-500/10',
    textColor: 'text-red-500',
    icon: AlertCircle,
    label: 'Error',
  },
  fatal: {
    color: 'bg-red-800',
    bgColor: 'bg-red-800/10',
    textColor: 'text-red-800',
    icon: Skull,
    label: 'Fatal',
  },
  panic: {
    color: 'bg-red-900',
    bgColor: 'bg-red-900/10',
    textColor: 'text-red-900',
    icon: Zap,
    label: 'Panic',
  },
}

export const LOG_CATEGORY_CONFIG: Record<
  LogCategory,
  {
    label: string
    icon: typeof Server
    description: string
  }
> = {
  system: {
    label: 'System',
    icon: Server,
    description: 'Application and system logs',
  },
  http: {
    label: 'HTTP',
    icon: Globe,
    description: 'HTTP request/response logs',
  },
  security: {
    label: 'Security',
    icon: Shield,
    description: 'Authentication and audit events',
  },
  execution: {
    label: 'Execution',
    icon: Play,
    description: 'Function, job, and RPC execution logs',
  },
  ai: {
    label: 'AI',
    icon: Bot,
    description: 'AI query audit logs',
  },
  custom: {
    label: 'Custom',
    icon: Tag,
    description: 'User-defined custom logs',
  },
}

export const LOG_LEVELS: LogLevel[] = [
  'trace',
  'debug',
  'info',
  'warn',
  'error',
  'fatal',
  'panic',
]

export const LOG_CATEGORIES: LogCategory[] = [
  'system',
  'http',
  'security',
  'execution',
  'ai',
  'custom',
]

export const DEFAULT_PAGE_SIZE = 50
export const STREAM_MAX_ENTRIES = 500
export const STREAM_RECONNECT_DELAY = 1000

// Time range presets for filtering
export const TIME_RANGE_PRESETS = [
  { label: 'Last 15 minutes', minutes: 15 },
  { label: 'Last 1 hour', minutes: 60 },
  { label: 'Last 6 hours', minutes: 360 },
  { label: 'Last 24 hours', minutes: 1440 },
  { label: 'Last 7 days', minutes: 10080 },
] as const
