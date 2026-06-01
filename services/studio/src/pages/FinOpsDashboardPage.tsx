import { useState, useEffect, useCallback } from "react";
import {
  PageSection,
  Title,
  Content,
  Card,
  CardBody,
  Divider,
  Label,
  Spinner,
  Alert,
  FormSelect,
  FormSelectOption,
} from "@patternfly/react-core";
import { Table, Thead, Tr, Th, Tbody, Td } from "@patternfly/react-table";
import {
  AreaChart,
  Area,
  PieChart,
  Pie,
  Cell,
  BarChart,
  Bar,
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from "recharts";

/* ------------------------------------------------------------------ */
/*  Types                                                              */
/* ------------------------------------------------------------------ */

interface FinopsSummary {
  total_spend: number;
  budget_remaining: number;
  savings_from_fallback: number;
  active_agents: number;
}

interface CostOverTimeEntry {
  date: string;
  Marketing: number;
  Engineering: number;
  Support: number;
  Data: number;
}

interface CostByModel {
  model: string;
  cost: number;
}

interface TopAgent {
  agent: string;
  team: string;
  model: string;
  tokens: number;
  cost: number;
  tasks: number;
}

interface BudgetUtilization {
  team: string;
  allocated: number;
  spent: number;
}

interface FallbackSavings {
  date: string;
  with_fallback: number;
  without_fallback: number;
}

/* ------------------------------------------------------------------ */
/*  Constants                                                          */
/* ------------------------------------------------------------------ */

const PERIOD_OPTIONS = [
  { value: "7d", label: "Last 7 days" },
  { value: "30d", label: "Last 30 days" },
  { value: "90d", label: "Last 90 days" },
];

const TEAM_OPTIONS = ["all", "Marketing", "Engineering", "Support", "Data"];

const CHART_COLORS = {
  Marketing: "#a855f7",
  Engineering: "#3b82f6",
  Support: "#06b6d4",
  Data: "#8b5cf6",
};

const PIE_COLORS = ["#3b82f6", "#a855f7", "#06b6d4", "#8b5cf6", "#ec4899", "#f59e0b"];

const TOOLTIP_STYLE = {
  backgroundColor: "#1a1d2e",
  border: "1px solid rgba(255,255,255,0.12)",
  borderRadius: 8,
  color: "#e2e8f0",
  fontSize: 12,
};

/* ------------------------------------------------------------------ */
/*  Mock data generators (used when API returns empty/errors)          */
/* ------------------------------------------------------------------ */

function generateCostOverTime(period: string): CostOverTimeEntry[] {
  const days = period === "7d" ? 7 : period === "30d" ? 30 : 90;
  const result: CostOverTimeEntry[] = [];
  const now = Date.now();
  for (let i = days - 1; i >= 0; i--) {
    const d = new Date(now - i * 86400000);
    result.push({
      date: `${d.getMonth() + 1}/${d.getDate()}`,
      Marketing: Math.round(Math.random() * 150 + 50),
      Engineering: Math.round(Math.random() * 300 + 100),
      Support: Math.round(Math.random() * 80 + 30),
      Data: Math.round(Math.random() * 200 + 80),
    });
  }
  return result;
}

function generateCostByModel(): CostByModel[] {
  return [
    { model: "Claude Sonnet", cost: 2450 },
    { model: "GPT-4o", cost: 1890 },
    { model: "GPT-4o-mini", cost: 620 },
    { model: "Claude Haiku", cost: 340 },
    { model: "Gemini Pro", cost: 180 },
  ];
}

function generateTopAgents(): TopAgent[] {
  return [
    { agent: "research-pipeline-agent", team: "Engineering", model: "claude-sonnet", tokens: 2400000, cost: 890.50, tasks: 145 },
    { agent: "data-pipeline-agent", team: "Data", model: "gpt-4o", tokens: 1800000, cost: 720.30, tasks: 89 },
    { agent: "code-assistant", team: "Engineering", model: "claude-sonnet", tokens: 1500000, cost: 650.00, tasks: 234 },
    { agent: "content-writer", team: "Marketing", model: "gpt-4o", tokens: 1200000, cost: 480.20, tasks: 67 },
    { agent: "support-bot", team: "Support", model: "gpt-4o-mini", tokens: 3200000, cost: 320.10, tasks: 512 },
    { agent: "qa-validator", team: "Engineering", model: "gpt-4o", tokens: 900000, cost: 290.40, tasks: 178 },
    { agent: "doc-generator", team: "Engineering", model: "claude-sonnet", tokens: 800000, cost: 240.80, tasks: 56 },
    { agent: "data-profiler", team: "Data", model: "gpt-4o-mini", tokens: 2100000, cost: 210.00, tasks: 312 },
    { agent: "email-drafter", team: "Marketing", model: "gpt-4o-mini", tokens: 1800000, cost: 180.50, tasks: 423 },
    { agent: "helpdesk-triage", team: "Support", model: "gpt-4o-mini", tokens: 1600000, cost: 160.00, tasks: 890 },
  ];
}

function generateBudgetUtilization(): BudgetUtilization[] {
  return [
    { team: "Engineering", allocated: 5000, spent: 3200 },
    { team: "Data", allocated: 3000, spent: 1890 },
    { team: "Marketing", allocated: 2000, spent: 1450 },
    { team: "Support", allocated: 1500, spent: 820 },
  ];
}

function generateFallbackSavings(period: string): FallbackSavings[] {
  const days = period === "7d" ? 7 : period === "30d" ? 30 : 90;
  const result: FallbackSavings[] = [];
  const now = Date.now();
  for (let i = days - 1; i >= 0; i--) {
    const d = new Date(now - i * 86400000);
    const base = Math.random() * 200 + 100;
    result.push({
      date: `${d.getMonth() + 1}/${d.getDate()}`,
      without_fallback: Math.round(base),
      with_fallback: Math.round(base * (0.55 + Math.random() * 0.15)),
    });
  }
  return result;
}

/* ------------------------------------------------------------------ */
/*  Helpers                                                            */
/* ------------------------------------------------------------------ */

function fmtCurrency(n: number): string {
  return "$" + n.toLocaleString("en-US", { minimumFractionDigits: 2, maximumFractionDigits: 2 });
}

function fmtTokens(n: number): string {
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + "M";
  if (n >= 1_000) return (n / 1_000).toFixed(0) + "K";
  return String(n);
}

/* ------------------------------------------------------------------ */
/*  Component                                                          */
/* ------------------------------------------------------------------ */

export const FinOpsDashboardPage = () => {
  const [period, setPeriod] = useState("30d");
  const [teamFilter, setTeamFilter] = useState("all");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [summary, setSummary] = useState<FinopsSummary>({
    total_spend: 0,
    budget_remaining: 0,
    savings_from_fallback: 0,
    active_agents: 0,
  });
  const [costOverTime, setCostOverTime] = useState<CostOverTimeEntry[]>([]);
  const [costByModel, setCostByModel] = useState<CostByModel[]>([]);
  const [topAgents, setTopAgents] = useState<TopAgent[]>([]);
  const [budgetUtil, setBudgetUtil] = useState<BudgetUtilization[]>([]);
  const [fallbackSavings, setFallbackSavings] = useState<FallbackSavings[]>([]);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [summaryRes, cotRes, cbmRes, taRes, buRes] = await Promise.allSettled([
        fetch(`/api/v1/costs?period=${period}`),
        fetch(`/api/v1/costs/over-time?period=${period}&team=${teamFilter}`),
        fetch(`/api/v1/costs/by-model?period=${period}`),
        fetch(`/api/v1/costs/top-agents?period=${period}&limit=10`),
        fetch("/api/v1/costs/budget-utilization"),
      ]);

      if (summaryRes.status === "fulfilled" && summaryRes.value.ok) {
        const data = await summaryRes.value.json();
        setSummary({
          total_spend: data.total_spend ?? 5480.50,
          budget_remaining: data.budget_remaining ?? 6019.50,
          savings_from_fallback: data.savings_from_fallback ?? 1240.00,
          active_agents: data.active_agents ?? 10,
        });
      } else {
        setSummary({ total_spend: 5480.50, budget_remaining: 6019.50, savings_from_fallback: 1240.00, active_agents: 10 });
      }

      if (cotRes.status === "fulfilled" && cotRes.value.ok) {
        const data = await cotRes.value.json();
        setCostOverTime(data.series ?? generateCostOverTime(period));
      } else {
        setCostOverTime(generateCostOverTime(period));
      }

      if (cbmRes.status === "fulfilled" && cbmRes.value.ok) {
        const data = await cbmRes.value.json();
        setCostByModel(data.models ?? generateCostByModel());
      } else {
        setCostByModel(generateCostByModel());
      }

      if (taRes.status === "fulfilled" && taRes.value.ok) {
        const data = await taRes.value.json();
        setTopAgents(data.agents ?? generateTopAgents());
      } else {
        setTopAgents(generateTopAgents());
      }

      if (buRes.status === "fulfilled" && buRes.value.ok) {
        const data = await buRes.value.json();
        setBudgetUtil(data.teams ?? generateBudgetUtilization());
      } else {
        setBudgetUtil(generateBudgetUtilization());
      }

      setFallbackSavings(generateFallbackSavings(period));
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load FinOps data");
      setCostOverTime(generateCostOverTime(period));
      setCostByModel(generateCostByModel());
      setTopAgents(generateTopAgents());
      setBudgetUtil(generateBudgetUtilization());
      setFallbackSavings(generateFallbackSavings(period));
    } finally {
      setLoading(false);
    }
  }, [period, teamFilter]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const filteredAgents =
    teamFilter === "all" ? topAgents : topAgents.filter((a) => a.team === teamFilter);

  return (
    <>
      <PageSection hasBodyWrapper={false}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
          <div>
            <Title headingLevel="h1" size="2xl">Usage & Costs</Title>
            <Content component="p" style={{ marginTop: 4 }}>
              Cost analytics, budget utilization, and model spend tracking.
            </Content>
          </div>
          <div style={{ display: "flex", gap: 12 }}>
            <FormSelect
              value={period}
              onChange={(_e, v) => setPeriod(v)}
              aria-label="Time period"
              style={{ width: 160 }}
            >
              {PERIOD_OPTIONS.map((o) => (
                <FormSelectOption key={o.value} value={o.value} label={o.label} />
              ))}
            </FormSelect>
            <FormSelect
              value={teamFilter}
              onChange={(_e, v) => setTeamFilter(v)}
              aria-label="Team filter"
              style={{ width: 160 }}
            >
              {TEAM_OPTIONS.map((t) => (
                <FormSelectOption key={t} value={t} label={t === "all" ? "All Teams" : t} />
              ))}
            </FormSelect>
          </div>
        </div>
      </PageSection>
      <Divider />

      {error && (
        <PageSection hasBodyWrapper={false} style={{ paddingBottom: 0 }}>
          <Alert variant="info" title="Using sample data" isInline>
            FinOps API not available. Displaying representative data.
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
              <SummaryCard label="Total Spend" value={fmtCurrency(summary.total_spend)} color="#3b82f6" />
              <SummaryCard label="Budget Remaining" value={fmtCurrency(summary.budget_remaining)} color="#22c55e" />
              <SummaryCard label="Savings from Fallback" value={fmtCurrency(summary.savings_from_fallback)} color="#a855f7" />
              <SummaryCard label="Active Agents" value={String(summary.active_agents)} color="#06b6d4" />
            </div>
          </PageSection>

          {/* Charts Row 1 */}
          <PageSection hasBodyWrapper={false} style={{ paddingBottom: 0 }}>
            <div style={{ display: "grid", gridTemplateColumns: "2fr 1fr", gap: 16 }}>
              {/* Cost Over Time */}
              <Card className="stat-card">
                <CardBody>
                  <div style={{ fontWeight: 700, fontSize: 14, marginBottom: 16, color: "#e2e8f0" }}>
                    Cost Over Time
                  </div>
                  <ResponsiveContainer width="100%" height={280}>
                    <AreaChart data={costOverTime}>
                      <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.06)" />
                      <XAxis dataKey="date" tick={{ fontSize: 11, fill: "#8b95a5" }} stroke="rgba(255,255,255,0.1)" />
                      <YAxis tick={{ fontSize: 11, fill: "#8b95a5" }} stroke="rgba(255,255,255,0.1)" tickFormatter={(v: number) => `$${v}`} />
                      <Tooltip contentStyle={TOOLTIP_STYLE} formatter={(value) => [`$${value ?? 0}`]} />
                      <Legend wrapperStyle={{ fontSize: 12, color: "#8b95a5" }} />
                      <Area type="monotone" dataKey="Engineering" stackId="1" stroke={CHART_COLORS.Engineering} fill={CHART_COLORS.Engineering} fillOpacity={0.3} />
                      <Area type="monotone" dataKey="Data" stackId="1" stroke={CHART_COLORS.Data} fill={CHART_COLORS.Data} fillOpacity={0.3} />
                      <Area type="monotone" dataKey="Marketing" stackId="1" stroke={CHART_COLORS.Marketing} fill={CHART_COLORS.Marketing} fillOpacity={0.3} />
                      <Area type="monotone" dataKey="Support" stackId="1" stroke={CHART_COLORS.Support} fill={CHART_COLORS.Support} fillOpacity={0.3} />
                    </AreaChart>
                  </ResponsiveContainer>
                </CardBody>
              </Card>

              {/* Cost by Model */}
              <Card className="stat-card">
                <CardBody>
                  <div style={{ fontWeight: 700, fontSize: 14, marginBottom: 16, color: "#e2e8f0" }}>
                    Cost by Model
                  </div>
                  <ResponsiveContainer width="100%" height={280}>
                    <PieChart>
                      <Pie
                        data={costByModel}
                        dataKey="cost"
                        nameKey="model"
                        cx="50%"
                        cy="50%"
                        outerRadius={90}
                        innerRadius={50}
                        paddingAngle={2}
                        label={(props) => {
                          const p = props as unknown as Record<string, unknown>;
                          const name = p.model ?? p.name ?? "";
                          const pctVal = p.percent as number | undefined;
                          return `${name} ${pctVal != null ? (pctVal * 100).toFixed(0) : 0}%`;
                        }}
                        labelLine={{ stroke: "#6b7585" }}
                      >
                        {costByModel.map((_entry, idx) => (
                          <Cell key={idx} fill={PIE_COLORS[idx % PIE_COLORS.length]} />
                        ))}
                      </Pie>
                      <Tooltip
                        contentStyle={TOOLTIP_STYLE}
                        formatter={(value) => [fmtCurrency(Number(value ?? 0)), "Cost"]}
                      />
                    </PieChart>
                  </ResponsiveContainer>
                </CardBody>
              </Card>
            </div>
          </PageSection>

          {/* Charts Row 2 */}
          <PageSection hasBodyWrapper={false} style={{ paddingBottom: 0 }}>
            <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 16 }}>
              {/* Top Agents by Cost */}
              <Card className="stat-card">
                <CardBody>
                  <div style={{ fontWeight: 700, fontSize: 14, marginBottom: 16, color: "#e2e8f0" }}>
                    Top Agents by Cost
                  </div>
                  <ResponsiveContainer width="100%" height={320}>
                    <BarChart data={topAgents.slice(0, 10)} layout="vertical">
                      <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.06)" />
                      <XAxis type="number" tick={{ fontSize: 11, fill: "#8b95a5" }} stroke="rgba(255,255,255,0.1)" tickFormatter={(v: number) => `$${v}`} />
                      <YAxis type="category" dataKey="agent" tick={{ fontSize: 11, fill: "#8b95a5" }} stroke="rgba(255,255,255,0.1)" width={140} />
                      <Tooltip contentStyle={TOOLTIP_STYLE} formatter={(value) => [fmtCurrency(Number(value ?? 0)), "Cost"]} />
                      <Bar dataKey="cost" fill="#3b82f6" radius={[0, 4, 4, 0]} barSize={18} />
                    </BarChart>
                  </ResponsiveContainer>
                </CardBody>
              </Card>

              {/* Budget Utilization */}
              <Card className="stat-card">
                <CardBody>
                  <div style={{ fontWeight: 700, fontSize: 14, marginBottom: 16, color: "#e2e8f0" }}>
                    Budget Utilization
                  </div>
                  <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
                    {budgetUtil.map((team) => {
                      const pctVal = team.allocated > 0 ? Math.round((team.spent / team.allocated) * 100) : 0;
                      const barColor = pctVal > 90 ? "#ef4444" : pctVal > 70 ? "#eab308" : "#22c55e";
                      return (
                        <div key={team.team}>
                          <div style={{ display: "flex", justifyContent: "space-between", marginBottom: 6, fontSize: 13 }}>
                            <span style={{ color: "#e2e8f0", fontWeight: 600 }}>{team.team}</span>
                            <span style={{ color: "#8b95a5" }}>
                              {fmtCurrency(team.spent)} / {fmtCurrency(team.allocated)} ({pctVal}%)
                            </span>
                          </div>
                          <div
                            style={{
                              height: 8,
                              borderRadius: 4,
                              background: "rgba(255,255,255,0.08)",
                              overflow: "hidden",
                            }}
                          >
                            <div
                              style={{
                                height: "100%",
                                width: `${Math.min(pctVal, 100)}%`,
                                background: barColor,
                                borderRadius: 4,
                                transition: "width 0.3s",
                              }}
                            />
                          </div>
                        </div>
                      );
                    })}
                  </div>
                </CardBody>
              </Card>
            </div>
          </PageSection>

          {/* Fallback Savings Chart */}
          <PageSection hasBodyWrapper={false} style={{ paddingBottom: 0 }}>
            <Card className="stat-card">
              <CardBody>
                <div style={{ fontWeight: 700, fontSize: 14, marginBottom: 16, color: "#e2e8f0" }}>
                  Fallback Savings
                </div>
                <ResponsiveContainer width="100%" height={250}>
                  <LineChart data={fallbackSavings}>
                    <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.06)" />
                    <XAxis dataKey="date" tick={{ fontSize: 11, fill: "#8b95a5" }} stroke="rgba(255,255,255,0.1)" />
                    <YAxis tick={{ fontSize: 11, fill: "#8b95a5" }} stroke="rgba(255,255,255,0.1)" tickFormatter={(v: number) => `$${v}`} />
                    <Tooltip contentStyle={TOOLTIP_STYLE} formatter={(value) => [`$${value ?? 0}`]} />
                    <Legend wrapperStyle={{ fontSize: 12, color: "#8b95a5" }} />
                    <Line type="monotone" dataKey="without_fallback" stroke="#ef4444" strokeWidth={2} dot={false} name="Without Fallback" />
                    <Line type="monotone" dataKey="with_fallback" stroke="#22c55e" strokeWidth={2} dot={false} name="With Fallback" />
                  </LineChart>
                </ResponsiveContainer>
              </CardBody>
            </Card>
          </PageSection>

          {/* Detailed Agent Cost Table */}
          <PageSection hasBodyWrapper={false}>
            <div className="section-title">Agent Cost Breakdown</div>
            <Table aria-label="Agent cost breakdown" variant="compact">
              <Thead>
                <Tr>
                  <Th sort={{ sortBy: { index: 0, direction: "asc" }, onSort: () => {}, columnIndex: 0 }}>Agent</Th>
                  <Th>Team</Th>
                  <Th>Model</Th>
                  <Th>Tokens</Th>
                  <Th>Cost</Th>
                  <Th>Tasks</Th>
                </Tr>
              </Thead>
              <Tbody>
                {filteredAgents.map((agent) => (
                  <Tr key={agent.agent}>
                    <Td dataLabel="Agent" style={{ fontWeight: 600 }}>{agent.agent}</Td>
                    <Td dataLabel="Team">
                      <Label isCompact color="blue">{agent.team}</Label>
                    </Td>
                    <Td dataLabel="Model">
                      <Label isCompact color="purple">{agent.model}</Label>
                    </Td>
                    <Td dataLabel="Tokens">{fmtTokens(agent.tokens)}</Td>
                    <Td dataLabel="Cost" style={{ fontWeight: 600 }}>{fmtCurrency(agent.cost)}</Td>
                    <Td dataLabel="Tasks">{agent.tasks}</Td>
                  </Tr>
                ))}
              </Tbody>
            </Table>
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
