import { useEffect, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useFluxbaseClient } from "@nimbleflux/fluxbase-sdk-react";
import type { ToolIntegration } from "@nimbleflux/fluxbase-sdk";
import { toast } from "sonner";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";

interface EditIntegrationDialogProps {
  integration: ToolIntegration | null;
  onOpenChange: (open: boolean) => void;
}

/**
 * EditIntegrationDialog — form for editing an existing tool integration.
 *
 * API key field shows "***masked***" placeholder when the integration
 * already has a key. Leaving the field unchanged (or sending the mask
 * back) preserves the existing encrypted value. Entering new text
 * overwrites.
 */
export function EditIntegrationDialog({
  integration,
  onOpenChange,
}: EditIntegrationDialogProps) {
  const client = useFluxbaseClient();
  const queryClient = useQueryClient();

  const [name, setName] = useState("");
  const [apiKey, setApiKey] = useState("");
  const [enabled, setEnabled] = useState(true);

  // Sync local state when integration prop changes
  useEffect(() => {
    if (integration) {
      setName(integration.name);
      setApiKey(integration.config?.api_key ? "***masked***" : "");
      setEnabled(integration.enabled);
    }
  }, [integration]);

  const updateMutation = useMutation({
    mutationFn: () =>
      client.admin.ai.updateIntegration(integration!.id, {
        name,
        // ponytail: server treats "***masked***" as "preserve existing"
        config: { api_key: apiKey },
        enabled,
      }),
    onSuccess: async () => {
      toast.success("Integration updated");
      await queryClient.invalidateQueries({
        queryKey: ["ai", "integrations"],
      });
      onOpenChange(false);
    },
    onError: (error) =>
      toast.error(`Update failed: ${(error as Error).message}`),
  });

  if (!integration) return null;

  return (
    <Dialog open={!!integration} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Edit Integration</DialogTitle>
          <DialogDescription>
            {integration.read_only
              ? "This integration is read-only (configured via env/YAML). Changes here will be rejected by the server."
              : `Provider: ${integration.provider} · Type: ${integration.integration_type}`}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="edit-name">Name</Label>
            <Input
              id="edit-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              disabled={integration.read_only}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="edit-apiKey">API Key</Label>
            <Input
              id="edit-apiKey"
              type="password"
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              placeholder="Enter new key to replace"
              disabled={integration.read_only}
            />
            <p className="text-xs text-muted-foreground">
              {integration.config?.api_key
                ? "Leave unchanged to preserve existing key."
                : "No key set."}
            </p>
          </div>

          <div className="flex items-center justify-between">
            <Label htmlFor="edit-enabled">Enabled</Label>
            <Switch
              id="edit-enabled"
              checked={enabled}
              onCheckedChange={setEnabled}
              disabled={integration.read_only}
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            disabled={
              updateMutation.isPending || integration.read_only
            }
            onClick={() => updateMutation.mutate()}
          >
            {updateMutation.isPending ? "Saving..." : "Save"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
