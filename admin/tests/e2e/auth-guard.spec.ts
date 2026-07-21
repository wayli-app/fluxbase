import { test, expect } from "./fixtures";

test.describe("Auth Guard", () => {
  const protectedRoutes = [
    "",
    "tables",
    "schema",
    "sql-editor",
    "storage",
    "functions",
    "jobs",
    "users",
    "tenants",
    "settings",
    "extensions",
    "service-keys",
    "client-keys",
    "webhooks",
    "secrets",
    "logs",
    "monitoring",
    "rpc",
    "policies",
    "email-settings",
    "security-settings",
    "features",
    "instance-settings",
    "realtime",
    "chatbots",
    "mcp-tools",
  ];

  test("unauthenticated access to protected routes redirects to login", async ({
    page,
  }) => {
    // Navigate to login first to set a real origin and clear tokens
    await page.goto("login", { waitUntil: "networkidle" });

    // Clear any existing tokens (cookies + localStorage)
    await page.evaluate(() => {
      document.cookie =
        "fluxbase_admin_token=; path=/; max-age=0; SameSite=Lax";
      document.cookie =
        "fluxbase_admin_refresh_token=; path=/; max-age=0; SameSite=Lax";
      localStorage.removeItem("fluxbase_admin_user");
    });

    for (const route of protectedRoutes) {
      await page.goto(route || "./");

      // Should redirect to login page
      await expect(page).toHaveURL(/\/login/, { timeout: 5_000 });
    }
  });

  test("expired token treated as unauthenticated", async ({ page }) => {
    // Navigate to a real page first
    await page.goto("login", { waitUntil: "networkidle" });

    // Set an invalid/expired token in cookie (Zustand auth store reads cookies)
    await page.evaluate(() => {
      document.cookie = `${encodeURIComponent(JSON.stringify("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMDEwMjAxIn0ifX0.yyyyyyyy"))}; path=/; max-age=${60 * 60 * 24 * 7}; SameSite=Lax`;
    });

    // Navigate to a protected route
    await page.goto("./");

    // Should redirect to login (token is invalid)
    await expect(page).toHaveURL(/\/login/, { timeout: 5_000 });
  });

  test("navigation between routes preserves auth state", async ({
    adminPage,
  }) => {
    // Navigate explicitly to the admin dashboard and wait for it to settle.
    // The admin app may redirect to a sub-route (e.g., /admin/ai-providers)
    // so we only assert that we're somewhere under /admin/, not a specific path.
    await adminPage.goto("./", { waitUntil: "networkidle" });
    await expect(adminPage).toHaveURL(/\/admin/);
    await expect(adminPage).not.toHaveURL(/\/login/);

    // Navigate to various routes (relative to base)
    const routes = ["sql-editor", "storage", "functions", "jobs", "users"];
    for (const route of routes) {
      await adminPage.goto(route);
      // Should NOT redirect to login
      await expect(adminPage).not.toHaveURL(/\/login/);
    }

    // Token should still be present in cookie
    const token = await adminPage.evaluate(() => {
      const prefix = "fluxbase_admin_token=";
      const parts = document.cookie.split("; ");
      for (const part of parts) {
        if (part.startsWith(prefix)) {
          try {
            return JSON.parse(part.substring(prefix.length));
          } catch {
            return part.substring(prefix.length);
          }
        }
      }
      return null;
    });
    expect(token).toBeTruthy();
  });
});
