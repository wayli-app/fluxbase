import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useState, useEffect } from 'react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import {
  InputOTP,
  InputOTPGroup,
  InputOTPSlot,
  InputOTPSeparator,
} from '@/components/ui/input-otp'
import { dashboardAuthAPI } from '@/lib/api'
import { setAuthToken } from '@/lib/fluxbase-client'
import { useAuthStore } from '@/stores/auth-store'

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

      // Clear the stored user_id
      sessionStorage.removeItem('2fa_user_id')

      // Store access token in Zustand auth store
      auth.setAccessToken(response.access_token)

      // Store tokens and user in localStorage
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
        ? (error as { response?: { data?: { error?: string } } }).response?.data?.error || 'Invalid verification code'
        : 'Invalid verification code'
      toast.error('Verification failed', {
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
          <h1 className='mt-6 text-3xl font-bold tracking-tight'>Two-Factor Authentication</h1>
          <p className='mt-2 text-sm text-muted-foreground'>
            Enter the 6-digit code from your authenticator app
          </p>
        </div>

        {/* OTP Form */}
        <Card>
          <CardHeader>
            <CardTitle>Verification Code</CardTitle>
            <CardDescription>Open your authenticator app to view your code</CardDescription>
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

              <Button type='submit' className='w-full' disabled={code.length !== 6 || isLoading}>
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
