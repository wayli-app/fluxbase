import { useState } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { ArrowLeft, ArrowRight, Loader2 } from 'lucide-react'
import { toast } from 'sonner'
import { dashboardAuthAPI } from '@/lib/api'
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

export function ForgotPassword() {
  const navigate = useNavigate()
  const [isLoading, setIsLoading] = useState(false)
  const [emailSent, setEmailSent] = useState(false)
  const [email, setEmail] = useState('')

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    if (!email) {
      toast.error('Validation Error', {
        description: 'Please enter your email address',
      })
      return
    }

    setIsLoading(true)

    try {
      await dashboardAuthAPI.requestPasswordReset(email)
      setEmailSent(true)
      toast.success(`Password reset email sent to ${email}`)

      // Redirect to login after 3 seconds
      setTimeout(() => {
        navigate({ to: '/login' })
      }, 3000)
    } catch (error) {
      const errorMessage =
        error instanceof Error
          ? error.message
          : 'Failed to send password reset email'
      toast.error(errorMessage)
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
            Forgot Password
          </h1>
          <p className='text-muted-foreground mt-2 text-sm'>
            Enter your email and we'll send you a reset link
          </p>
        </div>

        {/* Forgot Password Form */}
        <Card>
          <CardHeader>
            <CardTitle>Reset Password</CardTitle>
            <CardDescription>
              Enter the email associated with your account
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit} className='space-y-4'>
              <div className='space-y-2'>
                <Label htmlFor='email'>Email</Label>
                <Input
                  id='email'
                  type='email'
                  placeholder='admin@example.com'
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  disabled={isLoading || emailSent}
                  autoComplete='email'
                  autoFocus
                />
              </div>

              {emailSent && (
                <div className='rounded-md bg-green-50 p-3 text-sm text-green-800 dark:bg-green-900/20 dark:text-green-200'>
                  Check your email for a password reset link. Redirecting to
                  login...
                </div>
              )}

              <Button
                type='submit'
                className='w-full'
                disabled={isLoading || emailSent}
              >
                {isLoading ? (
                  <>
                    Sending email...
                    <Loader2 className='ml-2 h-4 w-4 animate-spin' />
                  </>
                ) : emailSent ? (
                  'Email sent'
                ) : (
                  <>
                    Send Reset Link
                    <ArrowRight className='ml-2 h-4 w-4' />
                  </>
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
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
