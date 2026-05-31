import { test, expect } from "@playwright/test";

test.describe("Enterprise Features", () => {
  test("GET /api/v1/enterprise/config returns enterprise configuration", async ({
    request,
  }) => {
    const res = await request.get("/api/v1/enterprise/config");
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(data.auth_mode).toBeDefined();
    expect(data.rbac_enabled).toBe(true);
    expect(data.rate_limiting_enabled).toBe(true);
    expect(data.audit_enabled).toBe(true);
    expect(data.multi_tenancy).toBe(true);
    expect(data.compliance_frameworks).toContain("soc2");
    expect(data.compliance_frameworks).toContain("gdpr");
    expect(data.roles).toContain("admin");
    expect(data.roles).toContain("developer");
    expect(data.roles).toContain("data-engineer");
    expect(data.roles).toContain("sre");
    expect(data.roles).toContain("auditor");
    expect(data.roles).toContain("user");
  });

  test("GET /api/v1/auth/me returns current identity", async ({ request }) => {
    const res = await request.get("/api/v1/auth/me");
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(data.user_id).toBeDefined();
    expect(data.tenant).toBeDefined();
    expect(data.roles).toBeDefined();
    expect(data.auth_type).toBeDefined();
  });

  test("GET /api/v1/auth/roles returns RBAC roles", async ({ request }) => {
    const res = await request.get("/api/v1/auth/roles");
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(data.roles).toBeDefined();
    expect(data.roles.length).toBeGreaterThanOrEqual(3);
    const roleNames = data.roles.map((r: { name: string }) => r.name);
    expect(roleNames).toContain("admin");
    expect(roleNames).toContain("developer");
    expect(roleNames).toContain("user");
  });

  test("GET /api/v1/auth/keys returns API keys", async ({ request }) => {
    const res = await request.get("/api/v1/auth/keys");
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(data.keys).toBeDefined();
    expect(data.keys.length).toBeGreaterThanOrEqual(2);
  });

  test("POST /api/v1/auth/keys creates a new API key", async ({ request }) => {
    const res = await request.post("/api/v1/auth/keys", {
      data: {
        name: "test-key",
        user_id: "pw-test",
        tenant: "default",
        roles: ["viewer"],
        scopes: ["agents:read"],
        rate_limit_per_second: 10,
      },
    });
    expect(res.status()).toBe(201);
    const data = await res.json();
    expect(data.key).toBeDefined();
    expect(data.warning).toContain("Store this key securely");
    expect(data.details.name).toBe("test-key");
  });

  test("GET /api/v1/tenants returns tenants", async ({ request }) => {
    const res = await request.get("/api/v1/tenants");
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(data.tenants).toBeDefined();
    expect(data.tenants.length).toBeGreaterThanOrEqual(1);
    const defaultTenant = data.tenants.find(
      (t: { id: string }) => t.id === "default"
    );
    expect(defaultTenant).toBeDefined();
    expect(defaultTenant.namespace).toBe("arcana");
  });

  test("POST /api/v1/tenants creates a new tenant", async ({ request }) => {
    const res = await request.post("/api/v1/tenants", {
      data: {
        id: "test-org",
        name: "Test Organization",
        max_agents: 25,
        max_models: 10,
        budget_limit_usd: 2500,
      },
    });
    expect(res.status()).toBe(201);
    const data = await res.json();
    expect(data.id).toBe("test-org");
    expect(data.namespace).toBe("arcana-tenant-test-org");
  });

  test("GET /api/v1/enterprise/audit returns audit entries", async ({
    request,
  }) => {
    const res = await request.get("/api/v1/enterprise/audit");
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(data.entries).toBeDefined();
    expect(data.count).toBeGreaterThanOrEqual(0);
  });

  test("GET /api/v1/enterprise/audit/stats returns audit stats", async ({
    request,
  }) => {
    const res = await request.get("/api/v1/enterprise/audit/stats");
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(data.total_entries).toBeGreaterThanOrEqual(0);
    expect(data.chain_intact).toBe(true);
  });

  test("GET /api/v1/compliance?framework=soc2 returns SOC2 report", async ({
    request,
  }) => {
    const res = await request.get("/api/v1/compliance?framework=soc2");
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(data.framework).toBe("soc2");
    expect(data.total_checks).toBeGreaterThan(0);
    expect(data.passed).toBeGreaterThan(0);
    expect(data.checks).toBeDefined();
  });

  test("GET /api/v1/compliance?framework=gdpr returns GDPR report", async ({
    request,
  }) => {
    const res = await request.get("/api/v1/compliance?framework=gdpr");
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(data.framework).toBe("gdpr");
    expect(data.checks.length).toBeGreaterThan(0);
  });

  test("GET /api/v1/compliance?framework=hipaa returns HIPAA report", async ({
    request,
  }) => {
    const res = await request.get("/api/v1/compliance?framework=hipaa");
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(data.framework).toBe("hipaa");
  });

  test("GET /api/v1/compliance?framework=iso27001 returns ISO27001 report", async ({
    request,
  }) => {
    const res = await request.get("/api/v1/compliance?framework=iso27001");
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(data.framework).toBe("iso27001");
  });

  test("GET /api/v1/compliance?framework=euaiact returns EU AI Act report", async ({
    request,
  }) => {
    const res = await request.get("/api/v1/compliance?framework=euaiact");
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(data.framework).toBe("euaiact");
    const transparency = data.checks.find(
      (c: { control: string }) => c.control === "Art. 13 - Transparency"
    );
    expect(transparency).toBeDefined();
    expect(transparency.status).toBe("pass");
  });

  test("GET /api/v1/compliance without framework lists options", async ({
    request,
  }) => {
    const res = await request.get("/api/v1/compliance");
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(data.supported_frameworks).toContain("soc2");
    expect(data.supported_frameworks).toContain("euaiact");
  });
});
