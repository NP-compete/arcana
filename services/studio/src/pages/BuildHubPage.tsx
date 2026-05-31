import { useState, useEffect, useCallback } from "react";
import {
  PageSection,
  Title,
  Content,
  Grid,
  GridItem,
  Card,
  CardBody,
  Button,
  Wizard,
  WizardStep,
  Form,
  FormGroup,
  TextInput,
  TextArea,
  FormSelect,
  FormSelectOption,
  Slider,
  Alert,
  Label,
  Spinner,
  Divider,
} from "@patternfly/react-core";
import {
  RobotIcon,
  CubesIcon,
  BrainIcon,
  CodeIcon,
  UsersIcon,
  PlusCircleIcon,
  TimesIcon,
} from "@patternfly/react-icons";

/* ------------------------------------------------------------------ */
/*  Types                                                              */
/* ------------------------------------------------------------------ */

type PackageKind = "agent" | "skill" | "model" | "mcp" | "subagent";

interface EnvVar {
  key: string;
  value: string;
}

interface TestCase {
  input: string;
  expected_output: string;
}

interface CatalogItem {
  name: string;
  description?: string;
}

/* ------------------------------------------------------------------ */
/*  Constants                                                          */
/* ------------------------------------------------------------------ */

const PACKAGE_CARDS: {
  kind: PackageKind;
  label: string;
  icon: React.ReactNode;
  color: string;
  desc: string;
}[] = [
  { kind: "agent", label: "Create Agent", icon: <RobotIcon />, color: "#3b82f6", desc: "Deploy an AI agent with custom skills and tools" },
  { kind: "skill", label: "Create Skill", icon: <CubesIcon />, color: "#a855f7", desc: "Build a reusable capability for agents" },
  { kind: "model", label: "Create Model", icon: <BrainIcon />, color: "#22c55e", desc: "Register an LLM provider and model" },
  { kind: "mcp", label: "Create MCP Server", icon: <CodeIcon />, color: "#06b6d4", desc: "Connect external tools via MCP" },
  { kind: "subagent", label: "Create Subagent", icon: <UsersIcon />, color: "#f97316", desc: "Build a specialized agent as a tool" },
];

const PROVIDERS = [
  { value: "openai", label: "OpenAI" },
  { value: "anthropic", label: "Anthropic" },
  { value: "google", label: "Google" },
  { value: "vllm", label: "vLLM" },
];

const TIER_OPTIONS = [
  { value: "planning", label: "Planning" },
  { value: "functional", label: "Functional" },
  { value: "atomic", label: "Atomic" },
];

const EXEC_KINDS = [
  { value: "llm", label: "LLM" },
  { value: "code", label: "Code" },
  { value: "api", label: "API" },
];

const GATE_MODES = [
  { value: "auto", label: "Auto" },
  { value: "manual", label: "Manual" },
  { value: "ci", label: "CI Pipeline" },
];

/* ------------------------------------------------------------------ */
/*  Main Component                                                     */
/* ------------------------------------------------------------------ */

export const BuildHubPage = () => {
  const [activeWizard, setActiveWizard] = useState<PackageKind | null>(null);

  if (activeWizard === "agent") {
    return <AgentWizard onClose={() => setActiveWizard(null)} />;
  }
  if (activeWizard === "skill") {
    return <SkillWizard onClose={() => setActiveWizard(null)} />;
  }
  if (activeWizard === "model") {
    return <ModelWizard onClose={() => setActiveWizard(null)} />;
  }
  if (activeWizard === "mcp" || activeWizard === "subagent") {
    return <GenericWizard kind={activeWizard} onClose={() => setActiveWizard(null)} />;
  }

  return (
    <>
      <PageSection hasBodyWrapper={false}>
        <div style={{ textAlign: "center", padding: "40px 0 24px" }}>
          <Title headingLevel="h1" size="2xl">Build something amazing</Title>
          <Content component="p" style={{ marginTop: 8, maxWidth: 520, margin: "8px auto 0", color: "var(--pf-t--global--text--color--subtle)" }}>
            Create agents, skills, models, and MCP servers from a single hub.
            Choose a package type below to get started.
          </Content>
        </div>
      </PageSection>
      <Divider />
      <PageSection hasBodyWrapper={false}>
        <Grid hasGutter>
          {PACKAGE_CARDS.map((card) => (
            <GridItem span={4} key={card.kind}>
              <Card
                isClickable
                isFullHeight
                className="marketplace-card"
                onClick={() => setActiveWizard(card.kind)}
                style={{ cursor: "pointer" }}
              >
                <CardBody style={{ display: "flex", flexDirection: "column", alignItems: "center", textAlign: "center", padding: "32px 24px" }}>
                  <div style={{
                    width: 56,
                    height: 56,
                    borderRadius: 14,
                    display: "flex",
                    alignItems: "center",
                    justifyContent: "center",
                    fontSize: 26,
                    background: `${card.color}18`,
                    color: card.color,
                    marginBottom: 16,
                  }}>
                    {card.icon}
                  </div>
                  <div style={{ fontWeight: 700, fontSize: 16, color: "#e2e8f0", marginBottom: 8 }}>
                    {card.label}
                  </div>
                  <Content component="p" style={{ fontSize: 13, color: "#8b95a5" }}>
                    {card.desc}
                  </Content>
                </CardBody>
              </Card>
            </GridItem>
          ))}
        </Grid>
      </PageSection>
    </>
  );
};

/* ------------------------------------------------------------------ */
/*  Agent Creation Wizard                                              */
/* ------------------------------------------------------------------ */

const AgentWizard = ({ onClose }: { onClose: () => void }) => {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [team, setTeam] = useState("");

  const [provider, setProvider] = useState("openai");
  const [modelId, setModelId] = useState("");
  const [temperature, setTemperature] = useState(70);
  const [maxTokens, setMaxTokens] = useState("4096");

  const [availableSkills, setAvailableSkills] = useState<CatalogItem[]>([]);
  const [selectedSkills, setSelectedSkills] = useState<string[]>([]);
  const [availableMcp, setAvailableMcp] = useState<CatalogItem[]>([]);
  const [selectedMcp, setSelectedMcp] = useState<string[]>([]);
  const [subagents, setSubagents] = useState("");

  const [systemPrompt, setSystemPrompt] = useState("");
  const [envVars, setEnvVars] = useState<EnvVar[]>([{ key: "", value: "" }]);
  const [budgetLimit, setBudgetLimit] = useState("");

  const [submitting, setSubmitting] = useState(false);
  const [submitResult, setSubmitResult] = useState<{ type: "success" | "danger"; message: string } | null>(null);
  const [showYaml, setShowYaml] = useState(false);

  const fetchCatalog = useCallback(async () => {
    try {
      const [skillsRes, mcpRes] = await Promise.allSettled([
        fetch("/api/v1/catalog/skills"),
        fetch("/api/v1/catalog/mcp"),
      ]);
      if (skillsRes.status === "fulfilled" && skillsRes.value.ok) {
        const data = await skillsRes.value.json();
        setAvailableSkills((data.entries ?? []).map((e: Record<string, string>) => ({ name: e.name, description: e.description })));
      }
      if (mcpRes.status === "fulfilled" && mcpRes.value.ok) {
        const data = await mcpRes.value.json();
        setAvailableMcp((data.entries ?? []).map((e: Record<string, string>) => ({ name: e.name, description: e.description })));
      }
    } catch {
      /* best effort */
    }
  }, []);

  useEffect(() => {
    fetchCatalog();
  }, [fetchCatalog]);

  const handleDeploy = async () => {
    if (!name.trim()) return;
    setSubmitting(true);
    setSubmitResult(null);
    try {
      const res = await fetch("/api/v1/agents/register", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: name.trim(),
          description,
          team,
          provider,
          model_id: modelId,
          temperature: temperature / 100,
          max_tokens: parseInt(maxTokens, 10) || 4096,
          skills: selectedSkills,
          mcp_servers: selectedMcp,
          subagents: subagents.split(",").map((s) => s.trim()).filter(Boolean),
          system_prompt: systemPrompt,
          env_vars: envVars.filter((e) => e.key.trim()),
          budget_limit: budgetLimit ? parseFloat(budgetLimit) : undefined,
        }),
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error((data as Record<string, string>).error ?? `HTTP ${res.status}`);
      }
      setSubmitResult({ type: "success", message: `Agent "${name}" deployed successfully.` });
    } catch (e) {
      setSubmitResult({ type: "danger", message: e instanceof Error ? e.message : "Deploy failed" });
    } finally {
      setSubmitting(false);
    }
  };

  const addEnvVar = () => setEnvVars([...envVars, { key: "", value: "" }]);
  const removeEnvVar = (i: number) => setEnvVars(envVars.filter((_, idx) => idx !== i));
  const updateEnvVar = (i: number, field: "key" | "value", v: string) => {
    const updated = [...envVars];
    updated[i] = { ...updated[i], [field]: v };
    setEnvVars(updated);
  };

  const yamlPreview = `name: ${name || "<agent-name>"}
description: ${description || "—"}
team: ${team || "—"}
spec:
  provider: ${provider}
  model_id: ${modelId || "—"}
  temperature: ${(temperature / 100).toFixed(2)}
  max_tokens: ${maxTokens}
  skills: [${selectedSkills.join(", ")}]
  mcp_servers: [${selectedMcp.join(", ")}]
  system_prompt: |
    ${systemPrompt.split("\n").join("\n    ") || "—"}
  env_vars:
${envVars.filter((e) => e.key.trim()).map((e) => `    ${e.key}: "${e.value}"`).join("\n") || "    {}"}
  budget_limit: ${budgetLimit || "null"}
`;

  return (
    <PageSection hasBodyWrapper={false}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 16 }}>
        <Title headingLevel="h1" size="xl">Create Agent</Title>
        <Button variant="link" onClick={onClose}>Back to Build Hub</Button>
      </div>
      {submitResult && (
        <Alert variant={submitResult.type} title={submitResult.message} isInline style={{ marginBottom: 16 }} />
      )}
      <Wizard height={520} title="Create Agent" onClose={onClose}>
        <WizardStep name="Basic Info" id="agent-basic">
          <Form>
            <FormGroup label="Name" isRequired fieldId="agent-name">
              <TextInput id="agent-name" value={name} onChange={(_e, v) => setName(v)} isRequired />
            </FormGroup>
            <FormGroup label="Description" fieldId="agent-desc">
              <TextArea id="agent-desc" value={description} onChange={(_e, v) => setDescription(v)} rows={3} />
            </FormGroup>
            <FormGroup label="Team" fieldId="agent-team">
              <TextInput id="agent-team" value={team} onChange={(_e, v) => setTeam(v)} placeholder="e.g. platform-engineering" />
            </FormGroup>
          </Form>
        </WizardStep>

        <WizardStep name="Model" id="agent-model">
          <Form>
            <FormGroup label="Provider" fieldId="agent-provider">
              <FormSelect id="agent-provider" value={provider} onChange={(_e, v) => setProvider(v)} aria-label="Provider">
                {PROVIDERS.map((p) => (
                  <FormSelectOption key={p.value} value={p.value} label={p.label} />
                ))}
              </FormSelect>
            </FormGroup>
            <FormGroup label="Model ID" fieldId="agent-model-id">
              <TextInput id="agent-model-id" value={modelId} onChange={(_e, v) => setModelId(v)} placeholder="e.g. gpt-4o" />
            </FormGroup>
            <FormGroup label={`Temperature: ${(temperature / 100).toFixed(2)}`} fieldId="agent-temp">
              <Slider
                id="agent-temp"
                value={temperature}
                onChange={(_e, v) => setTemperature(v)}
                max={100}
                min={0}
                showTicks={false}
                aria-label="Temperature"
              />
            </FormGroup>
            <FormGroup label="Max Tokens" fieldId="agent-max-tokens">
              <TextInput id="agent-max-tokens" value={maxTokens} onChange={(_e, v) => setMaxTokens(v)} type="number" />
            </FormGroup>
          </Form>
        </WizardStep>

        <WizardStep name="Capabilities" id="agent-caps">
          <div style={{ display: "flex", flexDirection: "column", gap: 24 }}>
            <div>
              <div style={{ fontSize: 13, fontWeight: 700, marginBottom: 8, color: "#8b95a5", textTransform: "uppercase", letterSpacing: "0.5px" }}>Skills</div>
              {availableSkills.length > 0 ? (
                <div style={{ display: "flex", flexWrap: "wrap", gap: 8 }}>
                  {availableSkills.map((s) => {
                    const selected = selectedSkills.includes(s.name);
                    return (
                      <Button
                        key={s.name}
                        variant={selected ? "primary" : "tertiary"}
                        size="sm"
                        onClick={() => {
                          setSelectedSkills(selected
                            ? selectedSkills.filter((n) => n !== s.name)
                            : [...selectedSkills, s.name]);
                        }}
                      >
                        {s.name}
                      </Button>
                    );
                  })}
                </div>
              ) : (
                <Content component="p" style={{ fontSize: 13, color: "#8b95a5" }}>
                  No skills available. You can type skill names manually in the Review step.
                </Content>
              )}
            </div>
            <div>
              <div style={{ fontSize: 13, fontWeight: 700, marginBottom: 8, color: "#8b95a5", textTransform: "uppercase", letterSpacing: "0.5px" }}>MCP Servers</div>
              {availableMcp.length > 0 ? (
                <div style={{ display: "flex", flexWrap: "wrap", gap: 8 }}>
                  {availableMcp.map((m) => {
                    const selected = selectedMcp.includes(m.name);
                    return (
                      <Button
                        key={m.name}
                        variant={selected ? "primary" : "tertiary"}
                        size="sm"
                        onClick={() => {
                          setSelectedMcp(selected
                            ? selectedMcp.filter((n) => n !== m.name)
                            : [...selectedMcp, m.name]);
                        }}
                      >
                        {m.name}
                      </Button>
                    );
                  })}
                </div>
              ) : (
                <Content component="p" style={{ fontSize: 13, color: "#8b95a5" }}>
                  No MCP servers available.
                </Content>
              )}
            </div>
            <FormGroup label="Subagents (comma-separated)" fieldId="agent-subagents">
              <TextInput id="agent-subagents" value={subagents} onChange={(_e, v) => setSubagents(v)} placeholder="e.g. code-reviewer, data-analyst" />
            </FormGroup>
          </div>
        </WizardStep>

        <WizardStep name="Configuration" id="agent-config">
          <Form>
            <FormGroup label="System Prompt" fieldId="agent-prompt">
              <TextArea
                id="agent-prompt"
                value={systemPrompt}
                onChange={(_e, v) => setSystemPrompt(v)}
                rows={6}
                placeholder="You are a helpful assistant..."
                style={{ fontFamily: "'JetBrains Mono', 'Fira Code', monospace", fontSize: 13 }}
              />
            </FormGroup>
            <div>
              <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 8 }}>
                <div style={{ fontSize: 13, fontWeight: 700, color: "#8b95a5", textTransform: "uppercase", letterSpacing: "0.5px" }}>
                  Environment Variables
                </div>
                <Button variant="link" size="sm" icon={<PlusCircleIcon />} onClick={addEnvVar}>Add</Button>
              </div>
              {envVars.map((ev, i) => (
                <div key={i} style={{ display: "flex", gap: 8, marginBottom: 8 }}>
                  <TextInput
                    value={ev.key}
                    onChange={(_e, v) => updateEnvVar(i, "key", v)}
                    placeholder="KEY"
                    aria-label={`Env var key ${i}`}
                    style={{ flex: 1 }}
                  />
                  <TextInput
                    value={ev.value}
                    onChange={(_e, v) => updateEnvVar(i, "value", v)}
                    placeholder="value"
                    aria-label={`Env var value ${i}`}
                    style={{ flex: 2 }}
                  />
                  <Button variant="plain" aria-label="Remove env var" onClick={() => removeEnvVar(i)}>
                    <TimesIcon />
                  </Button>
                </div>
              ))}
            </div>
            <FormGroup label="Budget Limit ($)" fieldId="agent-budget">
              <TextInput id="agent-budget" value={budgetLimit} onChange={(_e, v) => setBudgetLimit(v)} type="number" placeholder="e.g. 100" />
            </FormGroup>
          </Form>
        </WizardStep>

        <WizardStep name="Review & Deploy" id="agent-review" footer={{ nextButtonText: "Deploy Agent", onNext: handleDeploy }}>
          <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
            <div style={{
              background: "rgba(255,255,255,0.04)",
              borderRadius: 12,
              padding: 20,
              border: "1px solid rgba(255,255,255,0.08)",
            }}>
              <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 12, marginBottom: 16 }}>
                <SummaryField label="Name" value={name || "—"} />
                <SummaryField label="Provider" value={provider} />
                <SummaryField label="Model" value={modelId || "—"} />
                <SummaryField label="Temperature" value={(temperature / 100).toFixed(2)} />
                <SummaryField label="Max Tokens" value={maxTokens} />
                <SummaryField label="Team" value={team || "—"} />
              </div>
              {selectedSkills.length > 0 && (
                <div style={{ marginBottom: 8 }}>
                  <span style={{ fontSize: 12, color: "#8b95a5", fontWeight: 600 }}>Skills: </span>
                  {selectedSkills.map((s) => <Label key={s} isCompact color="purple" style={{ marginRight: 4 }}>{s}</Label>)}
                </div>
              )}
              {selectedMcp.length > 0 && (
                <div>
                  <span style={{ fontSize: 12, color: "#8b95a5", fontWeight: 600 }}>MCP: </span>
                  {selectedMcp.map((m) => <Label key={m} isCompact color="teal" style={{ marginRight: 4 }}>{m}</Label>)}
                </div>
              )}
            </div>

            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
              <div style={{ fontSize: 13, fontWeight: 700, color: "#8b95a5", textTransform: "uppercase", letterSpacing: "0.5px" }}>YAML Preview</div>
              <Button variant="link" size="sm" onClick={() => setShowYaml(!showYaml)}>
                {showYaml ? "Hide" : "Show"} YAML
              </Button>
            </div>
            {showYaml && (
              <pre style={{
                margin: 0,
                padding: 16,
                background: "#0d0f14",
                borderRadius: 8,
                fontSize: 12,
                color: "#c5cdd8",
                overflow: "auto",
                maxHeight: 300,
                border: "1px solid rgba(255,255,255,0.08)",
                fontFamily: "'JetBrains Mono', 'Fira Code', 'Consolas', monospace",
              }}>
                {yamlPreview}
              </pre>
            )}

            {submitting && (
              <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
                <Spinner size="md" /> <span style={{ color: "#8b95a5", fontSize: 13 }}>Deploying agent...</span>
              </div>
            )}
          </div>
        </WizardStep>
      </Wizard>
    </PageSection>
  );
};

/* ------------------------------------------------------------------ */
/*  Skill Creation Wizard                                              */
/* ------------------------------------------------------------------ */

const SkillWizard = ({ onClose }: { onClose: () => void }) => {
  const [name, setName] = useState("");
  const [tier, setTier] = useState("functional");
  const [description, setDescription] = useState("");
  const [skillMd, setSkillMd] = useState("");
  const [execKind, setExecKind] = useState("llm");
  const [testCases, setTestCases] = useState<TestCase[]>([{ input: "", expected_output: "" }]);
  const [gateMode, setGateMode] = useState("auto");

  const [submitting, setSubmitting] = useState(false);
  const [submitResult, setSubmitResult] = useState<{ type: "success" | "danger"; message: string } | null>(null);
  const [showYaml, setShowYaml] = useState(false);

  const addTestCase = () => setTestCases([...testCases, { input: "", expected_output: "" }]);
  const removeTestCase = (i: number) => setTestCases(testCases.filter((_, idx) => idx !== i));
  const updateTestCase = (i: number, field: "input" | "expected_output", v: string) => {
    const updated = [...testCases];
    updated[i] = { ...updated[i], [field]: v };
    setTestCases(updated);
  };

  const handlePublish = async () => {
    if (!name.trim()) return;
    setSubmitting(true);
    setSubmitResult(null);
    try {
      const res = await fetch("/api/v1/skills", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: name.trim(),
          tier,
          description,
          skill_md: skillMd,
          execution_kind: execKind,
          test_cases: testCases.filter((tc) => tc.input.trim()),
          gate_mode: gateMode,
        }),
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error((data as Record<string, string>).error ?? `HTTP ${res.status}`);
      }
      setSubmitResult({ type: "success", message: `Skill "${name}" published.` });
    } catch (e) {
      setSubmitResult({ type: "danger", message: e instanceof Error ? e.message : "Publish failed" });
    } finally {
      setSubmitting(false);
    }
  };

  const yamlPreview = `name: ${name || "<skill-name>"}
tier: ${tier}
execution_kind: ${execKind}
gate_mode: ${gateMode}
description: ${description || "—"}
skill_md: |
  ${skillMd.split("\n").join("\n  ") || "—"}
test_cases:
${testCases.filter((tc) => tc.input.trim()).map((tc) => `  - input: "${tc.input}"\n    expected: "${tc.expected_output}"`).join("\n") || "  []"}
`;

  return (
    <PageSection hasBodyWrapper={false}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 16 }}>
        <Title headingLevel="h1" size="xl">Create Skill</Title>
        <Button variant="link" onClick={onClose}>Back to Build Hub</Button>
      </div>
      {submitResult && (
        <Alert variant={submitResult.type} title={submitResult.message} isInline style={{ marginBottom: 16 }} />
      )}
      <Wizard height={480} title="Create Skill" onClose={onClose}>
        <WizardStep name="Basic Info" id="skill-basic">
          <Form>
            <FormGroup label="Name" isRequired fieldId="skill-name">
              <TextInput id="skill-name" value={name} onChange={(_e, v) => setName(v)} isRequired />
            </FormGroup>
            <FormGroup label="Tier" fieldId="skill-tier">
              <FormSelect id="skill-tier" value={tier} onChange={(_e, v) => setTier(v)} aria-label="Tier">
                {TIER_OPTIONS.map((t) => <FormSelectOption key={t.value} value={t.value} label={t.label} />)}
              </FormSelect>
            </FormGroup>
            <FormGroup label="Description" fieldId="skill-desc">
              <TextArea id="skill-desc" value={description} onChange={(_e, v) => setDescription(v)} rows={3} />
            </FormGroup>
          </Form>
        </WizardStep>

        <WizardStep name="Definition" id="skill-definition">
          <Form>
            <FormGroup label="SKILL.md" fieldId="skill-md">
              <TextArea
                id="skill-md"
                value={skillMd}
                onChange={(_e, v) => setSkillMd(v)}
                rows={10}
                placeholder="# Skill Name\n\nDescribe the skill behavior in markdown..."
                style={{ fontFamily: "'JetBrains Mono', 'Fira Code', monospace", fontSize: 13 }}
              />
            </FormGroup>
            <FormGroup label="Execution Kind" fieldId="skill-exec">
              <FormSelect id="skill-exec" value={execKind} onChange={(_e, v) => setExecKind(v)} aria-label="Execution kind">
                {EXEC_KINDS.map((k) => <FormSelectOption key={k.value} value={k.value} label={k.label} />)}
              </FormSelect>
            </FormGroup>
          </Form>
        </WizardStep>

        <WizardStep name="Testing" id="skill-testing">
          <div>
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 12 }}>
              <div style={{ fontSize: 13, fontWeight: 700, color: "#8b95a5", textTransform: "uppercase", letterSpacing: "0.5px" }}>
                Test Cases
              </div>
              <Button variant="link" size="sm" icon={<PlusCircleIcon />} onClick={addTestCase}>Add</Button>
            </div>
            {testCases.map((tc, i) => (
              <div key={i} style={{
                display: "flex", gap: 8, marginBottom: 10,
                background: "rgba(255,255,255,0.03)", padding: 12, borderRadius: 8,
                border: "1px solid rgba(255,255,255,0.06)",
              }}>
                <TextInput
                  value={tc.input}
                  onChange={(_e, v) => updateTestCase(i, "input", v)}
                  placeholder="Input"
                  aria-label={`Test input ${i}`}
                  style={{ flex: 1 }}
                />
                <TextInput
                  value={tc.expected_output}
                  onChange={(_e, v) => updateTestCase(i, "expected_output", v)}
                  placeholder="Expected output"
                  aria-label={`Expected output ${i}`}
                  style={{ flex: 1 }}
                />
                <Button variant="plain" aria-label="Remove test case" onClick={() => removeTestCase(i)}>
                  <TimesIcon />
                </Button>
              </div>
            ))}
          </div>
        </WizardStep>

        <WizardStep name="Review & Publish" id="skill-review" footer={{ nextButtonText: "Publish Skill", onNext: handlePublish }}>
          <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
            <div style={{
              background: "rgba(255,255,255,0.04)",
              borderRadius: 12,
              padding: 20,
              border: "1px solid rgba(255,255,255,0.08)",
            }}>
              <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 12 }}>
                <SummaryField label="Name" value={name || "—"} />
                <SummaryField label="Tier" value={tier} />
                <SummaryField label="Execution" value={execKind} />
                <SummaryField label="Test Cases" value={String(testCases.filter((tc) => tc.input.trim()).length)} />
              </div>
            </div>
            <FormGroup label="Gate Mode" fieldId="skill-gate">
              <FormSelect id="skill-gate" value={gateMode} onChange={(_e, v) => setGateMode(v)} aria-label="Gate mode">
                {GATE_MODES.map((g) => <FormSelectOption key={g.value} value={g.value} label={g.label} />)}
              </FormSelect>
            </FormGroup>
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
              <div style={{ fontSize: 13, fontWeight: 700, color: "#8b95a5", textTransform: "uppercase", letterSpacing: "0.5px" }}>YAML Preview</div>
              <Button variant="link" size="sm" onClick={() => setShowYaml(!showYaml)}>
                {showYaml ? "Hide" : "Show"} YAML
              </Button>
            </div>
            {showYaml && (
              <pre style={{
                margin: 0, padding: 16, background: "#0d0f14", borderRadius: 8,
                fontSize: 12, color: "#c5cdd8", overflow: "auto", maxHeight: 300,
                border: "1px solid rgba(255,255,255,0.08)",
                fontFamily: "'JetBrains Mono', 'Fira Code', 'Consolas', monospace",
              }}>
                {yamlPreview}
              </pre>
            )}
            {submitting && (
              <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
                <Spinner size="md" /> <span style={{ color: "#8b95a5", fontSize: 13 }}>Publishing skill...</span>
              </div>
            )}
          </div>
        </WizardStep>
      </Wizard>
    </PageSection>
  );
};

/* ------------------------------------------------------------------ */
/*  Model Creation Wizard                                              */
/* ------------------------------------------------------------------ */

const ModelWizard = ({ onClose }: { onClose: () => void }) => {
  const [provider, setProvider] = useState("openai");
  const [modelId, setModelId] = useState("");
  const [apiKey, setApiKey] = useState("");
  const [serviceAccountJson, setServiceAccountJson] = useState("");
  const [temperature, setTemperature] = useState(70);
  const [maxTokens, setMaxTokens] = useState("4096");
  const [advancedConfig, setAdvancedConfig] = useState("");

  const [submitting, setSubmitting] = useState(false);
  const [submitResult, setSubmitResult] = useState<{ type: "success" | "danger"; message: string } | null>(null);

  const handleRegister = async () => {
    if (!modelId.trim()) return;
    setSubmitting(true);
    setSubmitResult(null);
    try {
      const res = await fetch("/api/v1/models", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          provider,
          model_id: modelId.trim(),
          api_key: apiKey,
          service_account_json: serviceAccountJson || undefined,
          temperature: temperature / 100,
          max_tokens: parseInt(maxTokens, 10) || 4096,
          advanced_config: advancedConfig || undefined,
        }),
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error((data as Record<string, string>).error ?? `HTTP ${res.status}`);
      }
      setSubmitResult({ type: "success", message: `Model "${modelId}" registered.` });
    } catch (e) {
      setSubmitResult({ type: "danger", message: e instanceof Error ? e.message : "Register failed" });
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <PageSection hasBodyWrapper={false}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 16 }}>
        <Title headingLevel="h1" size="xl">Register Model</Title>
        <Button variant="link" onClick={onClose}>Back to Build Hub</Button>
      </div>
      {submitResult && (
        <Alert variant={submitResult.type} title={submitResult.message} isInline style={{ marginBottom: 16 }} />
      )}
      <Wizard height={420} title="Register Model" onClose={onClose}>
        <WizardStep name="Provider" id="model-provider">
          <Form>
            <FormGroup label="Provider" fieldId="model-provider-select">
              <FormSelect id="model-provider-select" value={provider} onChange={(_e, v) => setProvider(v)} aria-label="Provider">
                {PROVIDERS.map((p) => <FormSelectOption key={p.value} value={p.value} label={p.label} />)}
              </FormSelect>
            </FormGroup>
            <FormGroup label="Model ID" isRequired fieldId="model-id">
              <TextInput id="model-id" value={modelId} onChange={(_e, v) => setModelId(v)} isRequired placeholder="e.g. gpt-4o" />
            </FormGroup>
          </Form>
        </WizardStep>

        <WizardStep name="Credentials" id="model-creds">
          <Form>
            <FormGroup label="API Key" fieldId="model-api-key">
              <TextInput id="model-api-key" value={apiKey} onChange={(_e, v) => setApiKey(v)} type="password" placeholder="sk-..." />
            </FormGroup>
            {provider === "google" && (
              <FormGroup label="Service Account JSON" fieldId="model-sa-json">
                <TextArea
                  id="model-sa-json"
                  value={serviceAccountJson}
                  onChange={(_e, v) => setServiceAccountJson(v)}
                  rows={6}
                  placeholder='Paste service account JSON here...'
                  style={{ fontFamily: "monospace", fontSize: 12 }}
                />
              </FormGroup>
            )}
          </Form>
        </WizardStep>

        <WizardStep name="Settings" id="model-settings" footer={{ nextButtonText: "Register Model", onNext: handleRegister }}>
          <Form>
            <FormGroup label={`Temperature: ${(temperature / 100).toFixed(2)}`} fieldId="model-temp">
              <Slider
                id="model-temp"
                value={temperature}
                onChange={(_e, v) => setTemperature(v)}
                max={100}
                min={0}
                showTicks={false}
                aria-label="Temperature"
              />
            </FormGroup>
            <FormGroup label="Max Tokens" fieldId="model-max-tokens">
              <TextInput id="model-max-tokens" value={maxTokens} onChange={(_e, v) => setMaxTokens(v)} type="number" />
            </FormGroup>
            <FormGroup label="Advanced Configuration (JSON)" fieldId="model-advanced">
              <TextArea
                id="model-advanced"
                value={advancedConfig}
                onChange={(_e, v) => setAdvancedConfig(v)}
                rows={4}
                placeholder='{"top_p": 0.95, "frequency_penalty": 0}'
                style={{ fontFamily: "monospace", fontSize: 12 }}
              />
            </FormGroup>
            {submitting && (
              <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
                <Spinner size="md" /> <span style={{ color: "#8b95a5", fontSize: 13 }}>Registering model...</span>
              </div>
            )}
          </Form>
        </WizardStep>
      </Wizard>
    </PageSection>
  );
};

/* ------------------------------------------------------------------ */
/*  Generic Wizard (MCP / Subagent)                                    */
/* ------------------------------------------------------------------ */

const GenericWizard = ({ kind, onClose }: { kind: "mcp" | "subagent"; onClose: () => void }) => {
  const isMcp = kind === "mcp";
  const title = isMcp ? "Register MCP Server" : "Create Subagent";
  const endpoint = isMcp ? "/api/v1/mcp" : "/api/v1/agents/register";

  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [config, setConfig] = useState("");

  const [submitting, setSubmitting] = useState(false);
  const [submitResult, setSubmitResult] = useState<{ type: "success" | "danger"; message: string } | null>(null);

  const handleSubmit = async () => {
    if (!name.trim()) return;
    setSubmitting(true);
    setSubmitResult(null);
    try {
      const res = await fetch(endpoint, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: name.trim(),
          description,
          config: config || undefined,
          type: isMcp ? "mcp" : "subagent",
        }),
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error((data as Record<string, string>).error ?? `HTTP ${res.status}`);
      }
      setSubmitResult({ type: "success", message: `${isMcp ? "MCP Server" : "Subagent"} "${name}" created.` });
    } catch (e) {
      setSubmitResult({ type: "danger", message: e instanceof Error ? e.message : "Failed" });
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <PageSection hasBodyWrapper={false}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 16 }}>
        <Title headingLevel="h1" size="xl">{title}</Title>
        <Button variant="link" onClick={onClose}>Back to Build Hub</Button>
      </div>
      {submitResult && (
        <Alert variant={submitResult.type} title={submitResult.message} isInline style={{ marginBottom: 16 }} />
      )}
      <Wizard height={380} title={title} onClose={onClose}>
        <WizardStep name="Details" id={`${kind}-details`}>
          <Form>
            <FormGroup label="Name" isRequired fieldId={`${kind}-name`}>
              <TextInput id={`${kind}-name`} value={name} onChange={(_e, v) => setName(v)} isRequired />
            </FormGroup>
            <FormGroup label="Description" fieldId={`${kind}-desc`}>
              <TextArea id={`${kind}-desc`} value={description} onChange={(_e, v) => setDescription(v)} rows={3} />
            </FormGroup>
          </Form>
        </WizardStep>

        <WizardStep name="Configuration" id={`${kind}-config`} footer={{ nextButtonText: isMcp ? "Register" : "Create", onNext: handleSubmit }}>
          <Form>
            <FormGroup label="Configuration (YAML or JSON)" fieldId={`${kind}-config-input`}>
              <TextArea
                id={`${kind}-config-input`}
                value={config}
                onChange={(_e, v) => setConfig(v)}
                rows={10}
                placeholder={isMcp ? "transport: stdio\ncommand: npx\nargs: [my-mcp-server]" : "role: analyst\ntools: [search, sql]\nmodel: gpt-4o"}
                style={{ fontFamily: "'JetBrains Mono', 'Fira Code', monospace", fontSize: 13 }}
              />
            </FormGroup>
            {submitting && (
              <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
                <Spinner size="md" /> <span style={{ color: "#8b95a5", fontSize: 13 }}>Creating...</span>
              </div>
            )}
          </Form>
        </WizardStep>
      </Wizard>
    </PageSection>
  );
};

/* ------------------------------------------------------------------ */
/*  Sub-components                                                     */
/* ------------------------------------------------------------------ */

const SummaryField = ({ label, value }: { label: string; value: string }) => (
  <div>
    <div style={{ fontSize: 11, color: "#8b95a5", fontWeight: 600, textTransform: "uppercase", letterSpacing: "0.5px", marginBottom: 4 }}>
      {label}
    </div>
    <div style={{ fontSize: 14, fontWeight: 600, color: "#c5cdd8" }}>{value}</div>
  </div>
);
