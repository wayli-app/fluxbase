import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useFluxbaseClient } from "@nimbleflux/fluxbase-sdk-react";
import type { ToolIntegration } from "@nimbleflux/fluxbase-sdk";
import {
  Globe,
  Plus,
  Star,
  MoreHorizontal,
  Pencil,
  Trash2,
  Check,
  FlaskConical,
  Loader2,
} from "lucide-react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { CreateIntegrationDialog } from "./create-integration-dialog";
import { EditIntegrationDialog } from "./edit-integration-dialog";

/**
 * ToolIntegrationsTab lists, creates, edits, and tests non-LLM tool
 * integrations (currently: web search via Tavily). Mirrors AIProvidersTab
 * shape: table with row actions + create/edit dialogs + test-connection
 * button.
 */
export function ToolIntegrationsTab() {
  const client = useFluxbaseClient();
  const queryClient = useQueryClient();

  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [editingIntegration, setEditingIntegration] =
    useState<ToolIntegration | null>(null);
  const [deletingIntegration, setDeletingIntegration] =
    useState<ToolIntegration | null>(null);
  const [testingId, setTestingId] = useState<string | null>(null);

  const { data: integrations, isLoading } = useQuery({
    queryKey: ["ai-integrations", client.admin.ai],
    queryFn: () => client.admin.ai.listIntegrations(),
    select: (res) => res.data ?? [],
  });

  const testMutation = useMutation({
    mutationFn: (id: string) => client.admin.ai.testIntegration(id),
    onMutate: (id) => setTestingId(id),
    onSuccess: async (res) => {
      if (res.data?.status === "ok") {
        toast.success("Connection test passed");
      } else {
        toast.error(`Test failed: ${res.data?.error ?? "unknown error"}`);
      }
      await queryClient.invalidateQueries({
        queryKey: ["ai-integrations"],
      });
    },
    onError: (error) => toast.error(`Test failed: ${(error as Error).message}`),
    onSettled: () => setTestingId(null),
  });

  const setDefaultMutation = useMutation({
    mutationFn: (id: string) => client.admin.ai.setDefaultIntegration(id),
    onSuccess: async () => {
      toast.success("Default integration updated");
      await queryClient.invalidateQueries({
        queryKey: ["ai-integrations"],
      });
    },
    onError: (error) => toast.error(`Failed: ${(error as Error).message}`),
  });

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0">
        <div>
          <CardTitle className="flex items-center gap-2">
            <Globe className="h-5 w-5" />
            Tool Integrations
          </CardTitle>
          <CardDescription>
            External services that chatbot specialists can call. Currently
            supports web search via Tavily.
          </CardDescription>
        </div>
        <Button onClick={() => setShowCreateDialog(true)}>
          <Plus className="mr-2 h-4 w-4" />
          New Integration
        </Button>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="flex items-center justify-center py-8 text-muted-foreground">
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            Loading integrations...
          </div>
        ) : !integrations || integrations.length === 0 ? (
          <div className="rounded-lg border border-dashed py-12 text-center">
            <Globe className="mx-auto mb-3 h-8 w-8 text-muted-foreground" />
            <p className="text-sm text-muted-foreground">
              No tool integrations configured. Add one to enable the Web
              Agent specialist for your chatbots.
            </p>
          </div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Type</TableHead>
                <TableHead>Provider</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Default</TableHead>
                <TableHead className="w-12"></TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {integrations.map((integration) => (
                <TableRow key={integration.id}>
                  <TableCell className="font-medium">
                    {integration.name}
                    {integration.read_only && (
                      <Badge variant="outline" className="ml-2 text-xs">
                        Read-only
                      </Badge>
                    )}
                  </TableCell>
                  <TableCell>
                    <Badge variant="secondary">
                      {integration.integration_type}
                    </Badge>
                  </TableCell>
                  <TableCell className="capitalize">
                    {integration.provider}
                  </TableCell>
                  <TableCell>
                    {integration.last_test_status === "ok" && (
                      <Badge className="bg-green-100 text-green-800">
                        <Check className="mr-1 h-3 w-3" />
                        Tested OK
                      </Badge>
                    )}
                    {integration.last_test_status === "failed" && (
                      <Badge variant="destructive">Test failed</Badge>
                    )}
                    {!integration.last_test_status && (
                      <Badge variant="outline">Not tested</Badge>
                    )}
                  </TableCell>
                  <TableCell>
                    {integration.is_default && (
                      <Star className="h-4 w-4 fill-yellow-400 text-yellow-400" />
                    )}
                  </TableCell>
                  <TableCell>
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="ghost" size="icon">
                          <MoreHorizontal className="h-4 w-4" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        <DropdownMenuItem
                          onClick={() => setEditingIntegration(integration)}
                        >
                          <Pencil className="mr-2 h-4 w-4" />
                          Edit
                        </DropdownMenuItem>
                        <DropdownMenuItem
                          disabled={
                            testingId === integration.id ||
                            integration.read_only
                          }
                          onClick={() =>
                            testMutation.mutate(integration.id)
                          }
                        >
                          {testingId === integration.id ? (
                            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                          ) : (
                            <FlaskConical className="mr-2 h-4 w-4" />
                          )}
                          Test connection
                        </DropdownMenuItem>
                        {!integration.is_default && (
                          <DropdownMenuItem
                            disabled={integration.read_only}
                            onClick={() =>
                              setDefaultMutation.mutate(integration.id)
                            }
                          >
                            <Star className="mr-2 h-4 w-4" />
                            Set as default
                          </DropdownMenuItem>
                        )}
                        {!integration.read_only && (
                          <>
                            <DropdownMenuSeparator />
                            <DropdownMenuItem
                              className="text-destructive"
                              onClick={() =>
                                setDeletingIntegration(integration)
                              }
                            >
                              <Trash2 className="mr-2 h-4 w-4" />
                              Delete
                            </DropdownMenuItem>
                          </>
                        )}
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </CardContent>

      <CreateIntegrationDialog
        open={showCreateDialog}
        onOpenChange={setShowCreateDialog}
      />
      <EditIntegrationDialog
        integration={editingIntegration}
        onOpenChange={(open) => !open && setEditingIntegration(null)}
      />
      <ConfirmDialog
        open={!!deletingIntegration}
        onOpenChange={(open) => !open && setDeletingIntegration(null)}
        title="Delete integration"
        desc={`Delete "${deletingIntegration?.name}"? Chatbots using this integration will lose access to the Web Agent.`}
        confirmText="Delete"
        destructive
        handleConfirm={async () => {
          if (!deletingIntegration) return;
          const { error } = await client.admin.ai.deleteIntegration(
            deletingIntegration.id,
          );
          if (error) {
            toast.error(`Delete failed: ${error.message}`);
            return;
          }
          toast.success("Integration deleted");
          await queryClient.invalidateQueries({
            queryKey: ["ai-integrations"],
          });
          setDeletingIntegration(null);
        }}
      />
    </Card>
  );
}
