import { useState, useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { createFileRoute, useNavigate, Link } from '@tanstack/react-router'
import { KeyRound, Shield } from 'lucide-react'
import { toast } from 'sonner'
import { useAuthStore } from '@/stores/auth-store'
import { dashboardAuthAPI, type SSOProvider } from '@/lib/api'
import { setAuthToken } from '@/lib/fluxbase-client'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'
import { PasswordInput } from '@/components/password-input'

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
  const passwordLoginDisabled = ssoData?.password_login_disabled || false

  // Show error from URL (e.g., SSO callback error)
  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    const urlError = params.get('error')
    if (urlError) {
      toast.error('Authentication Error', {
        description: urlError,
      })
      // Clear the error from URL
      window.history.replaceState({}, '', '/admin/login')
    }
  }, [])

  // Redirect to dashboard if already authenticated
  useEffect(() => {
    const accessToken = localStorage.getItem('fluxbase_admin_access_token')
    if (accessToken && auth.user) {
      // User is already logged in, redirect to dashboard
      const params = new URLSearchParams(window.location.search)
      const redirect = params.get('redirect') || '/'
      navigate({ to: redirect })
    }
  }, [auth.user, navigate])

  // Redirect to OTP page if there's a pending 2FA session
  useEffect(() => {
    const pendingUserId = sessionStorage.getItem('2fa_user_id')
    if (pendingUserId) {
      navigate({ to: '/login/otp' })
    }
  }, [navigate])

  // Handle SSO login
  const handleSSOLogin = (provider: SSOProvider) => {
    const baseURL =
      window.__FLUXBASE_CONFIG__?.publicBaseURL ||
      import.meta.env.VITE_API_URL ||
      ''
    const redirectTo = '/'

    if (provider.type === 'oauth') {
      // Redirect to OAuth login endpoint
      window.location.href = `${baseURL}/api/v1/auth/oauth/${provider.id}/authorize?redirect_to=${encodeURIComponent(redirectTo)}`
    } else if (provider.type === 'saml') {
      // Redirect to SAML login endpoint
      window.location.href = `${baseURL}/api/v1/auth/saml/${provider.id}?redirect_to=${encodeURIComponent(redirectTo)}`
    }
  }

  // Get icon for SSO provider
  const getSSOProviderIcon = (provider: SSOProvider) => {
    const iconClass = 'h-4 w-4 mr-2 shrink-0'

    switch (provider.id.toLowerCase()) {
      case 'google':
        return (
          <svg className={iconClass} viewBox='0 0 24 24'>
            <path
              fill='#4285F4'
              d='M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z'
            />
            <path
              fill='#34A853'
              d='M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z'
            />
            <path
              fill='#FBBC05'
              d='M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z'
            />
            <path
              fill='#EA4335'
              d='M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z'
            />
          </svg>
        )
      case 'github':
        return (
          <svg className={iconClass} viewBox='0 0 24 24' fill='currentColor'>
            <path d='M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z' />
          </svg>
        )
      case 'microsoft':
        return (
          <svg className={iconClass} viewBox='0 0 24 24'>
            <path fill='#F25022' d='M1 1h10v10H1z' />
            <path fill='#00A4EF' d='M1 13h10v10H1z' />
            <path fill='#7FBA00' d='M13 1h10v10H13z' />
            <path fill='#FFB900' d='M13 13h10v10H13z' />
          </svg>
        )
      case 'apple':
        return (
          <svg className={iconClass} viewBox='0 0 24 24' fill='currentColor'>
            <path d='M18.71 19.5c-.83 1.24-1.71 2.45-3.05 2.47-1.34.03-1.77-.79-3.29-.79-1.53 0-2 .77-3.27.82-1.31.05-2.3-1.32-3.14-2.53C4.25 17 2.94 12.45 4.7 9.39c.87-1.52 2.43-2.48 4.12-2.51 1.28-.02 2.5.87 3.29.87.78 0 2.26-1.07 3.81-.91.65.03 2.47.26 3.64 1.98-.09.06-2.17 1.28-2.15 3.81.03 3.02 2.65 4.03 2.68 4.04-.03.07-.42 1.44-1.38 2.83M13 3.5c.73-.83 1.94-1.46 2.94-1.5.13 1.17-.34 2.35-1.04 3.19-.69.85-1.83 1.51-2.95 1.42-.15-1.15.41-2.35 1.05-3.11z' />
          </svg>
        )
      case 'facebook':
        return (
          <svg className={iconClass} viewBox='0 0 24 24' fill='#1877F2'>
            <path d='M24 12.073c0-6.627-5.373-12-12-12s-12 5.373-12 12c0 5.99 4.388 10.954 10.125 11.854v-8.385H7.078v-3.47h3.047V9.43c0-3.007 1.792-4.669 4.533-4.669 1.312 0 2.686.235 2.686.235v2.953H15.83c-1.491 0-1.956.925-1.956 1.874v2.25h3.328l-.532 3.47h-2.796v8.385C19.612 23.027 24 18.062 24 12.073z' />
          </svg>
        )
      case 'gitlab':
        return (
          <svg className={iconClass} viewBox='0 0 24 24'>
            <path fill='#E24329' d='m12 22.135-4.782-14.72h9.564z' />
            <path fill='#FC6D26' d='m12 22.135-4.782-14.72H1.032z' />
            <path
              fill='#FCA326'
              d='M1.032 7.415.067 10.383a.657.657 0 0 0 .238.734L12 22.135z'
            />
            <path
              fill='#E24329'
              d='M1.032 7.415h6.186L4.861.923a.328.328 0 0 0-.624 0z'
            />
            <path fill='#FC6D26' d='m12 22.135 4.782-14.72h6.186z' />
            <path
              fill='#FCA326'
              d='m22.968 7.415.965 2.968a.657.657 0 0 1-.238.734L12 22.135z'
            />
            <path
              fill='#E24329'
              d='M22.968 7.415h-6.186l2.357-6.492a.328.328 0 0 1 .624 0z'
            />
          </svg>
        )
      case 'bitbucket':
        return (
          <svg className={iconClass} viewBox='0 0 24 24' fill='#2684FF'>
            <path d='M.778 1.211a.768.768 0 0 0-.768.892l3.263 19.81c.084.5.515.868 1.022.873H19.95a.772.772 0 0 0 .77-.646l3.27-20.03a.768.768 0 0 0-.768-.891zM14.52 15.53H9.522L8.17 8.466h7.561z' />
          </svg>
        )
      case 'linkedin':
        return (
          <svg className={iconClass} viewBox='0 0 24 24' fill='#0A66C2'>
            <path d='M20.447 20.452h-3.554v-5.569c0-1.328-.027-3.037-1.852-3.037-1.853 0-2.136 1.445-2.136 2.939v5.667H9.351V9h3.414v1.561h.046c.477-.9 1.637-1.85 3.37-1.85 3.601 0 4.267 2.37 4.267 5.455v6.286zM5.337 7.433c-1.144 0-2.063-.926-2.063-2.065 0-1.138.92-2.063 2.063-2.063 1.14 0 2.064.925 2.064 2.063 0 1.139-.925 2.065-2.064 2.065zm1.782 13.019H3.555V9h3.564v11.452zM22.225 0H1.771C.792 0 0 .774 0 1.729v20.542C0 23.227.792 24 1.771 24h20.451C23.2 24 24 23.227 24 22.271V1.729C24 .774 23.2 0 22.222 0h.003z' />
          </svg>
        )
      case 'twitter':
      case 'x':
        return (
          <svg className={iconClass} viewBox='0 0 24 24' fill='currentColor'>
            <path d='M18.244 2.25h3.308l-7.227 8.26 8.502 11.24H16.17l-5.214-6.817L4.99 21.75H1.68l7.73-8.835L1.254 2.25H8.08l4.713 6.231zm-1.161 17.52h1.833L7.084 4.126H5.117z' />
          </svg>
        )
      default:
        // Fallback icons
        if (provider.type === 'saml') {
          return <Shield className={iconClass} />
        }
        return <KeyRound className={iconClass} />
    }
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
      localStorage.setItem(
        'fluxbase_admin_refresh_token',
        response.refresh_token
      )
      localStorage.setItem('fluxbase_admin_user', JSON.stringify(response.user))

      // Also set token in Fluxbase SDK
      setAuthToken(response.access_token)

      toast.success('Welcome back!', {
        description: 'You have successfully logged in.',
      })

      // Redirect to dashboard
      navigate({ to: '/' })
    } catch (error: unknown) {
      const errorMessage =
        error instanceof Error && 'response' in error
          ? (error as { response?: { data?: { error?: string } } }).response
              ?.data?.error || 'Invalid email or password'
          : 'Invalid email or password'
      toast.error('Login failed', {
        description: errorMessage,
      })
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className='from-background to-muted flex min-h-screen flex-col items-center justify-center bg-gradient-to-br p-4'>
      <div className='w-full max-w-md space-y-8'>
        {/* Logo and Title */}
        <div className='text-center'>
          <img
            src='/admin/images/logo-icon.svg'
            alt='Fluxbase'
            className='mx-auto h-16 w-16'
          />
          <h1 className='mt-6 text-3xl font-bold tracking-tight'>
            Admin Login
          </h1>
          <p className='text-muted-foreground mt-2 text-sm'>
            Sign in to access your Fluxbase admin panel
          </p>
        </div>

        {/* Login Form */}
        <Card>
          <CardHeader>
            <CardTitle>Sign In</CardTitle>
            <CardDescription>
              {passwordLoginDisabled
                ? "Use your organization's SSO to sign in"
                : 'Enter your admin credentials to continue'}
            </CardDescription>
          </CardHeader>
          <CardContent>
            {/* Password Form - only show if not disabled */}
            {!passwordLoginDisabled && (
              <form onSubmit={handleSubmit} className='space-y-4'>
                <div className='space-y-2'>
                  <Label htmlFor='email'>Email</Label>
                  <Input
                    id='email'
                    type='email'
                    placeholder='admin@example.com'
                    value={formData.email}
                    onChange={(e) =>
                      setFormData({ ...formData, email: e.target.value })
                    }
                    disabled={isLoading}
                    autoComplete='email'
                    autoFocus
                  />
                </div>

                <div className='space-y-2'>
                  <div className='flex items-center justify-between'>
                    <Label htmlFor='password'>Password</Label>
                    <Link
                      to='/forgot-password'
                      className='text-muted-foreground hover:text-primary text-sm'
                    >
                      Forgot password?
                    </Link>
                  </div>
                  <PasswordInput
                    id='password'
                    placeholder='Enter your password'
                    value={formData.password}
                    onChange={(e) =>
                      setFormData({ ...formData, password: e.target.value })
                    }
                    disabled={isLoading}
                    autoComplete='current-password'
                  />
                </div>

                <Button type='submit' className='w-full' disabled={isLoading}>
                  {isLoading ? 'Signing in...' : 'Sign In'}
                </Button>
              </form>
            )}

            {/* SSO Login Options */}
            {ssoProviders.length > 0 && (
              <>
                {!passwordLoginDisabled && (
                  <div className='relative my-4'>
                    <div className='absolute inset-0 flex items-center'>
                      <Separator className='w-full' />
                    </div>
                    <div className='relative flex justify-center text-xs uppercase'>
                      <span className='bg-card text-muted-foreground px-2'>
                        Or continue with
                      </span>
                    </div>
                  </div>
                )}

                <div
                  className={`space-y-2 ${passwordLoginDisabled ? '' : 'mt-4'}`}
                >
                  {ssoProviders.map((provider) => (
                    <Button
                      key={provider.id}
                      type='button'
                      variant={passwordLoginDisabled ? 'default' : 'outline'}
                      className='w-full'
                      onClick={() => handleSSOLogin(provider)}
                      disabled={isLoading}
                    >
                      {getSSOProviderIcon(provider)}
                      {provider.name}
                    </Button>
                  ))}
                </div>
              </>
            )}

            {/* Show error message if password disabled but no SSO providers */}
            {passwordLoginDisabled && ssoProviders.length === 0 && (
              <div className='rounded-md border border-red-200 bg-red-50 p-4 text-center dark:border-red-800 dark:bg-red-950'>
                <p className='text-sm text-red-800 dark:text-red-200'>
                  Password login is disabled but no SSO providers are available.
                  Contact your administrator.
                </p>
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
