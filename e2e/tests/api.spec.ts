import { test, expect } from "@playwright/test";

test.describe("API Gateway", () => {
  test("GET /api/v1/version returns platform info", async ({ request }) => {
    const res = await request.get("/api/v1/version");
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(data.name).toBe("arcana");
    expect(data.services).toBeGreaterThanOrEqual(20);
    expect(data.crds).toBeGreaterThanOrEqual(10);
    expect(data.planes).toBe(8);
  });

  test("GET /api/v1/health returns service health", async ({ request }) => {
    const res = await request.get("/api/v1/health");
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(data.services).toBeDefined();
    expect(Array.isArray(data.services)).toBeTruthy();
    expect(data.services.length).toBeGreaterThan(0);
  });

  test("GET /api/v1/routes returns route list", async ({ request }) => {
    const res = await request.get("/api/v1/routes");
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(Array.isArray(data)).toBeTruthy();
    expect(data.length).toBeGreaterThan(10);
  });

  test("GET /api/v1/agents returns agent list", async ({ request }) => {
    const res = await request.get("/api/v1/agents");
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(data).toHaveProperty("agents");
    expect(data).toHaveProperty("total");
  });

  test("GET /api/v1/tasks returns task list", async ({ request }) => {
    const res = await request.get("/api/v1/tasks");
    expect(res.ok()).toBeTruthy();
  });

  test("GET /api/v1/tools returns tools", async ({ request }) => {
    const res = await request.get("/api/v1/tools");
    expect(res.ok()).toBeTruthy();
  });

  test("GET /api/v1/models returns models", async ({ request }) => {
    const res = await request.get("/api/v1/models");
    expect(res.ok()).toBeTruthy();
  });

  test("GET /api/v1/costs returns costs", async ({ request }) => {
    const res = await request.get("/api/v1/costs");
    expect(res.ok()).toBeTruthy();
  });

  test("GET /api/v1/connectors returns connectors", async ({ request }) => {
    const res = await request.get("/api/v1/connectors");
    expect(res.ok()).toBeTruthy();
  });

  test("GET /api/v1/budget returns budget info", async ({ request }) => {
    const res = await request.get("/api/v1/budget");
    expect(res.ok()).toBeTruthy();
  });
});
