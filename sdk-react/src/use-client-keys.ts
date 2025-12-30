import { useState, useEffect, useCallback } from 'react'
import { useFluxbaseClient } from './context'
import type { ClientKey, CreateClientKeyRequest } from '@fluxbase/sdk'

export interface UseClientKeysOptions {
  /**
   * Whether to automatically fetch client keys on mount
   * @default true
   */
  autoFetch?: boolean
}

export interface UseClientKeysReturn {
  /**
   * Array of client keys
   */
  keys: ClientKey[]

  /**
   * Whether keys are being fetched
   */
  isLoading: boolean

  /**
   * Any error that occurred
   */
  error: Error | null

  /**
   * Refetch client keys
   */
  refetch: () => Promise<void>

  /**
   * Create a new client key
   */
  createKey: (request: CreateClientKeyRequest) => Promise<{ key: string; keyData: ClientKey }>

  /**
   * Update a client key
   */
  updateKey: (keyId: string, update: { name?: string; description?: string }) => Promise<void>

  /**
   * Revoke a client key
   */
  revokeKey: (keyId: string) => Promise<void>

  /**
   * Delete a client key
   */
  deleteKey: (keyId: string) => Promise<void>
}

/**
 * Hook for managing client keys
 *
 * Provides client key list and management functions.
 *
 * @example
 * ```tsx
 * function ClientKeyManager() {
 *   const { keys, isLoading, createKey, revokeKey } = useClientKeys()
 *
 *   const handleCreate = async () => {
 *     const { key, keyData } = await createKey({
 *       name: 'Backend Service',
 *       description: 'Client key for backend',
 *       expires_at: new Date(Date.now() + 365 * 24 * 60 * 60 * 1000).toISOString()
 *     })
 *     alert(`Key created: ${key}`)
 *   }
 *
 *   return (
 *     <div>
 *       <button onClick={handleCreate}>Create Key</button>
 *       {keys.map(k => (
 *         <div key={k.id}>
 *           {k.name}
 *           <button onClick={() => revokeKey(k.id)}>Revoke</button>
 *         </div>
 *       ))}
 *     </div>
 *   )
 * }
 * ```
 */
export function useClientKeys(options: UseClientKeysOptions = {}): UseClientKeysReturn {
  const { autoFetch = true } = options
  const client = useFluxbaseClient()

  const [keys, setKeys] = useState<ClientKey[]>([])
  const [isLoading, setIsLoading] = useState(autoFetch)
  const [error, setError] = useState<Error | null>(null)

  /**
   * Fetch client keys from API
   */
  const fetchKeys = useCallback(async () => {
    try {
      setIsLoading(true)
      setError(null)
      const response = await client.admin.management.clientKeys.list()
      setKeys(response.client_keys)
    } catch (err) {
      setError(err as Error)
    } finally {
      setIsLoading(false)
    }
  }, [client])

  /**
   * Create a new client key
   */
  const createKey = useCallback(
    async (request: CreateClientKeyRequest): Promise<{ key: string; keyData: ClientKey }> => {
      const response = await client.admin.management.clientKeys.create(request)
      await fetchKeys() // Refresh list
      return { key: response.key, keyData: response.client_key }
    },
    [client, fetchKeys]
  )

  /**
   * Update a client key
   */
  const updateKey = useCallback(
    async (keyId: string, update: { name?: string; description?: string }): Promise<void> => {
      await client.admin.management.clientKeys.update(keyId, update)
      await fetchKeys() // Refresh list
    },
    [client, fetchKeys]
  )

  /**
   * Revoke a client key
   */
  const revokeKey = useCallback(
    async (keyId: string): Promise<void> => {
      await client.admin.management.clientKeys.revoke(keyId)
      await fetchKeys() // Refresh list
    },
    [client, fetchKeys]
  )

  /**
   * Delete a client key
   */
  const deleteKey = useCallback(
    async (keyId: string): Promise<void> => {
      await client.admin.management.clientKeys.delete(keyId)
      await fetchKeys() // Refresh list
    },
    [client, fetchKeys]
  )

  // Auto-fetch on mount
  useEffect(() => {
    if (autoFetch) {
      fetchKeys()
    }
  }, [autoFetch, fetchKeys])

  return {
    keys,
    isLoading,
    error,
    refetch: fetchKeys,
    createKey,
    updateKey,
    revokeKey,
    deleteKey
  }
}

/**
 * @deprecated Use useClientKeys instead
 */
export const useAPIKeys = useClientKeys

/** @deprecated Use UseClientKeysOptions instead */
export type UseAPIKeysOptions = UseClientKeysOptions

/** @deprecated Use UseClientKeysReturn instead */
export type UseAPIKeysReturn = UseClientKeysReturn
