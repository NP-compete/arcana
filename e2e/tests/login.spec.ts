import { test, expect } from "@playwright/test";

test.describe("Login & Authentication UI", () => {
  test("auth/me returns identity with role header", async ({ request }) => {
    const res = await request.get("/api/v1/auth/me", {
      headers: { "X-Arcana-Role": "admin" },
    });
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(data.user_id).toBe("anonymous");
    expect(data.auth_type).toBe("open");
    expect(data.roles).toContain("admin");
  });

  test("auth/me returns developer persona", async ({ request }) => {
    const res = await request.get("/api/v1/auth/me", {
      headers: { "X-Arcana-Role": "developer" },
    });
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(data.user_id).toBe("alex");
    expect(data.roles).toContain("developer");
  });

  test("auth/me returns user persona", async ({ request }) => {
    const res = await request.get("/api/v1/auth/me", {
      headers: { "X-Arcana-Role": "user" },
    });
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(data.user_id).toBe("maya");
    expect(data.roles).toContain("user");
  });

  test("user role is denied access to enterprise config", async ({ request }) => {
    const res = await request.get("/api/v1/enterprise/config", {
      headers: { "X-Arcana-Role": "user" },
    });
    expect(res.status()).toBe(403);
  });

  test("developer role can access agents", async ({ request }) => {
    const res = await request.get("/api/v1/agents", {
      headers: { "X-Arcana-Role": "developer" },
    });
    expect(res.ok()).toBeTruthy();
  });

  test("auth/me returns sre persona", async ({ request }) => {
    const res = await request.get("/api/v1/auth/me", {
      headers: { "X-Arcana-Role": "sre" },
    });
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(data.user_id).toBe("jordan");
    expect(data.roles).toContain("sre");
  });

  test("auth/me returns auditor persona", async ({ request }) => {
    const res = await request.get("/api/v1/auth/me", {
      headers: { "X-Arcana-Role": "auditor" },
    });
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(data.user_id).toBe("sam");
    expect(data.roles).toContain("auditor");
  });

  test("auth/me returns data-engineer persona", async ({ request }) => {
    const res = await request.get("/api/v1/auth/me", {
      headers: { "X-Arcana-Role": "data-engineer" },
    });
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(data.user_id).toBe("priya");
    expect(data.roles).toContain("data-engineer");
  });

  test("auditor denied creating agents", async ({ request }) => {
    const res = await request.post("/api/v1/agents", {
      headers: { "X-Arcana-Role": "auditor" },
    });
    expect(res.status()).toBe(403);
  });

  test("sre denied creating agents", async ({ request }) => {
    const res = await request.post("/api/v1/agents", {
      headers: { "X-Arcana-Role": "sre" },
    });
    expect(res.status()).toBe(403);
  });

  test("login page shows all 6 role cards", async ({ page }) => {
    await page.goto("/");
    await page.waitForLoadState("networkidle");
    await expect(page.getByLabel("Sign in as User")).toBeVisible({ timeout: 5000 });
    await expect(page.getByLabel("Sign in as Developer")).toBeVisible();
    await expect(page.getByLabel("Sign in as Data Engineer")).toBeVisible();
    await expect(page.getByLabel("Sign in as SRE")).toBeVisible();
    await expect(page.getByLabel("Sign in as Auditor")).toBeVisible();
    await expect(page.getByLabel("Sign in as Administrator")).toBeVisible();
  });

  test("sign in as User shows limited nav", async ({ page }) => {
    await page.goto("/");
    await page.waitForLoadState("networkidle");
    await page.getByLabel("Sign in as User").click();
    await page.waitForLoadState("networkidle");
    const nav = page.locator("nav");
    await expect(nav.getByText("Dashboard")).toBeVisible({ timeout: 5000 });
    await expect(nav.getByText("Agents")).toBeVisible();
    await expect(nav.getByText("Settings")).not.toBeVisible();
    await expect(nav.getByText("Connectors")).not.toBeVisible();
  });

  test("sign in as Developer shows dev nav", async ({ page }) => {
    await page.goto("/");
    await page.waitForLoadState("networkidle");
    await page.getByLabel("Sign in as Developer").click();
    await page.waitForLoadState("networkidle");
    const nav = page.locator("nav");
    await expect(nav.getByText("Dashboard")).toBeVisible({ timeout: 5000 });
    await expect(nav.getByText("Agents")).toBeVisible();
    await expect(nav.getByText("Connectors")).toBeVisible();
    await expect(nav.getByText("Settings")).not.toBeVisible();
  });

  test("sign in as Admin shows all nav", async ({ page }) => {
    await page.goto("/");
    await page.waitForLoadState("networkidle");
    await page.getByLabel("Sign in as Administrator").click();
    await page.waitForLoadState("networkidle");
    const nav = page.locator("nav");
    await expect(nav.getByText("Dashboard")).toBeVisible({ timeout: 5000 });
    await expect(nav.getByText("Settings")).toBeVisible();
    await expect(nav.getByText("Connectors")).toBeVisible();
  });

  test("sign in as Data Engineer shows data nav", async ({ page }) => {
    await page.goto("/");
    await page.waitForLoadState("networkidle");
    await page.getByLabel("Sign in as Data Engineer").click();
    await page.waitForLoadState("networkidle");
    const nav = page.locator("nav");
    await expect(nav.getByText("Dashboard")).toBeVisible({ timeout: 5000 });
    await expect(nav.getByText("Agents")).toBeVisible();
    await expect(nav.getByText("Connectors")).toBeVisible();
    await expect(nav.getByText("MCP Servers")).toBeVisible();
    await expect(nav.getByText("Models")).toBeVisible();
    await expect(nav.getByText("Skills")).not.toBeVisible();
    await expect(nav.getByText("Settings")).not.toBeVisible();
  });

  test("sign in as SRE shows ops nav", async ({ page }) => {
    await page.goto("/");
    await page.waitForLoadState("networkidle");
    await page.getByLabel("Sign in as SRE").click();
    await page.waitForLoadState("networkidle");
    const nav = page.locator("nav");
    await expect(nav.getByText("Dashboard")).toBeVisible({ timeout: 5000 });
    await expect(nav.getByText("Agents")).toBeVisible();
    await expect(nav.getByText("Settings")).toBeVisible();
    await expect(nav.getByText("Connectors")).not.toBeVisible();
    await expect(nav.getByText("Skills")).not.toBeVisible();
  });

  test("sign in as Auditor shows audit nav", async ({ page }) => {
    await page.goto("/");
    await page.waitForLoadState("networkidle");
    await page.getByLabel("Sign in as Auditor").click();
    await page.waitForLoadState("networkidle");
    const nav = page.locator("nav");
    await expect(nav.getByText("Dashboard")).toBeVisible({ timeout: 5000 });
    await expect(nav.getByText("Settings")).toBeVisible();
    await expect(nav.getByText("Connectors")).not.toBeVisible();
    await expect(nav.getByText("Skills")).not.toBeVisible();
    await expect(nav.getByText("Evaluations")).not.toBeVisible();
  });

  test("sign out returns to login page", async ({ page }) => {
    await page.goto("/");
    await page.waitForLoadState("networkidle");
    await page.getByLabel("Sign in as Administrator").click();
    await page.waitForLoadState("networkidle");
    await page.getByLabel("Sign out").click();
    await expect(page.getByLabel("Sign in as User")).toBeVisible({ timeout: 5000 });
  });
});
