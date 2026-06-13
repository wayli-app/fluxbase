import z from "zod";
import { createFileRoute, getRouteApi } from "@tanstack/react-router";
import { Key, Settings, Users, Building2, Shield } from "lucide-react";
import { PageHeader } from "@/components/layout/page-header";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  OAuthProvidersTab,
  SAMLProvidersTab,
  AuthSettingsTab,
  ActiveSessionsTab,
} from "@/components/authentication";
import { useTenantStore } from "@/stores/tenant-store";

const authenticationSearchSchema = z.object({
  tab: z.string().optional().catch("providers"),
});

const route = getRouteApi("/_authenticated/authentication/");

const AuthenticationPage = () => {
  const search = route.useSearch();
  const navigate = route.useNavigate();
  const currentTenant = useTenantStore((s) => s.currentTenant);
  const isInstanceLevel = !currentTenant?.id || currentTenant.is_default;

  return (
    <div className="flex h-full flex-col">
      <PageHeader
        icon={<Shield />}
        title="Authentication"
        description={
          isInstanceLevel
            ? "Configuring instance-level providers (available to all tenants)"
            : `Configuring providers for "${currentTenant.name}"`
        }
      />

      <div className="flex-1 overflow-auto p-6">
        <Tabs
          value={search.tab || "providers"}
          onValueChange={(tab) => navigate({ search: { tab } })}
          className="w-full"
        >
          <TabsList className="grid w-full grid-cols-4">
            <TabsTrigger value="providers">
              <Key className="mr-2 h-4 w-4" />
              OAuth Providers
            </TabsTrigger>
            <TabsTrigger value="saml">
              <Building2 className="mr-2 h-4 w-4" />
              SAML SSO
            </TabsTrigger>
            <TabsTrigger value="settings">
              <Settings className="mr-2 h-4 w-4" />
              Auth Settings
            </TabsTrigger>
            <TabsTrigger value="sessions">
              <Users className="mr-2 h-4 w-4" />
              Active Sessions
            </TabsTrigger>
          </TabsList>

          <TabsContent value="providers" className="space-y-4">
            <OAuthProvidersTab />
          </TabsContent>

          <TabsContent value="saml" className="space-y-4">
            <SAMLProvidersTab />
          </TabsContent>

          <TabsContent value="settings" className="space-y-4">
            <AuthSettingsTab />
          </TabsContent>

          <TabsContent value="sessions" className="space-y-4">
            <ActiveSessionsTab />
          </TabsContent>
        </Tabs>
      </div>
    </div>
  );
};

export const Route = createFileRoute("/_authenticated/authentication/")({
  validateSearch: authenticationSearchSchema,
  component: AuthenticationPage,
});
