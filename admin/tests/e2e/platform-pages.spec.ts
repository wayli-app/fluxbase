import { test, expect } from "./fixtures";

test.describe("Configuration / Features page", () => {
  test("loads and shows feature toggles", async ({ adminPage }) => {
    await adminPage.goto("features", { waitUntil: "networkidle" });
    await expect(
      adminPage.getByText(/config|feature|setting/i).first(),
    ).toBeVisible({ timeout: 10_000 });
  });
});

test.describe("Instance Settings page", () => {
  test("loads and shows settings form", async ({ adminPage }) => {
    await adminPage.goto("instance-settings", { waitUntil: "networkidle" });
    await expect(
      adminPage.getByText(/instance|setting/i).first(),
    ).toBeVisible({ timeout: 10_000 });
  });
});

test.describe("Database Config page", () => {
  test("loads and shows database settings", async ({ adminPage }) => {
    await adminPage.goto("database-config", { waitUntil: "networkidle" });
    await expect(
      adminPage.getByText(/database|config|setting/i).first(),
    ).toBeVisible({ timeout: 10_000 });
  });
});

test.describe("Storage Config page", () => {
  test("loads and shows storage settings", async ({ adminPage }) => {
    await adminPage.goto("storage-config", { waitUntil: "networkidle" });
    await expect(
      adminPage.getByText(/storage|config|provider/i).first(),
    ).toBeVisible({ timeout: 10_000 });
  });
});

test.describe("Monitoring page", () => {
  test("loads and shows monitoring dashboard", async ({ adminPage }) => {
    await adminPage.goto("monitoring", { waitUntil: "networkidle" });
    await expect(
      adminPage.getByText(/monitor|metric|health/i).first(),
    ).toBeVisible({ timeout: 10_000 });
  });
});
