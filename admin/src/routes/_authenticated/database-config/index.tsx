import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import { Database } from 'lucide-react'
import { monitoringApi } from '@/lib/api'
import { requireInstanceAdmin } from '@/lib/route-guards'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

interface DatabaseConfig {
  host: string
  port: number
  database: string
  max_connections: number
  min_connections: number
  max_lifetime_seconds: number
  max_idle_seconds: number
}

const DatabaseConfigPage = () => {
  const [dbConfig] = useState<DatabaseConfig>({
    host: 'postgres',
    port: 5432,
    database: 'fluxbase',
    max_connections: 100,
    min_connections: 10,
    max_lifetime_seconds: 3600,
    max_idle_seconds: 600,
  })

  const { data: systemInfo } = useQuery({
    queryKey: ['system-info'],
    queryFn: monitoringApi.getMetrics,
    refetchInterval: 30000,
  })

  return (
    <div className='flex h-full flex-col'>
      {/* Header */}
      <div className='flex-1 overflow-auto p-6'>
        <Card>
          <CardHeader>
            <CardTitle className='flex items-center gap-2'>
              <Database className='h-5 w-5' />
              Database Configuration
            </CardTitle>
            <CardDescription>
              PostgreSQL connection settings and connection pool configuration
            </CardDescription>
          </CardHeader>
          <CardContent className='space-y-6'>
            {/* Connection Settings */}
            <div className='space-y-4'>
              <h3 className='text-sm font-semibold'>
                Connection Settings (Read-only)
              </h3>
              <div className='grid grid-cols-1 gap-4 md:grid-cols-2'>
                <div className='space-y-2'>
                  <Label>Host</Label>
                  <Input value={dbConfig.host} readOnly />
                </div>
                <div className='space-y-2'>
                  <Label>Port</Label>
                  <Input value={dbConfig.port} readOnly />
                </div>
                <div className='space-y-2'>
                  <Label>Database</Label>
                  <Input value={dbConfig.database} readOnly />
                </div>
              </div>
              <p className='text-muted-foreground text-xs'>
                Database connection settings are configured via environment
                variables (POSTGRES_HOST, POSTGRES_PORT, POSTGRES_DB)
              </p>
            </div>

            {/* Connection Pool Settings */}
            <div className='space-y-4 border-t pt-4'>
              <h3 className='text-sm font-semibold'>
                Connection Pool Settings
              </h3>
              <div className='grid grid-cols-1 gap-4 md:grid-cols-2'>
                <div className='space-y-2'>
                  <Label>Max Connections</Label>
                  <Input
                    type='number'
                    value={dbConfig.max_connections}
                    readOnly
                  />
                  <p className='text-muted-foreground text-xs'>
                    Maximum number of connections in the pool
                  </p>
                </div>
                <div className='space-y-2'>
                  <Label>Min Connections</Label>
                  <Input
                    type='number'
                    value={dbConfig.min_connections}
                    readOnly
                  />
                  <p className='text-muted-foreground text-xs'>
                    Minimum number of idle connections
                  </p>
                </div>
                <div className='space-y-2'>
                  <Label>Max Connection Lifetime</Label>
                  <Input
                    type='number'
                    value={dbConfig.max_lifetime_seconds}
                    readOnly
                  />
                  <p className='text-muted-foreground text-xs'>
                    Maximum lifetime in seconds
                  </p>
                </div>
                <div className='space-y-2'>
                  <Label>Max Idle Time</Label>
                  <Input
                    type='number'
                    value={dbConfig.max_idle_seconds}
                    readOnly
                  />
                  <p className='text-muted-foreground text-xs'>
                    Maximum idle time in seconds
                  </p>
                </div>
              </div>
            </div>

            {/* Current Pool Status */}
            <div className='space-y-4 border-t pt-4'>
              <h3 className='text-sm font-semibold'>Current Pool Status</h3>
              <div className='grid grid-cols-2 gap-4 md:grid-cols-4'>
                <div className='rounded-lg border p-3'>
                  <div className='text-2xl font-bold'>
                    {systemInfo?.database.total_conns || 0}
                  </div>
                  <p className='text-muted-foreground mt-1 text-xs'>
                    Total Connections
                  </p>
                </div>
                <div className='rounded-lg border p-3'>
                  <div className='text-2xl font-bold'>
                    {systemInfo?.database.idle_conns || 0}
                  </div>
                  <p className='text-muted-foreground mt-1 text-xs'>Idle</p>
                </div>
                <div className='rounded-lg border p-3'>
                  <div className='text-2xl font-bold'>
                    {systemInfo?.database.acquired_conns || 0}
                  </div>
                  <p className='text-muted-foreground mt-1 text-xs'>Acquired</p>
                </div>
                <div className='rounded-lg border p-3'>
                  <div className='text-2xl font-bold'>
                    {systemInfo?.database.max_conns || 0}
                  </div>
                  <p className='text-muted-foreground mt-1 text-xs'>Max</p>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

export const Route = createFileRoute('/_authenticated/database-config/')({
  beforeLoad: () => {
    requireInstanceAdmin()
  },
  component: DatabaseConfigPage,
})
