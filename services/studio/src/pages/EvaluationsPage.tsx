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
  Checkbox,
  Alert,
  Spinner,
} from "@patternfly/react-core";
import { Table, Thead, Tr, Th, Tbody, Td } from "@patternfly/react-table";
import { ChartBarIcon, PlusCircleIcon } from "@patternfly/react-icons";

interface EvalRun {
  run_id: string;
  skill_ref: string;
  status: string;
  badge?: { level: string } | null;
}

const JUDGE_TIERS = [
  { tier: "Deterministic", desc: "Exact match, regex, JSON schema validators", color: "#38a169", badge: "Fastest", key: "deterministic" },
  { tier: "Script", desc: "Custom Python/Go scoring functions", color: "#3182ce", badge: "Flexible", key: "script" },
  { tier: "LLM", desc: "GPT-4o / Claude judge with rubric scoring", color: "#805ad5", badge: "Nuanced", key: "llm" },
  { tier: "Trajectory", desc: "Full agent trace analysis across steps", color: "#d69e2e", badge: "Deepest", key: "trajectory" },
];

const BADGES = [
  { name: "Gold", color: "#d69e2e", bg: "#fefcbf", criteria: ">95% pass, 5+ security judges" },
  { name: "Silver", color: "#718096", bg: "#e2e8f0", criteria: ">85% pass, 3+ judges" },
  { name: "Bronze", color: "#c05621", bg: "#feebc8", criteria: ">70% pass, basic coverage" },
  { name: "Untested", color: "#a0aec0", bg: "#f7fafc", criteria: "No eval suite configured" },
];

const JUDGE_OPTIONS = [
  { key: "deterministic", label: "Deterministic" },
  { key: "script", label: "Script" },
  { key: "llm", label: "LLM" },
  { key: "trajectory", label: "Trajectory" },
] as const;

const statusColor = (status: string): "green" | "blue" | "orange" | "red" | "grey" => {
  switch (status) {
    case "completed":
      return "green";
    case "running":
      return "blue";
    case "pending":
      return "orange";
    case "failed":
      return "red";
    default:
      return "grey";
  }
};

const badgeColor = (level: string): "yellow" | "blue" | "orange" | "grey" | "red" => {
  switch (level?.toLowerCase()) {
    case "gold":
      return "yellow";
    case "silver":
      return "blue";
    case "bronze":
      return "orange";
    case "failed":
      return "red";
    default:
      return "grey";
  }
};

export const EvaluationsPage = () => {
  const [runs, setRuns] = useState<EvalRun[]>([]);
  const [loading, setLoading] = useState(true);
  const [fetchError, setFetchError] = useState<string | null>(null);

  const [modalOpen, setModalOpen] = useState(false);
  const [skillRef, setSkillRef] = useState("");
  const [judges, setJudges] = useState<Record<string, boolean>>({
    deterministic: true,
    script: false,
    llm: true,
    trajectory: false,
  });
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [submitSuccess, setSubmitSuccess] = useState<string | null>(null);

  const fetchRuns = useCallback(async () => {
    try {
      const res = await fetch("/api/v1/eval/runs");
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      setRuns(data.runs ?? []);
      setFetchError(null);
    } catch (e) {
      setFetchError(e instanceof Error ? e.message : "Failed to load eval runs");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchRuns();
  }, [fetchRuns]);

  const openModal = () => {
    setSkillRef("");
    setJudges({ deterministic: true, script: false, llm: true, trajectory: false });
    setSubmitError(null);
    setSubmitSuccess(null);
    setModalOpen(true);
  };

  const closeModal = () => {
    setModalOpen(false);
    setSubmitError(null);
  };

  const handleRunEval = async () => {
    if (!skillRef.trim()) {
      setSubmitError("Skill reference is required");
      return;
    }
    const selectedJudges = JUDGE_OPTIONS.filter((j) => judges[j.key]).map((j) => ({
      name: `${j.key}_judge`,
      tier: j.key,
      weight: 1 / Math.max(JUDGE_OPTIONS.filter((o) => judges[o.key]).length, 1),
    }));

    if (selectedJudges.length === 0) {
      setSubmitError("Select at least one judge type");
      return;
    }

    setSubmitting(true);
    setSubmitError(null);
    setSubmitSuccess(null);
    try {
      const res = await fetch("/api/v1/eval/run", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          skill_ref: skillRef.trim(),
          cases: [
            { id: "case-1", input: "Sample input for evaluation", expected: "Sample input for evaluation" },
          ],
          judges: selectedJudges,
        }),
      });
      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.detail ?? data.error ?? `HTTP ${res.status}`);
      }
      setSubmitSuccess(`Evaluation run ${data.run_id} started`);
      await fetchRuns();
    } catch (e) {
      setSubmitError(e instanceof Error ? e.message : "Failed to start evaluation");
    } finally {
      setSubmitting(false);
    }
  };

  const hasRuns = runs.length > 0;

  return (
    <>
      <PageSection hasBodyWrapper={false}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
          <div>
            <Title headingLevel="h1" size="2xl">Evaluations</Title>
            <Content component="p" style={{ marginTop: 4 }}>
              Three-condition testing with a 4-tier judge pipeline and quality badges.
            </Content>
          </div>
          <Button variant="primary" icon={<PlusCircleIcon />} onClick={openModal}>
            Run Evaluation
          </Button>
        </div>
      </PageSection>
      <Divider />
      <PageSection hasBodyWrapper={false}>
        {fetchError && (
          <Alert variant="warning" title="Could not load eval runs" isInline style={{ marginBottom: 16 }}>
            {fetchError}
          </Alert>
        )}

        {loading ? (
          <div style={{ textAlign: "center", padding: 40 }}>
            <Spinner size="xl" />
          </div>
        ) : hasRuns ? (
          <>
            <div className="section-title">Evaluation Runs ({runs.length})</div>
            <Table aria-label="Evaluation runs" variant="compact">
              <Thead>
                <Tr>
                  <Th>Run ID</Th>
                  <Th>Skill</Th>
                  <Th>Status</Th>
                  <Th>Badge</Th>
                </Tr>
              </Thead>
              <Tbody>
                {runs.map((run) => (
                  <Tr key={run.run_id}>
                    <Td dataLabel="Run ID">
                      <code style={{ fontSize: 12 }}>{run.run_id.slice(0, 8)}...</code>
                    </Td>
                    <Td dataLabel="Skill">{run.skill_ref}</Td>
                    <Td dataLabel="Status">
                      <Label color={statusColor(run.status)} isCompact>
                        {run.status}
                      </Label>
                    </Td>
                    <Td dataLabel="Badge">
                      {run.badge ? (
                        <Label color={badgeColor(run.badge.level)} isCompact>
                          {run.badge.level}
                        </Label>
                      ) : (
                        <Label color="grey" isCompact>—</Label>
                      )}
                    </Td>
                  </Tr>
                ))}
              </Tbody>
            </Table>
          </>
        ) : (
          <div className="arcana-empty-state" style={{ paddingBottom: 32 }}>
            <div className="arcana-empty-icon">
              <ChartBarIcon />
            </div>
            <Title headingLevel="h2" size="xl" style={{ marginBottom: 8 }}>
              No evaluation suites yet
            </Title>
            <Content component="p" style={{ maxWidth: 480, margin: "0 auto", color: "var(--pf-t--global--text--color--subtle)" }}>
              Create an ArcanaEvalSuite CR to define test cases, judges, and quality gates for your skills.
            </Content>
          </div>
        )}

        <div className="section-title">Judge Pipeline</div>
        <Grid hasGutter>
          {JUDGE_TIERS.map((j, i) => (
            <GridItem span={3} key={j.tier}>
              <Card className="stat-card" isFullHeight>
                <CardBody>
                  <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 12 }}>
                    <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
                      <div style={{
                        width: 28, height: 28, borderRadius: 8,
                        background: `${j.color}15`, color: j.color,
                        display: "flex", alignItems: "center", justifyContent: "center",
                        fontWeight: 800, fontSize: 14,
                      }}>
                        {i + 1}
                      </div>
                      <span style={{ fontWeight: 700, fontSize: 14 }}>{j.tier}</span>
                    </div>
                    <Label isCompact color="blue">{j.badge}</Label>
                  </div>
                  <Content component="p" style={{ fontSize: 13, color: "var(--pf-t--global--text--color--subtle)" }}>
                    {j.desc}
                  </Content>
                </CardBody>
              </Card>
            </GridItem>
          ))}
        </Grid>

        <div className="section-title" style={{ marginTop: 32 }}>Quality Badges</div>
        <div style={{ display: "flex", gap: 16, flexWrap: "wrap" }}>
          {BADGES.map((b) => (
            <Card className="stat-card" key={b.name} style={{ flex: 1, minWidth: 200 }}>
              <CardBody>
                <div style={{ display: "flex", alignItems: "center", gap: 10, marginBottom: 8 }}>
                  <div style={{
                    width: 24, height: 24, borderRadius: "50%",
                    background: b.bg, border: `2px solid ${b.color}`,
                  }} />
                  <span style={{ fontWeight: 700, fontSize: 16, color: b.color }}>{b.name}</span>
                </div>
                <Content component="p" style={{ fontSize: 12, color: "var(--pf-t--global--text--color--subtle)" }}>
                  {b.criteria}
                </Content>
              </CardBody>
            </Card>
          ))}
        </div>
      </PageSection>

      <Modal
        variant={ModalVariant.medium}
        isOpen={modalOpen}
        onClose={closeModal}
        aria-labelledby="run-eval-title"
      >
        <ModalHeader title="Run Evaluation" labelId="run-eval-title" />
        <ModalBody>
          {submitError && (
            <Alert variant="danger" title="Evaluation failed" isInline style={{ marginBottom: 16 }}>
              {submitError}
            </Alert>
          )}
          {submitSuccess && (
            <Alert variant="success" title="Success" isInline style={{ marginBottom: 16 }}>
              {submitSuccess}
            </Alert>
          )}
          <Form id="run-eval-form">
            <FormGroup label="Skill reference" isRequired fieldId="eval-skill-ref">
              <TextInput
                id="eval-skill-ref"
                value={skillRef}
                onChange={(_e, v) => setSkillRef(v)}
                placeholder="summarize"
                isRequired
              />
            </FormGroup>
            <FormGroup label="Judge types" fieldId="eval-judges">
              {JUDGE_OPTIONS.map((j) => (
                <Checkbox
                  key={j.key}
                  id={`judge-${j.key}`}
                  label={j.label}
                  isChecked={judges[j.key]}
                  onChange={(_e, checked) =>
                    setJudges((prev) => ({ ...prev, [j.key]: checked }))
                  }
                />
              ))}
            </FormGroup>
          </Form>
        </ModalBody>
        <ModalFooter>
          <Button
            variant="primary"
            onClick={handleRunEval}
            isDisabled={submitting}
            isLoading={submitting}
          >
            Run
          </Button>
          <Button variant="link" onClick={closeModal}>
            Cancel
          </Button>
        </ModalFooter>
      </Modal>
    </>
  );
};
