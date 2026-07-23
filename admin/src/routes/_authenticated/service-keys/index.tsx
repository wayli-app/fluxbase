import { useState } from 'react'
import { formatDistanceToNow } from 'date-fns'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import {
  KeyRound,
  Plus,
  Trash2,
  Check,
  X,
  Search,
  Power,
  PowerOff,
  Pencil,
  ShieldAlert,
  Clock,
  RefreshCw,
  History,
} from 'lucide-react'
import { toast } from 'sonner'
import { useTenantStore } from '@/stores/tenant-store'
import {
  serviceKeysApi,
  type ServiceKey,
  type ServiceKeyWithPlaintext,
  type CreateServiceKeyRequest,
  type UpdateServiceKeyRequest,
  type RevokeServiceKeyRequest,
  type DeprecateServiceKeyRequest,
  type RotateServiceKeyRequest,
  type RotateServiceKeyResponse,
  type ServiceKeyRevocation,
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
import { Input } from '@/components/ui/input'
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
import {
  CreateServiceKeyDialog,
  EditServiceKeyDialog,
  CreatedKeyDialog,
  RevokeKeyDialog,
  DeprecateKeyDialog,
  TenantRotateKeyDialog,
  RotatedKeyDialog,
  HistoryDialog,
  isExpired,
  getKeyStatus,
  canModify,
  formatRateLimit,
} from '@/components/service-keys'

export const Route = createFileRoute('/_authenticated/service-keys/')({
  component: ServiceKeysPage,
})

function ServiceKeysPage() {
  const queryClient = useQueryClient()
  const currentTenantId = useTenantStore((state) => state.currentTenant?.id)
  const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false)
  const [isKeyDialogOpen, setIsKeyDialogOpen] = useState(false)
  const [isEditDialogOpen, setIsEditDialogOpen] = useState(false)
  const [createdKey, setCreatedKey] = useState<ServiceKeyWithPlaintext | null>(
    null
  )
  const [editingKey, setEditingKey] = useState<ServiceKey | null>(null)
  const [searchQuery, setSearchQuery] = useState('')

  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [selectedScopes, setSelectedScopes] = useState<string[]>(['*'])
  const [rateLimitPerMinute, setRateLimitPerMinute] = useState<
    number | undefined
  >(undefined)
  const [rateLimitPerHour, setRateLimitPerHour] = useState<number | undefined>(
    undefined
  )
  const [expiresAt, setExpiresAt] = useState('')

  const [editName, setEditName] = useState('')
  const [editDescription, setEditDescription] = useState('')
  const [editScopes, setEditScopes] = useState<string[]>([])
  const [editRateLimitPerMinute, setEditRateLimitPerMinute] = useState<
    number | undefined
  >(undefined)
  const [editRateLimitPerHour, setEditRateLimitPerHour] = useState<
    number | undefined
  >(undefined)

  const [isRevokeDialogOpen, setIsRevokeDialogOpen] = useState(false)
  const [isDeprecateDialogOpen, setIsDeprecateDialogOpen] = useState(false)
  const [isRotateDialogOpen, setIsRotateDialogOpen] = useState(false)
  const [isRotatedKeyDialogOpen, setIsRotatedKeyDialogOpen] = useState(false)
  const [isHistoryDialogOpen, setIsHistoryDialogOpen] = useState(false)
  const [targetKey, setTargetKey] = useState<ServiceKey | null>(null)
  const [revokeReason, setRevokeReason] = useState('')
  const [deprecateReason, setDeprecateReason] = useState('')
  const [gracePeriod, setGracePeriod] = useState('24h')
  const [rotatedKey, setRotatedKey] = useState<RotateServiceKeyResponse | null>(
    null
  )
  const [revocationHistory, setRevocationHistory] = useState<
    ServiceKeyRevocation[]
  >([])

  const { data: serviceKeys, isLoading } = useQuery<ServiceKey[]>({
    queryKey: ['service-keys', currentTenantId],
    queryFn: serviceKeysApi.list,
    enabled: !!currentTenantId,
  })

  const createMutation = useMutation({
    mutationFn: serviceKeysApi.create,
    onSuccess: (data) => {
      queryClient.invalidateQueries({
        queryKey: ['service-keys', currentTenantId],
      })
      setCreatedKey(data)
      setIsCreateDialogOpen(false)
      setIsKeyDialogOpen(true)
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

  const updateMutation = useMutation({
    mutationFn: ({
      id,
      request,
    }: {
      id: string
      request: UpdateServiceKeyRequest
    }) => serviceKeysApi.update(id, request),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ['service-keys', currentTenantId],
      })
      setIsEditDialogOpen(false)
      setEditingKey(null)
      toast.success('Service key updated successfully')
    },
    onError: () => {
      toast.error('Failed to update service key')
    },
  })

  const enableMutation = useMutation({
    mutationFn: serviceKeysApi.enable,
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ['service-keys', currentTenantId],
      })
      toast.success('Service key enabled')
    },
    onError: () => {
      toast.error('Failed to enable service key')
    },
  })

  const disableMutation = useMutation({
    mutationFn: serviceKeysApi.disable,
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ['service-keys', currentTenantId],
      })
      toast.success('Service key disabled')
    },
    onError: () => {
      toast.error('Failed to disable service key')
    },
  })

  const deleteMutation = useMutation({
    mutationFn: serviceKeysApi.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ['service-keys', currentTenantId],
      })
      toast.success('Service key deleted successfully')
    },
    onError: () => {
      toast.error('Failed to delete service key')
    },
  })

  const revokeMutation = useMutation({
    mutationFn: ({
      id,
      request,
    }: {
      id: string
      request: RevokeServiceKeyRequest
    }) => serviceKeysApi.revoke(id, request),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ['service-keys', currentTenantId],
      })
      setIsRevokeDialogOpen(false)
      setTargetKey(null)
      setRevokeReason('')
      toast.success('Service key revoked')
    },
    onError: () => {
      toast.error('Failed to revoke service key')
    },
  })

  const deprecateMutation = useMutation({
    mutationFn: ({
      id,
      request,
    }: {
      id: string
      request: DeprecateServiceKeyRequest
    }) => serviceKeysApi.deprecate(id, request),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ['service-keys', currentTenantId],
      })
      setIsDeprecateDialogOpen(false)
      setTargetKey(null)
      setDeprecateReason('')
      setGracePeriod('24h')
      toast.success('Service key deprecated')
    },
    onError: () => {
      toast.error('Failed to deprecate service key')
    },
  })

  const rotateMutation = useMutation({
    mutationFn: ({
      id,
      request,
    }: {
      id: string
      request: RotateServiceKeyRequest
    }) => serviceKeysApi.rotate(id, request),
    onSuccess: (data) => {
      queryClient.invalidateQueries({
        queryKey: ['service-keys', currentTenantId],
      })
      setRotatedKey(data)
      setIsRotateDialogOpen(false)
      setIsRotatedKeyDialogOpen(true)
      setTargetKey(null)
      setGracePeriod('24h')
    },
    onError: () => {
      toast.error('Failed to rotate service key')
    },
  })

  const handleCreateKey = (request: CreateServiceKeyRequest) => {
    if (!request.name?.trim()) {
      toast.error('Please enter a key name')
      return
    }
    createMutation.mutate(request)
  }

  const handleEditKey = (request: UpdateServiceKeyRequest) => {
    if (!editingKey) return
    updateMutation.mutate({ id: editingKey.id, request })
  }

  const openEditDialog = (key: ServiceKey) => {
    setEditingKey(key)
    setEditName(key.name)
    setEditDescription(key.description || '')
    setEditScopes(key.scopes || ['*'])
    setEditRateLimitPerMinute(key.rate_limit_per_minute)
    setEditRateLimitPerHour(key.rate_limit_per_hour)
    setIsEditDialogOpen(true)
  }

  const openRevokeDialog = (key: ServiceKey) => {
    setTargetKey(key)
    setRevokeReason('')
    setIsRevokeDialogOpen(true)
  }

  const openDeprecateDialog = (key: ServiceKey) => {
    setTargetKey(key)
    setDeprecateReason('')
    setGracePeriod('24h')
    setIsDeprecateDialogOpen(true)
  }

  const openRotateDialog = (key: ServiceKey) => {
    setTargetKey(key)
    setGracePeriod('24h')
    setIsRotateDialogOpen(true)
  }

  const openHistoryDialog = async (key: ServiceKey) => {
    setTargetKey(key)
    try {
      const history = await serviceKeysApi.revocations(key.id)
      setRevocationHistory(history)
      setIsHistoryDialogOpen(true)
    } catch {
      toast.error('Failed to load revocation history')
    }
  }

  const handleRevoke = () => {
    if (!targetKey || !revokeReason.trim()) {
      toast.error('Please provide a reason for revocation')
      return
    }
    revokeMutation.mutate({
      id: targetKey.id,
      request: { reason: revokeReason.trim() },
    })
  }

  const handleDeprecate = () => {
    if (!targetKey) return
    deprecateMutation.mutate({
      id: targetKey.id,
      request: {
        grace_period: gracePeriod,
        reason: deprecateReason.trim() || undefined,
      },
    })
  }

  const handleRotate = () => {
    if (!targetKey) return
    rotateMutation.mutate({
      id: targetKey.id,
      request: { grace_period: gracePeriod },
    })
  }

  const filteredKeys = serviceKeys?.filter(
    (key) =>
      key.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      key.description?.toLowerCase().includes(searchQuery.toLowerCase()) ||
      key.key_prefix.toLowerCase().includes(searchQuery.toLowerCase())
  )

  return (
    <div className='flex h-full flex-col'>
      <div className='flex-1 overflow-auto p-6'>
        <div className='flex flex-col gap-6'>
          <div className='grid gap-4 md:grid-cols-3'>
            <Card>
              <CardHeader className='flex flex-row items-center justify-between space-y-0 pb-2'>
                <CardTitle className='text-sm font-medium'>
                  Total Keys
                </CardTitle>
                <KeyRound className='text-muted-foreground h-4 w-4' />
              </CardHeader>
              <CardContent>
                <div className='text-2xl font-bold'>
                  {serviceKeys?.length || 0}
                </div>
              </CardContent>
            </Card>
            <Card>
              <CardHeader className='flex flex-row items-center justify-between space-y-0 pb-2'>
                <CardTitle className='text-sm font-medium'>
                  Active Keys
                </CardTitle>
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
                <CardTitle className='text-sm font-medium'>
                  Disabled Keys
                </CardTitle>
                <X className='text-muted-foreground h-4 w-4' />
              </CardHeader>
              <CardContent>
                <div className='text-2xl font-bold'>
                  {serviceKeys?.filter((k) => !k.enabled).length || 0}
                </div>
              </CardContent>
            </Card>
          </div>

          <Card>
            <CardHeader>
              <div className='flex items-center justify-between'>
                <div>
                  <CardTitle>Service Keys</CardTitle>
                  <CardDescription>
                    Service keys are used for programmatic access to admin APIs
                    like migrations
                  </CardDescription>
                </div>
                <Button onClick={() => setIsCreateDialogOpen(true)}>
                  <Plus className='mr-2 h-4 w-4' />
                  Create Service Key
                </Button>
              </div>
            </CardHeader>
            <CardContent>
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
                              ? formatDistanceToNow(
                                  new Date(key.last_used_at),
                                  {
                                    addSuffix: true,
                                  }
                                )
                              : 'Never'}
                          </TableCell>
                          <TableCell>
                            <Badge variant={status.variant}>
                              {status.label}
                            </Badge>
                          </TableCell>
                          <TableCell className='text-right'>
                            <div className='flex justify-end gap-1'>
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <Button
                                    variant='ghost'
                                    size='sm'
                                    onClick={() => openHistoryDialog(key)}
                                  >
                                    <History className='h-4 w-4' />
                                  </Button>
                                </TooltipTrigger>
                                <TooltipContent>View history</TooltipContent>
                              </Tooltip>
                              {canModify(key) && (
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
                                  <TooltipContent>
                                    Edit service key
                                  </TooltipContent>
                                </Tooltip>
                              )}
                              {canModify(key) &&
                                key.enabled &&
                                !key.deprecated_at && (
                                  <Tooltip>
                                    <TooltipTrigger asChild>
                                      <Button
                                        variant='ghost'
                                        size='sm'
                                        onClick={() => openRotateDialog(key)}
                                      >
                                        <RefreshCw className='h-4 w-4' />
                                      </Button>
                                    </TooltipTrigger>
                                    <TooltipContent>Rotate key</TooltipContent>
                                  </Tooltip>
                                )}
                              {canModify(key) &&
                                key.enabled &&
                                !key.deprecated_at && (
                                  <Tooltip>
                                    <TooltipTrigger asChild>
                                      <Button
                                        variant='ghost'
                                        size='sm'
                                        onClick={() => openDeprecateDialog(key)}
                                      >
                                        <Clock className='h-4 w-4' />
                                      </Button>
                                    </TooltipTrigger>
                                    <TooltipContent>
                                      Deprecate with grace period
                                    </TooltipContent>
                                  </Tooltip>
                                )}
                              {canModify(key) &&
                                !key.deprecated_at &&
                                (key.enabled ? (
                                  <Tooltip>
                                    <TooltipTrigger asChild>
                                      <Button
                                        variant='ghost'
                                        size='sm'
                                        onClick={() =>
                                          disableMutation.mutate(key.id)
                                        }
                                        disabled={disableMutation.isPending}
                                      >
                                        <PowerOff className='h-4 w-4' />
                                      </Button>
                                    </TooltipTrigger>
                                    <TooltipContent>
                                      Disable service key
                                    </TooltipContent>
                                  </Tooltip>
                                ) : (
                                  <Tooltip>
                                    <TooltipTrigger asChild>
                                      <Button
                                        variant='ghost'
                                        size='sm'
                                        onClick={() =>
                                          enableMutation.mutate(key.id)
                                        }
                                        disabled={enableMutation.isPending}
                                      >
                                        <Power className='h-4 w-4' />
                                      </Button>
                                    </TooltipTrigger>
                                    <TooltipContent>
                                      Enable service key
                                    </TooltipContent>
                                  </Tooltip>
                                ))}
                              {canModify(key) && (
                                <Tooltip>
                                  <TooltipTrigger asChild>
                                    <Button
                                      variant='ghost'
                                      size='sm'
                                      onClick={() => openRevokeDialog(key)}
                                      className='text-destructive hover:text-destructive hover:bg-destructive/10'
                                    >
                                      <ShieldAlert className='h-4 w-4' />
                                    </Button>
                                  </TooltipTrigger>
                                  <TooltipContent>
                                    Revoke (emergency)
                                  </TooltipContent>
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
                                  <TooltipContent>
                                    Delete service key
                                  </TooltipContent>
                                </Tooltip>
                                <AlertDialogContent>
                                  <AlertDialogHeader>
                                    <AlertDialogTitle>
                                      Delete Service Key
                                    </AlertDialogTitle>
                                    <AlertDialogDescription>
                                      Are you sure you want to delete "
                                      {key.name}"? Any applications using this
                                      key will lose access immediately.
                                    </AlertDialogDescription>
                                  </AlertDialogHeader>
                                  <AlertDialogFooter>
                                    <AlertDialogCancel>
                                      Cancel
                                    </AlertDialogCancel>
                                    <AlertDialogAction
                                      onClick={() =>
                                        deleteMutation.mutate(key.id)
                                      }
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
                      onClick={() => setIsCreateDialogOpen(true)}
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
        </div>
      </div>

      <CreateServiceKeyDialog
        open={isCreateDialogOpen}
        onOpenChange={setIsCreateDialogOpen}
        name={name}
        onNameChange={setName}
        description={description}
        onDescriptionChange={setDescription}
        selectedScopes={selectedScopes}
        onScopesChange={setSelectedScopes}
        rateLimitPerMinute={rateLimitPerMinute}
        onRateLimitPerMinuteChange={setRateLimitPerMinute}
        rateLimitPerHour={rateLimitPerHour}
        onRateLimitPerHourChange={setRateLimitPerHour}
        expiresAt={expiresAt}
        onExpiresAtChange={setExpiresAt}
        onSubmit={handleCreateKey}
        isPending={createMutation.isPending}
      />

      <EditServiceKeyDialog
        open={isEditDialogOpen}
        onOpenChange={setIsEditDialogOpen}
        editName={editName}
        onEditNameChange={setEditName}
        editDescription={editDescription}
        onEditDescriptionChange={setEditDescription}
        editScopes={editScopes}
        onEditScopesChange={setEditScopes}
        editRateLimitPerMinute={editRateLimitPerMinute}
        onEditRateLimitPerMinuteChange={setEditRateLimitPerMinute}
        editRateLimitPerHour={editRateLimitPerHour}
        onEditRateLimitPerHourChange={setEditRateLimitPerHour}
        onSubmit={handleEditKey}
        isPending={updateMutation.isPending}
      />

      <CreatedKeyDialog
        open={isKeyDialogOpen}
        onOpenChange={setIsKeyDialogOpen}
        createdKey={createdKey}
      />

      <RevokeKeyDialog
        open={isRevokeDialogOpen}
        onOpenChange={setIsRevokeDialogOpen}
        targetKey={targetKey}
        revokeReason={revokeReason}
        onRevokeReasonChange={setRevokeReason}
        onRevoke={handleRevoke}
        isPending={revokeMutation.isPending}
      />

      <DeprecateKeyDialog
        open={isDeprecateDialogOpen}
        onOpenChange={setIsDeprecateDialogOpen}
        targetKey={targetKey}
        gracePeriod={gracePeriod}
        onGracePeriodChange={setGracePeriod}
        deprecateReason={deprecateReason}
        onDeprecateReasonChange={setDeprecateReason}
        onDeprecate={handleDeprecate}
        isPending={deprecateMutation.isPending}
      />

      <TenantRotateKeyDialog
        open={isRotateDialogOpen}
        onOpenChange={setIsRotateDialogOpen}
        targetKey={targetKey}
        gracePeriod={gracePeriod}
        onGracePeriodChange={setGracePeriod}
        onRotate={handleRotate}
        isPending={rotateMutation.isPending}
      />

      <RotatedKeyDialog
        open={isRotatedKeyDialogOpen}
        onOpenChange={setIsRotatedKeyDialogOpen}
        rotatedKey={rotatedKey}
      />

      <HistoryDialog
        open={isHistoryDialogOpen}
        onOpenChange={setIsHistoryDialogOpen}
        targetKey={targetKey}
        revocationHistory={revocationHistory}
      />
    </div>
  )
}
