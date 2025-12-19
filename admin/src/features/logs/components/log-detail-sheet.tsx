import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Copy, ExternalLink } from 'lucide-react'
import { toast } from 'sonner'
import { LogLevelBadge } from './log-level-badge'
import { LOG_CATEGORY_CONFIG } from '../constants'
import type { LogEntry, LogCategory } from '../types'

interface LogDetailSheetProps {
  log: LogEntry | null
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function LogDetailSheet({ log, open, onOpenChange }: LogDetailSheetProps) {
  if (!log) return null

  const copyToClipboard = (text: string, label: string) => {
    navigator.clipboard.writeText(text)
    toast.success(`${label} copied to clipboard`)
  }

  const copyAllDetails = () => {
    const details = JSON.stringify(log, null, 2)
    copyToClipboard(details, 'Log details')
  }

  const categoryConfig = LOG_CATEGORY_CONFIG[log.category as LogCategory]
  const CategoryIcon = categoryConfig?.icon

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="w-[500px] sm:max-w-[500px] flex flex-col">
        <SheetHeader>
          <SheetTitle className="flex items-center gap-2">
            Log Details
            <Button variant="ghost" size="sm" onClick={copyAllDetails}>
              <Copy className="h-3 w-3" />
            </Button>
          </SheetTitle>
        </SheetHeader>

        <ScrollArea className="flex-1 min-h-0 mt-4">
          <div className="space-y-4 px-4">
            {/* Core Info */}
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="text-xs text-muted-foreground">Level</label>
                <div className="mt-1">
                  <LogLevelBadge level={log.level} />
                </div>
              </div>
              <div>
                <label className="text-xs text-muted-foreground">Category</label>
                <div className="mt-1 flex items-center gap-1.5">
                  {CategoryIcon && (
                    <CategoryIcon className="h-3.5 w-3.5 text-muted-foreground" />
                  )}
                  <span className="text-sm font-medium capitalize">
                    {log.category}
                  </span>
                  {log.custom_category && (
                    <Badge variant="outline" className="ml-1 text-xs">
                      {log.custom_category}
                    </Badge>
                  )}
                </div>
              </div>
              <div>
                <label className="text-xs text-muted-foreground">Timestamp</label>
                <p className="text-sm font-mono">
                  {new Date(log.timestamp).toLocaleString()}
                </p>
              </div>
              <div>
                <label className="text-xs text-muted-foreground">Component</label>
                <p className="text-sm font-mono">{log.component || '-'}</p>
              </div>
            </div>

            {/* Message */}
            <div>
              <label className="text-xs text-muted-foreground">Message</label>
              <div className="bg-muted rounded-md p-3 mt-1">
                <pre className="text-sm whitespace-pre-wrap break-words font-mono">
                  {log.message}
                </pre>
              </div>
            </div>

            {/* Correlation IDs */}
            {(log.request_id || log.trace_id) && (
              <div className="space-y-3">
                <label className="text-xs text-muted-foreground">
                  Correlation IDs
                </label>
                <div className="grid grid-cols-1 gap-2">
                  {log.request_id && (
                    <div className="flex items-center justify-between bg-muted/50 rounded p-2">
                      <div>
                        <span className="text-xs text-muted-foreground">
                          Request ID
                        </span>
                        <p className="text-xs font-mono truncate max-w-[300px]">
                          {log.request_id}
                        </p>
                      </div>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() =>
                          copyToClipboard(log.request_id!, 'Request ID')
                        }
                      >
                        <Copy className="h-3 w-3" />
                      </Button>
                    </div>
                  )}
                  {log.trace_id && (
                    <div className="flex items-center justify-between bg-muted/50 rounded p-2">
                      <div>
                        <span className="text-xs text-muted-foreground">
                          Trace ID
                        </span>
                        <p className="text-xs font-mono truncate max-w-[300px]">
                          {log.trace_id}
                        </p>
                      </div>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => copyToClipboard(log.trace_id!, 'Trace ID')}
                      >
                        <Copy className="h-3 w-3" />
                      </Button>
                    </div>
                  )}
                </div>
              </div>
            )}

            {/* User Info */}
            {(log.user_id || log.ip_address) && (
              <div className="grid grid-cols-2 gap-4">
                {log.user_id && (
                  <div>
                    <label className="text-xs text-muted-foreground">
                      User ID
                    </label>
                    <p className="text-sm font-mono truncate">{log.user_id}</p>
                  </div>
                )}
                {log.ip_address && (
                  <div>
                    <label className="text-xs text-muted-foreground">
                      IP Address
                    </label>
                    <p className="text-sm font-mono">{log.ip_address}</p>
                  </div>
                )}
              </div>
            )}

            {/* Execution Info */}
            {log.execution_id && (
              <div className="p-3 border rounded-md bg-muted/50">
                <label className="text-xs text-muted-foreground">Execution</label>
                <div className="flex items-center gap-2 mt-1">
                  <Badge variant="outline">{log.execution_type}</Badge>
                  <span className="text-sm font-mono truncate flex-1">
                    {log.execution_id}
                  </span>
                  {log.line_number !== undefined && (
                    <Badge variant="secondary">Line {log.line_number}</Badge>
                  )}
                  <Button variant="ghost" size="sm" asChild>
                    <a href={`/${log.execution_type}s`}>
                      <ExternalLink className="h-3 w-3" />
                    </a>
                  </Button>
                </div>
              </div>
            )}

            {/* Additional Fields */}
            {log.fields && Object.keys(log.fields).length > 0 && (
              <div>
                <div className="flex items-center justify-between mb-1">
                  <label className="text-xs text-muted-foreground">
                    Additional Fields
                  </label>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() =>
                      copyToClipboard(
                        JSON.stringify(log.fields, null, 2),
                        'Fields'
                      )
                    }
                  >
                    <Copy className="h-3 w-3" />
                  </Button>
                </div>
                <div className="bg-muted rounded-md p-3">
                  <pre className="text-xs overflow-auto font-mono">
                    {JSON.stringify(log.fields, null, 2)}
                  </pre>
                </div>
              </div>
            )}

            {/* Log ID */}
            <div className="pt-4 border-t">
              <div className="flex items-center justify-between">
                <div>
                  <label className="text-xs text-muted-foreground">Log ID</label>
                  <p className="text-xs font-mono text-muted-foreground">
                    {log.id}
                  </p>
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => copyToClipboard(log.id, 'Log ID')}
                >
                  <Copy className="h-3 w-3" />
                </Button>
              </div>
            </div>
          </div>
        </ScrollArea>
      </SheetContent>
    </Sheet>
  )
}
