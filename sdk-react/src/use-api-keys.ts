import { useState, useEffect, useCallback } from 'react'
import { useFluxbaseClient } from './context'
import type { APIKey, CreateAPIKeyRequest } from '@fluxbase/sdk'

export interface UseAPIKeysOptions {
  /**
   * Whether to automatically fetch API keys on mount
   * @default true
   */
  autoFetch?: boolean
}

export interface UseAPIKeysReturn {
  /**
   * Array of API keys
   */
  keys: APIKey[]

  /**
   * Whether keys are being fetched
   */
  isLoading: boolean

  /**
   * Any error that occurred
   */
  error: Error | null

  /**
   * Refetch API keys
   */
  refetch: () => Promise<void>

  /**
   * Create a new API key
   */
  createKey: (request: CreateAPIKeyRequest) => Promise<{ key: string; keyData: APIKey }>

  /**
   * Update an API key
   */
  updateKey: (keyId: string, update: { name?: string; description?: string }) => Promise<void>

  /**
   * Revoke an API key
   */
  revokeKey: (keyId: string) => Promise<void>

  /**
   * Delete an API key
   */
  deleteKey: (keyId: string) => Promise<void>
}

/**
 * Hook for managing API keys
 *
 * Provides API key list and management functions.
 *
 * @example
 * ```tsx
 * function APIKeyManager() {
 *   const { keys, isLoading, createKey, revokeKey } = useAPIKeys()
 *
 *   const handleCreate = async () => {
 *     const { key, keyData } = await createKey({
 *       name: 'Backend Service',
 *       description: 'API key for backend',
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
export function useAPIKeys(options: UseAPIKeysOptions = {}): UseAPIKeysReturn {
  const { autoFetch = true } = options
  const client = useFluxbaseClient()

  const [keys, setKeys] = useState<APIKey[]>([])
  const [isLoading, setIsLoading] = useState(autoFetch)
  const [error, setError] = useState<Error | null>(null)

  /**
   * Fetch API keys from API
   */
  const fetchKeys = useCallback(async () => {
    try {
      setIsLoading(true)
      setError(null)
      const response = await client.admin.management.apiKeys.list()
      setKeys(response.api_keys)
    } catch (err) {
      setError(err as Error)
    } finally {
      setIsLoading(false)
    }
  }, [client])

  /**
   * Create a new API key
   */
  const createKey = useCallback(
    async (request: CreateAPIKeyRequest): Promise<{ key: string; keyData: APIKey }> => {
      const response = await client.admin.management.apiKeys.create(request)
      await fetchKeys() // Refresh list
      return { key: response.key, keyData: response.api_key }
    },
    [client, fetchKeys]
  )

  /**
   * Update an API key
   */
  const updateKey = useCallback(
    async (keyId: string, update: { name?: string; description?: string }): Promise<void> => {
      await client.admin.management.apiKeys.update(keyId, update)
      await fetchKeys() // Refresh list
    },
    [client, fetchKeys]
  )

  /**
   * Revoke an API key
   */
  const revokeKey = useCallback(
    async (keyId: string): Promise<void> => {
      await client.admin.management.apiKeys.revoke(keyId)
      await fetchKeys() // Refresh list
    },
    [client, fetchKeys]
  )

  /**
   * Delete an API key
   */
  const deleteKey = useCallback(
    async (keyId: string): Promise<void> => {
      await client.admin.management.apiKeys.delete(keyId)
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
