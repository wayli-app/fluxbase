import z from 'zod'
import { createFileRoute, getRouteApi } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Activity, Database, HardDrive, Zap, Cpu, MemoryStick, Network, CheckCircle2, AlertCircle, XCircle, Bot, MessageSquare, ChevronLeft, ChevronRight } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useState } from 'react'
import { monitoringApi, type SystemMetrics, type SystemHealth, aiMetricsApi, type AIMetrics, conversationsApi } from '@/lib/api'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

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

  // Pagination state for conversations
  const [convPage, setConvPage] = useState(0)
  const [convPageSize, setConvPageSize] = useState(25)


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

  // Fetch AI metrics
  const { data: aiMetrics } = useQuery<AIMetrics>({
    queryKey: ['ai-metrics'],
    queryFn: aiMetricsApi.getMetrics,
    refetchInterval: autoRefresh ? 10000 : false,
  })

  // Fetch conversations
  const { data: conversationsData } = useQuery({
    queryKey: ['ai-conversations', convPage, convPageSize],
    queryFn: () => conversationsApi.list({ limit: convPageSize, offset: convPage * convPageSize }),
    refetchInterval: autoRefresh ? 15000 : false,
  })
  const conversations = conversationsData?.conversations || []
  const convTotalCount = conversationsData?.total_count || 0
  const convTotalPages = Math.ceil(convTotalCount / convPageSize)

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
    <div className='flex flex-1 flex-col gap-6 p-6'>
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
          <TabsTrigger value='ai'>AI Chatbots</TabsTrigger>
          <TabsTrigger value='conversations'>Conversations</TabsTrigger>
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

        {/* AI Chatbots Tab */}
        <TabsContent value='ai' className='space-y-4'>
          <div className='grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4'>
            <Card>
              <CardHeader className='pb-2'>
                <CardTitle className='text-sm font-medium text-muted-foreground'>Total Requests</CardTitle>
              </CardHeader>
              <CardContent>
                <div className='text-2xl font-bold'>{aiMetrics?.total_requests?.toLocaleString() || 0}</div>
                <p className='text-xs text-muted-foreground mt-1'>AI chat requests</p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className='pb-2'>
                <CardTitle className='text-sm font-medium text-muted-foreground'>Total Tokens</CardTitle>
              </CardHeader>
              <CardContent>
                <div className='text-2xl font-bold'>{aiMetrics?.total_tokens?.toLocaleString() || 0}</div>
                <p className='text-xs text-muted-foreground mt-1'>
                  {aiMetrics?.total_prompt_tokens?.toLocaleString() || 0} prompt + {aiMetrics?.total_completion_tokens?.toLocaleString() || 0} completion
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className='pb-2'>
                <CardTitle className='text-sm font-medium text-muted-foreground'>Active Conversations</CardTitle>
              </CardHeader>
              <CardContent>
                <div className='text-2xl font-bold'>{aiMetrics?.active_conversations || 0}</div>
                <p className='text-xs text-muted-foreground mt-1'>
                  of {aiMetrics?.total_conversations || 0} total
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className='pb-2'>
                <CardTitle className='text-sm font-medium text-muted-foreground'>Error Rate</CardTitle>
              </CardHeader>
              <CardContent>
                <div className='text-2xl font-bold'>
                  {aiMetrics?.error_rate ? aiMetrics.error_rate.toFixed(2) : '0.00'}%
                </div>
                <p className='text-xs text-muted-foreground mt-1'>
                  Avg: {aiMetrics?.avg_response_time_ms ? aiMetrics.avg_response_time_ms.toFixed(0) : '0'}ms
                </p>
              </CardContent>
            </Card>
          </div>

          {/* Chatbot Breakdown */}
          {aiMetrics?.chatbot_stats && aiMetrics.chatbot_stats.length > 0 && (
            <Card>
              <CardHeader>
                <CardTitle className='flex items-center gap-2'>
                  <Bot className='h-5 w-5' />
                  Chatbot Usage Breakdown
                </CardTitle>
                <CardDescription>Request counts, token usage, and errors by chatbot</CardDescription>
              </CardHeader>
              <CardContent>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Chatbot</TableHead>
                      <TableHead className='text-right'>Requests</TableHead>
                      <TableHead className='text-right'>Tokens</TableHead>
                      <TableHead className='text-right'>Errors</TableHead>
                      <TableHead className='text-right'>Error Rate</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {aiMetrics.chatbot_stats.map((stat) => {
                      const errorRate = stat.requests > 0 ? (stat.error_count / stat.requests) * 100 : 0
                      return (
                        <TableRow key={stat.chatbot_id}>
                          <TableCell className='font-medium'>{stat.chatbot_name}</TableCell>
                          <TableCell className='text-right font-mono'>{stat.requests.toLocaleString()}</TableCell>
                          <TableCell className='text-right font-mono'>{stat.tokens.toLocaleString()}</TableCell>
                          <TableCell className='text-right font-mono'>
                            {stat.error_count > 0 ? (
                              <span className='text-destructive'>{stat.error_count}</span>
                            ) : (
                              stat.error_count
                            )}
                          </TableCell>
                          <TableCell className='text-right font-mono'>
                            {errorRate > 0 ? (
                              <Badge variant={errorRate > 5 ? 'destructive' : 'secondary'}>
                                {errorRate.toFixed(2)}%
                              </Badge>
                            ) : (
                              <span className='text-muted-foreground'>0.00%</span>
                            )}
                          </TableCell>
                        </TableRow>
                      )
                    })}
                  </TableBody>
                </Table>
              </CardContent>
            </Card>
          )}

          {/* No Data State */}
          {(!aiMetrics?.chatbot_stats || aiMetrics.chatbot_stats.length === 0) && (
            <Card>
              <CardContent className='p-8 text-center'>
                <Bot className='h-12 w-12 mx-auto mb-4 text-muted-foreground' />
                <p className='text-lg font-medium mb-1'>No AI activity yet</p>
                <p className='text-sm text-muted-foreground'>
                  AI chatbot metrics will appear here once they start receiving requests
                </p>
              </CardContent>
            </Card>
          )}
        </TabsContent>

        {/* Conversations Tab */}
        <TabsContent value='conversations' className='space-y-4'>
          <Card>
            <CardHeader>
              <CardTitle className='flex items-center gap-2'>
                <MessageSquare className='h-5 w-5' />
                Conversation History
              </CardTitle>
              <CardDescription>Active and past AI chatbot conversations</CardDescription>
            </CardHeader>
            <CardContent>
              {conversations && conversations.length > 0 ? (
                <>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Conversation ID</TableHead>
                      <TableHead>Chatbot</TableHead>
                      <TableHead>User</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead className='text-right'>Messages</TableHead>
                      <TableHead className='text-right'>Tokens</TableHead>
                      <TableHead className='text-right'>Started</TableHead>
                      <TableHead className='text-right'>Last Activity</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {conversations.map((conv) => (
                      <TableRow key={conv.id}>
                        <TableCell className='font-mono text-xs'>{conv.id.substring(0, 8)}...</TableCell>
                        <TableCell className='font-medium'>{conv.chatbot_name}</TableCell>
                        <TableCell className='text-sm text-muted-foreground'>{conv.user_id ? conv.user_id.substring(0, 8) + '...' : 'Anonymous'}</TableCell>
                        <TableCell>
                          <Badge variant={conv.status === 'active' ? 'default' : 'secondary'}>
                            {conv.status}
                          </Badge>
                        </TableCell>
                        <TableCell className='text-right font-mono'>{conv.turn_count}</TableCell>
                        <TableCell className='text-right font-mono'>
                          {(conv.total_prompt_tokens + conv.total_completion_tokens).toLocaleString()}
                          <span className='text-xs text-muted-foreground ml-1'>
                            ({conv.total_prompt_tokens}+{conv.total_completion_tokens})
                          </span>
                        </TableCell>
                        <TableCell className='text-right text-sm'>
                          {new Date(conv.created_at).toLocaleDateString()} {new Date(conv.created_at).toLocaleTimeString()}
                        </TableCell>
                        <TableCell className='text-right text-sm'>
                          {new Date(conv.updated_at).toLocaleDateString()} {new Date(conv.updated_at).toLocaleTimeString()}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
                {/* Pagination Controls */}
                <div className='flex items-center justify-between px-2 py-4'>
                  <div className='flex items-center gap-2'>
                    <span className='text-sm text-muted-foreground'>Rows per page</span>
                    <Select
                      value={`${convPageSize}`}
                      onValueChange={(value) => {
                        setConvPageSize(Number(value))
                        setConvPage(0)
                      }}
                    >
                      <SelectTrigger className='h-8 w-[70px]'>
                        <SelectValue placeholder={convPageSize} />
                      </SelectTrigger>
                      <SelectContent side='top'>
                        {[10, 25, 50, 100].map((pageSize) => (
                          <SelectItem key={pageSize} value={`${pageSize}`}>
                            {pageSize}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                  <div className='flex items-center gap-2'>
                    <span className='text-sm text-muted-foreground'>
                      Page {convPage + 1} of {convTotalPages || 1} ({convTotalCount} total)
                    </span>
                    <Button
                      variant='outline'
                      size='sm'
                      onClick={() => setConvPage((p) => Math.max(0, p - 1))}
                      disabled={convPage === 0}
                    >
                      <ChevronLeft className='h-4 w-4' />
                    </Button>
                    <Button
                      variant='outline'
                      size='sm'
                      onClick={() => setConvPage((p) => Math.min(convTotalPages - 1, p + 1))}
                      disabled={convPage >= convTotalPages - 1}
                    >
                      <ChevronRight className='h-4 w-4' />
                    </Button>
                  </div>
                </div>
                </>
              ) : (
                <div className='p-8 text-center'>
                  <MessageSquare className='h-12 w-12 mx-auto mb-4 text-muted-foreground' />
                  <p className='text-lg font-medium mb-1'>No conversations yet</p>
                  <p className='text-sm text-muted-foreground'>
                    Conversations will appear here once users start chatting with AI chatbots
                  </p>
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

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
