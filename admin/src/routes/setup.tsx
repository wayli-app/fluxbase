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

export const Route = createFileRoute('/setup')({
  component: SetupPage,
})

function SetupPage() {
  const navigate = useNavigate()
  const [isLoading, setIsLoading] = useState(false)
  const [formData, setFormData] = useState({
    name: '',
    email: '',
    password: '',
    confirmPassword: '',
    setupToken: '',
  })
  const [errors, setErrors] = useState<Record<string, string>>({})

  const validateForm = () => {
    const newErrors: Record<string, string> = {}

    if (!formData.setupToken.trim()) {
      newErrors.setupToken = 'Setup token is required'
    }

    if (!formData.name.trim()) {
      newErrors.name = 'Name is required'
    }

    if (!formData.email.trim()) {
      newErrors.email = 'Email is required'
    } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formData.email)) {
      newErrors.email = 'Invalid email address'
    }

    if (!formData.password) {
      newErrors.password = 'Password is required'
    } else if (formData.password.length < 12) {
      newErrors.password = 'Password must be at least 12 characters'
    }

    if (formData.password !== formData.confirmPassword) {
      newErrors.confirmPassword = 'Passwords do not match'
    }

    setErrors(newErrors)
    return Object.keys(newErrors).length === 0
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    if (!validateForm()) {
      return
    }

    setIsLoading(true)

    try {
      const response = await adminAuthAPI.initialSetup({
        name: formData.name,
        email: formData.email,
        password: formData.password,
        setup_token: formData.setupToken,
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

      toast.success('Welcome to Fluxbase!', {
        description: 'Your admin account has been created successfully.',
      })

      // Redirect to dashboard
      navigate({ to: '/' })
    } catch (error: unknown) {
      const errorMessage = error instanceof Error && 'response' in error
        ? (error as { response?: { data?: { error?: string } } }).response?.data?.error || 'Failed to create admin account'
        : 'Failed to create admin account'
      toast.error('Setup failed', {
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
          <h1 className='mt-6 text-3xl font-bold tracking-tight'>Welcome to Fluxbase</h1>
          <p className='mt-2 text-sm text-muted-foreground'>
            Set up your admin account to get started
          </p>
        </div>

        {/* Setup Form */}
        <Card>
          <CardHeader>
            <CardTitle>Create Admin Account</CardTitle>
            <CardDescription>
              This will be the first admin user with full access to your Fluxbase instance.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit} className='space-y-4'>
              <div className='space-y-2'>
                <Label htmlFor='setupToken'>Setup Token</Label>
                <PasswordInput
                  id='setupToken'
                  placeholder='Enter your setup token from deployment config'
                  value={formData.setupToken}
                  onChange={(e) => setFormData({ ...formData, setupToken: e.target.value })}
                  disabled={isLoading}
                />
                {errors.setupToken && (
                  <p className='text-sm text-destructive'>{errors.setupToken}</p>
                )}
                <p className='text-xs text-muted-foreground'>
                  This is the FLUXBASE_SECURITY_SETUP_TOKEN value from your deployment configuration.
                </p>
              </div>

              <div className='space-y-2'>
                <Label htmlFor='name'>Full Name</Label>
                <Input
                  id='name'
                  placeholder='John Doe'
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  disabled={isLoading}
                />
                {errors.name && <p className='text-sm text-destructive'>{errors.name}</p>}
              </div>

              <div className='space-y-2'>
                <Label htmlFor='email'>Email</Label>
                <Input
                  id='email'
                  type='email'
                  placeholder='admin@example.com'
                  value={formData.email}
                  onChange={(e) => setFormData({ ...formData, email: e.target.value })}
                  disabled={isLoading}
                />
                {errors.email && <p className='text-sm text-destructive'>{errors.email}</p>}
              </div>

              <div className='space-y-2'>
                <Label htmlFor='password'>Password</Label>
                <PasswordInput
                  id='password'
                  placeholder='Enter a strong password (min 12 characters)'
                  value={formData.password}
                  onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                  disabled={isLoading}
                />
                {errors.password && (
                  <p className='text-sm text-destructive'>{errors.password}</p>
                )}
              </div>

              <div className='space-y-2'>
                <Label htmlFor='confirmPassword'>Confirm Password</Label>
                <PasswordInput
                  id='confirmPassword'
                  placeholder='Confirm your password'
                  value={formData.confirmPassword}
                  onChange={(e) =>
                    setFormData({ ...formData, confirmPassword: e.target.value })
                  }
                  disabled={isLoading}
                />
                {errors.confirmPassword && (
                  <p className='text-sm text-destructive'>{errors.confirmPassword}</p>
                )}
              </div>

              <Button type='submit' className='w-full' disabled={isLoading}>
                {isLoading ? 'Creating Admin Account...' : 'Complete Setup'}
              </Button>
            </form>
          </CardContent>
        </Card>

        {/* Security Note */}
        <Card className='border-muted-foreground/20'>
          <CardContent>
            <p className='text-xs text-muted-foreground'>
              <strong>Security Note:</strong> This setup page will only be accessible when no users
              exist in the database. After creating your admin account, you'll need to sign in to
              access the admin panel.
            </p>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
