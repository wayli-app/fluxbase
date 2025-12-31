import { useState, useEffect } from 'react'
import { useNavigate, getRouteApi } from '@tanstack/react-router'
import { ArrowLeft, CheckCircle, Loader2, AlertCircle } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { PasswordInput } from '@/components/password-input'
import { PasswordStrength } from '@/components/password-strength'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { dashboardAuthAPI } from '@/lib/api'

const route = getRouteApi('/(auth)/reset-password')

export function ResetPassword() {
  const navigate = useNavigate()
  const search = route.useSearch()
  const [isLoading, setIsLoading] = useState(false)
  const [isValidating, setIsValidating] = useState(true)
  const [tokenValid, setTokenValid] = useState(false)
  const [tokenError, setTokenError] = useState<string | null>(null)
  const [resetSuccess, setResetSuccess] = useState(false)
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')

  // Validate token on mount
  useEffect(() => {
    const validateToken = async () => {
      if (!search.token) {
        setTokenError('No reset token provided')
        setTokenValid(false)
        setIsValidating(false)
        return
      }

      try {
        const result = await dashboardAuthAPI.verifyResetToken(search.token)
        if (result.valid) {
          setTokenValid(true)
          setTokenError(null)
        } else {
          setTokenValid(false)
          setTokenError(result.message || 'Invalid or expired reset token')
        }
      } catch {
        setTokenValid(false)
        setTokenError('Failed to validate reset token')
      } finally {
        setIsValidating(false)
      }
    }

    validateToken()
  }, [search.token])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    if (!search.token) {
      toast.error('No reset token provided')
      return
    }

    if (password.length < 8) {
      toast.error('Password must be at least 8 characters')
      return
    }

    if (password !== confirmPassword) {
      toast.error("Passwords don't match")
      return
    }

    setIsLoading(true)

    try {
      await dashboardAuthAPI.resetPassword(search.token, password)
      setResetSuccess(true)
      toast.success('Password reset successfully!')

      // Redirect to login after 2 seconds
      setTimeout(() => {
        navigate({ to: '/login' })
      }, 2000)
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to reset password'
      toast.error(errorMessage)
    } finally {
      setIsLoading(false)
    }
  }

  const renderContent = () => {
    if (isValidating) {
      return (
        <div className='flex items-center justify-center py-8'>
          <Loader2 className='h-8 w-8 animate-spin text-muted-foreground' />
        </div>
      )
    }

    if (!tokenValid) {
      return (
        <div className='space-y-4'>
          <Alert variant='destructive'>
            <AlertCircle className='h-4 w-4' />
            <AlertDescription>
              {tokenError || 'Invalid or expired reset token. Please request a new password reset.'}
            </AlertDescription>
          </Alert>
          <Button
            type='button'
            variant='ghost'
            className='w-full'
            onClick={() => navigate({ to: '/forgot-password' })}
          >
            <ArrowLeft className='mr-2 h-4 w-4' />
            Request New Reset Link
          </Button>
        </div>
      )
    }

    if (resetSuccess) {
      return (
        <Alert className='border-green-200 bg-green-50 dark:border-green-800 dark:bg-green-900/20'>
          <CheckCircle className='h-4 w-4 text-green-600 dark:text-green-400' />
          <AlertDescription className='text-green-800 dark:text-green-200'>
            Password reset successfully! Redirecting to login...
          </AlertDescription>
        </Alert>
      )
    }

    return (
      <form onSubmit={handleSubmit} className='space-y-4'>
        <div className='space-y-2'>
          <Label htmlFor='password'>New Password</Label>
          <PasswordInput
            id='password'
            placeholder='Create a new secure password'
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            disabled={isLoading}
            autoComplete='new-password'
            autoFocus
          />
          <PasswordStrength password={password} />
        </div>

        <div className='space-y-2'>
          <Label htmlFor='confirmPassword'>Confirm Password</Label>
          <PasswordInput
            id='confirmPassword'
            placeholder='Re-enter your new password'
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
            disabled={isLoading}
            autoComplete='new-password'
          />
        </div>

        <Button type='submit' className='w-full' disabled={isLoading}>
          {isLoading ? (
            <>
              Resetting password...
              <Loader2 className='animate-spin ml-2 h-4 w-4' />
            </>
          ) : (
            'Reset Password'
          )}
        </Button>

        <Button
          type='button'
          variant='ghost'
          className='w-full'
          onClick={() => navigate({ to: '/login' })}
        >
          <ArrowLeft className='mr-2 h-4 w-4' />
          Back to Login
        </Button>
      </form>
    )
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
          <h1 className='mt-6 text-3xl font-bold tracking-tight'>Reset Password</h1>
          <p className='mt-2 text-sm text-muted-foreground'>
            Create a new password for your account
          </p>
        </div>

        {/* Reset Password Form */}
        <Card>
          <CardHeader>
            <CardTitle>New Password</CardTitle>
            <CardDescription>
              Enter your new password below
            </CardDescription>
          </CardHeader>
          <CardContent>
            {renderContent()}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
