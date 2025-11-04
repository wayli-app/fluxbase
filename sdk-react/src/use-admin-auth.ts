import { useState, useEffect, useCallback } from 'react'
import { useFluxbaseClient } from './context'
import type { AdminAuthResponse } from '@fluxbase/sdk'

/**
 * Simplified admin user type returned by authentication
 */
export interface AdminUser {
  id: string
  email: string
  role: string
}

export interface UseAdminAuthOptions {
  /**
   * Automatically check authentication status on mount
   * @default true
   */
  autoCheck?: boolean
}

export interface UseAdminAuthReturn {
  /**
   * Current admin user if authenticated
   */
  user: AdminUser | null

  /**
   * Whether the admin is authenticated
   */
  isAuthenticated: boolean

  /**
   * Whether the authentication check is in progress
   */
  isLoading: boolean

  /**
   * Any error that occurred during authentication
   */
  error: Error | null

  /**
   * Login as admin
   */
  login: (email: string, password: string) => Promise<AdminAuthResponse>

  /**
   * Logout admin
   */
  logout: () => Promise<void>

  /**
   * Refresh admin user info
   */
  refresh: () => Promise<void>
}

/**
 * Hook for admin authentication
 *
 * Manages admin login state, authentication checks, and user info.
 *
 * @example
 * ```tsx
 * function AdminLogin() {
 *   const { user, isAuthenticated, isLoading, login, logout } = useAdminAuth()
 *
 *   const handleLogin = async (e: React.FormEvent) => {
 *     e.preventDefault()
 *     await login(email, password)
 *   }
 *
 *   if (isLoading) return <div>Loading...</div>
 *   if (isAuthenticated) return <div>Welcome {user?.email}</div>
 *
 *   return <form onSubmit={handleLogin}>...</form>
 * }
 * ```
 */
export function useAdminAuth(options: UseAdminAuthOptions = {}): UseAdminAuthReturn {
  const { autoCheck = true } = options
  const client = useFluxbaseClient()

  const [user, setUser] = useState<AdminUser | null>(null)
  const [isLoading, setIsLoading] = useState(autoCheck)
  const [error, setError] = useState<Error | null>(null)

  /**
   * Check current authentication status
   */
  const checkAuth = useCallback(async () => {
    try {
      setIsLoading(true)
      setError(null)
      const { user } = await client.admin.me()
      setUser(user)
    } catch (err) {
      setUser(null)
      setError(err as Error)
    } finally {
      setIsLoading(false)
    }
  }, [client])

  /**
   * Login as admin
   */
  const login = useCallback(
    async (email: string, password: string): Promise<AdminAuthResponse> => {
      try {
        setIsLoading(true)
        setError(null)
        const response = await client.admin.login({ email, password })
        setUser(response.user)
        return response
      } catch (err) {
        setError(err as Error)
        throw err
      } finally {
        setIsLoading(false)
      }
    },
    [client]
  )

  /**
   * Logout admin
   */
  const logout = useCallback(async (): Promise<void> => {
    try {
      setIsLoading(true)
      setError(null)
      // Clear user state
      setUser(null)
      // Note: Add logout endpoint call here when available
    } catch (err) {
      setError(err as Error)
      throw err
    } finally {
      setIsLoading(false)
    }
  }, [])

  /**
   * Refresh admin user info
   */
  const refresh = useCallback(async (): Promise<void> => {
    await checkAuth()
  }, [checkAuth])

  // Auto-check authentication on mount
  useEffect(() => {
    if (autoCheck) {
      checkAuth()
    }
  }, [autoCheck, checkAuth])

  return {
    user,
    isAuthenticated: user !== null,
    isLoading,
    error,
    login,
    logout,
    refresh
  }
}
