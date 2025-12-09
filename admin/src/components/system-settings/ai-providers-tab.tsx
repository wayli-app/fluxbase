import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Bot, Plus, Trash2, Star } from 'lucide-react'
import { toast } from 'sonner'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import type { AIProvider } from '@/lib/api'

interface CreateProviderRequest {
  name: string
  display_name: string
  provider_type: 'openai' | 'azure' | 'ollama'
  config: Record<string, string>
}

export function AIProvidersTab() {
  const queryClient = useQueryClient()
  const [createDialogOpen, setCreateDialogOpen] = useState(false)
  const [deleteConfirm, setDeleteConfirm] = useState<AIProvider | null>(null)

  // Fetch providers
  const { data: providersData, isLoading } = useQuery<{ providers: AIProvider[] }>({
    queryKey: ['ai-providers'],
    queryFn: async () => {
      const response = await fetch('/api/v1/admin/ai/providers', {
        headers: {
          Authorization: `Bearer ${localStorage.getItem('access_token')}`,
        },
      })
      if (!response.ok) throw new Error('Failed to fetch providers')
      return response.json()
    },
  })

  const providers: AIProvider[] = providersData?.providers || []

  // Set default provider mutation
  const setDefaultMutation = useMutation({
    mutationFn: async (id: string) => {
      const response = await fetch(`/api/v1/admin/ai/providers/${id}/default`, {
        method: 'PUT',
        headers: {
          Authorization: `Bearer ${localStorage.getItem('access_token')}`,
        },
      })
      if (!response.ok) {
        const error = await response.json()
        throw new Error(error.error || 'Failed to set default provider')
      }
      return response.json()
    },
    onSuccess: () => {
      toast.success('Default provider updated')
      queryClient.invalidateQueries({ queryKey: ['ai-providers'] })
    },
    onError: (error: Error) => {
      toast.error(error.message)
    },
  })

  // Delete provider mutation
  const deleteMutation = useMutation({
    mutationFn: async (id: string) => {
      const response = await fetch(`/api/v1/admin/ai/providers/${id}`, {
        method: 'DELETE',
        headers: {
          Authorization: `Bearer ${localStorage.getItem('access_token')}`,
        },
      })
      if (!response.ok) {
        const error = await response.json()
        throw new Error(error.error || 'Failed to delete provider')
      }
      return response.json()
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

  // Create provider mutation
  const createMutation = useMutation({
    mutationFn: async (data: CreateProviderRequest) => {
      const response = await fetch('/api/v1/admin/ai/providers', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${localStorage.getItem('access_token')}`,
        },
        body: JSON.stringify(data),
      })
      if (!response.ok) {
        const error = await response.json()
        throw new Error(error.error || 'Failed to create provider')
      }
      return response.json()
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
                Manage AI providers for chatbot functionality. Providers configured via environment
                variables are read-only.
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
            <div className='text-center py-8'>
              <p className='text-muted-foreground'>Loading providers...</p>
            </div>
          ) : providers.length === 0 ? (
            <div className='text-center py-8'>
              <Bot className='h-12 w-12 mx-auto mb-4 text-muted-foreground' />
              <p className='text-lg font-medium mb-1'>No providers configured</p>
              <p className='text-sm text-muted-foreground mb-4'>
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
                    <TableCell className='text-sm text-muted-foreground'>
                      {provider.config.model || (provider.provider_type === 'openai' ? 'gpt-4-turbo' : '-')}
                    </TableCell>
                    <TableCell>
                      {provider.enabled ? (
                        <Badge variant='outline' className='border-green-500 text-green-500'>
                          Enabled
                        </Badge>
                      ) : (
                        <Badge variant='outline' className='border-gray-500 text-gray-500'>
                          Disabled
                        </Badge>
                      )}
                    </TableCell>
                    <TableCell className='text-right'>
                      <div className='flex items-center justify-end gap-2'>
                        {!provider.from_config && !provider.is_default && (
                          <Button
                            size='sm'
                            variant='ghost'
                            onClick={() => setDefaultMutation.mutate(provider.id)}
                            disabled={setDefaultMutation.isPending}
                          >
                            <Star className='h-4 w-4' />
                          </Button>
                        )}
                        {!provider.from_config && (
                          <Button
                            size='sm'
                            variant='ghost'
                            onClick={() => setDeleteConfirm(provider)}
                            className='text-destructive hover:text-destructive hover:bg-destructive/10'
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

      {/* Delete Confirmation Dialog */}
      <AlertDialog open={deleteConfirm !== null} onOpenChange={() => setDeleteConfirm(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Provider</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete <strong>{deleteConfirm?.display_name}</strong>? This
              action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => deleteConfirm && deleteMutation.mutate(deleteConfirm.id)}
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
  const [providerType, setProviderType] = useState<'openai' | 'azure' | 'ollama'>('openai')
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
        toast.error('API key, endpoint, and deployment name are required for Azure')
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
                onValueChange={(value) => setProviderType(value as 'openai' | 'azure' | 'ollama')}
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
                  <Label htmlFor='organizationId'>Organization ID (optional)</Label>
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
            <Button type='button' variant='outline' onClick={() => onOpenChange(false)}>
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
