import { useState } from 'react'
import { formatDistanceToNow } from 'date-fns'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import {
  Lock,
  Plus,
  Trash2,
  AlertCircle,
  Search,
  History,
  RotateCcw,
  Clock,
  Globe,
  FolderOpen,
} from 'lucide-react'
import { toast } from 'sonner'
import {
  secretsApi,
  type Secret,
  type SecretVersion,
  type CreateSecretRequest,
  type UpdateSecretRequest,
  type SecretsStats,
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Textarea } from '@/components/ui/textarea'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

export const Route = createFileRoute('/_authenticated/secrets/')({
  component: SecretsPage,
})

function SecretsPage() {
  const queryClient = useQueryClient()
  const [showCreateDialog, setShowCreateDialog] = useState(false)
  const [showEditDialog, setShowEditDialog] = useState(false)
  const [showHistoryDialog, setShowHistoryDialog] = useState(false)
  const [selectedSecret, setSelectedSecret] = useState<Secret | null>(null)
  const [searchQuery, setSearchQuery] = useState('')
  const [scopeFilter, setScopeFilter] = useState<string>('')

  // Form state
  const [name, setName] = useState('')
  const [value, setValue] = useState('')
  const [scope, setScope] = useState<'global' | 'namespace'>('global')
  const [namespace, setNamespace] = useState('')
  const [description, setDescription] = useState('')
  const [expiresAt, setExpiresAt] = useState('')

  // Fetch secrets
  const { data: secrets, isLoading } = useQuery<Secret[]>({
    queryKey: ['secrets', scopeFilter],
    queryFn: () => secretsApi.list(scopeFilter || undefined),
  })

  // Fetch stats
  const { data: stats } = useQuery<SecretsStats>({
    queryKey: ['secrets-stats'],
    queryFn: secretsApi.getStats,
  })

  // Fetch versions for selected secret
  const { data: versions } = useQuery<SecretVersion[]>({
    queryKey: ['secret-versions', selectedSecret?.id],
    queryFn: () => secretsApi.getVersions(selectedSecret!.id),
    enabled: !!selectedSecret && showHistoryDialog,
  })

  // Create secret
  const createMutation = useMutation({
    mutationFn: secretsApi.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['secrets'] })
      queryClient.invalidateQueries({ queryKey: ['secrets-stats'] })
      setShowCreateDialog(false)
      resetForm()
      toast.success('Secret created successfully')
    },
    onError: (error: Error) => {
      toast.error(`Failed to create secret: ${error.message}`)
    },
  })

  // Update secret
  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateSecretRequest }) =>
      secretsApi.update(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['secrets'] })
      setShowEditDialog(false)
      setSelectedSecret(null)
      resetForm()
      toast.success('Secret updated successfully')
    },
    onError: (error: Error) => {
      toast.error(`Failed to update secret: ${error.message}`)
    },
  })

  // Delete secret
  const deleteMutation = useMutation({
    mutationFn: secretsApi.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['secrets'] })
      queryClient.invalidateQueries({ queryKey: ['secrets-stats'] })
      toast.success('Secret deleted successfully')
    },
    onError: (error: Error) => {
      toast.error(`Failed to delete secret: ${error.message}`)
    },
  })

  // Rollback secret
  const rollbackMutation = useMutation({
    mutationFn: ({ id, version }: { id: string; version: number }) =>
      secretsApi.rollback(id, version),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['secrets'] })
      queryClient.invalidateQueries({ queryKey: ['secret-versions'] })
      toast.success('Secret rolled back successfully')
    },
    onError: (error: Error) => {
      toast.error(`Failed to rollback secret: ${error.message}`)
    },
  })

  const resetForm = () => {
    setName('')
    setValue('')
    setScope('global')
    setNamespace('')
    setDescription('')
    setExpiresAt('')
  }

  const handleCreateSecret = () => {
    if (!name.trim()) {
      toast.error('Please enter a secret name')
      return
    }
    if (!value.trim()) {
      toast.error('Please enter a secret value')
      return
    }
    if (scope === 'namespace' && !namespace.trim()) {
      toast.error('Please enter a namespace for namespace-scoped secrets')
      return
    }

    const request: CreateSecretRequest = {
      name: name
        .trim()
        .toUpperCase()
        .replace(/[^A-Z0-9_]/g, '_'),
      value: value,
      scope: scope,
      namespace: scope === 'namespace' ? namespace.trim() : undefined,
      description: description.trim() || undefined,
      expires_at: expiresAt ? new Date(expiresAt).toISOString() : undefined,
    }

    createMutation.mutate(request)
  }

  const handleUpdateSecret = () => {
    if (!selectedSecret) return

    const request: UpdateSecretRequest = {}
    if (value.trim()) {
      request.value = value
    }
    if (description !== selectedSecret.description) {
      request.description = description.trim() || undefined
    }

    updateMutation.mutate({ id: selectedSecret.id, data: request })
  }

  const openEditDialog = (secret: Secret) => {
    setSelectedSecret(secret)
    setDescription(secret.description || '')
    setValue('')
    setShowEditDialog(true)
  }

  const openHistoryDialog = (secret: Secret) => {
    setSelectedSecret(secret)
    setShowHistoryDialog(true)
  }

  // Filter secrets by search query
  const filteredSecrets = secrets?.filter(
    (secret) =>
      secret.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      secret.description?.toLowerCase().includes(searchQuery.toLowerCase()) ||
      secret.namespace?.toLowerCase().includes(searchQuery.toLowerCase())
  )

  return (
    <div className='flex flex-1 flex-col gap-6 p-6'>
      <div>
        <h1 className='flex items-center gap-2 text-3xl font-bold tracking-tight'>
          <Lock className='h-8 w-8' />
          Secrets
        </h1>
        <p className='text-muted-foreground mt-2'>
          Manage encrypted secrets that are injected into edge functions and
          background jobs at runtime
        </p>
      </div>

      {/* Stats Cards */}
      <div className='grid gap-4 md:grid-cols-3'>
        <Card>
          <CardHeader className='flex flex-row items-center justify-between space-y-0 pb-2'>
            <CardTitle className='text-sm font-medium'>Total Secrets</CardTitle>
            <Lock className='text-muted-foreground h-4 w-4' />
          </CardHeader>
          <CardContent>
            <div className='text-2xl font-bold'>{stats?.total || 0}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className='flex flex-row items-center justify-between space-y-0 pb-2'>
            <CardTitle className='text-sm font-medium'>Expiring Soon</CardTitle>
            <Clock className='text-muted-foreground h-4 w-4' />
          </CardHeader>
          <CardContent>
            <div className='text-2xl font-bold text-yellow-600'>
              {stats?.expiring_soon || 0}
            </div>
            <p className='text-muted-foreground text-xs'>Within 7 days</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className='flex flex-row items-center justify-between space-y-0 pb-2'>
            <CardTitle className='text-sm font-medium'>Expired</CardTitle>
            <AlertCircle className='text-muted-foreground h-4 w-4' />
          </CardHeader>
          <CardContent>
            <div className='text-destructive text-2xl font-bold'>
              {stats?.expired || 0}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Main Card */}
      <Card>
        <CardHeader>
          <div className='flex items-center justify-between'>
            <div>
              <CardTitle>Secrets</CardTitle>
              <CardDescription>
                Secrets are available as FLUXBASE_SECRET_NAME environment
                variables in edge functions and background jobs
              </CardDescription>
            </div>
            <Button onClick={() => setShowCreateDialog(true)}>
              <Plus className='mr-2 h-4 w-4' />
              Create Secret
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {/* Filters */}
          <div className='mb-4 flex gap-4'>
            <div className='relative flex-1'>
              <Search className='text-muted-foreground absolute top-2.5 left-2 h-4 w-4' />
              <Input
                placeholder='Search by name, description, or namespace...'
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className='pl-8'
              />
            </div>
            <Select value={scopeFilter} onValueChange={setScopeFilter}>
              <SelectTrigger className='w-[180px]'>
                <SelectValue placeholder='All scopes' />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value='all'>All scopes</SelectItem>
                <SelectItem value='global'>Global</SelectItem>
                <SelectItem value='namespace'>Namespace</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {isLoading ? (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Scope</TableHead>
                  <TableHead>Version</TableHead>
                  <TableHead>Expires</TableHead>
                  <TableHead>Updated</TableHead>
                  <TableHead className='text-right'>Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {Array(3)
                  .fill(0)
                  .map((_, i) => (
                    <TableRow key={i}>
                      <TableCell>
                        <Skeleton className='h-4 w-28' />
                      </TableCell>
                      <TableCell>
                        <Skeleton className='h-5 w-16' />
                      </TableCell>
                      <TableCell>
                        <Skeleton className='h-4 w-8' />
                      </TableCell>
                      <TableCell>
                        <Skeleton className='h-4 w-20' />
                      </TableCell>
                      <TableCell>
                        <Skeleton className='h-4 w-24' />
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
          ) : filteredSecrets && filteredSecrets.length > 0 ? (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Scope</TableHead>
                  <TableHead>Version</TableHead>
                  <TableHead>Expires</TableHead>
                  <TableHead>Updated</TableHead>
                  <TableHead className='text-right'>Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredSecrets.map((secret) => (
                  <TableRow key={secret.id}>
                    <TableCell>
                      <div>
                        <div className='font-mono font-medium'>
                          FLUXBASE_SECRET_{secret.name}
                        </div>
                        {secret.description && (
                          <div className='text-muted-foreground text-xs'>
                            {secret.description}
                          </div>
                        )}
                      </div>
                    </TableCell>
                    <TableCell>
                      <div className='flex items-center gap-1'>
                        {secret.scope === 'global' ? (
                          <Badge variant='default' className='gap-1'>
                            <Globe className='h-3 w-3' />
                            Global
                          </Badge>
                        ) : (
                          <Badge variant='secondary' className='gap-1'>
                            <FolderOpen className='h-3 w-3' />
                            {secret.namespace}
                          </Badge>
                        )}
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant='outline'>v{secret.version}</Badge>
                    </TableCell>
                    <TableCell>
                      {secret.expires_at ? (
                        <span
                          className={
                            secret.is_expired ? 'text-destructive' : ''
                          }
                        >
                          {secret.is_expired
                            ? 'Expired'
                            : formatDistanceToNow(new Date(secret.expires_at), {
                                addSuffix: true,
                              })}
                        </span>
                      ) : (
                        <span className='text-muted-foreground'>Never</span>
                      )}
                    </TableCell>
                    <TableCell className='text-muted-foreground text-sm'>
                      {formatDistanceToNow(new Date(secret.updated_at), {
                        addSuffix: true,
                      })}
                    </TableCell>
                    <TableCell className='text-right'>
                      <div className='flex justify-end gap-1'>
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <Button
                              variant='ghost'
                              size='sm'
                              onClick={() => openEditDialog(secret)}
                            >
                              <Lock className='h-4 w-4' />
                            </Button>
                          </TooltipTrigger>
                          <TooltipContent>Update secret value</TooltipContent>
                        </Tooltip>
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <Button
                              variant='ghost'
                              size='sm'
                              onClick={() => openHistoryDialog(secret)}
                            >
                              <History className='h-4 w-4' />
                            </Button>
                          </TooltipTrigger>
                          <TooltipContent>Version history</TooltipContent>
                        </Tooltip>
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
                            <TooltipContent>Delete secret</TooltipContent>
                          </Tooltip>
                          <AlertDialogContent>
                            <AlertDialogHeader>
                              <AlertDialogTitle>Delete Secret</AlertDialogTitle>
                              <AlertDialogDescription>
                                Are you sure you want to delete "{secret.name}"?
                                This action cannot be undone and any functions
                                or jobs using this secret will fail.
                              </AlertDialogDescription>
                            </AlertDialogHeader>
                            <AlertDialogFooter>
                              <AlertDialogCancel>Cancel</AlertDialogCancel>
                              <AlertDialogAction
                                onClick={() => deleteMutation.mutate(secret.id)}
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
                ))}
              </TableBody>
            </Table>
          ) : (
            <div className='flex flex-col items-center justify-center py-12 text-center'>
              <Lock className='text-muted-foreground mb-4 h-12 w-12' />
              <p className='text-muted-foreground'>
                {searchQuery
                  ? 'No secrets match your search'
                  : 'No secrets yet'}
              </p>
              {!searchQuery && (
                <Button
                  onClick={() => setShowCreateDialog(true)}
                  variant='outline'
                  className='mt-4'
                >
                  Create Your First Secret
                </Button>
              )}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Create Secret Dialog */}
      <Dialog open={showCreateDialog} onOpenChange={setShowCreateDialog}>
        <DialogContent className='max-w-lg'>
          <DialogHeader>
            <DialogTitle>Create Secret</DialogTitle>
            <DialogDescription>
              Create a new encrypted secret. The value will be securely stored
              and available to edge functions and background jobs.
            </DialogDescription>
          </DialogHeader>
          <div className='grid gap-4 py-4'>
            <div className='grid gap-2'>
              <Label htmlFor='name'>
                Name <span className='text-destructive'>*</span>
              </Label>
              <Input
                id='name'
                placeholder='API_KEY'
                value={name}
                onChange={(e) => setName(e.target.value.toUpperCase())}
              />
              <p className='text-muted-foreground text-xs'>
                Available as FLUXBASE_SECRET_{name || 'NAME'}
              </p>
            </div>
            <div className='grid gap-2'>
              <Label htmlFor='value'>
                Value <span className='text-destructive'>*</span>
              </Label>
              <Textarea
                id='value'
                placeholder='Enter secret value...'
                value={value}
                onChange={(e) => setValue(e.target.value)}
                className='font-mono'
              />
            </div>
            <div className='grid gap-2'>
              <Label htmlFor='scope'>Scope</Label>
              <Select
                value={scope}
                onValueChange={(v) => setScope(v as 'global' | 'namespace')}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value='global'>Global (all functions)</SelectItem>
                  <SelectItem value='namespace'>
                    Namespace (specific namespace)
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>
            {scope === 'namespace' && (
              <div className='grid gap-2'>
                <Label htmlFor='namespace'>
                  Namespace <span className='text-destructive'>*</span>
                </Label>
                <Input
                  id='namespace'
                  placeholder='my-namespace'
                  value={namespace}
                  onChange={(e) => setNamespace(e.target.value)}
                />
              </div>
            )}
            <div className='grid gap-2'>
              <Label htmlFor='description'>Description</Label>
              <Input
                id='description'
                placeholder='Optional description...'
                value={description}
                onChange={(e) => setDescription(e.target.value)}
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
              <p className='text-muted-foreground text-xs'>
                Expired secrets are automatically excluded from function and job
                execution
              </p>
            </div>
          </div>
          <DialogFooter>
            <Button
              variant='outline'
              onClick={() => {
                setShowCreateDialog(false)
                resetForm()
              }}
            >
              Cancel
            </Button>
            <Button
              onClick={handleCreateSecret}
              disabled={createMutation.isPending}
            >
              {createMutation.isPending ? 'Creating...' : 'Create Secret'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Secret Dialog */}
      <Dialog open={showEditDialog} onOpenChange={setShowEditDialog}>
        <DialogContent className='max-w-lg'>
          <DialogHeader>
            <DialogTitle>Update Secret</DialogTitle>
            <DialogDescription>
              Update the value for {selectedSecret?.name}. This will create a
              new version.
            </DialogDescription>
          </DialogHeader>
          <div className='grid gap-4 py-4'>
            <div className='rounded-md bg-yellow-50 p-4 dark:bg-yellow-950'>
              <div className='flex'>
                <AlertCircle className='h-5 w-5 text-yellow-600 dark:text-yellow-400' />
                <div className='ml-3'>
                  <p className='text-sm text-yellow-700 dark:text-yellow-300'>
                    The current secret value cannot be viewed. Enter a new value
                    to update.
                  </p>
                </div>
              </div>
            </div>
            <div className='grid gap-2'>
              <Label htmlFor='editValue'>New Value</Label>
              <Textarea
                id='editValue'
                placeholder='Enter new secret value...'
                value={value}
                onChange={(e) => setValue(e.target.value)}
                className='font-mono'
              />
              <p className='text-muted-foreground text-xs'>
                Leave empty to keep the current value
              </p>
            </div>
            <div className='grid gap-2'>
              <Label htmlFor='editDescription'>Description</Label>
              <Input
                id='editDescription'
                placeholder='Optional description...'
                value={description}
                onChange={(e) => setDescription(e.target.value)}
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              variant='outline'
              onClick={() => {
                setShowEditDialog(false)
                setSelectedSecret(null)
                resetForm()
              }}
            >
              Cancel
            </Button>
            <Button
              onClick={handleUpdateSecret}
              disabled={updateMutation.isPending}
            >
              {updateMutation.isPending ? 'Updating...' : 'Update Secret'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Version History Dialog */}
      <Dialog open={showHistoryDialog} onOpenChange={setShowHistoryDialog}>
        <DialogContent className='max-w-lg'>
          <DialogHeader>
            <DialogTitle>Version History</DialogTitle>
            <DialogDescription>
              Version history for {selectedSecret?.name}. Current version: v
              {selectedSecret?.version}
            </DialogDescription>
          </DialogHeader>
          <div className='py-4'>
            {versions && versions.length > 0 ? (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Version</TableHead>
                    <TableHead>Created</TableHead>
                    <TableHead className='text-right'>Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {versions.map((version) => (
                    <TableRow key={version.id}>
                      <TableCell>
                        <Badge
                          variant={
                            version.version === selectedSecret?.version
                              ? 'default'
                              : 'outline'
                          }
                        >
                          v{version.version}
                          {version.version === selectedSecret?.version &&
                            ' (current)'}
                        </Badge>
                      </TableCell>
                      <TableCell className='text-muted-foreground text-sm'>
                        {formatDistanceToNow(new Date(version.created_at), {
                          addSuffix: true,
                        })}
                      </TableCell>
                      <TableCell className='text-right'>
                        {version.version !== selectedSecret?.version && (
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <Button
                                variant='ghost'
                                size='sm'
                                onClick={() => {
                                  if (selectedSecret) {
                                    rollbackMutation.mutate({
                                      id: selectedSecret.id,
                                      version: version.version,
                                    })
                                  }
                                }}
                                disabled={rollbackMutation.isPending}
                              >
                                <RotateCcw className='h-4 w-4' />
                              </Button>
                            </TooltipTrigger>
                            <TooltipContent>
                              Rollback to v{version.version}
                            </TooltipContent>
                          </Tooltip>
                        )}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            ) : (
              <p className='text-muted-foreground text-center text-sm'>
                No version history available
              </p>
            )}
          </div>
          <DialogFooter>
            <Button
              onClick={() => {
                setShowHistoryDialog(false)
                setSelectedSecret(null)
              }}
            >
              Close
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
