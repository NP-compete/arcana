import { test as base, createBdd } from "playwright-bdd";
import { APIRequestContext, Page, expect } from "@playwright/test";

const ROLE_LABELS: Record<string, string> = {
  admin: "Administrator",
  developer: "Developer",
  "data-engineer": "Data Engineer",
  sre: "SRE",
  auditor: "Auditor",
  user: "User",
};

type TestContext = Record<string, any>;

type Fixtures = {
  testContext: TestContext;
  loginAs: (role: string) => Promise<void>;
  apiAs: (role: string) => Promise<APIRequestContext>;
};

export const test = base.extend<Fixtures>({
  testContext: async ({}, use) => {
    await use({});
  },

  loginAs: async ({ page }, use) => {
    const fn = async (role: string) => {
      await page.goto("/");
      await page.waitForLoadState("networkidle");
      const label = ROLE_LABELS[role] || role;
      const btn = page.getByLabel(`Sign in as ${label}`);
      if (await btn.isVisible({ timeout: 3000 }).catch(() => false)) {
        await btn.click();
        await page.waitForLoadState("networkidle");
      }
    };
    await use(fn);
  },

  apiAs: async ({ playwright }, use) => {
    const contexts: APIRequestContext[] = [];
    const fn = async (role: string) => {
      const ctx = await playwright.request.newContext({
        baseURL: process.env.BASE_URL || "http://localhost:8080",
        extraHTTPHeaders: { "X-Arcana-Role": role },
      });
      contexts.push(ctx);
      return ctx;
    };
    await use(fn);
    for (const ctx of contexts) {
      await ctx.dispose();
    }
  },
});

export const { Given, When, Then } = createBdd(test);
export { expect };
