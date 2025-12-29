import { createFileRoute } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { HardDrive, ImageIcon, Info, Lock } from 'lucide-react'
import { useState } from 'react'
import api, { monitoringApi } from '@/lib/api'

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

interface TransformConfig {
  enabled: boolean
  default_quality: number
  max_width: number
  max_height: number
  allowed_formats?: string[]
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

  // Fetch transform config from storage config endpoint
  const { data: transformConfig, isLoading: isLoadingTransform } = useQuery<TransformConfig>({
    queryKey: ['storage-transform-config'],
    queryFn: async () => {
      const response = await api.get('/api/v1/storage/config/transforms')
      return response.data
    },
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

      {/* Image Transformations Card */}
      <Card>
        <CardHeader>
          <CardTitle className='flex items-center gap-2'>
            <ImageIcon className='h-5 w-5' />
            Image Transformations
          </CardTitle>
          <CardDescription>
            On-the-fly image resize, crop, and format conversion using libvips
          </CardDescription>
        </CardHeader>
        <CardContent className='space-y-6'>
          {isLoadingTransform ? (
            <div className='flex items-center justify-center py-8'>
              <div className='animate-spin rounded-full h-8 w-8 border-b-2 border-primary' />
            </div>
          ) : transformConfig?.enabled ? (
            <>
              {/* Status */}
              <div className='flex items-center gap-2'>
                <Badge variant='default' className='bg-green-600'>Enabled</Badge>
                <span className='text-sm text-muted-foreground flex items-center gap-1'>
                  <span title='Read-only (set via config)'><Lock className='h-3 w-3' /></span>
                  Configured via YAML or environment variables
                </span>
              </div>

              {/* Transform Settings */}
              <div className='space-y-4'>
                <h3 className='text-sm font-semibold'>Transformation Settings</h3>
                <div className='grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4'>
                  <div className='space-y-2'>
                    <Label>Default Quality</Label>
                    <Input value={transformConfig.default_quality} disabled />
                    <p className='text-xs text-muted-foreground'>1-100, used when not specified</p>
                  </div>
                  <div className='space-y-2'>
                    <Label>Max Width</Label>
                    <Input value={`${transformConfig.max_width}px`} disabled />
                    <p className='text-xs text-muted-foreground'>Maximum output width</p>
                  </div>
                  <div className='space-y-2'>
                    <Label>Max Height</Label>
                    <Input value={`${transformConfig.max_height}px`} disabled />
                    <p className='text-xs text-muted-foreground'>Maximum output height</p>
                  </div>
                  <div className='space-y-2'>
                    <Label>Supported Formats</Label>
                    <div className='flex flex-wrap gap-1'>
                      {['webp', 'jpg', 'png', 'avif'].map((fmt) => (
                        <Badge key={fmt} variant='secondary' className='text-xs'>{fmt.toUpperCase()}</Badge>
                      ))}
                    </div>
                    <p className='text-xs text-muted-foreground'>Output formats available</p>
                  </div>
                </div>
              </div>

              {/* Usage Examples */}
              <div className='space-y-4 pt-4 border-t'>
                <h3 className='text-sm font-semibold'>Usage Examples</h3>
                <div className='bg-muted rounded-lg p-4 space-y-3'>
                  <div>
                    <p className='text-xs font-medium text-muted-foreground mb-1'>Resize to 300x200 WebP:</p>
                    <code className='text-xs bg-background px-2 py-1 rounded block overflow-x-auto'>
                      GET /api/v1/storage/bucket/image.jpg?w=300&h=200&fmt=webp
                    </code>
                  </div>
                  <div>
                    <p className='text-xs font-medium text-muted-foreground mb-1'>Resize width only (maintain aspect ratio):</p>
                    <code className='text-xs bg-background px-2 py-1 rounded block overflow-x-auto'>
                      GET /api/v1/storage/bucket/image.jpg?w=800
                    </code>
                  </div>
                  <div>
                    <p className='text-xs font-medium text-muted-foreground mb-1'>With fit mode and quality:</p>
                    <code className='text-xs bg-background px-2 py-1 rounded block overflow-x-auto'>
                      GET /api/v1/storage/bucket/image.jpg?w=400&h=400&fit=cover&q=85
                    </code>
                  </div>
                </div>
              </div>

              {/* Fit Modes Reference */}
              <div className='space-y-4 pt-4 border-t'>
                <h3 className='text-sm font-semibold'>Fit Modes</h3>
                <div className='grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3'>
                  <div className='border rounded-lg p-3'>
                    <p className='font-medium text-sm'>cover</p>
                    <p className='text-xs text-muted-foreground'>Resize to cover target, cropping if needed (default)</p>
                  </div>
                  <div className='border rounded-lg p-3'>
                    <p className='font-medium text-sm'>contain</p>
                    <p className='text-xs text-muted-foreground'>Resize to fit within target, letterboxing if needed</p>
                  </div>
                  <div className='border rounded-lg p-3'>
                    <p className='font-medium text-sm'>fill</p>
                    <p className='text-xs text-muted-foreground'>Stretch to exactly fill target dimensions</p>
                  </div>
                  <div className='border rounded-lg p-3'>
                    <p className='font-medium text-sm'>inside</p>
                    <p className='text-xs text-muted-foreground'>Resize to fit within target, only scale down</p>
                  </div>
                  <div className='border rounded-lg p-3'>
                    <p className='font-medium text-sm'>outside</p>
                    <p className='text-xs text-muted-foreground'>Resize to be at least as large as target</p>
                  </div>
                </div>
              </div>
            </>
          ) : (
            <div className='flex flex-col items-center justify-center py-8 text-center'>
              <ImageIcon className='h-12 w-12 text-muted-foreground mb-4' />
              <p className='text-muted-foreground mb-2'>Image transformations are disabled</p>
              <div className='bg-muted rounded-lg p-4 text-left max-w-md'>
                <p className='text-sm font-medium mb-2 flex items-center gap-1'>
                  <Info className='h-4 w-4' /> To enable image transformations:
                </p>
                <ol className='text-xs text-muted-foreground space-y-1 list-decimal list-inside'>
                  <li>Install libvips on your server</li>
                  <li>Set <code className='bg-background px-1 rounded'>FLUXBASE_STORAGE_TRANSFORMS_ENABLED=true</code></li>
                  <li>Restart the Fluxbase server</li>
                </ol>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
