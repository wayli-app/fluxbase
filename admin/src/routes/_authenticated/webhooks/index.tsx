import { createFileRoute } from '@tanstack/react-router'
import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { formatDistanceToNow } from 'date-fns'
import {
  Webhook,
  Plus,
  Trash2,
  Send,
  Check,
  X,
  Search,
  Clock,
} from 'lucide-react'
import { toast } from 'sonner'

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
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Checkbox } from '@/components/ui/checkbox'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Switch } from '@/components/ui/switch'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import { ProfileDropdown } from '@/components/profile-dropdown'
import { Search as SearchComponent } from '@/components/search'
import { ThemeSwitch } from '@/components/theme-switch'
import { ConfigDrawer } from '@/components/config-drawer'

export const Route = createFileRoute('/_authenticated/webhooks/')({
  component: WebhooksPage,
})

interface EventConfig {
  table: string
  operations: string[]
}

interface WebhookType {
  id: string
  name: string
  description?: string
  url: string
  secret?: string
  enabled: boolean
  events: EventConfig[]
  max_retries: number
  retry_backoff_seconds: number
  timeout_seconds: number
  headers: Record<string, string>
  created_at: string
  updated_at: string
}

interface WebhookDelivery {
  id: string
  webhook_id: string
  event_type: string
  table_name: string
  record_id?: string
  payload: any
  attempt_number: number
  status: string
  http_status_code?: number
  response_body?: string
  error_message?: string
  created_at: string
  delivered_at?: string
}

const OPERATIONS = ['INSERT', 'UPDATE', 'DELETE']

function WebhooksPage() {
  const queryClient = useQueryClient()
  const [showCreateDialog, setShowCreateDialog] = useState(false)
  const [selectedWebhook, setSelectedWebhook] = useState<WebhookType | null>(null)
  const [searchQuery, setSearchQuery] = useState('')

  // Form state
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [url, setUrl] = useState('')
  const [secret, setSecret] = useState('')
  const [enabled, setEnabled] = useState(true)
  const [tableName, setTableName] = useState('')
  const [selectedOps, setSelectedOps] = useState<string[]>(['INSERT', 'UPDATE', 'DELETE'])
  const [events, setEvents] = useState<EventConfig[]>([])
  const [maxRetries, setMaxRetries] = useState(3)
  const [timeoutSeconds, setTimeoutSeconds] = useState(30)

  // Fetch webhooks
  const { data: webhooks, isLoading } = useQuery<WebhookType[]>({
    queryKey: ['webhooks'],
    queryFn: async () => {
      const response = await fetch('/api/v1/webhooks', {
        headers: {
          Authorization: `Bearer ${localStorage.getItem('access_token')}`,
        },
      })
      if (!response.ok) throw new Error('Failed to fetch webhooks')
      return response.json()
    },
  })

  // Fetch deliveries for selected webhook
  const { data: deliveries } = useQuery<WebhookDelivery[]>({
    queryKey: ['webhook-deliveries', selectedWebhook?.id],
    queryFn: async () => {
      if (!selectedWebhook) return []
      const response = await fetch(`/api/v1/webhooks/${selectedWebhook.id}/deliveries?limit=50`, {
        headers: {
          Authorization: `Bearer ${localStorage.getItem('access_token')}`,
        },
      })
      if (!response.ok) throw new Error('Failed to fetch deliveries')
      return response.json()
    },
    enabled: !!selectedWebhook,
  })

  // Fetch available tables
  const { data: tables } = useQuery<Array<{ schema: string; name: string }>>({
    queryKey: ['tables'],
    queryFn: async () => {
      const response = await fetch('/api/v1/admin/tables', {
        headers: {
          Authorization: `Bearer ${localStorage.getItem('access_token')}`,
        },
      })
      if (!response.ok) throw new Error('Failed to fetch tables')
      return response.json()
    },
  })

  // Create webhook
  const createMutation = useMutation({
    mutationFn: async (webhook: Partial<WebhookType>) => {
      const response = await fetch('/api/v1/webhooks', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${localStorage.getItem('access_token')}`,
        },
        body: JSON.stringify(webhook),
      })
      if (!response.ok) throw new Error('Failed to create webhook')
      return response.json()
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['webhooks'] })
      setShowCreateDialog(false)
      resetForm()
      toast.success('Webhook created successfully')
    },
    onError: () => {
      toast.error('Failed to create webhook')
    },
  })

  // Delete webhook
  const deleteMutation = useMutation({
    mutationFn: async (id: string) => {
      const response = await fetch(`/api/v1/webhooks/${id}`, {
        method: 'DELETE',
        headers: {
          Authorization: `Bearer ${localStorage.getItem('access_token')}`,
        },
      })
      if (!response.ok) throw new Error('Failed to delete webhook')
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['webhooks'] })
      toast.success('Webhook deleted successfully')
    },
    onError: () => {
      toast.error('Failed to delete webhook')
    },
  })

  // Test webhook
  const testMutation = useMutation({
    mutationFn: async (id: string) => {
      const response = await fetch(`/api/v1/webhooks/${id}/test`, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${localStorage.getItem('access_token')}`,
        },
      })
      if (!response.ok) throw new Error('Failed to test webhook')
      return response.json()
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['webhook-deliveries'] })
      toast.success('Test webhook sent successfully')
    },
    onError: () => {
      toast.error('Failed to send test webhook')
    },
  })

  // Toggle webhook enabled
  const toggleMutation = useMutation({
    mutationFn: async ({ id, enabled }: { id: string; enabled: boolean }) => {
      const webhook = webhooks?.find((w) => w.id === id)
      if (!webhook) throw new Error('Webhook not found')

      const response = await fetch(`/api/v1/webhooks/${id}`, {
        method: 'PATCH',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${localStorage.getItem('access_token')}`,
        },
        body: JSON.stringify({ ...webhook, enabled }),
      })
      if (!response.ok) throw new Error('Failed to toggle webhook')
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['webhooks'] })
      toast.success('Webhook updated successfully')
    },
    onError: () => {
      toast.error('Failed to update webhook')
    },
  })

  const resetForm = () => {
    setName('')
    setDescription('')
    setUrl('')
    setSecret('')
    setEnabled(true)
    setTableName('')
    setSelectedOps(['INSERT', 'UPDATE', 'DELETE'])
    setEvents([])
    setMaxRetries(3)
    setTimeoutSeconds(30)
  }

  const addEvent = () => {
    if (!tableName.trim()) {
      toast.error('Please enter a table name')
      return
    }
    if (selectedOps.length === 0) {
      toast.error('Please select at least one operation')
      return
    }

    setEvents([...events, { table: tableName.trim(), operations: selectedOps }])
    setTableName('')
    setSelectedOps(['INSERT', 'UPDATE', 'DELETE'])
  }

  const removeEvent = (index: number) => {
    setEvents(events.filter((_, i) => i !== index))
  }

  const handleCreate = () => {
    if (!name.trim()) {
      toast.error('Please enter a webhook name')
      return
    }
    if (!url.trim()) {
      toast.error('Please enter a webhook URL')
      return
    }
    if (events.length === 0) {
      toast.error('Please add at least one event')
      return
    }

    createMutation.mutate({
      name: name.trim(),
      description: description.trim() || undefined,
      url: url.trim(),
      secret: secret.trim() || undefined,
      enabled,
      events,
      max_retries: maxRetries,
      retry_backoff_seconds: 5,
      timeout_seconds: timeoutSeconds,
      headers: {},
    })
  }

  const filteredWebhooks = webhooks?.filter(
    (webhook) =>
      webhook.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      webhook.url.toLowerCase().includes(searchQuery.toLowerCase())
  )

  const getStatusVariant = (status: string): 'default' | 'secondary' | 'destructive' => {
    if (status === 'success') return 'default'
    if (status === 'failed') return 'destructive'
    return 'secondary'
  }

  return (
    <>
      <Header fixed>
        <SearchComponent />
        <div className='ms-auto flex items-center space-x-4'>
          <ThemeSwitch />
          <ConfigDrawer />
          <ProfileDropdown />
        </div>
      </Header>

      <Main>
        <div className='space-y-4'>
          <div>
            <h1 className='text-3xl font-bold tracking-tight'>Webhooks</h1>
            <p className='text-muted-foreground'>
              Configure webhooks to receive real-time event notifications
            </p>
          </div>

      {/* Stats Cards */}
      <div className='grid gap-4 md:grid-cols-3'>
        <Card>
          <CardHeader className='flex flex-row items-center justify-between space-y-0 pb-2'>
            <CardTitle className='text-sm font-medium'>Total Webhooks</CardTitle>
            <Webhook className='h-4 w-4 text-muted-foreground' />
          </CardHeader>
          <CardContent>
            <div className='text-2xl font-bold'>{webhooks?.length || 0}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className='flex flex-row items-center justify-between space-y-0 pb-2'>
            <CardTitle className='text-sm font-medium'>Active</CardTitle>
            <Check className='h-4 w-4 text-muted-foreground' />
          </CardHeader>
          <CardContent>
            <div className='text-2xl font-bold'>
              {webhooks?.filter((w) => w.enabled).length || 0}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className='flex flex-row items-center justify-between space-y-0 pb-2'>
            <CardTitle className='text-sm font-medium'>Disabled</CardTitle>
            <X className='h-4 w-4 text-muted-foreground' />
          </CardHeader>
          <CardContent>
            <div className='text-2xl font-bold'>
              {webhooks?.filter((w) => !w.enabled).length || 0}
            </div>
          </CardContent>
        </Card>
      </div>

      <Tabs defaultValue='webhooks' className='space-y-4'>
        <TabsList>
          <TabsTrigger value='webhooks'>Webhooks</TabsTrigger>
          <TabsTrigger value='deliveries' disabled={!selectedWebhook}>
            Deliveries {selectedWebhook && `(${selectedWebhook.name})`}
          </TabsTrigger>
        </TabsList>

        <TabsContent value='webhooks' className='space-y-4'>
          <Card>
            <CardHeader>
              <div className='flex items-center justify-between'>
                <div>
                  <CardTitle>Webhooks</CardTitle>
                  <CardDescription>Manage webhook configurations</CardDescription>
                </div>
                <Button onClick={() => setShowCreateDialog(true)}>
                  <Plus className='mr-2 h-4 w-4' />
                  Create Webhook
                </Button>
              </div>
            </CardHeader>
            <CardContent>
              <div className='mb-4'>
                <div className='relative'>
                  <Search className='absolute left-2 top-2.5 h-4 w-4 text-muted-foreground' />
                  <Input
                    placeholder='Search webhooks...'
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    className='pl-8'
                  />
                </div>
              </div>

              {isLoading ? (
                <div className='flex items-center justify-center py-8 text-muted-foreground'>
                  Loading webhooks...
                </div>
              ) : filteredWebhooks && filteredWebhooks.length > 0 ? (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Name</TableHead>
                      <TableHead>URL</TableHead>
                      <TableHead>Events</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead className='text-right'>Actions</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {filteredWebhooks.map((webhook) => (
                      <TableRow key={webhook.id}>
                        <TableCell>
                          <div>
                            <div className='font-medium'>{webhook.name}</div>
                            {webhook.description && (
                              <div className='text-xs text-muted-foreground'>
                                {webhook.description}
                              </div>
                            )}
                          </div>
                        </TableCell>
                        <TableCell>
                          <code className='text-xs'>{webhook.url}</code>
                        </TableCell>
                        <TableCell>
                          <div className='flex flex-wrap gap-1'>
                            {webhook.events.slice(0, 2).map((event, i) => (
                              <Badge key={i} variant='outline' className='text-xs'>
                                {event.table}: {event.operations.join(', ')}
                              </Badge>
                            ))}
                            {webhook.events.length > 2 && (
                              <Badge variant='outline' className='text-xs'>
                                +{webhook.events.length - 2} more
                              </Badge>
                            )}
                          </div>
                        </TableCell>
                        <TableCell>
                          <div className='flex items-center gap-2'>
                            <Switch
                              checked={webhook.enabled}
                              onCheckedChange={(checked) =>
                                toggleMutation.mutate({ id: webhook.id, enabled: checked })
                              }
                            />
                            <Badge variant={webhook.enabled ? 'default' : 'secondary'}>
                              {webhook.enabled ? 'Enabled' : 'Disabled'}
                            </Badge>
                          </div>
                        </TableCell>
                        <TableCell className='text-right'>
                          <div className='flex justify-end gap-2'>
                            <Button
                              variant='outline'
                              size='sm'
                              onClick={() => setSelectedWebhook(webhook)}
                            >
                              <Clock className='h-4 w-4' />
                            </Button>
                            <Button
                              variant='outline'
                              size='sm'
                              onClick={() => testMutation.mutate(webhook.id)}
                              disabled={testMutation.isPending}
                            >
                              <Send className='h-4 w-4' />
                            </Button>
                            <Button
                              variant='destructive'
                              size='sm'
                              onClick={() => {
                                if (confirm('Are you sure you want to delete this webhook?')) {
                                  deleteMutation.mutate(webhook.id)
                                }
                              }}
                              disabled={deleteMutation.isPending}
                            >
                              <Trash2 className='h-4 w-4' />
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              ) : (
                <div className='flex flex-col items-center justify-center py-12 text-center'>
                  <Webhook className='mb-4 h-12 w-12 text-muted-foreground' />
                  <p className='text-muted-foreground'>
                    {searchQuery ? 'No webhooks match your search' : 'No webhooks yet'}
                  </p>
                  {!searchQuery && (
                    <Button
                      onClick={() => setShowCreateDialog(true)}
                      variant='outline'
                      className='mt-4'
                    >
                      Create Your First Webhook
                    </Button>
                  )}
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value='deliveries' className='space-y-4'>
          <Card>
            <CardHeader>
              <CardTitle>Delivery History</CardTitle>
              <CardDescription>Recent webhook delivery attempts</CardDescription>
            </CardHeader>
            <CardContent>
              {deliveries && deliveries.length > 0 ? (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Event</TableHead>
                      <TableHead>Table</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>HTTP Code</TableHead>
                      <TableHead>Time</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {deliveries.map((delivery) => (
                      <TableRow key={delivery.id}>
                        <TableCell>
                          <Badge variant='outline'>{delivery.event_type}</Badge>
                        </TableCell>
                        <TableCell>{delivery.table_name}</TableCell>
                        <TableCell>
                          <Badge variant={getStatusVariant(delivery.status)}>
                            {delivery.status}
                          </Badge>
                        </TableCell>
                        <TableCell>{delivery.http_status_code || 'N/A'}</TableCell>
                        <TableCell className='text-sm text-muted-foreground'>
                          {formatDistanceToNow(new Date(delivery.created_at), { addSuffix: true })}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              ) : (
                <div className='flex flex-col items-center justify-center py-12 text-center'>
                  <Clock className='mb-4 h-12 w-12 text-muted-foreground' />
                  <p className='text-muted-foreground'>No delivery history yet</p>
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* Create Webhook Dialog */}
      <Dialog open={showCreateDialog} onOpenChange={setShowCreateDialog}>
        <DialogContent className='max-w-2xl max-h-[90vh] overflow-y-auto'>
          <DialogHeader>
            <DialogTitle>Create Webhook</DialogTitle>
            <DialogDescription>
              Configure a webhook to receive HTTP notifications for database events
            </DialogDescription>
          </DialogHeader>
          <div className='grid gap-4 py-4'>
            <div className='grid gap-2'>
              <Label htmlFor='name'>
                Name <span className='text-destructive'>*</span>
              </Label>
              <Input
                id='name'
                placeholder='My Webhook'
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
            </div>
            <div className='grid gap-2'>
              <Label htmlFor='description'>Description</Label>
              <Input
                id='description'
                placeholder='Webhook for order notifications'
                value={description}
                onChange={(e) => setDescription(e.target.value)}
              />
            </div>
            <div className='grid gap-2'>
              <Label htmlFor='url'>
                URL <span className='text-destructive'>*</span>
              </Label>
              <Input
                id='url'
                placeholder='https://example.com/webhook'
                value={url}
                onChange={(e) => setUrl(e.target.value)}
              />
            </div>
            <div className='grid gap-2'>
              <Label htmlFor='secret'>Secret (for HMAC verification)</Label>
              <Input
                id='secret'
                placeholder='Optional webhook secret'
                value={secret}
                onChange={(e) => setSecret(e.target.value)}
              />
            </div>

            <div className='grid gap-2'>
              <Label>Events Configuration</Label>
              <div className='space-y-2 rounded-md border p-4'>
                <div className='grid grid-cols-2 gap-2'>
                  <div>
                    <Label htmlFor='tableName'>Table Name</Label>
                    <Select value={tableName} onValueChange={setTableName}>
                      <SelectTrigger>
                        <SelectValue placeholder='Select a table' />
                      </SelectTrigger>
                      <SelectContent>
                        {tables?.map((table) => (
                          <SelectItem key={`${table.schema}.${table.name}`} value={table.name}>
                            {table.schema}.{table.name}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                  <div>
                    <Label>Operations</Label>
                    <div className='flex flex-col gap-2 pt-2'>
                      {OPERATIONS.map((op) => (
                        <div key={op} className='flex items-center space-x-2'>
                          <Checkbox
                            id={op}
                            checked={selectedOps.includes(op)}
                            onCheckedChange={(checked) => {
                              if (checked) {
                                setSelectedOps([...selectedOps, op])
                              } else {
                                setSelectedOps(selectedOps.filter((o) => o !== op))
                              }
                            }}
                          />
                          <label htmlFor={op} className='text-sm'>
                            {op}
                          </label>
                        </div>
                      ))}
                    </div>
                  </div>
                </div>
                <Button type='button' variant='outline' size='sm' onClick={addEvent}>
                  Add Event
                </Button>

                {events.length > 0 && (
                  <div className='mt-2 space-y-2'>
                    <Label>Configured Events:</Label>
                    {events.map((event, index) => (
                      <div
                        key={index}
                        className='flex items-center justify-between rounded border p-2'
                      >
                        <span className='text-sm'>
                          <strong>{event.table}</strong>: {event.operations.join(', ')}
                        </span>
                        <Button
                          type='button'
                          variant='ghost'
                          size='sm'
                          onClick={() => removeEvent(index)}
                        >
                          <X className='h-4 w-4' />
                        </Button>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>

            <div className='grid grid-cols-2 gap-4'>
              <div className='grid gap-2'>
                <Label htmlFor='maxRetries'>Max Retries</Label>
                <Input
                  id='maxRetries'
                  type='number'
                  min='0'
                  max='10'
                  value={maxRetries}
                  onChange={(e) => setMaxRetries(parseInt(e.target.value) || 3)}
                />
              </div>
              <div className='grid gap-2'>
                <Label htmlFor='timeout'>Timeout (seconds)</Label>
                <Input
                  id='timeout'
                  type='number'
                  min='5'
                  max='300'
                  value={timeoutSeconds}
                  onChange={(e) => setTimeoutSeconds(parseInt(e.target.value) || 30)}
                />
              </div>
            </div>

            <div className='flex items-center space-x-2'>
              <Switch id='enabled' checked={enabled} onCheckedChange={setEnabled} />
              <Label htmlFor='enabled'>Enable webhook immediately</Label>
            </div>
          </div>
          <DialogFooter>
            <Button variant='outline' onClick={() => setShowCreateDialog(false)}>
              Cancel
            </Button>
            <Button onClick={handleCreate} disabled={createMutation.isPending}>
              {createMutation.isPending ? 'Creating...' : 'Create Webhook'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
        </div>
      </Main>
    </>
  )
}
