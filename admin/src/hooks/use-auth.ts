import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import type { SignUpCredentials } from '@fluxbase/sdk'
import { useFluxbaseClient } from '@fluxbase/sdk-react'
import { toast } from 'sonner'
import { useAuthStore } from '@/stores/auth-store'
import { useImpersonationStore } from '@/stores/impersonation-store'
import { syncAuthToken } from '@/lib/fluxbase-client'
import { dashboardAuthAPI, type DashboardLoginRequest } from '@/lib/api'

export function useAuth() {
  const { auth } = useAuthStore()
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const client = useFluxbaseClient()

  // Fetch current user data from dashboard auth endpoint
  const { data: dashboardUser, isLoading: isLoadingUser } = useQuery({
    queryKey: ['auth', 'user'],
    queryFn: async () => {
      return await dashboardAuthAPI.me()
    },
    enabled: !!auth.accessToken,
    retry: false,
    staleTime: 5 * 60 * 1000, // 5 minutes
  })

  // Use dashboard user (with role) or fall back to Zustand store
  const user = dashboardUser || auth.user

  // Sign in mutation - uses dashboard authentication (dashboard.users)
  const signInMutation = useMutation({
    mutationFn: async (data: DashboardLoginRequest) => {
      return await dashboardAuthAPI.login(data)
    },
    onSuccess: (response) => {
      // Clear any stale impersonation state from previous session
      useImpersonationStore.getState().stopImpersonation()

      // Check if 2FA is required
      if (response.requires_2fa) {
        // Handle 2FA flow - don't store tokens yet
        toast.info('Two-factor authentication required')
        // Navigate to 2FA verification page with user_id
        navigate({ to: '/login/otp' })
        return
      }

      const { access_token, refresh_token, expires_in, user } = response

      // Store tokens
      auth.setAccessToken(access_token)
      localStorage.setItem('refresh_token', refresh_token)

      // Store user in Zustand with dashboard_admin role
      auth.setUser({
        accountNo: user.id,
        email: user.email,
        role: ['dashboard_admin'],
        exp: Date.now() + expires_in * 1000,
      })

      // Sync SDK token with new admin token
      syncAuthToken()

      // Invalidate and refetch user query to get full user data with role
      queryClient.invalidateQueries({ queryKey: ['auth', 'user'] })

      toast.success(`Welcome back, ${user.email}!`)
    },
    onError: (error: unknown) => {
      // Extract error message from axios error response
      const axiosError = error as { response?: { data?: { error?: string } } }
      const message = axiosError.response?.data?.error ||
        (error instanceof Error ? error.message : 'Failed to sign in')
      toast.error(message)
    },
  })

  // Sign up mutation
  const signUpMutation = useMutation({
    mutationFn: async (data: SignUpCredentials) => {
      return await client.auth.signUp(data)
    },
    onSuccess: (response) => {
      if (!response.data) {
        toast.error('Invalid response from server')
        return
      }

      const { session, user } = response.data

      if (!session) {
        toast.error('No session returned from server')
        return
      }

      // Store tokens
      auth.setAccessToken(session.access_token)
      localStorage.setItem('refresh_token', session.refresh_token)

      // Store user in Zustand
      auth.setUser({
        accountNo: user.id,
        email: user.email,
        role: [user.role],
        exp: Date.now() + session.expires_in * 1000,
      })

      // Invalidate and refetch user query
      queryClient.invalidateQueries({ queryKey: ['auth', 'user'] })

      toast.success(`Account created successfully! Welcome, ${user.email}!`)
    },
    onError: (error: Error) => {
      toast.error(error.message || 'Failed to create account')
    },
  })

  // Sign out mutation
  const signOutMutation = useMutation({
    mutationFn: async () => {
      await client.auth.signOut()
    },
    onSuccess: () => {
      // Clear tokens and user data
      auth.reset()
      localStorage.removeItem('refresh_token')

      // Clear all queries
      queryClient.clear()

      // Redirect to login
      navigate({ to: '/login', replace: true })

      toast.success('Signed out successfully')
    },
    onError: (error: Error) => {
      // Even if signout fails on server, clear local data
      auth.reset()
      localStorage.removeItem('refresh_token')
      queryClient.clear()
      navigate({ to: '/login', replace: true })

      toast.error(error.message || 'Failed to sign out')
    },
  })

  return {
    user,
    isAuthenticated: !!auth.accessToken,
    isLoading: isLoadingUser,
    signIn: signInMutation.mutate,
    signUp: signUpMutation.mutate,
    signOut: signOutMutation.mutate,
    isSigningIn: signInMutation.isPending,
    isSigningUp: signUpMutation.isPending,
    isSigningOut: signOutMutation.isPending,
  }
}
