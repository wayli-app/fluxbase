import { useState } from 'react'
import { formatDistanceToNow } from 'date-fns'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import {
  KeyRound,
  Plus,
  Trash2,
  Copy,
  AlertCircle,
  Check,
  X,
  Search,
  Power,
  PowerOff,
  Pencil,
} from 'lucide-react'
import { toast } from 'sonner'
import {
  serviceKeysApi,
  type ServiceKey,
  type ServiceKeyWithPlaintext,
  type CreateServiceKeyRequest,
  type UpdateServiceKeyRequest,
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

export const Route = createFileRoute('/_authenticated/service-keys/')({
  component: ServiceKeysPage,
})

const AVAILABLE_SCOPES = [
  { id: 'migrations:*', name: 'Migrations (All)', description: 'Full access to migrations API' },
  { id: 'migrations:read', name: 'Migrations (Read)', description: 'Read migration status' },
  { id: 'migrations:write', name: 'Migrations (Write)', description: 'Apply migrations' },
  { id: '*', name: 'All Scopes', description: 'Full access to all APIs' },
]

function ServiceKeysPage() {
  const queryClient = useQueryClient()
  const [showCreateDialog, setShowCreateDialog] = useState(false)
  const [showKeyDialog, setShowKeyDialog] = useState(false)
  const [showEditDialog, setShowEditDialog] = useState(false)
  const [createdKey, setCreatedKey] = useState<ServiceKeyWithPlaintext | null>(null)
  const [editingKey, setEditingKey] = useState<ServiceKey | null>(null)
  const [searchQuery, setSearchQuery] = useState('')

  // Form state for create
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [selectedScopes, setSelectedScopes] = useState<string[]>(['*'])
  const [rateLimitPerMinute, setRateLimitPerMinute] = useState<number | undefined>(undefined)
  const [rateLimitPerHour, setRateLimitPerHour] = useState<number | undefined>(undefined)
  const [expiresAt, setExpiresAt] = useState('')

  // Form state for edit
  const [editName, setEditName] = useState('')
  const [editDescription, setEditDescription] = useState('')
  const [editRateLimitPerMinute, setEditRateLimitPerMinute] = useState<number | undefined>(undefined)
  const [editRateLimitPerHour, setEditRateLimitPerHour] = useState<number | undefined>(undefined)

  // Fetch service keys
  const { data: serviceKeys, isLoading } = useQuery<ServiceKey[]>({
    queryKey: ['service-keys'],
    queryFn: serviceKeysApi.list,
  })

  // Create service key
  const createMutation = useMutation({
    mutationFn: serviceKeysApi.create,
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['service-keys'] })
      setCreatedKey(data)
      setShowCreateDialog(false)
      setShowKeyDialog(true)
      // Reset form
      setName('')
      setDescription('')
      setSelectedScopes(['*'])
      setRateLimitPerMinute(undefined)
      setRateLimitPerHour(undefined)
      setExpiresAt('')
    },
    onError: () => {
      toast.error('Failed to create service key')
    },
  })

  // Update service key
  const updateMutation = useMutation({
    mutationFn: ({ id, request }: { id: string; request: UpdateServiceKeyRequest }) =>
      serviceKeysApi.update(id, request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['service-keys'] })
      setShowEditDialog(false)
      setEditingKey(null)
      toast.success('Service key updated successfully')
    },
    onError: () => {
      toast.error('Failed to update service key')
    },
  })

  // Enable service key
  const enableMutation = useMutation({
    mutationFn: serviceKeysApi.enable,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['service-keys'] })
      toast.success('Service key enabled')
    },
    onError: () => {
      toast.error('Failed to enable service key')
    },
  })

  // Disable service key
  const disableMutation = useMutation({
    mutationFn: serviceKeysApi.disable,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['service-keys'] })
      toast.success('Service key disabled')
    },
    onError: () => {
      toast.error('Failed to disable service key')
    },
  })

  // Delete service key
  const deleteMutation = useMutation({
    mutationFn: serviceKeysApi.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['service-keys'] })
      toast.success('Service key deleted successfully')
    },
    onError: () => {
      toast.error('Failed to delete service key')
    },
  })

  const handleCreateKey = () => {
    if (!name.trim()) {
      toast.error('Please enter a key name')
      return
    }

    const request: CreateServiceKeyRequest = {
      name: name.trim(),
      description: description.trim() || undefined,
      scopes: selectedScopes.length > 0 ? selectedScopes : undefined,
      rate_limit_per_minute: rateLimitPerMinute,
      rate_limit_per_hour: rateLimitPerHour,
      expires_at: expiresAt || undefined,
    }

    createMutation.mutate(request)
  }

  const handleEditKey = () => {
    if (!editingKey) return

    const request: UpdateServiceKeyRequest = {
      name: editName.trim() || undefined,
      description: editDescription.trim(),
      rate_limit_per_minute: editRateLimitPerMinute,
      rate_limit_per_hour: editRateLimitPerHour,
    }

    updateMutation.mutate({ id: editingKey.id, request })
  }

  const openEditDialog = (key: ServiceKey) => {
    setEditingKey(key)
    setEditName(key.name)
    setEditDescription(key.description || '')
    setEditRateLimitPerMinute(key.rate_limit_per_minute)
    setEditRateLimitPerHour(key.rate_limit_per_hour)
    setShowEditDialog(true)
  }

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text)
    toast.success('Copied to clipboard')
  }

  const isExpired = (expiresAt?: string) => {
    if (!expiresAt) return false
    return new Date(expiresAt) < new Date()
  }

  const getKeyStatus = (key: ServiceKey) => {
    if (!key.enabled)
      return { label: 'Disabled', variant: 'secondary' as const }
    if (isExpired(key.expires_at))
      return { label: 'Expired', variant: 'destructive' as const }
    return { label: 'Active', variant: 'default' as const }
  }

  const formatRateLimit = (key: ServiceKey) => {
    const parts: string[] = []
    if (key.rate_limit_per_minute) {
      parts.push(`${key.rate_limit_per_minute}/min`)
    }
    if (key.rate_limit_per_hour) {
      parts.push(`${key.rate_limit_per_hour}/hr`)
    }
    return parts.length > 0 ? parts.join(', ') : 'Unlimited'
  }

  // Filter keys by search query
  const filteredKeys = serviceKeys?.filter(
    (key) =>
      key.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      key.description?.toLowerCase().includes(searchQuery.toLowerCase()) ||
      key.key_prefix.toLowerCase().includes(searchQuery.toLowerCase())
  )

  return (
    <div className='flex flex-1 flex-col gap-6 p-6'>
      <div>
        <h1 className='flex items-center gap-2 text-3xl font-bold tracking-tight'>
          <KeyRound className='h-8 w-8' />
          Service Keys
        </h1>
        <p className='text-muted-foreground mt-2'>
          Manage service keys for server-to-server API access (e.g., migrations, CLI tools)
        </p>
      </div>

      {/* Stats Cards */}
      <div className='grid gap-4 md:grid-cols-3'>
        <Card>
          <CardHeader className='flex flex-row items-center justify-between space-y-0 pb-2'>
            <CardTitle className='text-sm font-medium'>Total Keys</CardTitle>
            <KeyRound className='text-muted-foreground h-4 w-4' />
          </CardHeader>
          <CardContent>
            <div className='text-2xl font-bold'>{serviceKeys?.length || 0}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className='flex flex-row items-center justify-between space-y-0 pb-2'>
            <CardTitle className='text-sm font-medium'>Active Keys</CardTitle>
            <Check className='text-muted-foreground h-4 w-4' />
          </CardHeader>
          <CardContent>
            <div className='text-2xl font-bold'>
              {serviceKeys?.filter(
                (k) => k.enabled && !isExpired(k.expires_at)
              ).length || 0}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className='flex flex-row items-center justify-between space-y-0 pb-2'>
            <CardTitle className='text-sm font-medium'>Disabled Keys</CardTitle>
            <X className='text-muted-foreground h-4 w-4' />
          </CardHeader>
          <CardContent>
            <div className='text-2xl font-bold'>
              {serviceKeys?.filter((k) => !k.enabled).length || 0}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Main Card */}
      <Card>
        <CardHeader>
          <div className='flex items-center justify-between'>
            <div>
              <CardTitle>Service Keys</CardTitle>
              <CardDescription>
                Service keys are used for programmatic access to admin APIs like migrations
              </CardDescription>
            </div>
            <Button onClick={() => setShowCreateDialog(true)}>
              <Plus className='mr-2 h-4 w-4' />
              Create Service Key
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
                        {formatRateLimit(key)}
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
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <Button
                                variant='ghost'
                                size='sm'
                                onClick={() => openEditDialog(key)}
                              >
                                <Pencil className='h-4 w-4' />
                              </Button>
                            </TooltipTrigger>
                            <TooltipContent>Edit service key</TooltipContent>
                          </Tooltip>
                          {key.enabled ? (
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Button
                                  variant='ghost'
                                  size='sm'
                                  onClick={() => disableMutation.mutate(key.id)}
                                  disabled={disableMutation.isPending}
                                >
                                  <PowerOff className='h-4 w-4' />
                                </Button>
                              </TooltipTrigger>
                              <TooltipContent>Disable service key</TooltipContent>
                            </Tooltip>
                          ) : (
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Button
                                  variant='ghost'
                                  size='sm'
                                  onClick={() => enableMutation.mutate(key.id)}
                                  disabled={enableMutation.isPending}
                                >
                                  <Power className='h-4 w-4' />
                                </Button>
                              </TooltipTrigger>
                              <TooltipContent>Enable service key</TooltipContent>
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
                              <TooltipContent>Delete service key</TooltipContent>
                            </Tooltip>
                            <AlertDialogContent>
                              <AlertDialogHeader>
                                <AlertDialogTitle>
                                  Delete Service Key
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
              <KeyRound className='text-muted-foreground mb-4 h-12 w-12' />
              <p className='text-muted-foreground'>
                {searchQuery
                  ? 'No service keys match your search'
                  : 'No service keys yet'}
              </p>
              {!searchQuery && (
                <Button
                  onClick={() => setShowCreateDialog(true)}
                  variant='outline'
                  className='mt-4'
                >
                  Create Your First Service Key
                </Button>
              )}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Create Service Key Dialog */}
      <Dialog open={showCreateDialog} onOpenChange={setShowCreateDialog}>
        <DialogContent className='max-h-[90vh] max-w-xl overflow-y-auto'>
          <DialogHeader>
            <DialogTitle>Create Service Key</DialogTitle>
            <DialogDescription>
              Generate a new service key for server-to-server API access. The key will be
              shown only once.
            </DialogDescription>
          </DialogHeader>
          <div className='grid gap-4 py-4'>
            <div className='grid gap-2'>
              <Label htmlFor='name'>
                Name <span className='text-destructive'>*</span>
              </Label>
              <Input
                id='name'
                placeholder='Migrations Key'
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
            </div>
            <div className='grid gap-2'>
              <Label htmlFor='description'>Description</Label>
              <Input
                id='description'
                placeholder='Used by CI/CD pipeline for migrations'
                value={description}
                onChange={(e) => setDescription(e.target.value)}
              />
            </div>
            <div className='grid gap-2'>
              <Label>Scopes</Label>
              <div className='grid gap-2 rounded-md border p-4'>
                {AVAILABLE_SCOPES.map((scope) => (
                  <div key={scope.id} className='flex items-center space-x-2'>
                    <input
                      type='checkbox'
                      id={scope.id}
                      checked={selectedScopes.includes(scope.id)}
                      onChange={(e) => {
                        if (e.target.checked) {
                          setSelectedScopes([...selectedScopes, scope.id])
                        } else {
                          setSelectedScopes(selectedScopes.filter((s) => s !== scope.id))
                        }
                      }}
                      className='h-4 w-4 rounded border-gray-300'
                    />
                    <label htmlFor={scope.id} className='text-sm'>
                      <span className='font-medium'>{scope.name}</span>
                      <span className='text-muted-foreground ml-2'>{scope.description}</span>
                    </label>
                  </div>
                ))}
              </div>
            </div>
            <div className='grid grid-cols-2 gap-4'>
              <div className='grid gap-2'>
                <Label htmlFor='rateLimitPerMinute'>
                  Rate Limit (per minute)
                </Label>
                <Input
                  id='rateLimitPerMinute'
                  type='number'
                  min='0'
                  placeholder='Unlimited'
                  value={rateLimitPerMinute ?? ''}
                  onChange={(e) => setRateLimitPerMinute(e.target.value ? parseInt(e.target.value) : undefined)}
                />
              </div>
              <div className='grid gap-2'>
                <Label htmlFor='rateLimitPerHour'>
                  Rate Limit (per hour)
                </Label>
                <Input
                  id='rateLimitPerHour'
                  type='number'
                  min='0'
                  placeholder='Unlimited'
                  value={rateLimitPerHour ?? ''}
                  onChange={(e) => setRateLimitPerHour(e.target.value ? parseInt(e.target.value) : undefined)}
                />
              </div>
            </div>
            <div className='grid gap-2'>
              <Label htmlFor='expiresAt'>Expiration Date (optional)</Label>
              <Input
                id='expiresAt'
                type='datetime-local'
                value={expiresAt}
                onChange={(e) => setExpiresAt(e.target.value)}
              />
              <p className='text-muted-foreground text-xs'>
                Leave empty for no expiration
              </p>
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
              {createMutation.isPending ? 'Creating...' : 'Generate Service Key'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Service Key Dialog */}
      <Dialog open={showEditDialog} onOpenChange={setShowEditDialog}>
        <DialogContent className='max-w-xl'>
          <DialogHeader>
            <DialogTitle>Edit Service Key</DialogTitle>
            <DialogDescription>
              Update service key properties. The key value cannot be changed.
            </DialogDescription>
          </DialogHeader>
          <div className='grid gap-4 py-4'>
            <div className='grid gap-2'>
              <Label htmlFor='editName'>Name</Label>
              <Input
                id='editName'
                value={editName}
                onChange={(e) => setEditName(e.target.value)}
              />
            </div>
            <div className='grid gap-2'>
              <Label htmlFor='editDescription'>Description</Label>
              <Input
                id='editDescription'
                value={editDescription}
                onChange={(e) => setEditDescription(e.target.value)}
              />
            </div>
            <div className='grid grid-cols-2 gap-4'>
              <div className='grid gap-2'>
                <Label htmlFor='editRateLimitPerMinute'>
                  Rate Limit (per minute)
                </Label>
                <Input
                  id='editRateLimitPerMinute'
                  type='number'
                  min='0'
                  placeholder='Unlimited'
                  value={editRateLimitPerMinute ?? ''}
                  onChange={(e) => setEditRateLimitPerMinute(e.target.value ? parseInt(e.target.value) : undefined)}
                />
              </div>
              <div className='grid gap-2'>
                <Label htmlFor='editRateLimitPerHour'>
                  Rate Limit (per hour)
                </Label>
                <Input
                  id='editRateLimitPerHour'
                  type='number'
                  min='0'
                  placeholder='Unlimited'
                  value={editRateLimitPerHour ?? ''}
                  onChange={(e) => setEditRateLimitPerHour(e.target.value ? parseInt(e.target.value) : undefined)}
                />
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button
              variant='outline'
              onClick={() => setShowEditDialog(false)}
            >
              Cancel
            </Button>
            <Button
              onClick={handleEditKey}
              disabled={updateMutation.isPending}
            >
              {updateMutation.isPending ? 'Saving...' : 'Save Changes'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Show Created Key Dialog */}
      <Dialog open={showKeyDialog} onOpenChange={setShowKeyDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Service Key Created</DialogTitle>
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
                      This is the only time you'll see the full service key. Store
                      it securely.
                    </p>
                  </div>
                </div>
              </div>
            </div>
            <div className='grid gap-2'>
              <Label>Service Key</Label>
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
