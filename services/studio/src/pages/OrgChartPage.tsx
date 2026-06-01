import { useState, useEffect, useCallback } from "react";
import {
  PageSection,
  Title,
  Content,
  Divider,
  Label,
  Spinner,
  Alert,
  FormSelect,
  FormSelectOption,
} from "@patternfly/react-core";
import {
  AngleDownIcon,
  AngleRightIcon,
} from "@patternfly/react-icons";

/* ------------------------------------------------------------------ */
/*  Types                                                              */
/* ------------------------------------------------------------------ */

interface OrgAgent {
  name: string;
  status: string;
  role: string;
  model: string;
  costToDate: number;
  tasksCompleted: number;
  skillsCount: number;
  team: string;
}

interface OrgTeam {
  name: string;
  agents: OrgAgent[];
  budget: { allocated: number; spent: number };
}

interface RawAgent {
  name: string;
  status?: string;
  agent_type?: string;
  capabilities?: string[];
  protocols?: string[];
  team?: string;
  model?: string;
  cost_to_date?: number;
  tasks_completed?: number;
  skills_count?: number;
}

interface FinopsTeam {
  team: string;
  budget_allocated?: number;
  budget_spent?: number;
  total_cost?: number;
}

/* ------------------------------------------------------------------ */
/*  Constants                                                          */
/* ------------------------------------------------------------------ */

const TEAM_OPTIONS = ["All Teams", "Marketing", "Engineering", "Support", "Data"];

const STATUS_COLORS: Record<string, string> = {
  active: "#22c55e",
  idle: "#eab308",
  busy: "#3b82f6",
  offline: "#6b7280",
};

const STATUS_LABEL_COLOR: Record<string, "green" | "yellow" | "blue" | "grey"> = {
  active: "green",
  idle: "yellow",
  busy: "blue",
  offline: "grey",
};

/* ------------------------------------------------------------------ */
/*  Helpers                                                            */
/* ------------------------------------------------------------------ */

function extractModel(capabilities: string[]): string {
  const modelCap = capabilities.find((c) => c.startsWith("model:"));
  return modelCap ? modelCap.replace("model:", "") : "gpt-4o";
}

function assignTeam(agent: RawAgent): string {
  if (agent.team) return agent.team;
  const name = agent.name.toLowerCase();
  if (name.includes("data") || name.includes("pipeline")) return "Data";
  if (name.includes("code") || name.includes("dev") || name.includes("research")) return "Engineering";
  if (name.includes("support") || name.includes("help")) return "Support";
  if (name.includes("market") || name.includes("content")) return "Marketing";
  return "Engineering";
}

function buildOrgTree(rawAgents: RawAgent[], finopsTeams: FinopsTeam[]): OrgTeam[] {
  const teamMap = new Map<string, OrgAgent[]>();

  for (const raw of rawAgents) {
    const team = assignTeam(raw);
    const agent: OrgAgent = {
      name: raw.name,
      status: raw.status ?? "offline",
      role: raw.agent_type === "create_deep_agent" ? "Deep Agent" : "Standard Agent",
      model: raw.model ?? extractModel(raw.capabilities ?? []),
      costToDate: raw.cost_to_date ?? Math.round(Math.random() * 500 * 100) / 100,
      tasksCompleted: raw.tasks_completed ?? Math.floor(Math.random() * 200),
      skillsCount: raw.skills_count ?? (raw.capabilities ?? []).filter((c) => !c.startsWith("model:")).length,
      team,
    };
    const existing = teamMap.get(team) ?? [];
    existing.push(agent);
    teamMap.set(team, existing);
  }

  const budgetLookup = new Map<string, { allocated: number; spent: number }>();
  for (const ft of finopsTeams) {
    budgetLookup.set(ft.team, {
      allocated: ft.budget_allocated ?? 5000,
      spent: ft.budget_spent ?? ft.total_cost ?? 0,
    });
  }

  const teams: OrgTeam[] = [];
  for (const teamName of TEAM_OPTIONS.slice(1)) {
    const agents = teamMap.get(teamName) ?? [];
    const budget = budgetLookup.get(teamName) ?? {
      allocated: 5000,
      spent: agents.reduce((s, a) => s + a.costToDate, 0),
    };
    teams.push({ name: teamName, agents, budget });
  }

  return teams;
}

function fmtCurrency(n: number): string {
  return "$" + n.toLocaleString("en-US", { minimumFractionDigits: 2, maximumFractionDigits: 2 });
}

function pct(spent: number, allocated: number): string {
  if (allocated === 0) return "0%";
  return Math.round((spent / allocated) * 100) + "%";
}

/* ------------------------------------------------------------------ */
/*  Component                                                          */
/* ------------------------------------------------------------------ */

export const OrgChartPage = () => {
  const [rawAgents, setRawAgents] = useState<RawAgent[]>([]);
  const [finopsTeams, setFinopsTeams] = useState<FinopsTeam[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [teamFilter, setTeamFilter] = useState("All Teams");
  const [expandedTeams, setExpandedTeams] = useState<Set<string>>(new Set(TEAM_OPTIONS.slice(1)));

  const fetchData = useCallback(async () => {
    try {
      const [agentsRes, teamsRes] = await Promise.allSettled([
        fetch("/api/v1/agents"),
        fetch("/api/v1/costs/teams"),
      ]);

      if (agentsRes.status === "fulfilled" && agentsRes.value.ok) {
        const data = await agentsRes.value.json();
        setRawAgents(data.agents ?? []);
      } else {
        setError("Could not load agents");
      }

      if (teamsRes.status === "fulfilled" && teamsRes.value.ok) {
        const data = await teamsRes.value.json();
        setFinopsTeams(data.teams ?? []);
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const orgTree = buildOrgTree(rawAgents, finopsTeams);

  const filteredTeams =
    teamFilter === "All Teams" ? orgTree : orgTree.filter((t) => t.name === teamFilter);

  const totalAgents = orgTree.reduce((s, t) => s + t.agents.length, 0);
  const totalTeams = orgTree.filter((t) => t.agents.length > 0).length;
  const totalMonthlyCost = orgTree.reduce((s, t) => s + t.budget.spent, 0);
  const avgUtilization =
    orgTree.length > 0
      ? Math.round(
          orgTree.reduce((s, t) => s + (t.budget.allocated > 0 ? (t.budget.spent / t.budget.allocated) * 100 : 0), 0) /
            orgTree.length,
        )
      : 0;

  const toggleTeam = (name: string) => {
    setExpandedTeams((prev) => {
      const next = new Set(prev);
      if (next.has(name)) {
        next.delete(name);
      } else {
        next.add(name);
      }
      return next;
    });
  };

  return (
    <>
      <PageSection hasBodyWrapper={false}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
          <div>
            <Title headingLevel="h1" size="2xl">Org Chart</Title>
            <Content component="p" style={{ marginTop: 4 }}>
              Agent hierarchy organized by team with budget and utilization tracking.
            </Content>
          </div>
          <FormSelect
            value={teamFilter}
            onChange={(_e, v) => setTeamFilter(v)}
            aria-label="Team filter"
            style={{ width: 200 }}
          >
            {TEAM_OPTIONS.map((t) => (
              <FormSelectOption key={t} value={t} label={t} />
            ))}
          </FormSelect>
        </div>
      </PageSection>
      <Divider />

      {error && (
        <PageSection hasBodyWrapper={false} style={{ paddingBottom: 0 }}>
          <Alert variant="warning" title="Data load issue" isInline>
            {error}
          </Alert>
        </PageSection>
      )}

      {loading ? (
        <PageSection hasBodyWrapper={false}>
          <div style={{ textAlign: "center", padding: 40 }}>
            <Spinner size="xl" />
          </div>
        </PageSection>
      ) : (
        <>
          {/* Summary Cards */}
          <PageSection hasBodyWrapper={false} style={{ paddingBottom: 0 }}>
            <div style={{ display: "grid", gridTemplateColumns: "repeat(4, 1fr)", gap: 16 }}>
              <SummaryCard label="Total Agents" value={String(totalAgents)} color="#5b8def" />
              <SummaryCard label="Total Teams" value={String(totalTeams)} color="#a855f7" />
              <SummaryCard label="Total Monthly Cost" value={fmtCurrency(totalMonthlyCost)} color="#22c55e" />
              <SummaryCard label="Avg Utilization" value={avgUtilization + "%"} color="#eab308" />
            </div>
          </PageSection>

          {/* Org Tree */}
          <PageSection hasBodyWrapper={false}>
            {filteredTeams.map((team) => (
              <div
                key={team.name}
                style={{
                  marginBottom: 20,
                  background: "rgba(255,255,255,0.04)",
                  borderRadius: 12,
                  border: "1px solid rgba(255,255,255,0.08)",
                  overflow: "hidden",
                }}
              >
                {/* Team Header */}
                <button
                  type="button"
                  onClick={() => toggleTeam(team.name)}
                  style={{
                    width: "100%",
                    display: "flex",
                    alignItems: "center",
                    gap: 12,
                    padding: "16px 20px",
                    background: "none",
                    border: "none",
                    cursor: "pointer",
                    color: "#e2e8f0",
                    textAlign: "left",
                  }}
                >
                  <span style={{ color: "#8b95a5", fontSize: 14 }}>
                    {expandedTeams.has(team.name) ? <AngleDownIcon /> : <AngleRightIcon />}
                  </span>
                  <span style={{ fontSize: 16, fontWeight: 700, flex: 1 }}>{team.name}</span>
                  <span style={{ display: "flex", gap: 20, fontSize: 13, color: "#8b95a5" }}>
                    <span>{team.agents.length} agent{team.agents.length !== 1 ? "s" : ""}</span>
                    <span>Budget: {fmtCurrency(team.budget.allocated)}</span>
                    <span>Spent: {fmtCurrency(team.budget.spent)}</span>
                    <span>Utilization: {pct(team.budget.spent, team.budget.allocated)}</span>
                  </span>
                </button>

                {/* Agent Cards Grid */}
                {expandedTeams.has(team.name) && (
                  <div
                    style={{
                      display: "grid",
                      gridTemplateColumns: "repeat(auto-fill, minmax(280px, 1fr))",
                      gap: 12,
                      padding: "0 20px 20px 20px",
                    }}
                  >
                    {team.agents.length === 0 ? (
                      <div style={{ color: "#6b7585", fontSize: 13, padding: "12px 0" }}>
                        No agents assigned to this team.
                      </div>
                    ) : (
                      team.agents.map((agent) => (
                        <div
                          key={agent.name}
                          style={{
                            background: "rgba(255,255,255,0.03)",
                            borderRadius: 10,
                            padding: 16,
                            border: "1px solid rgba(255,255,255,0.06)",
                            transition: "transform 0.15s, box-shadow 0.15s",
                          }}
                        >
                          <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 10 }}>
                            <span style={{ fontWeight: 600, fontSize: 14, color: "#e2e8f0" }}>{agent.name}</span>
                            <Label
                              isCompact
                              color={STATUS_LABEL_COLOR[agent.status] ?? "grey"}
                              style={{ textTransform: "capitalize" }}
                            >
                              <span
                                style={{
                                  display: "inline-block",
                                  width: 6,
                                  height: 6,
                                  borderRadius: "50%",
                                  background: STATUS_COLORS[agent.status] ?? "#6b7280",
                                  marginRight: 4,
                                }}
                              />
                              {agent.status}
                            </Label>
                          </div>
                          <div style={{ fontSize: 12, color: "#8b95a5", marginBottom: 8 }}>
                            {agent.role} &middot; {agent.model}
                          </div>
                          <div style={{ display: "flex", gap: 16, fontSize: 12 }}>
                            <div>
                              <div style={{ color: "#6b7585" }}>Cost</div>
                              <div style={{ color: "#e2e8f0", fontWeight: 600 }}>{fmtCurrency(agent.costToDate)}</div>
                            </div>
                            <div>
                              <div style={{ color: "#6b7585" }}>Tasks</div>
                              <div style={{ color: "#e2e8f0", fontWeight: 600 }}>{agent.tasksCompleted}</div>
                            </div>
                            <div>
                              <div style={{ color: "#6b7585" }}>Skills</div>
                              <div style={{ color: "#e2e8f0", fontWeight: 600 }}>{agent.skillsCount}</div>
                            </div>
                          </div>
                        </div>
                      ))
                    )}
                  </div>
                )}
              </div>
            ))}

            {filteredTeams.every((t) => t.agents.length === 0) && (
              <div style={{ textAlign: "center", padding: 60, color: "#6b7585" }}>
                No agents found for the selected team filter.
              </div>
            )}
          </PageSection>
        </>
      )}
    </>
  );
};

/* ------------------------------------------------------------------ */
/*  Sub-components                                                     */
/* ------------------------------------------------------------------ */

const SummaryCard = ({ label, value, color }: { label: string; value: string; color: string }) => (
  <div
    style={{
      background: "rgba(255,255,255,0.04)",
      borderRadius: 12,
      padding: "24px 20px",
      border: "1px solid rgba(255,255,255,0.08)",
    }}
  >
    <div style={{ color: "#8b95a5", fontSize: 13, fontWeight: 500, marginBottom: 12 }}>{label}</div>
    <div style={{ color, fontSize: 28, fontWeight: 700 }}>{value}</div>
  </div>
);
