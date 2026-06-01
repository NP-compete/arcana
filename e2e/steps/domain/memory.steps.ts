import { Given, When, Then, expect } from "../../fixtures/base";

When(
  "I store short-term memory for agent {string} with key {string} and value {string}",
  async ({ request, testContext }, agentId: string, key: string, value: string) => {
    const res = await request.post("/api/v1/memory/short-term", {
      data: { agent_id: agentId, key, value, ttl: 3600 },
    });
    testContext.lastResponse = res;
    testContext.lastResponseBody = await res.json();
  }
);

When(
  "I retrieve short-term memory for agent {string}",
  async ({ request, testContext }, agentId: string) => {
    const res = await request.get(`/api/v1/memory/short-term/${agentId}`);
    testContext.lastResponse = res;
    testContext.lastResponseBody = await res.json();
  }
);

Then(
  "the short-term memory for agent {string} should have {int} entries",
  async ({ request, testContext }, agentId: string, count: number) => {
    const res = await request.get(`/api/v1/memory/short-term/${agentId}`);
    const data = await res.json();
    expect(Array.isArray(data)).toBeTruthy();
    if (count === 0) {
      expect(data.length).toBe(0);
    } else {
      expect(data.length).toBeGreaterThanOrEqual(count);
    }
  }
);

When(
  "I store long-term memory for agent {string} with content {string}",
  async ({ request, testContext }, agentId: string, content: string) => {
    const res = await request.post("/api/v1/memory/long-term", {
      data: {
        agent_id: agentId,
        content,
        metadata: { source: "bdd-test", type: "preference" },
      },
    });
    testContext.lastResponse = res;
    testContext.lastResponseBody = await res.json();
  }
);

When(
  "I search long-term memory for agent {string} with query {string}",
  async ({ request, testContext }, agentId: string, query: string) => {
    const encodedQuery = encodeURIComponent(query);
    const res = await request.get(
      `/api/v1/memory/long-term/${agentId}/search?query=${encodedQuery}&top_k=5`
    );
    testContext.lastResponse = res;
    testContext.lastResponseBody = await res.json();
  }
);

Then(
  "the search results should have at least {int} entries",
  async ({ testContext }, count: number) => {
    expect(testContext.lastResponseBody.results.length).toBeGreaterThanOrEqual(count);
  }
);

Then("the search results should have scores", async ({ testContext }) => {
  const first = testContext.lastResponseBody.results[0];
  expect(first.score).toBeDefined();
});

When(
  "I compact memory for agent {string}",
  async ({ request, testContext }, agentId: string) => {
    const res = await request.post("/api/v1/memory/compact", {
      data: { agent_id: agentId },
    });
    testContext.lastResponse = res;
    testContext.lastResponseBody = await res.json();
  }
);

Then(
  "the compaction should report at least {int} compacted entry",
  async ({ testContext }, count: number) => {
    expect(testContext.lastResponseBody.compacted).toBeGreaterThanOrEqual(count);
  }
);

Then("a summary ID should be returned", async ({ testContext }) => {
  expect(testContext.lastResponseBody.summary_id).toBeDefined();
});

Then(
  "the long-term memory embedding should have {int} dimensions",
  async ({ testContext }, dims: number) => {
    expect(testContext.lastResponseBody.embedding).toBeDefined();
    expect(testContext.lastResponseBody.embedding.length).toBe(dims);
  }
);
