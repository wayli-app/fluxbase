import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useState } from 'react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { PasswordInput } from '@/components/password-input'
import { dashboardAuthAPI } from '@/lib/api'
import { setAuthToken } from '@/lib/fluxbase-client'

export const Route = createFileRoute('/login')({
  component: LoginPage,
})

function LoginPage() {
  const navigate = useNavigate()
  const [isLoading, setIsLoading] = useState(false)
  const [formData, setFormData] = useState({
    email: '',
    password: '',
  })

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

      // Store access token and refresh token in localStorage
      localStorage.setItem('access_token', response.access_token)
      localStorage.setItem('fluxbase_admin_refresh_token', response.refresh_token)
      localStorage.setItem('user', JSON.stringify(response.user))

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
            </form>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
