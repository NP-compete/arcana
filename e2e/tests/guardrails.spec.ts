import { test, expect } from "@playwright/test";

test.describe("Guardrails", () => {
  test("POST /api/v1/check runs guardrail pipeline", async ({ request }) => {
    const res = await request.post("/api/v1/check", {
      data: { text: "Hello world", agent_id: "test", direction: "input" },
    });
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(data).toHaveProperty("verdict");
  });

  test("POST /api/v1/rules creates a rule", async ({ request }) => {
    const res = await request.post("/api/v1/rules", {
      data: {
        type: "pattern",
        pattern: "test-pw-pattern",
        action: "block",
        severity: "high",
        agent_id: "pw-guardrail-agent",
      },
    });
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(data.id).toBeDefined();
    expect(data.agent_id).toBe("pw-guardrail-agent");
  });

  test("GET /api/v1/rules/agent/{id} returns rules", async ({ request }) => {
    const res = await request.get("/api/v1/rules/agent/pw-guardrail-agent");
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(data.rules).toBeDefined();
    expect(
      data.rules.some(
        (r: { agent_id: string }) => r.agent_id === "pw-guardrail-agent"
      )
    ).toBeTruthy();
  });

  test("GET /api/v1/stats returns ward statistics", async ({ request }) => {
    const res = await request.get("/api/v1/stats");
    expect(res.ok()).toBeTruthy();
  });
});
