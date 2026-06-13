import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { Key, Plus } from "lucide-react";
import { toast } from "sonner";
import {
  clientKeysApi,
  type ClientKey,
  type CreateClientKeyRequest,
} from "@/lib/api";
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
  ClientKeyStatsCards,
  ClientKeyTableRow,
  CreateClientKeyDialog,
  ShowCreatedKeyDialog,
  SCOPE_GROUPS,
  type ClientKeyWithPlaintext,
} from "@/components/client-keys";

const ClientKeysPage = () => {
  const queryClient = useQueryClient();
  const currentTenantId = useTenantStore((state) => state.currentTenant?.id);
  const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false);
  const [isKeyDialogOpen, setIsKeyDialogOpen] = useState(false);
  const [createdKey, setCreatedKey] = useState<ClientKeyWithPlaintext | null>(
    null,
  );
  const [searchQuery, setSearchQuery] = useState("");
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [selectedScopes, setSelectedScopes] = useState<string[]>([]);
  const [rateLimit, setRateLimit] = useState(100);
  const [expiresAt, setExpiresAt] = useState("");

  const { data: clientKeys, isLoading } = useQuery<ClientKey[]>({
    queryKey: ["client-keys", currentTenantId],
    queryFn: clientKeysApi.list,
    enabled: !!currentTenantId,
  });

  const createMutation = useMutation({
    mutationFn: clientKeysApi.create,
    onSuccess: (data) => {
      queryClient.invalidateQueries({
        queryKey: ["client-keys", currentTenantId],
      });
      setCreatedKey(data as unknown as ClientKeyWithPlaintext);
      setIsCreateDialogOpen(false);
      setIsKeyDialogOpen(true);
      setName("");
      setDescription("");
      setSelectedScopes([]);
      setRateLimit(100);
      setExpiresAt("");
    },
    onError: () => toast.error("Failed to create client key"),
  });

  const revokeMutation = useMutation({
    mutationFn: clientKeysApi.revoke,
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["client-keys", currentTenantId],
      });
      toast.success("Client key revoked successfully");
    },
    onError: () => toast.error("Failed to revoke client key"),
  });

  const deleteMutation = useMutation({
    mutationFn: clientKeysApi.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["client-keys", currentTenantId],
      });
      toast.success("Client key deleted successfully");
    },
    onError: () => toast.error("Failed to delete client key"),
  });

  const handleCreateKey = () => {
    if (!name.trim()) {
      toast.error("Please enter a key name");
      return;
    }
    if (selectedScopes.length === 0) {
      toast.error("Please select at least one scope");
      return;
    }
    const request: CreateClientKeyRequest = {
      name: name.trim(),
      description: description.trim() || undefined,
      scopes: selectedScopes,
      rate_limit_per_minute: rateLimit,
      expires_at: expiresAt || undefined,
    };
    createMutation.mutate(request);
  };

  const toggleScope = (scopeId: string) => {
    setSelectedScopes((prev) =>
      prev.includes(scopeId)
        ? prev.filter((s) => s !== scopeId)
        : [...prev, scopeId],
    );
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    toast.success("Copied to clipboard");
  };

  const isExpired = (expiresAt?: string) =>
    expiresAt && new Date(expiresAt) < new Date();
  const isRevoked = (revokedAt?: string) => !!revokedAt;

  const filteredKeys = clientKeys?.filter(
    (key) =>
      key.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      key.description?.toLowerCase().includes(searchQuery.toLowerCase()) ||
      key.key_prefix.toLowerCase().includes(searchQuery.toLowerCase()),
  );

  const activeCount =
    clientKeys?.filter(
      (k) => !isRevoked(k.revoked_at) && !isExpired(k.expires_at),
    ).length || 0;
  const revokedCount =
    clientKeys?.filter((k) => isRevoked(k.revoked_at)).length || 0;

  return (
    <div className="flex h-full flex-col">
      <PageHeader
        icon={<Key />}
        title="Client Keys"
        description="Generate and manage client keys for programmatic access"
      />

      <div className="flex-1 overflow-auto p-6">
        <div className="flex flex-col gap-6">
          <ClientKeyStatsCards
            total={clientKeys?.length || 0}
            active={activeCount}
            revoked={revokedCount}
          />

          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle>Client Keys</CardTitle>
                  <CardDescription>
                    Manage your client keys for service-to-service
                    authentication
                  </CardDescription>
                </div>
                <Button onClick={() => setIsCreateDialogOpen(true)}>
                  <Plus className="mr-2 h-4 w-4" />
                  Create Client Key
                </Button>
              </div>
            </CardHeader>
            <CardContent>
              <div className="mb-4 relative">
                <Input
                  placeholder="Search by name, description, or key prefix..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="pl-8"
                />
              </div>

              {isLoading ? (
                <LoadingSkeleton />
              ) : filteredKeys && filteredKeys.length > 0 ? (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Name</TableHead>
                      <TableHead>Key Prefix</TableHead>
                      <TableHead>Scopes</TableHead>
                      <TableHead>Rate Limit</TableHead>
                      <TableHead>Last Used</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead className="text-right">Actions</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {filteredKeys.map((key) => (
                      <ClientKeyTableRow
                        key={key.id}
                        clientKey={key}
                        onRevoke={(id) => revokeMutation.mutate(id)}
                        onDelete={(id) => deleteMutation.mutate(id)}
                        isRevoking={revokeMutation.isPending}
                        isDeleting={deleteMutation.isPending}
                      />
                    ))}
                  </TableBody>
                </Table>
              ) : (
                <EmptyState
                  searchQuery={searchQuery}
                  onCreate={() => setIsCreateDialogOpen(true)}
                />
              )}
            </CardContent>
          </Card>

          <CreateClientKeyDialog
            open={isCreateDialogOpen}
            onOpenChange={setIsCreateDialogOpen}
            scopeGroups={SCOPE_GROUPS}
            selectedScopes={selectedScopes}
            onToggleScope={toggleScope}
            onSubmit={handleCreateKey}
            isPending={createMutation.isPending}
            name={name}
            onNameChange={setName}
            description={description}
            onDescriptionChange={setDescription}
            rateLimit={rateLimit}
            onRateLimitChange={setRateLimit}
            expiresAt={expiresAt}
            onExpiresAtChange={setExpiresAt}
          />

          <ShowCreatedKeyDialog
            open={isKeyDialogOpen}
            onOpenChange={setIsKeyDialogOpen}
            createdKey={createdKey}
            onCopy={copyToClipboard}
          />
        </div>
      </div>
    </div>
  );
};

function LoadingSkeleton() {
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Name</TableHead>
          <TableHead>Key Prefix</TableHead>
          <TableHead>Scopes</TableHead>
          <TableHead>Rate Limit</TableHead>
          <TableHead>Last Used</TableHead>
          <TableHead>Status</TableHead>
          <TableHead className="text-right">Actions</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {Array(3)
          .fill(0)
          .map((_, i) => (
            <TableRow key={i}>
              <TableCell>
                <div className="space-y-1">
                  <Skeleton className="h-4 w-28" />
                  <Skeleton className="h-3 w-20" />
                </div>
              </TableCell>
              <TableCell>
                <Skeleton className="h-4 w-24" />
              </TableCell>
              <TableCell>
                <Skeleton className="h-5 w-16" />
              </TableCell>
              <TableCell>
                <Skeleton className="h-4 w-20" />
              </TableCell>
              <TableCell>
                <Skeleton className="h-4 w-24" />
              </TableCell>
              <TableCell>
                <Skeleton className="h-5 w-16" />
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
  );
}

function EmptyState({
  searchQuery,
  onCreate,
}: {
  searchQuery: string;
  onCreate: () => void;
}) {
  return (
    <div className="flex flex-col items-center justify-center py-12 text-center">
      <Key className="text-muted-foreground mb-4 h-12 w-12" />
      <p className="text-muted-foreground">
        {searchQuery
          ? "No client keys match your search"
          : "No client keys yet"}
      </p>
      {!searchQuery && (
        <Button onClick={onCreate} variant="outline" className="mt-4">
          Create Your First Client Key
        </Button>
      )}
    </div>
  );
}

export const Route = createFileRoute("/_authenticated/client-keys/")({
  component: ClientKeysPage,
});
