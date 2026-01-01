import { useState } from 'react'
import { formatDistanceToNow } from 'date-fns'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import {
  Key,
  Plus,
  Trash2,
  Copy,
  AlertCircle,
  Check,
  X,
  Search,
} from 'lucide-react'
import { toast } from 'sonner'
import {
  clientKeysApi,
  type ClientKey,
  type CreateClientKeyRequest,
} from '@/lib/api'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

export const Route = createFileRoute('/_authenticated/client-keys/')({
  component: ClientKeysPage,
})

interface ClientKeyWithPlaintext extends ClientKey {
  key: string // Only returned on creation
}

// Grouped scopes for better UI organization
const SCOPE_GROUPS = [
  {
    name: 'Tables',
    description: 'Database table access',
    scopes: [
      {
        id: 'read:tables',
        label: 'Read',
        description: 'Query database tables',
      },
      {
        id: 'write:tables',
        label: 'Write',
        description: 'Insert, update, delete records',
      },
    ],
  },
  {
    name: 'Storage',
    description: 'File storage access',
    scopes: [
      { id: 'read:storage', label: 'Read', description: 'Download files' },
      {
        id: 'write:storage',
        label: 'Write',
        description: 'Upload and delete files',
      },
    ],
  },
  {
    name: 'Functions',
    description: 'Edge Functions',
    scopes: [
      { id: 'read:functions', label: 'Read', description: 'View functions' },
      {
        id: 'execute:functions',
        label: 'Execute',
        description: 'Invoke functions',
      },
    ],
  },
  {
    name: 'Auth',
    description: 'Authentication',
    scopes: [
      { id: 'read:auth', label: 'Read', description: 'View user profile' },
      {
        id: 'write:auth',
        label: 'Write',
        description: 'Update profile, manage 2FA',
      },
    ],
  },
  {
    name: 'Client Keys',
    description: 'API key management',
    scopes: [
      { id: 'read:clientkeys', label: 'Read', description: 'List client keys' },
      {
        id: 'write:clientkeys',
        label: 'Write',
        description: 'Create, update, revoke',
      },
    ],
  },
  {
    name: 'Webhooks',
    description: 'Webhook management',
    scopes: [
      { id: 'read:webhooks', label: 'Read', description: 'List webhooks' },
      {
        id: 'write:webhooks',
        label: 'Write',
        description: 'Create, update, delete',
      },
    ],
  },
  {
    name: 'Monitoring',
    description: 'System monitoring',
    scopes: [
      {
        id: 'read:monitoring',
        label: 'Read',
        description: 'View metrics, health, logs',
      },
    ],
  },
  {
    name: 'Realtime',
    description: 'WebSocket channels',
    scopes: [
      {
        id: 'realtime:connect',
        label: 'Connect',
        description: 'Connect to channels',
      },
      {
        id: 'realtime:broadcast',
        label: 'Broadcast',
        description: 'Send messages',
      },
    ],
  },
  {
    name: 'RPC',
    description: 'Remote procedures',
    scopes: [
      { id: 'read:rpc', label: 'Read', description: 'List procedures' },
      { id: 'execute:rpc', label: 'Execute', description: 'Invoke procedures' },
    ],
  },
  {
    name: 'Jobs',
    description: 'Background jobs',
    scopes: [
      { id: 'read:jobs', label: 'Read', description: 'View job queues' },
      { id: 'write:jobs', label: 'Write', description: 'Manage job entries' },
    ],
  },
  {
    name: 'AI',
    description: 'AI & chatbots',
    scopes: [
      { id: 'read:ai', label: 'Read', description: 'View conversations' },
      { id: 'write:ai', label: 'Write', description: 'Send messages' },
    ],
  },
  {
    name: 'Secrets',
    description: 'Secret management',
    scopes: [
      { id: 'read:secrets', label: 'Read', description: 'View secret names' },
      {
        id: 'write:secrets',
        label: 'Write',
        description: 'Create, update, delete',
      },
    ],
  },
]

function ClientKeysPage() {
  const queryClient = useQueryClient()
  const [showCreateDialog, setShowCreateDialog] = useState(false)
  const [showKeyDialog, setShowKeyDialog] = useState(false)
  const [createdKey, setCreatedKey] = useState<ClientKeyWithPlaintext | null>(
    null
  )
  const [searchQuery, setSearchQuery] = useState('')

  // Form state
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [selectedScopes, setSelectedScopes] = useState<string[]>([])
  const [rateLimit, setRateLimit] = useState(100)
  const [expiresAt, setExpiresAt] = useState('')

  // Fetch client keys
  const { data: clientKeys, isLoading } = useQuery<ClientKey[]>({
    queryKey: ['client-keys'],
    queryFn: clientKeysApi.list,
  })

  // Create client key
  const createMutation = useMutation({
    mutationFn: clientKeysApi.create,
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['client-keys'] })
      setCreatedKey(data as unknown as ClientKeyWithPlaintext)
      setShowCreateDialog(false)
      setShowKeyDialog(true)
      // Reset form
      setName('')
      setDescription('')
      setSelectedScopes([])
      setRateLimit(100)
      setExpiresAt('')
    },
    onError: () => {
      toast.error('Failed to create client key')
    },
  })

  // Revoke client key
  const revokeMutation = useMutation({
    mutationFn: clientKeysApi.revoke,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['client-keys'] })
      toast.success('Client key revoked successfully')
    },
    onError: () => {
      toast.error('Failed to revoke client key')
    },
  })

  // Delete client key
  const deleteMutation = useMutation({
    mutationFn: clientKeysApi.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['client-keys'] })
      toast.success('Client key deleted successfully')
    },
    onError: () => {
      toast.error('Failed to delete client key')
    },
  })

  const handleCreateKey = () => {
    if (!name.trim()) {
      toast.error('Please enter a key name')
      return
    }
    if (selectedScopes.length === 0) {
      toast.error('Please select at least one scope')
      return
    }

    const request: CreateClientKeyRequest = {
      name: name.trim(),
      description: description.trim() || undefined,
      scopes: selectedScopes,
      rate_limit_per_minute: rateLimit,
      expires_at: expiresAt || undefined,
    }

    createMutation.mutate(request)
  }

  const toggleScope = (scopeId: string) => {
    setSelectedScopes((prev) =>
      prev.includes(scopeId)
        ? prev.filter((s) => s !== scopeId)
        : [...prev, scopeId]
    )
  }

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text)
    toast.success('Copied to clipboard')
  }

  const isExpired = (expiresAt?: string) => {
    if (!expiresAt) return false
    return new Date(expiresAt) < new Date()
  }

  const isRevoked = (revokedAt?: string) => !!revokedAt

  const getKeyStatus = (key: ClientKey) => {
    if (isRevoked(key.revoked_at))
      return { label: 'Revoked', variant: 'secondary' as const }
    if (isExpired(key.expires_at))
      return { label: 'Expired', variant: 'destructive' as const }
    return { label: 'Active', variant: 'default' as const }
  }

  // Filter keys by search query
  const filteredKeys = clientKeys?.filter(
    (key) =>
      key.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      key.description?.toLowerCase().includes(searchQuery.toLowerCase()) ||
      key.key_prefix.toLowerCase().includes(searchQuery.toLowerCase())
  )

  return (
    <div className='flex flex-1 flex-col gap-6 p-6'>
      <div>
        <h1 className='flex items-center gap-2 text-3xl font-bold tracking-tight'>
          <Key className='h-8 w-8' />
          Client Keys
        </h1>
        <p className='text-muted-foreground mt-2'>
          Generate and manage client keys for programmatic access
        </p>
      </div>

      {/* Stats Cards */}
      <div className='grid gap-4 md:grid-cols-3'>
        <Card>
          <CardHeader className='flex flex-row items-center justify-between space-y-0 pb-2'>
            <CardTitle className='text-sm font-medium'>Total Keys</CardTitle>
            <Key className='text-muted-foreground h-4 w-4' />
          </CardHeader>
          <CardContent>
            <div className='text-2xl font-bold'>{clientKeys?.length || 0}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className='flex flex-row items-center justify-between space-y-0 pb-2'>
            <CardTitle className='text-sm font-medium'>Active Keys</CardTitle>
            <Check className='text-muted-foreground h-4 w-4' />
          </CardHeader>
          <CardContent>
            <div className='text-2xl font-bold'>
              {clientKeys?.filter(
                (k) => !isRevoked(k.revoked_at) && !isExpired(k.expires_at)
              ).length || 0}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className='flex flex-row items-center justify-between space-y-0 pb-2'>
            <CardTitle className='text-sm font-medium'>Revoked Keys</CardTitle>
            <X className='text-muted-foreground h-4 w-4' />
          </CardHeader>
          <CardContent>
            <div className='text-2xl font-bold'>
              {clientKeys?.filter((k) => isRevoked(k.revoked_at)).length || 0}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Main Card */}
      <Card>
        <CardHeader>
          <div className='flex items-center justify-between'>
            <div>
              <CardTitle>Client Keys</CardTitle>
              <CardDescription>
                Manage your client keys for service-to-service authentication
              </CardDescription>
            </div>
            <Button onClick={() => setShowCreateDialog(true)}>
              <Plus className='mr-2 h-4 w-4' />
              Create Client Key
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {/* Search */}
          <div className='mb-4'>
            <div className='relative'>
              <Search className='text-muted-foreground absolute top-2.5 left-2 h-4 w-4' />
              <Input
                placeholder='Search by name, description, or key prefix...'
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className='pl-8'
              />
            </div>
          </div>

          {isLoading ? (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Key Prefix</TableHead>
                  <TableHead>Scopes</TableHead>
                  <TableHead>Rate Limit</TableHead>
                  <TableHead>Last Used</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className='text-right'>Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {Array(3)
                  .fill(0)
                  .map((_, i) => (
                    <TableRow key={i}>
                      <TableCell>
                        <div className='space-y-1'>
                          <Skeleton className='h-4 w-28' />
                          <Skeleton className='h-3 w-20' />
                        </div>
                      </TableCell>
                      <TableCell>
                        <Skeleton className='h-4 w-24' />
                      </TableCell>
                      <TableCell>
                        <Skeleton className='h-5 w-16' />
                      </TableCell>
                      <TableCell>
                        <Skeleton className='h-4 w-20' />
                      </TableCell>
                      <TableCell>
                        <Skeleton className='h-4 w-24' />
                      </TableCell>
                      <TableCell>
                        <Skeleton className='h-5 w-16' />
                      </TableCell>
                      <TableCell className='text-right'>
                        <div className='flex justify-end gap-1'>
                          <Skeleton className='h-8 w-8' />
                          <Skeleton className='h-8 w-8' />
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
              </TableBody>
            </Table>
          ) : filteredKeys && filteredKeys.length > 0 ? (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Key Prefix</TableHead>
                  <TableHead>Scopes</TableHead>
                  <TableHead>Rate Limit</TableHead>
                  <TableHead>Last Used</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className='text-right'>Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredKeys.map((key) => {
                  const status = getKeyStatus(key)
                  return (
                    <TableRow key={key.id}>
                      <TableCell>
                        <div>
                          <div className='font-medium'>{key.name}</div>
                          {key.description && (
                            <div className='text-muted-foreground text-xs'>
                              {key.description}
                            </div>
                          )}
                        </div>
                      </TableCell>
                      <TableCell>
                        <code className='text-xs'>{key.key_prefix}...</code>
                      </TableCell>
                      <TableCell>
                        <div className='flex flex-wrap gap-1'>
                          {key.scopes.slice(0, 2).map((scope) => (
                            <Badge
                              key={scope}
                              variant='outline'
                              className='text-xs'
                            >
                              {scope}
                            </Badge>
                          ))}
                          {key.scopes.length > 2 && (
                            <Badge variant='outline' className='text-xs'>
                              +{key.scopes.length - 2}
                            </Badge>
                          )}
                        </div>
                      </TableCell>
                      <TableCell className='text-sm'>
                        {key.rate_limit_per_minute}/min
                      </TableCell>
                      <TableCell className='text-muted-foreground text-sm'>
                        {key.last_used_at
                          ? formatDistanceToNow(new Date(key.last_used_at), {
                              addSuffix: true,
                            })
                          : 'Never'}
                      </TableCell>
                      <TableCell>
                        <Badge variant={status.variant}>{status.label}</Badge>
                      </TableCell>
                      <TableCell className='text-right'>
                        <div className='flex justify-end gap-1'>
                          {!isRevoked(key.revoked_at) && (
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Button
                                  variant='ghost'
                                  size='sm'
                                  onClick={() => revokeMutation.mutate(key.id)}
                                  disabled={revokeMutation.isPending}
                                >
                                  <X className='h-4 w-4' />
                                </Button>
                              </TooltipTrigger>
                              <TooltipContent>Revoke client key</TooltipContent>
                            </Tooltip>
                          )}
                          <AlertDialog>
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <AlertDialogTrigger asChild>
                                  <Button
                                    variant='ghost'
                                    size='sm'
                                    disabled={deleteMutation.isPending}
                                    className='text-destructive hover:text-destructive hover:bg-destructive/10'
                                  >
                                    <Trash2 className='h-4 w-4' />
                                  </Button>
                                </AlertDialogTrigger>
                              </TooltipTrigger>
                              <TooltipContent>Delete client key</TooltipContent>
                            </Tooltip>
                            <AlertDialogContent>
                              <AlertDialogHeader>
                                <AlertDialogTitle>
                                  Delete Client Key
                                </AlertDialogTitle>
                                <AlertDialogDescription>
                                  Are you sure you want to delete "{key.name}"?
                                  Any applications using this key will lose
                                  access immediately.
                                </AlertDialogDescription>
                              </AlertDialogHeader>
                              <AlertDialogFooter>
                                <AlertDialogCancel>Cancel</AlertDialogCancel>
                                <AlertDialogAction
                                  onClick={() => deleteMutation.mutate(key.id)}
                                  className='bg-destructive text-destructive-foreground hover:bg-destructive/90'
                                >
                                  Delete
                                </AlertDialogAction>
                              </AlertDialogFooter>
                            </AlertDialogContent>
                          </AlertDialog>
                        </div>
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
          ) : (
            <div className='flex flex-col items-center justify-center py-12 text-center'>
              <Key className='text-muted-foreground mb-4 h-12 w-12' />
              <p className='text-muted-foreground'>
                {searchQuery
                  ? 'No client keys match your search'
                  : 'No client keys yet'}
              </p>
              {!searchQuery && (
                <Button
                  onClick={() => setShowCreateDialog(true)}
                  variant='outline'
                  className='mt-4'
                >
                  Create Your First Client Key
                </Button>
              )}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Create Client Key Dialog */}
      <Dialog open={showCreateDialog} onOpenChange={setShowCreateDialog}>
        <DialogContent className='max-h-[90vh] max-w-3xl overflow-y-auto'>
          <DialogHeader>
            <DialogTitle>Create Client Key</DialogTitle>
            <DialogDescription>
              Generate a new client key for programmatic access. The key will be
              shown only once.
            </DialogDescription>
          </DialogHeader>
          <div className='grid gap-4 py-4'>
            <div className='grid grid-cols-2 gap-4'>
              <div className='grid gap-2'>
                <Label htmlFor='name'>
                  Name <span className='text-destructive'>*</span>
                </Label>
                <Input
                  id='name'
                  placeholder='Production Client Key'
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                />
              </div>
              <div className='grid gap-2'>
                <Label htmlFor='description'>Description</Label>
                <Input
                  id='description'
                  placeholder='Used by the main application'
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                />
              </div>
            </div>
            <div className='grid gap-2'>
              <Label>
                Scopes/Permissions <span className='text-destructive'>*</span>
              </Label>
              <div className='grid grid-cols-2 gap-3 rounded-md border p-4'>
                {SCOPE_GROUPS.map((group) => (
                  <div key={group.name} className='space-y-1'>
                    <div className='text-sm font-medium'>{group.name}</div>
                    <div className='text-muted-foreground text-xs'>
                      {group.description}
                    </div>
                    <div className='flex flex-wrap gap-3 pt-1'>
                      {group.scopes.map((scope) => (
                        <div
                          key={scope.id}
                          className='flex items-center space-x-1.5'
                        >
                          <input
                            type='checkbox'
                            id={`create-${scope.id}`}
                            checked={selectedScopes.includes(scope.id)}
                            onChange={() => toggleScope(scope.id)}
                            className='h-3.5 w-3.5 rounded border-gray-300'
                          />
                          <label
                            htmlFor={`create-${scope.id}`}
                            className='text-xs'
                            title={scope.description}
                          >
                            {scope.label}
                          </label>
                        </div>
                      ))}
                    </div>
                  </div>
                ))}
              </div>
            </div>
            <div className='grid grid-cols-2 gap-4'>
              <div className='grid gap-2'>
                <Label htmlFor='rateLimit'>
                  Rate Limit (requests per minute)
                </Label>
                <Input
                  id='rateLimit'
                  type='number'
                  min='1'
                  max='10000'
                  value={rateLimit}
                  onChange={(e) =>
                    setRateLimit(parseInt(e.target.value) || 100)
                  }
                />
              </div>
              <div className='grid gap-2'>
                <Label htmlFor='expiresAt'>Expiration Date (optional)</Label>
                <Input
                  id='expiresAt'
                  type='datetime-local'
                  value={expiresAt}
                  onChange={(e) => setExpiresAt(e.target.value)}
                />
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button
              variant='outline'
              onClick={() => setShowCreateDialog(false)}
            >
              Cancel
            </Button>
            <Button
              onClick={handleCreateKey}
              disabled={createMutation.isPending}
            >
              {createMutation.isPending ? 'Creating...' : 'Generate Client Key'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Show Created Key Dialog */}
      <Dialog open={showKeyDialog} onOpenChange={setShowKeyDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Client Key Created</DialogTitle>
            <DialogDescription>
              Save this key now. You won't be able to see it again!
            </DialogDescription>
          </DialogHeader>
          <div className='space-y-4 py-4'>
            <div className='rounded-md bg-yellow-50 p-4 dark:bg-yellow-950'>
              <div className='flex'>
                <AlertCircle className='h-5 w-5 text-yellow-600 dark:text-yellow-400' />
                <div className='ml-3'>
                  <h3 className='text-sm font-medium text-yellow-800 dark:text-yellow-200'>
                    Important: Copy this key now
                  </h3>
                  <div className='mt-2 text-sm text-yellow-700 dark:text-yellow-300'>
                    <p>
                      This is the only time you'll see the full client key.
                      Store it securely.
                    </p>
                  </div>
                </div>
              </div>
            </div>
            <div className='grid gap-2'>
              <Label>Client Key</Label>
              <div className='flex gap-2'>
                <Input
                  value={createdKey?.key || ''}
                  readOnly
                  className='font-mono text-xs'
                />
                <Button
                  variant='outline'
                  size='icon'
                  onClick={() => copyToClipboard(createdKey?.key || '')}
                >
                  <Copy className='h-4 w-4' />
                </Button>
              </div>
            </div>
            <div className='grid gap-2'>
              <Label>Name</Label>
              <Input value={createdKey?.name || ''} readOnly />
            </div>
            <div className='grid gap-2'>
              <Label>Scopes</Label>
              <div className='flex flex-wrap gap-1'>
                {createdKey?.scopes.map((scope) => (
                  <Badge key={scope} variant='outline'>
                    {scope}
                  </Badge>
                ))}
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button onClick={() => setShowKeyDialog(false)}>
              I've Saved the Key
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
