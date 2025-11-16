import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import type { SignInCredentials, SignUpCredentials } from '@fluxbase/sdk'
import { useFluxbaseClient } from '@fluxbase/sdk-react'
import { toast } from 'sonner'
import { useAuthStore } from '@/stores/auth-store'

export function useAuth() {
  const { auth } = useAuthStore()
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const client = useFluxbaseClient()

  // Fetch current user data
  const { data: userResponse, isLoading: isLoadingUser } = useQuery({
    queryKey: ['auth', 'user'],
    queryFn: async () => {
      return await client.auth.getCurrentUser()
    },
    enabled: !!auth.accessToken,
    retry: false,
    staleTime: 5 * 60 * 1000, // 5 minutes
  })

  // Extract user from response
  const user = userResponse?.data?.user || auth.user

  // Sign in mutation
  const signInMutation = useMutation({
    mutationFn: async (data: SignInCredentials) => {
      return await client.auth.signIn(data)
    },
    onSuccess: (response) => {
      // Check if 2FA is required
      if (
        response.data &&
        'requires_2fa' in response.data &&
        response.data.requires_2fa
      ) {
        // Handle 2FA flow - don't store tokens yet
        toast.info(
          'message' in response.data
            ? response.data.message
            : 'Two-factor authentication required'
        )
        return
      }

      // Type guard: at this point we know it's an AuthResponseData
      if (!response.data || !('session' in response.data)) {
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

      toast.success(`Welcome back, ${user.email}!`)
    },
    onError: (error: Error) => {
      toast.error(error.message || 'Failed to sign in')
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

      // Redirect to sign in
      navigate({ to: '/sign-in', replace: true })

      toast.success('Signed out successfully')
    },
    onError: (error: Error) => {
      // Even if signout fails on server, clear local data
      auth.reset()
      localStorage.removeItem('refresh_token')
      queryClient.clear()
      navigate({ to: '/sign-in', replace: true })

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
