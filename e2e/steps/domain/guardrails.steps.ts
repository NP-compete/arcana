import { When, Then, expect } from "../../fixtures/base";

When(
  "I run the guardrail check with text {string} and direction {string}",
  async ({ request, testContext }, text: string, direction: string) => {
    const res = await request.post("/api/v1/check", {
      data: { text, agent_id: "bdd-test", direction },
    });
    testContext.lastResponse = res;
    testContext.lastResponseBody = await res.json();
  }
);

Then("the verdict should be defined", async ({ testContext }) => {
  expect(testContext.lastResponseBody).toHaveProperty("verdict");
});

When(
  "I create a guardrail rule of type {string} with pattern {string} for agent {string}",
  async (
    { request, testContext },
    type: string,
    pattern: string,
    agentId: string
  ) => {
    const res = await request.post("/api/v1/rules", {
      data: {
        type,
        pattern,
        action: "block",
        severity: "high",
        agent_id: agentId,
      },
    });
    testContext.lastResponse = res;
    testContext.lastResponseBody = await res.json();
  }
);

Then("the rule should be created with an ID", async ({ testContext }) => {
  expect(testContext.lastResponseBody.id).toBeDefined();
});

When(
  "I list guardrail rules for agent {string}",
  async ({ request, testContext }, agentId: string) => {
    const res = await request.get(`/api/v1/rules/agent/${agentId}`);
    testContext.lastResponse = res;
    testContext.lastResponseBody = await res.json();
  }
);

Then(
  "the rules list should contain agent {string}",
  async ({ testContext }, agentId: string) => {
    const hasAgent = testContext.lastResponseBody.rules.some(
      (r: any) => r.agent_id === agentId
    );
    expect(hasAgent).toBeTruthy();
  }
);
