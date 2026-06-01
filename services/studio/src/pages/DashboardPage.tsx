import { useState, useEffect, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import {
  PageSection,
  Spinner,
  Alert,
  Button,
  Label,
} from "@patternfly/react-core";
import {
  PlusCircleIcon,
  ExternalLinkAltIcon,
} from "@patternfly/react-icons";
import { useHealth } from "../hooks/useHealth";
import { useAuth } from "../auth/AuthContext";

const TECH_ROLES = new Set(["developer", "data-engineer", "sre", "auditor", "admin"]);

interface Agent {
  name: string;
  agent_type: string;
  capabilities: string[];
  protocols: string[];
  status: string;
  registered_at?: string;
}

const STATUS_DOT: Record<string, { color: string; label: string }> = {
  active: { color: "#22c55e", label: "Running" },
  busy: { color: "#f59e0b", label: "Busy" },
  idle: { color: "#8b95a5", label: "Sleeping" },
  offline: { color: "#ef4444", label: "Crashed" },
};

export const DashboardPage = () => {
  const navigate = useNavigate();
  const { health, error } = useHealth(5000);
  const { user, isAtLeast } = useAuth();
  const [agents, setAgents] = useState<Agent[]>([]);
  const [loading, setLoading] = useState(true);
  const [skillCount, setSkillCount] = useState(0);
  const [mcpCount, setMcpCount] = useState(0);

  const fetchData = useCallback(async () => {
    try {
      const [agentsRes, skillsRes, mcpRes] = await Promise.allSettled([
        fetch("/api/v1/agents"),
        fetch("/api/v1/skills"),
        fetch("/api/v1/tools"),
      ]);
      if (agentsRes.status === "fulfilled" && agentsRes.value.ok) {
        const data = await agentsRes.value.json();
        setAgents(data.agents ?? []);
      }
      if (skillsRes.status === "fulfilled" && skillsRes.value.ok) {
        const data = await skillsRes.value.json();
        setSkillCount(data.skills?.length ?? 0);
      }
      if (mcpRes.status === "fulfilled" && mcpRes.value.ok) {
        const data = await mcpRes.value.json();
        setMcpCount(data.servers?.length ?? 0);
      }
    } catch {
      /* best effort */
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const totalServices = health?.services.length ?? 0;
  const healthyServices = health?.services.filter((s) => s.status === "healthy").length ?? 0;
  const allHealthy = totalServices > 0 && healthyServices === totalServices;

  const runningCount = agents.filter((a) => a.status === "active").length;
  const sleepingCount = agents.filter((a) => a.status === "idle").length;
  const crashedCount = agents.filter((a) => a.status === "offline").length;
  const isTechUser = user?.roles?.some((r) => TECH_ROLES.has(r)) ?? false;

  return (
    <>
      {error && (
        <PageSection hasBodyWrapper={false} style={{ paddingBottom: 0 }}>
          <Alert variant="warning" title="API unreachable" isInline>
            Cannot reach the Arcana API. Run <code>make dev</code> to start.
          </Alert>
        </PageSection>
      )}

      <PageSection hasBodyWrapper={false}>
        {/* Header */}
        <div style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          marginBottom: 24,
        }}>
          <h1 style={{
            fontSize: 28,
            fontWeight: 700,
            color: "var(--arcana-text)",
            margin: 0,
            letterSpacing: "-0.5px",
          }}>
            {isTechUser ? "Overview" : "Home"}
          </h1>
          {isAtLeast("developer") && (
            <Button
              variant="primary"
              icon={<PlusCircleIcon />}
              onClick={() => navigate("/build")}
            >
              {isTechUser ? "Deploy agent" : "Create agent"}
            </Button>
          )}
        </div>

        {/* Summary cards */}
        <div style={{
          display: "grid",
          gridTemplateColumns: isTechUser ? "repeat(5, 1fr)" : "repeat(4, 1fr)",
          gap: 12,
          marginBottom: 32,
        }}>
          {[
            {
              label: isTechUser ? "Agents" : "My Agents",
              value: agents.length,
              sub: `${runningCount} running`,
              color: "#5b8def",
            },
            {
              label: "Running",
              value: runningCount,
              sub: "active now",
              color: "#22c55e",
            },
            {
              label: isTechUser ? "Skills" : "Capabilities",
              value: skillCount,
              sub: isTechUser ? "registered" : "available",
              color: "#a855f7",
            },
            ...(isTechUser ? [{
              label: "MCP Servers",
              value: mcpCount,
              sub: "connected",
              color: "#06b6d4",
            }] : []),
            {
              label: isTechUser ? "Platform" : "Status",
              value: allHealthy ? "Healthy" : `${healthyServices}/${totalServices}`,
              sub: isTechUser ? (totalServices > 0 ? `${totalServices} services` : "checking...") : (allHealthy ? "All systems go" : "Some issues"),
              color: allHealthy ? "#22c55e" : "#f59e0b",
            },
          ].map((card) => (
            <div
              key={card.label}
              style={{
                padding: "20px 18px",
                background: "var(--arcana-card-bg)",
                borderRadius: 10,
                border: "1px solid var(--arcana-card-border)",
              }}
            >
              <div style={{ fontSize: 12, color: "var(--arcana-text-muted)", fontWeight: 500, marginBottom: 8 }}>
                {card.label}
              </div>
              <div style={{
                fontSize: typeof card.value === "number" ? 28 : 20,
                fontWeight: 700,
                color: "var(--arcana-text)",
                marginBottom: 4,
              }}>
                {loading ? <Spinner size="md" /> : card.value}
              </div>
              <div style={{ fontSize: 11, color: "var(--arcana-text-muted)" }}>
                {card.sub}
              </div>
            </div>
          ))}
        </div>

        {/* Status breakdown — only if agents exist and some aren't running */}
        {agents.length > 0 && (sleepingCount > 0 || crashedCount > 0) && (
          <div style={{
            display: "flex",
            gap: 16,
            marginBottom: 24,
            padding: "12px 16px",
            background: "var(--arcana-card-bg)",
            borderRadius: 8,
            border: "1px solid var(--arcana-card-border)",
            fontSize: 13,
            color: "var(--arcana-text-secondary)",
            alignItems: "center",
          }}>
            <span style={{ display: "inline-flex", alignItems: "center", gap: 6 }}>
              <span style={{ width: 8, height: 8, borderRadius: "50%", background: "#22c55e" }} />
              {runningCount} running
            </span>
            {sleepingCount > 0 && (
              <span style={{ display: "inline-flex", alignItems: "center", gap: 6 }}>
                <span style={{ width: 8, height: 8, borderRadius: "50%", background: "#8b95a5" }} />
                {sleepingCount} sleeping
              </span>
            )}
            {crashedCount > 0 && (
              <span style={{ display: "inline-flex", alignItems: "center", gap: 6 }}>
                <span style={{ width: 8, height: 8, borderRadius: "50%", background: "#ef4444" }} />
                {crashedCount} crashed
              </span>
            )}
            {agents.filter((a) => a.status === "busy").length > 0 && (
              <span style={{ display: "inline-flex", alignItems: "center", gap: 6 }}>
                <span style={{ width: 8, height: 8, borderRadius: "50%", background: "#f59e0b" }} />
                {agents.filter((a) => a.status === "busy").length} busy
              </span>
            )}
          </div>
        )}

        {/* Agent list */}
        <div style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          marginBottom: 12,
        }}>
          <div style={{
            fontSize: 11,
            fontWeight: 700,
            textTransform: "uppercase",
            letterSpacing: "1.2px",
            color: "var(--arcana-text-muted)",
          }}>
            Agents
          </div>
          {agents.length > 0 && (
            <Button variant="link" size="sm" onClick={() => navigate("/agents")}>
              View all
            </Button>
          )}
        </div>

        {loading ? (
          <div style={{ textAlign: "center", padding: 60 }}>
            <Spinner size="xl" />
          </div>
        ) : agents.length === 0 ? (
          <div style={{
            textAlign: "center",
            padding: "60px 40px",
            background: "var(--arcana-card-bg)",
            borderRadius: 12,
            border: "1px solid var(--arcana-card-border)",
          }}>
            <div style={{ fontSize: 48, marginBottom: 16, opacity: 0.4 }}>/</div>
            <h2 style={{
              fontSize: 18,
              fontWeight: 600,
              color: "var(--arcana-text)",
              margin: "0 0 8px",
            }}>
              {isTechUser ? "No agents deployed" : "No agents available yet"}
            </h2>
            <p style={{
              fontSize: 13,
              color: "var(--arcana-text-secondary)",
              maxWidth: 380,
              margin: "0 auto 20px",
              lineHeight: 1.6,
            }}>
              {isTechUser
                ? "Deploy your first agent in under a minute. Pick a template or start from scratch."
                : "Your team hasn't set up any agents yet. Check back soon or ask your admin."}
            </p>
            {isAtLeast("developer") && (
              <Button variant="primary" onClick={() => navigate("/build")}>
                Deploy your first agent
              </Button>
            )}
          </div>
        ) : (
          <div style={{ display: "flex", flexDirection: "column", gap: 1 }}>
            {agents.slice(0, 10).map((agent) => {
              const dot = STATUS_DOT[agent.status] ?? STATUS_DOT.offline;
              const skills = (agent.capabilities ?? []).filter((c) => !c.startsWith("model:"));
              const model = (agent.capabilities ?? []).find((c) => c.startsWith("model:"))?.replace("model:", "");
              return (
                <div
                  key={agent.name}
                  onClick={() => navigate(`/agents/${agent.name}`)}
                  style={{
                    display: "flex",
                    alignItems: "center",
                    gap: 16,
                    padding: "14px 18px",
                    background: "var(--arcana-card-bg)",
                    borderRadius: 10,
                    border: "1px solid var(--arcana-card-border)",
                    cursor: "pointer",
                    transition: "background 0.15s, border-color 0.15s",
                    marginBottom: 6,
                  }}
                  onMouseEnter={(e) => {
                    (e.currentTarget as HTMLDivElement).style.background = "var(--arcana-hover-bg)";
                    (e.currentTarget as HTMLDivElement).style.borderColor = "rgba(91,141,239,0.3)";
                  }}
                  onMouseLeave={(e) => {
                    (e.currentTarget as HTMLDivElement).style.background = "var(--arcana-card-bg)";
                    (e.currentTarget as HTMLDivElement).style.borderColor = "var(--arcana-card-border)";
                  }}
                >
                  <span style={{
                    width: 10,
                    height: 10,
                    borderRadius: "50%",
                    background: dot.color,
                    flexShrink: 0,
                    boxShadow: agent.status === "active" ? `0 0 6px ${dot.color}80` : "none",
                  }} />
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{
                      fontSize: 14,
                      fontWeight: 600,
                      color: "var(--arcana-text)",
                      marginBottom: 2,
                    }}>
                      {agent.name}
                    </div>
                    <div style={{ fontSize: 12, color: "var(--arcana-text-muted)" }}>
                      {model ?? "—"} &middot; {dot.label}
                    </div>
                  </div>
                  <div style={{ display: "flex", gap: 4, flexWrap: "wrap", maxWidth: 260 }}>
                    {skills.slice(0, 3).map((s) => (
                      <Label key={s} isCompact color="grey">{s}</Label>
                    ))}
                    {skills.length > 3 && <Label isCompact color="grey">+{skills.length - 3}</Label>}
                  </div>
                  <ExternalLinkAltIcon style={{ color: "var(--arcana-text-muted)", fontSize: 12 }} />
                </div>
              );
            })}
            {agents.length > 10 && (
              <div style={{ textAlign: "center", padding: 8 }}>
                <Button variant="link" size="sm" onClick={() => navigate("/agents")}>
                  View all {agents.length} agents
                </Button>
              </div>
            )}
          </div>
        )}

        {/* Quick actions */}
        {isAtLeast("developer") && (
          <div style={{ marginTop: 28 }}>
            <div style={{
              fontSize: 11,
              fontWeight: 700,
              textTransform: "uppercase",
              letterSpacing: "1.2px",
              color: "var(--arcana-text-muted)",
              marginBottom: 12,
            }}>
              Quick actions
            </div>
            <div style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
              {(isTechUser ? [
                { label: "Deploy agent", path: "/build" },
                { label: "Marketplace", path: "/marketplace" },
                { label: "Guardrails", path: "/guardrails" },
                { label: "Usage & costs", path: "/finops" },
              ] : [
                { label: "Browse agents", path: "/marketplace" },
                { label: "Chat with agent", path: "/chat" },
              ]).map((action) => (
                <button
                  key={action.path}
                  type="button"
                  onClick={() => navigate(action.path)}
                  style={{
                    padding: "9px 18px",
                    borderRadius: 8,
                    border: "1px solid var(--arcana-card-border)",
                    background: "var(--arcana-card-bg)",
                    color: "var(--arcana-text-secondary)",
                    fontSize: 13,
                    fontWeight: 500,
                    cursor: "pointer",
                    transition: "all 0.15s",
                  }}
                  onMouseEnter={(e) => {
                    (e.currentTarget as HTMLButtonElement).style.borderColor = "rgba(91,141,239,0.3)";
                    (e.currentTarget as HTMLButtonElement).style.color = "var(--arcana-text)";
                  }}
                  onMouseLeave={(e) => {
                    (e.currentTarget as HTMLButtonElement).style.borderColor = "var(--arcana-card-border)";
                    (e.currentTarget as HTMLButtonElement).style.color = "var(--arcana-text-secondary)";
                  }}
                >
                  {action.label}
                </button>
              ))}
            </div>
          </div>
        )}
      </PageSection>
    </>
  );
};
