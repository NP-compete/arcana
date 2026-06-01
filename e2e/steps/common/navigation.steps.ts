import { When, Then, expect } from "../../fixtures/base";

When("I navigate to {string}", async ({ page }, path: string) => {
  await page.goto(path);
  await page.waitForLoadState("networkidle");
});

When(
  "I click {string} in the navigation",
  async ({ page }, label: string) => {
    await page.locator("nav").getByText(label, { exact: false }).click();
    await page.waitForLoadState("networkidle");
  }
);

Then("I should see the {string} page", async ({ page }, heading: string) => {
  await expect(
    page
      .locator("h1, h2, h3, h4")
      .filter({ hasText: heading })
      .first()
  ).toBeVisible({ timeout: 10000 });
});

Then("I should see {string}", async ({ page }, text: string) => {
  await expect(page.getByText(text).first()).toBeVisible({ timeout: 10000 });
});

Then("I should not see {string}", async ({ page }, text: string) => {
  await expect(page.getByText(text)).toHaveCount(0, { timeout: 5000 });
});

Then(
  "I should see the {string} button",
  async ({ page }, name: string) => {
    await expect(
      page.getByRole("button", { name: new RegExp(name, "i") }).first()
    ).toBeVisible({ timeout: 10000 });
  }
);

Then(
  "I should not see the {string} button",
  async ({ page }, name: string) => {
    await expect(
      page.getByRole("button", { name: new RegExp(name, "i") })
    ).toHaveCount(0, { timeout: 5000 });
  }
);

Then(
  "the navigation should show {string}",
  async ({ page }, label: string) => {
    await expect(
      page.locator("nav").getByText(label, { exact: false }).first()
    ).toBeVisible({ timeout: 10000 });
  }
);

Then(
  "the navigation should not show {string}",
  async ({ page }, label: string) => {
    await expect(
      page.locator("nav").getByText(label, { exact: true })
    ).toHaveCount(0, { timeout: 5000 });
  }
);
