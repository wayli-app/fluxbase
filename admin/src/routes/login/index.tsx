import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useState, useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { PasswordInput } from '@/components/password-input'
import { Separator } from '@/components/ui/separator'
import { dashboardAuthAPI, type SSOProvider } from '@/lib/api'
import { setAuthToken } from '@/lib/fluxbase-client'
import { useAuthStore } from '@/stores/auth-store'
import { KeyRound, Shield } from 'lucide-react'

export const Route = createFileRoute('/login/')({
  component: LoginPage,
})

function LoginPage() {
  const navigate = useNavigate()
  const { auth } = useAuthStore()
  const [isLoading, setIsLoading] = useState(false)
  const [formData, setFormData] = useState({
    email: '',
    password: '',
  })

  // Fetch SSO providers available for dashboard login
  const { data: ssoData } = useQuery({
    queryKey: ['sso-providers'],
    queryFn: () => dashboardAuthAPI.getSSOProviders(),
    staleTime: 5 * 60 * 1000, // 5 minutes
    retry: false, // Don't retry on failure - SSO is optional
  })

  const ssoProviders = ssoData?.providers || []

  // Show error from URL (e.g., SSO callback error)
  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    const urlError = params.get('error')
    if (urlError) {
      toast.error('Authentication Error', {
        description: urlError,
      })
      // Clear the error from URL
      window.history.replaceState({}, '', '/login')
    }
  }, [])

  // Redirect to OTP page if there's a pending 2FA session
  useEffect(() => {
    const pendingUserId = sessionStorage.getItem('2fa_user_id')
    if (pendingUserId) {
      navigate({ to: '/login/otp' })
    }
  }, [navigate])

  // Handle SSO login
  const handleSSOLogin = (provider: SSOProvider) => {
    const baseURL = window.__FLUXBASE_CONFIG__?.publicBaseURL || import.meta.env.VITE_API_URL || ''
    const redirectTo = '/'

    if (provider.type === 'oauth') {
      // Redirect to OAuth login endpoint
      window.location.href = `${baseURL}/dashboard/auth/sso/oauth/${provider.id}?redirect_to=${encodeURIComponent(redirectTo)}`
    } else if (provider.type === 'saml') {
      // Redirect to SAML login endpoint
      window.location.href = `${baseURL}/dashboard/auth/sso/saml/${provider.id}?redirect_to=${encodeURIComponent(redirectTo)}`
    }
  }

  // Get icon for SSO provider
  const getSSOProviderIcon = (provider: SSOProvider) => {
    if (provider.type === 'saml') {
      return <Shield className="h-4 w-4 mr-2" />
    }
    // For OAuth providers, we could add specific icons for each provider
    return <KeyRound className="h-4 w-4 mr-2" />
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    if (!formData.email || !formData.password) {
      toast.error('Validation Error', {
        description: 'Please enter both email and password',
      })
      return
    }

    setIsLoading(true)

    try {
      const response = await dashboardAuthAPI.login({
        email: formData.email,
        password: formData.password,
      })

      // Check if 2FA is required
      if (response.requires_2fa && response.user_id) {
        // Store user_id for OTP verification
        sessionStorage.setItem('2fa_user_id', response.user_id)
        toast.info('Two-factor authentication required')
        navigate({ to: '/login/otp' })
        return
      }

      // Store access token in Zustand auth store (also sets cookie and syncs SDK)
      auth.setAccessToken(response.access_token)

      // Store tokens and user in localStorage (for route guards and persistence)
      localStorage.setItem('fluxbase_admin_access_token', response.access_token)
      localStorage.setItem('fluxbase_admin_refresh_token', response.refresh_token)
      localStorage.setItem('fluxbase_admin_user', JSON.stringify(response.user))

      // Also set token in Fluxbase SDK
      setAuthToken(response.access_token)

      toast.success('Welcome back!', {
        description: 'You have successfully logged in.',
      })

      // Redirect to dashboard
      navigate({ to: '/' })
    } catch (error: unknown) {
      const errorMessage = error instanceof Error && 'response' in error
        ? (error as { response?: { data?: { error?: string } } }).response?.data?.error || 'Invalid email or password'
        : 'Invalid email or password'
      toast.error('Login failed', {
        description: errorMessage,
      })
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className='flex min-h-screen flex-col items-center justify-center bg-gradient-to-br from-background to-muted p-4'>
      <div className='w-full max-w-md space-y-8'>
        {/* Logo and Title */}
        <div className='text-center'>
          <img
            src='/admin/images/logo-icon.svg'
            alt='Fluxbase'
            className='mx-auto h-16 w-16'
          />
          <h1 className='mt-6 text-3xl font-bold tracking-tight'>Admin Login</h1>
          <p className='mt-2 text-sm text-muted-foreground'>
            Sign in to access your Fluxbase admin panel
          </p>
        </div>

        {/* Login Form */}
        <Card>
          <CardHeader>
            <CardTitle>Sign In</CardTitle>
            <CardDescription>Enter your admin credentials to continue</CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit} className='space-y-4'>
              <div className='space-y-2'>
                <Label htmlFor='email'>Email</Label>
                <Input
                  id='email'
                  type='email'
                  placeholder='admin@example.com'
                  value={formData.email}
                  onChange={(e) => setFormData({ ...formData, email: e.target.value })}
                  disabled={isLoading}
                  autoComplete='email'
                  autoFocus
                />
              </div>

              <div className='space-y-2'>
                <Label htmlFor='password'>Password</Label>
                <PasswordInput
                  id='password'
                  placeholder='Enter your password'
                  value={formData.password}
                  onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                  disabled={isLoading}
                  autoComplete='current-password'
                />
              </div>

              <Button type='submit' className='w-full' disabled={isLoading}>
                {isLoading ? 'Signing in...' : 'Sign In'}
              </Button>

              {/* SSO Login Options */}
              {ssoProviders.length > 0 && (
                <>
                  <div className="relative my-4">
                    <div className="absolute inset-0 flex items-center">
                      <Separator className="w-full" />
                    </div>
                    <div className="relative flex justify-center text-xs uppercase">
                      <span className="bg-card px-2 text-muted-foreground">
                        Or continue with
                      </span>
                    </div>
                  </div>

                  <div className="space-y-2">
                    {ssoProviders.map((provider) => (
                      <Button
                        key={provider.id}
                        type="button"
                        variant="outline"
                        className="w-full"
                        onClick={() => handleSSOLogin(provider)}
                        disabled={isLoading}
                      >
                        {getSSOProviderIcon(provider)}
                        Continue with {provider.name}
                      </Button>
                    ))}
                  </div>
                </>
              )}
            </form>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
