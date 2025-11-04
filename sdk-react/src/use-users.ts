import { useState, useEffect, useCallback } from 'react'
import { useFluxbaseClient } from './context'
import type { EnrichedUser, ListUsersOptions } from '@fluxbase/sdk'

export interface UseUsersOptions extends ListUsersOptions {
  /**
   * Whether to automatically fetch users on mount
   * @default true
   */
  autoFetch?: boolean

  /**
   * Refetch interval in milliseconds (0 to disable)
   * @default 0
   */
  refetchInterval?: number
}

export interface UseUsersReturn {
  /**
   * Array of users
   */
  users: EnrichedUser[]

  /**
   * Total number of users (for pagination)
   */
  total: number

  /**
   * Whether users are being fetched
   */
  isLoading: boolean

  /**
   * Any error that occurred
   */
  error: Error | null

  /**
   * Refetch users
   */
  refetch: () => Promise<void>

  /**
   * Invite a new user
   */
  inviteUser: (email: string, role: 'user' | 'admin') => Promise<void>

  /**
   * Update user role
   */
  updateUserRole: (userId: string, role: 'user' | 'admin') => Promise<void>

  /**
   * Delete a user
   */
  deleteUser: (userId: string) => Promise<void>

  /**
   * Reset user password
   */
  resetPassword: (userId: string) => Promise<{ message: string }>
}

/**
 * Hook for managing users
 *
 * Provides user list with pagination, search, and management functions.
 *
 * @example
 * ```tsx
 * function UserList() {
 *   const { users, total, isLoading, refetch, inviteUser, deleteUser } = useUsers({
 *     limit: 20,
 *     search: searchTerm
 *   })
 *
 *   return (
 *     <div>
 *       {isLoading ? <Spinner /> : (
 *         <ul>
 *           {users.map(user => (
 *             <li key={user.id}>
 *               {user.email} - {user.role}
 *               <button onClick={() => deleteUser(user.id)}>Delete</button>
 *             </li>
 *           ))}
 *         </ul>
 *       )}
 *     </div>
 *   )
 * }
 * ```
 */
export function useUsers(options: UseUsersOptions = {}): UseUsersReturn {
  const { autoFetch = true, refetchInterval = 0, ...listOptions } = options
  const client = useFluxbaseClient()

  const [users, setUsers] = useState<EnrichedUser[]>([])
  const [total, setTotal] = useState(0)
  const [isLoading, setIsLoading] = useState(autoFetch)
  const [error, setError] = useState<Error | null>(null)

  /**
   * Fetch users from API
   */
  const fetchUsers = useCallback(async () => {
    try {
      setIsLoading(true)
      setError(null)
      const response = await client.admin.listUsers(listOptions)
      setUsers(response.users)
      setTotal(response.total)
    } catch (err) {
      setError(err as Error)
    } finally {
      setIsLoading(false)
    }
  }, [client, JSON.stringify(listOptions)])

  /**
   * Invite a new user
   */
  const inviteUser = useCallback(
    async (email: string, role: 'user' | 'admin'): Promise<void> => {
      await client.admin.inviteUser({ email, role })
      await fetchUsers() // Refresh list
    },
    [client, fetchUsers]
  )

  /**
   * Update user role
   */
  const updateUserRole = useCallback(
    async (userId: string, role: 'user' | 'admin'): Promise<void> => {
      await client.admin.updateUserRole(userId, role)
      await fetchUsers() // Refresh list
    },
    [client, fetchUsers]
  )

  /**
   * Delete a user
   */
  const deleteUser = useCallback(
    async (userId: string): Promise<void> => {
      await client.admin.deleteUser(userId)
      await fetchUsers() // Refresh list
    },
    [client, fetchUsers]
  )

  /**
   * Reset user password
   */
  const resetPassword = useCallback(
    async (userId: string): Promise<{ message: string }> => {
      return await client.admin.resetUserPassword(userId)
    },
    [client]
  )

  // Auto-fetch on mount
  useEffect(() => {
    if (autoFetch) {
      fetchUsers()
    }
  }, [autoFetch, fetchUsers])

  // Set up refetch interval
  useEffect(() => {
    if (refetchInterval > 0) {
      const interval = setInterval(fetchUsers, refetchInterval)
      return () => clearInterval(interval)
    }
  }, [refetchInterval, fetchUsers])

  return {
    users,
    total,
    isLoading,
    error,
    refetch: fetchUsers,
    inviteUser,
    updateUserRole,
    deleteUser,
    resetPassword
  }
}
