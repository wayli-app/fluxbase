import { AlertTriangle, X } from 'lucide-react'
import { useImpersonationStore } from '../stores/impersonation-store'
import { impersonationApi } from '../lib/impersonation-api'
import { useState } from 'react'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { toast } from 'sonner'

export function ImpersonationBanner() {
  const {
    isImpersonating,
    impersonatedUser,
    impersonationType,
    stopImpersonation,
  } = useImpersonationStore()
  const [isStopping, setIsStopping] = useState(false)

  const handleStopImpersonation = async () => {
    try {
      setIsStopping(true)
      await impersonationApi.stopImpersonation()
      stopImpersonation()
      toast.success('Impersonation stopped')

      // Reload the page to refresh data with admin context
      window.location.reload()
    } catch (error: unknown) {
      const errorMessage = error instanceof Error && 'response' in error
        ? (error as { response?: { data?: { error?: string } } }).response?.data?.error
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
    <Alert className="rounded-none border-x-0 border-t-0 bg-amber-50 border-amber-200 dark:bg-amber-950 dark:border-amber-800">
      <AlertTriangle className="h-4 w-4 text-amber-600 dark:text-amber-400" />
      <AlertDescription className="flex items-center justify-between">
        <span className="text-amber-800 dark:text-amber-200 font-medium">
          {getDisplayText()}
        </span>
        <Button
          size="sm"
          variant="outline"
          onClick={handleStopImpersonation}
          disabled={isStopping}
          className="ml-4 border-amber-300 hover:bg-amber-100 dark:border-amber-700 dark:hover:bg-amber-900"
        >
          {isStopping ? (
            'Stopping...'
          ) : (
            <>
              <X className="h-3 w-3 mr-1" />
              Stop Impersonation
            </>
          )}
        </Button>
      </AlertDescription>
    </Alert>
  )
}
