import { createFileRoute } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { HardDrive } from 'lucide-react'
import { useState } from 'react'
import { monitoringApi } from '@/lib/api'

export const Route = createFileRoute('/_authenticated/storage-config/')({
  component: StorageConfigPage,
})

interface StorageConfig {
  provider: string
  local_path?: string
  s3_endpoint?: string
  s3_bucket?: string
  max_upload_size_mb: number
}

function StorageConfigPage() {
  const [storageConfig] = useState<StorageConfig>({
    provider: 'local',
    local_path: '/storage',
    max_upload_size_mb: 100,
  })

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
    <div className='flex flex-1 flex-col gap-6 p-6'>
      <div>
        <h1 className='text-3xl font-bold tracking-tight flex items-center gap-2'>
          <HardDrive className='h-8 w-8' />
          Storage
        </h1>
        <p className='text-sm text-muted-foreground mt-2'>File storage provider settings and upload limits</p>
      </div>

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
    </div>
  )
}
