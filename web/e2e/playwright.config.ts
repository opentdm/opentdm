import { defineConfig, devices } from "@playwright/test";
import path from "path";
import { fileURLToPath } from "url";

// __dirname equivalent for ESM
const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Resolve paths relative to the web/ root (parent of e2e/)
const webRoot = path.resolve(__dirname, "..");

export default defineConfig({
  testDir: "./journeys",
  fullyParallel: false,
  workers: 1,
  retries: 0,
  reporter: [
    ["html", { outputFolder: path.join(webRoot, "e2e/playwright-report"), open: "never" }],
    ["list"],
  ],
  outputDir: path.join(webRoot, "e2e/test-results"),
  use: {
    baseURL: process.env.E2E_BASE_URL ?? "http://localhost:18099",
    storageState: path.join(webRoot, "e2e/.auth/state.json"),
    screenshot: "on",
    trace: "on-first-retry",
  },
  globalSetup: "./global-setup.ts",
  projects: [
    {
      name: "chromium-light",
      use: {
        ...devices["Desktop Chrome"],
        colorScheme: "light",
      },
    },
    {
      name: "chromium-dark",
      use: {
        ...devices["Desktop Chrome"],
        colorScheme: "dark",
      },
    },
  ],
});
