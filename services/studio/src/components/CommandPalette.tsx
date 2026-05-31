import { useState, useEffect, useCallback, useRef } from "react";
import { useNavigate } from "react-router-dom";
import {
  HomeIcon,
  RobotIcon,
  CubesIcon,
  CodeIcon,
  BrainIcon,
  ChartBarIcon,
  CogIcon,
  PluggedIcon,
  ProjectDiagramIcon,
  ShoppingCartIcon,
  UsersIcon,
  MoneyBillIcon,
  ClipboardCheckIcon,
  CommentsIcon,
  ShieldAltIcon,
  PlusCircleIcon,
  CheckCircleIcon,
  EditIcon,
  RocketIcon,
  SearchIcon,
} from "@patternfly/react-icons";

/* ------------------------------------------------------------------ */
/*  Types                                                              */
/* ------------------------------------------------------------------ */

interface Command {
  id: string;
  group: "Navigation" | "Actions" | "Settings";
  label: string;
  icon: React.ReactNode;
  shortcut?: string;
  action: () => void;
}

/* ------------------------------------------------------------------ */
/*  Component                                                          */
/* ------------------------------------------------------------------ */

export const CommandPalette = () => {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [activeIndex, setActiveIndex] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);
  const navigate = useNavigate();

  const commands: Command[] = [
    /* Navigation */
    { id: "nav-dashboard", group: "Navigation", label: "Dashboard", icon: <HomeIcon />, action: () => navigate("/") },
    { id: "nav-chat", group: "Navigation", label: "Chat", icon: <CommentsIcon />, action: () => navigate("/chat") },
    { id: "nav-agents", group: "Navigation", label: "Agents", icon: <RobotIcon />, action: () => navigate("/agents") },
    { id: "nav-skills", group: "Navigation", label: "Skills", icon: <CubesIcon />, action: () => navigate("/skills") },
    { id: "nav-models", group: "Navigation", label: "Models", icon: <BrainIcon />, action: () => navigate("/models") },
    { id: "nav-mcp", group: "Navigation", label: "MCP Servers", icon: <CodeIcon />, action: () => navigate("/mcp") },
    { id: "nav-connectors", group: "Navigation", label: "Connectors", icon: <PluggedIcon />, action: () => navigate("/connectors") },
    { id: "nav-flow", group: "Navigation", label: "Flow Builder", icon: <ProjectDiagramIcon />, action: () => navigate("/flow-builder") },
    { id: "nav-build", group: "Navigation", label: "Build Hub", icon: <PlusCircleIcon />, action: () => navigate("/build") },
    { id: "nav-marketplace", group: "Navigation", label: "Marketplace", icon: <ShoppingCartIcon />, action: () => navigate("/marketplace") },
    { id: "nav-evals", group: "Navigation", label: "Evaluations", icon: <ChartBarIcon />, action: () => navigate("/evaluations") },
    { id: "nav-guardrails", group: "Navigation", label: "Guardrails", icon: <ShieldAltIcon />, action: () => navigate("/guardrails") },
    { id: "nav-orgchart", group: "Navigation", label: "Org Chart", icon: <UsersIcon />, action: () => navigate("/org-chart") },
    { id: "nav-finops", group: "Navigation", label: "FinOps", icon: <MoneyBillIcon />, action: () => navigate("/finops") },
    { id: "nav-audit", group: "Navigation", label: "Audit", icon: <ClipboardCheckIcon />, action: () => navigate("/audit") },
    { id: "nav-approvals", group: "Navigation", label: "Approvals", icon: <CheckCircleIcon />, action: () => navigate("/approvals") },
    { id: "nav-editor", group: "Navigation", label: "YAML Editor", icon: <EditIcon />, action: () => navigate("/editor") },

    /* Actions */
    { id: "act-deploy-agent", group: "Actions", label: "Deploy Agent", icon: <RocketIcon />, action: () => navigate("/build") },
    { id: "act-create-skill", group: "Actions", label: "Create Skill", icon: <CubesIcon />, action: () => navigate("/build") },
    { id: "act-register-model", group: "Actions", label: "Register Model", icon: <BrainIcon />, action: () => navigate("/build") },
    { id: "act-add-mcp", group: "Actions", label: "Add MCP Server", icon: <CodeIcon />, action: () => navigate("/build") },

    /* Settings */
    { id: "set-settings", group: "Settings", label: "Settings", icon: <CogIcon />, action: () => navigate("/settings") },
  ];

  /* Filter */
  const filtered = query.trim()
    ? commands.filter((c) => c.label.toLowerCase().includes(query.toLowerCase()))
    : commands;

  /* Group commands */
  const groups = ["Navigation", "Actions", "Settings"] as const;
  const grouped = groups
    .map((g) => ({ group: g, items: filtered.filter((c) => c.group === g) }))
    .filter((g) => g.items.length > 0);

  const flatItems = grouped.flatMap((g) => g.items);

  /* Keyboard listener for Cmd+K / Ctrl+K */
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === "k") {
        e.preventDefault();
        setOpen((prev) => !prev);
        setQuery("");
        setActiveIndex(0);
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, []);

  /* Focus input when opening */
  useEffect(() => {
    if (open) {
      setTimeout(() => inputRef.current?.focus(), 50);
    }
  }, [open]);

  /* Reset active index when query changes */
  useEffect(() => {
    setActiveIndex(0);
  }, [query]);

  /* Scroll active item into view */
  useEffect(() => {
    const activeEl = listRef.current?.querySelector(`[data-index="${activeIndex}"]`);
    if (activeEl) {
      activeEl.scrollIntoView({ block: "nearest" });
    }
  }, [activeIndex]);

  const executeCommand = useCallback((cmd: Command) => {
    setOpen(false);
    setQuery("");
    cmd.action();
  }, []);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Escape") {
      setOpen(false);
      setQuery("");
      return;
    }
    if (e.key === "ArrowDown") {
      e.preventDefault();
      setActiveIndex((prev) => (prev + 1) % flatItems.length);
      return;
    }
    if (e.key === "ArrowUp") {
      e.preventDefault();
      setActiveIndex((prev) => (prev - 1 + flatItems.length) % flatItems.length);
      return;
    }
    if (e.key === "Enter") {
      e.preventDefault();
      if (flatItems[activeIndex]) {
        executeCommand(flatItems[activeIndex]);
      }
      return;
    }
  };

  if (!open) return null;

  return (
    <>
      {/* Backdrop */}
      <div
        onClick={() => { setOpen(false); setQuery(""); }}
        style={{
          position: "fixed",
          inset: 0,
          background: "rgba(0, 0, 0, 0.6)",
          backdropFilter: "blur(4px)",
          zIndex: 10001,
        }}
      />

      {/* Palette */}
      <div
        style={{
          position: "fixed",
          top: "20%",
          left: "50%",
          transform: "translateX(-50%)",
          width: "100%",
          maxWidth: 560,
          zIndex: 10002,
          background: "#1a1d2e",
          borderRadius: 14,
          border: "1px solid rgba(255,255,255,0.12)",
          boxShadow: "0 24px 64px rgba(0,0,0,0.5)",
          overflow: "hidden",
          animation: "palette-slide-down 0.15s ease-out",
        }}
        onKeyDown={handleKeyDown}
      >
        {/* Search input */}
        <div style={{
          display: "flex",
          alignItems: "center",
          gap: 10,
          padding: "14px 18px",
          borderBottom: "1px solid rgba(255,255,255,0.08)",
        }}>
          <SearchIcon style={{ color: "#8b95a5", fontSize: 16 }} />
          <input
            ref={inputRef}
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Type a command..."
            style={{
              flex: 1,
              background: "transparent",
              border: "none",
              outline: "none",
              color: "#e2e8f0",
              fontSize: 15,
              fontWeight: 500,
            }}
          />
          <span style={{
            fontSize: 11,
            color: "#6b7280",
            background: "rgba(255,255,255,0.06)",
            padding: "2px 6px",
            borderRadius: 4,
            fontFamily: "monospace",
          }}>
            ESC
          </span>
        </div>

        {/* Results */}
        <div ref={listRef} style={{ maxHeight: 360, overflowY: "auto", padding: "8px 0" }}>
          {flatItems.length === 0 ? (
            <div style={{ padding: "24px 18px", textAlign: "center", color: "#6b7280", fontSize: 14 }}>
              No matching commands
            </div>
          ) : (
            grouped.map((g) => (
              <div key={g.group}>
                <div style={{
                  padding: "8px 18px 4px",
                  fontSize: 11,
                  fontWeight: 700,
                  color: "#6b7280",
                  textTransform: "uppercase",
                  letterSpacing: "0.8px",
                }}>
                  {g.group}
                </div>
                {g.items.map((cmd) => {
                  const idx = flatItems.indexOf(cmd);
                  const isActive = idx === activeIndex;
                  return (
                    <div
                      key={cmd.id}
                      data-index={idx}
                      onClick={() => executeCommand(cmd)}
                      onMouseEnter={() => setActiveIndex(idx)}
                      style={{
                        display: "flex",
                        alignItems: "center",
                        gap: 12,
                        padding: "10px 18px",
                        cursor: "pointer",
                        background: isActive ? "rgba(102, 126, 234, 0.12)" : "transparent",
                        borderLeft: isActive ? "2px solid #667eea" : "2px solid transparent",
                        transition: "background 0.08s ease",
                      }}
                    >
                      <span style={{ color: isActive ? "#667eea" : "#8b95a5", fontSize: 16, display: "flex" }}>
                        {cmd.icon}
                      </span>
                      <span style={{ flex: 1, fontSize: 14, fontWeight: 500, color: isActive ? "#e2e8f0" : "#c5cdd8" }}>
                        {cmd.label}
                      </span>
                      {cmd.shortcut && (
                        <span style={{
                          fontSize: 11,
                          color: "#6b7280",
                          fontFamily: "monospace",
                          background: "rgba(255,255,255,0.06)",
                          padding: "2px 6px",
                          borderRadius: 4,
                        }}>
                          {cmd.shortcut}
                        </span>
                      )}
                    </div>
                  );
                })}
              </div>
            ))
          )}
        </div>

        {/* Footer */}
        <div style={{
          padding: "10px 18px",
          borderTop: "1px solid rgba(255,255,255,0.08)",
          display: "flex",
          gap: 16,
          fontSize: 11,
          color: "#6b7280",
        }}>
          <span><kbd style={{ background: "rgba(255,255,255,0.06)", padding: "1px 4px", borderRadius: 3, fontFamily: "monospace" }}>↑↓</kbd> navigate</span>
          <span><kbd style={{ background: "rgba(255,255,255,0.06)", padding: "1px 4px", borderRadius: 3, fontFamily: "monospace" }}>↵</kbd> select</span>
          <span><kbd style={{ background: "rgba(255,255,255,0.06)", padding: "1px 4px", borderRadius: 3, fontFamily: "monospace" }}>esc</kbd> close</span>
        </div>
      </div>

      <style>{`
        @keyframes palette-slide-down {
          from { opacity: 0; transform: translateX(-50%) translateY(-10px); }
          to { opacity: 1; transform: translateX(-50%) translateY(0); }
        }
      `}</style>
    </>
  );
};
