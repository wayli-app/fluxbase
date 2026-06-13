import { useState, useEffect, useCallback, useRef, useMemo } from "react";
import z from "zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import {
  ListTodo,
  Search,
  RefreshCw,
  Clock,
  XCircle,
  Activity,
  CheckCircle,
  AlertCircle,
  Loader2,
  Filter,
  HardDrive,
  Timer,
  Target,
  Play,
  ChevronDown,
  History,
  Edit,
  Trash2,
} from "lucide-react";
import { toast } from "sonner";
import { useImpersonationStore } from "@/stores/impersonation-store";
import { useTenantStore } from "@/stores/tenant-store";
import { jobsApi, type JobFunction, type Job, type JobWorker } from "@/lib/api";
import { fluxbaseClient } from "@/lib/fluxbase-client";
import { useExecutionLogs } from "@/hooks/use-execution-logs";
import { Badge } from "@/components/ui/badge";
import { PageHeader } from "@/components/layout/page-header";
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
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { ImpersonationPopover } from "@/features/impersonation/components/impersonation-popover";
import {
  JobDetailsDialog,
  RunJobDialog,
  EditJobDialog,
  DeleteConfirmDialog,
  ExecutionHistoryDialog,
  type DeleteConfirm,
  type EditFormData,
} from "@/components/jobs";

const jobsSearchSchema = z.object({
  tab: z.string().optional().catch("queue"),
  namespace: z.string().optional().catch("default"),
});

export const Route = createFileRoute("/_authenticated/jobs/")({
  validateSearch: jobsSearchSchema,
  component: JobsPage,
});

const JOBS_PAGE_SIZE = 50;

const getStatusIcon = (status: string) => {
  switch (status) {
    case "completed":
      return <CheckCircle className="h-4 w-4 text-green-500" />;
    case "failed":
      return <AlertCircle className="h-4 w-4 text-red-500" />;
    case "running":
      return <Loader2 className="h-4 w-4 animate-spin text-blue-500" />;
    case "pending":
      return <Clock className="h-4 w-4 text-yellow-500" />;
    case "cancelled":
      return <XCircle className="h-4 w-4 text-gray-500" />;
    default:
      return <Activity className="h-4 w-4" />;
  }
};

const getStatusBadgeVariant = (
  status: string,
): "default" | "secondary" | "destructive" | "outline" => {
  switch (status) {
    case "completed":
      return "default";
    case "failed":
      return "destructive";
    case "running":
      return "secondary";
    default:
      return "outline";
  }
};

function JobsPage() {
  const currentTenantId = useTenantStore((state) => state.currentTenant?.id);
  const queryClient = useQueryClient();
  const [activeTab, setActiveTab] = useState<"functions" | "queue">("queue");
  const [jobs, setJobs] = useState<Job[]>([]);
  const [workers, setWorkers] = useState<JobWorker[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState<string>("all");
  const [selectedJob, setSelectedJob] = useState<Job | null>(null);
  const [isJobDetailsOpen, setIsJobDetailsOpen] = useState(false);
  const [syncing, setSyncing] = useState(false);
  const [namespaces, setNamespaces] = useState<string[]>(["default"]);
  const [selectedNamespace, setSelectedNamespace] = useState<string>("default");

  // Job functions via TanStack Query
  const { data: jobFunctions = [], refetch: refetchJobFunctions } = useQuery({
    queryKey: ["job-functions", selectedNamespace, currentTenantId],
    queryFn: async () => {
      const data = await jobsApi.listFunctions(selectedNamespace);
      return data || [];
    },
  });

  const [jobsOffset, setJobsOffset] = useState(0);
  const [hasMoreJobs, setHasMoreJobs] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);

  const [isRunDialogOpen, setIsRunDialogOpen] = useState(false);
  const [selectedFunction, setSelectedFunction] = useState<JobFunction | null>(
    null,
  );
  const [jobPayload, setJobPayload] = useState("");
  const [submittingJob, setSubmittingJob] = useState(false);
  const [togglingJob, setTogglingJob] = useState<string | null>(null);

  const [isEditDialogOpen, setIsEditDialogOpen] = useState(false);
  const [fetchingFunction, setFetchingFunction] = useState(false);
  const [editFormData, setEditFormData] = useState<EditFormData>({
    description: "",
    code: "",
    timeout_seconds: 30,
    max_retries: 3,
    schedule: "",
  });

  const [deleteConfirm, setDeleteConfirm] = useState<DeleteConfirm>(null);

  const [isHistoryDialogOpen, setIsHistoryDialogOpen] = useState(false);
  const [historyJobs, setHistoryJobs] = useState<Job[]>([]);
  const [historyLoading, setHistoryLoading] = useState(false);

  const [logLevelFilter, setLogLevelFilter] = useState<
    "debug" | "info" | "warning" | "error" | "fatal" | "all"
  >("all");

  const logsContainerRef = useRef<HTMLDivElement>(null);
  const isAtBottomRef = useRef<boolean>(true);

  const checkIfAtBottom = () => {
    if (!logsContainerRef.current) return true;
    const { scrollTop, scrollHeight, clientHeight } = logsContainerRef.current;
    return scrollHeight - scrollTop - clientHeight < 20;
  };

  const { logs: executionLogs, loading: loadingLogs } = useExecutionLogs({
    executionId: selectedJob?.id || null,
    executionType: "job",
    enabled: isJobDetailsOpen,
    onNewLog: () => {
      isAtBottomRef.current = checkIfAtBottom();
      setTimeout(() => {
        if (isAtBottomRef.current && logsContainerRef.current) {
          logsContainerRef.current.scrollTop =
            logsContainerRef.current.scrollHeight;
        }
      }, 50);
    },
  });

  useEffect(() => {
    const fetchNamespaces = async () => {
      try {
        const data = await jobsApi.listNamespaces();
        const validNamespaces = data.length > 0 ? data : ["default"];
        setNamespaces(validNamespaces);

        let bestNamespace = validNamespaces[0] || "default";

        if (validNamespaces.includes("default") && validNamespaces.length > 1) {
          try {
            const defaultJobs = await jobsApi.listFunctions("default");
            if (!defaultJobs || defaultJobs.length === 0) {
              for (const ns of validNamespaces) {
                if (ns !== "default") {
                  const nsJobs = await jobsApi.listFunctions(ns);
                  if (nsJobs && nsJobs.length > 0) {
                    bestNamespace = ns;
                    break;
                  }
                }
              }
            }
          } catch {
            /* Intentionally empty: namespace probe failed */
          }
        }

        if (!validNamespaces.includes(selectedNamespace)) {
          setSelectedNamespace(bestNamespace);
        }
      } catch {
        setNamespaces(["default"]);
      }
    };
    fetchNamespaces();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [currentTenantId]);

  const fetchJobs = useCallback(
    async (reset = true) => {
      try {
        const offset = reset ? 0 : jobsOffset;
        const filters: {
          status?: string;
          namespace?: string;
          limit: number;
          offset: number;
        } = {
          limit: JOBS_PAGE_SIZE,
          offset,
          namespace: selectedNamespace,
        };
        if (statusFilter !== "all") {
          filters.status = statusFilter;
        }
        const data = await jobsApi.listJobs(filters);
        const newJobs = data || [];

        if (reset) {
          setJobs(newJobs);
          setJobsOffset(JOBS_PAGE_SIZE);
        } else {
          setJobs((prev) => [...prev, ...newJobs]);
          setJobsOffset((prev) => prev + JOBS_PAGE_SIZE);
        }

        setHasMoreJobs(newJobs.length >= JOBS_PAGE_SIZE);
      } catch {
        toast.error("Failed to fetch jobs");
      }
    },
    [selectedNamespace, statusFilter, jobsOffset],
  );

  const loadMoreJobs = useCallback(async () => {
    setLoadingMore(true);
    try {
      await fetchJobs(false);
    } finally {
      setLoadingMore(false);
    }
  }, [fetchJobs]);

  useEffect(() => {
    if (!isJobDetailsOpen || !selectedJob) return;

    const isActiveJob =
      selectedJob.status === "running" || selectedJob.status === "pending";
    if (!isActiveJob) return;

    const channel = fluxbaseClient
      .channel(`job-details-${selectedJob.id}`)
      .on(
        "postgres_changes",
        {
          event: "UPDATE",
          schema: "jobs",
          table: "queue",
          filter: `id=eq.${selectedJob.id}`,
        },
        (payload) => {
          const updatedJob = payload.new as Job;
          setSelectedJob(updatedJob);

          if (
            updatedJob.status !== "running" &&
            updatedJob.status !== "pending"
          ) {
            fetchJobs(true);
          }
        },
      )
      .subscribe();

    return () => {
      channel.unsubscribe();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isJobDetailsOpen, selectedJob?.id, selectedJob?.status, fetchJobs]);

  useEffect(() => {
    if (isJobDetailsOpen && selectedJob?.id) {
      isAtBottomRef.current = true;
    }
  }, [isJobDetailsOpen, selectedJob?.id]);

  useEffect(() => {
    const channel = fluxbaseClient
      .channel("jobs-queue-updates")
      .on(
        "postgres_changes",
        {
          event: "*",
          schema: "jobs",
          table: "queue",
        },
        (payload) => {
          const eventType = payload.eventType;
          const newJob = payload.new as Job | undefined;
          const oldJob = payload.old as { id: string } | undefined;

          setJobs((prev) => {
            if (eventType === "INSERT" && newJob) {
              if (selectedNamespace && newJob.namespace !== selectedNamespace) {
                return prev;
              }
              if (statusFilter !== "all" && newJob.status !== statusFilter) {
                return prev;
              }
              if (prev.some((j) => j.id === newJob.id)) {
                return prev;
              }
              return [newJob, ...prev];
            }

            if (eventType === "UPDATE" && newJob) {
              const idx = prev.findIndex((j) => j.id === newJob.id);
              if (idx === -1) {
                if (
                  selectedNamespace &&
                  newJob.namespace !== selectedNamespace
                ) {
                  return prev;
                }
                if (statusFilter !== "all" && newJob.status !== statusFilter) {
                  return prev;
                }
                return [newJob, ...prev];
              }
              if (statusFilter !== "all" && newJob.status !== statusFilter) {
                return prev.filter((j) => j.id !== newJob.id);
              }
              const updated = [...prev];
              updated[idx] = newJob;
              return updated;
            }

            if (eventType === "DELETE" && oldJob) {
              return prev.filter((j) => j.id !== oldJob.id);
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

  const fetchWorkers = useCallback(async () => {
    try {
      const data = await jobsApi.listWorkers();
      setWorkers(data || []);
    } catch {
      /* Intentionally empty: workers fetch failed */
    }
  }, []);

  const refreshAllData = useCallback(async () => {
    setLoading(true);
    setJobsOffset(0);
    setHasMoreJobs(true);
    try {
      await Promise.all([
        refetchJobFunctions(),
        fetchJobs(true),
        fetchWorkers(),
      ]);
    } finally {
      setLoading(false);
    }
  }, [refetchJobFunctions, fetchJobs, fetchWorkers]);

  useEffect(() => {
    const loadInitialData = async () => {
      setLoading(true);
      try {
        const nsData = await jobsApi.listNamespaces();
        const availableNamespaces = nsData.length > 0 ? nsData : ["default"];
        setNamespaces(availableNamespaces);

        const ns = availableNamespaces.includes("default")
          ? "default"
          : availableNamespaces[0];

        const [jobsData, workersData] = await Promise.all([
          jobsApi.listJobs({ namespace: ns, limit: JOBS_PAGE_SIZE, offset: 0 }),
          jobsApi.listWorkers(),
        ]);

        setJobs(jobsData || []);
        setJobsOffset(JOBS_PAGE_SIZE);
        setHasMoreJobs((jobsData || []).length >= JOBS_PAGE_SIZE);
        setWorkers(workersData || []);
      } catch {
        toast.error("Failed to load jobs data");
      } finally {
        setLoading(false);
      }
    };
    loadInitialData();
  }, []);

  const isInitialMount = useRef(true);
  useEffect(() => {
    if (isInitialMount.current) {
      isInitialMount.current = false;
      return;
    }

    const refetchData = async () => {
      setLoading(true);
      setJobsOffset(0);
      setHasMoreJobs(true);
      try {
        const jobsData = await jobsApi.listJobs({
          namespace: selectedNamespace,
          status: statusFilter !== "all" ? statusFilter : undefined,
          limit: JOBS_PAGE_SIZE,
          offset: 0,
        });
        setJobs(jobsData || []);
        setJobsOffset(JOBS_PAGE_SIZE);
        setHasMoreJobs((jobsData || []).length >= JOBS_PAGE_SIZE);
      } catch {
        toast.error("Failed to fetch jobs");
      } finally {
        setLoading(false);
      }
    };
    refetchData();
  }, [selectedNamespace, statusFilter]);

  const handleSync = async () => {
    setSyncing(true);
    try {
      const result = await jobsApi.sync(selectedNamespace);
      const { created, updated, deleted, errors } = result.summary;

      if (errors > 0) {
        toast.error(`Sync completed with ${errors} errors`);
      } else if (created > 0 || updated > 0 || deleted > 0) {
        const messages = [];
        if (created > 0) messages.push(`${created} created`);
        if (updated > 0) messages.push(`${updated} updated`);
        if (deleted > 0) messages.push(`${deleted} deleted`);
        toast.success(
          `Jobs synced to "${selectedNamespace}": ${messages.join(", ")}`,
        );
      } else {
        toast.info("No changes detected");
      }

      const newNamespaces = await jobsApi.listNamespaces();
      setNamespaces(newNamespaces.length > 0 ? newNamespaces : ["default"]);

      queryClient.invalidateQueries({ queryKey: ["job-functions"] });
    } catch {
      toast.error("Failed to sync jobs from filesystem");
    } finally {
      setSyncing(false);
    }
  };

  const viewJobDetails = async (job: Job) => {
    try {
      const data = await jobsApi.getJob(job.id);
      setSelectedJob(data);
      setLogLevelFilter("all");
      setIsJobDetailsOpen(true);
    } catch {
      toast.error("Failed to fetch job details");
    }
  };

  const cancelJob = async (jobId: string) => {
    try {
      await jobsApi.cancelJob(jobId);
      toast.success("Job cancelled");
      fetchJobs();
    } catch {
      toast.error("Failed to cancel job");
    }
  };

  const resubmitJob = async (jobId: string) => {
    try {
      const newJob = await jobsApi.resubmitJob(jobId);
      toast.success(`Job resubmitted (new ID: ${newJob.id.slice(0, 8)}...)`);
      fetchJobs();
      if (isJobDetailsOpen) {
        setIsJobDetailsOpen(false);
        setSelectedJob(null);
      }
    } catch {
      toast.error("Failed to resubmit job");
    }
  };

  const openRunDialog = (fn: JobFunction) => {
    setSelectedFunction(fn);
    setJobPayload("{\n  \n}");
    setIsRunDialogOpen(true);
  };

  const handleSubmitJob = async () => {
    if (!selectedFunction) return;

    setSubmittingJob(true);
    try {
      let payload: Record<string, unknown> = {};
      if (jobPayload.trim()) {
        try {
          payload = JSON.parse(jobPayload);
        } catch {
          toast.error("Invalid JSON payload");
          setSubmittingJob(false);
          return;
        }
      }

      const { isImpersonating, impersonationToken } =
        useImpersonationStore.getState();
      const config: { headers?: Record<string, string> } = {};
      if (isImpersonating && impersonationToken) {
        config.headers = { "X-Impersonation-Token": impersonationToken };
      }

      const job = await jobsApi.submitJob(
        {
          job_name: selectedFunction.name,
          namespace: selectedNamespace,
          payload,
        },
        config,
      );

      toast.success(
        `Job submitted successfully (ID: ${job.id.slice(0, 8)}...)`,
      );
      setIsRunDialogOpen(false);
      setSelectedFunction(null);
      setJobPayload("");

      setActiveTab("queue");
      await fetchJobs();
    } catch {
      toast.error("Failed to submit job");
    } finally {
      setSubmittingJob(false);
    }
  };

  const toggleJobEnabled = async (fn: JobFunction) => {
    setTogglingJob(fn.id);
    try {
      await jobsApi.updateFunction(fn.namespace, fn.name, {
        enabled: !fn.enabled,
      });
      toast.success(`Job "${fn.name}" ${fn.enabled ? "disabled" : "enabled"}`);
      queryClient.invalidateQueries({ queryKey: ["job-functions"] });
    } catch {
      toast.error("Failed to update job function");
    } finally {
      setTogglingJob(null);
    }
  };

  const viewHistory = async (fn: JobFunction) => {
    setSelectedFunction(fn);
    setHistoryLoading(true);
    setIsHistoryDialogOpen(true);
    try {
      const jobs = await jobsApi.listJobs({
        namespace: fn.namespace,
        limit: 50,
        offset: 0,
      });
      const functionJobs = jobs.filter((j) => j.job_name === fn.name);
      setHistoryJobs(functionJobs);
    } catch {
      toast.error("Failed to fetch execution history");
    } finally {
      setHistoryLoading(false);
    }
  };

  const openEditDialog = async (fn: JobFunction) => {
    setSelectedFunction(fn);
    setFetchingFunction(true);
    setIsEditDialogOpen(true);
    try {
      const fullFunction = await jobsApi.getFunction(fn.namespace, fn.name);
      setEditFormData({
        description: fullFunction.description || "",
        code: fullFunction.code || "",
        timeout_seconds: fullFunction.timeout_seconds,
        max_retries: fullFunction.max_retries,
        schedule: fullFunction.schedule || "",
      });
    } catch {
      toast.error("Failed to load job function details");
      setIsEditDialogOpen(false);
    } finally {
      setFetchingFunction(false);
    }
  };

  const updateJobFunction = async () => {
    if (!selectedFunction) return;
    try {
      await jobsApi.updateFunction(
        selectedFunction.namespace,
        selectedFunction.name,
        {
          description: editFormData.description || undefined,
          code: editFormData.code || undefined,
          timeout_seconds: editFormData.timeout_seconds,
          max_retries: editFormData.max_retries,
          schedule: editFormData.schedule || undefined,
        },
      );
      toast.success("Job function updated");
      setIsEditDialogOpen(false);
      queryClient.invalidateQueries({ queryKey: ["job-functions"] });
    } catch {
      toast.error("Failed to update job function");
    }
  };

  const deleteJobFunction = async () => {
    if (!deleteConfirm) return;
    try {
      await jobsApi.deleteFunction(deleteConfirm.namespace, deleteConfirm.name);
      toast.success(`Job function "${deleteConfirm.name}" deleted`);
      setDeleteConfirm(null);
      queryClient.invalidateQueries({ queryKey: ["job-functions"] });
    } catch {
      toast.error("Failed to delete job function");
    }
  };

  const jobs24h = useMemo(() => {
    const cutoff = Date.now() - 24 * 60 * 60 * 1000;
    return jobs.filter((job) => {
      const jobTime = new Date(job.created_at).getTime();
      return jobTime >= cutoff;
    });
  }, [jobs]);

  const filteredStats = useMemo(
    () => ({
      pending: jobs24h.filter((j) => j.status === "pending").length,
      running: jobs24h.filter((j) => j.status === "running").length,
      completed: jobs24h.filter((j) => j.status === "completed").length,
      failed: jobs24h.filter((j) => j.status === "failed").length,
      cancelled: jobs24h.filter((j) => j.status === "cancelled").length,
      total: jobs24h.length,
    }),
    [jobs24h],
  );

  const filteredJobs = jobs.filter((job) => {
    if (
      searchQuery &&
      !job.job_name.toLowerCase().includes(searchQuery.toLowerCase())
    ) {
      return false;
    }
    return true;
  });

  if (loading) {
    return (
      <div className="flex h-96 items-center justify-center">
        <RefreshCw className="text-muted-foreground h-8 w-8 animate-spin" />
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col">
      <PageHeader
        icon={<ListTodo />}
        title="Background Jobs"
        description="Manage job functions and monitor background task execution"
        actions={
          <>
            <ImpersonationPopover
              contextLabel="Running as"
              defaultReason="Testing job submission"
            />
            <Button onClick={refreshAllData} variant="outline" size="sm">
              <RefreshCw className="mr-2 h-4 w-4" />
              Refresh
            </Button>
          </>
        }
      />

      <div className="flex-1 overflow-auto p-6">
        <Card className="!gap-0 !py-0 mb-6">
          <CardContent className="px-4 py-2">
            <div className="flex items-center gap-4">
              <span className="text-muted-foreground text-xs">
                (Past 24 hours)
              </span>
              <div className="flex items-center gap-1">
                <span className="text-muted-foreground text-xs">Pending:</span>
                <span className="text-sm font-semibold">
                  {filteredStats.pending}
                </span>
              </div>
              <div className="flex items-center gap-1">
                <span className="text-muted-foreground text-xs">Running:</span>
                <span className="text-sm font-semibold">
                  {filteredStats.running}
                </span>
              </div>
              <div className="flex items-center gap-1">
                <span className="text-muted-foreground text-xs">
                  Completed:
                </span>
                <span className="text-sm font-semibold">
                  {filteredStats.completed}
                </span>
              </div>
              <div className="flex items-center gap-1">
                <span className="text-muted-foreground text-xs">Failed:</span>
                <span className="text-sm font-semibold">
                  {filteredStats.failed}
                </span>
              </div>
              <div className="flex items-center gap-1">
                <span className="text-muted-foreground text-xs">Workers:</span>
                <span className="text-sm font-semibold">
                  {workers.filter((w) => w.status === "active").length}
                </span>
              </div>
              <div className="flex items-center gap-1">
                <Target className="text-muted-foreground h-3 w-3" />
                <span className="text-muted-foreground text-xs">Success:</span>
                {(() => {
                  const total = filteredStats.completed + filteredStats.failed;
                  const successRate =
                    total > 0
                      ? ((filteredStats.completed / total) * 100).toFixed(0)
                      : "0";
                  return (
                    <span className="text-sm font-semibold">
                      {successRate}%
                    </span>
                  );
                })()}
              </div>
              <div className="flex items-center gap-1">
                <Timer className="text-muted-foreground h-3 w-3" />
                <span className="text-muted-foreground text-xs">
                  Avg. Wait:
                </span>
                {(() => {
                  const pendingJobs = jobs24h.filter(
                    (j) => j.status === "pending",
                  );
                  const waitTimes = pendingJobs.map(
                    (j) => Date.now() - new Date(j.created_at).getTime(),
                  );
                  const avgWaitMs =
                    waitTimes.length > 0
                      ? waitTimes.reduce((a, b) => a + b, 0) / waitTimes.length
                      : 0;
                  const avgWaitSec = Math.round(avgWaitMs / 1000);
                  const displayTime =
                    avgWaitSec < 60
                      ? `${avgWaitSec}s`
                      : avgWaitSec < 3600
                        ? `${Math.round(avgWaitSec / 60)}m`
                        : `${Math.round(avgWaitSec / 3600)}h`;
                  return (
                    <span className="text-sm font-semibold">{displayTime}</span>
                  );
                })()}
              </div>
            </div>
          </CardContent>
        </Card>

        <Tabs
          value={activeTab}
          onValueChange={(v) => setActiveTab(v as "functions" | "queue")}
          className="flex min-h-0 flex-1 flex-col"
        >
          <TabsList className="grid w-full max-w-md grid-cols-2">
            <TabsTrigger value="queue">
              <Activity className="mr-2 h-4 w-4" />
              Job Queue
            </TabsTrigger>
            <TabsTrigger value="functions">
              <ListTodo className="mr-2 h-4 w-4" />
              Job Functions
            </TabsTrigger>
          </TabsList>

          <TabsContent
            value="queue"
            className="mt-6 flex min-h-0 flex-1 flex-col space-y-6"
          >
            <div className="flex items-center gap-3">
              <div className="flex items-center gap-2">
                <Label
                  htmlFor="queue-namespace-select"
                  className="text-muted-foreground text-sm whitespace-nowrap"
                >
                  Namespace:
                </Label>
                <Select
                  value={selectedNamespace}
                  onValueChange={setSelectedNamespace}
                >
                  <SelectTrigger
                    id="queue-namespace-select"
                    className="w-[180px]"
                  >
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
              <div className="relative flex-1">
                <Search className="text-muted-foreground absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2" />
                <Input
                  placeholder="Search jobs..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="pl-9"
                />
              </div>
              <Select
                value={statusFilter}
                onValueChange={(v) => {
                  setStatusFilter(v);
                  setJobsOffset(0);
                  setHasMoreJobs(true);
                  setTimeout(() => fetchJobs(true), 100);
                }}
              >
                <SelectTrigger className="w-[180px]">
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
                </SelectContent>
              </Select>
            </div>

            <ScrollArea className="min-h-0 flex-1">
              <div className="grid gap-4">
                {filteredJobs.length === 0 ? (
                  <Card>
                    <CardContent className="p-12 text-center">
                      <ListTodo className="text-muted-foreground mx-auto mb-4 h-12 w-12" />
                      <p className="mb-2 text-lg font-medium">No jobs found</p>
                      <p className="text-muted-foreground text-sm">
                        {searchQuery || statusFilter !== "all"
                          ? "Try adjusting your filters"
                          : "Submit a job to see it here"}
                      </p>
                    </CardContent>
                  </Card>
                ) : (
                  filteredJobs.map((job) => (
                    <div
                      key={job.id}
                      className="hover:border-primary/50 bg-card flex items-center justify-between gap-2 rounded-md border px-3 py-1.5 transition-colors"
                    >
                      <div className="flex min-w-0 flex-1 items-center gap-2">
                        {getStatusIcon(job.status)}
                        <span className="truncate text-sm font-medium">
                          {job.job_name}
                        </span>
                        <Badge
                          variant={getStatusBadgeVariant(job.status)}
                          className="h-4 shrink-0 px-1 py-0 text-[10px]"
                        >
                          {job.status}
                        </Badge>
                        {job.user_email && (
                          <span
                            className="text-muted-foreground max-w-[120px] shrink-0 truncate text-[10px]"
                            title={
                              job.user_name
                                ? `${job.user_name} (${job.user_email})`
                                : job.user_email
                            }
                          >
                            {job.user_email}
                          </span>
                        )}
                        {job.retry_count > 0 && (
                          <span className="text-muted-foreground shrink-0 text-[10px]">
                            #{job.retry_count}
                          </span>
                        )}
                        {(job.status === "running" ||
                          job.status === "pending") &&
                          job.progress_percent !== undefined && (
                            <div className="flex shrink-0 items-center gap-1">
                              <div className="bg-secondary h-1 w-16 overflow-hidden rounded-full">
                                <div
                                  className="h-full bg-blue-500 transition-all duration-300"
                                  style={{ width: `${job.progress_percent}%` }}
                                />
                              </div>
                              <span className="text-muted-foreground text-[10px]">
                                {job.progress_percent}%
                              </span>
                              {job.estimated_seconds_left !== undefined &&
                                job.estimated_seconds_left > 0 && (
                                  <span className="text-muted-foreground text-[10px]">
                                    (ETA:{" "}
                                    {job.estimated_seconds_left < 60
                                      ? `${job.estimated_seconds_left}s`
                                      : job.estimated_seconds_left < 3600
                                        ? `${Math.round(job.estimated_seconds_left / 60)}m`
                                        : `${Math.round(job.estimated_seconds_left / 3600)}h`}
                                    )
                                  </span>
                                )}
                            </div>
                          )}
                      </div>
                      <div className="flex shrink-0 items-center gap-1">
                        <span className="text-muted-foreground text-[10px]">
                          {new Date(job.created_at).toLocaleTimeString()}
                        </span>
                        <Button
                          onClick={() => viewJobDetails(job)}
                          size="sm"
                          variant="ghost"
                          className="h-6 px-1.5 text-xs"
                        >
                          View
                        </Button>
                        {(job.status === "running" ||
                          job.status === "pending") && (
                          <Button
                            onClick={() => cancelJob(job.id)}
                            size="sm"
                            variant="ghost"
                            className="h-6 w-6 p-0"
                          >
                            <XCircle className="h-3 w-3" />
                          </Button>
                        )}
                        {(job.status === "completed" ||
                          job.status === "cancelled" ||
                          job.status === "failed") && (
                          <Button
                            onClick={() => resubmitJob(job.id)}
                            size="sm"
                            variant="ghost"
                            className="h-6 w-6 p-0"
                            title="Re-submit as new job"
                          >
                            <RefreshCw className="h-3 w-3" />
                          </Button>
                        )}
                      </div>
                    </div>
                  ))
                )}

                {hasMoreJobs && filteredJobs.length > 0 && (
                  <div className="flex justify-center py-4">
                    <Button
                      onClick={loadMoreJobs}
                      variant="outline"
                      disabled={loadingMore}
                    >
                      {loadingMore ? (
                        <>
                          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                          Loading...
                        </>
                      ) : (
                        <>
                          <ChevronDown className="mr-2 h-4 w-4" />
                          Load More Jobs
                        </>
                      )}
                    </Button>
                  </div>
                )}
              </div>
            </ScrollArea>
          </TabsContent>

          <TabsContent
            value="functions"
            className="mt-6 flex min-h-0 flex-1 flex-col space-y-6"
          >
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Label
                  htmlFor="namespace-select"
                  className="text-muted-foreground text-sm whitespace-nowrap"
                >
                  Namespace:
                </Label>
                <Select
                  value={selectedNamespace}
                  onValueChange={setSelectedNamespace}
                >
                  <SelectTrigger id="namespace-select" className="w-[180px]">
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
                disabled={syncing}
              >
                {syncing ? (
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
            </div>

            <div className="grid gap-4 md:grid-cols-3">
              <Card className="!gap-0">
                <CardContent className="px-4 py-4">
                  <div className="text-muted-foreground mb-1 text-xs">
                    Total Functions
                  </div>
                  <div className="text-2xl font-bold">
                    {jobFunctions.length}
                  </div>
                </CardContent>
              </Card>
              <Card className="!gap-0">
                <CardContent className="px-4 py-4">
                  <div className="text-muted-foreground mb-1 text-xs">
                    Enabled
                  </div>
                  <div className="text-2xl font-bold">
                    {jobFunctions.filter((f) => f.enabled).length}
                  </div>
                </CardContent>
              </Card>
              <Card className="!gap-0">
                <CardContent className="px-4 py-4">
                  <div className="text-muted-foreground mb-1 text-xs">
                    Scheduled
                  </div>
                  <div className="text-2xl font-bold">
                    {jobFunctions.filter((f) => f.schedule).length}
                  </div>
                </CardContent>
              </Card>
            </div>

            <ScrollArea className="min-h-0 flex-1">
              <div className="grid gap-4">
                {jobFunctions.length === 0 ? (
                  <Card>
                    <CardContent className="p-12 text-center">
                      <ListTodo className="text-muted-foreground mx-auto mb-4 h-12 w-12" />
                      <p className="mb-2 text-lg font-medium">
                        No job functions yet
                      </p>
                      <p className="text-muted-foreground text-sm">
                        Place job function files in your jobs directory and sync
                      </p>
                    </CardContent>
                  </Card>
                ) : (
                  jobFunctions.map((fn) => (
                    <div
                      key={fn.id}
                      className="hover:border-primary/50 bg-card flex items-center justify-between gap-2 rounded-md border px-3 py-1.5 transition-colors"
                    >
                      <div className="flex min-w-0 flex-1 items-center gap-2">
                        <span className="truncate text-sm font-medium">
                          {fn.name}
                        </span>
                        <Badge
                          variant="outline"
                          className="h-4 shrink-0 px-1 py-0 text-[10px]"
                        >
                          v{fn.version}
                        </Badge>
                        {fn.schedule && (
                          <Badge
                            variant="outline"
                            className="h-4 shrink-0 px-1 py-0 text-[10px]"
                          >
                            <Clock className="mr-0.5 h-2.5 w-2.5" />
                            {fn.schedule}
                          </Badge>
                        )}
                        <Switch
                          id={`enable-${fn.id}`}
                          checked={fn.enabled}
                          disabled={togglingJob === fn.id}
                          onCheckedChange={() => toggleJobEnabled(fn)}
                          className="scale-75"
                        />
                      </div>
                      <div className="flex shrink-0 items-center gap-0.5">
                        {fn.source === "filesystem" && fn.updated_at && (
                          <span
                            className="text-muted-foreground mr-1 text-[10px]"
                            title={`Last synced: ${new Date(fn.updated_at).toLocaleString()}`}
                          >
                            synced{" "}
                            {new Date(fn.updated_at).toLocaleDateString()}
                          </span>
                        )}
                        <span className="text-muted-foreground mr-1 text-[10px]">
                          {fn.timeout_seconds}s / {fn.max_retries}r
                        </span>
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <Button
                              onClick={() => viewHistory(fn)}
                              variant="ghost"
                              size="sm"
                              className="h-6 w-6 p-0"
                            >
                              <History className="h-3 w-3" />
                            </Button>
                          </TooltipTrigger>
                          <TooltipContent>View history</TooltipContent>
                        </Tooltip>
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <Button
                              onClick={() => openRunDialog(fn)}
                              size="sm"
                              variant="ghost"
                              className="h-6 w-6 p-0"
                              disabled={!fn.enabled}
                            >
                              <Play className="h-3 w-3" />
                            </Button>
                          </TooltipTrigger>
                          <TooltipContent>Run job</TooltipContent>
                        </Tooltip>
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <Button
                              onClick={() => openEditDialog(fn)}
                              size="sm"
                              variant="ghost"
                              className="h-6 w-6 p-0"
                            >
                              <Edit className="h-3 w-3" />
                            </Button>
                          </TooltipTrigger>
                          <TooltipContent>Edit job function</TooltipContent>
                        </Tooltip>
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <Button
                              onClick={() =>
                                setDeleteConfirm({
                                  namespace: fn.namespace,
                                  name: fn.name,
                                })
                              }
                              size="sm"
                              variant="ghost"
                              className="text-destructive hover:text-destructive hover:bg-destructive/10 h-6 w-6 p-0"
                            >
                              <Trash2 className="h-3 w-3" />
                            </Button>
                          </TooltipTrigger>
                          <TooltipContent>Delete job function</TooltipContent>
                        </Tooltip>
                      </div>
                    </div>
                  ))
                )}
              </div>
            </ScrollArea>
          </TabsContent>
        </Tabs>
      </div>

      <JobDetailsDialog
        open={isJobDetailsOpen}
        onOpenChange={setIsJobDetailsOpen}
        job={selectedJob}
        executionLogs={executionLogs}
        loadingLogs={loadingLogs}
        logLevelFilter={logLevelFilter}
        onLogLevelFilterChange={setLogLevelFilter}
        onCancelJob={cancelJob}
        onResubmitJob={resubmitJob}
      />

      <RunJobDialog
        open={isRunDialogOpen}
        onOpenChange={setIsRunDialogOpen}
        jobFunction={selectedFunction}
        namespace={selectedNamespace}
        payload={jobPayload}
        onPayloadChange={setJobPayload}
        submitting={submittingJob}
        onSubmit={handleSubmitJob}
      />

      <EditJobDialog
        open={isEditDialogOpen}
        onOpenChange={setIsEditDialogOpen}
        jobFunction={selectedFunction}
        fetching={fetchingFunction}
        formData={editFormData}
        onFormDataChange={setEditFormData}
        onUpdate={updateJobFunction}
      />

      <DeleteConfirmDialog
        deleteConfirm={deleteConfirm}
        onOpenChange={() => setDeleteConfirm(null)}
        onDelete={deleteJobFunction}
      />

      <ExecutionHistoryDialog
        open={isHistoryDialogOpen}
        onOpenChange={setIsHistoryDialogOpen}
        jobFunction={selectedFunction}
        historyJobs={historyJobs}
        loading={historyLoading}
        onViewJobDetails={viewJobDetails}
      />
    </div>
  );
}
