import { test, expect } from "./fixtures";

test.describe("Account Settings page", () => {
  test("loads and shows profile form", async ({ adminPage }) => {
    await adminPage.goto("settings", { waitUntil: "networkidle" });
    await expect(
      adminPage.getByText(/account|profile|setting/i).first(),
    ).toBeVisible({ timeout: 10_000 });
  });
});

test.describe("Appearance Settings page", () => {
  test("loads and shows theme options", async ({ adminPage }) => {
    await adminPage.goto("settings/appearance", { waitUntil: "networkidle" });
    await expect(
      adminPage.getByText(/appearance|theme|dark|light/i).first(),
    ).toBeVisible({ timeout: 10_000 });
  });
});
