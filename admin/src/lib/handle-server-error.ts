import { AxiosError } from 'axios'
import { toast } from 'sonner'

export function handleServerError(error: unknown) {
  // eslint-disable-next-line no-console
  console.log(error)

  let errMsg = 'Something went wrong!'
  let description: string | undefined

  if (
    error &&
    typeof error === 'object' &&
    'status' in error &&
    Number(error.status) === 204
  ) {
    errMsg = 'Content not found.'
  }

  if (error instanceof AxiosError) {
    const data = error.response?.data

    // Extract error message from various possible fields
    // Backend primarily uses "error", but check others for compatibility
    errMsg =
      data?.error || data?.title || data?.message || data?.detail || errMsg

    // For ENV_OVERRIDE errors, provide a clearer message
    if (data?.code === 'ENV_OVERRIDE') {
      errMsg = 'Setting controlled by environment variable'
      description = data?.error || 'This setting cannot be changed via the UI'
    }
  }

  toast.error(errMsg, description ? { description } : undefined)
}
