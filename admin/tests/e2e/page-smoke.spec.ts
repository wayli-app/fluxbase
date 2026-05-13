import { test, expect } from "./fixtures";

test.describe("Page smoke tests (instance admin)", () => {
  const pages = [
    { path: "/", name: "Dashboard" },
    { path: "tables", name: "Tables" },
    { path: "schema", name: "Schema Viewer" },
    { path: "sql-editor", name: "SQL Editor" },
    { path: "users", name: "Users" },
    { path: "authentication", name: "Authentication" },
    { path: "knowledge-bases", name: "Knowledge Bases" },
    { path: "chatbots", name: "AI Chatbots" },
    { path: "mcp-tools", name: "MCP Tools" },
    { path: "api/rest", name: "API Explorer" },
    { path: "realtime", name: "Realtime" },
    { path: "storage", name: "Storage" },
    { path: "functions", name: "Functions" },
    { path: "jobs", name: "Jobs" },
    { path: "rpc", name: "RPC" },
    { path: "email-settings", name: "Email Settings" },
    { path: "ai-providers", name: "AI Providers" },
    { path: "policies", name: "RLS Policies" },
    { path: "security-settings", name: "Security Settings" },
    { path: "secrets", name: "Secrets" },
    { path: "client-keys", name: "Client Keys" },
    { path: "service-keys", name: "Service Keys" },
    { path: "webhooks", name: "Webhooks" },
    { path: "logs", name: "Log Stream" },
    { path: "monitoring", name: "Monitoring" },
    { path: "tenants", name: "Tenants" },
    { path: "features", name: "Configuration" },
    { path: "extensions", name: "Extensions" },
    { path: "database-config", name: "Database Config" },
    { path: "instance-settings", name: "Instance Settings" },
    { path: "storage-config", name: "Storage Config" },
    { path: "settings", name: "Account Settings" },
    { path: "settings/appearance", name: "Appearance" },
  ];

  for (const { path, name } of pages) {
    test(`${name} page loads without errors`, async ({ adminPage }) => {
      const apiErrors: string[] = [];
      const consoleErrors: string[] = [];

      adminPage.on("response", (response) => {
        if (response.status() >= 500) {
          apiErrors.push(`${response.status()} ${response.url()}`);
        }
      });
      adminPage.on("console", (msg) => {
        if (msg.type() === "error") {
          consoleErrors.push(msg.text());
        }
      });

      await adminPage.goto(path, { waitUntil: "networkidle" });

      expect(
        apiErrors,
        `${name}: no 500 API errors`,
      ).toEqual([]);
      expect(
        consoleErrors.filter((e) => !e.includes("favicon") && !e.includes("404")),
        `${name}: no JS console errors`,
      ).toEqual([]);
    });
  }
});

test.describe("Page smoke tests (tenant admin)", () => {
  const tenantAdminPages = [
    { path: "/", name: "Dashboard" },
    { path: "tables", name: "Tables" },
    { path: "schema", name: "Schema Viewer" },
    { path: "sql-editor", name: "SQL Editor" },
    { path: "users", name: "Users" },
    { path: "authentication", name: "Authentication" },
    { path: "knowledge-bases", name: "Knowledge Bases" },
    { path: "chatbots", name: "AI Chatbots" },
    { path: "mcp-tools", name: "MCP Tools" },
    { path: "api/rest", name: "API Explorer" },
    { path: "realtime", name: "Realtime" },
    { path: "storage", name: "Storage" },
    { path: "functions", name: "Functions" },
    { path: "jobs", name: "Jobs" },
    { path: "rpc", name: "RPC" },
    { path: "email-settings", name: "Email Settings" },
    { path: "ai-providers", name: "AI Providers" },
    { path: "policies", name: "RLS Policies" },
    { path: "security-settings", name: "Security Settings" },
    { path: "secrets", name: "Secrets" },
    { path: "client-keys", name: "Client Keys" },
    { path: "service-keys", name: "Service Keys" },
    { path: "webhooks", name: "Webhooks" },
    { path: "logs", name: "Log Stream" },
    { path: "monitoring", name: "Monitoring" },
    { path: "extensions", name: "Extensions" },
    { path: "settings", name: "Account Settings" },
    { path: "settings/appearance", name: "Appearance" },
  ];

  const instanceOnlyPages = [
    { path: "tenants", name: "Tenants" },
    { path: "features", name: "Configuration" },
    { path: "database-config", name: "Database Config" },
    { path: "instance-settings", name: "Instance Settings" },
    { path: "storage-config", name: "Storage Config" },
  ];

  for (const { path, name } of tenantAdminPages) {
    test(`${name} page loads for tenant admin`, async ({
      tenantAdminPage,
    }) => {
      const apiErrors: string[] = [];

      tenantAdminPage.on("response", (response) => {
        if (response.status() >= 500) {
          apiErrors.push(`${response.status()} ${response.url()}`);
        }
      });

      await tenantAdminPage.goto(path, { waitUntil: "networkidle" });

      expect(
        apiErrors,
        `${name}: no 500 API errors for tenant admin`,
      ).toEqual([]);
    });
  }

  for (const { path, name } of instanceOnlyPages) {
    test(`${name} page is inaccessible for tenant admin`, async ({
      tenantAdminPage,
    }) => {
      await tenantAdminPage.goto(path);
      const url = tenantAdminPage.url();
      expect(
        url,
        `${name}: tenant admin should be redirected away`,
      ).not.toContain(path);
    });
  }
});
