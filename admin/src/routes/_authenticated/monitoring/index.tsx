import z from 'zod'
import { createFileRoute, getRouteApi } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Activity, Database, HardDrive, Zap, Cpu, MemoryStick, Network, CheckCircle2, AlertCircle, XCircle } from 'lucide-react'
import { useState } from 'react'
import { monitoringApi, type SystemMetrics, type SystemHealth } from '@/lib/api'

const monitoringSearchSchema = z.object({
  tab: z.string().optional().catch('overview'),
})

export const Route = createFileRoute('/_authenticated/monitoring/')({
  validateSearch: monitoringSearchSchema,
  component: MonitoringPage,
})

const route = getRouteApi('/_authenticated/monitoring/')

function MonitoringPage() {
  const search = route.useSearch()
  const navigate = route.useNavigate()
  const [autoRefresh, setAutoRefresh] = useState(true)

  // Fetch metrics
  const { data: metrics } = useQuery<SystemMetrics>({
    queryKey: ['monitoring-metrics'],
    queryFn: monitoringApi.getMetrics,
    refetchInterval: autoRefresh ? 5000 : false,
  })

  // Fetch health
  const { data: health } = useQuery<SystemHealth>({
    queryKey: ['monitoring-health'],
    queryFn: monitoringApi.getHealth,
    refetchInterval: autoRefresh ? 10000 : false,
  })

  // Format uptime
  const formatUptime = (seconds: number) => {
    const days = Math.floor(seconds / 86400)
    const hours = Math.floor((seconds % 86400) / 3600)
    const minutes = Math.floor((seconds % 3600) / 60)
    const secs = Math.floor(seconds % 60)

    if (days > 0) return `${days}d ${hours}h ${minutes}m`
    if (hours > 0) return `${hours}h ${minutes}m ${secs}s`
    if (minutes > 0) return `${minutes}m ${secs}s`
    return `${secs}s`
  }

  // Get status badge
  const getStatusBadge = (status: string) => {
    if (status === 'healthy') {
      return (
        <Badge variant='outline' className='border-green-500 text-green-500'>
          <CheckCircle2 className='mr-1 h-3 w-3' />
          Healthy
        </Badge>
      )
    }
    if (status === 'degraded') {
      return (
        <Badge variant='outline' className='border-yellow-500 text-yellow-500'>
          <AlertCircle className='mr-1 h-3 w-3' />
          Degraded
        </Badge>
      )
    }
    return (
      <Badge variant='outline' className='border-red-500 text-red-500'>
        <XCircle className='mr-1 h-3 w-3' />
        Unhealthy
      </Badge>
    )
  }

  return (
    <div className='flex flex-col gap-6 p-6'>
      <div className='flex items-center justify-between'>
        <div>
          <h1 className='text-3xl font-bold'>System Monitoring</h1>
          <p className='text-sm text-muted-foreground mt-1'>Real-time system metrics and health status</p>
        </div>
        <div className='flex items-center gap-2'>
          <label className='flex items-center gap-2 text-sm'>
            <input type='checkbox' checked={autoRefresh} onChange={(e) => setAutoRefresh(e.target.checked)} className='rounded' />
            Auto-refresh
          </label>
        </div>
      </div>

      {/* System Status Cards */}
      <div className='grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4'>
        <Card>
          <CardHeader className='pb-2'>
            <CardTitle className='text-sm font-medium text-muted-foreground flex items-center gap-2'>
              <Activity className='h-4 w-4' />
              Uptime
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className='text-2xl font-bold'>{metrics ? formatUptime(metrics.uptime_seconds) : '-'}</div>
            <p className='text-xs text-muted-foreground mt-1'>{metrics?.go_version}</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className='pb-2'>
            <CardTitle className='text-sm font-medium text-muted-foreground flex items-center gap-2'>
              <Cpu className='h-4 w-4' />
              Goroutines
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className='text-2xl font-bold'>{metrics?.num_goroutines || 0}</div>
            <p className='text-xs text-muted-foreground mt-1'>Active goroutines</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className='pb-2'>
            <CardTitle className='text-sm font-medium text-muted-foreground flex items-center gap-2'>
              <MemoryStick className='h-4 w-4' />
              Memory
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className='text-2xl font-bold'>{metrics?.memory_alloc_mb || 0} MB</div>
            <p className='text-xs text-muted-foreground mt-1'>Allocated / {metrics?.memory_sys_mb || 0} MB system</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className='pb-2'>
            <CardTitle className='text-sm font-medium text-muted-foreground flex items-center gap-2'>
              <Network className='h-4 w-4' />
              Health
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className='flex items-center gap-2'>{health && getStatusBadge(health.status)}</div>
            <p className='text-xs text-muted-foreground mt-1'>All systems operational</p>
          </CardContent>
        </Card>
      </div>

      {/* Detailed Metrics Tabs */}
      <Tabs value={search.tab || 'overview'} onValueChange={(tab) => navigate({ search: { tab } })} className='w-full'>
        <TabsList>
          <TabsTrigger value='overview'>Overview</TabsTrigger>
          <TabsTrigger value='database'>Database</TabsTrigger>
          <TabsTrigger value='realtime'>Realtime</TabsTrigger>
          {metrics?.storage && <TabsTrigger value='storage'>Storage</TabsTrigger>}
          <TabsTrigger value='health'>Health Checks</TabsTrigger>
        </TabsList>

        {/* Overview Tab */}
        <TabsContent value='overview' className='space-y-4'>
          <div className='grid grid-cols-1 md:grid-cols-2 gap-4'>
            <Card>
              <CardHeader>
                <CardTitle>System Resources</CardTitle>
                <CardDescription>CPU, memory, and garbage collection metrics</CardDescription>
              </CardHeader>
              <CardContent className='space-y-3'>
                <div className='flex justify-between items-center'>
                  <span className='text-sm'>Memory Allocated</span>
                  <span className='font-mono font-semibold'>{metrics?.memory_alloc_mb} MB</span>
                </div>
                <div className='flex justify-between items-center'>
                  <span className='text-sm'>Total Allocated</span>
                  <span className='font-mono font-semibold'>{metrics?.memory_total_alloc_mb} MB</span>
                </div>
                <div className='flex justify-between items-center'>
                  <span className='text-sm'>System Memory</span>
                  <span className='font-mono font-semibold'>{metrics?.memory_sys_mb} MB</span>
                </div>
                <div className='flex justify-between items-center'>
                  <span className='text-sm'>GC Runs</span>
                  <span className='font-mono font-semibold'>{metrics?.num_gc}</span>
                </div>
                <div className='flex justify-between items-center'>
                  <span className='text-sm'>Last GC Pause</span>
                  <span className='font-mono font-semibold'>{metrics?.gc_pause_ms.toFixed(2)} ms</span>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Service Status</CardTitle>
                <CardDescription>Health status of all system components</CardDescription>
              </CardHeader>
              <CardContent className='space-y-3'>
                {health?.services &&
                  Object.entries(health.services).map(([name, status]) => (
                    <div key={name} className='flex justify-between items-center'>
                      <span className='text-sm capitalize'>{name}</span>
                      <div className='flex items-center gap-2'>
                        {status.latency_ms !== undefined && <span className='text-xs text-muted-foreground'>{status.latency_ms}ms</span>}
                        {getStatusBadge(status.status)}
                      </div>
                    </div>
                  ))}
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        {/* Database Tab */}
        <TabsContent value='database' className='space-y-4'>
          <Card>
            <CardHeader>
              <CardTitle className='flex items-center gap-2'>
                <Database className='h-5 w-5' />
                Database Connection Pool
              </CardTitle>
              <CardDescription>PostgreSQL connection pool statistics</CardDescription>
            </CardHeader>
            <CardContent>
              <div className='grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4'>
                <div className='space-y-2'>
                  <h4 className='text-sm font-medium text-muted-foreground'>Connections</h4>
                  <div className='space-y-1'>
                    <div className='flex justify-between'>
                      <span className='text-sm'>Total</span>
                      <span className='font-mono font-semibold'>{metrics?.database.total_conns}</span>
                    </div>
                    <div className='flex justify-between'>
                      <span className='text-sm'>Idle</span>
                      <span className='font-mono font-semibold'>{metrics?.database.idle_conns}</span>
                    </div>
                    <div className='flex justify-between'>
                      <span className='text-sm'>Acquired</span>
                      <span className='font-mono font-semibold'>{metrics?.database.acquired_conns}</span>
                    </div>
                    <div className='flex justify-between'>
                      <span className='text-sm'>Max</span>
                      <span className='font-mono font-semibold'>{metrics?.database.max_conns}</span>
                    </div>
                  </div>
                </div>

                <div className='space-y-2'>
                  <h4 className='text-sm font-medium text-muted-foreground'>Acquire Stats</h4>
                  <div className='space-y-1'>
                    <div className='flex justify-between'>
                      <span className='text-sm'>Total Acquires</span>
                      <span className='font-mono font-semibold'>{metrics?.database.acquire_count}</span>
                    </div>
                    <div className='flex justify-between'>
                      <span className='text-sm'>Canceled</span>
                      <span className='font-mono font-semibold'>{metrics?.database.canceled_acquire_count}</span>
                    </div>
                    <div className='flex justify-between'>
                      <span className='text-sm'>Empty</span>
                      <span className='font-mono font-semibold'>{metrics?.database.empty_acquire_count}</span>
                    </div>
                    <div className='flex justify-between'>
                      <span className='text-sm'>Duration</span>
                      <span className='font-mono font-semibold'>{metrics?.database.acquire_duration_ms} ms</span>
                    </div>
                  </div>
                </div>

                <div className='space-y-2'>
                  <h4 className='text-sm font-medium text-muted-foreground'>Lifecycle</h4>
                  <div className='space-y-1'>
                    <div className='flex justify-between'>
                      <span className='text-sm'>New Conns</span>
                      <span className='font-mono font-semibold'>{metrics?.database.new_conns_count}</span>
                    </div>
                    <div className='flex justify-between'>
                      <span className='text-sm'>Max Lifetime Destroys</span>
                      <span className='font-mono font-semibold'>{metrics?.database.max_lifetime_destroy_count}</span>
                    </div>
                    <div className='flex justify-between'>
                      <span className='text-sm'>Max Idle Destroys</span>
                      <span className='font-mono font-semibold'>{metrics?.database.max_idle_destroy_count}</span>
                    </div>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Realtime Tab */}
        <TabsContent value='realtime' className='space-y-4'>
          <Card>
            <CardHeader>
              <CardTitle className='flex items-center gap-2'>
                <Zap className='h-5 w-5' />
                Realtime WebSocket
              </CardTitle>
              <CardDescription>Active WebSocket connections and subscriptions</CardDescription>
            </CardHeader>
            <CardContent>
              <div className='grid grid-cols-1 md:grid-cols-3 gap-6'>
                <div className='text-center'>
                  <div className='text-4xl font-bold'>{metrics?.realtime.total_connections || 0}</div>
                  <p className='text-sm text-muted-foreground mt-1'>Active Connections</p>
                </div>
                <div className='text-center'>
                  <div className='text-4xl font-bold'>{metrics?.realtime.active_channels || 0}</div>
                  <p className='text-sm text-muted-foreground mt-1'>Active Channels</p>
                </div>
                <div className='text-center'>
                  <div className='text-4xl font-bold'>{metrics?.realtime.total_subscriptions || 0}</div>
                  <p className='text-sm text-muted-foreground mt-1'>Total Subscriptions</p>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Storage Tab */}
        {metrics?.storage && (
          <TabsContent value='storage' className='space-y-4'>
            <Card>
              <CardHeader>
                <CardTitle className='flex items-center gap-2'>
                  <HardDrive className='h-5 w-5' />
                  Storage
                </CardTitle>
                <CardDescription>File storage usage statistics</CardDescription>
              </CardHeader>
              <CardContent>
                <div className='grid grid-cols-1 md:grid-cols-3 gap-6'>
                  <div className='text-center'>
                    <div className='text-4xl font-bold'>{metrics.storage.total_buckets}</div>
                    <p className='text-sm text-muted-foreground mt-1'>Buckets</p>
                  </div>
                  <div className='text-center'>
                    <div className='text-4xl font-bold'>{metrics.storage.total_files}</div>
                    <p className='text-sm text-muted-foreground mt-1'>Files</p>
                  </div>
                  <div className='text-center'>
                    <div className='text-4xl font-bold'>{metrics.storage.total_size_gb.toFixed(2)} GB</div>
                    <p className='text-sm text-muted-foreground mt-1'>Total Size</p>
                  </div>
                </div>
              </CardContent>
            </Card>
          </TabsContent>
        )}

        {/* Health Checks Tab */}
        <TabsContent value='health' className='space-y-4'>
          <Card>
            <CardHeader>
              <CardTitle>Health Checks</CardTitle>
              <CardDescription>System component health status and latency</CardDescription>
            </CardHeader>
            <CardContent>
              <div className='space-y-4'>
                {health?.services &&
                  Object.entries(health.services).map(([name, status]) => (
                    <div key={name} className='border rounded-lg p-4'>
                      <div className='flex items-center justify-between mb-2'>
                        <h4 className='font-medium capitalize'>{name}</h4>
                        {getStatusBadge(status.status)}
                      </div>
                      {status.message && <p className='text-sm text-muted-foreground mb-2'>{status.message}</p>}
                      {status.latency_ms !== undefined && (
                        <div className='text-xs text-muted-foreground'>
                          Response time: <span className='font-mono'>{status.latency_ms}ms</span>
                        </div>
                      )}
                    </div>
                  ))}
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}
