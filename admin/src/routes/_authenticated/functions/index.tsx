import { useState, useEffect, useCallback, useMemo, useRef } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import {
  Play,
  Copy,
  History,
  RefreshCw,
  Plus,
  Edit,
  Trash2,
  Clock,
  HardDrive,
  Zap,
  Activity,
  Search,
  Filter,
  CheckCircle,
  XCircle,
  Loader2,
  ChevronLeft,
  ChevronRight,
  ChevronsLeft,
  ChevronsRight,
} from 'lucide-react'
import { getPageNumbers } from '@/lib/utils'
import { toast } from 'sonner'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardHeader,
  CardContent,
} from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { ImpersonationBanner } from '@/components/impersonation-banner'
import { ImpersonationPopover } from '@/features/impersonation/components/impersonation-popover'
import { useImpersonationStore } from '@/stores/impersonation-store'
import {
  functionsApi,
  type EdgeFunction,
  type EdgeFunctionExecution,
} from '@/lib/api'
import { useExecutionLogs } from '@/hooks/use-execution-logs'

export const Route = createFileRoute('/_authenticated/functions/')({
  component: FunctionsPage,
})

function FunctionsPage() {
  return (
    <div className='flex flex-1 flex-col gap-6 p-6'>
      <ImpersonationBanner />

      <div className='flex items-center justify-between'>
        <div>
          <h1 className='text-3xl font-bold'>Edge Functions</h1>
          <p className='text-muted-foreground'>
            Deploy and run TypeScript/JavaScript functions with Deno runtime
          </p>
        </div>
        <ImpersonationPopover
          contextLabel="Invoking as"
          defaultReason="Testing function invocation"
        />
      </div>

      <EdgeFunctionsTab />
    </div>
  )
}

// Headers Editor Component
interface HeadersEditorProps {
  headers: Array<{ key: string; value: string }>
  onChange: (headers: Array<{ key: string; value: string }>) => void
}

function HeadersEditor({ headers, onChange }: HeadersEditorProps) {
  const addHeader = () => {
    onChange([...headers, { key: '', value: '' }])
  }

  const updateHeader = (index: number, field: 'key' | 'value', value: string) => {
    const updated = [...headers]
    updated[index][field] = value
    onChange(updated)
  }

  const removeHeader = (index: number) => {
    onChange(headers.filter((_, i) => i !== index))
  }

  return (
    <div className='space-y-2'>
      {headers.map((header, index) => (
        <div key={index} className='flex gap-2'>
          <Input
            placeholder='Header name'
            value={header.key}
            onChange={(e) => updateHeader(index, 'key', e.target.value)}
            className='flex-1'
          />
          <Input
            placeholder='Header value'
            value={header.value}
            onChange={(e) => updateHeader(index, 'value', e.target.value)}
            className='flex-1'
          />
          <Button
            variant='ghost'
            size='sm'
            onClick={() => removeHeader(index)}
          >
            <Trash2 className='h-4 w-4' />
          </Button>
        </div>
      ))}
      <Button variant='outline' size='sm' onClick={addHeader}>
        <Plus className='mr-2 h-4 w-4' />
        Add Header
      </Button>
    </div>
  )
}

// Edge Functions Component
function EdgeFunctionsTab() {
  // Tab state
  const [activeTab, setActiveTab] = useState<'executions' | 'functions'>('executions')

  // Existing state
  const [edgeFunctions, setEdgeFunctions] = useState<EdgeFunction[]>([])
  const [loading, setLoading] = useState(true)
  const [showCreateDialog, setShowCreateDialog] = useState(false)
  const [showEditDialog, setShowEditDialog] = useState(false)
  const [showInvokeDialog, setShowInvokeDialog] = useState(false)
  const [showLogsDialog, setShowLogsDialog] = useState(false)
  const [showResultDialog, setShowResultDialog] = useState(false)
  const [selectedFunction, setSelectedFunction] = useState<EdgeFunction | null>(
    null
  )
  const [executions, setExecutions] = useState<EdgeFunctionExecution[]>([])
  const [invoking, setInvoking] = useState(false)

  // All executions state (for admin executions tab)
  const [allExecutions, setAllExecutions] = useState<EdgeFunctionExecution[]>([])
  const [executionsLoading, setExecutionsLoading] = useState(false)
  const [isInitialLoad, setIsInitialLoad] = useState(true)
  const [totalExecutions, setTotalExecutions] = useState(0)
  // Pagination state
  const [executionsPage, setExecutionsPage] = useState(0)  // 0-indexed
  const [executionsPageSize, setExecutionsPageSize] = useState(25)

  // Filters state
  const [searchQuery, setSearchQuery] = useState('')
  const [statusFilter, setStatusFilter] = useState<string>('all')
  const [invokeResult, setInvokeResult] = useState<{
    success: boolean
    data: string
    error?: string
  } | null>(null)
  const [wordWrap, setWordWrap] = useState(false)
  const [logsWordWrap, setLogsWordWrap] = useState(false)
  const [reloading, setReloading] = useState(false)
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null)
  const [fetchingFunction, setFetchingFunction] = useState(false)
  const [namespaces, setNamespaces] = useState<string[]>(['default'])
  const [selectedNamespace, setSelectedNamespace] = useState<string>('default')

  // Execution detail dialog state
  const [showExecutionDetailDialog, setShowExecutionDetailDialog] = useState(false)
  const [selectedExecution, setSelectedExecution] = useState<EdgeFunctionExecution | null>(null)
  const [logLevelFilter, setLogLevelFilter] = useState<string>('all')

  // Use the real-time execution logs hook
  const {
    logs: executionLogs,
    loading: executionLogsLoading,
  } = useExecutionLogs({
    executionId: selectedExecution?.id || null,
    executionType: 'function',
    enabled: showExecutionDetailDialog,
  })

  // Ref to track initial fetch (prevents debounced search from re-fetching on mount)
  const hasInitialFetch = useRef(false)
  // Ref to hold latest fetchAllExecutions to avoid it being a dependency in effects
  const fetchAllExecutionsRef = useRef<(reset?: boolean) => Promise<void>>(() => Promise.resolve())

  // Form state
  const [formData, setFormData] = useState({
    name: '',
    description: '',
    code: `interface Request {
  method: string;
  url: string;
  headers: Record<string, string>;
  body: string;
}

async function handler(req: Request) {
  // Your code here
  const data = JSON.parse(req.body || "{}");

  return {
    status: 200,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ message: "Hello from edge function!" })
  };
}`,
    timeout_seconds: 30,
    memory_limit_mb: 128,
    allow_net: true,
    allow_env: true,
    allow_read: false,
    allow_write: false,
    cron_schedule: '',
  })

  const [invokeBody, setInvokeBody] = useState('{}')
  const [invokeMethod, setInvokeMethod] = useState<'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH'>('POST')
  const [invokeHeaders, setInvokeHeaders] = useState<Array<{ key: string; value: string }>>([
    { key: '', value: '' }
  ])

  const reloadFunctionsFromDisk = async (showToast = false) => {
    try {
      const result = await functionsApi.reload()
      if (showToast) {
        const created = result.created?.length ?? 0
        const updated = result.updated?.length ?? 0
        const deleted = result.deleted?.length ?? 0
        const errors = result.errors?.length ?? 0

        if (created > 0 || updated > 0 || deleted > 0) {
          const messages = []
          if (created > 0) messages.push(`${created} created`)
          if (updated > 0) messages.push(`${updated} updated`)
          if (deleted > 0) messages.push(`${deleted} deleted`)

          toast.success(`Functions reloaded: ${messages.join(', ')}`)
        } else if (errors > 0) {
          toast.error(`Failed to reload functions: ${errors} errors`)
        } else {
          toast.info('No changes detected')
        }
      }
      return result
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error('Error reloading functions:', error)
      if (showToast) {
        toast.error('Failed to reload functions from filesystem')
      }
      throw error
    }
  }

  const handleReloadClick = async () => {
    setReloading(true)
    try {
      await reloadFunctionsFromDisk(true)
      await fetchEdgeFunctions(false) // Refresh the list without reloading again
    } finally {
      setReloading(false)
    }
  }

  const fetchEdgeFunctions = useCallback(async (shouldReload = true, namespace?: string) => {
    setLoading(true)
    try {
      // First, reload functions from disk (only on initial load or manual refresh)
      if (shouldReload) {
        await reloadFunctionsFromDisk()
      }

      const ns = namespace ?? selectedNamespace
      const data = await functionsApi.list(ns)
      setEdgeFunctions(data || [])
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error('Error fetching edge functions:', error)
      toast.error('Failed to fetch edge functions')
    } finally {
      setLoading(false)
    }
  }, [selectedNamespace])

  // Fetch all executions for the executions tab
  const fetchAllExecutions = useCallback(async () => {
    // Only show full loading spinner on initial load, not on refetches
    if (isInitialLoad) {
      setExecutionsLoading(true)
    }

    try {
      const offset = executionsPage * executionsPageSize
      const result = await functionsApi.listAllExecutions({
        namespace: selectedNamespace !== 'all' ? selectedNamespace : undefined,
        function_name: searchQuery || undefined,
        status: statusFilter !== 'all' ? statusFilter : undefined,
        limit: executionsPageSize,
        offset,
      })

      setAllExecutions(result.executions || [])
      setTotalExecutions(result.count || 0)

      // Mark initial load as complete
      if (isInitialLoad) {
        setIsInitialLoad(false)
      }
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error('Error fetching executions:', error)
      toast.error('Failed to fetch executions')
    } finally {
      setExecutionsLoading(false)
    }
  }, [selectedNamespace, searchQuery, statusFilter, executionsPage, executionsPageSize, isInitialLoad])

  // Keep the ref updated with the latest fetchAllExecutions
  useEffect(() => {
    fetchAllExecutionsRef.current = fetchAllExecutions
  }, [fetchAllExecutions])

  // Filter executions from past 24 hours (for stats display only)
  const executions24h = useMemo(() => {
    const cutoff = Date.now() - 24 * 60 * 60 * 1000
    return allExecutions.filter((exec) => {
      const execTime = new Date(exec.executed_at).getTime()
      return execTime >= cutoff
    })
  }, [allExecutions])

  // Calculate stats from past 24 hours
  const executionStats = useMemo(() => {
    const success = executions24h.filter((e) => e.status === 'success').length
    const failed = executions24h.filter((e) => e.status === 'error' || e.status === 'failed').length
    const total = executions24h.length
    const avgDuration = executions24h.length > 0
      ? Math.round(executions24h.reduce((sum, e) => sum + (e.duration_ms || 0), 0) / executions24h.length)
      : 0
    return { success, failed, total, avgDuration }
  }, [executions24h])

  // Filter logs by level
  const filteredLogs = useMemo(() => {
    if (logLevelFilter === 'all') return executionLogs
    return executionLogs.filter((log) => log.level === logLevelFilter)
  }, [executionLogs, logLevelFilter])

  // Open execution detail dialog
  const openExecutionDetail = (exec: EdgeFunctionExecution) => {
    setSelectedExecution(exec)
    setShowExecutionDetailDialog(true)
    setLogLevelFilter('all')
    // The useExecutionLogs hook handles fetching and real-time updates automatically
  }

  // Copy to clipboard helper
  const copyToClipboard = (text: string, label: string) => {
    navigator.clipboard.writeText(text)
    toast.success(`${label} copied to clipboard`)
  }

  // Fetch namespaces on mount
  useEffect(() => {
    const fetchNamespaces = async () => {
      try {
        const data = await functionsApi.listNamespaces()
        // Filter out empty strings to prevent Select component errors
        const validNamespaces = data.filter((ns: string) => ns !== '')
        setNamespaces(validNamespaces.length > 0 ? validNamespaces : ['default'])
        // If current namespace not in list, reset to first available
        // Use functional update to avoid dependency on selectedNamespace
        setSelectedNamespace((current) =>
          validNamespaces.includes(current) ? current : (validNamespaces[0] || 'default')
        )
      } catch {
        setNamespaces(['default'])
      }
    }
    fetchNamespaces()
  }, [])  // Only run on mount - no dependencies needed

  useEffect(() => {
    fetchEdgeFunctions()
  }, [fetchEdgeFunctions, selectedNamespace])

  // Fetch executions when tab changes or any fetch-related state changes
  useEffect(() => {
    if (activeTab === 'executions') {
      hasInitialFetch.current = true
      fetchAllExecutionsRef.current()
    }
  // Using ref to avoid fetchAllExecutions in dependencies which would cause double-fetches
  // All filter/pagination changes will trigger this effect via their state changes
  }, [activeTab, selectedNamespace, statusFilter, executionsPage, executionsPageSize])

  // Debounced search - resets page to 0 and fetches
  useEffect(() => {
    if (activeTab !== 'executions') return
    // Skip the first render - the main effect above handles initial fetch
    if (!hasInitialFetch.current) return
    const timer = setTimeout(() => {
      // Reset to page 0 when search changes - this will trigger the main effect
      setExecutionsPage(0)
    }, 300)
    return () => clearTimeout(timer)
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchQuery])

  const createFunction = async () => {
    try {
      await functionsApi.create({
        ...formData,
        cron_schedule: formData.cron_schedule || null,
      })
      toast.success('Edge function created successfully')
      setShowCreateDialog(false)
      resetForm()
      fetchEdgeFunctions(false) // Don't reload from disk after creating
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error('Error creating edge function:', error)
      toast.error('Failed to create edge function')
    }
  }

  const updateFunction = async () => {
    if (!selectedFunction) return

    try {
      await functionsApi.update(selectedFunction.name, {
        code: formData.code,
        description: formData.description,
        timeout_seconds: formData.timeout_seconds,
        allow_net: formData.allow_net,
        allow_env: formData.allow_env,
        allow_read: formData.allow_read,
        allow_write: formData.allow_write,
        cron_schedule: formData.cron_schedule || null,
      })
      toast.success('Edge function updated successfully')
      setShowEditDialog(false)
      fetchEdgeFunctions(false) // Don't reload from disk after updating
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error('Error updating edge function:', error)
      toast.error('Failed to update edge function')
    }
  }

  const deleteFunction = async (name: string) => {
    try {
      await functionsApi.delete(name)
      toast.success('Edge function deleted successfully')
      fetchEdgeFunctions(false) // Don't reload from disk after deleting
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error('Error deleting edge function:', error)
      toast.error('Failed to delete edge function')
    } finally {
      setDeleteConfirm(null)
    }
  }

  const toggleFunction = async (fn: EdgeFunction) => {
    const newEnabledState = !fn.enabled

    try {
      await functionsApi.update(fn.name, {
        code: fn.code,
        description: fn.description,
        timeout_seconds: fn.timeout_seconds,
        allow_net: fn.allow_net,
        allow_env: fn.allow_env,
        allow_read: fn.allow_read,
        allow_write: fn.allow_write,
        cron_schedule: fn.cron_schedule || null,
        enabled: newEnabledState,
      })
      toast.success(`Function ${newEnabledState ? 'enabled' : 'disabled'}`)
      fetchEdgeFunctions(false) // Don't reload from disk after toggling
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error('Error toggling function:', error)
      toast.error('Failed to toggle function')
    }
  }

  const invokeFunction = async () => {
    if (!selectedFunction) return

    setInvoking(true)
    try {
      // Convert headers array to object, filtering empty ones
      const headersObj = invokeHeaders
        .filter((h) => h.key.trim() !== '')
        .reduce((acc, h) => ({ ...acc, [h.key]: h.value }), {})

      // Build config with impersonation token if active
      const { isImpersonating, impersonationToken } = useImpersonationStore.getState()
      const config: { headers?: Record<string, string> } = {}
      if (isImpersonating && impersonationToken) {
        config.headers = { 'X-Impersonation-Token': impersonationToken }
      }

      const result = await functionsApi.invoke(
        selectedFunction.name,
        {
          method: invokeMethod,
          headers: headersObj,
          body: invokeBody,
        },
        config
      )
      toast.success('Function invoked successfully')
      setInvokeResult({ success: true, data: result })
      setShowInvokeDialog(false)
      setShowResultDialog(true)
    } catch (error: unknown) {
      // eslint-disable-next-line no-console
      console.error('Error invoking function:', error)
      toast.error('Failed to invoke function')
      const errorMessage =
        error instanceof Error ? error.message : 'Unknown error'
      setInvokeResult({ success: false, data: '', error: errorMessage })
      setShowInvokeDialog(false)
      setShowResultDialog(true)
    } finally {
      setInvoking(false)
    }
  }

  const fetchExecutions = async (functionName: string) => {
    try {
      const data = await functionsApi.getExecutions(functionName, 20)
      setExecutions(data || [])
      setShowLogsDialog(true)
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error('Error fetching executions:', error)
      toast.error('Failed to fetch execution logs')
    }
  }

  const openEditDialog = async (fn: EdgeFunction) => {
    setSelectedFunction(fn)
    setFetchingFunction(true)
    setShowEditDialog(true)
    try {
      // Fetch full function details including code
      const fullFunction = await functionsApi.get(fn.name)
      setFormData({
        name: fullFunction.name,
        description: fullFunction.description || '',
        code: fullFunction.code || '',
        timeout_seconds: fullFunction.timeout_seconds,
        memory_limit_mb: fullFunction.memory_limit_mb,
        allow_net: fullFunction.allow_net,
        allow_env: fullFunction.allow_env,
        allow_read: fullFunction.allow_read,
        allow_write: fullFunction.allow_write,
        cron_schedule: fullFunction.cron_schedule || '',
      })
    } catch {
      toast.error('Failed to load function details')
      setShowEditDialog(false)
    } finally {
      setFetchingFunction(false)
    }
  }

  const openInvokeDialog = (fn: EdgeFunction) => {
    setSelectedFunction(fn)
    setInvokeBody('{\n  "name": "World"\n}')
    setShowInvokeDialog(true)
  }

  const resetForm = () => {
    setFormData({
      name: '',
      description: '',
      code: `interface Request {
  method: string;
  url: string;
  headers: Record<string, string>;
  body: string;
}

async function handler(req: Request) {
  // Your code here
  const data = JSON.parse(req.body || "{}");

  return {
    status: 200,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ message: "Hello from edge function!" })
  };
}`,
      timeout_seconds: 30,
      memory_limit_mb: 128,
      allow_net: true,
      allow_env: true,
      allow_read: false,
      allow_write: false,
      cron_schedule: '',
    })
  }

  if (loading) {
    return (
      <div className='flex h-96 items-center justify-center'>
        <RefreshCw className='text-muted-foreground h-8 w-8 animate-spin' />
      </div>
    )
  }

  return (
    <>
      {/* Stats (Past 24 hours) */}
      <Card className='!gap-0 !py-0'>
        <CardContent className='px-4 py-2'>
          <div className='flex items-center gap-4'>
            <span className='text-muted-foreground text-xs'>(Past 24 hours)</span>
            <div className='flex items-center gap-1'>
              <span className='text-muted-foreground text-xs'>Success:</span>
              <span className='text-sm font-semibold'>
                {executionStats.success}
              </span>
            </div>
            <div className='flex items-center gap-1'>
              <span className='text-muted-foreground text-xs'>Failed:</span>
              <span className='text-sm font-semibold'>
                {executionStats.failed}
              </span>
            </div>
            <div className='flex items-center gap-1'>
              <span className='text-muted-foreground text-xs'>Total:</span>
              <span className='text-sm font-semibold'>
                {executionStats.total}
              </span>
            </div>
            <div className='flex items-center gap-1'>
              <span className='text-muted-foreground text-xs'>Success Rate:</span>
              {(() => {
                const total = executionStats.success + executionStats.failed
                const successRate =
                  total > 0
                    ? ((executionStats.success / total) * 100).toFixed(0)
                    : '0'
                return (
                  <span className='text-sm font-semibold'>
                    {successRate}%
                  </span>
                )
              })()}
            </div>
            <div className='flex items-center gap-1'>
              <span className='text-muted-foreground text-xs'>Avg. Duration:</span>
              <span className='text-sm font-semibold'>
                {executionStats.avgDuration}ms
              </span>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Tabs */}
      <Tabs
        value={activeTab}
        onValueChange={(v) => setActiveTab(v as 'executions' | 'functions')}
        className='flex min-h-0 flex-1 flex-col'
      >
        <div className='flex items-center justify-between mb-4'>
          <TabsList className='grid w-full max-w-md grid-cols-2'>
            <TabsTrigger value='executions'>
              <Activity className='mr-2 h-4 w-4' />
              Execution Logs
            </TabsTrigger>
            <TabsTrigger value='functions'>
              <Zap className='mr-2 h-4 w-4' />
              Functions
            </TabsTrigger>
          </TabsList>
        </div>

        {/* Executions Tab */}
        <TabsContent value='executions' className='flex-1 mt-0'>
          {/* Executions Filters */}
          <div className='flex items-center gap-3 mb-4'>
            <div className='flex items-center gap-2'>
              <Label htmlFor='exec-namespace-select' className='text-sm text-muted-foreground whitespace-nowrap'>
                Namespace:
              </Label>
              <Select
                value={selectedNamespace}
                onValueChange={(value) => {
                  setSelectedNamespace(value)
                  setExecutionsPage(0)
                }}
              >
                <SelectTrigger id='exec-namespace-select' className='w-[150px]'>
                  <SelectValue placeholder='Select namespace' />
                </SelectTrigger>
                <SelectContent>
                  {namespaces.map((ns) => (
                    <SelectItem key={ns} value={ns}>
                      {ns}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className='relative flex-1 max-w-xs'>
              <Search className='text-muted-foreground absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2' />
              <Input
                placeholder='Search by function name...'
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className='pl-9'
              />
            </div>
            <Select
              value={statusFilter}
              onValueChange={(value) => {
                setStatusFilter(value)
                setExecutionsPage(0)
              }}
            >
              <SelectTrigger className='w-[150px]'>
                <Filter className='mr-2 h-4 w-4' />
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value='all'>All Status</SelectItem>
                <SelectItem value='success'>Success</SelectItem>
                <SelectItem value='error'>Error</SelectItem>
              </SelectContent>
            </Select>
            <Button
              onClick={() => fetchAllExecutionsRef.current()}
              variant='outline'
              size='sm'
            >
              <RefreshCw className='mr-2 h-4 w-4' />
              Refresh
            </Button>
          </div>

          {/* Executions List */}
          <ScrollArea className='h-[calc(100vh-28rem)]'>
            {executionsLoading && isInitialLoad ? (
              <div className='flex h-48 items-center justify-center'>
                <Loader2 className='h-8 w-8 animate-spin text-muted-foreground' />
              </div>
            ) : allExecutions.length === 0 ? (
              <Card>
                <CardContent className='p-12 text-center'>
                  <Activity className='text-muted-foreground mx-auto mb-4 h-12 w-12' />
                  <p className='mb-2 text-lg font-medium'>No executions found</p>
                  <p className='text-muted-foreground text-sm'>
                    Execute some functions to see their logs here
                  </p>
                </CardContent>
              </Card>
            ) : (
              <div className='grid gap-1'>
                {/* Inline loading indicator for refetches */}
                {executionsLoading && !isInitialLoad && (
                  <div className='flex items-center justify-center py-2 text-muted-foreground'>
                    <Loader2 className='h-4 w-4 animate-spin mr-2' />
                    <span className='text-xs'>Refreshing...</span>
                  </div>
                )}
                {allExecutions.map((exec) => (
                  <div
                    key={exec.id}
                    className='flex items-center justify-between gap-2 px-3 py-2 rounded-md border hover:border-primary/50 transition-colors bg-card cursor-pointer'
                    onClick={() => openExecutionDetail(exec)}
                  >
                    <div className='flex items-center gap-3 min-w-0 flex-1'>
                      {exec.status === 'success' ? (
                        <CheckCircle className='h-4 w-4 text-green-500 shrink-0' />
                      ) : (
                        <XCircle className='h-4 w-4 text-red-500 shrink-0' />
                      )}
                      <span className='text-sm font-medium truncate'>
                        {exec.function_name || 'Unknown'}
                      </span>
                      <Badge variant={exec.status === 'success' ? 'secondary' : 'destructive'} className='shrink-0 text-[10px] px-1.5 py-0 h-4'>
                        {exec.status}
                      </Badge>
                      {exec.status_code && (
                        <Badge variant='outline' className='shrink-0 text-[10px] px-1.5 py-0 h-4'>
                          {exec.status_code}
                        </Badge>
                      )}
                    </div>
                    <div className='flex items-center gap-3 shrink-0'>
                      <span className='text-xs text-muted-foreground'>
                        {exec.duration_ms ? `${exec.duration_ms}ms` : '-'}
                      </span>
                      <span className='text-xs text-muted-foreground'>
                        {new Date(exec.executed_at).toLocaleString()}
                      </span>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </ScrollArea>

          {/* Pagination Controls */}
          {allExecutions.length > 0 && (
            <div className='flex items-center justify-between px-2 py-3 border-t'>
              <div className='flex items-center gap-2'>
                <span className='text-sm text-muted-foreground'>Rows per page</span>
                <Select
                  value={`${executionsPageSize}`}
                  onValueChange={(value) => {
                    setExecutionsPageSize(Number(value))
                    setExecutionsPage(0)
                  }}
                >
                  <SelectTrigger className='h-8 w-[70px]'>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent side='top'>
                    {[10, 25, 50, 100].map((size) => (
                      <SelectItem key={size} value={`${size}`}>{size}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className='flex items-center gap-2'>
                <span className='text-sm text-muted-foreground'>
                  Page {executionsPage + 1} of {Math.ceil(totalExecutions / executionsPageSize) || 1} ({totalExecutions} total)
                </span>

                {/* First page */}
                <Button
                  variant='outline'
                  size='sm'
                  className='h-8 w-8 p-0'
                  onClick={() => setExecutionsPage(0)}
                  disabled={executionsPage === 0}
                >
                  <ChevronsLeft className='h-4 w-4' />
                </Button>

                {/* Previous page */}
                <Button
                  variant='outline'
                  size='sm'
                  className='h-8 w-8 p-0'
                  onClick={() => setExecutionsPage((p) => p - 1)}
                  disabled={executionsPage === 0}
                >
                  <ChevronLeft className='h-4 w-4' />
                </Button>

                {/* Page numbers */}
                {getPageNumbers(executionsPage + 1, Math.ceil(totalExecutions / executionsPageSize) || 1).map((pageNum, idx) => (
                  pageNum === '...' ? (
                    <span key={`ellipsis-${idx}`} className='px-1 text-muted-foreground'>...</span>
                  ) : (
                    <Button
                      key={pageNum}
                      variant={executionsPage + 1 === pageNum ? 'default' : 'outline'}
                      size='sm'
                      className='h-8 min-w-8 px-2'
                      onClick={() => setExecutionsPage((pageNum as number) - 1)}
                    >
                      {pageNum}
                    </Button>
                  )
                ))}

                {/* Next page */}
                <Button
                  variant='outline'
                  size='sm'
                  className='h-8 w-8 p-0'
                  onClick={() => setExecutionsPage((p) => p + 1)}
                  disabled={executionsPage >= Math.ceil(totalExecutions / executionsPageSize) - 1}
                >
                  <ChevronRight className='h-4 w-4' />
                </Button>

                {/* Last page */}
                <Button
                  variant='outline'
                  size='sm'
                  className='h-8 w-8 p-0'
                  onClick={() => setExecutionsPage(Math.ceil(totalExecutions / executionsPageSize) - 1)}
                  disabled={executionsPage >= Math.ceil(totalExecutions / executionsPageSize) - 1}
                >
                  <ChevronsRight className='h-4 w-4' />
                </Button>
              </div>
            </div>
          )}
        </TabsContent>

        {/* Functions Tab */}
        <TabsContent value='functions' className='flex-1 mt-0'>
          {/* Functions Controls */}
          <div className='flex items-center justify-end gap-2 mb-4'>
            <div className='flex items-center gap-2'>
              <Label htmlFor='func-namespace-select' className='text-sm text-muted-foreground whitespace-nowrap'>
                Namespace:
              </Label>
              <Select value={selectedNamespace} onValueChange={setSelectedNamespace}>
                <SelectTrigger id='func-namespace-select' className='w-[180px]'>
                  <SelectValue placeholder='Select namespace' />
                </SelectTrigger>
                <SelectContent>
                  {namespaces.map((ns) => (
                    <SelectItem key={ns} value={ns}>
                      {ns}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <Button
              onClick={handleReloadClick}
              variant='outline'
              size='sm'
              disabled={reloading}
            >
              {reloading ? (
                <>
                  <RefreshCw className='mr-2 h-4 w-4 animate-spin' />
                  Reloading...
                </>
              ) : (
                <>
                  <HardDrive className='mr-2 h-4 w-4' />
                  Reload from Filesystem
                </>
              )}
            </Button>
            <Button
              onClick={() => fetchEdgeFunctions()}
              variant='outline'
              size='sm'
            >
              <RefreshCw className='mr-2 h-4 w-4' />
              Refresh
            </Button>
            <Button onClick={() => setShowCreateDialog(true)} size='sm'>
              <Plus className='mr-2 h-4 w-4' />
              New Function
            </Button>
          </div>

          {/* Functions Stats */}
          <div className='flex gap-4 text-sm mb-4'>
            <div className='flex items-center gap-1.5'>
              <span className='text-muted-foreground'>Total:</span>
              <Badge variant='secondary' className='h-5 px-2'>
                {edgeFunctions.length}
              </Badge>
            </div>
            <div className='flex items-center gap-1.5'>
              <span className='text-muted-foreground'>Active:</span>
              <Badge variant='secondary' className='h-5 px-2 bg-green-500/10 text-green-600 dark:text-green-400'>
                {edgeFunctions.filter((f) => f.enabled).length}
              </Badge>
            </div>
            <div className='flex items-center gap-1.5'>
              <span className='text-muted-foreground'>Scheduled:</span>
              <Badge variant='secondary' className='h-5 px-2'>
                {edgeFunctions.filter((f) => f.cron_schedule).length}
              </Badge>
            </div>
          </div>

          {/* Functions List */}
          <ScrollArea className='h-[calc(100vh-20rem)]'>
        <div className='grid gap-1'>
          {edgeFunctions.length === 0 ? (
            <Card>
              <CardContent className='p-12 text-center'>
                <Zap className='text-muted-foreground mx-auto mb-4 h-12 w-12' />
                <p className='mb-2 text-lg font-medium'>
                  No edge functions yet
                </p>
                <p className='text-muted-foreground mb-4 text-sm'>
                  Create your first edge function to get started
                </p>
                <Button onClick={() => setShowCreateDialog(true)}>
                  <Plus className='mr-2 h-4 w-4' />
                  Create Edge Function
                </Button>
              </CardContent>
            </Card>
          ) : (
            edgeFunctions.map((fn) => (
              <div
                key={fn.id}
                className='flex items-center justify-between gap-2 px-3 py-1.5 rounded-md border hover:border-primary/50 transition-colors bg-card'
              >
                <div className='flex items-center gap-2 min-w-0 flex-1'>
                  <span className='text-sm font-medium truncate'>{fn.name}</span>
                  <Badge variant='outline' className='shrink-0 text-[10px] px-1 py-0 h-4'>v{fn.version}</Badge>
                  {fn.cron_schedule && (
                    <Badge variant='outline' className='shrink-0 text-[10px] px-1 py-0 h-4'>
                      <Clock className='mr-0.5 h-2.5 w-2.5' />
                      cron
                    </Badge>
                  )}
                  <Switch
                    checked={fn.enabled}
                    onCheckedChange={() => toggleFunction(fn)}
                    className='scale-75'
                  />
                </div>
                <div className='flex items-center gap-0.5 shrink-0'>
                  {fn.source === 'filesystem' && fn.updated_at && (
                    <span
                      className='text-muted-foreground text-[10px] mr-1'
                      title={`Last synced: ${new Date(fn.updated_at).toLocaleString()}`}
                    >
                      synced {new Date(fn.updated_at).toLocaleDateString()}
                    </span>
                  )}
                  <span className='text-[10px] text-muted-foreground mr-1'>{fn.timeout_seconds}s</span>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        onClick={() => fetchExecutions(fn.name)}
                        variant='ghost'
                        size='sm'
                        className='h-6 w-6 p-0'
                      >
                        <History className='h-3 w-3' />
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>View logs</TooltipContent>
                  </Tooltip>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        onClick={() => openInvokeDialog(fn)}
                        size='sm'
                        variant='ghost'
                        className='h-6 w-6 p-0'
                        disabled={!fn.enabled}
                      >
                        <Play className='h-3 w-3' />
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>Invoke function</TooltipContent>
                  </Tooltip>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        onClick={() => openEditDialog(fn)}
                        size='sm'
                        variant='ghost'
                        className='h-6 w-6 p-0'
                      >
                        <Edit className='h-3 w-3' />
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>Edit function</TooltipContent>
                  </Tooltip>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        onClick={() => setDeleteConfirm(fn.name)}
                        size='sm'
                        variant='ghost'
                        className='h-6 w-6 p-0 text-destructive hover:text-destructive hover:bg-destructive/10'
                      >
                        <Trash2 className='h-3 w-3' />
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>Delete function</TooltipContent>
                  </Tooltip>
                </div>
              </div>
            ))
          )}
        </div>
          </ScrollArea>
        </TabsContent>
      </Tabs>

      {/* Delete Confirmation Dialog */}
      <AlertDialog open={deleteConfirm !== null} onOpenChange={(open) => !open && setDeleteConfirm(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Function</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete "{deleteConfirm}"? This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => deleteConfirm && deleteFunction(deleteConfirm)}
              className='bg-destructive text-destructive-foreground hover:bg-destructive/90'
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Create Function Dialog */}
      <Dialog open={showCreateDialog} onOpenChange={setShowCreateDialog}>
        <DialogContent className='max-h-[90vh] max-w-4xl overflow-y-auto'>
          <DialogHeader>
            <DialogTitle>Create Edge Function</DialogTitle>
            <DialogDescription>
              Deploy a new TypeScript/JavaScript function with Deno runtime
            </DialogDescription>
          </DialogHeader>

          <div className='space-y-4'>
            <div>
              <Label htmlFor='name'>Function Name</Label>
              <Input
                id='name'
                placeholder='my_function'
                value={formData.name}
                onChange={(e) =>
                  setFormData({ ...formData, name: e.target.value })
                }
              />
            </div>

            <div>
              <Label htmlFor='description'>Description (optional)</Label>
              <Input
                id='description'
                placeholder='What does this function do?'
                value={formData.description}
                onChange={(e) =>
                  setFormData({ ...formData, description: e.target.value })
                }
              />
            </div>

            <div>
              <Label htmlFor='code'>Code (TypeScript)</Label>
              <Textarea
                id='code'
                className='min-h-[400px] font-mono text-sm'
                value={formData.code}
                onChange={(e) =>
                  setFormData({ ...formData, code: e.target.value })
                }
              />
            </div>

            <div className='grid grid-cols-2 gap-4'>
              <div>
                <Label htmlFor='timeout'>Timeout (seconds)</Label>
                <Input
                  id='timeout'
                  type='number'
                  min={1}
                  max={300}
                  value={formData.timeout_seconds}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      timeout_seconds: parseInt(e.target.value),
                    })
                  }
                />
              </div>

              <div>
                <Label htmlFor='cron'>Cron Schedule (optional)</Label>
                <Input
                  id='cron'
                  placeholder='0 0 * * *'
                  value={formData.cron_schedule}
                  onChange={(e) =>
                    setFormData({ ...formData, cron_schedule: e.target.value })
                  }
                />
              </div>
            </div>

            <div>
              <Label>Permissions</Label>
              <div className='mt-2 grid grid-cols-2 gap-3'>
                <label className='flex cursor-pointer items-center gap-2'>
                  <input
                    type='checkbox'
                    checked={formData.allow_net}
                    onChange={(e) =>
                      setFormData({ ...formData, allow_net: e.target.checked })
                    }
                  />
                  <span>Allow Network Access</span>
                </label>
                <label className='flex cursor-pointer items-center gap-2'>
                  <input
                    type='checkbox'
                    checked={formData.allow_env}
                    onChange={(e) =>
                      setFormData({ ...formData, allow_env: e.target.checked })
                    }
                  />
                  <span>Allow Environment Variables</span>
                </label>
                <label className='flex cursor-pointer items-center gap-2'>
                  <input
                    type='checkbox'
                    checked={formData.allow_read}
                    onChange={(e) =>
                      setFormData({ ...formData, allow_read: e.target.checked })
                    }
                  />
                  <span>Allow File Read</span>
                </label>
                <label className='flex cursor-pointer items-center gap-2'>
                  <input
                    type='checkbox'
                    checked={formData.allow_write}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        allow_write: e.target.checked,
                      })
                    }
                  />
                  <span>Allow File Write</span>
                </label>
              </div>
            </div>
          </div>

          <DialogFooter>
            <Button
              variant='outline'
              onClick={() => setShowCreateDialog(false)}
            >
              Cancel
            </Button>
            <Button onClick={createFunction}>Create Function</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Function Dialog */}
      <Dialog open={showEditDialog} onOpenChange={setShowEditDialog}>
        <DialogContent className='max-h-[90vh] max-w-4xl overflow-y-auto'>
          <DialogHeader>
            <DialogTitle>Edit Edge Function</DialogTitle>
            <DialogDescription>
              Update function code and settings
            </DialogDescription>
          </DialogHeader>

          {fetchingFunction ? (
            <div className='flex items-center justify-center py-12'>
              <Loader2 className='h-8 w-8 animate-spin text-muted-foreground' />
            </div>
          ) : (
          <div className='space-y-4'>
            <div>
              <Label htmlFor='edit-description'>Description</Label>
              <Input
                id='edit-description'
                value={formData.description}
                onChange={(e) =>
                  setFormData({ ...formData, description: e.target.value })
                }
              />
            </div>

            <div>
              <Label htmlFor='edit-code'>Code</Label>
              <Textarea
                id='edit-code'
                className='min-h-[400px] font-mono text-sm'
                value={formData.code}
                onChange={(e) =>
                  setFormData({ ...formData, code: e.target.value })
                }
              />
            </div>

            <div className='grid grid-cols-2 gap-4'>
              <div>
                <Label htmlFor='edit-timeout'>Timeout (seconds)</Label>
                <Input
                  id='edit-timeout'
                  type='number'
                  min={1}
                  max={300}
                  value={formData.timeout_seconds}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      timeout_seconds: parseInt(e.target.value),
                    })
                  }
                />
              </div>

              <div>
                <Label htmlFor='edit-cron'>Cron Schedule</Label>
                <Input
                  id='edit-cron'
                  placeholder='0 0 * * *'
                  value={formData.cron_schedule}
                  onChange={(e) =>
                    setFormData({ ...formData, cron_schedule: e.target.value })
                  }
                />
              </div>
            </div>

            <div>
              <Label>Permissions</Label>
              <div className='mt-2 grid grid-cols-2 gap-3'>
                <label className='flex cursor-pointer items-center gap-2'>
                  <input
                    type='checkbox'
                    checked={formData.allow_net}
                    onChange={(e) =>
                      setFormData({ ...formData, allow_net: e.target.checked })
                    }
                  />
                  <span>Allow Network Access</span>
                </label>
                <label className='flex cursor-pointer items-center gap-2'>
                  <input
                    type='checkbox'
                    checked={formData.allow_env}
                    onChange={(e) =>
                      setFormData({ ...formData, allow_env: e.target.checked })
                    }
                  />
                  <span>Allow Environment Variables</span>
                </label>
                <label className='flex cursor-pointer items-center gap-2'>
                  <input
                    type='checkbox'
                    checked={formData.allow_read}
                    onChange={(e) =>
                      setFormData({ ...formData, allow_read: e.target.checked })
                    }
                  />
                  <span>Allow File Read</span>
                </label>
                <label className='flex cursor-pointer items-center gap-2'>
                  <input
                    type='checkbox'
                    checked={formData.allow_write}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        allow_write: e.target.checked,
                      })
                    }
                  />
                  <span>Allow File Write</span>
                </label>
              </div>
            </div>
          </div>
          )}

          <DialogFooter>
            <Button variant='outline' onClick={() => setShowEditDialog(false)}>
              Cancel
            </Button>
            <Button onClick={updateFunction} disabled={fetchingFunction}>Update Function</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Invoke Function Dialog */}
      <Dialog open={showInvokeDialog} onOpenChange={setShowInvokeDialog}>
        <DialogContent className='w-[90vw] max-w-5xl max-h-[90vh] overflow-y-auto'>
          <DialogHeader>
            <DialogTitle>Invoke Edge Function</DialogTitle>
            <DialogDescription>
              Test {selectedFunction?.name} with custom HTTP request
            </DialogDescription>
          </DialogHeader>

          <div className='space-y-4'>
            {/* Method selector */}
            <div className='flex items-center gap-4'>
              <Label htmlFor='method'>HTTP Method</Label>
              <Select
                value={invokeMethod}
                onValueChange={(value) => setInvokeMethod(value as 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH')}
              >
                <SelectTrigger className='w-[180px]' id='method'>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value='GET'>GET</SelectItem>
                  <SelectItem value='POST'>POST</SelectItem>
                  <SelectItem value='PUT'>PUT</SelectItem>
                  <SelectItem value='DELETE'>DELETE</SelectItem>
                  <SelectItem value='PATCH'>PATCH</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {/* Tabbed interface for Body/Headers */}
            <Tabs defaultValue='body' className='space-y-4'>
              <TabsList className='grid w-full grid-cols-2'>
                <TabsTrigger value='body'>Body</TabsTrigger>
                <TabsTrigger value='headers'>Headers</TabsTrigger>
              </TabsList>

              <TabsContent value='body' className='space-y-2'>
                <Label htmlFor='invoke-body'>Request Body (JSON)</Label>
                <Textarea
                  id='invoke-body'
                  className='min-h-[300px] font-mono text-sm'
                  value={invokeBody}
                  onChange={(e) => setInvokeBody(e.target.value)}
                  placeholder='{"key": "value"}'
                />
              </TabsContent>

              <TabsContent value='headers' className='space-y-2'>
                <Label>Custom Headers</Label>
                <ScrollArea className='max-h-[350px]'>
                  <HeadersEditor
                    headers={invokeHeaders}
                    onChange={setInvokeHeaders}
                  />
                </ScrollArea>
              </TabsContent>
            </Tabs>
          </div>

          <DialogFooter>
            <Button
              variant='outline'
              onClick={() => setShowInvokeDialog(false)}
            >
              Cancel
            </Button>
            <Button onClick={invokeFunction} disabled={invoking}>
              {invoking ? (
                <>
                  <RefreshCw className='mr-2 h-4 w-4 animate-spin' />
                  Invoking...
                </>
              ) : (
                <>
                  <Play className='mr-2 h-4 w-4' />
                  Invoke
                </>
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Execution Logs Dialog */}
      <Dialog open={showLogsDialog} onOpenChange={setShowLogsDialog}>
        <DialogContent className='flex max-h-[95vh] !w-[95vw] !max-w-[95vw] flex-col overflow-hidden'>
          <DialogHeader className='flex-shrink-0'>
            <DialogTitle>Execution Logs</DialogTitle>
            <DialogDescription>
              Recent executions for {selectedFunction?.name}
            </DialogDescription>
          </DialogHeader>

          <div className='flex flex-shrink-0 items-center space-x-2'>
            <Switch
              id='logs-word-wrap'
              checked={logsWordWrap}
              onCheckedChange={setLogsWordWrap}
            />
            <Label htmlFor='logs-word-wrap' className='cursor-pointer'>
              Word wrap
            </Label>
          </div>

          <div className='min-h-0 flex-1 space-y-3 overflow-auto rounded-lg border p-4'>
            {executions.length === 0 ? (
              <div className='text-muted-foreground py-12 text-center'>
                <History className='mx-auto mb-4 h-12 w-12 opacity-50' />
                <p>No executions yet</p>
              </div>
            ) : (
              executions.map((exec) => (
                <Card key={exec.id} className='overflow-hidden'>
                  <CardHeader className='pb-3'>
                    <div className='flex items-start justify-between'>
                      <div className='flex-1'>
                        <div className='mb-1 flex items-center gap-2'>
                          <Badge
                            variant={
                              exec.status === 'success'
                                ? 'default'
                                : 'destructive'
                            }
                          >
                            {exec.status}
                          </Badge>
                          <Badge variant='outline'>{exec.trigger_type}</Badge>
                          {exec.status_code && (
                            <Badge variant='secondary'>
                              {exec.status_code}
                            </Badge>
                          )}
                          {exec.duration_ms && (
                            <span className='text-muted-foreground text-xs'>
                              {exec.duration_ms}ms
                            </span>
                          )}
                        </div>
                        <p className='text-muted-foreground text-xs'>
                          {new Date(exec.executed_at).toLocaleString()}
                        </p>
                      </div>
                    </div>
                  </CardHeader>
                  {(exec.logs || exec.error_message || exec.result) && (
                    <CardContent className='overflow-hidden pt-0'>
                      {exec.error_message && (
                        <div className='mb-2 min-w-0'>
                          <Label className='text-destructive text-xs'>
                            Error:
                          </Label>
                          <div className='bg-destructive/10 mt-1 max-h-40 max-w-full overflow-auto rounded border'>
                            <pre
                              className={`min-w-0 p-2 text-xs ${logsWordWrap ? 'break-words whitespace-pre-wrap' : 'whitespace-pre'}`}
                            >
                              {exec.error_message}
                            </pre>
                          </div>
                        </div>
                      )}
                      {exec.logs && (
                        <div className='mb-2 min-w-0'>
                          <Label className='text-xs'>Logs:</Label>
                          <div className='bg-muted mt-1 max-h-40 max-w-full overflow-auto rounded border'>
                            <pre
                              className={`min-w-0 p-2 text-xs ${logsWordWrap ? 'break-words whitespace-pre-wrap' : 'whitespace-pre'}`}
                            >
                              {exec.logs}
                            </pre>
                          </div>
                        </div>
                      )}
                      {exec.result && !exec.error_message && (
                        <div className='min-w-0'>
                          <Label className='text-xs'>Result:</Label>
                          <div className='bg-muted mt-1 max-h-40 max-w-full overflow-auto rounded border'>
                            <pre
                              className={`min-w-0 p-2 text-xs ${logsWordWrap ? 'break-words whitespace-pre-wrap' : 'whitespace-pre'}`}
                            >
                              {exec.result}
                            </pre>
                          </div>
                        </div>
                      )}
                    </CardContent>
                  )}
                </Card>
              ))
            )}
          </div>
        </DialogContent>
      </Dialog>

      {/* Result Dialog */}
      <Dialog open={showResultDialog} onOpenChange={setShowResultDialog}>
        <DialogContent className='max-h-[95vh] w-[95vw] max-w-[95vw]'>
          <DialogHeader>
            <DialogTitle>
              {invokeResult?.success ? 'Function Result' : 'Function Error'}
            </DialogTitle>
            <DialogDescription>
              {invokeResult?.success
                ? 'Function executed successfully'
                : 'Function execution failed'}
            </DialogDescription>
          </DialogHeader>

          <div className='space-y-4'>
            <div className='flex items-center space-x-2'>
              <Switch
                id='word-wrap'
                checked={wordWrap}
                onCheckedChange={setWordWrap}
              />
              <Label htmlFor='word-wrap' className='cursor-pointer'>
                Word wrap
              </Label>
            </div>

            {invokeResult?.success ? (
              <div className='w-full overflow-hidden'>
                <Label>Response</Label>
                <div className='bg-muted mt-2 h-[70vh] overflow-auto rounded-lg border'>
                  <pre
                    className={`p-4 text-xs ${
                      wordWrap
                        ? 'break-words whitespace-pre-wrap'
                        : 'whitespace-pre'
                    }`}
                  >
                    {invokeResult.data}
                  </pre>
                </div>
              </div>
            ) : (
              <div className='w-full overflow-hidden'>
                <Label>Error</Label>
                <div className='bg-destructive/10 mt-2 h-[70vh] overflow-auto rounded-lg border'>
                  <pre
                    className={`text-destructive p-4 text-xs ${
                      wordWrap
                        ? 'break-words whitespace-pre-wrap'
                        : 'whitespace-pre'
                    }`}
                  >
                    {invokeResult?.error}
                  </pre>
                </div>
              </div>
            )}
          </div>

          <DialogFooter>
            <Button
              variant='outline'
              onClick={() => {
                if (invokeResult?.success) {
                  navigator.clipboard.writeText(invokeResult.data)
                  toast.success('Copied to clipboard')
                }
              }}
              disabled={!invokeResult?.success}
            >
              <Copy className='mr-2 h-4 w-4' />
              Copy
            </Button>
            <Button onClick={() => setShowResultDialog(false)}>Close</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Execution Detail Dialog */}
      <Dialog open={showExecutionDetailDialog} onOpenChange={setShowExecutionDetailDialog}>
        <DialogContent className='max-h-[90vh] w-[90vw] max-w-[1600px] overflow-hidden flex flex-col sm:max-w-none'>
          <DialogHeader>
            <DialogTitle>Execution Details</DialogTitle>
            <DialogDescription>
              {selectedExecution?.function_name || 'Unknown Function'} - {selectedExecution?.id?.slice(0, 8)}
            </DialogDescription>
          </DialogHeader>

          {selectedExecution && (
            <div className='flex flex-col gap-4 overflow-y-auto flex-1 pr-2'>
              {/* Status and Metadata */}
              <div className='grid grid-cols-2 md:grid-cols-4 gap-4'>
                <div>
                  <Label className='text-muted-foreground text-xs'>Status</Label>
                  <div className='mt-1'>
                    <Badge variant={selectedExecution.status === 'success' ? 'secondary' : 'destructive'}>
                      {selectedExecution.status}
                    </Badge>
                  </div>
                </div>
                <div>
                  <Label className='text-muted-foreground text-xs'>Status Code</Label>
                  <p className='font-mono text-sm'>{selectedExecution.status_code ?? '-'}</p>
                </div>
                <div>
                  <Label className='text-muted-foreground text-xs'>Duration</Label>
                  <p className='font-mono text-sm'>
                    {selectedExecution.duration_ms ? `${selectedExecution.duration_ms}ms` : '-'}
                  </p>
                </div>
                <div>
                  <Label className='text-muted-foreground text-xs'>Trigger</Label>
                  <p className='font-mono text-sm'>{selectedExecution.trigger_type}</p>
                </div>
              </div>

              <div className='grid grid-cols-2 gap-4'>
                <div>
                  <Label className='text-muted-foreground text-xs'>Started</Label>
                  <p className='font-mono text-sm'>
                    {new Date(selectedExecution.executed_at).toLocaleString()}
                  </p>
                </div>
                <div>
                  <Label className='text-muted-foreground text-xs'>Completed</Label>
                  <p className='font-mono text-sm'>
                    {selectedExecution.completed_at
                      ? new Date(selectedExecution.completed_at).toLocaleString()
                      : '-'}
                  </p>
                </div>
              </div>

              {/* Result */}
              {selectedExecution.result && (
                <div>
                  <div className='flex items-center justify-between mb-2'>
                    <Label>Result</Label>
                    <Button
                      variant='ghost'
                      size='sm'
                      onClick={() => copyToClipboard(selectedExecution.result || '', 'Result')}
                    >
                      <Copy className='h-3 w-3' />
                    </Button>
                  </div>
                  <pre className='bg-muted rounded-md p-3 text-xs overflow-auto max-h-32'>
                    {selectedExecution.result}
                  </pre>
                </div>
              )}

              {/* Error */}
              {selectedExecution.error_message && (
                <div>
                  <Label className='text-destructive'>Error</Label>
                  <pre className='bg-destructive/10 text-destructive rounded-md p-3 text-xs overflow-auto max-h-32 mt-2'>
                    {selectedExecution.error_message}
                  </pre>
                </div>
              )}

              {/* Logs Section */}
              <div className='flex-1 min-h-0'>
                <div className='flex items-center justify-between mb-2'>
                  <Label>Logs</Label>
                  <div className='flex items-center gap-2'>
                    <Select value={logLevelFilter} onValueChange={setLogLevelFilter}>
                      <SelectTrigger className='w-[120px] h-8 text-xs'>
                        <SelectValue placeholder='Filter level' />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value='all'>All Levels</SelectItem>
                        <SelectItem value='debug'>Debug</SelectItem>
                        <SelectItem value='info'>Info</SelectItem>
                        <SelectItem value='warn'>Warn</SelectItem>
                        <SelectItem value='error'>Error</SelectItem>
                      </SelectContent>
                    </Select>
                    <Button
                      variant='ghost'
                      size='sm'
                      onClick={() => {
                        const logsText = filteredLogs.length > 0
                          ? filteredLogs.map((l) => `[${l.level.toUpperCase()}] ${l.message}`).join('\n')
                          : selectedExecution.logs || ''
                        copyToClipboard(logsText, 'Logs')
                      }}
                    >
                      <Copy className='h-3 w-3' />
                    </Button>
                  </div>
                </div>

                {executionLogsLoading ? (
                  <div className='flex items-center justify-center h-48'>
                    <Loader2 className='h-6 w-6 animate-spin text-muted-foreground' />
                  </div>
                ) : filteredLogs.length > 0 ? (
                  <ScrollArea className='h-64 bg-muted rounded-md'>
                    <div className='p-3 space-y-1'>
                      {filteredLogs.map((log) => (
                        <div key={log.id} className='flex gap-2 text-xs font-mono'>
                          <Badge
                            variant={
                              log.level === 'error' ? 'destructive' :
                              log.level === 'warn' ? 'secondary' :
                              log.level === 'debug' ? 'outline' : 'default'
                            }
                            className='text-[10px] px-1 py-0 h-4 shrink-0'
                          >
                            {log.level.toUpperCase()}
                          </Badge>
                          <span className='break-all'>{log.message}</span>
                        </div>
                      ))}
                    </div>
                  </ScrollArea>
                ) : selectedExecution.logs ? (
                  <pre className='bg-muted rounded-md p-3 text-xs overflow-auto h-64'>
                    {selectedExecution.logs}
                  </pre>
                ) : (
                  <div className='flex items-center justify-center h-48 text-muted-foreground text-sm'>
                    No logs available
                  </div>
                )}
              </div>
            </div>
          )}

          <DialogFooter className='mt-4'>
            <Button onClick={() => setShowExecutionDetailDialog(false)}>Close</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
