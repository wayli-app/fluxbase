import { useState, useEffect, useCallback } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import {
  Play,
  Save,
  Plus,
  X,
  ChevronDown,
  ChevronRight,
  BookOpen,
  List,
} from 'lucide-react'
import { toast } from 'sonner'
import { getAccessToken } from '@/lib/auth'
import { useImpersonationStore } from '@/stores/impersonation-store'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { ImpersonationBanner } from '@/components/impersonation-banner'
import { PromptDialog } from '@/components/prompt-dialog'
import { DocumentationPanel } from '@/features/api-explorer/components/documentation-panel'
import { EndpointBrowser } from '@/features/api-explorer/components/endpoint-browser'
import type { OpenAPISpec, EndpointInfo } from '@/features/api-explorer/types'
import { ImpersonationPopover } from '@/features/impersonation/components/impersonation-popover'

export const Route = createFileRoute('/_authenticated/api/rest')({
  component: RestAPIExplorer,
})

// Types
interface RequestHistory {
  id: string
  timestamp: number
  method: string
  url: string
  headers: Record<string, string>
  body?: string
  response?: {
    status: number
    statusText: string
    headers: Record<string, string>
    data: unknown
    duration: number
  }
}

interface SavedRequest {
  id: string
  name: string
  method: string
  endpoint: string
  headers: Record<string, string>
  body?: string
  queryParams: Record<string, string>
}

const HTTP_METHODS = ['GET', 'POST', 'PATCH', 'PUT', 'DELETE'] as const
type HttpMethod = (typeof HTTP_METHODS)[number]

function RestAPIExplorer() {
  // State
  const [method, setMethod] = useState<HttpMethod>('GET')
  const [endpoint, setEndpoint] = useState('/api/v1/tables/')
  const [headers, setHeaders] = useState<Record<string, string>>({
    'Content-Type': 'application/json',
  })
  const [body, setBody] = useState('')
  const [queryParams, setQueryParams] = useState<Record<string, string>>({})
  const [response, setResponse] = useState<RequestHistory['response'] | null>(
    null
  )
  const [loading, setLoading] = useState(false)
  const [history, setHistory] = useState<RequestHistory[]>([])
  const [savedRequests, setSavedRequests] = useState<SavedRequest[]>([])
  const [showQueryBuilder, setShowQueryBuilder] = useState(false)
  const [activeTab, setActiveTab] = useState('request')
  const [includeAuthToken, setIncludeAuthToken] = useState(true)

  // New state for OpenAPI integration
  const [openAPISpec, setOpenAPISpec] = useState<OpenAPISpec | null>(null)
  const [selectedEndpoint, setSelectedEndpoint] = useState<EndpointInfo | null>(
    null
  )
  const [showEndpointBrowser, setShowEndpointBrowser] = useState(true)
  const [showDocumentation, setShowDocumentation] = useState(false)

  // Dialog state
  const [showClearHistoryConfirm, setShowClearHistoryConfirm] = useState(false)
  const [showHeaderPrompt, setShowHeaderPrompt] = useState(false)
  const [headerPromptStep, setHeaderPromptStep] = useState<'name' | 'value'>(
    'name'
  )
  const [pendingHeaderName, setPendingHeaderName] = useState('')
  const [showParamPrompt, setShowParamPrompt] = useState(false)
  const [paramPromptStep, setParamPromptStep] = useState<'name' | 'value'>(
    'name'
  )
  const [pendingParamName, setPendingParamName] = useState('')
  const [showSaveRequestPrompt, setShowSaveRequestPrompt] = useState(false)

  // Impersonation state
  const { isImpersonating, impersonationToken } = useImpersonationStore()

  // Load saved data from localStorage
  useEffect(() => {
    const savedHistory = localStorage.getItem('fluxbase-api-history')
    if (savedHistory) {
      try {
        setHistory(JSON.parse(savedHistory))
      } catch {
        // Failed to parse history
      }
    }

    const saved = localStorage.getItem('fluxbase-saved-requests')
    if (saved) {
      try {
        setSavedRequests(JSON.parse(saved))
      } catch {
        // Failed to parse saved requests
      }
    }

    // Load auth token if available
    const token = getAccessToken()
    if (token) {
      setHeaders((prev) => ({ ...prev, Authorization: `Bearer ${token}` }))
    }

    // Fetch OpenAPI specification
    fetchOpenAPISpec()
  }, [])

  // Update Authorization header when impersonation state changes
  useEffect(() => {
    const token =
      isImpersonating && impersonationToken
        ? impersonationToken
        : getAccessToken()

    if (token) {
      setHeaders((prev) => ({ ...prev, Authorization: `Bearer ${token}` }))
    }
  }, [isImpersonating, impersonationToken])

  const fetchOpenAPISpec = async () => {
    try {
      // Include auth token to get the full spec (with database tables)
      const token = getAccessToken()
      const res = await fetch('/openapi.json', {
        headers: token ? { Authorization: `Bearer ${token}` } : {},
      })
      if (res.ok) {
        const spec = await res.json()
        setOpenAPISpec(spec)
      }
    } catch {
      // Failed to fetch OpenAPI spec
    }
  }

  const generateExampleFromSchema = useCallback(
    (schema: Record<string, unknown> | unknown): unknown => {
      if (!schema || typeof schema !== 'object') return null

      const schemaObj = schema as Record<string, unknown>

      if (schemaObj.$ref) return null

      if (schemaObj.example !== undefined) return schemaObj.example

      if (schemaObj.type === 'object' && schemaObj.properties) {
        const example: Record<string, unknown> = {}
        const properties = schemaObj.properties as Record<string, unknown>
        Object.entries(properties).forEach(([key, prop]) => {
          const value = generateExampleFromSchema(prop)
          if (value !== null) {
            example[key] = value
          }
        })
        return Object.keys(example).length > 0 ? example : null
      }

      if (schemaObj.type === 'array' && schemaObj.items) {
        const itemExample = generateExampleFromSchema(schemaObj.items)
        return itemExample ? [itemExample] : null
      }

      // Default values by type
      const defaults: Record<string, unknown> = {
        string: '',
        number: 0,
        integer: 0,
        boolean: false,
      }

      return defaults[schemaObj.type as string] ?? null
    },
    []
  )

  const handleSelectEndpoint = useCallback(
    (endpoint: EndpointInfo) => {
      setSelectedEndpoint(endpoint)
      setMethod(endpoint.method as HttpMethod)
      setEndpoint(endpoint.path)

      // Clear previous state
      setQueryParams({})
      setBody('')

      // Populate parameters if available
      if (endpoint.parameters) {
        const newQueryParams: Record<string, string> = {}
        endpoint.parameters.forEach((param) => {
          if (param.in === 'query' && param.example !== undefined) {
            newQueryParams[param.name] = String(param.example)
          }
        })
        if (Object.keys(newQueryParams).length > 0) {
          setQueryParams(newQueryParams)
        }
      }

      // Populate request body if available
      if (endpoint.requestBody?.content) {
        const jsonContent = endpoint.requestBody.content['application/json']
        if (jsonContent?.example) {
          setBody(JSON.stringify(jsonContent.example, null, 2))
        } else if (jsonContent?.schema) {
          // Generate example from schema
          const example = generateExampleFromSchema(jsonContent.schema)
          if (example) {
            setBody(JSON.stringify(example, null, 2))
          }
        }
      }

      setShowDocumentation(true)
    },
    [generateExampleFromSchema]
  )

  const buildUrl = useCallback(() => {
    const params = new URLSearchParams()
    Object.entries(queryParams).forEach(([key, value]) => {
      if (value) params.append(key, value)
    })
    const queryString = params.toString()
    return queryString ? `${endpoint}?${queryString}` : endpoint
  }, [endpoint, queryParams])

  const executeRequest = async () => {
    setLoading(true)
    const startTime = performance.now()

    try {
      const url = buildUrl()

      // Filter headers based on includeAuthToken
      const filteredHeaders = Object.entries(headers).reduce(
        (acc, [key, value]) => {
          if (
            value &&
            (includeAuthToken || key.toLowerCase() !== 'authorization')
          ) {
            acc[key] = value
          }
          return acc
        },
        {} as Record<string, string>
      )

      // If impersonating, use the impersonation token for Authorization
      if (isImpersonating && impersonationToken && includeAuthToken) {
        filteredHeaders['Authorization'] = `Bearer ${impersonationToken}`
      }

      const options: RequestInit = {
        method,
        headers: filteredHeaders,
      }

      if (body && ['POST', 'PUT', 'PATCH'].includes(method)) {
        options.body = body
      }

      const res = await fetch(url, options)
      const responseHeaders: Record<string, string> = {}
      res.headers.forEach((value, key) => {
        responseHeaders[key] = value
      })

      let data
      const contentType = res.headers.get('content-type')
      if (contentType?.includes('application/json')) {
        data = await res.json()
      } else {
        data = await res.text()
      }

      const duration = performance.now() - startTime

      const responseData = {
        status: res.status,
        statusText: res.statusText,
        headers: responseHeaders,
        data,
        duration,
      }

      setResponse(responseData)

      // Add to history
      const historyEntry: RequestHistory = {
        id: Date.now().toString(),
        timestamp: Date.now(),
        method,
        url,
        headers,
        body,
        response: responseData,
      }

      const newHistory = [historyEntry, ...history].slice(0, 50) // Keep last 50
      setHistory(newHistory)
      localStorage.setItem('fluxbase-api-history', JSON.stringify(newHistory))

      if (res.ok) {
        toast.success(`${method} request successful (${res.status})`)
      } else {
        toast.error(`Request failed: ${res.status} ${res.statusText}`)
      }
    } catch (error: unknown) {
      const errorMessage =
        error instanceof Error ? error.message : 'Unknown error'
      toast.error(`Request failed: ${errorMessage}`)
      setResponse({
        status: 0,
        statusText: 'Network Error',
        headers: {},
        data: { error: errorMessage },
        duration: performance.now() - startTime,
      })
    } finally {
      setLoading(false)
    }
  }

  const addHeader = () => {
    setPendingHeaderName('')
    setHeaderPromptStep('name')
    setShowHeaderPrompt(true)
  }

  const handleHeaderPromptConfirm = (value: string) => {
    if (headerPromptStep === 'name') {
      setPendingHeaderName(value)
      setHeaderPromptStep('value')
    } else {
      setHeaders((prev) => ({ ...prev, [pendingHeaderName]: value }))
      setShowHeaderPrompt(false)
      setPendingHeaderName('')
      setHeaderPromptStep('name')
    }
  }

  const removeHeader = (key: string) => {
    setHeaders((prev) => {
      const newHeaders = { ...prev }
      delete newHeaders[key]
      return newHeaders
    })
  }

  const addQueryParam = () => {
    setPendingParamName('')
    setParamPromptStep('name')
    setShowParamPrompt(true)
  }

  const handleParamPromptConfirm = (value: string) => {
    if (paramPromptStep === 'name') {
      setPendingParamName(value)
      setParamPromptStep('value')
    } else {
      setQueryParams((prev) => ({ ...prev, [pendingParamName]: value }))
      setShowParamPrompt(false)
      setPendingParamName('')
      setParamPromptStep('name')
    }
  }

  const removeQueryParam = (key: string) => {
    setQueryParams((prev) => {
      const newParams = { ...prev }
      delete newParams[key]
      return newParams
    })
  }

  const saveRequest = () => {
    setShowSaveRequestPrompt(true)
  }

  const handleSaveRequestConfirm = (name: string) => {
    const request: SavedRequest = {
      id: Date.now().toString(),
      name,
      method,
      endpoint,
      headers,
      body,
      queryParams,
    }

    const newSaved = [...savedRequests, request]
    setSavedRequests(newSaved)
    localStorage.setItem('fluxbase-saved-requests', JSON.stringify(newSaved))
    toast.success('Request saved')
    setShowSaveRequestPrompt(false)
  }

  const loadSavedRequest = (request: SavedRequest) => {
    setMethod(request.method as HttpMethod)
    setEndpoint(request.endpoint)
    setHeaders(request.headers)
    setBody(request.body || '')
    setQueryParams(request.queryParams)
    toast.success(`Loaded: ${request.name}`)
  }

  const deleteSavedRequest = (id: string) => {
    const newSaved = savedRequests.filter((r) => r.id !== id)
    setSavedRequests(newSaved)
    localStorage.setItem('fluxbase-saved-requests', JSON.stringify(newSaved))
    toast.success('Request deleted')
  }

  const loadHistoryEntry = (entry: RequestHistory) => {
    setMethod(entry.method as HttpMethod)
    setEndpoint(new URL(entry.url, window.location.origin).pathname)
    setHeaders(entry.headers)
    setBody(entry.body || '')

    // Parse query params from URL
    const url = new URL(entry.url, window.location.origin)
    const params: Record<string, string> = {}
    url.searchParams.forEach((value, key) => {
      params[key] = value
    })
    setQueryParams(params)

    setResponse(entry.response || null)
    setActiveTab('response')
  }

  const clearHistory = () => {
    setShowClearHistoryConfirm(true)
  }

  const handleClearHistoryConfirm = () => {
    setHistory([])
    localStorage.removeItem('fluxbase-api-history')
    toast.success('History cleared')
    setShowClearHistoryConfirm(false)
  }

  const generateCode = (
    language: 'curl' | 'javascript' | 'typescript' | 'python'
  ) => {
    const url = buildUrl()
    let code = ''

    switch (language) {
      case 'curl':
        code = `curl -X ${method} "${window.location.origin}${url}"`
        Object.entries(headers).forEach(([key, value]) => {
          if (value) code += `\n  -H "${key}: ${value}"`
        })
        if (body && ['POST', 'PUT', 'PATCH'].includes(method)) {
          code += `\n  -d '${body}'`
        }
        break

      case 'javascript':
        code = `fetch("${window.location.origin}${url}", {
  method: "${method}",
  headers: ${JSON.stringify(headers, null, 2)},${
    body && ['POST', 'PUT', 'PATCH'].includes(method)
      ? `\n  body: ${JSON.stringify(body)},`
      : ''
  }
})
  .then(res => res.json())
  .then(data => data)`
        break

      case 'typescript':
        code = `interface Response {
  // Define your response type here
}

const response = await fetch("${window.location.origin}${url}", {
  method: "${method}",
  headers: ${JSON.stringify(headers, null, 2)},${
    body && ['POST', 'PUT', 'PATCH'].includes(method)
      ? `\n  body: ${JSON.stringify(body)},`
      : ''
  }
})

const data: Response = await response.json()`
        break

      case 'python':
        code = `import requests

response = requests.${method.toLowerCase()}(
    "${window.location.origin}${url}",
    headers=${JSON.stringify(headers, null, 2).replace(/"/g, "'")},${
      body && ['POST', 'PUT', 'PATCH'].includes(method)
        ? `\n    json=${body},`
        : ''
    }
)

data = response.json()
print(data)`
        break
    }

    navigator.clipboard.writeText(code)
    toast.success(`${language} code copied to clipboard`)
  }

  return (
    <div className='flex h-full'>
      {/* Left Sidebar - Endpoint Browser or Saved/History */}
      {showEndpointBrowser ? (
        <div className='bg-muted/10 w-80 border-r'>
          <div className='flex items-center justify-between border-b p-4'>
            <h3 className='flex items-center gap-2 font-semibold'>
              <List className='h-4 w-4' />
              Endpoints
            </h3>
            <Button
              variant='ghost'
              size='sm'
              onClick={() => setShowEndpointBrowser(false)}
            >
              <X className='h-4 w-4' />
            </Button>
          </div>
          <EndpointBrowser
            spec={openAPISpec}
            onSelectEndpoint={handleSelectEndpoint}
            selectedEndpoint={selectedEndpoint}
          />
        </div>
      ) : (
        <div className='bg-muted/10 w-64 space-y-4 border-r p-4'>
          <div>
            <h3 className='mb-2 font-semibold'>Saved Requests</h3>
            <ScrollArea className='h-48'>
              <div className='space-y-1'>
                {savedRequests.map((request) => (
                  <div
                    key={request.id}
                    className='group hover:bg-muted/50 flex cursor-pointer items-center justify-between rounded p-2'
                    onClick={() => loadSavedRequest(request)}
                  >
                    <div className='flex min-w-0 flex-1 items-center gap-2'>
                      <Badge variant='outline' className='text-xs'>
                        {request.method}
                      </Badge>
                      <span className='truncate text-sm'>{request.name}</span>
                    </div>
                    <Button
                      variant='ghost'
                      size='icon'
                      className='h-6 w-6 opacity-0 group-hover:opacity-100'
                      onClick={(e) => {
                        e.stopPropagation()
                        deleteSavedRequest(request.id)
                      }}
                    >
                      <X className='h-3 w-3' />
                    </Button>
                  </div>
                ))}
                {savedRequests.length === 0 && (
                  <p className='text-muted-foreground text-sm'>
                    No saved requests
                  </p>
                )}
              </div>
            </ScrollArea>
          </div>

          <Separator />

          <div>
            <div className='mb-2 flex items-center justify-between'>
              <h3 className='font-semibold'>History</h3>
              {history.length > 0 && (
                <Button variant='ghost' size='sm' onClick={clearHistory}>
                  Clear
                </Button>
              )}
            </div>
            <ScrollArea className='h-64'>
              <div className='space-y-1'>
                {history.map((entry) => (
                  <div
                    key={entry.id}
                    className='hover:bg-muted/50 cursor-pointer rounded p-2'
                    onClick={() => loadHistoryEntry(entry)}
                  >
                    <div className='flex items-center gap-2'>
                      <Badge
                        variant={
                          entry.response?.status && entry.response.status < 400
                            ? 'default'
                            : 'destructive'
                        }
                        className='text-xs'
                      >
                        {entry.response?.status || '---'}
                      </Badge>
                      <Badge variant='outline' className='text-xs'>
                        {entry.method}
                      </Badge>
                      <span className='text-muted-foreground text-xs'>
                        {new URL(
                          entry.url,
                          window.location.origin
                        ).pathname.slice(0, 20)}
                        ...
                      </span>
                    </div>
                    <div className='text-muted-foreground mt-1 text-xs'>
                      {new Date(entry.timestamp).toLocaleTimeString()}
                      {entry.response?.duration && (
                        <span className='ml-2'>
                          {entry.response.duration.toFixed(0)}ms
                        </span>
                      )}
                    </div>
                  </div>
                ))}
                {history.length === 0 && (
                  <p className='text-muted-foreground text-sm'>No history</p>
                )}
              </div>
            </ScrollArea>
          </div>
        </div>
      )}

      {/* Main Content */}
      <div className='flex flex-1 flex-col'>
        {/* Impersonation Banner */}
        <ImpersonationBanner />

        <div className='flex-1 space-y-6 p-6'>
          {/* Toolbar */}
          <div className='flex items-center gap-2'>
            {!showEndpointBrowser && (
              <Button
                variant='outline'
                size='sm'
                onClick={() => setShowEndpointBrowser(true)}
              >
                <List className='mr-2 h-4 w-4' />
                Show Endpoints
              </Button>
            )}
            <Button
              variant={showDocumentation ? 'default' : 'outline'}
              size='sm'
              onClick={() => setShowDocumentation(!showDocumentation)}
            >
              <BookOpen className='mr-2 h-4 w-4' />
              {showDocumentation ? 'Hide' : 'Show'} Documentation
            </Button>
            <div className='flex-1' />
            <ImpersonationPopover
              contextLabel='Executing as'
              defaultReason='REST API Explorer testing'
            />
          </div>

          {/* Request Builder */}
          <Card>
            <CardHeader>
              <CardTitle>Request Builder</CardTitle>
              <CardDescription>
                Build and test API requests against your Fluxbase backend
              </CardDescription>
            </CardHeader>
            <CardContent className='space-y-4'>
              {/* Method and Endpoint */}
              <div className='flex gap-2'>
                <Select
                  value={method}
                  onValueChange={(v) => {
                    const newMethod = v as HttpMethod
                    setMethod(newMethod)

                    // If we have a selected endpoint and OpenAPI spec, try to find the matching variant
                    if (selectedEndpoint && openAPISpec) {
                      // Get the base path (without {id})
                      const basePath = endpoint.replace(/\/\{[^}]+\}$/, '')

                      // Look for an endpoint with the new method
                      const pathMethods =
                        openAPISpec.paths[endpoint] ||
                        openAPISpec.paths[basePath] ||
                        {}

                      // Prefer endpoints with {id} for single-resource operations, without for collections
                      const preferWithId = ['PUT', 'PATCH', 'DELETE'].includes(
                        newMethod
                      )
                      let targetPath = endpoint

                      // Check if the new method exists on current path
                      if (pathMethods[newMethod.toLowerCase()]) {
                        targetPath = endpoint
                      } else if (preferWithId && !endpoint.includes('{id}')) {
                        // Try adding {id}
                        const pathWithId = `${basePath}/{id}`
                        if (
                          openAPISpec.paths[pathWithId]?.[
                            newMethod.toLowerCase()
                          ]
                        ) {
                          targetPath = pathWithId
                        }
                      } else if (!preferWithId && endpoint.includes('{id}')) {
                        // Try removing {id}
                        if (
                          openAPISpec.paths[basePath]?.[newMethod.toLowerCase()]
                        ) {
                          targetPath = basePath
                        }
                      }

                      // Update endpoint path if we found a different one
                      if (targetPath !== endpoint) {
                        setEndpoint(targetPath)
                      }

                      // Update selected endpoint to match new method and path
                      const operation =
                        openAPISpec.paths[targetPath]?.[newMethod.toLowerCase()]
                      if (operation) {
                        const newEndpointInfo: EndpointInfo = {
                          path: targetPath,
                          method: newMethod,
                          summary: operation.summary,
                          description: operation.description,
                          operationId: operation.operationId,
                          parameters: operation.parameters,
                          requestBody: operation.requestBody,
                          responses: operation.responses,
                        }
                        setSelectedEndpoint(newEndpointInfo)
                      }
                    }
                  }}
                >
                  <SelectTrigger className='w-32'>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {HTTP_METHODS.map((m) => (
                      <SelectItem key={m} value={m}>
                        {m}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <Input
                  value={endpoint}
                  onChange={(e) => setEndpoint(e.target.value)}
                  placeholder='/api/v1/tables/users'
                  className='flex-1'
                />
                <Button onClick={executeRequest} disabled={loading}>
                  <Play className='mr-2 h-4 w-4' />
                  {loading ? 'Sending...' : 'Send'}
                </Button>
                <Button variant='outline' onClick={saveRequest}>
                  <Save className='h-4 w-4' />
                </Button>
              </div>

              {/* Auth Token Checkbox */}
              <div className='flex items-center space-x-2'>
                <Checkbox
                  id='include-auth'
                  checked={includeAuthToken}
                  onCheckedChange={(checked) =>
                    setIncludeAuthToken(checked as boolean)
                  }
                />
                <Label
                  htmlFor='include-auth'
                  className='cursor-pointer text-sm font-normal'
                >
                  Include Authorization token
                </Label>
              </div>

              {/* Query Builder Toggle */}
              <div className='flex items-center gap-2'>
                <Button
                  variant='outline'
                  size='sm'
                  onClick={() => setShowQueryBuilder(!showQueryBuilder)}
                >
                  {showQueryBuilder ? (
                    <ChevronDown className='mr-2 h-4 w-4' />
                  ) : (
                    <ChevronRight className='mr-2 h-4 w-4' />
                  )}
                  Query Builder
                </Button>
                <div className='flex gap-1'>
                  <Button
                    variant='ghost'
                    size='sm'
                    onClick={() => generateCode('curl')}
                  >
                    cURL
                  </Button>
                  <Button
                    variant='ghost'
                    size='sm'
                    onClick={() => generateCode('javascript')}
                  >
                    JS
                  </Button>
                  <Button
                    variant='ghost'
                    size='sm'
                    onClick={() => generateCode('typescript')}
                  >
                    TS
                  </Button>
                  <Button
                    variant='ghost'
                    size='sm'
                    onClick={() => generateCode('python')}
                  >
                    Python
                  </Button>
                </div>
              </div>

              {/* Query Builder */}
              {showQueryBuilder && (
                <Card className='bg-muted/20 space-y-4 p-4'>
                  <div className='space-y-2'>
                    <h4 className='text-sm font-semibold'>Query Parameters</h4>
                    <div className='space-y-2'>
                      {Object.entries(queryParams).map(([key, value]) => (
                        <div key={key} className='flex gap-2'>
                          <Input
                            value={key}
                            onChange={(e) => {
                              const newParams = { ...queryParams }
                              delete newParams[key]
                              newParams[e.target.value] = value
                              setQueryParams(newParams)
                            }}
                            placeholder='Parameter'
                            className='flex-1'
                          />
                          <Input
                            value={value}
                            onChange={(e) =>
                              setQueryParams((prev) => ({
                                ...prev,
                                [key]: e.target.value,
                              }))
                            }
                            placeholder='Value'
                            className='flex-1'
                          />
                          <Button
                            variant='ghost'
                            size='icon'
                            onClick={() => removeQueryParam(key)}
                          >
                            <X className='h-4 w-4' />
                          </Button>
                        </div>
                      ))}
                      <Button
                        variant='outline'
                        size='sm'
                        onClick={addQueryParam}
                      >
                        <Plus className='mr-2 h-4 w-4' />
                        Add Parameter
                      </Button>
                    </div>
                  </div>

                  <div className='text-muted-foreground text-xs'>
                    <strong>Common filters:</strong> select, order, limit,
                    offset, column.eq.value, column.like.pattern,
                    column.in.(1,2,3)
                  </div>
                </Card>
              )}

              {/* Tabs for Headers and Body */}
              <Tabs defaultValue='headers'>
                <TabsList>
                  <TabsTrigger value='headers'>Headers</TabsTrigger>
                  {['POST', 'PUT', 'PATCH'].includes(method) && (
                    <TabsTrigger value='body'>Body</TabsTrigger>
                  )}
                </TabsList>

                <TabsContent value='headers' className='space-y-2'>
                  {Object.entries(headers).map(([key, value]) => (
                    <div key={key} className='flex gap-2'>
                      <Input
                        value={key}
                        onChange={(e) => {
                          const newHeaders = { ...headers }
                          delete newHeaders[key]
                          newHeaders[e.target.value] = value
                          setHeaders(newHeaders)
                        }}
                        placeholder='Header'
                        className='flex-1'
                      />
                      <Input
                        value={value}
                        onChange={(e) =>
                          setHeaders((prev) => ({
                            ...prev,
                            [key]: e.target.value,
                          }))
                        }
                        placeholder='Value'
                        className='flex-1'
                      />
                      <Button
                        variant='ghost'
                        size='icon'
                        onClick={() => removeHeader(key)}
                      >
                        <X className='h-4 w-4' />
                      </Button>
                    </div>
                  ))}
                  <Button variant='outline' size='sm' onClick={addHeader}>
                    <Plus className='mr-2 h-4 w-4' />
                    Add Header
                  </Button>
                </TabsContent>

                {['POST', 'PUT', 'PATCH'].includes(method) && (
                  <TabsContent value='body'>
                    <Textarea
                      value={body}
                      onChange={(e) => setBody(e.target.value)}
                      placeholder='{"name": "John Doe", "email": "john@example.com"}'
                      className='font-mono text-sm'
                      rows={10}
                    />
                    <div className='mt-2'>
                      <Button
                        variant='outline'
                        size='sm'
                        onClick={() => {
                          try {
                            setBody(JSON.stringify(JSON.parse(body), null, 2))
                            toast.success('JSON formatted')
                          } catch {
                            toast.error('Invalid JSON')
                          }
                        }}
                      >
                        Format JSON
                      </Button>
                    </div>
                  </TabsContent>
                )}
              </Tabs>
            </CardContent>
          </Card>

          {/* Response */}
          {response && (
            <Card>
              <CardHeader>
                <CardTitle>Response</CardTitle>
                <div className='flex items-center gap-2'>
                  <Badge
                    variant={response.status < 400 ? 'default' : 'destructive'}
                  >
                    {response.status} {response.statusText}
                  </Badge>
                  <span className='text-muted-foreground text-sm'>
                    {response.duration.toFixed(0)}ms
                  </span>
                </div>
              </CardHeader>
              <CardContent>
                <Tabs value={activeTab} onValueChange={setActiveTab}>
                  <TabsList>
                    <TabsTrigger value='body'>Body</TabsTrigger>
                    <TabsTrigger value='headers'>Headers</TabsTrigger>
                    <TabsTrigger value='preview'>Preview</TabsTrigger>
                  </TabsList>

                  <TabsContent value='body'>
                    <ScrollArea className='h-96'>
                      <pre className='text-sm'>
                        {typeof response.data === 'object' &&
                        response.data !== null
                          ? JSON.stringify(response.data, null, 2)
                          : String(response.data ?? '')}
                      </pre>
                    </ScrollArea>
                  </TabsContent>

                  <TabsContent value='headers'>
                    <div className='space-y-1'>
                      {Object.entries(response.headers).map(([key, value]) => (
                        <div key={key} className='flex gap-2 text-sm'>
                          <span className='font-semibold'>{key}:</span>
                          <span className='text-muted-foreground'>{value}</span>
                        </div>
                      ))}
                    </div>
                  </TabsContent>

                  <TabsContent value='preview'>
                    {Array.isArray(response.data) ? (
                      <div className='rounded border'>
                        <table className='w-full text-sm'>
                          <thead className='bg-muted/50'>
                            <tr>
                              {response.data[0] &&
                                Object.keys(response.data[0]).map((key) => (
                                  <th
                                    key={key}
                                    className='border-b p-2 text-left'
                                  >
                                    {key}
                                  </th>
                                ))}
                            </tr>
                          </thead>
                          <tbody>
                            {response.data.slice(0, 10).map((row, i) => (
                              <tr key={i} className='border-b'>
                                {Object.values(row).map((value: unknown, j) => (
                                  <td key={j} className='p-2'>
                                    {typeof value === 'object'
                                      ? JSON.stringify(value)
                                      : String(value)}
                                  </td>
                                ))}
                              </tr>
                            ))}
                          </tbody>
                        </table>
                        {response.data.length > 10 && (
                          <div className='text-muted-foreground p-2 text-center text-sm'>
                            Showing 10 of {response.data.length} rows
                          </div>
                        )}
                      </div>
                    ) : (
                      <ScrollArea className='h-96'>
                        <pre className='text-sm'>
                          {typeof response.data === 'object' &&
                          response.data !== null
                            ? JSON.stringify(response.data, null, 2)
                            : String(response.data ?? '')}
                        </pre>
                      </ScrollArea>
                    )}
                  </TabsContent>
                </Tabs>
              </CardContent>
            </Card>
          )}

          {/* Documentation Panel */}
          {showDocumentation && (
            <Card>
              <CardHeader>
                <CardTitle className='flex items-center gap-2'>
                  <BookOpen className='h-5 w-5' />
                  API Documentation
                </CardTitle>
                <CardDescription>
                  {selectedEndpoint
                    ? `Documentation for ${selectedEndpoint.method} ${selectedEndpoint.path}`
                    : 'Select an endpoint from the browser to view its documentation'}
                </CardDescription>
              </CardHeader>
              <CardContent>
                <DocumentationPanel endpoint={selectedEndpoint} />
              </CardContent>
            </Card>
          )}
        </div>
      </div>

      {/* Clear History Confirmation */}
      <ConfirmDialog
        open={showClearHistoryConfirm}
        onOpenChange={setShowClearHistoryConfirm}
        title='Clear History'
        desc='Are you sure you want to clear all request history? This action cannot be undone.'
        confirmText='Clear'
        destructive
        handleConfirm={handleClearHistoryConfirm}
      />

      {/* Add Header Prompt (2-step) */}
      <PromptDialog
        open={showHeaderPrompt}
        onOpenChange={(open) => {
          if (!open) {
            setShowHeaderPrompt(false)
            setPendingHeaderName('')
            setHeaderPromptStep('name')
          }
        }}
        title={
          headerPromptStep === 'name'
            ? 'Add Header'
            : `Header: ${pendingHeaderName}`
        }
        description={
          headerPromptStep === 'name'
            ? 'Enter the header name'
            : 'Enter the header value'
        }
        placeholder={
          headerPromptStep === 'name'
            ? 'e.g., Authorization'
            : 'e.g., Bearer token'
        }
        confirmText={headerPromptStep === 'name' ? 'Next' : 'Add Header'}
        onConfirm={handleHeaderPromptConfirm}
      />

      {/* Add Parameter Prompt (2-step) */}
      <PromptDialog
        open={showParamPrompt}
        onOpenChange={(open) => {
          if (!open) {
            setShowParamPrompt(false)
            setPendingParamName('')
            setParamPromptStep('name')
          }
        }}
        title={
          paramPromptStep === 'name'
            ? 'Add Parameter'
            : `Parameter: ${pendingParamName}`
        }
        description={
          paramPromptStep === 'name'
            ? 'Enter the parameter name'
            : 'Enter the parameter value'
        }
        placeholder={paramPromptStep === 'name' ? 'e.g., limit' : 'e.g., 10'}
        confirmText={paramPromptStep === 'name' ? 'Next' : 'Add Parameter'}
        onConfirm={handleParamPromptConfirm}
      />

      {/* Save Request Prompt */}
      <PromptDialog
        open={showSaveRequestPrompt}
        onOpenChange={setShowSaveRequestPrompt}
        title='Save Request'
        description='Enter a name for this request'
        placeholder='e.g., Get all users'
        confirmText='Save'
        onConfirm={handleSaveRequestConfirm}
        validation={(value) => (value.trim() ? null : 'Name is required')}
      />
    </div>
  )
}
