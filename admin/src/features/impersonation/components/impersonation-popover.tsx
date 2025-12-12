import { useState } from 'react'
import { UserCog, User, UserX, Shield, X } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useAuth } from '@/hooks/use-auth'
import { useImpersonation } from '../hooks/use-impersonation'
import type { ImpersonationType } from '@/stores/impersonation-store'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { UserSearch } from './user-search'

export interface ImpersonationPopoverProps {
  /** Context label shown in status badge (e.g., "Executing as", "Viewing as", "Running as") */
  contextLabel?: string
  /** Whether to require a reason before starting impersonation (default: false for inline use) */
  requireReason?: boolean
  /** Default reason if requireReason is false (for audit trail) */
  defaultReason?: string
  /** Callback fired after impersonation starts successfully */
  onImpersonationStart?: () => void
  /** Callback fired after impersonation stops */
  onImpersonationStop?: () => void
  /** Additional CSS classes for the container */
  className?: string
  /** Size variant for the trigger button */
  size?: 'sm' | 'default'
}

export function ImpersonationPopover({
  contextLabel = 'Impersonating',
  requireReason = false,
  defaultReason = 'Admin impersonation',
  onImpersonationStart,
  onImpersonationStop,
  className,
  size = 'sm',
}: ImpersonationPopoverProps) {
  const { user } = useAuth()
  const [open, setOpen] = useState(false)
  const [impersonationType, setImpersonationType] =
    useState<ImpersonationType>('user')
  const [selectedUserId, setSelectedUserId] = useState('')
  const [selectedUserEmail, setSelectedUserEmail] = useState('')
  const [reason, setReason] = useState('')

  const {
    isImpersonating,
    impersonationType: activeType,
    isLoading,
    startUserImpersonation,
    startAnonImpersonation,
    startServiceImpersonation,
    stopImpersonation,
    getDisplayLabel,
  } = useImpersonation({
    defaultReason,
    onStart: onImpersonationStart,
    onStop: onImpersonationStop,
  })

  // Only show to dashboard_admin users
  const isDashboardAdmin =
    user && 'role' in user
      ? Array.isArray(user.role)
        ? user.role.includes('dashboard_admin')
        : user.role === 'dashboard_admin'
      : false

  if (!isDashboardAdmin) {
    return null
  }

  const handleStartImpersonation = async () => {
    const reasonToUse = requireReason ? reason : defaultReason

    if (requireReason && !reason.trim()) {
      return
    }

    if (impersonationType === 'user' && !selectedUserId) {
      return
    }

    switch (impersonationType) {
      case 'user':
        await startUserImpersonation(selectedUserId, selectedUserEmail, reasonToUse)
        break
      case 'anon':
        await startAnonImpersonation(reasonToUse)
        break
      case 'service':
        await startServiceImpersonation(reasonToUse)
        break
    }

    // Reset form and close popover on success
    setOpen(false)
    setSelectedUserId('')
    setSelectedUserEmail('')
    setReason('')
  }

  const handleUserSelect = (userId: string, userEmail: string) => {
    setSelectedUserId(userId)
    setSelectedUserEmail(userEmail)
  }

  const handleStopImpersonation = async () => {
    await stopImpersonation()
  }

  const getTypeIcon = (type: ImpersonationType) => {
    switch (type) {
      case 'user':
        return <User className='h-3.5 w-3.5' />
      case 'anon':
        return <UserX className='h-3.5 w-3.5' />
      case 'service':
        return <Shield className='h-3.5 w-3.5' />
    }
  }

  const getBadgeColors = () => {
    switch (activeType) {
      case 'anon':
        return 'border-orange-500 text-orange-600 dark:text-orange-400 bg-orange-50 dark:bg-orange-950'
      case 'service':
        return 'border-purple-500 text-purple-600 dark:text-purple-400 bg-purple-50 dark:bg-purple-950'
      case 'user':
      default:
        return 'border-blue-500 text-blue-600 dark:text-blue-400 bg-blue-50 dark:bg-blue-950'
    }
  }

  // When impersonating, show status badge with stop button
  if (isImpersonating) {
    return (
      <div className={cn('flex items-center gap-2', className)}>
        <Badge
          variant='outline'
          className={cn('gap-1.5 py-1.5 px-3', getBadgeColors())}
        >
          {getTypeIcon(activeType!)}
          <span className='truncate max-w-[200px]'>
            {contextLabel}: {getDisplayLabel()}
          </span>
        </Badge>
        <Button
          variant='ghost'
          size='sm'
          onClick={handleStopImpersonation}
          className='h-7 w-7 p-0'
          title='Stop impersonation'
        >
          <X className='h-4 w-4' />
        </Button>
      </div>
    )
  }

  // When not impersonating, show trigger button with popover
  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant='outline'
          size={size}
          disabled={isLoading}
          className={cn('gap-2', className)}
        >
          <UserCog className='h-4 w-4' />
          Impersonate
        </Button>
      </PopoverTrigger>
      <PopoverContent className='w-80' align='end'>
        <div className='grid gap-4'>
          <div className='space-y-2'>
            <h4 className='font-medium leading-none'>Impersonate User</h4>
            <p className='text-sm text-muted-foreground'>
              Execute operations as a different user or role
            </p>
          </div>

          <div className='grid gap-3'>
            <div className='grid gap-1.5'>
              <Label htmlFor='impersonation-type'>Type</Label>
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
              <div className='grid gap-1.5'>
                <Label htmlFor='user-select'>User</Label>
                <UserSearch
                  value={selectedUserId}
                  onSelect={handleUserSelect}
                  disabled={isLoading}
                />
              </div>
            )}

            {requireReason && (
              <div className='grid gap-1.5'>
                <Label htmlFor='reason'>Reason</Label>
                <Textarea
                  id='reason'
                  placeholder='e.g., Testing RLS policies...'
                  value={reason}
                  onChange={(e) => setReason(e.target.value)}
                  disabled={isLoading}
                  rows={2}
                  className='resize-none'
                />
                <p className='text-xs text-muted-foreground'>
                  Logged for audit trail
                </p>
              </div>
            )}

            <Button
              onClick={handleStartImpersonation}
              disabled={
                isLoading ||
                (impersonationType === 'user' && !selectedUserId) ||
                (requireReason && !reason.trim())
              }
              className='w-full'
            >
              {isLoading ? 'Starting...' : 'Start Impersonation'}
            </Button>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  )
}
