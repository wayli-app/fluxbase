import { createFileRoute } from '@tanstack/react-router'
import { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Separator } from '@/components/ui/separator'
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
} from 'lucide-react'
import { toast } from 'sonner'

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

function FunctionsPage() {
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
      const token = localStorage.getItem('fluxbase-auth-token')
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
      const token = localStorage.getItem('fluxbase-auth-token')
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
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">RPC Functions</h1>
          <p className="text-muted-foreground">
            Test and manage PostgreSQL functions via RPC endpoints
          </p>
        </div>
        <Button onClick={fetchFunctions} variant="outline" size="sm">
          <RefreshCw className="h-4 w-4 mr-2" />
          Refresh
        </Button>
      </div>

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
    </div>
  )
}
