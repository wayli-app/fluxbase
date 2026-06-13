import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { Settings, Shield, RefreshCw } from "lucide-react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { PageHeader } from "@/components/layout/page-header";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
} from "@/components/ui/dialog";
import { instanceSettingsApi } from "@/lib/api";
import { requireInstanceAdmin } from "@/lib/route-guards";

export const Route = createFileRoute("/_authenticated/instance-settings/")({
  beforeLoad: () => {
    requireInstanceAdmin();
  },
  component: InstanceSettingsPage,
});

// The API returns nested settings like { security: { enable_global_rate_limit: true } }
// but the frontend uses dot-notation internally
interface InstanceSettingsResponse {
  settings: Record<string, unknown>;
  overridable_settings?: string[];
}

interface OverridableResponse {
  overridable_settings: string[];
}

function InstanceSettingsPage() {
  const queryClient = useQueryClient();
  const [isOverridableDialogOpen, setIsOverridableDialogOpen] = useState(false);
  const [rateLimitValue, setRateLimitValue] = useState(100);

  // Fetch instance settings
  const { data: settingsData, isLoading } = useQuery<InstanceSettingsResponse>({
    queryKey: ["instance-settings"],
    queryFn: () => instanceSettingsApi.get(),
  });

  // Fetch overridable settings list
  const { data: overridableData } = useQuery<OverridableResponse>({
    queryKey: ["instance-settings", "overridable"],
    queryFn: () => instanceSettingsApi.getOverridable(),
  });

  // Get rate limit setting from nested structure
  const securitySettings = settingsData?.settings?.security as
    | Record<string, unknown>
    | undefined;
  const isRateLimitEnabled =
    securitySettings?.enable_global_rate_limit === true;
  const requestsPerMinute = securitySettings?.rate_limit_requests as
    | number
    | undefined;

  // Check if setting comes from config (read-only)
  // For now, we'll assume instance settings in the database are editable
  const isReadOnly = false;

  // Update instance settings mutation
  const updateMutation = useMutation({
    mutationFn: (data: { settings: Record<string, unknown> }) =>
      instanceSettingsApi.update(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["instance-settings"] });
      toast.success("Settings updated successfully");
    },
    onError: (error: Error) => {
      toast.error(`Failed to update settings: ${error.message}`);
    },
  });

  // Update overridable settings mutation
  const updateOverridableMutation = useMutation({
    mutationFn: (data: { overridable_settings: string[] }) =>
      instanceSettingsApi.updateOverridable(data),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["instance-settings", "overridable"],
      });
      toast.success("Overridable settings updated");
      setIsOverridableDialogOpen(false);
    },
    onError: (error: Error) => {
      toast.error(`Failed to update overridable settings: ${error.message}`);
    },
  });

  // Handle rate limit toggle
  const handleRateLimitToggle = (enabled: boolean) => {
    updateMutation.mutate({
      settings: { "security.enable_global_rate_limit": enabled },
    });
  };

  // Handle rate limit value change
  const handleRateLimitValueChange = () => {
    updateMutation.mutate({
      settings: { "security.rate_limit_requests": rateLimitValue },
    });
  };

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center">
        <RefreshCw className="text-muted-foreground h-8 w-8 animate-spin" />
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <PageHeader
        icon={<Settings />}
        title="Instance Settings"
        description="Instance-level configuration for security and tenant permissions"
        actions={
          <Button
            variant="outline"
            size="sm"
            onClick={() => setIsOverridableDialogOpen(true)}
          >
            <Shield className="h-4 w-4 mr-2" />
            Manage Tenant Overrides
          </Button>
        }
      />

      {/* Main content */}
      <div className="flex-1 overflow-auto p-6">
        <div className="max-w-2xl space-y-6">
          {/* Global Rate Limit Card */}
          <Card>
            <CardHeader>
              <div className="flex items-center gap-2">
                <Shield className="h-5 w-5 text-muted-foreground" />
                <CardTitle className="text-lg">Global Rate Limiting</CardTitle>
              </div>
              <CardDescription>
                Configure rate limiting for all API endpoints across all
                tenants. This applies a global throttle to protect against
                abuse.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              {/* Enable/Disable Toggle */}
              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label htmlFor="rate-limit" className="text-base font-medium">
                    Enable Global Rate Limit
                  </Label>
                  <p className="text-muted-foreground text-sm">
                    {isRateLimitEnabled
                      ? "Rate limiting is active for all API requests"
                      : "Rate limiting is disabled"}
                  </p>
                </div>
                <div className="flex items-center gap-3">
                  {isReadOnly && (
                    <Badge variant="secondary">Set in config</Badge>
                  )}
                  <Switch
                    id="rate-limit"
                    checked={isRateLimitEnabled}
                    onCheckedChange={handleRateLimitToggle}
                    disabled={updateMutation.isPending || isReadOnly}
                  />
                </div>
              </div>

              {/* Requests per minute configuration */}
              {isRateLimitEnabled && (
                <div className="border-t pt-6">
                  <div className="space-y-4">
                    <div className="space-y-2">
                      <Label htmlFor="rate-limit-value">
                        Requests per Minute (per IP)
                      </Label>
                      <div className="flex items-end gap-4">
                        <Input
                          id="rate-limit-value"
                          type="number"
                          min={1}
                          max={100000}
                          value={requestsPerMinute ?? rateLimitValue}
                          onChange={(e) =>
                            setRateLimitValue(parseInt(e.target.value) || 100)
                          }
                          className="max-w-[200px]"
                          readOnly={isReadOnly}
                        />
                        {!isReadOnly && (
                          <Button
                            variant="outline"
                            onClick={handleRateLimitValueChange}
                            disabled={updateMutation.isPending}
                          >
                            {updateMutation.isPending ? "Saving..." : "Update"}
                          </Button>
                        )}
                      </div>
                      <p className="text-muted-foreground text-xs">
                        Maximum number of requests per minute per IP address.
                        Default is 100 requests/minute.
                      </p>
                    </div>
                  </div>
                </div>
              )}
            </CardContent>
          </Card>

          {/* Info Card */}
          <Card className="border-dashed">
            <CardContent className="pt-6">
              <div className="flex items-start gap-3 text-muted-foreground">
                <Shield className="h-5 w-5 mt-0.5" />
                <div className="space-y-2 text-sm">
                  <p>
                    <strong className="text-foreground">
                      Instance settings
                    </strong>{" "}
                    apply to all tenants and cannot be overridden at the tenant
                    level.
                  </p>
                  <p>
                    <strong className="text-foreground">
                      Tenant-level settings
                    </strong>{" "}
                    (AI, Storage, Authentication) are configured per-tenant. Use
                    the "Manage Tenant Overrides" button above to control which
                    settings tenants can customize.
                  </p>
                  <p>
                    Email settings for system emails (password reset, invites)
                    are configured in{" "}
                    <a
                      href="/email-settings"
                      className="text-primary underline hover:no-underline"
                    >
                      Email Settings
                    </a>
                    .
                  </p>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>

      {/* Overridable settings dialog */}
      <Dialog
        open={isOverridableDialogOpen}
        onOpenChange={setIsOverridableDialogOpen}
      >
        <DialogContent className="max-w-lg">
          <DialogHeader>Manage Tenant-Overridable Settings</DialogHeader>
          <DialogDescription>
            Select which tenant-level settings tenants can override. Instance
            settings (Security) are never overridable.
          </DialogDescription>
          <div className="py-4">
            <Label className="text-sm font-medium">
              Allowed Setting Categories
            </Label>
            <p className="text-muted-foreground text-sm mt-1 mb-4">
              These control which settings from the default configuration
              tenants can customize for their own use.
            </p>
            <div className="space-y-3">
              {[
                {
                  id: "ai",
                  label: "AI",
                  description: "AI providers and models",
                },
                {
                  id: "storage",
                  label: "Storage",
                  description: "File upload limits and providers",
                },
                {
                  id: "auth",
                  label: "Authentication",
                  description: "OAuth, SAML, and auth methods",
                },
              ].map((category) => (
                <div
                  key={category.id}
                  className="flex items-center justify-between rounded-lg border p-3"
                >
                  <div>
                    <div className="font-medium">{category.label}</div>
                    <div className="text-muted-foreground text-xs">
                      {category.description}
                    </div>
                  </div>
                  <Switch
                    checked={overridableData?.overridable_settings?.some(
                      (s: string) =>
                        s.startsWith(`${category.id}.`) || s === category.id,
                    )}
                    onCheckedChange={(checked) => {
                      const current =
                        overridableData?.overridable_settings || [];
                      let newSettings: string[];
                      if (checked) {
                        newSettings = [...current, category.id];
                      } else {
                        newSettings = current.filter(
                          (s: string) =>
                            !s.startsWith(`${category.id}.`) &&
                            s !== category.id,
                        );
                      }
                      updateOverridableMutation.mutate({
                        overridable_settings: newSettings,
                      });
                    }}
                    disabled={updateOverridableMutation.isPending}
                  />
                </div>
              ))}
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setIsOverridableDialogOpen(false)}
            >
              Done
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
