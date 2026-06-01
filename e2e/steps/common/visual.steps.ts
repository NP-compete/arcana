import { Then, expect } from "../../fixtures/base";

Then(
  "the page should match the visual baseline {string}",
  async ({ page }, name: string) => {
    await page.waitForLoadState("networkidle");
    await page.waitForTimeout(500);
    await expect(page).toHaveScreenshot(`${name}.png`, {
      maxDiffPixelRatio: 0.01,
      fullPage: true,
    });
  }
);

Then(
  "the element {string} should match the visual baseline {string}",
  async ({ page }, selector: string, name: string) => {
    const element = page.locator(selector).first();
    await expect(element).toHaveScreenshot(`${name}.png`, {
      maxDiffPixelRatio: 0.01,
    });
  }
);

Then("I take a screenshot named {string}", async ({ page }, name: string) => {
  await page.screenshot({
    path: `screenshots/${name}.png`,
    fullPage: true,
  });
});
