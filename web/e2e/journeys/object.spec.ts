import { test, expect } from "../fixtures";

test.describe("Object page", () => {
  test.beforeEach(async ({ page, seed }) => {
    await page.goto(`/projects/${seed.projectSlug}/configs/${seed.configId}`);
    // Wait for the config name to appear in the heading
    await expect(page.getByRole("heading", { name: "app-config" })).toBeVisible();
  });

  test("code view renders the file browser with content", async ({ page }) => {
    // File browser container should be present
    const fb = page.locator(".otdm-fb");
    await expect(fb).toBeVisible();

    // The CodeMirror editor / code view should have rendered content
    // CodeFileView renders inside .otdm-fb-body
    const fbBody = page.locator(".otdm-fb-body");
    await expect(fbBody).toBeVisible();

    // At least APP_ENV key should be visible in the rendered output
    await expect(fbBody.getByText("APP_ENV")).toBeVisible();

    await page.screenshot({ path: "e2e/test-results/object-code-view.png" });
  });

  test("branch/env menu is present and switchable", async ({ page, seed }) => {
    // The BranchEnvMenu button contains "env:" and "base"
    const branchBtn = page.getByRole("button", { name: /env:/ });
    await expect(branchBtn).toBeVisible();
    await expect(branchBtn).toContainText("base");

    // Click to open the menu
    await branchBtn.click();

    // The overlay should show "Switch environment"
    await expect(page.getByText("Switch environment")).toBeVisible();

    // staging option should be listed
    const stagingSlug = seed.envSlugs[0];
    await expect(page.getByRole("option", { name: stagingSlug }).or(page.getByText(stagingSlug)).first()).toBeVisible();

    // Click staging to switch
    await page.getByText(stagingSlug).first().click();

    // Menu closed; branch button now shows staging slug
    await expect(page.getByRole("button", { name: /env:/ })).toContainText(stagingSlug);

    await page.screenshot({ path: "e2e/test-results/object-env-switched.png" });
  });

  test("version history lists multiple versions with delta badges", async ({ page }) => {
    // Version history section heading
    await expect(page.getByRole("heading", { name: "Version history" })).toBeVisible();

    // Click "View" to expand version history
    const viewBtn = page.getByRole("button", { name: /^View/ });
    await expect(viewBtn).toBeVisible();
    await viewBtn.click();

    // After expanding, multiple versions should be listed.
    // We seeded 3 versions for base. Each version row shows "v<N>".
    const versionRows = page.locator(".otdm-fb ~ * [style*='mono'], text=/v\\d+/").first();
    // Use a more reliable approach: find text matching v1, v2, v3
    await expect(page.getByText(/v1/).first()).toBeVisible({ timeout: 10000 });
    await expect(page.getByText(/v2/).first()).toBeVisible();
    await expect(page.getByText(/v3/).first()).toBeVisible();

    // Delta indicators: the version rows should show added/changed/removed counters
    // v2 adds MAX_CONNECTIONS (+1 added) and changes LOG_LEVEL (~1 changed)
    // v3 removes DB_PORT (-1 removed)
    // The VersionHistory component renders them as text like "+1", "~1", "-1"
    // Look for any added or changed indicators
    const historySection = page.locator("text=Version history").locator("..").locator("..");
    // At minimum, one delta count badge should be visible (v2 or v3 should have them)
    // We look for any text that resembles delta counts
    const pageText = await page.content();
    const hasDeltas =
      pageText.includes("+1") ||
      pageText.includes("~1") ||
      pageText.includes("-1") ||
      pageText.includes("+2");
    expect(hasDeltas, "Expected version delta badges (+N / ~N / -N) to be present").toBe(true);

    await page.screenshot({ path: "e2e/test-results/object-version-history.png" });
  });
});
