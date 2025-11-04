/**
 * Admin Settings and Management Hooks
 *
 * Hooks for managing application settings, system settings, and webhooks.
 */

import { useState, useEffect, useCallback } from 'react'
import { useFluxbaseClient } from './context'
import type {
  AppSettings,
  UpdateAppSettingsRequest,
  SystemSetting,
  UpdateSystemSettingRequest,
  Webhook,
  CreateWebhookRequest,
  UpdateWebhookRequest
} from '@fluxbase/sdk'

// ============================================================================
// useAppSettings Hook
// ============================================================================

export interface UseAppSettingsOptions {
  autoFetch?: boolean
}

export interface UseAppSettingsReturn {
  settings: AppSettings | null
  isLoading: boolean
  error: Error | null
  refetch: () => Promise<void>
  updateSettings: (update: UpdateAppSettingsRequest) => Promise<void>
}

/**
 * Hook for managing application settings
 *
 * @example
 * ```tsx
 * function SettingsPanel() {
 *   const { settings, isLoading, updateSettings } = useAppSettings({ autoFetch: true })
 *
 *   const handleToggleFeature = async (feature: string, enabled: boolean) => {
 *     await updateSettings({
 *       features: { ...settings?.features, [feature]: enabled }
 *     })
 *   }
 *
 *   return <div>...</div>
 * }
 * ```
 */
export function useAppSettings(options: UseAppSettingsOptions = {}): UseAppSettingsReturn {
  const { autoFetch = true } = options
  const client = useFluxbaseClient()

  const [settings, setSettings] = useState<AppSettings | null>(null)
  const [isLoading, setIsLoading] = useState(autoFetch)
  const [error, setError] = useState<Error | null>(null)

  const fetchSettings = useCallback(async () => {
    try {
      setIsLoading(true)
      setError(null)
      const appSettings = await client.admin.settings.app.get()
      setSettings(appSettings)
    } catch (err) {
      setError(err as Error)
    } finally {
      setIsLoading(false)
    }
  }, [client])

  const updateSettings = useCallback(
    async (update: UpdateAppSettingsRequest): Promise<void> => {
      await client.admin.settings.app.update(update)
      await fetchSettings()
    },
    [client, fetchSettings]
  )

  useEffect(() => {
    if (autoFetch) {
      fetchSettings()
    }
  }, [autoFetch, fetchSettings])

  return {
    settings,
    isLoading,
    error,
    refetch: fetchSettings,
    updateSettings
  }
}

// ============================================================================
// useSystemSettings Hook
// ============================================================================

export interface UseSystemSettingsOptions {
  autoFetch?: boolean
}

export interface UseSystemSettingsReturn {
  settings: SystemSetting[]
  isLoading: boolean
  error: Error | null
  refetch: () => Promise<void>
  getSetting: (key: string) => SystemSetting | undefined
  updateSetting: (key: string, update: UpdateSystemSettingRequest) => Promise<void>
  deleteSetting: (key: string) => Promise<void>
}

/**
 * Hook for managing system settings (key-value storage)
 *
 * @example
 * ```tsx
 * function SystemSettings() {
 *   const { settings, isLoading, updateSetting } = useSystemSettings({ autoFetch: true })
 *
 *   const handleUpdateSetting = async (key: string, value: any) => {
 *     await updateSetting(key, { value })
 *   }
 *
 *   return <div>...</div>
 * }
 * ```
 */
export function useSystemSettings(options: UseSystemSettingsOptions = {}): UseSystemSettingsReturn {
  const { autoFetch = true } = options
  const client = useFluxbaseClient()

  const [settings, setSettings] = useState<SystemSetting[]>([])
  const [isLoading, setIsLoading] = useState(autoFetch)
  const [error, setError] = useState<Error | null>(null)

  const fetchSettings = useCallback(async () => {
    try {
      setIsLoading(true)
      setError(null)
      const response = await client.admin.settings.system.list()
      setSettings(response.settings)
    } catch (err) {
      setError(err as Error)
    } finally {
      setIsLoading(false)
    }
  }, [client])

  const getSetting = useCallback(
    (key: string): SystemSetting | undefined => {
      return settings.find((s) => s.key === key)
    },
    [settings]
  )

  const updateSetting = useCallback(
    async (key: string, update: UpdateSystemSettingRequest): Promise<void> => {
      await client.admin.settings.system.update(key, update)
      await fetchSettings()
    },
    [client, fetchSettings]
  )

  const deleteSetting = useCallback(
    async (key: string): Promise<void> => {
      await client.admin.settings.system.delete(key)
      await fetchSettings()
    },
    [client, fetchSettings]
  )

  useEffect(() => {
    if (autoFetch) {
      fetchSettings()
    }
  }, [autoFetch, fetchSettings])

  return {
    settings,
    isLoading,
    error,
    refetch: fetchSettings,
    getSetting,
    updateSetting,
    deleteSetting
  }
}

// ============================================================================
// useWebhooks Hook
// ============================================================================

export interface UseWebhooksOptions {
  autoFetch?: boolean
  refetchInterval?: number
}

export interface UseWebhooksReturn {
  webhooks: Webhook[]
  isLoading: boolean
  error: Error | null
  refetch: () => Promise<void>
  createWebhook: (webhook: CreateWebhookRequest) => Promise<Webhook>
  updateWebhook: (id: string, update: UpdateWebhookRequest) => Promise<Webhook>
  deleteWebhook: (id: string) => Promise<void>
  testWebhook: (id: string) => Promise<void>
}

/**
 * Hook for managing webhooks
 *
 * @example
 * ```tsx
 * function WebhooksManager() {
 *   const { webhooks, isLoading, createWebhook, deleteWebhook } = useWebhooks({
 *     autoFetch: true
 *   })
 *
 *   const handleCreate = async () => {
 *     await createWebhook({
 *       url: 'https://example.com/webhook',
 *       events: ['user.created', 'user.updated'],
 *       enabled: true
 *     })
 *   }
 *
 *   return <div>...</div>
 * }
 * ```
 */
export function useWebhooks(options: UseWebhooksOptions = {}): UseWebhooksReturn {
  const { autoFetch = true, refetchInterval = 0 } = options
  const client = useFluxbaseClient()

  const [webhooks, setWebhooks] = useState<Webhook[]>([])
  const [isLoading, setIsLoading] = useState(autoFetch)
  const [error, setError] = useState<Error | null>(null)

  const fetchWebhooks = useCallback(async () => {
    try {
      setIsLoading(true)
      setError(null)
      const response = await client.admin.management.webhooks.list()
      setWebhooks(response.webhooks)
    } catch (err) {
      setError(err as Error)
    } finally {
      setIsLoading(false)
    }
  }, [client])

  const createWebhook = useCallback(
    async (webhook: CreateWebhookRequest): Promise<Webhook> => {
      const created = await client.admin.management.webhooks.create(webhook)
      await fetchWebhooks()
      return created
    },
    [client, fetchWebhooks]
  )

  const updateWebhook = useCallback(
    async (id: string, update: UpdateWebhookRequest): Promise<Webhook> => {
      const updated = await client.admin.management.webhooks.update(id, update)
      await fetchWebhooks()
      return updated
    },
    [client, fetchWebhooks]
  )

  const deleteWebhook = useCallback(
    async (id: string): Promise<void> => {
      await client.admin.management.webhooks.delete(id)
      await fetchWebhooks()
    },
    [client, fetchWebhooks]
  )

  const testWebhook = useCallback(
    async (id: string): Promise<void> => {
      await client.admin.management.webhooks.test(id)
    },
    [client]
  )

  useEffect(() => {
    if (autoFetch) {
      fetchWebhooks()
    }

    if (refetchInterval > 0) {
      const interval = setInterval(fetchWebhooks, refetchInterval)
      return () => clearInterval(interval)
    }
  }, [autoFetch, refetchInterval, fetchWebhooks])

  return {
    webhooks,
    isLoading,
    error,
    refetch: fetchWebhooks,
    createWebhook,
    updateWebhook,
    deleteWebhook,
    testWebhook
  }
}
