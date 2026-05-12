import { useMemo } from 'react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
} from '@/components/ui/select'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { userKnowledgeBasesApi, type KBPermissionGrant, type KBPermission } from '@/lib/api'
import { api } from '@/lib/api/client'

interface User {
  id: string
  email: string
  name?: string
}

interface ShareKnowledgeBaseDialogProps {
  kbId: string
  kbName: string
  open: boolean
  onClose: () => void
}

function ShareKnowledgeBaseDialog({ kbId, kbName, open, onClose }: ShareKnowledgeBaseDialogProps) {
  const queryClient = useQueryClient()

  const { data: usersData } = useQuery({
    queryKey: ['users'],
    queryFn: async () => {
      const res = await api.get<{ users: User[] }>('/api/v1/admin/users')
      return res.data
    },
  })

  const { data: permissionsData } = useQuery({
    queryKey: ['kb-permissions', kbId],
    queryFn: () => userKnowledgeBasesApi.listPermissions(kbId),
  })

  const permissions = useMemo<Record<string, string>>(() => {
    if (!permissionsData) return {}
    const perms: Record<string, string> = {}
    permissionsData.forEach((p: KBPermissionGrant) => {
      perms[p.user_id] = p.permission
    })
    return perms
  }, [permissionsData])

  const grantMutation = useMutation({
    mutationFn: ({ userId, permission }: { userId: string; permission: KBPermission }) =>
      userKnowledgeBasesApi.share(kbId, userId, permission),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['kb-permissions', kbId] })
      toast.success('Permission granted')
    },
    onError: (error: Error) => {
      toast.error('Failed to grant permission', { description: error.message })
    },
  })

  const revokeMutation = useMutation({
    mutationFn: (userId: string) => userKnowledgeBasesApi.revokePermission(kbId, userId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['kb-permissions', kbId] })
      toast.success('Permission revoked')
    },
    onError: (error: Error) => {
      toast.error('Failed to revoke permission', { description: error.message })
    },
  })

  const handlePermissionChange = (userId: string, permission: string) => {
    if (permission === 'none') {
      revokeMutation.mutate(userId)
    } else {
      grantMutation.mutate({ userId, permission: permission as KBPermission })
    }
  }

  const users = usersData?.users || []

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>Share Knowledge Base</DialogTitle>
          <DialogDescription>Grant access to &quot;{kbName}&quot;</DialogDescription>
        </DialogHeader>

        <div className="space-y-4 max-h-96 overflow-y-auto">
          {users.map((user: User) => (
            <div key={user.id} className="flex items-center justify-between py-2 border-b">
              <div className="flex-1">
                <div className="font-medium">{user.email}</div>
                {user.name && <div className="text-sm text-muted-foreground">{user.name}</div>}
              </div>
              <Select
                value={permissions[user.id] || 'none'}
                onValueChange={(value) => handlePermissionChange(user.id, value)}
              >
                <SelectTrigger className="w-40">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="none">No Access</SelectItem>
                  <SelectItem value="viewer">Viewer</SelectItem>
                  <SelectItem value="editor">Editor</SelectItem>
                </SelectContent>
              </Select>
            </div>
          ))}
          {users.length === 0 && (
            <div className="text-center py-8 text-muted-foreground">
              No users available to share with.
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}

export { ShareKnowledgeBaseDialog }
