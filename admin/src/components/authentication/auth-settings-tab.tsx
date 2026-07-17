import { useState, useMemo, useEffect, useRef } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Key,
  Shield,
  Settings,
  FileText,
  Users,
  Check,
  Loader2,
} from "lucide-react";
import { toast } from "sonner";
import {
  oauthProviderApi,
  authSettingsApi,
  samlProviderApi,
  dashboardAuthAPI,
  type AuthSettings,
} from "@/lib/api";
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

export function AuthSettingsTab() {
  const queryClient = useQueryClient();

  const { data: fetchedSettings, isLoading } = useQuery({
    queryKey: ["authSettings"],
    queryFn: authSettingsApi.get,
  });

  const { data: ssoData } = useQuery({
    queryKey: ["dashboard-sso-providers"],
    queryFn: dashboardAuthAPI.getSSOProviders,
  });

  const hasDashboardSSOProviders = (ssoData?.providers?.length ?? 0) > 0;

  const { data: oauthProviders = [] } = useQuery({
    queryKey: ["oauthProviders"],
    queryFn: oauthProviderApi.list,
  });

  const { data: samlProviders = [] } = useQuery({
    queryKey: ["samlProviders"],
    queryFn: samlProviderApi.list,
  });

  const hasAppSSOProviders =
    (oauthProviders?.filter((p) => p.allow_app_login)?.length ?? 0) > 0 ||
    (samlProviders?.filter((p) => p.allow_app_login)?.length ?? 0) > 0;

  const initialSettings = useMemo(
    () => fetchedSettings || null,
    [fetchedSettings],
  );

  const [settings, setSettings] = useState<AuthSettings | null>(
    initialSettings,
  );

  const prevFetchedRef = useRef<AuthSettings | null>(null);

  useEffect(() => {
    if (fetchedSettings && prevFetchedRef.current !== fetchedSettings) {
      prevFetchedRef.current = fetchedSettings;
      setSettings(fetchedSettings);
    }
  }, [fetchedSettings]);

  const updateSettingsMutation = useMutation({
    mutationFn: (data: AuthSettings) => authSettingsApi.update(data),
    onSuccess: (data) => {
      toast.success(data.message);
      queryClient.invalidateQueries({ queryKey: ["authSettings"] });
    },
    onError: (error: unknown) => {
      const errorMessage =
        error instanceof Error && "response" in error
          ? (error as { response?: { data?: { error?: string } } }).response
              ?.data?.error || "Failed to update auth settings"
          : "Failed to update auth settings";
      toast.error(errorMessage);
    },
  });

  const handleSaveSettings = () => {
    if (settings) {
      updateSettingsMutation.mutate(settings);
    }
  };

  if (isLoading || !settings) {
    return (
      <div className="flex justify-center p-8">
        <Loader2 className="h-6 w-6 animate-spin" />
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Key className="h-5 w-5" />
            Authentication Methods
          </CardTitle>
          <CardDescription>
            Enable or disable authentication methods
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <Label htmlFor="enableSignup">Enable User Signup</Label>
              <p className="text-muted-foreground text-sm">
                Allow new users to register accounts
              </p>
            </div>
            <Switch
              id="enableSignup"
              checked={settings.enable_signup}
              onCheckedChange={(checked) =>
                setSettings({ ...settings, enable_signup: checked })
              }
            />
          </div>
          <div className="flex items-center justify-between">
            <div>
              <Label htmlFor="enableMagicLink">Enable Magic Link</Label>
              <p className="text-muted-foreground text-sm">
                Allow users to sign in via email magic links
              </p>
            </div>
            <Switch
              id="enableMagicLink"
              checked={settings.enable_magic_link}
              onCheckedChange={(checked) =>
                setSettings({ ...settings, enable_magic_link: checked })
              }
            />
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Shield className="h-5 w-5" />
            Password Requirements
          </CardTitle>
          <CardDescription>
            Configure password complexity requirements
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-2">
            <Label htmlFor="minLength">Minimum Length</Label>
            <Input
              id="minLength"
              type="number"
              value={settings.password_min_length}
              onChange={(e) =>
                setSettings({
                  ...settings,
                  password_min_length: parseInt(e.target.value),
                })
              }
            />
          </div>
          <div className="flex items-center justify-between">
            <Label htmlFor="uppercase">Require Uppercase Letters</Label>
            <Switch
              id="uppercase"
              checked={settings.password_require_uppercase}
              onCheckedChange={(checked) =>
                setSettings({
                  ...settings,
                  password_require_uppercase: checked,
                })
              }
            />
          </div>
          <div className="flex items-center justify-between">
            <Label htmlFor="numbers">Require Numbers</Label>
            <Switch
              id="numbers"
              checked={settings.password_require_number}
              onCheckedChange={(checked) =>
                setSettings({ ...settings, password_require_number: checked })
              }
            />
          </div>
          <div className="flex items-center justify-between">
            <Label htmlFor="symbols">Require Symbols</Label>
            <Switch
              id="symbols"
              checked={settings.password_require_special}
              onCheckedChange={(checked) =>
                setSettings({ ...settings, password_require_special: checked })
              }
            />
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Settings className="h-5 w-5" />
            Session & Token Configurations
          </CardTitle>
          <CardDescription>
            Configure session and token expiration times
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-2">
            <Label htmlFor="sessionTimeout">Session Timeout (minutes)</Label>
            <Input
              id="sessionTimeout"
              type="number"
              value={settings.session_timeout_minutes}
              onChange={(e) =>
                setSettings({
                  ...settings,
                  session_timeout_minutes: parseInt(e.target.value),
                })
              }
            />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="maxSessions">Max Sessions Per User</Label>
            <Input
              id="maxSessions"
              type="number"
              value={settings.max_sessions_per_user}
              onChange={(e) =>
                setSettings({
                  ...settings,
                  max_sessions_per_user: parseInt(e.target.value),
                })
              }
            />
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <FileText className="h-5 w-5" />
            Email Verification
          </CardTitle>
          <CardDescription>
            Configure email verification requirements
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-between">
            <div>
              <Label htmlFor="emailVerification">
                Require Email Verification
              </Label>
              <p className="text-muted-foreground text-sm">
                Users must verify their email before accessing the application
              </p>
            </div>
            <Switch
              id="emailVerification"
              checked={settings.require_email_verification}
              onCheckedChange={(checked) =>
                setSettings({
                  ...settings,
                  require_email_verification: checked,
                })
              }
            />
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Shield className="h-5 w-5" />
            Dashboard Login
          </CardTitle>
          <CardDescription>
            Configure authentication methods for dashboard admins
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between">
            <div className="flex-1 pr-4">
              <Label htmlFor="disablePasswordLogin">
                Disable Password Login
              </Label>
              <p className="text-muted-foreground text-sm">
                Require SSO for all dashboard admin logins. Password
                authentication will be disabled.
              </p>
              {!hasDashboardSSOProviders && (
                <p className="mt-2 text-sm text-amber-600">
                  Configure at least one OAuth or SAML provider with "Allow
                  dashboard login" enabled before you can disable password
                  login.
                </p>
              )}
            </div>
            <Switch
              id="disablePasswordLogin"
              checked={settings.disable_dashboard_password_login}
              onCheckedChange={(checked) =>
                setSettings({
                  ...settings,
                  disable_dashboard_password_login: checked,
                })
              }
              disabled={
                !hasDashboardSSOProviders &&
                !settings.disable_dashboard_password_login
              }
            />
          </div>
          {settings.disable_dashboard_password_login && (
            <div className="rounded-md border border-amber-200 bg-amber-50 p-3 dark:border-amber-800 dark:bg-amber-950">
              <p className="text-sm text-amber-800 dark:text-amber-200">
                <strong>Recovery:</strong> If you get locked out, set the
                environment variable{" "}
                <code className="rounded bg-amber-100 px-1 dark:bg-amber-900">
                  FLUXBASE_DASHBOARD_FORCE_PASSWORD_LOGIN=true
                </code>{" "}
                to temporarily re-enable password login.
              </p>
            </div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Users className="h-5 w-5" />
            App User Login
          </CardTitle>
          <CardDescription>
            Configure authentication methods for application users
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between">
            <div className="flex-1 pr-4">
              <Label htmlFor="disableAppPasswordLogin">
                Disable Password Login
              </Label>
              <p className="text-muted-foreground text-sm">
                Require OAuth/SAML for all app user logins. Password
                authentication will be disabled.
              </p>
              {!hasAppSSOProviders && (
                <p className="mt-2 text-sm text-amber-600">
                  Configure at least one OAuth or SAML provider with "Allow app
                  login" enabled before you can disable password login.
                </p>
              )}
            </div>
            <Switch
              id="disableAppPasswordLogin"
              checked={settings.disable_app_password_login}
              onCheckedChange={(checked) =>
                setSettings({
                  ...settings,
                  disable_app_password_login: checked,
                })
              }
              disabled={
                !hasAppSSOProviders && !settings.disable_app_password_login
              }
            />
          </div>
          {settings.disable_app_password_login && (
            <div className="rounded-md border border-amber-200 bg-amber-50 p-3 dark:border-amber-800 dark:bg-amber-950">
              <p className="text-sm text-amber-800 dark:text-amber-200">
                <strong>Recovery:</strong> If users get locked out, set the
                environment variable{" "}
                <code className="rounded bg-amber-100 px-1 dark:bg-amber-900">
                  FLUXBASE_APP_FORCE_PASSWORD_LOGIN=true
                </code>{" "}
                to temporarily re-enable password login.
              </p>
            </div>
          )}
        </CardContent>
      </Card>

      <div className="flex justify-end">
        <Button onClick={handleSaveSettings}>
          <Check className="mr-2 h-4 w-4" />
          Save Settings
        </Button>
      </div>
    </div>
  );
}
