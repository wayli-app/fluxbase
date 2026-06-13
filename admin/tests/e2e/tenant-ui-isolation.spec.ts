import { test, expect } from "./fixtures";
import {
  rawCreateFunction,
  rawCreateSecret,
  rawCreateBucket,
  rawCreateKnowledgeBase,
  rawCreateWebhook,
  rawApiRequest,
  listTenants,
} from "./helpers/api";
import { selectTenant } from "./helpers/selectors";
import { ADMIN_EMAIL, ADMIN_PASSWORD } from "./helpers/constants";

test.describe("Tenant UI Data Isolation", () => {
  // Cleanup tracking
  const createdResources: Array<{
    type: string;
    id: string;
    tenantId: string;
  }> = [];

  let adminToken: string;
  let tenantAId: string;
  let tenantBId: string;
  let tenantAName: string;
  let tenantBName: string;

  // Resource names (set in beforeAll)
  let fnA: string;
  let fnB: string;
  let secretA: string;
  let secretB: string;
  let bucketA: string;
  let bucketB: string;
  let webhookA: string;
  let webhookB: string;
  let kbA: string;
  let kbB: string;

  const fnCode = `
export default function handler(req: Request): Response {
  return new Response(JSON.stringify({ message: "ui-iso-test" }), {
    headers: { "Content-Type": "application/json" },
  });
}
`;

  test.beforeAll(async () => {
    // Login and resolve tenant info
    const { rawLogin } = await import("./helpers/api");
    const loginResult = await rawLogin({
      email: ADMIN_EMAIL,
      password: ADMIN_PASSWORD,
    });
    adminToken = loginResult.body?.access_token;
    if (!adminToken) throw new Error("Failed to get admin token");

    const tenantsResult = await listTenants(adminToken);
    const tenants = (tenantsResult.body || []) as Array<{
      id: string;
      name: string;
      slug: string;
      is_default: boolean;
    }>;

    const defaultTenant = tenants.find((t) => t.is_default);
    const thirdTenant = tenants.find((t) => t.slug === "e2e-third-tenant");
    if (!defaultTenant || !thirdTenant) {
      throw new Error("Default and third tenants must exist");
    }

    tenantAId = defaultTenant.id;
    tenantBId = thirdTenant.id;
    tenantAName = defaultTenant.name;
    tenantBName = thirdTenant.name;

    const ts = Date.now();

    // Create resources in tenant A (default)
    fnA = `ui-iso-fn-A-${ts}`;
    const fnARes = await rawCreateFunction(
      { name: fnA, code: fnCode, verifyJWT: false },
      adminToken,
      tenantAId,
    );
    if ([200, 201].includes(fnARes.status)) {
      createdResources.push({
        type: "function",
        id: fnA,
        tenantId: tenantAId,
      });
    }

    secretA = `ui-iso-secret-A-${ts}`;
    const secretARes = await rawCreateSecret(
      { name: secretA, value: "value-a" },
      adminToken,
      tenantAId,
    );
    if ([200, 201].includes(secretARes.status) && secretARes.body?.id) {
      createdResources.push({
        type: "secret",
        id: secretARes.body.id,
        tenantId: tenantAId,
      });
    }

    bucketA = `ui-iso-bucket-a-${ts}`;
    const bucketARes = await rawCreateBucket(bucketA, adminToken, tenantAId);
    if ([200, 201].includes(bucketARes.status)) {
      createdResources.push({
        type: "bucket",
        id: bucketA,
        tenantId: tenantAId,
      });
    }

    webhookA = `ui-iso-wh-A-${ts}`;
    const whARes = await rawCreateWebhook(
      {
        name: webhookA,
        url: "https://example.com/a",
        events: [{ table: "public.test_ui_iso", operations: ["INSERT"] }],
      },
      adminToken,
      tenantAId,
    );
    if ([200, 201].includes(whARes.status) && whARes.body?.id) {
      createdResources.push({
        type: "webhook",
        id: whARes.body.id,
        tenantId: tenantAId,
      });
    }

    kbA = `ui-iso-kb-A-${ts}`;
    const kbARes = await rawCreateKnowledgeBase(
      { name: kbA, description: "UI isolation test KB A" },
      adminToken,
      tenantAId,
    );
    if ([200, 201].includes(kbARes.status) && kbARes.body?.id) {
      createdResources.push({
        type: "knowledge_base",
        id: kbARes.body.id,
        tenantId: tenantAId,
      });
    }

    // Create resources in tenant B (third)
    fnB = `ui-iso-fn-B-${ts}`;
    const fnBRes = await rawCreateFunction(
      { name: fnB, code: fnCode, verifyJWT: false },
      adminToken,
      tenantBId,
    );
    if ([200, 201].includes(fnBRes.status)) {
      createdResources.push({
        type: "function",
        id: fnB,
        tenantId: tenantBId,
      });
    }

    secretB = `ui-iso-secret-B-${ts}`;
    const secretBRes = await rawCreateSecret(
      { name: secretB, value: "value-b" },
      adminToken,
      tenantBId,
    );
    if ([200, 201].includes(secretBRes.status) && secretBRes.body?.id) {
      createdResources.push({
        type: "secret",
        id: secretBRes.body.id,
        tenantId: tenantBId,
      });
    }

    bucketB = `ui-iso-bucket-b-${ts}`;
    const bucketBRes = await rawCreateBucket(bucketB, adminToken, tenantBId);
    if ([200, 201].includes(bucketBRes.status)) {
      createdResources.push({
        type: "bucket",
        id: bucketB,
        tenantId: tenantBId,
      });
    }

    webhookB = `ui-iso-wh-B-${ts}`;
    const whBRes = await rawCreateWebhook(
      {
        name: webhookB,
        url: "https://example.com/b",
        events: [{ table: "public.test_ui_iso", operations: ["INSERT"] }],
      },
      adminToken,
      tenantBId,
    );
    if ([200, 201].includes(whBRes.status) && whBRes.body?.id) {
      createdResources.push({
        type: "webhook",
        id: whBRes.body.id,
        tenantId: tenantBId,
      });
    }

    kbB = `ui-iso-kb-B-${ts}`;
    const kbBRes = await rawCreateKnowledgeBase(
      { name: kbB, description: "UI isolation test KB B" },
      adminToken,
      tenantBId,
    );
    if ([200, 201].includes(kbBRes.status) && kbBRes.body?.id) {
      createdResources.push({
        type: "knowledge_base",
        id: kbBRes.body.id,
        tenantId: tenantBId,
      });
    }
  });

  test.afterAll(async () => {
    const { rawLogin } = await import("./helpers/api");
    const loginResult = await rawLogin({
      email: ADMIN_EMAIL,
      password: ADMIN_PASSWORD,
    });
    const token = loginResult.body?.access_token;
    if (!token) return;

    for (const { type, id, tenantId } of createdResources) {
      const headers: Record<string, string> = {
        Authorization: `Bearer ${token}`,
        "X-FB-Tenant": tenantId,
      };
      let path: string;
      switch (type) {
        case "function":
          path = `/api/v1/functions/${id}`;
          break;
        case "secret":
          path = `/api/v1/secrets/${id}`;
          break;
        case "bucket":
          path = `/api/v1/storage/buckets/${id}`;
          break;
        case "webhook":
          path = `/api/v1/webhooks/${id}`;
          break;
        case "knowledge_base":
          path = `/api/v1/ai/knowledge-bases/${id}`;
          break;
        default:
          continue;
      }
      await rawApiRequest({ method: "DELETE", path, headers }).catch(() => {});
    }
  });

  async function switchTenantAndWait(
    page: import("@playwright/test").Page,
    tenantName: string,
    tenantId: string,
  ) {
    await selectTenant(page, tenantName);
    // Wait for an API request with the new tenant header
    await page
      .waitForRequest((req) => req.headers()["x-fb-tenant"] === tenantId, {
        timeout: 10_000,
      })
      .catch(() => {});
    await page.waitForTimeout(500);
  }

  // ────────────────────────────────────────────────────────────────
  // Functions
  // ────────────────────────────────────────────────────────────────

  test("functions page shows tenant-scoped data", async ({ adminPage }) => {
    // Select tenant A
    await adminPage.goto("./", { waitUntil: "networkidle" });
    await switchTenantAndWait(adminPage, tenantAName, tenantAId);

    // Navigate to functions, click the Functions tab
    await adminPage.goto("functions", { waitUntil: "networkidle" });
    const functionsTab = adminPage.getByRole("tab", { name: "Functions" });
    if (await functionsTab.isVisible()) {
      await functionsTab.click();
      await adminPage.waitForTimeout(500);
    }

    // Tenant A's function should be visible, B's should not
    await expect(adminPage.getByText(fnA)).toBeVisible({ timeout: 10_000 });
    await expect(adminPage.getByText(fnB)).not.toBeVisible();

    // Switch to tenant B
    await switchTenantAndWait(adminPage, tenantBName, tenantBId);
    await adminPage.goto("functions", { waitUntil: "networkidle" });
    if (await functionsTab.isVisible()) {
      await functionsTab.click();
      await adminPage.waitForTimeout(500);
    }

    // Tenant B's function should be visible, A's should not
    await expect(adminPage.getByText(fnB)).toBeVisible({ timeout: 10_000 });
    await expect(adminPage.getByText(fnA)).not.toBeVisible();
  });

  // ────────────────────────────────────────────────────────────────
  // Secrets
  // ────────────────────────────────────────────────────────────────

  test("secrets page shows tenant-scoped data", async ({ adminPage }) => {
    // Select tenant A
    await adminPage.goto("./", { waitUntil: "networkidle" });
    await switchTenantAndWait(adminPage, tenantAName, tenantAId);

    await adminPage.goto("secrets", { waitUntil: "networkidle" });

    // Tenant A's secret should be visible, B's should not
    await expect(adminPage.getByText(secretA)).toBeVisible({
      timeout: 10_000,
    });
    await expect(adminPage.getByText(secretB)).not.toBeVisible();

    // Switch to tenant B
    await switchTenantAndWait(adminPage, tenantBName, tenantBId);
    await adminPage.goto("secrets", { waitUntil: "networkidle" });

    await expect(adminPage.getByText(secretB)).toBeVisible({
      timeout: 10_000,
    });
    await expect(adminPage.getByText(secretA)).not.toBeVisible();
  });

  // ────────────────────────────────────────────────────────────────
  // Storage
  // ────────────────────────────────────────────────────────────────

  test("storage page shows tenant-scoped buckets", async ({ adminPage }) => {
    // Select tenant A
    await adminPage.goto("./", { waitUntil: "networkidle" });
    await switchTenantAndWait(adminPage, tenantAName, tenantAId);

    await adminPage.goto("storage", { waitUntil: "networkidle" });

    // Tenant A's bucket should be visible, B's should not
    await expect(adminPage.getByText(bucketA)).toBeVisible({
      timeout: 10_000,
    });
    await expect(adminPage.getByText(bucketB)).not.toBeVisible();

    // Switch to tenant B
    await switchTenantAndWait(adminPage, tenantBName, tenantBId);
    await adminPage.goto("storage", { waitUntil: "networkidle" });

    await expect(adminPage.getByText(bucketB)).toBeVisible({
      timeout: 10_000,
    });
    await expect(adminPage.getByText(bucketA)).not.toBeVisible();
  });

  // ────────────────────────────────────────────────────────────────
  // Webhooks
  // ────────────────────────────────────────────────────────────────

  test("webhooks page shows tenant-scoped data", async ({ adminPage }) => {
    // Select tenant A
    await adminPage.goto("./", { waitUntil: "networkidle" });
    await switchTenantAndWait(adminPage, tenantAName, tenantAId);

    await adminPage.goto("webhooks", { waitUntil: "networkidle" });

    // Tenant A's webhook should be visible, B's should not
    await expect(adminPage.getByText(webhookA)).toBeVisible({
      timeout: 10_000,
    });
    await expect(adminPage.getByText(webhookB)).not.toBeVisible();

    // Switch to tenant B
    await switchTenantAndWait(adminPage, tenantBName, tenantBId);
    await adminPage.goto("webhooks", { waitUntil: "networkidle" });

    await expect(adminPage.getByText(webhookB)).toBeVisible({
      timeout: 10_000,
    });
    await expect(adminPage.getByText(webhookA)).not.toBeVisible();
  });

  // ────────────────────────────────────────────────────────────────
  // Knowledge Bases
  // ────────────────────────────────────────────────────────────────

  test("knowledge bases page shows tenant-scoped data", async ({
    adminPage,
  }) => {
    // Select tenant A
    await adminPage.goto("./", { waitUntil: "networkidle" });
    await switchTenantAndWait(adminPage, tenantAName, tenantAId);

    await adminPage.goto("knowledge-bases", {
      waitUntil: "load",
    });

    // Tenant A's KB should be visible, B's should not
    await expect(adminPage.getByText(kbA)).toBeVisible({ timeout: 10_000 });
    await expect(adminPage.getByText(kbB)).not.toBeVisible();

    // Switch to tenant B
    await switchTenantAndWait(adminPage, tenantBName, tenantBId);
    await adminPage.goto("knowledge-bases", {
      waitUntil: "load",
    });

    await expect(adminPage.getByText(kbB)).toBeVisible({ timeout: 10_000 });
    await expect(adminPage.getByText(kbA)).not.toBeVisible();
  });

  // ────────────────────────────────────────────────────────────────
  // Cross-page tenant switch
  // ────────────────────────────────────────────────────────────────

  test("switching tenants mid-page refreshes data", async ({ adminPage }) => {
    // Start on secrets page with tenant A
    await adminPage.goto("./", { waitUntil: "networkidle" });
    await switchTenantAndWait(adminPage, tenantAName, tenantAId);

    await adminPage.goto("secrets", { waitUntil: "networkidle" });
    await expect(adminPage.getByText(secretA)).toBeVisible({
      timeout: 10_000,
    });

    // Switch to tenant B while on the secrets page
    await switchTenantAndWait(adminPage, tenantBName, tenantBId);

    // Wait for the page to refresh with new tenant data
    await expect(adminPage.getByText(secretB)).toBeVisible({
      timeout: 10_000,
    });
    await expect(adminPage.getByText(secretA)).not.toBeVisible();
  });
});
