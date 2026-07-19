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
import { Switch } from "@/components/ui/switch";

interface CreateIntegrationDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

/**
 * CreateIntegrationDialog — form for adding a new tool integration.
 * Currently specialized to web_search + Tavily. When more integration
 * types ship, the form should switch on integration_type and render
 * provider-specific fields.
 *
 * The shape mirrors CreateProviderDialog. Secrets (api_key) are sent
 * plaintext to the server (over HTTPS) and encrypted at the storage
 * layer; they never appear in API responses afterward.
 */
export function CreateIntegrationDialog({
  open,
  onOpenChange,
}: CreateIntegrationDialogProps) {
  const client = useFluxbaseClient();
  const queryClient = useQueryClient();

  const [name, setName] = useState("");
  const [provider, setProvider] = useState<"tavily">("tavily");
  const [apiKey, setApiKey] = useState("");
  const [defaultDepth, setDefaultDepth] = useState<"basic" | "advanced">(
    "basic",
  );
  const [isDefault, setIsDefault] = useState(true);
  const [enabled, setEnabled] = useState(true);

  const createMutation = useMutation({
    mutationFn: () =>
      client.admin.ai.createIntegration({
        name,
        integration_type: "web_search",
        provider,
        config: {
          ...(apiKey ? { api_key: apiKey } : {}),
          ...(defaultDepth ? { default_depth: defaultDepth } : {}),
        },
        enabled,
        is_default: isDefault,
      }),
    onSuccess: async () => {
      toast.success("Integration created");
      await queryClient.invalidateQueries({
        queryKey: ["ai", "integrations"],
      });
      reset();
      onOpenChange(false);
    },
    onError: (error) =>
      toast.error(`Create failed: ${(error as Error).message}`),
  });

  const reset = () => {
    setName("");
    setProvider("tavily");
    setApiKey("");
    setDefaultDepth("basic");
    setIsDefault(true);
    setEnabled(true);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>New Tool Integration</DialogTitle>
          <DialogDescription>
            Configure an external service that chatbot specialists can
            call. Currently supports web search via Tavily.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="name">Name</Label>
            <Input
              id="name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Tavily (prod)"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="provider">Provider</Label>
            <Select
              value={provider}
              onValueChange={(v) => setProvider(v as "tavily")}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="tavily">Tavily</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <Label htmlFor="apiKey">API Key</Label>
            <Input
              id="apiKey"
              type="password"
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              placeholder="tvly-..."
            />
            <p className="text-xs text-muted-foreground">
              Encrypted at rest. Never shown again after save.
            </p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="depth">Default Search Depth</Label>
            <Select
              value={defaultDepth}
              onValueChange={(v) =>
                setDefaultDepth(v as "basic" | "advanced")
              }
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="basic">
                  Basic (faster, cheaper)
                </SelectItem>
                <SelectItem value="advanced">
                  Advanced (slower, more thorough)
                </SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div className="flex items-center justify-between">
            <Label htmlFor="default">Set as default for web_search</Label>
            <Switch
              id="default"
              checked={isDefault}
              onCheckedChange={setIsDefault}
            />
          </div>

          <div className="flex items-center justify-between">
            <Label htmlFor="enabled">Enabled</Label>
            <Switch
              id="enabled"
              checked={enabled}
              onCheckedChange={setEnabled}
            />
          </div>
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
          >
            Cancel
          </Button>
          <Button
            disabled={
              createMutation.isPending ||
              !name ||
              !apiKey ||
              !provider
            }
            onClick={() => createMutation.mutate()}
          >
            {createMutation.isPending ? "Creating..." : "Create"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
