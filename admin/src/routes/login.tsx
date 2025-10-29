import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useState } from 'react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { PasswordInput } from '@/components/password-input'
import { adminAuthAPI } from '@/lib/api'
import { setTokens } from '@/lib/auth'
import { setAuthToken } from '@/lib/fluxbase-client'
import { Command } from 'lucide-react'

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
      const response = await adminAuthAPI.login({
        email: formData.email,
        password: formData.password,
      })

      // Store tokens
      setTokens(
        {
          access_token: response.access_token,
          refresh_token: response.refresh_token,
          expires_in: response.expires_in,
        },
        response.user
      )

      // Also set token in Fluxbase SDK
      setAuthToken(response.access_token)

      toast.success('Welcome back!', {
        description: 'You have successfully logged in.',
      })

      // Redirect to dashboard
      navigate({ to: '/' })
    } catch (error: any) {
      console.error('Login error:', error)
      const errorMessage = error.response?.data?.error || 'Invalid email or password'
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
          <div className='mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-primary'>
            <Command className='h-8 w-8 text-primary-foreground' />
          </div>
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
