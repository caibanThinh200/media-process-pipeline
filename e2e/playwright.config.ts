import { defineConfig, devices } from "@playwright/test";

/**
 * Pure browser E2E. The frontend (and through it the REAL deployed AWS
 * stack: API Gateway → Lambda → S3 → SQS → worker → DynamoDB → CloudFront)
 * is the only thing under test. Nothing is mocked or stubbed.
 *
 * Set E2E_BASE_URL to test an already-running/deployed frontend; otherwise
 * `pnpm dev` is started in ../frontend using its .env.local.
 */
const baseURL = process.env.E2E_BASE_URL ?? "http://localhost:3000";

export default defineConfig({
  testDir: "./tests",
  fullyParallel: false,
  workers: 1,
  retries: 0,
  timeout: 3 * 60_000, // real cloud processing is slow
  expect: { timeout: 15_000 },
  reporter: [["list"], ["html", { open: "never" }]],
  use: {
    baseURL,
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
    video: "retain-on-failure",
  },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
  webServer: process.env.E2E_BASE_URL
    ? undefined
    : {
        command: "pnpm dev",
        cwd: "../frontend",
        url: baseURL,
        reuseExistingServer: true,
        timeout: 120_000,
      },
});
