import { useNavigate } from '@tanstack/react-router'
import { toast } from 'sonner'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { adminAuthAPI } from '@/lib/api'
import { clearTokens } from '@/lib/auth'

interface SignOutDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function SignOutDialog({ open, onOpenChange }: SignOutDialogProps) {
  const navigate = useNavigate()

  const handleSignOut = async () => {
    try {
      // Call backend logout endpoint to invalidate token
      await adminAuthAPI.logout()
    } catch {
      // Continue with local logout even if API call fails
    } finally {
      // Clear tokens from localStorage
      clearTokens()

      // Show success message
      toast.success('Signed out successfully')

      // Redirect to login
      navigate({
        to: '/login',
        replace: true,
      })
    }
  }

  return (
    <ConfirmDialog
      open={open}
      onOpenChange={onOpenChange}
      title='Sign out'
      desc='Are you sure you want to sign out? You will need to sign in again to access your account.'
      confirmText='Sign out'
      destructive
      handleConfirm={handleSignOut}
      className='sm:max-w-sm'
    />
  )
}
