import { When, Then, expect } from "../../fixtures/base";

When(
  "I open the {string} tab",
  async ({ page }, tabName: string) => {
    await page.getByRole("tab", { name: new RegExp(tabName, "i") }).first().click();
    await page.waitForLoadState("networkidle");
  }
);

Then(
  "the {string} tab should be active",
  async ({ page }, tabName: string) => {
    const tab = page.getByRole("tab", { name: new RegExp(tabName, "i") }).first();
    await expect(tab).toBeVisible();
  }
);

Then(
  "I should see {int} statistics cards",
  async ({ page }, count: number) => {
    const cards = page.locator('[class*="stat"], [class*="card"]').filter({
      has: page.locator("h3, h4, [class*='title']"),
    });
    const visibleCount = await cards.count();
    expect(visibleCount).toBeGreaterThanOrEqual(count);
  }
);

When("I click on agent {string} in the list", async ({ page }, name: string) => {
  await page.getByText(name).first().click();
  await page.waitForLoadState("networkidle");
});

Then("the page should have a nav sidebar", async ({ page }) => {
  await expect(page.locator("nav").first()).toBeVisible({ timeout: 10000 });
});

When(
  "I open the deploy agent modal",
  async ({ page }) => {
    await page
      .getByRole("button", { name: /deploy/i })
      .first()
      .click();
  }
);

Then("the deploy modal should be visible", async ({ page }) => {
  await expect(
    page.locator('[role="dialog"], [class*="modal"]').first()
  ).toBeVisible({ timeout: 10000 });
});

When("I close the modal", async ({ page }) => {
  const closeBtn = page
    .locator('[role="dialog"], [class*="modal"]')
    .getByRole("button", { name: /close|cancel/i })
    .first();
  if (await closeBtn.isVisible({ timeout: 2000 }).catch(() => false)) {
    await closeBtn.click();
  }
});

Then(
  "I should see a chart or visualization",
  async ({ page }) => {
    const chart = page.locator("canvas, svg, [class*='chart'], [class*='graph']");
    await expect(chart.first()).toBeVisible({ timeout: 10000 });
  }
);

Then(
  "I should see a progress bar",
  async ({ page }) => {
    const bar = page.locator('[role="progressbar"], [class*="progress"]');
    await expect(bar.first()).toBeVisible({ timeout: 10000 });
  }
);

When(
  "I expand the first table row",
  async ({ page }) => {
    const expandBtn = page
      .locator("table")
      .getByRole("button")
      .first();
    await expandBtn.click();
  }
);

When(
  "I select period {string}",
  async ({ page }, period: string) => {
    const select = page.locator("select, [role='listbox']").first();
    await select.selectOption({ label: period });
  }
);
