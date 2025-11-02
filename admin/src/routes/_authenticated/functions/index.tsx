import { createFileRoute } from '@tanstack/react-router'
import { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Separator } from '@/components/ui/separator'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
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
} from 'lucide-react'
import { toast } from 'sonner'
import { ImpersonationSelector } from '@/features/impersonation/components/impersonation-selector'
import { ImpersonationBanner } from '@/components/impersonation-banner'

export const Route = createFileRoute('/_authenticated/functions/')({
  component: FunctionsPage,
})

interface FunctionParam {
  name: string
  type: string
  mode: string
  has_default: boolean
  position: number
}

interface RPCFunction {
  schema: string
  name: string
  description: string
  parameters: FunctionParam[]
  return_type: string
  is_set_of: boolean
  volatility: string
  language: string
}

interface FunctionCall {
  function: RPCFunction
  params: Record<string, any>
  result: any
  timestamp: number
  status: 'success' | 'error'
}

interface EdgeFunction {
  id: string
  name: string
  description?: string
  code: string
  version: number
  cron_schedule?: string
  enabled: boolean
  timeout_seconds: number
  memory_limit_mb: number
  allow_net: boolean
  allow_env: boolean
  allow_read: boolean
  allow_write: boolean
  created_at: string
  updated_at: string
}

interface EdgeFunctionExecution {
  id: string
  function_id: string
  trigger_type: string
  status: string
  status_code?: number
  duration_ms?: number
  result?: string
  logs?: string
  error_message?: string
  executed_at: string
}

function FunctionsPage() {
  const [activeTab, setActiveTab] = useState<'rpc' | 'edge'>('rpc')
  const [functions, setFunctions] = useState<RPCFunction[]>([])
  const [filteredFunctions, setFilteredFunctions] = useState<RPCFunction[]>([])
  const [loading, setLoading] = useState(true)
  const [searchQuery, setSearchQuery] = useState('')
  const [schemaFilter, setSchemaFilter] = useState<string>('all')
  const [selectedFunction, setSelectedFunction] = useState<RPCFunction | null>(null)
  const [showTester, setShowTester] = useState(false)
  const [executing, setExecuting] = useState(false)
  const [paramValues, setParamValues] = useState<Record<string, any>>({})
  const [result, setResult] = useState<any>(null)
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
      // Use impersonation token if active, otherwise use admin token
      const impersonationToken = localStorage.getItem('fluxbase_impersonation_token')
      const adminToken = localStorage.getItem('access_token')
      const token = impersonationToken || adminToken

      const res = await fetch('/api/v1/rpc/', {
        headers: {
          Authorization: token ? `Bearer ${token}` : '',
        },
      })

      if (res.ok) {
        const data = await res.json()
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
      } else {
        toast.error('Failed to fetch functions')
      }
    } catch (error) {
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
    localStorage.setItem('fluxbase-function-history', JSON.stringify(newHistory))
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
      // Use impersonation token if active, otherwise use admin token
      const impersonationToken = localStorage.getItem('fluxbase_impersonation_token')
      const adminToken = localStorage.getItem('access_token')
      const token = impersonationToken || adminToken

      const path =
        selectedFunction.schema === 'public'
          ? `/api/v1/rpc/${selectedFunction.name}`
          : `/api/v1/rpc/${selectedFunction.schema}/${selectedFunction.name}`

      const res = await fetch(path, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: token ? `Bearer ${token}` : '',
        },
        body: JSON.stringify(paramValues),
      })

      const data = await res.json()

      if (res.ok) {
        setResult({ success: true, data })
        toast.success('Function executed successfully')
        saveHistory({
          function: selectedFunction,
          params: paramValues,
          result: data,
          timestamp: Date.now(),
          status: 'success',
        })
      } else {
        setResult({ success: false, error: data.error || 'Function execution failed' })
        toast.error(data.error || 'Function execution failed')
        saveHistory({
          function: selectedFunction,
          params: paramValues,
          result: data.error,
          timestamp: Date.now(),
          status: 'error',
        })
      }
    } catch (error: any) {
      console.error('Error executing function:', error)
      setResult({ success: false, error: error.message })
      toast.error('Failed to execute function')
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
console.log(data)`
    } else if (lang === 'typescript') {
      code = `import { fluxbase } from '@fluxbase/sdk'

const { data, error } = await fluxbase.rpc('${selectedFunction.name}', ${JSON.stringify(paramValues, null, 2)})

if (error) console.error(error)
else console.log(data)`
    }

    navigator.clipboard.writeText(code)
    toast.success(`${lang} code copied to clipboard`)
  }

  const schemas = Array.from(new Set(functions.map((fn) => fn.schema)))

  if (loading) {
    return (
      <div className="flex items-center justify-center h-96">
        <RefreshCw className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-6 p-6">
      <ImpersonationBanner />

      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Functions</h1>
          <p className="text-muted-foreground">
            Manage PostgreSQL RPC functions and Edge Functions (Deno runtime)
          </p>
        </div>
        <div className="flex items-center gap-2">
          <ImpersonationSelector />
          <Button onClick={fetchFunctions} variant="outline" size="sm">
            <RefreshCw className="h-4 w-4 mr-2" />
            Refresh
          </Button>
        </div>
      </div>

      <Tabs value={activeTab} onValueChange={(v) => setActiveTab(v as 'rpc' | 'edge')}>
        <TabsList className="grid w-full max-w-md grid-cols-2">
          <TabsTrigger value="rpc">
            <FileCode className="h-4 w-4 mr-2" />
            PostgreSQL Functions
          </TabsTrigger>
          <TabsTrigger value="edge">
            <Zap className="h-4 w-4 mr-2" />
            Edge Functions
          </TabsTrigger>
        </TabsList>

        <TabsContent value="rpc" className="space-y-6 mt-6">

      {/* Stats */}
      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium">Total Functions</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{functions.length}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium">Schemas</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{schemas.length}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium">History</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex items-center justify-between">
              <div className="text-2xl font-bold">{history.length}</div>
              <Button variant="ghost" size="sm" onClick={() => setShowHistory(true)}>
                <History className="h-4 w-4 mr-2" />
                View
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Filters */}
      <div className="flex items-center gap-3">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Search functions..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-9"
          />
        </div>
        <Select value={schemaFilter} onValueChange={setSchemaFilter}>
          <SelectTrigger className="w-[180px]">
            <Filter className="h-4 w-4 mr-2" />
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Schemas</SelectItem>
            {schemas.map((schema) => (
              <SelectItem key={schema} value={schema}>
                {schema}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {/* Functions List */}
      <ScrollArea className="h-[calc(100vh-28rem)]">
        <div className="grid gap-4">
          {filteredFunctions.length === 0 ? (
            <Card>
              <CardContent className="p-12 text-center">
                <FileCode className="h-12 w-12 mx-auto mb-4 text-muted-foreground" />
                <p className="text-lg font-medium mb-2">No functions found</p>
                <p className="text-sm text-muted-foreground">
                  {searchQuery || schemaFilter !== 'all'
                    ? 'Try adjusting your filters'
                    : 'Create functions in your PostgreSQL database to see them here'}
                </p>
              </CardContent>
            </Card>
          ) : (
            filteredFunctions.map((fn) => (
              <Card key={`${fn.schema}.${fn.name}`} className="hover:border-primary/50 transition-colors">
                <CardHeader>
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <div className="flex items-center gap-2 mb-2">
                        <CardTitle className="text-lg">{fn.name}</CardTitle>
                        <Badge variant="outline">{fn.schema}</Badge>
                        <Badge variant="secondary">{fn.volatility.toLowerCase()}</Badge>
                        {fn.is_set_of && <Badge>returns set</Badge>}
                      </div>
                      <CardDescription>
                        {fn.description || 'No description available'}
                      </CardDescription>
                    </div>
                    <Button onClick={() => openTester(fn)} size="sm">
                      <Play className="h-4 w-4 mr-2" />
                      Test
                    </Button>
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="space-y-2">
                    <div className="flex items-center gap-2 text-sm">
                      <span className="font-medium">Parameters:</span>
                      {!fn.parameters || fn.parameters.length === 0 ? (
                        <span className="text-muted-foreground">None</span>
                      ) : (
                        <span className="text-muted-foreground">
                          {fn.parameters
                            .map((p) => `${p.name || `arg${p.position}`}: ${p.type}`)
                            .join(', ')}
                        </span>
                      )}
                    </div>
                    <div className="flex items-center gap-2 text-sm">
                      <span className="font-medium">Returns:</span>
                      <span className="text-muted-foreground">
                        {fn.is_set_of ? `SETOF ${fn.return_type}` : fn.return_type}
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
        <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <FileCode className="h-5 w-5" />
              {selectedFunction?.schema}.{selectedFunction?.name}
            </DialogTitle>
            <DialogDescription>
              {selectedFunction?.description || 'Test this PostgreSQL function'}
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            {/* Function Info */}
            <div className="flex items-center gap-2 flex-wrap">
              <Badge variant="outline">{selectedFunction?.schema}</Badge>
              <Badge variant="secondary">{selectedFunction?.volatility.toLowerCase()}</Badge>
              <Badge>{selectedFunction?.language}</Badge>
              {selectedFunction?.is_set_of && <Badge>returns set</Badge>}
            </div>

            <Separator />

            {/* Parameters */}
            {selectedFunction && selectedFunction.parameters && selectedFunction.parameters.length > 0 && (
              <div className="space-y-3">
                <h4 className="font-medium">Parameters</h4>
                {selectedFunction.parameters.map((param) => (
                  <div key={param.position} className="space-y-2">
                    <label className="text-sm font-medium flex items-center gap-2">
                      {param.name || `arg${param.position}`}
                      <Badge variant="outline" className="font-normal">
                        {param.type}
                      </Badge>
                      {!param.has_default && (
                        <span className="text-xs text-destructive">required</span>
                      )}
                    </label>
                    <Input
                      placeholder={`Enter ${param.type} value...`}
                      value={paramValues[param.name || `arg${param.position}`] || ''}
                      onChange={(e) =>
                        setParamValues({
                          ...paramValues,
                          [param.name || `arg${param.position}`]: e.target.value,
                        })
                      }
                    />
                  </div>
                ))}
              </div>
            )}

            {selectedFunction && (!selectedFunction.parameters || selectedFunction.parameters.length === 0) && (
              <div className="text-sm text-muted-foreground">
                This function takes no parameters
              </div>
            )}

            {/* Result */}
            {result && (
              <div className="space-y-2">
                <h4 className="font-medium">Result</h4>
                <div
                  className={`p-4 rounded-lg font-mono text-sm overflow-x-auto ${
                    result.success
                      ? 'bg-green-500/10 border border-green-500/20'
                      : 'bg-destructive/10 border border-destructive/20'
                  }`}
                >
                  <pre>{JSON.stringify(result.success ? result.data : result.error, null, 2)}</pre>
                </div>
              </div>
            )}
          </div>

          <DialogFooter className="flex-col sm:flex-row gap-2">
            <div className="flex gap-2 flex-1">
              <Button variant="outline" size="sm" onClick={() => copyCode('curl')}>
                <Code2 className="h-4 w-4 mr-2" />
                cURL
              </Button>
              <Button variant="outline" size="sm" onClick={() => copyCode('javascript')}>
                <Code2 className="h-4 w-4 mr-2" />
                JavaScript
              </Button>
              <Button variant="outline" size="sm" onClick={() => copyCode('typescript')}>
                <Code2 className="h-4 w-4 mr-2" />
                TypeScript
              </Button>
            </div>
            <Button onClick={executeFunction} disabled={executing}>
              {executing ? (
                <>
                  <RefreshCw className="h-4 w-4 mr-2 animate-spin" />
                  Executing...
                </>
              ) : (
                <>
                  <Play className="h-4 w-4 mr-2" />
                  Execute
                </>
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* History Dialog */}
      <Dialog open={showHistory} onOpenChange={setShowHistory}>
        <DialogContent className="max-w-2xl max-h-[90vh]">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <History className="h-5 w-5" />
              Execution History
            </DialogTitle>
            <DialogDescription>Recent function calls (last 20)</DialogDescription>
          </DialogHeader>

          <ScrollArea className="h-[60vh]">
            <div className="space-y-3">
              {history.length === 0 ? (
                <div className="text-center py-12 text-muted-foreground">
                  <History className="h-12 w-12 mx-auto mb-4 opacity-50" />
                  <p>No execution history yet</p>
                </div>
              ) : (
                history.map((call, i) => (
                  <Card key={i} className="hover:border-primary/50 transition-colors cursor-pointer"
                    onClick={() => replayFromHistory(call)}>
                    <CardHeader className="pb-3">
                      <div className="flex items-start justify-between">
                        <div className="flex-1">
                          <div className="flex items-center gap-2 mb-1">
                            <span className="font-medium">
                              {call.function.schema}.{call.function.name}
                            </span>
                            <Badge variant={call.status === 'success' ? 'default' : 'destructive'}>
                              {call.status}
                            </Badge>
                          </div>
                          <p className="text-xs text-muted-foreground">
                            {new Date(call.timestamp).toLocaleString()}
                          </p>
                        </div>
                        <Button variant="ghost" size="sm">
                          <Copy className="h-4 w-4" />
                        </Button>
                      </div>
                    </CardHeader>
                    {Object.keys(call.params).length > 0 && (
                      <CardContent className="pt-0">
                        <div className="text-xs font-mono text-muted-foreground truncate">
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

        <TabsContent value="edge" className="space-y-6 mt-6">
          <EdgeFunctionsTab />
        </TabsContent>
      </Tabs>
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
  const [selectedFunction, setSelectedFunction] = useState<EdgeFunction | null>(null)
  const [executions, setExecutions] = useState<EdgeFunctionExecution[]>([])
  const [invoking, setInvoking] = useState(false)
  const [invokeResult, setInvokeResult] = useState<{ success: boolean; data: string; error?: string } | null>(null)
  const [wordWrap, setWordWrap] = useState(false)
  const [logsWordWrap, setLogsWordWrap] = useState(false)

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
    allow_net: true,
    allow_env: true,
    allow_read: false,
    allow_write: false,
    cron_schedule: '',
  })

  const [invokeBody, setInvokeBody] = useState('{}')

  useEffect(() => {
    fetchEdgeFunctions()
  }, [])

  const fetchEdgeFunctions = async () => {
    setLoading(true)
    try {
      const token = localStorage.getItem('fluxbase-auth-token')
      const res = await fetch('/api/v1/functions', {
        headers: {
          Authorization: token ? `Bearer ${token}` : '',
        },
      })

      if (res.ok) {
        const data = await res.json()
        setEdgeFunctions(data || [])
      } else {
        toast.error('Failed to fetch edge functions')
      }
    } catch (error) {
      console.error('Error fetching edge functions:', error)
      toast.error('Failed to fetch edge functions')
    } finally {
      setLoading(false)
    }
  }

  const createFunction = async () => {
    try {
      const token = localStorage.getItem('fluxbase-auth-token')
      const res = await fetch('/api/v1/functions', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: token ? `Bearer ${token}` : '',
        },
        body: JSON.stringify({
          ...formData,
          cron_schedule: formData.cron_schedule || null,
        }),
      })

      if (res.ok) {
        toast.success('Edge function created successfully')
        setShowCreateDialog(false)
        resetForm()
        fetchEdgeFunctions()
      } else {
        const error = await res.json()
        toast.error(error.error || 'Failed to create edge function')
      }
    } catch (error) {
      console.error('Error creating edge function:', error)
      toast.error('Failed to create edge function')
    }
  }

  const updateFunction = async () => {
    if (!selectedFunction) return

    try {
      const token = localStorage.getItem('fluxbase-auth-token')
      const res = await fetch(`/api/v1/functions/${selectedFunction.name}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          Authorization: token ? `Bearer ${token}` : '',
        },
        body: JSON.stringify({
          code: formData.code,
          description: formData.description,
          timeout_seconds: formData.timeout_seconds,
          allow_net: formData.allow_net,
          allow_env: formData.allow_env,
          allow_read: formData.allow_read,
          allow_write: formData.allow_write,
          cron_schedule: formData.cron_schedule || null,
        }),
      })

      if (res.ok) {
        toast.success('Edge function updated successfully')
        setShowEditDialog(false)
        fetchEdgeFunctions()
      } else {
        const error = await res.json()
        toast.error(error.error || 'Failed to update edge function')
      }
    } catch (error) {
      console.error('Error updating edge function:', error)
      toast.error('Failed to update edge function')
    }
  }

  const deleteFunction = async (name: string) => {
    if (!confirm(`Are you sure you want to delete function "${name}"?`)) return

    try {
      const token = localStorage.getItem('fluxbase-auth-token')
      const res = await fetch(`/api/v1/functions/${name}`, {
        method: 'DELETE',
        headers: {
          Authorization: token ? `Bearer ${token}` : '',
        },
      })

      if (res.ok) {
        toast.success('Edge function deleted successfully')
        fetchEdgeFunctions()
      } else {
        const error = await res.json()
        toast.error(error.error || 'Failed to delete edge function')
      }
    } catch (error) {
      console.error('Error deleting edge function:', error)
      toast.error('Failed to delete edge function')
    }
  }

  const toggleFunction = async (fn: EdgeFunction) => {
    const newEnabledState = !fn.enabled

    try {
      const token = localStorage.getItem('fluxbase-auth-token')
      const res = await fetch(`/api/v1/functions/${fn.name}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          Authorization: token ? `Bearer ${token}` : '',
        },
        body: JSON.stringify({
          code: fn.code,
          description: fn.description,
          timeout_seconds: fn.timeout_seconds,
          allow_net: fn.allow_net,
          allow_env: fn.allow_env,
          allow_read: fn.allow_read,
          allow_write: fn.allow_write,
          cron_schedule: fn.cron_schedule || null,
          enabled: newEnabledState,
        }),
      })

      if (res.ok) {
        toast.success(`Function ${newEnabledState ? 'enabled' : 'disabled'}`)
        fetchEdgeFunctions()
      } else {
        const error = await res.json()
        toast.error(error.error || 'Failed to toggle function')
      }
    } catch (error) {
      console.error('Error toggling function:', error)
      toast.error('Failed to toggle function')
    }
  }

  const invokeFunction = async () => {
    if (!selectedFunction) return

    setInvoking(true)
    try {
      const token = localStorage.getItem('fluxbase-auth-token')
      const res = await fetch(`/api/v1/functions/${selectedFunction.name}/invoke`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: token ? `Bearer ${token}` : '',
        },
        body: invokeBody,
      })

      const result = await res.text()

      if (res.ok) {
        toast.success('Function invoked successfully')
        setInvokeResult({ success: true, data: result })
        setShowInvokeDialog(false)
        setShowResultDialog(true)
      } else {
        toast.error('Function invocation failed')
        setInvokeResult({ success: false, data: '', error: result })
        setShowInvokeDialog(false)
        setShowResultDialog(true)
      }
    } catch (error: any) {
      console.error('Error invoking function:', error)
      toast.error('Failed to invoke function')
      setInvokeResult({ success: false, data: '', error: error.message })
      setShowInvokeDialog(false)
      setShowResultDialog(true)
    } finally {
      setInvoking(false)
    }
  }

  const fetchExecutions = async (functionName: string) => {
    try {
      const token = localStorage.getItem('fluxbase-auth-token')
      const res = await fetch(`/api/v1/functions/${functionName}/executions?limit=20`, {
        headers: {
          Authorization: token ? `Bearer ${token}` : '',
        },
      })

      if (res.ok) {
        const data = await res.json()
        setExecutions(data || [])
        setShowLogsDialog(true)
      } else {
        toast.error('Failed to fetch execution logs')
      }
    } catch (error) {
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
      allow_net: true,
      allow_env: true,
      allow_read: false,
      allow_write: false,
      cron_schedule: '',
    })
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-96">
        <RefreshCw className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  return (
    <>
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold">Edge Functions</h2>
          <p className="text-sm text-muted-foreground">
            Deploy and run TypeScript/JavaScript functions with Deno runtime
          </p>
        </div>
        <div className="flex gap-2">
          <Button onClick={fetchEdgeFunctions} variant="outline" size="sm">
            <RefreshCw className="h-4 w-4 mr-2" />
            Refresh
          </Button>
          <Button onClick={() => setShowCreateDialog(true)} size="sm">
            <Plus className="h-4 w-4 mr-2" />
            New Function
          </Button>
        </div>
      </div>

      {/* Stats */}
      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium">Total Functions</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{edgeFunctions.length}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium">Active</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {edgeFunctions.filter((f) => f.enabled).length}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium">Scheduled</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {edgeFunctions.filter((f) => f.cron_schedule).length}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Functions List */}
      <ScrollArea className="h-[calc(100vh-28rem)]">
        <div className="grid gap-4">
          {edgeFunctions.length === 0 ? (
            <Card>
              <CardContent className="p-12 text-center">
                <Zap className="h-12 w-12 mx-auto mb-4 text-muted-foreground" />
                <p className="text-lg font-medium mb-2">No edge functions yet</p>
                <p className="text-sm text-muted-foreground mb-4">
                  Create your first edge function to get started
                </p>
                <Button onClick={() => setShowCreateDialog(true)}>
                  <Plus className="h-4 w-4 mr-2" />
                  Create Edge Function
                </Button>
              </CardContent>
            </Card>
          ) : (
            edgeFunctions.map((fn) => (
              <Card key={fn.id} className="hover:border-primary/50 transition-colors">
                <CardHeader>
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <div className="flex items-center gap-2 mb-2">
                        <CardTitle className="text-lg">{fn.name}</CardTitle>
                        <div className="flex items-center gap-2">
                          <Switch
                            checked={fn.enabled}
                            onCheckedChange={() => toggleFunction(fn)}
                          />
                          <span className="text-sm text-muted-foreground">
                            {fn.enabled ? 'Enabled' : 'Disabled'}
                          </span>
                        </div>
                        <Badge variant="outline">v{fn.version}</Badge>
                        {fn.cron_schedule && (
                          <Badge variant="outline">
                            <Clock className="h-3 w-3 mr-1" />
                            scheduled
                          </Badge>
                        )}
                      </div>
                      <CardDescription>
                        {fn.description || 'No description'}
                      </CardDescription>
                    </div>
                    <div className="flex gap-2">
                      <Button
                        onClick={() => openInvokeDialog(fn)}
                        size="sm"
                        variant="outline"
                        disabled={!fn.enabled}
                      >
                        <Play className="h-4 w-4 mr-2" />
                        Invoke
                      </Button>
                      <Button onClick={() => openEditDialog(fn)} size="sm" variant="outline">
                        <Edit className="h-4 w-4" />
                      </Button>
                      <Button
                        onClick={() => deleteFunction(fn.name)}
                        size="sm"
                        variant="outline"
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="space-y-2 text-sm">
                    <div className="flex items-center gap-4">
                      <span className="text-muted-foreground">Timeout:</span>
                      <span>{fn.timeout_seconds}s</span>
                      <span className="text-muted-foreground">Memory:</span>
                      <span>{fn.memory_limit_mb}MB</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="text-muted-foreground">Permissions:</span>
                      {fn.allow_net && <Badge variant="outline">net</Badge>}
                      {fn.allow_env && <Badge variant="outline">env</Badge>}
                      {fn.allow_read && <Badge variant="outline">read</Badge>}
                      {fn.allow_write && <Badge variant="outline">write</Badge>}
                    </div>
                    <div className="flex items-center gap-2">
                      <Button
                        onClick={() => fetchExecutions(fn.name)}
                        variant="ghost"
                        size="sm"
                      >
                        <History className="h-4 w-4 mr-2" />
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
        <DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Create Edge Function</DialogTitle>
            <DialogDescription>
              Deploy a new TypeScript/JavaScript function with Deno runtime
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            <div>
              <Label htmlFor="name">Function Name</Label>
              <Input
                id="name"
                placeholder="my_function"
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              />
            </div>

            <div>
              <Label htmlFor="description">Description (optional)</Label>
              <Input
                id="description"
                placeholder="What does this function do?"
                value={formData.description}
                onChange={(e) => setFormData({ ...formData, description: e.target.value })}
              />
            </div>

            <div>
              <Label htmlFor="code">Code (TypeScript)</Label>
              <Textarea
                id="code"
                className="font-mono text-sm min-h-[400px]"
                value={formData.code}
                onChange={(e) => setFormData({ ...formData, code: e.target.value })}
              />
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <Label htmlFor="timeout">Timeout (seconds)</Label>
                <Input
                  id="timeout"
                  type="number"
                  min={1}
                  max={300}
                  value={formData.timeout_seconds}
                  onChange={(e) =>
                    setFormData({ ...formData, timeout_seconds: parseInt(e.target.value) })
                  }
                />
              </div>

              <div>
                <Label htmlFor="cron">Cron Schedule (optional)</Label>
                <Input
                  id="cron"
                  placeholder="0 0 * * *"
                  value={formData.cron_schedule}
                  onChange={(e) => setFormData({ ...formData, cron_schedule: e.target.value })}
                />
              </div>
            </div>

            <div>
              <Label>Permissions</Label>
              <div className="grid grid-cols-2 gap-3 mt-2">
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={formData.allow_net}
                    onChange={(e) => setFormData({ ...formData, allow_net: e.target.checked })}
                  />
                  <span>Allow Network Access</span>
                </label>
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={formData.allow_env}
                    onChange={(e) => setFormData({ ...formData, allow_env: e.target.checked })}
                  />
                  <span>Allow Environment Variables</span>
                </label>
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={formData.allow_read}
                    onChange={(e) => setFormData({ ...formData, allow_read: e.target.checked })}
                  />
                  <span>Allow File Read</span>
                </label>
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={formData.allow_write}
                    onChange={(e) => setFormData({ ...formData, allow_write: e.target.checked })}
                  />
                  <span>Allow File Write</span>
                </label>
              </div>
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setShowCreateDialog(false)}>
              Cancel
            </Button>
            <Button onClick={createFunction}>Create Function</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Function Dialog */}
      <Dialog open={showEditDialog} onOpenChange={setShowEditDialog}>
        <DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Edit Edge Function</DialogTitle>
            <DialogDescription>Update function code and settings</DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            <div>
              <Label htmlFor="edit-description">Description</Label>
              <Input
                id="edit-description"
                value={formData.description}
                onChange={(e) => setFormData({ ...formData, description: e.target.value })}
              />
            </div>

            <div>
              <Label htmlFor="edit-code">Code</Label>
              <Textarea
                id="edit-code"
                className="font-mono text-sm min-h-[400px]"
                value={formData.code}
                onChange={(e) => setFormData({ ...formData, code: e.target.value })}
              />
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <Label htmlFor="edit-timeout">Timeout (seconds)</Label>
                <Input
                  id="edit-timeout"
                  type="number"
                  min={1}
                  max={300}
                  value={formData.timeout_seconds}
                  onChange={(e) =>
                    setFormData({ ...formData, timeout_seconds: parseInt(e.target.value) })
                  }
                />
              </div>

              <div>
                <Label htmlFor="edit-cron">Cron Schedule</Label>
                <Input
                  id="edit-cron"
                  placeholder="0 0 * * *"
                  value={formData.cron_schedule}
                  onChange={(e) => setFormData({ ...formData, cron_schedule: e.target.value })}
                />
              </div>
            </div>

            <div>
              <Label>Permissions</Label>
              <div className="grid grid-cols-2 gap-3 mt-2">
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={formData.allow_net}
                    onChange={(e) => setFormData({ ...formData, allow_net: e.target.checked })}
                  />
                  <span>Allow Network Access</span>
                </label>
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={formData.allow_env}
                    onChange={(e) => setFormData({ ...formData, allow_env: e.target.checked })}
                  />
                  <span>Allow Environment Variables</span>
                </label>
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={formData.allow_read}
                    onChange={(e) => setFormData({ ...formData, allow_read: e.target.checked })}
                  />
                  <span>Allow File Read</span>
                </label>
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={formData.allow_write}
                    onChange={(e) => setFormData({ ...formData, allow_write: e.target.checked })}
                  />
                  <span>Allow File Write</span>
                </label>
              </div>
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setShowEditDialog(false)}>
              Cancel
            </Button>
            <Button onClick={updateFunction}>Update Function</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Invoke Function Dialog */}
      <Dialog open={showInvokeDialog} onOpenChange={setShowInvokeDialog}>
        <DialogContent className="max-w-5xl w-[90vw]">
          <DialogHeader>
            <DialogTitle>Invoke Edge Function</DialogTitle>
            <DialogDescription>
              Test {selectedFunction?.name} with custom input
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            <div>
              <Label htmlFor="invoke-body">Request Body (JSON)</Label>
              <Textarea
                id="invoke-body"
                className="font-mono text-sm min-h-[200px]"
                value={invokeBody}
                onChange={(e) => setInvokeBody(e.target.value)}
              />
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setShowInvokeDialog(false)}>
              Cancel
            </Button>
            <Button onClick={invokeFunction} disabled={invoking}>
              {invoking ? (
                <>
                  <RefreshCw className="h-4 w-4 mr-2 animate-spin" />
                  Invoking...
                </>
              ) : (
                <>
                  <Play className="h-4 w-4 mr-2" />
                  Invoke
                </>
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Execution Logs Dialog */}
      <Dialog open={showLogsDialog} onOpenChange={setShowLogsDialog}>
        <DialogContent className="!max-w-[95vw] !w-[95vw] max-h-[95vh] flex flex-col overflow-hidden">
          <DialogHeader className="flex-shrink-0">
            <DialogTitle>Execution Logs</DialogTitle>
            <DialogDescription>
              Recent executions for {selectedFunction?.name}
            </DialogDescription>
          </DialogHeader>

          <div className="flex items-center space-x-2 flex-shrink-0">
            <Switch
              id="logs-word-wrap"
              checked={logsWordWrap}
              onCheckedChange={setLogsWordWrap}
            />
            <Label htmlFor="logs-word-wrap" className="cursor-pointer">
              Word wrap
            </Label>
          </div>

          <div className="flex-1 overflow-auto min-h-0 border rounded-lg p-4 space-y-3">
            {executions.length === 0 ? (
              <div className="text-center py-12 text-muted-foreground">
                <History className="h-12 w-12 mx-auto mb-4 opacity-50" />
                <p>No executions yet</p>
              </div>
            ) : (
              executions.map((exec) => (
                <Card key={exec.id} className="overflow-hidden">
                    <CardHeader className="pb-3">
                      <div className="flex items-start justify-between">
                        <div className="flex-1">
                          <div className="flex items-center gap-2 mb-1">
                            <Badge
                              variant={exec.status === 'success' ? 'default' : 'destructive'}
                            >
                              {exec.status}
                            </Badge>
                            <Badge variant="outline">{exec.trigger_type}</Badge>
                            {exec.status_code && (
                              <Badge variant="secondary">{exec.status_code}</Badge>
                            )}
                            {exec.duration_ms && (
                              <span className="text-xs text-muted-foreground">
                                {exec.duration_ms}ms
                              </span>
                            )}
                          </div>
                          <p className="text-xs text-muted-foreground">
                            {new Date(exec.executed_at).toLocaleString()}
                          </p>
                        </div>
                      </div>
                    </CardHeader>
                    {(exec.logs || exec.error_message || exec.result) && (
                      <CardContent className="pt-0 overflow-hidden">
                        {exec.error_message && (
                          <div className="mb-2 min-w-0">
                            <Label className="text-xs text-destructive">Error:</Label>
                            <div className="mt-1 border rounded bg-destructive/10 overflow-auto max-h-40 max-w-full">
                              <pre className={`text-xs p-2 min-w-0 ${logsWordWrap ? 'whitespace-pre-wrap break-words' : 'whitespace-pre'}`}>
                                {exec.error_message}
                              </pre>
                            </div>
                          </div>
                        )}
                        {exec.logs && (
                          <div className="mb-2 min-w-0">
                            <Label className="text-xs">Logs:</Label>
                            <div className="mt-1 border rounded bg-muted overflow-auto max-h-40 max-w-full">
                              <pre className={`text-xs p-2 min-w-0 ${logsWordWrap ? 'whitespace-pre-wrap break-words' : 'whitespace-pre'}`}>
                                {exec.logs}
                              </pre>
                            </div>
                          </div>
                        )}
                        {exec.result && !exec.error_message && (
                          <div className="min-w-0">
                            <Label className="text-xs">Result:</Label>
                            <div className="mt-1 border rounded bg-muted overflow-auto max-h-40 max-w-full">
                              <pre className={`text-xs p-2 min-w-0 ${logsWordWrap ? 'whitespace-pre-wrap break-words' : 'whitespace-pre'}`}>
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
        <DialogContent className="max-w-[95vw] max-h-[95vh] w-[95vw]">
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

          <div className="space-y-4">
            <div className="flex items-center space-x-2">
              <Switch
                id="word-wrap"
                checked={wordWrap}
                onCheckedChange={setWordWrap}
              />
              <Label htmlFor="word-wrap" className="cursor-pointer">
                Word wrap
              </Label>
            </div>

            {invokeResult?.success ? (
              <div className="w-full overflow-hidden">
                <Label>Response</Label>
                <div className="mt-2 border rounded-lg bg-muted overflow-auto h-[70vh]">
                  <pre
                    className={`text-xs p-4 ${
                      wordWrap ? 'whitespace-pre-wrap break-words' : 'whitespace-pre'
                    }`}
                  >
                    {invokeResult.data}
                  </pre>
                </div>
              </div>
            ) : (
              <div className="w-full overflow-hidden">
                <Label>Error</Label>
                <div className="mt-2 border rounded-lg bg-destructive/10 overflow-auto h-[70vh]">
                  <pre
                    className={`text-xs text-destructive p-4 ${
                      wordWrap ? 'whitespace-pre-wrap break-words' : 'whitespace-pre'
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
              variant="outline"
              onClick={() => {
                if (invokeResult?.success) {
                  navigator.clipboard.writeText(invokeResult.data)
                  toast.success('Copied to clipboard')
                }
              }}
              disabled={!invokeResult?.success}
            >
              <Copy className="h-4 w-4 mr-2" />
              Copy
            </Button>
            <Button onClick={() => setShowResultDialog(false)}>Close</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
