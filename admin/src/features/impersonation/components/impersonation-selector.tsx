import { useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { UserCog, User, UserX, Shield } from 'lucide-react'
import { toast } from 'sonner'
import {
  useImpersonationStore,
  type ImpersonationType,
} from '@/stores/impersonation-store'
import { setAuthToken as setSDKAuthToken } from '@/lib/fluxbase-client'
import { impersonationApi } from '@/lib/impersonation-api'
import { useAuth } from '@/hooks/use-auth'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import { UserSearch } from './user-search'

export function ImpersonationSelector() {
  const { user } = useAuth()
  const { isImpersonating, startImpersonation } = useImpersonationStore()
  const queryClient = useQueryClient()
  const [open, setOpen] = useState(false)
  const [loading, setLoading] = useState(false)
  const [impersonationType, setImpersonationType] =
    useState<ImpersonationType>('user')
  const [selectedUserId, setSelectedUserId] = useState<string>('')
  const [reason, setReason] = useState('')

  const handleStartImpersonation = async () => {
    if (!reason.trim()) {
      toast.error('Please provide a reason for impersonation')
      return
    }

    if (impersonationType === 'user' && !selectedUserId) {
      toast.error('Please select a user to impersonate')
      return
    }

    try {
      setLoading(true)
      let response

      switch (impersonationType) {
        case 'user':
          response = await impersonationApi.startUserImpersonation(
            selectedUserId,
            reason
          )
          break
        case 'anon':
          response = await impersonationApi.startAnonImpersonation(reason)
          break
        case 'service':
          response = await impersonationApi.startServiceImpersonation(reason)
          break
      }

      startImpersonation(
        response.access_token,
        response.refresh_token,
        response.target_user,
        response.session,
        impersonationType
      )

      // Update SDK client token to use impersonation token
      setSDKAuthToken(response.access_token)

      toast.success(
        `Started impersonating ${
          impersonationType === 'user'
            ? response.target_user.email
            : impersonationType === 'anon'
              ? 'anonymous user'
              : 'service role'
        }`
      )

      // Reset form and close dialog
      setOpen(false)
      setSelectedUserId('')
      setReason('')

      // Invalidate all queries to refetch data with new impersonation context
      queryClient.invalidateQueries()
    } catch (error: unknown) {
      const errorMessage =
        error instanceof Error && 'response' in error
          ? (error as { response?: { data?: { error?: string } } }).response
              ?.data?.error
          : undefined
      toast.error(errorMessage || 'Failed to start impersonation')
    } finally {
      setLoading(false)
    }
  }

  const handleUserSelect = (userId: string, _userEmail: string) => {
    setSelectedUserId(userId)
  }

  const getIcon = () => {
    switch (impersonationType) {
      case 'user':
        return <User className='h-4 w-4' />
      case 'anon':
        return <UserX className='h-4 w-4' />
      case 'service':
        return <Shield className='h-4 w-4' />
    }
  }

  // Only show impersonation button to dashboard_admin users
  const isDashboardAdmin =
    user && 'role' in user
      ? Array.isArray(user.role)
        ? user.role.includes('dashboard_admin')
        : user.role === 'dashboard_admin'
      : false
  if (!isDashboardAdmin) {
    return null
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button
          variant='outline'
          size='sm'
          disabled={isImpersonating}
          className='gap-2'
        >
          <UserCog className='h-4 w-4' />
          Impersonate User
        </Button>
      </DialogTrigger>
      <DialogContent className='sm:max-w-[500px]'>
        <DialogHeader>
          <DialogTitle>Start User Impersonation</DialogTitle>
          <DialogDescription>
            View data as it appears to a specific user, anonymous visitor, or
            with service-level permissions. All actions will be logged for audit
            purposes.
          </DialogDescription>
        </DialogHeader>

        <div className='grid gap-4 py-4'>
          <div className='grid gap-2'>
            <Label htmlFor='impersonation-type'>Impersonation Type</Label>
            <Select
              value={impersonationType}
              onValueChange={(value) =>
                setImpersonationType(value as ImpersonationType)
              }
            >
              <SelectTrigger id='impersonation-type'>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value='user'>
                  <div className='flex items-center gap-2'>
                    <User className='h-4 w-4' />
                    Specific User
                  </div>
                </SelectItem>
                <SelectItem value='anon'>
                  <div className='flex items-center gap-2'>
                    <UserX className='h-4 w-4' />
                    Anonymous (anon key)
                  </div>
                </SelectItem>
                <SelectItem value='service'>
                  <div className='flex items-center gap-2'>
                    <Shield className='h-4 w-4' />
                    Service Role
                  </div>
                </SelectItem>
              </SelectContent>
            </Select>
          </div>

          {impersonationType === 'user' && (
            <div className='grid gap-2'>
              <Label htmlFor='user-select'>User</Label>
              <UserSearch
                value={selectedUserId}
                onSelect={handleUserSelect}
                disabled={loading}
              />
            </div>
          )}

          <div className='grid gap-2'>
            <Label htmlFor='reason'>Reason</Label>
            <Textarea
              id='reason'
              placeholder='e.g., Customer support ticket #1234, debugging user-reported issue'
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              disabled={loading}
              rows={3}
            />
            <p className='text-muted-foreground text-xs'>
              This reason will be logged in the audit trail
            </p>
          </div>
        </div>

        <DialogFooter>
          <Button
            variant='outline'
            onClick={() => setOpen(false)}
            disabled={loading}
          >
            Cancel
          </Button>
          <Button onClick={handleStartImpersonation} disabled={loading}>
            {loading ? (
              'Starting...'
            ) : (
              <>
                {getIcon()}
                <span className='ml-2'>Start Impersonation</span>
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
