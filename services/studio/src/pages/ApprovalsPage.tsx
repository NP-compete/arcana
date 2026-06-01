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
  SearchInput,
  Modal,
  ModalHeader,
  ModalBody,
  ModalFooter,
  ModalVariant,
  TextArea,
  ToggleGroup,
  ToggleGroupItem,
} from "@patternfly/react-core";
import { Table, Thead, Tr, Th, Tbody, Td } from "@patternfly/react-table";
import {
  CheckCircleIcon,
  TimesCircleIcon,
  EyeIcon,
} from "@patternfly/react-icons";

/* ------------------------------------------------------------------ */
/*  Types                                                              */
/* ------------------------------------------------------------------ */

type ApprovalStatus = "pending" | "approved" | "rejected";
type PackageType = "Agent" | "Skill" | "Model" | "MCP";

interface ApprovalEntry {
  id: string;
  package_name: string;
  package_type: PackageType;
  author: string;
  version: string;
  status: ApprovalStatus;
  submitted_at: string;
  description: string;
  diff_summary: string;
  config_yaml: string;
}

interface ApprovalStats {
  pending: number;
  approved_today: number;
  rejected_today: number;
}

/* ------------------------------------------------------------------ */
/*  Constants                                                          */
/* ------------------------------------------------------------------ */

const PACKAGE_TYPES: PackageType[] = ["Agent", "Skill", "Model", "MCP"];

const TYPE_COLORS: Record<PackageType, "blue" | "purple" | "green" | "teal"> = {
  Agent: "blue",
  Skill: "purple",
  Model: "green",
  MCP: "teal",
};

const STATUS_COLORS: Record<ApprovalStatus, "yellow" | "green" | "red"> = {
  pending: "yellow",
  approved: "green",
  rejected: "red",
};

/* ------------------------------------------------------------------ */
/*  Mock data                                                          */
/* ------------------------------------------------------------------ */

function generateMockApprovals(): ApprovalEntry[] {
  const names = [
    "research-pipeline-agent",
    "code-gen-skill",
    "gpt-4o-turbo",
    "github-mcp",
    "data-analyst-agent",
    "sql-query-skill",
    "claude-3-opus",
    "slack-mcp",
  ];
  const types: PackageType[] = ["Agent", "Skill", "Model", "MCP"];
  const authors = ["alice@acme.com", "bob@acme.com", "charlie@acme.com", "dave@acme.com"];
  const statuses: ApprovalStatus[] = ["pending", "pending", "pending", "approved", "rejected"];

  const entries: ApprovalEntry[] = [];
  const now = Date.now();
  for (let i = 0; i < names.length; i++) {
    const ts = new Date(now - Math.random() * 3 * 86400000);
    entries.push({
      id: `approval-${crypto.randomUUID().slice(0, 8)}`,
      package_name: names[i],
      package_type: types[i % types.length],
      author: authors[i % authors.length],
      version: `${Math.floor(Math.random() * 3) + 1}.${Math.floor(Math.random() * 10)}.${Math.floor(Math.random() * 10)}`,
      status: statuses[i % statuses.length],
      submitted_at: ts.toISOString(),
      description: `Updated ${names[i]} with improved configuration and new capabilities.`,
      diff_summary: `+${Math.floor(Math.random() * 50) + 5} lines, -${Math.floor(Math.random() * 20)} lines across ${Math.floor(Math.random() * 5) + 1} files`,
      config_yaml: `name: ${names[i]}\nversion: 1.0.0\ntype: ${types[i % types.length].toLowerCase()}\nmetadata:\n  tier: functional\n  description: "${names[i]} configuration"\n  owner: ${authors[i % authors.length]}\nspec:\n  model: gpt-4o\n  max_tokens: 4096\n  temperature: 0.7\n`,
    });
  }
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
  return `${Math.floor(diffSec / 86400)}d ago`;
}

/* ------------------------------------------------------------------ */
/*  Component                                                          */
/* ------------------------------------------------------------------ */

export const ApprovalsPage = () => {
  const [entries, setEntries] = useState<ApprovalEntry[]>([]);
  const [stats, setStats] = useState<ApprovalStats>({ pending: 0, approved_today: 0, rejected_today: 0 });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionResult, setActionResult] = useState<{ type: "success" | "danger"; message: string } | null>(null);

  /* Filters */
  const [statusFilter, setStatusFilter] = useState<"all" | ApprovalStatus>("all");
  const [typeFilter, setTypeFilter] = useState<"all" | PackageType>("all");
  const [searchText, setSearchText] = useState("");

  /* Review modal */
  const [reviewTarget, setReviewTarget] = useState<ApprovalEntry | null>(null);
  const [comment, setComment] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const params = new URLSearchParams();
      if (statusFilter !== "all") params.set("status", statusFilter);
      const res = await fetch(`/api/v1/promotions?${params}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      const fetched: ApprovalEntry[] = data.entries ?? data.approvals ?? [];
      setEntries(fetched);
      computeStats(fetched);
    } catch {
      const mock = generateMockApprovals();
      setEntries(mock);
      computeStats(mock);
      setError("Approvals API not available. Displaying sample data.");
    } finally {
      setLoading(false);
    }
  }, [statusFilter]);

  const computeStats = (items: ApprovalEntry[]) => {
    const todayStart = new Date();
    todayStart.setHours(0, 0, 0, 0);
    const todayMs = todayStart.getTime();
    setStats({
      pending: items.filter((e) => e.status === "pending").length,
      approved_today: items.filter((e) => e.status === "approved" && new Date(e.submitted_at).getTime() >= todayMs).length,
      rejected_today: items.filter((e) => e.status === "rejected" && new Date(e.submitted_at).getTime() >= todayMs).length,
    });
  };

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  /* Actions */
  const handleApprove = async () => {
    if (!reviewTarget) return;
    setSubmitting(true);
    try {
      const res = await fetch(`/api/v1/promotions/${encodeURIComponent(reviewTarget.id)}/approve`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ comment }),
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error((data as Record<string, string>).error ?? `HTTP ${res.status}`);
      }
      setActionResult({ type: "success", message: `"${reviewTarget.package_name}" approved successfully.` });
      setReviewTarget(null);
      setComment("");
      await fetchData();
    } catch (e) {
      setActionResult({ type: "danger", message: e instanceof Error ? e.message : "Approve failed" });
    } finally {
      setSubmitting(false);
    }
  };

  const handleReject = async () => {
    if (!reviewTarget) return;
    setSubmitting(true);
    try {
      const res = await fetch(`/api/v1/promotions/${encodeURIComponent(reviewTarget.id)}/reject`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ comment }),
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error((data as Record<string, string>).error ?? `HTTP ${res.status}`);
      }
      setActionResult({ type: "success", message: `"${reviewTarget.package_name}" rejected.` });
      setReviewTarget(null);
      setComment("");
      await fetchData();
    } catch (e) {
      setActionResult({ type: "danger", message: e instanceof Error ? e.message : "Reject failed" });
    } finally {
      setSubmitting(false);
    }
  };

  /* Filtering */
  const filtered = entries.filter((e) => {
    if (typeFilter !== "all" && e.package_type !== typeFilter) return false;
    if (statusFilter !== "all" && e.status !== statusFilter) return false;
    if (searchText) {
      const q = searchText.toLowerCase();
      return (
        e.package_name.toLowerCase().includes(q) ||
        e.author.toLowerCase().includes(q) ||
        e.package_type.toLowerCase().includes(q)
      );
    }
    return true;
  });

  return (
    <>
      <PageSection hasBodyWrapper={false}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
          <div>
            <Title headingLevel="h1" size="2xl">Approvals</Title>
            <Content component="p" style={{ marginTop: 4, color: "var(--pf-t--global--text--color--subtle)" }}>
              Review and approve package changes before publishing.
            </Content>
          </div>
          <Label isCompact color="blue">{filtered.length} items</Label>
        </div>
      </PageSection>
      <Divider />

      {error && (
        <PageSection hasBodyWrapper={false} style={{ paddingBottom: 0 }}>
          <Alert variant="info" title="Using sample data" isInline>
            {error}
          </Alert>
        </PageSection>
      )}
      {actionResult && (
        <PageSection hasBodyWrapper={false} style={{ paddingBottom: 0 }}>
          <Alert variant={actionResult.type} title={actionResult.message} isInline />
        </PageSection>
      )}

      {/* Stats Row */}
      <PageSection hasBodyWrapper={false} style={{ paddingBottom: 0 }}>
        <div style={{ display: "grid", gridTemplateColumns: "repeat(3, 1fr)", gap: 16 }}>
          <StatCard label="Pending" value={String(stats.pending)} color="#eab308" />
          <StatCard label="Approved Today" value={String(stats.approved_today)} color="#22c55e" />
          <StatCard label="Rejected Today" value={String(stats.rejected_today)} color="#ef4444" />
        </div>
      </PageSection>

      {/* Filters */}
      <PageSection hasBodyWrapper={false} style={{ paddingBottom: 0 }}>
        <div style={{ display: "flex", gap: 12, alignItems: "center", flexWrap: "wrap" }}>
          <SearchInput
            placeholder="Search packages..."
            value={searchText}
            onChange={(_e, val) => setSearchText(val)}
            onClear={() => setSearchText("")}
            style={{ maxWidth: 280 }}
          />
          <ToggleGroup aria-label="Type filter">
            <ToggleGroupItem
              text="All"
              buttonId="type-all"
              isSelected={typeFilter === "all"}
              onChange={() => setTypeFilter("all")}
            />
            {PACKAGE_TYPES.map((t) => (
              <ToggleGroupItem
                key={t}
                text={t}
                buttonId={`type-${t}`}
                isSelected={typeFilter === t}
                onChange={() => setTypeFilter(t)}
              />
            ))}
          </ToggleGroup>
          <ToggleGroup aria-label="Status filter">
            {(["all", "pending", "approved", "rejected"] as const).map((s) => (
              <ToggleGroupItem
                key={s}
                text={s === "all" ? "All Status" : s.charAt(0).toUpperCase() + s.slice(1)}
                buttonId={`status-${s}`}
                isSelected={statusFilter === s}
                onChange={() => setStatusFilter(s)}
              />
            ))}
          </ToggleGroup>
        </div>
      </PageSection>

      {/* Table */}
      <PageSection hasBodyWrapper={false}>
        {loading ? (
          <div style={{ textAlign: "center", padding: 40 }}>
            <Spinner size="xl" />
          </div>
        ) : filtered.length === 0 ? (
          <div className="arcana-empty-state">
            <div className="arcana-empty-icon">
              <CheckCircleIcon />
            </div>
            <Title headingLevel="h2" size="xl" style={{ marginBottom: 8 }}>
              No approvals found
            </Title>
            <Content component="p" style={{ maxWidth: 480, margin: "0 auto", color: "var(--pf-t--global--text--color--subtle)" }}>
              Adjust your filters or check back later.
            </Content>
          </div>
        ) : (
          <Table aria-label="Approvals" variant="compact">
            <Thead>
              <Tr>
                <Th>Package Name</Th>
                <Th>Type</Th>
                <Th>Author</Th>
                <Th>Version</Th>
                <Th>Status</Th>
                <Th>Submitted</Th>
                <Th>Actions</Th>
              </Tr>
            </Thead>
            <Tbody>
              {filtered.map((entry) => (
                <Tr key={entry.id}>
                  <Td dataLabel="Package Name" style={{ fontWeight: 600 }}>{entry.package_name}</Td>
                  <Td dataLabel="Type">
                    <Label isCompact color={TYPE_COLORS[entry.package_type]}>{entry.package_type}</Label>
                  </Td>
                  <Td dataLabel="Author" style={{ fontSize: 13 }}>{entry.author}</Td>
                  <Td dataLabel="Version">
                    <Label isCompact color="grey">{entry.version}</Label>
                  </Td>
                  <Td dataLabel="Status">
                    <Label isCompact color={STATUS_COLORS[entry.status]}>
                      {entry.status.charAt(0).toUpperCase() + entry.status.slice(1)}
                    </Label>
                  </Td>
                  <Td dataLabel="Submitted">
                    <span title={new Date(entry.submitted_at).toLocaleString()} style={{ cursor: "help", fontSize: 13 }}>
                      {relativeTime(entry.submitted_at)}
                    </span>
                  </Td>
                  <Td dataLabel="Actions">
                    <Button
                      variant="secondary"
                      size="sm"
                      icon={<EyeIcon />}
                      onClick={() => {
                        setReviewTarget(entry);
                        setComment("");
                      }}
                    >
                      Review
                    </Button>
                  </Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        )}
      </PageSection>

      {/* Review Modal */}
      <Modal
        variant={ModalVariant.large}
        isOpen={reviewTarget !== null}
        onClose={() => setReviewTarget(null)}
        aria-labelledby="review-modal-title"
      >
        <ModalHeader title={`Review: ${reviewTarget?.package_name ?? ""}`} labelId="review-modal-title" />
        <ModalBody>
          {reviewTarget && (
            <div style={{ display: "flex", flexDirection: "column", gap: 20 }}>
              {/* Info grid */}
              <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 12 }}>
                <InfoField label="Package Name" value={reviewTarget.package_name} />
                <InfoField label="Version" value={reviewTarget.version} />
                <InfoField label="Type" value={reviewTarget.package_type} />
                <InfoField label="Author" value={reviewTarget.author} />
              </div>

              <div>
                <div style={{ fontSize: 12, color: "#8b95a5", fontWeight: 600, marginBottom: 6, textTransform: "uppercase", letterSpacing: "0.5px" }}>
                  Description
                </div>
                <div style={{ fontSize: 14, color: "#c5cdd8" }}>{reviewTarget.description}</div>
              </div>

              <div>
                <div style={{ fontSize: 12, color: "#8b95a5", fontWeight: 600, marginBottom: 6, textTransform: "uppercase", letterSpacing: "0.5px" }}>
                  Diff Summary
                </div>
                <div style={{
                  padding: "8px 12px",
                  background: "rgba(255,255,255,0.04)",
                  borderRadius: 8,
                  fontFamily: "monospace",
                  fontSize: 13,
                  color: "#c5cdd8",
                }}>
                  {reviewTarget.diff_summary}
                </div>
              </div>

              <div>
                <div style={{ fontSize: 12, color: "#8b95a5", fontWeight: 600, marginBottom: 6, textTransform: "uppercase", letterSpacing: "0.5px" }}>
                  YAML Configuration
                </div>
                <pre style={{
                  margin: 0,
                  padding: 16,
                  background: "#0d0f14",
                  borderRadius: 8,
                  fontSize: 12,
                  color: "#c5cdd8",
                  overflow: "auto",
                  maxHeight: 240,
                  border: "1px solid rgba(255,255,255,0.08)",
                  fontFamily: "'JetBrains Mono', 'Fira Code', 'Consolas', monospace",
                }}>
                  {reviewTarget.config_yaml}
                </pre>
              </div>

              <div>
                <div style={{ fontSize: 12, color: "#8b95a5", fontWeight: 600, marginBottom: 6, textTransform: "uppercase", letterSpacing: "0.5px" }}>
                  Approval Notes
                </div>
                <TextArea
                  value={comment}
                  onChange={(_e, v) => setComment(v)}
                  placeholder="Add a comment (optional)..."
                  rows={3}
                  aria-label="Approval notes"
                />
              </div>
            </div>
          )}
        </ModalBody>
        <ModalFooter>
          <Button
            variant="primary"
            icon={<CheckCircleIcon />}
            onClick={handleApprove}
            isDisabled={submitting}
            isLoading={submitting}
            style={{ background: "#22c55e", borderColor: "#22c55e" }}
          >
            Approve
          </Button>
          <Button
            variant="danger"
            icon={<TimesCircleIcon />}
            onClick={handleReject}
            isDisabled={submitting}
            isLoading={submitting}
          >
            Reject
          </Button>
          <Button variant="link" onClick={() => setReviewTarget(null)}>
            Cancel
          </Button>
        </ModalFooter>
      </Modal>
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

const InfoField = ({ label, value }: { label: string; value: string }) => (
  <div>
    <div style={{ fontSize: 11, color: "#8b95a5", fontWeight: 600, marginBottom: 4, textTransform: "uppercase", letterSpacing: "0.5px" }}>
      {label}
    </div>
    <div style={{ fontSize: 14, fontWeight: 600, color: "#c5cdd8" }}>{value}</div>
  </div>
);
