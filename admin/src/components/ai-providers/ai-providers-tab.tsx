import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import type { AIProvider, UpdateAIProviderRequest } from '@fluxbase/sdk'
import { useFluxbaseClient } from '@fluxbase/sdk-react'
import { Bot, Plus, Trash2, Star, Pencil } from 'lucide-react'
import { toast } from 'sonner'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

interface CreateProviderRequest {
  name: string
  display_name: string
  provider_type: 'openai' | 'azure' | 'ollama'
  config: Record<string, string>
}

export function AIProvidersTab() {
  const client = useFluxbaseClient()
  const queryClient = useQueryClient()
  const [createDialogOpen, setCreateDialogOpen] = useState(false)
  const [editProvider, setEditProvider] = useState<AIProvider | null>(null)
  const [deleteConfirm, setDeleteConfirm] = useState<AIProvider | null>(null)

  // Fetch providers using SDK
  const { data: providers = [], isLoading } = useQuery<AIProvider[]>({
    queryKey: ['ai-providers'],
    queryFn: async () => {
      const { data, error } = await client.admin.ai.listProviders()
      if (error) throw error
      return data ?? []
    },
  })

  // Set default provider mutation using SDK
  const setDefaultMutation = useMutation({
    mutationFn: async (id: string) => {
      const { error } = await client.admin.ai.setDefaultProvider(id)
      if (error) throw error
    },
    onSuccess: () => {
      toast.success('Default provider updated')
      queryClient.invalidateQueries({ queryKey: ['ai-providers'] })
    },
    onError: (error: Error) => {
      toast.error(error.message)
    },
  })

  // Delete provider mutation using SDK
  const deleteMutation = useMutation({
    mutationFn: async (id: string) => {
      const { error } = await client.admin.ai.deleteProvider(id)
      if (error) throw error
    },
    onSuccess: () => {
      toast.success('Provider deleted')
      queryClient.invalidateQueries({ queryKey: ['ai-providers'] })
      setDeleteConfirm(null)
    },
    onError: (error: Error) => {
      toast.error(error.message)
    },
  })

  // Create provider mutation using SDK
  const createMutation = useMutation({
    mutationFn: async (data: CreateProviderRequest) => {
      const { error } = await client.admin.ai.createProvider(data)
      if (error) throw error
    },
    onSuccess: () => {
      toast.success('Provider created')
      queryClient.invalidateQueries({ queryKey: ['ai-providers'] })
      setCreateDialogOpen(false)
    },
    onError: (error: Error) => {
      toast.error(error.message)
    },
  })

  // Update provider mutation using SDK
  const updateMutation = useMutation({
    mutationFn: async ({
      id,
      data,
    }: {
      id: string
      data: UpdateAIProviderRequest
    }) => {
      const { error } = await client.admin.ai.updateProvider(id, data)
      if (error) throw error
    },
    onSuccess: () => {
      toast.success('Provider updated')
      queryClient.invalidateQueries({ queryKey: ['ai-providers'] })
      setEditProvider(null)
    },
    onError: (error: Error) => {
      toast.error(error.message)
    },
  })

  return (
    <div className='space-y-4'>
      <Card>
        <CardHeader>
          <div className='flex items-center justify-between'>
            <div>
              <CardTitle className='flex items-center gap-2'>
                <Bot className='h-5 w-5' />
                AI Providers
              </CardTitle>
              <CardDescription>
                Manage AI providers for chatbot functionality. Providers
                configured via environment variables are read-only.
              </CardDescription>
            </div>
            <Button onClick={() => setCreateDialogOpen(true)}>
              <Plus className='mr-2 h-4 w-4' />
              Add Provider
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className='py-8 text-center'>
              <p className='text-muted-foreground'>Loading providers...</p>
            </div>
          ) : providers.length === 0 ? (
            <div className='py-8 text-center'>
              <Bot className='text-muted-foreground mx-auto mb-4 h-12 w-12' />
              <p className='mb-1 text-lg font-medium'>
                No providers configured
              </p>
              <p className='text-muted-foreground mb-4 text-sm'>
                Add an AI provider to enable chatbot functionality
              </p>
              <Button onClick={() => setCreateDialogOpen(true)}>
                <Plus className='mr-2 h-4 w-4' />
                Add Provider
              </Button>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Type</TableHead>
                  <TableHead>Model</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className='text-right'>Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {providers.map((provider) => (
                  <TableRow key={provider.id}>
                    <TableCell className='font-medium'>
                      <div className='flex items-center gap-2'>
                        {provider.display_name}
                        {provider.is_default && (
                          <Badge variant='default' className='text-xs'>
                            <Star className='mr-1 h-3 w-3' />
                            Default
                          </Badge>
                        )}
                        {provider.from_config && (
                          <Badge variant='secondary' className='text-xs'>
                            Config
                          </Badge>
                        )}
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant='outline'>{provider.provider_type}</Badge>
                    </TableCell>
                    <TableCell className='text-muted-foreground text-sm'>
                      {provider.config.model ||
                        (provider.provider_type === 'openai'
                          ? 'gpt-4-turbo'
                          : '-')}
                    </TableCell>
                    <TableCell>
                      {provider.enabled ? (
                        <Badge
                          variant='outline'
                          className='border-green-500 text-green-500'
                        >
                          Enabled
                        </Badge>
                      ) : (
                        <Badge
                          variant='outline'
                          className='border-gray-500 text-gray-500'
                        >
                          Disabled
                        </Badge>
                      )}
                    </TableCell>
                    <TableCell className='text-right'>
                      <div className='flex items-center justify-end gap-2'>
                        {/* Default toggle - always visible for non-config providers */}
                        {!provider.from_config && (
                          <Button
                            size='sm'
                            variant='ghost'
                            onClick={() =>
                              !provider.is_default &&
                              setDefaultMutation.mutate(provider.id)
                            }
                            disabled={
                              setDefaultMutation.isPending ||
                              provider.is_default
                            }
                            title={
                              provider.is_default
                                ? 'Current default'
                                : 'Set as default'
                            }
                            className={
                              provider.is_default
                                ? 'text-yellow-500'
                                : 'text-muted-foreground hover:text-yellow-500'
                            }
                          >
                            <Star
                              className={`h-4 w-4 ${provider.is_default ? 'fill-current' : ''}`}
                            />
                          </Button>
                        )}
                        {/* Edit button */}
                        {!provider.from_config && (
                          <Button
                            size='sm'
                            variant='ghost'
                            onClick={() => setEditProvider(provider)}
                            title='Edit provider'
                          >
                            <Pencil className='h-4 w-4' />
                          </Button>
                        )}
                        {/* Delete button */}
                        {!provider.from_config && (
                          <Button
                            size='sm'
                            variant='ghost'
                            onClick={() => setDeleteConfirm(provider)}
                            className='text-destructive hover:text-destructive hover:bg-destructive/10'
                            title='Delete provider'
                          >
                            <Trash2 className='h-4 w-4' />
                          </Button>
                        )}
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* Create Provider Dialog */}
      <CreateProviderDialog
        open={createDialogOpen}
        onOpenChange={setCreateDialogOpen}
        onSubmit={(data) => createMutation.mutate(data)}
        isPending={createMutation.isPending}
      />

      {/* Edit Provider Dialog */}
      {editProvider && (
        <EditProviderDialog
          provider={editProvider}
          open={editProvider !== null}
          onOpenChange={(open) => !open && setEditProvider(null)}
          onSubmit={(data) =>
            updateMutation.mutate({ id: editProvider.id, data })
          }
          isPending={updateMutation.isPending}
        />
      )}

      {/* Delete Confirmation Dialog */}
      <AlertDialog
        open={deleteConfirm !== null}
        onOpenChange={() => setDeleteConfirm(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Provider</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete{' '}
              <strong>{deleteConfirm?.display_name}</strong>? This action cannot
              be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() =>
                deleteConfirm && deleteMutation.mutate(deleteConfirm.id)
              }
              className='bg-destructive text-destructive-foreground hover:bg-destructive/90'
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}

interface CreateProviderDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSubmit: (data: CreateProviderRequest) => void
  isPending: boolean
}

function CreateProviderDialog({
  open,
  onOpenChange,
  onSubmit,
  isPending,
}: CreateProviderDialogProps) {
  const [name, setName] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [providerType, setProviderType] = useState<
    'openai' | 'azure' | 'ollama'
  >('openai')
  const [apiKey, setApiKey] = useState('')
  const [endpoint, setEndpoint] = useState('')
  const [model, setModel] = useState('')
  const [organizationId, setOrganizationId] = useState('')
  const [deploymentName, setDeploymentName] = useState('')

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()

    if (!name || !displayName) {
      toast.error('Name and display name are required')
      return
    }

    const config: Record<string, string> = {}

    if (providerType === 'openai') {
      if (!apiKey) {
        toast.error('API key is required for OpenAI')
        return
      }
      config.api_key = apiKey
      if (organizationId) config.organization_id = organizationId
      if (endpoint) config.base_url = endpoint
    } else if (providerType === 'azure') {
      if (!apiKey || !endpoint || !deploymentName) {
        toast.error(
          'API key, endpoint, and deployment name are required for Azure'
        )
        return
      }
      config.api_key = apiKey
      config.endpoint = endpoint
      config.deployment_name = deploymentName
    } else if (providerType === 'ollama') {
      if (!model) {
        toast.error('Model is required for Ollama')
        return
      }
      if (endpoint) config.endpoint = endpoint
    }

    if (model) config.model = model

    onSubmit({
      name,
      display_name: displayName,
      provider_type: providerType,
      config,
    })
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='max-w-md'>
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Add AI Provider</DialogTitle>
            <DialogDescription>
              Configure a new AI provider for your chatbots
            </DialogDescription>
          </DialogHeader>

          <div className='space-y-4 py-4'>
            <div className='space-y-2'>
              <Label htmlFor='name'>Name (internal)</Label>
              <Input
                id='name'
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder='my-openai-provider'
                required
              />
            </div>

            <div className='space-y-2'>
              <Label htmlFor='displayName'>Display Name</Label>
              <Input
                id='displayName'
                value={displayName}
                onChange={(e) => setDisplayName(e.target.value)}
                placeholder='My OpenAI Provider'
                required
              />
            </div>

            <div className='space-y-2'>
              <Label htmlFor='providerType'>Provider Type</Label>
              <Select
                value={providerType}
                onValueChange={(value) =>
                  setProviderType(value as 'openai' | 'azure' | 'ollama')
                }
              >
                <SelectTrigger id='providerType'>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value='openai'>OpenAI</SelectItem>
                  <SelectItem value='azure'>Azure OpenAI</SelectItem>
                  <SelectItem value='ollama'>Ollama</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {providerType === 'openai' && (
              <>
                <div className='space-y-2'>
                  <Label htmlFor='apiKey'>API Key</Label>
                  <Input
                    id='apiKey'
                    type='password'
                    value={apiKey}
                    onChange={(e) => setApiKey(e.target.value)}
                    placeholder='sk-...'
                    required
                  />
                </div>
                <div className='space-y-2'>
                  <Label htmlFor='organizationId'>
                    Organization ID (optional)
                  </Label>
                  <Input
                    id='organizationId'
                    value={organizationId}
                    onChange={(e) => setOrganizationId(e.target.value)}
                    placeholder='org-...'
                  />
                </div>
                <div className='space-y-2'>
                  <Label htmlFor='endpoint'>Custom Base URL (optional)</Label>
                  <Input
                    id='endpoint'
                    value={endpoint}
                    onChange={(e) => setEndpoint(e.target.value)}
                    placeholder='https://api.openai.com/v1'
                  />
                </div>
              </>
            )}

            {providerType === 'azure' && (
              <>
                <div className='space-y-2'>
                  <Label htmlFor='apiKey'>API Key</Label>
                  <Input
                    id='apiKey'
                    type='password'
                    value={apiKey}
                    onChange={(e) => setApiKey(e.target.value)}
                    required
                  />
                </div>
                <div className='space-y-2'>
                  <Label htmlFor='endpoint'>Endpoint</Label>
                  <Input
                    id='endpoint'
                    value={endpoint}
                    onChange={(e) => setEndpoint(e.target.value)}
                    placeholder='https://your-resource.openai.azure.com'
                    required
                  />
                </div>
                <div className='space-y-2'>
                  <Label htmlFor='deploymentName'>Deployment Name</Label>
                  <Input
                    id='deploymentName'
                    value={deploymentName}
                    onChange={(e) => setDeploymentName(e.target.value)}
                    placeholder='gpt-4'
                    required
                  />
                </div>
              </>
            )}

            {providerType === 'ollama' && (
              <>
                <div className='space-y-2'>
                  <Label htmlFor='endpoint'>Endpoint (optional)</Label>
                  <Input
                    id='endpoint'
                    value={endpoint}
                    onChange={(e) => setEndpoint(e.target.value)}
                    placeholder='http://localhost:11434'
                  />
                </div>
                <div className='space-y-2'>
                  <Label htmlFor='model'>Model</Label>
                  <Input
                    id='model'
                    value={model}
                    onChange={(e) => setModel(e.target.value)}
                    placeholder='llama2'
                    required
                  />
                </div>
              </>
            )}
          </div>

          <DialogFooter>
            <Button
              type='button'
              variant='outline'
              onClick={() => onOpenChange(false)}
            >
              Cancel
            </Button>
            <Button type='submit' disabled={isPending}>
              {isPending ? 'Creating...' : 'Create Provider'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

interface EditProviderDialogProps {
  provider: AIProvider
  open: boolean
  onOpenChange: (open: boolean) => void
  onSubmit: (data: UpdateAIProviderRequest) => void
  isPending: boolean
}

function EditProviderDialog({
  provider,
  open,
  onOpenChange,
  onSubmit,
  isPending,
}: EditProviderDialogProps) {
  const [displayName, setDisplayName] = useState(provider.display_name)
  const [apiKey, setApiKey] = useState(provider.config.api_key || '')
  const [endpoint, setEndpoint] = useState(
    provider.config.endpoint || provider.config.base_url || ''
  )
  const [model, setModel] = useState(provider.config.model || '')
  const [organizationId, setOrganizationId] = useState(
    provider.config.organization_id || ''
  )
  const [deploymentName, setDeploymentName] = useState(
    provider.config.deployment_name || ''
  )
  const [enabled, setEnabled] = useState(provider.enabled)

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()

    const config: Record<string, string> = {}

    if (provider.provider_type === 'openai') {
      if (apiKey) config.api_key = apiKey
      if (organizationId) config.organization_id = organizationId
      if (endpoint) config.base_url = endpoint
    } else if (provider.provider_type === 'azure') {
      if (apiKey) config.api_key = apiKey
      if (endpoint) config.endpoint = endpoint
      if (deploymentName) config.deployment_name = deploymentName
    } else if (provider.provider_type === 'ollama') {
      if (endpoint) config.endpoint = endpoint
    }

    if (model) config.model = model

    onSubmit({
      display_name: displayName,
      config,
      enabled,
    })
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='max-w-md'>
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Edit AI Provider</DialogTitle>
            <DialogDescription>
              Update the configuration for {provider.display_name}
            </DialogDescription>
          </DialogHeader>

          <div className='space-y-4 py-4'>
            <div className='space-y-2'>
              <Label htmlFor='edit-displayName'>Display Name</Label>
              <Input
                id='edit-displayName'
                value={displayName}
                onChange={(e) => setDisplayName(e.target.value)}
                placeholder='My OpenAI Provider'
                required
              />
            </div>

            <div className='space-y-2'>
              <Label htmlFor='edit-enabled'>Status</Label>
              <Select
                value={enabled ? 'enabled' : 'disabled'}
                onValueChange={(value) => setEnabled(value === 'enabled')}
              >
                <SelectTrigger id='edit-enabled'>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value='enabled'>Enabled</SelectItem>
                  <SelectItem value='disabled'>Disabled</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {provider.provider_type === 'openai' && (
              <>
                <div className='space-y-2'>
                  <Label htmlFor='edit-apiKey'>API Key</Label>
                  <Input
                    id='edit-apiKey'
                    type='password'
                    value={apiKey}
                    onChange={(e) => setApiKey(e.target.value)}
                    placeholder='sk-... (leave masked to keep existing)'
                  />
                  <p className='text-muted-foreground text-xs'>
                    Leave as masked value to keep existing key
                  </p>
                </div>
                <div className='space-y-2'>
                  <Label htmlFor='edit-organizationId'>
                    Organization ID (optional)
                  </Label>
                  <Input
                    id='edit-organizationId'
                    value={organizationId}
                    onChange={(e) => setOrganizationId(e.target.value)}
                    placeholder='org-...'
                  />
                </div>
                <div className='space-y-2'>
                  <Label htmlFor='edit-endpoint'>
                    Custom Base URL (optional)
                  </Label>
                  <Input
                    id='edit-endpoint'
                    value={endpoint}
                    onChange={(e) => setEndpoint(e.target.value)}
                    placeholder='https://api.openai.com/v1'
                  />
                </div>
                <div className='space-y-2'>
                  <Label htmlFor='edit-model'>Model (optional)</Label>
                  <Input
                    id='edit-model'
                    value={model}
                    onChange={(e) => setModel(e.target.value)}
                    placeholder='gpt-4-turbo'
                  />
                </div>
              </>
            )}

            {provider.provider_type === 'azure' && (
              <>
                <div className='space-y-2'>
                  <Label htmlFor='edit-apiKey'>API Key</Label>
                  <Input
                    id='edit-apiKey'
                    type='password'
                    value={apiKey}
                    onChange={(e) => setApiKey(e.target.value)}
                    placeholder='Leave masked to keep existing'
                  />
                  <p className='text-muted-foreground text-xs'>
                    Leave as masked value to keep existing key
                  </p>
                </div>
                <div className='space-y-2'>
                  <Label htmlFor='edit-endpoint'>Endpoint</Label>
                  <Input
                    id='edit-endpoint'
                    value={endpoint}
                    onChange={(e) => setEndpoint(e.target.value)}
                    placeholder='https://your-resource.openai.azure.com'
                  />
                </div>
                <div className='space-y-2'>
                  <Label htmlFor='edit-deploymentName'>Deployment Name</Label>
                  <Input
                    id='edit-deploymentName'
                    value={deploymentName}
                    onChange={(e) => setDeploymentName(e.target.value)}
                    placeholder='gpt-4'
                  />
                </div>
                <div className='space-y-2'>
                  <Label htmlFor='edit-model'>Model (optional)</Label>
                  <Input
                    id='edit-model'
                    value={model}
                    onChange={(e) => setModel(e.target.value)}
                    placeholder='gpt-4'
                  />
                </div>
              </>
            )}

            {provider.provider_type === 'ollama' && (
              <>
                <div className='space-y-2'>
                  <Label htmlFor='edit-endpoint'>Endpoint (optional)</Label>
                  <Input
                    id='edit-endpoint'
                    value={endpoint}
                    onChange={(e) => setEndpoint(e.target.value)}
                    placeholder='http://localhost:11434'
                  />
                </div>
                <div className='space-y-2'>
                  <Label htmlFor='edit-model'>Model</Label>
                  <Input
                    id='edit-model'
                    value={model}
                    onChange={(e) => setModel(e.target.value)}
                    placeholder='llama2'
                  />
                </div>
              </>
            )}
          </div>

          <DialogFooter>
            <Button
              type='button'
              variant='outline'
              onClick={() => onOpenChange(false)}
            >
              Cancel
            </Button>
            <Button type='submit' disabled={isPending}>
              {isPending ? 'Saving...' : 'Save Changes'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
