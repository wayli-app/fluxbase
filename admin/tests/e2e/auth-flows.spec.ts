import { test, expect } from "./fixtures";

test.describe("Auth flow pages", () => {
  test("forgot password page renders", async ({ page }) => {
    await page.goto("forgot-password", { waitUntil: "networkidle" });
    await expect(page.locator("form")).toBeVisible();
    await expect(page.getByText(/forgot.*password|reset.*password/i).first()).toBeVisible();
  });

  test("reset password page renders without token", async ({ page }) => {
    await page.goto("reset-password", { waitUntil: "networkidle" });
    const url = page.url();
    expect(url).toContain("reset-password");
  });

  test("OTP login page renders", async ({ page }) => {
    await page.goto("login/otp", { waitUntil: "networkidle" });
    const url = page.url();
    expect(url).toContain("login");
  });

  const errorPages = [
    { path: "401", status: 401, label: "Unauthorized" },
    { path: "403", status: 403, label: "Forbidden" },
    { path: "404", status: 404, label: "Not Found" },
    { path: "500", status: 500, label: "Internal Server Error" },
    { path: "503", status: 503, label: "Service Unavailable" },
  ];

  for (const { path, status } of errorPages) {
    test(`${status} error page renders`, async ({ adminPage }) => {
      await adminPage.goto(path);
      await expect(
        adminPage.getByText(new RegExp(String(status))),
      ).toBeVisible({ timeout: 10_000 });
    });
  }
});
