/// <reference types="vitest/config" />
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  server: {
    port: 3000,
    proxy: {
      "/api": "http://localhost:8080",
      "/events": "http://localhost:8084",
    },
  },
  test: {
    environment: "jsdom",
    globals: false,
  },
});
