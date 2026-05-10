import { useState } from "react";
import z from "zod";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, getRouteApi } from "@tanstack/react-router";
import type {
  EmailProviderSettings,
  TenantEmailProviderSettings,
} from "@nimbleflux/fluxbase-sdk";
import {
  Mail,
  FileText,
  Send,
  Loader2,
  Settings2,
  Building2,
  ArrowLeft,
  Trash2,
} from "lucide-react";
import { toast } from "sonner";
import { getErrorMessage } from "@/lib/get-error-message";
import { apiClient } from "@/lib/api";
import { fluxbaseClient } from "@/lib/fluxbase-client";
import { useTenantStore } from "@/stores/tenant-store";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  OverridableSelect,
  SelectItem,
} from "@/components/admin/overridable-select";
import { OverridableSwitch } from "@/components/admin/overridable-switch";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { PromptDialog } from "@/components/prompt-dialog";
import {
  SmtpForm,
  SendgridForm,
  MailgunForm,
  SesForm,
  CommonFields,
  TemplateCard,
  TemplateEditor,
  type EmailTemplate,
  type ProviderFormState,
  type ProviderType,
} from "@/components/email-settings";

const emailSettingsSearchSchema = z.object({
  tab: z.string().optional().catch("configuration"),
});

export const Route = createFileRoute("/_authenticated/email-settings/")({
  validateSearch: emailSettingsSearchSchema,
  component: EmailSettingsPage,
});

const route = getRouteApi("/_authenticated/email-settings/");

function EmailSettingsPage() {
  const queryClient = useQueryClient();
  const search = route.useSearch();
  const navigate = route.useNavigate();
  const { currentTenant, isInstanceAdmin, clearTenant } = useTenantStore();
  const isDefaultTenant = currentTenant?.is_default ?? false;
  const [selectedTemplate, setSelectedTemplate] = useState<string | null>(null);
  const [editingTemplate, setEditingTemplate] =
    useState<Partial<EmailTemplate> | null>(null);
  const [isResetConfirmOpen, setIsResetConfirmOpen] = useState(false);
  const [resetTemplateType, setResetTemplateType] = useState<string | null>(
    null,
  );
  const [isTestEmailPromptOpen, setIsTestEmailPromptOpen] = useState(false);
  const [testTemplateType, setTestTemplateType] = useState<string | null>(null);

  const [formState, setFormState] = useState<ProviderFormState>({
    from_address: "",
    from_name: "",
    smtp_host: "",
    smtp_port: "587",
    smtp_username: "",
    smtp_password: "",
    smtp_tls: true,
    sendgrid_api_key: "",
    mailgun_api_key: "",
    mailgun_domain: "",
    ses_access_key: "",
    ses_secret_key: "",
    ses_region: "us-east-1",
  });
  const [showPassword, setShowPassword] = useState(false);
  const [hasUnsavedChanges, setHasUnsavedChanges] = useState(false);
  const [initializedFromDataUpdatedAt, setInitializedFromDataUpdatedAt] =
    useState<number | null>(null);

  // Determine which settings to fetch
  const hasTenant = !!currentTenant;
  const showTenantLevel = isInstanceAdmin && hasTenant;

  // Instance-level settings (always fetched for instance admins)
  const {
    data: instanceSettings,
    isLoading: instanceSettingsLoading,
    dataUpdatedAt: instanceDataUpdatedAt,
  } = useQuery<EmailProviderSettings>({
    queryKey: ["email-provider-settings"],
    queryFn: () => fluxbaseClient.admin.settings.email.get(),
  });

  // Tenant-level settings (fetched when tenant is selected)
  const {
    data: tenantSettings,
    isLoading: tenantSettingsLoading,
    dataUpdatedAt: tenantSettingsDataUpdatedAt,
  } = useQuery<TenantEmailProviderSettings>({
    queryKey: ["email-provider-settings", "tenant", currentTenant?.id],
    queryFn: () => fluxbaseClient.admin.settings.email.getForTenant(),
    enabled: showTenantLevel,
  });

  // Use the appropriate settings based on context
  const settings = showTenantLevel ? tenantSettings : instanceSettings;
  const settingsLoading = showTenantLevel
    ? tenantSettingsLoading
    : instanceSettingsLoading;
  const dataUpdatedAt = showTenantLevel
    ? tenantSettingsDataUpdatedAt
    : instanceDataUpdatedAt;

  if (settings && dataUpdatedAt !== initializedFromDataUpdatedAt) {
    setInitializedFromDataUpdatedAt(dataUpdatedAt);
    setFormState({
      from_address: settings.from_address || "",
      from_name: settings.from_name || "",
      smtp_host: settings.smtp_host || "",
      smtp_port: String(settings.smtp_port || 587),
      smtp_username: settings.smtp_username || "",
      smtp_password: "",
      smtp_tls: settings.smtp_tls ?? true,
      sendgrid_api_key: "",
      mailgun_api_key: "",
      mailgun_domain: settings.mailgun_domain || "",
      ses_access_key: "",
      ses_secret_key: "",
      ses_region: settings.ses_region || "us-east-1",
    });
    setHasUnsavedChanges(false);
  }

  const { data: templates, isLoading: templatesLoading } = useQuery<
    EmailTemplate[]
  >({
    queryKey: ["email-templates"],
    queryFn: async () => {
      const response = await apiClient.get("/api/v1/admin/email/templates");
      return response.data;
    },
  });

  // Instance-level mutations
  const updateInstanceSettingsMutation = useMutation({
    mutationFn: (
      data: Parameters<typeof fluxbaseClient.admin.settings.email.update>[0],
    ) => fluxbaseClient.admin.settings.email.update(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["email-provider-settings"] });
      setHasUnsavedChanges(false);
      toast.success("Email settings updated");
    },
    onError: (error: unknown) => {
      if (error && typeof error === "object" && "response" in error) {
        const err = error as {
          response?: {
            status?: number;
            data?: { code?: string };
          };
        };
        if (
          err.response?.status === 409 &&
          err.response?.data?.code === "ENV_OVERRIDE"
        ) {
          toast.error(
            "This setting is controlled by an environment variable and cannot be changed",
          );
          return;
        }
      }
      toast.error(getErrorMessage(error));
    },
  });

  // Tenant-level mutations
  const updateTenantSettingsMutation = useMutation({
    mutationFn: (
      data: Parameters<
        typeof fluxbaseClient.admin.settings.email.updateForTenant
      >[0],
    ) => fluxbaseClient.admin.settings.email.updateForTenant(data),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["email-provider-settings", "tenant"],
      });
      setHasUnsavedChanges(false);
      toast.success("Tenant email settings updated");
    },
    onError: (error: unknown) => {
      toast.error(getErrorMessage(error));
    },
  });

  const deleteTenantOverrideMutation = useMutation({
    mutationFn: (field: string) =>
      fluxbaseClient.admin.settings.email.deleteTenantOverride(field),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["email-provider-settings", "tenant"],
      });
      toast.success("Tenant override removed, reverted to instance default");
    },
    onError: () => {
      toast.error("Failed to remove tenant override");
    },
  });

  const testSettingsMutation = useMutation({
    mutationFn: (email: string) => {
      if (showTenantLevel) {
        return fluxbaseClient.admin.settings.email.testForTenant(email);
      }
      return fluxbaseClient.admin.settings.email.test(email);
    },
    onSuccess: () => {
      toast.success("Test email sent successfully");
    },
    onError: (error: unknown) => {
      toast.error(getErrorMessage(error));
    },
  });

  const updateTemplateMutation = useMutation({
    mutationFn: async ({
      type,
      data,
    }: {
      type: string;
      data: Partial<EmailTemplate>;
    }) => {
      const response = await apiClient.put(
        `/api/v1/admin/email/templates/${type}`,
        data,
      );
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["email-templates"] });
      setEditingTemplate(null);
      toast.success("Template updated successfully");
    },
    onError: () => {
      toast.error("Failed to update template");
    },
  });

  const resetTemplateMutation = useMutation({
    mutationFn: async (type: string) => {
      const response = await apiClient.post(
        `/api/v1/admin/email/templates/${type}/reset`,
      );
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["email-templates"] });
      setEditingTemplate(null);
      toast.success("Template reset to default");
    },
    onError: () => {
      toast.error("Failed to reset template");
    },
  });

  const testTemplateMutation = useMutation({
    mutationFn: async ({ type, email }: { type: string; email: string }) => {
      await apiClient.post(`/api/v1/admin/email/templates/${type}/test`, {
        recipient_email: email,
      });
    },
    onSuccess: () => {
      toast.success("Test email sent (when email service is configured)");
    },
    onError: () => {
      toast.error("Failed to send test email");
    },
  });

  const handleToggleEnabled = (checked: boolean) => {
    if (showTenantLevel) {
      updateTenantSettingsMutation.mutate({ enabled: checked });
    } else {
      updateInstanceSettingsMutation.mutate({ enabled: checked });
    }
  };

  const handleProviderChange = (provider: string) => {
    if (showTenantLevel) {
      updateTenantSettingsMutation.mutate({
        provider: provider as ProviderType,
      });
    } else {
      updateInstanceSettingsMutation.mutate({
        provider: provider as ProviderType,
      });
    }
  };

  const handleFormChange = (
    field: keyof ProviderFormState,
    value: string | boolean,
  ) => {
    setFormState((prev) => ({ ...prev, [field]: value }));
    setHasUnsavedChanges(true);
  };

  const handleSaveProviderSettings = () => {
    const provider = settings?.provider || "smtp";
    const data: Record<string, unknown> = {
      from_address: formState.from_address || undefined,
      from_name: formState.from_name || undefined,
    };

    if (provider === "smtp") {
      data.smtp_host = formState.smtp_host || undefined;
      data.smtp_port = formState.smtp_port
        ? parseInt(formState.smtp_port)
        : undefined;
      data.smtp_username = formState.smtp_username || undefined;
      if (formState.smtp_password) {
        data.smtp_password = formState.smtp_password;
      }
      data.smtp_tls = formState.smtp_tls;
    } else if (provider === "sendgrid") {
      if (formState.sendgrid_api_key) {
        data.sendgrid_api_key = formState.sendgrid_api_key;
      }
    } else if (provider === "mailgun") {
      if (formState.mailgun_api_key) {
        data.mailgun_api_key = formState.mailgun_api_key;
      }
      data.mailgun_domain = formState.mailgun_domain || undefined;
    } else if (provider === "ses") {
      if (formState.ses_access_key) {
        data.ses_access_key = formState.ses_access_key;
      }
      if (formState.ses_secret_key) {
        data.ses_secret_key = formState.ses_secret_key;
      }
      data.ses_region = formState.ses_region || undefined;
    }

    if (showTenantLevel) {
      updateTenantSettingsMutation.mutate(data);
    } else {
      updateInstanceSettingsMutation.mutate(data);
    }
  };

  const handleTestConfiguration = () => {
    setTestTemplateType("config");
    setIsTestEmailPromptOpen(true);
  };

  const handleEditTemplate = (template: EmailTemplate) => {
    setSelectedTemplate(template.template_type);
    setEditingTemplate({
      subject: template.subject,
      html_body: template.html_body,
      text_body: template.text_body,
    });
  };

  const handleSaveTemplate = () => {
    if (!selectedTemplate || !editingTemplate) return;
    updateTemplateMutation.mutate({
      type: selectedTemplate,
      data: editingTemplate,
    });
  };

  const handleResetTemplate = (type: string) => {
    setResetTemplateType(type);
    setIsResetConfirmOpen(true);
  };

  const handleTestTemplate = (type: string) => {
    setTestTemplateType(type);
    setIsTestEmailPromptOpen(true);
  };

  const activeMutation = showTenantLevel
    ? updateTenantSettingsMutation
    : updateInstanceSettingsMutation;

  const providerFormProps = {
    formState,
    settings,
    showPassword,
    onFormChange: handleFormChange,
    onTogglePassword: () => setShowPassword(!showPassword),
  };

  if (settingsLoading || templatesLoading) {
    return (
      <div className="flex h-full items-center justify-center">
        <Loader2 className="text-muted-foreground h-8 w-8 animate-spin" />
      </div>
    );
  }

  const currentProvider = settings?.provider || "smtp";
  const sources = (tenantSettings as TenantEmailProviderSettings)?._sources;

  return (
    <div className="flex h-full flex-col">
      <div className="bg-background flex items-center justify-between border-b px-6 py-4">
        <div className="flex items-center gap-3">
          <div className="bg-primary/10 flex h-10 w-10 items-center justify-center rounded-lg">
            <Mail className="text-primary h-5 w-5" />
          </div>
          <div>
            <h1 className="text-xl font-semibold">Email Settings</h1>
            <p className="text-muted-foreground text-sm">
              {showTenantLevel
                ? `Tenant overrides for ${currentTenant?.name || "tenant"} — inherits from instance defaults`
                : "Configure email service and customize email templates"}
            </p>
          </div>
        </div>
        {showTenantLevel && (
          <div className="flex items-center gap-2">
            <Badge
              variant="outline"
              className="border-blue-300 bg-blue-50 text-blue-700 dark:border-blue-700 dark:bg-blue-950 dark:text-blue-300"
            >
              <Building2 className="mr-1 h-3 w-3" />
              {currentTenant?.name}
            </Badge>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => {
                clearTenant();
                navigate({
                  to: "/email-settings",
                  search: { tab: search.tab },
                });
              }}
            >
              <ArrowLeft className="mr-1 h-4 w-4" />
              Instance Settings
            </Button>
          </div>
        )}
      </div>

      <div className="flex-1 overflow-auto p-6">
        <Tabs
          value={search.tab || "configuration"}
          onValueChange={(tab) => navigate({ search: { tab } })}
          className="space-y-4"
        >
          <TabsList>
            <TabsTrigger
              value="configuration"
              className="flex items-center gap-2"
            >
              <Mail className="h-4 w-4" />
              Configuration
            </TabsTrigger>
            <TabsTrigger value="templates" className="flex items-center gap-2">
              <FileText className="h-4 w-4" />
              Email Templates
            </TabsTrigger>
          </TabsList>

          <TabsContent value="configuration" className="space-y-4">
            {/* Source indicators for tenant-level settings */}
            {showTenantLevel && sources && (
              <Card>
                <CardHeader className="pb-3">
                  <CardTitle className="text-base">Settings Source</CardTitle>
                  <CardDescription>
                    Fields highlighted in blue are overridden at the tenant
                    level. Others inherit from instance defaults.
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="flex flex-wrap gap-2">
                    {Object.entries(sources)
                      .filter(([, source]) => source === "tenant")
                      .map(([field]) => (
                        <Badge
                          key={field}
                          variant="outline"
                          className="border-blue-300 bg-blue-50 text-blue-700 dark:border-blue-700 dark:bg-blue-950 dark:text-blue-300"
                        >
                          {field}
                        </Badge>
                      ))}
                    {Object.entries(sources).filter(
                      ([, source]) => source === "tenant",
                    ).length === 0 && (
                      <span className="text-muted-foreground text-sm">
                        No tenant overrides — all settings inherited from
                        instance
                      </span>
                    )}
                  </div>
                </CardContent>
              </Card>
            )}

            <Card>
              <CardHeader>
                <CardTitle>Email Service Configuration</CardTitle>
                <CardDescription>
                  {showTenantLevel
                    ? "Override instance-level email settings for this tenant"
                    : "Configure your email service provider and settings"}
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-6">
                <OverridableSwitch
                  id="email-enabled"
                  label="Enable Email Service"
                  description="Enable or disable email functionality"
                  checked={settings?.enabled || false}
                  onCheckedChange={handleToggleEnabled}
                  override={settings?._overrides?.enabled}
                  disabled={activeMutation.isPending}
                />

                <OverridableSelect
                  id="email-provider"
                  label="Email Provider"
                  description="Select your email service provider"
                  value={currentProvider}
                  onValueChange={handleProviderChange}
                  override={settings?._overrides?.provider}
                  disabled={activeMutation.isPending}
                >
                  <SelectItem value="smtp">SMTP</SelectItem>
                  <SelectItem value="sendgrid">SendGrid</SelectItem>
                  <SelectItem value="mailgun">Mailgun</SelectItem>
                  <SelectItem value="ses">AWS SES</SelectItem>
                </OverridableSelect>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle className="flex items-center justify-between">
                  <span className="flex items-center gap-2">
                    <Settings2 className="h-5 w-5" />
                    {currentProvider === "smtp" && "SMTP Settings"}
                    {currentProvider === "sendgrid" && "SendGrid Settings"}
                    {currentProvider === "mailgun" && "Mailgun Settings"}
                    {currentProvider === "ses" && "AWS SES Settings"}
                  </span>
                  {showTenantLevel && isDefaultTenant && (
                    <Badge
                      variant="outline"
                      className="border-blue-300 bg-blue-50 text-blue-700 dark:border-blue-700 dark:bg-blue-950 dark:text-blue-300"
                    >
                      <Building2 className="mr-1 h-3 w-3" />
                      Tenant Override
                    </Badge>
                  )}
                </CardTitle>
                <CardDescription>
                  {showTenantLevel
                    ? `Override ${currentProvider.toUpperCase()} settings for this tenant`
                    : `Configure your ${currentProvider.toUpperCase()} provider settings`}
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <CommonFields
                  formState={formState}
                  settings={settings}
                  onFormChange={handleFormChange}
                />

                {currentProvider === "smtp" && (
                  <SmtpForm {...providerFormProps} />
                )}
                {currentProvider === "sendgrid" && (
                  <SendgridForm {...providerFormProps} />
                )}
                {currentProvider === "mailgun" && (
                  <MailgunForm {...providerFormProps} />
                )}
                {currentProvider === "ses" && (
                  <SesForm {...providerFormProps} />
                )}

                <div className="flex gap-2 pt-4">
                  <Button
                    onClick={handleSaveProviderSettings}
                    disabled={activeMutation.isPending || !hasUnsavedChanges}
                  >
                    {activeMutation.isPending && (
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    )}
                    Save Settings
                  </Button>
                  <Button
                    variant="outline"
                    onClick={handleTestConfiguration}
                    disabled={testSettingsMutation.isPending}
                  >
                    {testSettingsMutation.isPending && (
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    )}
                    <Send className="mr-2 h-4 w-4" />
                    Test Configuration
                  </Button>
                  {showTenantLevel && isDefaultTenant && (
                    <Button
                      variant="outline"
                      onClick={() => {
                        if (currentProvider) {
                          deleteTenantOverrideMutation.mutate(currentProvider);
                        }
                      }}
                      disabled={deleteTenantOverrideMutation.isPending}
                      className="text-destructive"
                    >
                      <Trash2 className="mr-2 h-4 w-4" />
                      Reset to Instance Default
                    </Button>
                  )}
                </div>
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="templates" className="space-y-4">
            {!selectedTemplate ? (
              <div className="grid gap-4 md:grid-cols-3">
                {templates?.map((template) => (
                  <TemplateCard
                    key={template.template_type}
                    template={template}
                    onEdit={handleEditTemplate}
                    onReset={handleResetTemplate}
                    onTest={handleTestTemplate}
                    isResetting={resetTemplateMutation.isPending}
                    isTesting={testTemplateMutation.isPending}
                  />
                ))}
              </div>
            ) : (
              <TemplateEditor
                selectedTemplate={selectedTemplate}
                editingTemplate={editingTemplate}
                isSaving={updateTemplateMutation.isPending}
                onSave={handleSaveTemplate}
                onCancel={() => {
                  setSelectedTemplate(null);
                  setEditingTemplate(null);
                }}
                onUpdate={setEditingTemplate}
              />
            )}
          </TabsContent>
        </Tabs>

        <ConfirmDialog
          open={isResetConfirmOpen}
          onOpenChange={setIsResetConfirmOpen}
          title="Reset Template"
          desc="Are you sure you want to reset this template to default? Any customizations will be lost."
          confirmText="Reset"
          destructive
          isLoading={resetTemplateMutation.isPending}
          handleConfirm={() => {
            if (resetTemplateType) {
              resetTemplateMutation.mutate(resetTemplateType, {
                onSuccess: () => {
                  setIsResetConfirmOpen(false);
                  setResetTemplateType(null);
                },
              });
            }
          }}
        />

        <PromptDialog
          open={isTestEmailPromptOpen}
          onOpenChange={setIsTestEmailPromptOpen}
          title="Send Test Email"
          description="Enter an email address to send a test email."
          placeholder="email@example.com"
          inputType="email"
          confirmText="Send Test"
          isLoading={
            testTemplateMutation.isPending || testSettingsMutation.isPending
          }
          validation={(value) => {
            if (!value) return "Email is required";
            if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value))
              return "Invalid email address";
            return null;
          }}
          onConfirm={(email) => {
            if (testTemplateType === "config") {
              testSettingsMutation.mutate(email, {
                onSuccess: () => {
                  setIsTestEmailPromptOpen(false);
                  setTestTemplateType(null);
                },
              });
            } else if (testTemplateType) {
              testTemplateMutation.mutate(
                { type: testTemplateType, email },
                {
                  onSuccess: () => {
                    setIsTestEmailPromptOpen(false);
                    setTestTemplateType(null);
                  },
                },
              );
            }
          }}
        />
      </div>
    </div>
  );
}
