import { defineConfig } from "@playwright/test";

export default defineConfig({
  fullyParallel: false,
  workers: 1,
  retries: 0,
  reporter: "line",
  timeout: 60_000,
  expect: { timeout: 10_000 },
  use: {
    baseURL: "http://localhost:5050/admin/",
    trace: "on-first-retry",
    screenshot: "only-on-failure",
  },
  webServer: {
    command: "true",
    port: 5050,
    reuseExistingServer: true,
    timeout: 5_000,
  },
  projects: [
    {
      name: "e2e",
      testDir: "./tests/e2e",
      testIgnore: [/setup\.spec\.ts$/, /_provisioning\.spec\.ts$/],
    },
  ],
});
