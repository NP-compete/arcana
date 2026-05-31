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

test.describe("Agent Detail View UI", () => {
  test.beforeAll(async ({ request }) => {
    const sid = `ui-setup-${Date.now()}`;
    await request.post("/api/v1/chat", {
      data: {
        message: "Create an agent called ui-detail-agent for content",
        session_id: sid,
      },
    });
    await request.post("/api/v1/chat", {
      data: { message: "1", session_id: sid },
    });
    await request.post("/api/v1/chat", {
      data: { message: "none", session_id: sid },
    });
    await request.post("/api/v1/chat", {
      data: { message: "15", session_id: sid },
    });
    await request.post("/api/v1/chat", {
      data: { message: "yes", session_id: sid },
    });
  });

  test("agent detail page loads", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/agents/ui-detail-agent");
    await page.waitForLoadState("networkidle");
    const heading = page.locator("h1, h2, h3, h4").filter({ hasText: "ui-detail-agent" });
    await expect(heading.first()).toBeVisible({ timeout: 10000 });
  });

  test("agent detail shows agent name", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/agents/ui-detail-agent");
    await page.waitForLoadState("networkidle");
    await expect(page.locator("text=ui-detail-agent").first()).toBeVisible({ timeout: 10000 });
  });

  test("agent detail has Open Chat button", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/agents/ui-detail-agent");
    await page.waitForLoadState("networkidle");
    await expect(page.locator("text=ui-detail-agent").first()).toBeVisible({ timeout: 10000 });
    const chatBtn = page.getByRole("button", { name: /chat/i }).first();
    await expect(chatBtn).toBeVisible({ timeout: 10000 });
  });

  test("agent detail has Memory section", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/agents/ui-detail-agent");
    await page.waitForLoadState("networkidle");
    await expect(page.locator("text=ui-detail-agent").first()).toBeVisible({ timeout: 10000 });
    await expect(page.locator("text=Memory").first()).toBeVisible({
      timeout: 10000,
    });
  });

  test("agent detail has Guardrails section", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/agents/ui-detail-agent");
    await page.waitForLoadState("networkidle");
    await expect(page.locator("text=ui-detail-agent").first()).toBeVisible({ timeout: 10000 });
    await expect(page.locator("text=Guardrail").first()).toBeVisible({
      timeout: 10000,
    });
  });

  test("agent chat page loads", async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto("/agents/ui-detail-agent/chat");
    await page.waitForLoadState("networkidle");
    await expect(page.locator("body")).toBeVisible({ timeout: 10000 });
  });
});
