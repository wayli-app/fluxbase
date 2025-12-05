import { useState, useEffect, useCallback } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import {
  Play,
  Copy,
  History,
  RefreshCw,
  Plus,
  Edit,
  Trash2,
  Clock,
  HardDrive,
} from 'lucide-react'
import { toast } from 'sonner'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
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
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { ImpersonationBanner } from '@/components/impersonation-banner'
import { ImpersonationSelector } from '@/features/impersonation/components/impersonation-selector'
import {
  functionsApi,
  type EdgeFunction,
  type EdgeFunctionExecution,
} from '@/lib/api'

export const Route = createFileRoute('/_authenticated/functions/')({
  component: FunctionsPage,
})

function FunctionsPage() {
  return (
    <div className='flex flex-col gap-6 p-6'>
      <ImpersonationBanner />

      <div className='flex items-center justify-between'>
        <div>
          <h1 className='text-3xl font-bold'>Edge Functions</h1>
          <p className='text-muted-foreground'>
            Deploy and run TypeScript/JavaScript functions with Deno runtime
          </p>
        </div>
        <ImpersonationSelector />
      </div>

      <EdgeFunctionsTab />
    </div>
  )
}

// Headers Editor Component
interface HeadersEditorProps {
  headers: Array<{ key: string; value: string }>
  onChange: (headers: Array<{ key: string; value: string }>) => void
}

function HeadersEditor({ headers, onChange }: HeadersEditorProps) {
  const addHeader = () => {
    onChange([...headers, { key: '', value: '' }])
  }

  const updateHeader = (index: number, field: 'key' | 'value', value: string) => {
    const updated = [...headers]
    updated[index][field] = value
    onChange(updated)
  }

  const removeHeader = (index: number) => {
    onChange(headers.filter((_, i) => i !== index))
  }

  return (
    <div className='space-y-2'>
      {headers.map((header, index) => (
        <div key={index} className='flex gap-2'>
          <Input
            placeholder='Header name'
            value={header.key}
            onChange={(e) => updateHeader(index, 'key', e.target.value)}
            className='flex-1'
          />
          <Input
            placeholder='Header value'
            value={header.value}
            onChange={(e) => updateHeader(index, 'value', e.target.value)}
            className='flex-1'
          />
          <Button
            variant='ghost'
            size='sm'
            onClick={() => removeHeader(index)}
          >
            <Trash2 className='h-4 w-4' />
          </Button>
        </div>
      ))}
      <Button variant='outline' size='sm' onClick={addHeader}>
        <Plus className='mr-2 h-4 w-4' />
        Add Header
      </Button>
    </div>
  )
}

// Edge Functions Component
function EdgeFunctionsTab() {
  const [edgeFunctions, setEdgeFunctions] = useState<EdgeFunction[]>([])
  const [loading, setLoading] = useState(true)
  const [showCreateDialog, setShowCreateDialog] = useState(false)
  const [showEditDialog, setShowEditDialog] = useState(false)
  const [showInvokeDialog, setShowInvokeDialog] = useState(false)
  const [showLogsDialog, setShowLogsDialog] = useState(false)
  const [showResultDialog, setShowResultDialog] = useState(false)
  const [selectedFunction, setSelectedFunction] = useState<EdgeFunction | null>(
    null
  )
  const [executions, setExecutions] = useState<EdgeFunctionExecution[]>([])
  const [invoking, setInvoking] = useState(false)
  const [invokeResult, setInvokeResult] = useState<{
    success: boolean
    data: string
    error?: string
  } | null>(null)
  const [wordWrap, setWordWrap] = useState(false)
  const [logsWordWrap, setLogsWordWrap] = useState(false)
  const [reloading, setReloading] = useState(false)
  const [namespaces, setNamespaces] = useState<string[]>(['default'])
  const [selectedNamespace, setSelectedNamespace] = useState<string>('default')

  // Form state
  const [formData, setFormData] = useState({
    name: '',
    description: '',
    code: `interface Request {
  method: string;
  url: string;
  headers: Record<string, string>;
  body: string;
}

async function handler(req: Request) {
  // Your code here
  const data = JSON.parse(req.body || "{}");

  return {
    status: 200,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ message: "Hello from edge function!" })
  };
}`,
    timeout_seconds: 30,
    memory_limit_mb: 128,
    allow_net: true,
    allow_env: true,
    allow_read: false,
    allow_write: false,
    cron_schedule: '',
  })

  const [invokeBody, setInvokeBody] = useState('{}')
  const [invokeMethod, setInvokeMethod] = useState<'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH'>('POST')
  const [invokeHeaders, setInvokeHeaders] = useState<Array<{ key: string; value: string }>>([
    { key: '', value: '' }
  ])

  const reloadFunctionsFromDisk = async (showToast = false) => {
    try {
      const result = await functionsApi.reload()
      if (showToast) {
        const created = result.created?.length ?? 0
        const updated = result.updated?.length ?? 0
        const deleted = result.deleted?.length ?? 0
        const errors = result.errors?.length ?? 0

        if (created > 0 || updated > 0 || deleted > 0) {
          const messages = []
          if (created > 0) messages.push(`${created} created`)
          if (updated > 0) messages.push(`${updated} updated`)
          if (deleted > 0) messages.push(`${deleted} deleted`)

          toast.success(`Functions reloaded: ${messages.join(', ')}`)
        } else if (errors > 0) {
          toast.error(`Failed to reload functions: ${errors} errors`)
        } else {
          toast.info('No changes detected')
        }
      }
      return result
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error('Error reloading functions:', error)
      if (showToast) {
        toast.error('Failed to reload functions from filesystem')
      }
      throw error
    }
  }

  const handleReloadClick = async () => {
    setReloading(true)
    try {
      await reloadFunctionsFromDisk(true)
      await fetchEdgeFunctions(false) // Refresh the list without reloading again
    } finally {
      setReloading(false)
    }
  }

  const fetchEdgeFunctions = useCallback(async (shouldReload = true, namespace?: string) => {
    setLoading(true)
    try {
      // First, reload functions from disk (only on initial load or manual refresh)
      if (shouldReload) {
        await reloadFunctionsFromDisk()
      }

      const ns = namespace ?? selectedNamespace
      const data = await functionsApi.list(ns)
      setEdgeFunctions(data || [])
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error('Error fetching edge functions:', error)
      toast.error('Failed to fetch edge functions')
    } finally {
      setLoading(false)
    }
  }, [selectedNamespace])

  // Fetch namespaces on mount
  useEffect(() => {
    const fetchNamespaces = async () => {
      try {
        const data = await functionsApi.listNamespaces()
        setNamespaces(data.length > 0 ? data : ['default'])
        // If current namespace not in list, reset to first available
        if (!data.includes(selectedNamespace)) {
          setSelectedNamespace(data[0] || 'default')
        }
      } catch {
        setNamespaces(['default'])
      }
    }
    fetchNamespaces()
  }, [selectedNamespace])

  useEffect(() => {
    fetchEdgeFunctions()
  }, [fetchEdgeFunctions, selectedNamespace])

  const createFunction = async () => {
    try {
      await functionsApi.create({
        ...formData,
        cron_schedule: formData.cron_schedule || null,
      })
      toast.success('Edge function created successfully')
      setShowCreateDialog(false)
      resetForm()
      fetchEdgeFunctions(false) // Don't reload from disk after creating
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error('Error creating edge function:', error)
      toast.error('Failed to create edge function')
    }
  }

  const updateFunction = async () => {
    if (!selectedFunction) return

    try {
      await functionsApi.update(selectedFunction.name, {
        code: formData.code,
        description: formData.description,
        timeout_seconds: formData.timeout_seconds,
        allow_net: formData.allow_net,
        allow_env: formData.allow_env,
        allow_read: formData.allow_read,
        allow_write: formData.allow_write,
        cron_schedule: formData.cron_schedule || null,
      })
      toast.success('Edge function updated successfully')
      setShowEditDialog(false)
      fetchEdgeFunctions(false) // Don't reload from disk after updating
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error('Error updating edge function:', error)
      toast.error('Failed to update edge function')
    }
  }

  const deleteFunction = async (name: string) => {
    if (!confirm(`Are you sure you want to delete function "${name}"?`)) return

    try {
      await functionsApi.delete(name)
      toast.success('Edge function deleted successfully')
      fetchEdgeFunctions(false) // Don't reload from disk after deleting
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error('Error deleting edge function:', error)
      toast.error('Failed to delete edge function')
    }
  }

  const toggleFunction = async (fn: EdgeFunction) => {
    const newEnabledState = !fn.enabled

    try {
      await functionsApi.update(fn.name, {
        code: fn.code,
        description: fn.description,
        timeout_seconds: fn.timeout_seconds,
        allow_net: fn.allow_net,
        allow_env: fn.allow_env,
        allow_read: fn.allow_read,
        allow_write: fn.allow_write,
        cron_schedule: fn.cron_schedule || null,
        enabled: newEnabledState,
      })
      toast.success(`Function ${newEnabledState ? 'enabled' : 'disabled'}`)
      fetchEdgeFunctions(false) // Don't reload from disk after toggling
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error('Error toggling function:', error)
      toast.error('Failed to toggle function')
    }
  }

  const invokeFunction = async () => {
    if (!selectedFunction) return

    setInvoking(true)
    try {
      // Convert headers array to object, filtering empty ones
      const headersObj = invokeHeaders
        .filter((h) => h.key.trim() !== '')
        .reduce((acc, h) => ({ ...acc, [h.key]: h.value }), {})

      const result = await functionsApi.invoke(selectedFunction.name, {
        method: invokeMethod,
        headers: headersObj,
        body: invokeBody,
      })
      toast.success('Function invoked successfully')
      setInvokeResult({ success: true, data: result })
      setShowInvokeDialog(false)
      setShowResultDialog(true)
    } catch (error: unknown) {
      // eslint-disable-next-line no-console
      console.error('Error invoking function:', error)
      toast.error('Failed to invoke function')
      const errorMessage =
        error instanceof Error ? error.message : 'Unknown error'
      setInvokeResult({ success: false, data: '', error: errorMessage })
      setShowInvokeDialog(false)
      setShowResultDialog(true)
    } finally {
      setInvoking(false)
    }
  }

  const fetchExecutions = async (functionName: string) => {
    try {
      const data = await functionsApi.getExecutions(functionName, 20)
      setExecutions(data || [])
      setShowLogsDialog(true)
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error('Error fetching executions:', error)
      toast.error('Failed to fetch execution logs')
    }
  }

  const openEditDialog = (fn: EdgeFunction) => {
    setSelectedFunction(fn)
    setFormData({
      name: fn.name,
      description: fn.description || '',
      code: fn.code,
      timeout_seconds: fn.timeout_seconds,
      memory_limit_mb: fn.memory_limit_mb,
      allow_net: fn.allow_net,
      allow_env: fn.allow_env,
      allow_read: fn.allow_read,
      allow_write: fn.allow_write,
      cron_schedule: fn.cron_schedule || '',
    })
    setShowEditDialog(true)
  }

  const openInvokeDialog = (fn: EdgeFunction) => {
    setSelectedFunction(fn)
    setInvokeBody('{\n  "name": "World"\n}')
    setShowInvokeDialog(true)
  }

  const resetForm = () => {
    setFormData({
      name: '',
      description: '',
      code: `interface Request {
  method: string;
  url: string;
  headers: Record<string, string>;
  body: string;
}

async function handler(req: Request) {
  // Your code here
  const data = JSON.parse(req.body || "{}");

  return {
    status: 200,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ message: "Hello from edge function!" })
  };
}`,
      timeout_seconds: 30,
      memory_limit_mb: 128,
      allow_net: true,
      allow_env: true,
      allow_read: false,
      allow_write: false,
      cron_schedule: '',
    })
  }

  if (loading) {
    return (
      <div className='flex h-96 items-center justify-center'>
        <RefreshCw className='text-muted-foreground h-8 w-8 animate-spin' />
      </div>
    )
  }

  return (
    <>
      <div className='flex items-center justify-between'>
        <div>
          <h2 className='text-2xl font-bold'>Edge Functions</h2>
          <p className='text-muted-foreground text-sm'>
            Deploy and run TypeScript/JavaScript functions with Deno runtime
          </p>
        </div>
        <div className='flex items-center gap-2'>
          <div className='flex items-center gap-2'>
            <Label htmlFor='edge-namespace-select' className='text-sm text-muted-foreground whitespace-nowrap'>
              Namespace:
            </Label>
            <Select value={selectedNamespace} onValueChange={setSelectedNamespace}>
              <SelectTrigger id='edge-namespace-select' className='w-[180px]'>
                <SelectValue placeholder='Select namespace' />
              </SelectTrigger>
              <SelectContent>
                {namespaces.map((ns) => (
                  <SelectItem key={ns} value={ns}>
                    {ns}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <Button
            onClick={handleReloadClick}
            variant='outline'
            size='sm'
            disabled={reloading}
          >
            {reloading ? (
              <>
                <RefreshCw className='mr-2 h-4 w-4 animate-spin' />
                Reloading...
              </>
            ) : (
              <>
                <HardDrive className='mr-2 h-4 w-4' />
                Reload from Filesystem
              </>
            )}
          </Button>
          <Button
            onClick={() => fetchEdgeFunctions()}
            variant='outline'
            size='sm'
          >
            <RefreshCw className='mr-2 h-4 w-4' />
            Refresh
          </Button>
          <Button onClick={() => setShowCreateDialog(true)} size='sm'>
            <Plus className='mr-2 h-4 w-4' />
            New Function
          </Button>
        </div>
      </div>

      {/* Stats */}
      <div className='grid gap-4 md:grid-cols-3'>
        <Card>
          <CardHeader className='pb-3'>
            <CardTitle className='text-sm font-medium'>
              Total Functions
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className='text-2xl font-bold'>{edgeFunctions.length}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className='pb-3'>
            <CardTitle className='text-sm font-medium'>Active</CardTitle>
          </CardHeader>
          <CardContent>
            <div className='text-2xl font-bold'>
              {edgeFunctions.filter((f) => f.enabled).length}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className='pb-3'>
            <CardTitle className='text-sm font-medium'>Scheduled</CardTitle>
          </CardHeader>
          <CardContent>
            <div className='text-2xl font-bold'>
              {edgeFunctions.filter((f) => f.cron_schedule).length}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Functions List */}
      <ScrollArea className='h-[calc(100vh-28rem)]'>
        <div className='grid gap-4'>
          {edgeFunctions.length === 0 ? (
            <Card>
              <CardContent className='p-12 text-center'>
                <Zap className='text-muted-foreground mx-auto mb-4 h-12 w-12' />
                <p className='mb-2 text-lg font-medium'>
                  No edge functions yet
                </p>
                <p className='text-muted-foreground mb-4 text-sm'>
                  Create your first edge function to get started
                </p>
                <Button onClick={() => setShowCreateDialog(true)}>
                  <Plus className='mr-2 h-4 w-4' />
                  Create Edge Function
                </Button>
              </CardContent>
            </Card>
          ) : (
            edgeFunctions.map((fn) => (
              <div
                key={fn.id}
                className='flex items-center justify-between gap-2 px-3 py-1.5 rounded-md border hover:border-primary/50 transition-colors bg-card'
              >
                <div className='flex items-center gap-2 min-w-0 flex-1'>
                  <span className='text-sm font-medium truncate'>{fn.name}</span>
                  <Badge variant='outline' className='shrink-0 text-[10px] px-1 py-0 h-4'>v{fn.version}</Badge>
                  {fn.cron_schedule && (
                    <Badge variant='outline' className='shrink-0 text-[10px] px-1 py-0 h-4'>
                      <Clock className='mr-0.5 h-2.5 w-2.5' />
                      cron
                    </Badge>
                  )}
                  <Switch
                    checked={fn.enabled}
                    onCheckedChange={() => toggleFunction(fn)}
                    className='scale-75'
                  />
                </div>
                <div className='flex items-center gap-1 shrink-0'>
                  <span className='text-[10px] text-muted-foreground'>{fn.timeout_seconds}s</span>
                  <Button
                    onClick={() => fetchExecutions(fn.name)}
                    variant='ghost'
                    size='sm'
                    className='h-6 w-6 p-0'
                    title='View Logs'
                  >
                    <History className='h-3 w-3' />
                  </Button>
                  <Button
                    onClick={() => openInvokeDialog(fn)}
                    size='sm'
                    variant='ghost'
                    className='h-6 px-1.5 text-xs'
                    disabled={!fn.enabled}
                  >
                    <Play className='h-3 w-3' />
                  </Button>
                  <Button
                    onClick={() => openEditDialog(fn)}
                    size='sm'
                    variant='ghost'
                    className='h-6 w-6 p-0'
                  >
                    <Edit className='h-3 w-3' />
                  </Button>
                  <Button
                    onClick={() => deleteFunction(fn.name)}
                    size='sm'
                    variant='ghost'
                    className='h-6 w-6 p-0'
                  >
                    <Trash2 className='h-3 w-3' />
                  </Button>
                </div>
              </div>
            ))
          )}
        </div>
      </ScrollArea>

      {/* Create Function Dialog */}
      <Dialog open={showCreateDialog} onOpenChange={setShowCreateDialog}>
        <DialogContent className='max-h-[90vh] max-w-4xl overflow-y-auto'>
          <DialogHeader>
            <DialogTitle>Create Edge Function</DialogTitle>
            <DialogDescription>
              Deploy a new TypeScript/JavaScript function with Deno runtime
            </DialogDescription>
          </DialogHeader>

          <div className='space-y-4'>
            <div>
              <Label htmlFor='name'>Function Name</Label>
              <Input
                id='name'
                placeholder='my_function'
                value={formData.name}
                onChange={(e) =>
                  setFormData({ ...formData, name: e.target.value })
                }
              />
            </div>

            <div>
              <Label htmlFor='description'>Description (optional)</Label>
              <Input
                id='description'
                placeholder='What does this function do?'
                value={formData.description}
                onChange={(e) =>
                  setFormData({ ...formData, description: e.target.value })
                }
              />
            </div>

            <div>
              <Label htmlFor='code'>Code (TypeScript)</Label>
              <Textarea
                id='code'
                className='min-h-[400px] font-mono text-sm'
                value={formData.code}
                onChange={(e) =>
                  setFormData({ ...formData, code: e.target.value })
                }
              />
            </div>

            <div className='grid grid-cols-2 gap-4'>
              <div>
                <Label htmlFor='timeout'>Timeout (seconds)</Label>
                <Input
                  id='timeout'
                  type='number'
                  min={1}
                  max={300}
                  value={formData.timeout_seconds}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      timeout_seconds: parseInt(e.target.value),
                    })
                  }
                />
              </div>

              <div>
                <Label htmlFor='cron'>Cron Schedule (optional)</Label>
                <Input
                  id='cron'
                  placeholder='0 0 * * *'
                  value={formData.cron_schedule}
                  onChange={(e) =>
                    setFormData({ ...formData, cron_schedule: e.target.value })
                  }
                />
              </div>
            </div>

            <div>
              <Label>Permissions</Label>
              <div className='mt-2 grid grid-cols-2 gap-3'>
                <label className='flex cursor-pointer items-center gap-2'>
                  <input
                    type='checkbox'
                    checked={formData.allow_net}
                    onChange={(e) =>
                      setFormData({ ...formData, allow_net: e.target.checked })
                    }
                  />
                  <span>Allow Network Access</span>
                </label>
                <label className='flex cursor-pointer items-center gap-2'>
                  <input
                    type='checkbox'
                    checked={formData.allow_env}
                    onChange={(e) =>
                      setFormData({ ...formData, allow_env: e.target.checked })
                    }
                  />
                  <span>Allow Environment Variables</span>
                </label>
                <label className='flex cursor-pointer items-center gap-2'>
                  <input
                    type='checkbox'
                    checked={formData.allow_read}
                    onChange={(e) =>
                      setFormData({ ...formData, allow_read: e.target.checked })
                    }
                  />
                  <span>Allow File Read</span>
                </label>
                <label className='flex cursor-pointer items-center gap-2'>
                  <input
                    type='checkbox'
                    checked={formData.allow_write}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        allow_write: e.target.checked,
                      })
                    }
                  />
                  <span>Allow File Write</span>
                </label>
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
            <Button onClick={createFunction}>Create Function</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Function Dialog */}
      <Dialog open={showEditDialog} onOpenChange={setShowEditDialog}>
        <DialogContent className='max-h-[90vh] max-w-4xl overflow-y-auto'>
          <DialogHeader>
            <DialogTitle>Edit Edge Function</DialogTitle>
            <DialogDescription>
              Update function code and settings
            </DialogDescription>
          </DialogHeader>

          <div className='space-y-4'>
            <div>
              <Label htmlFor='edit-description'>Description</Label>
              <Input
                id='edit-description'
                value={formData.description}
                onChange={(e) =>
                  setFormData({ ...formData, description: e.target.value })
                }
              />
            </div>

            <div>
              <Label htmlFor='edit-code'>Code</Label>
              <Textarea
                id='edit-code'
                className='min-h-[400px] font-mono text-sm'
                value={formData.code}
                onChange={(e) =>
                  setFormData({ ...formData, code: e.target.value })
                }
              />
            </div>

            <div className='grid grid-cols-2 gap-4'>
              <div>
                <Label htmlFor='edit-timeout'>Timeout (seconds)</Label>
                <Input
                  id='edit-timeout'
                  type='number'
                  min={1}
                  max={300}
                  value={formData.timeout_seconds}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      timeout_seconds: parseInt(e.target.value),
                    })
                  }
                />
              </div>

              <div>
                <Label htmlFor='edit-cron'>Cron Schedule</Label>
                <Input
                  id='edit-cron'
                  placeholder='0 0 * * *'
                  value={formData.cron_schedule}
                  onChange={(e) =>
                    setFormData({ ...formData, cron_schedule: e.target.value })
                  }
                />
              </div>
            </div>

            <div>
              <Label>Permissions</Label>
              <div className='mt-2 grid grid-cols-2 gap-3'>
                <label className='flex cursor-pointer items-center gap-2'>
                  <input
                    type='checkbox'
                    checked={formData.allow_net}
                    onChange={(e) =>
                      setFormData({ ...formData, allow_net: e.target.checked })
                    }
                  />
                  <span>Allow Network Access</span>
                </label>
                <label className='flex cursor-pointer items-center gap-2'>
                  <input
                    type='checkbox'
                    checked={formData.allow_env}
                    onChange={(e) =>
                      setFormData({ ...formData, allow_env: e.target.checked })
                    }
                  />
                  <span>Allow Environment Variables</span>
                </label>
                <label className='flex cursor-pointer items-center gap-2'>
                  <input
                    type='checkbox'
                    checked={formData.allow_read}
                    onChange={(e) =>
                      setFormData({ ...formData, allow_read: e.target.checked })
                    }
                  />
                  <span>Allow File Read</span>
                </label>
                <label className='flex cursor-pointer items-center gap-2'>
                  <input
                    type='checkbox'
                    checked={formData.allow_write}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        allow_write: e.target.checked,
                      })
                    }
                  />
                  <span>Allow File Write</span>
                </label>
              </div>
            </div>
          </div>

          <DialogFooter>
            <Button variant='outline' onClick={() => setShowEditDialog(false)}>
              Cancel
            </Button>
            <Button onClick={updateFunction}>Update Function</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Invoke Function Dialog */}
      <Dialog open={showInvokeDialog} onOpenChange={setShowInvokeDialog}>
        <DialogContent className='w-[90vw] max-w-5xl max-h-[90vh] overflow-y-auto'>
          <DialogHeader>
            <DialogTitle>Invoke Edge Function</DialogTitle>
            <DialogDescription>
              Test {selectedFunction?.name} with custom HTTP request
            </DialogDescription>
          </DialogHeader>

          <div className='space-y-4'>
            {/* Method selector */}
            <div className='flex items-center gap-4'>
              <Label htmlFor='method'>HTTP Method</Label>
              <Select
                value={invokeMethod}
                onValueChange={(value) => setInvokeMethod(value as 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH')}
              >
                <SelectTrigger className='w-[180px]' id='method'>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value='GET'>GET</SelectItem>
                  <SelectItem value='POST'>POST</SelectItem>
                  <SelectItem value='PUT'>PUT</SelectItem>
                  <SelectItem value='DELETE'>DELETE</SelectItem>
                  <SelectItem value='PATCH'>PATCH</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {/* Tabbed interface for Body/Headers */}
            <Tabs defaultValue='body' className='space-y-4'>
              <TabsList className='grid w-full grid-cols-2'>
                <TabsTrigger value='body'>Body</TabsTrigger>
                <TabsTrigger value='headers'>Headers</TabsTrigger>
              </TabsList>

              <TabsContent value='body' className='space-y-2'>
                <Label htmlFor='invoke-body'>Request Body (JSON)</Label>
                <Textarea
                  id='invoke-body'
                  className='min-h-[300px] font-mono text-sm'
                  value={invokeBody}
                  onChange={(e) => setInvokeBody(e.target.value)}
                  placeholder='{"key": "value"}'
                />
              </TabsContent>

              <TabsContent value='headers' className='space-y-2'>
                <Label>Custom Headers</Label>
                <ScrollArea className='max-h-[350px]'>
                  <HeadersEditor
                    headers={invokeHeaders}
                    onChange={setInvokeHeaders}
                  />
                </ScrollArea>
              </TabsContent>
            </Tabs>
          </div>

          <DialogFooter>
            <Button
              variant='outline'
              onClick={() => setShowInvokeDialog(false)}
            >
              Cancel
            </Button>
            <Button onClick={invokeFunction} disabled={invoking}>
              {invoking ? (
                <>
                  <RefreshCw className='mr-2 h-4 w-4 animate-spin' />
                  Invoking...
                </>
              ) : (
                <>
                  <Play className='mr-2 h-4 w-4' />
                  Invoke
                </>
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Execution Logs Dialog */}
      <Dialog open={showLogsDialog} onOpenChange={setShowLogsDialog}>
        <DialogContent className='flex max-h-[95vh] !w-[95vw] !max-w-[95vw] flex-col overflow-hidden'>
          <DialogHeader className='flex-shrink-0'>
            <DialogTitle>Execution Logs</DialogTitle>
            <DialogDescription>
              Recent executions for {selectedFunction?.name}
            </DialogDescription>
          </DialogHeader>

          <div className='flex flex-shrink-0 items-center space-x-2'>
            <Switch
              id='logs-word-wrap'
              checked={logsWordWrap}
              onCheckedChange={setLogsWordWrap}
            />
            <Label htmlFor='logs-word-wrap' className='cursor-pointer'>
              Word wrap
            </Label>
          </div>

          <div className='min-h-0 flex-1 space-y-3 overflow-auto rounded-lg border p-4'>
            {executions.length === 0 ? (
              <div className='text-muted-foreground py-12 text-center'>
                <History className='mx-auto mb-4 h-12 w-12 opacity-50' />
                <p>No executions yet</p>
              </div>
            ) : (
              executions.map((exec) => (
                <Card key={exec.id} className='overflow-hidden'>
                  <CardHeader className='pb-3'>
                    <div className='flex items-start justify-between'>
                      <div className='flex-1'>
                        <div className='mb-1 flex items-center gap-2'>
                          <Badge
                            variant={
                              exec.status === 'success'
                                ? 'default'
                                : 'destructive'
                            }
                          >
                            {exec.status}
                          </Badge>
                          <Badge variant='outline'>{exec.trigger_type}</Badge>
                          {exec.status_code && (
                            <Badge variant='secondary'>
                              {exec.status_code}
                            </Badge>
                          )}
                          {exec.duration_ms && (
                            <span className='text-muted-foreground text-xs'>
                              {exec.duration_ms}ms
                            </span>
                          )}
                        </div>
                        <p className='text-muted-foreground text-xs'>
                          {new Date(exec.executed_at).toLocaleString()}
                        </p>
                      </div>
                    </div>
                  </CardHeader>
                  {(exec.logs || exec.error_message || exec.result) && (
                    <CardContent className='overflow-hidden pt-0'>
                      {exec.error_message && (
                        <div className='mb-2 min-w-0'>
                          <Label className='text-destructive text-xs'>
                            Error:
                          </Label>
                          <div className='bg-destructive/10 mt-1 max-h-40 max-w-full overflow-auto rounded border'>
                            <pre
                              className={`min-w-0 p-2 text-xs ${logsWordWrap ? 'break-words whitespace-pre-wrap' : 'whitespace-pre'}`}
                            >
                              {exec.error_message}
                            </pre>
                          </div>
                        </div>
                      )}
                      {exec.logs && (
                        <div className='mb-2 min-w-0'>
                          <Label className='text-xs'>Logs:</Label>
                          <div className='bg-muted mt-1 max-h-40 max-w-full overflow-auto rounded border'>
                            <pre
                              className={`min-w-0 p-2 text-xs ${logsWordWrap ? 'break-words whitespace-pre-wrap' : 'whitespace-pre'}`}
                            >
                              {exec.logs}
                            </pre>
                          </div>
                        </div>
                      )}
                      {exec.result && !exec.error_message && (
                        <div className='min-w-0'>
                          <Label className='text-xs'>Result:</Label>
                          <div className='bg-muted mt-1 max-h-40 max-w-full overflow-auto rounded border'>
                            <pre
                              className={`min-w-0 p-2 text-xs ${logsWordWrap ? 'break-words whitespace-pre-wrap' : 'whitespace-pre'}`}
                            >
                              {exec.result}
                            </pre>
                          </div>
                        </div>
                      )}
                    </CardContent>
                  )}
                </Card>
              ))
            )}
          </div>
        </DialogContent>
      </Dialog>

      {/* Result Dialog */}
      <Dialog open={showResultDialog} onOpenChange={setShowResultDialog}>
        <DialogContent className='max-h-[95vh] w-[95vw] max-w-[95vw]'>
          <DialogHeader>
            <DialogTitle>
              {invokeResult?.success ? 'Function Result' : 'Function Error'}
            </DialogTitle>
            <DialogDescription>
              {invokeResult?.success
                ? 'Function executed successfully'
                : 'Function execution failed'}
            </DialogDescription>
          </DialogHeader>

          <div className='space-y-4'>
            <div className='flex items-center space-x-2'>
              <Switch
                id='word-wrap'
                checked={wordWrap}
                onCheckedChange={setWordWrap}
              />
              <Label htmlFor='word-wrap' className='cursor-pointer'>
                Word wrap
              </Label>
            </div>

            {invokeResult?.success ? (
              <div className='w-full overflow-hidden'>
                <Label>Response</Label>
                <div className='bg-muted mt-2 h-[70vh] overflow-auto rounded-lg border'>
                  <pre
                    className={`p-4 text-xs ${
                      wordWrap
                        ? 'break-words whitespace-pre-wrap'
                        : 'whitespace-pre'
                    }`}
                  >
                    {invokeResult.data}
                  </pre>
                </div>
              </div>
            ) : (
              <div className='w-full overflow-hidden'>
                <Label>Error</Label>
                <div className='bg-destructive/10 mt-2 h-[70vh] overflow-auto rounded-lg border'>
                  <pre
                    className={`text-destructive p-4 text-xs ${
                      wordWrap
                        ? 'break-words whitespace-pre-wrap'
                        : 'whitespace-pre'
                    }`}
                  >
                    {invokeResult?.error}
                  </pre>
                </div>
              </div>
            )}
          </div>

          <DialogFooter>
            <Button
              variant='outline'
              onClick={() => {
                if (invokeResult?.success) {
                  navigator.clipboard.writeText(invokeResult.data)
                  toast.success('Copied to clipboard')
                }
              }}
              disabled={!invokeResult?.success}
            >
              <Copy className='mr-2 h-4 w-4' />
              Copy
            </Button>
            <Button onClick={() => setShowResultDialog(false)}>Close</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
