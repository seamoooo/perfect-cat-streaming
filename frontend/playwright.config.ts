import { defineConfig, devices } from "@playwright/test";

// E2E config for the NR telemetry loop. Runs against the running compose stack:
//   frontend at $E2E_BASE_URL (default http://frontend:5173 inside the compose
//   network) and backend at $API_BASE (default http://backend:8080).
//
// Intentionally minimal — no parallelism, no auto-retries, no global hooks.
// Each `test.spec.ts` drives its own full lifecycle.

export default defineConfig({
  testDir: "./e2e",
  fullyParallel: false,
  workers: 1,
  retries: 0,
  reporter: [["list"]],
  timeout: 10 * 60_000,
  expect: { timeout: 30_000 },

  use: {
    baseURL: process.env.E2E_BASE_URL ?? "http://frontend:5173",
    headless: true,
    trace: "off",
    video: "off",
    screenshot: "off",
    actionTimeout: 30_000,
    navigationTimeout: 30_000,
    // Browsers block autoplay unless a user gesture happens; let any origin
    // play video without prompting.
    launchOptions: {
      args: ["--autoplay-policy=no-user-gesture-required"],
    },
  },

  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
  ],
});
