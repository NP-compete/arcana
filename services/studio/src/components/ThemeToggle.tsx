import { useState, useEffect, useCallback } from "react";
import { Button } from "@patternfly/react-core";
import { SunIcon, MoonIcon } from "@patternfly/react-icons";

const STORAGE_KEY = "arcana_theme";

function getSystemPreference(): "dark" | "light" {
  if (typeof window === "undefined") return "dark";
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

function getStoredTheme(): "dark" | "light" | null {
  if (typeof window === "undefined") return null;
  const stored = localStorage.getItem(STORAGE_KEY);
  if (stored === "dark" || stored === "light") return stored;
  return null;
}

function applyTheme(theme: "dark" | "light") {
  document.documentElement.setAttribute("data-theme", theme);
  if (theme === "light") {
    document.documentElement.classList.remove("pf-v6-theme-dark");
    document.body.style.backgroundColor = "#f0f2f5";
    document.body.style.color = "#1a1d2e";
  } else {
    document.documentElement.classList.add("pf-v6-theme-dark");
    document.body.style.backgroundColor = "#0f1117";
    document.body.style.color = "#e2e8f0";
  }
}

export const ThemeToggle = () => {
  const [theme, setTheme] = useState<"dark" | "light">(() => {
    return getStoredTheme() ?? getSystemPreference();
  });

  useEffect(() => {
    applyTheme(theme);
  }, [theme]);

  const toggle = useCallback(() => {
    const next = theme === "dark" ? "light" : "dark";
    setTheme(next);
    localStorage.setItem(STORAGE_KEY, next);
  }, [theme]);

  return (
    <Button
      variant="plain"
      aria-label={`Switch to ${theme === "dark" ? "light" : "dark"} theme`}
      onClick={toggle}
      style={{ color: "#fff", padding: 4 }}
    >
      {theme === "dark" ? <SunIcon /> : <MoonIcon />}
    </Button>
  );
};
