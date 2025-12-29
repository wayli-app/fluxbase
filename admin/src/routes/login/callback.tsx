import { createFileRoute } from '@tanstack/react-router'
import { useEffect, useRef } from 'react'
import { toast } from 'sonner'
import { setAuthToken } from '@/lib/fluxbase-client'
import { useAuthStore } from '@/stores/auth-store'

export const Route = createFileRoute('/login/callback')({
  component: SSOCallbackPage,
})

function SSOCallbackPage() {
  const { auth } = useAuthStore()
  const processedRef = useRef(false)

  // Parse search params from URL
  const params = new URLSearchParams(window.location.search)
  const access_token = params.get('access_token')
  const refresh_token = params.get('refresh_token')
  const redirect_to = params.get('redirect_to')
  const error = params.get('error')

  useEffect(() => {
    // Prevent double processing
    if (processedRef.current) return
    processedRef.current = true

    const handleCallback = async () => {
      // Handle error
      if (error) {
        toast.error('SSO Login Failed', {
          description: error,
        })
        window.location.href = '/login'
        return
      }

      // Handle missing tokens
      if (!access_token || !refresh_token) {
        toast.error('SSO Login Failed', {
          description: 'No authentication tokens received',
        })
        window.location.href = '/login'
        return
      }

      try {
        // Store access token in Zustand auth store (also sets cookie and syncs SDK)
        auth.setAccessToken(access_token)

        // Store tokens in localStorage (for route guards and persistence)
        localStorage.setItem('fluxbase_admin_access_token', access_token)
        localStorage.setItem('fluxbase_admin_refresh_token', refresh_token)

        // Also set token in Fluxbase SDK
        setAuthToken(access_token)

        toast.success('Welcome!', {
          description: 'You have successfully logged in via SSO.',
        })

        // Redirect to the intended destination or dashboard
        const destination = redirect_to && redirect_to !== '/' ? redirect_to : '/'
        window.location.href = destination
      } catch {
        toast.error('SSO Login Failed', {
          description: 'Failed to complete authentication',
        })
        window.location.href = '/login'
      }
    }

    handleCallback()
  }, [access_token, refresh_token, redirect_to, error, auth])

  return (
    <div className="flex min-h-screen flex-col items-center justify-center bg-gradient-to-br from-background to-muted p-4">
      <div className="text-center space-y-4">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary mx-auto" />
        <p className="text-muted-foreground">Completing SSO login...</p>
      </div>
    </div>
  )
}
