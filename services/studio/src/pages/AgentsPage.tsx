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

interface MeshAgent {
  name: string;
  capabilities: string[];
  protocols: string[];
  status: string;
  registered_at?: string;
}

interface AgentTemplate {
  name: string;
  desc: string;
  model: string;
  skills: string[];
  icon: React.ReactNode;
  agentName: string;
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
  },
  {
    name: "Code Assistant",
    agentName: "code-assistant",
    desc: "Specialized for code review, generation, and refactoring",
    model: "claude-sonnet",
    skills: ["code-review", "refactor", "test-gen"],
    icon: <CodeIcon />,
  },
  {
    name: "Data Pipeline Agent",
    agentName: "data-pipeline-agent",
    desc: "Orchestrates ETL workflows and data quality checks",
    model: "gpt-4o-mini",
    skills: ["sql-gen", "schema-validate", "data-profile"],
    icon: <CubesIcon />,
  },
  {
    name: "Ops Automation Agent",
    agentName: "ops-automation-agent",
    desc: "SRE agent for incident response and infrastructure management",
    model: "claude-sonnet",
    skills: ["k8s-ops", "log-analyze", "runbook-exec"],
    icon: <AutomationIcon />,
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
});

export const AgentsPage = () => {
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
      const res = await fetch("/api/v1/agents/register", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: form.name.trim(),
          capabilities,
          protocols,
          status: "active",
        }),
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
                  <Th>Status</Th>
                  <Th>Capabilities</Th>
                  <Th>Protocols</Th>
                </Tr>
              </Thead>
              <Tbody>
                {agents.map((agent) => (
                  <Tr key={agent.name}>
                    <Td dataLabel="Name">{agent.name}</Td>
                    <Td dataLabel="Status">
                      <Label color={statusColor(agent.status)} isCompact>
                        {agent.status}
                      </Label>
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
                        <Label color="purple" isCompact>{t.model}</Label>
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
