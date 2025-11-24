import { useState, useEffect, useCallback } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import {
  FileCode,
  Search,
  Play,
  Copy,
  History,
  Code2,
  Filter,
  RefreshCw,
  Zap,
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
  CardDescription,
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
import { Separator } from '@/components/ui/separator'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { ImpersonationBanner } from '@/components/impersonation-banner'
import { ImpersonationSelector } from '@/features/impersonation/components/impersonation-selector'
import {
  functionsApi,
  rpcApi,
  type EdgeFunction,
  type EdgeFunctionExecution,
  type RPCFunction,
} from '@/lib/api'

export const Route = createFileRoute('/_authenticated/functions/')({
  component: FunctionsPage,
})

// Local interfaces not in api.ts
interface FunctionCall {
  function: RPCFunction
  params: Record<string, unknown>
  result: unknown
  timestamp: number
  status: 'success' | 'error'
}

interface FunctionResult {
  success: boolean
  data?: unknown
  error?: unknown
}

function FunctionsPage() {
  const [activeTab, setActiveTab] = useState<'rpc' | 'edge'>('rpc')
  const [functions, setFunctions] = useState<RPCFunction[]>([])
  const [filteredFunctions, setFilteredFunctions] = useState<RPCFunction[]>([])
  const [loading, setLoading] = useState(true)
  const [searchQuery, setSearchQuery] = useState('')
  const [schemaFilter, setSchemaFilter] = useState<string>('all')
  const [selectedFunction, setSelectedFunction] = useState<RPCFunction | null>(
    null
  )
  const [showTester, setShowTester] = useState(false)
  const [executing, setExecuting] = useState(false)
  const [paramValues, setParamValues] = useState<Record<string, unknown>>({})
  const [result, setResult] = useState<FunctionResult | null>(null)
  const [history, setHistory] = useState<FunctionCall[]>([])
  const [showHistory, setShowHistory] = useState(false)

  useEffect(() => {
    fetchFunctions()
    loadHistory()
  }, [])

  useEffect(() => {
    let filtered = functions

    // Search filter
    if (searchQuery) {
      filtered = filtered.filter(
        (fn) =>
          fn.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
          fn.description?.toLowerCase().includes(searchQuery.toLowerCase())
      )
    }

    // Schema filter
    if (schemaFilter !== 'all') {
      filtered = filtered.filter((fn) => fn.schema === schemaFilter)
    }

    setFilteredFunctions(filtered)
  }, [searchQuery, schemaFilter, functions])

  const fetchFunctions = async () => {
    setLoading(true)
    try {
      const data = await rpcApi.list()
      // Filter to show only user-defined functions (exclude internal PostgreSQL functions)
      // User functions are typically in plpgsql or sql, while internal functions use c or internal
      const userFunctions = data.filter(
        (fn: RPCFunction) =>
          // Keep functions in plpgsql, sql, or other high-level languages
          fn.language !== 'c' &&
          fn.language !== 'internal' &&
          // Exclude trigger functions (usually internal housekeeping)
          fn.return_type !== 'trigger'
      )
      setFunctions(userFunctions)
      setFilteredFunctions(userFunctions)
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error('Error fetching functions:', error)
      toast.error('Failed to fetch functions')
    } finally {
      setLoading(false)
    }
  }

  const loadHistory = () => {
    const saved = localStorage.getItem('fluxbase-function-history')
    if (saved) {
      setHistory(JSON.parse(saved))
    }
  }

  const saveHistory = (call: FunctionCall) => {
    const newHistory = [call, ...history].slice(0, 20) // Keep last 20
    setHistory(newHistory)
    localStorage.setItem(
      'fluxbase-function-history',
      JSON.stringify(newHistory)
    )
  }

  const openTester = (fn: RPCFunction) => {
    setSelectedFunction(fn)
    setParamValues({})
    setResult(null)
    setShowTester(true)
  }

  const executeFunction = async () => {
    if (!selectedFunction) return

    setExecuting(true)
    setResult(null)

    try {
      const data = await rpcApi.execute(
        selectedFunction.schema,
        selectedFunction.name,
        paramValues
      )

      setResult({ success: true, data })
      toast.success('Function executed successfully')
      saveHistory({
        function: selectedFunction,
        params: paramValues,
        result: data,
        timestamp: Date.now(),
        status: 'success',
      })
    } catch (error: unknown) {
      // eslint-disable-next-line no-console
      console.error('Error executing function:', error)
      const errorMessage =
        error instanceof Error ? error.message : 'Unknown error'
      setResult({ success: false, error: errorMessage })
      toast.error('Failed to execute function')
      saveHistory({
        function: selectedFunction,
        params: paramValues,
        result: errorMessage,
        timestamp: Date.now(),
        status: 'error',
      })
    } finally {
      setExecuting(false)
    }
  }

  const replayFromHistory = (call: FunctionCall) => {
    setSelectedFunction(call.function)
    setParamValues(call.params)
    setResult(null)
    setShowHistory(false)
    setShowTester(true)
  }

  const copyCode = (lang: 'curl' | 'javascript' | 'typescript') => {
    if (!selectedFunction) return

    const path =
      selectedFunction.schema === 'public'
        ? `/api/v1/rpc/${selectedFunction.name}`
        : `/api/v1/rpc/${selectedFunction.schema}/${selectedFunction.name}`

    let code = ''

    if (lang === 'curl') {
      code = `curl -X POST '${window.location.origin}${path}' \\
  -H 'Content-Type: application/json' \\
  -H 'Authorization: Bearer YOUR_TOKEN' \\
  -d '${JSON.stringify(paramValues, null, 2)}'`
    } else if (lang === 'javascript') {
      code = `const response = await fetch('${window.location.origin}${path}', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer ' + token
  },
  body: JSON.stringify(${JSON.stringify(paramValues, null, 2)})
})

const data = await response.json()
// eslint-disable-next-line no-console
console.log(data)`
    } else if (lang === 'typescript') {
      code = `import { fluxbase } from '@fluxbase/sdk'

const { data, error } = await fluxbase.rpc('${selectedFunction.name}', ${JSON.stringify(paramValues, null, 2)})

// eslint-disable-next-line no-console
if (error) console.error(error)
// eslint-disable-next-line no-console
else console.log(data)`
    }

    navigator.clipboard.writeText(code)
    toast.success(`${lang} code copied to clipboard`)
  }

  const schemas = Array.from(new Set(functions.map((fn) => fn.schema)))

  if (loading) {
    return (
      <div className='flex h-96 items-center justify-center'>
        <RefreshCw className='text-muted-foreground h-8 w-8 animate-spin' />
      </div>
    )
  }

  return (
    <div className='flex flex-col gap-6 p-6'>
      <ImpersonationBanner />

      <div className='flex items-center justify-between'>
        <div>
          <h1 className='text-3xl font-bold'>Functions</h1>
          <p className='text-muted-foreground'>
            Manage PostgreSQL RPC functions and Edge Functions (Deno runtime)
          </p>
        </div>
        <div className='flex items-center gap-2'>
          <ImpersonationSelector />
          <Button onClick={fetchFunctions} variant='outline' size='sm'>
            <RefreshCw className='mr-2 h-4 w-4' />
            Refresh
          </Button>
        </div>
      </div>

      <Tabs
        value={activeTab}
        onValueChange={(v) => setActiveTab(v as 'rpc' | 'edge')}
      >
        <TabsList className='grid w-full max-w-md grid-cols-2'>
          <TabsTrigger value='rpc'>
            <FileCode className='mr-2 h-4 w-4' />
            PostgreSQL Functions
          </TabsTrigger>
          <TabsTrigger value='edge'>
            <Zap className='mr-2 h-4 w-4' />
            Edge Functions
          </TabsTrigger>
        </TabsList>

        <TabsContent value='rpc' className='mt-6 space-y-6'>
          {/* Stats */}
          <div className='grid gap-4 md:grid-cols-3'>
            <Card>
              <CardHeader className='pb-3'>
                <CardTitle className='text-sm font-medium'>
                  Total Functions
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className='text-2xl font-bold'>{functions.length}</div>
              </CardContent>
            </Card>
            <Card>
              <CardHeader className='pb-3'>
                <CardTitle className='text-sm font-medium'>Schemas</CardTitle>
              </CardHeader>
              <CardContent>
                <div className='text-2xl font-bold'>{schemas.length}</div>
              </CardContent>
            </Card>
            <Card>
              <CardHeader className='pb-3'>
                <CardTitle className='text-sm font-medium'>History</CardTitle>
              </CardHeader>
              <CardContent>
                <div className='flex items-center justify-between'>
                  <div className='text-2xl font-bold'>{history.length}</div>
                  <Button
                    variant='ghost'
                    size='sm'
                    onClick={() => setShowHistory(true)}
                  >
                    <History className='mr-2 h-4 w-4' />
                    View
                  </Button>
                </div>
              </CardContent>
            </Card>
          </div>

          {/* Filters */}
          <div className='flex items-center gap-3'>
            <div className='relative flex-1'>
              <Search className='text-muted-foreground absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2' />
              <Input
                placeholder='Search functions...'
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className='pl-9'
              />
            </div>
            <Select value={schemaFilter} onValueChange={setSchemaFilter}>
              <SelectTrigger className='w-[180px]'>
                <Filter className='mr-2 h-4 w-4' />
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value='all'>All Schemas</SelectItem>
                {schemas.map((schema) => (
                  <SelectItem key={schema} value={schema}>
                    {schema}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {/* Functions List */}
          <ScrollArea className='h-[calc(100vh-28rem)]'>
            <div className='grid gap-4'>
              {filteredFunctions.length === 0 ? (
                <Card>
                  <CardContent className='p-12 text-center'>
                    <FileCode className='text-muted-foreground mx-auto mb-4 h-12 w-12' />
                    <p className='mb-2 text-lg font-medium'>
                      No functions found
                    </p>
                    <p className='text-muted-foreground text-sm'>
                      {searchQuery || schemaFilter !== 'all'
                        ? 'Try adjusting your filters'
                        : 'Create functions in your PostgreSQL database to see them here'}
                    </p>
                  </CardContent>
                </Card>
              ) : (
                filteredFunctions.map((fn) => (
                  <Card
                    key={`${fn.schema}.${fn.name}`}
                    className='hover:border-primary/50 transition-colors'
                  >
                    <CardHeader>
                      <div className='flex items-start justify-between'>
                        <div className='flex-1'>
                          <div className='mb-2 flex items-center gap-2'>
                            <CardTitle className='text-lg'>{fn.name}</CardTitle>
                            <Badge variant='outline'>{fn.schema}</Badge>
                            <Badge variant='secondary'>
                              {fn.volatility.toLowerCase()}
                            </Badge>
                            {fn.is_set_of && <Badge>returns set</Badge>}
                          </div>
                          <CardDescription>
                            {fn.description || 'No description available'}
                          </CardDescription>
                        </div>
                        <Button onClick={() => openTester(fn)} size='sm'>
                          <Play className='mr-2 h-4 w-4' />
                          Test
                        </Button>
                      </div>
                    </CardHeader>
                    <CardContent>
                      <div className='space-y-2'>
                        <div className='flex items-center gap-2 text-sm'>
                          <span className='font-medium'>Parameters:</span>
                          {!fn.parameters || fn.parameters.length === 0 ? (
                            <span className='text-muted-foreground'>None</span>
                          ) : (
                            <span className='text-muted-foreground'>
                              {fn.parameters
                                .map(
                                  (p) =>
                                    `${p.name || `arg${p.position}`}: ${p.type}`
                                )
                                .join(', ')}
                            </span>
                          )}
                        </div>
                        <div className='flex items-center gap-2 text-sm'>
                          <span className='font-medium'>Returns:</span>
                          <span className='text-muted-foreground'>
                            {fn.is_set_of
                              ? `SETOF ${fn.return_type}`
                              : fn.return_type}
                          </span>
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                ))
              )}
            </div>
          </ScrollArea>

          {/* Function Tester Dialog */}
          <Dialog open={showTester} onOpenChange={setShowTester}>
            <DialogContent className='max-h-[90vh] max-w-2xl overflow-y-auto'>
              <DialogHeader>
                <DialogTitle className='flex items-center gap-2'>
                  <FileCode className='h-5 w-5' />
                  {selectedFunction?.schema}.{selectedFunction?.name}
                </DialogTitle>
                <DialogDescription>
                  {selectedFunction?.description ||
                    'Test this PostgreSQL function'}
                </DialogDescription>
              </DialogHeader>

              <div className='space-y-4'>
                {/* Function Info */}
                <div className='flex flex-wrap items-center gap-2'>
                  <Badge variant='outline'>{selectedFunction?.schema}</Badge>
                  <Badge variant='secondary'>
                    {selectedFunction?.volatility.toLowerCase()}
                  </Badge>
                  <Badge>{selectedFunction?.language}</Badge>
                  {selectedFunction?.is_set_of && <Badge>returns set</Badge>}
                </div>

                <Separator />

                {/* Parameters */}
                {selectedFunction &&
                  selectedFunction.parameters &&
                  selectedFunction.parameters.length > 0 && (
                    <div className='space-y-3'>
                      <h4 className='font-medium'>Parameters</h4>
                      {selectedFunction.parameters.map((param) => (
                        <div key={param.position} className='space-y-2'>
                          <label className='flex items-center gap-2 text-sm font-medium'>
                            {param.name || `arg${param.position}`}
                            <Badge variant='outline' className='font-normal'>
                              {param.type}
                            </Badge>
                            {!param.has_default && (
                              <span className='text-destructive text-xs'>
                                required
                              </span>
                            )}
                          </label>
                          <Input
                            placeholder={`Enter ${param.type} value...`}
                            value={String(
                              paramValues[
                                param.name || `arg${param.position}`
                              ] ?? ''
                            )}
                            onChange={(e) =>
                              setParamValues({
                                ...paramValues,
                                [param.name || `arg${param.position}`]:
                                  e.target.value,
                              })
                            }
                          />
                        </div>
                      ))}
                    </div>
                  )}

                {selectedFunction &&
                  (!selectedFunction.parameters ||
                    selectedFunction.parameters.length === 0) && (
                    <div className='text-muted-foreground text-sm'>
                      This function takes no parameters
                    </div>
                  )}

                {/* Result */}
                {result && (
                  <div className='space-y-2'>
                    <h4 className='font-medium'>Result</h4>
                    <div
                      className={`overflow-x-auto rounded-lg p-4 font-mono text-sm ${
                        result.success
                          ? 'border border-green-500/20 bg-green-500/10'
                          : 'bg-destructive/10 border-destructive/20 border'
                      }`}
                    >
                      <pre>
                        {JSON.stringify(
                          result.success ? result.data : result.error,
                          null,
                          2
                        )}
                      </pre>
                    </div>
                  </div>
                )}
              </div>

              <DialogFooter className='flex-col gap-2 sm:flex-row'>
                <div className='flex flex-1 gap-2'>
                  <Button
                    variant='outline'
                    size='sm'
                    onClick={() => copyCode('curl')}
                  >
                    <Code2 className='mr-2 h-4 w-4' />
                    cURL
                  </Button>
                  <Button
                    variant='outline'
                    size='sm'
                    onClick={() => copyCode('javascript')}
                  >
                    <Code2 className='mr-2 h-4 w-4' />
                    JavaScript
                  </Button>
                  <Button
                    variant='outline'
                    size='sm'
                    onClick={() => copyCode('typescript')}
                  >
                    <Code2 className='mr-2 h-4 w-4' />
                    TypeScript
                  </Button>
                </div>
                <Button onClick={executeFunction} disabled={executing}>
                  {executing ? (
                    <>
                      <RefreshCw className='mr-2 h-4 w-4 animate-spin' />
                      Executing...
                    </>
                  ) : (
                    <>
                      <Play className='mr-2 h-4 w-4' />
                      Execute
                    </>
                  )}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>

          {/* History Dialog */}
          <Dialog open={showHistory} onOpenChange={setShowHistory}>
            <DialogContent className='max-h-[90vh] max-w-2xl'>
              <DialogHeader>
                <DialogTitle className='flex items-center gap-2'>
                  <History className='h-5 w-5' />
                  Execution History
                </DialogTitle>
                <DialogDescription>
                  Recent function calls (last 20)
                </DialogDescription>
              </DialogHeader>

              <ScrollArea className='h-[60vh]'>
                <div className='space-y-3'>
                  {history.length === 0 ? (
                    <div className='text-muted-foreground py-12 text-center'>
                      <History className='mx-auto mb-4 h-12 w-12 opacity-50' />
                      <p>No execution history yet</p>
                    </div>
                  ) : (
                    history.map((call, i) => (
                      <Card
                        key={i}
                        className='hover:border-primary/50 cursor-pointer transition-colors'
                        onClick={() => replayFromHistory(call)}
                      >
                        <CardHeader className='pb-3'>
                          <div className='flex items-start justify-between'>
                            <div className='flex-1'>
                              <div className='mb-1 flex items-center gap-2'>
                                <span className='font-medium'>
                                  {call.function.schema}.{call.function.name}
                                </span>
                                <Badge
                                  variant={
                                    call.status === 'success'
                                      ? 'default'
                                      : 'destructive'
                                  }
                                >
                                  {call.status}
                                </Badge>
                              </div>
                              <p className='text-muted-foreground text-xs'>
                                {new Date(call.timestamp).toLocaleString()}
                              </p>
                            </div>
                            <Button variant='ghost' size='sm'>
                              <Copy className='h-4 w-4' />
                            </Button>
                          </div>
                        </CardHeader>
                        {Object.keys(call.params).length > 0 && (
                          <CardContent className='pt-0'>
                            <div className='text-muted-foreground truncate font-mono text-xs'>
                              {JSON.stringify(call.params)}
                            </div>
                          </CardContent>
                        )}
                      </Card>
                    ))
                  )}
                </div>
              </ScrollArea>
            </DialogContent>
          </Dialog>
        </TabsContent>

        <TabsContent value='edge' className='mt-6 space-y-6'>
          <EdgeFunctionsTab />
        </TabsContent>
      </Tabs>
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

  const fetchEdgeFunctions = useCallback(async (shouldReload = true) => {
    setLoading(true)
    try {
      // First, reload functions from disk (only on initial load or manual refresh)
      if (shouldReload) {
        await reloadFunctionsFromDisk()
      }

      const data = await functionsApi.list()
      setEdgeFunctions(data || [])
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error('Error fetching edge functions:', error)
      toast.error('Failed to fetch edge functions')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchEdgeFunctions()
  }, [fetchEdgeFunctions])

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
        <div className='flex gap-2'>
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
              <Card
                key={fn.id}
                className='hover:border-primary/50 transition-colors'
              >
                <CardHeader>
                  <div className='flex items-start justify-between'>
                    <div className='flex-1'>
                      <div className='mb-2 flex items-center gap-2'>
                        <CardTitle className='text-lg'>{fn.name}</CardTitle>
                        <div className='flex items-center gap-2'>
                          <Switch
                            checked={fn.enabled}
                            onCheckedChange={() => toggleFunction(fn)}
                          />
                          <span className='text-muted-foreground text-sm'>
                            {fn.enabled ? 'Enabled' : 'Disabled'}
                          </span>
                        </div>
                        <Badge variant='outline'>v{fn.version}</Badge>
                        {fn.cron_schedule && (
                          <Badge variant='outline'>
                            <Clock className='mr-1 h-3 w-3' />
                            scheduled
                          </Badge>
                        )}
                      </div>
                      <CardDescription>
                        {fn.description || 'No description'}
                      </CardDescription>
                    </div>
                    <div className='flex gap-2'>
                      <Button
                        onClick={() => openInvokeDialog(fn)}
                        size='sm'
                        variant='outline'
                        disabled={!fn.enabled}
                      >
                        <Play className='mr-2 h-4 w-4' />
                        Invoke
                      </Button>
                      <Button
                        onClick={() => openEditDialog(fn)}
                        size='sm'
                        variant='outline'
                      >
                        <Edit className='h-4 w-4' />
                      </Button>
                      <Button
                        onClick={() => deleteFunction(fn.name)}
                        size='sm'
                        variant='outline'
                      >
                        <Trash2 className='h-4 w-4' />
                      </Button>
                    </div>
                  </div>
                </CardHeader>
                <CardContent>
                  <div className='space-y-2 text-sm'>
                    <div className='flex items-center gap-4'>
                      <span className='text-muted-foreground'>Timeout:</span>
                      <span>{fn.timeout_seconds}s</span>
                      <span className='text-muted-foreground'>Memory:</span>
                      <span>{fn.memory_limit_mb}MB</span>
                    </div>
                    <div className='flex items-center gap-2'>
                      <span className='text-muted-foreground'>
                        Permissions:
                      </span>
                      {fn.allow_net && <Badge variant='outline'>net</Badge>}
                      {fn.allow_env && <Badge variant='outline'>env</Badge>}
                      {fn.allow_read && <Badge variant='outline'>read</Badge>}
                      {fn.allow_write && <Badge variant='outline'>write</Badge>}
                    </div>
                    <div className='flex items-center gap-2'>
                      <Button
                        onClick={() => fetchExecutions(fn.name)}
                        variant='ghost'
                        size='sm'
                      >
                        <History className='mr-2 h-4 w-4' />
                        View Logs
                      </Button>
                    </div>
                  </div>
                </CardContent>
              </Card>
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
