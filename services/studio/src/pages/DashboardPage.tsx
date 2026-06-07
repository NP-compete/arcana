import { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import {
  PageSection,
  Spinner,
  Alert,
} from "@patternfly/react-core";
import { useHealth } from "../hooks/useHealth";
import { useAgentsHealthOverview } from "../hooks/useAgentHealth";
import { useAuth } from "../auth/AuthContext";

export const DashboardPage = () => {
  const navigate = useNavigate();
  const { health, error, loading } = useHealth(5000);
  const { overview: agentHealth } = useAgentsHealthOverview();
  const { user, isAtLeast } = useAuth();
  const [agentCount, setAgentCount] = useState<{ total: number } | null>(null);
  const [skillCount, setSkillCount] = useState(0);
  const [mcpCount, setMcpCount] = useState(0);

  useEffect(() => {
    fetch("/api/v1/agents")
      .then((r) => r.json())
      .then((d) => setAgentCount({ total: d.total ?? 0 }))
      .catch(() => {});
    fetch("/api/v1/skills")
      .then((r) => r.json())
      .then((d) => setSkillCount(d.skills?.length ?? 0))
      .catch(() => {});
    fetch("/api/v1/mcp")
      .then((r) => r.json())
      .then((d) => setMcpCount(d.servers?.length ?? 0))
      .catch(() => {});
  }, []);

  const totalServices = health?.services.length ?? 0;
  const healthyServices = health?.services.filter((s) => s.status === "healthy").length ?? 0;

  return (
    <>
      {error && (
        <PageSection hasBodyWrapper={false} style={{ paddingBottom: 0 }}>
          <Alert variant="warning" title="API unreachable" isInline>
            Cannot reach arcana-api. Run <code>make dev</code> to start.
          </Alert>
        </PageSection>
      )}

      {/* Hero Section */}
      <PageSection hasBodyWrapper={false} style={{ padding: 0 }}>
        <div
          style={{
            background: "linear-gradient(135deg, #1a1d2e 0%, #0d2137 50%, #1a1035 100%)",
            padding: "64px 48px",
            position: "relative",
            overflow: "hidden",
          }}
        >
          <div
            style={{
              position: "absolute",
              top: -80,
              right: -80,
              width: 400,
              height: 400,
              borderRadius: "50%",
              background: "radial-gradient(circle, rgba(91,141,239,0.08) 0%, transparent 70%)",
            }}
          />
          <div
            style={{
              position: "absolute",
              bottom: -60,
              left: "30%",
              width: 300,
              height: 300,
              borderRadius: "50%",
              background: "radial-gradient(circle, rgba(168,85,247,0.06) 0%, transparent 70%)",
            }}
          />

          <div style={{ position: "relative", maxWidth: 720 }}>
            <h1
              style={{
                fontSize: 36,
                fontWeight: 700,
                color: "#fff",
                margin: 0,
                lineHeight: 1.2,
                letterSpacing: "-0.5px",
              }}
            >
              Build agents your teams can trust
            </h1>
            <p
              style={{
                fontSize: 16,
                color: "#8b95a5",
                marginTop: 16,
                lineHeight: 1.7,
                maxWidth: 640,
              }}
            >
              Arcana Studio helps {user?.roles?.[0] === "user" ? "you use" : "you compose, publish, and run"} AI
              agents using skills, sub-agents, MCPs, and shared artifacts—so teams
              collaborate instead of duplicating work.
            </p>
            <div style={{ display: "flex", gap: 12, marginTop: 28 }}>
              {isAtLeast("developer") && (
                <button
                  type="button"
                  onClick={() => navigate("/agents")}
                  style={{
                    padding: "12px 24px",
                    borderRadius: 8,
                    border: "none",
                    background: "linear-gradient(135deg, #5b8def 0%, #4a6cf7 100%)",
                    color: "#fff",
                    fontSize: 14,
                    fontWeight: 600,
                    cursor: "pointer",
                    transition: "transform 0.15s, box-shadow 0.15s",
                  }}
                >
                  Start building
                </button>
              )}
              <button
                type="button"
                onClick={() => navigate("/agents")}
                style={{
                  padding: "12px 24px",
                  borderRadius: 8,
                  border: "1px solid rgba(255,255,255,0.15)",
                  background: "rgba(255,255,255,0.05)",
                  color: "#c5cdd8",
                  fontSize: 14,
                  fontWeight: 600,
                  cursor: "pointer",
                  transition: "background 0.15s",
                }}
              >
                Go to Agents
              </button>
              {isAtLeast("admin") && (
                <button
                  type="button"
                  onClick={() => navigate("/settings")}
                  style={{
                    padding: "12px 24px",
                    borderRadius: 8,
                    border: "1px solid rgba(255,255,255,0.15)",
                    background: "rgba(255,255,255,0.05)",
                    color: "#c5cdd8",
                    fontSize: 14,
                    fontWeight: 600,
                    cursor: "pointer",
                    transition: "background 0.15s",
                  }}
                >
                  View operations
                </button>
              )}
            </div>
          </div>
        </div>
      </PageSection>

      {/* Statistics Section */}
      <PageSection hasBodyWrapper={false}>
        <h2
          style={{
            fontSize: 20,
            fontWeight: 600,
            color: "#e2e8f0",
            marginBottom: 6,
          }}
        >
          Arcana Statistics
        </h2>
        <p style={{ color: "#6b7585", fontSize: 14, marginBottom: 24 }}>
          Platform-wide usage across all teams and organizations.
        </p>

        <div
          style={{
            display: "grid",
            gridTemplateColumns: "repeat(5, 1fr)",
            gap: 16,
          }}
        >
          {/* Agents Deployed */}
          <div
            style={{
              background: "rgba(255,255,255,0.04)",
              borderRadius: 12,
              padding: "24px 20px",
              border: "1px solid rgba(255,255,255,0.08)",
            }}
          >
            <div style={{ color: "#8b95a5", fontSize: 13, fontWeight: 500, marginBottom: 16 }}>
              Agents Deployed
            </div>
            <div style={{ display: "flex", gap: 24 }}>
              <div>
                <div style={{ color: "#6b7585", fontSize: 11, marginBottom: 4 }}>Total</div>
                <div style={{ color: "#fff", fontSize: 28, fontWeight: 700 }}>
                  {loading ? <Spinner size="md" /> : (agentHealth?.total_agents ?? agentCount?.total ?? "–")}
                </div>
              </div>
              {agentHealth && (
                <>
                  <div>
                    <div style={{ color: "#6b7585", fontSize: 11, marginBottom: 4 }}>Healthy</div>
                    <div style={{ color: "#22c55e", fontSize: 28, fontWeight: 700 }}>
                      {agentHealth.healthy_agents}
                    </div>
                  </div>
                  {agentHealth.total_restarts > 0 && (
                    <div>
                      <div style={{ color: "#6b7585", fontSize: 11, marginBottom: 4 }}>Restarts</div>
                      <div style={{ color: "#f59e0b", fontSize: 28, fontWeight: 700 }}>
                        {agentHealth.total_restarts}
                      </div>
                    </div>
                  )}
                </>
              )}
            </div>
          </div>

          {/* Skills Created */}
          <div
            style={{
              background: "rgba(255,255,255,0.04)",
              borderRadius: 12,
              padding: "24px 20px",
              border: "1px solid rgba(255,255,255,0.08)",
            }}
          >
            <div style={{ color: "#8b95a5", fontSize: 13, fontWeight: 500, marginBottom: 16 }}>
              Skills Created
            </div>
            <div>
              <div style={{ color: "#6b7585", fontSize: 11, marginBottom: 4 }}>Registered</div>
              <div style={{ color: "#fff", fontSize: 28, fontWeight: 700 }}>
                {skillCount || "–"}
              </div>
            </div>
          </div>

          {/* MCP Integrations */}
          <div
            style={{
              background: "rgba(255,255,255,0.04)",
              borderRadius: 12,
              padding: "24px 20px",
              border: "1px solid rgba(255,255,255,0.08)",
            }}
          >
            <div style={{ color: "#8b95a5", fontSize: 13, fontWeight: 500, marginBottom: 16 }}>
              MCP Integrations
            </div>
            <div style={{ color: "#fff", fontSize: 28, fontWeight: 700 }}>
              {mcpCount || "–"}
            </div>
          </div>

          {/* Services */}
          <div
            style={{
              background: "rgba(255,255,255,0.04)",
              borderRadius: 12,
              padding: "24px 20px",
              border: "1px solid rgba(255,255,255,0.08)",
            }}
          >
            <div style={{ color: "#8b95a5", fontSize: 13, fontWeight: 500, marginBottom: 16 }}>
              Services
            </div>
            <div style={{ color: "#fff", fontSize: 28, fontWeight: 700 }}>
              {loading ? <Spinner size="md" /> : `${healthyServices}/${totalServices}`}
            </div>
            <div
              style={{
                marginTop: 8,
                height: 4,
                borderRadius: 2,
                background: "rgba(255,255,255,0.08)",
                overflow: "hidden",
              }}
            >
              <div
                style={{
                  height: "100%",
                  width: totalServices > 0 ? `${(healthyServices / totalServices) * 100}%` : "0%",
                  background: healthyServices === totalServices ? "#22c55e" : "#f59e0b",
                  borderRadius: 2,
                  transition: "width 0.3s",
                }}
              />
            </div>
          </div>

          {/* Planes */}
          <div
            style={{
              background: "rgba(255,255,255,0.04)",
              borderRadius: 12,
              padding: "24px 20px",
              border: "1px solid rgba(255,255,255,0.08)",
            }}
          >
            <div style={{ color: "#8b95a5", fontSize: 13, fontWeight: 500, marginBottom: 16 }}>
              Active Planes
            </div>
            <div style={{ color: "#fff", fontSize: 28, fontWeight: 700 }}>8</div>
          </div>
        </div>
      </PageSection>

      {/* What you can do here */}
      <PageSection hasBodyWrapper={false}>
        <h2
          style={{
            fontSize: 20,
            fontWeight: 600,
            color: "#e2e8f0",
            marginBottom: 6,
          }}
        >
          What you can do here
        </h2>
        <p style={{ color: "#6b7585", fontSize: 14, marginBottom: 24 }}>
          Three ways to get value from Arcana today.
        </p>

        <div
          style={{
            display: "grid",
            gridTemplateColumns: "repeat(3, 1fr)",
            gap: 20,
          }}
        >
          {/* BUILD card */}
          <div
            style={{
              background: "rgba(255,255,255,0.04)",
              borderRadius: 12,
              padding: 28,
              border: "1px solid rgba(255,255,255,0.08)",
              display: "flex",
              flexDirection: "column",
            }}
          >
            <span
              style={{
                fontSize: 11,
                fontWeight: 700,
                letterSpacing: "1.5px",
                color: "#5b8def",
                marginBottom: 12,
              }}
            >
              BUILD
            </span>
            <h3 style={{ fontSize: 18, fontWeight: 600, color: "#fff", margin: "0 0 12px" }}>
              Compose agents
            </h3>
            <p style={{ color: "#8b95a5", fontSize: 14, lineHeight: 1.6, flex: 1 }}>
              Pick skills from the ecosystem, attach sub-agents, and wire MCPs with
              guardrails—then package a version your stakeholders can review.
            </p>
            <button
              type="button"
              onClick={() => navigate("/agents")}
              style={{
                marginTop: 16,
                padding: 0,
                border: "none",
                background: "none",
                color: "#5b8def",
                fontSize: 14,
                fontWeight: 600,
                cursor: "pointer",
                textAlign: "left",
              }}
            >
              Open builder →
            </button>
          </div>

          {/* DISCOVER card */}
          <div
            style={{
              background: "rgba(255,255,255,0.04)",
              borderRadius: 12,
              padding: 28,
              border: "1px solid rgba(255,255,255,0.08)",
              display: "flex",
              flexDirection: "column",
            }}
          >
            <span
              style={{
                fontSize: 11,
                fontWeight: 700,
                letterSpacing: "1.5px",
                color: "#a855f7",
                marginBottom: 12,
              }}
            >
              DISCOVER
            </span>
            <h3 style={{ fontSize: 18, fontWeight: 600, color: "#fff", margin: "0 0 12px" }}>
              Find what already exists
            </h3>
            <p style={{ color: "#8b95a5", fontSize: 14, lineHeight: 1.6, flex: 1 }}>
              Search the registry for skills, agents, MCPs, and connectors. See who owns them
              and how many teams reuse them before you build from scratch.
            </p>
            <button
              type="button"
              onClick={() => navigate("/skills")}
              style={{
                marginTop: 16,
                padding: 0,
                border: "none",
                background: "none",
                color: "#a855f7",
                fontSize: 14,
                fontWeight: 600,
                cursor: "pointer",
                textAlign: "left",
              }}
            >
              Explore registry →
            </button>
          </div>

          {/* OPERATE card */}
          <div
            style={{
              background: "rgba(255,255,255,0.04)",
              borderRadius: 12,
              padding: 28,
              border: "1px solid rgba(255,255,255,0.08)",
              display: "flex",
              flexDirection: "column",
            }}
          >
            <span
              style={{
                fontSize: 11,
                fontWeight: 700,
                letterSpacing: "1.5px",
                color: "#22c55e",
                marginBottom: 12,
              }}
            >
              OPERATE
            </span>
            <h3 style={{ fontSize: 18, fontWeight: 600, color: "#fff", margin: "0 0 12px" }}>
              Run with confidence
            </h3>
            <p style={{ color: "#8b95a5", fontSize: 14, lineHeight: 1.6, flex: 1 }}>
              Monitor deployments, health, and recent runs from one operations view—prototype
              for day-two ownership inside the enterprise.
            </p>
            <button
              type="button"
              onClick={() => navigate("/settings")}
              style={{
                marginTop: 16,
                padding: 0,
                border: "none",
                background: "none",
                color: "#22c55e",
                fontSize: 14,
                fontWeight: 600,
                cursor: "pointer",
                textAlign: "left",
              }}
            >
              Go to operations →
            </button>
          </div>
        </div>
      </PageSection>
    </>
  );
};
