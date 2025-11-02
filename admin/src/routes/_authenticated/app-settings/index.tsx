import { createFileRoute } from '@tanstack/react-router'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Input } from '@/components/ui/input'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Badge } from '@/components/ui/badge'
import { Sliders, Shield, Mail, Zap, AlertCircle, CheckCircle2, Loader2 } from 'lucide-react'
import { toast } from 'sonner'
import { apiClient } from '@/lib/api'

export const Route = createFileRoute('/_authenticated/app-settings/')({
  component: AppSettingsPage,
})

interface AppSettings {
  authentication: {
    enable_signup: boolean
    enable_magic_link: boolean
    password_min_length: number
    require_email_verification: boolean
  }
  features: {
    enable_realtime: boolean
    enable_storage: boolean
    enable_functions: boolean
  }
  email: {
    enabled: boolean
    provider: string
  }
  security: {
    enable_global_rate_limit: boolean
  }
}

function AppSettingsPage() {
  const queryClient = useQueryClient()

  // Fetch app settings
  const { data: settings, isLoading, error } = useQuery<AppSettings>({
    queryKey: ['app-settings'],
    queryFn: async () => {
      const response = await apiClient.get('/api/v1/admin/app/settings')
      return response.data
    },
  })

  // Update app settings mutation
  const updateMutation = useMutation({
    mutationFn: async (updates: Partial<AppSettings>) => {
      const response = await apiClient.put('/api/v1/admin/app/settings', updates)
      return response.data
    },
    onMutate: async (updates) => {
      // Cancel any outgoing refetches
      await queryClient.cancelQueries({ queryKey: ['app-settings'] })

      // Snapshot the previous value
      const previousSettings = queryClient.getQueryData<AppSettings>(['app-settings'])

      // Optimistically update to the new value
      if (previousSettings) {
        queryClient.setQueryData<AppSettings>(['app-settings'], {
          authentication: {
            ...(previousSettings.authentication || {}),
            ...(updates.authentication || {}),
          },
          features: {
            ...(previousSettings.features || {}),
            ...(updates.features || {}),
          },
          email: {
            ...(previousSettings.email || {}),
            ...(updates.email || {}),
          },
          security: {
            ...(previousSettings.security || {}),
            ...(updates.security || {}),
          },
        })
      }

      // Return a context object with the snapshotted value
      return { previousSettings }
    },
    onSuccess: (data) => {
      // Update with the actual response from server
      queryClient.setQueryData(['app-settings'], data)
      toast.success('Settings updated successfully')
    },
    onError: (error: any, _variables, context) => {
      // Only rollback if we got an actual error response from the server
      // Network errors (CORS, connection refused) might mean the request succeeded but we can't verify
      if (error.response && context?.previousSettings) {
        queryClient.setQueryData(['app-settings'], context.previousSettings)
        toast.error(error.response?.data?.error || 'Failed to update settings')
      } else {
        // Network error - request may have succeeded, refetch to get actual state
        queryClient.invalidateQueries({ queryKey: ['app-settings'] })
        toast.error('Network error - please check if the backend is running')
      }
    },
  })

  // Reset app settings mutation
  const resetMutation = useMutation({
    mutationFn: async () => {
      const response = await apiClient.post('/api/v1/admin/app/settings/reset')
      return response.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['app-settings'] })
      toast.success('All settings have been reset to defaults')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.error || 'Failed to reset settings')
    },
  })

  const handleAuthSettingChange = (key: string, value: boolean | number) => {
    if (!settings) return
    updateMutation.mutate({
      authentication: {
        ...(settings.authentication || {}),
        [key]: value,
      },
    })
  }

  const handleFeatureSettingChange = (key: string, value: boolean) => {
    if (!settings) return
    updateMutation.mutate({
      features: {
        ...(settings.features || {}),
        [key]: value,
      },
    })
  }

  const handleEmailSettingChange = (key: string, value: boolean | string) => {
    if (!settings) return
    updateMutation.mutate({
      email: {
        ...(settings.email || {}),
        [key]: value,
      },
    })
  }

  const handleSecuritySettingChange = (key: string, value: boolean) => {
    if (!settings) return
    updateMutation.mutate({
      security: {
        ...(settings.security || {}),
        [key]: value,
      },
    })
  }

  const handleReset = () => {
    if (confirm('Are you sure you want to reset all settings to their default values? This action cannot be undone.')) {
      resetMutation.mutate()
    }
  }

  if (isLoading) {
    return (
      <div className='flex flex-col items-center justify-center h-96 gap-4'>
        <Loader2 className='h-8 w-8 animate-spin text-muted-foreground' />
        <p className='text-sm text-muted-foreground'>Loading app settings...</p>
      </div>
    )
  }

  if (error) {
    return (
      <div className='flex flex-col items-center justify-center h-96 gap-4'>
        <AlertCircle className='h-8 w-8 text-destructive' />
        <div className='text-center'>
          <p className='text-sm font-medium'>Failed to load app settings</p>
          <p className='text-xs text-muted-foreground mt-1'>Please try refreshing the page</p>
        </div>
      </div>
    )
  }

  return (
    <div className='flex flex-col gap-6 p-6'>
      <div className='flex items-center justify-between'>
        <div>
          <h1 className='text-3xl font-bold'>App Settings</h1>
          <p className='text-sm text-muted-foreground mt-1'>
            Configure authentication, features, email, and security settings for your application
          </p>
        </div>
        <div className='flex items-center gap-2'>
          <Badge variant='outline' className='border-blue-500 text-blue-500'>
            <Sliders className='mr-1 h-3 w-3' />
            Application
          </Badge>
          <Button
            variant='outline'
            size='sm'
            onClick={handleReset}
            disabled={resetMutation.isPending}
          >
            {resetMutation.isPending ? (
              <>
                <Loader2 className='mr-2 h-4 w-4 animate-spin' />
                Resetting...
              </>
            ) : (
              'Reset to Defaults'
            )}
          </Button>
        </div>
      </div>

      <Tabs defaultValue='authentication' className='w-full'>
        <TabsList className='grid w-full grid-cols-4'>
          <TabsTrigger value='authentication'>Authentication</TabsTrigger>
          <TabsTrigger value='features'>Features</TabsTrigger>
          <TabsTrigger value='email'>Email</TabsTrigger>
          <TabsTrigger value='security'>Security</TabsTrigger>
        </TabsList>

        {/* Authentication Tab */}
        <TabsContent value='authentication' className='space-y-4'>
          <Card>
            <CardHeader>
              <CardTitle className='flex items-center gap-2'>
                <Shield className='h-5 w-5' />
                Authentication Settings
              </CardTitle>
              <CardDescription>
                Control user registration, authentication methods, and password requirements
              </CardDescription>
            </CardHeader>
            <CardContent className='space-y-6'>
              {/* User Signup */}
              <div className='flex items-center justify-between py-3 border-b'>
                <div className='space-y-0.5'>
                  <Label className='text-base font-medium'>Enable User Signup</Label>
                  <p className='text-sm text-muted-foreground'>
                    Allow new users to register accounts via the signup endpoint
                  </p>
                </div>
                <Switch
                  checked={settings?.authentication?.enable_signup ?? false}
                  onCheckedChange={(checked) => handleAuthSettingChange('enable_signup', checked)}
                  disabled={updateMutation.isPending}
                />
              </div>

              {/* Magic Link */}
              <div className='flex items-center justify-between py-3 border-b'>
                <div className='space-y-0.5'>
                  <Label className='text-base font-medium'>Enable Magic Link</Label>
                  <p className='text-sm text-muted-foreground'>
                    Allow passwordless authentication via email magic links
                  </p>
                </div>
                <Switch
                  checked={settings?.authentication?.enable_magic_link ?? true}
                  onCheckedChange={(checked) => handleAuthSettingChange('enable_magic_link', checked)}
                  disabled={updateMutation.isPending}
                />
              </div>

              {/* Email Verification */}
              <div className='flex items-center justify-between py-3 border-b'>
                <div className='space-y-0.5'>
                  <Label className='text-base font-medium'>Require Email Verification</Label>
                  <p className='text-sm text-muted-foreground'>
                    Require users to verify their email address before accessing the application
                  </p>
                </div>
                <Switch
                  checked={settings?.authentication?.require_email_verification ?? false}
                  onCheckedChange={(checked) => handleAuthSettingChange('require_email_verification', checked)}
                  disabled={updateMutation.isPending}
                />
              </div>

              {/* Password Min Length */}
              <div className='space-y-3 py-3'>
                <div>
                  <Label className='text-base font-medium'>Minimum Password Length</Label>
                  <p className='text-sm text-muted-foreground'>
                    Minimum number of characters required for user passwords
                  </p>
                </div>
                <div className='flex items-center gap-4 max-w-xs'>
                  <Input
                    type='number'
                    min={6}
                    max={128}
                    value={settings?.authentication?.password_min_length ?? 8}
                    onChange={(e) => handleAuthSettingChange('password_min_length', parseInt(e.target.value))}
                    disabled={updateMutation.isPending}
                  />
                  <span className='text-sm text-muted-foreground whitespace-nowrap'>characters</span>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Features Tab */}
        <TabsContent value='features' className='space-y-4'>
          <Card>
            <CardHeader>
              <CardTitle className='flex items-center gap-2'>
                <Zap className='h-5 w-5' />
                Feature Flags
              </CardTitle>
              <CardDescription>
                Enable or disable platform features like realtime, storage, and functions
              </CardDescription>
            </CardHeader>
            <CardContent className='space-y-6'>
              {/* Realtime */}
              <div className='flex items-center justify-between py-3 border-b'>
                <div className='space-y-0.5'>
                  <Label className='text-base font-medium'>Enable Realtime</Label>
                  <p className='text-sm text-muted-foreground'>
                    Enable WebSocket-based realtime database subscriptions
                  </p>
                </div>
                <Switch
                  checked={settings?.features?.enable_realtime ?? true}
                  onCheckedChange={(checked) => handleFeatureSettingChange('enable_realtime', checked)}
                  disabled={updateMutation.isPending}
                />
              </div>

              {/* Storage */}
              <div className='flex items-center justify-between py-3 border-b'>
                <div className='space-y-0.5'>
                  <Label className='text-base font-medium'>Enable Storage</Label>
                  <p className='text-sm text-muted-foreground'>
                    Enable file storage and bucket management features
                  </p>
                </div>
                <Switch
                  checked={settings?.features?.enable_storage ?? true}
                  onCheckedChange={(checked) => handleFeatureSettingChange('enable_storage', checked)}
                  disabled={updateMutation.isPending}
                />
              </div>

              {/* Functions */}
              <div className='flex items-center justify-between py-3'>
                <div className='space-y-0.5'>
                  <Label className='text-base font-medium'>Enable Edge Functions</Label>
                  <p className='text-sm text-muted-foreground'>
                    Enable serverless edge functions for custom business logic
                  </p>
                </div>
                <Switch
                  checked={settings?.features?.enable_functions ?? true}
                  onCheckedChange={(checked) => handleFeatureSettingChange('enable_functions', checked)}
                  disabled={updateMutation.isPending}
                />
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Email Tab */}
        <TabsContent value='email' className='space-y-4'>
          <Card>
            <CardHeader>
              <CardTitle className='flex items-center gap-2'>
                <Mail className='h-5 w-5' />
                Email Settings
              </CardTitle>
              <CardDescription>
                Configure email service for sending authentication emails, notifications, and more
              </CardDescription>
            </CardHeader>
            <CardContent className='space-y-6'>
              {/* Email Enabled */}
              <div className='flex items-center justify-between py-3 border-b'>
                <div className='space-y-0.5'>
                  <Label className='text-base font-medium'>Enable Email Service</Label>
                  <p className='text-sm text-muted-foreground'>
                    Enable sending emails for magic links, verification, and notifications
                  </p>
                </div>
                <Switch
                  checked={settings?.email?.enabled ?? false}
                  onCheckedChange={(checked) => handleEmailSettingChange('enabled', checked)}
                  disabled={updateMutation.isPending}
                />
              </div>

              {/* Email Provider */}
              <div className='space-y-3 py-3'>
                <div>
                  <Label className='text-base font-medium'>Email Provider</Label>
                  <p className='text-sm text-muted-foreground'>
                    Current email provider (configured via environment variables)
                  </p>
                </div>
                <Badge variant='outline'>{settings?.email?.provider || 'SMTP'}</Badge>
              </div>

              {!settings?.email?.enabled && (
                <div className='flex items-start gap-2 p-4 border rounded-lg bg-amber-50 dark:bg-amber-950/20'>
                  <AlertCircle className='h-5 w-5 text-amber-600 dark:text-amber-500 mt-0.5' />
                  <div className='space-y-1'>
                    <p className='text-sm font-medium text-amber-900 dark:text-amber-100'>
                      Email service is disabled
                    </p>
                    <p className='text-xs text-amber-700 dark:text-amber-300'>
                      Features like magic link authentication and email verification will not work until email is enabled.
                    </p>
                  </div>
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* Security Tab */}
        <TabsContent value='security' className='space-y-4'>
          <Card>
            <CardHeader>
              <CardTitle className='flex items-center gap-2'>
                <Shield className='h-5 w-5' />
                Security Settings
              </CardTitle>
              <CardDescription>
                Configure security features like rate limiting to protect your application
              </CardDescription>
            </CardHeader>
            <CardContent className='space-y-6'>
              {/* Global Rate Limit */}
              <div className='flex items-center justify-between py-3'>
                <div className='space-y-0.5'>
                  <Label className='text-base font-medium'>Enable Global Rate Limiting</Label>
                  <p className='text-sm text-muted-foreground'>
                    Protect your API from abuse with global rate limiting across all endpoints
                  </p>
                </div>
                <Switch
                  checked={settings?.security?.enable_global_rate_limit ?? false}
                  onCheckedChange={(checked) => handleSecuritySettingChange('enable_global_rate_limit', checked)}
                  disabled={updateMutation.isPending}
                />
              </div>

              <div className='flex items-start gap-2 p-4 border rounded-lg bg-blue-50 dark:bg-blue-950/20'>
                <CheckCircle2 className='h-5 w-5 text-blue-600 dark:text-blue-400 mt-0.5' />
                <div className='space-y-1'>
                  <p className='text-sm font-medium text-blue-900 dark:text-blue-100'>
                    Endpoint-specific rate limits are always active
                  </p>
                  <p className='text-xs text-blue-700 dark:text-blue-300'>
                    Critical endpoints like signup, login, and admin setup have dedicated rate limits regardless of this setting.
                  </p>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}
