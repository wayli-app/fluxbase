import { createFileRoute } from '@tanstack/react-router'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Shield, AlertCircle, Loader2 } from 'lucide-react'
import { toast } from 'sonner'
import { apiClient } from '@/lib/api'

export const Route = createFileRoute('/_authenticated/security-settings/')({
  component: SecuritySettingsPage,
})

interface SecuritySettings {
  enable_global_rate_limit: boolean
}

function SecuritySettingsPage() {
  const queryClient = useQueryClient()

  // Fetch security settings
  const { data: settings, isLoading } = useQuery<SecuritySettings>({
    queryKey: ['security-settings'],
    queryFn: async () => {
      const response = await apiClient.get('/api/v1/admin/system/settings/app.security.enable_global_rate_limit')
      return {
        enable_global_rate_limit: response.data.value.value,
      }
    },
  })

  // Update setting mutation
  const updateSettingMutation = useMutation({
    mutationFn: async ({ key, value }: { key: string; value: boolean }) => {
      await apiClient.put(`/api/v1/admin/system/settings/${key}`, {
        value: { value },
      })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['security-settings'] })
      toast.success('Security settings updated')
    },
    onError: () => {
      toast.error('Failed to update security settings')
    },
  })

  const handleToggleRateLimit = (checked: boolean) => {
    updateSettingMutation.mutate({
      key: 'app.security.enable_global_rate_limit',
      value: checked,
    })
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-full">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-6 p-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight flex items-center gap-2">
          <Shield className="h-8 w-8" />
          Security Settings
        </h1>
        <p className="text-muted-foreground mt-2">
          Configure security and rate limiting settings
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Rate Limiting</CardTitle>
          <CardDescription>
            Configure global rate limiting for API requests
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="flex items-center justify-between">
            <div className="space-y-0.5">
              <Label htmlFor="rate-limit">Enable Global Rate Limiting</Label>
              <p className="text-sm text-muted-foreground">
                Limit API requests to 100 per minute per IP address
              </p>
            </div>
            <Switch
              id="rate-limit"
              checked={settings?.enable_global_rate_limit || false}
              onCheckedChange={handleToggleRateLimit}
              disabled={updateSettingMutation.isPending}
            />
          </div>

          <div className="rounded-lg bg-muted p-4">
            <div className="flex gap-2">
              <AlertCircle className="h-5 w-5 text-muted-foreground shrink-0 mt-0.5" />
              <div className="text-sm space-y-1">
                <p className="font-medium">Rate Limit Behavior</p>
                <p className="text-muted-foreground">
                  When enabled, this applies a global rate limit of 100 requests per minute per IP address.
                  This helps protect your API from abuse and ensures fair usage across all clients.
                </p>
                <p className="text-muted-foreground">
                  Note: Individual endpoints may have their own specific rate limits that apply in addition to this global limit.
                </p>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Additional Security Features</CardTitle>
          <CardDescription>
            Additional security configuration options
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="rounded-lg bg-muted p-4">
            <div className="flex gap-2">
              <AlertCircle className="h-5 w-5 text-muted-foreground shrink-0 mt-0.5" />
              <div className="text-sm space-y-1">
                <p className="font-medium">Coming Soon</p>
                <p className="text-muted-foreground">
                  Additional security features like IP whitelisting, CORS configuration, and advanced authentication options will be available in future releases.
                </p>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
