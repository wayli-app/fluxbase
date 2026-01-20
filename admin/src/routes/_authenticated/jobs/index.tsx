import { useState, useEffect, useCallback, useRef, useMemo } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import {
  ListTodo,
  Search,
  RefreshCw,
  Clock,
  XCircle,
  Activity,
  CheckCircle,
  AlertCircle,
  Loader2,
  Filter,
  HardDrive,
  Timer,
  Target,
  Play,
  Copy,
  ChevronDown,
  History,
  Edit,
  Trash2,
} from 'lucide-react'
import { toast } from 'sonner'
import { useImpersonationStore } from '@/stores/impersonation-store'
import {
  jobsApi,
  type JobFunction,
  type Job,
  type JobWorker,
  type LogLevel,
} from '@/lib/api'
import { fluxbaseClient } from '@/lib/fluxbase-client'
import {
  useExecutionLogs,
  type ExecutionLog,
  type ExecutionLogLevel,
} from '@/hooks/use-execution-logs'
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
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
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
import { Separator } from '@/components/ui/separator'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { ImpersonationBanner } from '@/components/impersonation-banner'
import { ImpersonationPopover } from '@/features/impersonation/components/impersonation-popover'

export const Route = createFileRoute('/_authenticated/jobs/')({
  component: JobsPage,
})

const JOBS_PAGE_SIZE = 50

function JobsPage() {
  const [activeTab, setActiveTab] = useState<'functions' | 'queue'>('queue')
  const [jobFunctions, setJobFunctions] = useState<JobFunction[]>([])
  const [jobs, setJobs] = useState<Job[]>([])
  const [workers, setWorkers] = useState<JobWorker[]>([])
  const [loading, setLoading] = useState(true)
  const [searchQuery, setSearchQuery] = useState('')
  const [statusFilter, setStatusFilter] = useState<string>('all')
  const [selectedJob, setSelectedJob] = useState<Job | null>(null)
  const [showJobDetails, setShowJobDetails] = useState(false)
  const [syncing, setSyncing] = useState(false)
  const [namespaces, setNamespaces] = useState<string[]>(['default'])
  const [selectedNamespace, setSelectedNamespace] = useState<string>('default')

  // Pagination state
  const [jobsOffset, setJobsOffset] = useState(0)
  const [hasMoreJobs, setHasMoreJobs] = useState(true)
  const [loadingMore, setLoadingMore] = useState(false)

  // Run job dialog state
  const [showRunDialog, setShowRunDialog] = useState(false)
  const [selectedFunction, setSelectedFunction] = useState<JobFunction | null>(
    null
  )
  const [jobPayload, setJobPayload] = useState('')
  const [submittingJob, setSubmittingJob] = useState(false)
  const [togglingJob, setTogglingJob] = useState<string | null>(null)

  // Edit job function dialog state
  const [showEditDialog, setShowEditDialog] = useState(false)
  const [fetchingFunction, setFetchingFunction] = useState(false)
  const [editFormData, setEditFormData] = useState({
    description: '',
    code: '',
    timeout_seconds: 30,
    max_retries: 3,
    schedule: '',
  })

  // Delete confirmation state
  const [deleteConfirm, setDeleteConfirm] = useState<{ namespace: string; name: string } | null>(null)

  // Execution history dialog state
  const [showHistoryDialog, setShowHistoryDialog] = useState(false)
  const [historyJobs, setHistoryJobs] = useState<Job[]>([])
  const [historyLoading, setHistoryLoading] = useState(false)

  // Execution logs state
  const [logLevelFilter, setLogLevelFilter] = useState<LogLevel | 'all'>('all')

  // Ref for auto-scrolling logs
  const logsContainerRef = useRef<HTMLDivElement>(null)
  const isAtBottomRef = useRef<boolean>(true)

  // Helper to check if scrolled to bottom (with small threshold for rounding)
  const checkIfAtBottom = () => {
    if (!logsContainerRef.current) return true
    const { scrollTop, scrollHeight, clientHeight } = logsContainerRef.current
    return scrollHeight - scrollTop - clientHeight < 20
  }

  // Use the real-time execution logs hook
  const { logs: executionLogs, loading: loadingLogs } = useExecutionLogs({
    executionId: selectedJob?.id || null,
    executionType: 'job',
    enabled: showJobDetails,
    onNewLog: () => {
      // Check if at bottom BEFORE the log is added
      isAtBottomRef.current = checkIfAtBottom()
      // Auto-scroll only if user was at bottom
      setTimeout(() => {
        if (isAtBottomRef.current && logsContainerRef.current) {
          logsContainerRef.current.scrollTop =
            logsContainerRef.current.scrollHeight
        }
      }, 50)
    },
  })

  // Fetch namespaces on mount and select best default
  useEffect(() => {
    const fetchNamespaces = async () => {
      try {
        const data = await jobsApi.listNamespaces()
        const validNamespaces = data.length > 0 ? data : ['default']
        setNamespaces(validNamespaces)

        // Smart namespace selection: if 'default' is empty but other namespaces have items,
        // select a non-empty namespace instead
        let bestNamespace = validNamespaces[0] || 'default'

        if (validNamespaces.includes('default') && validNamespaces.length > 1) {
          // Check if 'default' namespace has any jobs
          try {
            const defaultJobs = await jobsApi.listFunctions('default')
            if (!defaultJobs || defaultJobs.length === 0) {
              // Default is empty, find first non-empty namespace
              for (const ns of validNamespaces) {
                if (ns !== 'default') {
                  const nsJobs = await jobsApi.listFunctions(ns)
                  if (nsJobs && nsJobs.length > 0) {
                    bestNamespace = ns
                    break
                  }
                }
              }
            }
          } catch {
            // If checking fails, stick with default
          }
        }

        // If current namespace not in list, reset to best available
        if (!validNamespaces.includes(selectedNamespace)) {
          setSelectedNamespace(bestNamespace)
        }
      } catch {
        setNamespaces(['default'])
      }
    }
    fetchNamespaces()
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])  // Only run on mount

  const fetchJobFunctions = useCallback(async () => {
    try {
      const data = await jobsApi.listFunctions(selectedNamespace)
      setJobFunctions(data || [])
    } catch {
      toast.error('Failed to fetch job functions')
    }
  }, [selectedNamespace])

  const fetchJobs = useCallback(
    async (reset = true) => {
      try {
        const offset = reset ? 0 : jobsOffset
        const filters: {
          status?: string
          namespace?: string
          limit: number
          offset: number
        } = {
          limit: JOBS_PAGE_SIZE,
          offset,
          namespace: selectedNamespace,
        }
        if (statusFilter !== 'all') {
          filters.status = statusFilter
        }
        const data = await jobsApi.listJobs(filters)
        const newJobs = data || []

        if (reset) {
          setJobs(newJobs)
          setJobsOffset(JOBS_PAGE_SIZE)
        } else {
          setJobs((prev) => [...prev, ...newJobs])
          setJobsOffset((prev) => prev + JOBS_PAGE_SIZE)
        }

        // If we got fewer jobs than requested, there are no more
        setHasMoreJobs(newJobs.length >= JOBS_PAGE_SIZE)
      } catch {
        toast.error('Failed to fetch jobs')
      }
    },
    [selectedNamespace, statusFilter, jobsOffset]
  )

  const loadMoreJobs = useCallback(async () => {
    setLoadingMore(true)
    try {
      await fetchJobs(false)
    } finally {
      setLoadingMore(false)
    }
  }, [fetchJobs])

  // Subscribe to job updates via Realtime when modal is open and job is running/pending
  // Note: Execution logs are now handled by the useExecutionLogs hook
  useEffect(() => {
    if (!showJobDetails || !selectedJob) return

    // Only subscribe for active jobs
    const isActiveJob =
      selectedJob.status === 'running' || selectedJob.status === 'pending'
    if (!isActiveJob) return

    const channel = fluxbaseClient
      .channel(`job-details-${selectedJob.id}`)
      .on(
        'postgres_changes',
        {
          event: 'UPDATE',
          schema: 'jobs',
          table: 'queue',
          filter: `id=eq.${selectedJob.id}`,
        },
        (payload) => {
          const updatedJob = payload.new as Job
          setSelectedJob(updatedJob)

          // Refresh the jobs list when job completes
          if (
            updatedJob.status !== 'running' &&
            updatedJob.status !== 'pending'
          ) {
            fetchJobs(true)
          }
        }
      )
      .subscribe()

    return () => {
      channel.unsubscribe()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- We only want to restart subscription when job ID or status changes
  }, [showJobDetails, selectedJob?.id, selectedJob?.status, fetchJobs])

  // Reset scroll position ref when opening a new job
  useEffect(() => {
    if (showJobDetails && selectedJob?.id) {
      isAtBottomRef.current = true // Start at bottom for new jobs
    }
  }, [showJobDetails, selectedJob?.id])

  // Subscribe to jobs.queue changes via Realtime for live updates
  useEffect(() => {
    const channel = fluxbaseClient
      .channel('jobs-queue-updates')
      .on(
        'postgres_changes',
        {
          event: '*',
          schema: 'jobs',
          table: 'queue',
        },
        (payload) => {
          const eventType = payload.eventType
          const newJob = payload.new as Job | undefined
          const oldJob = payload.old as { id: string } | undefined

          setJobs((prev) => {
            if (eventType === 'INSERT' && newJob) {
              // Check if job matches current filters
              if (selectedNamespace && newJob.namespace !== selectedNamespace) {
                return prev
              }
              if (statusFilter !== 'all' && newJob.status !== statusFilter) {
                return prev
              }
              // Add to beginning of list if not already present
              if (prev.some((j) => j.id === newJob.id)) {
                return prev
              }
              return [newJob, ...prev]
            }

            if (eventType === 'UPDATE' && newJob) {
              // Update existing job in list
              const idx = prev.findIndex((j) => j.id === newJob.id)
              if (idx === -1) {
                // Job not in current list - might match filters now
                if (
                  selectedNamespace &&
                  newJob.namespace !== selectedNamespace
                ) {
                  return prev
                }
                if (statusFilter !== 'all' && newJob.status !== statusFilter) {
                  return prev
                }
                return [newJob, ...prev]
              }
              // Check if job should be removed due to filter change
              if (statusFilter !== 'all' && newJob.status !== statusFilter) {
                return prev.filter((j) => j.id !== newJob.id)
              }
              const updated = [...prev]
              updated[idx] = newJob
              return updated
            }

            if (eventType === 'DELETE' && oldJob) {
              return prev.filter((j) => j.id !== oldJob.id)
            }

            return prev
          })
        }
      )
      .subscribe()

    return () => {
      channel.unsubscribe()
    }
  }, [selectedNamespace, statusFilter])

  const fetchWorkers = useCallback(async () => {
    try {
      const data = await jobsApi.listWorkers()
      setWorkers(data || [])
    } catch {
      // Silently fail - workers are optional
    }
  }, [])

  // Function to refresh all data (for manual refresh button)
  const refreshAllData = useCallback(async () => {
    setLoading(true)
    setJobsOffset(0)
    setHasMoreJobs(true)
    try {
      await Promise.all([fetchJobFunctions(), fetchJobs(true), fetchWorkers()])
    } finally {
      setLoading(false)
    }
  }, [fetchJobFunctions, fetchJobs, fetchWorkers])

  // Initial data fetch - runs once on mount
  useEffect(() => {
    const loadInitialData = async () => {
      setLoading(true)
      try {
        // Fetch namespaces first
        const nsData = await jobsApi.listNamespaces()
        const availableNamespaces = nsData.length > 0 ? nsData : ['default']
        setNamespaces(availableNamespaces)

        // Use 'default' namespace or first available
        const ns = availableNamespaces.includes('default')
          ? 'default'
          : availableNamespaces[0]

        // Fetch functions, jobs, and workers in parallel
        const [functionsData, jobsData, workersData] = await Promise.all([
          jobsApi.listFunctions(ns),
          jobsApi.listJobs({ namespace: ns, limit: JOBS_PAGE_SIZE, offset: 0 }),
          jobsApi.listWorkers(),
        ])

        setJobFunctions(functionsData || [])
        setJobs(jobsData || [])
        setJobsOffset(JOBS_PAGE_SIZE)
        setHasMoreJobs((jobsData || []).length >= JOBS_PAGE_SIZE)
        setWorkers(workersData || [])
      } catch {
        toast.error('Failed to load jobs data')
      } finally {
        setLoading(false)
      }
    }
    loadInitialData()
  }, []) // Empty deps - only run once on mount

  // Refetch jobs when namespace or status filter changes (after initial load)
  const isInitialMount = useRef(true)
  useEffect(() => {
    // Skip initial mount since loadInitialData already fetches jobs
    if (isInitialMount.current) {
      isInitialMount.current = false
      return
    }

    // Refetch jobs and functions when namespace or status filter changes
    const refetchData = async () => {
      setLoading(true)
      setJobsOffset(0)
      setHasMoreJobs(true)
      try {
        const [functionsData, jobsData] = await Promise.all([
          jobsApi.listFunctions(selectedNamespace),
          jobsApi.listJobs({
            namespace: selectedNamespace,
            status: statusFilter !== 'all' ? statusFilter : undefined,
            limit: JOBS_PAGE_SIZE,
            offset: 0,
          }),
        ])
        setJobFunctions(functionsData || [])
        setJobs(jobsData || [])
        setJobsOffset(JOBS_PAGE_SIZE)
        setHasMoreJobs((jobsData || []).length >= JOBS_PAGE_SIZE)
      } catch {
        toast.error('Failed to fetch jobs')
      } finally {
        setLoading(false)
      }
    }
    refetchData()
  }, [selectedNamespace, statusFilter])

  const handleSync = async () => {
    setSyncing(true)
    try {
      const result = await jobsApi.sync(selectedNamespace)
      const { created, updated, deleted, errors } = result.summary

      if (errors > 0) {
        toast.error(`Sync completed with ${errors} errors`)
      } else if (created > 0 || updated > 0 || deleted > 0) {
        const messages = []
        if (created > 0) messages.push(`${created} created`)
        if (updated > 0) messages.push(`${updated} updated`)
        if (deleted > 0) messages.push(`${deleted} deleted`)
        toast.success(
          `Jobs synced to "${selectedNamespace}": ${messages.join(', ')}`
        )
      } else {
        toast.info('No changes detected')
      }

      // Refresh namespaces in case new ones were created
      const newNamespaces = await jobsApi.listNamespaces()
      setNamespaces(newNamespaces.length > 0 ? newNamespaces : ['default'])

      await fetchJobFunctions()
    } catch {
      toast.error('Failed to sync jobs from filesystem')
    } finally {
      setSyncing(false)
    }
  }

  const viewJobDetails = async (job: Job) => {
    try {
      const data = await jobsApi.getJob(job.id)
      setSelectedJob(data)
      setLogLevelFilter('all') // Reset filter when opening new job
      setShowJobDetails(true)
    } catch {
      toast.error('Failed to fetch job details')
    }
  }

  const cancelJob = async (jobId: string) => {
    try {
      await jobsApi.cancelJob(jobId)
      toast.success('Job cancelled')
      fetchJobs()
    } catch {
      toast.error('Failed to cancel job')
    }
  }

  const resubmitJob = async (jobId: string) => {
    try {
      const newJob = await jobsApi.resubmitJob(jobId)
      toast.success(`Job resubmitted (new ID: ${newJob.id.slice(0, 8)}...)`)
      fetchJobs()
      // Close the job details dialog if open
      if (showJobDetails) {
        setShowJobDetails(false)
        setSelectedJob(null)
      }
    } catch {
      toast.error('Failed to resubmit job')
    }
  }

  const openRunDialog = (fn: JobFunction) => {
    setSelectedFunction(fn)
    setJobPayload('{\n  \n}')
    setShowRunDialog(true)
  }

  const handleSubmitJob = async () => {
    if (!selectedFunction) return

    setSubmittingJob(true)
    try {
      // Parse payload as JSON
      let payload: Record<string, unknown> = {}
      if (jobPayload.trim()) {
        try {
          payload = JSON.parse(jobPayload)
        } catch {
          toast.error('Invalid JSON payload')
          setSubmittingJob(false)
          return
        }
      }

      // Build config with impersonation token if active
      const { isImpersonating, impersonationToken } =
        useImpersonationStore.getState()
      const config: { headers?: Record<string, string> } = {}
      if (isImpersonating && impersonationToken) {
        config.headers = { 'X-Impersonation-Token': impersonationToken }
      }

      const job = await jobsApi.submitJob(
        {
          job_name: selectedFunction.name,
          namespace: selectedNamespace,
          payload,
        },
        config
      )

      toast.success(`Job submitted successfully (ID: ${job.id.slice(0, 8)}...)`)
      setShowRunDialog(false)
      setSelectedFunction(null)
      setJobPayload('')

      // Switch to queue tab and refresh
      setActiveTab('queue')
      await fetchJobs()
    } catch {
      toast.error('Failed to submit job')
    } finally {
      setSubmittingJob(false)
    }
  }

  const toggleJobEnabled = async (fn: JobFunction) => {
    setTogglingJob(fn.id)
    try {
      await jobsApi.updateFunction(fn.namespace, fn.name, {
        enabled: !fn.enabled,
      })
      toast.success(`Job "${fn.name}" ${fn.enabled ? 'disabled' : 'enabled'}`)
      await fetchJobFunctions()
    } catch {
      toast.error('Failed to update job function')
    } finally {
      setTogglingJob(null)
    }
  }

  // View execution history for a job function
  const viewHistory = async (fn: JobFunction) => {
    setSelectedFunction(fn)
    setHistoryLoading(true)
    setShowHistoryDialog(true)
    try {
      // Fetch jobs that match this job function name
      const jobs = await jobsApi.listJobs({
        namespace: fn.namespace,
        limit: 50,
        offset: 0,
      })
      // Filter to only jobs matching this function name
      const functionJobs = jobs.filter((j) => j.job_name === fn.name)
      setHistoryJobs(functionJobs)
    } catch {
      toast.error('Failed to fetch execution history')
    } finally {
      setHistoryLoading(false)
    }
  }

  // Open edit dialog for job function
  const openEditDialog = async (fn: JobFunction) => {
    setSelectedFunction(fn)
    setFetchingFunction(true)
    setShowEditDialog(true)
    try {
      const fullFunction = await jobsApi.getFunction(fn.namespace, fn.name)
      setEditFormData({
        description: fullFunction.description || '',
        code: fullFunction.code || '',
        timeout_seconds: fullFunction.timeout_seconds,
        max_retries: fullFunction.max_retries,
        schedule: fullFunction.schedule || '',
      })
    } catch {
      toast.error('Failed to load job function details')
      setShowEditDialog(false)
    } finally {
      setFetchingFunction(false)
    }
  }

  // Update job function
  const updateJobFunction = async () => {
    if (!selectedFunction) return
    try {
      await jobsApi.updateFunction(selectedFunction.namespace, selectedFunction.name, {
        description: editFormData.description || undefined,
        code: editFormData.code || undefined,
        timeout_seconds: editFormData.timeout_seconds,
        max_retries: editFormData.max_retries,
        schedule: editFormData.schedule || undefined,
      })
      toast.success('Job function updated')
      setShowEditDialog(false)
      await fetchJobFunctions()
    } catch {
      toast.error('Failed to update job function')
    }
  }

  // Delete job function
  const deleteJobFunction = async () => {
    if (!deleteConfirm) return
    try {
      await jobsApi.deleteFunction(deleteConfirm.namespace, deleteConfirm.name)
      toast.success(`Job function "${deleteConfirm.name}" deleted`)
      setDeleteConfirm(null)
      await fetchJobFunctions()
    } catch {
      toast.error('Failed to delete job function')
    }
  }

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'completed':
        return <CheckCircle className='h-4 w-4 text-green-500' />
      case 'failed':
        return <AlertCircle className='h-4 w-4 text-red-500' />
      case 'running':
        return <Loader2 className='h-4 w-4 animate-spin text-blue-500' />
      case 'pending':
        return <Clock className='h-4 w-4 text-yellow-500' />
      case 'cancelled':
        return <XCircle className='h-4 w-4 text-gray-500' />
      default:
        return <Activity className='h-4 w-4' />
    }
  }

  const getStatusBadgeVariant = (
    status: string
  ): 'default' | 'secondary' | 'destructive' | 'outline' => {
    switch (status) {
      case 'completed':
        return 'default'
      case 'failed':
        return 'destructive'
      case 'running':
        return 'secondary'
      default:
        return 'outline'
    }
  }

  // Format payload/result for display, handling both object and stringified JSON
  const formatJsonValue = (value: unknown): string => {
    if (value === null || value === undefined) {
      return ''
    }
    // If it's a string, try to parse it as JSON for pretty-printing
    if (typeof value === 'string') {
      try {
        const parsed = JSON.parse(value)
        return JSON.stringify(parsed, null, 2)
      } catch {
        // Not valid JSON, return as-is
        return value
      }
    }
    // Already an object, stringify it
    return JSON.stringify(value, null, 2)
  }

  // Log level colors for display (supports both 'warn' and 'warning')
  const LOG_LEVEL_COLORS: Record<ExecutionLogLevel, string> = {
    debug: 'text-gray-400',
    info: 'text-green-400',
    warn: 'text-yellow-400',
    warning: 'text-yellow-400',
    error: 'text-red-400',
    fatal: 'text-red-600 font-bold',
  }

  // Log level badge variants (supports both 'warn' and 'warning')
  const LOG_LEVEL_BADGE_COLORS: Record<ExecutionLogLevel, string> = {
    debug: 'bg-gray-600',
    info: 'bg-green-600',
    warn: 'bg-yellow-600',
    warning: 'bg-yellow-600',
    error: 'bg-red-600',
    fatal: 'bg-red-800',
  }

  // Collapse consecutive duplicate log messages with count prefix
  type CollapsedLog = {
    id: string
    level: ExecutionLogLevel
    message: string
    count: number
  }

  const collapseConsecutiveLogs = (logs: ExecutionLog[]): CollapsedLog[] => {
    if (logs.length === 0) return []

    const result: CollapsedLog[] = []
    let currentLog = logs[0]
    let count = 1

    for (let i = 1; i < logs.length; i++) {
      if (
        logs[i].message === currentLog.message &&
        logs[i].level === currentLog.level
      ) {
        count++
      } else {
        result.push({
          id: `log-${currentLog.id}-${count}`,
          level: currentLog.level || 'info',
          message: currentLog.message,
          count,
        })
        currentLog = logs[i]
        count = 1
      }
    }
    // Push the last group
    result.push({
      id: `log-${currentLog.id}-${count}`,
      level: currentLog.level || 'info',
      message: currentLog.message,
      count,
    })

    return result
  }

  // Priority mapping for log levels (includes both 'warn' and 'warning' for compatibility)
  const LOG_LEVEL_PRIORITY_MAP: Record<ExecutionLogLevel, number> = {
    debug: 0,
    info: 1,
    warn: 2,
    warning: 2,
    error: 3,
    fatal: 4,
  }

  // Filter logs by level
  const filterLogsByLevel = (logs: ExecutionLog[]): ExecutionLog[] => {
    if (logLevelFilter === 'all') return logs
    // Map 'warning' to 'warn' for comparison
    const filterLevel = logLevelFilter === 'warning' ? 'warn' : logLevelFilter
    const minPriority =
      LOG_LEVEL_PRIORITY_MAP[filterLevel as ExecutionLogLevel] ?? 0
    return logs.filter(
      (log) => LOG_LEVEL_PRIORITY_MAP[log.level || 'info'] >= minPriority
    )
  }

  // Format logs for clipboard (plain text)
  const formatLogsForClipboard = (logs: ExecutionLog[]): string => {
    return collapseConsecutiveLogs(logs)
      .map((log) => {
        const prefix = log.count > 1 ? `(${log.count}x) ` : ''
        return `[${log.level.toUpperCase()}] ${prefix}${log.message}`
      })
      .join('\n')
  }

  // Copy text to clipboard with toast feedback
  const copyToClipboard = async (text: string, label: string) => {
    try {
      await navigator.clipboard.writeText(text)
      toast.success(`${label} copied to clipboard`)
    } catch {
      toast.error('Failed to copy to clipboard')
    }
  }

  // Copy all job details to clipboard
  const copyAllJobDetails = () => {
    if (!selectedJob) return

    const parts: string[] = []

    parts.push(`=== Job Details ===`)
    parts.push(`Job: ${selectedJob.job_name}`)
    parts.push(`ID: ${selectedJob.id}`)
    parts.push(`Status: ${selectedJob.status}`)
    parts.push(`Created: ${new Date(selectedJob.created_at).toLocaleString()}`)
    if (selectedJob.started_at) {
      parts.push(
        `Started: ${new Date(selectedJob.started_at).toLocaleString()}`
      )
    }
    if (selectedJob.completed_at) {
      parts.push(
        `Completed: ${new Date(selectedJob.completed_at).toLocaleString()}`
      )
    }
    parts.push('')

    if (selectedJob.payload !== undefined && selectedJob.payload !== null) {
      parts.push(`=== Payload ===`)
      parts.push(formatJsonValue(selectedJob.payload))
      parts.push('')
    }

    if (executionLogs.length > 0) {
      parts.push(`=== Logs ===`)
      parts.push(collapseConsecutiveLogs(executionLogs).join('\n'))
      parts.push('')
    }

    if (selectedJob.result !== undefined && selectedJob.result !== null) {
      parts.push(`=== Result ===`)
      parts.push(formatJsonValue(selectedJob.result))
      parts.push('')
    }

    if (selectedJob.error_message) {
      parts.push(`=== Error ===`)
      parts.push(selectedJob.error_message)
    }

    copyToClipboard(parts.join('\n'), 'All job details')
  }

  // Memoize filtered and collapsed logs to prevent re-renders
  const filteredAndCollapsedLogs = useMemo(() => {
    const filtered = filterLogsByLevel(executionLogs)
    return collapseConsecutiveLogs(filtered)
    // filterLogsByLevel and collapseConsecutiveLogs are stable functions defined in component
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [executionLogs, logLevelFilter])

  // Filter jobs from past 24 hours (for stats display only)
  const jobs24h = useMemo(() => {
    const cutoff = Date.now() - 24 * 60 * 60 * 1000
    return jobs.filter((job) => {
      const jobTime = new Date(job.created_at).getTime()
      return jobTime >= cutoff
    })
  }, [jobs])

  // Stats based on past 24 hours
  const filteredStats = useMemo(
    () => ({
      pending: jobs24h.filter((j) => j.status === 'pending').length,
      running: jobs24h.filter((j) => j.status === 'running').length,
      completed: jobs24h.filter((j) => j.status === 'completed').length,
      failed: jobs24h.filter((j) => j.status === 'failed').length,
      cancelled: jobs24h.filter((j) => j.status === 'cancelled').length,
      total: jobs24h.length,
    }),
    [jobs24h]
  )

  // Filter by search query only (no time filter for display)
  const filteredJobs = jobs.filter((job) => {
    if (
      searchQuery &&
      !job.job_name.toLowerCase().includes(searchQuery.toLowerCase())
    ) {
      return false
    }
    return true
  })

  if (loading) {
    return (
      <div className='flex h-96 items-center justify-center'>
        <RefreshCw className='text-muted-foreground h-8 w-8 animate-spin' />
      </div>
    )
  }

  return (
    <div className='flex flex-1 flex-col gap-6 p-6'>
      <ImpersonationBanner />

      <div className='flex items-center justify-between'>
        <div>
          <h1 className='text-3xl font-bold'>Background Jobs</h1>
          <p className='text-muted-foreground'>
            Manage job functions and monitor background task execution
          </p>
        </div>
        <div className='flex items-center gap-2'>
          <ImpersonationPopover
            contextLabel='Running as'
            defaultReason='Testing job submission'
          />
          <Button onClick={refreshAllData} variant='outline' size='sm'>
            <RefreshCw className='mr-2 h-4 w-4' />
            Refresh
          </Button>
        </div>
      </div>

      {/* Stats (Past 24 hours) */}
      <Card className='!gap-0 !py-0'>
        <CardContent className='px-4 py-2'>
          <div className='flex items-center gap-4'>
            <span className='text-muted-foreground text-xs'>
              (Past 24 hours)
            </span>
            <div className='flex items-center gap-1'>
              <span className='text-muted-foreground text-xs'>Pending:</span>
              <span className='text-sm font-semibold'>
                {filteredStats.pending}
              </span>
            </div>
            <div className='flex items-center gap-1'>
              <span className='text-muted-foreground text-xs'>Running:</span>
              <span className='text-sm font-semibold'>
                {filteredStats.running}
              </span>
            </div>
            <div className='flex items-center gap-1'>
              <span className='text-muted-foreground text-xs'>Completed:</span>
              <span className='text-sm font-semibold'>
                {filteredStats.completed}
              </span>
            </div>
            <div className='flex items-center gap-1'>
              <span className='text-muted-foreground text-xs'>Failed:</span>
              <span className='text-sm font-semibold'>
                {filteredStats.failed}
              </span>
            </div>
            <div className='flex items-center gap-1'>
              <span className='text-muted-foreground text-xs'>Workers:</span>
              <span className='text-sm font-semibold'>
                {workers.filter((w) => w.status === 'active').length}
              </span>
            </div>
            <div className='flex items-center gap-1'>
              <Target className='text-muted-foreground h-3 w-3' />
              <span className='text-muted-foreground text-xs'>Success:</span>
              {(() => {
                const total = filteredStats.completed + filteredStats.failed
                const successRate =
                  total > 0
                    ? ((filteredStats.completed / total) * 100).toFixed(0)
                    : '0'
                return (
                  <span className='text-sm font-semibold'>{successRate}%</span>
                )
              })()}
            </div>
            <div className='flex items-center gap-1'>
              <Timer className='text-muted-foreground h-3 w-3' />
              <span className='text-muted-foreground text-xs'>Avg. Wait:</span>
              {(() => {
                const pendingJobs = jobs24h.filter(
                  (j) => j.status === 'pending'
                )
                const waitTimes = pendingJobs.map(
                  (j) => Date.now() - new Date(j.created_at).getTime()
                )
                const avgWaitMs =
                  waitTimes.length > 0
                    ? waitTimes.reduce((a, b) => a + b, 0) / waitTimes.length
                    : 0
                const avgWaitSec = Math.round(avgWaitMs / 1000)
                const displayTime =
                  avgWaitSec < 60
                    ? `${avgWaitSec}s`
                    : avgWaitSec < 3600
                      ? `${Math.round(avgWaitSec / 60)}m`
                      : `${Math.round(avgWaitSec / 3600)}h`
                return (
                  <span className='text-sm font-semibold'>{displayTime}</span>
                )
              })()}
            </div>
          </div>
        </CardContent>
      </Card>

      <Tabs
        value={activeTab}
        onValueChange={(v) => setActiveTab(v as 'functions' | 'queue')}
        className='flex min-h-0 flex-1 flex-col'
      >
        <TabsList className='grid w-full max-w-md grid-cols-2'>
          <TabsTrigger value='queue'>
            <Activity className='mr-2 h-4 w-4' />
            Job Queue
          </TabsTrigger>
          <TabsTrigger value='functions'>
            <ListTodo className='mr-2 h-4 w-4' />
            Job Functions
          </TabsTrigger>
        </TabsList>

        <TabsContent
          value='queue'
          className='mt-6 flex min-h-0 flex-1 flex-col space-y-6'
        >
          {/* Filters */}
          <div className='flex items-center gap-3'>
            <div className='flex items-center gap-2'>
              <Label
                htmlFor='queue-namespace-select'
                className='text-muted-foreground text-sm whitespace-nowrap'
              >
                Namespace:
              </Label>
              <Select
                value={selectedNamespace}
                onValueChange={setSelectedNamespace}
              >
                <SelectTrigger
                  id='queue-namespace-select'
                  className='w-[180px]'
                >
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
            <div className='relative flex-1'>
              <Search className='text-muted-foreground absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2' />
              <Input
                placeholder='Search jobs...'
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className='pl-9'
              />
            </div>
            <Select
              value={statusFilter}
              onValueChange={(v) => {
                setStatusFilter(v)
                setJobsOffset(0)
                setHasMoreJobs(true)
                setTimeout(() => fetchJobs(true), 100)
              }}
            >
              <SelectTrigger className='w-[180px]'>
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
              </SelectContent>
            </Select>
          </div>

          {/* Jobs List */}
          <ScrollArea className='min-h-0 flex-1'>
            <div className='grid gap-4'>
              {filteredJobs.length === 0 ? (
                <Card>
                  <CardContent className='p-12 text-center'>
                    <ListTodo className='text-muted-foreground mx-auto mb-4 h-12 w-12' />
                    <p className='mb-2 text-lg font-medium'>No jobs found</p>
                    <p className='text-muted-foreground text-sm'>
                      {searchQuery || statusFilter !== 'all'
                        ? 'Try adjusting your filters'
                        : 'Submit a job to see it here'}
                    </p>
                  </CardContent>
                </Card>
              ) : (
                filteredJobs.map((job) => (
                  <div
                    key={job.id}
                    className='hover:border-primary/50 bg-card flex items-center justify-between gap-2 rounded-md border px-3 py-1.5 transition-colors'
                  >
                    <div className='flex min-w-0 flex-1 items-center gap-2'>
                      {getStatusIcon(job.status)}
                      <span className='truncate text-sm font-medium'>
                        {job.job_name}
                      </span>
                      <Badge
                        variant={getStatusBadgeVariant(job.status)}
                        className='h-4 shrink-0 px-1 py-0 text-[10px]'
                      >
                        {job.status}
                      </Badge>
                      {job.user_email && (
                        <span
                          className='text-muted-foreground max-w-[120px] shrink-0 truncate text-[10px]'
                          title={
                            job.user_name
                              ? `${job.user_name} (${job.user_email})`
                              : job.user_email
                          }
                        >
                          {job.user_email}
                        </span>
                      )}
                      {job.retry_count > 0 && (
                        <span className='text-muted-foreground shrink-0 text-[10px]'>
                          #{job.retry_count}
                        </span>
                      )}
                      {(job.status === 'running' || job.status === 'pending') &&
                        job.progress_percent !== undefined && (
                          <div className='flex shrink-0 items-center gap-1'>
                            <div className='bg-secondary h-1 w-16 overflow-hidden rounded-full'>
                              <div
                                className='h-full bg-blue-500 transition-all duration-300'
                                style={{ width: `${job.progress_percent}%` }}
                              />
                            </div>
                            <span className='text-muted-foreground text-[10px]'>
                              {job.progress_percent}%
                            </span>
                            {job.estimated_seconds_left !== undefined &&
                              job.estimated_seconds_left > 0 && (
                                <span className='text-muted-foreground text-[10px]'>
                                  (ETA:{' '}
                                  {job.estimated_seconds_left < 60
                                    ? `${job.estimated_seconds_left}s`
                                    : job.estimated_seconds_left < 3600
                                      ? `${Math.round(job.estimated_seconds_left / 60)}m`
                                      : `${Math.round(job.estimated_seconds_left / 3600)}h`}
                                  )
                                </span>
                              )}
                          </div>
                        )}
                    </div>
                    <div className='flex shrink-0 items-center gap-1'>
                      <span className='text-muted-foreground text-[10px]'>
                        {new Date(job.created_at).toLocaleTimeString()}
                      </span>
                      <Button
                        onClick={() => viewJobDetails(job)}
                        size='sm'
                        variant='ghost'
                        className='h-6 px-1.5 text-xs'
                      >
                        View
                      </Button>
                      {(job.status === 'running' ||
                        job.status === 'pending') && (
                        <Button
                          onClick={() => cancelJob(job.id)}
                          size='sm'
                          variant='ghost'
                          className='h-6 w-6 p-0'
                        >
                          <XCircle className='h-3 w-3' />
                        </Button>
                      )}
                      {(job.status === 'completed' ||
                        job.status === 'cancelled' ||
                        job.status === 'failed') && (
                        <Button
                          onClick={() => resubmitJob(job.id)}
                          size='sm'
                          variant='ghost'
                          className='h-6 w-6 p-0'
                          title='Re-submit as new job'
                        >
                          <RefreshCw className='h-3 w-3' />
                        </Button>
                      )}
                    </div>
                  </div>
                ))
              )}

              {/* Load More Button */}
              {hasMoreJobs && filteredJobs.length > 0 && (
                <div className='flex justify-center py-4'>
                  <Button
                    onClick={loadMoreJobs}
                    variant='outline'
                    disabled={loadingMore}
                  >
                    {loadingMore ? (
                      <>
                        <Loader2 className='mr-2 h-4 w-4 animate-spin' />
                        Loading...
                      </>
                    ) : (
                      <>
                        <ChevronDown className='mr-2 h-4 w-4' />
                        Load More Jobs
                      </>
                    )}
                  </Button>
                </div>
              )}
            </div>
          </ScrollArea>
        </TabsContent>

        <TabsContent
          value='functions'
          className='mt-6 flex min-h-0 flex-1 flex-col space-y-6'
        >
          {/* Namespace Selector and Sync */}
          <div className='flex items-center justify-between'>
            <div className='flex items-center gap-2'>
              <Label
                htmlFor='namespace-select'
                className='text-muted-foreground text-sm whitespace-nowrap'
              >
                Namespace:
              </Label>
              <Select
                value={selectedNamespace}
                onValueChange={setSelectedNamespace}
              >
                <SelectTrigger id='namespace-select' className='w-[180px]'>
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
          </div>

          <div className='grid gap-4 md:grid-cols-3'>
            <Card className='!gap-0'>
              <CardContent className='px-4 py-4'>
                <div className='text-muted-foreground mb-1 text-xs'>
                  Total Functions
                </div>
                <div className='text-2xl font-bold'>{jobFunctions.length}</div>
              </CardContent>
            </Card>
            <Card className='!gap-0'>
              <CardContent className='px-4 py-4'>
                <div className='text-muted-foreground mb-1 text-xs'>
                  Enabled
                </div>
                <div className='text-2xl font-bold'>
                  {jobFunctions.filter((f) => f.enabled).length}
                </div>
              </CardContent>
            </Card>
            <Card className='!gap-0'>
              <CardContent className='px-4 py-4'>
                <div className='text-muted-foreground mb-1 text-xs'>
                  Scheduled
                </div>
                <div className='text-2xl font-bold'>
                  {jobFunctions.filter((f) => f.schedule).length}
                </div>
              </CardContent>
            </Card>
          </div>

          <ScrollArea className='min-h-0 flex-1'>
            <div className='grid gap-4'>
              {jobFunctions.length === 0 ? (
                <Card>
                  <CardContent className='p-12 text-center'>
                    <ListTodo className='text-muted-foreground mx-auto mb-4 h-12 w-12' />
                    <p className='mb-2 text-lg font-medium'>
                      No job functions yet
                    </p>
                    <p className='text-muted-foreground text-sm'>
                      Place job function files in your jobs directory and sync
                    </p>
                  </CardContent>
                </Card>
              ) : (
                jobFunctions.map((fn) => (
                  <div
                    key={fn.id}
                    className='hover:border-primary/50 bg-card flex items-center justify-between gap-2 rounded-md border px-3 py-1.5 transition-colors'
                  >
                    <div className='flex min-w-0 flex-1 items-center gap-2'>
                      <span className='truncate text-sm font-medium'>
                        {fn.name}
                      </span>
                      <Badge
                        variant='outline'
                        className='h-4 shrink-0 px-1 py-0 text-[10px]'
                      >
                        v{fn.version}
                      </Badge>
                      {fn.schedule && (
                        <Badge
                          variant='outline'
                          className='h-4 shrink-0 px-1 py-0 text-[10px]'
                        >
                          <Clock className='mr-0.5 h-2.5 w-2.5' />
                          {fn.schedule}
                        </Badge>
                      )}
                      <Switch
                        id={`enable-${fn.id}`}
                        checked={fn.enabled}
                        disabled={togglingJob === fn.id}
                        onCheckedChange={() => toggleJobEnabled(fn)}
                        className='scale-75'
                      />
                    </div>
                    <div className='flex shrink-0 items-center gap-0.5'>
                      {fn.source === 'filesystem' && fn.updated_at && (
                        <span
                          className='text-muted-foreground text-[10px] mr-1'
                          title={`Last synced: ${new Date(fn.updated_at).toLocaleString()}`}
                        >
                          synced {new Date(fn.updated_at).toLocaleDateString()}
                        </span>
                      )}
                      <span className='text-muted-foreground text-[10px] mr-1'>
                        {fn.timeout_seconds}s / {fn.max_retries}r
                      </span>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <Button
                            onClick={() => viewHistory(fn)}
                            variant='ghost'
                            size='sm'
                            className='h-6 w-6 p-0'
                          >
                            <History className='h-3 w-3' />
                          </Button>
                        </TooltipTrigger>
                        <TooltipContent>View history</TooltipContent>
                      </Tooltip>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <Button
                            onClick={() => openRunDialog(fn)}
                            size='sm'
                            variant='ghost'
                            className='h-6 w-6 p-0'
                            disabled={!fn.enabled}
                          >
                            <Play className='h-3 w-3' />
                          </Button>
                        </TooltipTrigger>
                        <TooltipContent>Run job</TooltipContent>
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
                        <TooltipContent>Edit job function</TooltipContent>
                      </Tooltip>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <Button
                            onClick={() => setDeleteConfirm({ namespace: fn.namespace, name: fn.name })}
                            size='sm'
                            variant='ghost'
                            className='h-6 w-6 p-0 text-destructive hover:text-destructive hover:bg-destructive/10'
                          >
                            <Trash2 className='h-3 w-3' />
                          </Button>
                        </TooltipTrigger>
                        <TooltipContent>Delete job function</TooltipContent>
                      </Tooltip>
                    </div>
                  </div>
                ))
              )}
            </div>
          </ScrollArea>
        </TabsContent>
      </Tabs>

      {/* Job Details Dialog */}
      <Dialog open={showJobDetails} onOpenChange={setShowJobDetails}>
        <DialogContent className='max-h-[90vh] w-[90vw] max-w-[1600px] overflow-y-auto sm:max-w-none'>
          <DialogHeader className='flex flex-row items-start justify-between'>
            <div>
              <DialogTitle className='flex items-center gap-2'>
                {selectedJob && getStatusIcon(selectedJob.status)}
                Job Details
              </DialogTitle>
              <DialogDescription>
                {selectedJob?.job_name} - {selectedJob?.id}
              </DialogDescription>
            </div>
            <Button
              variant='outline'
              size='sm'
              onClick={copyAllJobDetails}
              className='shrink-0'
            >
              <Copy className='mr-2 h-4 w-4' />
              Copy All
            </Button>
          </DialogHeader>

          {selectedJob && (
            <div className='space-y-4'>
              <div className='flex flex-wrap gap-2'>
                <Badge variant={getStatusBadgeVariant(selectedJob.status)}>
                  {selectedJob.status}
                </Badge>
                {selectedJob.user_email && (
                  <Badge
                    variant='outline'
                    title={
                      selectedJob.user_name
                        ? `${selectedJob.user_name} (${selectedJob.user_email})`
                        : selectedJob.user_email
                    }
                  >
                    {selectedJob.user_email}
                  </Badge>
                )}
                {selectedJob.user_role && (
                  <Badge variant='outline'>role: {selectedJob.user_role}</Badge>
                )}
              </div>

              <Separator />

              <div className='grid gap-3'>
                <div>
                  <Label className='text-muted-foreground text-xs'>
                    Created
                  </Label>
                  <p className='text-sm'>
                    {new Date(selectedJob.created_at).toLocaleString()}
                  </p>
                </div>
                {selectedJob.started_at && (
                  <div>
                    <Label className='text-muted-foreground text-xs'>
                      Started
                    </Label>
                    <p className='text-sm'>
                      {new Date(selectedJob.started_at).toLocaleString()}
                    </p>
                  </div>
                )}
                {selectedJob.completed_at && (
                  <div>
                    <Label className='text-muted-foreground text-xs'>
                      Completed
                    </Label>
                    <p className='text-sm'>
                      {new Date(selectedJob.completed_at).toLocaleString()}
                    </p>
                  </div>
                )}
                {selectedJob.progress_percent !== undefined && (
                  <div className='space-y-2'>
                    <Label className='text-muted-foreground text-xs'>
                      Progress
                    </Label>
                    <div className='space-y-1'>
                      <div className='flex items-center justify-between text-sm'>
                        <span className='font-medium'>
                          {selectedJob.progress_percent}%
                        </span>
                        {selectedJob.estimated_seconds_left !== undefined &&
                          selectedJob.estimated_seconds_left > 0 && (
                            <span className='text-muted-foreground'>
                              ~
                              {selectedJob.estimated_seconds_left < 60
                                ? `${selectedJob.estimated_seconds_left}s`
                                : selectedJob.estimated_seconds_left < 3600
                                  ? `${Math.round(selectedJob.estimated_seconds_left / 60)}m`
                                  : `${Math.round(selectedJob.estimated_seconds_left / 3600)}h`}{' '}
                              remaining
                            </span>
                          )}
                      </div>
                      <div className='bg-secondary h-3 w-full overflow-hidden rounded-full'>
                        <div
                          className={`h-full transition-all duration-300 ${
                            selectedJob.status === 'running'
                              ? 'bg-blue-500'
                              : selectedJob.status === 'completed'
                                ? 'bg-green-500'
                                : selectedJob.status === 'failed'
                                  ? 'bg-red-500'
                                  : 'bg-primary'
                          }`}
                          style={{ width: `${selectedJob.progress_percent}%` }}
                        />
                      </div>
                      {selectedJob.progress_message && (
                        <p className='text-muted-foreground text-sm'>
                          {selectedJob.progress_message}
                        </p>
                      )}
                      {selectedJob.last_progress_at && (
                        <p className='text-muted-foreground text-xs'>
                          Last updated:{' '}
                          {new Date(
                            selectedJob.last_progress_at
                          ).toLocaleString()}
                        </p>
                      )}
                    </div>
                  </div>
                )}
              </div>

              <Separator />

              {selectedJob.payload !== undefined &&
                selectedJob.payload !== null && (
                  <div>
                    <div className='mb-2 flex items-center justify-between'>
                      <Label>Payload</Label>
                      <Button
                        variant='ghost'
                        size='sm'
                        className='h-6 px-2'
                        onClick={() =>
                          copyToClipboard(
                            formatJsonValue(selectedJob.payload),
                            'Payload'
                          )
                        }
                      >
                        <Copy className='h-3 w-3' />
                      </Button>
                    </div>
                    <div className='bg-muted max-h-48 overflow-auto rounded-lg border p-4'>
                      <pre className='text-xs break-all whitespace-pre-wrap'>
                        {formatJsonValue(selectedJob.payload)}
                      </pre>
                    </div>
                  </div>
                )}

              {/* Logs and Result/Error side by side */}
              <div className='grid grid-cols-1 gap-4 lg:grid-cols-2'>
                {/* Logs Column - Always show for consistent layout */}
                <div className='flex flex-col'>
                  <div className='mb-2 flex items-center justify-between'>
                    <Label>Logs</Label>
                    <div className='flex items-center gap-2'>
                      <Select
                        value={logLevelFilter}
                        onValueChange={(value) =>
                          setLogLevelFilter(value as LogLevel | 'all')
                        }
                      >
                        <SelectTrigger className='h-6 w-24 text-xs'>
                          <SelectValue placeholder='Level' />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value='all'>All</SelectItem>
                          <SelectItem value='debug'>Debug</SelectItem>
                          <SelectItem value='info'>Info</SelectItem>
                          <SelectItem value='warning'>Warning</SelectItem>
                          <SelectItem value='error'>Error</SelectItem>
                          <SelectItem value='fatal'>Fatal</SelectItem>
                        </SelectContent>
                      </Select>
                      {executionLogs.length > 0 && (
                        <Button
                          variant='ghost'
                          size='sm'
                          className='h-6 px-2'
                          onClick={() =>
                            copyToClipboard(
                              formatLogsForClipboard(
                                filterLogsByLevel(executionLogs)
                              ),
                              'Logs'
                            )
                          }
                        >
                          <Copy className='h-3 w-3' />
                        </Button>
                      )}
                    </div>
                  </div>
                  <div
                    ref={logsContainerRef}
                    className='h-[400px] overflow-y-auto rounded-lg border bg-black/90 p-4 font-mono'
                    onScroll={() => {
                      isAtBottomRef.current = checkIfAtBottom()
                    }}
                  >
                    {loadingLogs ? (
                      <span className='text-muted-foreground text-xs italic'>
                        Loading logs...
                      </span>
                    ) : executionLogs.length > 0 ? (
                      <div className='flex flex-col gap-0.5'>
                        {filteredAndCollapsedLogs.map((log) => (
                          <div
                            key={log.id}
                            className='flex items-start gap-2 text-xs'
                          >
                            <span
                              className={`w-12 shrink-0 rounded px-1 py-0.5 text-center text-[10px] font-medium text-white uppercase ${LOG_LEVEL_BADGE_COLORS[log.level]}`}
                            >
                              {log.level}
                            </span>
                            <span
                              className={`break-words ${LOG_LEVEL_COLORS[log.level]}`}
                            >
                              {log.count > 1 && (
                                <span className='text-gray-500'>
                                  ({log.count}x){' '}
                                </span>
                              )}
                              {log.message}
                            </span>
                          </div>
                        ))}
                      </div>
                    ) : (
                      <span className='text-muted-foreground text-xs italic'>
                        No logs available
                      </span>
                    )}
                  </div>
                </div>

                {/* Result/Error Column */}
                {(selectedJob.result !== undefined &&
                  selectedJob.result !== null) ||
                selectedJob.error_message ? (
                  <div className='flex flex-col gap-4'>
                    {selectedJob.result !== undefined &&
                      selectedJob.result !== null && (
                        <div className='flex flex-1 flex-col'>
                          <div className='mb-2 flex items-center justify-between'>
                            <Label>Result</Label>
                            <Button
                              variant='ghost'
                              size='sm'
                              className='h-6 px-2'
                              onClick={() =>
                                copyToClipboard(
                                  formatJsonValue(selectedJob.result),
                                  'Result'
                                )
                              }
                            >
                              <Copy className='h-3 w-3' />
                            </Button>
                          </div>
                          <div className='bg-muted max-h-[200px] min-h-[100px] flex-1 overflow-auto rounded-lg border p-4'>
                            <pre className='text-xs break-all whitespace-pre-wrap'>
                              {formatJsonValue(selectedJob.result)}
                            </pre>
                          </div>
                        </div>
                      )}

                    {selectedJob.error_message && (
                      <div className='flex flex-1 flex-col'>
                        <div className='mb-2 flex items-center justify-between'>
                          <Label className='text-destructive'>Error</Label>
                          <Button
                            variant='ghost'
                            size='sm'
                            className='h-6 px-2'
                            onClick={() =>
                              copyToClipboard(
                                selectedJob.error_message || '',
                                'Error'
                              )
                            }
                          >
                            <Copy className='h-3 w-3' />
                          </Button>
                        </div>
                        <div className='bg-destructive/10 border-destructive/20 max-h-[200px] min-h-[100px] flex-1 overflow-auto rounded-lg border p-4'>
                          <pre className='text-destructive text-xs break-all whitespace-pre-wrap'>
                            {selectedJob.error_message}
                          </pre>
                        </div>
                      </div>
                    )}
                  </div>
                ) : null}
              </div>
            </div>
          )}

          <DialogFooter className='flex gap-2'>
            {selectedJob &&
              (selectedJob.status === 'pending' ||
                selectedJob.status === 'running') && (
                <Button
                  variant='destructive'
                  onClick={() => {
                    cancelJob(selectedJob.id)
                    setShowJobDetails(false)
                  }}
                >
                  <XCircle className='mr-2 h-4 w-4' />
                  Cancel Job
                </Button>
              )}
            {selectedJob &&
              (selectedJob.status === 'completed' ||
                selectedJob.status === 'cancelled' ||
                selectedJob.status === 'failed') && (
                <Button
                  variant='secondary'
                  onClick={() => resubmitJob(selectedJob.id)}
                >
                  <RefreshCw className='mr-2 h-4 w-4' />
                  Re-submit
                </Button>
              )}
            <Button variant='outline' onClick={() => setShowJobDetails(false)}>
              Close
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Run Job Dialog */}
      <Dialog open={showRunDialog} onOpenChange={setShowRunDialog}>
        <DialogContent className='max-w-lg'>
          <DialogHeader>
            <DialogTitle className='flex items-center gap-2'>
              <Play className='h-5 w-5' />
              Run Job
            </DialogTitle>
            <DialogDescription>
              Submit a new job for "{selectedFunction?.name}" in the "
              {selectedNamespace}" namespace
            </DialogDescription>
          </DialogHeader>

          <div className='space-y-4'>
            {selectedFunction && (
              <div className='bg-muted/50 rounded-lg border p-3'>
                <div className='mb-2 flex items-center gap-2'>
                  <span className='font-medium'>{selectedFunction.name}</span>
                  <Badge variant='outline'>v{selectedFunction.version}</Badge>
                </div>
                <p className='text-muted-foreground text-sm'>
                  {selectedFunction.description || 'No description'}
                </p>
                <div className='text-muted-foreground mt-2 flex items-center gap-4 text-xs'>
                  <span>Timeout: {selectedFunction.timeout_seconds}s</span>
                  <span>Max retries: {selectedFunction.max_retries}</span>
                </div>
              </div>
            )}

            <div className='space-y-2'>
              <Label htmlFor='job-payload'>Payload (JSON)</Label>
              <Textarea
                id='job-payload'
                value={jobPayload}
                onChange={(e) => setJobPayload(e.target.value)}
                placeholder='{\n  "key": "value"\n}'
                className='min-h-[150px] font-mono text-sm'
              />
              <p className='text-muted-foreground text-xs'>
                Enter the JSON payload to pass to the job's handler function.
                This will be available as{' '}
                <code className='bg-muted rounded px-1'>request.payload</code>{' '}
                in your job code.
              </p>
            </div>
          </div>

          <DialogFooter>
            <Button
              variant='outline'
              onClick={() => {
                setShowRunDialog(false)
                setSelectedFunction(null)
                setJobPayload('')
              }}
            >
              Cancel
            </Button>
            <Button onClick={handleSubmitJob} disabled={submittingJob}>
              {submittingJob ? (
                <>
                  <Loader2 className='mr-2 h-4 w-4 animate-spin' />
                  Submitting...
                </>
              ) : (
                <>
                  <Play className='mr-2 h-4 w-4' />
                  Run Job
                </>
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Job Function Dialog */}
      <Dialog open={showEditDialog} onOpenChange={setShowEditDialog}>
        <DialogContent className='max-h-[90vh] max-w-4xl overflow-y-auto'>
          <DialogHeader>
            <DialogTitle>Edit Job Function</DialogTitle>
            <DialogDescription>
              Update job function code and settings for "{selectedFunction?.name}"
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
                value={editFormData.description}
                onChange={(e) =>
                  setEditFormData({ ...editFormData, description: e.target.value })
                }
              />
            </div>

            <div>
              <Label htmlFor='edit-code'>Code</Label>
              <Textarea
                id='edit-code'
                className='min-h-[400px] font-mono text-sm'
                value={editFormData.code}
                onChange={(e) =>
                  setEditFormData({ ...editFormData, code: e.target.value })
                }
              />
            </div>

            <div className='grid grid-cols-3 gap-4'>
              <div>
                <Label htmlFor='edit-timeout'>Timeout (seconds)</Label>
                <Input
                  id='edit-timeout'
                  type='number'
                  min={1}
                  max={3600}
                  value={editFormData.timeout_seconds}
                  onChange={(e) =>
                    setEditFormData({
                      ...editFormData,
                      timeout_seconds: parseInt(e.target.value),
                    })
                  }
                />
              </div>

              <div>
                <Label htmlFor='edit-retries'>Max Retries</Label>
                <Input
                  id='edit-retries'
                  type='number'
                  min={0}
                  max={10}
                  value={editFormData.max_retries}
                  onChange={(e) =>
                    setEditFormData({
                      ...editFormData,
                      max_retries: parseInt(e.target.value),
                    })
                  }
                />
              </div>

              <div>
                <Label htmlFor='edit-schedule'>Schedule (cron)</Label>
                <Input
                  id='edit-schedule'
                  placeholder='0 0 * * *'
                  value={editFormData.schedule}
                  onChange={(e) =>
                    setEditFormData({ ...editFormData, schedule: e.target.value })
                  }
                />
              </div>
            </div>
          </div>
          )}

          <DialogFooter>
            <Button variant='outline' onClick={() => setShowEditDialog(false)}>
              Cancel
            </Button>
            <Button onClick={updateJobFunction} disabled={fetchingFunction}>
              Update Job Function
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <AlertDialog open={deleteConfirm !== null} onOpenChange={(open) => !open && setDeleteConfirm(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Job Function</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete "{deleteConfirm?.name}"? This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={deleteJobFunction}
              className='bg-destructive text-destructive-foreground hover:bg-destructive/90'
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Execution History Dialog */}
      <Dialog open={showHistoryDialog} onOpenChange={setShowHistoryDialog}>
        <DialogContent className='max-h-[90vh] max-w-4xl overflow-y-auto'>
          <DialogHeader>
            <DialogTitle className='flex items-center gap-2'>
              <History className='h-5 w-5' />
              Execution History
            </DialogTitle>
            <DialogDescription>
              Recent executions for "{selectedFunction?.name}"
            </DialogDescription>
          </DialogHeader>

          {historyLoading ? (
            <div className='flex items-center justify-center py-12'>
              <Loader2 className='h-8 w-8 animate-spin text-muted-foreground' />
            </div>
          ) : historyJobs.length === 0 ? (
            <div className='text-center py-12'>
              <Activity className='mx-auto h-12 w-12 text-muted-foreground mb-4' />
              <p className='text-muted-foreground'>No executions found</p>
            </div>
          ) : (
            <ScrollArea className='h-[400px]'>
              <div className='space-y-2'>
                {historyJobs.map((job) => (
                  <div
                    key={job.id}
                    className='flex items-center justify-between rounded-lg border p-3 hover:bg-muted/50 cursor-pointer'
                    onClick={() => {
                      setShowHistoryDialog(false)
                      viewJobDetails(job)
                    }}
                  >
                    <div className='flex items-center gap-3'>
                      {getStatusIcon(job.status)}
                      <div>
                        <div className='flex items-center gap-2'>
                          <span className='text-sm font-medium'>{job.id.slice(0, 8)}...</span>
                          <Badge variant={getStatusBadgeVariant(job.status)}>{job.status}</Badge>
                        </div>
                        <span className='text-xs text-muted-foreground'>
                          {new Date(job.created_at).toLocaleString()}
                        </span>
                      </div>
                    </div>
                    <div className='text-right'>
                      {job.started_at && job.completed_at && (
                        <span className='text-xs text-muted-foreground'>
                          {new Date(job.completed_at).getTime() - new Date(job.started_at).getTime()}ms
                        </span>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            </ScrollArea>
          )}

          <DialogFooter>
            <Button variant='outline' onClick={() => setShowHistoryDialog(false)}>
              Close
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
