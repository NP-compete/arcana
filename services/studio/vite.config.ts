/// <reference types="vitest/config" />
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  server: {
    port: 3000,
    proxy: {
      "/api": "http://arcana.localhost.me:8080",
      "/auth": "http://arcana.localhost.me:8080",
      "/agents": "http://arcana.localhost.me:8080",
      "/events": "http://arcana.localhost.me:8080",
    },
  },
  test: {
    environment: "jsdom",
    globals: false,
  },
});
