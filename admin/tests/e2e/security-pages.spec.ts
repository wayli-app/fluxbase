import { test, expect } from "./fixtures";

test.describe("Security Settings page", () => {
  test("loads and shows settings form", async ({ adminPage }) => {
    await adminPage.goto("security-settings", { waitUntil: "networkidle" });
    await expect(
      adminPage.getByText(/security|settings/i).first(),
    ).toBeVisible({ timeout: 10_000 });
  });
});

test.describe("Client Keys page", () => {
  test("loads and renders page content", async ({ adminPage }) => {
    await adminPage.goto("client-keys", { waitUntil: "networkidle" });
    await expect(
      adminPage.getByText(/client.*key|key/i).first(),
    ).toBeVisible({ timeout: 10_000 });
  });
});

test.describe("RLS Policies page", () => {
  test("loads and shows policies or empty state", async ({ adminPage }) => {
    await adminPage.goto("policies", { waitUntil: "networkidle" });
    await expect(
      adminPage.getByText(/polic|table|schema/i).first(),
    ).toBeVisible({ timeout: 10_000 });
  });
});
