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
  ShieldAlert,
  Clock,
  RefreshCw,
  History,
} from 'lucide-react'
import { toast } from 'sonner'
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

// Grouped scopes for better UI organization
const SCOPE_GROUPS = [
  {
    name: 'Tables',
    description: 'Database table access',
    scopes: [
      { id: 'read:tables', label: 'Read', description: 'Query database tables' },
      { id: 'write:tables', label: 'Write', description: 'Insert, update, delete records' },
    ],
  },
  {
    name: 'Storage',
    description: 'File storage access',
    scopes: [
      { id: 'read:storage', label: 'Read', description: 'Download files' },
      { id: 'write:storage', label: 'Write', description: 'Upload and delete files' },
    ],
  },
  {
    name: 'Functions',
    description: 'Edge Functions',
    scopes: [
      { id: 'read:functions', label: 'Read', description: 'View functions' },
      { id: 'execute:functions', label: 'Execute', description: 'Invoke functions' },
    ],
  },
  {
    name: 'Auth',
    description: 'Authentication',
    scopes: [
      { id: 'read:auth', label: 'Read', description: 'View user profile' },
      { id: 'write:auth', label: 'Write', description: 'Update profile, manage 2FA' },
    ],
  },
  {
    name: 'Client Keys',
    description: 'API key management',
    scopes: [
      { id: 'read:clientkeys', label: 'Read', description: 'List client keys' },
      { id: 'write:clientkeys', label: 'Write', description: 'Create, update, revoke' },
    ],
  },
  {
    name: 'Webhooks',
    description: 'Webhook management',
    scopes: [
      { id: 'read:webhooks', label: 'Read', description: 'List webhooks' },
      { id: 'write:webhooks', label: 'Write', description: 'Create, update, delete' },
    ],
  },
  {
    name: 'Monitoring',
    description: 'System monitoring',
    scopes: [
      { id: 'read:monitoring', label: 'Read', description: 'View metrics, health, logs' },
    ],
  },
  {
    name: 'Realtime',
    description: 'WebSocket channels',
    scopes: [
      { id: 'realtime:connect', label: 'Connect', description: 'Connect to channels' },
      { id: 'realtime:broadcast', label: 'Broadcast', description: 'Send messages' },
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
      { id: 'write:secrets', label: 'Write', description: 'Create, update, delete' },
    ],
  },
  {
    name: 'Migrations',
    description: 'Database migrations',
    scopes: [
      { id: 'migrations:read', label: 'Read', description: 'View migration status' },
      { id: 'migrations:execute', label: 'Execute', description: 'Apply migrations' },
    ],
  },
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
  const [editScopes, setEditScopes] = useState<string[]>([])
  const [editRateLimitPerMinute, setEditRateLimitPerMinute] = useState<number | undefined>(undefined)
  const [editRateLimitPerHour, setEditRateLimitPerHour] = useState<number | undefined>(undefined)

  // Revocation state
  const [showRevokeDialog, setShowRevokeDialog] = useState(false)
  const [showDeprecateDialog, setShowDeprecateDialog] = useState(false)
  const [showRotateDialog, setShowRotateDialog] = useState(false)
  const [showRotatedKeyDialog, setShowRotatedKeyDialog] = useState(false)
  const [showHistoryDialog, setShowHistoryDialog] = useState(false)
  const [targetKey, setTargetKey] = useState<ServiceKey | null>(null)
  const [revokeReason, setRevokeReason] = useState('')
  const [deprecateReason, setDeprecateReason] = useState('')
  const [gracePeriod, setGracePeriod] = useState('24h')
  const [rotatedKey, setRotatedKey] = useState<RotateServiceKeyResponse | null>(null)
  const [revocationHistory, setRevocationHistory] = useState<ServiceKeyRevocation[]>([])

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

  // Revoke service key
  const revokeMutation = useMutation({
    mutationFn: ({ id, request }: { id: string; request: RevokeServiceKeyRequest }) =>
      serviceKeysApi.revoke(id, request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['service-keys'] })
      setShowRevokeDialog(false)
      setTargetKey(null)
      setRevokeReason('')
      toast.success('Service key revoked')
    },
    onError: () => {
      toast.error('Failed to revoke service key')
    },
  })

  // Deprecate service key
  const deprecateMutation = useMutation({
    mutationFn: ({ id, request }: { id: string; request: DeprecateServiceKeyRequest }) =>
      serviceKeysApi.deprecate(id, request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['service-keys'] })
      setShowDeprecateDialog(false)
      setTargetKey(null)
      setDeprecateReason('')
      setGracePeriod('24h')
      toast.success('Service key deprecated')
    },
    onError: () => {
      toast.error('Failed to deprecate service key')
    },
  })

  // Rotate service key
  const rotateMutation = useMutation({
    mutationFn: ({ id, request }: { id: string; request: RotateServiceKeyRequest }) =>
      serviceKeysApi.rotate(id, request),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['service-keys'] })
      setRotatedKey(data)
      setShowRotateDialog(false)
      setShowRotatedKeyDialog(true)
      setTargetKey(null)
      setGracePeriod('24h')
    },
    onError: () => {
      toast.error('Failed to rotate service key')
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
      scopes: editScopes.length > 0 ? editScopes : undefined,
      rate_limit_per_minute: editRateLimitPerMinute,
      rate_limit_per_hour: editRateLimitPerHour,
    }

    updateMutation.mutate({ id: editingKey.id, request })
  }

  const openEditDialog = (key: ServiceKey) => {
    setEditingKey(key)
    setEditName(key.name)
    setEditDescription(key.description || '')
    setEditScopes(key.scopes || ['*'])
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
    // Check if revoked (highest priority)
    if (key.revoked_at)
      return { label: 'Revoked', variant: 'destructive' as const }
    // Check if deprecated (within grace period)
    if (key.deprecated_at) {
      if (key.grace_period_ends_at && new Date(key.grace_period_ends_at) > new Date()) {
        return { label: 'Deprecated', variant: 'outline' as const }
      }
      return { label: 'Expired', variant: 'destructive' as const }
    }
    if (!key.enabled)
      return { label: 'Disabled', variant: 'secondary' as const }
    if (isExpired(key.expires_at))
      return { label: 'Expired', variant: 'destructive' as const }
    return { label: 'Active', variant: 'default' as const }
  }

  // Check if key can be modified (not revoked)
  const canModify = (key: ServiceKey) => !key.revoked_at

  // Open revoke dialog
  const openRevokeDialog = (key: ServiceKey) => {
    setTargetKey(key)
    setRevokeReason('')
    setShowRevokeDialog(true)
  }

  // Open deprecate dialog
  const openDeprecateDialog = (key: ServiceKey) => {
    setTargetKey(key)
    setDeprecateReason('')
    setGracePeriod('24h')
    setShowDeprecateDialog(true)
  }

  // Open rotate dialog
  const openRotateDialog = (key: ServiceKey) => {
    setTargetKey(key)
    setGracePeriod('24h')
    setShowRotateDialog(true)
  }

  // Open history dialog
  const openHistoryDialog = async (key: ServiceKey) => {
    setTargetKey(key)
    try {
      const history = await serviceKeysApi.revocations(key.id)
      setRevocationHistory(history)
      setShowHistoryDialog(true)
    } catch {
      toast.error('Failed to load revocation history')
    }
  }

  // Handle revoke
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

  // Handle deprecate
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

  // Handle rotate
  const handleRotate = () => {
    if (!targetKey) return
    rotateMutation.mutate({
      id: targetKey.id,
      request: { grace_period: gracePeriod },
    })
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
                          {/* History button - always available */}
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
                          {/* Edit - only if not revoked */}
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
                              <TooltipContent>Edit service key</TooltipContent>
                            </Tooltip>
                          )}
                          {/* Rotate - only if active */}
                          {canModify(key) && key.enabled && !key.deprecated_at && (
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
                          {/* Deprecate - only if active and not already deprecated */}
                          {canModify(key) && key.enabled && !key.deprecated_at && (
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
                              <TooltipContent>Deprecate with grace period</TooltipContent>
                            </Tooltip>
                          )}
                          {/* Enable/Disable - only if not revoked */}
                          {canModify(key) && !key.deprecated_at && (
                            key.enabled ? (
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
                            )
                          )}
                          {/* Revoke - only if not already revoked */}
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
                              <TooltipContent>Revoke (emergency)</TooltipContent>
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
        <DialogContent className='max-h-[90vh] max-w-3xl overflow-y-auto'>
          <DialogHeader>
            <DialogTitle>Create Service Key</DialogTitle>
            <DialogDescription>
              Generate a new service key for server-to-server API access. The key will be
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
                  placeholder='Migrations Key'
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                />
              </div>
              <div className='grid gap-2'>
                <Label htmlFor='description'>Description</Label>
                <Input
                  id='description'
                  placeholder='Used by CI/CD pipeline'
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
              />
            </div>
            </div>
            <div className='grid gap-2'>
              <div className='flex items-center justify-between'>
                <Label>Scopes</Label>
                <div className='flex items-center space-x-2'>
                  <input
                    type='checkbox'
                    id='wildcard-scope'
                    checked={selectedScopes.includes('*')}
                    onChange={(e) => {
                      if (e.target.checked) {
                        setSelectedScopes(['*'])
                      } else {
                        setSelectedScopes([])
                      }
                    }}
                    className='h-4 w-4 rounded border-gray-300'
                  />
                  <label htmlFor='wildcard-scope' className='text-sm font-medium'>
                    All Scopes
                  </label>
                </div>
              </div>
              <div className='grid grid-cols-2 gap-3 rounded-md border p-4'>
                {SCOPE_GROUPS.map((group) => (
                  <div key={group.name} className='space-y-1'>
                    <div className='text-sm font-medium'>{group.name}</div>
                    <div className='text-muted-foreground text-xs'>{group.description}</div>
                    <div className='flex flex-wrap gap-3 pt-1'>
                      {group.scopes.map((scope) => (
                        <div key={scope.id} className='flex items-center space-x-1.5'>
                          <input
                            type='checkbox'
                            id={`create-${scope.id}`}
                            checked={selectedScopes.includes(scope.id) || selectedScopes.includes('*')}
                            disabled={selectedScopes.includes('*')}
                            onChange={(e) => {
                              if (e.target.checked) {
                                setSelectedScopes([...selectedScopes, scope.id])
                              } else {
                                setSelectedScopes(selectedScopes.filter((s) => s !== scope.id))
                              }
                            }}
                            className='h-3.5 w-3.5 rounded border-gray-300'
                          />
                          <label htmlFor={`create-${scope.id}`} className='text-xs' title={scope.description}>
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
        <DialogContent className='max-h-[90vh] max-w-3xl overflow-y-auto'>
          <DialogHeader>
            <DialogTitle>Edit Service Key</DialogTitle>
            <DialogDescription>
              Update service key properties. The key value cannot be changed.
            </DialogDescription>
          </DialogHeader>
          <div className='grid gap-4 py-4'>
            <div className='grid grid-cols-2 gap-4'>
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
            </div>
            <div className='grid gap-2'>
              <div className='flex items-center justify-between'>
                <Label>Scopes</Label>
                <div className='flex items-center space-x-2'>
                  <input
                    type='checkbox'
                    id='edit-wildcard-scope'
                    checked={editScopes.includes('*')}
                    onChange={(e) => {
                      if (e.target.checked) {
                        setEditScopes(['*'])
                      } else {
                        setEditScopes([])
                      }
                    }}
                    className='h-4 w-4 rounded border-gray-300'
                  />
                  <label htmlFor='edit-wildcard-scope' className='text-sm font-medium'>
                    All Scopes
                  </label>
                </div>
              </div>
              <div className='grid grid-cols-2 gap-3 rounded-md border p-4'>
                {SCOPE_GROUPS.map((group) => (
                  <div key={group.name} className='space-y-1'>
                    <div className='text-sm font-medium'>{group.name}</div>
                    <div className='text-muted-foreground text-xs'>{group.description}</div>
                    <div className='flex flex-wrap gap-3 pt-1'>
                      {group.scopes.map((scope) => (
                        <div key={scope.id} className='flex items-center space-x-1.5'>
                          <input
                            type='checkbox'
                            id={`edit-${scope.id}`}
                            checked={editScopes.includes(scope.id) || editScopes.includes('*')}
                            disabled={editScopes.includes('*')}
                            onChange={(e) => {
                              if (e.target.checked) {
                                setEditScopes([...editScopes, scope.id])
                              } else {
                                setEditScopes(editScopes.filter((s) => s !== scope.id))
                              }
                            }}
                            className='h-3.5 w-3.5 rounded border-gray-300'
                          />
                          <label htmlFor={`edit-${scope.id}`} className='text-xs' title={scope.description}>
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

      {/* Revoke Service Key Dialog */}
      <Dialog open={showRevokeDialog} onOpenChange={setShowRevokeDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle className='flex items-center gap-2 text-destructive'>
              <ShieldAlert className='h-5 w-5' />
              Emergency Revoke
            </DialogTitle>
            <DialogDescription>
              This action is irreversible. The key "{targetKey?.name}" will be immediately
              disabled and marked as revoked. Any applications using this key will lose
              access instantly.
            </DialogDescription>
          </DialogHeader>
          <div className='space-y-4 py-4'>
            <div className='rounded-md bg-red-50 p-4 dark:bg-red-950'>
              <div className='flex'>
                <AlertCircle className='h-5 w-5 text-red-600 dark:text-red-400' />
                <div className='ml-3'>
                  <h3 className='text-sm font-medium text-red-800 dark:text-red-200'>
                    Warning: This cannot be undone
                  </h3>
                  <div className='mt-2 text-sm text-red-700 dark:text-red-300'>
                    <p>
                      Use this only for security incidents. For planned key rotation,
                      use the Rotate or Deprecate options instead.
                    </p>
                  </div>
                </div>
              </div>
            </div>
            <div className='grid gap-2'>
              <Label htmlFor='revokeReason'>
                Reason for revocation <span className='text-destructive'>*</span>
              </Label>
              <Input
                id='revokeReason'
                placeholder='e.g., Key compromised, employee departure'
                value={revokeReason}
                onChange={(e) => setRevokeReason(e.target.value)}
              />
              <p className='text-muted-foreground text-xs'>
                This will be recorded in the audit log.
              </p>
            </div>
          </div>
          <DialogFooter>
            <Button
              variant='outline'
              onClick={() => setShowRevokeDialog(false)}
            >
              Cancel
            </Button>
            <Button
              variant='destructive'
              onClick={handleRevoke}
              disabled={revokeMutation.isPending || !revokeReason.trim()}
            >
              {revokeMutation.isPending ? 'Revoking...' : 'Revoke Key'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Deprecate Service Key Dialog */}
      <Dialog open={showDeprecateDialog} onOpenChange={setShowDeprecateDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle className='flex items-center gap-2'>
              <Clock className='h-5 w-5' />
              Deprecate Service Key
            </DialogTitle>
            <DialogDescription>
              Mark "{targetKey?.name}" as deprecated with a grace period. The key will
              continue working during the grace period, allowing time for migration.
            </DialogDescription>
          </DialogHeader>
          <div className='space-y-4 py-4'>
            <div className='grid gap-2'>
              <Label htmlFor='gracePeriodDeprecate'>Grace Period</Label>
              <select
                id='gracePeriodDeprecate'
                value={gracePeriod}
                onChange={(e) => setGracePeriod(e.target.value)}
                className='flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background'
              >
                <option value='1h'>1 hour</option>
                <option value='6h'>6 hours</option>
                <option value='12h'>12 hours</option>
                <option value='24h'>24 hours</option>
                <option value='48h'>48 hours</option>
                <option value='7d'>7 days</option>
                <option value='14d'>14 days</option>
                <option value='30d'>30 days</option>
              </select>
              <p className='text-muted-foreground text-xs'>
                The key will stop working after this period.
              </p>
            </div>
            <div className='grid gap-2'>
              <Label htmlFor='deprecateReason'>Reason (optional)</Label>
              <Input
                id='deprecateReason'
                placeholder='e.g., Scheduled rotation, security policy'
                value={deprecateReason}
                onChange={(e) => setDeprecateReason(e.target.value)}
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              variant='outline'
              onClick={() => setShowDeprecateDialog(false)}
            >
              Cancel
            </Button>
            <Button
              onClick={handleDeprecate}
              disabled={deprecateMutation.isPending}
            >
              {deprecateMutation.isPending ? 'Deprecating...' : 'Deprecate Key'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Rotate Service Key Dialog */}
      <Dialog open={showRotateDialog} onOpenChange={setShowRotateDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle className='flex items-center gap-2'>
              <RefreshCw className='h-5 w-5' />
              Rotate Service Key
            </DialogTitle>
            <DialogDescription>
              Create a new key to replace "{targetKey?.name}". The old key will be
              deprecated with a grace period for migration.
            </DialogDescription>
          </DialogHeader>
          <div className='space-y-4 py-4'>
            <div className='grid gap-2'>
              <Label htmlFor='gracePeriodRotate'>Grace Period for Old Key</Label>
              <select
                id='gracePeriodRotate'
                value={gracePeriod}
                onChange={(e) => setGracePeriod(e.target.value)}
                className='flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background'
              >
                <option value='1h'>1 hour</option>
                <option value='6h'>6 hours</option>
                <option value='12h'>12 hours</option>
                <option value='24h'>24 hours</option>
                <option value='48h'>48 hours</option>
                <option value='7d'>7 days</option>
                <option value='14d'>14 days</option>
                <option value='30d'>30 days</option>
              </select>
              <p className='text-muted-foreground text-xs'>
                The old key will continue working for this period.
              </p>
            </div>
            <div className='rounded-md bg-blue-50 p-4 dark:bg-blue-950'>
              <div className='text-sm text-blue-700 dark:text-blue-300'>
                <p className='font-medium'>What happens on rotation:</p>
                <ul className='mt-2 list-disc pl-5 space-y-1'>
                  <li>A new key is created with the same configuration</li>
                  <li>The old key is marked as deprecated</li>
                  <li>The old key continues working during the grace period</li>
                  <li>After the grace period, the old key stops working</li>
                </ul>
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button
              variant='outline'
              onClick={() => setShowRotateDialog(false)}
            >
              Cancel
            </Button>
            <Button
              onClick={handleRotate}
              disabled={rotateMutation.isPending}
            >
              {rotateMutation.isPending ? 'Rotating...' : 'Rotate Key'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Show Rotated Key Dialog */}
      <Dialog open={showRotatedKeyDialog} onOpenChange={setShowRotatedKeyDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Key Rotated Successfully</DialogTitle>
            <DialogDescription>
              Save the new key now. You won't be able to see it again!
            </DialogDescription>
          </DialogHeader>
          <div className='space-y-4 py-4'>
            <div className='rounded-md bg-yellow-50 p-4 dark:bg-yellow-950'>
              <div className='flex'>
                <AlertCircle className='h-5 w-5 text-yellow-600 dark:text-yellow-400' />
                <div className='ml-3'>
                  <h3 className='text-sm font-medium text-yellow-800 dark:text-yellow-200'>
                    Important: Copy the new key now
                  </h3>
                  <div className='mt-2 text-sm text-yellow-700 dark:text-yellow-300'>
                    <p>
                      This is the only time you'll see the new service key. The old key
                      will continue working during the grace period.
                    </p>
                  </div>
                </div>
              </div>
            </div>
            <div className='grid gap-2'>
              <Label>New Service Key</Label>
              <div className='flex gap-2'>
                <Input
                  value={rotatedKey?.key || ''}
                  readOnly
                  className='font-mono text-xs'
                />
                <Button
                  variant='outline'
                  size='icon'
                  onClick={() => copyToClipboard(rotatedKey?.key || '')}
                >
                  <Copy className='h-4 w-4' />
                </Button>
              </div>
            </div>
            <div className='grid gap-2'>
              <Label>Name</Label>
              <Input value={rotatedKey?.name || ''} readOnly />
            </div>
            {rotatedKey?.grace_period_ends_at && (
              <div className='grid gap-2'>
                <Label>Old Key Expires</Label>
                <Input
                  value={new Date(rotatedKey.grace_period_ends_at).toLocaleString()}
                  readOnly
                />
              </div>
            )}
          </div>
          <DialogFooter>
            <Button onClick={() => setShowRotatedKeyDialog(false)}>
              I've Saved the New Key
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Revocation History Dialog */}
      <Dialog open={showHistoryDialog} onOpenChange={setShowHistoryDialog}>
        <DialogContent className='max-w-2xl'>
          <DialogHeader>
            <DialogTitle className='flex items-center gap-2'>
              <History className='h-5 w-5' />
              Key History: {targetKey?.name}
            </DialogTitle>
            <DialogDescription>
              View the revocation and rotation history for this service key.
            </DialogDescription>
          </DialogHeader>
          <div className='py-4'>
            {revocationHistory.length === 0 ? (
              <div className='text-center py-8 text-muted-foreground'>
                <History className='h-12 w-12 mx-auto mb-4 opacity-50' />
                <p>No revocation history for this key.</p>
              </div>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Type</TableHead>
                    <TableHead>Reason</TableHead>
                    <TableHead>By</TableHead>
                    <TableHead>Date</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {revocationHistory.map((rev) => (
                    <TableRow key={rev.id}>
                      <TableCell>
                        <Badge
                          variant={
                            rev.revocation_type === 'emergency'
                              ? 'destructive'
                              : rev.revocation_type === 'rotation'
                              ? 'default'
                              : 'secondary'
                          }
                        >
                          {rev.revocation_type}
                        </Badge>
                      </TableCell>
                      <TableCell className='max-w-[200px] truncate'>
                        {rev.reason || '-'}
                      </TableCell>
                      <TableCell className='text-sm text-muted-foreground'>
                        {rev.revoked_by || '-'}
                      </TableCell>
                      <TableCell className='text-sm text-muted-foreground'>
                        {formatDistanceToNow(new Date(rev.created_at), {
                          addSuffix: true,
                        })}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </div>
          <DialogFooter>
            <Button onClick={() => setShowHistoryDialog(false)}>
              Close
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
