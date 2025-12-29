import { createFileRoute } from '@tanstack/react-router'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Checkbox } from '@/components/ui/checkbox'
import { Shield, AlertCircle, Loader2, Bot, Lock, Info } from 'lucide-react'
import { toast } from 'sonner'
import { apiClient } from '@/lib/api'

export const Route = createFileRoute('/_authenticated/security-settings/')({
  component: SecuritySettingsPage,
})

interface SecuritySettings {
  enable_global_rate_limit: boolean
}

interface CaptchaConfig {
  enabled: boolean
  provider?: string
  site_key?: string
  endpoints?: string[]
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

  // Fetch CAPTCHA configuration (public endpoint)
  const { data: captchaConfig, isLoading: isLoadingCaptcha } = useQuery<CaptchaConfig>({
    queryKey: ['captcha-config'],
    queryFn: async () => {
      const response = await apiClient.get('/api/v1/auth/captcha/config')
      return response.data
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

  // Helper to get friendly provider name
  const getProviderDisplayName = (provider?: string) => {
    switch (provider) {
      case 'hcaptcha':
        return 'hCaptcha'
      case 'recaptcha_v3':
        return 'reCAPTCHA v3'
      case 'turnstile':
        return 'Cloudflare Turnstile'
      default:
        return provider || 'None'
    }
  }

  // Helper to get friendly endpoint name
  const getEndpointDisplayName = (endpoint: string) => {
    switch (endpoint) {
      case 'signup':
        return 'Sign Up'
      case 'login':
        return 'Sign In'
      case 'password_reset':
        return 'Password Reset'
      case 'magic_link':
        return 'Magic Link'
      default:
        return endpoint
    }
  }

  if (isLoading || isLoadingCaptcha) {
    return (
      <div className="flex items-center justify-center h-full">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  return (
    <div className="flex flex-1 flex-col gap-6 p-6">
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

      {/* CAPTCHA Protection */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="flex items-center gap-2">
                <Bot className="h-5 w-5" />
                CAPTCHA Protection
              </CardTitle>
              <CardDescription>
                Bot protection for authentication endpoints
              </CardDescription>
            </div>
            <Badge variant={captchaConfig?.enabled ? 'default' : 'secondary'}>
              {captchaConfig?.enabled ? 'Enabled' : 'Disabled'}
            </Badge>
          </div>
        </CardHeader>
        <CardContent className="space-y-6">
          {captchaConfig?.enabled ? (
            <>
              {/* Provider Info */}
              <div className="grid gap-4 md:grid-cols-2">
                <div className="space-y-2">
                  <Label>Provider</Label>
                  <div className="flex items-center gap-2">
                    <Input
                      value={getProviderDisplayName(captchaConfig.provider)}
                      readOnly
                      className="bg-muted"
                    />
                    <span title="Read-only (set via config)">
                      <Lock className="h-4 w-4 text-muted-foreground" />
                    </span>
                  </div>
                </div>
                <div className="space-y-2">
                  <Label>Site Key</Label>
                  <div className="flex items-center gap-2">
                    <Input
                      value={captchaConfig.site_key || ''}
                      readOnly
                      className="bg-muted font-mono text-xs"
                    />
                    <span title="Read-only (set via config)">
                      <Lock className="h-4 w-4 text-muted-foreground" />
                    </span>
                  </div>
                </div>
              </div>

              {/* Protected Endpoints */}
              <div className="space-y-3">
                <Label>Protected Endpoints</Label>
                <div className="grid gap-2 sm:grid-cols-2">
                  {['signup', 'login', 'password_reset', 'magic_link'].map((endpoint) => {
                    const isProtected = captchaConfig.endpoints?.includes(endpoint) ?? false
                    return (
                      <div key={endpoint} className="flex items-center gap-2">
                        <Checkbox
                          id={`endpoint-${endpoint}`}
                          checked={isProtected}
                          disabled
                          className="cursor-not-allowed"
                        />
                        <Label
                          htmlFor={`endpoint-${endpoint}`}
                          className={`text-sm cursor-not-allowed ${!isProtected ? 'text-muted-foreground' : ''}`}
                        >
                          {getEndpointDisplayName(endpoint)}
                        </Label>
                      </div>
                    )
                  })}
                </div>
              </div>

              {/* Config Notice */}
              <div className="rounded-lg bg-muted p-4">
                <div className="flex gap-2">
                  <Info className="h-5 w-5 text-muted-foreground shrink-0 mt-0.5" />
                  <div className="text-sm space-y-1">
                    <p className="font-medium">Configuration via YAML/Environment</p>
                    <p className="text-muted-foreground">
                      CAPTCHA settings are configured via the <code className="bg-background px-1 py-0.5 rounded text-xs">fluxbase.yaml</code> configuration file or environment variables.
                      Update <code className="bg-background px-1 py-0.5 rounded text-xs">security.captcha</code> settings and restart the server to apply changes.
                    </p>
                  </div>
                </div>
              </div>
            </>
          ) : (
            <>
              <div className="rounded-lg bg-muted p-4">
                <div className="flex gap-2">
                  <AlertCircle className="h-5 w-5 text-muted-foreground shrink-0 mt-0.5" />
                  <div className="text-sm space-y-1">
                    <p className="font-medium">CAPTCHA Not Configured</p>
                    <p className="text-muted-foreground">
                      CAPTCHA protection is currently disabled. To enable it, add CAPTCHA configuration to your <code className="bg-background px-1 py-0.5 rounded text-xs">fluxbase.yaml</code> file:
                    </p>
                    <pre className="mt-2 bg-background p-3 rounded text-xs overflow-x-auto">
{`security:
  captcha:
    enabled: true
    provider: hcaptcha  # or recaptcha_v3, turnstile
    site_key: "your-site-key"
    secret_key: "your-secret-key"
    endpoints:
      - signup
      - login
      - password_reset
      - magic_link`}
                    </pre>
                    <p className="text-muted-foreground mt-2">
                      Supported providers: <strong>hCaptcha</strong>, <strong>reCAPTCHA v3</strong>, <strong>Cloudflare Turnstile</strong>
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
