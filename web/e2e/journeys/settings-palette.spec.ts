import { test, expect } from "../fixtures";

test.describe("Settings pages", () => {
  test("Profile settings page loads", async ({ page }) => {
    await page.goto("/settings/account");
    // The settings page heading
    await expect(page.getByRole("heading", { name: /account|settings/i }).first()).toBeVisible();
    // Profile section heading in the content area
    await expect(page.getByRole("heading", { name: "Profile" })).toBeVisible();
    await page.screenshot({ path: "e2e/test-results/settings-profile.png" });
  });

  test("Appearance settings page loads", async ({ page }) => {
    await page.goto("/settings/appearance");
    // The appearance section heading
    await expect(page.getByRole("heading", { name: "Appearance" })).toBeVisible();
    await page.screenshot({ path: "e2e/test-results/settings-appearance.png" });
  });

  test("Activity settings page loads (admin)", async ({ page }) => {
    await page.goto("/settings/activity");
    // The Activity feed should be visible for admin users
    await expect(page.getByRole("heading", { name: /activity/i }).first()).toBeVisible({ timeout: 8000 });
    await page.screenshot({ path: "e2e/test-results/settings-activity.png" });
  });

  test("Users settings page loads (admin)", async ({ page }) => {
    await page.goto("/settings/users");
    await expect(page.getByRole("heading", { name: /users/i }).first()).toBeVisible({ timeout: 8000 });
    // The seeded admin user should appear in the users list
    const username = process.env.E2E_USERNAME ?? "e2e-admin";
    await expect(page.getByText(username).first()).toBeVisible({ timeout: 8000 });
    await page.screenshot({ path: "e2e/test-results/settings-users.png" });
  });
});

test.describe("Command palette", () => {
  test("opens on Ctrl+K / Cmd+K and shows results", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByRole("heading", { name: "Projects" })).toBeVisible();

    // Open with Ctrl+K (works on all platforms in Playwright)
    await page.keyboard.press("Control+k");

    // The command palette dialog should appear
    const palette = page.locator('[role="dialog"][aria-label="Command palette"]');
    await expect(palette).toBeVisible();

    // The search input should be focused
    const input = palette.locator("input");
    await expect(input).toBeVisible();
    await expect(input).toBeFocused();

    // Initial state: shows projects and nav items (the "Go to" section)
    await expect(palette.getByText("All projects")).toBeVisible();

    // Type a query — "payments" should match our seeded project
    await input.fill("payments");
    await expect(palette.getByText("Payments API")).toBeVisible({ timeout: 5000 });

    await page.screenshot({ path: "e2e/test-results/command-palette-open.png" });

    // Close with Escape
    await page.keyboard.press("Escape");
    await expect(palette).not.toBeVisible();
  });

  test("dark mode toggle switches theme", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByRole("heading", { name: "Projects" })).toBeVisible();

    // Find the theme toggle button by accessible name (works even if aria-label is on inner element)
    // "Switch to dark theme" appears in light mode; "Switch to light theme" in dark mode
    const switchToDark = page.getByRole("button", { name: "Switch to dark theme" });
    const switchToLight = page.getByRole("button", { name: "Switch to light theme" });

    // At least one of the two states should be visible
    const isDarkMode = await switchToLight.isVisible();
    const initialDark = isDarkMode;

    if (!isDarkMode) {
      // Currently light mode — click to go dark
      await expect(switchToDark).toBeVisible();
      await switchToDark.click();
      // Now should show "Switch to light theme" (dark mode active)
      await expect(switchToLight).toBeVisible({ timeout: 3000 });
    } else {
      // Currently dark mode — click to go light
      await switchToLight.click();
      // Now should show "Switch to dark theme" (light mode active)
      await expect(switchToDark).toBeVisible({ timeout: 3000 });
    }

    await page.screenshot({ path: "e2e/test-results/theme-toggled.png" });

    // Toggle back to original state
    if (initialDark) {
      await switchToDark.click();
      await expect(switchToLight).toBeVisible({ timeout: 3000 });
    } else {
      await switchToLight.click();
      await expect(switchToDark).toBeVisible({ timeout: 3000 });
    }

    await page.screenshot({ path: "e2e/test-results/theme-restored.png" });
  });
});
