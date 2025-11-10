import { useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { AlertTriangle, X } from 'lucide-react'
import { toast } from 'sonner'
import { getAccessToken } from '@/lib/auth'
import { setAuthToken as setSDKAuthToken } from '@/lib/fluxbase-client'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { impersonationApi } from '../lib/impersonation-api'
import { useImpersonationStore } from '../stores/impersonation-store'

export function ImpersonationBanner() {
  const {
    isImpersonating,
    impersonatedUser,
    impersonationType,
    stopImpersonation,
  } = useImpersonationStore()
  const queryClient = useQueryClient()
  const [isStopping, setIsStopping] = useState(false)

  const handleStopImpersonation = async () => {
    try {
      setIsStopping(true)
      await impersonationApi.stopImpersonation()
      stopImpersonation()

      // Reset SDK client token to admin token
      const adminToken = getAccessToken()
      if (adminToken) {
        setSDKAuthToken(adminToken)
      }

      toast.success('Impersonation stopped')

      // Invalidate all queries to refetch data with admin context
      queryClient.invalidateQueries()
    } catch (error: unknown) {
      const errorMessage =
        error instanceof Error && 'response' in error
          ? (error as { response?: { data?: { error?: string } } }).response
              ?.data?.error
          : undefined
      toast.error(errorMessage || 'Failed to stop impersonation')
    } finally {
      setIsStopping(false)
    }
  }

  if (!isImpersonating) {
    return null
  }

  const getDisplayText = () => {
    switch (impersonationType) {
      case 'user':
        return `Impersonating: ${impersonatedUser?.email || 'Unknown User'}`
      case 'anon':
        return 'Impersonating: Anonymous User (anon key)'
      case 'service':
        return 'Impersonating: Service Role'
      default:
        return 'Impersonating'
    }
  }

  return (
    <Alert className='rounded-none border-x-0 border-t-0 border-amber-200 bg-amber-50 dark:border-amber-800 dark:bg-amber-950'>
      <AlertTriangle className='h-4 w-4 text-amber-600 dark:text-amber-400' />
      <AlertDescription className='flex items-center justify-between'>
        <span className='font-medium text-amber-800 dark:text-amber-200'>
          {getDisplayText()}
        </span>
        <Button
          size='sm'
          variant='outline'
          onClick={handleStopImpersonation}
          disabled={isStopping}
          className='ml-4 border-amber-300 hover:bg-amber-100 dark:border-amber-700 dark:hover:bg-amber-900'
        >
          {isStopping ? (
            'Stopping...'
          ) : (
            <>
              <X className='mr-1 h-3 w-3' />
              Stop Impersonation
            </>
          )}
        </Button>
      </AlertDescription>
    </Alert>
  )
}
