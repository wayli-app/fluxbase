import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useFluxbaseClient } from "@nimbleflux/fluxbase-sdk-react";
import { toast } from "sonner";
import type { AIProvider } from "@nimbleflux/fluxbase-sdk";
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

interface EditProviderDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  provider: AIProvider | null;
}

export function EditProviderDialog({
  open,
  onOpenChange,
  provider,
}: EditProviderDialogProps) {
  const client = useFluxbaseClient();
  const queryClient = useQueryClient();

  const [displayName, setDisplayName] = useState(
    () => provider?.display_name || "",
  );
  const [apiKey, setApiKey] = useState("");
  const [model, setModel] = useState(() => provider?.config?.model || "");
  const [embeddingModel, setEmbeddingModel] = useState(
    () => provider?.embedding_model || "",
  );
  const [enabled, setEnabled] = useState(() => provider?.enabled ?? true);

  const updateMutation = useMutation({
    mutationFn: async () => {
      if (!provider) return;
      const config: Record<string, unknown> = {};
      if (model) config.model = model;
      if (apiKey) config.api_key = apiKey;
      if (provider.provider_type === "ollama" && provider.config?.base_url) {
        config.base_url = provider.config.base_url;
      }
      Object.keys(provider.config || {}).forEach((key) => {
        if (key !== "api_key" && key !== "model" && key !== "base_url") {
          config[key] = provider.config![key];
        }
      });

      const { error } = await client.admin.ai.updateProvider(provider.id, {
        display_name: displayName || undefined,
        enabled,
        config: Object.keys(config).length > 0 ? config : undefined,
      });
      if (error) throw error;

      if (embeddingModel !== (provider.embedding_model || "")) {
        const { error: embError } = await client.admin.ai.updateProvider(
          provider.id,
          {
            embedding_model: embeddingModel || undefined,
          },
        );
        if (embError) throw embError;
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["ai-providers"] });
      onOpenChange(false);
      toast.success("Provider updated successfully");
    },
    onError: (error: Error) => {
      toast.error(error.message || "Failed to update provider");
    },
  });

  const handleOpenChange = (open: boolean) => {
    if (!open) {
      setApiKey("");
    }
    onOpenChange(open);
  };

  if (!provider) return null;

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Edit AI Provider</DialogTitle>
          <DialogDescription>
            Update settings for {provider.display_name || provider.name}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div>
            <Label htmlFor="edit-display-name">Display Name</Label>
            <Input
              id="edit-display-name"
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
            />
          </div>

          <div>
            <Label htmlFor="edit-api-key">
              API Key (leave blank to keep current)
            </Label>
            <Input
              id="edit-api-key"
              type="password"
              placeholder="Enter new key or leave blank"
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
            />
          </div>

          <div>
            <Label htmlFor="edit-model">Model</Label>
            <Input
              id="edit-model"
              value={model}
              onChange={(e) => setModel(e.target.value)}
            />
          </div>

          <div>
            <Label htmlFor="edit-embedding-model">Embedding Model</Label>
            <Input
              id="edit-embedding-model"
              value={embeddingModel}
              onChange={(e) => setEmbeddingModel(e.target.value)}
            />
          </div>

          <div className="flex items-center gap-2">
            <input
              id="edit-enabled"
              type="checkbox"
              checked={enabled}
              onChange={(e) => setEnabled(e.target.checked)}
              className="rounded"
            />
            <Label htmlFor="edit-enabled">Enabled</Label>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => handleOpenChange(false)}>
            Cancel
          </Button>
          <Button
            onClick={() => updateMutation.mutate()}
            disabled={updateMutation.isPending}
          >
            {updateMutation.isPending ? "Saving..." : "Save Changes"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
