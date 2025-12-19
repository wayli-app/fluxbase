import { DotsHorizontalIcon } from '@radix-ui/react-icons'
import { type Row } from '@tanstack/react-table'
import { Lock, Trash2, Unlock, UserPen } from 'lucide-react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { type User } from '../data/schema'
import { useUsers } from './users-provider'

type DataTableRowActionsProps = {
  row: Row<User>
}

export function DataTableRowActions({ row }: DataTableRowActionsProps) {
  const { setOpen, setCurrentRow, userType } = useUsers()
  const queryClient = useQueryClient()
  const user = row.original

  const lockMutation = useMutation({
    mutationFn: async () => {
      const response = await fetch(
        `/api/v1/admin/users/${user.id}/lock?type=${userType}`,
        { method: 'POST' }
      )
      if (!response.ok) {
        const data = await response.json()
        throw new Error(data.error || 'Failed to lock user')
      }
      return response.json()
    },
    onSuccess: () => {
      toast.success('User account locked')
      queryClient.invalidateQueries({ queryKey: ['users', userType] })
    },
    onError: (error: Error) => {
      toast.error(error.message)
    },
  })

  const unlockMutation = useMutation({
    mutationFn: async () => {
      const response = await fetch(
        `/api/v1/admin/users/${user.id}/unlock?type=${userType}`,
        { method: 'POST' }
      )
      if (!response.ok) {
        const data = await response.json()
        throw new Error(data.error || 'Failed to unlock user')
      }
      return response.json()
    },
    onSuccess: () => {
      toast.success('User account unlocked')
      queryClient.invalidateQueries({ queryKey: ['users', userType] })
    },
    onError: (error: Error) => {
      toast.error(error.message)
    },
  })

  return (
    <>
      <DropdownMenu modal={false}>
        <DropdownMenuTrigger asChild>
          <Button
            variant='ghost'
            className='data-[state=open]:bg-muted flex h-8 w-8 p-0'
          >
            <DotsHorizontalIcon className='h-4 w-4' />
            <span className='sr-only'>Open menu</span>
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align='end' className='w-[160px]'>
          <DropdownMenuItem
            onClick={() => {
              setCurrentRow(row.original)
              setOpen('edit')
            }}
          >
            Edit
            <DropdownMenuShortcut>
              <UserPen size={16} />
            </DropdownMenuShortcut>
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          {user.is_locked ? (
            <DropdownMenuItem
              onClick={() => unlockMutation.mutate()}
              disabled={unlockMutation.isPending}
            >
              Unlock
              <DropdownMenuShortcut>
                <Unlock size={16} />
              </DropdownMenuShortcut>
            </DropdownMenuItem>
          ) : (
            <DropdownMenuItem
              onClick={() => lockMutation.mutate()}
              disabled={lockMutation.isPending}
              className='text-orange-600'
            >
              Lock
              <DropdownMenuShortcut>
                <Lock size={16} />
              </DropdownMenuShortcut>
            </DropdownMenuItem>
          )}
          <DropdownMenuSeparator />
          <DropdownMenuItem
            onClick={() => {
              setCurrentRow(row.original)
              setOpen('delete')
            }}
            className='text-red-500!'
          >
            Delete
            <DropdownMenuShortcut>
              <Trash2 size={16} />
            </DropdownMenuShortcut>
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </>
  )
}
