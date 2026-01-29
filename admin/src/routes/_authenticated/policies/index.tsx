import { createFileRoute } from '@tanstack/react-router'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useState, useMemo } from 'react'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Switch } from '@/components/ui/switch'
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
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import {
  Shield,
  ShieldAlert,
  ShieldCheck,
  ShieldOff,
  Plus,
  Trash2,
  Loader2,
  AlertCircle,
  AlertTriangle,
  Search,
  ChevronDown,
  ChevronRight,
  FileCode,
  Info,
  CheckCircle2,
  XCircle,
  Eye,
  Edit,
} from 'lucide-react'
import {
  policyApi,
  type RLSPolicy,
  type TableRLSStatus,
  type SecurityWarning,
  type PolicyTemplate,
  type CreatePolicyRequest,
} from '@/lib/api'
import { toast } from 'sonner'
import { cn } from '@/lib/utils'

export const Route = createFileRoute('/_authenticated/policies/')({
  component: PoliciesPage,
})

function PoliciesPage() {
  const [searchQuery, setSearchQuery] = useState('')
  const [activeTab, setActiveTab] = useState('tables')
  const [selectedTable, setSelectedTable] = useState<{
    schema: string
    table: string
  } | null>(null)
  const [createDialogOpen, setCreateDialogOpen] = useState(false)
  const [deleteDialog, setDeleteDialog] = useState<{
    open: boolean
    policy: RLSPolicy | null
  }>({ open: false, policy: null })

  const queryClient = useQueryClient()

  // Fetch tables with RLS status (returns array directly)
  const { data: tablesData, isLoading: tablesLoading } = useQuery({
    queryKey: ['tables-rls'],
    queryFn: () => policyApi.getTablesWithRLS('public'),
  })

  // Fetch security warnings
  const { data: warningsData, isLoading: warningsLoading } = useQuery({
    queryKey: ['security-warnings'],
    queryFn: () => policyApi.getSecurityWarnings(),
  })

  // Fetch policy templates
  const { data: templates } = useQuery({
    queryKey: ['policy-templates'],
    queryFn: () => policyApi.getTemplates(),
  })

  // Fetch selected table details
  const { data: tableDetails, isLoading: detailsLoading } = useQuery({
    queryKey: ['table-rls-status', selectedTable],
    queryFn: () =>
      selectedTable
        ? policyApi.getTableRLSStatus(selectedTable.schema, selectedTable.table)
        : null,
    enabled: !!selectedTable,
  })

  // Toggle RLS mutation
  const toggleRLSMutation = useMutation({
    mutationFn: ({
      schema,
      table,
      enable,
      forceRLS,
    }: {
      schema: string
      table: string
      enable: boolean
      forceRLS?: boolean
    }) => policyApi.toggleTableRLS(schema, table, enable, forceRLS),
    onSuccess: (data, variables) => {
      queryClient.invalidateQueries({ queryKey: ['tables-rls'] })
      queryClient.invalidateQueries({ queryKey: ['table-rls-status'] })
      queryClient.invalidateQueries({ queryKey: ['security-warnings'] })
      toast.success(data.message)
    },
    onError: () => {
      toast.error('Failed to toggle RLS')
    },
  })

  // Create policy mutation
  const createPolicyMutation = useMutation({
    mutationFn: (data: CreatePolicyRequest) => policyApi.create(data),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['table-rls-status'] })
      queryClient.invalidateQueries({ queryKey: ['security-warnings'] })
      setCreateDialogOpen(false)
      toast.success(data.message)
    },
    onError: () => {
      toast.error('Failed to create policy')
    },
  })

  // Delete policy mutation
  const deletePolicyMutation = useMutation({
    mutationFn: ({
      schema,
      table,
      name,
    }: {
      schema: string
      table: string
      name: string
    }) => policyApi.delete(schema, table, name),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['table-rls-status'] })
      queryClient.invalidateQueries({ queryKey: ['security-warnings'] })
      setDeleteDialog({ open: false, policy: null })
      toast.success(data.message)
    },
    onError: () => {
      toast.error('Failed to delete policy')
    },
  })

  // Filter tables based on search
  const filteredTables = useMemo(() => {
    if (!tablesData) return []
    if (!searchQuery) return tablesData
    const query = searchQuery.toLowerCase()
    return tablesData.filter(
      (t) =>
        t.table.toLowerCase().includes(query) ||
        t.schema.toLowerCase().includes(query)
    )
  }, [tablesData, searchQuery])

  // Get severity badge variant
  const getSeverityVariant = (
    severity: string
  ): 'destructive' | 'secondary' | 'outline' | 'default' => {
    switch (severity) {
      case 'critical':
        return 'destructive'
      case 'high':
        return 'destructive'
      case 'medium':
        return 'secondary'
      default:
        return 'outline'
    }
  }

  return (
    <div className="flex flex-1 flex-col gap-6 p-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight flex items-center gap-2">
            <Shield className="h-8 w-8" />
            Row Level Security
          </h1>
          <p className="text-sm text-muted-foreground mt-2">
            Manage RLS policies and security settings for your tables
          </p>
        </div>
      </div>

      {/* Security Summary */}
      {warningsData && (
        <div className="grid grid-cols-4 gap-4">
          <Card
            className={cn(
              warningsData.summary.critical > 0 &&
                'border-red-500/50 bg-red-500/5'
            )}
          >
            <CardHeader className="pb-2">
              <CardDescription>Critical Issues</CardDescription>
              <CardTitle className="text-2xl flex items-center gap-2">
                <AlertCircle className="h-5 w-5 text-red-500" />
                {warningsData.summary.critical}
              </CardTitle>
            </CardHeader>
          </Card>
          <Card
            className={cn(
              warningsData.summary.high > 0 &&
                'border-orange-500/50 bg-orange-500/5'
            )}
          >
            <CardHeader className="pb-2">
              <CardDescription>High Priority</CardDescription>
              <CardTitle className="text-2xl flex items-center gap-2">
                <AlertTriangle className="h-5 w-5 text-orange-500" />
                {warningsData.summary.high}
              </CardTitle>
            </CardHeader>
          </Card>
          <Card>
            <CardHeader className="pb-2">
              <CardDescription>Medium Priority</CardDescription>
              <CardTitle className="text-2xl flex items-center gap-2">
                <Info className="h-5 w-5 text-yellow-500" />
                {warningsData.summary.medium}
              </CardTitle>
            </CardHeader>
          </Card>
          <Card>
            <CardHeader className="pb-2">
              <CardDescription>Tables with RLS</CardDescription>
              <CardTitle className="text-2xl flex items-center gap-2">
                <ShieldCheck className="h-5 w-5 text-green-500" />
                {tablesData?.filter((t) => t.rls_enabled).length || 0}/
                {tablesData?.length || 0}
              </CardTitle>
            </CardHeader>
          </Card>
        </div>
      )}

      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList>
          <TabsTrigger value="tables">Tables</TabsTrigger>
          <TabsTrigger value="warnings" className="gap-2">
            Security Warnings
            {warningsData && warningsData.summary.total > 0 && (
              <Badge variant="destructive" className="ml-1">
                {warningsData.summary.total}
              </Badge>
            )}
          </TabsTrigger>
          <TabsTrigger value="templates">Policy Templates</TabsTrigger>
        </TabsList>

        {/* Tables Tab */}
        <TabsContent value="tables" className="space-y-4">
          <div className="flex items-center gap-4">
            <div className="relative flex-1 max-w-sm">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="Search tables..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-9"
              />
            </div>
          </div>

          <div className="flex gap-6">
            {/* Tables List */}
            <Card className="flex-1">
              <CardHeader>
                <CardTitle>Tables</CardTitle>
                <CardDescription>
                  Click a table to view and manage its RLS policies
                </CardDescription>
              </CardHeader>
              <CardContent>
                {tablesLoading ? (
                  <div className="flex justify-center py-8">
                    <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                  </div>
                ) : (
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Table</TableHead>
                        <TableHead>Schema</TableHead>
                        <TableHead>RLS</TableHead>
                        <TableHead>Force RLS</TableHead>
                        <TableHead>Policies</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {filteredTables.map((table) => (
                        <TableRow
                          key={`${table.schema}.${table.table}`}
                          className={cn(
                            'cursor-pointer',
                            selectedTable?.schema === table.schema &&
                              selectedTable?.table === table.table &&
                              'bg-muted'
                          )}
                          onClick={() =>
                            setSelectedTable({
                              schema: table.schema,
                              table: table.table,
                            })
                          }
                        >
                          <TableCell className="font-medium">
                            {table.table}
                          </TableCell>
                          <TableCell>
                            <Badge variant="outline">{table.schema}</Badge>
                          </TableCell>
                          <TableCell>
                            <Switch
                              checked={table.rls_enabled}
                              onCheckedChange={(checked) =>
                                toggleRLSMutation.mutate({
                                  schema: table.schema,
                                  table: table.table,
                                  enable: checked,
                                })
                              }
                              onClick={(e) => e.stopPropagation()}
                            />
                          </TableCell>
                          <TableCell>
                            {table.rls_forced ? (
                              <CheckCircle2 className="h-4 w-4 text-green-500" />
                            ) : (
                              <XCircle className="h-4 w-4 text-muted-foreground" />
                            )}
                          </TableCell>
                          <TableCell>
                            <Badge
                              variant={
                                table.policy_count > 0 ? 'default' : 'secondary'
                              }
                            >
                              {table.policy_count}
                            </Badge>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                )}
              </CardContent>
            </Card>

            {/* Table Details Panel */}
            {selectedTable && (
              <Card className="w-[500px] shrink-0">
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <div>
                      <CardTitle className="flex items-center gap-2">
                        <Shield className="h-5 w-5" />
                        {selectedTable.table}
                      </CardTitle>
                      <CardDescription>
                        {selectedTable.schema}.{selectedTable.table}
                      </CardDescription>
                    </div>
                    <Button
                      size="sm"
                      onClick={() => setCreateDialogOpen(true)}
                    >
                      <Plus className="h-4 w-4 mr-2" />
                      Add Policy
                    </Button>
                  </div>
                </CardHeader>
                <CardContent className="space-y-4">
                  {detailsLoading ? (
                    <div className="flex justify-center py-8">
                      <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                    </div>
                  ) : tableDetails ? (
                    <>
                      {/* RLS Status */}
                      <div className="flex items-center justify-between p-3 rounded-lg bg-muted/50">
                        <div className="flex items-center gap-3">
                          {tableDetails.rls_enabled ? (
                            <ShieldCheck className="h-5 w-5 text-green-500" />
                          ) : (
                            <ShieldOff className="h-5 w-5 text-muted-foreground" />
                          )}
                          <div>
                            <div className="font-medium">
                              RLS {tableDetails.rls_enabled ? 'Enabled' : 'Disabled'}
                            </div>
                            <div className="text-sm text-muted-foreground">
                              Force RLS:{' '}
                              {tableDetails.rls_forced ? 'Yes' : 'No'}
                            </div>
                          </div>
                        </div>
                        <Switch
                          checked={tableDetails.rls_enabled}
                          onCheckedChange={(checked) =>
                            toggleRLSMutation.mutate({
                              schema: selectedTable.schema,
                              table: selectedTable.table,
                              enable: checked,
                            })
                          }
                        />
                      </div>

                      {/* Policies */}
                      <div>
                        <h4 className="text-sm font-medium mb-3">
                          Policies ({tableDetails.policies.length})
                        </h4>
                        {tableDetails.policies.length === 0 ? (
                          <div className="text-center py-8 text-muted-foreground">
                            <ShieldOff className="h-8 w-8 mx-auto mb-2" />
                            <p>No policies defined</p>
                            {tableDetails.rls_enabled && (
                              <p className="text-sm mt-1">
                                All access will be denied by default
                              </p>
                            )}
                          </div>
                        ) : (
                          <div className="space-y-2">
                            {tableDetails.policies.map((policy) => (
                              <PolicyCard
                                key={policy.policy_name}
                                policy={policy}
                                onDelete={() =>
                                  setDeleteDialog({ open: true, policy })
                                }
                              />
                            ))}
                          </div>
                        )}
                      </div>
                    </>
                  ) : null}
                </CardContent>
              </Card>
            )}
          </div>
        </TabsContent>

        {/* Security Warnings Tab */}
        <TabsContent value="warnings">
          <Card>
            <CardHeader>
              <CardTitle>Security Warnings</CardTitle>
              <CardDescription>
                Issues that may indicate security vulnerabilities in your RLS
                configuration
              </CardDescription>
            </CardHeader>
            <CardContent>
              {warningsLoading ? (
                <div className="flex justify-center py-8">
                  <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                </div>
              ) : warningsData?.warnings.length === 0 ? (
                <div className="text-center py-12 text-muted-foreground">
                  <ShieldCheck className="h-12 w-12 mx-auto mb-4 text-green-500" />
                  <h3 className="text-lg font-medium">No Security Issues Found</h3>
                  <p className="text-sm mt-1">
                    Your RLS configuration looks good
                  </p>
                </div>
              ) : (
                <div className="space-y-3">
                  {warningsData?.warnings.map((warning, index) => (
                    <WarningCard
                      key={`${warning.id}-${index}`}
                      warning={warning}
                      onNavigate={() => {
                        setSelectedTable({
                          schema: warning.schema,
                          table: warning.table,
                        })
                        setActiveTab('tables')
                      }}
                    />
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* Templates Tab */}
        <TabsContent value="templates">
          <Card>
            <CardHeader>
              <CardTitle>Policy Templates</CardTitle>
              <CardDescription>
                Common policy patterns you can use as starting points
              </CardDescription>
            </CardHeader>
            <CardContent>
              {templates?.length === 0 ? (
                <div className="text-center py-12 text-muted-foreground">
                  <FileCode className="h-12 w-12 mx-auto mb-4" />
                  <h3 className="text-lg font-medium">No Templates Available</h3>
                </div>
              ) : (
                <div className="grid gap-4 md:grid-cols-2">
                  {templates?.map((template) => (
                    <TemplateCard
                      key={template.id}
                      template={template}
                      onUse={() => {
                        if (selectedTable) {
                          setCreateDialogOpen(true)
                        } else {
                          toast.info('Please select a table first')
                        }
                      }}
                    />
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* Create Policy Dialog */}
      {selectedTable && (
        <CreatePolicyDialog
          open={createDialogOpen}
          onOpenChange={setCreateDialogOpen}
          schema={selectedTable.schema}
          table={selectedTable.table}
          templates={templates || []}
          onSubmit={(data) => createPolicyMutation.mutate(data)}
          isLoading={createPolicyMutation.isPending}
        />
      )}

      {/* Delete Policy Confirmation */}
      <AlertDialog
        open={deleteDialog.open}
        onOpenChange={(open) =>
          setDeleteDialog({ open, policy: open ? deleteDialog.policy : null })
        }
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Policy</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete the policy &quot;
              {deleteDialog.policy?.policy_name}&quot;? This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (deleteDialog.policy) {
                  deletePolicyMutation.mutate({
                    schema: deleteDialog.policy.schema,
                    table: deleteDialog.policy.table,
                    name: deleteDialog.policy.policy_name,
                  })
                }
              }}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {deletePolicyMutation.isPending ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                'Delete'
              )}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}

// Policy Card Component
function PolicyCard({
  policy,
  onDelete,
}: {
  policy: RLSPolicy
  onDelete: () => void
}) {
  const [expanded, setExpanded] = useState(false)
  const isPermissive = policy.permissive === 'PERMISSIVE'

  return (
    <Collapsible open={expanded} onOpenChange={setExpanded}>
      <div className="border rounded-lg">
        <CollapsibleTrigger className="flex items-center justify-between w-full p-3 hover:bg-muted/50">
          <div className="flex items-center gap-3">
            {expanded ? (
              <ChevronDown className="h-4 w-4" />
            ) : (
              <ChevronRight className="h-4 w-4" />
            )}
            <span className="font-medium">{policy.policy_name}</span>
            <Badge variant="outline">{policy.command}</Badge>
            <Badge variant={isPermissive ? 'default' : 'secondary'}>
              {policy.permissive}
            </Badge>
          </div>
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8"
            onClick={(e) => {
              e.stopPropagation()
              onDelete()
            }}
          >
            <Trash2 className="h-4 w-4 text-destructive" />
          </Button>
        </CollapsibleTrigger>
        <CollapsibleContent>
          <div className="px-3 pb-3 pt-0 space-y-3 border-t">
            <div className="pt-3">
              <Label className="text-xs text-muted-foreground">Roles</Label>
              <div className="flex gap-1 mt-1">
                {policy.roles.map((role) => (
                  <Badge key={role} variant="secondary">
                    {role}
                  </Badge>
                ))}
              </div>
            </div>
            {policy.using && (
              <div>
                <Label className="text-xs text-muted-foreground">
                  USING Expression
                </Label>
                <pre className="mt-1 p-2 bg-muted rounded text-xs overflow-auto">
                  {policy.using}
                </pre>
              </div>
            )}
            {policy.with_check && (
              <div>
                <Label className="text-xs text-muted-foreground">
                  WITH CHECK Expression
                </Label>
                <pre className="mt-1 p-2 bg-muted rounded text-xs overflow-auto">
                  {policy.with_check}
                </pre>
              </div>
            )}
          </div>
        </CollapsibleContent>
      </div>
    </Collapsible>
  )
}

// Warning Card Component
function WarningCard({
  warning,
  onNavigate,
}: {
  warning: SecurityWarning
  onNavigate: () => void
}) {
  const severityColors = {
    critical: 'border-red-500/50 bg-red-500/5',
    high: 'border-orange-500/50 bg-orange-500/5',
    medium: 'border-yellow-500/50 bg-yellow-500/5',
    low: 'border-blue-500/50 bg-blue-500/5',
  }

  const severityIcons = {
    critical: <AlertCircle className="h-5 w-5 text-red-500" />,
    high: <AlertTriangle className="h-5 w-5 text-orange-500" />,
    medium: <Info className="h-5 w-5 text-yellow-500" />,
    low: <Info className="h-5 w-5 text-blue-500" />,
  }

  return (
    <div
      className={cn(
        'border rounded-lg p-4 cursor-pointer hover:shadow-md transition-shadow',
        severityColors[warning.severity]
      )}
      onClick={onNavigate}
    >
      <div className="flex items-start gap-3">
        {severityIcons[warning.severity]}
        <div className="flex-1">
          <div className="flex items-center gap-2 flex-wrap">
            <Badge
              variant={
                warning.severity === 'critical' || warning.severity === 'high'
                  ? 'destructive'
                  : 'secondary'
              }
            >
              {warning.severity}
            </Badge>
            <Badge variant="outline">{warning.category}</Badge>
          </div>
          <p className="text-sm mt-2">{warning.message}</p>
          <div className="flex items-center gap-2 mt-2">
            <Badge variant="outline">
              {warning.schema}.{warning.table}
            </Badge>
            {warning.policy_name && (
              <Badge variant="secondary">{warning.policy_name}</Badge>
            )}
          </div>
          <p className="text-sm mt-2 p-2 bg-muted rounded">
            <strong>Suggestion:</strong> {warning.suggestion}
          </p>
          {warning.fix_sql && (
            <pre className="text-xs mt-2 p-2 bg-muted rounded overflow-auto">
              {warning.fix_sql}
            </pre>
          )}
        </div>
      </div>
    </div>
  )
}

// Template Card Component
function TemplateCard({
  template,
  onUse,
}: {
  template: PolicyTemplate
  onUse: () => void
}) {
  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-base">{template.name}</CardTitle>
        <CardDescription>{template.description}</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-2">
          <Badge variant="outline">{template.command}</Badge>
          <pre className="p-2 bg-muted rounded text-xs overflow-auto">
            {template.using}
          </pre>
          {template.with_check && (
            <pre className="p-2 bg-muted rounded text-xs overflow-auto">
              WITH CHECK: {template.with_check}
            </pre>
          )}
          <Button size="sm" onClick={onUse} className="w-full">
            Use Template
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}

// Create Policy Dialog
function CreatePolicyDialog({
  open,
  onOpenChange,
  schema,
  table,
  templates,
  onSubmit,
  isLoading,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  schema: string
  table: string
  templates: PolicyTemplate[]
  onSubmit: (data: CreatePolicyRequest) => void
  isLoading: boolean
}) {
  const [formData, setFormData] = useState<CreatePolicyRequest>({
    schema,
    table,
    name: '',
    command: 'ALL',
    roles: ['authenticated'],
    using: '',
    with_check: '',
    permissive: true,
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    onSubmit({
      ...formData,
      schema,
      table,
    })
  }

  const handleTemplateSelect = (templateId: string) => {
    const template = templates.find((t) => t.id === templateId)
    if (template) {
      setFormData({
        ...formData,
        command: template.command,
        using: template.using,
        with_check: template.with_check || '',
      })
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>Create Policy</DialogTitle>
          <DialogDescription>
            Create a new RLS policy for {schema}.{table}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="name">Policy Name</Label>
              <Input
                id="name"
                value={formData.name}
                onChange={(e) =>
                  setFormData({ ...formData, name: e.target.value })
                }
                placeholder="e.g., users_select_own"
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="command">Command</Label>
              <Select
                value={formData.command}
                onValueChange={(value) =>
                  setFormData({ ...formData, command: value })
                }
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="ALL">ALL</SelectItem>
                  <SelectItem value="SELECT">SELECT</SelectItem>
                  <SelectItem value="INSERT">INSERT</SelectItem>
                  <SelectItem value="UPDATE">UPDATE</SelectItem>
                  <SelectItem value="DELETE">DELETE</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          {templates.length > 0 && (
            <div className="space-y-2">
              <Label>Use Template</Label>
              <Select onValueChange={handleTemplateSelect}>
                <SelectTrigger>
                  <SelectValue placeholder="Select a template..." />
                </SelectTrigger>
                <SelectContent>
                  {templates.map((t) => (
                    <SelectItem key={t.id} value={t.id}>
                      {t.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          )}

          <div className="space-y-2">
            <Label htmlFor="using">USING Expression</Label>
            <Textarea
              id="using"
              value={formData.using || ''}
              onChange={(e) =>
                setFormData({ ...formData, using: e.target.value })
              }
              placeholder="e.g., auth.uid() = user_id"
              rows={3}
              className="font-mono text-sm"
            />
            <p className="text-xs text-muted-foreground">
              Expression that returns true for rows the user can access
            </p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="check">WITH CHECK Expression (optional)</Label>
            <Textarea
              id="check"
              value={formData.with_check || ''}
              onChange={(e) =>
                setFormData({
                  ...formData,
                  with_check: e.target.value,
                })
              }
              placeholder="e.g., auth.uid() = user_id"
              rows={3}
              className="font-mono text-sm"
            />
            <p className="text-xs text-muted-foreground">
              Expression that must be true for new/modified rows (INSERT/UPDATE)
            </p>
          </div>

          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Switch
                id="permissive"
                checked={formData.permissive}
                onCheckedChange={(checked) =>
                  setFormData({ ...formData, permissive: checked })
                }
              />
              <Label htmlFor="permissive">Permissive</Label>
            </div>
            <p className="text-xs text-muted-foreground">
              Permissive policies are combined with OR, restrictive with AND
            </p>
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={isLoading}>
              {isLoading ? (
                <Loader2 className="h-4 w-4 animate-spin mr-2" />
              ) : null}
              Create Policy
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
