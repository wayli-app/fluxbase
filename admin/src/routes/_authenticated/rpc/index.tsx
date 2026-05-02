import { useState, useMemo, useRef, useEffect } from "react";
import z from "zod";
import {
  useQuery,
  useMutation,
  useQueryClient,
  keepPreviousData,
} from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import {
  Terminal,
  RefreshCw,
  HardDrive,
  Activity,
  Search,
  Filter,
  Loader2,
  StopCircle,
  Globe,
  Lock,
  Timer,
  Eye,
} from "lucide-react";
import { toast } from "sonner";
import { rpcApi, type RPCProcedure, type RPCExecution } from "@/lib/api";
import { useTenantStore } from "@/stores/tenant-store";
import { useExecutionLogs } from "@/hooks/use-execution-logs";
import { fluxbaseClient } from "@/lib/fluxbase-client";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { ImpersonationPopover } from "@/features/impersonation/components/impersonation-popover";
import {
  ExecutionDetailsDialog,
  ProcedureDetailsDialog,
  getStatusIcon,
  getStatusVariant,
  canCancelExecution,
} from "@/components/rpc";

const rpcSearchSchema = z.object({
  tab: z.string().optional().catch("executions"),
  namespace: z.string().optional().catch("default"),
});

export const Route = createFileRoute("/_authenticated/rpc/")({
  validateSearch: rpcSearchSchema,
  component: RPCPage,
});

const RPC_PAGE_SIZE = 50;

function RPCPage() {
  return (
    <div className="flex h-full flex-col">
      <div className="bg-background flex items-center justify-between border-b px-6 py-4">
        <div className="flex items-center gap-3">
          <div className="bg-primary/10 flex h-10 w-10 items-center justify-center rounded-lg">
            <Terminal className="text-primary h-5 w-5" />
          </div>
          <div>
            <h1 className="text-xl font-semibold">RPC Procedures</h1>
            <p className="text-muted-foreground text-sm">
              Execute SQL procedures securely via API
            </p>
          </div>
        </div>
        <ImpersonationPopover
          contextLabel="Executing as"
          defaultReason="Testing RPC procedure execution"
        />
      </div>

      <div className="flex-1 overflow-auto p-6">
        <RPCContent />
      </div>
    </div>
  );
}

function RPCContent() {
  const queryClient = useQueryClient();
  const currentTenantId = useTenantStore((state) => state.currentTenant?.id);
  const [activeTab, setActiveTab] = useState<"executions" | "procedures">(
    "executions",
  );

  const [namespaces, setNamespaces] = useState<string[]>(["default"]);
  const [selectedNamespace, setSelectedNamespace] = useState<string>("default");

  const [searchQuery, setSearchQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState<string>("all");

  const [namespacesLoaded, setNamespacesLoaded] = useState(false);

  // Executions pagination state
  const executionsOffsetRef = useRef(0);
  const [hasMoreExecutions, setHasMoreExecutions] = useState(true);
  const [loadingMoreExecutions, setLoadingMoreExecutions] = useState(false);

  // Dialog state
  const [selectedExecution, setSelectedExecution] =
    useState<RPCExecution | null>(null);
  const [isExecutionDetailsOpen, setShowExecutionDetails] = useState(false);

  const { logs: executionLogs, loading: loadingLogs } = useExecutionLogs({
    executionId: selectedExecution?.id || null,
    executionType: "rpc",
    enabled: isExecutionDetailsOpen,
  });

  // Realtime subscription: update selected execution when it changes
  useEffect(() => {
    if (!isExecutionDetailsOpen || !selectedExecution) return;

    const isActive =
      selectedExecution.status === "running" ||
      selectedExecution.status === "pending";
    if (!isActive) return;

    const channel = fluxbaseClient
      .channel(`rpc-exec-detail-${selectedExecution.id}`)
      .on(
        "postgres_changes",
        {
          event: "UPDATE",
          schema: "rpc",
          table: "executions",
          filter: `id=eq.${selectedExecution.id}`,
        },
        (payload) => {
          const updated = payload.new as RPCExecution;
          setSelectedExecution((prev) =>
            prev ? { ...prev, ...updated } : prev,
          );

          if (
            updated.status !== "running" &&
            updated.status !== "pending"
          ) {
            refetchExecutions();
          }
        },
      )
      .subscribe();

    return () => {
      channel.unsubscribe();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isExecutionDetailsOpen, selectedExecution?.id, selectedExecution?.status]);

  const [cancellingExecutionId, setCancellingExecutionId] = useState<
    string | null
  >(null);

  const [selectedProcedure, setSelectedProcedure] =
    useState<RPCProcedure | null>(null);
  const [showProcedureDetails, setShowProcedureDetails] = useState(false);
  const [loadingProcedure, setLoadingProcedure] = useState(false);

  // --- Queries ---

  // Procedures query
  const {
    data: procedures = [],
    isLoading: isLoadingProcedures,
    refetch: refetchProcedures,
  } = useQuery({
    queryKey: ["rpc-procedures", currentTenantId, selectedNamespace],
    queryFn: async () => {
      const data = await rpcApi.listProcedures(selectedNamespace);
      return data || [];
    },
    enabled: namespacesLoaded,
  });

  // Executions query (initial page)
  const {
    data: executionsData,
    isLoading: isLoadingExecutions,
    refetch: refetchExecutions,
  } = useQuery({
    queryKey: [
      "rpc-executions",
      currentTenantId,
      selectedNamespace,
      searchQuery,
      statusFilter,
    ],
    queryFn: async () => {
      const result = await rpcApi.listExecutions({
        namespace: selectedNamespace !== "all" ? selectedNamespace : undefined,
        procedure: searchQuery || undefined,
        status:
          statusFilter !== "all"
            ? (statusFilter as
                | "pending"
                | "running"
                | "completed"
                | "failed"
                | "cancelled"
                | "timeout")
            : undefined,
        limit: RPC_PAGE_SIZE,
        offset: 0,
      });
      return result;
    },
    enabled: activeTab === "executions" && namespacesLoaded,
    placeholderData: keepPreviousData,
  });

  const [extraExecutions, setExtraExecutions] = useState<RPCExecution[]>([]);

  // Merge initial page + extra pages
  const executions = useMemo(() => {
    const initial = executionsData?.executions || [];
    return [...initial, ...extraExecutions];
  }, [executionsData, extraExecutions]);

  const totalExecutions = executionsData?.total || 0;

  // Update hasMore when initial data changes
  useMemo(() => {
    const initial = executionsData?.executions || [];
    setHasMoreExecutions(initial.length >= RPC_PAGE_SIZE);
    setExtraExecutions([]);
    executionsOffsetRef.current = RPC_PAGE_SIZE;
  }, [executionsData]);

  // --- Mutations ---

  const syncMutation = useMutation({
    mutationFn: async (namespace: string) => {
      return await rpcApi.sync(namespace);
    },
    onSuccess: (result) => {
      const { created, updated, deleted } = result.summary;
      if (created > 0 || updated > 0 || deleted > 0) {
        const messages = [];
        if (created > 0) messages.push(`${created} created`);
        if (updated > 0) messages.push(`${updated} updated`);
        if (deleted > 0) messages.push(`${deleted} deleted`);
        toast.success(`Procedures synced: ${messages.join(", ")}`);
      } else {
        toast.info("No changes detected");
      }
      queryClient.invalidateQueries({ queryKey: ["rpc-procedures"] });
    },
    onError: () => {
      toast.error("Failed to sync procedures");
    },
  });

  const toggleMutation = useMutation({
    mutationFn: async (proc: RPCProcedure) => {
      return await rpcApi.updateProcedure(proc.namespace, proc.name, {
        enabled: !proc.enabled,
      });
    },
    onSuccess: (_data, proc) => {
      queryClient.invalidateQueries({ queryKey: ["rpc-procedures"] });
      toast.success(`Procedure ${proc.enabled ? "disabled" : "enabled"}`);
    },
    onError: () => {
      toast.error("Failed to update procedure");
    },
  });

  const cancelMutation = useMutation({
    mutationFn: async (execId: string) => {
      await rpcApi.cancelExecution(execId);
      return execId;
    },
    onSuccess: (execId) => {
      toast.success("Execution cancelled");
      queryClient.invalidateQueries({ queryKey: ["rpc-executions"] });
      if (selectedExecution?.id === execId) {
        setSelectedExecution((prev) =>
          prev ? { ...prev, status: "cancelled" } : null,
        );
      }
    },
    onError: () => {
      toast.error("Failed to cancel execution");
    },
  });

  // --- Handlers ---

  const openExecutionDetails = (exec: RPCExecution) => {
    setSelectedExecution(exec);
    setShowExecutionDetails(true);
  };

  const cancelExecution = (execId: string, e?: React.MouseEvent) => {
    e?.stopPropagation();
    setCancellingExecutionId(execId);
    cancelMutation.mutate(execId, {
      onSettled: () => setCancellingExecutionId(null),
    });
  };

  const openProcedureDetails = async (proc: RPCProcedure) => {
    setSelectedProcedure(proc);
    setShowProcedureDetails(true);
    setLoadingProcedure(true);
    try {
      const fullProcedure = await rpcApi.getProcedure(
        proc.namespace,
        proc.name,
      );
      setSelectedProcedure(fullProcedure);
    } catch {
      toast.error("Failed to fetch procedure details");
    } finally {
      setLoadingProcedure(false);
    }
  };

  const handleSync = () => {
    syncMutation.mutate(selectedNamespace);
  };

  const toggleProcedure = (proc: RPCProcedure) => {
    toggleMutation.mutate(proc);
  };

  const loadMoreExecutions = async () => {
    setLoadingMoreExecutions(true);
    try {
      const offset = executionsOffsetRef.current;
      const result = await rpcApi.listExecutions({
        namespace: selectedNamespace !== "all" ? selectedNamespace : undefined,
        procedure: searchQuery || undefined,
        status:
          statusFilter !== "all"
            ? (statusFilter as
                | "pending"
                | "running"
                | "completed"
                | "failed"
                | "cancelled"
                | "timeout")
            : undefined,
        limit: RPC_PAGE_SIZE,
        offset,
      });
      const execList = result.executions || [];
      setExtraExecutions((prev) => [...prev, ...execList]);
      executionsOffsetRef.current += RPC_PAGE_SIZE;
      setHasMoreExecutions(execList.length >= RPC_PAGE_SIZE);
    } catch {
      toast.error("Failed to load more executions");
    } finally {
      setLoadingMoreExecutions(false);
    }
  };

  const copyToClipboard = (text: string, label: string) => {
    navigator.clipboard.writeText(text);
    toast.success(`${label} copied to clipboard`);
  };

  // --- Namespace loading effect ---
  // Using a ref to avoid re-running on every render
  const namespaceFetchRef = useRef(false);
  if (!namespacesLoaded && !namespaceFetchRef.current) {
    namespaceFetchRef.current = true;
    (async () => {
      try {
        const data = await rpcApi.listNamespaces();
        const validNamespaces = data.length > 0 ? data : ["default"];
        setNamespaces(validNamespaces);

        let bestNamespace = validNamespaces[0] || "default";

        if (validNamespaces.includes("default") && validNamespaces.length > 1) {
          try {
            const [defaultProcs, defaultExecs] = await Promise.all([
              rpcApi.listProcedures("default"),
              rpcApi.listExecutions({ namespace: "default", limit: 1 }),
            ]);
            const defaultHasContent =
              (defaultProcs && defaultProcs.length > 0) ||
              defaultExecs.total > 0;
            if (!defaultHasContent) {
              for (const ns of validNamespaces) {
                if (ns !== "default") {
                  const [nsProcs, nsExecs] = await Promise.all([
                    rpcApi.listProcedures(ns),
                    rpcApi.listExecutions({ namespace: ns, limit: 1 }),
                  ]);
                  if ((nsProcs && nsProcs.length > 0) || nsExecs.total > 0) {
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

        if (!validNamespaces.includes(selectedNamespace)) {
          setSelectedNamespace(bestNamespace);
        }
      } catch {
        setNamespaces(["default"]);
      } finally {
        setNamespacesLoaded(true);
      }
    })();
  }

  // --- Derived data ---

  const executions24h = useMemo(() => {
    const cutoff = Date.now() - 24 * 60 * 60 * 1000;
    return executions.filter((exec) => {
      const execTime = new Date(exec.created_at).getTime();
      return execTime >= cutoff;
    });
  }, [executions]);

  const executionStats = useMemo(() => {
    const pending = executions24h.filter((e) => e.status === "pending").length;
    const running = executions24h.filter((e) => e.status === "running").length;
    const completed = executions24h.filter(
      (e) => e.status === "completed",
    ).length;
    const failed = executions24h.filter(
      (e) =>
        e.status === "failed" ||
        e.status === "cancelled" ||
        e.status === "timeout",
    ).length;
    const total = executions24h.length;
    const avgDuration =
      executions24h.length > 0
        ? Math.round(
            executions24h.reduce((sum, e) => sum + (e.duration_ms || 0), 0) /
              executions24h.length,
          )
        : 0;
    return { pending, running, completed, failed, total, avgDuration };
  }, [executions24h]);

  if (isLoadingProcedures) {
    return (
      <div className="flex h-96 items-center justify-center">
        <RefreshCw className="text-muted-foreground h-8 w-8 animate-spin" />
      </div>
    );
  }

  return (
    <>
      <Card className="!gap-0 !py-0 mb-6">
        <CardContent className="px-4 py-2">
          <div className="flex items-center gap-4">
            <span className="text-muted-foreground text-xs">
              (Past 24 hours)
            </span>
            <div className="flex items-center gap-1">
              <span className="text-muted-foreground text-xs">Pending:</span>
              <span className="text-sm font-semibold">
                {executionStats.pending}
              </span>
            </div>
            <div className="flex items-center gap-1">
              <span className="text-muted-foreground text-xs">Running:</span>
              <span className="text-sm font-semibold">
                {executionStats.running}
              </span>
            </div>
            <div className="flex items-center gap-1">
              <span className="text-muted-foreground text-xs">Completed:</span>
              <span className="text-sm font-semibold">
                {executionStats.completed}
              </span>
            </div>
            <div className="flex items-center gap-1">
              <span className="text-muted-foreground text-xs">Failed:</span>
              <span className="text-sm font-semibold">
                {executionStats.failed}
              </span>
            </div>
            <div className="flex items-center gap-1">
              <span className="text-muted-foreground text-xs">Success:</span>
              {(() => {
                const total = executionStats.completed + executionStats.failed;
                const successRate =
                  total > 0
                    ? ((executionStats.completed / total) * 100).toFixed(0)
                    : "0";
                return (
                  <span className="text-sm font-semibold">{successRate}%</span>
                );
              })()}
            </div>
            <div className="flex items-center gap-1">
              <span className="text-muted-foreground text-xs">
                Avg. Duration:
              </span>
              <span className="text-sm font-semibold">
                {executionStats.avgDuration}ms
              </span>
            </div>
          </div>
        </CardContent>
      </Card>

      <Tabs
        value={activeTab}
        onValueChange={(v) => setActiveTab(v as "executions" | "procedures")}
        className="flex min-h-0 flex-1 flex-col"
      >
        <div className="mb-4 flex items-center justify-between">
          <TabsList className="grid w-full max-w-md grid-cols-2">
            <TabsTrigger value="executions">
              <Activity className="mr-2 h-4 w-4" />
              Execution Logs
            </TabsTrigger>
            <TabsTrigger value="procedures">
              <Terminal className="mr-2 h-4 w-4" />
              Procedures
            </TabsTrigger>
          </TabsList>
        </div>

        <TabsContent value="executions" className="mt-0 flex-1">
          <div className="mb-4 flex items-center gap-3">
            <div className="flex items-center gap-2">
              <Label
                htmlFor="exec-namespace-select"
                className="text-muted-foreground text-sm whitespace-nowrap"
              >
                Namespace:
              </Label>
              <Select
                value={selectedNamespace}
                onValueChange={setSelectedNamespace}
              >
                <SelectTrigger id="exec-namespace-select" className="w-[150px]">
                  <SelectValue placeholder="Select namespace" />
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
            <div className="relative max-w-xs flex-1">
              <Search className="text-muted-foreground absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2" />
              <Input
                placeholder="Search by procedure name..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-9"
              />
            </div>
            <Select value={statusFilter} onValueChange={setStatusFilter}>
              <SelectTrigger className="w-[150px]">
                <Filter className="mr-2 h-4 w-4" />
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Status</SelectItem>
                <SelectItem value="pending">Pending</SelectItem>
                <SelectItem value="running">Running</SelectItem>
                <SelectItem value="completed">Completed</SelectItem>
                <SelectItem value="failed">Failed</SelectItem>
                <SelectItem value="cancelled">Cancelled</SelectItem>
                <SelectItem value="timeout">Timeout</SelectItem>
              </SelectContent>
            </Select>
            <Button
              onClick={() => refetchExecutions()}
              variant="outline"
              size="sm"
            >
              <RefreshCw className="mr-2 h-4 w-4" />
              Refresh
            </Button>
          </div>

          <ScrollArea className="h-[calc(100vh-24rem)]">
            {isLoadingExecutions ? (
              <div className="flex h-48 items-center justify-center">
                <Loader2 className="text-muted-foreground h-8 w-8 animate-spin" />
              </div>
            ) : executions.length === 0 ? (
              <Card>
                <CardContent className="p-12 text-center">
                  <Activity className="text-muted-foreground mx-auto mb-4 h-12 w-12" />
                  <p className="mb-2 text-lg font-medium">
                    No executions found
                  </p>
                  <p className="text-muted-foreground text-sm">
                    Execute some RPC procedures to see their logs here
                  </p>
                </CardContent>
              </Card>
            ) : (
              <div className="grid gap-1">
                {executions.map((exec) => (
                  <div
                    key={exec.id}
                    className="hover:border-primary/50 bg-card flex cursor-pointer items-center justify-between gap-2 rounded-md border px-3 py-2 transition-colors"
                    onClick={() => openExecutionDetails(exec)}
                  >
                    <div className="flex min-w-0 flex-1 items-center gap-3">
                      {getStatusIcon(exec.status)}
                      <span className="truncate text-sm font-medium">
                        {exec.procedure_name}
                      </span>
                      <Badge
                        variant={getStatusVariant(exec.status)}
                        className="h-4 shrink-0 px-1.5 py-0 text-[10px]"
                      >
                        {exec.status}
                      </Badge>
                      {exec.user_email && (
                        <span className="text-muted-foreground truncate text-xs">
                          {exec.user_email}
                        </span>
                      )}
                    </div>
                    <div className="flex shrink-0 items-center gap-3">
                      {exec.rows_returned !== undefined && (
                        <span className="text-muted-foreground text-xs">
                          {exec.rows_returned} rows
                        </span>
                      )}
                      <span className="text-muted-foreground text-xs">
                        {exec.duration_ms ? `${exec.duration_ms}ms` : "-"}
                      </span>
                      <span className="text-muted-foreground text-xs">
                        {new Date(exec.created_at).toLocaleString()}
                      </span>
                      {canCancelExecution(exec.status) && (
                        <Button
                          variant="ghost"
                          size="sm"
                          className="text-destructive hover:text-destructive hover:bg-destructive/10 h-6 px-2"
                          onClick={(e) => cancelExecution(exec.id, e)}
                          disabled={cancellingExecutionId === exec.id}
                        >
                          {cancellingExecutionId === exec.id ? (
                            <Loader2 className="h-3 w-3 animate-spin" />
                          ) : (
                            <StopCircle className="h-3 w-3" />
                          )}
                          <span className="ml-1 text-xs">Cancel</span>
                        </Button>
                      )}
                    </div>
                  </div>
                ))}
                {hasMoreExecutions && (
                  <div className="mt-4 flex flex-col items-center gap-2">
                    <span className="text-muted-foreground text-xs">
                      Showing {executions.length} of {totalExecutions}{" "}
                      executions
                    </span>
                    <Button
                      variant="outline"
                      onClick={loadMoreExecutions}
                      disabled={loadingMoreExecutions}
                    >
                      {loadingMoreExecutions ? (
                        <>
                          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                          Loading...
                        </>
                      ) : (
                        "Load More"
                      )}
                    </Button>
                  </div>
                )}
              </div>
            )}
          </ScrollArea>
        </TabsContent>

        <TabsContent value="procedures" className="mt-0 flex-1">
          <div className="mb-4 flex items-center justify-end gap-2">
            <div className="flex items-center gap-2">
              <Label
                htmlFor="proc-namespace-select"
                className="text-muted-foreground text-sm whitespace-nowrap"
              >
                Namespace:
              </Label>
              <Select
                value={selectedNamespace}
                onValueChange={setSelectedNamespace}
              >
                <SelectTrigger id="proc-namespace-select" className="w-[180px]">
                  <SelectValue placeholder="Select namespace" />
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
              onClick={handleSync}
              variant="outline"
              size="sm"
              disabled={syncMutation.isPending}
            >
              {syncMutation.isPending ? (
                <>
                  <RefreshCw className="mr-2 h-4 w-4 animate-spin" />
                  Syncing...
                </>
              ) : (
                <>
                  <HardDrive className="mr-2 h-4 w-4" />
                  Sync from Filesystem
                </>
              )}
            </Button>
            <Button
              onClick={() => refetchProcedures()}
              variant="outline"
              size="sm"
            >
              <RefreshCw className="mr-2 h-4 w-4" />
              Refresh
            </Button>
          </div>

          <div className="mb-4 flex gap-4 text-sm">
            <div className="flex items-center gap-1.5">
              <span className="text-muted-foreground">Total:</span>
              <Badge variant="secondary" className="h-5 px-2">
                {procedures.length}
              </Badge>
            </div>
            <div className="flex items-center gap-1.5">
              <span className="text-muted-foreground">Enabled:</span>
              <Badge
                variant="secondary"
                className="h-5 bg-green-500/10 px-2 text-green-600 dark:text-green-400"
              >
                {procedures.filter((p) => p.enabled).length}
              </Badge>
            </div>
            <div className="flex items-center gap-1.5">
              <span className="text-muted-foreground">Public:</span>
              <Badge variant="secondary" className="h-5 px-2">
                {procedures.filter((p) => p.is_public).length}
              </Badge>
            </div>
          </div>

          <ScrollArea className="h-[calc(100vh-20rem)]">
            <div className="grid gap-1">
              {procedures.length === 0 ? (
                <Card>
                  <CardContent className="p-12 text-center">
                    <Terminal className="text-muted-foreground mx-auto mb-4 h-12 w-12" />
                    <p className="mb-2 text-lg font-medium">
                      No RPC procedures yet
                    </p>
                    <p className="text-muted-foreground mb-4 text-sm">
                      Sync procedures from the filesystem to get started
                    </p>
                    <Button
                      onClick={handleSync}
                      disabled={syncMutation.isPending}
                    >
                      <HardDrive className="mr-2 h-4 w-4" />
                      Sync from Filesystem
                    </Button>
                  </CardContent>
                </Card>
              ) : (
                procedures.map((proc) => (
                  <div
                    key={proc.id}
                    className="hover:border-primary/50 bg-card flex cursor-pointer items-center justify-between gap-2 rounded-md border px-3 py-2 transition-colors"
                    onClick={() => openProcedureDetails(proc)}
                  >
                    <div className="flex min-w-0 flex-1 items-center gap-3">
                      <span className="truncate text-sm font-medium">
                        {proc.name}
                      </span>
                      <Badge
                        variant="outline"
                        className="h-4 shrink-0 px-1 py-0 text-[10px]"
                      >
                        v{proc.version}
                      </Badge>
                      {proc.is_public ? (
                        <Badge
                          variant="outline"
                          className="h-4 shrink-0 px-1 py-0 text-[10px]"
                        >
                          <Globe className="mr-0.5 h-2.5 w-2.5" />
                          public
                        </Badge>
                      ) : (
                        <Badge
                          variant="outline"
                          className="h-4 shrink-0 px-1 py-0 text-[10px]"
                        >
                          <Lock className="mr-0.5 h-2.5 w-2.5" />
                          private
                        </Badge>
                      )}
                      {proc.require_role && (
                        <Badge
                          variant="outline"
                          className="h-4 shrink-0 px-1 py-0 text-[10px]"
                        >
                          role: {proc.require_role}
                        </Badge>
                      )}
                      {proc.schedule && (
                        <Badge
                          variant="outline"
                          className="h-4 shrink-0 px-1 py-0 text-[10px]"
                        >
                          <Timer className="mr-0.5 h-2.5 w-2.5" />
                          scheduled
                        </Badge>
                      )}
                      <Switch
                        checked={proc.enabled}
                        onCheckedChange={() => toggleProcedure(proc)}
                        onClick={(e) => e.stopPropagation()}
                        className="scale-75"
                      />
                    </div>
                    <div className="flex shrink-0 items-center gap-2">
                      {proc.source === "filesystem" && proc.updated_at && (
                        <span
                          className="text-muted-foreground text-[10px]"
                          title={`Last synced: ${new Date(proc.updated_at).toLocaleString()}`}
                        >
                          synced{" "}
                          {new Date(proc.updated_at).toLocaleDateString()}
                        </span>
                      )}
                      <span className="text-muted-foreground text-[10px]">
                        {proc.max_execution_time_seconds}s max
                      </span>
                      {proc.description && (
                        <span
                          className="text-muted-foreground max-w-[200px] truncate text-xs"
                          title={proc.description}
                        >
                          {proc.description}
                        </span>
                      )}
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-6 px-2"
                        onClick={(e) => {
                          e.stopPropagation();
                          openProcedureDetails(proc);
                        }}
                      >
                        <Eye className="h-3 w-3" />
                        <span className="ml-1 text-xs">View</span>
                      </Button>
                    </div>
                  </div>
                ))
              )}
            </div>
          </ScrollArea>
        </TabsContent>
      </Tabs>

      <ExecutionDetailsDialog
        open={isExecutionDetailsOpen}
        onOpenChange={setShowExecutionDetails}
        execution={selectedExecution}
        logs={executionLogs}
        loadingLogs={loadingLogs}
        cancellingExecutionId={cancellingExecutionId}
        onCancelExecution={(execId: string) => cancelExecution(execId)}
        onCopy={copyToClipboard}
      />

      <ProcedureDetailsDialog
        open={showProcedureDetails}
        onOpenChange={setShowProcedureDetails}
        procedure={selectedProcedure}
        loading={loadingProcedure}
        onCopy={copyToClipboard}
      />
    </>
  );
}
