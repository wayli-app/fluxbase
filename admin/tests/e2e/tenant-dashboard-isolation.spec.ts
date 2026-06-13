import { test, expect } from "./fixtures";

test.describe("Tenant Dashboard Isolation", () => {
  // Pages to smoke-test with a tenant selected.
  // Each test verifies no 500 API responses and no JS errors related to API failures.
  const pages = [
    { path: "jobs", name: "Jobs" },
    { path: "functions", name: "Functions" },
    { path: "secrets", name: "Secrets" },
    { path: "storage", name: "Storage" },
    { path: "knowledge-bases", name: "Knowledge Bases" },
    { path: "chatbots", name: "Chatbots" },
    { path: "settings", name: "Settings" },
    { path: "webhooks", name: "Webhooks" },
    { path: "rpc", name: "RPC" },
    { path: "logs", name: "Logs" },
    { path: "extensions", name: "Extensions" },
  ];

  for (const { path, name } of pages) {
    test(`${name} page loads without errors with tenant selected`, async ({
      adminPage,
    }) => {
      const apiErrors: string[] = [];
      adminPage.on("response", (response) => {
        if (response.status() >= 500) {
          apiErrors.push(`${response.status()} ${response.url()}`);
        }
      });

      await adminPage.goto(path, { waitUntil: "networkidle" });

      expect(
        apiErrors,
        `No 500 errors on ${name} page with tenant selected`,
      ).toEqual([]);
    });
  }

  test("all dashboard pages load without 500 errors", async ({ adminPage }) => {
    const allApiErrors: string[] = [];

    adminPage.on("response", (response) => {
      if (response.status() >= 500) {
        allApiErrors.push(`${response.status()} ${response.url()}`);
      }
    });

    for (const { path } of pages) {
      await adminPage.goto(path, { waitUntil: "networkidle" });
    }

    expect(allApiErrors).toEqual([]);
  });
});
