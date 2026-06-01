import React from "react";
import ReactDOM from "react-dom/client";
import "@patternfly/react-core/dist/styles/base.css";
import "./styles.css";
import { App } from "./App";

const originalFetch = window.fetch;
window.fetch = function (input: RequestInfo | URL, init?: RequestInit) {
  const url = typeof input === "string" ? input : input instanceof URL ? input.toString() : input.url;
  if (url.startsWith("/api/")) {
    const token = localStorage.getItem("arcana_auth_token");
    const role = localStorage.getItem("arcana_auth_role");
    const headers = new Headers(init?.headers);
    if (token && token !== "open" && !headers.has("Authorization")) {
      headers.set("Authorization", `Bearer ${token}`);
    }
    if (role && !headers.has("X-Arcana-Role")) {
      headers.set("X-Arcana-Role", role);
    }
    return originalFetch(input, { ...init, headers });
  }
  return originalFetch(input, init);
};

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);
