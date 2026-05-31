import { test, expect } from "@playwright/test";

test.describe("Chat Agent Creation Flow (Maya's Use Case)", () => {
  const sessionId = `pw-${Date.now()}`;

  test("step 1: initiate agent creation via chat", async ({ request }) => {
    const res = await request.post("/api/v1/chat", {
      data: {
        message: "Create an agent called playwright-test-agent for testing",
        session_id: sessionId,
      },
    });
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(data.reply).toContain("playwright-test-agent");
    expect(data.reply).toContain("create_agent");
  });

  test("step 2: select agent type (standard)", async ({ request }) => {
    const res = await request.post("/api/v1/chat", {
      data: { message: "1", session_id: sessionId },
    });
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(data.reply).toContain("connectors");
  });

  test("step 3: skip connectors", async ({ request }) => {
    const res = await request.post("/api/v1/chat", {
      data: { message: "none", session_id: sessionId },
    });
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(data.reply).toContain("budget");
  });

  test("step 4: set budget", async ({ request }) => {
    const res = await request.post("/api/v1/chat", {
      data: { message: "$10", session_id: sessionId },
    });
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(data.reply).toContain("proceed");
  });

  test("step 5: confirm creation", async ({ request }) => {
    const res = await request.post("/api/v1/chat", {
      data: { message: "yes", session_id: sessionId },
    });
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(data.reply).toContain("live");
    expect(data.steps).toBeDefined();
    expect(data.steps.length).toBeGreaterThan(0);
  });

  test("step 6: verify agent exists with type", async ({ request }) => {
    const res = await request.get("/api/v1/agents");
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    const agent = data.agents.find(
      (a: { name: string }) => a.name === "playwright-test-agent"
    );
    expect(agent).toBeDefined();
    expect(agent.agent_type).toBe("create_agent");
  });

  test("step 7: agent detail has correct type", async ({ request }) => {
    const res = await request.get(
      "/api/v1/agents/playwright-test-agent/detail"
    );
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(data.name).toBe("playwright-test-agent");
    expect(data.agent_type).toBe("create_agent");
    expect(data.status).toBe("active");
    expect(data.memory).toBeDefined();
  });
});

test.describe("Deep Agent Creation Flow", () => {
  const sessionId = `pw-deep-${Date.now()}`;

  test("create deep agent end-to-end", async ({ request }) => {
    const r1 = await request.post("/api/v1/chat", {
      data: {
        message: "Build a research pipeline agent",
        session_id: sessionId,
      },
    });
    expect(r1.ok()).toBeTruthy();
    const d1 = await r1.json();
    expect(d1.reply).toContain("create_deep_agent");

    const r2 = await request.post("/api/v1/chat", {
      data: { message: "deep", session_id: sessionId },
    });
    expect(r2.ok()).toBeTruthy();

    const r3 = await request.post("/api/v1/chat", {
      data: { message: "Snowflake", session_id: sessionId },
    });
    expect(r3.ok()).toBeTruthy();

    const r4 = await request.post("/api/v1/chat", {
      data: { message: "50", session_id: sessionId },
    });
    expect(r4.ok()).toBeTruthy();
    const d4 = await r4.json();
    expect(d4.reply).toContain("deep agent");

    const r5 = await request.post("/api/v1/chat", {
      data: { message: "world_model, skill_graph", session_id: sessionId },
    });
    expect(r5.ok()).toBeTruthy();
    const d5 = await r5.json();
    expect(d5.reply).toContain("parameter");

    const r6 = await request.post("/api/v1/chat", {
      data: {
        message: "temperature=0.7, max_tokens=4096",
        session_id: sessionId,
      },
    });
    expect(r6.ok()).toBeTruthy();
    const d6 = await r6.json();
    expect(d6.reply).toContain("create_deep_agent");
    expect(d6.reply).toContain("proceed");

    const r7 = await request.post("/api/v1/chat", {
      data: { message: "yes", session_id: sessionId },
    });
    expect(r7.ok()).toBeTruthy();
    const d7 = await r7.json();
    expect(d7.reply).toContain("live");
    expect(d7.steps).toBeDefined();
  });

  test("verify deep agent has correct type and config", async ({
    request,
  }) => {
    const res = await request.get("/api/v1/agents");
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    const deep = data.agents.find(
      (a: { agent_type: string }) => a.agent_type === "create_deep_agent"
    );
    expect(deep).toBeDefined();
    expect(deep.deep_config).toBeDefined();
    expect(deep.deep_config.world_model).toBe(true);
  });
});
