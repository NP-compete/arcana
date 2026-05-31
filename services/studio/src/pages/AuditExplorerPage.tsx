import { useState, useEffect, useCallback } from "react";
import {
  PageSection,
  Title,
  Content,
  Divider,
  Label,
  Spinner,
  Alert,
  Button,
  FormSelect,
  FormSelectOption,
  TextInput,
  Pagination,
} from "@patternfly/react-core";
import { Table, Thead, Tr, Th, Tbody, Td, ExpandableRowContent } from "@patternfly/react-table";
import {
  SearchIcon,
  DownloadIcon,
} from "@patternfly/react-icons";

/* ------------------------------------------------------------------ */
/*  Types                                                              */
/* ------------------------------------------------------------------ */

interface AuditEntry {
  id: string;
  timestamp: string;
  agent: string;
  user: string;
  action: string;
  resource: string;
  verdict: "allowed" | "blocked";
  ip: string;
  detail: Record<string, unknown>;
}

interface AuditStats {
  total_events: number;
  unique_agents: number;
  unique_users: number;
  ward_blocks: number;
}

/* ------------------------------------------------------------------ */
/*  Constants                                                          */
/* ------------------------------------------------------------------ */

const PERIOD_OPTIONS = [
  { value: "1d", label: "Last 24 hours" },
  { value: "7d", label: "Last 7 days" },
  { value: "30d", label: "Last 30 days" },
  { value: "90d", label: "Last 90 days" },
];

const ACTION_OPTIONS = [
  "all",
  "tool_call",
  "model_invoke",
  "agent_deploy",
  "agent_delete",
  "config_change",
  "auth_login",
  "auth_logout",
  "data_access",
  "guardrail_trigger",
];

const PAGE_SIZE = 50;

/* ------------------------------------------------------------------ */
/*  Mock data (used when API is unavailable)                           */
/* ------------------------------------------------------------------ */

function generateMockEntries(count: number): AuditEntry[] {
  const agents = ["research-pipeline-agent", "code-assistant", "data-pipeline-agent", "support-bot", "content-writer"];
  const users = ["alice@acme.com", "bob@acme.com", "charlie@acme.com", "system"];
  const actions = ["tool_call", "model_invoke", "agent_deploy", "config_change", "data_access", "guardrail_trigger"];
  const resources = ["/api/v1/agents", "/api/v1/skills/search", "/api/v1/mcp/execute", "/api/v1/eval/run", "/api/v1/connectors"];
  const verdicts: Array<"allowed" | "blocked"> = ["allowed", "allowed", "allowed", "allowed", "blocked"];

  const entries: AuditEntry[] = [];
  const now = Date.now();
  for (let i = 0; i < count; i++) {
    const ts = new Date(now - Math.random() * 7 * 86400000);
    const verdict = verdicts[Math.floor(Math.random() * verdicts.length)];
    entries.push({
      id: `audit-${crypto.randomUUID().slice(0, 8)}`,
      timestamp: ts.toISOString(),
      agent: agents[Math.floor(Math.random() * agents.length)],
      user: users[Math.floor(Math.random() * users.length)],
      action: actions[Math.floor(Math.random() * actions.length)],
      resource: resources[Math.floor(Math.random() * resources.length)],
      verdict,
      ip: `10.${Math.floor(Math.random() * 256)}.${Math.floor(Math.random() * 256)}.${Math.floor(Math.random() * 256)}`,
      detail: {
        duration_ms: Math.floor(Math.random() * 2000),
        tokens_used: verdict === "blocked" ? 0 : Math.floor(Math.random() * 5000),
        reason: verdict === "blocked" ? "Ward policy violation: PII detected" : undefined,
        model: verdict === "blocked" ? undefined : "gpt-4o",
      },
    });
  }
  entries.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime());
  return entries;
}

/* ------------------------------------------------------------------ */
/*  Helpers                                                            */
/* ------------------------------------------------------------------ */

function relativeTime(isoString: string): string {
  const now = Date.now();
  const then = new Date(isoString).getTime();
  const diffSec = Math.floor((now - then) / 1000);

  if (diffSec < 60) return `${diffSec}s ago`;
  if (diffSec < 3600) return `${Math.floor(diffSec / 60)}m ago`;
  if (diffSec < 86400) return `${Math.floor(diffSec / 3600)}h ago`;
  if (diffSec < 604800) return `${Math.floor(diffSec / 86400)}d ago`;
  return new Date(isoString).toLocaleDateString();
}

function downloadJSON(data: AuditEntry[], filename: string): void {
  const blob = new Blob([JSON.stringify(data, null, 2)], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

function downloadCSV(data: AuditEntry[], filename: string): void {
  const headers = ["Timestamp", "Agent", "User", "Action", "Resource", "Verdict", "IP"];
  const rows = data.map((e) => [
    e.timestamp,
    e.agent,
    e.user,
    e.action,
    e.resource,
    e.verdict,
    e.ip,
  ]);
  const csv = [headers.join(","), ...rows.map((r) => r.map((c) => `"${c}"`).join(","))].join("\n");
  const blob = new Blob([csv], { type: "text/csv" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

/* ------------------------------------------------------------------ */
/*  Component                                                          */
/* ------------------------------------------------------------------ */

export const AuditExplorerPage = () => {
  const [entries, setEntries] = useState<AuditEntry[]>([]);
  const [stats, setStats] = useState<AuditStats>({ total_events: 0, unique_agents: 0, unique_users: 0, ward_blocks: 0 });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Filters
  const [period, setPeriod] = useState("7d");
  const [agentFilter, setAgentFilter] = useState("");
  const [userFilter, setUserFilter] = useState("");
  const [actionFilter, setActionFilter] = useState("all");
  const [searchText, setSearchText] = useState("");

  // Pagination
  const [page, setPage] = useState(1);
  const [expandedRows, setExpandedRows] = useState<Set<string>>(new Set());

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const params = new URLSearchParams({ since: period, limit: "500" });
      if (agentFilter) params.set("agent", agentFilter);
      if (userFilter) params.set("user", userFilter);
      if (actionFilter !== "all") params.set("action", actionFilter);

      const [entriesRes, statsRes] = await Promise.allSettled([
        fetch(`/api/v1/enterprise/audit?${params}`),
        fetch(`/api/v1/enterprise/audit/stats?since=${period}`),
      ]);

      let fetchedEntries: AuditEntry[];
      if (entriesRes.status === "fulfilled" && entriesRes.value.ok) {
        const data = await entriesRes.value.json();
        fetchedEntries = data.entries ?? data.events ?? [];
      } else {
        fetchedEntries = generateMockEntries(120);
      }
      setEntries(fetchedEntries);

      if (statsRes.status === "fulfilled" && statsRes.value.ok) {
        const data = await statsRes.value.json();
        setStats({
          total_events: data.total_events ?? fetchedEntries.length,
          unique_agents: data.unique_agents ?? new Set(fetchedEntries.map((e) => e.agent)).size,
          unique_users: data.unique_users ?? new Set(fetchedEntries.map((e) => e.user)).size,
          ward_blocks: data.ward_blocks ?? fetchedEntries.filter((e) => e.verdict === "blocked").length,
        });
      } else {
        setStats({
          total_events: fetchedEntries.length,
          unique_agents: new Set(fetchedEntries.map((e) => e.agent)).size,
          unique_users: new Set(fetchedEntries.map((e) => e.user)).size,
          ward_blocks: fetchedEntries.filter((e) => e.verdict === "blocked").length,
        });
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load audit data");
      const mock = generateMockEntries(120);
      setEntries(mock);
      setStats({
        total_events: mock.length,
        unique_agents: new Set(mock.map((e) => e.agent)).size,
        unique_users: new Set(mock.map((e) => e.user)).size,
        ward_blocks: mock.filter((e) => e.verdict === "blocked").length,
      });
    } finally {
      setLoading(false);
    }
  }, [period, agentFilter, userFilter, actionFilter]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  // Client-side search filter
  const filteredEntries = searchText
    ? entries.filter(
        (e) =>
          e.agent.toLowerCase().includes(searchText.toLowerCase()) ||
          e.user.toLowerCase().includes(searchText.toLowerCase()) ||
          e.action.toLowerCase().includes(searchText.toLowerCase()) ||
          e.resource.toLowerCase().includes(searchText.toLowerCase()),
      )
    : entries;

  // Update stats for filtered results
  const filteredStats: AuditStats = searchText
    ? {
        total_events: filteredEntries.length,
        unique_agents: new Set(filteredEntries.map((e) => e.agent)).size,
        unique_users: new Set(filteredEntries.map((e) => e.user)).size,
        ward_blocks: filteredEntries.filter((e) => e.verdict === "blocked").length,
      }
    : stats;

  const totalPages = Math.ceil(filteredEntries.length / PAGE_SIZE);
  const pagedEntries = filteredEntries.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE);

  const toggleExpand = (id: string) => {
    setExpandedRows((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  const handleApply = () => {
    setPage(1);
    fetchData();
  };

  return (
    <>
      <PageSection hasBodyWrapper={false}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
          <div>
            <Title headingLevel="h1" size="2xl">Audit Explorer</Title>
            <Content component="p" style={{ marginTop: 4 }}>
              Immutable action log with filtering, search, and export.
            </Content>
          </div>
          <div style={{ display: "flex", gap: 8 }}>
            <Button
              variant="secondary"
              icon={<DownloadIcon />}
              onClick={() => downloadJSON(filteredEntries, "audit-export.json")}
            >
              JSON
            </Button>
            <Button
              variant="secondary"
              icon={<DownloadIcon />}
              onClick={() => downloadCSV(filteredEntries, "audit-export.csv")}
            >
              CSV
            </Button>
          </div>
        </div>
      </PageSection>
      <Divider />

      {error && (
        <PageSection hasBodyWrapper={false} style={{ paddingBottom: 0 }}>
          <Alert variant="info" title="Using sample data" isInline>
            Audit API not available. Displaying representative data.
          </Alert>
        </PageSection>
      )}

      {/* Filter Bar */}
      <PageSection hasBodyWrapper={false} style={{ paddingBottom: 0 }}>
        <div
          style={{
            display: "flex",
            gap: 12,
            alignItems: "flex-end",
            flexWrap: "wrap",
            padding: "16px 20px",
            background: "rgba(255,255,255,0.03)",
            borderRadius: 10,
            border: "1px solid rgba(255,255,255,0.06)",
          }}
        >
          <div>
            <div style={{ fontSize: 11, color: "#8b95a5", marginBottom: 4, fontWeight: 600 }}>Period</div>
            <FormSelect
              value={period}
              onChange={(_e, v) => setPeriod(v)}
              aria-label="Date period"
              style={{ width: 150 }}
            >
              {PERIOD_OPTIONS.map((o) => (
                <FormSelectOption key={o.value} value={o.value} label={o.label} />
              ))}
            </FormSelect>
          </div>
          <div>
            <div style={{ fontSize: 11, color: "#8b95a5", marginBottom: 4, fontWeight: 600 }}>Agent</div>
            <TextInput
              value={agentFilter}
              onChange={(_e, v) => setAgentFilter(v)}
              placeholder="Filter by agent"
              aria-label="Agent filter"
              style={{ width: 160 }}
            />
          </div>
          <div>
            <div style={{ fontSize: 11, color: "#8b95a5", marginBottom: 4, fontWeight: 600 }}>User</div>
            <TextInput
              value={userFilter}
              onChange={(_e, v) => setUserFilter(v)}
              placeholder="Filter by user"
              aria-label="User filter"
              style={{ width: 160 }}
            />
          </div>
          <div>
            <div style={{ fontSize: 11, color: "#8b95a5", marginBottom: 4, fontWeight: 600 }}>Action</div>
            <FormSelect
              value={actionFilter}
              onChange={(_e, v) => setActionFilter(v)}
              aria-label="Action filter"
              style={{ width: 160 }}
            >
              {ACTION_OPTIONS.map((a) => (
                <FormSelectOption key={a} value={a} label={a === "all" ? "All Actions" : a} />
              ))}
            </FormSelect>
          </div>
          <div>
            <div style={{ fontSize: 11, color: "#8b95a5", marginBottom: 4, fontWeight: 600 }}>Search</div>
            <TextInput
              value={searchText}
              onChange={(_e, v) => setSearchText(v)}
              placeholder="Search entries..."
              aria-label="Search"
              style={{ width: 200 }}
              customIcon={<SearchIcon />}
            />
          </div>
          <Button variant="primary" onClick={handleApply} style={{ alignSelf: "flex-end" }}>
            Apply
          </Button>
        </div>
      </PageSection>

      {loading ? (
        <PageSection hasBodyWrapper={false}>
          <div style={{ textAlign: "center", padding: 40 }}>
            <Spinner size="xl" />
          </div>
        </PageSection>
      ) : (
        <>
          {/* Stats Row */}
          <PageSection hasBodyWrapper={false} style={{ paddingBottom: 0 }}>
            <div style={{ display: "grid", gridTemplateColumns: "repeat(4, 1fr)", gap: 16 }}>
              <StatCard label="Total Events" value={String(filteredStats.total_events)} color="#3b82f6" />
              <StatCard label="Unique Agents" value={String(filteredStats.unique_agents)} color="#a855f7" />
              <StatCard label="Unique Users" value={String(filteredStats.unique_users)} color="#06b6d4" />
              <StatCard label="Ward Blocks" value={String(filteredStats.ward_blocks)} color="#ef4444" />
            </div>
          </PageSection>

          {/* Event Table */}
          <PageSection hasBodyWrapper={false}>
            <Table aria-label="Audit log" variant="compact">
              <Thead>
                <Tr>
                  <Th screenReaderText="Row expand" />
                  <Th>Timestamp</Th>
                  <Th>Agent</Th>
                  <Th>User</Th>
                  <Th>Action</Th>
                  <Th>Resource</Th>
                  <Th>Verdict</Th>
                  <Th>IP</Th>
                </Tr>
              </Thead>
              <Tbody>
                {pagedEntries.map((entry, rowIndex) => {
                  const isExpanded = expandedRows.has(entry.id);
                  return (
                    <EventRow
                      key={entry.id}
                      entry={entry}
                      rowIndex={rowIndex}
                      isExpanded={isExpanded}
                      onToggle={() => toggleExpand(entry.id)}
                    />
                  );
                })}
              </Tbody>
            </Table>

            {totalPages > 1 && (
              <Pagination
                itemCount={filteredEntries.length}
                perPage={PAGE_SIZE}
                page={page}
                onSetPage={(_e, p) => setPage(p)}
                onPerPageSelect={() => {}}
                style={{ marginTop: 16 }}
              />
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

const StatCard = ({ label, value, color }: { label: string; value: string; color: string }) => (
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

const EventRow = ({
  entry,
  rowIndex,
  isExpanded,
  onToggle,
}: {
  entry: AuditEntry;
  rowIndex: number;
  isExpanded: boolean;
  onToggle: () => void;
}) => (
  <>
    <Tr>
      <Td
        expand={{
          rowIndex,
          isExpanded,
          onToggle,
          expandId: `expand-${entry.id}`,
        }}
      />
      <Td dataLabel="Timestamp">
        <span title={new Date(entry.timestamp).toLocaleString()} style={{ cursor: "help" }}>
          {relativeTime(entry.timestamp)}
        </span>
      </Td>
      <Td dataLabel="Agent" style={{ fontWeight: 600, fontSize: 13 }}>{entry.agent}</Td>
      <Td dataLabel="User" style={{ fontSize: 13 }}>{entry.user}</Td>
      <Td dataLabel="Action">
        <Label isCompact color="blue">{entry.action}</Label>
      </Td>
      <Td dataLabel="Resource" style={{ fontSize: 12, fontFamily: "monospace", color: "#8b95a5" }}>
        {entry.resource}
      </Td>
      <Td dataLabel="Verdict">
        <Label isCompact color={entry.verdict === "allowed" ? "green" : "red"}>
          {entry.verdict === "allowed" ? "Allowed" : "Blocked"}
        </Label>
      </Td>
      <Td dataLabel="IP" style={{ fontSize: 12, fontFamily: "monospace", color: "#8b95a5" }}>
        {entry.ip}
      </Td>
    </Tr>
    {isExpanded && (
      <Tr isExpanded>
        <Td colSpan={8}>
          <ExpandableRowContent>
            <pre
              style={{
                margin: 0,
                padding: 16,
                background: "rgba(255,255,255,0.03)",
                borderRadius: 8,
                fontSize: 12,
                color: "#c5cdd8",
                overflow: "auto",
                maxHeight: 300,
              }}
            >
              {JSON.stringify(entry.detail, null, 2)}
            </pre>
          </ExpandableRowContent>
        </Td>
      </Tr>
    )}
  </>
);
