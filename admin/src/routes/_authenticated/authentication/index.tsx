import { useState, useMemo, useEffect, useRef } from 'react'
import z from 'zod'
import { formatDistanceToNow } from 'date-fns'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { createFileRoute, getRouteApi } from '@tanstack/react-router'
import {
  Key,
  Settings,
  Users,
  Loader2,
  X,
  Check,
  AlertCircle,
  Shield,
  ChevronLeft,
  ChevronRight,
  Building2,
  Copy,
  Plus,
  Upload,
  Pencil,
  Trash2,
  FileText,
  Link,
} from 'lucide-react'
import { toast } from 'sonner'
import api, {
  oauthProviderApi,
  authSettingsApi,
  samlProviderApi,
  dashboardAuthAPI,
  type OAuthProviderConfig,
  type CreateOAuthProviderRequest,
  type UpdateOAuthProviderRequest,
  type AuthSettings,
  type SAMLProviderConfig,
  type CreateSAMLProviderRequest,
  type UpdateSAMLProviderRequest,
} from '@/lib/api'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { KeyValueArrayEditor } from '@/components/key-value-array-editor'
import { StringArrayEditor } from '@/components/string-array-editor'

const authenticationSearchSchema = z.object({
  tab: z.string().optional().catch('providers'),
})

export const Route = createFileRoute('/_authenticated/authentication/')({
  validateSearch: authenticationSearchSchema,
  component: AuthenticationPage,
})

const route = getRouteApi('/_authenticated/authentication/')

interface Session {
  id: string
  user_id: string
  expires_at: string
  created_at: string
  user_email?: string
}

function AuthenticationPage() {
  const search = route.useSearch()
  const navigate = route.useNavigate()

  return (
    <div className='flex flex-1 flex-col gap-6 p-6'>
      <div>
        <h1 className='flex items-center gap-2 text-3xl font-bold tracking-tight'>
          <Shield className='h-8 w-8' />
          Authentication
        </h1>
        <p className='text-muted-foreground mt-2'>
          Manage OAuth providers, auth settings, and user sessions
        </p>
      </div>

      <Tabs
        value={search.tab || 'providers'}
        onValueChange={(tab) => navigate({ search: { tab } })}
        className='w-full'
      >
        <TabsList className='grid w-full grid-cols-4'>
          <TabsTrigger value='providers'>
            <Key className='mr-2 h-4 w-4' />
            OAuth Providers
          </TabsTrigger>
          <TabsTrigger value='saml'>
            <Building2 className='mr-2 h-4 w-4' />
            SAML SSO
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

        <TabsContent value='saml' className='space-y-4'>
          <SAMLProvidersTab />
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
  const [editingProvider, setEditingProvider] =
    useState<OAuthProviderConfig | null>(null)
  const [selectedProvider, setSelectedProvider] = useState<string>('')
  const [customProviderName, setCustomProviderName] = useState('')
  const [customAuthUrl, setCustomAuthUrl] = useState('')
  const [customTokenUrl, setCustomTokenUrl] = useState('')
  const [customUserInfoUrl, setCustomUserInfoUrl] = useState('')
  const [oidcDiscoveryUrl, setOidcDiscoveryUrl] = useState('')
  const [isDiscovering, setIsDiscovering] = useState(false)
  const [clientId, setClientId] = useState('')
  const [clientSecret, setClientSecret] = useState('')
  const [allowDashboardLogin, setAllowDashboardLogin] = useState(false)
  const [allowAppLogin, setAllowAppLogin] = useState(true)
  const [requiredClaims, setRequiredClaims] = useState<
    Record<string, string[]>
  >({})
  const [deniedClaims, setDeniedClaims] = useState<Record<string, string[]>>({})
  const [showDeleteProviderConfirm, setShowDeleteProviderConfirm] =
    useState(false)
  const [deletingProvider, setDeletingProvider] =
    useState<OAuthProviderConfig | null>(null)

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
    mutationFn: (data: CreateOAuthProviderRequest) =>
      oauthProviderApi.create(data),
    onSuccess: (data) => {
      toast.success(data.message)
      queryClient.invalidateQueries({ queryKey: ['oauthProviders'] })
      setShowAddProvider(false)
      resetForm()
    },
    onError: (error: unknown) => {
      const errorMessage =
        error instanceof Error && 'response' in error
          ? (error as { response?: { data?: { error?: string } } }).response
              ?.data?.error || 'Failed to create OAuth provider'
          : 'Failed to create OAuth provider'
      toast.error(errorMessage)
    },
  })

  // Update OAuth provider mutation
  const updateProviderMutation = useMutation({
    mutationFn: ({
      id,
      data,
    }: {
      id: string
      data: UpdateOAuthProviderRequest
    }) => oauthProviderApi.update(id, data),
    onSuccess: (data) => {
      toast.success(data.message)
      queryClient.invalidateQueries({ queryKey: ['oauthProviders'] })
      setShowEditProvider(false)
      setEditingProvider(null)
      resetForm()
    },
    onError: (error: unknown) => {
      const errorMessage =
        error instanceof Error && 'response' in error
          ? (error as { response?: { data?: { error?: string } } }).response
              ?.data?.error || 'Failed to update OAuth provider'
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
      const errorMessage =
        error instanceof Error && 'response' in error
          ? (error as { response?: { data?: { error?: string } } }).response
              ?.data?.error || 'Failed to delete OAuth provider'
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
    setOidcDiscoveryUrl('')
    setIsDiscovering(false)
    setClientId('')
    setClientSecret('')
    setAllowDashboardLogin(false)
    setAllowAppLogin(true)
    setRequiredClaims({})
    setDeniedClaims({})
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
              <AlertCircle className='text-muted-foreground mb-4 h-12 w-12' />
              <p className='text-muted-foreground'>
                No OAuth providers configured
              </p>
              <Button
                onClick={() => setShowAddProvider(true)}
                variant='outline'
                className='mt-4'
              >
                Add Your First Provider
              </Button>
            </div>
          ) : (
            <div className='space-y-4'>
              {enabledProviders.map((provider) => (
                <Card key={provider.id}>
                  <CardContent className='pt-6'>
                    <div className='flex items-start justify-between'>
                      <div className='flex-1 space-y-2'>
                        <div className='flex items-center gap-2'>
                          <h3 className='text-lg font-semibold'>
                            {provider.display_name}
                          </h3>
                          {provider.enabled ? (
                            <Badge variant='default' className='gap-1'>
                              <Check className='h-3 w-3' />
                              Enabled
                            </Badge>
                          ) : (
                            <Badge variant='secondary'>Disabled</Badge>
                          )}
                          {provider.allow_dashboard_login && (
                            <Badge variant='secondary' className='text-xs'>
                              Dashboard
                            </Badge>
                          )}
                          {provider.allow_app_login && (
                            <Badge variant='outline' className='text-xs'>
                              App
                            </Badge>
                          )}
                          {provider.source === 'config' && (
                            <Badge
                              variant='secondary'
                              className='gap-1 text-xs'
                            >
                              <Settings className='h-3 w-3' />
                              Config
                            </Badge>
                          )}
                        </div>
                        <div className='grid grid-cols-2 gap-4 text-sm'>
                          <div>
                            <Label className='text-muted-foreground'>
                              Client ID
                            </Label>
                            <p className='font-mono text-xs break-all'>
                              {provider.client_id}
                            </p>
                          </div>
                          <div>
                            <Label className='text-muted-foreground'>
                              Client Secret
                            </Label>
                            <p className='font-mono text-xs'>
                              {provider.has_secret ? '••••••••' : 'Not set'}
                            </p>
                          </div>
                          <div className='col-span-2'>
                            <Label className='text-muted-foreground'>
                              Redirect URL
                            </Label>
                            <p className='font-mono text-xs break-all'>
                              {provider.redirect_url}
                            </p>
                          </div>
                          {provider.is_custom && (
                            <>
                              <div className='col-span-2'>
                                <Label className='text-muted-foreground'>
                                  Authorization URL
                                </Label>
                                <p className='font-mono text-xs break-all'>
                                  {provider.authorization_url}
                                </p>
                              </div>
                              <div className='col-span-2'>
                                <Label className='text-muted-foreground'>
                                  Token URL
                                </Label>
                                <p className='font-mono text-xs break-all'>
                                  {provider.token_url}
                                </p>
                              </div>
                              <div className='col-span-2'>
                                <Label className='text-muted-foreground'>
                                  User Info URL
                                </Label>
                                <p className='font-mono text-xs break-all'>
                                  {provider.user_info_url}
                                </p>
                              </div>
                            </>
                          )}
                          <div className='col-span-2'>
                            <Label className='text-muted-foreground'>
                              Scopes
                            </Label>
                            <div className='mt-1 flex flex-wrap gap-1'>
                              {provider.scopes.map((scope) => (
                                <Badge
                                  key={scope}
                                  variant='outline'
                                  className='text-xs'
                                >
                                  {scope}
                                </Badge>
                              ))}
                            </div>
                          </div>
                          {(provider.required_claims ||
                            provider.denied_claims) && (
                            <div className='col-span-2 border-t pt-3'>
                              <Label className='text-muted-foreground mb-2 block'>
                                RBAC Rules
                              </Label>
                              {provider.required_claims &&
                                Object.keys(provider.required_claims).length >
                                  0 && (
                                  <div className='mb-2'>
                                    <span className='text-muted-foreground text-xs'>
                                      Required Claims:{' '}
                                    </span>
                                    <div className='mt-1 flex flex-wrap gap-1'>
                                      {Object.entries(
                                        provider.required_claims
                                      ).map(([key, values]) => (
                                        <Badge
                                          key={key}
                                          variant='outline'
                                          className='text-xs'
                                        >
                                          {key}: {values.join(', ')}
                                        </Badge>
                                      ))}
                                    </div>
                                  </div>
                                )}
                              {provider.denied_claims &&
                                Object.keys(provider.denied_claims).length >
                                  0 && (
                                  <div>
                                    <span className='text-muted-foreground text-xs'>
                                      Denied Claims:{' '}
                                    </span>
                                    <div className='mt-1 flex flex-wrap gap-1'>
                                      {Object.entries(
                                        provider.denied_claims
                                      ).map(([key, values]) => (
                                        <Badge
                                          key={key}
                                          variant='destructive'
                                          className='text-xs'
                                        >
                                          {key}: {values.join(', ')}
                                        </Badge>
                                      ))}
                                    </div>
                                  </div>
                                )}
                            </div>
                          )}
                        </div>
                      </div>
                      <div className='ml-4 flex gap-2'>
                        {provider.source !== 'config' && (
                          <Button
                            variant='outline'
                            size='sm'
                            onClick={() => {
                              setEditingProvider(provider)
                              setSelectedProvider(provider.id)
                              setCustomProviderName(provider.display_name)
                              setClientId(provider.client_id)
                              setClientSecret('')
                              setAllowDashboardLogin(
                                provider.allow_dashboard_login
                              )
                              setAllowAppLogin(provider.allow_app_login)
                              setRequiredClaims(provider.required_claims || {})
                              setDeniedClaims(provider.denied_claims || {})
                              if (provider.is_custom) {
                                setCustomAuthUrl(
                                  provider.authorization_url || ''
                                )
                                setCustomTokenUrl(provider.token_url || '')
                                setCustomUserInfoUrl(
                                  provider.user_info_url || ''
                                )
                              }
                              setShowEditProvider(true)
                            }}
                          >
                            Edit
                          </Button>
                        )}
                        <Button
                          variant='outline'
                          size='sm'
                          onClick={() => {
                            const authUrl = provider.is_custom
                              ? provider.authorization_url
                              : `https://accounts.google.com/o/oauth2/v2/auth?client_id=${provider.client_id}&redirect_uri=${encodeURIComponent(provider.redirect_url)}&response_type=code&scope=${provider.scopes.join(' ')}`
                            window.open(
                              authUrl,
                              '_blank',
                              'width=500,height=600'
                            )
                            toast.success('Test authentication window opened')
                          }}
                        >
                          Test
                        </Button>
                        {provider.source !== 'config' && (
                          <Button
                            variant='destructive'
                            size='sm'
                            onClick={() => {
                              setDeletingProvider(provider)
                              setShowDeleteProviderConfirm(true)
                            }}
                            disabled={deleteProviderMutation.isPending}
                          >
                            <X className='h-4 w-4' />
                          </Button>
                        )}
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
        <DialogContent className='max-h-[90vh] max-w-2xl overflow-y-auto'>
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
                className='border-input bg-background ring-offset-background flex h-10 w-full rounded-md border px-3 py-2 text-sm'
                value={selectedProvider}
                onChange={(e) => setSelectedProvider(e.target.value)}
              >
                <option value=''>Select a provider...</option>
                {availableProviders
                  .filter(
                    (p) =>
                      !enabledProviders.some(
                        (ep) => ep.id === p.id && p.id !== 'custom'
                      )
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

                {/* OIDC Discovery */}
                <div className='grid gap-2'>
                  <Label htmlFor='oidcDiscoveryUrl'>
                    OpenID Discovery URL (Optional)
                  </Label>
                  <div className='flex gap-2'>
                    <Input
                      id='oidcDiscoveryUrl'
                      placeholder='https://auth.example.com'
                      value={oidcDiscoveryUrl}
                      onChange={(e) => setOidcDiscoveryUrl(e.target.value)}
                    />
                    <Button
                      type='button'
                      variant='outline'
                      onClick={async () => {
                        if (!oidcDiscoveryUrl) {
                          toast.error('Please enter a discovery URL')
                          return
                        }

                        try {
                          setIsDiscovering(true)

                          // Normalize the URL for auto-discovery
                          // If it doesn't contain .well-known, append it
                          let discoveryUrl = oidcDiscoveryUrl.trim()
                          if (!discoveryUrl.includes('.well-known')) {
                            // Remove trailing slash if present
                            discoveryUrl = discoveryUrl.replace(/\/$/, '')
                            // Append well-known endpoint
                            discoveryUrl = `${discoveryUrl}/.well-known/openid-configuration`
                          }

                          const response = await fetch(discoveryUrl)
                          if (!response.ok) {
                            throw new Error(
                              `Failed to fetch: ${response.statusText}`
                            )
                          }

                          const config = await response.json()

                          // Auto-fill the fields
                          if (config.authorization_endpoint) {
                            setCustomAuthUrl(config.authorization_endpoint)
                          }
                          if (config.token_endpoint) {
                            setCustomTokenUrl(config.token_endpoint)
                          }
                          if (config.userinfo_endpoint) {
                            setCustomUserInfoUrl(config.userinfo_endpoint)
                          }

                          toast.success('Auto-discovered OAuth endpoints!')
                        } catch (error) {
                          toast.error(
                            `Discovery failed: ${error instanceof Error ? error.message : 'Unknown error'}`
                          )
                        } finally {
                          setIsDiscovering(false)
                        }
                      }}
                      disabled={!oidcDiscoveryUrl || isDiscovering}
                    >
                      {isDiscovering ? (
                        <>
                          <Loader2 className='mr-2 h-4 w-4 animate-spin' />
                          Discovering...
                        </>
                      ) : (
                        'Auto-discover'
                      )}
                    </Button>
                  </div>
                  <p className='text-muted-foreground text-xs'>
                    Supports base URLs or full discovery URLs. Auto-discovery
                    will be used:
                    <br />• Base URL:{' '}
                    <code className='text-xs'>
                      https://auth.example.com
                    </code>{' '}
                    (auto-appends /.well-known/openid-configuration)
                    <br />• Auth0:{' '}
                    <code className='text-xs'>
                      https://YOUR-DOMAIN.auth0.com
                    </code>
                    <br />• Keycloak:{' '}
                    <code className='text-xs'>
                      https://YOUR-DOMAIN/realms/YOUR-REALM
                    </code>
                    <br />• Custom:{' '}
                    <code className='text-xs'>
                      https://auth.example.com/.well-known/custom-oidc
                    </code>
                  </p>
                </div>

                <div className='grid gap-2'>
                  <Label htmlFor='authorizationUrl'>Authorization URL</Label>
                  <Input
                    id='authorizationUrl'
                    placeholder='https://your-provider.com/oauth/authorize'
                    value={customAuthUrl}
                    onChange={(e) => setCustomAuthUrl(e.target.value)}
                  />
                  <p className='text-muted-foreground text-xs'>
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
                  <p className='text-muted-foreground text-xs'>
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
                  <p className='text-muted-foreground text-xs'>
                    The endpoint to retrieve user information
                  </p>
                </div>
              </>
            )}

            <div className='grid gap-2'>
              <Label htmlFor='clientId'>Client ID</Label>
              <Input
                id='clientId'
                placeholder='your-client-id'
                value={clientId}
                onChange={(e) => setClientId(e.target.value)}
              />
            </div>
            <div className='grid gap-2'>
              <Label htmlFor='clientSecret'>Client Secret</Label>
              <Input
                id='clientSecret'
                type='password'
                placeholder='your-client-secret'
                value={clientSecret}
                onChange={(e) => setClientSecret(e.target.value)}
              />
            </div>
            <div className='grid gap-2'>
              <Label htmlFor='redirectUrl'>Redirect URL</Label>
              <Input
                id='redirectUrl'
                value={
                  selectedProvider === 'custom'
                    ? `${window.location.origin}/api/v1/auth/oauth/${customProviderName.toLowerCase().replace(/\s+/g, '_') || 'custom'}/callback`
                    : `${window.location.origin}/api/v1/auth/oauth/${selectedProvider}/callback`
                }
                readOnly
                className='font-mono text-xs'
              />
              <p className='text-muted-foreground text-xs'>
                Use this URL in your OAuth provider configuration
              </p>
            </div>

            {/* Provider Targeting */}
            <div className='space-y-3 border-t pt-4'>
              <div>
                <Label className='text-sm font-semibold'>
                  Provider Targeting
                </Label>
                <p className='text-muted-foreground mt-1 text-xs'>
                  Control which authentication contexts can use this provider
                </p>
              </div>
              <div className='space-y-3'>
                <div className='flex items-center justify-between'>
                  <div>
                    <Label>Allow for App Users</Label>
                    <p className='text-muted-foreground text-xs'>
                      Enable this provider for application user authentication
                    </p>
                  </div>
                  <Switch
                    checked={allowAppLogin}
                    onCheckedChange={setAllowAppLogin}
                  />
                </div>
                <div className='flex items-center justify-between'>
                  <div>
                    <Label>Allow for Dashboard</Label>
                    <p className='text-muted-foreground text-xs'>
                      Enable this provider for dashboard admin authentication
                    </p>
                  </div>
                  <Switch
                    checked={allowDashboardLogin}
                    onCheckedChange={setAllowDashboardLogin}
                  />
                </div>
              </div>
            </div>

            {/* RBAC Section */}
            <div className='space-y-4 border-t pt-4'>
              <div>
                <Label className='text-sm font-semibold'>
                  Role-Based Access Control (Optional)
                </Label>
                <p className='text-muted-foreground mt-1 text-xs'>
                  Filter users based on ID token claims (e.g., roles, groups)
                </p>
              </div>

              {/* Required Claims */}
              <div className='space-y-2'>
                <Label>Required Claims (OR logic)</Label>
                <p className='text-muted-foreground text-xs'>
                  User must have at least ONE matching value per claim
                </p>
                <KeyValueArrayEditor
                  value={requiredClaims}
                  onChange={setRequiredClaims}
                  keyPlaceholder='Claim name (e.g., roles)'
                  valuePlaceholder='Allowed value'
                  addButtonText='Add Required Claim'
                />
              </div>

              {/* Denied Claims */}
              <div className='space-y-2'>
                <Label>Denied Claims (Blocklist)</Label>
                <p className='text-muted-foreground text-xs'>
                  Reject users if ANY value matches
                </p>
                <KeyValueArrayEditor
                  value={deniedClaims}
                  onChange={setDeniedClaims}
                  keyPlaceholder='Claim name (e.g., status)'
                  valuePlaceholder='Denied value'
                  addButtonText='Add Denied Claim'
                />
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button variant='outline' onClick={() => setShowAddProvider(false)}>
              Cancel
            </Button>
            <Button
              onClick={() => {
                const isCustom = selectedProvider === 'custom'
                const providerName = isCustom
                  ? customProviderName.toLowerCase().replace(/\s+/g, '_')
                  : selectedProvider

                const data: CreateOAuthProviderRequest = {
                  provider_name: providerName,
                  display_name: isCustom
                    ? customProviderName
                    : selectedProvider.charAt(0).toUpperCase() +
                      selectedProvider.slice(1),
                  enabled: true,
                  client_id: clientId,
                  client_secret: clientSecret,
                  redirect_url: `${window.location.origin}/api/v1/auth/oauth/${providerName}/callback`,
                  scopes: ['openid', 'email', 'profile'],
                  is_custom: isCustom,
                  allow_dashboard_login: allowDashboardLogin,
                  allow_app_login: allowAppLogin,
                  ...(isCustom && {
                    authorization_url: customAuthUrl,
                    token_url: customTokenUrl,
                    user_info_url: customUserInfoUrl,
                  }),
                  ...(Object.keys(requiredClaims).length > 0 && {
                    required_claims: requiredClaims,
                  }),
                  ...(Object.keys(deniedClaims).length > 0 && {
                    denied_claims: deniedClaims,
                  }),
                }

                createProviderMutation.mutate(data)
              }}
              disabled={
                !selectedProvider ||
                !clientId ||
                !clientSecret ||
                createProviderMutation.isPending
              }
            >
              {createProviderMutation.isPending ? 'Saving...' : 'Save Provider'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Provider Dialog */}
      <Dialog open={showEditProvider} onOpenChange={setShowEditProvider}>
        <DialogContent className='max-h-[90vh] max-w-2xl overflow-y-auto'>
          <DialogHeader>
            <DialogTitle>Edit OAuth Provider</DialogTitle>
            <DialogDescription>
              Update the configuration for {editingProvider?.display_name}
            </DialogDescription>
          </DialogHeader>
          <div className='grid gap-4 py-4'>
            <div className='grid gap-2'>
              <Label>Provider</Label>
              <Input
                value={editingProvider?.display_name || ''}
                disabled
                className='bg-muted'
              />
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
                  <Label htmlFor='editAuthorizationUrl'>
                    Authorization URL
                  </Label>
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
              <p className='text-muted-foreground text-xs'>
                Only provide a new secret if you want to change it
              </p>
            </div>
            <div className='grid gap-2'>
              <Label htmlFor='editRedirectUrl'>Redirect URL</Label>
              <Input
                id='editRedirectUrl'
                value={editingProvider?.redirect_url || ''}
                readOnly
                className='bg-muted font-mono text-xs'
              />
            </div>

            {/* Provider Targeting */}
            <div className='space-y-3 border-t pt-4'>
              <div>
                <Label className='text-sm font-semibold'>
                  Provider Targeting
                </Label>
                <p className='text-muted-foreground mt-1 text-xs'>
                  Control which authentication contexts can use this provider
                </p>
              </div>
              <div className='space-y-3'>
                <div className='flex items-center justify-between'>
                  <div>
                    <Label>Allow for App Users</Label>
                    <p className='text-muted-foreground text-xs'>
                      Enable this provider for application user authentication
                    </p>
                  </div>
                  <Switch
                    checked={allowAppLogin}
                    onCheckedChange={setAllowAppLogin}
                  />
                </div>
                <div className='flex items-center justify-between'>
                  <div>
                    <Label>Allow for Dashboard</Label>
                    <p className='text-muted-foreground text-xs'>
                      Enable this provider for dashboard admin authentication
                    </p>
                  </div>
                  <Switch
                    checked={allowDashboardLogin}
                    onCheckedChange={setAllowDashboardLogin}
                  />
                </div>
              </div>
            </div>

            {/* RBAC Section */}
            <div className='space-y-4 border-t pt-4'>
              <div>
                <Label className='text-sm font-semibold'>
                  Role-Based Access Control (Optional)
                </Label>
                <p className='text-muted-foreground mt-1 text-xs'>
                  Filter users based on ID token claims (e.g., roles, groups)
                </p>
              </div>

              {/* Required Claims */}
              <div className='space-y-2'>
                <Label>Required Claims (OR logic)</Label>
                <p className='text-muted-foreground text-xs'>
                  User must have at least ONE matching value per claim
                </p>
                <KeyValueArrayEditor
                  value={requiredClaims}
                  onChange={setRequiredClaims}
                  keyPlaceholder='Claim name (e.g., roles)'
                  valuePlaceholder='Allowed value'
                  addButtonText='Add Required Claim'
                />
              </div>

              {/* Denied Claims */}
              <div className='space-y-2'>
                <Label>Denied Claims (Blocklist)</Label>
                <p className='text-muted-foreground text-xs'>
                  Reject users if ANY value matches
                </p>
                <KeyValueArrayEditor
                  value={deniedClaims}
                  onChange={setDeniedClaims}
                  keyPlaceholder='Claim name (e.g., status)'
                  valuePlaceholder='Denied value'
                  addButtonText='Add Denied Claim'
                />
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button
              variant='outline'
              onClick={() => {
                setShowEditProvider(false)
                setEditingProvider(null)
              }}
            >
              Cancel
            </Button>
            <Button
              onClick={() => {
                if (!editingProvider) return

                const data: UpdateOAuthProviderRequest = {
                  display_name: editingProvider.display_name,
                  enabled: editingProvider.enabled,
                  client_id: clientId,
                  allow_dashboard_login: allowDashboardLogin,
                  allow_app_login: allowAppLogin,
                  ...(clientSecret && { client_secret: clientSecret }),
                  ...(Object.keys(requiredClaims).length > 0 && {
                    required_claims: requiredClaims,
                  }),
                  ...(Object.keys(deniedClaims).length > 0 && {
                    denied_claims: deniedClaims,
                  }),
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

      {/* Delete Provider Confirmation */}
      <ConfirmDialog
        open={showDeleteProviderConfirm}
        onOpenChange={setShowDeleteProviderConfirm}
        title='Remove OAuth Provider'
        desc={`Are you sure you want to remove ${deletingProvider?.display_name}? Users will no longer be able to sign in with this provider.`}
        confirmText='Remove'
        destructive
        isLoading={deleteProviderMutation.isPending}
        handleConfirm={() => {
          if (deletingProvider) {
            deleteProviderMutation.mutate(deletingProvider.id, {
              onSuccess: () => {
                setShowDeleteProviderConfirm(false)
                setDeletingProvider(null)
              },
            })
          }
        }}
      />
    </div>
  )
}

function SAMLProvidersTab() {
  const queryClient = useQueryClient()
  const baseUrl = window.location.origin
  const [showAddProvider, setShowAddProvider] = useState(false)
  const [showEditProvider, setShowEditProvider] = useState(false)
  const [editingProvider, setEditingProvider] =
    useState<SAMLProviderConfig | null>(null)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [deletingProvider, setDeletingProvider] =
    useState<SAMLProviderConfig | null>(null)

  // Form state
  const [providerName, setProviderName] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [metadataSource, setMetadataSource] = useState<'url' | 'xml'>('url')
  const [metadataUrl, setMetadataUrl] = useState('')
  const [metadataXml, setMetadataXml] = useState('')
  const [autoCreateUsers, setAutoCreateUsers] = useState(true)
  const [defaultRole, setDefaultRole] = useState('authenticated')
  const [allowDashboardLogin, setAllowDashboardLogin] = useState(false)
  const [allowAppLogin, setAllowAppLogin] = useState(true)
  const [allowIdpInitiated, setAllowIdpInitiated] = useState(false)
  const [requiredGroups, setRequiredGroups] = useState<string[]>([])
  const [requiredGroupsAll, setRequiredGroupsAll] = useState<string[]>([])
  const [deniedGroups, setDeniedGroups] = useState<string[]>([])
  const [groupAttribute, setGroupAttribute] = useState('groups')
  const [validatingMetadata, setValidatingMetadata] = useState(false)
  const [metadataValid, setMetadataValid] = useState<boolean | null>(null)
  const [metadataError, setMetadataError] = useState<string | null>(null)

  // Fetch SAML providers
  const { data: providers = [], isLoading } = useQuery({
    queryKey: ['samlProviders'],
    queryFn: samlProviderApi.list,
  })

  // Create mutation
  const createMutation = useMutation({
    mutationFn: (data: CreateSAMLProviderRequest) =>
      samlProviderApi.create(data),
    onSuccess: (data) => {
      toast.success(data.message)
      queryClient.invalidateQueries({ queryKey: ['samlProviders'] })
      setShowAddProvider(false)
      resetForm()
    },
    onError: (error: unknown) => {
      const errorMessage =
        error instanceof Error && 'response' in error
          ? (error as { response?: { data?: { error?: string } } }).response
              ?.data?.error || 'Failed to create SAML provider'
          : 'Failed to create SAML provider'
      toast.error(errorMessage)
    },
  })

  // Update mutation
  const updateMutation = useMutation({
    mutationFn: ({
      id,
      data,
    }: {
      id: string
      data: UpdateSAMLProviderRequest
    }) => samlProviderApi.update(id, data),
    onSuccess: (data) => {
      toast.success(data.message)
      queryClient.invalidateQueries({ queryKey: ['samlProviders'] })
      setShowEditProvider(false)
      setEditingProvider(null)
      resetForm()
    },
    onError: (error: unknown) => {
      const errorMessage =
        error instanceof Error && 'response' in error
          ? (error as { response?: { data?: { error?: string } } }).response
              ?.data?.error || 'Failed to update SAML provider'
          : 'Failed to update SAML provider'
      toast.error(errorMessage)
    },
  })

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: (id: string) => samlProviderApi.delete(id),
    onSuccess: (data) => {
      toast.success(data.message)
      queryClient.invalidateQueries({ queryKey: ['samlProviders'] })
      setShowDeleteConfirm(false)
      setDeletingProvider(null)
    },
    onError: (error: unknown) => {
      const errorMessage =
        error instanceof Error && 'response' in error
          ? (error as { response?: { data?: { error?: string } } }).response
              ?.data?.error || 'Failed to delete SAML provider'
          : 'Failed to delete SAML provider'
      toast.error(errorMessage)
    },
  })

  const resetForm = () => {
    setProviderName('')
    setDisplayName('')
    setMetadataSource('url')
    setMetadataUrl('')
    setMetadataXml('')
    setAutoCreateUsers(true)
    setDefaultRole('authenticated')
    setAllowDashboardLogin(false)
    setAllowAppLogin(true)
    setAllowIdpInitiated(false)
    setRequiredGroups([])
    setRequiredGroupsAll([])
    setDeniedGroups([])
    setGroupAttribute('groups')
    setMetadataValid(null)
    setMetadataError(null)
  }

  const validateMetadata = async () => {
    setValidatingMetadata(true)
    setMetadataValid(null)
    setMetadataError(null)
    try {
      const result = await samlProviderApi.validateMetadata(
        metadataSource === 'url' ? metadataUrl : undefined,
        metadataSource === 'xml' ? metadataXml : undefined
      )
      if (result.valid) {
        setMetadataValid(true)
        toast.success(`Metadata valid! IdP Entity ID: ${result.entity_id}`)
      } else {
        setMetadataValid(false)
        setMetadataError(result.error || 'Invalid metadata')
      }
    } catch {
      setMetadataValid(false)
      setMetadataError('Failed to validate metadata')
    } finally {
      setValidatingMetadata(false)
    }
  }

  const handleFileUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return

    try {
      const result = await samlProviderApi.uploadMetadata(file)
      if (result.valid && result.metadata) {
        setMetadataXml(result.metadata)
        setMetadataValid(true)
        toast.success(`Metadata uploaded! IdP Entity ID: ${result.entity_id}`)
      } else {
        setMetadataValid(false)
        setMetadataError(result.error || 'Invalid metadata file')
      }
    } catch {
      setMetadataError('Failed to upload metadata file')
    }
  }

  const handleCreateProvider = () => {
    if (!providerName) {
      toast.error('Provider name is required')
      return
    }
    if (metadataSource === 'url' && !metadataUrl) {
      toast.error('Metadata URL is required')
      return
    }
    if (metadataSource === 'xml' && !metadataXml) {
      toast.error('Metadata XML is required')
      return
    }

    createMutation.mutate({
      name: providerName.toLowerCase().replace(/[^a-z0-9_-]/g, '_'),
      display_name: displayName || providerName,
      enabled: true,
      idp_metadata_url: metadataSource === 'url' ? metadataUrl : undefined,
      idp_metadata_xml: metadataSource === 'xml' ? metadataXml : undefined,
      auto_create_users: autoCreateUsers,
      default_role: defaultRole,
      allow_dashboard_login: allowDashboardLogin,
      allow_app_login: allowAppLogin,
      allow_idp_initiated: allowIdpInitiated,
      ...(requiredGroups.length > 0 && { required_groups: requiredGroups }),
      ...(requiredGroupsAll.length > 0 && {
        required_groups_all: requiredGroupsAll,
      }),
      ...(deniedGroups.length > 0 && { denied_groups: deniedGroups }),
      group_attribute: groupAttribute || 'groups',
    })
  }

  const handleEditProvider = (provider: SAMLProviderConfig) => {
    setEditingProvider(provider)
    setProviderName(provider.name)
    setDisplayName(provider.display_name)
    setMetadataUrl(provider.idp_metadata_url || '')
    setMetadataXml(provider.idp_metadata_xml || '')
    setMetadataSource(provider.idp_metadata_url ? 'url' : 'xml')
    setAutoCreateUsers(provider.auto_create_users)
    setDefaultRole(provider.default_role)
    setAllowDashboardLogin(provider.allow_dashboard_login)
    setAllowAppLogin(provider.allow_app_login)
    setAllowIdpInitiated(provider.allow_idp_initiated)
    setRequiredGroups(provider.required_groups || [])
    setRequiredGroupsAll(provider.required_groups_all || [])
    setDeniedGroups(provider.denied_groups || [])
    setGroupAttribute(provider.group_attribute || 'groups')
    setShowEditProvider(true)
  }

  const handleUpdateProvider = () => {
    if (!editingProvider) return

    updateMutation.mutate({
      id: editingProvider.id,
      data: {
        display_name: displayName || undefined,
        idp_metadata_url: metadataSource === 'url' ? metadataUrl : undefined,
        idp_metadata_xml: metadataSource === 'xml' ? metadataXml : undefined,
        auto_create_users: autoCreateUsers,
        default_role: defaultRole,
        allow_dashboard_login: allowDashboardLogin,
        allow_app_login: allowAppLogin,
        allow_idp_initiated: allowIdpInitiated,
        ...(requiredGroups.length > 0 && { required_groups: requiredGroups }),
        ...(requiredGroupsAll.length > 0 && {
          required_groups_all: requiredGroupsAll,
        }),
        ...(deniedGroups.length > 0 && { denied_groups: deniedGroups }),
        group_attribute: groupAttribute || 'groups',
      },
    })
  }

  const copyToClipboard = (text: string, label: string) => {
    navigator.clipboard.writeText(text)
    toast.success(`${label} copied to clipboard`)
  }

  if (isLoading) {
    return (
      <div className='flex justify-center p-8'>
        <Loader2 className='h-6 w-6 animate-spin' />
      </div>
    )
  }

  return (
    <div className='space-y-4'>
      <Card>
        <CardHeader>
          <div className='flex items-center justify-between'>
            <div>
              <CardTitle className='flex items-center gap-2'>
                <Building2 className='h-5 w-5' />
                SAML SSO Providers
              </CardTitle>
              <CardDescription>
                Enterprise Single Sign-On via SAML 2.0. Configure Identity
                Providers like Okta, Azure AD, or OneLogin.
              </CardDescription>
            </div>
            <Button onClick={() => setShowAddProvider(true)}>
              <Plus className='mr-2 h-4 w-4' />
              Add Provider
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {providers.length === 0 ? (
            <div className='flex flex-col items-center justify-center py-12 text-center'>
              <Building2 className='text-muted-foreground mb-4 h-12 w-12' />
              <p className='text-muted-foreground mb-2'>
                No SAML providers configured
              </p>
              <Button
                variant='outline'
                onClick={() => setShowAddProvider(true)}
              >
                <Plus className='mr-2 h-4 w-4' />
                Add your first SAML provider
              </Button>
            </div>
          ) : (
            <div className='space-y-4'>
              {providers.map((provider) => (
                <Card
                  key={provider.id}
                  className={
                    provider.source === 'config' ? 'border-dashed' : ''
                  }
                >
                  <CardContent className='pt-6'>
                    <div className='flex items-start justify-between'>
                      <div className='flex-1 space-y-4'>
                        <div className='flex flex-wrap items-center gap-2'>
                          <h3 className='text-lg font-semibold'>
                            {provider.display_name || provider.name}
                          </h3>
                          {provider.enabled ? (
                            <Badge variant='default' className='gap-1'>
                              <Check className='h-3 w-3' />
                              Enabled
                            </Badge>
                          ) : (
                            <Badge variant='secondary'>Disabled</Badge>
                          )}
                          {provider.source === 'config' && (
                            <Badge variant='outline'>
                              <FileText className='mr-1 h-3 w-3' />
                              Config File
                            </Badge>
                          )}
                          {provider.allow_dashboard_login && (
                            <Badge variant='outline'>Dashboard Login</Badge>
                          )}
                          {provider.allow_app_login && (
                            <Badge variant='outline'>App Login</Badge>
                          )}
                          {provider.auto_create_users && (
                            <Badge variant='outline'>Auto-create Users</Badge>
                          )}
                        </div>

                        <div className='grid grid-cols-1 gap-4 text-sm md:grid-cols-2'>
                          <div>
                            <Label className='text-muted-foreground'>
                              Provider Name
                            </Label>
                            <p className='mt-1 font-mono text-xs'>
                              {provider.name}
                            </p>
                          </div>
                          <div>
                            <Label className='text-muted-foreground'>
                              Default Role
                            </Label>
                            <p className='mt-1 font-mono text-xs'>
                              {provider.default_role}
                            </p>
                          </div>
                          <div>
                            <Label className='text-muted-foreground'>
                              Entity ID (SP)
                            </Label>
                            <div className='mt-1 flex items-center gap-2'>
                              <p className='flex-1 font-mono text-xs break-all'>
                                {provider.entity_id}
                              </p>
                              <Button
                                variant='ghost'
                                size='sm'
                                className='h-6 w-6 p-0'
                                onClick={() =>
                                  copyToClipboard(
                                    provider.entity_id,
                                    'Entity ID'
                                  )
                                }
                              >
                                <Copy className='h-3 w-3' />
                              </Button>
                            </div>
                          </div>
                          <div>
                            <Label className='text-muted-foreground'>
                              ACS URL
                            </Label>
                            <div className='mt-1 flex items-center gap-2'>
                              <p className='flex-1 font-mono text-xs break-all'>
                                {provider.acs_url}
                              </p>
                              <Button
                                variant='ghost'
                                size='sm'
                                className='h-6 w-6 p-0'
                                onClick={() =>
                                  copyToClipboard(provider.acs_url, 'ACS URL')
                                }
                              >
                                <Copy className='h-3 w-3' />
                              </Button>
                            </div>
                          </div>
                        </div>

                        {/* SP Metadata */}
                        <div className='mt-4 border-t pt-4'>
                          <Label className='text-muted-foreground'>
                            SP Metadata URL
                          </Label>
                          <div className='mt-1 flex items-center gap-2'>
                            <code className='bg-muted flex-1 rounded px-2 py-1 text-xs'>
                              {baseUrl}/api/v1/auth/saml/metadata/
                              {provider.name}
                            </code>
                            <Button
                              variant='outline'
                              size='sm'
                              onClick={() =>
                                copyToClipboard(
                                  `${baseUrl}/api/v1/auth/saml/metadata/${provider.name}`,
                                  'SP Metadata URL'
                                )
                              }
                            >
                              <Copy className='mr-1 h-3 w-3' />
                              Copy
                            </Button>
                          </div>
                        </div>

                        {/* RBAC Rules */}
                        {(provider.required_groups ||
                          provider.required_groups_all ||
                          provider.denied_groups) && (
                          <div className='mt-4 border-t pt-4'>
                            <Label className='text-muted-foreground mb-2 block'>
                              RBAC Rules
                            </Label>
                            <div className='space-y-2'>
                              {provider.required_groups &&
                                provider.required_groups.length > 0 && (
                                  <div>
                                    <span className='text-muted-foreground text-xs'>
                                      Required Groups (OR):{' '}
                                    </span>
                                    <div className='mt-1 flex flex-wrap gap-1'>
                                      {provider.required_groups.map((group) => (
                                        <Badge
                                          key={group}
                                          variant='outline'
                                          className='text-xs'
                                        >
                                          {group}
                                        </Badge>
                                      ))}
                                    </div>
                                  </div>
                                )}
                              {provider.required_groups_all &&
                                provider.required_groups_all.length > 0 && (
                                  <div>
                                    <span className='text-muted-foreground text-xs'>
                                      Required Groups (AND):{' '}
                                    </span>
                                    <div className='mt-1 flex flex-wrap gap-1'>
                                      {provider.required_groups_all.map(
                                        (group) => (
                                          <Badge
                                            key={group}
                                            variant='secondary'
                                            className='text-xs'
                                          >
                                            {group}
                                          </Badge>
                                        )
                                      )}
                                    </div>
                                  </div>
                                )}
                              {provider.denied_groups &&
                                provider.denied_groups.length > 0 && (
                                  <div>
                                    <span className='text-muted-foreground text-xs'>
                                      Denied Groups:{' '}
                                    </span>
                                    <div className='mt-1 flex flex-wrap gap-1'>
                                      {provider.denied_groups.map((group) => (
                                        <Badge
                                          key={group}
                                          variant='destructive'
                                          className='text-xs'
                                        >
                                          {group}
                                        </Badge>
                                      ))}
                                    </div>
                                  </div>
                                )}
                              {provider.group_attribute &&
                                provider.group_attribute !== 'groups' && (
                                  <div>
                                    <span className='text-muted-foreground text-xs'>
                                      Group Attribute:{' '}
                                    </span>
                                    <Badge
                                      variant='outline'
                                      className='text-xs'
                                    >
                                      {provider.group_attribute}
                                    </Badge>
                                  </div>
                                )}
                            </div>
                          </div>
                        )}
                      </div>

                      {/* Actions */}
                      {provider.source !== 'config' && (
                        <div className='ml-4 flex gap-2'>
                          <Button
                            variant='outline'
                            size='sm'
                            onClick={() => handleEditProvider(provider)}
                          >
                            <Pencil className='h-4 w-4' />
                          </Button>
                          <Button
                            variant='outline'
                            size='sm'
                            onClick={() => {
                              setDeletingProvider(provider)
                              setShowDeleteConfirm(true)
                            }}
                          >
                            <Trash2 className='h-4 w-4' />
                          </Button>
                        </div>
                      )}
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
        <DialogContent className='max-h-[90vh] max-w-2xl overflow-y-auto'>
          <DialogHeader>
            <DialogTitle>Add SAML Provider</DialogTitle>
            <DialogDescription>
              Configure a new SAML 2.0 Identity Provider for enterprise SSO.
            </DialogDescription>
          </DialogHeader>

          <div className='space-y-4'>
            <div className='grid grid-cols-2 gap-4'>
              <div className='space-y-2'>
                <Label>Provider Name *</Label>
                <Input
                  placeholder='okta'
                  value={providerName}
                  onChange={(e) => setProviderName(e.target.value)}
                />
                <p className='text-muted-foreground text-xs'>
                  Lowercase letters, numbers, underscores, hyphens only
                </p>
              </div>
              <div className='space-y-2'>
                <Label>Display Name</Label>
                <Input
                  placeholder='Okta SSO'
                  value={displayName}
                  onChange={(e) => setDisplayName(e.target.value)}
                />
              </div>
            </div>

            <div className='space-y-2'>
              <Label>IdP Metadata Source *</Label>
              <div className='flex gap-4'>
                <Button
                  type='button'
                  variant={metadataSource === 'url' ? 'default' : 'outline'}
                  size='sm'
                  onClick={() => setMetadataSource('url')}
                >
                  <Link className='mr-2 h-4 w-4' />
                  URL
                </Button>
                <Button
                  type='button'
                  variant={metadataSource === 'xml' ? 'default' : 'outline'}
                  size='sm'
                  onClick={() => setMetadataSource('xml')}
                >
                  <Upload className='mr-2 h-4 w-4' />
                  Upload XML
                </Button>
              </div>
            </div>

            {metadataSource === 'url' ? (
              <div className='space-y-2'>
                <Label>IdP Metadata URL *</Label>
                <div className='flex gap-2'>
                  <Input
                    placeholder='https://company.okta.com/app/xxx/sso/saml/metadata'
                    value={metadataUrl}
                    onChange={(e) => {
                      setMetadataUrl(e.target.value)
                      setMetadataValid(null)
                    }}
                    className='flex-1'
                  />
                  <Button
                    type='button'
                    variant='outline'
                    onClick={validateMetadata}
                    disabled={!metadataUrl || validatingMetadata}
                  >
                    {validatingMetadata ? (
                      <Loader2 className='h-4 w-4 animate-spin' />
                    ) : metadataValid ? (
                      <Check className='h-4 w-4 text-green-500' />
                    ) : metadataValid === false ? (
                      <X className='h-4 w-4 text-red-500' />
                    ) : (
                      'Validate'
                    )}
                  </Button>
                </div>
                {metadataError && (
                  <p className='text-xs text-red-500'>{metadataError}</p>
                )}
              </div>
            ) : (
              <div className='space-y-2'>
                <Label>IdP Metadata XML *</Label>
                <div className='space-y-2'>
                  <Input
                    type='file'
                    accept='.xml,text/xml,application/xml'
                    onChange={handleFileUpload}
                  />
                  <textarea
                    className='h-32 w-full rounded-md border p-2 font-mono text-xs'
                    placeholder='Paste IdP metadata XML here...'
                    value={metadataXml}
                    onChange={(e) => {
                      setMetadataXml(e.target.value)
                      setMetadataValid(null)
                    }}
                  />
                  <Button
                    type='button'
                    variant='outline'
                    size='sm'
                    onClick={validateMetadata}
                    disabled={!metadataXml || validatingMetadata}
                  >
                    {validatingMetadata ? (
                      <Loader2 className='mr-2 h-4 w-4 animate-spin' />
                    ) : null}
                    Validate XML
                  </Button>
                  {metadataError && (
                    <p className='text-xs text-red-500'>{metadataError}</p>
                  )}
                </div>
              </div>
            )}

            <div className='grid grid-cols-2 gap-4'>
              <div className='space-y-2'>
                <Label>Default Role</Label>
                <Input
                  placeholder='authenticated'
                  value={defaultRole}
                  onChange={(e) => setDefaultRole(e.target.value)}
                />
              </div>
            </div>

            <div className='space-y-4 border-t pt-4'>
              <Label className='text-base font-semibold'>Options</Label>
              <div className='grid grid-cols-2 gap-4'>
                <div className='flex items-center justify-between'>
                  <div>
                    <Label>Auto-create Users</Label>
                    <p className='text-muted-foreground text-xs'>
                      Create user if not exists
                    </p>
                  </div>
                  <Switch
                    checked={autoCreateUsers}
                    onCheckedChange={setAutoCreateUsers}
                  />
                </div>
                <div className='flex items-center justify-between'>
                  <div>
                    <Label>Allow IdP-Initiated SSO</Label>
                    <p className='text-muted-foreground text-xs'>Less secure</p>
                  </div>
                  <Switch
                    checked={allowIdpInitiated}
                    onCheckedChange={setAllowIdpInitiated}
                  />
                </div>
                <div className='flex items-center justify-between'>
                  <div>
                    <Label>Allow for App Users</Label>
                    <p className='text-muted-foreground text-xs'>
                      End-user authentication
                    </p>
                  </div>
                  <Switch
                    checked={allowAppLogin}
                    onCheckedChange={setAllowAppLogin}
                  />
                </div>
                <div className='flex items-center justify-between'>
                  <div>
                    <Label>Allow for Dashboard</Label>
                    <p className='text-muted-foreground text-xs'>Admin login</p>
                  </div>
                  <Switch
                    checked={allowDashboardLogin}
                    onCheckedChange={setAllowDashboardLogin}
                  />
                </div>
              </div>
            </div>

            {/* RBAC Section */}
            <div className='space-y-4 border-t pt-4'>
              <div>
                <Label className='text-sm font-semibold'>
                  Role-Based Access Control (Optional)
                </Label>
                <p className='text-muted-foreground mt-1 text-xs'>
                  Filter users based on SAML assertion groups/attributes
                </p>
              </div>

              {/* Group Attribute Name */}
              <div className='space-y-2'>
                <Label htmlFor='groupAttribute'>Group Attribute Name</Label>
                <Input
                  id='groupAttribute'
                  value={groupAttribute}
                  onChange={(e) => setGroupAttribute(e.target.value)}
                  placeholder='groups'
                />
                <p className='text-muted-foreground text-xs'>
                  SAML attribute containing group memberships (default:
                  "groups")
                </p>
              </div>

              {/* Required Groups (OR) */}
              <div className='space-y-2'>
                <Label>Required Groups (OR logic)</Label>
                <p className='text-muted-foreground text-xs'>
                  User must be in at least ONE of these groups
                </p>
                <StringArrayEditor
                  value={requiredGroups}
                  onChange={setRequiredGroups}
                  placeholder='FluxbaseAdmins'
                  addButtonText='Add Required Group'
                />
              </div>

              {/* Required Groups (AND) */}
              <div className='space-y-2'>
                <Label>Required Groups (AND logic)</Label>
                <p className='text-muted-foreground text-xs'>
                  User must be in ALL of these groups
                </p>
                <StringArrayEditor
                  value={requiredGroupsAll}
                  onChange={setRequiredGroupsAll}
                  placeholder='Verified'
                  addButtonText='Add Required Group'
                />
              </div>

              {/* Denied Groups */}
              <div className='space-y-2'>
                <Label>Denied Groups (Blocklist)</Label>
                <p className='text-muted-foreground text-xs'>
                  Reject users in ANY of these groups
                </p>
                <StringArrayEditor
                  value={deniedGroups}
                  onChange={setDeniedGroups}
                  placeholder='Contractors'
                  addButtonText='Add Denied Group'
                />
              </div>
            </div>
          </div>

          <DialogFooter>
            <Button
              variant='outline'
              onClick={() => {
                setShowAddProvider(false)
                resetForm()
              }}
            >
              Cancel
            </Button>
            <Button
              onClick={handleCreateProvider}
              disabled={createMutation.isPending}
            >
              {createMutation.isPending ? (
                <Loader2 className='mr-2 h-4 w-4 animate-spin' />
              ) : null}
              Create Provider
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Provider Dialog */}
      <Dialog open={showEditProvider} onOpenChange={setShowEditProvider}>
        <DialogContent className='max-h-[90vh] max-w-2xl overflow-y-auto'>
          <DialogHeader>
            <DialogTitle>Edit SAML Provider</DialogTitle>
            <DialogDescription>
              Update the configuration for{' '}
              {editingProvider?.display_name || editingProvider?.name}.
            </DialogDescription>
          </DialogHeader>

          <div className='space-y-4'>
            <div className='space-y-2'>
              <Label>Display Name</Label>
              <Input
                placeholder='Okta SSO'
                value={displayName}
                onChange={(e) => setDisplayName(e.target.value)}
              />
            </div>

            <div className='space-y-2'>
              <Label>IdP Metadata Source</Label>
              <div className='flex gap-4'>
                <Button
                  type='button'
                  variant={metadataSource === 'url' ? 'default' : 'outline'}
                  size='sm'
                  onClick={() => setMetadataSource('url')}
                >
                  <Link className='mr-2 h-4 w-4' />
                  URL
                </Button>
                <Button
                  type='button'
                  variant={metadataSource === 'xml' ? 'default' : 'outline'}
                  size='sm'
                  onClick={() => setMetadataSource('xml')}
                >
                  <Upload className='mr-2 h-4 w-4' />
                  Upload XML
                </Button>
              </div>
            </div>

            {metadataSource === 'url' ? (
              <div className='space-y-2'>
                <Label>IdP Metadata URL</Label>
                <div className='flex gap-2'>
                  <Input
                    placeholder='https://company.okta.com/app/xxx/sso/saml/metadata'
                    value={metadataUrl}
                    onChange={(e) => setMetadataUrl(e.target.value)}
                    className='flex-1'
                  />
                  <Button
                    type='button'
                    variant='outline'
                    onClick={validateMetadata}
                    disabled={!metadataUrl || validatingMetadata}
                  >
                    {validatingMetadata ? (
                      <Loader2 className='h-4 w-4 animate-spin' />
                    ) : (
                      'Validate'
                    )}
                  </Button>
                </div>
                {metadataError && (
                  <p className='text-xs text-red-500'>{metadataError}</p>
                )}
              </div>
            ) : (
              <div className='space-y-2'>
                <Label>IdP Metadata XML</Label>
                <Input
                  type='file'
                  accept='.xml,text/xml,application/xml'
                  onChange={handleFileUpload}
                />
                <textarea
                  className='h-32 w-full rounded-md border p-2 font-mono text-xs'
                  placeholder='Paste IdP metadata XML here...'
                  value={metadataXml}
                  onChange={(e) => {
                    setMetadataXml(e.target.value)
                    setMetadataValid(null)
                  }}
                />
                <Button
                  type='button'
                  variant='outline'
                  size='sm'
                  onClick={validateMetadata}
                  disabled={!metadataXml || validatingMetadata}
                >
                  {validatingMetadata ? (
                    <Loader2 className='mr-2 h-4 w-4 animate-spin' />
                  ) : null}
                  Validate XML
                </Button>
                {metadataError && (
                  <p className='text-xs text-red-500'>{metadataError}</p>
                )}
              </div>
            )}

            <div className='space-y-2'>
              <Label>Default Role</Label>
              <Input
                placeholder='authenticated'
                value={defaultRole}
                onChange={(e) => setDefaultRole(e.target.value)}
              />
            </div>

            <div className='space-y-4 border-t pt-4'>
              <Label className='text-base font-semibold'>Options</Label>
              <div className='grid grid-cols-2 gap-4'>
                <div className='flex items-center justify-between'>
                  <Label>Auto-create Users</Label>
                  <Switch
                    checked={autoCreateUsers}
                    onCheckedChange={setAutoCreateUsers}
                  />
                </div>
                <div className='flex items-center justify-between'>
                  <Label>Allow IdP-Initiated SSO</Label>
                  <Switch
                    checked={allowIdpInitiated}
                    onCheckedChange={setAllowIdpInitiated}
                  />
                </div>
                <div className='flex items-center justify-between'>
                  <Label>Allow for App Users</Label>
                  <Switch
                    checked={allowAppLogin}
                    onCheckedChange={setAllowAppLogin}
                  />
                </div>
                <div className='flex items-center justify-between'>
                  <Label>Allow for Dashboard</Label>
                  <Switch
                    checked={allowDashboardLogin}
                    onCheckedChange={setAllowDashboardLogin}
                  />
                </div>
              </div>
            </div>

            {/* RBAC Section */}
            <div className='space-y-4 border-t pt-4'>
              <div>
                <Label className='text-sm font-semibold'>
                  Role-Based Access Control (Optional)
                </Label>
                <p className='text-muted-foreground mt-1 text-xs'>
                  Filter users based on SAML assertion groups/attributes
                </p>
              </div>

              {/* Group Attribute Name */}
              <div className='space-y-2'>
                <Label htmlFor='editGroupAttribute'>Group Attribute Name</Label>
                <Input
                  id='editGroupAttribute'
                  value={groupAttribute}
                  onChange={(e) => setGroupAttribute(e.target.value)}
                  placeholder='groups'
                />
                <p className='text-muted-foreground text-xs'>
                  SAML attribute containing group memberships (default:
                  "groups")
                </p>
              </div>

              {/* Required Groups (OR) */}
              <div className='space-y-2'>
                <Label>Required Groups (OR logic)</Label>
                <p className='text-muted-foreground text-xs'>
                  User must be in at least ONE of these groups
                </p>
                <StringArrayEditor
                  value={requiredGroups}
                  onChange={setRequiredGroups}
                  placeholder='FluxbaseAdmins'
                  addButtonText='Add Required Group'
                />
              </div>

              {/* Required Groups (AND) */}
              <div className='space-y-2'>
                <Label>Required Groups (AND logic)</Label>
                <p className='text-muted-foreground text-xs'>
                  User must be in ALL of these groups
                </p>
                <StringArrayEditor
                  value={requiredGroupsAll}
                  onChange={setRequiredGroupsAll}
                  placeholder='Verified'
                  addButtonText='Add Required Group'
                />
              </div>

              {/* Denied Groups */}
              <div className='space-y-2'>
                <Label>Denied Groups (Blocklist)</Label>
                <p className='text-muted-foreground text-xs'>
                  Reject users in ANY of these groups
                </p>
                <StringArrayEditor
                  value={deniedGroups}
                  onChange={setDeniedGroups}
                  placeholder='Contractors'
                  addButtonText='Add Denied Group'
                />
              </div>
            </div>
          </div>

          <DialogFooter>
            <Button
              variant='outline'
              onClick={() => {
                setShowEditProvider(false)
                resetForm()
              }}
            >
              Cancel
            </Button>
            <Button
              onClick={handleUpdateProvider}
              disabled={updateMutation.isPending}
            >
              {updateMutation.isPending ? (
                <Loader2 className='mr-2 h-4 w-4 animate-spin' />
              ) : null}
              Save Changes
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation */}
      <ConfirmDialog
        open={showDeleteConfirm}
        onOpenChange={setShowDeleteConfirm}
        title='Delete SAML Provider'
        desc={`Are you sure you want to delete the SAML provider "${deletingProvider?.display_name || deletingProvider?.name}"? This action cannot be undone.`}
        confirmText='Delete'
        handleConfirm={() => {
          if (deletingProvider) {
            deleteMutation.mutate(deletingProvider.id)
          }
        }}
        isLoading={deleteMutation.isPending}
        destructive
      />
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

  // Fetch SSO providers to check if dashboard SSO is available
  const { data: ssoData } = useQuery({
    queryKey: ['dashboard-sso-providers'],
    queryFn: dashboardAuthAPI.getSSOProviders,
  })

  const hasDashboardSSOProviders = (ssoData?.providers?.length ?? 0) > 0

  // Fetch OAuth and SAML providers for app login check
  const { data: oauthProviders = [] } = useQuery({
    queryKey: ['oauthProviders'],
    queryFn: oauthProviderApi.list,
  })

  const { data: samlProviders = [] } = useQuery({
    queryKey: ['samlProviders'],
    queryFn: samlProviderApi.list,
  })

  const hasAppSSOProviders =
    (oauthProviders?.filter((p) => p.allow_app_login)?.length ?? 0) > 0 ||
    (samlProviders?.filter((p) => p.allow_app_login)?.length ?? 0) > 0

  // Use useMemo to derive the initial settings value from fetched data
  const initialSettings = useMemo(
    () => fetchedSettings || null,
    [fetchedSettings]
  )

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
      const errorMessage =
        error instanceof Error && 'response' in error
          ? (error as { response?: { data?: { error?: string } } }).response
              ?.data?.error || 'Failed to update auth settings'
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
    return (
      <div className='flex justify-center p-8'>
        <Loader2 className='h-6 w-6 animate-spin' />
      </div>
    )
  }

  return (
    <div className='space-y-4'>
      <Card>
        <CardHeader>
          <CardTitle>Authentication Methods</CardTitle>
          <CardDescription>
            Enable or disable authentication methods
          </CardDescription>
        </CardHeader>
        <CardContent className='space-y-4'>
          <div className='flex items-center justify-between'>
            <div>
              <Label htmlFor='enableSignup'>Enable User Signup</Label>
              <p className='text-muted-foreground text-sm'>
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
              <p className='text-muted-foreground text-sm'>
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
          <CardDescription>
            Configure password complexity requirements
          </CardDescription>
        </CardHeader>
        <CardContent className='space-y-4'>
          <div className='grid gap-2'>
            <Label htmlFor='minLength'>Minimum Length</Label>
            <Input
              id='minLength'
              type='number'
              value={settings.password_min_length}
              onChange={(e) =>
                setSettings({
                  ...settings,
                  password_min_length: parseInt(e.target.value),
                })
              }
            />
          </div>
          <div className='flex items-center justify-between'>
            <Label htmlFor='uppercase'>Require Uppercase Letters</Label>
            <Switch
              id='uppercase'
              checked={settings.password_require_uppercase}
              onCheckedChange={(checked) =>
                setSettings({
                  ...settings,
                  password_require_uppercase: checked,
                })
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
          <CardDescription>
            Configure session and token expiration times
          </CardDescription>
        </CardHeader>
        <CardContent className='space-y-4'>
          <div className='grid gap-2'>
            <Label htmlFor='sessionTimeout'>Session Timeout (minutes)</Label>
            <Input
              id='sessionTimeout'
              type='number'
              value={settings.session_timeout_minutes}
              onChange={(e) =>
                setSettings({
                  ...settings,
                  session_timeout_minutes: parseInt(e.target.value),
                })
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
                setSettings({
                  ...settings,
                  max_sessions_per_user: parseInt(e.target.value),
                })
              }
            />
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Email Verification</CardTitle>
          <CardDescription>
            Configure email verification requirements
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className='flex items-center justify-between'>
            <div>
              <Label htmlFor='emailVerification'>
                Require Email Verification
              </Label>
              <p className='text-muted-foreground text-sm'>
                Users must verify their email before accessing the application
              </p>
            </div>
            <Switch
              id='emailVerification'
              checked={settings.require_email_verification}
              onCheckedChange={(checked) =>
                setSettings({
                  ...settings,
                  require_email_verification: checked,
                })
              }
            />
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Dashboard Login</CardTitle>
          <CardDescription>
            Configure authentication methods for dashboard admins
          </CardDescription>
        </CardHeader>
        <CardContent className='space-y-4'>
          <div className='flex items-center justify-between'>
            <div className='flex-1 pr-4'>
              <Label htmlFor='disablePasswordLogin'>
                Disable Password Login
              </Label>
              <p className='text-muted-foreground text-sm'>
                Require SSO for all dashboard admin logins. Password
                authentication will be disabled.
              </p>
              {!hasDashboardSSOProviders && (
                <p className='mt-2 text-sm text-amber-600'>
                  Configure at least one OAuth or SAML provider with "Allow
                  dashboard login" enabled before you can disable password
                  login.
                </p>
              )}
            </div>
            <Switch
              id='disablePasswordLogin'
              checked={settings.disable_dashboard_password_login}
              onCheckedChange={(checked) =>
                setSettings({
                  ...settings,
                  disable_dashboard_password_login: checked,
                })
              }
              disabled={
                !hasDashboardSSOProviders &&
                !settings.disable_dashboard_password_login
              }
            />
          </div>
          {settings.disable_dashboard_password_login && (
            <div className='rounded-md border border-amber-200 bg-amber-50 p-3 dark:border-amber-800 dark:bg-amber-950'>
              <p className='text-sm text-amber-800 dark:text-amber-200'>
                <strong>Recovery:</strong> If you get locked out, set the
                environment variable{' '}
                <code className='rounded bg-amber-100 px-1 dark:bg-amber-900'>
                  FLUXBASE_DASHBOARD_FORCE_PASSWORD_LOGIN=true
                </code>{' '}
                to temporarily re-enable password login.
              </p>
            </div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>App User Login</CardTitle>
          <CardDescription>
            Configure authentication methods for application users
          </CardDescription>
        </CardHeader>
        <CardContent className='space-y-4'>
          <div className='flex items-center justify-between'>
            <div className='flex-1 pr-4'>
              <Label htmlFor='disableAppPasswordLogin'>
                Disable Password Login
              </Label>
              <p className='text-muted-foreground text-sm'>
                Require OAuth/SAML for all app user logins. Password
                authentication will be disabled.
              </p>
              {!hasAppSSOProviders && (
                <p className='mt-2 text-sm text-amber-600'>
                  Configure at least one OAuth or SAML provider with "Allow app
                  login" enabled before you can disable password login.
                </p>
              )}
            </div>
            <Switch
              id='disableAppPasswordLogin'
              checked={settings.disable_app_password_login}
              onCheckedChange={(checked) =>
                setSettings({
                  ...settings,
                  disable_app_password_login: checked,
                })
              }
              disabled={
                !hasAppSSOProviders && !settings.disable_app_password_login
              }
            />
          </div>
          {settings.disable_app_password_login && (
            <div className='rounded-md border border-amber-200 bg-amber-50 p-3 dark:border-amber-800 dark:bg-amber-950'>
              <p className='text-sm text-amber-800 dark:text-amber-200'>
                <strong>Recovery:</strong> If users get locked out, set the
                environment variable{' '}
                <code className='rounded bg-amber-100 px-1 dark:bg-amber-900'>
                  FLUXBASE_APP_FORCE_PASSWORD_LOGIN=true
                </code>{' '}
                to temporarily re-enable password login.
              </p>
            </div>
          )}
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

  // Pagination state
  const [page, setPage] = useState(0)
  const [pageSize, setPageSize] = useState(25)

  // Fetch active sessions from the admin API with pagination
  const { data: sessionsData, isLoading } = useQuery({
    queryKey: ['sessions', page, pageSize],
    queryFn: async () => {
      const response = await api.get<{
        sessions: Session[]
        count: number
        total_count: number
      }>(
        `/api/v1/admin/auth/sessions?include_expired=true&limit=${pageSize}&offset=${page * pageSize}`
      )
      return response.data
    },
  })

  const sessions = sessionsData?.sessions || []
  const totalCount = sessionsData?.total_count || 0
  const totalPages = Math.ceil(totalCount / pageSize)

  const revokeSessionMutation = useMutation({
    mutationFn: async (sessionId: string) => {
      await api.delete(`/api/v1/admin/auth/sessions/${sessionId}`)
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
      await api.delete(`/api/v1/admin/auth/sessions/user/${userId}`)
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
                {totalCount} Total
              </Badge>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className='flex items-center justify-center py-8'>
              <Loader2 className='text-muted-foreground h-8 w-8 animate-spin' />
            </div>
          ) : sessions && sessions.length > 0 ? (
            <>
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
                        {session.user_email || 'Unknown'}
                      </TableCell>
                      <TableCell className='font-mono text-xs'>
                        {session.id.substring(0, 8)}...
                      </TableCell>
                      <TableCell className='text-muted-foreground text-sm'>
                        {formatDistanceToNow(new Date(session.created_at), {
                          addSuffix: true,
                        })}
                      </TableCell>
                      <TableCell className='text-muted-foreground text-sm'>
                        {formatDistanceToNow(new Date(session.expires_at), {
                          addSuffix: true,
                        })}
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
                            onClick={() =>
                              revokeSessionMutation.mutate(session.id)
                            }
                            disabled={revokeSessionMutation.isPending}
                          >
                            Revoke
                          </Button>
                          <Button
                            variant='destructive'
                            size='sm'
                            onClick={() =>
                              revokeAllUserSessionsMutation.mutate(
                                session.user_id
                              )
                            }
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

              {/* Pagination Controls */}
              <div className='mt-4 flex items-center justify-between border-t pt-4'>
                <div className='flex items-center gap-2'>
                  <span className='text-muted-foreground text-sm'>
                    Rows per page
                  </span>
                  <Select
                    value={`${pageSize}`}
                    onValueChange={(value) => {
                      setPageSize(Number(value))
                      setPage(0)
                    }}
                  >
                    <SelectTrigger className='h-8 w-[70px]'>
                      <SelectValue placeholder={pageSize} />
                    </SelectTrigger>
                    <SelectContent side='top'>
                      {[10, 25, 50, 100].map((size) => (
                        <SelectItem key={size} value={`${size}`}>
                          {size}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className='flex items-center gap-2'>
                  <span className='text-muted-foreground text-sm'>
                    Page {page + 1} of {totalPages || 1} ({totalCount} total)
                  </span>
                  <Button
                    variant='outline'
                    size='sm'
                    onClick={() => setPage((p) => Math.max(0, p - 1))}
                    disabled={page === 0}
                  >
                    <ChevronLeft className='h-4 w-4' />
                  </Button>
                  <Button
                    variant='outline'
                    size='sm'
                    onClick={() =>
                      setPage((p) => Math.min(totalPages - 1, p + 1))
                    }
                    disabled={page >= totalPages - 1}
                  >
                    <ChevronRight className='h-4 w-4' />
                  </Button>
                </div>
              </div>
            </>
          ) : (
            <div className='flex flex-col items-center justify-center py-12 text-center'>
              <Users className='text-muted-foreground mb-4 h-12 w-12' />
              <p className='text-muted-foreground'>No active sessions found</p>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
