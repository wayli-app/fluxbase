import { test, expect } from "./fixtures";
import {
  rawLogin,
  rawCreateOAuthProvider,
  rawDeleteOAuthProvider,
  rawListOAuthProviders,
} from "./helpers/api";
import { ADMIN_EMAIL, ADMIN_PASSWORD } from "./helpers/constants";

const TEST_PROVIDER = {
  provider_name: "test_sso",
  display_name: "Test SSO Provider",
  client_id: "test-client-id",
  client_secret: "test-client-secret",
  redirect_url:
    "http://localhost:8080/dashboard/auth/sso/oauth/test_sso/callback",
  scopes: ["openid", "email", "profile"],
  is_custom: true,
  authorization_url: "http://localhost:5556/dex/auth",
  token_url: "http://localhost:5556/dex/token",
  user_info_url: "http://localhost:5556/dex/userinfo",
};

async function getAdminToken() {
  const resp = await rawLogin({
    email: ADMIN_EMAIL,
    password: ADMIN_PASSWORD,
  });
  expect(resp.status).toBe(200);
  return resp.body.access_token;
}

async function cleanupTestProviders(accessToken: string) {
  const list = await rawListOAuthProviders(accessToken);
  if (list.body?.providers) {
    for (const p of list.body.providers) {
      if (p.provider_name === TEST_PROVIDER.provider_name) {
        await rawDeleteOAuthProvider(p.id, accessToken);
      }
    }
  }
}

test.describe("SSO Login", () => {
  test("no SSO buttons when no providers configured for dashboard login", async ({
    page,
  }) => {
    await page.goto("login", { waitUntil: "networkidle" });
    await expect(page.locator("#email")).toBeVisible({ timeout: 15_000 });

    // No "Or continue with" separator
    await expect(page.getByText("Or continue with")).not.toBeVisible();
  });

  test("SSO button appears when OAuth provider has allow_dashboard_login", async ({
    page,
  }) => {
    const token = await getAdminToken();
    await cleanupTestProviders(token);

    const resp = await rawCreateOAuthProvider(
      { ...TEST_PROVIDER, allow_dashboard_login: true },
      token,
    );
    expect(resp.status).toBe(201);
    const providerId = resp.body.id;

    try {
      await page.goto("login", { waitUntil: "networkidle" });
      await expect(page.locator("#email")).toBeVisible({ timeout: 15_000 });

      // SSO button should be visible
      const ssoButton = page.getByRole("button", {
        name: /Test SSO Provider/i,
      });
      await expect(ssoButton).toBeVisible({ timeout: 5_000 });

      // "Or continue with" separator should be visible
      await expect(page.getByText("Or continue with")).toBeVisible();
    } finally {
      await rawDeleteOAuthProvider(providerId, token);
    }
  });

  test("SSO button initiates authorize redirect", async ({ page }) => {
    const token = await getAdminToken();
    await cleanupTestProviders(token);

    const resp = await rawCreateOAuthProvider(
      { ...TEST_PROVIDER, allow_dashboard_login: true },
      token,
    );
    expect(resp.status).toBe(201);
    const providerId = resp.body.id;

    try {
      // Intercept the authorize request to prevent actual navigation to Dex
      let capturedUrl = "";
      await page.route(
        "**/dashboard/auth/sso/oauth/test_sso**",
        async (route) => {
          capturedUrl = route.request().url();
          await route.fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify({
              url: "http://localhost:5556/dex/auth?mocked=true",
            }),
          });
        },
      );

      // Intercept the navigation to the mocked Dex URL
      let navigatedToDex = false;
      await page.route("**/dex/auth**", async (route) => {
        navigatedToDex = true;
        // Return a simple page to avoid actual Dex interaction
        await route.fulfill({
          status: 200,
          contentType: "text/html",
          body: "<html><body>Mocked OAuth Provider</body></html>",
        });
      });

      await page.goto("login", { waitUntil: "networkidle" });
      await expect(page.locator("#email")).toBeVisible({ timeout: 15_000 });

      const ssoButton = page.getByRole("button", {
        name: /Test SSO Provider/i,
      });
      await expect(ssoButton).toBeVisible({ timeout: 5_000 });

      await ssoButton.click();

      // Verify authorize request was made with correct parameters
      await page.waitForTimeout(2000);
      expect(capturedUrl).toContain("test_sso");
      expect(capturedUrl).toContain("redirect_to=");
      expect(navigatedToDex).toBe(true);
    } finally {
      await rawDeleteOAuthProvider(providerId, token);
    }
  });

  test("callback page processes tokens and stores them", async ({ page }) => {
    // Get real tokens via API login
    const loginResult = await rawLogin({
      email: ADMIN_EMAIL,
      password: ADMIN_PASSWORD,
    });
    expect(loginResult.status).toBe(200);
    const accessToken = loginResult.body.access_token;
    const refreshToken = loginResult.body.refresh_token;

    // Navigate directly to callback with hash parameters (relative path to match /admin/ base URL)
    const callbackUrl = `login/callback#access_token=${encodeURIComponent(accessToken)}&refresh_token=${encodeURIComponent(refreshToken)}&redirect_to=%2Fadmin`;
    await page.goto(callbackUrl);

    // Wait for token processing — the useEffect sets cookies via Zustand before redirecting
    // Check cookies for the stored access token
    await expect
      .poll(
        async () => {
          return page.evaluate(() => {
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
        },
        { timeout: 5_000 },
      )
      .toBe(accessToken);

    // Verify refresh token is also stored in cookie
    const storedRefresh = await page.evaluate(() => {
      const prefix = "fluxbase_admin_refresh_token=";
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
    expect(storedRefresh).toBe(refreshToken);
  });

  test("callback page handles missing tokens", async ({ page }) => {
    // Navigate to callback with no hash and no error (relative path for /admin/ base)
    await page.goto("login/callback");

    // Should show "Completing SSO login..." briefly, then process
    // The component detects no tokens and shows an error
    await page.waitForTimeout(2000);

    // The callback should have detected no tokens and attempted to redirect
    // We can verify by checking that auth cookies were NOT set
    const storedToken = await page.evaluate(() => {
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
    expect(storedToken).toBeNull();
  });
});
