import { Then, expect } from "../../fixtures/base";

Then(
  "I should see a table with at least {int} rows",
  async ({ page }, count: number) => {
    const rows = page.locator("table tbody tr, table tr").filter({ hasNotText: /^$/ });
    const rowCount = await rows.count();
    expect(rowCount).toBeGreaterThanOrEqual(count);
  }
);

Then(
  "the table should contain {string}",
  async ({ page }, text: string) => {
    await expect(
      page.locator("table").getByText(text).first()
    ).toBeVisible({ timeout: 10000 });
  }
);

Then(
  "the table row {string} should have status {string}",
  async ({ page }, rowText: string, status: string) => {
    const row = page.locator("tr").filter({ hasText: rowText }).first();
    await expect(row.getByText(status)).toBeVisible();
  }
);
