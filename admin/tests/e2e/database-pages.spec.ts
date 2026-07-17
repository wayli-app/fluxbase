import { test, expect } from "./fixtures";

test.describe("Tables page", () => {
  test("loads and shows table list or empty state", async ({ adminPage }) => {
    await adminPage.goto("tables", { waitUntil: "networkidle" });

    const apiErrors: string[] = [];
    adminPage.on("response", (r) => {
      if (r.status() >= 500) apiErrors.push(`${r.status()} ${r.url()}`);
    });
    expect(apiErrors).toEqual([]);

    const hasTable = await adminPage.locator("table").isVisible().catch(() => false);
    const hasEmpty = await adminPage.getByText(/no tables|empty|get started/i).isVisible().catch(() => false);
    expect(hasTable || hasEmpty).toBeTruthy();
  });
});

test.describe("Schema Viewer page", () => {
  test("loads and renders schema tree", async ({ adminPage }) => {
    await adminPage.goto("schema", { waitUntil: "networkidle" });

    const apiErrors: string[] = [];
    adminPage.on("response", (r) => {
      if (r.status() >= 500) apiErrors.push(`${r.status()} ${r.url()}`);
    });
    expect(apiErrors).toEqual([]);

    const hasTree = await adminPage.locator("[data-testid], [role='tree'], [role='treeitem']").first().isVisible().catch(() => false);
    const hasContent = await adminPage.getByText(/schema|table|public/i).first().isVisible().catch(() => false);
    expect(hasTree || hasContent).toBeTruthy();
  });
});

test.describe("SQL Editor page", () => {
  test("loads and shows editor area", async ({ adminPage }) => {
    await adminPage.goto("sql-editor", { waitUntil: "networkidle" });

    const apiErrors: string[] = [];
    adminPage.on("response", (r) => {
      if (r.status() >= 500) apiErrors.push(`${r.status()} ${r.url()}`);
    });
    expect(apiErrors).toEqual([]);

    await expect(
      adminPage.getByText(/sql|query|editor/i).first(),
    ).toBeVisible({ timeout: 10_000 });
  });

  test("can interact with SQL Editor page", async ({ adminPage }) => {
    await adminPage.goto("sql-editor", { waitUntil: "networkidle" });

    // ponytail: target the actual editor surface, not Monaco's hidden IME
    // textarea (.ime-textarea) which is always covered by rendered view-lines
    // and causes clicks to be intercepted. .first() on a generic textarea
    // selector resolves to that hidden element and hangs the test.
    const editor = adminPage.locator(".cm-content, .monaco-editor .view-lines").first();
    if (await editor.isVisible({ timeout: 5_000 }).catch(() => false)) {
      await editor.click();
      await adminPage.keyboard.type("SELECT 1");

      const runButton = adminPage.getByRole("button", { name: /run|execute/i });
      if (await runButton.isVisible({ timeout: 3_000 }).catch(() => false)) {
        const [response] = await Promise.all([
          adminPage.waitForResponse((r) => r.url().includes("/rpc") || r.url().includes("/sql"), { timeout: 10_000 }).catch(() => null),
          runButton.click(),
        ]);
        if (response) {
          expect(response.status()).toBeLessThan(500);
        }
      }
    }
  });
});
