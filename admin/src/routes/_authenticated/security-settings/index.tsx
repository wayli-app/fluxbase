import { createFileRoute } from '@tanstack/react-router'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Checkbox } from '@/components/ui/checkbox'
import { Button } from '@/components/ui/button'
import { OverridableSwitch } from '@/components/admin/overridable-switch'
import { OverridableSelect, SelectItem } from '@/components/admin/overridable-select'
import { Shield, AlertCircle, Loader2, Bot, CheckCircle2, Info } from 'lucide-react'
import { toast } from 'sonner'
import { apiClient, captchaSettingsApi, type CaptchaSettingsResponse, type UpdateCaptchaSettingsRequest } from '@/lib/api'

export const Route = createFileRoute('/_authenticated/security-settings/')({
  component: SecuritySettingsPage,
})

interface SecuritySettings {
  enable_global_rate_limit: boolean
}

// Captcha form state
interface CaptchaFormState {
  provider: string
  site_key: string
  secret_key: string
  score_threshold: number
  endpoints: string[]
  cap_server_url: string
  cap_api_key: string
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

  // Fetch CAPTCHA settings (admin endpoint with management capabilities)
  const {
    data: captchaSettings,
    isLoading: isLoadingCaptcha,
    dataUpdatedAt
  } = useQuery<CaptchaSettingsResponse>({
    queryKey: ['captcha-settings'],
    queryFn: () => captchaSettingsApi.get(),
  })

  // Captcha form state - continuous editing pattern
  const [captchaForm, setCaptchaForm] = useState<CaptchaFormState>({
    provider: 'hcaptcha',
    site_key: '',
    secret_key: '',
    score_threshold: 0.5,
    endpoints: ['signup', 'login', 'password_reset', 'magic_link'],
    cap_server_url: '',
    cap_api_key: '',
  })
  const [hasUnsavedChanges, setHasUnsavedChanges] = useState(false)
  const [initializedFromDataUpdatedAt, setInitializedFromDataUpdatedAt] = useState<number | null>(null)

  // Initialize form state when settings are first loaded or refetched
  if (captchaSettings && dataUpdatedAt !== initializedFromDataUpdatedAt) {
    setInitializedFromDataUpdatedAt(dataUpdatedAt)
    setCaptchaForm({
      provider: captchaSettings.provider || 'hcaptcha',
      site_key: captchaSettings.site_key || '',
      secret_key: '', // Never populate from server
      score_threshold: captchaSettings.score_threshold || 0.5,
      endpoints: captchaSettings.endpoints || [],
      cap_server_url: captchaSettings.cap_server_url || '',
      cap_api_key: '', // Never populate from server
    })
    setHasUnsavedChanges(false)
  }

  // Update captcha settings mutation
  const updateCaptchaMutation = useMutation({
    mutationFn: (request: UpdateCaptchaSettingsRequest) => captchaSettingsApi.update(request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['captcha-settings'] })
      setHasUnsavedChanges(false)
      toast.success('Captcha settings updated successfully')
    },
    onError: (error: unknown) => {
      if (error && typeof error === 'object' && 'response' in error) {
        const err = error as {
          response?: {
            status?: number
            data?: { code?: string; error?: string }
          }
        }
        if (
          err.response?.status === 409 &&
          err.response?.data?.code === 'CONFIG_OVERRIDE'
        ) {
          toast.error(
            'This setting is controlled by configuration file or environment variable'
          )
          return
        }
        if (err.response?.data?.error) {
          toast.error(err.response.data.error)
          return
        }
      }
      toast.error('Failed to update captcha settings')
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

  // Helper to check if a field is overridden
  const isOverridden = (field: string) => {
    return captchaSettings?._overrides?.[field]?.is_overridden ?? false
  }

  // Helper to get environment variable name
  const getEnvVar = (field: string) => {
    return captchaSettings?._overrides?.[field]?.env_var || ''
  }

  // Helper to convert API override to component override format
  const getOverride = (field: string) => {
    const override = captchaSettings?._overrides?.[field]
    if (!override?.is_overridden) return undefined
    return {
      is_overridden: override.is_overridden,
      env_var: override.env_var || '',
    }
  }

  // Update form field and mark as changed
  const updateFormField = <K extends keyof CaptchaFormState>(field: K, value: CaptchaFormState[K]) => {
    setCaptchaForm((prev) => ({ ...prev, [field]: value }))
    setHasUnsavedChanges(true)
  }

  // Toggle endpoint selection
  const toggleEndpoint = (endpoint: string) => {
    const newEndpoints = captchaForm.endpoints.includes(endpoint)
      ? captchaForm.endpoints.filter((e) => e !== endpoint)
      : [...captchaForm.endpoints, endpoint]
    updateFormField('endpoints', newEndpoints)
  }

  // Save captcha settings
  const handleSaveCaptcha = () => {
    const request: UpdateCaptchaSettingsRequest = {
      provider: captchaForm.provider,
      site_key: captchaForm.site_key,
      score_threshold: captchaForm.score_threshold,
      endpoints: captchaForm.endpoints,
      cap_server_url: captchaForm.cap_server_url,
    }

    // Only include secrets if they were changed (non-empty)
    if (captchaForm.secret_key) {
      request.secret_key = captchaForm.secret_key
    }
    if (captchaForm.cap_api_key) {
      request.cap_api_key = captchaForm.cap_api_key
    }

    updateCaptchaMutation.mutate(request)
  }

  // Toggle enabled state
  const handleToggleEnabled = (enabled: boolean) => {
    updateCaptchaMutation.mutate({ enabled })
  }

  if (isLoading || isLoadingCaptcha) {
    return (
      <div className="flex items-center justify-center h-full">
        <Loader2 className="w-8 h-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  return (
    <div className="container mx-auto py-8 px-4 max-w-4xl">
      <div className="mb-8">
        <h1 className="text-3xl font-bold flex items-center gap-2">
          <Shield className="w-8 h-8" />
          Security Settings
        </h1>
        <p className="text-muted-foreground mt-2">
          Configure security features and access controls for your application
        </p>
      </div>

      <div className="space-y-6">
        {/* Global Rate Limiting */}
        <Card>
          <CardHeader>
            <CardTitle>Rate Limiting</CardTitle>
            <CardDescription>
              Protect your API from abuse by limiting request rates
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex items-center justify-between">
              <div>
                <Label htmlFor="rate-limit">Enable Global Rate Limit</Label>
                <p className="text-sm text-muted-foreground">
                  Apply rate limits to all API endpoints
                </p>
              </div>
              <Switch
                id="rate-limit"
                checked={settings?.enable_global_rate_limit ?? false}
                onCheckedChange={(checked) => {
                  updateSettingMutation.mutate({
                    key: 'app.security.enable_global_rate_limit',
                    value: checked,
                  })
                }}
              />
            </div>
          </CardContent>
        </Card>

        {/* CAPTCHA Settings */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Bot className="w-5 h-5" />
              CAPTCHA Protection
            </CardTitle>
            <CardDescription>
              Protect authentication endpoints from automated attacks
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            {/* Enabled Toggle */}
            <OverridableSwitch
              id="captcha-enabled"
              label="Enable CAPTCHA"
              description="Require CAPTCHA verification on protected endpoints"
              checked={captchaSettings?.enabled ?? false}
              onCheckedChange={handleToggleEnabled}
              disabled={updateCaptchaMutation.isPending}
              override={getOverride('enabled')}
            />

            {captchaSettings?.enabled && (
              <>
                {/* Provider Selection */}
                <OverridableSelect
                  id="captcha-provider"
                  label="CAPTCHA Provider"
                  value={captchaForm.provider}
                  onValueChange={(value) => updateFormField('provider', value)}
                  override={getOverride('provider')}
                >
                  <SelectItem value="hcaptcha">hCaptcha</SelectItem>
                  <SelectItem value="recaptcha_v3">reCAPTCHA v3</SelectItem>
                  <SelectItem value="turnstile">Cloudflare Turnstile</SelectItem>
                  <SelectItem value="cap">Cap (Self-hosted)</SelectItem>
                </OverridableSelect>

                {/* Site Key */}
                <div className="space-y-2">
                  <Label htmlFor="site_key">Site Key</Label>
                  <div className="relative">
                    <Input
                      id="site_key"
                      value={captchaForm.site_key}
                      onChange={(e) => updateFormField('site_key', e.target.value)}
                      disabled={isOverridden('site_key')}
                      placeholder="Enter your CAPTCHA site key"
                    />
                    {isOverridden('site_key') && (
                      <Badge variant="outline" className="absolute right-2 top-1/2 -translate-y-1/2">
                        ENV: {getEnvVar('site_key')}
                      </Badge>
                    )}
                  </div>
                </div>

                {/* Secret Key */}
                <div className="space-y-2">
                  <Label htmlFor="secret_key">Secret Key</Label>
                  <div className="space-y-2">
                    <div className="relative">
                      <Input
                        id="secret_key"
                        type="password"
                        value={captchaForm.secret_key}
                        onChange={(e) => updateFormField('secret_key', e.target.value)}
                        disabled={isOverridden('secret_key')}
                        placeholder="Leave empty to keep current secret"
                      />
                      {isOverridden('secret_key') && (
                        <Badge variant="outline" className="absolute right-2 top-1/2 -translate-y-1/2">
                          ENV: {getEnvVar('secret_key')}
                        </Badge>
                      )}
                    </div>
                    {captchaSettings?.secret_key_set && (
                      <Badge variant="secondary" className="flex items-center gap-1 w-fit">
                        <CheckCircle2 className="h-3 w-3" />
                        Secret configured
                      </Badge>
                    )}
                  </div>
                </div>

                {/* Score Threshold (reCAPTCHA v3 only) */}
                {captchaForm.provider === 'recaptcha_v3' && (
                  <div className="space-y-2">
                    <Label htmlFor="score_threshold">Score Threshold</Label>
                    <div className="relative">
                      <Input
                        id="score_threshold"
                        type="number"
                        min="0"
                        max="1"
                        step="0.1"
                        value={captchaForm.score_threshold}
                        onChange={(e) => updateFormField('score_threshold', parseFloat(e.target.value))}
                        disabled={isOverridden('score_threshold')}
                      />
                      {isOverridden('score_threshold') && (
                        <Badge variant="outline" className="absolute right-2 top-1/2 -translate-y-1/2">
                          ENV: {getEnvVar('score_threshold')}
                        </Badge>
                      )}
                    </div>
                    <p className="text-sm text-muted-foreground">
                      Minimum score (0.0-1.0) required to pass verification
                    </p>
                  </div>
                )}

                {/* Cap Provider Settings */}
                {captchaForm.provider === 'cap' && (
                  <>
                    <div className="space-y-2">
                      <Label htmlFor="cap_server_url">Cap Server URL</Label>
                      <div className="relative">
                        <Input
                          id="cap_server_url"
                          value={captchaForm.cap_server_url}
                          onChange={(e) => updateFormField('cap_server_url', e.target.value)}
                          disabled={isOverridden('cap_server_url')}
                          placeholder="https://cap.example.com"
                        />
                        {isOverridden('cap_server_url') && (
                          <Badge variant="outline" className="absolute right-2 top-1/2 -translate-y-1/2">
                            ENV: {getEnvVar('cap_server_url')}
                          </Badge>
                        )}
                      </div>
                    </div>

                    <div className="space-y-2">
                      <Label htmlFor="cap_api_key">Cap API Key</Label>
                      <div className="space-y-2">
                        <div className="relative">
                          <Input
                            id="cap_api_key"
                            type="password"
                            value={captchaForm.cap_api_key}
                            onChange={(e) => updateFormField('cap_api_key', e.target.value)}
                            disabled={isOverridden('cap_api_key')}
                            placeholder="Leave empty to keep current API key"
                          />
                          {isOverridden('cap_api_key') && (
                            <Badge variant="outline" className="absolute right-2 top-1/2 -translate-y-1/2">
                              ENV: {getEnvVar('cap_api_key')}
                            </Badge>
                          )}
                        </div>
                        {captchaSettings?.cap_api_key_set && (
                          <Badge variant="secondary" className="flex items-center gap-1 w-fit">
                            <CheckCircle2 className="h-3 w-3" />
                            API key configured
                          </Badge>
                        )}
                      </div>
                    </div>
                  </>
                )}

                {/* Protected Endpoints */}
                <div className="space-y-2">
                  <Label>Protected Endpoints</Label>
                  <p className="text-sm text-muted-foreground mb-2">
                    Select which authentication endpoints require CAPTCHA verification
                  </p>
                  <div className="space-y-2">
                    {[
                      { id: 'signup', label: 'Signup' },
                      { id: 'login', label: 'Login' },
                      { id: 'password_reset', label: 'Password Reset' },
                      { id: 'magic_link', label: 'Magic Link' },
                    ].map((endpoint) => (
                      <div key={endpoint.id} className="flex items-center space-x-2">
                        <Checkbox
                          id={endpoint.id}
                          checked={captchaForm.endpoints.includes(endpoint.id)}
                          onCheckedChange={() => toggleEndpoint(endpoint.id)}
                          disabled={isOverridden('endpoints')}
                        />
                        <Label htmlFor={endpoint.id} className="cursor-pointer">
                          {endpoint.label}
                        </Label>
                      </div>
                    ))}
                  </div>
                  {isOverridden('endpoints') && (
                    <Badge variant="outline" className="mt-2">
                      ENV: {getEnvVar('endpoints')}
                    </Badge>
                  )}
                </div>

                {/* Save Button */}
                <div className="flex items-center justify-between pt-4 border-t">
                  <div>
                    {hasUnsavedChanges && (
                      <p className="text-sm text-muted-foreground flex items-center gap-2">
                        <Info className="w-4 h-4" />
                        You have unsaved changes
                      </p>
                    )}
                  </div>
                  <Button
                    onClick={handleSaveCaptcha}
                    disabled={!hasUnsavedChanges || updateCaptchaMutation.isPending}
                  >
                    {updateCaptchaMutation.isPending ? (
                      <>
                        <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                        Saving...
                      </>
                    ) : (
                      'Save Changes'
                    )}
                  </Button>
                </div>
              </>
            )}

            {/* Warning about config overrides */}
            {captchaSettings && Object.values(captchaSettings._overrides).some((o) => o.is_overridden) && (
              <div className="flex items-start gap-2 p-4 bg-muted rounded-lg">
                <AlertCircle className="w-5 h-5 text-muted-foreground mt-0.5" />
                <div className="text-sm">
                  <p className="font-medium">Some settings are controlled by configuration</p>
                  <p className="text-muted-foreground">
                    Settings marked with ENV cannot be changed through the dashboard. Update your
                    configuration file or environment variables to modify these settings.
                  </p>
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
