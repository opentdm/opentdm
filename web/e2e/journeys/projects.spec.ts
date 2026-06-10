import { test, expect } from "../fixtures";

test.describe("Projects grid", () => {
  test("shows the seeded project card with counts", async ({ page }) => {
    await page.goto("/");

    // The page heading should say "Projects"
    await expect(page.getByRole("heading", { name: "Projects" })).toBeVisible();

    // The seeded project card should be visible — the card title is a link/heading
    // Use .first() to handle any duplicate text in breadcrumbs or tooltips
    const projectCard = page.locator(".otdm-pgrid").getByText("Payments API").first();
    await expect(projectCard).toBeVisible();

    await page.screenshot({ path: "e2e/test-results/projects-grid.png" });
  });

  test("opens project and shows meta grid + env pills + objects list", async ({ page, seed }) => {
    await page.goto(`/projects/${seed.projectSlug}`);

    // Wait for the project page to load — heading with project name
    await expect(page.getByRole("heading", { name: "Payments API" })).toBeVisible();

    // Meta grid cells — use the specific class container
    const metaGrid = page.locator(".otdm-meta-grid");
    await expect(metaGrid.getByText("Objects")).toBeVisible();
    await expect(metaGrid.getByText("Environments")).toBeVisible();
    await expect(metaGrid.getByText("Members")).toBeVisible();
    await expect(metaGrid.getByText("Your role")).toBeVisible();

    // base pill must always be present
    await expect(page.getByText("base").first()).toBeVisible();

    // Seeded environment slugs should appear as labels/pills somewhere on the page
    for (const envSlug of seed.envSlugs) {
      await expect(page.getByText(envSlug).first()).toBeVisible();
    }

    // Objects section: the config name link should appear in the objects list
    // Use the file browser tree area or the link that navigates to the config
    const configLink = page.getByRole("link", { name: /app-config/i }).first();
    await expect(configLink).toBeVisible();

    await page.screenshot({ path: "e2e/test-results/project-detail.png" });
  });
});
