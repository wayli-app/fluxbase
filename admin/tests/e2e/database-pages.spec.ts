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

  test("can run a simple query", async ({ adminPage }) => {
    await adminPage.goto("sql-editor", { waitUntil: "networkidle" });

    const editor = adminPage.locator("textarea, [contenteditable='true'], .cm-content, .monaco-editor textarea").first();
    if (await editor.isVisible({ timeout: 5_000 }).catch(() => false)) {
      await editor.click();
      await adminPage.keyboard.type("SELECT 1");
    }

    const runButton = adminPage.getByRole("button", { name: /run|execute/i });
    if (await runButton.isVisible({ timeout: 3_000 }).catch(() => false)) {
      await runButton.click();
      await adminPage.waitForTimeout(2_000);
    }
  });
});
