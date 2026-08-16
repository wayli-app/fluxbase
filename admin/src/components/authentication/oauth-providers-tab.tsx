import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Key,
  Loader2,
  X,
  Check,
  AlertCircle,
  Settings,
  Plus,
} from "lucide-react";
import { toast } from "sonner";
import {
  oauthProviderApi,
  type OAuthProviderConfig,
  type CreateOAuthProviderRequest,
  type UpdateOAuthProviderRequest,
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
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { KeyValueArrayEditor } from "@/components/key-value-array-editor";
import { OAUTH_AVAILABLE_PROVIDERS } from "./types";

interface OAuthProvidersTabProps {
  onProviderTest?: (provider: OAuthProviderConfig) => void;
}

export function OAuthProvidersTab({ onProviderTest }: OAuthProvidersTabProps) {
  const queryClient = useQueryClient();
  const [isAddProviderOpen, setIsAddProviderOpen] = useState(false);
  const [isEditProviderOpen, setIsEditProviderOpen] = useState(false);
  const [editingProvider, setEditingProvider] =
    useState<OAuthProviderConfig | null>(null);
  const [selectedProvider, setSelectedProvider] = useState<string>("");
  const [customProviderName, setCustomProviderName] = useState("");
  const [customAuthUrl, setCustomAuthUrl] = useState("");
  const [customTokenUrl, setCustomTokenUrl] = useState("");
  const [customUserInfoUrl, setCustomUserInfoUrl] = useState("");
  const [oidcDiscoveryUrl, setOidcDiscoveryUrl] = useState("");
  const [isDiscovering, setIsDiscovering] = useState(false);
  const [clientId, setClientId] = useState("");
  const [clientSecret, setClientSecret] = useState("");
  const [editRedirectUrls, setEditRedirectUrls] = useState<string[]>([]);
  const [allowDashboardLogin, setAllowDashboardLogin] = useState(false);
  const [allowAppLogin, setAllowAppLogin] = useState(true);
  const [requiredClaims, setRequiredClaims] = useState<
    Record<string, string[]>
  >({});
  const [deniedClaims, setDeniedClaims] = useState<Record<string, string[]>>(
    {},
  );
  const [isDeleteProviderConfirmOpen, setIsDeleteProviderConfirmOpen] =
    useState(false);
  const [deletingProvider, setDeletingProvider] =
    useState<OAuthProviderConfig | null>(null);

  const { data: enabledProviders = [] } = useQuery({
    queryKey: ["oauthProviders"],
    queryFn: oauthProviderApi.list,
  });

  const createProviderMutation = useMutation({
    mutationFn: (data: CreateOAuthProviderRequest) =>
      oauthProviderApi.create(data),
    onSuccess: (data) => {
      toast.success(data.message);
      queryClient.invalidateQueries({ queryKey: ["oauthProviders"] });
      setIsAddProviderOpen(false);
      resetForm();
    },
    onError: (error: unknown) => {
      const errorMessage =
        error instanceof Error && "response" in error
          ? (error as { response?: { data?: { error?: string } } }).response
              ?.data?.error || "Failed to create OAuth provider"
          : "Failed to create OAuth provider";
      toast.error(errorMessage);
    },
  });

  const updateProviderMutation = useMutation({
    mutationFn: ({
      id,
      data,
    }: {
      id: string;
      data: UpdateOAuthProviderRequest;
    }) => oauthProviderApi.update(id, data),
    onSuccess: (data) => {
      toast.success(data.message);
      queryClient.invalidateQueries({ queryKey: ["oauthProviders"] });
      setIsEditProviderOpen(false);
      setEditingProvider(null);
      resetForm();
    },
    onError: (error: unknown) => {
      const errorMessage =
        error instanceof Error && "response" in error
          ? (error as { response?: { data?: { error?: string } } }).response
              ?.data?.error || "Failed to update OAuth provider"
          : "Failed to update OAuth provider";
      toast.error(errorMessage);
    },
  });

  const deleteProviderMutation = useMutation({
    mutationFn: (id: string) => oauthProviderApi.delete(id),
    onSuccess: (data) => {
      toast.success(data.message);
      queryClient.invalidateQueries({ queryKey: ["oauthProviders"] });
    },
    onError: (error: unknown) => {
      const errorMessage =
        error instanceof Error && "response" in error
          ? (error as { response?: { data?: { error?: string } } }).response
              ?.data?.error || "Failed to delete OAuth provider"
          : "Failed to delete OAuth provider";
      toast.error(errorMessage);
    },
  });

  const resetForm = () => {
    setSelectedProvider("");
    setCustomProviderName("");
    setCustomAuthUrl("");
    setCustomTokenUrl("");
    setCustomUserInfoUrl("");
    setOidcDiscoveryUrl("");
    setIsDiscovering(false);
    setClientId("");
    setClientSecret("");
    setEditRedirectUrls([]);
    setAllowDashboardLogin(false);
    setAllowAppLogin(true);
    setRequiredClaims({});
    setDeniedClaims({});
  };

  const handleEditProvider = (provider: OAuthProviderConfig) => {
    setEditingProvider(provider);
    setSelectedProvider(provider.id);
    setCustomProviderName(provider.display_name);
    setClientId(provider.client_id);
    setClientSecret("");
    setEditRedirectUrls(
      provider.redirect_urls?.length
        ? [...provider.redirect_urls]
        : provider.redirect_url
          ? [provider.redirect_url]
          : [],
    );
    setAllowDashboardLogin(provider.allow_dashboard_login);
    setAllowAppLogin(provider.allow_app_login);
    setRequiredClaims(provider.required_claims || {});
    setDeniedClaims(provider.denied_claims || {});
    if (provider.is_custom) {
      setCustomAuthUrl(provider.authorization_url || "");
      setCustomTokenUrl(provider.token_url || "");
      setCustomUserInfoUrl(provider.user_info_url || "");
    }
    setIsEditProviderOpen(true);
  };

  const handleTestProvider = (provider: OAuthProviderConfig) => {
    if (onProviderTest) {
      onProviderTest(provider);
    } else {
      const redirectUrl =
        provider.redirect_urls?.[0] ?? provider.redirect_url ?? "";
      const authUrl = provider.is_custom
        ? provider.authorization_url
        : `https://accounts.google.com/o/oauth2/v2/auth?client_id=${provider.client_id}&redirect_uri=${encodeURIComponent(redirectUrl)}&response_type=code&scope=${provider.scopes.join(" ")}`;
      window.open(authUrl, "_blank", "width=500,height=600");
      toast.success("Test authentication window opened");
    }
  };

  const handleCreateProvider = () => {
    const isCustom = selectedProvider === "custom";
    const providerName = isCustom
      ? customProviderName.toLowerCase().replace(/\s+/g, "_")
      : selectedProvider;

    const data: CreateOAuthProviderRequest = {
      provider_name: providerName,
      display_name: isCustom
        ? customProviderName
        : selectedProvider.charAt(0).toUpperCase() + selectedProvider.slice(1),
      enabled: true,
      client_id: clientId,
      client_secret: clientSecret,
      redirect_urls: [
        `${window.location.origin}/api/v1/auth/oauth/${providerName}/callback`,
      ],
      scopes: ["openid", "email", "profile"],
      is_custom: isCustom,
      allow_dashboard_login: allowDashboardLogin,
      allow_app_login: allowAppLogin,
      ...(isCustom && {
        authorization_url: customAuthUrl,
        token_url: customTokenUrl,
        user_info_url: customUserInfoUrl,
      }),
      ...(Object.keys(requiredClaims).length > 0 && {
        required_claims: requiredClaims,
      }),
      ...(Object.keys(deniedClaims).length > 0 && {
        denied_claims: deniedClaims,
      }),
    };

    createProviderMutation.mutate(data);
  };

  const handleUpdateProvider = () => {
    if (!editingProvider) return;

    const redirectUrls = editRedirectUrls
      .map((url) => url.trim())
      .filter((url) => url !== "");
    if (redirectUrls.length === 0) {
      toast.error("At least one redirect URL is required");
      return;
    }

    const data: UpdateOAuthProviderRequest = {
      display_name: editingProvider.display_name,
      enabled: editingProvider.enabled,
      client_id: clientId,
      redirect_urls: redirectUrls,
      allow_dashboard_login: allowDashboardLogin,
      allow_app_login: allowAppLogin,
      ...(clientSecret && { client_secret: clientSecret }),
      ...(Object.keys(requiredClaims).length > 0 && {
        required_claims: requiredClaims,
      }),
      ...(Object.keys(deniedClaims).length > 0 && {
        denied_claims: deniedClaims,
      }),
    };

    updateProviderMutation.mutate({ id: editingProvider.id, data });
  };

  const handleDiscover = async () => {
    if (!oidcDiscoveryUrl) {
      toast.error("Please enter a discovery URL");
      return;
    }

    try {
      setIsDiscovering(true);

      let discoveryUrl = oidcDiscoveryUrl.trim();
      if (!discoveryUrl.includes(".well-known")) {
        discoveryUrl = discoveryUrl.replace(/\/$/, "");
        discoveryUrl = `${discoveryUrl}/.well-known/openid-configuration`;
      }

      const response = await fetch(discoveryUrl);
      if (!response.ok) {
        throw new Error(`Failed to fetch: ${response.statusText}`);
      }

      const config = await response.json();

      if (config.authorization_endpoint) {
        setCustomAuthUrl(config.authorization_endpoint);
      }
      if (config.token_endpoint) {
        setCustomTokenUrl(config.token_endpoint);
      }
      if (config.userinfo_endpoint) {
        setCustomUserInfoUrl(config.userinfo_endpoint);
      }

      toast.success("Auto-discovered OAuth endpoints!");
    } catch (error) {
      toast.error(
        `Discovery failed: ${error instanceof Error ? error.message : "Unknown error"}`,
      );
    } finally {
      setIsDiscovering(false);
    }
  };

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>OAuth Providers</CardTitle>
              <CardDescription>
                Configure external OAuth providers for social authentication
              </CardDescription>
            </div>
            <Button onClick={() => setIsAddProviderOpen(true)}>
              <Key className="mr-2 h-4 w-4" />
              Add Provider
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {enabledProviders.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 text-center">
              <AlertCircle className="text-muted-foreground mb-4 h-12 w-12" />
              <p className="text-muted-foreground">
                No OAuth providers configured
              </p>
              <Button
                onClick={() => setIsAddProviderOpen(true)}
                variant="outline"
                className="mt-4"
              >
                Add Your First Provider
              </Button>
            </div>
          ) : (
            <div className="space-y-4">
              {enabledProviders.map((provider) => (
                <Card key={provider.id}>
                  <CardContent className="pt-6">
                    <div className="flex items-start justify-between">
                      <div className="flex-1 space-y-2">
                        <div className="flex items-center gap-2">
                          <h3 className="text-lg font-semibold">
                            {provider.display_name}
                          </h3>
                          {provider.enabled ? (
                            <Badge variant="default" className="gap-1">
                              <Check className="h-3 w-3" />
                              Enabled
                            </Badge>
                          ) : (
                            <Badge variant="secondary">Disabled</Badge>
                          )}
                          {provider.allow_dashboard_login && (
                            <Badge variant="secondary" className="text-xs">
                              Dashboard
                            </Badge>
                          )}
                          {provider.allow_app_login && (
                            <Badge variant="outline" className="text-xs">
                              App
                            </Badge>
                          )}
                          {provider.source === "config" && (
                            <Badge
                              variant="secondary"
                              className="gap-1 text-xs"
                            >
                              <Settings className="h-3 w-3" />
                              Config
                            </Badge>
                          )}
                          {provider.tenant_id ? (
                            <Badge variant="outline" className="text-xs">
                              Tenant
                            </Badge>
                          ) : (
                            <Badge variant="secondary" className="text-xs">
                              Instance
                            </Badge>
                          )}
                        </div>
                        <div className="grid grid-cols-2 gap-4 text-sm">
                          <div>
                            <Label className="text-muted-foreground">
                              Client ID
                            </Label>
                            <p className="font-mono text-xs break-all">
                              {provider.client_id}
                            </p>
                          </div>
                          <div>
                            <Label className="text-muted-foreground">
                              Client Secret
                            </Label>
                            <p className="font-mono text-xs">
                              {provider.has_secret ? "••••••••" : "Not set"}
                            </p>
                          </div>
                          <div className="col-span-2">
                            <Label className="text-muted-foreground">
                              Redirect URL
                              {(provider.redirect_urls?.length ?? 0) > 1 && "s"}
                            </Label>
                            {(provider.redirect_urls?.length
                              ? provider.redirect_urls
                              : provider.redirect_url
                                ? [provider.redirect_url]
                                : []
                            ).map((url) => (
                              <p
                                key={url}
                                className="font-mono text-xs break-all"
                              >
                                {url}
                              </p>
                            ))}
                          </div>
                          {provider.is_custom && (
                            <>
                              <div className="col-span-2">
                                <Label className="text-muted-foreground">
                                  Authorization URL
                                </Label>
                                <p className="font-mono text-xs break-all">
                                  {provider.authorization_url}
                                </p>
                              </div>
                              <div className="col-span-2">
                                <Label className="text-muted-foreground">
                                  Token URL
                                </Label>
                                <p className="font-mono text-xs break-all">
                                  {provider.token_url}
                                </p>
                              </div>
                              <div className="col-span-2">
                                <Label className="text-muted-foreground">
                                  User Info URL
                                </Label>
                                <p className="font-mono text-xs break-all">
                                  {provider.user_info_url}
                                </p>
                              </div>
                            </>
                          )}
                          <div className="col-span-2">
                            <Label className="text-muted-foreground">
                              Scopes
                            </Label>
                            <div className="mt-1 flex flex-wrap gap-1">
                              {provider.scopes.map((scope) => (
                                <Badge
                                  key={scope}
                                  variant="outline"
                                  className="text-xs"
                                >
                                  {scope}
                                </Badge>
                              ))}
                            </div>
                          </div>
                          {(provider.required_claims ||
                            provider.denied_claims) && (
                            <div className="col-span-2 border-t pt-3">
                              <Label className="text-muted-foreground mb-2 block">
                                RBAC Rules
                              </Label>
                              {provider.required_claims &&
                                Object.keys(provider.required_claims).length >
                                  0 && (
                                  <div className="mb-2">
                                    <span className="text-muted-foreground text-xs">
                                      Required Claims:{" "}
                                    </span>
                                    <div className="mt-1 flex flex-wrap gap-1">
                                      {Object.entries(
                                        provider.required_claims,
                                      ).map(([key, values]) => (
                                        <Badge
                                          key={key}
                                          variant="outline"
                                          className="text-xs"
                                        >
                                          {key}: {values.join(", ")}
                                        </Badge>
                                      ))}
                                    </div>
                                  </div>
                                )}
                              {provider.denied_claims &&
                                Object.keys(provider.denied_claims).length >
                                  0 && (
                                  <div>
                                    <span className="text-muted-foreground text-xs">
                                      Denied Claims:{" "}
                                    </span>
                                    <div className="mt-1 flex flex-wrap gap-1">
                                      {Object.entries(
                                        provider.denied_claims,
                                      ).map(([key, values]) => (
                                        <Badge
                                          key={key}
                                          variant="destructive"
                                          className="text-xs"
                                        >
                                          {key}: {values.join(", ")}
                                        </Badge>
                                      ))}
                                    </div>
                                  </div>
                                )}
                            </div>
                          )}
                        </div>
                      </div>
                      <div className="ml-4 flex gap-2">
                        {provider.source !== "config" && (
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => handleEditProvider(provider)}
                          >
                            Edit
                          </Button>
                        )}
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => handleTestProvider(provider)}
                        >
                          Test
                        </Button>
                        {provider.source !== "config" && (
                          <Button
                            variant="destructive"
                            size="sm"
                            onClick={() => {
                              setDeletingProvider(provider);
                              setIsDeleteProviderConfirmOpen(true);
                            }}
                            disabled={deleteProviderMutation.isPending}
                          >
                            <X className="h-4 w-4" />
                          </Button>
                        )}
                      </div>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      <Dialog open={isAddProviderOpen} onOpenChange={setIsAddProviderOpen}>
        <DialogContent className="max-h-[90vh] max-w-2xl overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Add OAuth Provider</DialogTitle>
            <DialogDescription>
              Configure a new OAuth provider for social authentication
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="provider">Provider</Label>
              <select
                id="provider"
                className="border-input bg-background ring-offset-background flex h-10 w-full rounded-md border px-3 py-2 text-sm"
                value={selectedProvider}
                onChange={(e) => setSelectedProvider(e.target.value)}
              >
                <option value="">Select a provider...</option>
                {OAUTH_AVAILABLE_PROVIDERS.filter(
                  (p) =>
                    !enabledProviders.some(
                      (ep) => ep.id === p.id && p.id !== "custom",
                    ),
                ).map((provider) => (
                  <option key={provider.id} value={provider.id}>
                    {provider.icon} {provider.name}
                  </option>
                ))}
              </select>
            </div>

            {selectedProvider === "custom" && (
              <>
                <div className="grid gap-2">
                  <Label htmlFor="customProviderName">Provider Name</Label>
                  <Input
                    id="customProviderName"
                    placeholder="e.g., Okta, Auth0, Keycloak"
                    value={customProviderName}
                    onChange={(e) => setCustomProviderName(e.target.value)}
                  />
                </div>

                <div className="grid gap-2">
                  <Label htmlFor="oidcDiscoveryUrl">
                    OpenID Discovery URL (Optional)
                  </Label>
                  <div className="flex gap-2">
                    <Input
                      id="oidcDiscoveryUrl"
                      placeholder="https://auth.example.com"
                      value={oidcDiscoveryUrl}
                      onChange={(e) => setOidcDiscoveryUrl(e.target.value)}
                    />
                    <Button
                      type="button"
                      variant="outline"
                      onClick={handleDiscover}
                      disabled={!oidcDiscoveryUrl || isDiscovering}
                    >
                      {isDiscovering ? (
                        <>
                          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                          Discovering...
                        </>
                      ) : (
                        "Auto-discover"
                      )}
                    </Button>
                  </div>
                  <p className="text-muted-foreground text-xs">
                    Supports base URLs or full discovery URLs. Auto-discovery
                    will be used:
                    <br />• Base URL:{" "}
                    <code className="text-xs">
                      https://auth.example.com
                    </code>{" "}
                    (auto-appends /.well-known/openid-configuration)
                    <br />• Auth0:{" "}
                    <code className="text-xs">
                      https://YOUR-DOMAIN.auth0.com
                    </code>
                    <br />• Keycloak:{" "}
                    <code className="text-xs">
                      https://YOUR-DOMAIN/realms/YOUR-REALM
                    </code>
                    <br />• Custom:{" "}
                    <code className="text-xs">
                      https://auth.example.com/.well-known/custom-oidc
                    </code>
                  </p>
                </div>

                <div className="grid gap-2">
                  <Label htmlFor="authorizationUrl">Authorization URL</Label>
                  <Input
                    id="authorizationUrl"
                    placeholder="https://your-provider.com/oauth/authorize"
                    value={customAuthUrl}
                    onChange={(e) => setCustomAuthUrl(e.target.value)}
                  />
                  <p className="text-muted-foreground text-xs">
                    The OAuth authorization endpoint
                  </p>
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="tokenUrl">Token URL</Label>
                  <Input
                    id="tokenUrl"
                    placeholder="https://your-provider.com/oauth/token"
                    value={customTokenUrl}
                    onChange={(e) => setCustomTokenUrl(e.target.value)}
                  />
                  <p className="text-muted-foreground text-xs">
                    The OAuth token exchange endpoint
                  </p>
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="userInfoUrl">User Info URL</Label>
                  <Input
                    id="userInfoUrl"
                    placeholder="https://your-provider.com/oauth/userinfo"
                    value={customUserInfoUrl}
                    onChange={(e) => setCustomUserInfoUrl(e.target.value)}
                  />
                  <p className="text-muted-foreground text-xs">
                    The endpoint to retrieve user information
                  </p>
                </div>
              </>
            )}

            <div className="grid gap-2">
              <Label htmlFor="clientId">Client ID</Label>
              <Input
                id="clientId"
                placeholder="your-client-id"
                value={clientId}
                onChange={(e) => setClientId(e.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="clientSecret">Client Secret</Label>
              <Input
                id="clientSecret"
                type="password"
                placeholder="your-client-secret"
                value={clientSecret}
                onChange={(e) => setClientSecret(e.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="redirectUrl">Redirect URL</Label>
              <Input
                id="redirectUrl"
                value={
                  selectedProvider === "custom"
                    ? `${window.location.origin}/api/v1/auth/oauth/${customProviderName.toLowerCase().replace(/\s+/g, "_") || "custom"}/callback`
                    : `${window.location.origin}/api/v1/auth/oauth/${selectedProvider}/callback`
                }
                readOnly
                className="font-mono text-xs"
              />
              <p className="text-muted-foreground text-xs">
                Use this URL in your OAuth provider configuration
              </p>
            </div>

            <div className="space-y-3 border-t pt-4">
              <div>
                <Label className="text-sm font-semibold">
                  Provider Targeting
                </Label>
                <p className="text-muted-foreground mt-1 text-xs">
                  Control which authentication contexts can use this provider
                </p>
              </div>
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <div>
                    <Label>Allow for App Users</Label>
                    <p className="text-muted-foreground text-xs">
                      Enable this provider for application user authentication
                    </p>
                  </div>
                  <Switch
                    checked={allowAppLogin}
                    onCheckedChange={setAllowAppLogin}
                  />
                </div>
                <div className="flex items-center justify-between">
                  <div>
                    <Label>Allow for Dashboard</Label>
                    <p className="text-muted-foreground text-xs">
                      Enable this provider for dashboard admin authentication
                    </p>
                  </div>
                  <Switch
                    checked={allowDashboardLogin}
                    onCheckedChange={setAllowDashboardLogin}
                  />
                </div>
              </div>
            </div>

            <div className="space-y-4 border-t pt-4">
              <div>
                <Label className="text-sm font-semibold">
                  Role-Based Access Control (Optional)
                </Label>
                <p className="text-muted-foreground mt-1 text-xs">
                  Filter users based on ID token claims (e.g., roles, groups)
                </p>
              </div>

              <div className="space-y-2">
                <Label>Required Claims (OR logic)</Label>
                <p className="text-muted-foreground text-xs">
                  User must have at least ONE matching value per claim
                </p>
                <KeyValueArrayEditor
                  value={requiredClaims}
                  onChange={setRequiredClaims}
                  keyPlaceholder="Claim name (e.g., roles)"
                  valuePlaceholder="Allowed value"
                  addButtonText="Add Required Claim"
                />
              </div>

              <div className="space-y-2">
                <Label>Denied Claims (Blocklist)</Label>
                <p className="text-muted-foreground text-xs">
                  Reject users if ANY value matches
                </p>
                <KeyValueArrayEditor
                  value={deniedClaims}
                  onChange={setDeniedClaims}
                  keyPlaceholder="Claim name (e.g., status)"
                  valuePlaceholder="Denied value"
                  addButtonText="Add Denied Claim"
                />
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setIsAddProviderOpen(false)}
            >
              Cancel
            </Button>
            <Button
              onClick={handleCreateProvider}
              disabled={
                !selectedProvider ||
                !clientId ||
                !clientSecret ||
                createProviderMutation.isPending
              }
            >
              {createProviderMutation.isPending ? "Saving..." : "Save Provider"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={isEditProviderOpen} onOpenChange={setIsEditProviderOpen}>
        <DialogContent className="max-h-[90vh] max-w-2xl overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Edit OAuth Provider</DialogTitle>
            <DialogDescription>
              Update the configuration for {editingProvider?.display_name}
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label>Provider</Label>
              <Input
                value={editingProvider?.display_name || ""}
                disabled
                className="bg-muted"
              />
            </div>

            {editingProvider?.is_custom && (
              <>
                <div className="grid gap-2">
                  <Label htmlFor="editProviderName">Provider Name</Label>
                  <Input
                    id="editProviderName"
                    value={customProviderName}
                    onChange={(e) => setCustomProviderName(e.target.value)}
                  />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="editAuthorizationUrl">
                    Authorization URL
                  </Label>
                  <Input
                    id="editAuthorizationUrl"
                    value={customAuthUrl}
                    onChange={(e) => setCustomAuthUrl(e.target.value)}
                  />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="editTokenUrl">Token URL</Label>
                  <Input
                    id="editTokenUrl"
                    value={customTokenUrl}
                    onChange={(e) => setCustomTokenUrl(e.target.value)}
                  />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="editUserInfoUrl">User Info URL</Label>
                  <Input
                    id="editUserInfoUrl"
                    value={customUserInfoUrl}
                    onChange={(e) => setCustomUserInfoUrl(e.target.value)}
                  />
                </div>
              </>
            )}

            <div className="grid gap-2">
              <Label htmlFor="editClientId">Client ID</Label>
              <Input
                id="editClientId"
                value={clientId}
                onChange={(e) => setClientId(e.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="editClientSecret">Client Secret</Label>
              <Input
                id="editClientSecret"
                type="password"
                placeholder="Leave empty to keep current secret"
                value={clientSecret}
                onChange={(e) => setClientSecret(e.target.value)}
              />
              <p className="text-muted-foreground text-xs">
                Only provide a new secret if you want to change it
              </p>
            </div>
            <div className="grid gap-2">
              <Label>Redirect URLs</Label>
              <p className="text-muted-foreground text-xs">
                Allowed OAuth callback URLs. The first URL is the default; each
                URL must also be registered with the provider.
              </p>
              <div className="space-y-2">
                {editRedirectUrls.map((url, index) => (
                  <div key={index} className="flex items-center gap-2">
                    <Input
                      value={url}
                      onChange={(e) => {
                        const next = [...editRedirectUrls];
                        next[index] = e.target.value;
                        setEditRedirectUrls(next);
                      }}
                      className="bg-muted font-mono text-xs"
                      placeholder="https://app.example.com/api/v1/auth/oauth/{provider}/callback"
                      aria-label={`Redirect URL ${index + 1}`}
                    />
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      onClick={() =>
                        setEditRedirectUrls(
                          editRedirectUrls.filter((_, i) => i !== index),
                        )
                      }
                      disabled={editRedirectUrls.length <= 1}
                      aria-label={`Remove redirect URL ${index + 1}`}
                    >
                      <X className="h-4 w-4" />
                    </Button>
                  </div>
                ))}
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => setEditRedirectUrls([...editRedirectUrls, ""])}
                >
                  <Plus className="mr-2 h-4 w-4" />
                  Add URL
                </Button>
              </div>
            </div>

            <div className="space-y-3 border-t pt-4">
              <div>
                <Label className="text-sm font-semibold">
                  Provider Targeting
                </Label>
                <p className="text-muted-foreground mt-1 text-xs">
                  Control which authentication contexts can use this provider
                </p>
              </div>
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <div>
                    <Label>Allow for App Users</Label>
                    <p className="text-muted-foreground text-xs">
                      Enable this provider for application user authentication
                    </p>
                  </div>
                  <Switch
                    checked={allowAppLogin}
                    onCheckedChange={setAllowAppLogin}
                  />
                </div>
                <div className="flex items-center justify-between">
                  <div>
                    <Label>Allow for Dashboard</Label>
                    <p className="text-muted-foreground text-xs">
                      Enable this provider for dashboard admin authentication
                    </p>
                  </div>
                  <Switch
                    checked={allowDashboardLogin}
                    onCheckedChange={setAllowDashboardLogin}
                  />
                </div>
              </div>
            </div>

            <div className="space-y-4 border-t pt-4">
              <div>
                <Label className="text-sm font-semibold">
                  Role-Based Access Control (Optional)
                </Label>
                <p className="text-muted-foreground mt-1 text-xs">
                  Filter users based on ID token claims (e.g., roles, groups)
                </p>
              </div>

              <div className="space-y-2">
                <Label>Required Claims (OR logic)</Label>
                <p className="text-muted-foreground text-xs">
                  User must have at least ONE matching value per claim
                </p>
                <KeyValueArrayEditor
                  value={requiredClaims}
                  onChange={setRequiredClaims}
                  keyPlaceholder="Claim name (e.g., roles)"
                  valuePlaceholder="Allowed value"
                  addButtonText="Add Required Claim"
                />
              </div>

              <div className="space-y-2">
                <Label>Denied Claims (Blocklist)</Label>
                <p className="text-muted-foreground text-xs">
                  Reject users if ANY value matches
                </p>
                <KeyValueArrayEditor
                  value={deniedClaims}
                  onChange={setDeniedClaims}
                  keyPlaceholder="Claim name (e.g., status)"
                  valuePlaceholder="Denied value"
                  addButtonText="Add Denied Claim"
                />
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setIsEditProviderOpen(false);
                setEditingProvider(null);
              }}
            >
              Cancel
            </Button>
            <Button
              onClick={handleUpdateProvider}
              disabled={!editingProvider || updateProviderMutation.isPending}
            >
              {updateProviderMutation.isPending ? "Saving..." : "Save Changes"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={isDeleteProviderConfirmOpen}
        onOpenChange={setIsDeleteProviderConfirmOpen}
        title="Remove OAuth Provider"
        desc={`Are you sure you want to remove ${deletingProvider?.display_name}? Users will no longer be able to sign in with this provider.`}
        confirmText="Remove"
        destructive
        isLoading={deleteProviderMutation.isPending}
        handleConfirm={() => {
          if (deletingProvider) {
            deleteProviderMutation.mutate(deletingProvider.id, {
              onSuccess: () => {
                setIsDeleteProviderConfirmOpen(false);
                setDeletingProvider(null);
              },
            });
          }
        }}
      />
    </div>
  );
}
