import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { Lock, Plus, Search } from "lucide-react";
import { toast } from "sonner";
import { secretsApi, type Secret, type SecretVersion, type CreateSecretRequest, type UpdateSecretRequest, type SecretsStats } from "@/lib/api";
import { useTenantStore } from "@/stores/tenant-store";
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  SecretsStatsCards,
  SecretTableRow,
  CreateSecretDialog,
  EditSecretDialog,
  VersionHistoryDialog,
} from "@/components/secrets";

export const Route = createFileRoute("/_authenticated/secrets/")({
  component: SecretsPage,
});

function SecretsPage() {
  const queryClient = useQueryClient();
  const currentTenantId = useTenantStore((state) => state.currentTenant?.id);
  const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false);
  const [isEditDialogOpen, setIsEditDialogOpen] = useState(false);
  const [isHistoryDialogOpen, setIsHistoryDialogOpen] = useState(false);
  const [selectedSecret, setSelectedSecret] = useState<Secret | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const [scopeFilter, setScopeFilter] = useState<string>("");

  const { data: secrets, isLoading } = useQuery<Secret[]>({
    queryKey: ["secrets", scopeFilter, currentTenantId],
    queryFn: () => secretsApi.list(scopeFilter || undefined),
  });

  const { data: stats } = useQuery<SecretsStats>({
    queryKey: ["secrets-stats", currentTenantId],
    queryFn: secretsApi.getStats,
  });

  const { data: versions } = useQuery<SecretVersion[]>({
    queryKey: ["secret-versions", selectedSecret?.id, currentTenantId],
    queryFn: () => secretsApi.getVersions(selectedSecret!.id),
    enabled: !!selectedSecret && isHistoryDialogOpen,
  });

  const createMutation = useMutation({
    mutationFn: secretsApi.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["secrets"] });
      queryClient.invalidateQueries({ queryKey: ["secrets-stats"] });
      setIsCreateDialogOpen(false);
      toast.success("Secret created successfully");
    },
    onError: (error: Error) => {
      toast.error(`Failed to create secret: ${error.message}`);
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateSecretRequest }) =>
      secretsApi.update(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["secrets"] });
      setIsEditDialogOpen(false);
      setSelectedSecret(null);
      toast.success("Secret updated successfully");
    },
    onError: (error: Error) => {
      toast.error(`Failed to update secret: ${error.message}`);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: secretsApi.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["secrets"] });
      queryClient.invalidateQueries({ queryKey: ["secrets-stats"] });
      toast.success("Secret deleted successfully");
    },
    onError: (error: Error) => {
      toast.error(`Failed to delete secret: ${error.message}`);
    },
  });

  const rollbackMutation = useMutation({
    mutationFn: ({ id, version }: { id: string; version: number }) =>
      secretsApi.rollback(id, version),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["secrets"] });
      queryClient.invalidateQueries({ queryKey: ["secret-versions"] });
      toast.success("Secret rolled back successfully");
    },
    onError: (error: Error) => {
      toast.error(`Failed to rollback secret: ${error.message}`);
    },
  });

  const handleCreateSecret = (data: {
    name: string;
    value: string;
    scope: "global" | "namespace";
    namespace?: string;
    description?: string;
    expires_at?: string;
  }) => {
    const request: CreateSecretRequest = {
      name: data.name,
      value: data.value,
      scope: data.scope,
      namespace: data.namespace,
      description: data.description,
      expires_at: data.expires_at,
    };
    createMutation.mutate(request);
  };

  const handleUpdateSecret = (data: {
    value?: string;
    description?: string;
  }) => {
    if (!selectedSecret) return;

    const request: UpdateSecretRequest = {};
    if (data.value?.trim()) {
      request.value = data.value;
    }
    if (data.description !== selectedSecret.description) {
      request.description = data.description?.trim() || undefined;
    }

    updateMutation.mutate({ id: selectedSecret.id, data: request });
  };

  const openEditDialog = (secret: Secret) => {
    setSelectedSecret(secret);
    setIsEditDialogOpen(true);
  };

  const openHistoryDialog = (secret: Secret) => {
    setSelectedSecret(secret);
    setIsHistoryDialogOpen(true);
  };

  const filteredSecrets = secrets?.filter(
    (secret) =>
      secret.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      secret.description?.toLowerCase().includes(searchQuery.toLowerCase()) ||
      secret.namespace?.toLowerCase().includes(searchQuery.toLowerCase()),
  );

  return (
    <div className="flex h-full flex-col">
      <PageHeader
        icon={<Lock />}
        title="Secrets"
        description="Manage encrypted secrets that are injected into edge functions and background jobs"
      />

      <div className="flex-1 overflow-auto p-6">
        <SecretsStatsCards
          total={stats?.total || 0}
          expiringSoon={stats?.expiring_soon || 0}
          expired={stats?.expired || 0}
        />

        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <div>
                <CardTitle>Secrets</CardTitle>
                <CardDescription>
                  Secrets are available as FLUXBASE_SECRET_NAME environment
                  variables in edge functions and background jobs
                </CardDescription>
              </div>
              <Button onClick={() => setIsCreateDialogOpen(true)}>
                <Plus className="mr-2 h-4 w-4" />
                Create Secret
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            <div className="mb-4 flex gap-4">
              <div className="relative flex-1">
                <Search className="text-muted-foreground absolute top-2.5 left-2 h-4 w-4" />
                <Input
                  placeholder="Search by name, description, or namespace..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="pl-8"
                />
              </div>
              <Select value={scopeFilter} onValueChange={setScopeFilter}>
                <SelectTrigger className="w-[180px]">
                  <SelectValue placeholder="All scopes" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All scopes</SelectItem>
                  <SelectItem value="global">Global</SelectItem>
                  <SelectItem value="namespace">Namespace</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {isLoading ? (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Scope</TableHead>
                    <TableHead>Version</TableHead>
                    <TableHead>Expires</TableHead>
                    <TableHead>Updated</TableHead>
                    <TableHead className="text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {Array(3)
                    .fill(0)
                    .map((_, i) => (
                      <TableRow key={i}>
                        <TableCell>
                          <Skeleton className="h-4 w-28" />
                        </TableCell>
                        <TableCell>
                          <Skeleton className="h-5 w-16" />
                        </TableCell>
                        <TableCell>
                          <Skeleton className="h-4 w-8" />
                        </TableCell>
                        <TableCell>
                          <Skeleton className="h-4 w-20" />
                        </TableCell>
                        <TableCell>
                          <Skeleton className="h-4 w-24" />
                        </TableCell>
                        <TableCell className="text-right">
                          <div className="flex justify-end gap-1">
                            <Skeleton className="h-8 w-8" />
                            <Skeleton className="h-8 w-8" />
                          </div>
                        </TableCell>
                      </TableRow>
                    ))}
                </TableBody>
              </Table>
            ) : filteredSecrets && filteredSecrets.length > 0 ? (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Scope</TableHead>
                    <TableHead>Version</TableHead>
                    <TableHead>Expires</TableHead>
                    <TableHead>Updated</TableHead>
                    <TableHead className="text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filteredSecrets.map((secret) => (
                    <SecretTableRow
                      key={secret.id}
                      secret={secret}
                      onEdit={openEditDialog}
                      onHistory={openHistoryDialog}
                      onDelete={(id) => deleteMutation.mutate(id)}
                      isDeleting={deleteMutation.isPending}
                    />
                  ))}
                </TableBody>
              </Table>
            ) : (
              <div className="flex flex-col items-center justify-center py-12 text-center">
                <Lock className="text-muted-foreground mb-4 h-12 w-12" />
                <p className="text-muted-foreground">
                  {searchQuery
                    ? "No secrets match your search"
                    : "No secrets yet"}
                </p>
                {!searchQuery && (
                  <Button
                    onClick={() => setIsCreateDialogOpen(true)}
                    variant="outline"
                    className="mt-4"
                  >
                    Create Your First Secret
                  </Button>
                )}
              </div>
            )}
          </CardContent>
        </Card>

        <CreateSecretDialog
          open={isCreateDialogOpen}
          onOpenChange={setIsCreateDialogOpen}
          onSubmit={handleCreateSecret}
          isPending={createMutation.isPending}
        />

        <EditSecretDialog
          open={isEditDialogOpen}
          onOpenChange={setIsEditDialogOpen}
          secret={selectedSecret}
          onSubmit={handleUpdateSecret}
          isPending={updateMutation.isPending}
        />

        <VersionHistoryDialog
          open={isHistoryDialogOpen}
          onOpenChange={setIsHistoryDialogOpen}
          secret={selectedSecret}
          versions={versions}
          onRollback={(id, version) => rollbackMutation.mutate({ id, version })}
          isRollbackPending={rollbackMutation.isPending}
        />
      </div>
    </div>
  );
}
