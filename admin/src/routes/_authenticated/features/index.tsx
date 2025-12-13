import { createFileRoute } from '@tanstack/react-router'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Zap, AlertCircle, Loader2 } from 'lucide-react'
import { apiClient } from '@/lib/api'
import { toast } from 'sonner'
import { OverridableSwitch } from '@/components/admin/overridable-switch'

export const Route = createFileRoute('/_authenticated/features/')({
  component: FeaturesPage,
})

interface SystemSetting {
  id: string
  key: string
  value: {
    value: boolean
  }
  description?: string
  is_overridden: boolean
  override_source?: string
  created_at: string
  updated_at: string
}

interface FeatureSettings {
  enable_realtime: boolean
  enable_storage: boolean
  enable_functions: boolean
  enable_ai: boolean
  _overrides?: {
    enable_realtime?: { is_overridden: boolean; env_var: string }
    enable_storage?: { is_overridden: boolean; env_var: string }
    enable_functions?: { is_overridden: boolean; env_var: string }
    enable_ai?: { is_overridden: boolean; env_var: string }
  }
}

function FeaturesPage() {
  const queryClient = useQueryClient()

  const { data: features, isLoading } = useQuery<FeatureSettings>({
    queryKey: ['feature-settings'],
    queryFn: async () => {
      const [realtime, storage, functions, ai] = await Promise.all([
        apiClient.get<SystemSetting>('/api/v1/admin/system/settings/app.features.enable_realtime'),
        apiClient.get<SystemSetting>('/api/v1/admin/system/settings/app.features.enable_storage'),
        apiClient.get<SystemSetting>('/api/v1/admin/system/settings/app.features.enable_functions'),
        apiClient.get<SystemSetting>('/api/v1/admin/system/settings/app.features.enable_ai'),
      ])
      return {
        enable_realtime: realtime.data.value.value,
        enable_storage: storage.data.value.value,
        enable_functions: functions.data.value.value,
        enable_ai: ai.data.value.value,
        _overrides: {
          enable_realtime: realtime.data.is_overridden ? {
            is_overridden: true,
            env_var: realtime.data.override_source || '',
          } : undefined,
          enable_storage: storage.data.is_overridden ? {
            is_overridden: true,
            env_var: storage.data.override_source || '',
          } : undefined,
          enable_functions: functions.data.is_overridden ? {
            is_overridden: true,
            env_var: functions.data.override_source || '',
          } : undefined,
          enable_ai: ai.data.is_overridden ? {
            is_overridden: true,
            env_var: ai.data.override_source || '',
          } : undefined,
        },
      }
    },
  })

  const updateFeatureMutation = useMutation({
    mutationFn: async ({ key, value }: { key: string; value: boolean }) => {
      await apiClient.put(`/api/v1/admin/system/settings/${key}`, {
        value: { value },
      })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['feature-settings'] })
      toast.success('Feature settings updated')
    },
    onError: (error: unknown) => {
      if (error && typeof error === 'object' && 'response' in error) {
        const err = error as { response?: { status?: number; data?: { code?: string } } }
        if (err.response?.status === 409 && err.response?.data?.code === 'ENV_OVERRIDE') {
          toast.error('This setting is controlled by an environment variable and cannot be changed')
          return
        }
      }
      toast.error('Failed to update feature settings')
    },
  })

  return (
    <div className='flex flex-1 flex-col gap-6 p-6'>
      <div>
        <h1 className='text-3xl font-bold tracking-tight flex items-center gap-2'>
          <Zap className='h-8 w-8' />
          Features
        </h1>
        <p className='text-sm text-muted-foreground mt-2'>Enable or disable platform features</p>
      </div>

      <Card>
        <CardHeader>
          <div className='flex items-center gap-2'>
            <Zap className='h-5 w-5' />
            <CardTitle>Feature Flags</CardTitle>
          </div>
          <CardDescription>Enable or disable platform features</CardDescription>
        </CardHeader>
        <CardContent className='space-y-6'>
          {isLoading ? (
            <div className='flex justify-center py-8'>
              <Loader2 className='h-6 w-6 animate-spin text-muted-foreground' />
            </div>
          ) : (
            <>
              <OverridableSwitch
                id='enable-realtime'
                label='Enable Realtime'
                description='Real-time subscriptions and WebSocket connections'
                checked={features?.enable_realtime || false}
                onCheckedChange={(checked) => {
                  updateFeatureMutation.mutate({
                    key: 'app.features.enable_realtime',
                    value: checked,
                  })
                }}
                override={features?._overrides?.enable_realtime}
                disabled={updateFeatureMutation.isPending}
              />

              <OverridableSwitch
                id='enable-storage'
                label='Enable Storage'
                description='File storage and media management'
                checked={features?.enable_storage || false}
                onCheckedChange={(checked) => {
                  updateFeatureMutation.mutate({
                    key: 'app.features.enable_storage',
                    value: checked,
                  })
                }}
                override={features?._overrides?.enable_storage}
                disabled={updateFeatureMutation.isPending}
              />

              <OverridableSwitch
                id='enable-functions'
                label='Enable Edge Functions'
                description='Serverless functions and custom business logic'
                checked={features?.enable_functions || false}
                onCheckedChange={(checked) => {
                  updateFeatureMutation.mutate({
                    key: 'app.features.enable_functions',
                    value: checked,
                  })
                }}
                override={features?._overrides?.enable_functions}
                disabled={updateFeatureMutation.isPending}
              />

              <OverridableSwitch
                id='enable-ai'
                label='Enable AI Chatbots'
                description='AI-powered chatbots with database query capabilities'
                checked={features?.enable_ai || false}
                onCheckedChange={(checked) => {
                  updateFeatureMutation.mutate({
                    key: 'app.features.enable_ai',
                    value: checked,
                  })
                }}
                override={features?._overrides?.enable_ai}
                disabled={updateFeatureMutation.isPending}
              />

              <div className='rounded-lg bg-muted p-4'>
                <div className='flex gap-2'>
                  <AlertCircle className='h-5 w-5 text-muted-foreground shrink-0 mt-0.5' />
                  <div className='text-sm space-y-1'>
                    <p className='font-medium'>Feature Availability</p>
                    <p className='text-muted-foreground'>
                      Disabling features will prevent users from accessing related functionality.
                      Existing data will be preserved but inaccessible until re-enabled.
                    </p>
                  </div>
                </div>
              </div>
            </>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
