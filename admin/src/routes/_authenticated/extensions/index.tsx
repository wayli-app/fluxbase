import { createFileRoute } from '@tanstack/react-router'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Switch } from '@/components/ui/switch'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Puzzle, RefreshCw, Loader2, AlertCircle, CheckCircle2, Info } from 'lucide-react'
import { apiClient } from '@/lib/api'
import { toast } from 'sonner'
import { cn } from '@/lib/utils'

export const Route = createFileRoute('/_authenticated/extensions/')({
  component: ExtensionsPage,
})

interface Extension {
  id: string
  name: string
  display_name: string
  description?: string
  category: string
  is_core: boolean
  requires_restart: boolean
  documentation_url?: string
  is_enabled: boolean
  is_installed: boolean
  installed_version?: string
  enabled_at?: string
  enabled_by?: string
}

interface Category {
  id: string
  name: string
  count: number
}

interface ListExtensionsResponse {
  extensions: Extension[]
  categories: Category[]
}

interface EnableDisableResponse {
  name: string
  success: boolean
  message: string
  version?: string
}

const categoryDisplayNames: Record<string, string> = {
  core: 'Core',
  geospatial: 'Geospatial',
  ai_ml: 'AI & Machine Learning',
  monitoring: 'Monitoring',
  scheduling: 'Scheduling',
  data_types: 'Data Types',
  text_search: 'Text Search',
  indexing: 'Indexing',
  networking: 'Networking',
  testing: 'Testing',
}

const categoryOrder = [
  'core',
  'ai_ml',
  'geospatial',
  'monitoring',
  'scheduling',
  'data_types',
  'indexing',
  'networking',
  'testing',
]

function ExtensionsPage() {
  const queryClient = useQueryClient()

  const { data, isLoading, error } = useQuery<ListExtensionsResponse>({
    queryKey: ['extensions'],
    queryFn: async () => {
      const response = await apiClient.get<ListExtensionsResponse>('/api/v1/admin/extensions')
      return response.data
    },
  })

  const enableMutation = useMutation({
    mutationFn: async (name: string) => {
      const response = await apiClient.post<EnableDisableResponse>(
        `/api/v1/admin/extensions/${name}/enable`
      )
      return response.data
    },
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['extensions'] })
      if (data.success) {
        toast.success(`Extension "${data.name}" enabled successfully`)
      } else {
        toast.error(data.message || 'Failed to enable extension')
      }
    },
    onError: (error: unknown) => {
      const axiosError = error as { response?: { data?: EnableDisableResponse } }
      const message = axiosError.response?.data?.message || 'Failed to enable extension'
      toast.error(message)
    },
  })

  const disableMutation = useMutation({
    mutationFn: async (name: string) => {
      const response = await apiClient.post<EnableDisableResponse>(
        `/api/v1/admin/extensions/${name}/disable`
      )
      return response.data
    },
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['extensions'] })
      if (data.success) {
        toast.success(`Extension "${data.name}" disabled successfully`)
      } else {
        toast.error(data.message || 'Failed to disable extension')
      }
    },
    onError: (error: unknown) => {
      const axiosError = error as { response?: { data?: EnableDisableResponse } }
      const message = axiosError.response?.data?.message || 'Failed to disable extension'
      toast.error(message)
    },
  })

  const syncMutation = useMutation({
    mutationFn: async () => {
      await apiClient.post('/api/v1/admin/extensions/sync')
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['extensions'] })
      toast.success('Extensions synced successfully')
    },
    onError: () => {
      toast.error('Failed to sync extensions')
    },
  })

  const handleToggle = (extension: Extension) => {
    if (extension.is_core) return // Core extensions cannot be toggled

    if (extension.is_enabled) {
      disableMutation.mutate(extension.name)
    } else {
      enableMutation.mutate(extension.name)
    }
  }

  const isPending = enableMutation.isPending || disableMutation.isPending

  // Group extensions by category
  const extensionsByCategory = data?.extensions.reduce(
    (acc, ext) => {
      if (!acc[ext.category]) {
        acc[ext.category] = []
      }
      acc[ext.category].push(ext)
      return acc
    },
    {} as Record<string, Extension[]>
  )

  // Sort categories by defined order
  const sortedCategories = Object.keys(extensionsByCategory || {}).sort((a, b) => {
    const aIndex = categoryOrder.indexOf(a)
    const bIndex = categoryOrder.indexOf(b)
    if (aIndex === -1 && bIndex === -1) return a.localeCompare(b)
    if (aIndex === -1) return 1
    if (bIndex === -1) return -1
    return aIndex - bIndex
  })

  if (error) {
    return (
      <div className='flex flex-1 flex-col gap-6 p-6'>
        <div className='flex items-center gap-2 text-destructive'>
          <AlertCircle className='h-5 w-5' />
          <span>Failed to load extensions</span>
        </div>
      </div>
    )
  }

  return (
    <div className='flex flex-1 flex-col gap-6 p-6'>
      <div className='flex items-center justify-between'>
        <div>
          <h1 className='text-3xl font-bold tracking-tight flex items-center gap-2'>
            <Puzzle className='h-8 w-8' />
            Extensions
          </h1>
          <p className='text-sm text-muted-foreground mt-2'>
            Manage PostgreSQL extensions for your database
          </p>
        </div>
        <Button
          variant='outline'
          size='sm'
          onClick={() => syncMutation.mutate()}
          disabled={syncMutation.isPending}
        >
          {syncMutation.isPending ? (
            <Loader2 className='h-4 w-4 mr-2 animate-spin' />
          ) : (
            <RefreshCw className='h-4 w-4 mr-2' />
          )}
          Sync from Database
        </Button>
      </div>

      {isLoading ? (
        <div className='flex justify-center py-12'>
          <Loader2 className='h-8 w-8 animate-spin text-muted-foreground' />
        </div>
      ) : (
        <div className='space-y-6'>
          {sortedCategories.map((category) => (
            <Card key={category}>
              <CardHeader>
                <CardTitle className='text-lg'>
                  {categoryDisplayNames[category] || category}
                </CardTitle>
                <CardDescription>
                  {category === 'core' &&
                    'Essential extensions required for Fluxbase to function. These cannot be disabled.'}
                  {category === 'ai_ml' &&
                    'Extensions for AI/ML workloads including vector similarity search.'}
                  {category === 'geospatial' &&
                    'Extensions for working with geographic and spatial data.'}
                  {category === 'monitoring' &&
                    'Extensions for monitoring and analyzing database performance.'}
                  {category === 'scheduling' && 'Extensions for scheduling jobs within PostgreSQL.'}
                  {category === 'data_types' && 'Extensions that add additional data types.'}
                  {category === 'indexing' && 'Extensions for advanced indexing capabilities.'}
                  {category === 'networking' &&
                    'Extensions for network operations from within PostgreSQL.'}
                  {category === 'testing' && 'Extensions for database testing and validation.'}
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className='space-y-4'>
                  {extensionsByCategory?.[category]?.map((extension) => (
                    <div
                      key={extension.id}
                      className={cn(
                        'flex items-start justify-between p-4 rounded-lg border',
                        extension.is_enabled && 'bg-muted/30'
                      )}
                    >
                      <div className='space-y-1 flex-1'>
                        <div className='flex items-center gap-2'>
                          <span className='font-medium'>{extension.display_name}</span>
                          <code className='text-xs bg-muted px-1.5 py-0.5 rounded'>
                            {extension.name}
                          </code>
                          {extension.is_core && (
                            <Badge variant='secondary' className='text-xs'>
                              Core
                            </Badge>
                          )}
                          {extension.requires_restart && !extension.is_core && (
                            <Badge variant='outline' className='text-xs text-orange-600'>
                              Requires Restart
                            </Badge>
                          )}
                        </div>
                        {extension.description && (
                          <p className='text-sm text-muted-foreground'>{extension.description}</p>
                        )}
                        <div className='flex items-center gap-3 text-xs text-muted-foreground pt-1'>
                          {extension.is_enabled && extension.installed_version && (
                            <span className='flex items-center gap-1'>
                              <CheckCircle2 className='h-3 w-3 text-green-500' />
                              v{extension.installed_version}
                            </span>
                          )}
                          {extension.is_installed && !extension.is_enabled && (
                            <span className='flex items-center gap-1'>
                              <Info className='h-3 w-3' />
                              Available to enable
                            </span>
                          )}
                        </div>
                      </div>
                      <div className='flex items-center gap-2'>
                        <Switch
                          checked={extension.is_enabled}
                          onCheckedChange={() => handleToggle(extension)}
                          disabled={extension.is_core || isPending}
                          aria-label={`Toggle ${extension.display_name}`}
                        />
                      </div>
                    </div>
                  ))}
                </div>
              </CardContent>
            </Card>
          ))}

          <div className='rounded-lg bg-muted p-4'>
            <div className='flex gap-2'>
              <AlertCircle className='h-5 w-5 text-muted-foreground shrink-0 mt-0.5' />
              <div className='text-sm space-y-1'>
                <p className='font-medium'>Extension Management</p>
                <p className='text-muted-foreground'>
                  Extensions are installed into the PostgreSQL database. Some extensions may require
                  a database restart to take effect. Core extensions are required for Fluxbase
                  functionality and cannot be disabled.
                </p>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
