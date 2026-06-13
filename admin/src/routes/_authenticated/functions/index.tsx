import { useState, useEffect, useCallback, useMemo, useRef } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { Zap, Activity, RefreshCw } from "lucide-react";
import { toast } from "sonner";
import { useImpersonationStore } from "@/stores/impersonation-store";
import { useTenantStore } from "@/stores/tenant-store";
import { fluxbaseClient } from "@/lib/fluxbase-client";
import {
  functionsApi,
  type EdgeFunction,
  type EdgeFunctionExecution,
} from "@/lib/api";
import { useExecutionLogs } from "@/hooks/use-execution-logs";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { ImpersonationPopover } from "@/features/impersonation/components/impersonation-popover";
import { PageHeader } from "@/components/layout/page-header";
import {
  StatsCard,
  ExecutionFilters,
  ExecutionsList,
  PaginationControls,
  FunctionsList,
  DeleteConfirmDialog,
  CreateFunctionDialog,
  EditFunctionDialog,
  InvokeFunctionDialog,
  ExecutionLogsDialog,
  ResultDialog,
  ExecutionDetailDialog,
  type FunctionFormData,
  type InvokeResult,
  DEFAULT_FORM_DATA,
} from "@/components/functions";

export const Route = createFileRoute("/_authenticated/functions/")({
  component: FunctionsPage,
});

function FunctionsPage() {
  return (
    <div className="flex h-full flex-col">
      <PageHeader
        icon={<Zap />}
        title="Edge Functions"
        description="Deploy and run TypeScript/JavaScript functions with Deno runtime"
        actions={
          <ImpersonationPopover
            contextLabel="Invoking as"
            defaultReason="Testing function invocation"
          />
        }
      />

      <div className="flex-1 overflow-auto p-6">
        <EdgeFunctionsTab />
      </div>
    </div>
  );
}

function EdgeFunctionsTab() {
  const currentTenantId = useTenantStore((state) => state.currentTenant?.id);
  const queryClient = useQueryClient();

  // Tab state
  const [activeTab, setActiveTab] = useState<"executions" | "functions">(
    "executions",
  );

  // Existing state
  const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false);
  const [isEditDialogOpen, setIsEditDialogOpen] = useState(false);
  const [isInvokeDialogOpen, setIsInvokeDialogOpen] = useState(false);
  const [isLogsDialogOpen, setIsLogsDialogOpen] = useState(false);
  const [isResultDialogOpen, setIsResultDialogOpen] = useState(false);
  const [selectedFunction, setSelectedFunction] = useState<EdgeFunction | null>(
    null,
  );
  const [executions, setExecutions] = useState<EdgeFunctionExecution[]>([]);
  const [invoking, setInvoking] = useState(false);

  // All executions state (for admin executions tab)
  const [allExecutions, setAllExecutions] = useState<EdgeFunctionExecution[]>(
    [],
  );
  const [executionsLoading, setExecutionsLoading] = useState(false);
  const [isInitialLoad, setIsInitialLoad] = useState(true);
  const [totalExecutions, setTotalExecutions] = useState(0);
  // Pagination state
  const [executionsPage, setExecutionsPage] = useState(0); // 0-indexed
  const [executionsPageSize, setExecutionsPageSize] = useState(25);

  // Filters state
  const [searchQuery, setSearchQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState<string>("all");
  const [invokeResult, setInvokeResult] = useState<InvokeResult | null>(null);
  const [wordWrap, setWordWrap] = useState(false);
  const [logsWordWrap, setLogsWordWrap] = useState(false);
  const [reloading, setReloading] = useState(false);
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null);
  const [fetchingFunction, setFetchingFunction] = useState(false);
  const [namespaces, setNamespaces] = useState<string[]>(["default"]);
  const [selectedNamespace, setSelectedNamespace] = useState<string>("default");

  // Functions list via TanStack Query
  const {
    data: edgeFunctions = [],
    isLoading: loading,
    refetch: refetchFunctions,
  } = useQuery({
    queryKey: ["edge-functions", selectedNamespace, currentTenantId],
    queryFn: async () => {
      const data = await functionsApi.list(selectedNamespace);
      return data || [];
    },
  });

  // Execution detail dialog state
  const [showExecutionDetailDialog, setShowExecutionDetailDialog] =
    useState(false);
  const [selectedExecution, setSelectedExecution] =
    useState<EdgeFunctionExecution | null>(null);
  const [logLevelFilter, setLogLevelFilter] = useState<string>("all");

  // Use the real-time execution logs hook
  const { logs: executionLogs, loading: executionLogsLoading } =
    useExecutionLogs({
      executionId: selectedExecution?.id || null,
      executionType: "function",
      enabled: showExecutionDetailDialog,
    });

  // Realtime subscription: update selected execution when it changes
  useEffect(() => {
    if (!showExecutionDetailDialog || !selectedExecution) return;

    const isActive =
      selectedExecution.status === "running" ||
      selectedExecution.status === "pending";
    if (!isActive) return;

    const channel = fluxbaseClient
      .channel(`exec-detail-${selectedExecution.id}`)
      .on(
        "postgres_changes",
        {
          event: "UPDATE",
          schema: "functions",
          table: "edge_executions",
          filter: `id=eq.${selectedExecution.id}`,
        },
        (payload) => {
          const updated = payload.new as EdgeFunctionExecution;
          setSelectedExecution((prev) =>
            prev ? { ...prev, ...updated } : prev,
          );

          if (updated.status !== "running" && updated.status !== "pending") {
            fetchAllExecutionsRef.current();
          }
        },
      )
      .subscribe();

    return () => {
      channel.unsubscribe();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [showExecutionDetailDialog, selectedExecution?.id, selectedExecution?.status]);

  // Realtime subscription: live execution list updates
  useEffect(() => {
    const channel = fluxbaseClient
      .channel("functions-executions-updates")
      .on(
        "postgres_changes",
        {
          event: "*",
          schema: "functions",
          table: "edge_executions",
        },
        (payload) => {
          const eventType = payload.eventType;
          const newExec = payload.new as
            | EdgeFunctionExecution
            | undefined;
          const oldExec = payload.old as { id: string } | undefined;

          setAllExecutions((prev) => {
            if (eventType === "INSERT" && newExec) {
              if (
                selectedNamespace !== "all" &&
                newExec.namespace !== selectedNamespace
              ) {
                return prev;
              }
              if (
                statusFilter !== "all" &&
                newExec.status !== statusFilter
              ) {
                return prev;
              }
              if (prev.some((e) => e.id === newExec.id)) {
                return prev;
              }
              return [newExec, ...prev];
            }

            if (eventType === "UPDATE" && newExec) {
              const idx = prev.findIndex((e) => e.id === newExec.id);
              if (idx === -1) {
                if (
                  selectedNamespace !== "all" &&
                  newExec.namespace !== selectedNamespace
                ) {
                  return prev;
                }
                if (
                  statusFilter !== "all" &&
                  newExec.status !== statusFilter
                ) {
                  return prev;
                }
                return [newExec, ...prev];
              }
              if (
                statusFilter !== "all" &&
                newExec.status !== statusFilter
              ) {
                return prev.filter((e) => e.id !== newExec.id);
              }
              const updated = [...prev];
              updated[idx] = { ...updated[idx], ...newExec };
              return updated;
            }

            if (eventType === "DELETE" && oldExec) {
              return prev.filter((e) => e.id !== oldExec.id);
            }

            return prev;
          });
        },
      )
      .subscribe();

    return () => {
      channel.unsubscribe();
    };
  }, [selectedNamespace, statusFilter]);

  // Ref to track initial fetch (prevents debounced search from re-fetching on mount)
  const hasInitialFetch = useRef(false);
  // Ref to hold latest fetchAllExecutions to avoid it being a dependency in effects
  const fetchAllExecutionsRef = useRef<(reset?: boolean) => Promise<void>>(() =>
    Promise.resolve(),
  );

  // Form state
  const [formData, setFormData] = useState<FunctionFormData>(DEFAULT_FORM_DATA);

  const [invokeBody, setInvokeBody] = useState("{}");
  const [invokeMethod, setInvokeMethod] = useState<
    "GET" | "POST" | "PUT" | "DELETE" | "PATCH"
  >("POST");
  const [invokeHeaders, setInvokeHeaders] = useState<
    Array<{ key: string; value: string }>
  >([{ key: "", value: "" }]);

  const reloadFunctionsFromDisk = async (showToast = false) => {
    try {
      const result = await functionsApi.reload();
      if (showToast) {
        const created = result.created?.length ?? 0;
        const updated = result.updated?.length ?? 0;
        const deleted = result.deleted?.length ?? 0;
        const errors = result.errors?.length ?? 0;

        if (created > 0 || updated > 0 || deleted > 0) {
          const messages = [];
          if (created > 0) messages.push(`${created} created`);
          if (updated > 0) messages.push(`${updated} updated`);
          if (deleted > 0) messages.push(`${deleted} deleted`);

          toast.success(`Functions reloaded: ${messages.join(", ")}`);
        } else if (errors > 0) {
          toast.error(`Failed to reload functions: ${errors} errors`);
        } else {
          toast.info("No changes detected");
        }
      }
      return result;
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error("Error reloading functions:", error);
      if (showToast) {
        toast.error("Failed to reload functions from filesystem");
      }
      throw error;
    }
  };

  const handleReloadClick = async () => {
    setReloading(true);
    try {
      await reloadFunctionsFromDisk(true);
      await refetchFunctions();
    } finally {
      setReloading(false);
    }
  };

  // Fetch all executions for the executions tab
  const fetchAllExecutions = useCallback(async () => {
    // Only show full loading spinner on initial load, not on refetches
    if (isInitialLoad) {
      setExecutionsLoading(true);
    }

    try {
      const offset = executionsPage * executionsPageSize;
      const result = await functionsApi.listAllExecutions({
        namespace: selectedNamespace !== "all" ? selectedNamespace : undefined,
        function_name: searchQuery || undefined,
        status: statusFilter !== "all" ? statusFilter : undefined,
        limit: executionsPageSize,
        offset,
      });

      setAllExecutions(result.executions || []);
      setTotalExecutions(result.count || 0);

      // Mark initial load as complete
      if (isInitialLoad) {
        setIsInitialLoad(false);
      }
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error("Error fetching executions:", error);
      toast.error("Failed to fetch executions");
    } finally {
      setExecutionsLoading(false);
    }
  }, [
    selectedNamespace,
    searchQuery,
    statusFilter,
    executionsPage,
    executionsPageSize,
    isInitialLoad,
  ]);

  // Keep the ref updated with the latest fetchAllExecutions
  useEffect(() => {
    fetchAllExecutionsRef.current = fetchAllExecutions;
  }, [fetchAllExecutions]);

  // Filter executions from past 24 hours (for stats display only)
  const executions24h = useMemo(() => {
    const cutoff = Date.now() - 24 * 60 * 60 * 1000;
    return allExecutions.filter((exec) => {
      const execTime = new Date(exec.executed_at).getTime();
      return execTime >= cutoff;
    });
  }, [allExecutions]);

  // Calculate stats from past 24 hours
  const executionStats = useMemo(() => {
    const success = executions24h.filter((e) => e.status === "success").length;
    const failed = executions24h.filter(
      (e) => e.status === "error" || e.status === "failed",
    ).length;
    const total = executions24h.length;
    const avgDuration =
      executions24h.length > 0
        ? Math.round(
            executions24h.reduce((sum, e) => sum + (e.duration_ms || 0), 0) /
              executions24h.length,
          )
        : 0;
    return { success, failed, total, avgDuration };
  }, [executions24h]);

  // Filter logs by level
  const filteredLogs = useMemo(() => {
    if (logLevelFilter === "all") return executionLogs;
    return executionLogs.filter((log) => log.level === logLevelFilter);
  }, [executionLogs, logLevelFilter]);

  // Open execution detail dialog
  const openExecutionDetail = (exec: EdgeFunctionExecution) => {
    setSelectedExecution(exec);
    setShowExecutionDetailDialog(true);
    setLogLevelFilter("all");
  };

  // Fetch namespaces on mount and select best default
  useEffect(() => {
    const fetchNamespaces = async () => {
      try {
        const data = await functionsApi.listNamespaces();
        // Filter out empty strings to prevent Select component errors
        const validNamespaces = data.filter((ns: string) => ns !== "");
        setNamespaces(
          validNamespaces.length > 0 ? validNamespaces : ["default"],
        );

        // Smart namespace selection: if 'default' is empty but other namespaces have items,
        // select a non-empty namespace instead
        let bestNamespace = validNamespaces[0] || "default";

        if (validNamespaces.includes("default") && validNamespaces.length > 1) {
          // Check if 'default' namespace has any functions
          try {
            const defaultFunctions = await functionsApi.list("default");
            if (!defaultFunctions || defaultFunctions.length === 0) {
              // Default is empty, find first non-empty namespace
              for (const ns of validNamespaces) {
                if (ns !== "default") {
                  const nsFunctions = await functionsApi.list(ns);
                  if (nsFunctions && nsFunctions.length > 0) {
                    bestNamespace = ns;
                    break;
                  }
                }
              }
            }
          } catch {
            // If checking fails, stick with default
          }
        }

        setSelectedNamespace((current) =>
          validNamespaces.includes(current) ? current : bestNamespace,
        );
      } catch {
        setNamespaces(["default"]);
      }
    };
    fetchNamespaces();
  }, [currentTenantId]); // Re-fetch when tenant changes

  // Fetch executions when tab changes or any fetch-related state changes
  useEffect(() => {
    if (activeTab === "executions") {
      hasInitialFetch.current = true;
      fetchAllExecutionsRef.current();
    }
    // Using ref to avoid fetchAllExecutions in dependencies which would cause double-fetches
    // All filter/pagination changes will trigger this effect via their state changes
  }, [
    activeTab,
    selectedNamespace,
    statusFilter,
    executionsPage,
    executionsPageSize,
    currentTenantId,
  ]);

  // Debounced search - resets page to 0 and fetches
  useEffect(() => {
    if (activeTab !== "executions") return;
    // Skip the first render - the main effect above handles initial fetch
    if (!hasInitialFetch.current) return;
    const timer = setTimeout(() => {
      // Reset to page 0 when search changes - this will trigger the main effect
      setExecutionsPage(0);
    }, 300);
    return () => clearTimeout(timer);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchQuery]);

  const createFunction = async () => {
    try {
      await functionsApi.create({
        ...formData,
        cron_schedule: formData.cron_schedule || null,
      });
      toast.success("Edge function created successfully");
      setIsCreateDialogOpen(false);
      resetForm();
      queryClient.invalidateQueries({ queryKey: ["edge-functions"] });
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error("Error creating edge function:", error);
      toast.error("Failed to create edge function");
    }
  };

  const updateFunction = async () => {
    if (!selectedFunction) return;

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
      });
      toast.success("Edge function updated successfully");
      setIsEditDialogOpen(false);
      queryClient.invalidateQueries({ queryKey: ["edge-functions"] });
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error("Error updating edge function:", error);
      toast.error("Failed to update edge function");
    }
  };

  const deleteFunction = async (name: string) => {
    try {
      await functionsApi.delete(name);
      toast.success("Edge function deleted successfully");
      queryClient.invalidateQueries({ queryKey: ["edge-functions"] });
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error("Error deleting edge function:", error);
      toast.error("Failed to delete edge function");
    } finally {
      setDeleteConfirm(null);
    }
  };

  const toggleFunction = async (fn: EdgeFunction) => {
    const newEnabledState = !fn.enabled;

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
      });
      toast.success(`Function ${newEnabledState ? "enabled" : "disabled"}`);
      queryClient.invalidateQueries({ queryKey: ["edge-functions"] });
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error("Error toggling function:", error);
      toast.error("Failed to toggle function");
    }
  };

  const invokeFunction = async () => {
    if (!selectedFunction) return;

    setInvoking(true);
    try {
      // Convert headers array to object, filtering empty ones
      const headersObj = invokeHeaders
        .filter((h) => h.key.trim() !== "")
        .reduce((acc, h) => ({ ...acc, [h.key]: h.value }), {});

      // Build config with impersonation token if active
      const { isImpersonating, impersonationToken } =
        useImpersonationStore.getState();
      const config: { headers?: Record<string, string> } = {};
      if (isImpersonating && impersonationToken) {
        config.headers = { "X-Impersonation-Token": impersonationToken };
      }

      const result = await functionsApi.invoke(
        selectedFunction.name,
        {
          method: invokeMethod,
          headers: headersObj,
          body: invokeBody,
        },
        config,
      );
      toast.success("Function invoked successfully");
      setInvokeResult({ success: true, data: result });
      setIsInvokeDialogOpen(false);
      setIsResultDialogOpen(true);
    } catch (error: unknown) {
      // eslint-disable-next-line no-console
      console.error("Error invoking function:", error);
      toast.error("Failed to invoke function");
      const errorMessage =
        error instanceof Error ? error.message : "Unknown error";
      setInvokeResult({ success: false, data: "", error: errorMessage });
      setIsInvokeDialogOpen(false);
      setIsResultDialogOpen(true);
    } finally {
      setInvoking(false);
    }
  };

  const fetchExecutions = async (functionName: string) => {
    try {
      const data = await functionsApi.getExecutions(functionName, 20);
      setExecutions(data || []);
      setIsLogsDialogOpen(true);
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error("Error fetching executions:", error);
      toast.error("Failed to fetch execution logs");
    }
  };

  const openEditDialog = async (fn: EdgeFunction) => {
    setSelectedFunction(fn);
    setFetchingFunction(true);
    setIsEditDialogOpen(true);
    try {
      // Fetch full function details including code
      const fullFunction = await functionsApi.get(fn.name);
      setFormData({
        name: fullFunction.name,
        description: fullFunction.description || "",
        code: fullFunction.code || "",
        timeout_seconds: fullFunction.timeout_seconds,
        memory_limit_mb: fullFunction.memory_limit_mb,
        allow_net: fullFunction.allow_net,
        allow_env: fullFunction.allow_env,
        allow_read: fullFunction.allow_read,
        allow_write: fullFunction.allow_write,
        cron_schedule: fullFunction.cron_schedule || "",
      });
    } catch {
      toast.error("Failed to load function details");
      setIsEditDialogOpen(false);
    } finally {
      setFetchingFunction(false);
    }
  };

  const openInvokeDialog = (fn: EdgeFunction) => {
    setSelectedFunction(fn);
    setInvokeBody('{\n  "name": "World"\n}');
    setIsInvokeDialogOpen(true);
  };

  const resetForm = () => {
    setFormData(DEFAULT_FORM_DATA);
  };

  if (loading) {
    return (
      <div className="flex h-96 items-center justify-center">
        <RefreshCw className="text-muted-foreground h-8 w-8 animate-spin" />
      </div>
    );
  }

  return (
    <>
      <StatsCard stats={executionStats} />

      {/* Tabs */}
      <Tabs
        value={activeTab}
        onValueChange={(v) => setActiveTab(v as "executions" | "functions")}
        className="flex min-h-0 flex-1 flex-col"
      >
        <div className="mb-4 flex items-center justify-between">
          <TabsList className="grid w-full max-w-md grid-cols-2">
            <TabsTrigger value="executions">
              <Activity className="mr-2 h-4 w-4" />
              Execution Logs
            </TabsTrigger>
            <TabsTrigger value="functions">
              <Zap className="mr-2 h-4 w-4" />
              Functions
            </TabsTrigger>
          </TabsList>
        </div>

        {/* Executions Tab */}
        <TabsContent value="executions" className="mt-0 flex-1">
          <ExecutionFilters
            namespaces={namespaces}
            selectedNamespace={selectedNamespace}
            onNamespaceChange={(ns) => {
              setSelectedNamespace(ns);
              setExecutionsPage(0);
            }}
            searchQuery={searchQuery}
            onSearchChange={setSearchQuery}
            statusFilter={statusFilter}
            onStatusFilterChange={(status) => {
              setStatusFilter(status);
              setExecutionsPage(0);
            }}
            onRefresh={() => fetchAllExecutionsRef.current()}
          />

          <ExecutionsList
            executions={allExecutions}
            loading={executionsLoading}
            isInitialLoad={isInitialLoad}
            onExecutionClick={openExecutionDetail}
          />

          <PaginationControls
            currentPage={executionsPage}
            pageSize={executionsPageSize}
            total={totalExecutions}
            onPageChange={setExecutionsPage}
            onPageSizeChange={(size) => {
              setExecutionsPageSize(size);
              setExecutionsPage(0);
            }}
          />
        </TabsContent>

        {/* Functions Tab */}
        <TabsContent value="functions" className="mt-0 flex-1">
          <FunctionsList
            edgeFunctions={edgeFunctions}
            namespaces={namespaces}
            selectedNamespace={selectedNamespace}
            reloading={reloading}
            onNamespaceChange={setSelectedNamespace}
            onReload={handleReloadClick}
            onRefresh={() => refetchFunctions()}
            onCreateFunction={() => setIsCreateDialogOpen(true)}
            onEditFunction={openEditDialog}
            onInvokeFunction={openInvokeDialog}
            onViewLogs={(fn) => fetchExecutions(fn.name)}
            onDeleteFunction={(name) => setDeleteConfirm(name)}
            onToggleFunction={toggleFunction}
          />
        </TabsContent>
      </Tabs>

      <DeleteConfirmDialog
        open={deleteConfirm !== null}
        onOpenChange={(open) => !open && setDeleteConfirm(null)}
        functionName={deleteConfirm}
        onConfirm={deleteFunction}
      />

      <CreateFunctionDialog
        open={isCreateDialogOpen}
        onOpenChange={setIsCreateDialogOpen}
        formData={formData}
        onFormDataChange={setFormData}
        onSubmit={createFunction}
        onReset={resetForm}
      />

      <EditFunctionDialog
        open={isEditDialogOpen}
        onOpenChange={setIsEditDialogOpen}
        formData={formData}
        onFormDataChange={setFormData}
        fetching={fetchingFunction}
        onSubmit={updateFunction}
      />

      <InvokeFunctionDialog
        open={isInvokeDialogOpen}
        onOpenChange={setIsInvokeDialogOpen}
        selectedFunction={selectedFunction}
        invokeMethod={invokeMethod}
        onInvokeMethodChange={setInvokeMethod}
        invokeBody={invokeBody}
        onInvokeBodyChange={setInvokeBody}
        invokeHeaders={invokeHeaders}
        onInvokeHeadersChange={setInvokeHeaders}
        invoking={invoking}
        onSubmit={invokeFunction}
      />

      <ExecutionLogsDialog
        open={isLogsDialogOpen}
        onOpenChange={setIsLogsDialogOpen}
        selectedFunction={selectedFunction}
        executions={executions}
        wordWrap={logsWordWrap}
        onWordWrapChange={setLogsWordWrap}
      />

      <ResultDialog
        open={isResultDialogOpen}
        onOpenChange={setIsResultDialogOpen}
        result={invokeResult}
        wordWrap={wordWrap}
        onWordWrapChange={setWordWrap}
      />

      <ExecutionDetailDialog
        open={showExecutionDetailDialog}
        onOpenChange={setShowExecutionDetailDialog}
        selectedExecution={selectedExecution}
        executionLogs={executionLogs}
        executionLogsLoading={executionLogsLoading}
        logLevelFilter={logLevelFilter}
        onLogLevelFilterChange={setLogLevelFilter}
        filteredLogs={filteredLogs}
      />
    </>
  );
}
