import { useState, useEffect, useCallback, useMemo, useRef } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import {
  Terminal,
  RefreshCw,
  HardDrive,
  Activity,
  Search,
  Filter,
  CheckCircle,
  XCircle,
  Loader2,
  Clock,
  Copy,
  AlertCircle,
  Globe,
  Lock,
  StopCircle,
  Eye,
  Code,
  Shield,
  Database,
  Timer,
} from 'lucide-react'
import { toast } from 'sonner'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
} from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
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
import { ImpersonationBanner } from '@/components/impersonation-banner'
import { ImpersonationPopover } from '@/features/impersonation/components/impersonation-popover'
import {
  rpcApi,
  type RPCProcedure,
  type RPCExecution,
  type RPCExecutionLog,
} from '@/lib/api'

export const Route = createFileRoute('/_authenticated/rpc/')({
  component: RPCPage,
})

const RPC_PAGE_SIZE = 50

function RPCPage() {
  return (
    <div className='flex flex-1 flex-col gap-6 p-6'>
      <ImpersonationBanner />

      <div className='flex items-center justify-between'>
        <div>
          <h1 className='text-3xl font-bold'>RPC Procedures</h1>
          <p className='text-muted-foreground'>
            Execute SQL procedures securely via API
          </p>
        </div>
        <ImpersonationPopover
          contextLabel="Executing as"
          defaultReason="Testing RPC procedure execution"
        />
      </div>

      <RPCContent />
    </div>
  )
}

function RPCContent() {
  // Tab state
  const [activeTab, setActiveTab] = useState<'executions' | 'procedures'>('executions')

  // Procedures state
  const [procedures, setProcedures] = useState<RPCProcedure[]>([])
  const [loading, setLoading] = useState(true)
  const [syncing, setSyncing] = useState(false)
  const [namespaces, setNamespaces] = useState<string[]>(['default'])
  const [selectedNamespace, setSelectedNamespace] = useState<string>('default')

  // Executions state
  const [executions, setExecutions] = useState<RPCExecution[]>([])
  const [executionsLoading, setExecutionsLoading] = useState(false)
  const [executionsOffset, setExecutionsOffset] = useState(0)
  const [hasMoreExecutions, setHasMoreExecutions] = useState(true)
  const [loadingMoreExecutions, setLoadingMoreExecutions] = useState(false)
  const [totalExecutions, setTotalExecutions] = useState(0)

  // Filters state
  const [searchQuery, setSearchQuery] = useState('')
  const [statusFilter, setStatusFilter] = useState<string>('all')

  // Ref to track initial fetch (prevents debounced search from re-fetching on mount)
  const hasInitialFetch = useRef(false)

  // Execution detail dialog
  const [selectedExecution, setSelectedExecution] = useState<RPCExecution | null>(null)
  const [showExecutionDetails, setShowExecutionDetails] = useState(false)
  const [executionLogs, setExecutionLogs] = useState<RPCExecutionLog[]>([])
  const [loadingLogs, setLoadingLogs] = useState(false)

  // Cancel execution state
  const [cancellingExecutionId, setCancellingExecutionId] = useState<string | null>(null)

  // Procedure detail dialog
  const [selectedProcedure, setSelectedProcedure] = useState<RPCProcedure | null>(null)
  const [showProcedureDetails, setShowProcedureDetails] = useState(false)
  const [loadingProcedure, setLoadingProcedure] = useState(false)

  // Request tracking to prevent race conditions
  const executionsFetchIdRef = useRef(0)

  // Track if namespaces have been loaded
  const [namespacesLoaded, setNamespacesLoaded] = useState(false)

  // Fetch procedures
  const fetchProcedures = useCallback(async () => {
    setLoading(true)
    try {
      const data = await rpcApi.listProcedures(selectedNamespace)
      setProcedures(data || [])
    } catch {
      toast.error('Failed to fetch RPC procedures')
    } finally {
      setLoading(false)
    }
  }, [selectedNamespace])

  // Fetch executions
  const fetchExecutions = useCallback(async (reset = true) => {
    const isReset = reset
    const fetchId = ++executionsFetchIdRef.current

    if (isReset) {
      setExecutionsLoading(true)
      setExecutionsOffset(0)
    } else {
      setLoadingMoreExecutions(true)
    }

    try {
      const offset = isReset ? 0 : executionsOffset
      const result = await rpcApi.listExecutions({
        namespace: selectedNamespace !== 'all' ? selectedNamespace : undefined,
        procedure: searchQuery || undefined,
        status: statusFilter !== 'all' ? statusFilter as 'pending' | 'running' | 'completed' | 'failed' | 'cancelled' | 'timeout' : undefined,
        limit: RPC_PAGE_SIZE,
        offset,
      })

      // Ignore stale responses from earlier requests
      if (fetchId !== executionsFetchIdRef.current) {
        return
      }

      const execList = result.executions || []
      if (isReset) {
        setExecutions(execList)
        setExecutionsOffset(RPC_PAGE_SIZE)
      } else {
        setExecutions((prev) => [...prev, ...execList])
        setExecutionsOffset((prev) => prev + RPC_PAGE_SIZE)
      }

      setTotalExecutions(result.total || 0)
      setHasMoreExecutions(execList.length >= RPC_PAGE_SIZE)
    } catch {
      // Only show error if this is still the current request
      if (fetchId === executionsFetchIdRef.current) {
        toast.error('Failed to fetch executions')
      }
    } finally {
      // Only update loading state if this is still the current request
      if (fetchId === executionsFetchIdRef.current) {
        setExecutionsLoading(false)
        setLoadingMoreExecutions(false)
      }
    }
    // Note: executionsOffset intentionally excluded from deps to prevent stale closure issues
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedNamespace, searchQuery, statusFilter])

  // Fetch execution logs
  const fetchExecutionLogs = useCallback(async (executionId: string) => {
    setLoadingLogs(true)
    try {
      const logs = await rpcApi.getExecutionLogs(executionId)
      setExecutionLogs(logs || [])
    } catch {
      toast.error('Failed to fetch execution logs')
    } finally {
      setLoadingLogs(false)
    }
  }, [])

  // Open execution detail dialog
  const openExecutionDetails = async (exec: RPCExecution) => {
    setSelectedExecution(exec)
    setShowExecutionDetails(true)
    await fetchExecutionLogs(exec.id)
  }

  // Cancel execution
  const cancelExecution = async (execId: string, e?: React.MouseEvent) => {
    e?.stopPropagation() // Prevent opening the details dialog when clicking cancel
    setCancellingExecutionId(execId)
    try {
      await rpcApi.cancelExecution(execId)
      toast.success('Execution cancelled')
      // Update the execution in the list
      setExecutions((prev) =>
        prev.map((ex) =>
          ex.id === execId ? { ...ex, status: 'cancelled' } : ex
        )
      )
      // Update selected execution if it's the one being cancelled
      if (selectedExecution?.id === execId) {
        setSelectedExecution((prev) => prev ? { ...prev, status: 'cancelled' } : null)
      }
    } catch {
      toast.error('Failed to cancel execution')
    } finally {
      setCancellingExecutionId(null)
    }
  }

  // Check if execution can be cancelled
  const canCancelExecution = (status: string) => {
    return status === 'pending' || status === 'running'
  }

  // Open procedure detail dialog
  const openProcedureDetails = async (proc: RPCProcedure) => {
    setSelectedProcedure(proc) // Show immediately with partial data
    setShowProcedureDetails(true)
    setLoadingProcedure(true)
    try {
      const fullProcedure = await rpcApi.getProcedure(proc.namespace, proc.name)
      setSelectedProcedure(fullProcedure)
    } catch {
      toast.error('Failed to fetch procedure details')
    } finally {
      setLoadingProcedure(false)
    }
  }

  // Filter executions from past 24 hours (for stats display only)
  const executions24h = useMemo(() => {
    const cutoff = Date.now() - 24 * 60 * 60 * 1000
    return executions.filter((exec) => {
      const execTime = new Date(exec.created_at).getTime()
      return execTime >= cutoff
    })
  }, [executions])

  // Calculate stats from past 24 hours
  const executionStats = useMemo(() => {
    const pending = executions24h.filter((e) => e.status === 'pending').length
    const running = executions24h.filter((e) => e.status === 'running').length
    const completed = executions24h.filter((e) => e.status === 'completed').length
    const failed = executions24h.filter((e) => e.status === 'failed' || e.status === 'cancelled' || e.status === 'timeout').length
    const total = executions24h.length
    const avgDuration = executions24h.length > 0
      ? Math.round(executions24h.reduce((sum, e) => sum + (e.duration_ms || 0), 0) / executions24h.length)
      : 0
    return { pending, running, completed, failed, total, avgDuration }
  }, [executions24h])

  // Sync procedures
  const handleSync = async () => {
    setSyncing(true)
    try {
      const result = await rpcApi.sync(selectedNamespace)
      const { created, updated, deleted } = result.summary
      if (created > 0 || updated > 0 || deleted > 0) {
        const messages = []
        if (created > 0) messages.push(`${created} created`)
        if (updated > 0) messages.push(`${updated} updated`)
        if (deleted > 0) messages.push(`${deleted} deleted`)
        toast.success(`Procedures synced: ${messages.join(', ')}`)
      } else {
        toast.info('No changes detected')
      }
      await fetchProcedures()
    } catch {
      toast.error('Failed to sync procedures')
    } finally {
      setSyncing(false)
    }
  }

  // Toggle procedure enabled state
  const toggleProcedure = async (proc: RPCProcedure) => {
    try {
      await rpcApi.updateProcedure(proc.namespace, proc.name, { enabled: !proc.enabled })
      setProcedures((prev) =>
        prev.map((p) =>
          p.id === proc.id ? { ...p, enabled: !p.enabled } : p
        )
      )
      toast.success(`Procedure ${proc.enabled ? 'disabled' : 'enabled'}`)
    } catch {
      toast.error('Failed to update procedure')
    }
  }

  // Fetch namespaces
  useEffect(() => {
    const fetchNamespaces = async () => {
      try {
        const data = await rpcApi.listNamespaces()
        setNamespaces(data.length > 0 ? data : ['default'])
        if (!data.includes(selectedNamespace)) {
          setSelectedNamespace(data[0] || 'default')
        }
      } catch {
        setNamespaces(['default'])
      } finally {
        setNamespacesLoaded(true)
      }
    }
    fetchNamespaces()
  }, [selectedNamespace])

  // Fetch data on mount and namespace change
  useEffect(() => {
    fetchProcedures()
  }, [fetchProcedures])

  // Fetch executions when tab changes or filters change
  // Wait until namespaces are loaded to ensure we have the correct namespace
  useEffect(() => {
    if (activeTab === 'executions' && namespacesLoaded) {
      hasInitialFetch.current = true
      fetchExecutions(true)
    }
  }, [activeTab, selectedNamespace, statusFilter, namespacesLoaded, fetchExecutions])

  // Debounced search - only for search query changes
  useEffect(() => {
    if (activeTab !== 'executions' || !namespacesLoaded) return
    // Skip the first render - the main effect above handles initial fetch
    if (!hasInitialFetch.current) return
    const timer = setTimeout(() => {
      fetchExecutions(true)
    }, 300)
    return () => clearTimeout(timer)
    // Only searchQuery triggers this debounced effect
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchQuery])

  // Copy to clipboard helper
  const copyToClipboard = (text: string, label: string) => {
    navigator.clipboard.writeText(text)
    toast.success(`${label} copied to clipboard`)
  }

  // Get status icon
  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'completed':
        return <CheckCircle className='h-4 w-4 text-green-500 shrink-0' />
      case 'running':
        return <Loader2 className='h-4 w-4 text-blue-500 shrink-0 animate-spin' />
      case 'pending':
        return <Clock className='h-4 w-4 text-yellow-500 shrink-0' />
      case 'failed':
      case 'cancelled':
      case 'timeout':
        return <XCircle className='h-4 w-4 text-red-500 shrink-0' />
      default:
        return <AlertCircle className='h-4 w-4 text-muted-foreground shrink-0' />
    }
  }

  // Get status badge variant
  const getStatusVariant = (status: string): 'secondary' | 'destructive' | 'outline' => {
    switch (status) {
      case 'completed':
        return 'secondary'
      case 'failed':
      case 'cancelled':
      case 'timeout':
        return 'destructive'
      default:
        return 'outline'
    }
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
              <span className='text-muted-foreground text-xs'>Pending:</span>
              <span className='text-sm font-semibold'>
                {executionStats.pending}
              </span>
            </div>
            <div className='flex items-center gap-1'>
              <span className='text-muted-foreground text-xs'>Running:</span>
              <span className='text-sm font-semibold'>
                {executionStats.running}
              </span>
            </div>
            <div className='flex items-center gap-1'>
              <span className='text-muted-foreground text-xs'>Completed:</span>
              <span className='text-sm font-semibold'>
                {executionStats.completed}
              </span>
            </div>
            <div className='flex items-center gap-1'>
              <span className='text-muted-foreground text-xs'>Failed:</span>
              <span className='text-sm font-semibold'>
                {executionStats.failed}
              </span>
            </div>
            <div className='flex items-center gap-1'>
              <span className='text-muted-foreground text-xs'>Success:</span>
              {(() => {
                const total = executionStats.completed + executionStats.failed
                const successRate =
                  total > 0
                    ? ((executionStats.completed / total) * 100).toFixed(0)
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
        onValueChange={(v) => setActiveTab(v as 'executions' | 'procedures')}
        className='flex min-h-0 flex-1 flex-col'
      >
        <div className='flex items-center justify-between mb-4'>
          <TabsList className='grid w-full max-w-md grid-cols-2'>
            <TabsTrigger value='executions'>
              <Activity className='mr-2 h-4 w-4' />
              Execution Logs
            </TabsTrigger>
            <TabsTrigger value='procedures'>
              <Terminal className='mr-2 h-4 w-4' />
              Procedures
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
              <Select value={selectedNamespace} onValueChange={setSelectedNamespace}>
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
                placeholder='Search by procedure name...'
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className='pl-9'
              />
            </div>
            <Select value={statusFilter} onValueChange={setStatusFilter}>
              <SelectTrigger className='w-[150px]'>
                <Filter className='mr-2 h-4 w-4' />
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value='all'>All Status</SelectItem>
                <SelectItem value='pending'>Pending</SelectItem>
                <SelectItem value='running'>Running</SelectItem>
                <SelectItem value='completed'>Completed</SelectItem>
                <SelectItem value='failed'>Failed</SelectItem>
                <SelectItem value='cancelled'>Cancelled</SelectItem>
                <SelectItem value='timeout'>Timeout</SelectItem>
              </SelectContent>
            </Select>
            <Button
              onClick={() => fetchExecutions(true)}
              variant='outline'
              size='sm'
            >
              <RefreshCw className='mr-2 h-4 w-4' />
              Refresh
            </Button>
          </div>

          {/* Executions List */}
          <ScrollArea className='h-[calc(100vh-24rem)]'>
            {executionsLoading ? (
              <div className='flex h-48 items-center justify-center'>
                <Loader2 className='h-8 w-8 animate-spin text-muted-foreground' />
              </div>
            ) : executions.length === 0 ? (
              <Card>
                <CardContent className='p-12 text-center'>
                  <Activity className='text-muted-foreground mx-auto mb-4 h-12 w-12' />
                  <p className='mb-2 text-lg font-medium'>No executions found</p>
                  <p className='text-muted-foreground text-sm'>
                    Execute some RPC procedures to see their logs here
                  </p>
                </CardContent>
              </Card>
            ) : (
              <div className='grid gap-1'>
                {executions.map((exec) => (
                  <div
                    key={exec.id}
                    className='flex items-center justify-between gap-2 px-3 py-2 rounded-md border hover:border-primary/50 transition-colors bg-card cursor-pointer'
                    onClick={() => openExecutionDetails(exec)}
                  >
                    <div className='flex items-center gap-3 min-w-0 flex-1'>
                      {getStatusIcon(exec.status)}
                      <span className='text-sm font-medium truncate'>
                        {exec.procedure_name}
                      </span>
                      <Badge variant={getStatusVariant(exec.status)} className='shrink-0 text-[10px] px-1.5 py-0 h-4'>
                        {exec.status}
                      </Badge>
                      {exec.user_email && (
                        <span className='text-xs text-muted-foreground truncate'>
                          {exec.user_email}
                        </span>
                      )}
                    </div>
                    <div className='flex items-center gap-3 shrink-0'>
                      {exec.rows_returned !== undefined && (
                        <span className='text-xs text-muted-foreground'>
                          {exec.rows_returned} rows
                        </span>
                      )}
                      <span className='text-xs text-muted-foreground'>
                        {exec.duration_ms ? `${exec.duration_ms}ms` : '-'}
                      </span>
                      <span className='text-xs text-muted-foreground'>
                        {new Date(exec.created_at).toLocaleString()}
                      </span>
                      {canCancelExecution(exec.status) && (
                        <Button
                          variant='ghost'
                          size='sm'
                          className='h-6 px-2 text-destructive hover:text-destructive hover:bg-destructive/10'
                          onClick={(e) => cancelExecution(exec.id, e)}
                          disabled={cancellingExecutionId === exec.id}
                        >
                          {cancellingExecutionId === exec.id ? (
                            <Loader2 className='h-3 w-3 animate-spin' />
                          ) : (
                            <StopCircle className='h-3 w-3' />
                          )}
                          <span className='ml-1 text-xs'>Cancel</span>
                        </Button>
                      )}
                    </div>
                  </div>
                ))}
                {hasMoreExecutions && (
                  <div className='mt-4 flex flex-col items-center gap-2'>
                    <span className='text-xs text-muted-foreground'>
                      Showing {executions.length} of {totalExecutions} executions
                    </span>
                    <Button
                      variant='outline'
                      onClick={() => fetchExecutions(false)}
                      disabled={loadingMoreExecutions}
                    >
                      {loadingMoreExecutions ? (
                        <>
                          <Loader2 className='mr-2 h-4 w-4 animate-spin' />
                          Loading...
                        </>
                      ) : (
                        'Load More'
                      )}
                    </Button>
                  </div>
                )}
              </div>
            )}
          </ScrollArea>
        </TabsContent>

        {/* Procedures Tab */}
        <TabsContent value='procedures' className='flex-1 mt-0'>
          {/* Procedures Controls */}
          <div className='flex items-center justify-end gap-2 mb-4'>
            <div className='flex items-center gap-2'>
              <Label htmlFor='proc-namespace-select' className='text-sm text-muted-foreground whitespace-nowrap'>
                Namespace:
              </Label>
              <Select value={selectedNamespace} onValueChange={setSelectedNamespace}>
                <SelectTrigger id='proc-namespace-select' className='w-[180px]'>
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
              onClick={handleSync}
              variant='outline'
              size='sm'
              disabled={syncing}
            >
              {syncing ? (
                <>
                  <RefreshCw className='mr-2 h-4 w-4 animate-spin' />
                  Syncing...
                </>
              ) : (
                <>
                  <HardDrive className='mr-2 h-4 w-4' />
                  Sync from Filesystem
                </>
              )}
            </Button>
            <Button
              onClick={() => fetchProcedures()}
              variant='outline'
              size='sm'
            >
              <RefreshCw className='mr-2 h-4 w-4' />
              Refresh
            </Button>
          </div>

          {/* Procedures Stats */}
          <div className='flex gap-4 text-sm mb-4'>
            <div className='flex items-center gap-1.5'>
              <span className='text-muted-foreground'>Total:</span>
              <Badge variant='secondary' className='h-5 px-2'>
                {procedures.length}
              </Badge>
            </div>
            <div className='flex items-center gap-1.5'>
              <span className='text-muted-foreground'>Enabled:</span>
              <Badge variant='secondary' className='h-5 px-2 bg-green-500/10 text-green-600 dark:text-green-400'>
                {procedures.filter((p) => p.enabled).length}
              </Badge>
            </div>
            <div className='flex items-center gap-1.5'>
              <span className='text-muted-foreground'>Public:</span>
              <Badge variant='secondary' className='h-5 px-2'>
                {procedures.filter((p) => p.is_public).length}
              </Badge>
            </div>
          </div>

          {/* Procedures List */}
          <ScrollArea className='h-[calc(100vh-20rem)]'>
            <div className='grid gap-1'>
              {procedures.length === 0 ? (
                <Card>
                  <CardContent className='p-12 text-center'>
                    <Terminal className='text-muted-foreground mx-auto mb-4 h-12 w-12' />
                    <p className='mb-2 text-lg font-medium'>
                      No RPC procedures yet
                    </p>
                    <p className='text-muted-foreground mb-4 text-sm'>
                      Sync procedures from the filesystem to get started
                    </p>
                    <Button onClick={handleSync} disabled={syncing}>
                      <HardDrive className='mr-2 h-4 w-4' />
                      Sync from Filesystem
                    </Button>
                  </CardContent>
                </Card>
              ) : (
                procedures.map((proc) => (
                  <div
                    key={proc.id}
                    className='flex items-center justify-between gap-2 px-3 py-2 rounded-md border hover:border-primary/50 transition-colors bg-card cursor-pointer'
                    onClick={() => openProcedureDetails(proc)}
                  >
                    <div className='flex items-center gap-3 min-w-0 flex-1'>
                      <span className='text-sm font-medium truncate'>{proc.name}</span>
                      <Badge variant='outline' className='shrink-0 text-[10px] px-1 py-0 h-4'>
                        v{proc.version}
                      </Badge>
                      {proc.is_public ? (
                        <Badge variant='outline' className='shrink-0 text-[10px] px-1 py-0 h-4'>
                          <Globe className='mr-0.5 h-2.5 w-2.5' />
                          public
                        </Badge>
                      ) : (
                        <Badge variant='outline' className='shrink-0 text-[10px] px-1 py-0 h-4'>
                          <Lock className='mr-0.5 h-2.5 w-2.5' />
                          private
                        </Badge>
                      )}
                      {proc.require_role && (
                        <Badge variant='outline' className='shrink-0 text-[10px] px-1 py-0 h-4'>
                          role: {proc.require_role}
                        </Badge>
                      )}
                      <Switch
                        checked={proc.enabled}
                        onCheckedChange={() => toggleProcedure(proc)}
                        onClick={(e) => e.stopPropagation()}
                        className='scale-75'
                      />
                    </div>
                    <div className='flex items-center gap-2 shrink-0'>
                      <span className='text-[10px] text-muted-foreground'>
                        {proc.max_execution_time_seconds}s max
                      </span>
                      {proc.description && (
                        <span className='text-xs text-muted-foreground truncate max-w-[200px]' title={proc.description}>
                          {proc.description}
                        </span>
                      )}
                      <Button
                        variant='ghost'
                        size='sm'
                        className='h-6 px-2'
                        onClick={(e) => {
                          e.stopPropagation()
                          openProcedureDetails(proc)
                        }}
                      >
                        <Eye className='h-3 w-3' />
                        <span className='ml-1 text-xs'>View</span>
                      </Button>
                    </div>
                  </div>
                ))
              )}
            </div>
          </ScrollArea>
        </TabsContent>
      </Tabs>

      {/* Execution Details Dialog */}
      <Dialog open={showExecutionDetails} onOpenChange={setShowExecutionDetails}>
        <DialogContent className='max-h-[90vh] max-w-4xl overflow-y-auto'>
          <DialogHeader>
            <DialogTitle className='flex items-center gap-2'>
              {selectedExecution && getStatusIcon(selectedExecution.status)}
              Execution Details
            </DialogTitle>
            <DialogDescription>
              {selectedExecution?.procedure_name} - {selectedExecution?.id}
            </DialogDescription>
          </DialogHeader>

          {selectedExecution && (
            <div className='space-y-4'>
              {/* Status and Info */}
              <div className='flex flex-wrap items-center gap-2'>
                <Badge variant={getStatusVariant(selectedExecution.status)}>
                  {selectedExecution.status}
                </Badge>
                {selectedExecution.user_email && (
                  <Badge variant='outline'>{selectedExecution.user_email}</Badge>
                )}
                {selectedExecution.user_role && (
                  <Badge variant='outline'>role: {selectedExecution.user_role}</Badge>
                )}
                {selectedExecution.is_async && (
                  <Badge variant='outline'>async</Badge>
                )}
                {canCancelExecution(selectedExecution.status) && (
                  <Button
                    variant='destructive'
                    size='sm'
                    className='ml-auto'
                    onClick={() => cancelExecution(selectedExecution.id)}
                    disabled={cancellingExecutionId === selectedExecution.id}
                  >
                    {cancellingExecutionId === selectedExecution.id ? (
                      <Loader2 className='mr-2 h-4 w-4 animate-spin' />
                    ) : (
                      <StopCircle className='mr-2 h-4 w-4' />
                    )}
                    Cancel Execution
                  </Button>
                )}
              </div>

              {/* Timestamps */}
              <div className='grid grid-cols-3 gap-4 text-sm'>
                <div>
                  <span className='text-muted-foreground'>Created:</span>
                  <p>{new Date(selectedExecution.created_at).toLocaleString()}</p>
                </div>
                {selectedExecution.started_at && (
                  <div>
                    <span className='text-muted-foreground'>Started:</span>
                    <p>{new Date(selectedExecution.started_at).toLocaleString()}</p>
                  </div>
                )}
                {selectedExecution.completed_at && (
                  <div>
                    <span className='text-muted-foreground'>Completed:</span>
                    <p>{new Date(selectedExecution.completed_at).toLocaleString()}</p>
                  </div>
                )}
              </div>

              {/* Duration and Rows */}
              <div className='flex gap-4 text-sm'>
                {selectedExecution.duration_ms !== undefined && (
                  <div>
                    <span className='text-muted-foreground'>Duration: </span>
                    <span className='font-medium'>{selectedExecution.duration_ms}ms</span>
                  </div>
                )}
                {selectedExecution.rows_returned !== undefined && (
                  <div>
                    <span className='text-muted-foreground'>Rows Returned: </span>
                    <span className='font-medium'>{selectedExecution.rows_returned}</span>
                  </div>
                )}
              </div>

              {/* Input Params */}
              {selectedExecution.input_params && Object.keys(selectedExecution.input_params).length > 0 && (
                <div>
                  <div className='flex items-center justify-between mb-2'>
                    <Label>Input Parameters</Label>
                    <Button
                      variant='ghost'
                      size='sm'
                      onClick={() => copyToClipboard(JSON.stringify(selectedExecution.input_params, null, 2), 'Input params')}
                    >
                      <Copy className='h-3 w-3' />
                    </Button>
                  </div>
                  <pre className='bg-muted rounded-md p-3 text-xs overflow-auto max-h-32'>
                    {JSON.stringify(selectedExecution.input_params, null, 2)}
                  </pre>
                </div>
              )}

              {/* Result */}
              {selectedExecution.result !== undefined && selectedExecution.result !== null && (
                <div>
                  <div className='flex items-center justify-between mb-2'>
                    <Label>Result</Label>
                    <Button
                      variant='ghost'
                      size='sm'
                      onClick={() => copyToClipboard(JSON.stringify(selectedExecution.result, null, 2), 'Result')}
                    >
                      <Copy className='h-3 w-3' />
                    </Button>
                  </div>
                  <pre className='bg-muted rounded-md p-3 text-xs overflow-auto max-h-48'>
                    {typeof selectedExecution.result === 'string'
                      ? selectedExecution.result
                      : JSON.stringify(selectedExecution.result, null, 2)}
                  </pre>
                </div>
              )}

              {/* Error */}
              {selectedExecution.error_message && (
                <div>
                  <Label className='text-destructive'>Error</Label>
                  <div className='bg-destructive/10 rounded-md p-3 mt-2'>
                    <p className='text-sm text-destructive'>{selectedExecution.error_message}</p>
                  </div>
                </div>
              )}

              {/* Logs */}
              <div>
                <div className='flex items-center justify-between mb-2'>
                  <Label>Logs</Label>
                  {executionLogs.length > 0 && (
                    <Button
                      variant='ghost'
                      size='sm'
                      onClick={() => copyToClipboard(executionLogs.map(l => `[${l.level}] ${l.message}`).join('\n'), 'Logs')}
                    >
                      <Copy className='h-3 w-3' />
                    </Button>
                  )}
                </div>
                <ScrollArea className='h-48 rounded-md border bg-muted p-3'>
                  {loadingLogs ? (
                    <div className='flex items-center justify-center h-full'>
                      <Loader2 className='h-6 w-6 animate-spin text-muted-foreground' />
                    </div>
                  ) : executionLogs.length === 0 ? (
                    <p className='text-sm text-muted-foreground text-center'>No logs available</p>
                  ) : (
                    <div className='space-y-1'>
                      {executionLogs.map((log) => (
                        <div key={log.id} className='text-xs font-mono'>
                          <span className={
                            log.level === 'error' ? 'text-red-500' :
                              log.level === 'warn' ? 'text-yellow-500' :
                                log.level === 'info' ? 'text-blue-500' :
                                  'text-muted-foreground'
                          }>
                            [{log.level}]
                          </span>
                          <span className='ml-2'>{log.message}</span>
                        </div>
                      ))}
                    </div>
                  )}
                </ScrollArea>
              </div>
            </div>
          )}
        </DialogContent>
      </Dialog>

      {/* Procedure Details Dialog */}
      <Dialog open={showProcedureDetails} onOpenChange={setShowProcedureDetails}>
        <DialogContent className='max-h-[90vh] max-w-5xl w-[90vw] overflow-y-auto'>
          <DialogHeader>
            <DialogTitle className='flex items-center gap-2'>
              <Terminal className='h-5 w-5' />
              Procedure Details
            </DialogTitle>
            <DialogDescription>
              {selectedProcedure?.namespace}/{selectedProcedure?.name}
            </DialogDescription>
          </DialogHeader>

          {selectedProcedure && (
            <div className='space-y-4'>
              {/* Status Badges */}
              <div className='flex flex-wrap items-center gap-2'>
                <Badge variant={selectedProcedure.enabled ? 'secondary' : 'outline'}>
                  {selectedProcedure.enabled ? 'Enabled' : 'Disabled'}
                </Badge>
                <Badge variant='outline'>v{selectedProcedure.version}</Badge>
                {selectedProcedure.is_public ? (
                  <Badge variant='outline'>
                    <Globe className='mr-1 h-3 w-3' />
                    Public
                  </Badge>
                ) : (
                  <Badge variant='outline'>
                    <Lock className='mr-1 h-3 w-3' />
                    Private
                  </Badge>
                )}
                {selectedProcedure.require_role && (
                  <Badge variant='outline'>
                    <Shield className='mr-1 h-3 w-3' />
                    Role: {selectedProcedure.require_role}
                  </Badge>
                )}
                <Badge variant='outline'>
                  <Timer className='mr-1 h-3 w-3' />
                  {selectedProcedure.max_execution_time_seconds}s timeout
                </Badge>
              </div>

              {/* Description */}
              {selectedProcedure.description && (
                <div>
                  <Label className='text-muted-foreground text-xs'>Description</Label>
                  <p className='text-sm mt-1'>{selectedProcedure.description}</p>
                </div>
              )}

              {/* RPC Source Code */}
              {loadingProcedure ? (
                <div className='flex items-center justify-center py-8'>
                  <Loader2 className='h-6 w-6 animate-spin text-muted-foreground' />
                </div>
              ) : selectedProcedure.original_code ? (
                <div>
                  <div className='flex items-center justify-between mb-2'>
                    <Label className='flex items-center gap-1'>
                      <Code className='h-3 w-3' />
                      RPC Code
                    </Label>
                    <Button
                      variant='ghost'
                      size='sm'
                      onClick={() => copyToClipboard(selectedProcedure.original_code!, 'RPC code')}
                    >
                      <Copy className='h-3 w-3' />
                    </Button>
                  </div>
                  <pre className='bg-muted rounded-md p-3 text-xs overflow-x-auto max-h-64 font-mono whitespace-pre-wrap'>
                    {selectedProcedure.original_code}
                  </pre>
                </div>
              ) : null}

              {/* SDK Usage Example */}
              <div>
                <div className='flex items-center justify-between mb-2'>
                  <Label className='flex items-center gap-1'>
                    <Terminal className='h-3 w-3' />
                    SDK Usage
                  </Label>
                  <Button
                    variant='ghost'
                    size='sm'
                    onClick={() => {
                      const params = selectedProcedure.input_schema
                        ? Object.keys(selectedProcedure.input_schema).map(k => `${k}: /* ${selectedProcedure.input_schema![k]} */`).join(', ')
                        : ''
                      const code = `const { data, error } = await client.rpc.invoke('${selectedProcedure.name}'${params ? `, { ${params} }` : ''})`
                      copyToClipboard(code, 'SDK usage')
                    }}
                  >
                    <Copy className='h-3 w-3' />
                  </Button>
                </div>
                <pre className='bg-muted rounded-md p-3 text-xs overflow-x-auto font-mono whitespace-pre-wrap'>
                  {`const { data, error } = await client.rpc.invoke('${selectedProcedure.name}'${selectedProcedure.input_schema && Object.keys(selectedProcedure.input_schema).length > 0
                      ? `, {\n${Object.entries(selectedProcedure.input_schema).map(([k, v]) => `  ${k}: /* ${v} */`).join(',\n')}\n}`
                      : ''
                    })`}
                </pre>
              </div>

              {/* Input/Output Schema */}
              <div className='grid grid-cols-2 gap-4'>
                {selectedProcedure.input_schema && Object.keys(selectedProcedure.input_schema).length > 0 && (
                  <div>
                    <Label className='text-muted-foreground text-xs'>Input Parameters</Label>
                    <div className='mt-1 space-y-1'>
                      {Object.entries(selectedProcedure.input_schema).map(([name, type]) => (
                        <div key={name} className='flex items-center gap-2 text-sm'>
                          <code className='bg-muted px-1 rounded text-xs'>{name}</code>
                          <span className='text-muted-foreground text-xs'>{type}</span>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
                {selectedProcedure.output_schema && Object.keys(selectedProcedure.output_schema).length > 0 && (
                  <div>
                    <Label className='text-muted-foreground text-xs'>Output Columns</Label>
                    <div className='mt-1 space-y-1'>
                      {Object.entries(selectedProcedure.output_schema).map(([name, type]) => (
                        <div key={name} className='flex items-center gap-2 text-sm'>
                          <code className='bg-muted px-1 rounded text-xs'>{name}</code>
                          <span className='text-muted-foreground text-xs'>{type}</span>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>

              {/* Access Control */}
              <div className='grid grid-cols-2 gap-4'>
                {selectedProcedure.allowed_tables && selectedProcedure.allowed_tables.length > 0 && (
                  <div>
                    <Label className='flex items-center gap-1 text-muted-foreground text-xs'>
                      <Database className='h-3 w-3' />
                      Allowed Tables
                    </Label>
                    <div className='mt-1 flex flex-wrap gap-1'>
                      {selectedProcedure.allowed_tables.map((table) => (
                        <Badge key={table} variant='outline' className='text-xs'>
                          {table}
                        </Badge>
                      ))}
                    </div>
                  </div>
                )}
                {selectedProcedure.allowed_schemas && selectedProcedure.allowed_schemas.length > 0 && (
                  <div>
                    <Label className='flex items-center gap-1 text-muted-foreground text-xs'>
                      <Database className='h-3 w-3' />
                      Allowed Schemas
                    </Label>
                    <div className='mt-1 flex flex-wrap gap-1'>
                      {selectedProcedure.allowed_schemas.map((schema) => (
                        <Badge key={schema} variant='outline' className='text-xs'>
                          {schema}
                        </Badge>
                      ))}
                    </div>
                  </div>
                )}
              </div>

              {/* Metadata */}
              <div className='grid grid-cols-3 gap-4 text-sm pt-2 border-t'>
                <div>
                  <span className='text-muted-foreground text-xs'>Source:</span>
                  <p className='text-xs'>{selectedProcedure.source}</p>
                </div>
                <div>
                  <span className='text-muted-foreground text-xs'>Created:</span>
                  <p className='text-xs'>{new Date(selectedProcedure.created_at).toLocaleString()}</p>
                </div>
                <div>
                  <span className='text-muted-foreground text-xs'>Updated:</span>
                  <p className='text-xs'>{new Date(selectedProcedure.updated_at).toLocaleString()}</p>
                </div>
              </div>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </>
  )
}
