/* eslint-disable react-hooks/rules-of-hooks, no-empty-pattern */
import { test as base, expect, type Page } from "@playwright/test";
import { rawLogin, listTenants } from "./helpers/api";
import { getUserByEmail } from "./helpers/db";
import {
  ADMIN_EMAIL,
  ADMIN_PASSWORD,
  SECOND_TENANT_SLUG,
  TENANT_ADMIN_EMAIL,
  TENANT_ADMIN_PASSWORD,
} from "./helpers/constants";

const BASE_URL =
  process.env.PLAYWRIGHT_BASE_URL || "http://localhost:5050/admin/";

interface TenantInfo {
  id: string;
  name: string;
  slug: string;
  is_default: boolean;
}

interface Fixtures {
  adminPage: Page;
  adminToken: string;
  tenantAdminPage: Page;
  tenantAdminToken: string;
  tenantAdminInfo: {
    userId: string;
    email: string;
    tenantId: string;
    tenantName: string;
  };
  multipleTenants: TenantInfo[];
  defaultTenantId: string;
  thirdTenantId: string;
}

const THIRD_TENANT_SLUG = "e2e-third-tenant";

/**
 * Get an admin access token via direct API call (no browser needed).
 */
async function getAdminToken(): Promise<string> {
  const result = await rawLogin({
    email: ADMIN_EMAIL,
    password: ADMIN_PASSWORD,
  });
  if (result.status !== 200 || !result.body?.access_token) {
    throw new Error(
      `Failed to get admin token: ${result.status} ${JSON.stringify(result.body)}`,
    );
  }
  return result.body.access_token;
}

/**
 * Login a user via the browser and return the page.
 */
async function browserLogin(
  page: Page,
  email: string,
  password: string,
): Promise<Page> {
  await page.goto("login");
  await expect(page.locator("#email")).toBeVisible({ timeout: 15_000 });

  await page.fill("#email", email);
  await page.fill("#password", password);
  await page.click('button[type="submit"]');

  // Wait for dashboard to load (increased timeout for slow server under load)
  await expect(page).toHaveURL(/\/admin\/?$/, { timeout: 60_000 });

  const token = await page.evaluate(() => {
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

  // Select the default tenant so tenant-scoped pages work.
  // Only instance admins see the tenant selector — tenant admins don't.
  // When no tenant is selected, the button shows "Instance".
  try {
    const tenantCombo = page.getByRole("combobox", { name: /select tenant/i });
    if (await tenantCombo.isVisible({ timeout: 500 }).catch(() => false)) {
      const comboText = await tenantCombo.innerText().catch(() => "");
      if (comboText.includes("Instance")) {
        await tenantCombo.click();
        const listbox = page.getByRole("listbox");
        await expect(listbox).toBeVisible({ timeout: 2_000 });
        const options = page.getByRole("option");
        const firstTenant = options.nth(1);
        if (await firstTenant.isVisible().catch(() => false)) {
          await firstTenant.click();
          await page.waitForTimeout(200);
        }
      }
    }
  } catch {
    // Tenant selection is optional — some roles or pages may not need it
  }

  return page;
}

/**
 * Start impersonation via the API and set tokens in localStorage.
 */
async function _setupImpersonation(
  page: Page,
  _accessToken: string,
  impersonationToken: string,
  type: "user" | "anon" | "service",
  targetUser?: { id: string; email: string },
) {
  // Store impersonation tokens
  await page.evaluate(
    ({ token, type, user }) => {
      localStorage.setItem("fluxbase_impersonation_token", token);
      localStorage.setItem("fluxbase_impersonation_type", type);
      if (user) {
        localStorage.setItem(
          "fluxbase_impersonated_user",
          JSON.stringify({ id: user.id, email: user.email }),
        );
      }
      localStorage.setItem(
        "fluxbase_impersonation_session",
        JSON.stringify({
          id: "test-session",
          admin_user_id: "admin",
          impersonation_type: type,
          reason: "E2E test",
          started_at: new Date().toISOString(),
          is_active: true,
          ...(user ? { target_user_id: user.id } : {}),
        }),
      );
    },
    { token: impersonationToken, type, user: targetUser },
  );
}

export const test = base.extend<Fixtures>({
  // Worker-scoped: logs in once per worker, reused across all tests in that worker.
  // With workers:2, at most 2 concurrent browser logins happen (one per worker),
  // eliminating the race condition where two first-tests log in simultaneously.
  adminPage: [
    async ({ browser }, use) => {
      const context = await browser.newContext();
      const page = await context.newPage();
      await browserLogin(page, ADMIN_EMAIL, ADMIN_PASSWORD);
      await use(page);
      await context.close();
    },
    { scope: "worker" },
  ],

  adminToken: [
    async ({}, use) => {
      const token = await getAdminToken();
      await use(token);
    },
    { scope: "worker" },
  ],

  /**
   * Provides a tenant admin page (logged in as tenant admin).
   * Worker-scoped to avoid repeated browser logins that cause race conditions
   * under parallel execution. Requires global setup to have created the
   * second tenant + tenant admin user.
   */
  tenantAdminPage: [
    async ({ browser }, use) => {
      const context = await browser.newContext();
      const page = await context.newPage();
      await browserLogin(page, TENANT_ADMIN_EMAIL, TENANT_ADMIN_PASSWORD);
      await use(page);
      await context.close();
    },
    { scope: "worker" },
  ],

  tenantAdminToken: async ({ tenantAdminPage }, use) => {
    const token = await tenantAdminPage.evaluate(() => {
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
    await use(token || "");
  },

  tenantAdminInfo: async ({ adminToken }, use) => {
    const user = await getUserByEmail(TENANT_ADMIN_EMAIL);
    if (!user) {
      throw new Error(
        "Tenant admin user not found. Global setup must run first.",
      );
    }

    const tenantsResult = await listTenants(adminToken);
    const secondTenant = (tenantsResult.body as TenantInfo[])?.find(
      (t) => t.slug === SECOND_TENANT_SLUG,
    );
    if (!secondTenant) {
      throw new Error("Second tenant not found. Global setup must run first.");
    }

    await use({
      userId: user.id,
      email: TENANT_ADMIN_EMAIL,
      tenantId: secondTenant.id,
      tenantName: secondTenant.name,
    });
  },

  /**
   * Provides a list of all tenants (ensures at least 2 exist).
   */
  multipleTenants: async ({ adminToken }, use) => {
    const tenantsResult = await listTenants(adminToken);
    const tenants = (tenantsResult.body as TenantInfo[]) || [];

    if (tenants.length < 2) {
      throw new Error(
        "Expected at least 2 tenants. Global setup must run first.",
      );
    }

    await use(tenants);
  },

  /**
   * Provides the default tenant ID.
   */
  defaultTenantId: async ({ adminToken }, use) => {
    const tenantsResult = await listTenants(adminToken);
    const defaultTenant = (tenantsResult.body as TenantInfo[])?.find(
      (t) => t.is_default === true,
    );
    if (!defaultTenant) {
      throw new Error("Default tenant not found.");
    }
    await use(defaultTenant.id);
  },

  /**
   * Provides the third tenant ID (for isolation tests).
   */
  thirdTenantId: async ({ adminToken }, use) => {
    const tenantsResult = await listTenants(adminToken);
    const thirdTenant = (tenantsResult.body as TenantInfo[])?.find(
      (t) => t.slug === THIRD_TENANT_SLUG,
    );
    if (!thirdTenant) {
      throw new Error("Third tenant not found. Global setup must run first.");
    }
    await use(thirdTenant.id);
  },
});

export { expect, BASE_URL };
export type { TenantInfo };
