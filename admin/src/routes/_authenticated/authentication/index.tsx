import { createFileRoute } from '@tanstack/react-router'
import { useState, useMemo, useEffect, useRef } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Badge } from '@/components/ui/badge'
import { toast } from 'sonner'
import { Key, Settings, Users, Loader2, X, Check, AlertCircle, Shield } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { formatDistanceToNow } from 'date-fns'
import api, { oauthProviderApi, authSettingsApi, type OAuthProviderConfig, type CreateOAuthProviderRequest, type UpdateOAuthProviderRequest, type AuthSettings } from '@/lib/api'

export const Route = createFileRoute('/_authenticated/authentication/')({
  component: AuthenticationPage,
})

interface Session {
  id: string
  user_id: string
  access_token: string
  refresh_token: string
  expires_at: string
  created_at: string
  updated_at: string
  user?: {
    email: string
  }
}

function AuthenticationPage() {
  const [selectedTab, setSelectedTab] = useState('providers')

  return (
    <div className="flex flex-col gap-6 p-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight flex items-center gap-2">
          <Shield className="h-8 w-8" />
          Authentication
        </h1>
        <p className="text-muted-foreground mt-2">
          Manage OAuth providers, auth settings, and user sessions
        </p>
      </div>

          <Tabs value={selectedTab} onValueChange={setSelectedTab} className='w-full'>
            <TabsList className='grid w-full grid-cols-3'>
              <TabsTrigger value='providers'>
                <Key className='mr-2 h-4 w-4' />
                OAuth Providers
              </TabsTrigger>
              <TabsTrigger value='settings'>
                <Settings className='mr-2 h-4 w-4' />
                Auth Settings
              </TabsTrigger>
              <TabsTrigger value='sessions'>
                <Users className='mr-2 h-4 w-4' />
                Active Sessions
              </TabsTrigger>
            </TabsList>

            <TabsContent value='providers' className='space-y-4'>
              <OAuthProvidersTab />
            </TabsContent>

            <TabsContent value='settings' className='space-y-4'>
              <AuthSettingsTab />
            </TabsContent>

            <TabsContent value='sessions' className='space-y-4'>
              <ActiveSessionsTab />
            </TabsContent>
          </Tabs>
    </div>
  )
}

function OAuthProvidersTab() {
  const queryClient = useQueryClient()
  const [showAddProvider, setShowAddProvider] = useState(false)
  const [showEditProvider, setShowEditProvider] = useState(false)
  const [editingProvider, setEditingProvider] = useState<OAuthProviderConfig | null>(null)
  const [selectedProvider, setSelectedProvider] = useState<string>('')
  const [customProviderName, setCustomProviderName] = useState('')
  const [customAuthUrl, setCustomAuthUrl] = useState('')
  const [customTokenUrl, setCustomTokenUrl] = useState('')
  const [customUserInfoUrl, setCustomUserInfoUrl] = useState('')
  const [clientId, setClientId] = useState('')
  const [clientSecret, setClientSecret] = useState('')

  const availableProviders = [
    { id: 'google', name: 'Google', icon: '🔵' },
    { id: 'github', name: 'GitHub', icon: '⚫' },
    { id: 'microsoft', name: 'Microsoft', icon: '🟦' },
    { id: 'apple', name: 'Apple', icon: '⚪' },
    { id: 'facebook', name: 'Facebook', icon: '🔵' },
    { id: 'twitter', name: 'Twitter', icon: '🔵' },
    { id: 'linkedin', name: 'LinkedIn', icon: '🔵' },
    { id: 'gitlab', name: 'GitLab', icon: '🟠' },
    { id: 'bitbucket', name: 'Bitbucket', icon: '🔵' },
    { id: 'custom', name: 'Custom Provider', icon: '⚙️' },
  ]

  // Fetch OAuth providers from backend
  const { data: enabledProviders = [] } = useQuery({
    queryKey: ['oauthProviders'],
    queryFn: oauthProviderApi.list,
  })

  // Create OAuth provider mutation
  const createProviderMutation = useMutation({
    mutationFn: (data: CreateOAuthProviderRequest) => oauthProviderApi.create(data),
    onSuccess: (data) => {
      toast.success(data.message)
      queryClient.invalidateQueries({ queryKey: ['oauthProviders'] })
      setShowAddProvider(false)
      resetForm()
    },
    onError: (error: unknown) => {
      const errorMessage = error instanceof Error && 'response' in error
        ? (error as { response?: { data?: { error?: string } } }).response?.data?.error || 'Failed to create OAuth provider'
        : 'Failed to create OAuth provider'
      toast.error(errorMessage)
    },
  })

  // Update OAuth provider mutation
  const updateProviderMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateOAuthProviderRequest }) =>
      oauthProviderApi.update(id, data),
    onSuccess: (data) => {
      toast.success(data.message)
      queryClient.invalidateQueries({ queryKey: ['oauthProviders'] })
      setShowEditProvider(false)
      setEditingProvider(null)
      resetForm()
    },
    onError: (error: unknown) => {
      const errorMessage = error instanceof Error && 'response' in error
        ? (error as { response?: { data?: { error?: string } } }).response?.data?.error || 'Failed to update OAuth provider'
        : 'Failed to update OAuth provider'
      toast.error(errorMessage)
    },
  })

  // Delete OAuth provider mutation
  const deleteProviderMutation = useMutation({
    mutationFn: (id: string) => oauthProviderApi.delete(id),
    onSuccess: (data) => {
      toast.success(data.message)
      queryClient.invalidateQueries({ queryKey: ['oauthProviders'] })
    },
    onError: (error: unknown) => {
      const errorMessage = error instanceof Error && 'response' in error
        ? (error as { response?: { data?: { error?: string } } }).response?.data?.error || 'Failed to delete OAuth provider'
        : 'Failed to delete OAuth provider'
      toast.error(errorMessage)
    },
  })

  const resetForm = () => {
    setSelectedProvider('')
    setCustomProviderName('')
    setCustomAuthUrl('')
    setCustomTokenUrl('')
    setCustomUserInfoUrl('')
    setClientId('')
    setClientSecret('')
  }

  return (
    <div className='space-y-4'>
      <Card>
        <CardHeader>
          <div className='flex items-center justify-between'>
            <div>
              <CardTitle>OAuth Providers</CardTitle>
              <CardDescription>
                Configure external OAuth providers for social authentication
              </CardDescription>
            </div>
            <Button onClick={() => setShowAddProvider(true)}>
              <Key className='mr-2 h-4 w-4' />
              Add Provider
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {enabledProviders.length === 0 ? (
            <div className='flex flex-col items-center justify-center py-12 text-center'>
              <AlertCircle className='mb-4 h-12 w-12 text-muted-foreground' />
              <p className='text-muted-foreground'>No OAuth providers configured</p>
              <Button onClick={() => setShowAddProvider(true)} variant='outline' className='mt-4'>
                Add Your First Provider
              </Button>
            </div>
          ) : (
            <div className='space-y-4'>
              {enabledProviders.map((provider) => (
                <Card key={provider.id}>
                  <CardContent className='pt-6'>
                    <div className='flex items-start justify-between'>
                      <div className='space-y-2 flex-1'>
                        <div className='flex items-center gap-2'>
                          <h3 className='font-semibold text-lg'>{provider.display_name}</h3>
                          {provider.enabled ? (
                            <Badge variant='default' className='gap-1'>
                              <Check className='h-3 w-3' />
                              Enabled
                            </Badge>
                          ) : (
                            <Badge variant='secondary'>Disabled</Badge>
                          )}
                        </div>
                        <div className='grid grid-cols-2 gap-4 text-sm'>
                          <div>
                            <Label className='text-muted-foreground'>Client ID</Label>
                            <p className='font-mono text-xs break-all'>{provider.client_id}</p>
                          </div>
                          <div>
                            <Label className='text-muted-foreground'>Client Secret</Label>
                            <p className='font-mono text-xs'>{provider.client_secret ? '••••••••' : 'Not set'}</p>
                          </div>
                          <div className='col-span-2'>
                            <Label className='text-muted-foreground'>Redirect URL</Label>
                            <p className='font-mono text-xs break-all'>{provider.redirect_url}</p>
                          </div>
                          {provider.is_custom && (
                            <>
                              <div className='col-span-2'>
                                <Label className='text-muted-foreground'>Authorization URL</Label>
                                <p className='font-mono text-xs break-all'>{provider.authorization_url}</p>
                              </div>
                              <div className='col-span-2'>
                                <Label className='text-muted-foreground'>Token URL</Label>
                                <p className='font-mono text-xs break-all'>{provider.token_url}</p>
                              </div>
                              <div className='col-span-2'>
                                <Label className='text-muted-foreground'>User Info URL</Label>
                                <p className='font-mono text-xs break-all'>{provider.user_info_url}</p>
                              </div>
                            </>
                          )}
                          <div className='col-span-2'>
                            <Label className='text-muted-foreground'>Scopes</Label>
                            <div className='flex flex-wrap gap-1 mt-1'>
                              {provider.scopes.map((scope) => (
                                <Badge key={scope} variant='outline' className='text-xs'>
                                  {scope}
                                </Badge>
                              ))}
                            </div>
                          </div>
                        </div>
                      </div>
                      <div className='flex gap-2 ml-4'>
                        <Button
                          variant='outline'
                          size='sm'
                          onClick={() => {
                            setEditingProvider(provider)
                            setSelectedProvider(provider.id)
                            setCustomProviderName(provider.display_name)
                            setClientId(provider.client_id)
                            setClientSecret('')
                            if (provider.is_custom) {
                              setCustomAuthUrl(provider.authorization_url || '')
                              setCustomTokenUrl(provider.token_url || '')
                              setCustomUserInfoUrl(provider.user_info_url || '')
                            }
                            setShowEditProvider(true)
                          }}
                        >
                          Edit
                        </Button>
                        <Button
                          variant='outline'
                          size='sm'
                          onClick={() => {
                            const authUrl = provider.is_custom
                              ? provider.authorization_url
                              : `https://accounts.google.com/o/oauth2/v2/auth?client_id=${provider.client_id}&redirect_uri=${encodeURIComponent(provider.redirect_url)}&response_type=code&scope=${provider.scopes.join(' ')}`
                            window.open(authUrl, '_blank', 'width=500,height=600')
                            toast.success('Test authentication window opened')
                          }}
                        >
                          Test
                        </Button>
                        <Button
                          variant='destructive'
                          size='sm'
                          onClick={() => {
                            if (confirm(`Are you sure you want to remove ${provider.display_name}?`)) {
                              deleteProviderMutation.mutate(provider.id)
                            }
                          }}
                          disabled={deleteProviderMutation.isPending}
                        >
                          <X className='h-4 w-4' />
                        </Button>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Add Provider Dialog */}
      <Dialog open={showAddProvider} onOpenChange={setShowAddProvider}>
        <DialogContent className='max-w-2xl max-h-[90vh] overflow-y-auto'>
          <DialogHeader>
            <DialogTitle>Add OAuth Provider</DialogTitle>
            <DialogDescription>
              Configure a new OAuth provider for social authentication
            </DialogDescription>
          </DialogHeader>
          <div className='grid gap-4 py-4'>
            <div className='grid gap-2'>
              <Label htmlFor='provider'>Provider</Label>
              <select
                id='provider'
                className='flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background'
                value={selectedProvider}
                onChange={(e) => setSelectedProvider(e.target.value)}
              >
                <option value=''>Select a provider...</option>
                {availableProviders
                  .filter(
                    (p) => !enabledProviders.some((ep) => ep.id === p.id && p.id !== 'custom')
                  )
                  .map((provider) => (
                    <option key={provider.id} value={provider.id}>
                      {provider.icon} {provider.name}
                    </option>
                  ))}
              </select>
            </div>

            {selectedProvider === 'custom' && (
              <>
                <div className='grid gap-2'>
                  <Label htmlFor='customProviderName'>Provider Name</Label>
                  <Input
                    id='customProviderName'
                    placeholder='e.g., Okta, Auth0, Keycloak'
                    value={customProviderName}
                    onChange={(e) => setCustomProviderName(e.target.value)}
                  />
                </div>
                <div className='grid gap-2'>
                  <Label htmlFor='authorizationUrl'>Authorization URL</Label>
                  <Input
                    id='authorizationUrl'
                    placeholder='https://your-provider.com/oauth/authorize'
                    value={customAuthUrl}
                    onChange={(e) => setCustomAuthUrl(e.target.value)}
                  />
                  <p className='text-xs text-muted-foreground'>
                    The OAuth authorization endpoint
                  </p>
                </div>
                <div className='grid gap-2'>
                  <Label htmlFor='tokenUrl'>Token URL</Label>
                  <Input
                    id='tokenUrl'
                    placeholder='https://your-provider.com/oauth/token'
                    value={customTokenUrl}
                    onChange={(e) => setCustomTokenUrl(e.target.value)}
                  />
                  <p className='text-xs text-muted-foreground'>
                    The OAuth token exchange endpoint
                  </p>
                </div>
                <div className='grid gap-2'>
                  <Label htmlFor='userInfoUrl'>User Info URL</Label>
                  <Input
                    id='userInfoUrl'
                    placeholder='https://your-provider.com/oauth/userinfo'
                    value={customUserInfoUrl}
                    onChange={(e) => setCustomUserInfoUrl(e.target.value)}
                  />
                  <p className='text-xs text-muted-foreground'>
                    The endpoint to retrieve user information
                  </p>
                </div>
              </>
            )}

            <div className='grid gap-2'>
              <Label htmlFor='clientId'>Client ID</Label>
              <Input id='clientId' placeholder='your-client-id' />
            </div>
            <div className='grid gap-2'>
              <Label htmlFor='clientSecret'>Client Secret</Label>
              <Input id='clientSecret' type='password' placeholder='your-client-secret' />
            </div>
            <div className='grid gap-2'>
              <Label htmlFor='redirectUrl'>Redirect URL</Label>
              <Input
                id='redirectUrl'
                value={selectedProvider === 'custom'
                  ? `${window.location.origin}/api/v1/auth/oauth/${customProviderName.toLowerCase().replace(/\s+/g, '-') || 'custom'}/callback`
                  : `${window.location.origin}/api/v1/auth/oauth/${selectedProvider}/callback`}
                readOnly
                className='font-mono text-xs'
              />
              <p className='text-xs text-muted-foreground'>
                Use this URL in your OAuth provider configuration
              </p>
            </div>
          </div>
          <DialogFooter>
            <Button variant='outline' onClick={() => setShowAddProvider(false)}>
              Cancel
            </Button>
            <Button
              onClick={() => {
                const isCustom = selectedProvider === 'custom'
                const providerName = isCustom ? customProviderName.toLowerCase().replace(/\s+/g, '_') : selectedProvider

                const data: CreateOAuthProviderRequest = {
                  provider_name: providerName,
                  display_name: isCustom ? customProviderName : selectedProvider.charAt(0).toUpperCase() + selectedProvider.slice(1),
                  enabled: true,
                  client_id: clientId,
                  client_secret: clientSecret,
                  redirect_url: `${window.location.origin}/api/v1/auth/oauth/${providerName}/callback`,
                  scopes: ['openid', 'email', 'profile'],
                  is_custom: isCustom,
                  ...(isCustom && {
                    authorization_url: customAuthUrl,
                    token_url: customTokenUrl,
                    user_info_url: customUserInfoUrl,
                  }),
                }

                createProviderMutation.mutate(data)
              }}
              disabled={!selectedProvider || !clientId || !clientSecret || createProviderMutation.isPending}
            >
              {createProviderMutation.isPending ? 'Saving...' : 'Save Provider'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Provider Dialog */}
      <Dialog open={showEditProvider} onOpenChange={setShowEditProvider}>
        <DialogContent className='max-w-2xl max-h-[90vh] overflow-y-auto'>
          <DialogHeader>
            <DialogTitle>Edit OAuth Provider</DialogTitle>
            <DialogDescription>
              Update the configuration for {editingProvider?.display_name}
            </DialogDescription>
          </DialogHeader>
          <div className='grid gap-4 py-4'>
            <div className='grid gap-2'>
              <Label>Provider</Label>
              <Input value={editingProvider?.display_name || ''} disabled className='bg-muted' />
            </div>

            {editingProvider?.is_custom && (
              <>
                <div className='grid gap-2'>
                  <Label htmlFor='editProviderName'>Provider Name</Label>
                  <Input
                    id='editProviderName'
                    value={customProviderName}
                    onChange={(e) => setCustomProviderName(e.target.value)}
                  />
                </div>
                <div className='grid gap-2'>
                  <Label htmlFor='editAuthorizationUrl'>Authorization URL</Label>
                  <Input
                    id='editAuthorizationUrl'
                    value={customAuthUrl}
                    onChange={(e) => setCustomAuthUrl(e.target.value)}
                  />
                </div>
                <div className='grid gap-2'>
                  <Label htmlFor='editTokenUrl'>Token URL</Label>
                  <Input
                    id='editTokenUrl'
                    value={customTokenUrl}
                    onChange={(e) => setCustomTokenUrl(e.target.value)}
                  />
                </div>
                <div className='grid gap-2'>
                  <Label htmlFor='editUserInfoUrl'>User Info URL</Label>
                  <Input
                    id='editUserInfoUrl'
                    value={customUserInfoUrl}
                    onChange={(e) => setCustomUserInfoUrl(e.target.value)}
                  />
                </div>
              </>
            )}

            <div className='grid gap-2'>
              <Label htmlFor='editClientId'>Client ID</Label>
              <Input
                id='editClientId'
                value={clientId}
                onChange={(e) => setClientId(e.target.value)}
              />
            </div>
            <div className='grid gap-2'>
              <Label htmlFor='editClientSecret'>Client Secret</Label>
              <Input
                id='editClientSecret'
                type='password'
                placeholder='Leave empty to keep current secret'
                value={clientSecret}
                onChange={(e) => setClientSecret(e.target.value)}
              />
              <p className='text-xs text-muted-foreground'>
                Only provide a new secret if you want to change it
              </p>
            </div>
            <div className='grid gap-2'>
              <Label htmlFor='editRedirectUrl'>Redirect URL</Label>
              <Input
                id='editRedirectUrl'
                value={editingProvider?.redirect_url || ''}
                readOnly
                className='font-mono text-xs bg-muted'
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant='outline' onClick={() => {
              setShowEditProvider(false)
              setEditingProvider(null)
            }}>
              Cancel
            </Button>
            <Button
              onClick={() => {
                if (!editingProvider) return

                const data: UpdateOAuthProviderRequest = {
                  display_name: editingProvider.display_name,
                  enabled: editingProvider.enabled,
                  client_id: clientId,
                  ...(clientSecret && { client_secret: clientSecret }),
                }

                updateProviderMutation.mutate({ id: editingProvider.id, data })
              }}
              disabled={!editingProvider || updateProviderMutation.isPending}
            >
              {updateProviderMutation.isPending ? 'Saving...' : 'Save Changes'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

function AuthSettingsTab() {
  const queryClient = useQueryClient()

  // Fetch auth settings from backend
  const { data: fetchedSettings, isLoading } = useQuery({
    queryKey: ['authSettings'],
    queryFn: authSettingsApi.get,
  })

  // Use useMemo to derive the initial settings value from fetched data
  const initialSettings = useMemo(() => fetchedSettings || null, [fetchedSettings])

  // Local state for editing
  const [settings, setSettings] = useState<AuthSettings | null>(initialSettings)

  // Track previous fetchedSettings to avoid unnecessary updates
  const prevFetchedRef = useRef<AuthSettings | null>(null)

  // Sync settings when fetchedSettings changes (only on data fetch, not on user edits)
  useEffect(() => {
    if (fetchedSettings && prevFetchedRef.current !== fetchedSettings) {
      prevFetchedRef.current = fetchedSettings
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setSettings(fetchedSettings)
    }
  }, [fetchedSettings])

  // Update auth settings mutation
  const updateSettingsMutation = useMutation({
    mutationFn: (data: AuthSettings) => authSettingsApi.update(data),
    onSuccess: (data) => {
      toast.success(data.message)
      queryClient.invalidateQueries({ queryKey: ['authSettings'] })
    },
    onError: (error: unknown) => {
      const errorMessage = error instanceof Error && 'response' in error
        ? (error as { response?: { data?: { error?: string } } }).response?.data?.error || 'Failed to update auth settings'
        : 'Failed to update auth settings'
      toast.error(errorMessage)
    },
  })

  const handleSaveSettings = () => {
    if (settings) {
      updateSettingsMutation.mutate(settings)
    }
  }

  if (isLoading || !settings) {
    return <div className='flex justify-center p-8'><Loader2 className='h-6 w-6 animate-spin' /></div>
  }

  return (
    <div className='space-y-4'>
      <Card>
        <CardHeader>
          <CardTitle>Authentication Methods</CardTitle>
          <CardDescription>Enable or disable authentication methods</CardDescription>
        </CardHeader>
        <CardContent className='space-y-4'>
          <div className='flex items-center justify-between'>
            <div>
              <Label htmlFor='enableSignup'>Enable User Signup</Label>
              <p className='text-sm text-muted-foreground'>
                Allow new users to register accounts
              </p>
            </div>
            <Switch
              id='enableSignup'
              checked={settings.enable_signup}
              onCheckedChange={(checked) =>
                setSettings({ ...settings, enable_signup: checked })
              }
            />
          </div>
          <div className='flex items-center justify-between'>
            <div>
              <Label htmlFor='enableMagicLink'>Enable Magic Link</Label>
              <p className='text-sm text-muted-foreground'>
                Allow users to sign in via email magic links
              </p>
            </div>
            <Switch
              id='enableMagicLink'
              checked={settings.enable_magic_link}
              onCheckedChange={(checked) =>
                setSettings({ ...settings, enable_magic_link: checked })
              }
            />
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Password Requirements</CardTitle>
          <CardDescription>Configure password complexity requirements</CardDescription>
        </CardHeader>
        <CardContent className='space-y-4'>
          <div className='grid gap-2'>
            <Label htmlFor='minLength'>Minimum Length</Label>
            <Input
              id='minLength'
              type='number'
              value={settings.password_min_length}
              onChange={(e) =>
                setSettings({ ...settings, password_min_length: parseInt(e.target.value) })
              }
            />
          </div>
          <div className='flex items-center justify-between'>
            <Label htmlFor='uppercase'>Require Uppercase Letters</Label>
            <Switch
              id='uppercase'
              checked={settings.password_require_uppercase}
              onCheckedChange={(checked) =>
                setSettings({ ...settings, password_require_uppercase: checked })
              }
            />
          </div>
          <div className='flex items-center justify-between'>
            <Label htmlFor='numbers'>Require Numbers</Label>
            <Switch
              id='numbers'
              checked={settings.password_require_number}
              onCheckedChange={(checked) =>
                setSettings({ ...settings, password_require_number: checked })
              }
            />
          </div>
          <div className='flex items-center justify-between'>
            <Label htmlFor='symbols'>Require Symbols</Label>
            <Switch
              id='symbols'
              checked={settings.password_require_special}
              onCheckedChange={(checked) =>
                setSettings({ ...settings, password_require_special: checked })
              }
            />
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Session & Token Configuration</CardTitle>
          <CardDescription>Configure session and token expiration times</CardDescription>
        </CardHeader>
        <CardContent className='space-y-4'>
          <div className='grid gap-2'>
            <Label htmlFor='sessionTimeout'>Session Timeout (minutes)</Label>
            <Input
              id='sessionTimeout'
              type='number'
              value={settings.session_timeout_minutes}
              onChange={(e) =>
                setSettings({ ...settings, session_timeout_minutes: parseInt(e.target.value) })
              }
            />
          </div>
          <div className='grid gap-2'>
            <Label htmlFor='maxSessions'>Max Sessions Per User</Label>
            <Input
              id='maxSessions'
              type='number'
              value={settings.max_sessions_per_user}
              onChange={(e) =>
                setSettings({ ...settings, max_sessions_per_user: parseInt(e.target.value) })
              }
            />
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Email Verification</CardTitle>
          <CardDescription>Configure email verification requirements</CardDescription>
        </CardHeader>
        <CardContent>
          <div className='flex items-center justify-between'>
            <div>
              <Label htmlFor='emailVerification'>Require Email Verification</Label>
              <p className='text-sm text-muted-foreground'>
                Users must verify their email before accessing the application
              </p>
            </div>
            <Switch
              id='emailVerification'
              checked={settings.require_email_verification}
              onCheckedChange={(checked) =>
                setSettings({ ...settings, require_email_verification: checked })
              }
            />
          </div>
        </CardContent>
      </Card>

      <div className='flex justify-end'>
        <Button onClick={handleSaveSettings}>
          <Check className='mr-2 h-4 w-4' />
          Save Settings
        </Button>
      </div>
    </div>
  )
}

function ActiveSessionsTab() {
  const queryClient = useQueryClient()

  // Fetch active sessions from the database
  const { data: sessions, isLoading } = useQuery<Session[]>({
    queryKey: ['sessions'],
    queryFn: async () => {
      const response = await api.get<Session[]>('/api/v1/tables/auth.sessions?select=*,user:user_id(email)')
      return response.data
    },
  })

  const revokeSessionMutation = useMutation({
    mutationFn: async (sessionId: string) => {
      await api.delete(`/api/v1/tables/auth.sessions/${sessionId}`)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['sessions'] })
      toast.success('Session revoked successfully')
    },
    onError: () => {
      toast.error('Failed to revoke session')
    },
  })

  const revokeAllUserSessionsMutation = useMutation({
    mutationFn: async (userId: string) => {
      await api.delete(`/api/v1/tables/auth.sessions?user_id=eq.${userId}`)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['sessions'] })
      toast.success('All user sessions revoked successfully')
    },
    onError: () => {
      toast.error('Failed to revoke user sessions')
    },
  })

  const isExpired = (expiresAt: string) => new Date(expiresAt) < new Date()

  return (
    <div className='space-y-4'>
      <Card>
        <CardHeader>
          <div className='flex items-center justify-between'>
            <div>
              <CardTitle>Active Sessions</CardTitle>
              <CardDescription>
                Monitor and manage active user sessions across the platform
              </CardDescription>
            </div>
            <div className='flex gap-2'>
              <Badge variant='outline' className='text-sm'>
                {sessions?.filter((s) => !isExpired(s.expires_at)).length || 0} Active
              </Badge>
              <Badge variant='secondary' className='text-sm'>
                {sessions?.filter((s) => isExpired(s.expires_at)).length || 0} Expired
              </Badge>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className='flex items-center justify-center py-8'>
              <Loader2 className='h-8 w-8 animate-spin text-muted-foreground' />
            </div>
          ) : sessions && sessions.length > 0 ? (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>User</TableHead>
                  <TableHead>Session ID</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead>Expires</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className='text-right'>Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {sessions.map((session) => (
                  <TableRow key={session.id}>
                    <TableCell className='font-medium'>
                      {session.user?.email || 'Unknown'}
                    </TableCell>
                    <TableCell className='font-mono text-xs'>
                      {session.id.substring(0, 8)}...
                    </TableCell>
                    <TableCell className='text-sm text-muted-foreground'>
                      {formatDistanceToNow(new Date(session.created_at), { addSuffix: true })}
                    </TableCell>
                    <TableCell className='text-sm text-muted-foreground'>
                      {formatDistanceToNow(new Date(session.expires_at), { addSuffix: true })}
                    </TableCell>
                    <TableCell>
                      {isExpired(session.expires_at) ? (
                        <Badge variant='secondary'>Expired</Badge>
                      ) : (
                        <Badge variant='default'>Active</Badge>
                      )}
                    </TableCell>
                    <TableCell className='text-right'>
                      <div className='flex justify-end gap-2'>
                        <Button
                          variant='outline'
                          size='sm'
                          onClick={() => revokeSessionMutation.mutate(session.id)}
                          disabled={revokeSessionMutation.isPending}
                        >
                          Revoke
                        </Button>
                        <Button
                          variant='destructive'
                          size='sm'
                          onClick={() => revokeAllUserSessionsMutation.mutate(session.user_id)}
                          disabled={revokeAllUserSessionsMutation.isPending}
                        >
                          Revoke All
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          ) : (
            <div className='flex flex-col items-center justify-center py-12 text-center'>
              <Users className='mb-4 h-12 w-12 text-muted-foreground' />
              <p className='text-muted-foreground'>No active sessions found</p>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
