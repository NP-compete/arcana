import { When } from "../../fixtures/base";

When(
  "I fill in {string} with {string}",
  async ({ page }, field: string, value: string) => {
    const input = page.getByLabel(field).or(page.getByPlaceholder(field));
    await input.first().fill(value);
  }
);

When(
  "I type {string} in the search box",
  async ({ page }, text: string) => {
    const search = page
      .getByRole("searchbox")
      .or(page.getByPlaceholder(/search/i))
      .or(page.locator('input[type="search"]'));
    await search.first().fill(text);
  }
);

When(
  "I select {string} from the {string} dropdown",
  async ({ page }, option: string, label: string) => {
    const select = page.getByLabel(label).or(page.locator(`select`).filter({ hasText: label }));
    await select.first().selectOption({ label: option });
  }
);

When(
  "I click the {string} button",
  async ({ page }, name: string) => {
    await page
      .getByRole("button", { name: new RegExp(name, "i") })
      .first()
      .click();
  }
);

When("I check the {string} checkbox", async ({ page }, label: string) => {
  await page.getByLabel(label).check();
});

When("I uncheck the {string} checkbox", async ({ page }, label: string) => {
  await page.getByLabel(label).uncheck();
});
