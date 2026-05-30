import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { App } from "./App";

describe("App", () => {
  it("renders the title", () => {
    render(<App />);
    expect(screen.getByText("Arcana")).toBeDefined();
  });

  it("renders navigation items", () => {
    render(<App />);
    expect(screen.getByText("Dashboard")).toBeDefined();
    expect(screen.getByText("Agents")).toBeDefined();
    expect(screen.getByText("Skills")).toBeDefined();
    expect(screen.getByText("Evaluations")).toBeDefined();
    expect(screen.getByText("Settings")).toBeDefined();
  });
});
