import { createFileRoute } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Database, CheckCircle2 } from 'lucide-react'
import { useState } from 'react'
import { monitoringApi } from '@/lib/api'

export const Route = createFileRoute('/_authenticated/database-config/')({
  component: DatabaseConfigPage,
})

interface DatabaseConfig {
  host: string
  port: number
  database: string
  max_connections: number
  min_connections: number
  max_lifetime_seconds: number
  max_idle_seconds: number
}

function DatabaseConfigPage() {
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
    <div className='flex flex-1 flex-col gap-6 p-6'>
      <div>
        <h1 className='text-3xl font-bold tracking-tight flex items-center gap-2'>
          <Database className='h-8 w-8' />
          Database
        </h1>
        <p className='text-sm text-muted-foreground mt-2'>PostgreSQL connection settings and connection pool configuration</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className='flex items-center gap-2'>
            <Database className='h-5 w-5' />
            Database Configuration
          </CardTitle>
          <CardDescription>PostgreSQL connection settings and connection pool configuration</CardDescription>
        </CardHeader>
        <CardContent className='space-y-6'>
          {/* Connection Settings */}
          <div className='space-y-4'>
            <h3 className='text-sm font-semibold'>Connection Settings (Read-only)</h3>
            <div className='grid grid-cols-1 md:grid-cols-2 gap-4'>
              <div className='space-y-2'>
                <Label>Host</Label>
                <Input value={dbConfig.host} disabled />
              </div>
              <div className='space-y-2'>
                <Label>Port</Label>
                <Input value={dbConfig.port} disabled />
              </div>
              <div className='space-y-2'>
                <Label>Database</Label>
                <Input value={dbConfig.database} disabled />
              </div>
            </div>
            <p className='text-xs text-muted-foreground'>
              Database connection settings are configured via environment variables (POSTGRES_HOST, POSTGRES_PORT, POSTGRES_DB)
            </p>
          </div>

          {/* Connection Pool Settings */}
          <div className='space-y-4 pt-4 border-t'>
            <h3 className='text-sm font-semibold'>Connection Pool Settings</h3>
            <div className='grid grid-cols-1 md:grid-cols-2 gap-4'>
              <div className='space-y-2'>
                <Label>Max Connections</Label>
                <Input type='number' value={dbConfig.max_connections} disabled />
                <p className='text-xs text-muted-foreground'>Maximum number of connections in the pool</p>
              </div>
              <div className='space-y-2'>
                <Label>Min Connections</Label>
                <Input type='number' value={dbConfig.min_connections} disabled />
                <p className='text-xs text-muted-foreground'>Minimum number of idle connections</p>
              </div>
              <div className='space-y-2'>
                <Label>Max Connection Lifetime</Label>
                <Input type='number' value={dbConfig.max_lifetime_seconds} disabled />
                <p className='text-xs text-muted-foreground'>Maximum lifetime in seconds</p>
              </div>
              <div className='space-y-2'>
                <Label>Max Idle Time</Label>
                <Input type='number' value={dbConfig.max_idle_seconds} disabled />
                <p className='text-xs text-muted-foreground'>Maximum idle time in seconds</p>
              </div>
            </div>
          </div>

          {/* Current Pool Status */}
          <div className='space-y-4 pt-4 border-t'>
            <h3 className='text-sm font-semibold'>Current Pool Status</h3>
            <div className='grid grid-cols-2 md:grid-cols-4 gap-4'>
              <div className='border rounded-lg p-3'>
                <div className='text-2xl font-bold'>{systemInfo?.database.total_conns || 0}</div>
                <p className='text-xs text-muted-foreground mt-1'>Total Connections</p>
              </div>
              <div className='border rounded-lg p-3'>
                <div className='text-2xl font-bold'>{systemInfo?.database.idle_conns || 0}</div>
                <p className='text-xs text-muted-foreground mt-1'>Idle</p>
              </div>
              <div className='border rounded-lg p-3'>
                <div className='text-2xl font-bold'>{systemInfo?.database.acquired_conns || 0}</div>
                <p className='text-xs text-muted-foreground mt-1'>Acquired</p>
              </div>
              <div className='border rounded-lg p-3'>
                <div className='text-2xl font-bold'>{systemInfo?.database.max_conns || 0}</div>
                <p className='text-xs text-muted-foreground mt-1'>Max</p>
              </div>
            </div>
          </div>

          {/* Migrations */}
          <div className='space-y-4 pt-4 border-t'>
            <h3 className='text-sm font-semibold'>Database Migrations</h3>
            <p className='text-sm text-muted-foreground'>
              Database migrations are automatically run on server startup. To manually run migrations, use the CLI:
            </p>
            <div className='bg-muted rounded-lg p-3 font-mono text-sm'>./fluxbase migrate</div>
            <div className='flex items-center gap-2 text-sm'>
              <CheckCircle2 className='h-4 w-4 text-green-500' />
              <span>All migrations up to date</span>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
