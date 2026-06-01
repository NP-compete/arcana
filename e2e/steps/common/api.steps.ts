import { Given, When, Then, expect } from "../../fixtures/base";

Given(
  "I set the API role to {string}",
  async ({ testContext, apiAs }, role: string) => {
    testContext.apiRole = role;
    testContext.apiContext = await apiAs(role);
  }
);

When(
  "I send a GET request to {string}",
  async ({ request, testContext }, url: string) => {
    const ctx = testContext.apiContext || request;
    const res = await ctx.get(url);
    testContext.lastResponse = res;
    testContext.lastResponseBody = await res.json().catch(() => null);
  }
);

When(
  "I send a POST request to {string} with body:",
  async ({ request, testContext }, url: string, body: string) => {
    const ctx = testContext.apiContext || request;
    const res = await ctx.post(url, { data: JSON.parse(body) });
    testContext.lastResponse = res;
    testContext.lastResponseBody = await res.json().catch(() => null);
  }
);

When(
  "I send a DELETE request to {string}",
  async ({ request, testContext }, url: string) => {
    const ctx = testContext.apiContext || request;
    const res = await ctx.delete(url);
    testContext.lastResponse = res;
    testContext.lastResponseBody = await res.json().catch(() => null);
  }
);

Then(
  "the response status should be {int}",
  async ({ testContext }, status: number) => {
    expect(testContext.lastResponse.status()).toBe(status);
  }
);

Then(
  "the response should be successful",
  async ({ testContext }) => {
    expect(testContext.lastResponse.ok()).toBeTruthy();
  }
);

Then(
  "the response should contain {string}",
  async ({ testContext }, text: string) => {
    const body = JSON.stringify(testContext.lastResponseBody);
    expect(body).toContain(text);
  }
);

Then(
  "the response JSON should have property {string}",
  async ({ testContext }, prop: string) => {
    expect(testContext.lastResponseBody).toHaveProperty(prop);
  }
);

Then(
  "the response JSON at {string} should be {string}",
  async ({ testContext }, path: string, value: string) => {
    const keys = path.split(".");
    let current = testContext.lastResponseBody;
    for (const key of keys) {
      current = current[key];
    }
    expect(String(current)).toBe(value);
  }
);
