import { useState, useEffect, useCallback, useRef } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { BookOpen, Plus, RefreshCw, AlertTriangle } from "lucide-react";
import { toast } from "sonner";
import {
  knowledgeBasesApi,
  userManagementApi,
  aiProvidersApi,
  type KnowledgeBaseSummary,
  type CreateKnowledgeBaseRequest,
  type EnrichedUser,
  type AIProvider,
} from "@/lib/api";
import { useTenantStore } from "@/stores/tenant-store";
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
import { Card, CardContent } from "@/components/ui/card";
import { ScrollArea } from "@/components/ui/scroll-area";
import { PageHeader } from "@/components/layout/page-header";
import {
  CreateKnowledgeBaseDialog,
  KnowledgeBaseCard,
  type FeatureDisabledError,
} from "@/components/knowledge-bases";

export const Route = createFileRoute("/_authenticated/knowledge-bases/")({
  component: KnowledgeBasesPage,
});

function KnowledgeBasesPage() {
  const navigate = useNavigate();
  const currentTenantId = useTenantStore((state) => state.currentTenant?.id);
  const queryClient = useQueryClient();
  const [featureDisabled, setFeatureDisabled] = useState(false);
  const featureDisabledRef = useRef(false);
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [users, setUsers] = useState<EnrichedUser[]>([]);
  const [usersLoading, setUsersLoading] = useState(false);
  const [providers, setProviders] = useState<AIProvider[]>([]);
  const [providersLoading, setProvidersLoading] = useState(false);
  const usersLoadedRef = useRef(false);

  const [newKB, setNewKB] = useState<CreateKnowledgeBaseRequest>({
    name: "",
    description: "",
    visibility: "private",
    embedding_model: "",
    chunk_size: 512,
    chunk_overlap: 50,
    chunk_strategy: "recursive",
    initial_permissions: [],
  });

  // Fetch knowledge bases
  type KBQueryResult = {
    bases: KnowledgeBaseSummary[];
    disabled: boolean;
  };

  const { data: knowledgeBases = [], isLoading } = useQuery<
    KBQueryResult,
    Error,
    KnowledgeBaseSummary[]
  >({
    queryKey: ["knowledge-bases", currentTenantId],
    queryFn: async () => {
      try {
        const data = await knowledgeBasesApi.list();
        return { bases: data || [], disabled: false };
      } catch (error) {
        const axiosError = error as {
          response?: { status?: number; data?: FeatureDisabledError };
        };
        if (
          axiosError.response?.status === 503 &&
          axiosError.response?.data?.code === "FEATURE_DISABLED"
        ) {
          return { bases: [] as KnowledgeBaseSummary[], disabled: true };
        }
        throw error;
      }
    },
    select: (result) => {
      if (featureDisabledRef.current !== result.disabled) {
        featureDisabledRef.current = result.disabled;
        setFeatureDisabled(result.disabled);
      }
      return result.bases;
    },
  });

  // Create knowledge base mutation
  const createMutation = useMutation({
    mutationFn: async (req: CreateKnowledgeBaseRequest) => {
      await knowledgeBasesApi.create(req);
    },
    onSuccess: () => {
      toast.success("Knowledge base created");
      setCreateDialogOpen(false);
      setNewKB({
        name: "",
        description: "",
        visibility: "private",
        embedding_model: "",
        chunk_size: 512,
        chunk_overlap: 50,
        chunk_strategy: "recursive",
        initial_permissions: [],
      });
      queryClient.invalidateQueries({ queryKey: ["knowledge-bases"] });
    },
    onError: (error) => {
      const message =
        (error as { response?: { data?: { error?: string } } })?.response?.data
          ?.error || "Failed to create knowledge base";
      toast.error(message);
    },
  });

  // Delete knowledge base mutation
  const deleteMutation = useMutation({
    mutationFn: async (id: string) => {
      await knowledgeBasesApi.delete(id);
    },
    onSuccess: () => {
      toast.success("Knowledge base deleted");
      queryClient.invalidateQueries({ queryKey: ["knowledge-bases"] });
    },
    onError: () => {
      toast.error("Failed to delete knowledge base");
    },
    onSettled: () => {
      setDeleteConfirm(null);
    },
  });

  // Toggle enabled mutation
  const toggleMutation = useMutation({
    mutationFn: async (kb: KnowledgeBaseSummary) => {
      await knowledgeBasesApi.update(kb.id, { enabled: !kb.enabled });
      return !kb.enabled;
    },
    onSuccess: (newEnabledState) => {
      toast.success(
        `Knowledge base ${newEnabledState ? "disabled" : "enabled"}`,
      );
      queryClient.invalidateQueries({ queryKey: ["knowledge-bases"] });
    },
    onError: () => {
      toast.error("Failed to update knowledge base");
    },
  });

  const handleCreate = () => {
    if (!newKB.name.trim()) {
      toast.error("Name is required");
      return;
    }
    createMutation.mutate(newKB);
  };

  const fetchUsers = useCallback(async () => {
    if (usersLoadedRef.current) return;
    setUsersLoading(true);
    try {
      const { users: data } = await userManagementApi.listUsers("app");
      setUsers(data || []);
      usersLoadedRef.current = true;
    } catch {
      toast.error("Failed to fetch users");
    } finally {
      setUsersLoading(false);
    }
  }, []);

  const fetchProviders = useCallback(async () => {
    setProvidersLoading(true);
    try {
      const data = await aiProvidersApi.list();
      setProviders((data || []).filter((p) => p.enabled));
    } catch {
      toast.error("Failed to fetch AI providers");
    } finally {
      setProvidersLoading(false);
    }
  }, []);

  useEffect(() => {
    if (createDialogOpen) {
      fetchUsers();
      fetchProviders();
    }
  }, [createDialogOpen, fetchUsers, fetchProviders]);

  if (isLoading) {
    return (
      <div className="flex h-96 items-center justify-center">
        <RefreshCw className="text-muted-foreground h-8 w-8 animate-spin" />
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col">
      <PageHeader
        icon={<BookOpen />}
        title="Knowledge Bases"
        description="Manage knowledge bases for RAG-powered AI chatbots"
      />

      <div className="flex-1 overflow-auto p-6">
        <div className="flex flex-col gap-6">
          <div className="flex items-center justify-between">
            <div className="flex gap-4 text-sm">
              <div className="flex items-center gap-1.5">
                <span className="text-muted-foreground">Total:</span>
                <Badge variant="secondary" className="h-5 px-2">
                  {knowledgeBases.length}
                </Badge>
              </div>
              <div className="flex items-center gap-1.5">
                <span className="text-muted-foreground">Active:</span>
                <Badge
                  variant="secondary"
                  className="h-5 bg-green-500/10 px-2 text-green-600 dark:text-green-400"
                >
                  {knowledgeBases.filter((kb) => kb.enabled).length}
                </Badge>
              </div>
              <div className="flex items-center gap-1.5">
                <span className="text-muted-foreground">Documents:</span>
                <Badge variant="secondary" className="h-5 px-2">
                  {knowledgeBases.reduce(
                    (sum, kb) => sum + kb.document_count,
                    0,
                  )}
                </Badge>
              </div>
            </div>
            <div className="flex items-center gap-2">
              <Button
                onClick={() =>
                  queryClient.invalidateQueries({
                    queryKey: ["knowledge-bases"],
                  })
                }
                variant="outline"
                size="sm"
              >
                <RefreshCw className="mr-2 h-4 w-4" />
                Refresh
              </Button>
              <Button size="sm" onClick={() => setCreateDialogOpen(true)}>
                <Plus className="mr-2 h-4 w-4" />
                Create Knowledge Base
              </Button>
            </div>
          </div>

          <ScrollArea className="h-[calc(100vh-16rem)]">
            {featureDisabled ? (
              <Card className="border-amber-200 bg-amber-50 dark:border-amber-900 dark:bg-amber-950">
                <CardContent className="p-12 text-center">
                  <AlertTriangle className="text-amber-600 mx-auto mb-4 h-12 w-12 dark:text-amber-400" />
                  <p className="mb-2 text-lg font-medium text-amber-800 dark:text-amber-200">
                    AI Features Not Enabled
                  </p>
                  <p className="text-amber-700 dark:text-amber-300 mb-4 text-sm">
                    Knowledge bases require AI features to be enabled. Enable AI
                    in your configuration to use this feature.
                  </p>
                  <p className="text-amber-600 dark:text-amber-400 text-xs">
                    Set{" "}
                    <code className="bg-amber-100 dark:bg-amber-900 px-1 rounded">
                      FLUXBASE_AI_ENABLED=true
                    </code>{" "}
                    in your environment or configure it in instance settings.
                  </p>
                </CardContent>
              </Card>
            ) : knowledgeBases.length === 0 ? (
              <Card>
                <CardContent className="p-12 text-center">
                  <BookOpen className="text-muted-foreground mx-auto mb-4 h-12 w-12" />
                  <p className="mb-2 text-lg font-medium">
                    No knowledge bases yet
                  </p>
                  <p className="text-muted-foreground mb-4 text-sm">
                    Create a knowledge base to store documents for RAG-powered
                    AI chatbots
                  </p>
                  <Button onClick={() => setCreateDialogOpen(true)}>
                    <Plus className="mr-2 h-4 w-4" />
                    Create Knowledge Base
                  </Button>
                </CardContent>
              </Card>
            ) : (
              <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
                {knowledgeBases.map((kb) => (
                  <KnowledgeBaseCard
                    key={kb.id}
                    kb={kb}
                    onToggleEnabled={(kb) => toggleMutation.mutate(kb)}
                    onDelete={(id) => setDeleteConfirm(id)}
                    onNavigate={(path) => navigate({ to: path })}
                  />
                ))}
              </div>
            )}
          </ScrollArea>

          <AlertDialog
            open={deleteConfirm !== null}
            onOpenChange={(open) => !open && setDeleteConfirm(null)}
          >
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>Delete Knowledge Base</AlertDialogTitle>
                <AlertDialogDescription>
                  Are you sure you want to delete this knowledge base? This will
                  permanently delete all documents and chunks. This action
                  cannot be undone.
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>Cancel</AlertDialogCancel>
                <AlertDialogAction
                  onClick={() =>
                    deleteConfirm && deleteMutation.mutate(deleteConfirm)
                  }
                  className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                >
                  Delete
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        </div>
      </div>

      <CreateKnowledgeBaseDialog
        open={createDialogOpen}
        onOpenChange={setCreateDialogOpen}
        newKB={newKB}
        onNewKBChange={setNewKB}
        onCreate={handleCreate}
        users={users}
        usersLoading={usersLoading}
        providers={providers}
        providersLoading={providersLoading}
      />
    </div>
  );
}
