import { useState, useEffect } from 'react'
import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { toast } from 'sonner'
import { useAuthStore } from '@/stores/auth-store'
import { dashboardAuthAPI } from '@/lib/api'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  InputOTP,
  InputOTPGroup,
  InputOTPSlot,
  InputOTPSeparator,
} from '@/components/ui/input-otp'

export const Route = createFileRoute('/login/otp')({
  component: OtpPage,
})

function OtpPage() {
  const navigate = useNavigate()
  const { auth } = useAuthStore()
  const [isLoading, setIsLoading] = useState(false)
  const [code, setCode] = useState('')
  const [userId, setUserId] = useState<string | null>(null)

  useEffect(() => {
    // Get user_id from session storage
    const storedUserId = sessionStorage.getItem('2fa_user_id')
    if (!storedUserId) {
      toast.error('Session expired', {
        description: 'Please log in again.',
      })
      navigate({ to: '/login' })
      return
    }
    setUserId(storedUserId)
  }, [navigate])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    if (!userId) {
      toast.error('Session expired', {
        description: 'Please log in again.',
      })
      navigate({ to: '/login' })
      return
    }

    if (code.length !== 6) {
      toast.error('Invalid code', {
        description: 'Please enter a 6-digit code.',
      })
      return
    }

    setIsLoading(true)

    try {
      const response = await dashboardAuthAPI.verify2FA({
        user_id: userId,
        code: code,
      })

      sessionStorage.removeItem('2fa_user_id')

      auth.setTokens(response.access_token, response.refresh_token)

      auth.setUser({
        accountNo: response.user.id,
        email: response.user.email,
        role: [response.user.role || 'tenant_admin'],
        exp: Date.now() + response.expires_in * 1000,
      })

      localStorage.setItem('fluxbase_admin_user', JSON.stringify(response.user))

      toast.success('Welcome back!', {
        description: 'You have successfully logged in.',
      })

      // Redirect to dashboard
      navigate({ to: '/' })
    } catch (error: unknown) {
      const errorMessage =
        error instanceof Error && 'response' in error
          ? (error as { response?: { data?: { error?: string } } }).response
              ?.data?.error || 'Invalid verification code'
          : 'Invalid verification code'
      toast.error('Verification failed', {
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
            className='mx-auto h-16 w-16 rounded-2xl bg-white/80 backdrop-blur-sm dark:bg-white/10 dark:backdrop-blur-md'
          />
          <h1 className='mt-6 text-3xl font-bold tracking-tight'>
            Two-Factor Authentication
          </h1>
          <p className='text-muted-foreground mt-2 text-sm'>
            Enter the 6-digit code from your authenticator app
          </p>
        </div>

        {/* OTP Form */}
        <Card>
          <CardHeader>
            <CardTitle>Verification Code</CardTitle>
            <CardDescription>
              Open your authenticator app to view your code
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit} className='space-y-6'>
              <div className='flex justify-center'>
                <InputOTP
                  maxLength={6}
                  value={code}
                  onChange={setCode}
                  disabled={isLoading}
                >
                  <InputOTPGroup>
                    <InputOTPSlot index={0} />
                    <InputOTPSlot index={1} />
                    <InputOTPSlot index={2} />
                  </InputOTPGroup>
                  <InputOTPSeparator />
                  <InputOTPGroup>
                    <InputOTPSlot index={3} />
                    <InputOTPSlot index={4} />
                    <InputOTPSlot index={5} />
                  </InputOTPGroup>
                </InputOTP>
              </div>

              <Button
                type='submit'
                className='w-full'
                disabled={code.length !== 6 || isLoading}
              >
                {isLoading ? 'Verifying...' : 'Verify'}
              </Button>

              <div className='text-center'>
                <Button
                  type='button'
                  variant='link'
                  onClick={() => {
                    sessionStorage.removeItem('2fa_user_id')
                    navigate({ to: '/login' })
                  }}
                >
                  Back to login
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
