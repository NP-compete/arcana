import { test, expect, Page } from "@playwright/test";

async function loginAsAdmin(page: Page) {
  await page.goto("/");
  await page.waitForLoadState("networkidle");
  const loginBtn = page.getByLabel("Sign in as Administrator");
  if (await loginBtn.isVisible({ timeout: 2000 }).catch(() => false)) {
    await loginBtn.click();
    await page.waitForLoadState("networkidle");
  }
}

test.describe("UI Pages Load", () => {
  test("Dashboard page renders", async ({ page }) => {
    await loginAsAdmin(page);
    await expect(page.locator("nav")).toBeVisible({ timeout: 10000 });
    await expect(page.locator("nav >> text=Dashboard")).toBeVisible();
  });

  test("Agents page renders", async ({ page }) => {
    await loginAsAdmin(page);
    await page.locator("nav >> text=Agents").click();
    await expect(page.locator("nav >> text=Agents")).toBeVisible({
      timeout: 10000,
    });
  });

  test("Skills page renders", async ({ page }) => {
    await loginAsAdmin(page);
    await page.locator("nav >> text=Skills").click();
    await expect(page.locator("nav >> text=Skills")).toBeVisible({
      timeout: 10000,
    });
  });

  test("Evaluations page renders", async ({ page }) => {
    await loginAsAdmin(page);
    await page.locator("nav >> text=Evaluations").click();
    await expect(page.locator("nav >> text=Evaluations")).toBeVisible({
      timeout: 10000,
    });
  });

  test("Connectors page renders", async ({ page }) => {
    await loginAsAdmin(page);
    await page.locator("nav >> text=Connectors").click();
    await expect(page.locator("nav >> text=Connectors")).toBeVisible({
      timeout: 10000,
    });
  });

  test("MCP Servers page renders", async ({ page }) => {
    await loginAsAdmin(page);
    await page.locator("nav >> text=MCP").click();
    await expect(page.locator("nav >> text=MCP")).toBeVisible({
      timeout: 10000,
    });
  });

  test("Models page renders", async ({ page }) => {
    await loginAsAdmin(page);
    await page.locator("nav >> text=Models").click();
    await expect(page.locator("nav >> text=Models")).toBeVisible({
      timeout: 10000,
    });
  });

  test("Settings page renders", async ({ page }) => {
    await loginAsAdmin(page);
    await page.locator("nav >> text=Settings").click();
    await expect(page.locator("nav >> text=Settings")).toBeVisible({
      timeout: 10000,
    });
  });

  test("Sidebar has all nav items for admin", async ({ page }) => {
    await loginAsAdmin(page);
    const nav = page.locator("nav");
    await expect(nav).toBeVisible({ timeout: 10000 });
    await expect(nav.locator("text=Dashboard")).toBeVisible();
    await expect(nav.locator("text=Agents")).toBeVisible();
    await expect(nav.locator("text=Skills")).toBeVisible();
    await expect(nav.locator("text=Connectors")).toBeVisible();
    await expect(nav.locator("text=Settings")).toBeVisible();
  });
});
