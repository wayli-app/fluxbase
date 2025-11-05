import { useState, useEffect, useCallback } from 'react'
import { Check, ChevronsUpDown, Loader2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { impersonationApi } from '@/lib/impersonation-api'

interface User {
  id: string
  email: string
  created_at: string
  role: string
}

interface UserSearchProps {
  value?: string
  onSelect: (userId: string, userEmail: string) => void
  disabled?: boolean
}

export function UserSearch({ value, onSelect, disabled }: UserSearchProps) {
  const [open, setOpen] = useState(false)
  const [users, setUsers] = useState<User[]>([])
  const [loading, setLoading] = useState(false)
  const [search, setSearch] = useState('')

  const selectedUser = users.find((user) => user.id === value)

  const loadUsers = useCallback(async (searchTerm: string) => {
    try {
      setLoading(true)
      const response = await impersonationApi.listUsers(searchTerm || undefined, 20)
      setUsers(response.users)
    } catch {
      setUsers([])
    } finally {
      setLoading(false)
    }
  }, [])

  // Load initial users when opening
  useEffect(() => {
    if (open && users.length === 0) {
      loadUsers('')
    }
  }, [open, users.length, loadUsers])

  // Debounced search
  useEffect(() => {
    if (!open) return

    const timer = setTimeout(() => {
      loadUsers(search)
    }, 300)

    return () => clearTimeout(timer)
  }, [search, open, loadUsers])

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          role="combobox"
          aria-expanded={open}
          className="w-full justify-between"
          disabled={disabled}
        >
          {selectedUser ? (
            <span className="truncate">{selectedUser.email}</span>
          ) : (
            'Select user...'
          )}
          <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-[400px] p-0">
        <Command shouldFilter={false}>
          <CommandInput
            placeholder="Search users by email..."
            value={search}
            onValueChange={setSearch}
          />
          <CommandList>
            {loading ? (
              <div className="flex items-center justify-center py-6">
                <Loader2 className="h-4 w-4 animate-spin" />
              </div>
            ) : (
              <>
                <CommandEmpty>No users found.</CommandEmpty>
                <CommandGroup>
                  {users.map((user) => (
                    <CommandItem
                      key={user.id}
                      value={user.id}
                      onSelect={(currentValue) => {
                        const selectedUser = users.find((u) => u.id === currentValue)
                        if (selectedUser) {
                          onSelect(selectedUser.id, selectedUser.email)
                          setOpen(false)
                        }
                      }}
                    >
                      <Check
                        className={cn(
                          'mr-2 h-4 w-4',
                          value === user.id ? 'opacity-100' : 'opacity-0'
                        )}
                      />
                      <div className="flex flex-col">
                        <span className="font-medium">{user.email}</span>
                        <span className="text-xs text-muted-foreground">
                          Created: {new Date(user.created_at).toLocaleDateString()}
                        </span>
                      </div>
                    </CommandItem>
                  ))}
                </CommandGroup>
              </>
            )}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  )
}
