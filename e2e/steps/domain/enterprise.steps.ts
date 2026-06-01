import { When, Then, expect } from "../../fixtures/base";

When(
  "I create a tenant with ID {string} and name {string}",
  async ({ request, testContext }, id: string, name: string) => {
    const res = await request.post("/api/v1/tenants", {
      data: {
        id,
        name,
        namespace: `ns-${id}`,
        max_agents: 10,
        max_models: 5,
        budget_limit: 1000,
      },
    });
    testContext.lastResponse = res;
    testContext.lastResponseBody = await res.json();
  }
);

When("I list all tenants", async ({ request, testContext }) => {
  const res = await request.get("/api/v1/tenants");
  testContext.lastResponse = res;
  testContext.lastResponseBody = await res.json();
});

Then(
  "the tenant {string} should exist",
  async ({ request }, id: string) => {
    const res = await request.get("/api/v1/tenants");
    const data = await res.json();
    const tenant = data.tenants?.find((t: any) => t.id === id);
    expect(tenant).toBeDefined();
  }
);

When(
  "I check compliance for framework {string}",
  async ({ request, testContext }, framework: string) => {
    const res = await request.get(`/api/v1/compliance?framework=${framework}`);
    testContext.lastResponse = res;
    testContext.lastResponseBody = await res.json();
  }
);

Then(
  "the compliance report should have framework {string}",
  async ({ testContext }, framework: string) => {
    expect(testContext.lastResponseBody.framework).toBe(framework);
  }
);

Then(
  "the compliance report should have controls",
  async ({ testContext }) => {
    expect(testContext.lastResponseBody.controls).toBeDefined();
    expect(testContext.lastResponseBody.controls.length).toBeGreaterThan(0);
  }
);

Then("the audit chain should be intact", async ({ request }) => {
  const res = await request.get("/api/v1/enterprise/audit/stats");
  const data = await res.json();
  expect(data.chain_valid).toBe(true);
});

When("I fetch audit entries", async ({ request, testContext }) => {
  const res = await request.get("/api/v1/enterprise/audit");
  testContext.lastResponse = res;
  testContext.lastResponseBody = await res.json();
});

Then(
  "the audit log should have at least {int} entries",
  async ({ testContext }, count: number) => {
    expect(testContext.lastResponseBody.entries.length).toBeGreaterThanOrEqual(
      count
    );
  }
);
