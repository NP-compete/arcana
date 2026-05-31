import { test, expect } from "@playwright/test";

test.describe("Memory API", () => {
  const agentId = `mem-api-${Date.now()}`;

  test("POST short-term memory stores entry", async ({ request }) => {
    const res = await request.post("/api/v1/memory/short-term", {
      data: {
        agent_id: agentId,
        key: "preference",
        value: "formal tone",
        ttl: 3600,
      },
    });
    expect(res.status()).toBe(201);
    const data = await res.json();
    expect(data.agent_id).toBe(agentId);
    expect(data.key).toBe("preference");
    expect(data.value).toBe("formal tone");
  });

  test("GET short-term memory retrieves entries", async ({ request }) => {
    const res = await request.get(`/api/v1/memory/short-term/${agentId}`);
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(Array.isArray(data)).toBeTruthy();
    expect(data.length).toBeGreaterThan(0);
    expect(data[0].key).toBe("preference");
  });

  test("POST long-term memory stores with embedding", async ({ request }) => {
    const res = await request.post("/api/v1/memory/long-term", {
      data: {
        agent_id: agentId,
        content: "User prefers concise email subjects under 10 words",
        metadata: { source: "annotation", type: "preference" },
      },
    });
    expect(res.status()).toBe(201);
    const data = await res.json();
    expect(data.embedding).toBeDefined();
    expect(data.embedding.length).toBe(64);
  });

  test("GET long-term search returns scored results", async ({ request }) => {
    const res = await request.get(
      `/api/v1/memory/long-term/${agentId}/search?query=email%20subject&top_k=5`
    );
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(data.results.length).toBeGreaterThan(0);
    expect(data.results[0].score).toBeDefined();
    expect(data.results[0].memory.content).toContain("email");
  });

  test("POST compact consolidates short-term to long-term", async ({
    request,
  }) => {
    const res = await request.post("/api/v1/memory/compact", {
      data: { agent_id: agentId },
    });
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(data.compacted_entries).toBeGreaterThan(0);
    expect(data.summary_id).toBeDefined();
  });

  test("short-term is empty after compaction", async ({ request }) => {
    const res = await request.get(`/api/v1/memory/short-term/${agentId}`);
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(data.length).toBe(0);
  });

  test("long-term now has compacted summary", async ({ request }) => {
    const res = await request.get(
      `/api/v1/memory/long-term/${agentId}/search?query=compaction&top_k=10`
    );
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(data.results.length).toBeGreaterThanOrEqual(2);
  });
});
