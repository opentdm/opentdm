import { test as base, expect } from "@playwright/test";
import * as fs from "fs";
import * as path from "path";
import { fileURLToPath } from "url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

interface SeedData {
  projectSlug: string;
  configId: string;
  envSlugs: string[];
}

interface E2EFixtures {
  seed: SeedData;
}

// Extend the base test with:
// - automatic pageerror detection (fails the test on any uncaught JS exception)
// - console.error collection (fails on non-benign errors)
// - parsed seed.json access
export const test = base.extend<E2EFixtures>({
  page: async ({ page }, use) => {
    const pageErrors: Error[] = [];
    const consoleErrors: string[] = [];

    // Capture uncaught JS exceptions
    page.on("pageerror", (err) => {
      pageErrors.push(err);
    });

    // Capture console.error calls (filter out benign network/resource lines)
    page.on("console", (msg) => {
      if (msg.type() === "error") {
        const text = msg.text();
        // Ignore known benign patterns: favicon 404s, DevTools noise, etc.
        const benign =
          text.includes("favicon") ||
          text.includes("ERR_NAME_NOT_RESOLVED") ||
          text.includes("ERR_INTERNET_DISCONNECTED") ||
          text.includes("net::ERR") ||
          // Playwright injects this for coverage
          text.includes("__playwright");
        if (!benign) {
          consoleErrors.push(text);
        }
      }
    });

    await use(page);

    // Assert after test body completes
    expect(pageErrors, `Uncaught page errors: ${pageErrors.map((e) => e.message).join(", ")}`).toHaveLength(0);
    expect(consoleErrors, `Console errors: ${consoleErrors.join(", ")}`).toHaveLength(0);
  },

  seed: async (
    // Playwright fixture callbacks receive a fixtures object; we don't need any deps here.
    // eslint-disable-next-line no-empty-pattern
    {},
    use,
  ) => {
    const seedPath = path.join(__dirname, ".auth", "seed.json");
    if (!fs.existsSync(seedPath)) {
      throw new Error(`seed.json not found at ${seedPath} — did global-setup run?`);
    }
    const seed = JSON.parse(fs.readFileSync(seedPath, "utf-8")) as SeedData;
    await use(seed);
  },
});

export { expect };
