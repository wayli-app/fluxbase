import { useState, useCallback } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import {
  useImpersonationStore,
  type ImpersonationType,
  type ImpersonatedUser,
} from '@/stores/impersonation-store'
import { impersonationApi } from '@/lib/impersonation-api'
import { syncAuthToken } from '@/lib/fluxbase-client'

export interface UseImpersonationOptions {
  /** Default reason for audit trail when not requiring user input */
  defaultReason?: string
  /** Callback fired after impersonation starts successfully */
  onStart?: () => void
  /** Callback fired after impersonation stops */
  onStop?: () => void
}

export interface UseImpersonationReturn {
  // State
  isImpersonating: boolean
  impersonationType: ImpersonationType | null
  impersonatedUser: ImpersonatedUser | null
  isLoading: boolean

  // Actions
  startUserImpersonation: (
    userId: string,
    userEmail: string,
    reason?: string
  ) => Promise<void>
  startAnonImpersonation: (reason?: string) => Promise<void>
  startServiceImpersonation: (reason?: string) => Promise<void>
  stopImpersonation: () => Promise<void>

  // Helpers
  getDisplayLabel: () => string | null
}

export function useImpersonation(
  options: UseImpersonationOptions = {}
): UseImpersonationReturn {
  const { defaultReason = 'Admin impersonation', onStart, onStop } = options

  const [isLoading, setIsLoading] = useState(false)
  const queryClient = useQueryClient()

  const {
    isImpersonating,
    impersonationType,
    impersonatedUser,
    stopImpersonation: clearImpersonationStore,
  } = useImpersonationStore()

  const startUserImpersonation = useCallback(
    async (userId: string, userEmail: string, reason?: string) => {
      try {
        setIsLoading(true)
        const response = await impersonationApi.startUserImpersonation(
          userId,
          reason || defaultReason
        )

        useImpersonationStore.getState().startImpersonation(
          response.access_token,
          response.refresh_token,
          response.target_user,
          response.session,
          'user'
        )

        syncAuthToken()
        queryClient.invalidateQueries()
        toast.success(`Now impersonating ${userEmail}`)
        onStart?.()
      } catch {
        toast.error('Failed to start impersonation')
      } finally {
        setIsLoading(false)
      }
    },
    [defaultReason, onStart, queryClient]
  )

  const startAnonImpersonation = useCallback(
    async (reason?: string) => {
      try {
        setIsLoading(true)
        const response = await impersonationApi.startAnonImpersonation(
          reason || defaultReason
        )

        useImpersonationStore.getState().startImpersonation(
          response.access_token,
          response.refresh_token,
          response.target_user,
          response.session,
          'anon'
        )

        syncAuthToken()
        queryClient.invalidateQueries()
        toast.success('Now impersonating Anonymous user')
        onStart?.()
      } catch {
        toast.error('Failed to start anonymous impersonation')
      } finally {
        setIsLoading(false)
      }
    },
    [defaultReason, onStart, queryClient]
  )

  const startServiceImpersonation = useCallback(
    async (reason?: string) => {
      try {
        setIsLoading(true)
        const response = await impersonationApi.startServiceImpersonation(
          reason || defaultReason
        )

        useImpersonationStore.getState().startImpersonation(
          response.access_token,
          response.refresh_token,
          response.target_user,
          response.session,
          'service'
        )

        syncAuthToken()
        queryClient.invalidateQueries()
        toast.success('Now impersonating Service Role')
        onStart?.()
      } catch {
        toast.error('Failed to start service role impersonation')
      } finally {
        setIsLoading(false)
      }
    },
    [defaultReason, onStart, queryClient]
  )

  const stopImpersonation = useCallback(async () => {
    try {
      await impersonationApi.stopImpersonation()
      toast.success('Stopped impersonation')
    } catch {
      toast.info('Cleared impersonation (session may have already expired)')
    } finally {
      clearImpersonationStore()
      syncAuthToken()
      queryClient.invalidateQueries()
      onStop?.()
    }
  }, [clearImpersonationStore, onStop, queryClient])

  const getDisplayLabel = useCallback(() => {
    if (!isImpersonating) return null
    switch (impersonationType) {
      case 'anon':
        return 'Anonymous'
      case 'service':
        return 'Service Role'
      default:
        return impersonatedUser?.email || 'Unknown User'
    }
  }, [isImpersonating, impersonationType, impersonatedUser])

  return {
    isImpersonating,
    impersonationType,
    impersonatedUser,
    isLoading,
    startUserImpersonation,
    startAnonImpersonation,
    startServiceImpersonation,
    stopImpersonation,
    getDisplayLabel,
  }
}
