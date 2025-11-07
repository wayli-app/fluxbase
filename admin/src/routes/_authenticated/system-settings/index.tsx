import { createFileRoute } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Badge } from '@/components/ui/badge'
import { Database, Mail, HardDrive, Download, Settings2, CheckCircle2, AlertCircle } from 'lucide-react'
import { useState } from 'react'
import { monitoringApi } from '@/lib/api'

export const Route = createFileRoute('/_authenticated/system-settings/')({
  component: SystemSettingsPage,
})

interface DatabaseConfig {
  host: string
  port: number
  database: string
  max_connections: number
  min_connections: number
  max_lifetime_seconds: number
  max_idle_seconds: number
  stats: {
    total_conns: number
    idle_conns: number
    acquired_conns: number
    max_conns: number
  }
}

interface EmailConfig {
  provider: string
  smtp_host: string
  smtp_port: number
  smtp_username: string
  smtp_from: string
  smtp_from_name: string
}

interface StorageConfig {
  provider: string
  local_path?: string
  s3_endpoint?: string
  s3_bucket?: string
  max_upload_size_mb: number
}

function SystemSettingsPage() {
  // Mock data - In production, these would come from API endpoints
  const [dbConfig] = useState<DatabaseConfig>({
    host: 'postgres',
    port: 5432,
    database: 'fluxbase',
    max_connections: 100,
    min_connections: 10,
    max_lifetime_seconds: 3600,
    max_idle_seconds: 600,
    stats: {
      total_conns: 4,
      idle_conns: 3,
      acquired_conns: 1,
      max_conns: 100,
    },
  })

  const [emailConfig] = useState<EmailConfig>({
    provider: 'SMTP',
    smtp_host: 'mailhog',
    smtp_port: 1025,
    smtp_username: '',
    smtp_from: 'noreply@fluxbase.eu',
    smtp_from_name: 'Fluxbase',
  })

  const [storageConfig] = useState<StorageConfig>({
    provider: 'local',
    local_path: '/storage',
    max_upload_size_mb: 100,
  })

  // Fetch current config (placeholder)
  const { data: systemInfo } = useQuery({
    queryKey: ['system-info'],
    queryFn: monitoringApi.getMetrics,
    refetchInterval: 30000,
  })

  const getStorageProviderBadge = (provider: string) => {
    if (provider === 'local') {
      return <Badge variant='outline'>Local Filesystem</Badge>
    }
    if (provider === 's3') {
      return <Badge variant='outline'>S3 Compatible</Badge>
    }
    return <Badge variant='outline'>{provider}</Badge>
  }

  return (
    <div className='flex flex-col gap-6 p-6'>
      <div className='flex items-center justify-between'>
        <div>
          <h1 className='text-3xl font-bold'>System Settings</h1>
          <p className='text-sm text-muted-foreground mt-1'>Configure database, email, storage, and backup settings</p>
        </div>
        <Badge variant='outline' className='border-blue-500 text-blue-500'>
          <Settings2 className='mr-1 h-3 w-3' />
          Configuration
        </Badge>
      </div>

      <Tabs defaultValue='database' className='w-full'>
        <TabsList className='grid w-full grid-cols-4'>
          <TabsTrigger value='database'>Database</TabsTrigger>
          <TabsTrigger value='email'>Email</TabsTrigger>
          <TabsTrigger value='storage'>Storage</TabsTrigger>
          <TabsTrigger value='backup'>Backup</TabsTrigger>
        </TabsList>

        {/* Database Settings Tab */}
        <TabsContent value='database' className='space-y-4'>
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
        </TabsContent>

        {/* Email Configuration Tab */}
        <TabsContent value='email' className='space-y-4'>
          <Card>
            <CardHeader>
              <CardTitle className='flex items-center gap-2'>
                <Mail className='h-5 w-5' />
                Email Configuration
              </CardTitle>
              <CardDescription>SMTP settings for sending emails (magic links, notifications, etc.)</CardDescription>
            </CardHeader>
            <CardContent className='space-y-6'>
              {/* Email Provider */}
              <div className='space-y-4'>
                <h3 className='text-sm font-semibold'>Email Provider</h3>
                <div className='space-y-2'>
                  <Label>Provider</Label>
                  <Input value={emailConfig.provider} disabled />
                  <p className='text-xs text-muted-foreground'>Configured via EMAIL_PROVIDER environment variable (smtp, sendgrid, mailgun, ses)</p>
                </div>
              </div>

              {/* SMTP Settings */}
              <div className='space-y-4 pt-4 border-t'>
                <h3 className='text-sm font-semibold'>SMTP Settings (Read-only)</h3>
                <div className='grid grid-cols-1 md:grid-cols-2 gap-4'>
                  <div className='space-y-2'>
                    <Label>SMTP Host</Label>
                    <Input value={emailConfig.smtp_host} disabled />
                  </div>
                  <div className='space-y-2'>
                    <Label>SMTP Port</Label>
                    <Input value={emailConfig.smtp_port} disabled />
                  </div>
                  <div className='space-y-2'>
                    <Label>Username</Label>
                    <Input value={emailConfig.smtp_username || '(none)'} disabled />
                  </div>
                  <div className='space-y-2'>
                    <Label>From Email</Label>
                    <Input value={emailConfig.smtp_from} disabled />
                  </div>
                  <div className='space-y-2'>
                    <Label>From Name</Label>
                    <Input value={emailConfig.smtp_from_name} disabled />
                  </div>
                </div>
                <p className='text-xs text-muted-foreground'>
                  Email settings are configured via environment variables (SMTP_HOST, SMTP_PORT, SMTP_USERNAME, SMTP_PASSWORD, SMTP_FROM, SMTP_FROM_NAME)
                </p>
              </div>

              {/* Test Email */}
              <div className='space-y-4 pt-4 border-t'>
                <h3 className='text-sm font-semibold'>Test Email</h3>
                <div className='space-y-4'>
                  <div className='space-y-2'>
                    <Label>Recipient Email</Label>
                    <Input type='email' placeholder='test@example.com' />
                  </div>
                  <Button disabled>
                    <Mail className='mr-2 h-4 w-4' />
                    Send Test Email
                  </Button>
                  <p className='text-xs text-muted-foreground'>
                    Send a test email to verify your SMTP configuration is working correctly
                  </p>
                </div>
              </div>

              {/* Email Templates */}
              <div className='space-y-4 pt-4 border-t'>
                <h3 className='text-sm font-semibold'>Email Templates</h3>
                <p className='text-sm text-muted-foreground'>Email templates are managed in the codebase (internal/email/templates/)</p>
                <div className='space-y-2'>
                  <div className='flex items-center justify-between p-2 border rounded'>
                    <span className='text-sm'>Magic Link Email</span>
                    <Badge variant='outline'>Active</Badge>
                  </div>
                  <div className='flex items-center justify-between p-2 border rounded'>
                    <span className='text-sm'>Email Verification</span>
                    <Badge variant='outline'>Active</Badge>
                  </div>
                  <div className='flex items-center justify-between p-2 border rounded'>
                    <span className='text-sm'>Password Reset</span>
                    <Badge variant='outline'>Coming Soon</Badge>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Storage Configuration Tab */}
        <TabsContent value='storage' className='space-y-4'>
          <Card>
            <CardHeader>
              <CardTitle className='flex items-center gap-2'>
                <HardDrive className='h-5 w-5' />
                Storage Configuration
              </CardTitle>
              <CardDescription>File storage provider settings and upload limits</CardDescription>
            </CardHeader>
            <CardContent className='space-y-6'>
              {/* Storage Provider */}
              <div className='space-y-4'>
                <h3 className='text-sm font-semibold'>Storage Provider</h3>
                <div className='flex items-center gap-4'>
                  <div className='flex-1 space-y-2'>
                    <Label>Provider</Label>
                    <Input value={storageConfig.provider} disabled />
                  </div>
                  {getStorageProviderBadge(storageConfig.provider)}
                </div>
                <p className='text-xs text-muted-foreground'>
                  Storage provider is configured via STORAGE_PROVIDER environment variable (local, s3, minio)
                </p>
              </div>

              {/* Local Storage Settings */}
              {storageConfig.provider === 'local' && (
                <div className='space-y-4 pt-4 border-t'>
                  <h3 className='text-sm font-semibold'>Local Storage Settings</h3>
                  <div className='space-y-2'>
                    <Label>Storage Path</Label>
                    <Input value={storageConfig.local_path} disabled />
                    <p className='text-xs text-muted-foreground'>Directory where files are stored on the server filesystem</p>
                  </div>
                </div>
              )}

              {/* S3 Storage Settings */}
              {storageConfig.provider === 's3' && (
                <div className='space-y-4 pt-4 border-t'>
                  <h3 className='text-sm font-semibold'>S3 Storage Settings</h3>
                  <div className='grid grid-cols-1 md:grid-cols-2 gap-4'>
                    <div className='space-y-2'>
                      <Label>S3 Endpoint</Label>
                      <Input value={storageConfig.s3_endpoint} disabled />
                    </div>
                    <div className='space-y-2'>
                      <Label>Default Bucket</Label>
                      <Input value={storageConfig.s3_bucket} disabled />
                    </div>
                  </div>
                  <p className='text-xs text-muted-foreground'>
                    S3 settings are configured via environment variables (S3_ENDPOINT, S3_ACCESS_KEY, S3_SECRET_KEY, S3_BUCKET)
                  </p>
                </div>
              )}

              {/* Upload Limits */}
              <div className='space-y-4 pt-4 border-t'>
                <h3 className='text-sm font-semibold'>Upload Limits</h3>
                <div className='space-y-2'>
                  <Label>Max Upload Size</Label>
                  <Input type='number' value={storageConfig.max_upload_size_mb} disabled />
                  <p className='text-xs text-muted-foreground'>Maximum file size in MB (configured via MAX_UPLOAD_SIZE_MB)</p>
                </div>
              </div>

              {/* Storage Stats */}
              <div className='space-y-4 pt-4 border-t'>
                <h3 className='text-sm font-semibold'>Storage Statistics</h3>
                <div className='grid grid-cols-1 md:grid-cols-3 gap-4'>
                  <div className='border rounded-lg p-3'>
                    <div className='text-2xl font-bold'>{systemInfo?.storage?.total_buckets || 0}</div>
                    <p className='text-xs text-muted-foreground mt-1'>Buckets</p>
                  </div>
                  <div className='border rounded-lg p-3'>
                    <div className='text-2xl font-bold'>{systemInfo?.storage?.total_files || 0}</div>
                    <p className='text-xs text-muted-foreground mt-1'>Files</p>
                  </div>
                  <div className='border rounded-lg p-3'>
                    <div className='text-2xl font-bold'>{systemInfo?.storage?.total_size_gb?.toFixed(2) || '0.00'} GB</div>
                    <p className='text-xs text-muted-foreground mt-1'>Total Size</p>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Backup & Restore Tab */}
        <TabsContent value='backup' className='space-y-4'>
          <Card>
            <CardHeader>
              <CardTitle className='flex items-center gap-2'>
                <Download className='h-5 w-5' />
                Backup & Restore
              </CardTitle>
              <CardDescription>Database backup and restore operations</CardDescription>
            </CardHeader>
            <CardContent className='space-y-6'>
              {/* Manual Backup */}
              <div className='space-y-4'>
                <h3 className='text-sm font-semibold'>Manual Backup</h3>
                <p className='text-sm text-muted-foreground'>Create a full backup of your database using PostgreSQL's pg_dump utility</p>
                <div className='space-y-4'>
                  <div className='space-y-2'>
                    <Label>Backup Name</Label>
                    <Input placeholder='backup-2025-10-27' />
                  </div>
                  <Button disabled>
                    <Download className='mr-2 h-4 w-4' />
                    Create Backup
                  </Button>
                </div>
              </div>

              {/* CLI Instructions */}
              <div className='space-y-4 pt-4 border-t'>
                <h3 className='text-sm font-semibold'>Manual Backup via CLI</h3>
                <p className='text-sm text-muted-foreground'>You can create backups directly using PostgreSQL tools:</p>
                <div className='space-y-2'>
                  <div className='bg-muted rounded-lg p-3 space-y-1'>
                    <p className='text-xs text-muted-foreground'>Create backup:</p>
                    <code className='text-sm'>pg_dump -h postgres -U postgres -d fluxbase &gt; backup.sql</code>
                  </div>
                  <div className='bg-muted rounded-lg p-3 space-y-1'>
                    <p className='text-xs text-muted-foreground'>Restore backup:</p>
                    <code className='text-sm'>psql -h postgres -U postgres -d fluxbase &lt; backup.sql</code>
                  </div>
                </div>
              </div>

              {/* Automated Backups */}
              <div className='space-y-4 pt-4 border-t'>
                <h3 className='text-sm font-semibold'>Automated Backups</h3>
                <div className='flex items-center gap-2 p-3 border rounded-lg bg-muted'>
                  <AlertCircle className='h-5 w-5 text-yellow-500' />
                  <div>
                    <p className='text-sm font-medium'>Not Configured</p>
                    <p className='text-xs text-muted-foreground'>Automated backups can be configured using cron jobs or Kubernetes CronJobs</p>
                  </div>
                </div>
              </div>

              {/* Backup Best Practices */}
              <div className='space-y-4 pt-4 border-t'>
                <h3 className='text-sm font-semibold'>Best Practices</h3>
                <ul className='space-y-2 text-sm text-muted-foreground list-disc list-inside'>
                  <li>Schedule regular automated backups (daily or weekly)</li>
                  <li>Store backups in a different location than your database</li>
                  <li>Test your backup restoration process regularly</li>
                  <li>Keep multiple backup versions (retention policy)</li>
                  <li>Encrypt backups if they contain sensitive data</li>
                </ul>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}
