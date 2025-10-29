import { createFileRoute } from '@tanstack/react-router'
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Badge } from '@/components/ui/badge'
import { toast } from 'sonner'
import { Key, Settings, Users, Loader2, X, Check, AlertCircle } from 'lucide-react'
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
import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import { ProfileDropdown } from '@/components/profile-dropdown'
import { Search } from '@/components/search'
import { ThemeSwitch } from '@/components/theme-switch'
import { ConfigDrawer } from '@/components/config-drawer'

export const Route = createFileRoute('/_authenticated/authentication/')({
  component: AuthenticationPage,
})

interface OAuthProvider {
  id: string
  name: string
  enabled: boolean
  clientId: string
  clientSecret: string
  redirectUrl: string
  scopes: string[]
  isCustom?: boolean
  authorizationUrl?: string
  tokenUrl?: string
  userInfoUrl?: string
}

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
    <>
      <Header fixed>
        <Search />
        <div className='ms-auto flex items-center space-x-4'>
          <ThemeSwitch />
          <ConfigDrawer />
          <ProfileDropdown />
        </div>
      </Header>

      <Main>
        <div className='flex flex-1 flex-col gap-4'>
          <div className='flex items-center justify-between'>
            <div>
              <h1 className='text-3xl font-bold'>Authentication</h1>
              <p className='text-muted-foreground'>
                Manage OAuth providers, auth settings, and user sessions
              </p>
            </div>
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
      </Main>
    </>
  )
}

function OAuthProvidersTab() {
  const [showAddProvider, setShowAddProvider] = useState(false)
  const [selectedProvider, setSelectedProvider] = useState<string>('')
  const [customProviderName, setCustomProviderName] = useState('')
  const [customAuthUrl, setCustomAuthUrl] = useState('')
  const [customTokenUrl, setCustomTokenUrl] = useState('')
  const [customUserInfoUrl, setCustomUserInfoUrl] = useState('')

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

  // Mock enabled providers (would come from backend in real implementation)
  const enabledProviders: OAuthProvider[] = [
    {
      id: 'google',
      name: 'Google',
      enabled: true,
      clientId: 'YOUR_GOOGLE_CLIENT_ID',
      clientSecret: '***hidden***',
      redirectUrl: `${window.location.origin}/api/v1/auth/callback/google`,
      scopes: ['openid', 'email', 'profile'],
    },
    {
      id: 'github',
      name: 'GitHub',
      enabled: true,
      clientId: 'YOUR_GITHUB_CLIENT_ID',
      clientSecret: '***hidden***',
      redirectUrl: `${window.location.origin}/api/v1/auth/callback/github`,
      scopes: ['user:email'],
    },
    {
      id: 'okta-custom',
      name: 'Okta (Custom)',
      enabled: true,
      clientId: 'YOUR_OKTA_CLIENT_ID',
      clientSecret: '***hidden***',
      redirectUrl: `${window.location.origin}/api/v1/auth/callback/okta`,
      scopes: ['openid', 'email', 'profile'],
      isCustom: true,
      authorizationUrl: 'https://your-domain.okta.com/oauth2/v1/authorize',
      tokenUrl: 'https://your-domain.okta.com/oauth2/v1/token',
      userInfoUrl: 'https://your-domain.okta.com/oauth2/v1/userinfo',
    },
  ]

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
                          <h3 className='font-semibold text-lg'>{provider.name}</h3>
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
                            <p className='font-mono text-xs break-all'>{provider.clientId}</p>
                          </div>
                          <div>
                            <Label className='text-muted-foreground'>Client Secret</Label>
                            <p className='font-mono text-xs'>{provider.clientSecret}</p>
                          </div>
                          <div className='col-span-2'>
                            <Label className='text-muted-foreground'>Redirect URL</Label>
                            <p className='font-mono text-xs break-all'>{provider.redirectUrl}</p>
                          </div>
                          {provider.isCustom && (
                            <>
                              <div className='col-span-2'>
                                <Label className='text-muted-foreground'>Authorization URL</Label>
                                <p className='font-mono text-xs break-all'>{provider.authorizationUrl}</p>
                              </div>
                              <div className='col-span-2'>
                                <Label className='text-muted-foreground'>Token URL</Label>
                                <p className='font-mono text-xs break-all'>{provider.tokenUrl}</p>
                              </div>
                              <div className='col-span-2'>
                                <Label className='text-muted-foreground'>User Info URL</Label>
                                <p className='font-mono text-xs break-all'>{provider.userInfoUrl}</p>
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
                            toast.info('OAuth provider editing coming soon')
                          }}
                        >
                          Edit
                        </Button>
                        <Button
                          variant='outline'
                          size='sm'
                          onClick={() => {
                            const authUrl = provider.isCustom
                              ? provider.authorizationUrl
                              : `https://accounts.google.com/o/oauth2/v2/auth?client_id=${provider.clientId}&redirect_uri=${encodeURIComponent(provider.redirectUrl)}&response_type=code&scope=${provider.scopes.join(' ')}`
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
                            if (confirm(`Are you sure you want to remove ${provider.name}?`)) {
                              toast.success(`${provider.name} removed (demo mode)`)
                            }
                          }}
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
        <DialogContent className='max-w-2xl'>
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
                  ? `${window.location.origin}/api/v1/auth/callback/${customProviderName.toLowerCase().replace(/\s+/g, '-') || 'custom'}`
                  : `${window.location.origin}/api/v1/auth/callback/${selectedProvider}`}
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
            <Button onClick={() => {
              toast.success('OAuth provider configuration saved (demo mode)')
              setShowAddProvider(false)
            }}>
              Save Provider
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

function AuthSettingsTab() {
  const [settings, setSettings] = useState({
    passwordMinLength: 8,
    passwordRequireUppercase: true,
    passwordRequireNumbers: true,
    passwordRequireSymbols: false,
    sessionTimeout: 24, // hours
    accessTokenExpiry: 15, // minutes
    refreshTokenExpiry: 7, // days
    magicLinkExpiry: 15, // minutes
    emailVerificationRequired: true,
  })

  const handleSaveSettings = () => {
    toast.success('Auth settings saved (demo mode)')
  }

  return (
    <div className='space-y-4'>
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
              value={settings.passwordMinLength}
              onChange={(e) =>
                setSettings({ ...settings, passwordMinLength: parseInt(e.target.value) })
              }
            />
          </div>
          <div className='flex items-center justify-between'>
            <Label htmlFor='uppercase'>Require Uppercase Letters</Label>
            <Switch
              id='uppercase'
              checked={settings.passwordRequireUppercase}
              onCheckedChange={(checked) =>
                setSettings({ ...settings, passwordRequireUppercase: checked })
              }
            />
          </div>
          <div className='flex items-center justify-between'>
            <Label htmlFor='numbers'>Require Numbers</Label>
            <Switch
              id='numbers'
              checked={settings.passwordRequireNumbers}
              onCheckedChange={(checked) =>
                setSettings({ ...settings, passwordRequireNumbers: checked })
              }
            />
          </div>
          <div className='flex items-center justify-between'>
            <Label htmlFor='symbols'>Require Symbols</Label>
            <Switch
              id='symbols'
              checked={settings.passwordRequireSymbols}
              onCheckedChange={(checked) =>
                setSettings({ ...settings, passwordRequireSymbols: checked })
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
            <Label htmlFor='sessionTimeout'>Session Timeout (hours)</Label>
            <Input
              id='sessionTimeout'
              type='number'
              value={settings.sessionTimeout}
              onChange={(e) =>
                setSettings({ ...settings, sessionTimeout: parseInt(e.target.value) })
              }
            />
          </div>
          <div className='grid gap-2'>
            <Label htmlFor='accessToken'>Access Token Expiry (minutes)</Label>
            <Input
              id='accessToken'
              type='number'
              value={settings.accessTokenExpiry}
              onChange={(e) =>
                setSettings({ ...settings, accessTokenExpiry: parseInt(e.target.value) })
              }
            />
          </div>
          <div className='grid gap-2'>
            <Label htmlFor='refreshToken'>Refresh Token Expiry (days)</Label>
            <Input
              id='refreshToken'
              type='number'
              value={settings.refreshTokenExpiry}
              onChange={(e) =>
                setSettings({ ...settings, refreshTokenExpiry: parseInt(e.target.value) })
              }
            />
          </div>
          <div className='grid gap-2'>
            <Label htmlFor='magicLink'>Magic Link Expiry (minutes)</Label>
            <Input
              id='magicLink'
              type='number'
              value={settings.magicLinkExpiry}
              onChange={(e) =>
                setSettings({ ...settings, magicLinkExpiry: parseInt(e.target.value) })
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
              checked={settings.emailVerificationRequired}
              onCheckedChange={(checked) =>
                setSettings({ ...settings, emailVerificationRequired: checked })
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
      const response = await fetch('/api/v1/tables/auth.sessions?select=*,user:user_id(email)', {
        headers: {
          Authorization: `Bearer ${localStorage.getItem('access_token')}`,
        },
      })
      if (!response.ok) throw new Error('Failed to fetch sessions')
      return response.json()
    },
  })

  const revokeSessionMutation = useMutation({
    mutationFn: async (sessionId: string) => {
      const response = await fetch(`/api/v1/tables/auth.sessions/${sessionId}`, {
        method: 'DELETE',
        headers: {
          Authorization: `Bearer ${localStorage.getItem('access_token')}`,
        },
      })
      if (!response.ok) throw new Error('Failed to revoke session')
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
      const response = await fetch(`/api/v1/tables/auth.sessions?user_id=eq.${userId}`, {
        method: 'DELETE',
        headers: {
          Authorization: `Bearer ${localStorage.getItem('access_token')}`,
        },
      })
      if (!response.ok) throw new Error('Failed to revoke sessions')
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
