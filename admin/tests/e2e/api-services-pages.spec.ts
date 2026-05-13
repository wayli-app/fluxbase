import { test, expect } from "./fixtures";

test.describe("API Explorer page", () => {
  test("loads and shows API documentation", async ({ adminPage }) => {
    await adminPage.goto("api/rest", { waitUntil: "networkidle" });
    await expect(
      adminPage.getByText(/api|endpoint|rest|swagger|openapi/i).first(),
    ).toBeVisible({ timeout: 10_000 });
  });
});

test.describe("Realtime page", () => {
  test("loads and shows realtime channels", async ({ adminPage }) => {
    await adminPage.goto("realtime", { waitUntil: "networkidle" });
    await expect(
      adminPage.getByText(/realtime|channel|websocket|subscribe/i).first(),
    ).toBeVisible({ timeout: 10_000 });
  });
});

test.describe("Email Settings page", () => {
  test("loads and shows email provider config", async ({ adminPage }) => {
    await adminPage.goto("email-settings", { waitUntil: "networkidle" });
    await expect(
      adminPage.getByText(/email|smtp|provider|sendgrid/i).first(),
    ).toBeVisible({ timeout: 10_000 });
  });
});

test.describe("AI Providers page", () => {
  test("loads and shows provider list", async ({ adminPage }) => {
    await adminPage.goto("ai-providers", { waitUntil: "networkidle" });
    await expect(
      adminPage.getByText(/provider|openai|anthropic|ai/i).first(),
    ).toBeVisible({ timeout: 10_000 });
  });
});
