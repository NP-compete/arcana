import { defineConfig } from "@playwright/test";
import { defineBddConfig } from "playwright-bdd";

const bddTestDir = defineBddConfig({
  features: "features/**/*.feature",
  steps: ["steps/**/*.ts", "fixtures/**/*.ts"],
});

export default defineConfig({
  timeout: 30000,
  retries: 1,
  use: {
    baseURL: process.env.BASE_URL || "http://localhost:8080",
    headless: true,
    screenshot: "only-on-failure",
  },
  projects: [
    {
      name: "bdd",
      testDir: bddTestDir,
      use: { browserName: "chromium" },
    },
    {
      name: "existing",
      testDir: "./tests",
      use: { browserName: "chromium" },
    },
  ],
});
