import { useState, useEffect, useCallback } from "react";
import {
  PageSection,
  Title,
  Button,
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
  FormSelect,
  FormSelectOption,
  Alert,
  Spinner,
} from "@patternfly/react-core";
import {
  RobotIcon,
  PlusCircleIcon,
  CodeIcon,
} from "@patternfly/react-icons";
import { ARCANA_AGENT_YAML } from "../constants/arcanaAgentYaml";
import { useNavigate } from "react-router-dom";

interface Agent {
  name: string;
  agent_type: string;
  capabilities: string[];
  protocols: string[];
  status: string;
  registered_at?: string;
}

const STATUS_MAP: Record<string, { color: "green" | "orange" | "blue" | "red" | "grey"; label: string }> = {
  active: { color: "green", label: "Running" },
  busy: { color: "orange", label: "Busy" },
  idle: { color: "blue", label: "Sleeping" },
  offline: { color: "red", label: "Crashed" },
};

const MODEL_OPTIONS = ["gpt-4o", "claude-sonnet", "gpt-4o-mini"];

const TEMPLATES = [
  {
    name: "Conversational",
    agentName: "conversational-agent",
    desc: "Chat-based agent with memory and skill execution",
    model: "gpt-4o",
    skills: "search, summarize, code-gen",
  },
  {
    name: "Code Assistant",
    agentName: "code-assistant",
    desc: "Code review, generation, and refactoring",
    model: "claude-sonnet",
    skills: "code-review, refactor, test-gen",
  },
  {
    name: "Data Pipeline",
    agentName: "data-pipeline-agent",
    desc: "ETL workflows with HITL gates",
    model: "gpt-4o",
    skills: "sql-gen, schema-validate, data-profile",
  },
  {
    name: "Research Agent",
    agentName: "research-agent",
    desc: "Multi-step research with fact-checking",
    model: "claude-sonnet",
    skills: "research, summarize, fact-check",
  },
];

export const AgentsPage = () => {
  const navigate = useNavigate();
  const [agents, setAgents] = useState<Agent[]>([]);
  const [loading, setLoading] = useState(true);
  const [fetchError, setFetchError] = useState<string | null>(null);

  const [deployOpen, setDeployOpen] = useState(false);
  const [yamlOpen, setYamlOpen] = useState(false);
  const [form, setForm] = useState({ name: "", model: MODEL_OPTIONS[0], skills: "" });
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [submitSuccess, setSubmitSuccess] = useState<string | null>(null);

  const fetchAgents = useCallback(async () => {
    try {
      const res = await fetch("/api/v1/agents");
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      setAgents(data.agents ?? []);
      setFetchError(null);
    } catch (e) {
      setFetchError(e instanceof Error ? e.message : "Failed to load agents");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchAgents();
  }, [fetchAgents]);

  const openDeploy = (prefill?: Partial<typeof form>) => {
    setForm({ name: "", model: MODEL_OPTIONS[0], skills: "", ...prefill });
    setSubmitError(null);
    setSubmitSuccess(null);
    setDeployOpen(true);
  };

  const handleDeploy = async () => {
    if (!form.name.trim()) {
      setSubmitError("Agent name is required");
      return;
    }
    const capabilities = form.skills
      .split(",")
      .map((s) => s.trim())
      .filter(Boolean);
    capabilities.push(`model:${form.model}`);

    setSubmitting(true);
    setSubmitError(null);
    setSubmitSuccess(null);
    try {
      const res = await fetch("/api/v1/agents/register", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: form.name.trim(),
          agent_type: "create_deep_agent",
          capabilities,
          protocols: ["a2a", "acp"],
          status: "active",
          deep_config: {
            world_model: true,
            skill_graph: true,
            blueprint: "",
            memory_policy: "tri-scope",
            sub_agents: [],
            hitl_enabled: false,
            self_improve: true,
            system_prompt: "",
            temperature: 0.0,
            max_tokens: 8192,
            model_call_limit: 50,
            tool_call_limit: 200,
          },
        }),
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error ?? `HTTP ${res.status}`);
      setSubmitSuccess(`Agent "${data.name}" deployed`);
      await fetchAgents();
    } catch (e) {
      setSubmitError(e instanceof Error ? e.message : "Deploy failed");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <>
      <PageSection hasBodyWrapper={false}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
          <div>
            <Title headingLevel="h1" size="2xl">Agents</Title>
            <Content component="p" style={{ marginTop: 4 }}>
              All your deployed agents in one place.
            </Content>
          </div>
          <Button variant="primary" icon={<PlusCircleIcon />} onClick={() => openDeploy()}>
            Deploy agent
          </Button>
        </div>
      </PageSection>
      <Divider />
      <PageSection hasBodyWrapper={false}>
        {fetchError && (
          <Alert variant="warning" title="Could not load agents" isInline style={{ marginBottom: 16 }}>
            {fetchError}
          </Alert>
        )}

        {loading ? (
          <div style={{ textAlign: "center", padding: 40 }}>
            <Spinner size="xl" />
          </div>
        ) : agents.length > 0 ? (
          <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
            {agents.map((agent) => {
              const st = STATUS_MAP[agent.status] ?? STATUS_MAP.offline;
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
                    padding: "16px 20px",
                    background: "var(--arcana-card-bg)",
                    borderRadius: 10,
                    border: "1px solid var(--arcana-card-border)",
                    cursor: "pointer",
                    transition: "background 0.15s",
                  }}
                  onMouseEnter={(e) => { (e.currentTarget as HTMLDivElement).style.background = "var(--arcana-hover-bg)"; }}
                  onMouseLeave={(e) => { (e.currentTarget as HTMLDivElement).style.background = "var(--arcana-card-bg)"; }}
                >
                  <span style={{
                    width: 10,
                    height: 10,
                    borderRadius: "50%",
                    background: st.color === "green" ? "#22c55e" : st.color === "orange" ? "#f59e0b" : st.color === "red" ? "#ef4444" : "#8b95a5",
                    flexShrink: 0,
                  }} />
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ fontSize: 15, fontWeight: 600, color: "var(--arcana-text)" }}>
                      {agent.name}
                    </div>
                    <div style={{ fontSize: 12, color: "var(--arcana-text-muted)" }}>
                      {model ?? "—"} &middot; {st.label}
                    </div>
                  </div>
                  <div style={{ display: "flex", gap: 4, flexWrap: "wrap" }}>
                    {skills.slice(0, 4).map((s) => (
                      <Label key={s} isCompact color="grey">{s}</Label>
                    ))}
                    {skills.length > 4 && (
                      <Label isCompact color="grey">+{skills.length - 4}</Label>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        ) : (
          <div className="arcana-empty-state">
            <div className="arcana-empty-icon">
              <RobotIcon />
            </div>
            <Title headingLevel="h2" size="xl" style={{ marginBottom: 8 }}>
              No agents yet
            </Title>
            <Content component="p" style={{ maxWidth: 480, margin: "0 auto 32px auto", color: "var(--pf-t--global--text--color--subtle)" }}>
              Deploy from a template below, or use your own YAML.
            </Content>
          </div>
        )}

        {/* Templates */}
        <div style={{ marginTop: 32 }}>
          <div style={{
            fontSize: 11,
            fontWeight: 700,
            textTransform: "uppercase",
            letterSpacing: "1.2px",
            color: "var(--arcana-text-muted)",
            marginBottom: 12,
          }}>
            Templates
          </div>
          <div style={{ display: "grid", gridTemplateColumns: "repeat(2, 1fr)", gap: 12 }}>
            {TEMPLATES.map((t) => (
              <div
                key={t.name}
                style={{
                  padding: "20px 24px",
                  background: "var(--arcana-card-bg)",
                  borderRadius: 10,
                  border: "1px solid var(--arcana-card-border)",
                  display: "flex",
                  justifyContent: "space-between",
                  alignItems: "center",
                  gap: 16,
                  cursor: "pointer",
                  transition: "border-color 0.15s",
                }}
                onClick={() => openDeploy({ name: t.agentName, model: t.model, skills: t.skills })}
                onMouseEnter={(e) => { (e.currentTarget as HTMLDivElement).style.borderColor = "rgba(91,141,239,0.3)"; }}
                onMouseLeave={(e) => { (e.currentTarget as HTMLDivElement).style.borderColor = "var(--arcana-card-border)"; }}
              >
                <div>
                  <div style={{ fontSize: 15, fontWeight: 600, color: "var(--arcana-text)", marginBottom: 4 }}>
                    {t.name}
                  </div>
                  <div style={{ fontSize: 12, color: "var(--arcana-text-muted)" }}>
                    {t.desc}
                  </div>
                </div>
                <Button variant="secondary" size="sm">
                  Deploy
                </Button>
              </div>
            ))}
          </div>
          <div style={{ marginTop: 16, textAlign: "center" }}>
            <Button variant="link" icon={<CodeIcon />} onClick={() => setYamlOpen(true)}>
              View YAML example
            </Button>
          </div>
        </div>
      </PageSection>

      {/* Deploy modal — simplified, no type selector */}
      <Modal
        variant={ModalVariant.medium}
        isOpen={deployOpen}
        onClose={() => setDeployOpen(false)}
        aria-labelledby="deploy-agent-title"
      >
        <ModalHeader title="Deploy agent" labelId="deploy-agent-title" />
        <ModalBody>
          {submitError && (
            <Alert variant="danger" title="Deploy failed" isInline style={{ marginBottom: 16 }}>
              {submitError}
            </Alert>
          )}
          {submitSuccess && (
            <Alert variant="success" title={submitSuccess} isInline style={{ marginBottom: 16 }} />
          )}
          <Form id="deploy-form">
            <FormGroup label="Name" isRequired fieldId="agent-name">
              <TextInput
                id="agent-name"
                value={form.name}
                onChange={(_e, v) => setForm((f) => ({ ...f, name: v }))}
                placeholder="my-agent"
                isRequired
              />
            </FormGroup>
            <FormGroup label="Model" fieldId="agent-model">
              <FormSelect
                id="agent-model"
                value={form.model}
                onChange={(_e, v) => setForm((f) => ({ ...f, model: v }))}
                aria-label="Model"
              >
                {MODEL_OPTIONS.map((m) => (
                  <FormSelectOption key={m} value={m} label={m} />
                ))}
              </FormSelect>
            </FormGroup>
            <FormGroup label="Skills" fieldId="agent-skills">
              <TextInput
                id="agent-skills"
                value={form.skills}
                onChange={(_e, v) => setForm((f) => ({ ...f, skills: v }))}
                placeholder="search, summarize, code-gen"
              />
            </FormGroup>
          </Form>
        </ModalBody>
        <ModalFooter>
          <Button
            variant="primary"
            onClick={handleDeploy}
            isDisabled={submitting}
            isLoading={submitting}
          >
            Deploy
          </Button>
          <Button variant="link" onClick={() => setDeployOpen(false)}>
            Cancel
          </Button>
        </ModalFooter>
      </Modal>

      {/* YAML example modal */}
      <Modal
        variant={ModalVariant.medium}
        isOpen={yamlOpen}
        onClose={() => setYamlOpen(false)}
        aria-labelledby="yaml-title"
      >
        <ModalHeader title="Agent YAML Example" labelId="yaml-title" />
        <ModalBody>
          <pre style={{
            margin: 0,
            padding: 16,
            background: "var(--pf-t--global--background--color--secondary--default)",
            borderRadius: 4,
            fontSize: 12,
            overflow: "auto",
          }}>
            {ARCANA_AGENT_YAML}
          </pre>
        </ModalBody>
        <ModalFooter>
          <Button variant="primary" onClick={() => setYamlOpen(false)}>Close</Button>
        </ModalFooter>
      </Modal>
    </>
  );
};
