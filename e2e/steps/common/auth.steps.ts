import { Given, When, Then, expect } from "../../fixtures/base";

Given("I am logged in as {string}", async ({ loginAs }, role: string) => {
  await loginAs(role);
});

Given("I am an authenticated {string}", async ({ loginAs }, role: string) => {
  await loginAs(role);
});

Given("I am not logged in", async ({ page }) => {
  await page.goto("/");
  await page.waitForLoadState("networkidle");
});

When("I sign out", async ({ page }) => {
  const signOutBtn = page.getByRole("button", { name: /sign out/i });
  if (await signOutBtn.isVisible({ timeout: 3000 }).catch(() => false)) {
    await signOutBtn.click();
    await page.waitForLoadState("networkidle");
  }
});

Then("I should be on the login page", async ({ page }) => {
  const loginBtn = page.getByLabel("Sign in as Administrator");
  await expect(loginBtn).toBeVisible({ timeout: 10000 });
});

Then(
  "the login page should show {int} role cards",
  async ({ page }, count: number) => {
    const cards = page.locator('[aria-label^="Sign in as"]');
    await expect(cards).toHaveCount(count);
  }
);
