import { useState, useEffect, useCallback, useRef } from 'react'
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
  TrendingUp,
  Timer,
  Target,
  ChevronDown,
  ChevronUp,
  Play,
  Copy,
} from 'lucide-react'
import { toast } from 'sonner'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
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
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Switch } from '@/components/ui/switch'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { ImpersonationBanner } from '@/components/impersonation-banner'
import { ImpersonationSelector } from '@/features/impersonation/components/impersonation-selector'
import {
  jobsApi,
  type JobFunction,
  type Job,
  type JobWorker,
  type ExecutionLog,
} from '@/lib/api'
import { fluxbaseClient } from '@/lib/fluxbase-client'
import {
  PieChart,
  Pie,
  Cell,
  ResponsiveContainer,
  Tooltip,
  Legend,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
} from 'recharts'

export const Route = createFileRoute('/_authenticated/jobs/')({
  component: JobsPage,
})

const JOBS_PAGE_SIZE = 25

function JobsPage() {
  const [activeTab, setActiveTab] = useState<'functions' | 'queue'>('queue')
  const [jobFunctions, setJobFunctions] = useState<JobFunction[]>([])
  const [jobs, setJobs] = useState<Job[]>([])
  const [workers, setWorkers] = useState<JobWorker[]>([])
  const [loading, setLoading] = useState(true)
  const [searchQuery, setSearchQuery] = useState('')
  const [statusFilter, setStatusFilter] = useState<string>('all')
  const [timeRange, setTimeRange] = useState<string>('1h')
  const [selectedJob, setSelectedJob] = useState<Job | null>(null)
  const [showJobDetails, setShowJobDetails] = useState(false)
  const [syncing, setSyncing] = useState(false)
  const [statsExpanded, setStatsExpanded] = useState(false)
  const [namespaces, setNamespaces] = useState<string[]>(['default'])
  const [selectedNamespace, setSelectedNamespace] = useState<string>('default')

  // Pagination state
  const [jobsOffset, setJobsOffset] = useState(0)
  const [hasMoreJobs, setHasMoreJobs] = useState(true)
  const [loadingMore, setLoadingMore] = useState(false)

  // Run job dialog state
  const [showRunDialog, setShowRunDialog] = useState(false)
  const [selectedFunction, setSelectedFunction] = useState<JobFunction | null>(null)
  const [jobPayload, setJobPayload] = useState('')
  const [submittingJob, setSubmittingJob] = useState(false)
  const [togglingJob, setTogglingJob] = useState<string | null>(null)

  // Execution logs state (fetched from separate table)
  const [executionLogs, setExecutionLogs] = useState<ExecutionLog[]>([])
  const [loadingLogs, setLoadingLogs] = useState(false)

  // Ref for auto-scrolling logs
  const logsContainerRef = useRef<HTMLDivElement>(null)
  const prevLogsLengthRef = useRef<number>(0)

  // Fetch namespaces on mount
  useEffect(() => {
    const fetchNamespaces = async () => {
      try {
        const data = await jobsApi.listNamespaces()
        setNamespaces(data.length > 0 ? data : ['default'])
        // If current namespace not in list, reset to first available
        if (!data.includes(selectedNamespace)) {
          setSelectedNamespace(data[0] || 'default')
        }
      } catch {
        setNamespaces(['default'])
      }
    }
    fetchNamespaces()
  }, [selectedNamespace])

  const fetchJobFunctions = useCallback(async () => {
    try {
      const data = await jobsApi.listFunctions(selectedNamespace)
      setJobFunctions(data || [])
    } catch {
      toast.error('Failed to fetch job functions')
    }
  }, [selectedNamespace])

  const fetchJobs = useCallback(async (reset = true) => {
    try {
      const offset = reset ? 0 : jobsOffset
      const filters: { status?: string; namespace?: string; limit: number; offset: number } = {
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
        setJobs(prev => [...prev, ...newJobs])
        setJobsOffset(prev => prev + JOBS_PAGE_SIZE)
      }

      // If we got fewer jobs than requested, there are no more
      setHasMoreJobs(newJobs.length >= JOBS_PAGE_SIZE)
    } catch {
      toast.error('Failed to fetch jobs')
    }
  }, [selectedNamespace, statusFilter, jobsOffset])

  const loadMoreJobs = useCallback(async () => {
    setLoadingMore(true)
    try {
      await fetchJobs(false)
    } finally {
      setLoadingMore(false)
    }
  }, [fetchJobs])

  // Fetch execution logs and subscribe to Realtime when modal is open
  useEffect(() => {
    if (!showJobDetails || !selectedJob) return

    const jobId = selectedJob.id
    const isActiveJob = selectedJob.status === 'running' || selectedJob.status === 'pending'

    // Fetch initial logs
    const fetchLogs = async () => {
      setLoadingLogs(true)
      try {
        const logs = await jobsApi.getJobLogs(jobId)
        setExecutionLogs(logs)
        // Scroll to bottom after loading
        setTimeout(() => {
          if (logsContainerRef.current) {
            logsContainerRef.current.scrollTop = logsContainerRef.current.scrollHeight
          }
        }, 50)
      } catch {
        // Silently fail
      } finally {
        setLoadingLogs(false)
      }
    }
    fetchLogs()

    // Subscribe to Realtime for new logs if job is active
    let channel: ReturnType<typeof fluxbaseClient.channel> | null = null
    if (isActiveJob) {
      channel = fluxbaseClient
        .channel(`job-logs-${jobId}`)
        .on(
          'postgres_changes',
          {
            event: 'INSERT',
            schema: 'jobs',
            table: 'execution_logs',
            filter: `job_id=eq.${jobId}`,
          },
          (payload) => {
            const newLog = payload.new as ExecutionLog
            setExecutionLogs((prev) => [...prev, newLog])
            // Auto-scroll on new log
            setTimeout(() => {
              if (logsContainerRef.current) {
                logsContainerRef.current.scrollTop = logsContainerRef.current.scrollHeight
              }
            }, 50)
          }
        )
        .subscribe()
    }

    return () => {
      if (channel) {
        channel.unsubscribe()
      }
      setExecutionLogs([])
    }
  }, [showJobDetails, selectedJob?.id, selectedJob?.status])

  // Poll for job updates when modal is open and job is running/pending
  useEffect(() => {
    if (!showJobDetails || !selectedJob) return

    // Only poll for active jobs
    const isActiveJob = selectedJob.status === 'running' || selectedJob.status === 'pending'
    if (!isActiveJob) return

    const pollInterval = setInterval(async () => {
      try {
        const updatedJob = await jobsApi.getJob(selectedJob.id)
        setSelectedJob(updatedJob)

        // Stop polling if job is no longer active
        if (updatedJob.status !== 'running' && updatedJob.status !== 'pending') {
          // Refresh the jobs list to update statuses
          fetchJobs(true)
        }
      } catch {
        // Silently fail - modal may have been closed
      }
    }, 1000) // Poll every second

    return () => clearInterval(pollInterval)
    // eslint-disable-next-line react-hooks/exhaustive-deps -- We only want to restart polling when job ID or status changes, not on every selectedJob object change
  }, [showJobDetails, selectedJob?.id, selectedJob?.status, fetchJobs])

  // Reset logs scroll ref when opening a new job
  useEffect(() => {
    if (showJobDetails && selectedJob) {
      prevLogsLengthRef.current = executionLogs.length
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- Only reset on job ID change, not every selectedJob update
  }, [showJobDetails, selectedJob?.id])

  // Track if we have active jobs (state to trigger effect when it changes)
  const [hasActiveJobs, setHasActiveJobs] = useState(false)
  useEffect(() => {
    const active = jobs.some(j => j.status === 'running' || j.status === 'pending')
    setHasActiveJobs(active)
  }, [jobs])

  // Auto-refresh jobs list when there are running/pending jobs (only when modal is closed)
  // Only create interval when we actually have active jobs
  useEffect(() => {
    // Don't poll if modal is open or no active jobs
    if (showJobDetails || !hasActiveJobs) return

    const pollInterval = setInterval(async () => {
      try {
        // Fetch just the first page to update visible jobs
        const filters: { status?: string; namespace?: string; limit: number; offset: number } = {
          limit: JOBS_PAGE_SIZE,
          offset: 0,
          namespace: selectedNamespace,
        }
        if (statusFilter !== 'all') {
          filters.status = statusFilter
        }
        const data = await jobsApi.listJobs(filters)
        const newJobs = data || []

        // Only update if data has changed (compare by serializing relevant fields)
        setJobs(prev => {
          // Quick check: if lengths differ, definitely update
          const firstPagePrev = prev.slice(0, JOBS_PAGE_SIZE)
          if (firstPagePrev.length !== newJobs.length) {
            if (prev.length <= JOBS_PAGE_SIZE) {
              return newJobs
            }
            const additionalJobs = prev.slice(JOBS_PAGE_SIZE)
            return [...newJobs, ...additionalJobs]
          }

          // Check if any job has changed (status, progress, etc.)
          let hasChanges = false
          for (let i = 0; i < newJobs.length; i++) {
            const oldJob = firstPagePrev[i]
            const newJob = newJobs[i]
            if (!oldJob || oldJob.id !== newJob.id ||
              oldJob.status !== newJob.status ||
              oldJob.progress_percent !== newJob.progress_percent ||
              oldJob.progress_message !== newJob.progress_message ||
              oldJob.error_message !== newJob.error_message) {
              hasChanges = true
              break
            }
          }

          // No changes, return previous state to avoid re-render
          if (!hasChanges) return prev

          if (prev.length <= JOBS_PAGE_SIZE) {
            return newJobs
          }
          const additionalJobs = prev.slice(JOBS_PAGE_SIZE)
          return [...newJobs, ...additionalJobs]
        })
      } catch {
        // Silently fail
      }
    }, 3000) // Poll every 3 seconds for the list

    return () => clearInterval(pollInterval)
  }, [showJobDetails, hasActiveJobs, selectedNamespace, statusFilter])

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
      await Promise.all([
        fetchJobFunctions(),
        fetchJobs(true),
        fetchWorkers(),
      ])
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
        const ns = availableNamespaces.includes('default') ? 'default' : availableNamespaces[0]

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
        toast.success(`Jobs synced to "${selectedNamespace}": ${messages.join(', ')}`)
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

      const job = await jobsApi.submitJob({
        job_name: selectedFunction.name,
        namespace: selectedNamespace,
        payload,
      })

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
      parts.push(`Started: ${new Date(selectedJob.started_at).toLocaleString()}`)
    }
    if (selectedJob.completed_at) {
      parts.push(`Completed: ${new Date(selectedJob.completed_at).toLocaleString()}`)
    }
    parts.push('')

    if (selectedJob.payload !== undefined && selectedJob.payload !== null) {
      parts.push(`=== Payload ===`)
      parts.push(formatJsonValue(selectedJob.payload))
      parts.push('')
    }

    if (executionLogs.length > 0) {
      parts.push(`=== Logs ===`)
      parts.push(executionLogs.map((l) => l.message).join('\n'))
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

  // Parse time range to milliseconds
  const getTimeRangeMs = (range: string): number => {
    const value = parseInt(range.slice(0, -1))
    const unit = range.slice(-1)
    switch (unit) {
      case 'm':
        return value * 60 * 1000
      case 'h':
        return value * 60 * 60 * 1000
      default:
        return 0
    }
  }

  // Filter jobs by time range
  const timeRangeMs = getTimeRangeMs(timeRange)
  const timeRangeStart = Date.now() - timeRangeMs
  const filteredByTimeJobs = jobs.filter((job) => {
    const jobTime = new Date(job.created_at).getTime()
    return jobTime >= timeRangeStart
  })

  // Recalculate stats based on time-filtered jobs
  const filteredStats = {
    pending: filteredByTimeJobs.filter(j => j.status === 'pending').length,
    running: filteredByTimeJobs.filter(j => j.status === 'running').length,
    completed: filteredByTimeJobs.filter(j => j.status === 'completed').length,
    failed: filteredByTimeJobs.filter(j => j.status === 'failed').length,
    cancelled: filteredByTimeJobs.filter(j => j.status === 'cancelled').length,
    total: filteredByTimeJobs.length,
  }

  const filteredJobs = filteredByTimeJobs.filter((job) => {
    if (searchQuery && !job.job_name.toLowerCase().includes(searchQuery.toLowerCase())) {
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
    <div className='flex flex-col gap-6 p-6'>
      <ImpersonationBanner />

      <div className='flex items-center justify-between'>
        <div>
          <h1 className='text-3xl font-bold'>Background Jobs</h1>
          <p className='text-muted-foreground'>
            Manage job functions and monitor background task execution
          </p>
        </div>
        <div className='flex items-center gap-2'>
          <ImpersonationSelector />
          <Button onClick={refreshAllData} variant='outline' size='sm'>
            <RefreshCw className='mr-2 h-4 w-4' />
            Refresh
          </Button>
        </div>
      </div>

      {/* Time Range Selector */}
      <div className='flex items-center gap-2'>
        <span className='text-sm text-muted-foreground'>Time range:</span>
        {['1m', '5m', '30m', '1h', '6h', '12h', '24h'].map((range) => (
          <Button
            key={range}
            variant={timeRange === range ? 'default' : 'outline'}
            size='sm'
            onClick={() => setTimeRange(range)}
            className='h-7 px-3 text-xs'
          >
            {range}
          </Button>
        ))}
      </div>

      {/* Collapsible Metrics */}
      <Card className='!gap-0 !py-0'>
        <CardContent className='py-2 px-4'>
          {/* Collapsed View - Single Line with Key Figures */}
          <div className='flex items-center justify-between'>
            <div className='flex items-center gap-4'>
              <div className='flex items-center gap-1'>
                <span className='text-xs text-muted-foreground'>Pending:</span>
                <span className='text-sm font-semibold'>{filteredStats.pending}</span>
              </div>
              <div className='flex items-center gap-1'>
                <span className='text-xs text-muted-foreground'>Running:</span>
                <span className='text-sm font-semibold'>{filteredStats.running}</span>
              </div>
              <div className='flex items-center gap-1'>
                <span className='text-xs text-muted-foreground'>Completed:</span>
                <span className='text-sm font-semibold'>{filteredStats.completed}</span>
              </div>
              <div className='flex items-center gap-1'>
                <span className='text-xs text-muted-foreground'>Failed:</span>
                <span className='text-sm font-semibold'>{filteredStats.failed}</span>
              </div>
              <div className='flex items-center gap-1'>
                <span className='text-xs text-muted-foreground'>Workers:</span>
                <span className='text-sm font-semibold'>
                  {workers.filter((w) => w.status === 'active').length}
                </span>
              </div>
              <div className='flex items-center gap-1'>
                <Target className='h-3 w-3 text-muted-foreground' />
                <span className='text-xs text-muted-foreground'>Success:</span>
                {(() => {
                  const total = filteredStats.completed + filteredStats.failed
                  const successRate = total > 0
                    ? ((filteredStats.completed / total) * 100).toFixed(0)
                    : '0'
                  return <span className='text-sm font-semibold'>{successRate}%</span>
                })()}
              </div>
              <div className='flex items-center gap-1'>
                <Timer className='h-3 w-3 text-muted-foreground' />
                <span className='text-xs text-muted-foreground'>Avg. Wait:</span>
                {(() => {
                  const pendingJobs = filteredByTimeJobs.filter(j => j.status === 'pending')
                  const waitTimes = pendingJobs.map(j =>
                    Date.now() - new Date(j.created_at).getTime()
                  )
                  const avgWaitMs = waitTimes.length > 0
                    ? waitTimes.reduce((a, b) => a + b, 0) / waitTimes.length
                    : 0
                  const avgWaitSec = Math.round(avgWaitMs / 1000)
                  const displayTime = avgWaitSec < 60
                    ? `${avgWaitSec}s`
                    : avgWaitSec < 3600
                      ? `${Math.round(avgWaitSec / 60)}m`
                      : `${Math.round(avgWaitSec / 3600)}h`
                  return <span className='text-sm font-semibold'>{displayTime}</span>
                })()}
              </div>
            </div>
            <Button
              variant='ghost'
              size='sm'
              onClick={() => setStatsExpanded(!statsExpanded)}
              className='h-8 w-8 p-0'
            >
              {statsExpanded ? (
                <ChevronUp className='h-4 w-4' />
              ) : (
                <ChevronDown className='h-4 w-4' />
              )}
            </Button>
          </div>

          {/* Expanded View - Charts */}
          {statsExpanded && (
            <div className='mt-4 space-y-4'>
              <Separator />

              {/* Distribution Chart */}
              <div>
                <div className='mb-2 flex items-center gap-2'>
                  <TrendingUp className='h-4 w-4 text-muted-foreground' />
                  <span className='text-sm font-medium'>Status Distribution</span>
                </div>
                {(() => {
                  const chartData = [
                    { name: 'Completed', value: filteredStats.completed, color: '#22c55e' },
                    { name: 'Running', value: filteredStats.running, color: '#3b82f6' },
                    { name: 'Pending', value: filteredStats.pending, color: '#eab308' },
                    { name: 'Failed', value: filteredStats.failed, color: '#ef4444' },
                  ].filter(item => item.value > 0)

                  const total = chartData.reduce((sum, item) => sum + item.value, 0)

                  if (total === 0) {
                    return (
                      <div className='text-muted-foreground flex h-[120px] items-center justify-center text-sm'>
                        No jobs in range
                      </div>
                    )
                  }

                  return (
                    <ResponsiveContainer width='100%' height={120}>
                      <PieChart>
                        <Pie
                          data={chartData}
                          cx='50%'
                          cy='50%'
                          innerRadius={30}
                          outerRadius={50}
                          paddingAngle={2}
                          dataKey='value'
                        >
                          {chartData.map((entry, index) => (
                            <Cell key={`cell-${index}`} fill={entry.color} />
                          ))}
                        </Pie>
                        <Tooltip
                          contentStyle={{
                            backgroundColor: 'hsl(var(--background))',
                            border: '1px solid hsl(var(--border))',
                            borderRadius: '6px',
                            fontSize: '11px',
                          }}
                          formatter={(value: number, name: string) => [`${value}`, name]}
                        />
                        <Legend
                          verticalAlign='bottom'
                          height={36}
                          iconType='circle'
                          formatter={(value) => (
                            <span className='text-xs text-muted-foreground'>{value}</span>
                          )}
                        />
                      </PieChart>
                    </ResponsiveContainer>
                  )
                })()}
              </div>

              {/* Worker Activity Chart */}
              <div>
                <div className='mb-2 flex items-center justify-between'>
                  <div className='flex items-center gap-2'>
                    <Activity className='h-4 w-4 text-muted-foreground' />
                    <span className='text-sm font-medium'>Worker Activity</span>
                  </div>
                  <Badge variant='outline' className='text-xs'>{workers.length} workers</Badge>
                </div>
                {(() => {
                  if (workers.length === 0) {
                    return (
                      <div className='text-muted-foreground flex h-[120px] items-center justify-center text-sm'>
                        No workers available
                      </div>
                    )
                  }

                  const workerData = workers.map((worker) => ({
                    id: worker.id.substring(0, 8),
                    current: worker.current_jobs,
                    completed: worker.total_completed,
                    status: worker.status,
                  }))

                  return (
                    <ResponsiveContainer width='100%' height={120}>
                      <BarChart data={workerData}>
                        <CartesianGrid strokeDasharray='3 3' stroke='hsl(var(--border))' />
                        <XAxis
                          dataKey='id'
                          tick={{ fill: 'hsl(var(--muted-foreground))', fontSize: 10 }}
                        />
                        <YAxis
                          tick={{ fill: 'hsl(var(--muted-foreground))', fontSize: 10 }}
                        />
                        <Tooltip
                          contentStyle={{
                            backgroundColor: 'hsl(var(--background))',
                            border: '1px solid hsl(--border))',
                            borderRadius: '6px',
                            fontSize: '11px',
                          }}
                          cursor={{ fill: 'hsl(var(--muted) / 0.2)' }}
                        />
                        <Legend
                          wrapperStyle={{ fontSize: '10px' }}
                          iconType='circle'
                          iconSize={8}
                        />
                        <Bar
                          dataKey='current'
                          name='Current'
                          fill='#3b82f6'
                          radius={[3, 3, 0, 0]}
                        />
                        <Bar
                          dataKey='completed'
                          name='Completed'
                          fill='#22c55e'
                          radius={[3, 3, 0, 0]}
                        />
                      </BarChart>
                    </ResponsiveContainer>
                  )
                })()}
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      <Tabs
        value={activeTab}
        onValueChange={(v) => setActiveTab(v as 'functions' | 'queue')}
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

        <TabsContent value='queue' className='mt-6 space-y-6'>
          {/* Filters */}
          <div className='flex items-center gap-3'>
            <div className='flex items-center gap-2'>
              <Label htmlFor='queue-namespace-select' className='text-sm text-muted-foreground whitespace-nowrap'>
                Namespace:
              </Label>
              <Select value={selectedNamespace} onValueChange={setSelectedNamespace}>
                <SelectTrigger id='queue-namespace-select' className='w-[180px]'>
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
            <Select value={statusFilter} onValueChange={(v) => {
              setStatusFilter(v)
              setJobsOffset(0)
              setHasMoreJobs(true)
              setTimeout(() => fetchJobs(true), 100)
            }}>
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
          <ScrollArea className='h-[calc(100vh-32rem)]'>
            <div className='grid gap-4'>
              {filteredJobs.length === 0 ? (
                <Card>
                  <CardContent className='p-12 text-center'>
                    <ListTodo className='text-muted-foreground mx-auto mb-4 h-12 w-12' />
                    <p className='mb-2 text-lg font-medium'>
                      No jobs found
                    </p>
                    <p className='text-muted-foreground text-sm'>
                      {searchQuery || statusFilter !== 'all'
                        ? 'Try adjusting your filters'
                        : 'Submit a job to see it here'}
                    </p>
                  </CardContent>
                </Card>
              ) : (
                filteredJobs.map((job) => (
                  <Card
                    key={job.id}
                    className='hover:border-primary/50 transition-colors'
                  >
                    <CardHeader>
                      <div className='flex items-start justify-between'>
                        <div className='flex-1'>
                          <div className='mb-2 flex items-center gap-2'>
                            {getStatusIcon(job.status)}
                            <CardTitle className='text-lg'>
                              {job.job_name}
                            </CardTitle>
                            <Badge variant={getStatusBadgeVariant(job.status)}>
                              {job.status}
                            </Badge>
                            {job.user_email && (
                              <Badge variant='outline' className='text-xs'>
                                {job.user_email}
                              </Badge>
                            )}
                          </div>
                          <CardDescription className='flex items-center gap-4 text-xs'>
                            <span>ID: {job.id.substring(0, 8)}...</span>
                            {job.progress_percent !== undefined && (
                              <span>Progress: {job.progress_percent}%</span>
                            )}
                            {job.retry_count > 0 && (
                              <span>Retry: {job.retry_count}/{job.max_retries}</span>
                            )}
                            <span>
                              {new Date(job.created_at).toLocaleString()}
                            </span>
                          </CardDescription>
                        </div>
                        <div className='flex gap-2'>
                          <Button
                            onClick={() => viewJobDetails(job)}
                            size='sm'
                            variant='outline'
                          >
                            View
                          </Button>
                          {(job.status === 'running' || job.status === 'pending') && (
                            <Button
                              onClick={() => cancelJob(job.id)}
                              size='sm'
                              variant='outline'
                            >
                              <XCircle className='h-4 w-4' />
                            </Button>
                          )}
                          {(job.status === 'completed' || job.status === 'cancelled' || job.status === 'failed') && (
                            <Button
                              onClick={() => resubmitJob(job.id)}
                              size='sm'
                              variant='outline'
                              title='Re-submit as new job'
                            >
                              <RefreshCw className='h-4 w-4' />
                            </Button>
                          )}
                        </div>
                      </div>
                    </CardHeader>
                    {(job.status === 'running' && job.progress_percent !== undefined) || job.progress_message ? (
                      <CardContent className='pt-0 pb-4'>
                        {job.progress_message && (
                          <div className='flex items-center gap-2 text-sm mb-2'>
                            <Activity className='h-3 w-3' />
                            <span className='text-muted-foreground'>
                              {job.progress_message}
                            </span>
                          </div>
                        )}
                        {job.progress_percent !== undefined && (
                          <div className='space-y-1'>
                            <div className='flex items-center justify-between text-xs text-muted-foreground'>
                              <span>{job.progress_percent}%</span>
                              {job.estimated_seconds_left !== undefined && job.estimated_seconds_left > 0 && (
                                <span>
                                  ~{job.estimated_seconds_left < 60
                                    ? `${job.estimated_seconds_left}s`
                                    : job.estimated_seconds_left < 3600
                                      ? `${Math.round(job.estimated_seconds_left / 60)}m`
                                      : `${Math.round(job.estimated_seconds_left / 3600)}h`} remaining
                                </span>
                              )}
                            </div>
                            <div className='h-2 w-full overflow-hidden rounded-full bg-secondary'>
                              <div
                                className={`h-full transition-all duration-300 ${job.status === 'running' ? 'bg-blue-500' :
                                    job.status === 'completed' ? 'bg-green-500' :
                                      job.status === 'failed' ? 'bg-red-500' : 'bg-primary'
                                  }`}
                                style={{ width: `${job.progress_percent}%` }}
                              />
                            </div>
                          </div>
                        )}
                      </CardContent>
                    ) : null}
                  </Card>
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

        <TabsContent value='functions' className='mt-6 space-y-6'>
          {/* Namespace Selector and Sync */}
          <div className='flex items-center justify-between'>
            <div className='flex items-center gap-2'>
              <Label htmlFor='namespace-select' className='text-sm text-muted-foreground whitespace-nowrap'>
                Namespace:
              </Label>
              <Select value={selectedNamespace} onValueChange={setSelectedNamespace}>
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
              <CardContent className='py-4 px-4'>
                <div className='text-xs text-muted-foreground mb-1'>Total Functions</div>
                <div className='text-2xl font-bold'>{jobFunctions.length}</div>
              </CardContent>
            </Card>
            <Card className='!gap-0'>
              <CardContent className='py-4 px-4'>
                <div className='text-xs text-muted-foreground mb-1'>Enabled</div>
                <div className='text-2xl font-bold'>
                  {jobFunctions.filter((f) => f.enabled).length}
                </div>
              </CardContent>
            </Card>
            <Card className='!gap-0'>
              <CardContent className='py-4 px-4'>
                <div className='text-xs text-muted-foreground mb-1'>Scheduled</div>
                <div className='text-2xl font-bold'>
                  {jobFunctions.filter((f) => f.schedule).length}
                </div>
              </CardContent>
            </Card>
          </div>

          <ScrollArea className='h-[calc(100vh-32rem)]'>
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
                  <Card
                    key={fn.id}
                    className='hover:border-primary/50 transition-colors'
                  >
                    <CardHeader className='pt-4 pb-3'>
                      <div className='flex items-start justify-between'>
                        <div className='flex-1'>
                          <div className='mb-2 flex items-center gap-2'>
                            <CardTitle className='text-lg'>{fn.name}</CardTitle>
                            <Badge variant='outline'>v{fn.version}</Badge>
                            {fn.schedule && (
                              <Badge variant='outline'>
                                <Clock className='mr-1 h-3 w-3' />
                                {fn.schedule}
                              </Badge>
                            )}
                            {fn.require_role && (
                              <Badge variant='outline'>
                                requires: {fn.require_role}
                              </Badge>
                            )}
                          </div>
                          <CardDescription>
                            {fn.description || 'No description'}
                          </CardDescription>
                        </div>
                        <div className='flex items-center gap-3'>
                          <div className='flex items-center gap-2'>
                            <Label
                              htmlFor={`enable-${fn.id}`}
                              className='text-xs text-muted-foreground'
                            >
                              {fn.enabled ? 'Enabled' : 'Disabled'}
                            </Label>
                            <Switch
                              id={`enable-${fn.id}`}
                              checked={fn.enabled}
                              disabled={togglingJob === fn.id}
                              onCheckedChange={() => toggleJobEnabled(fn)}
                            />
                          </div>
                          <Button
                            size='sm'
                            variant='default'
                            onClick={() => openRunDialog(fn)}
                            disabled={!fn.enabled}
                          >
                            <Play className='mr-1 h-3 w-3' />
                            Run
                          </Button>
                        </div>
                      </div>
                    </CardHeader>
                    <CardContent className='pt-0 pb-4'>
                      <div className='space-y-2 text-sm'>
                        <div className='flex items-center gap-4'>
                          <span className='text-muted-foreground'>Timeout:</span>
                          <span>{fn.timeout_seconds}s</span>
                          <span className='text-muted-foreground'>Memory:</span>
                          <span>{fn.memory_limit_mb}MB</span>
                          <span className='text-muted-foreground'>Retries:</span>
                          <span>{fn.max_retries}</span>
                        </div>
                        <div className='flex items-center gap-2'>
                          <span className='text-muted-foreground'>
                            Permissions:
                          </span>
                          {fn.allow_net && <Badge variant='outline'>net</Badge>}
                          {fn.allow_env && <Badge variant='outline'>env</Badge>}
                          {fn.allow_read && <Badge variant='outline'>read</Badge>}
                          {fn.allow_write && <Badge variant='outline'>write</Badge>}
                          {!fn.allow_net && !fn.allow_env && !fn.allow_read && !fn.allow_write && (
                            <span className='text-muted-foreground italic'>none</span>
                          )}
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                ))
              )}
            </div>
          </ScrollArea>
        </TabsContent>
      </Tabs>

      {/* Job Details Dialog */}
      <Dialog open={showJobDetails} onOpenChange={setShowJobDetails}>
        <DialogContent className='max-h-[90vh] w-[90vw] sm:max-w-none max-w-[1600px] overflow-y-auto'>
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
                  <Badge variant='outline'>{selectedJob.user_email}</Badge>
                )}
                {selectedJob.user_role && (
                  <Badge variant='outline'>role: {selectedJob.user_role}</Badge>
                )}
              </div>

              <Separator />

              <div className='grid gap-3'>
                <div>
                  <Label className='text-xs text-muted-foreground'>Created</Label>
                  <p className='text-sm'>{new Date(selectedJob.created_at).toLocaleString()}</p>
                </div>
                {selectedJob.started_at && (
                  <div>
                    <Label className='text-xs text-muted-foreground'>Started</Label>
                    <p className='text-sm'>{new Date(selectedJob.started_at).toLocaleString()}</p>
                  </div>
                )}
                {selectedJob.completed_at && (
                  <div>
                    <Label className='text-xs text-muted-foreground'>Completed</Label>
                    <p className='text-sm'>{new Date(selectedJob.completed_at).toLocaleString()}</p>
                  </div>
                )}
                {selectedJob.progress_percent !== undefined && (
                  <div className='space-y-2'>
                    <Label className='text-xs text-muted-foreground'>Progress</Label>
                    <div className='space-y-1'>
                      <div className='flex items-center justify-between text-sm'>
                        <span className='font-medium'>{selectedJob.progress_percent}%</span>
                        {selectedJob.estimated_seconds_left !== undefined && selectedJob.estimated_seconds_left > 0 && (
                          <span className='text-muted-foreground'>
                            ~{selectedJob.estimated_seconds_left < 60
                              ? `${selectedJob.estimated_seconds_left}s`
                              : selectedJob.estimated_seconds_left < 3600
                                ? `${Math.round(selectedJob.estimated_seconds_left / 60)}m`
                                : `${Math.round(selectedJob.estimated_seconds_left / 3600)}h`} remaining
                          </span>
                        )}
                      </div>
                      <div className='h-3 w-full overflow-hidden rounded-full bg-secondary'>
                        <div
                          className={`h-full transition-all duration-300 ${selectedJob.status === 'running' ? 'bg-blue-500' :
                              selectedJob.status === 'completed' ? 'bg-green-500' :
                                selectedJob.status === 'failed' ? 'bg-red-500' : 'bg-primary'
                            }`}
                          style={{ width: `${selectedJob.progress_percent}%` }}
                        />
                      </div>
                      {selectedJob.progress_message && (
                        <p className='text-sm text-muted-foreground'>{selectedJob.progress_message}</p>
                      )}
                      {selectedJob.last_progress_at && (
                        <p className='text-xs text-muted-foreground'>
                          Last updated: {new Date(selectedJob.last_progress_at).toLocaleString()}
                        </p>
                      )}
                    </div>
                  </div>
                )}
              </div>

              <Separator />

              {selectedJob.payload !== undefined && selectedJob.payload !== null && (
                <div>
                  <div className='flex items-center justify-between mb-2'>
                    <Label>Payload</Label>
                    <Button
                      variant='ghost'
                      size='sm'
                      className='h-6 px-2'
                      onClick={() => copyToClipboard(formatJsonValue(selectedJob.payload), 'Payload')}
                    >
                      <Copy className='h-3 w-3' />
                    </Button>
                  </div>
                  <div className='bg-muted max-h-48 overflow-auto rounded-lg border p-4'>
                    <pre className='text-xs whitespace-pre-wrap break-all'>
                      {formatJsonValue(selectedJob.payload)}
                    </pre>
                  </div>
                </div>
              )}

              {/* Logs and Result/Error side by side */}
              <div className='grid grid-cols-1 lg:grid-cols-2 gap-4'>
                {/* Logs Column - Always show for consistent layout */}
                <div className='flex flex-col'>
                  <div className='flex items-center justify-between mb-2'>
                    <Label>Logs</Label>
                    <div className='flex items-center gap-2'>
                      {(selectedJob.status === 'running' || selectedJob.status === 'pending') && (
                        <div className='flex items-center gap-2 text-xs text-muted-foreground'>
                          <Loader2 className='h-3 w-3 animate-spin' />
                          <span>Live updating...</span>
                        </div>
                      )}
                      {executionLogs.length > 0 && (
                        <Button
                          variant='ghost'
                          size='sm'
                          className='h-6 px-2'
                          onClick={() => copyToClipboard(executionLogs.map((l) => l.message).join('\n'), 'Logs')}
                        >
                          <Copy className='h-3 w-3' />
                        </Button>
                      )}
                    </div>
                  </div>
                  <div
                    ref={logsContainerRef}
                    className='bg-black/90 flex-1 min-h-[200px] max-h-[400px] overflow-y-auto rounded-lg border p-4 font-mono'
                  >
                    {loadingLogs ? (
                      <span className='text-xs text-muted-foreground italic'>
                        Loading logs...
                      </span>
                    ) : executionLogs.length > 0 ? (
                      <pre className='text-xs text-green-400 whitespace-pre-wrap break-words'>
                        {executionLogs.map((l) => l.message).join('\n')}
                      </pre>
                    ) : (
                      <span className='text-xs text-muted-foreground italic'>
                        No logs available
                      </span>
                    )}
                  </div>
                </div>

                {/* Result/Error Column */}
                {(selectedJob.result !== undefined && selectedJob.result !== null) || selectedJob.error_message ? (
                  <div className='flex flex-col gap-4'>
                    {selectedJob.result !== undefined && selectedJob.result !== null && (
                      <div className='flex flex-col flex-1'>
                        <div className='flex items-center justify-between mb-2'>
                          <Label>Result</Label>
                          <Button
                            variant='ghost'
                            size='sm'
                            className='h-6 px-2'
                            onClick={() => copyToClipboard(formatJsonValue(selectedJob.result), 'Result')}
                          >
                            <Copy className='h-3 w-3' />
                          </Button>
                        </div>
                        <div className='bg-muted flex-1 min-h-[100px] max-h-[200px] overflow-auto rounded-lg border p-4'>
                          <pre className='text-xs whitespace-pre-wrap break-all'>
                            {formatJsonValue(selectedJob.result)}
                          </pre>
                        </div>
                      </div>
                    )}

                    {selectedJob.error_message && (
                      <div className='flex flex-col flex-1'>
                        <div className='flex items-center justify-between mb-2'>
                          <Label className='text-destructive'>Error</Label>
                          <Button
                            variant='ghost'
                            size='sm'
                            className='h-6 px-2'
                            onClick={() => copyToClipboard(selectedJob.error_message || '', 'Error')}
                          >
                            <Copy className='h-3 w-3' />
                          </Button>
                        </div>
                        <div className='bg-destructive/10 border-destructive/20 flex-1 min-h-[100px] max-h-[200px] overflow-auto rounded-lg border p-4'>
                          <pre className='text-xs text-destructive whitespace-pre-wrap break-all'>
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
            {selectedJob && (selectedJob.status === 'completed' || selectedJob.status === 'cancelled' || selectedJob.status === 'failed') && (
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
              Submit a new job for "{selectedFunction?.name}" in the "{selectedNamespace}" namespace
            </DialogDescription>
          </DialogHeader>

          <div className='space-y-4'>
            {selectedFunction && (
              <div className='rounded-lg border bg-muted/50 p-3'>
                <div className='flex items-center gap-2 mb-2'>
                  <span className='font-medium'>{selectedFunction.name}</span>
                  <Badge variant='outline'>v{selectedFunction.version}</Badge>
                </div>
                <p className='text-sm text-muted-foreground'>
                  {selectedFunction.description || 'No description'}
                </p>
                <div className='mt-2 flex items-center gap-4 text-xs text-muted-foreground'>
                  <span>Timeout: {selectedFunction.timeout_seconds}s</span>
                  <span>Max retries: {selectedFunction.max_retries}</span>
                </div>
              </div>
            )}

            <div className='space-y-2'>
              <Label htmlFor='job-payload'>
                Payload (JSON)
              </Label>
              <Textarea
                id='job-payload'
                value={jobPayload}
                onChange={(e) => setJobPayload(e.target.value)}
                placeholder='{\n  "key": "value"\n}'
                className='font-mono text-sm min-h-[150px]'
              />
              <p className='text-xs text-muted-foreground'>
                Enter the JSON payload to pass to the job's handler function. This will be available as <code className='bg-muted px-1 rounded'>request.payload</code> in your job code.
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
            <Button
              onClick={handleSubmitJob}
              disabled={submittingJob}
            >
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
    </div>
  )
}
