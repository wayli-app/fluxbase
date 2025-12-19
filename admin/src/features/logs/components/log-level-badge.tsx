import { Badge } from '@/components/ui/badge'
import { LOG_LEVEL_CONFIG } from '../constants'
import type { LogLevel } from '../types'
import { cn } from '@/lib/utils'

interface LogLevelBadgeProps {
  level: LogLevel
  className?: string
  showIcon?: boolean
}

export function LogLevelBadge({
  level,
  className,
  showIcon = false,
}: LogLevelBadgeProps) {
  const config = LOG_LEVEL_CONFIG[level] || LOG_LEVEL_CONFIG.info
  const Icon = config.icon

  return (
    <Badge
      className={cn(
        'text-[10px] px-1.5 py-0 h-4 uppercase font-medium',
        config.color,
        'text-white border-0',
        className
      )}
    >
      {showIcon && <Icon className="h-2.5 w-2.5 mr-1" />}
      {level}
    </Badge>
  )
}

interface LogLevelIndicatorProps {
  level: LogLevel
  className?: string
}

/**
 * Simple colored dot indicator for log level
 */
export function LogLevelIndicator({ level, className }: LogLevelIndicatorProps) {
  const config = LOG_LEVEL_CONFIG[level] || LOG_LEVEL_CONFIG.info

  return (
    <span
      className={cn('inline-block w-2 h-2 rounded-full', config.color, className)}
      title={config.label}
    />
  )
}
