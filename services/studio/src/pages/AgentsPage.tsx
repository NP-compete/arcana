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
  FormSelect,
  FormSelectOption,
  Checkbox,
  Alert,
  Spinner,
  TextArea,
  ExpandableSection,
  Slider,
} from "@patternfly/react-core";
import { Table, Thead, Tr, Th, Tbody, Td } from "@patternfly/react-table";
import {
  RobotIcon,
  PlusCircleIcon,
  CodeIcon,
  CubesIcon,
  AutomationIcon,
} from "@patternfly/react-icons";
import { ARCANA_AGENT_YAML } from "../constants/arcanaAgentYaml";
import { useNavigate } from "react-router-dom";

interface MeshAgent {
  name: string;
  agent_type: string;
  capabilities: string[];
  protocols: string[];
  status: string;
  registered_at?: string;
}

interface DeepConfig {
  world_model: boolean;
  skill_graph: boolean;
  blueprint: string;
  memory_policy: string;
  sub_agents: string[];
  hitl_enabled: boolean;
  self_improve: boolean;
  system_prompt: string;
  temperature: number;
  max_tokens: number;
  model_call_limit: number;
  tool_call_limit: number;
}

interface AgentTemplate {
  name: string;
  desc: string;
  model: string;
  skills: string[];
  icon: React.ReactNode;
  agentName: string;
  agentType: "create_agent" | "create_deep_agent";
  deepConfig?: DeepConfig;
}

const MODEL_OPTIONS = ["gpt-4o", "claude-sonnet", "gpt-4o-mini"];
const PROTOCOL_OPTIONS = ["a2a", "acp", "mcp"] as const;

const AGENT_TEMPLATES: AgentTemplate[] = [
  {
    name: "Conversational Agent",
    agentName: "conversational-agent",
    desc: "Chat-based agent with memory, skill execution, and guardrails",
    model: "gpt-4o",
    skills: ["search", "code-gen", "summarize"],
    icon: <RobotIcon />,
    agentType: "create_agent",
  },
  {
    name: "Code Assistant",
    agentName: "code-assistant",
    desc: "Specialized for code review, generation, and refactoring",
    model: "claude-sonnet",
    skills: ["code-review", "refactor", "test-gen"],
    icon: <CodeIcon />,
    agentType: "create_agent",
  },
  {
    name: "Data Pipeline Agent",
    agentName: "data-pipeline-agent",
    desc: "Orchestrates ETL workflows with world model prediction, skill graph, and HITL gates",
    model: "gpt-4o",
    skills: ["sql-gen", "schema-validate", "data-profile"],
    icon: <CubesIcon />,
    agentType: "create_deep_agent",
    deepConfig: {
      world_model: true,
      skill_graph: true,
      blueprint: "",
      memory_policy: "tri-scope",
      sub_agents: [],
      hitl_enabled: true,
      self_improve: true,
      system_prompt: "",
      temperature: 0.0,
      max_tokens: 8192,
      model_call_limit: 50,
      tool_call_limit: 200,
    },
  },
  {
    name: "Research Pipeline Agent",
    agentName: "research-pipeline-agent",
    desc: "Multi-agent research pipeline with sub-agent delegation, self-improvement, and Oracle world model",
    model: "claude-sonnet",
    skills: ["research", "summarize", "fact-check", "publish"],
    icon: <AutomationIcon />,
    agentType: "create_deep_agent",
    deepConfig: {
      world_model: true,
      skill_graph: true,
      blueprint: "",
      memory_policy: "tri-scope",
      sub_agents: ["enabled"],
      hitl_enabled: true,
      self_improve: true,
      system_prompt: "",
      temperature: 0.0,
      max_tokens: 8192,
      model_call_limit: 50,
      tool_call_limit: 200,
    },
  },
];

const statusColor = (status: string): "green" | "blue" | "orange" | "grey" | "red" => {
  switch (status) {
    case "active":
      return "green";
    case "busy":
      return "orange";
    case "idle":
      return "blue";
    case "offline":
      return "red";
    default:
      return "grey";
  }
};

const emptyDeployForm = () => ({
  name: "",
  model: MODEL_OPTIONS[0],
  capabilities: "",
  protocols: { a2a: true, acp: false, mcp: false } as Record<string, boolean>,
  agentType: "create_agent" as "create_agent" | "create_deep_agent",
  deepConfig: undefined as DeepConfig | undefined,
});

export const AgentsPage = () => {
  const navigate = useNavigate();
  const [agents, setAgents] = useState<MeshAgent[]>([]);
  const [loading, setLoading] = useState(true);
  const [fetchError, setFetchError] = useState<string | null>(null);

  const [deployModalOpen, setDeployModalOpen] = useState(false);
  const [yamlModalOpen, setYamlModalOpen] = useState(false);
  const [form, setForm] = useState(emptyDeployForm());
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


  const openDeployModal = (prefill?: Partial<ReturnType<typeof emptyDeployForm>>) => {
    setForm({ ...emptyDeployForm(), ...prefill });
    setSubmitError(null);
    setSubmitSuccess(null);
    setDeployModalOpen(true);
  };

  const closeDeployModal = () => {
    setDeployModalOpen(false);
    setSubmitError(null);
  };

  const openTemplateDeploy = (template: AgentTemplate) => {
    openDeployModal({
      name: template.agentName,
      model: template.model,
      capabilities: template.skills.join(", "),
      protocols: { a2a: true, acp: true, mcp: false },
      agentType: template.agentType,
      deepConfig: template.deepConfig,
    });
  };

  const handleDeploy = async () => {
    if (!form.name.trim()) {
      setSubmitError("Agent name is required");
      return;
    }
    const capabilities = form.capabilities
      .split(",")
      .map((s) => s.trim())
      .filter(Boolean);
    capabilities.push(`model:${form.model}`);

    const protocols = PROTOCOL_OPTIONS.filter((p) => form.protocols[p]);

    setSubmitting(true);
    setSubmitError(null);
    setSubmitSuccess(null);
    try {
      const body: Record<string, unknown> = {
        name: form.name.trim(),
        agent_type: form.agentType,
        capabilities,
        protocols,
        status: "active",
      };
      if (form.agentType === "create_deep_agent" && form.deepConfig) {
        body.deep_config = form.deepConfig;
      }
      const res = await fetch("/api/v1/agents/register", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error ?? `HTTP ${res.status}`);
      }
      setSubmitSuccess(`Agent "${data.name}" registered successfully`);
      await fetchAgents();
    } catch (e) {
      setSubmitError(e instanceof Error ? e.message : "Failed to register agent");
    } finally {
      setSubmitting(false);
    }
  };

  const hasAgents = agents.length > 0;

  return (
    <>
      <PageSection hasBodyWrapper={false}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
          <div>
            <Title headingLevel="h1" size="2xl">Agents</Title>
            <Content component="p" style={{ marginTop: 4 }}>
              Deploy and manage Kubernetes-native AI agents.
            </Content>
          </div>
          <Button variant="primary" icon={<PlusCircleIcon />} onClick={() => openDeployModal()}>
            Deploy Agent
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
        ) : hasAgents ? (
          <>
            <div className="section-title">Registered Agents ({agents.length})</div>
            <Table aria-label="Registered agents" variant="compact">
              <Thead>
                <Tr>
                  <Th>Name</Th>
                  <Th>Type</Th>
                  <Th>Status</Th>
                  <Th>Health</Th>
                  <Th>Capabilities</Th>
                  <Th>Protocols</Th>
                </Tr>
              </Thead>
              <Tbody>
                {agents.map((agent) => (
                  <Tr
                    key={agent.name}
                    isClickable
                    onRowClick={() => navigate(`/agents/${agent.name}`)}
                  >
                    <Td dataLabel="Name">
                      <Button variant="link" isInline onClick={() => navigate(`/agents/${agent.name}`)}>
                        {agent.name}
                      </Button>
                    </Td>
                    <Td dataLabel="Type">
                      <Label
                        color={agent.agent_type === "create_deep_agent" ? "purple" : "blue"}
                        isCompact
                      >
                        {agent.agent_type === "create_deep_agent" ? "deep" : "standard"}
                      </Label>
                    </Td>
                    <Td dataLabel="Status">
                      <Label color={statusColor(agent.status)} isCompact>
                        {agent.status}
                      </Label>
                    </Td>
                    <Td dataLabel="Health">
                      {agent.status === "active" || agent.status === "busy" ? (
                        <Label color="green" isCompact>healthy</Label>
                      ) : agent.status === "offline" ? (
                        <Label color="red" isCompact>down</Label>
                      ) : agent.status === "suspended" ? (
                        <Label color="grey" isCompact>suspended</Label>
                      ) : (
                        <Label color="grey" isCompact>—</Label>
                      )}
                    </Td>
                    <Td dataLabel="Capabilities">
                      <div style={{ display: "flex", gap: 4, flexWrap: "wrap" }}>
                        {(agent.capabilities ?? []).map((c) => (
                          <Label color="grey" isCompact key={c}>{c}</Label>
                        ))}
                      </div>
                    </Td>
                    <Td dataLabel="Protocols">
                      <div style={{ display: "flex", gap: 4, flexWrap: "wrap" }}>
                        {(agent.protocols ?? []).map((p) => (
                          <Label color="purple" isCompact key={p}>{p}</Label>
                        ))}
                      </div>
                    </Td>
                  </Tr>
                ))}
              </Tbody>
            </Table>
          </>
        ) : (
          <div className="arcana-empty-state">
            <div className="arcana-empty-icon">
              <RobotIcon />
            </div>
            <Title headingLevel="h2" size="xl" style={{ marginBottom: 8 }}>
              No agents deployed yet
            </Title>
            <Content component="p" style={{ maxWidth: 480, margin: "0 auto 32px auto", color: "var(--pf-t--global--text--color--subtle)" }}>
              Get started by deploying from a template below, or apply your own ArcanaAgent YAML to the cluster.
            </Content>
          </div>
        )}

        <div className="section-title">Agent Templates</div>
        <Grid hasGutter>
          {AGENT_TEMPLATES.map((t) => (
            <GridItem span={6} key={t.name}>
              <Card className="stat-card">
                <CardBody>
                  <div style={{ display: "flex", gap: 16 }}>
                    <div className="action-card-icon" style={{ margin: 0, flexShrink: 0 }}>
                      {t.icon}
                    </div>
                    <div style={{ flex: 1 }}>
                      <div style={{ fontWeight: 700, fontSize: 16, marginBottom: 4 }}>{t.name}</div>
                      <div style={{ fontSize: 13, color: "var(--pf-t--global--text--color--subtle)", marginBottom: 10 }}>
                        {t.desc}
                      </div>
                      <div style={{ display: "flex", gap: 6, flexWrap: "wrap", alignItems: "center" }}>
                        <Label
                          color={t.agentType === "create_deep_agent" ? "purple" : "blue"}
                          isCompact
                        >
                          {t.agentType === "create_deep_agent" ? "deep agent" : "standard agent"}
                        </Label>
                        <Label color="teal" isCompact>{t.model}</Label>
                        {t.skills.map((s) => (
                          <Label color="grey" isCompact key={s}>{s}</Label>
                        ))}
                      </div>
                    </div>
                    <Button variant="secondary" size="sm" style={{ alignSelf: "center" }} onClick={() => openTemplateDeploy(t)}>
                      Deploy
                    </Button>
                  </div>
                </CardBody>
              </Card>
            </GridItem>
          ))}
        </Grid>

        <div style={{ marginTop: 24, textAlign: "center" }}>
          <Button variant="link" icon={<CodeIcon />} onClick={() => setYamlModalOpen(true)}>
            View YAML example for custom agent
          </Button>
        </div>
      </PageSection>

      <Modal
        variant={ModalVariant.medium}
        isOpen={deployModalOpen}
        onClose={closeDeployModal}
        aria-labelledby="deploy-agent-title"
      >
        <ModalHeader title="Deploy Agent" labelId="deploy-agent-title" />
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
          <Form id="deploy-agent-form">
            <FormGroup label="Name" isRequired fieldId="agent-name">
              <TextInput
                id="agent-name"
                value={form.name}
                onChange={(_e, v) => setForm((f) => ({ ...f, name: v }))}
                isRequired
              />
            </FormGroup>
            <FormGroup label="Agent Type" fieldId="agent-type">
              <FormSelect
                id="agent-type"
                value={form.agentType}
                onChange={(_e, v) =>
                  setForm((f) => ({
                    ...f,
                    agentType: v as "create_agent" | "create_deep_agent",
                    deepConfig:
                      v === "create_deep_agent"
                        ? f.deepConfig ?? {
                            world_model: true,
                            skill_graph: true,
                            blueprint: "",
                            memory_policy: "tri-scope",
                            sub_agents: [],
                            hitl_enabled: false,
                            self_improve: false,
                            system_prompt: "",
                            temperature: 0.0,
                            max_tokens: 8192,
                            model_call_limit: 50,
                            tool_call_limit: 200,
                          }
                        : undefined,
                  }))
                }
                aria-label="Agent type"
              >
                <FormSelectOption value="create_agent" label="create_agent  —  Standard agent" />
                <FormSelectOption value="create_deep_agent" label="create_deep_agent  —  Deep agent (world model, skills, HITL)" />
              </FormSelect>
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
            <FormGroup label="Capabilities" fieldId="agent-capabilities">
              <TextInput
                id="agent-capabilities"
                value={form.capabilities}
                onChange={(_e, v) => setForm((f) => ({ ...f, capabilities: v }))}
                placeholder="search, summarize, code-gen"
              />
            </FormGroup>
            <FormGroup label="Protocols" fieldId="agent-protocols">
              {PROTOCOL_OPTIONS.map((p) => (
                <Checkbox
                  key={p}
                  id={`protocol-${p}`}
                  label={p.toUpperCase()}
                  isChecked={form.protocols[p]}
                  onChange={(_e, checked) =>
                    setForm((f) => ({ ...f, protocols: { ...f.protocols, [p]: checked } }))
                  }
                />
              ))}
            </FormGroup>
            {form.agentType === "create_deep_agent" && (
              <>
                <Divider style={{ margin: "16px 0" }} />
                <div style={{ fontWeight: 700, marginBottom: 8 }}>Deep Agent Features</div>
                <FormGroup fieldId="deep-features">
                  <Checkbox
                    id="deep-world-model"
                    label="World Model (Oracle L2 — tool outcome prediction)"
                    isChecked={form.deepConfig?.world_model ?? false}
                    onChange={(_e, c) =>
                      setForm((f) => ({ ...f, deepConfig: { ...f.deepConfig!, world_model: c } }))
                    }
                  />
                  <Checkbox
                    id="deep-skill-graph"
                    label="Skill Graph (3-tier with experiential memory)"
                    isChecked={form.deepConfig?.skill_graph ?? false}
                    onChange={(_e, c) =>
                      setForm((f) => ({ ...f, deepConfig: { ...f.deepConfig!, skill_graph: c } }))
                    }
                  />
                  <Checkbox
                    id="deep-hitl"
                    label="HITL Gates (human-in-the-loop approval)"
                    isChecked={form.deepConfig?.hitl_enabled ?? false}
                    onChange={(_e, c) =>
                      setForm((f) => ({ ...f, deepConfig: { ...f.deepConfig!, hitl_enabled: c } }))
                    }
                  />
                  <Checkbox
                    id="deep-self-improve"
                    label="Self-Improvement (auto-crystallize skills)"
                    isChecked={form.deepConfig?.self_improve ?? false}
                    onChange={(_e, c) =>
                      setForm((f) => ({ ...f, deepConfig: { ...f.deepConfig!, self_improve: c } }))
                    }
                  />
                  <Checkbox
                    id="deep-sub-agents"
                    label="Sub-Agents (multi-agent delegation)"
                    isChecked={(form.deepConfig?.sub_agents?.length ?? 0) > 0}
                    onChange={(_e, c) =>
                      setForm((f) => ({
                        ...f,
                        deepConfig: { ...f.deepConfig!, sub_agents: c ? ["enabled"] : [] },
                      }))
                    }
                  />
                </FormGroup>

                <ExpandableSection
                  toggleText="Advanced Parameters"
                  style={{ marginTop: 16 }}
                >
                  <FormGroup label="System Prompt" fieldId="deep-system-prompt">
                    <TextArea
                      id="deep-system-prompt"
                      value={form.deepConfig?.system_prompt ?? ""}
                      onChange={(_e, v) =>
                        setForm((f) => ({ ...f, deepConfig: { ...f.deepConfig!, system_prompt: v } }))
                      }
                      placeholder="Custom system prompt (leave blank for auto-generated)"
                      rows={4}
                      resizeOrientation="vertical"
                    />
                  </FormGroup>
                  <FormGroup
                    label={`Temperature: ${form.deepConfig?.temperature?.toFixed(1) ?? "0.0"}`}
                    fieldId="deep-temperature"
                    style={{ marginTop: 12 }}
                  >
                    <Slider
                      id="deep-temperature"
                      value={(form.deepConfig?.temperature ?? 0) * 100}
                      max={200}
                      min={0}
                      step={10}
                      onChange={(_e, v) =>
                        setForm((f) => ({
                          ...f,
                          deepConfig: { ...f.deepConfig!, temperature: v / 100 },
                        }))
                      }
                      areCustomStepsContinuous
                    />
                  </FormGroup>
                  <FormGroup label="Max Output Tokens" fieldId="deep-max-tokens" style={{ marginTop: 12 }}>
                    <TextInput
                      id="deep-max-tokens"
                      type="number"
                      value={form.deepConfig?.max_tokens ?? 8192}
                      onChange={(_e, v) =>
                        setForm((f) => ({
                          ...f,
                          deepConfig: { ...f.deepConfig!, max_tokens: parseInt(v) || 8192 },
                        }))
                      }
                    />
                  </FormGroup>
                  <FormGroup label="Memory Policy" fieldId="deep-memory-policy" style={{ marginTop: 12 }}>
                    <FormSelect
                      id="deep-memory-policy"
                      value={form.deepConfig?.memory_policy ?? "tri-scope"}
                      onChange={(_e, v) =>
                        setForm((f) => ({ ...f, deepConfig: { ...f.deepConfig!, memory_policy: v } }))
                      }
                    >
                      <FormSelectOption value="tri-scope" label="Tri-Scope (short + long + skill)" />
                      <FormSelectOption value="long-term" label="Long-Term Only" />
                      <FormSelectOption value="short-term" label="Short-Term Only" />
                      <FormSelectOption value="none" label="No Memory" />
                    </FormSelect>
                  </FormGroup>
                  <div style={{ display: "flex", gap: 16, marginTop: 12 }}>
                    <FormGroup label="Model Call Limit" fieldId="deep-model-limit" style={{ flex: 1 }}>
                      <TextInput
                        id="deep-model-limit"
                        type="number"
                        value={form.deepConfig?.model_call_limit ?? 50}
                        onChange={(_e, v) =>
                          setForm((f) => ({
                            ...f,
                            deepConfig: { ...f.deepConfig!, model_call_limit: parseInt(v) || 50 },
                          }))
                        }
                      />
                    </FormGroup>
                    <FormGroup label="Tool Call Limit" fieldId="deep-tool-limit" style={{ flex: 1 }}>
                      <TextInput
                        id="deep-tool-limit"
                        type="number"
                        value={form.deepConfig?.tool_call_limit ?? 200}
                        onChange={(_e, v) =>
                          setForm((f) => ({
                            ...f,
                            deepConfig: { ...f.deepConfig!, tool_call_limit: parseInt(v) || 200 },
                          }))
                        }
                      />
                    </FormGroup>
                  </div>
                </ExpandableSection>
              </>
            )}
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
          <Button variant="link" onClick={closeDeployModal}>
            Cancel
          </Button>
        </ModalFooter>
      </Modal>

      <Modal
        variant={ModalVariant.medium}
        isOpen={yamlModalOpen}
        onClose={() => setYamlModalOpen(false)}
        aria-labelledby="agent-yaml-title"
      >
        <ModalHeader
          title="ArcanaAgent YAML Example"
          description="Custom agent Custom Resource definition."
          labelId="agent-yaml-title"
        />
        <ModalBody>
          <pre style={{
            margin: 0,
            padding: 16,
            background: "var(--pf-t--global--background--color--secondary--default)",
            borderRadius: 4,
            fontSize: 12,
            overflow: "auto",
          }}
          >
            {ARCANA_AGENT_YAML}
          </pre>
        </ModalBody>
        <ModalFooter>
          <Button variant="primary" onClick={() => setYamlModalOpen(false)}>
            Close
          </Button>
        </ModalFooter>
      </Modal>
    </>
  );
};
