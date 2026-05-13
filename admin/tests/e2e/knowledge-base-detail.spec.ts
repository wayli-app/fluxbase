import { test, expect } from "./fixtures";
import { rawCreateKnowledgeBase } from "./helpers/api";

test.describe("Knowledge Base detail pages", () => {
  test("detail sub-pages load for an existing knowledge base", async ({
    adminPage,
    adminToken,
  }) => {
    const kb = await rawCreateKnowledgeBase(adminToken, {
      name: "E2E KB Detail Test",
      description: "Test KB for page smoke",
    });
    const kbId = (kb.body as Record<string, unknown>).id as string;

    const subPages = [
      { path: `knowledge-bases/${kbId}`, name: "Overview" },
      { path: `knowledge-bases/${kbId}/tables`, name: "Tables" },
      { path: `knowledge-bases/${kbId}/graph`, name: "Graph" },
      { path: `knowledge-bases/${kbId}/search`, name: "Search" },
      { path: `knowledge-bases/${kbId}/settings`, name: "Settings" },
    ];

    for (const { path, name } of subPages) {
      const apiErrors: string[] = [];
      adminPage.on("response", (r) => {
        if (r.status() >= 500) apiErrors.push(`${r.status()} ${r.url()}`);
      });

      await adminPage.goto(path, { waitUntil: "networkidle" });

      expect(apiErrors, `${name}: no 500 errors`).toEqual([]);
      const url = adminPage.url();
      expect(url, `${name}: should stay on detail page`).toContain(kbId);
    }
  });
});
