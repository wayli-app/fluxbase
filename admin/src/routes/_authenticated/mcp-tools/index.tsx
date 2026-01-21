import { useState, useEffect, useCallback } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import {
  Plus,
  Edit,
  Trash2,
  Play,
  RefreshCw,
  Search,
  CheckCircle,
  XCircle,
  Loader2,
  Wrench,
  FileText,
  Settings,
  FolderOpen,
  Power,
} from 'lucide-react'
import { toast } from 'sonner'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardContent } from '@/components/ui/card'
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
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import {
  mcpToolsApi,
  mcpResourcesApi,
  mcpConfigApi,
  type MCPTool,
  type MCPResource,
  type MCPConfig,
} from '@/lib/api'

export const Route = createFileRoute('/_authenticated/mcp-tools/')({
  component: MCPToolsPage,
})

function MCPToolsPage() {
  const [config, setConfig] = useState<MCPConfig | null>(null)

  useEffect(() => {
    mcpConfigApi.get().then(setConfig).catch(() => {
      // Config fetch is optional - silently ignore errors
    })
  }, [])

  return (
    <div className="flex flex-1 flex-col gap-6 p-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Custom MCP Tools</h1>
          <p className="text-muted-foreground">
            Create and manage custom MCP tools and resources for AI assistants
          </p>
        </div>
      </div>

      {config && (
        <Card>
          <CardHeader className="pb-3">
            <div className="flex items-center gap-2">
              <Settings className="h-4 w-4 text-muted-foreground" />
              <span className="text-sm font-medium">Configuration</span>
            </div>
          </CardHeader>
          <CardContent>
            <div className="flex flex-wrap gap-6 text-sm">
              <div className="flex items-center gap-2">
                <FolderOpen className="h-4 w-4 text-muted-foreground" />
                <span className="text-muted-foreground">Tools Directory:</span>
                <code className="rounded bg-muted px-1.5 py-0.5 text-xs">
                  {config.tools_dir}
                </code>
              </div>
              <div className="flex items-center gap-2">
                <Power className="h-4 w-4 text-muted-foreground" />
                <span className="text-muted-foreground">Auto-load on Boot:</span>
                <Badge variant={config.auto_load_on_boot ? 'default' : 'secondary'}>
                  {config.auto_load_on_boot ? 'Enabled' : 'Disabled'}
                </Badge>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      <Tabs defaultValue="tools" className="flex-1">
        <TabsList>
          <TabsTrigger value="tools" className="gap-2">
            <Wrench className="h-4 w-4" />
            Tools
          </TabsTrigger>
          <TabsTrigger value="resources" className="gap-2">
            <FileText className="h-4 w-4" />
            Resources
          </TabsTrigger>
        </TabsList>

        <TabsContent value="tools" className="mt-4">
          <ToolsTab />
        </TabsContent>

        <TabsContent value="resources" className="mt-4">
          <ResourcesTab />
        </TabsContent>
      </Tabs>
    </div>
  )
}

// Tools Tab Component
function ToolsTab() {
  const [tools, setTools] = useState<MCPTool[]>([])
  const [loading, setLoading] = useState(true)
  const [searchQuery, setSearchQuery] = useState('')
  const [showCreateDialog, setShowCreateDialog] = useState(false)
  const [showEditDialog, setShowEditDialog] = useState(false)
  const [showTestDialog, setShowTestDialog] = useState(false)
  const [selectedTool, setSelectedTool] = useState<MCPTool | null>(null)
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null)

  const fetchTools = useCallback(async () => {
    try {
      setLoading(true)
      const data = await mcpToolsApi.list()
      setTools(data)
    } catch {
      toast.error('Failed to load MCP tools')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchTools()
  }, [fetchTools])

  const handleDelete = async (id: string) => {
    try {
      await mcpToolsApi.delete(id)
      toast.success('Tool deleted')
      fetchTools()
    } catch {
      toast.error('Failed to delete tool')
    }
    setDeleteConfirm(null)
  }

  const handleToggleEnabled = async (tool: MCPTool) => {
    try {
      await mcpToolsApi.update(tool.id, { enabled: !tool.enabled })
      toast.success(tool.enabled ? 'Tool disabled' : 'Tool enabled')
      fetchTools()
    } catch {
      toast.error('Failed to update tool')
    }
  }

  const filteredTools = tools.filter(
    (tool) =>
      tool.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      tool.description?.toLowerCase().includes(searchQuery.toLowerCase())
  )

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-4">
        <div className="flex items-center gap-4">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              placeholder="Search tools..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-64 pl-9"
            />
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={fetchTools}>
            <RefreshCw className="mr-2 h-4 w-4" />
            Refresh
          </Button>
          <Button size="sm" onClick={() => setShowCreateDialog(true)}>
            <Plus className="mr-2 h-4 w-4" />
            New Tool
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        {loading ? (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
          </div>
        ) : filteredTools.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
            <Wrench className="mb-2 h-12 w-12" />
            <p>No custom MCP tools found</p>
            <p className="text-sm">Create your first tool to get started</p>
          </div>
        ) : (
          <div className="space-y-2">
            {filteredTools.map((tool) => (
              <div
                key={tool.id}
                className="flex items-center justify-between rounded-lg border p-4"
              >
                <div className="flex-1">
                  <div className="flex items-center gap-2">
                    <span className="font-medium">{tool.name}</span>
                    {tool.namespace !== 'default' && (
                      <Badge variant="outline">{tool.namespace}</Badge>
                    )}
                    {tool.enabled ? (
                      <Badge variant="default" className="bg-green-600">
                        <CheckCircle className="mr-1 h-3 w-3" />
                        Enabled
                      </Badge>
                    ) : (
                      <Badge variant="secondary">
                        <XCircle className="mr-1 h-3 w-3" />
                        Disabled
                      </Badge>
                    )}
                  </div>
                  {tool.description && (
                    <p className="mt-1 text-sm text-muted-foreground">
                      {tool.description}
                    </p>
                  )}
                  <div className="mt-2 flex gap-2 text-xs text-muted-foreground">
                    <span>Timeout: {tool.timeout_seconds}s</span>
                    <span>•</span>
                    <span>Memory: {tool.memory_limit_mb}MB</span>
                    {tool.allow_net && (
                      <>
                        <span>•</span>
                        <span>Network: Yes</span>
                      </>
                    )}
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => {
                          setSelectedTool(tool)
                          setShowTestDialog(true)
                        }}
                      >
                        <Play className="h-4 w-4" />
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>Test Tool</TooltipContent>
                  </Tooltip>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => {
                          setSelectedTool(tool)
                          setShowEditDialog(true)
                        }}
                      >
                        <Edit className="h-4 w-4" />
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>Edit</TooltipContent>
                  </Tooltip>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => handleToggleEnabled(tool)}
                      >
                        <Switch checked={tool.enabled} />
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>
                      {tool.enabled ? 'Disable' : 'Enable'}
                    </TooltipContent>
                  </Tooltip>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setDeleteConfirm(tool.id)}
                      >
                        <Trash2 className="h-4 w-4 text-destructive" />
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>Delete</TooltipContent>
                  </Tooltip>
                </div>
              </div>
            ))}
          </div>
        )}
      </CardContent>

      {/* Create Dialog */}
      <ToolDialog
        open={showCreateDialog}
        onOpenChange={setShowCreateDialog}
        onSave={async (data) => {
          await mcpToolsApi.create(data)
          toast.success('Tool created')
          fetchTools()
          setShowCreateDialog(false)
        }}
      />

      {/* Edit Dialog */}
      {selectedTool && (
        <ToolDialog
          open={showEditDialog}
          onOpenChange={setShowEditDialog}
          tool={selectedTool}
          onSave={async (data) => {
            await mcpToolsApi.update(selectedTool.id, data)
            toast.success('Tool updated')
            fetchTools()
            setShowEditDialog(false)
          }}
        />
      )}

      {/* Test Dialog */}
      {selectedTool && (
        <TestToolDialog
          open={showTestDialog}
          onOpenChange={setShowTestDialog}
          tool={selectedTool}
        />
      )}

      {/* Delete Confirmation */}
      <AlertDialog
        open={deleteConfirm !== null}
        onOpenChange={() => setDeleteConfirm(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Tool</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete this tool? This action cannot be
              undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => deleteConfirm && handleDelete(deleteConfirm)}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </Card>
  )
}

// Tool Dialog Component
interface ToolDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  tool?: MCPTool
  onSave: (data: {
    name: string
    namespace: string
    description: string
    code: string
    timeout_seconds: number
    memory_limit_mb: number
    allow_net: boolean
    allow_env: boolean
    enabled: boolean
  }) => Promise<void>
}

function ToolDialog({ open, onOpenChange, tool, onSave }: ToolDialogProps) {
  const [name, setName] = useState(tool?.name || '')
  const [namespace, setNamespace] = useState(tool?.namespace || 'default')
  const [description, setDescription] = useState(tool?.description || '')
  const [code, setCode] = useState(
    tool?.code ||
      `// @fluxbase:description Your tool description here

export async function handler(
  args: { param1: string },
  fluxbase,
  fluxbaseService,
  utils
) {
  // Your tool implementation here
  return {
    content: [{ type: "text", text: "Result: " + args.param1 }]
  };
}`
  )
  const [timeoutSeconds, setTimeoutSeconds] = useState(tool?.timeout_seconds || 30)
  const [memoryLimitMb, setMemoryLimitMb] = useState(tool?.memory_limit_mb || 128)
  const [allowNet, setAllowNet] = useState(tool?.allow_net || false)
  const [allowEnv, setAllowEnv] = useState(tool?.allow_env || false)
  const [enabled, setEnabled] = useState(tool?.enabled ?? true)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (tool) {
      setName(tool.name)
      setNamespace(tool.namespace)
      setDescription(tool.description || '')
      setCode(tool.code)
      setTimeoutSeconds(tool.timeout_seconds)
      setMemoryLimitMb(tool.memory_limit_mb)
      setAllowNet(tool.allow_net)
      setAllowEnv(tool.allow_env)
      setEnabled(tool.enabled)
    }
  }, [tool])

  const handleSave = async () => {
    if (!name.trim()) {
      toast.error('Name is required')
      return
    }
    if (!code.trim()) {
      toast.error('Code is required')
      return
    }

    try {
      setSaving(true)
      await onSave({
        name,
        namespace,
        description,
        code,
        timeout_seconds: timeoutSeconds,
        memory_limit_mb: memoryLimitMb,
        allow_net: allowNet,
        allow_env: allowEnv,
        enabled,
      })
    } catch {
      toast.error('Failed to save tool')
    } finally {
      setSaving(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl max-h-[90vh] overflow-hidden flex flex-col">
        <DialogHeader>
          <DialogTitle>{tool ? 'Edit Tool' : 'Create Tool'}</DialogTitle>
          <DialogDescription>
            {tool
              ? 'Update your custom MCP tool'
              : 'Create a new custom MCP tool for AI assistants'}
          </DialogDescription>
        </DialogHeader>

        <ScrollArea className="flex-1 pr-4">
          <div className="space-y-4 py-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="name">Name</Label>
                <Input
                  id="name"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="my_tool"
                  disabled={!!tool}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="namespace">Namespace</Label>
                <Input
                  id="namespace"
                  value={namespace}
                  onChange={(e) => setNamespace(e.target.value)}
                  placeholder="default"
                />
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="description">Description</Label>
              <Input
                id="description"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="What does this tool do?"
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="code">Code</Label>
              <Textarea
                id="code"
                value={code}
                onChange={(e) => setCode(e.target.value)}
                className="font-mono text-sm min-h-[300px]"
                placeholder="export async function handler(args, fluxbase, fluxbaseService, utils) { ... }"
              />
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="timeout">Timeout (seconds)</Label>
                <Input
                  id="timeout"
                  type="number"
                  value={timeoutSeconds}
                  onChange={(e) => setTimeoutSeconds(parseInt(e.target.value) || 30)}
                  min={1}
                  max={300}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="memory">Memory Limit (MB)</Label>
                <Input
                  id="memory"
                  type="number"
                  value={memoryLimitMb}
                  onChange={(e) => setMemoryLimitMb(parseInt(e.target.value) || 128)}
                  min={32}
                  max={1024}
                />
              </div>
            </div>

            <div className="flex flex-wrap gap-6">
              <div className="flex items-center gap-2">
                <Switch
                  id="allow_net"
                  checked={allowNet}
                  onCheckedChange={setAllowNet}
                />
                <Label htmlFor="allow_net">Allow Network</Label>
              </div>
              <div className="flex items-center gap-2">
                <Switch
                  id="allow_env"
                  checked={allowEnv}
                  onCheckedChange={setAllowEnv}
                />
                <Label htmlFor="allow_env">Allow Environment</Label>
              </div>
              <div className="flex items-center gap-2">
                <Switch
                  id="enabled"
                  checked={enabled}
                  onCheckedChange={setEnabled}
                />
                <Label htmlFor="enabled">Enabled</Label>
              </div>
            </div>
          </div>
        </ScrollArea>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={handleSave} disabled={saving}>
            {saving && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            {tool ? 'Update' : 'Create'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// Test Tool Dialog
interface TestToolDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  tool: MCPTool
}

function TestToolDialog({ open, onOpenChange, tool }: TestToolDialogProps) {
  const [args, setArgs] = useState('{}')
  const [testing, setTesting] = useState(false)
  const [result, setResult] = useState<{
    success: boolean
    result: { content: Array<{ type: string; text: string }>; isError?: boolean }
  } | null>(null)

  const handleTest = async () => {
    try {
      setTesting(true)
      const parsedArgs = JSON.parse(args)
      const testResult = await mcpToolsApi.test(tool.id, parsedArgs)
      setResult(testResult)
    } catch (error) {
      if (error instanceof SyntaxError) {
        toast.error('Invalid JSON arguments')
      } else {
        toast.error('Test failed')
      }
    } finally {
      setTesting(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>Test Tool: {tool.name}</DialogTitle>
          <DialogDescription>
            Test your tool with sample arguments
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="args">Arguments (JSON)</Label>
            <Textarea
              id="args"
              value={args}
              onChange={(e) => setArgs(e.target.value)}
              className="font-mono text-sm min-h-[100px]"
              placeholder='{"param1": "value1"}'
            />
          </div>

          {result && (
            <div className="space-y-2">
              <Label>Result</Label>
              <div
                className={`rounded-lg border p-4 ${
                  result.success && !result.result.isError
                    ? 'border-green-500 bg-green-50 dark:bg-green-950'
                    : 'border-red-500 bg-red-50 dark:bg-red-950'
                }`}
              >
                <pre className="whitespace-pre-wrap text-sm">
                  {result.result.content.map((c) => c.text).join('\n')}
                </pre>
              </div>
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Close
          </Button>
          <Button onClick={handleTest} disabled={testing}>
            {testing && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            <Play className="mr-2 h-4 w-4" />
            Test
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// Resources Tab Component
function ResourcesTab() {
  const [resources, setResources] = useState<MCPResource[]>([])
  const [loading, setLoading] = useState(true)
  const [searchQuery, setSearchQuery] = useState('')
  const [showCreateDialog, setShowCreateDialog] = useState(false)
  const [showEditDialog, setShowEditDialog] = useState(false)
  const [selectedResource, setSelectedResource] = useState<MCPResource | null>(
    null
  )
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null)

  const fetchResources = useCallback(async () => {
    try {
      setLoading(true)
      const data = await mcpResourcesApi.list()
      setResources(data)
    } catch {
      toast.error('Failed to load MCP resources')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchResources()
  }, [fetchResources])

  const handleDelete = async (id: string) => {
    try {
      await mcpResourcesApi.delete(id)
      toast.success('Resource deleted')
      fetchResources()
    } catch {
      toast.error('Failed to delete resource')
    }
    setDeleteConfirm(null)
  }

  const handleToggleEnabled = async (resource: MCPResource) => {
    try {
      await mcpResourcesApi.update(resource.id, { enabled: !resource.enabled })
      toast.success(resource.enabled ? 'Resource disabled' : 'Resource enabled')
      fetchResources()
    } catch {
      toast.error('Failed to update resource')
    }
  }

  const filteredResources = resources.filter(
    (resource) =>
      resource.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      resource.uri.toLowerCase().includes(searchQuery.toLowerCase()) ||
      resource.description?.toLowerCase().includes(searchQuery.toLowerCase())
  )

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-4">
        <div className="flex items-center gap-4">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              placeholder="Search resources..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-64 pl-9"
            />
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={fetchResources}>
            <RefreshCw className="mr-2 h-4 w-4" />
            Refresh
          </Button>
          <Button size="sm" onClick={() => setShowCreateDialog(true)}>
            <Plus className="mr-2 h-4 w-4" />
            New Resource
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        {loading ? (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
          </div>
        ) : filteredResources.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
            <FileText className="mb-2 h-12 w-12" />
            <p>No custom MCP resources found</p>
            <p className="text-sm">Create your first resource to get started</p>
          </div>
        ) : (
          <div className="space-y-2">
            {filteredResources.map((resource) => (
              <div
                key={resource.id}
                className="flex items-center justify-between rounded-lg border p-4"
              >
                <div className="flex-1">
                  <div className="flex items-center gap-2">
                    <span className="font-medium">{resource.name}</span>
                    {resource.namespace !== 'default' && (
                      <Badge variant="outline">{resource.namespace}</Badge>
                    )}
                    {resource.is_template && (
                      <Badge variant="secondary">Template</Badge>
                    )}
                    {resource.enabled ? (
                      <Badge variant="default" className="bg-green-600">
                        <CheckCircle className="mr-1 h-3 w-3" />
                        Enabled
                      </Badge>
                    ) : (
                      <Badge variant="secondary">
                        <XCircle className="mr-1 h-3 w-3" />
                        Disabled
                      </Badge>
                    )}
                  </div>
                  <p className="mt-1 text-sm text-muted-foreground font-mono">
                    {resource.uri}
                  </p>
                  {resource.description && (
                    <p className="mt-1 text-sm text-muted-foreground">
                      {resource.description}
                    </p>
                  )}
                </div>
                <div className="flex items-center gap-2">
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => {
                          setSelectedResource(resource)
                          setShowEditDialog(true)
                        }}
                      >
                        <Edit className="h-4 w-4" />
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>Edit</TooltipContent>
                  </Tooltip>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => handleToggleEnabled(resource)}
                      >
                        <Switch checked={resource.enabled} />
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>
                      {resource.enabled ? 'Disable' : 'Enable'}
                    </TooltipContent>
                  </Tooltip>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setDeleteConfirm(resource.id)}
                      >
                        <Trash2 className="h-4 w-4 text-destructive" />
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>Delete</TooltipContent>
                  </Tooltip>
                </div>
              </div>
            ))}
          </div>
        )}
      </CardContent>

      {/* Create Dialog */}
      <ResourceDialog
        open={showCreateDialog}
        onOpenChange={setShowCreateDialog}
        onSave={async (data) => {
          await mcpResourcesApi.create(data)
          toast.success('Resource created')
          fetchResources()
          setShowCreateDialog(false)
        }}
      />

      {/* Edit Dialog */}
      {selectedResource && (
        <ResourceDialog
          open={showEditDialog}
          onOpenChange={setShowEditDialog}
          resource={selectedResource}
          onSave={async (data) => {
            await mcpResourcesApi.update(selectedResource.id, data)
            toast.success('Resource updated')
            fetchResources()
            setShowEditDialog(false)
          }}
        />
      )}

      {/* Delete Confirmation */}
      <AlertDialog
        open={deleteConfirm !== null}
        onOpenChange={() => setDeleteConfirm(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Resource</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete this resource? This action cannot
              be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => deleteConfirm && handleDelete(deleteConfirm)}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </Card>
  )
}

// Resource Dialog Component
interface ResourceDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  resource?: MCPResource
  onSave: (data: {
    uri: string
    name: string
    namespace: string
    description: string
    mime_type: string
    code: string
    timeout_seconds: number
    memory_limit_mb: number
    allow_net: boolean
    allow_env: boolean
    enabled: boolean
  }) => Promise<void>
}

function ResourceDialog({
  open,
  onOpenChange,
  resource,
  onSave,
}: ResourceDialogProps) {
  const [uri, setUri] = useState(resource?.uri || 'fluxbase://custom/')
  const [name, setName] = useState(resource?.name || '')
  const [namespace, setNamespace] = useState(resource?.namespace || 'default')
  const [description, setDescription] = useState(resource?.description || '')
  const [mimeType, setMimeType] = useState(resource?.mime_type || 'text/plain')
  const [code, setCode] = useState(
    resource?.code ||
      `// @fluxbase:description Your resource description here

export async function handler(
  params: {},
  fluxbase,
  fluxbaseService,
  utils
) {
  // Your resource implementation here
  return [
    { type: "text", text: JSON.stringify({ data: "example" }) }
  ];
}`
  )
  const [timeoutSeconds, setTimeoutSeconds] = useState(
    resource?.timeout_seconds || 30
  )
  const [memoryLimitMb, setMemoryLimitMb] = useState(
    resource?.memory_limit_mb || 128
  )
  const [allowNet, setAllowNet] = useState(resource?.allow_net || false)
  const [allowEnv, setAllowEnv] = useState(resource?.allow_env || false)
  const [enabled, setEnabled] = useState(resource?.enabled ?? true)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (resource) {
      setUri(resource.uri)
      setName(resource.name)
      setNamespace(resource.namespace)
      setDescription(resource.description || '')
      setMimeType(resource.mime_type)
      setCode(resource.code)
      setTimeoutSeconds(resource.timeout_seconds)
      setMemoryLimitMb(resource.memory_limit_mb)
      setAllowNet(resource.allow_net)
      setAllowEnv(resource.allow_env)
      setEnabled(resource.enabled)
    }
  }, [resource])

  const handleSave = async () => {
    if (!uri.trim()) {
      toast.error('URI is required')
      return
    }
    if (!name.trim()) {
      toast.error('Name is required')
      return
    }
    if (!code.trim()) {
      toast.error('Code is required')
      return
    }

    try {
      setSaving(true)
      await onSave({
        uri,
        name,
        namespace,
        description,
        mime_type: mimeType,
        code,
        timeout_seconds: timeoutSeconds,
        memory_limit_mb: memoryLimitMb,
        allow_net: allowNet,
        allow_env: allowEnv,
        enabled,
      })
    } catch {
      toast.error('Failed to save resource')
    } finally {
      setSaving(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl max-h-[90vh] overflow-hidden flex flex-col">
        <DialogHeader>
          <DialogTitle>
            {resource ? 'Edit Resource' : 'Create Resource'}
          </DialogTitle>
          <DialogDescription>
            {resource
              ? 'Update your custom MCP resource'
              : 'Create a new custom MCP resource for AI assistants'}
          </DialogDescription>
        </DialogHeader>

        <ScrollArea className="flex-1 pr-4">
          <div className="space-y-4 py-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="uri">URI</Label>
                <Input
                  id="uri"
                  value={uri}
                  onChange={(e) => setUri(e.target.value)}
                  placeholder="fluxbase://custom/my-resource"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="name">Name</Label>
                <Input
                  id="name"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="my_resource"
                />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="namespace">Namespace</Label>
                <Input
                  id="namespace"
                  value={namespace}
                  onChange={(e) => setNamespace(e.target.value)}
                  placeholder="default"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="mime_type">MIME Type</Label>
                <Input
                  id="mime_type"
                  value={mimeType}
                  onChange={(e) => setMimeType(e.target.value)}
                  placeholder="text/plain"
                />
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="description">Description</Label>
              <Input
                id="description"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="What does this resource provide?"
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="code">Code</Label>
              <Textarea
                id="code"
                value={code}
                onChange={(e) => setCode(e.target.value)}
                className="font-mono text-sm min-h-[300px]"
                placeholder="export async function handler(params, fluxbase, fluxbaseService, utils) { ... }"
              />
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="timeout">Timeout (seconds)</Label>
                <Input
                  id="timeout"
                  type="number"
                  value={timeoutSeconds}
                  onChange={(e) =>
                    setTimeoutSeconds(parseInt(e.target.value) || 30)
                  }
                  min={1}
                  max={300}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="memory">Memory Limit (MB)</Label>
                <Input
                  id="memory"
                  type="number"
                  value={memoryLimitMb}
                  onChange={(e) =>
                    setMemoryLimitMb(parseInt(e.target.value) || 128)
                  }
                  min={32}
                  max={1024}
                />
              </div>
            </div>

            <div className="flex flex-wrap gap-6">
              <div className="flex items-center gap-2">
                <Switch
                  id="allow_net"
                  checked={allowNet}
                  onCheckedChange={setAllowNet}
                />
                <Label htmlFor="allow_net">Allow Network</Label>
              </div>
              <div className="flex items-center gap-2">
                <Switch
                  id="allow_env"
                  checked={allowEnv}
                  onCheckedChange={setAllowEnv}
                />
                <Label htmlFor="allow_env">Allow Environment</Label>
              </div>
              <div className="flex items-center gap-2">
                <Switch
                  id="enabled"
                  checked={enabled}
                  onCheckedChange={setEnabled}
                />
                <Label htmlFor="enabled">Enabled</Label>
              </div>
            </div>
          </div>
        </ScrollArea>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={handleSave} disabled={saving}>
            {saving && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            {resource ? 'Update' : 'Create'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
