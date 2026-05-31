import { test, expect } from "@playwright/test";

test.describe("Agent Chat & Memory", () => {
  const sessionId = `pw-mem-${Date.now()}`;

  test.beforeAll(async ({ request }) => {
    const sid = `setup-${Date.now()}`;
    await request.post("/api/v1/chat", {
      data: {
        message: "Create an agent called memory-test-agent",
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
      data: { message: "10", session_id: sid },
    });
    await request.post("/api/v1/chat", {
      data: { message: "yes", session_id: sid },
    });
  });

  test("agent chat returns capabilities", async ({ request }) => {
    const res = await request.post("/api/v1/agents/memory-test-agent/chat", {
      data: { message: "Hello", session_id: sessionId },
    });
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(data.agent).toBe("memory-test-agent");
    expect(data.reply).toContain("Recall memory");
    expect(data.reply).toContain("Run a task");
  });

  test("agent chat handles task submission", async ({ request }) => {
    const res = await request.post("/api/v1/agents/memory-test-agent/chat", {
      data: {
        message: "Run task: generate weekly report",
        session_id: sessionId,
      },
    });
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(data.steps).toBeDefined();
    expect(data.steps.length).toBeGreaterThan(0);
    expect(data.reply).toContain("Task submitted");
  });

  test("agent chat handles cost check", async ({ request }) => {
    const res = await request.post("/api/v1/agents/memory-test-agent/chat", {
      data: { message: "What is my spend?", session_id: sessionId },
    });
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(data.reply).toContain("cost to date");
  });

  test("agent chat handles knowledge search", async ({ request }) => {
    const res = await request.post("/api/v1/agents/memory-test-agent/chat", {
      data: { message: "Search for documentation", session_id: sessionId },
    });
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(data.reply).toContain("searched");
  });

  test("short-term memory is populated", async ({ request }) => {
    await new Promise((r) => setTimeout(r, 2000));
    const res = await request.get(
      "/api/v1/memory/short-term/memory-test-agent"
    );
    expect(res.ok()).toBeTruthy();
    const entries = await res.json();
    expect(Array.isArray(entries)).toBeTruthy();
    expect(entries.length).toBeGreaterThan(0);
  });

  test("long-term memory has conversation entries", async ({ request }) => {
    const res = await request.get(
      "/api/v1/memory/long-term/memory-test-agent/search?query=report&top_k=5"
    );
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(data.results).toBeDefined();
    expect(data.results.length).toBeGreaterThan(0);
  });

  test("agent chat can recall memory", async ({ request }) => {
    const res = await request.post("/api/v1/agents/memory-test-agent/chat", {
      data: {
        message: "What do you recall from earlier?",
        session_id: sessionId,
      },
    });
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(data.steps).toBeDefined();
    const memoryStep = data.steps.find(
      (s: { service: string }) => s.service === "memory"
    );
    expect(memoryStep).toBeDefined();
  });

  test("memory compaction works", async ({ request }) => {
    const res = await request.post("/api/v1/memory/compact", {
      data: { agent_id: "memory-test-agent" },
    });
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(data.compacted_entries).toBeGreaterThan(0);
    expect(data.summary_id).toBeDefined();
  });

  test("agent detail includes memory counts", async ({ request }) => {
    const res = await request.get(
      "/api/v1/agents/memory-test-agent/detail"
    );
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(data.memory).toBeDefined();
    expect(data.memory.long_term_count).toBeGreaterThan(0);
  });
});
