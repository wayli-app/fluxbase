import { useState, useMemo } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import {
  Shield,
  ShieldCheck,
  AlertCircle,
  AlertTriangle,
  Search,
  Info,
  CheckCircle2,
  XCircle,
  Copy,
  Database,
  FileCode,
  Loader2,
} from "lucide-react";
import { toast } from "sonner";
import {
  policyApi,
  type RLSPolicy,
  type SecurityWarning,
  type PolicyTemplate,
  type CreatePolicyRequest,
} from "@/lib/api";
import { cn } from "@/lib/utils";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { PageHeader } from "@/components/layout/page-header";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  WarningCard,
  TemplateCard,
  CreatePolicyDialog,
  EditPolicyDialog,
  PolicyManagementModal,
  TemplateApplicationDialog,
} from "@/components/policies";

export const Route = createFileRoute("/_authenticated/policies/")({
  component: PoliciesPage,
});

function PoliciesPage() {
  const [searchQuery, setSearchQuery] = useState("");
  const [activeTab, setActiveTab] = useState("tables");
  const [policyModal, setPolicyModal] = useState<{
    open: boolean;
    schema: string;
    table: string;
    warning?: SecurityWarning | null;
  } | null>(null);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [deleteDialog, setDeleteDialog] = useState<{
    open: boolean;
    policy: RLSPolicy | null;
  }>({ open: false, policy: null });
  const [editDialog, setEditDialog] = useState<{
    open: boolean;
    policy: RLSPolicy | null;
  }>({ open: false, policy: null });
  const [templateDialog, setTemplateDialog] = useState<{
    open: boolean;
    template: PolicyTemplate | null;
    selectedTable: string;
  }>({ open: false, template: null, selectedTable: "" });

  const queryClient = useQueryClient();

  const { data: tablesData, isLoading: tablesLoading } = useQuery({
    queryKey: ["tables-rls"],
    queryFn: () => policyApi.getTablesWithRLS("public"),
  });

  const { data: warningsData, isLoading: warningsLoading } = useQuery({
    queryKey: ["security-warnings"],
    queryFn: () => policyApi.getSecurityWarnings(),
  });

  const { data: templates } = useQuery({
    queryKey: ["policy-templates"],
    queryFn: () => policyApi.getTemplates(),
  });

  const { data: tableDetails, isLoading: detailsLoading } = useQuery({
    queryKey: ["table-rls-status", policyModal],
    queryFn: () =>
      policyModal
        ? policyApi.getTableRLSStatus(policyModal.schema, policyModal.table)
        : null,
    enabled: !!policyModal?.open,
  });

  const toggleRLSMutation = useMutation({
    mutationFn: ({
      schema,
      table,
      enable,
      forceRLS,
    }: {
      schema: string;
      table: string;
      enable: boolean;
      forceRLS?: boolean;
    }) => policyApi.toggleTableRLS(schema, table, enable, forceRLS),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ["tables-rls"] });
      queryClient.invalidateQueries({ queryKey: ["table-rls-status"] });
      queryClient.invalidateQueries({ queryKey: ["security-warnings"] });
      toast.success(data.message);
    },
    onError: () => {
      toast.error("Failed to toggle RLS");
    },
  });

  const createPolicyMutation = useMutation({
    mutationFn: (data: CreatePolicyRequest) => policyApi.create(data),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ["table-rls-status"] });
      queryClient.invalidateQueries({ queryKey: ["security-warnings"] });
      setCreateDialogOpen(false);
      toast.success(data.message);
    },
    onError: () => {
      toast.error("Failed to create policy");
    },
  });

  const deletePolicyMutation = useMutation({
    mutationFn: ({
      schema,
      table,
      name,
    }: {
      schema: string;
      table: string;
      name: string;
    }) => policyApi.delete(schema, table, name),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ["table-rls-status"] });
      queryClient.invalidateQueries({ queryKey: ["security-warnings"] });
      setDeleteDialog({ open: false, policy: null });
      toast.success(data.message);
    },
    onError: () => {
      toast.error("Failed to delete policy");
    },
  });

  const updatePolicyMutation = useMutation({
    mutationFn: ({
      schema,
      table,
      name,
      data,
    }: {
      schema: string;
      table: string;
      name: string;
      data: {
        roles?: string[];
        using?: string | null;
        with_check?: string | null;
      };
    }) => policyApi.update(schema, table, name, data),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ["table-rls-status"] });
      queryClient.invalidateQueries({ queryKey: ["security-warnings"] });
      setEditDialog({ open: false, policy: null });
      toast.success(data.message);
    },
    onError: () => {
      toast.error("Failed to update policy");
    },
  });

  const filteredTables = useMemo(() => {
    if (!tablesData) return [];
    if (!searchQuery) return tablesData;
    const query = searchQuery.toLowerCase();
    return tablesData.filter(
      (t) =>
        t.table.toLowerCase().includes(query) ||
        t.schema.toLowerCase().includes(query),
    );
  }, [tablesData, searchQuery]);

  const sortedWarnings = useMemo(() => {
    if (!warningsData?.warnings) return [];
    const severityOrder: Record<string, number> = {
      critical: 0,
      high: 1,
      medium: 2,
      low: 3,
    };
    return [...warningsData.warnings].sort(
      (a, b) =>
        (severityOrder[a.severity] ?? 4) - (severityOrder[b.severity] ?? 4),
    );
  }, [warningsData]);

  const copyWarningsToClipboard = () => {
    if (!sortedWarnings.length) return;
    const header = "severity,policy_name,table,message,suggestion";
    const csv = sortedWarnings
      .map((w) => {
        const escape = (s: string | undefined) => {
          if (!s) return "";
          if (s.includes(",") || s.includes('"') || s.includes("\n")) {
            return `"${s.replace(/"/g, '""')}"`;
          }
          return s;
        };
        return [
          escape(w.severity),
          escape(w.policy_name || ""),
          escape(`${w.schema}.${w.table}`),
          escape(w.message),
          escape(w.suggestion),
        ].join(",");
      })
      .join("\n");
    navigator.clipboard.writeText(header + "\n" + csv);
    toast.success(`Copied ${sortedWarnings.length} warnings to clipboard`);
  };

  return (
    <div className="flex h-full flex-col">
      <PageHeader
        icon={<Shield />}
        title="Row Level Security"
        description="Manage RLS policies and security settings for your tables"
      />

      <div className="flex-1 overflow-auto p-6">
        {warningsData && (
          <div className="grid grid-cols-4 gap-4">
            <Card
              className={cn(
                warningsData.summary.critical > 0 &&
                  "border-red-500/50 bg-red-500/5",
              )}
            >
              <CardHeader className="pb-2">
                <CardDescription>Critical Issues</CardDescription>
                <CardTitle className="flex items-center gap-2 text-2xl">
                  <AlertCircle className="h-5 w-5 text-red-500" />
                  {warningsData.summary.critical}
                </CardTitle>
              </CardHeader>
            </Card>
            <Card
              className={cn(
                warningsData.summary.high > 0 &&
                  "border-orange-500/50 bg-orange-500/5",
              )}
            >
              <CardHeader className="pb-2">
                <CardDescription>High Priority</CardDescription>
                <CardTitle className="flex items-center gap-2 text-2xl">
                  <AlertTriangle className="h-5 w-5 text-orange-500" />
                  {warningsData.summary.high}
                </CardTitle>
              </CardHeader>
            </Card>
            <Card>
              <CardHeader className="pb-2">
                <CardDescription>Medium Priority</CardDescription>
                <CardTitle className="flex items-center gap-2 text-2xl">
                  <Info className="h-5 w-5 text-yellow-500" />
                  {warningsData.summary.medium}
                </CardTitle>
              </CardHeader>
            </Card>
            <Card>
              <CardHeader className="pb-2">
                <CardDescription>Tables with RLS</CardDescription>
                <CardTitle className="flex items-center gap-2 text-2xl">
                  <ShieldCheck className="h-5 w-5 text-green-500" />
                  {tablesData?.filter((t) => t.rls_enabled).length || 0}/
                  {tablesData?.length || 0}
                </CardTitle>
              </CardHeader>
            </Card>
          </div>
        )}

        <Tabs value={activeTab} onValueChange={setActiveTab} className="mt-6">
          <TabsList>
            <TabsTrigger value="tables" className="flex items-center gap-2">
              <Database className="h-4 w-4" />
              Tables
            </TabsTrigger>
            <TabsTrigger value="warnings" className="flex items-center gap-2">
              <AlertTriangle className="h-4 w-4" />
              Security Warnings
              {warningsData && warningsData.summary.total > 0 && (
                <Badge variant="destructive" className="ml-1">
                  {warningsData.summary.total}
                </Badge>
              )}
            </TabsTrigger>
            <TabsTrigger value="templates" className="flex items-center gap-2">
              <FileCode className="h-4 w-4" />
              Policy Templates
            </TabsTrigger>
          </TabsList>

          <TabsContent value="tables" className="space-y-4">
            <div className="flex items-center gap-4">
              <div className="relative max-w-sm flex-1">
                <Search className="text-muted-foreground absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2" />
                <Input
                  placeholder="Search tables..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="pl-9"
                />
              </div>
            </div>

            <div>
              <Card>
                <CardHeader>
                  <CardTitle>Tables</CardTitle>
                  <CardDescription>
                    Click a table to view and manage its RLS policies
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  {tablesLoading ? (
                    <div className="flex justify-center py-8">
                      <Loader2 className="text-muted-foreground h-6 w-6 animate-spin" />
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
                            className="hover:bg-muted cursor-pointer"
                            onClick={() => {
                              setPolicyModal({
                                open: true,
                                schema: table.schema,
                                table: table.table,
                              });
                            }}
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
                                <XCircle className="text-muted-foreground h-4 w-4" />
                              )}
                            </TableCell>
                            <TableCell>
                              <Badge
                                variant={
                                  table.policy_count > 0
                                    ? "default"
                                    : "secondary"
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
            </div>
          </TabsContent>

          <TabsContent value="warnings">
            <Card>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <div>
                    <CardTitle>Security Warnings</CardTitle>
                    <CardDescription>
                      Issues that may indicate security vulnerabilities in your
                      RLS configuration
                    </CardDescription>
                  </div>
                  {sortedWarnings.length > 0 && (
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={copyWarningsToClipboard}
                    >
                      <Copy className="mr-2 h-4 w-4" />
                      Copy All
                    </Button>
                  )}
                </div>
              </CardHeader>
              <CardContent>
                {warningsLoading ? (
                  <div className="flex justify-center py-8">
                    <Loader2 className="text-muted-foreground h-6 w-6 animate-spin" />
                  </div>
                ) : sortedWarnings.length === 0 ? (
                  <div className="text-muted-foreground py-12 text-center">
                    <ShieldCheck className="mx-auto mb-4 h-12 w-12 text-green-500" />
                    <h3 className="text-lg font-medium">
                      No Security Issues Found
                    </h3>
                    <p className="mt-1 text-sm">
                      Your RLS configuration looks good
                    </p>
                  </div>
                ) : (
                  <div className="space-y-3">
                    {sortedWarnings.map((warning, index) => (
                      <WarningCard
                        key={`${warning.id}-${index}`}
                        warning={warning}
                        onNavigate={() => {
                          setPolicyModal({
                            open: true,
                            schema: warning.schema,
                            table: warning.table,
                            warning: warning,
                          });
                        }}
                      />
                    ))}
                  </div>
                )}
              </CardContent>
            </Card>
          </TabsContent>

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
                  <div className="text-muted-foreground py-12 text-center">
                    <FileCode className="mx-auto mb-4 h-12 w-12" />
                    <h3 className="text-lg font-medium">
                      No Templates Available
                    </h3>
                  </div>
                ) : (
                  <div className="grid gap-4 md:grid-cols-2">
                    {templates?.map((template) => (
                      <TemplateCard
                        key={template.id}
                        template={template}
                        onUse={() => {
                          setTemplateDialog({
                            open: true,
                            template,
                            selectedTable: "",
                          });
                        }}
                      />
                    ))}
                  </div>
                )}
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>

        <PolicyManagementModal
          open={!!policyModal?.open}
          onOpenChange={(open) => !open && setPolicyModal(null)}
          schema={policyModal?.schema || ""}
          table={policyModal?.table || ""}
          warning={policyModal?.warning}
          tableDetails={tableDetails}
          detailsLoading={detailsLoading}
          onToggleRLS={(enable) => {
            if (policyModal) {
              toggleRLSMutation.mutate({
                schema: policyModal.schema,
                table: policyModal.table,
                enable,
              });
            }
          }}
          onEditPolicy={(policy) => setEditDialog({ open: true, policy })}
          onDeletePolicy={(policy) => setDeleteDialog({ open: true, policy })}
          onCreatePolicy={() => setCreateDialogOpen(true)}
        />

        {policyModal && (
          <CreatePolicyDialog
            open={createDialogOpen}
            onOpenChange={setCreateDialogOpen}
            schema={policyModal.schema}
            table={policyModal.table}
            templates={templates || []}
            onSubmit={(data) => createPolicyMutation.mutate(data)}
            isLoading={createPolicyMutation.isPending}
          />
        )}

        <TemplateApplicationDialog
          open={templateDialog.open}
          onOpenChange={(open) =>
            setTemplateDialog({
              open,
              template: open ? templateDialog.template : null,
              selectedTable: "",
            })
          }
          template={templateDialog.template}
          tables={tablesData || []}
          selectedTable={templateDialog.selectedTable}
          onTableSelect={(table) =>
            setTemplateDialog({ ...templateDialog, selectedTable: table })
          }
          onApply={() => {
            if (templateDialog.template && templateDialog.selectedTable) {
              const [schema, table] = templateDialog.selectedTable.split(".");
              setPolicyModal({ open: true, schema, table });
              setCreateDialogOpen(true);
              setTemplateDialog({
                open: false,
                template: null,
                selectedTable: "",
              });
            }
          }}
        />

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
                {deleteDialog.policy?.policy_name}&quot;? This action cannot be
                undone.
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
                    });
                  }
                }}
                className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              >
                {deletePolicyMutation.isPending ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  "Delete"
                )}
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>

        {editDialog.policy && (
          <EditPolicyDialog
            open={editDialog.open}
            onOpenChange={(open) =>
              setEditDialog({ open, policy: open ? editDialog.policy : null })
            }
            policy={editDialog.policy}
            onSubmit={(data) => {
              if (editDialog.policy) {
                updatePolicyMutation.mutate({
                  schema: editDialog.policy.schema,
                  table: editDialog.policy.table,
                  name: editDialog.policy.policy_name,
                  data,
                });
              }
            }}
            isLoading={updatePolicyMutation.isPending}
          />
        )}
      </div>
    </div>
  );
}
