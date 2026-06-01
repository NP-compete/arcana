import { Given, When, Then, expect } from "../../fixtures/base";

Given(
  "an agent named {string} exists",
  async ({ request, testContext }, name: string) => {
    const res = await request.get("/api/v1/agents");
    const data = await res.json();
    const exists = data.agents?.some((a: any) => a.name === name);
    if (!exists) {
      const sid = `setup-${Date.now()}`;
      await request.post("/api/v1/chat", {
        data: { message: `Create an agent called ${name}`, session_id: sid },
      });
      await request.post("/api/v1/chat", {
        data: { message: "1", session_id: sid },
      });
      await request.post("/api/v1/chat", {
        data: { message: "none", session_id: sid },
      });
      await request.post("/api/v1/chat", {
        data: { message: "10", session_id: sid },
      });
      await request.post("/api/v1/chat", {
        data: { message: "yes", session_id: sid },
      });
    }
    testContext.currentAgent = name;
  }
);

When(
  "I start creating an agent via chat with name {string}",
  async ({ request, testContext }, name: string) => {
    const sid = `bdd-${Date.now()}`;
    testContext.chatSessionId = sid;
    testContext.currentAgent = name;
    const res = await request.post("/api/v1/chat", {
      data: {
        message: `Create an agent called ${name}`,
        session_id: sid,
      },
    });
    testContext.lastChatResponse = await res.json();
  }
);

When(
  "I respond with {string} in the chat",
  async ({ request, testContext }, message: string) => {
    const res = await request.post("/api/v1/chat", {
      data: { message, session_id: testContext.chatSessionId },
    });
    testContext.lastChatResponse = await res.json();
  }
);

Then(
  "the agent reply should contain {string}",
  async ({ testContext }, text: string) => {
    expect(testContext.lastChatResponse.reply).toContain(text);
  }
);

Then(
  "the agent {string} should be active",
  async ({ request }, name: string) => {
    const res = await request.get("/api/v1/agents");
    const data = await res.json();
    const agent = data.agents?.find((a: any) => a.name === name);
    expect(agent).toBeDefined();
  }
);

Then(
  "the agent {string} should have type {string}",
  async ({ request }, name: string, type: string) => {
    const res = await request.get("/api/v1/agents");
    const data = await res.json();
    const agent = data.agents?.find((a: any) => a.name === name);
    expect(agent).toBeDefined();
    expect(agent.type).toBe(type);
  }
);

When(
  "I send {string} to agent {string}",
  async ({ request, testContext }, message: string, agentName: string) => {
    const sid = testContext.chatSessionId || `chat-${Date.now()}`;
    testContext.chatSessionId = sid;
    const res = await request.post(`/api/v1/agents/${agentName}/chat`, {
      data: { message, session_id: sid },
    });
    testContext.lastChatResponse = await res.json();
  }
);

When(
  "I open the agent detail page for {string}",
  async ({ page }, name: string) => {
    await page.goto(`/agents/${name}`);
    await page.waitForLoadState("networkidle");
  }
);

Then(
  "the agent detail should show {string}",
  async ({ page }, text: string) => {
    await expect(page.getByText(text).first()).toBeVisible({ timeout: 10000 });
  }
);

Then(
  "the chat response should have steps",
  async ({ testContext }) => {
    expect(testContext.lastChatResponse.steps).toBeDefined();
    expect(testContext.lastChatResponse.steps.length).toBeGreaterThan(0);
  }
);
