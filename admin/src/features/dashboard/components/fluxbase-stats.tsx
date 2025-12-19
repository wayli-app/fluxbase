import { useQuery } from '@tanstack/react-query'
import { useFluxbaseClient } from '@fluxbase/sdk-react'
import { Database, Users, Activity, Server } from 'lucide-react'
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'

export function FluxbaseStats() {
  const client = useFluxbaseClient()

  // Fetch health status using SDK
  const { data: health, isLoading: isLoadingHealth} = useQuery({
    queryKey: ['health'],
    queryFn: async () => {
      const { data, error } = await client.admin.getHealth()
      if (error) throw error
      return data
    },
    refetchInterval: 10000, // Refresh every 10 seconds
  })

  // Fetch table count using SDK (uses unique key to avoid conflict with other table queries)
  const { data: tables, isLoading: isLoadingTables } = useQuery({
    queryKey: ['dashboard', 'table-count'],
    queryFn: async () => {
      const { data, error } = await client.admin.ddl.listTables()
      if (error) throw error
      return data?.tables.map((t: { schema: string; name: string }) => `${t.schema}.${t.name}`) || []
    },
    refetchInterval: 30000, // Refresh every 30 seconds
  })

  // Fetch user count using SDK
  const { data: users } = useQuery({
    queryKey: ['users', 'count'],
    queryFn: async () => {
      try {
        const { data, error } = await client.admin.listUsers()
        if (error) return 0
        return data?.total || 0
      } catch {
        return 0
      }
    },
    refetchInterval: 30000,
  })

  return (
    <div className='grid gap-4 sm:grid-cols-2 lg:grid-cols-4'>
      {/* System Status */}
      <Card>
        <CardHeader className='flex flex-row items-center justify-between space-y-0 pb-2'>
          <CardTitle className='text-sm font-medium'>System Status</CardTitle>
          <Server className='text-muted-foreground h-4 w-4' />
        </CardHeader>
        <CardContent>
          {isLoadingHealth ? (
            <Skeleton className='h-8 w-20' />
          ) : (
            <>
              <div className='text-2xl font-bold'>
                {health?.status === 'ok' ? (
                  <span className='text-green-600 dark:text-green-400'>
                    Healthy
                  </span>
                ) : (
                  <span className='text-yellow-600 dark:text-yellow-400'>
                    Degraded
                  </span>
                )}
              </div>
              <p className='text-muted-foreground text-xs'>
                Database:{' '}
                {health?.services.database ? 'Connected' : 'Disconnected'}
              </p>
            </>
          )}
        </CardContent>
      </Card>

      {/* Total Users */}
      <Card>
        <CardHeader className='flex flex-row items-center justify-between space-y-0 pb-2'>
          <CardTitle className='text-sm font-medium'>Total Users</CardTitle>
          <Users className='text-muted-foreground h-4 w-4' />
        </CardHeader>
        <CardContent>
          <div className='text-2xl font-bold'>{users?.toLocaleString() || 0}</div>
          <p className='text-muted-foreground text-xs'>Registered accounts</p>
        </CardContent>
      </Card>

      {/* Database Tables */}
      <Card>
        <CardHeader className='flex flex-row items-center justify-between space-y-0 pb-2'>
          <CardTitle className='text-sm font-medium'>Database Tables</CardTitle>
          <Database className='text-muted-foreground h-4 w-4' />
        </CardHeader>
        <CardContent>
          {isLoadingTables ? (
            <Skeleton className='h-8 w-12' />
          ) : (
            <>
              <div className='text-2xl font-bold'>{tables?.length || 0}</div>
              <p className='text-muted-foreground text-xs'>
                Available for REST API
              </p>
            </>
          )}
        </CardContent>
      </Card>

      {/* API Status */}
      <Card>
        <CardHeader className='flex flex-row items-center justify-between space-y-0 pb-2'>
          <CardTitle className='text-sm font-medium'>API Status</CardTitle>
          <Activity className='text-muted-foreground h-4 w-4' />
        </CardHeader>
        <CardContent>
          {isLoadingHealth ? (
            <Skeleton className='h-8 w-20' />
          ) : (
            <>
              <div className='text-2xl font-bold'>
                {health?.status === 'ok' ? (
                  <span className='text-green-600 dark:text-green-400'>Live</span>
                ) : (
                  <span className='text-red-600 dark:text-red-400'>Down</span>
                )}
              </div>
              <p className='text-muted-foreground text-xs'>
                Realtime:{' '}
                {health?.services.realtime ? 'Enabled' : 'Disabled'}
              </p>
            </>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
