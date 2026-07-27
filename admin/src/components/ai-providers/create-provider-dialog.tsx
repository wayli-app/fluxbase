import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useFluxbaseClient } from "@nimbleflux/fluxbase-sdk-react";
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

interface CreateProviderDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function CreateProviderDialog({
  open,
  onOpenChange,
}: CreateProviderDialogProps) {
  const client = useFluxbaseClient();
  const queryClient = useQueryClient();

  const [name, setName] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [providerType, setProviderType] = useState("openai");
  const [apiKey, setApiKey] = useState("");
  const [model, setModel] = useState("");
  const [baseUrl, setBaseUrl] = useState("");

  const resetForm = () => {
    setName("");
    setDisplayName("");
    setProviderType("openai");
    setApiKey("");
    setModel("");
    setBaseUrl("");
  };

  const createMutation = useMutation({
    mutationFn: async () => {
      const config: Record<string, unknown> = {};
      if (apiKey) config.api_key = apiKey;
      if (model) config.model = model;
      if (providerType === "ollama" && baseUrl) config.base_url = baseUrl;

      const { error } = await client.admin.ai.createProvider({
        name,
        display_name: displayName || undefined,
        provider_type: providerType,
        config: Object.keys(config).length > 0 ? config : undefined,
      });
      if (error) throw error;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["ai-providers"] });
      onOpenChange(false);
      resetForm();
      toast.success("AI provider created successfully");
    },
    onError: (error: Error) => {
      toast.error(error.message || "Failed to create provider");
    },
  });

  const handleOpenChange = (open: boolean) => {
    if (!open) resetForm();
    onOpenChange(open);
  };

  const handleSubmit = () => {
    if (!name.trim()) {
      toast.error("Provider name is required");
      return;
    }
    createMutation.mutate();
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Create AI Provider</DialogTitle>
          <DialogDescription>
            Add a new AI provider for chatbot and embedding functionality
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div>
            <Label htmlFor="provider-name">Name</Label>
            <Input
              id="provider-name"
              placeholder="my-openai-provider"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </div>

          <div>
            <Label htmlFor="provider-display-name">Display Name</Label>
            <Input
              id="provider-display-name"
              placeholder="My OpenAI Provider"
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
            />
          </div>

          <div>
            <Label htmlFor="provider-type">Provider Type</Label>
            <Select value={providerType} onValueChange={setProviderType}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="openai">OpenAI</SelectItem>
                <SelectItem value="anthropic">Anthropic (Claude)</SelectItem>
                <SelectItem value="azure">Azure OpenAI</SelectItem>
                <SelectItem value="ollama">Ollama</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div>
            <Label htmlFor="provider-api-key">
              API Key{providerType === "ollama" ? " (optional)" : ""}
            </Label>
            <Input
              id="provider-api-key"
              type="password"
              placeholder={
                providerType === "ollama" ? "Optional for Ollama" : "sk-..."
              }
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
            />
          </div>

          <div>
            <Label htmlFor="provider-model">Model</Label>
            <Input
              id="provider-model"
              placeholder={
                providerType === "openai"
                  ? "gpt-4-turbo"
                  : providerType === "azure"
                    ? "gpt-4"
                    : "llama2"
              }
              value={model}
              onChange={(e) => setModel(e.target.value)}
            />
          </div>

          {providerType === "ollama" && (
            <div>
              <Label htmlFor="provider-base-url">Base URL</Label>
              <Input
                id="provider-base-url"
                placeholder="http://localhost:11434"
                value={baseUrl}
                onChange={(e) => setBaseUrl(e.target.value)}
              />
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => handleOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={handleSubmit} disabled={createMutation.isPending}>
            {createMutation.isPending ? "Creating..." : "Create Provider"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
