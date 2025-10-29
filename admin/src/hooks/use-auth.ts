import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { toast } from 'sonner'
import { useFluxbaseClient } from '@fluxbase/sdk-react'
import type { SignInCredentials, SignUpCredentials } from '@fluxbase/sdk'
import { useAuthStore } from '@/stores/auth-store'

export function useAuth() {
  const { auth } = useAuthStore()
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const client = useFluxbaseClient()

  // Fetch current user data
  const { data: user, isLoading: isLoadingUser } = useQuery({
    queryKey: ['auth', 'user'],
    queryFn: async () => {
      return await client.auth.getCurrentUser()
    },
    enabled: !!auth.accessToken,
    retry: false,
    staleTime: 5 * 60 * 1000, // 5 minutes
  })

  // Sign in mutation
  const signInMutation = useMutation({
    mutationFn: async (data: SignInCredentials) => {
      return await client.auth.signIn(data)
    },
    onSuccess: (session) => {
      // Store tokens
      auth.setAccessToken(session.access_token)
      localStorage.setItem('refresh_token', session.refresh_token)

      // Store user in Zustand
      auth.setUser({
        accountNo: session.user.id,
        email: session.user.email,
        role: [session.user.role],
        exp: Date.now() + session.expires_in * 1000,
      })

      // Invalidate and refetch user query
      queryClient.invalidateQueries({ queryKey: ['auth', 'user'] })

      toast.success(`Welcome back, ${session.user.email}!`)
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
    onSuccess: (session) => {
      // Store tokens
      auth.setAccessToken(session.access_token)
      localStorage.setItem('refresh_token', session.refresh_token)

      // Store user in Zustand
      auth.setUser({
        accountNo: session.user.id,
        email: session.user.email,
        role: [session.user.role],
        exp: Date.now() + session.expires_in * 1000,
      })

      // Invalidate and refetch user query
      queryClient.invalidateQueries({ queryKey: ['auth', 'user'] })

      toast.success(`Account created successfully! Welcome, ${session.user.email}!`)
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
    user: user || auth.user,
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
