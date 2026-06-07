import { useState, useEffect, useCallback } from "react";
import {
  PageSection,
  Title,
  Card,
  CardBody,
  Button,
  Grid,
  GridItem,
  Content,
  Divider,
  Label,
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
  Alert,
  Spinner,
} from "@patternfly/react-core";
import { Table, Thead, Tr, Th, Tbody, Td } from "@patternfly/react-table";
import {
  CubesIcon,
  PlusCircleIcon,
  CheckCircleIcon,
  ExclamationCircleIcon,
  StarIcon,
} from "@patternfly/react-icons";
import { ShareBadge } from "../components/ShareBadge";
import {
  DescriptionList,
  DescriptionListGroup,
  DescriptionListTerm,
  DescriptionListDescription,
} from "@patternfly/react-core";

interface CatalogEntry {
  name: string;
  type: string;
  version: string;
  description: string;
  metadata?: Record<string, unknown>;
  created_at?: string;
}

interface SkillMemoryEntry {
  timestamp: string;
  agent_id: string;
  content: string;
}

interface EvalHistoryEntry {
  run_id: string;
  avg_score: number;
  pass_rate: number;
  badge: string;
  test_count: number;
  regression: boolean;
  created_at: string;
}

const badgeColor = (badge: string): "gold" | "orange" | "blue" | "red" | "grey" => {
  switch (badge?.toLowerCase()) {
    case "gold": return "gold";
    case "silver": return "orange";
    case "bronze": return "blue";
    case "failed": return "red";
    default: return "grey";
  }
};

const TIER_OPTIONS = [
  { value: "planning", label: "Planning" },
  { value: "functional", label: "Functional" },
  { value: "atomic", label: "Atomic" },
];

const SKILL_TIERS = [
  {
    tier: "Planning",
    color: "#805ad5" as const,
    desc: "High-level orchestration skills that decompose goals into sub-tasks",
    examples: ["task-decompose", "multi-step-plan", "goal-refine"],
  },
  {
    tier: "Functional",
    color: "#3182ce" as const,
    desc: "Domain-specific capabilities like code generation or data analysis",
    examples: ["code-gen", "sql-query", "summarize", "translate"],
  },
  {
    tier: "Atomic",
    color: "#38a169" as const,
    desc: "Low-level tool wrappers: API calls, file I/O, database queries",
    examples: ["http-get", "file-read", "db-query", "shell-exec"],
  },
];

const tierLabelColor = (tier: string): "purple" | "blue" | "green" | "grey" => {
  switch (tier?.toLowerCase()) {
    case "planning":
      return "purple";
    case "functional":
      return "blue";
    case "atomic":
      return "green";
    default:
      return "grey";
  }
};

export const SkillsPage = () => {
  const [skills, setSkills] = useState<CatalogEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [fetchError, setFetchError] = useState<string | null>(null);

  const [modalOpen, setModalOpen] = useState(false);
  const [name, setName] = useState("");
  const [tier, setTier] = useState(TIER_OPTIONS[1].value);
  const [version, setVersion] = useState("1.0.0");
  const [description, setDescription] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [submitSuccess, setSubmitSuccess] = useState<string | null>(null);

  const [expandedSkill, setExpandedSkill] = useState<string | null>(null);
  const [skillMemory, setSkillMemory] = useState<SkillMemoryEntry[]>([]);
  const [evalHistory, setEvalHistory] = useState<EvalHistoryEntry[]>([]);
  const [skillBadge, setSkillBadge] = useState<string>("untested");
  const [detailLoading, setDetailLoading] = useState(false);

  const toggleSkillDetail = async (skillName: string) => {
    if (expandedSkill === skillName) {
      setExpandedSkill(null);
      return;
    }
    setExpandedSkill(skillName);
    setDetailLoading(true);
    try {
      const [memRes, evalRes, skillRes] = await Promise.all([
        fetch(`/api/v1/skills/${skillName}/memory`).then(r => r.ok ? r.json() : { memory: [] }),
        fetch(`/api/v1/skills/${skillName}/eval-history`).then(r => r.ok ? r.json() : { eval_history: [] }),
        fetch(`/api/v1/skills/${skillName}`).then(r => r.ok ? r.json() : {}),
      ]);
      setSkillMemory(memRes.memory ?? []);
      setEvalHistory(evalRes.eval_history ?? []);
      setSkillBadge(skillRes.quality_badge ?? "untested");
    } catch {
      setSkillMemory([]);
      setEvalHistory([]);
      setSkillBadge("untested");
    } finally {
      setDetailLoading(false);
    }
  };

  const fetchSkills = useCallback(async () => {
    try {
      const res = await fetch("/api/v1/catalog/skills");
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      setSkills(data.entries ?? []);
      setFetchError(null);
    } catch (e) {
      setFetchError(e instanceof Error ? e.message : "Failed to load skills");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchSkills();
  }, [fetchSkills]);

  const openModal = () => {
    setName("");
    setTier(TIER_OPTIONS[1].value);
    setVersion("1.0.0");
    setDescription("");
    setSubmitError(null);
    setSubmitSuccess(null);
    setModalOpen(true);
  };

  const closeModal = () => {
    setModalOpen(false);
    setSubmitError(null);
  };

  const handleRegister = async () => {
    if (!name.trim()) {
      setSubmitError("Skill name is required");
      return;
    }
    setSubmitting(true);
    setSubmitError(null);
    setSubmitSuccess(null);
    try {
      const res = await fetch("/api/v1/catalog/skills", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: name.trim(),
          version: version.trim() || "1.0.0",
          description: description.trim(),
          metadata: { tier },
        }),
      });
      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error ?? `HTTP ${res.status}`);
      }
      setSubmitSuccess(`Skill "${data.name}" registered successfully`);
      await fetchSkills();
    } catch (e) {
      setSubmitError(e instanceof Error ? e.message : "Failed to register skill");
    } finally {
      setSubmitting(false);
    }
  };

  const hasSkills = skills.length > 0;

  return (
    <>
      <PageSection hasBodyWrapper={false}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
          <div>
            <Title headingLevel="h1" size="2xl">Skills</Title>
            <Content component="p" style={{ marginTop: 4 }}>
              3-tier skill catalog with automated evaluation and quality badges.
            </Content>
          </div>
          <Button variant="primary" icon={<PlusCircleIcon />} onClick={openModal}>
            Register Skill
          </Button>
        </div>
      </PageSection>
      <Divider />
      <PageSection hasBodyWrapper={false}>
        {fetchError && (
          <Alert variant="warning" title="Could not load skills" isInline style={{ marginBottom: 16 }}>
            {fetchError}
          </Alert>
        )}

        {loading ? (
          <div style={{ textAlign: "center", padding: 40 }}>
            <Spinner size="xl" />
          </div>
        ) : hasSkills ? (
          <>
            <div className="section-title">Registered Skills ({skills.length})</div>
            <Table aria-label="Registered skills" variant="compact">
              <Thead>
                <Tr>
                  <Th>Name</Th>
                  <Th>Tier</Th>
                  <Th>Version</Th>
                  <Th>Sharing</Th>
                  <Th>Description</Th>
                </Tr>
              </Thead>
              <Tbody>
                {skills.map((skill) => {
                  const rawTier = skill.metadata?.tier;
                  const skillTier = typeof rawTier === "string" ? rawTier : "—";
                  const isExpanded = expandedSkill === skill.name;
                  return (
                    <>
                      <Tr
                        key={skill.name}
                        isClickable
                        onRowClick={() => toggleSkillDetail(skill.name)}
                        style={{ cursor: "pointer" }}
                      >
                        <Td dataLabel="Name">
                          <Button variant="link" isInline onClick={() => toggleSkillDetail(skill.name)}>
                            {skill.name}
                          </Button>
                        </Td>
                        <Td dataLabel="Tier">
                          <Label color={tierLabelColor(skillTier)} isCompact>
                            {skillTier}
                          </Label>
                        </Td>
                        <Td dataLabel="Version">{skill.version}</Td>
                        <Td dataLabel="Sharing">
                          <ShareBadge assetType="skill" assetName={skill.name} compact />
                        </Td>
                        <Td dataLabel="Description">{skill.description}</Td>
                      </Tr>
                      {isExpanded && (
                        <Tr key={`${skill.name}-detail`}>
                          <Td colSpan={5}>
                            {detailLoading ? (
                              <div style={{ textAlign: "center", padding: 24 }}>
                                <Spinner size="lg" />
                              </div>
                            ) : (
                              <Grid hasGutter style={{ padding: "16px 8px" }}>
                                <GridItem span={4}>
                                  <Card isPlain>
                                    <CardBody>
                                      <div style={{ fontWeight: 600, marginBottom: 12 }}>Quality Badge</div>
                                      <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
                                        <StarIcon color={
                                          skillBadge === "gold" ? "#d4a017"
                                            : skillBadge === "silver" ? "#a0a0a0"
                                            : skillBadge === "bronze" ? "#cd7f32"
                                            : "#6b7585"
                                        } />
                                        <Label color={badgeColor(skillBadge)} isCompact>
                                          {skillBadge}
                                        </Label>
                                      </div>
                                    </CardBody>
                                  </Card>
                                </GridItem>
                                <GridItem span={4}>
                                  <Card isPlain>
                                    <CardBody>
                                      <div style={{ fontWeight: 600, marginBottom: 12 }}>
                                        Experiential Memory ({skillMemory.length})
                                      </div>
                                      {skillMemory.length > 0 ? (
                                        <div style={{ maxHeight: 200, overflow: "auto", fontSize: 13 }}>
                                          {skillMemory.slice(0, 5).map((m, i) => (
                                            <div key={i} style={{ marginBottom: 8, padding: "4px 0", borderBottom: "1px solid rgba(255,255,255,0.06)" }}>
                                              <span style={{ color: "var(--pf-t--global--text--color--subtle)", fontSize: 11 }}>
                                                {m.timestamp} · {m.agent_id}
                                              </span>
                                              <div>{m.content}</div>
                                            </div>
                                          ))}
                                        </div>
                                      ) : (
                                        <Content component="p" style={{ color: "var(--pf-t--global--text--color--subtle)", fontSize: 13 }}>
                                          No memory entries yet.
                                        </Content>
                                      )}
                                    </CardBody>
                                  </Card>
                                </GridItem>
                                <GridItem span={4}>
                                  <Card isPlain>
                                    <CardBody>
                                      <div style={{ fontWeight: 600, marginBottom: 12 }}>
                                        Eval History ({evalHistory.length})
                                      </div>
                                      {evalHistory.length > 0 ? (
                                        <div style={{ maxHeight: 200, overflow: "auto", fontSize: 13 }}>
                                          {evalHistory.slice(0, 5).map((e) => (
                                            <div key={e.run_id} style={{ marginBottom: 8, padding: "4px 0", borderBottom: "1px solid rgba(255,255,255,0.06)" }}>
                                              <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
                                                <Label color={badgeColor(e.badge)} isCompact>{e.badge}</Label>
                                                <span>{(e.avg_score * 100).toFixed(1)}%</span>
                                                {e.regression && (
                                                  <Label color="red" isCompact>
                                                    <ExclamationCircleIcon /> regression
                                                  </Label>
                                                )}
                                              </div>
                                              <span style={{ color: "var(--pf-t--global--text--color--subtle)", fontSize: 11 }}>
                                                {e.test_count} tests · {e.created_at ? new Date(e.created_at).toLocaleDateString() : "—"}
                                              </span>
                                            </div>
                                          ))}
                                        </div>
                                      ) : (
                                        <Content component="p" style={{ color: "var(--pf-t--global--text--color--subtle)", fontSize: 13 }}>
                                          No evaluations yet.
                                        </Content>
                                      )}
                                    </CardBody>
                                  </Card>
                                </GridItem>
                              </Grid>
                            )}
                          </Td>
                        </Tr>
                      )}
                    </>
                  );
                })}
              </Tbody>
            </Table>
          </>
        ) : (
          <div className="arcana-empty-state" style={{ paddingBottom: 32 }}>
            <div className="arcana-empty-icon">
              <CubesIcon />
            </div>
            <Title headingLevel="h2" size="xl" style={{ marginBottom: 8 }}>
              Skill registry is empty
            </Title>
            <Content component="p" style={{ maxWidth: 480, margin: "0 auto", color: "var(--pf-t--global--text--color--subtle)" }}>
              Register your first skill to build your agent&apos;s capabilities.
              Skills are auto-evaluated and assigned Gold, Silver, or Bronze badges.
            </Content>
          </div>
        )}

        <div className="section-title">Skill Tier Architecture</div>
        <Grid hasGutter>
          {SKILL_TIERS.map((t) => (
            <GridItem span={4} key={t.tier}>
              <Card className="stat-card" isFullHeight>
                <CardBody>
                  <div style={{ display: "flex", alignItems: "center", gap: 10, marginBottom: 12 }}>
                    <div style={{
                      width: 12, height: 12, borderRadius: 4,
                      background: t.color,
                    }} />
                    <span style={{ fontWeight: 700, fontSize: 16 }}>{t.tier}</span>
                  </div>
                  <Content component="p" style={{ fontSize: 13, marginBottom: 14, color: "var(--pf-t--global--text--color--subtle)" }}>
                    {t.desc}
                  </Content>
                  <div style={{ display: "flex", gap: 6, flexWrap: "wrap" }}>
                    {t.examples.map((e) => (
                      <Label color="grey" isCompact key={e}>{e}</Label>
                    ))}
                  </div>
                </CardBody>
              </Card>
            </GridItem>
          ))}
        </Grid>
      </PageSection>

      <Modal
        variant={ModalVariant.medium}
        isOpen={modalOpen}
        onClose={closeModal}
        aria-labelledby="register-skill-title"
      >
        <ModalHeader title="Register Skill" labelId="register-skill-title" />
        <ModalBody>
          {submitError && (
            <Alert variant="danger" title="Registration failed" isInline style={{ marginBottom: 16 }}>
              {submitError}
            </Alert>
          )}
          {submitSuccess && (
            <Alert variant="success" title="Success" isInline style={{ marginBottom: 16 }}>
              {submitSuccess}
            </Alert>
          )}
          <Form id="register-skill-form">
            <FormGroup label="Name" isRequired fieldId="skill-name">
              <TextInput
                id="skill-name"
                value={name}
                onChange={(_e, v) => setName(v)}
                isRequired
              />
            </FormGroup>
            <FormGroup label="Tier" fieldId="skill-tier">
              <FormSelect
                id="skill-tier"
                value={tier}
                onChange={(_e, v) => setTier(v)}
                aria-label="Tier"
              >
                {TIER_OPTIONS.map((t) => (
                  <FormSelectOption key={t.value} value={t.value} label={t.label} />
                ))}
              </FormSelect>
            </FormGroup>
            <FormGroup label="Version" fieldId="skill-version">
              <TextInput
                id="skill-version"
                value={version}
                onChange={(_e, v) => setVersion(v)}
              />
            </FormGroup>
            <FormGroup label="Description" fieldId="skill-description">
              <TextArea
                id="skill-description"
                value={description}
                onChange={(_e, v) => setDescription(v)}
                rows={3}
              />
            </FormGroup>
          </Form>
        </ModalBody>
        <ModalFooter>
          <Button
            variant="primary"
            onClick={handleRegister}
            isDisabled={submitting}
            isLoading={submitting}
          >
            Register
          </Button>
          <Button variant="link" onClick={closeModal}>
            Cancel
          </Button>
        </ModalFooter>
      </Modal>
    </>
  );
};
