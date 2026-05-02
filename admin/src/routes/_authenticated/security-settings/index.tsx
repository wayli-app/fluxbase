import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import {
  Shield,
  AlertCircle,
  Loader2,
  Bot,
  CheckCircle2,
  Info,
} from "lucide-react";
import { toast } from "sonner";
import {
  captchaSettingsApi,
  type CaptchaSettingsResponse,
  type UpdateCaptchaSettingsRequest,
} from "@/lib/api";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  OverridableSelect,
  SelectItem,
} from "@/components/admin/overridable-select";
import { OverridableSwitch } from "@/components/admin/overridable-switch";

export const Route = createFileRoute("/_authenticated/security-settings/")({
  component: SecuritySettingsPage,
});

// Captcha form state
interface CaptchaFormState {
  provider: string;
  site_key: string;
  secret_key: string;
  score_threshold: number;
  endpoints: string[];
  cap_server_url: string;
  cap_api_key: string;
}

function SecuritySettingsPage() {
  const queryClient = useQueryClient();

  // Fetch CAPTCHA settings (admin endpoint with management capabilities)
  const {
    data: captchaSettings,
    isLoading,
    dataUpdatedAt,
  } = useQuery<CaptchaSettingsResponse>({
    queryKey: ["captcha-settings"],
    queryFn: () => captchaSettingsApi.get(),
  });

  // Captcha form state - continuous editing pattern
  const [captchaForm, setCaptchaForm] = useState<CaptchaFormState>({
    provider: "hcaptcha",
    site_key: "",
    secret_key: "",
    score_threshold: 0.5,
    endpoints: ["signup", "login", "password_reset", "magic_link"],
    cap_server_url: "",
    cap_api_key: "",
  });
  const [hasUnsavedChanges, setHasUnsavedChanges] = useState(false);
  const [initializedFromDataUpdatedAt, setInitializedFromDataUpdatedAt] =
    useState<number | null>(null);

  // Initialize form state when settings are first loaded or refetched
  if (captchaSettings && dataUpdatedAt !== initializedFromDataUpdatedAt) {
    setInitializedFromDataUpdatedAt(dataUpdatedAt);
    setCaptchaForm({
      provider: captchaSettings.provider || "hcaptcha",
      site_key: captchaSettings.site_key || "",
      secret_key: "", // Never populate from server
      score_threshold: captchaSettings.score_threshold || 0.5,
      endpoints: captchaSettings.endpoints || [],
      cap_server_url: captchaSettings.cap_server_url || "",
      cap_api_key: "", // Never populate from server
    });
    setHasUnsavedChanges(false);
  }

  // Update captcha settings mutation
  const updateCaptchaMutation = useMutation({
    mutationFn: (request: UpdateCaptchaSettingsRequest) =>
      captchaSettingsApi.update(request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["captcha-settings"] });
      setHasUnsavedChanges(false);
      toast.success("Captcha settings updated successfully");
    },
    onError: (error: unknown) => {
      if (error && typeof error === "object" && "response" in error) {
        const err = error as {
          response?: {
            status?: number;
            data?: { code?: string; error?: string };
          };
        };
        if (
          err.response?.status === 409 &&
          err.response?.data?.code === "CONFIG_OVERRIDE"
        ) {
          toast.error(
            "This setting is controlled by configuration file or environment variable",
          );
          return;
        }
        if (err.response?.data?.error) {
          toast.error(err.response.data.error);
          return;
        }
      }
      toast.error("Failed to update captcha settings");
    },
  });

  // Helper to check if a field is overridden
  const isOverridden = (field: string) => {
    return captchaSettings?._overrides?.[field]?.is_overridden ?? false;
  };

  // Helper to get environment variable name
  const getEnvVar = (field: string) => {
    return captchaSettings?._overrides?.[field]?.env_var || "";
  };

  // Helper to convert API override to component override format
  const getOverride = (field: string) => {
    const override = captchaSettings?._overrides?.[field];
    if (!override?.is_overridden) return undefined;
    return {
      is_overridden: override.is_overridden,
      env_var: override.env_var || "",
    };
  };

  // Update form field and mark as changed
  const updateFormField = <K extends keyof CaptchaFormState>(
    field: K,
    value: CaptchaFormState[K],
  ) => {
    setCaptchaForm((prev) => ({ ...prev, [field]: value }));
    setHasUnsavedChanges(true);
  };

  // Toggle endpoint selection
  const toggleEndpoint = (endpoint: string) => {
    const newEndpoints = captchaForm.endpoints.includes(endpoint)
      ? captchaForm.endpoints.filter((e) => e !== endpoint)
      : [...captchaForm.endpoints, endpoint];
    updateFormField("endpoints", newEndpoints);
  };

  // Save captcha settings
  const handleSaveCaptcha = () => {
    const request: UpdateCaptchaSettingsRequest = {
      provider: captchaForm.provider,
      site_key: captchaForm.site_key,
      score_threshold: captchaForm.score_threshold,
      endpoints: captchaForm.endpoints,
      cap_server_url: captchaForm.cap_server_url,
    };

    // Only include secrets if they were changed (non-empty)
    if (captchaForm.secret_key) {
      request.secret_key = captchaForm.secret_key;
    }
    if (captchaForm.cap_api_key) {
      request.cap_api_key = captchaForm.cap_api_key;
    }

    updateCaptchaMutation.mutate(request);
  };

  // Toggle enabled state
  const handleToggleEnabled = (enabled: boolean) => {
    updateCaptchaMutation.mutate({ enabled });
  };

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center">
        <Loader2 className="text-muted-foreground h-8 w-8 animate-spin" />
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col">
      <div className="bg-background flex items-center justify-between border-b px-6 py-4">
        <div className="flex items-center gap-3">
          <div className="bg-primary/10 flex h-10 w-10 items-center justify-center rounded-lg">
            <Shield className="text-primary h-5 w-5" />
          </div>
          <div>
            <h1 className="text-xl font-semibold">Security Settings</h1>
            <p className="text-muted-foreground text-sm">
              Configure CAPTCHA protection for authentication endpoints
            </p>
          </div>
        </div>
      </div>

      <div className="flex-1 overflow-auto p-6">
        <div className="space-y-4">
          {/* CAPTCHA Settings */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Bot className="h-5 w-5" />
                CAPTCHA Protection
              </CardTitle>
              <CardDescription>
                Protect authentication endpoints from automated attacks
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              {/* Enabled Toggle */}
              <OverridableSwitch
                id="captcha-enabled"
                label="Enable CAPTCHA"
                description="Require CAPTCHA verification on protected endpoints"
                checked={captchaSettings?.enabled ?? false}
                onCheckedChange={handleToggleEnabled}
                disabled={updateCaptchaMutation.isPending}
                override={getOverride("enabled")}
              />

              {captchaSettings?.enabled && (
                <>
                  {/* Provider Selection */}
                  <OverridableSelect
                    id="captcha-provider"
                    label="CAPTCHA Provider"
                    value={captchaForm.provider}
                    onValueChange={(value) =>
                      updateFormField("provider", value)
                    }
                    override={getOverride("provider")}
                  >
                    <SelectItem value="hcaptcha">hCaptcha</SelectItem>
                    <SelectItem value="recaptcha_v3">reCAPTCHA v3</SelectItem>
                    <SelectItem value="turnstile">
                      Cloudflare Turnstile
                    </SelectItem>
                    <SelectItem value="cap">Cap (Self-hosted)</SelectItem>
                  </OverridableSelect>

                  {/* Site Key */}
                  <div className="space-y-2">
                    <Label htmlFor="site_key">Site Key</Label>
                    <div className="relative">
                      <Input
                        id="site_key"
                        value={captchaForm.site_key}
                        onChange={(e) =>
                          updateFormField("site_key", e.target.value)
                        }
                          readOnly={isOverridden("site_key")}
                        placeholder="Enter your CAPTCHA site key"
                      />
                      {isOverridden("site_key") && (
                        <Badge
                          variant="outline"
                          className="absolute top-1/2 right-2 -translate-y-1/2"
                        >
                          ENV: {getEnvVar("site_key")}
                        </Badge>
                      )}
                    </div>
                  </div>

                  {/* Secret Key */}
                  <div className="space-y-2">
                    <Label htmlFor="secret_key">Secret Key</Label>
                    <div className="space-y-2">
                      <div className="relative">
                        <Input
                          id="secret_key"
                          type="password"
                          value={captchaForm.secret_key}
                          onChange={(e) =>
                            updateFormField("secret_key", e.target.value)
                          }
                          readOnly={isOverridden("secret_key")}
                          placeholder="Leave empty to keep current secret"
                        />
                        {isOverridden("secret_key") && (
                          <Badge
                            variant="outline"
                            className="absolute top-1/2 right-2 -translate-y-1/2"
                          >
                            ENV: {getEnvVar("secret_key")}
                          </Badge>
                        )}
                      </div>
                      {captchaSettings?.secret_key_set && (
                        <Badge
                          variant="secondary"
                          className="flex w-fit items-center gap-1"
                        >
                          <CheckCircle2 className="h-3 w-3" />
                          Secret configured
                        </Badge>
                      )}
                    </div>
                  </div>

                  {/* Score Threshold (reCAPTCHA v3 only) */}
                  {captchaForm.provider === "recaptcha_v3" && (
                    <div className="space-y-2">
                      <Label htmlFor="score_threshold">Score Threshold</Label>
                      <div className="relative">
                        <Input
                          id="score_threshold"
                          type="number"
                          min="0"
                          max="1"
                          step="0.1"
                          value={captchaForm.score_threshold}
                          onChange={(e) =>
                            updateFormField(
                              "score_threshold",
                              parseFloat(e.target.value),
                            )
                          }
                          readOnly={isOverridden("score_threshold")}
                        />
                        {isOverridden("score_threshold") && (
                          <Badge
                            variant="outline"
                            className="absolute top-1/2 right-2 -translate-y-1/2"
                          >
                            ENV: {getEnvVar("score_threshold")}
                          </Badge>
                        )}
                      </div>
                      <p className="text-muted-foreground text-sm">
                        Minimum score (0.0-1.0) required to pass verification
                      </p>
                    </div>
                  )}

                  {/* Cap Provider Settings */}
                  {captchaForm.provider === "cap" && (
                    <>
                      <div className="space-y-2">
                        <Label htmlFor="cap_server_url">Cap Server URL</Label>
                        <div className="relative">
                          <Input
                            id="cap_server_url"
                            value={captchaForm.cap_server_url}
                            onChange={(e) =>
                              updateFormField("cap_server_url", e.target.value)
                            }
                            readOnly={isOverridden("cap_server_url")}
                            placeholder="https://cap.example.com"
                          />
                          {isOverridden("cap_server_url") && (
                            <Badge
                              variant="outline"
                              className="absolute top-1/2 right-2 -translate-y-1/2"
                            >
                              ENV: {getEnvVar("cap_server_url")}
                            </Badge>
                          )}
                        </div>
                      </div>

                      <div className="space-y-2">
                        <Label htmlFor="cap_api_key">Cap API Key</Label>
                        <div className="space-y-2">
                          <div className="relative">
                            <Input
                              id="cap_api_key"
                              type="password"
                              value={captchaForm.cap_api_key}
                              onChange={(e) =>
                                updateFormField("cap_api_key", e.target.value)
                              }
                              readOnly={isOverridden("cap_api_key")}
                              placeholder="Leave empty to keep current API key"
                            />
                            {isOverridden("cap_api_key") && (
                              <Badge
                                variant="outline"
                                className="absolute top-1/2 right-2 -translate-y-1/2"
                              >
                                ENV: {getEnvVar("cap_api_key")}
                              </Badge>
                            )}
                          </div>
                          {captchaSettings?.cap_api_key_set && (
                            <Badge
                              variant="secondary"
                              className="flex w-fit items-center gap-1"
                            >
                              <CheckCircle2 className="h-3 w-3" />
                              API key configured
                            </Badge>
                          )}
                        </div>
                      </div>
                    </>
                  )}

                  {/* Protected Endpoints */}
                  <div className="space-y-2">
                    <Label>Protected Endpoints</Label>
                    <p className="text-muted-foreground mb-2 text-sm">
                      Select which authentication endpoints require CAPTCHA
                      verification
                    </p>
                    <div className="space-y-2">
                      {[
                        { id: "signup", label: "Signup" },
                        { id: "login", label: "Login" },
                        { id: "password_reset", label: "Password Reset" },
                        { id: "magic_link", label: "Magic Link" },
                      ].map((endpoint) => (
                        <div
                          key={endpoint.id}
                          className="flex items-center space-x-2"
                        >
                          <Checkbox
                            id={endpoint.id}
                            checked={captchaForm.endpoints.includes(
                              endpoint.id,
                            )}
                            onCheckedChange={() => toggleEndpoint(endpoint.id)}
                            disabled={isOverridden("endpoints")}
                          />
                          <Label
                            htmlFor={endpoint.id}
                            className="cursor-pointer"
                          >
                            {endpoint.label}
                          </Label>
                        </div>
                      ))}
                    </div>
                    {isOverridden("endpoints") && (
                      <Badge variant="outline" className="mt-2">
                        ENV: {getEnvVar("endpoints")}
                      </Badge>
                    )}
                  </div>

                  {/* Save Button */}
                  <div className="flex items-center justify-between border-t pt-4">
                    <div>
                      {hasUnsavedChanges && (
                        <p className="text-muted-foreground flex items-center gap-2 text-sm">
                          <Info className="h-4 w-4" />
                          You have unsaved changes
                        </p>
                      )}
                    </div>
                    <Button
                      onClick={handleSaveCaptcha}
                      disabled={
                        !hasUnsavedChanges || updateCaptchaMutation.isPending
                      }
                    >
                      {updateCaptchaMutation.isPending ? (
                        <>
                          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                          Saving...
                        </>
                      ) : (
                        "Save Changes"
                      )}
                    </Button>
                  </div>
                </>
              )}

              {/* Warning about config overrides */}
              {captchaSettings &&
                Object.values(captchaSettings._overrides).some(
                  (o) => o.is_overridden,
                ) && (
                  <div className="bg-muted flex items-start gap-2 rounded-lg p-4">
                    <AlertCircle className="text-muted-foreground mt-0.5 h-5 w-5" />
                    <div className="text-sm">
                      <p className="font-medium">
                        Some settings are controlled by configuration
                      </p>
                      <p className="text-muted-foreground">
                        Settings marked with ENV cannot be changed through the
                        platform. Update your configuration file or environment
                        variables to modify these settings.
                      </p>
                    </div>
                  </div>
                )}
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
