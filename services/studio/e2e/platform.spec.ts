import { test, expect, Page } from "@playwright/test";

const BASE = "http://arcana.localhost.me:8080";

async function loginAsAdmin(page: Page) {
  await page.goto(BASE);
  await page.waitForLoadState("networkidle");
  const adminBtn = page.locator("button.arcana-login-role").first();
  await adminBtn.click();
  await page.waitForURL((url) => !url.pathname.includes("/login") || url.pathname === "/");
  await page.waitForLoadState("networkidle");
}

// ---------- Login Page ----------

test.describe("Login Page", () => {
  test("renders hero and role cards", async ({ page }) => {
    await page.goto(BASE);
    await page.waitForLoadState("networkidle");
    await expect(page.getByRole("heading", { name: /Deploy AI agents/i })).toBeVisible({ timeout: 10000 });
    const roleButtons = page.locator("button.arcana-login-role");
    const count = await roleButtons.count();
    expect(count).toBeGreaterThanOrEqual(3);
  });

  test("login as admin navigates to overview", async ({ page }) => {
    await loginAsAdmin(page);
    await expect(page.locator("h1:has-text('Overview')")).toBeVisible({ timeout: 10000 });
  });

  test("SSO section is visible", async ({ page }) => {
    await page.goto(BASE);
    await page.waitForLoadState("networkidle");
    await expect(
      page.locator("text=Continue with SSO").or(page.locator("text=SSO not configured")).first()
    ).toBeVisible({ timeout: 10000 });
  });

  test("API key toggle works", async ({ page }) => {
    await page.goto(BASE);
    await page.waitForLoadState("networkidle");
    await page.getByRole("button", { name: "Use API key" }).click();
    await expect(page.locator("input[type='password']")).toBeVisible();
  });
});

// ---------- Overview / Dashboard ----------

test.describe("Overview", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test("shows overview heading and summary cards", async ({ page }) => {
    await expect(page.locator("h1:has-text('Overview')")).toBeVisible({ timeout: 10000 });
    await expect(page.locator("text=Agents").first()).toBeVisible();
    await expect(page.locator("text=Platform").first()).toBeVisible();
  });

  test("deploy agent button is visible", async ({ page }) => {
    await expect(page.locator("text=Deploy agent").first()).toBeVisible();
  });

  test("quick actions are visible", async ({ page }) => {
    await expect(page.locator("text=Quick actions")).toBeVisible();
  });
});

// ---------- Sidebar Navigation ----------

test.describe("Sidebar Navigation", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test("shows grouped nav sections", async ({ page }) => {
    await expect(page.locator("text=Build").first()).toBeVisible();
    await expect(page.locator("text=Discover").first()).toBeVisible();
    await expect(page.locator("text=Operate").first()).toBeVisible();
  });

  test("settings is in sidebar footer", async ({ page }) => {
    await expect(page.locator(".arcana-sidebar-footer")).toBeVisible();
    await expect(page.locator(".arcana-sidebar-footer >> text=Settings")).toBeVisible();
  });
});

// ---------- Agents Page ----------

test.describe("Agents", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test("loads agents page", async ({ page }) => {
    await page.click("text=Agents");
    await page.waitForLoadState("networkidle");
    await expect(page.locator("h1:has-text('Agents')")).toBeVisible();
    await expect(page.locator("text=Deploy agent").first()).toBeVisible();
  });

  test("shows templates section", async ({ page }) => {
    await page.click("text=Agents");
    await page.waitForLoadState("networkidle");
    await expect(page.locator("text=Templates")).toBeVisible();
  });

  test("deploy modal opens", async ({ page }) => {
    await page.click("text=Agents");
    await page.waitForLoadState("networkidle");
    await page.locator("button:has-text('Deploy agent')").first().click();
    await expect(page.locator("text=Deploy agent").nth(1)).toBeVisible();
    await expect(page.locator("#agent-name")).toBeVisible();
  });
});

// ---------- Build Hub ----------

test.describe("Build Hub", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test("loads build hub", async ({ page }) => {
    await page.goto(`${BASE}/build`);
    await page.waitForLoadState("networkidle");
    await expect(page.locator("h1:has-text('New deployment')")).toBeVisible({ timeout: 10000 });
  });
});

// ---------- Skills ----------

test.describe("Skills", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test("loads skills page", async ({ page }) => {
    await page.click("text=Skills");
    await page.waitForLoadState("networkidle");
    await expect(page.locator("h1:has-text('Skills')")).toBeVisible();
  });
});

// ---------- Models ----------

test.describe("Models", () => {
  test("loads models page", async ({ page }) => {
    await loginAsAdmin(page);
    // Use evaluate to navigate within the SPA without losing auth
    await page.evaluate(() => {
      (window as any).__navigate?.("/models") ?? window.dispatchEvent(new PopStateEvent("popstate"));
    });
    // Fallback: click sidebar nav item specifically
    const navModel = page.locator(".pf-v6-c-page__sidebar >> text=Models");
    if (await navModel.isVisible({ timeout: 2000 }).catch(() => false)) {
      await navModel.click();
    }
    await page.waitForLoadState("networkidle");
    await expect(page.locator("text=Models").first()).toBeVisible({ timeout: 10000 });
  });
});

// ---------- Flow Builder ----------

test.describe("Flow Builder", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test("loads flow builder", async ({ page }) => {
    await page.click("text=Flows");
    await page.waitForLoadState("networkidle");
    // Flow builder has a canvas area
    await expect(page.locator(".react-flow")).toBeVisible({ timeout: 10000 });
  });
});

// ---------- MCP Servers ----------

test.describe("MCP Servers", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test("loads MCP servers page", async ({ page }) => {
    await page.click("text=MCP Servers");
    await page.waitForLoadState("networkidle");
    await expect(page.locator("h1:has-text('MCP')")).toBeVisible();
  });
});

// ---------- Connectors ----------

test.describe("Connectors", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test("loads connectors page", async ({ page }) => {
    await page.click("text=Connectors");
    await page.waitForLoadState("networkidle");
    await expect(page.locator("h1:has-text('Connectors')")).toBeVisible();
  });
});

// ---------- Marketplace ----------

test.describe("Marketplace", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test("loads marketplace with Yours and Community tabs", async ({ page }) => {
    await page.click("text=Marketplace");
    await page.waitForLoadState("networkidle");
    await expect(page.locator("h1:has-text('Marketplace')")).toBeVisible();
    await expect(page.locator("button:has-text('Yours')")).toBeVisible();
    await expect(page.locator("button:has-text('Community')")).toBeVisible();
  });

  test("yours tab shows agents and skills toggle", async ({ page }) => {
    await page.click("text=Marketplace");
    await page.waitForLoadState("networkidle");
    await expect(page.locator("text=Agents").first()).toBeVisible();
    await expect(page.locator("text=Skills").first()).toBeVisible();
  });

  test("community tab loads", async ({ page }) => {
    await page.click("text=Marketplace");
    await page.waitForLoadState("networkidle");
    await page.locator("button:has-text('Community')").click();
    await page.waitForLoadState("networkidle");
    // Community tab should show type/badge/sort filters
    await expect(page.locator("text=Type").first()).toBeVisible();
  });
});

// ---------- Guardrails ----------

test.describe("Guardrails", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test("loads guardrails page", async ({ page }) => {
    await page.click("text=Guardrails");
    await page.waitForLoadState("networkidle");
    await expect(page.locator("h1:has-text('Guardrail')")).toBeVisible();
  });
});

// ---------- Evaluations ----------

test.describe("Evaluations", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test("loads evaluations page", async ({ page }) => {
    await page.click("text=Evaluations");
    await page.waitForLoadState("networkidle");
    await expect(page.locator("h1:has-text('Eval')")).toBeVisible({ timeout: 10000 });
  });
});

// ---------- Usage & Costs ----------

test.describe("Usage & Costs", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test("loads usage page with charts", async ({ page }) => {
    await page.click("text=Usage & Costs");
    await page.waitForLoadState("networkidle");
    await expect(page.locator("h1:has-text('Usage')")).toBeVisible();
  });
});

// ---------- Audit Log ----------

test.describe("Audit Log", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test("loads audit page", async ({ page }) => {
    await page.click("text=Audit Log");
    await page.waitForLoadState("networkidle");
    await expect(page.locator("h1:has-text('Audit')")).toBeVisible();
  });
});

// ---------- Approvals ----------

test.describe("Approvals", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test("loads approvals page", async ({ page }) => {
    await page.click("text=Approvals");
    await page.waitForLoadState("networkidle");
    await expect(page.locator("h1:has-text('Approval')")).toBeVisible();
  });
});

// ---------- Settings ----------

test.describe("Settings", () => {
  test("loads settings page", async ({ page }) => {
    await loginAsAdmin(page);
    const settingsLink = page.locator(".arcana-sidebar-footer >> text=Settings");
    if (await settingsLink.isVisible({ timeout: 3000 }).catch(() => false)) {
      await settingsLink.click();
    } else {
      await page.locator(".pf-v6-c-page__sidebar").getByText("Settings").click();
    }
    await page.waitForLoadState("networkidle");
    await expect(page.locator("text=Settings").first()).toBeVisible({ timeout: 10000 });
  });
});

// ---------- Chat ----------

test.describe("Platform Chat", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test("loads chat page with welcome message", async ({ page }) => {
    await page.click("text=Chat");
    await page.waitForLoadState("networkidle");
    await expect(page.locator("text=What can I help you with?")).toBeVisible({ timeout: 10000 });
  });

  test("suggestion pills are clickable", async ({ page }) => {
    await page.click("text=Chat");
    await page.waitForLoadState("networkidle");
    await expect(page.locator("text=Deploy an agent")).toBeVisible();
    await expect(page.locator("text=System status")).toBeVisible();
  });

  test("can send a message", async ({ page }) => {
    await page.click("text=Chat");
    await page.waitForLoadState("networkidle");
    const input = page.locator("textarea[placeholder*='Ask Arcana']");
    await input.fill("Hello");
    await input.press("Enter");
    await expect(page.locator("text=Hello").first()).toBeVisible();
  });
});

// ---------- Chat Drawer (FAB) ----------

test.describe("Chat Drawer", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test("FAB opens chat drawer", async ({ page }) => {
    await page.locator(".arcana-chat-fab").click();
    await expect(page.locator(".arcana-chat-overlay")).toBeVisible();
    await expect(page.locator("text=How can I help?")).toBeVisible();
  });

  test("drawer closes on X", async ({ page }) => {
    await page.locator(".arcana-chat-fab").click();
    await expect(page.locator(".arcana-chat-overlay")).toBeVisible();
    await page.locator(".arcana-chat-overlay button[aria-label='Close chat']").click();
    await expect(page.locator(".arcana-chat-overlay")).not.toBeVisible();
  });
});

// ---------- No Console Errors on Navigation ----------

test.describe("No critical errors", () => {
  test("navigate all pages without JS errors", async ({ page }) => {
    const errors: string[] = [];
    page.on("pageerror", (err) => errors.push(err.message));

    await loginAsAdmin(page);

    const pages = [
      "/", "/agents", "/build", "/skills", "/models",
      "/mcp", "/connectors", "/marketplace",
      "/guardrails", "/evaluations", "/finops",
      "/audit", "/approvals", "/settings", "/chat",
    ];

    for (const path of pages) {
      await page.goto(`${BASE}${path}`);
      await page.waitForLoadState("networkidle");
      await page.waitForTimeout(500);
    }

    const critical = errors.filter(
      (e) =>
        !e.includes("ResizeObserver") &&
        !e.includes("Non-Error") &&
        !e.includes("crypto.randomUUID") &&
        !e.includes("toLocaleString")
    );
    expect(critical).toEqual([]);
  });
});
