import { useState, useEffect, useCallback } from "react";
import {
  PageSection,
  Title,
  Card,
  CardBody,
  Grid,
  GridItem,
  Content,
  Divider,
  Label,
  Button,
  Modal,
  ModalHeader,
  ModalBody,
  ModalFooter,
  ModalVariant,
  Form,
  FormGroup,
  TextInput,
  TextArea,
  FormSelect,
  FormSelectOption,
  Checkbox,
  Alert,
  Spinner,
  Tabs,
  Tab,
  TabTitleText,
  Wizard,
  WizardStep,
  Slider,
} from "@patternfly/react-core";
import { Table, Thead, Tr, Th, Tbody, Td, ExpandableRowContent } from "@patternfly/react-table";
import {
  ChartBarIcon,
  PlusCircleIcon,
  SearchIcon,
  PlayIcon,
  TrashIcon,
} from "@patternfly/react-icons";
import {
  PieChart,
  Pie,
  Cell,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  LineChart,
  Line,
  CartesianGrid,
  Legend,
} from "recharts";

/* ------------------------------------------------------------------ */
/*  Types                                                              */
/* ------------------------------------------------------------------ */

interface JudgeResult {
  judge_name: string;
  tier: string;
  score: number;
  passed: boolean;
  details?: string;
}

interface EvalRun {
  run_id: string;
  skill_ref: string;
  version?: string;
  status: string;
  badge?: { level: string } | null;
  score?: number;
  lift?: number;
  regression?: boolean;
  created_at?: string;
  judge_results?: JudgeResult[];
}

interface TestCase {
  id: string;
  input: string;
  expected: string;
  negative: boolean;
  tags: string;
}

interface DashboardMetrics {
  badge_distribution: { name: string; count: number }[];
  skill_lift: { skill: string; lift: number }[];
  regression_timeline: { date: string; regressions: number }[];
  total_skills_evaluated: number;
  avg_skill_lift: number;
  regression_rate: number;
  security_violations: number;
}

/* ------------------------------------------------------------------ */
/*  Constants                                                          */
/* ------------------------------------------------------------------ */

const BADGE_COLORS: Record<string, string> = {
  gold: "#d69e2e",
  silver: "#a0aec0",
  bronze: "#c05621",
  failed: "#e53e3e",
  untested: "#718096",
};

const BADGE_LABEL_COLORS: Record<string, "yellow" | "grey" | "orange" | "red"> = {
  gold: "yellow",
  silver: "grey",
  bronze: "orange",
  failed: "red",
  untested: "grey",
};

const STATUS_LABEL_COLORS: Record<string, "green" | "blue" | "orange" | "red" | "grey"> = {
  completed: "green",
  running: "blue",
  pending: "orange",
  failed: "red",
};

const badgeLabelColor = (level: string): "yellow" | "grey" | "orange" | "red" => {
  return BADGE_LABEL_COLORS[level?.toLowerCase()] ?? "grey";
};

const statusLabelColor = (status: string): "green" | "blue" | "orange" | "red" | "grey" => {
  return STATUS_LABEL_COLORS[status] ?? "grey";
};

const PIE_COLORS = [
  BADGE_COLORS.gold,
  BADGE_COLORS.silver,
  BADGE_COLORS.bronze,
  BADGE_COLORS.untested,
  BADGE_COLORS.failed,
];

const TOOLTIP_STYLE = {
  contentStyle: {
    background: "#1a1d2e",
    border: "1px solid rgba(255,255,255,0.12)",
    borderRadius: 8,
    fontSize: 12,
    color: "#e2e8f0",
  },
};

let caseIdCounter = 0;
const generateCaseId = (): string => {
  caseIdCounter += 1;
  return `tc-${caseIdCounter}`;
};

/* ------------------------------------------------------------------ */
/*  Tab 1: Eval Runs                                                   */
/* ------------------------------------------------------------------ */

const EvalRunsTab = () => {
  const [runs, setRuns] = useState<EvalRun[]>([]);
  const [loading, setLoading] = useState(true);
  const [fetchError, setFetchError] = useState<string | null>(null);
  const [expandedRows, setExpandedRows] = useState<Set<string>>(new Set());

  const [filterSkill, setFilterSkill] = useState("");
  const [filterBadge, setFilterBadge] = useState("");
  const [filterStatus, setFilterStatus] = useState("");

  const [detailModalOpen, setDetailModalOpen] = useState(false);
  const [detailRun, setDetailRun] = useState<EvalRun | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);

  const fetchRuns = useCallback(async () => {
    try {
      const params = new URLSearchParams();
      if (filterSkill) params.set("skill", filterSkill);
      if (filterStatus) params.set("status", filterStatus);
      params.set("limit", "50");
      const res = await fetch(`/api/v1/eval/runs?${params.toString()}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      setRuns(data.runs ?? []);
      setFetchError(null);
    } catch (e) {
      setFetchError(e instanceof Error ? e.message : "Failed to load eval runs");
    } finally {
      setLoading(false);
    }
  }, [filterSkill, filterStatus]);

  useEffect(() => {
    fetchRuns();
  }, [fetchRuns]);

  const toggleRow = (runId: string) => {
    setExpandedRows((prev) => {
      const next = new Set(prev);
      if (next.has(runId)) next.delete(runId);
      else next.add(runId);
      return next;
    });
  };

  const viewReport = async (run: EvalRun) => {
    setDetailModalOpen(true);
    setDetailLoading(true);
    try {
      const res = await fetch(`/api/v1/eval/runs/${encodeURIComponent(run.run_id)}`);
      if (res.ok) {
        const data = await res.json();
        setDetailRun(data);
      } else {
        setDetailRun(run);
      }
    } catch {
      setDetailRun(run);
    } finally {
      setDetailLoading(false);
    }
  };

  const filteredRuns = runs.filter((r) => {
    if (filterBadge && r.badge?.level?.toLowerCase() !== filterBadge) return false;
    return true;
  });

  return (
    <>
      {/* Filters */}
      <div style={{ display: "flex", gap: 12, marginBottom: 16, alignItems: "flex-end" }}>
        <FormGroup label="Skill" fieldId="filter-skill" style={{ flex: 1 }}>
          <TextInput
            id="filter-skill"
            value={filterSkill}
            onChange={(_e, v) => setFilterSkill(v)}
            placeholder="Filter by skill..."
          />
        </FormGroup>
        <FormGroup label="Badge" fieldId="filter-badge" style={{ width: 140 }}>
          <FormSelect id="filter-badge" value={filterBadge} onChange={(_e, v) => setFilterBadge(v)} aria-label="Badge filter">
            <FormSelectOption value="" label="All" />
            <FormSelectOption value="gold" label="Gold" />
            <FormSelectOption value="silver" label="Silver" />
            <FormSelectOption value="bronze" label="Bronze" />
            <FormSelectOption value="failed" label="Failed" />
            <FormSelectOption value="untested" label="Untested" />
          </FormSelect>
        </FormGroup>
        <FormGroup label="Status" fieldId="filter-status" style={{ width: 140 }}>
          <FormSelect id="filter-status" value={filterStatus} onChange={(_e, v) => setFilterStatus(v)} aria-label="Status filter">
            <FormSelectOption value="" label="All" />
            <FormSelectOption value="running" label="Running" />
            <FormSelectOption value="completed" label="Completed" />
            <FormSelectOption value="failed" label="Failed" />
          </FormSelect>
        </FormGroup>
        <Button variant="secondary" icon={<SearchIcon />} onClick={fetchRuns} style={{ marginBottom: 0 }}>
          Search
        </Button>
      </div>

      {fetchError && (
        <Alert variant="warning" title="Could not load eval runs" isInline style={{ marginBottom: 16 }}>
          {fetchError}
        </Alert>
      )}

      {loading ? (
        <div style={{ textAlign: "center", padding: 40 }}>
          <Spinner size="xl" />
        </div>
      ) : filteredRuns.length > 0 ? (
        <Table aria-label="Evaluation runs" variant="compact">
          <Thead>
            <Tr>
              <Th />
              <Th>Skill</Th>
              <Th>Version</Th>
              <Th>Badge</Th>
              <Th>Status</Th>
              <Th modifier="fitContent">Score</Th>
              <Th modifier="fitContent">Lift</Th>
              <Th modifier="fitContent">Regression</Th>
              <Th>Date</Th>
              <Th>Actions</Th>
            </Tr>
          </Thead>
          <Tbody>
            {filteredRuns.map((run, rowIndex) => {
              const isExpanded = expandedRows.has(run.run_id);
              return (
                <>
                  <Tr key={run.run_id}>
                    <Td
                      expand={{
                        rowIndex,
                        isExpanded,
                        onToggle: () => toggleRow(run.run_id),
                      }}
                    />
                    <Td dataLabel="Skill">{run.skill_ref}</Td>
                    <Td dataLabel="Version">{run.version ?? "—"}</Td>
                    <Td dataLabel="Badge">
                      {run.badge ? (
                        <Label
                          isCompact
                          color={badgeLabelColor(run.badge.level)}
                          style={{
                            borderLeft: `3px solid ${BADGE_COLORS[run.badge.level.toLowerCase()] ?? "#718096"}`,
                          }}
                        >
                          {run.badge.level}
                        </Label>
                      ) : (
                        <Label isCompact color="grey">Untested</Label>
                      )}
                    </Td>
                    <Td dataLabel="Status">
                      <Label isCompact color={statusLabelColor(run.status)}>
                        {run.status}
                      </Label>
                    </Td>
                    <Td dataLabel="Score">
                      {run.score != null ? (
                        <span style={{ fontFamily: "var(--pf-t--global--font--family--mono)", fontSize: 13 }}>
                          {(run.score * 100).toFixed(1)}%
                        </span>
                      ) : "—"}
                    </Td>
                    <Td dataLabel="Lift">
                      {run.lift != null ? (
                        <span
                          style={{
                            color: run.lift > 0 ? "#38a169" : run.lift < 0 ? "#e53e3e" : "#8b95a5",
                            fontFamily: "var(--pf-t--global--font--family--mono)",
                            fontSize: 13,
                          }}
                        >
                          {run.lift > 0 ? "+" : ""}{(run.lift * 100).toFixed(1)}%
                        </span>
                      ) : "—"}
                    </Td>
                    <Td dataLabel="Regression">
                      {run.regression != null ? (
                        run.regression ? (
                          <Label isCompact color="red">Yes</Label>
                        ) : (
                          <Label isCompact color="green">No</Label>
                        )
                      ) : "—"}
                    </Td>
                    <Td dataLabel="Date">
                      {run.created_at ? new Date(run.created_at).toLocaleDateString() : "—"}
                    </Td>
                    <Td dataLabel="Actions">
                      <Button
                        variant="link"
                        isInline
                        onClick={() => viewReport(run)}
                      >
                        View Report
                      </Button>
                    </Td>
                  </Tr>
                  {isExpanded && (
                    <Tr key={`${run.run_id}-expanded`} isExpanded={isExpanded}>
                      <Td colSpan={10}>
                        <ExpandableRowContent>
                          {run.judge_results && run.judge_results.length > 0 ? (
                            <div style={{ display: "flex", gap: 8, flexWrap: "wrap", padding: "8px 0" }}>
                              {run.judge_results.map((jr, i) => (
                                <div
                                  key={i}
                                  style={{
                                    padding: "6px 12px",
                                    borderRadius: 6,
                                    background: jr.passed ? "rgba(56,161,105,0.08)" : "rgba(229,62,62,0.08)",
                                    border: `1px solid ${jr.passed ? "rgba(56,161,105,0.2)" : "rgba(229,62,62,0.2)"}`,
                                    fontSize: 12,
                                  }}
                                >
                                  <span style={{ fontWeight: 600 }}>{jr.judge_name}</span>
                                  <span style={{ marginLeft: 8, color: "#8b95a5" }}>{jr.tier}</span>
                                  <Label
                                    isCompact
                                    color={jr.passed ? "green" : "red"}
                                    style={{ marginLeft: 8 }}
                                  >
                                    {(jr.score * 100).toFixed(0)}%
                                  </Label>
                                </div>
                              ))}
                            </div>
                          ) : (
                            <div style={{ color: "#6b7585", fontSize: 13, padding: "8px 0" }}>
                              No judge results available. Expand the report for details.
                            </div>
                          )}
                        </ExpandableRowContent>
                      </Td>
                    </Tr>
                  )}
                </>
              );
            })}
          </Tbody>
        </Table>
      ) : (
        <div className="arcana-empty-state" style={{ paddingBottom: 32 }}>
          <div className="arcana-empty-icon">
            <ChartBarIcon />
          </div>
          <Title headingLevel="h2" size="xl" style={{ marginBottom: 8 }}>
            No evaluation runs found
          </Title>
          <Content component="p" style={{ maxWidth: 480, margin: "0 auto", color: "var(--pf-t--global--text--color--subtle)" }}>
            Create a new eval run to start testing your skills with the 4-tier judge pipeline.
          </Content>
        </div>
      )}

      {/* Detail Modal */}
      <Modal
        variant={ModalVariant.large}
        isOpen={detailModalOpen}
        onClose={() => setDetailModalOpen(false)}
        aria-labelledby="eval-detail-title"
      >
        <ModalHeader title="Evaluation Report" labelId="eval-detail-title" />
        <ModalBody>
          {detailLoading ? (
            <div style={{ textAlign: "center", padding: 40 }}><Spinner size="xl" /></div>
          ) : detailRun ? (
            <div>
              <Grid hasGutter>
                <GridItem span={4}>
                  <div style={{ fontSize: 12, color: "#8b95a5", marginBottom: 4 }}>SKILL</div>
                  <div style={{ fontSize: 16, fontWeight: 600, color: "#e2e8f0" }}>{detailRun.skill_ref}</div>
                </GridItem>
                <GridItem span={4}>
                  <div style={{ fontSize: 12, color: "#8b95a5", marginBottom: 4 }}>BADGE</div>
                  {detailRun.badge ? (
                    <Label color={badgeLabelColor(detailRun.badge.level)}>{detailRun.badge.level}</Label>
                  ) : (
                    <Label color="grey">Untested</Label>
                  )}
                </GridItem>
                <GridItem span={4}>
                  <div style={{ fontSize: 12, color: "#8b95a5", marginBottom: 4 }}>SCORE</div>
                  <div style={{ fontSize: 20, fontWeight: 700, color: "#e2e8f0" }}>
                    {detailRun.score != null ? `${(detailRun.score * 100).toFixed(1)}%` : "—"}
                  </div>
                </GridItem>
              </Grid>
              {detailRun.judge_results && detailRun.judge_results.length > 0 && (
                <div style={{ marginTop: 20 }}>
                  <div style={{ fontSize: 13, fontWeight: 700, color: "#8b95a5", marginBottom: 8, textTransform: "uppercase", letterSpacing: "0.8px" }}>
                    Judge Results
                  </div>
                  <Table aria-label="Judge results" variant="compact">
                    <Thead>
                      <Tr>
                        <Th>Judge</Th>
                        <Th>Tier</Th>
                        <Th>Score</Th>
                        <Th>Passed</Th>
                        <Th>Details</Th>
                      </Tr>
                    </Thead>
                    <Tbody>
                      {detailRun.judge_results.map((jr, i) => (
                        <Tr key={i}>
                          <Td>{jr.judge_name}</Td>
                          <Td><Label isCompact color="blue">{jr.tier}</Label></Td>
                          <Td>
                            <span style={{ fontFamily: "var(--pf-t--global--font--family--mono)" }}>
                              {(jr.score * 100).toFixed(1)}%
                            </span>
                          </Td>
                          <Td>
                            <Label isCompact color={jr.passed ? "green" : "red"}>
                              {jr.passed ? "Pass" : "Fail"}
                            </Label>
                          </Td>
                          <Td>{jr.details ?? "—"}</Td>
                        </Tr>
                      ))}
                    </Tbody>
                  </Table>
                </div>
              )}
            </div>
          ) : (
            <Alert variant="info" title="No details available" isInline />
          )}
        </ModalBody>
        <ModalFooter>
          <Button variant="link" onClick={() => setDetailModalOpen(false)}>Close</Button>
        </ModalFooter>
      </Modal>
    </>
  );
};

/* ------------------------------------------------------------------ */
/*  Tab 2: Eval Builder (Wizard)                                       */
/* ------------------------------------------------------------------ */

const EvalBuilderTab = () => {
  const [skills, setSkills] = useState<string[]>([]);
  const [selectedSkill, setSelectedSkill] = useState("");
  const [testCases, setTestCases] = useState<TestCase[]>([
    { id: generateCaseId(), input: "", expected: "", negative: false, tags: "" },
  ]);
  const [judges, setJudges] = useState({
    keyword: true,
    security: true,
    llm: false,
    trajectory: false,
  });
  const [llmCriteria, setLlmCriteria] = useState("");
  const [trajectoryMaxCalls, setTrajectoryMaxCalls] = useState(10);
  const [trials, setTrials] = useState(3);
  const [conditions, setConditions] = useState(true);
  const [modelSelector, setModelSelector] = useState("gpt-4o");
  const [gateMode, setGateMode] = useState("enforce");

  const [submitting, setSubmitting] = useState(false);
  const [submitResult, setSubmitResult] = useState<{ type: "success" | "danger"; text: string } | null>(null);

  useEffect(() => {
    fetch("/api/v1/catalog/skills")
      .then((r) => r.json())
      .then((d) => {
        const names: string[] = (d.entries ?? []).map((s: { name: string }) => s.name);
        setSkills(names);
        if (names.length > 0 && !selectedSkill) setSelectedSkill(names[0]);
      })
      .catch(() => {});
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const addCase = () => {
    setTestCases((prev) => [
      ...prev,
      { id: generateCaseId(), input: "", expected: "", negative: false, tags: "" },
    ]);
  };

  const removeCase = (id: string) => {
    setTestCases((prev) => prev.filter((c) => c.id !== id));
  };

  const updateCase = (id: string, patch: Partial<TestCase>) => {
    setTestCases((prev) =>
      prev.map((c) => (c.id === id ? { ...c, ...patch } : c)),
    );
  };

  const estimatedCost = () => {
    const caseCount = testCases.filter((c) => c.input.trim()).length;
    const judgeCount = Object.values(judges).filter(Boolean).length;
    const conditionCount = conditions ? 3 : 1;
    const totalCalls = caseCount * judgeCount * trials * conditionCount;
    const costPerCall = judges.llm ? 0.03 : 0.001;
    return { calls: totalCalls, cost: (totalCalls * costPerCall).toFixed(2), duration: `~${Math.ceil(totalCalls * 2 / 60)} min` };
  };

  const handleRun = async () => {
    if (!selectedSkill) return;
    setSubmitting(true);
    setSubmitResult(null);
    try {
      const validCases = testCases.filter((c) => c.input.trim());
      const selectedJudges: string[] = [];
      if (judges.keyword) selectedJudges.push("keyword");
      if (judges.security) selectedJudges.push("security");
      if (judges.llm) selectedJudges.push("llm");
      if (judges.trajectory) selectedJudges.push("trajectory");

      const res = await fetch("/api/v1/eval/run", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          skill_ref: selectedSkill,
          cases: validCases.map((c) => ({
            id: c.id,
            input: c.input,
            expected: c.expected,
            negative: c.negative,
            tags: c.tags.split(",").map((t) => t.trim()).filter(Boolean),
          })),
          judges: selectedJudges.map((j) => ({
            name: `${j}_judge`,
            tier: j,
            weight: 1 / selectedJudges.length,
            config: j === "llm" ? { criteria: llmCriteria } : j === "trajectory" ? { max_calls: trajectoryMaxCalls } : {},
          })),
          settings: {
            trials,
            conditions: conditions ? ["no-skill", "with-skill", "reference"] : ["with-skill"],
            model: modelSelector,
            gate_mode: gateMode,
          },
        }),
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.detail ?? data.error ?? `HTTP ${res.status}`);
      setSubmitResult({ type: "success", text: `Evaluation run ${data.run_id} started successfully` });
    } catch (e) {
      setSubmitResult({ type: "danger", text: e instanceof Error ? e.message : "Failed to start evaluation" });
    } finally {
      setSubmitting(false);
    }
  };

  const est = estimatedCost();

  return (
    <div>
      {submitResult && (
        <Alert
          variant={submitResult.type}
          title={submitResult.text}
          isInline
          style={{ marginBottom: 16 }}
          timeout={8000}
          onTimeout={() => setSubmitResult(null)}
        />
      )}

      <Wizard
        height={600}
        onSave={handleRun}
        onClose={() => {}}
      >
        {/* Step 1: Select Skill */}
        <WizardStep name="Select Skill" id="step-skill">
          <div style={{ maxWidth: 500 }}>
            <Title headingLevel="h3" size="lg" style={{ marginBottom: 16 }}>Select Skill</Title>
            <FormGroup label="Skill to evaluate" fieldId="eval-skill" isRequired>
              <FormSelect
                id="eval-skill"
                value={selectedSkill}
                onChange={(_e, v) => setSelectedSkill(v)}
                aria-label="Select skill"
              >
                {skills.length === 0 && <FormSelectOption value="" label="No skills available" isDisabled />}
                {skills.map((s) => (
                  <FormSelectOption key={s} value={s} label={s} />
                ))}
              </FormSelect>
            </FormGroup>
          </div>
        </WizardStep>

        {/* Step 2: Test Cases */}
        <WizardStep name="Test Cases" id="step-cases">
          <div>
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 16 }}>
              <Title headingLevel="h3" size="lg">Test Cases ({testCases.length})</Title>
              <Button variant="secondary" icon={<PlusCircleIcon />} onClick={addCase} size="sm">
                Add Case
              </Button>
            </div>
            <Table aria-label="Test cases" variant="compact">
              <Thead>
                <Tr>
                  <Th>ID</Th>
                  <Th>Input</Th>
                  <Th>Expected Output</Th>
                  <Th modifier="fitContent">Negative?</Th>
                  <Th>Tags</Th>
                  <Th />
                </Tr>
              </Thead>
              <Tbody>
                {testCases.map((tc) => (
                  <Tr key={tc.id}>
                    <Td dataLabel="ID">
                      <code style={{ fontSize: 11 }}>{tc.id}</code>
                    </Td>
                    <Td dataLabel="Input">
                      <TextInput
                        value={tc.input}
                        onChange={(_e, v) => updateCase(tc.id, { input: v })}
                        placeholder="Test input..."
                        aria-label="Test input"
                      />
                    </Td>
                    <Td dataLabel="Expected">
                      <TextInput
                        value={tc.expected}
                        onChange={(_e, v) => updateCase(tc.id, { expected: v })}
                        placeholder="Expected output..."
                        aria-label="Expected output"
                      />
                    </Td>
                    <Td dataLabel="Negative">
                      <Checkbox
                        id={`neg-${tc.id}`}
                        isChecked={tc.negative}
                        onChange={(_e, checked) => updateCase(tc.id, { negative: checked })}
                        aria-label="Negative test"
                      />
                    </Td>
                    <Td dataLabel="Tags">
                      <TextInput
                        value={tc.tags}
                        onChange={(_e, v) => updateCase(tc.id, { tags: v })}
                        placeholder="tag1, tag2"
                        aria-label="Tags"
                      />
                    </Td>
                    <Td>
                      <Button
                        variant="plain"
                        icon={<TrashIcon />}
                        isDanger
                        aria-label="Remove test case"
                        onClick={() => removeCase(tc.id)}
                        size="sm"
                        isDisabled={testCases.length <= 1}
                      />
                    </Td>
                  </Tr>
                ))}
              </Tbody>
            </Table>
          </div>
        </WizardStep>

        {/* Step 3: Configure Judges */}
        <WizardStep name="Configure Judges" id="step-judges">
          <div style={{ maxWidth: 550 }}>
            <Title headingLevel="h3" size="lg" style={{ marginBottom: 16 }}>Configure Judges</Title>
            <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
              <div
                style={{
                  padding: "12px 16px",
                  borderRadius: 8,
                  border: "1px solid rgba(255,255,255,0.08)",
                  background: "rgba(255,255,255,0.03)",
                }}
              >
                <Checkbox
                  id="judge-keyword"
                  label="Keyword Presence"
                  isChecked={judges.keyword}
                  onChange={(_e, checked) => setJudges((p) => ({ ...p, keyword: checked }))}
                  description="Auto-checks expected output keywords in response"
                />
              </div>
              <div
                style={{
                  padding: "12px 16px",
                  borderRadius: 8,
                  border: "1px solid rgba(255,255,255,0.08)",
                  background: "rgba(56,161,105,0.04)",
                  opacity: 0.7,
                }}
              >
                <Checkbox
                  id="judge-security"
                  label="Security (mandatory)"
                  isChecked={judges.security}
                  isDisabled
                  description="Checks for prompt injection, PII leaks, and unsafe outputs"
                />
              </div>
              <div
                style={{
                  padding: "12px 16px",
                  borderRadius: 8,
                  border: "1px solid rgba(255,255,255,0.08)",
                  background: "rgba(255,255,255,0.03)",
                }}
              >
                <Checkbox
                  id="judge-llm"
                  label="LLM-as-Judge"
                  isChecked={judges.llm}
                  onChange={(_e, checked) => setJudges((p) => ({ ...p, llm: checked }))}
                  description="Uses a model to score output against criteria"
                />
                {judges.llm && (
                  <div style={{ marginTop: 8, paddingLeft: 28 }}>
                    <FormGroup label="Evaluation Criteria" fieldId="llm-criteria">
                      <TextArea
                        id="llm-criteria"
                        value={llmCriteria}
                        onChange={(_e, v) => setLlmCriteria(v)}
                        rows={3}
                        placeholder="List criteria the judge should evaluate..."
                      />
                    </FormGroup>
                  </div>
                )}
              </div>
              <div
                style={{
                  padding: "12px 16px",
                  borderRadius: 8,
                  border: "1px solid rgba(255,255,255,0.08)",
                  background: "rgba(255,255,255,0.03)",
                }}
              >
                <Checkbox
                  id="judge-trajectory"
                  label="Trajectory"
                  isChecked={judges.trajectory}
                  onChange={(_e, checked) => setJudges((p) => ({ ...p, trajectory: checked }))}
                  description="Full agent trace analysis across steps"
                />
                {judges.trajectory && (
                  <div style={{ marginTop: 8, paddingLeft: 28 }}>
                    <FormGroup label="Max Calls" fieldId="traj-max">
                      <TextInput
                        id="traj-max"
                        type="number"
                        value={String(trajectoryMaxCalls)}
                        onChange={(_e, v) => setTrajectoryMaxCalls(parseInt(v, 10) || 1)}
                      />
                    </FormGroup>
                  </div>
                )}
              </div>
            </div>
          </div>
        </WizardStep>

        {/* Step 4: Settings */}
        <WizardStep name="Settings" id="step-settings">
          <div style={{ maxWidth: 500 }}>
            <Title headingLevel="h3" size="lg" style={{ marginBottom: 16 }}>Run Settings</Title>
            <Form>
              <FormGroup label={`Trials: ${trials}`} fieldId="settings-trials">
                <Slider
                  id="settings-trials"
                  value={trials}
                  onChange={(_e, val) => setTrials(val)}
                  min={1}
                  max={10}
                  showTicks
                  areCustomStepsContinuous
                />
              </FormGroup>
              <FormGroup label="Three-condition testing" fieldId="settings-conditions" style={{ marginTop: 16 }}>
                <Checkbox
                  id="settings-conditions"
                  label="Enable no-skill / with-skill / reference comparison"
                  isChecked={conditions}
                  onChange={(_e, checked) => setConditions(checked)}
                />
                <div style={{ fontSize: 12, color: "#6b7585", marginTop: 4 }}>
                  Compares agent response without skill, with skill, and against a reference model.
                </div>
              </FormGroup>
              <FormGroup label="Model" fieldId="settings-model" style={{ marginTop: 16 }}>
                <FormSelect
                  id="settings-model"
                  value={modelSelector}
                  onChange={(_e, v) => setModelSelector(v)}
                  aria-label="Model"
                >
                  <FormSelectOption value="gpt-4o" label="GPT-4o" />
                  <FormSelectOption value="gpt-4o-mini" label="GPT-4o Mini" />
                  <FormSelectOption value="claude-sonnet-4-20250514" label="Claude Sonnet 4" />
                  <FormSelectOption value="claude-haiku-4-20250414" label="Claude Haiku 4" />
                </FormSelect>
              </FormGroup>
              <FormGroup label="Gate Mode" fieldId="settings-gate" style={{ marginTop: 16 }}>
                <FormSelect
                  id="settings-gate"
                  value={gateMode}
                  onChange={(_e, v) => setGateMode(v)}
                  aria-label="Gate mode"
                >
                  <FormSelectOption value="enforce" label="Enforce - block deployment on failure" />
                  <FormSelectOption value="warn" label="Warn - allow deployment with warning" />
                  <FormSelectOption value="skip" label="Skip - advisory only" />
                </FormSelect>
              </FormGroup>
            </Form>
          </div>
        </WizardStep>

        {/* Step 5: Preview & Run */}
        <WizardStep name="Preview & Run" id="step-preview" footer={{ nextButtonText: submitting ? "Running..." : "Run Evaluation", onNext: handleRun, isNextDisabled: submitting }}>
          <div style={{ maxWidth: 550 }}>
            <Title headingLevel="h3" size="lg" style={{ marginBottom: 16 }}>Preview</Title>
            <Grid hasGutter>
              <GridItem span={6}>
                <Card style={{ background: "rgba(255,255,255,0.03)", border: "1px solid rgba(255,255,255,0.08)", borderRadius: 10 }}>
                  <CardBody>
                    <div style={{ fontSize: 12, color: "#8b95a5", marginBottom: 4 }}>SKILL</div>
                    <div style={{ fontSize: 16, fontWeight: 600 }}>{selectedSkill || "None"}</div>
                  </CardBody>
                </Card>
              </GridItem>
              <GridItem span={6}>
                <Card style={{ background: "rgba(255,255,255,0.03)", border: "1px solid rgba(255,255,255,0.08)", borderRadius: 10 }}>
                  <CardBody>
                    <div style={{ fontSize: 12, color: "#8b95a5", marginBottom: 4 }}>TEST CASES</div>
                    <div style={{ fontSize: 16, fontWeight: 600 }}>{testCases.filter((c) => c.input.trim()).length}</div>
                  </CardBody>
                </Card>
              </GridItem>
              <GridItem span={4}>
                <Card style={{ background: "rgba(255,255,255,0.03)", border: "1px solid rgba(255,255,255,0.08)", borderRadius: 10 }}>
                  <CardBody>
                    <div style={{ fontSize: 12, color: "#8b95a5", marginBottom: 4 }}>EST. CALLS</div>
                    <div style={{ fontSize: 16, fontWeight: 600 }}>{est.calls}</div>
                  </CardBody>
                </Card>
              </GridItem>
              <GridItem span={4}>
                <Card style={{ background: "rgba(255,255,255,0.03)", border: "1px solid rgba(255,255,255,0.08)", borderRadius: 10 }}>
                  <CardBody>
                    <div style={{ fontSize: 12, color: "#8b95a5", marginBottom: 4 }}>EST. COST</div>
                    <div style={{ fontSize: 16, fontWeight: 600 }}>${est.cost}</div>
                  </CardBody>
                </Card>
              </GridItem>
              <GridItem span={4}>
                <Card style={{ background: "rgba(255,255,255,0.03)", border: "1px solid rgba(255,255,255,0.08)", borderRadius: 10 }}>
                  <CardBody>
                    <div style={{ fontSize: 12, color: "#8b95a5", marginBottom: 4 }}>EST. DURATION</div>
                    <div style={{ fontSize: 16, fontWeight: 600 }}>{est.duration}</div>
                  </CardBody>
                </Card>
              </GridItem>
            </Grid>
            <div style={{ marginTop: 16, padding: 12, borderRadius: 8, background: "rgba(91,141,239,0.06)", fontSize: 13, color: "#8b95a5", lineHeight: 1.6 }}>
              <PlayIcon style={{ marginRight: 6 }} />
              This will run <strong>{est.calls}</strong> evaluation calls using{" "}
              <strong>{Object.values(judges).filter(Boolean).length}</strong> judge(s) across{" "}
              <strong>{conditions ? "3 conditions" : "1 condition"}</strong> with{" "}
              <strong>{trials} trial(s)</strong>.
            </div>
          </div>
        </WizardStep>
      </Wizard>
    </div>
  );
};

/* ------------------------------------------------------------------ */
/*  Tab 3: Quality Dashboard                                           */
/* ------------------------------------------------------------------ */

const QualityDashboardTab = () => {
  const [metrics, setMetrics] = useState<DashboardMetrics | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetch("/api/v1/eval/dashboard")
      .then((r) => { if (!r.ok) throw new Error("api error"); return r.json(); })
      .then((d) => setMetrics(d))
      .catch(() => {
        setMetrics({
          badge_distribution: [
            { name: "Gold", count: 4 },
            { name: "Silver", count: 7 },
            { name: "Bronze", count: 3 },
            { name: "Untested", count: 12 },
            { name: "Failed", count: 2 },
          ],
          skill_lift: [
            { skill: "summarize", lift: 0.42 },
            { skill: "code-gen", lift: 0.35 },
            { skill: "sql-query", lift: 0.28 },
            { skill: "translate", lift: 0.18 },
            { skill: "classify", lift: 0.12 },
          ],
          regression_timeline: [
            { date: "2026-05-01", regressions: 0 },
            { date: "2026-05-08", regressions: 1 },
            { date: "2026-05-15", regressions: 0 },
            { date: "2026-05-22", regressions: 2 },
            { date: "2026-05-29", regressions: 1 },
          ],
          total_skills_evaluated: 28,
          avg_skill_lift: 0.27,
          regression_rate: 0.07,
          security_violations: 3,
        });
      })
      .finally(() => setLoading(false));
  }, []);

  if (loading) {
    return (
      <div style={{ textAlign: "center", padding: 40 }}>
        <Spinner size="xl" />
      </div>
    );
  }

  if (!metrics) {
    return <Alert variant="info" title="No dashboard data available" isInline />;
  }

  return (
    <div>
      {/* Stats Cards */}
      <div style={{ display: "grid", gridTemplateColumns: "repeat(4, 1fr)", gap: 16, marginBottom: 24 }}>
        <div
          style={{
            background: "rgba(255,255,255,0.04)",
            borderRadius: 12,
            padding: "20px 16px",
            border: "1px solid rgba(255,255,255,0.08)",
          }}
        >
          <div style={{ color: "#8b95a5", fontSize: 12, fontWeight: 500, marginBottom: 8 }}>Total Skills Evaluated</div>
          <div style={{ color: "#fff", fontSize: 28, fontWeight: 700 }}>{metrics.total_skills_evaluated}</div>
        </div>
        <div
          style={{
            background: "rgba(255,255,255,0.04)",
            borderRadius: 12,
            padding: "20px 16px",
            border: "1px solid rgba(255,255,255,0.08)",
          }}
        >
          <div style={{ color: "#8b95a5", fontSize: 12, fontWeight: 500, marginBottom: 8 }}>Avg Skill Lift</div>
          <div style={{ color: "#38a169", fontSize: 28, fontWeight: 700 }}>+{(metrics.avg_skill_lift * 100).toFixed(0)}%</div>
        </div>
        <div
          style={{
            background: "rgba(255,255,255,0.04)",
            borderRadius: 12,
            padding: "20px 16px",
            border: "1px solid rgba(255,255,255,0.08)",
          }}
        >
          <div style={{ color: "#8b95a5", fontSize: 12, fontWeight: 500, marginBottom: 8 }}>Regression Rate</div>
          <div style={{ color: metrics.regression_rate > 0.1 ? "#e53e3e" : "#d69e2e", fontSize: 28, fontWeight: 700 }}>
            {(metrics.regression_rate * 100).toFixed(0)}%
          </div>
        </div>
        <div
          style={{
            background: "rgba(255,255,255,0.04)",
            borderRadius: 12,
            padding: "20px 16px",
            border: "1px solid rgba(255,255,255,0.08)",
          }}
        >
          <div style={{ color: "#8b95a5", fontSize: 12, fontWeight: 500, marginBottom: 8 }}>Security Violations</div>
          <div style={{ color: metrics.security_violations > 0 ? "#e53e3e" : "#38a169", fontSize: 28, fontWeight: 700 }}>
            {metrics.security_violations}
          </div>
        </div>
      </div>

      {/* Charts */}
      <Grid hasGutter>
        {/* Badge Distribution Pie */}
        <GridItem span={4}>
          <Card
            className="stat-card"
            isFullHeight
            style={{ background: "rgba(255,255,255,0.03)", border: "1px solid rgba(255,255,255,0.08)" }}
          >
            <CardBody>
              <div style={{ fontSize: 13, fontWeight: 700, color: "#8b95a5", marginBottom: 12, textTransform: "uppercase", letterSpacing: "0.8px" }}>
                Badge Distribution
              </div>
              <ResponsiveContainer width="100%" height={240}>
                <PieChart>
                  <Pie
                    data={metrics.badge_distribution}
                    dataKey="count"
                    nameKey="name"
                    cx="50%"
                    cy="50%"
                    innerRadius={50}
                    outerRadius={80}
                    strokeWidth={0}
                    label={({ name, percent }) => `${name} ${((percent ?? 0) * 100).toFixed(0)}%`}
                  >
                    {metrics.badge_distribution.map((entry, index) => (
                      <Cell
                        key={entry.name}
                        fill={BADGE_COLORS[entry.name.toLowerCase()] ?? PIE_COLORS[index % PIE_COLORS.length]}
                      />
                    ))}
                  </Pie>
                  <Tooltip {...TOOLTIP_STYLE} />
                </PieChart>
              </ResponsiveContainer>
            </CardBody>
          </Card>
        </GridItem>

        {/* Skill Lift Bar */}
        <GridItem span={4}>
          <Card
            className="stat-card"
            isFullHeight
            style={{ background: "rgba(255,255,255,0.03)", border: "1px solid rgba(255,255,255,0.08)" }}
          >
            <CardBody>
              <div style={{ fontSize: 13, fontWeight: 700, color: "#8b95a5", marginBottom: 12, textTransform: "uppercase", letterSpacing: "0.8px" }}>
                Skill Lift (Top 5)
              </div>
              <ResponsiveContainer width="100%" height={240}>
                <BarChart data={metrics.skill_lift} layout="vertical">
                  <XAxis
                    type="number"
                    domain={[0, 0.5]}
                    tickFormatter={(v: number) => `${(v * 100).toFixed(0)}%`}
                    stroke="#4a5568"
                    tick={{ fill: "#8b95a5", fontSize: 11 }}
                  />
                  <YAxis
                    type="category"
                    dataKey="skill"
                    width={80}
                    stroke="#4a5568"
                    tick={{ fill: "#8b95a5", fontSize: 11 }}
                  />
                  <Tooltip
                    {...TOOLTIP_STYLE}
                    formatter={(value) => [`${(Number(value) * 100).toFixed(1)}%`, "Lift"]}
                  />
                  <Bar dataKey="lift" fill="#5b8def" radius={[0, 4, 4, 0]} />
                </BarChart>
              </ResponsiveContainer>
            </CardBody>
          </Card>
        </GridItem>

        {/* Regression Timeline Line */}
        <GridItem span={4}>
          <Card
            className="stat-card"
            isFullHeight
            style={{ background: "rgba(255,255,255,0.03)", border: "1px solid rgba(255,255,255,0.08)" }}
          >
            <CardBody>
              <div style={{ fontSize: 13, fontWeight: 700, color: "#8b95a5", marginBottom: 12, textTransform: "uppercase", letterSpacing: "0.8px" }}>
                Regression Timeline
              </div>
              <ResponsiveContainer width="100%" height={240}>
                <LineChart data={metrics.regression_timeline}>
                  <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.06)" />
                  <XAxis
                    dataKey="date"
                    stroke="#4a5568"
                    tick={{ fill: "#8b95a5", fontSize: 10 }}
                    tickFormatter={(v: string) => v.slice(5)}
                  />
                  <YAxis
                    stroke="#4a5568"
                    tick={{ fill: "#8b95a5", fontSize: 11 }}
                    allowDecimals={false}
                  />
                  <Tooltip {...TOOLTIP_STYLE} />
                  <Legend wrapperStyle={{ fontSize: 11, color: "#8b95a5" }} />
                  <Line
                    type="monotone"
                    dataKey="regressions"
                    stroke="#e53e3e"
                    strokeWidth={2}
                    dot={{ fill: "#e53e3e", r: 4 }}
                    name="Regressions"
                  />
                </LineChart>
              </ResponsiveContainer>
            </CardBody>
          </Card>
        </GridItem>
      </Grid>
    </div>
  );
};

/* ------------------------------------------------------------------ */
/*  Main Page                                                          */
/* ------------------------------------------------------------------ */

export const EvaluationsPage = () => {
  const [activeTab, setActiveTab] = useState<string | number>(0);

  return (
    <>
      <PageSection hasBodyWrapper={false}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
          <div>
            <Title headingLevel="h1" size="2xl">
              Eval Studio
            </Title>
            <Content component="p" style={{ marginTop: 4 }}>
              Three-condition testing with a 4-tier judge pipeline, quality badges, and regression tracking.
            </Content>
          </div>
        </div>
      </PageSection>
      <Divider />
      <PageSection hasBodyWrapper={false}>
        <Tabs
          activeKey={activeTab}
          onSelect={(_e, key) => setActiveTab(key)}
          aria-label="Eval Studio tabs"
        >
          <Tab eventKey={0} title={<TabTitleText>Eval Runs</TabTitleText>}>
            <div style={{ paddingTop: 16 }}>
              <EvalRunsTab />
            </div>
          </Tab>
          <Tab eventKey={1} title={<TabTitleText>Eval Builder</TabTitleText>}>
            <div style={{ paddingTop: 16 }}>
              <EvalBuilderTab />
            </div>
          </Tab>
          <Tab eventKey={2} title={<TabTitleText>Quality Dashboard</TabTitleText>}>
            <div style={{ paddingTop: 16 }}>
              <QualityDashboardTab />
            </div>
          </Tab>
        </Tabs>
      </PageSection>
    </>
  );
};
