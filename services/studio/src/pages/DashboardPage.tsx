import { useState } from "react";
import {
  PageSection,
  Card,
  CardTitle,
  CardBody,
  Grid,
  GridItem,
  Spinner,
  Alert,
  Button,
  Tooltip,
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
} from "@patternfly/react-core";
import {
  CheckCircleIcon,
  ExclamationCircleIcon,
  DatabaseIcon,
  ServerIcon,
  ClusterIcon,
  StorageDomainIcon,
  CloudUploadAltIcon,
  RobotIcon,
  CubesIcon,
  PlusCircleIcon,
  SyncAltIcon,
  ArrowRightIcon,
  CogIcon,
  ShieldAltIcon,
  ChartBarIcon,
  CodeIcon,
  BrainIcon,
  SearchIcon,
  PluggedIcon,
  LockIcon,
  AutomationIcon,
  OptimizeIcon,
  OutlinedClockIcon,
  BundleIcon,
  ListIcon,
  MonitoringIcon,
  ProcessAutomationIcon,
} from "@patternfly/react-icons";
import { useHealth, type ServiceHealth } from "../hooks/useHealth";
import { ARCANA_AGENT_YAML } from "../constants/arcanaAgentYaml";

const PLANE_META: Record<string, { label: string; color: string; icon: React.ReactNode }> = {
  infra:   { label: "Infrastructure", color: "#3182ce", icon: <DatabaseIcon /> },
  agent:   { label: "Agent Plane",    color: "#805ad5", icon: <RobotIcon /> },
  data:    { label: "Data Plane",     color: "#38a169", icon: <SearchIcon /> },
  tool:    { label: "Tool Plane",     color: "#d69e2e", icon: <CogIcon /> },
  model:   { label: "Model Plane",    color: "#e53e3e", icon: <BrainIcon /> },
  govern:  { label: "Govern Plane",   color: "#dd6b20", icon: <ShieldAltIcon /> },
  quality: { label: "Quality Plane",  color: "#319795", icon: <ChartBarIcon /> },
  ops:     { label: "Ops Plane",      color: "#718096", icon: <ProcessAutomationIcon /> },
};

const SERVICE_ICONS: Record<string, React.ReactNode> = {
  PostgreSQL: <DatabaseIcon />,
  Redis: <ServerIcon />,
  Temporal: <ClusterIcon />,
  MinIO: <StorageDomainIcon />,
  NATS: <CloudUploadAltIcon />,
  engine: <AutomationIcon />,
  blueprint: <ProcessAutomationIcon />,
  oracle: <BrainIcon />,
  mesh: <PluggedIcon />,
  memory: <StorageDomainIcon />,
  "codex-router": <SearchIcon />,
  "codex-ingestor": <CloudUploadAltIcon />,
  connectors: <PluggedIcon />,
  graph: <BundleIcon />,
  tools: <CogIcon />,
  sandbox: <CodeIcon />,
  forge: <OptimizeIcon />,
  models: <BrainIcon />,
  ward: <ShieldAltIcon />,
  audit: <LockIcon />,
  probe: <ChartBarIcon />,
  annotate: <ListIcon />,
  skills: <CubesIcon />,
  scheduler: <OutlinedClockIcon />,
  registry: <BundleIcon />,
  finops: <MonitoringIcon />,
  gitops: <ProcessAutomationIcon />,
};

const PROTOCOLS = [
  { name: "MCP", desc: "Model Context Protocol — tool access" },
  { name: "A2A", desc: "Agent-to-Agent — primary mesh" },
  { name: "ACP", desc: "Agent Communication Protocol — IBM adapter" },
  { name: "AG-UI", desc: "Agent-User Interaction — streaming" },
  { name: "ACS", desc: "Agent Control Standard — lifecycle" },
];

const CRDS = [
  { name: "ArcanaAgent", short: "aag", purpose: "Agent lifecycle & config" },
  { name: "ArcanaTenant", short: "aten", purpose: "Multi-tenant isolation" },
  { name: "ArcanaSkillRegistry", short: "askr", purpose: "Skill catalog & versioning" },
  { name: "ArcanaEvalSuite", short: "aes", purpose: "Skill evaluation pipelines" },
  { name: "ArcanaRole", short: "arole", purpose: "RBAC + ABAC policies" },
  { name: "ArcanaBudget", short: "abud", purpose: "FinOps token/compute budgets" },
  { name: "ArcanaBackupPolicy", short: "abkp", purpose: "Backup scheduling & retention" },
  { name: "ArcanaPromotion", short: "aprom", purpose: "Environment promotion gates" },
  { name: "ArcanaBlueprint", short: "abp", purpose: "DAG execution blueprints" },
  { name: "ArcanaModel", short: "amod", purpose: "Model registry & serving" },
  { name: "ArcanaExperiment", short: "aexp", purpose: "Fine-tuning experiments" },
  { name: "ArcanaDataset", short: "ads", purpose: "Training/eval datasets" },
  { name: "ArcanaGuardrail", short: "agr", purpose: "6-layer guardrail rules" },
  { name: "ArcanaConnector", short: "acn", purpose: "Data source connectors" },
  { name: "ArcanaCodex", short: "acx", purpose: "Search index configuration" },
  { name: "ArcanaPlatform", short: "apf", purpose: "Platform-wide config" },
];

type QuickActionKey =
  | "deploy-agent"
  | "register-skill"
  | "run-evaluation"
  | "view-workflows"
  | "submit-task"
  | "search-knowledge";

const QUICK_ACTIONS: {
  key: QuickActionKey;
  label: string;
  desc: string;
  icon: React.ReactNode;
  href?: string;
}[] = [
  { key: "deploy-agent", label: "Deploy Agent", desc: "Create an ArcanaAgent CR", icon: <RobotIcon /> },
  { key: "register-skill", label: "Register Skill", desc: "Add to the skill catalog", icon: <CubesIcon /> },
  { key: "run-evaluation", label: "Run Evaluation", desc: "Test a skill with judges", icon: <CheckCircleIcon /> },
  { key: "view-workflows", label: "View Workflows", desc: "Open Temporal UI", icon: <ClusterIcon />, href: "http://localhost:8233" },
  { key: "submit-task", label: "Submit Task", desc: "Run an agent task via engine", icon: <AutomationIcon /> },
  { key: "search-knowledge", label: "Search Knowledge", desc: "Query the Codex search engine", icon: <SearchIcon /> },
];

const MODEL_OPTIONS = ["gpt-4o", "claude-sonnet", "gpt-4o-mini"];

function groupByPlane(services: ServiceHealth[]): Record<string, ServiceHealth[]> {
  const groups: Record<string, ServiceHealth[]> = {};
  for (const svc of services) {
    const plane = svc.plane || "unknown";
    if (!groups[plane]) groups[plane] = [];
    groups[plane].push(svc);
  }
  return groups;
}

interface DashboardPageProps {
  onNavigate?: (page: string) => void;
}

export const DashboardPage = ({ onNavigate }: DashboardPageProps) => {
  const { health, error, loading, refresh } = useHealth(5000);

  const [taskModalOpen, setTaskModalOpen] = useState(false);
  const [searchModalOpen, setSearchModalOpen] = useState(false);
  const [yamlModalOpen, setYamlModalOpen] = useState(false);

  const [taskAgent, setTaskAgent] = useState("");
  const [taskInput, setTaskInput] = useState("");
  const [taskModel, setTaskModel] = useState(MODEL_OPTIONS[0]);
  const [taskSubmitting, setTaskSubmitting] = useState(false);
  const [taskResult, setTaskResult] = useState<string | null>(null);
  const [taskError, setTaskError] = useState<string | null>(null);

  const [searchQuery, setSearchQuery] = useState("");
  const [searchSubmitting, setSearchSubmitting] = useState(false);
  const [searchResult, setSearchResult] = useState<string | null>(null);
  const [searchError, setSearchError] = useState<string | null>(null);

  const healthyCount = health?.services.filter((s) => s.status === "healthy").length ?? 0;
  const totalCount = health?.services.length ?? 0;
  const allHealthy = healthyCount === totalCount && totalCount > 0;
  const planeGroups = health ? groupByPlane(health.services) : {};
  const planeOrder = ["infra", "agent", "data", "tool", "model", "govern", "quality", "ops"];

  const handleQuickAction = (key: QuickActionKey, href?: string) => {
    switch (key) {
      case "deploy-agent":
        onNavigate?.("agents");
        break;
      case "register-skill":
        onNavigate?.("skills");
        break;
      case "run-evaluation":
        onNavigate?.("evaluations");
        break;
      case "view-workflows":
        if (href) window.open(href, "_blank");
        break;
      case "submit-task":
        setTaskResult(null);
        setTaskError(null);
        setTaskModalOpen(true);
        break;
      case "search-knowledge":
        setSearchResult(null);
        setSearchError(null);
        setSearchModalOpen(true);
        break;
    }
  };

  const closeTaskModal = () => {
    setTaskModalOpen(false);
    setTaskError(null);
  };

  const closeSearchModal = () => {
    setSearchModalOpen(false);
    setSearchError(null);
  };

  const handleSubmitTask = async () => {
    if (!taskAgent.trim()) {
      setTaskError("Agent name is required");
      return;
    }
    setTaskSubmitting(true);
    setTaskError(null);
    setTaskResult(null);
    try {
      const res = await fetch("/api/v1/tasks", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          agent: taskAgent.trim(),
          input: { text: taskInput },
          model: { model: taskModel },
        }),
      });
      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error ?? `HTTP ${res.status}`);
      }
      setTaskResult(JSON.stringify(data, null, 2));
    } catch (e) {
      setTaskError(e instanceof Error ? e.message : "Failed to submit task");
    } finally {
      setTaskSubmitting(false);
    }
  };

  const handleSearch = async () => {
    if (!searchQuery.trim()) {
      setSearchError("Search query is required");
      return;
    }
    setSearchSubmitting(true);
    setSearchError(null);
    setSearchResult(null);
    try {
      const res = await fetch("/api/v1/search", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ query: searchQuery.trim(), profile: "default", top_k: 10 }),
      });
      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error ?? `HTTP ${res.status}`);
      }
      setSearchResult(JSON.stringify(data, null, 2));
    } catch (e) {
      setSearchError(e instanceof Error ? e.message : "Search failed");
    } finally {
      setSearchSubmitting(false);
    }
  };

  return (
    <>
      <PageSection hasBodyWrapper={false} style={{ paddingBottom: 0 }}>
        <div className="welcome-banner">
          <h2>Welcome to Arcana Studio</h2>
          <p>
            Your Kubernetes-native AI platform is running {totalCount} services across 8 planes.
            Build agents, manage skills, enforce guardrails, and monitor everything from one place.
          </p>
        </div>
      </PageSection>

      {error && (
        <PageSection hasBodyWrapper={false} style={{ paddingBottom: 0 }}>
          <Alert variant="warning" title="API unreachable" isInline isExpandable>
            <p>Cannot reach arcana-api. Run <code>make dev</code> to start.</p>
          </Alert>
        </PageSection>
      )}

      <PageSection hasBodyWrapper={false} style={{ paddingBottom: 0 }}>
        <Grid hasGutter>
          <GridItem span={3}>
            <Card className="stat-card">
              <CardBody>
                <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start" }}>
                  <div>
                    <div className="stat-card-value" style={{ color: allHealthy ? "var(--arcana-green)" : "var(--arcana-red)" }}>
                      {loading ? <Spinner size="lg" /> : `${healthyCount}/${totalCount}`}
                    </div>
                    <div className="stat-card-label">Services Healthy</div>
                  </div>
                  <div className="stat-card-icon stat-icon-green">
                    {allHealthy ? <CheckCircleIcon /> : <ExclamationCircleIcon />}
                  </div>
                </div>
              </CardBody>
            </Card>
          </GridItem>
          <GridItem span={3}>
            <Card className="stat-card">
              <CardBody>
                <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start" }}>
                  <div>
                    <div className="stat-card-value" style={{ color: "var(--arcana-indigo)" }}>8</div>
                    <div className="stat-card-label">Planes Active</div>
                  </div>
                  <div className="stat-card-icon stat-icon-indigo">
                    <CubesIcon />
                  </div>
                </div>
              </CardBody>
            </Card>
          </GridItem>
          <GridItem span={3}>
            <Card className="stat-card">
              <CardBody>
                <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start" }}>
                  <div>
                    <div className="stat-card-value" style={{ color: "var(--arcana-purple)" }}>{CRDS.length}</div>
                    <div className="stat-card-label">CRDs Installed</div>
                  </div>
                  <div className="stat-card-icon stat-icon-purple">
                    <CloudUploadAltIcon />
                  </div>
                </div>
              </CardBody>
            </Card>
          </GridItem>
          <GridItem span={3}>
            <Card className="stat-card">
              <CardBody>
                <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start" }}>
                  <div>
                    <div className="stat-card-value" style={{ color: "var(--arcana-cyan)" }}>
                      {health?.uptime ?? "—"}
                    </div>
                    <div className="stat-card-label">Platform Uptime</div>
                  </div>
                  <div className="stat-card-icon stat-icon-cyan">
                    <SyncAltIcon />
                  </div>
                </div>
              </CardBody>
            </Card>
          </GridItem>
        </Grid>
      </PageSection>

      <PageSection hasBodyWrapper={false}>
        <Grid hasGutter>
          <GridItem span={8}>
            <Card className="stat-card" isFullHeight>
              <CardTitle>
                <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
                  <div className="section-title" style={{ margin: 0 }}>
                    Platform Health — {totalCount} Services
                  </div>
                  <Button variant="plain" onClick={refresh} aria-label="Refresh">
                    <SyncAltIcon />
                  </Button>
                </div>
              </CardTitle>
              <CardBody>
                {loading && !health ? (
                  <div style={{ textAlign: "center", padding: 40 }}>
                    <Spinner size="xl" />
                  </div>
                ) : (
                  <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
                    {planeOrder.map((plane) => {
                      const svcs = planeGroups[plane];
                      if (!svcs) return null;
                      const pm = PLANE_META[plane] ?? { label: plane, color: "#718096", icon: <ServerIcon /> };
                      const planeHealthy = svcs.filter((s) => s.status === "healthy").length;
                      return (
                        <div key={plane}>
                          <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 8 }}>
                            <span style={{ color: pm.color }}>{pm.icon}</span>
                            <span style={{ fontWeight: 600, fontSize: 14 }}>{pm.label}</span>
                            <Label
                              color={planeHealthy === svcs.length ? "green" : "orange"}
                              isCompact
                            >
                              {planeHealthy}/{svcs.length}
                            </Label>
                          </div>
                          <div style={{ display: "flex", flexDirection: "column", gap: 2, paddingLeft: 8 }}>
                            {svcs.map((svc) => (
                              <div className="service-row" key={svc.name}>
                                <div
                                  className="service-row-icon"
                                  style={{ background: `${pm.color}15`, color: pm.color }}
                                >
                                  {SERVICE_ICONS[svc.name] ?? <ServerIcon />}
                                </div>
                                <div style={{ flex: 1 }}>
                                  <div className="service-row-name">{svc.name}</div>
                                </div>
                                <span
                                  className={`health-dot ${svc.status === "healthy" ? "health-dot-healthy" : "health-dot-down"}`}
                                />
                                <span className="service-row-port">:{svc.port}</span>
                                <span className="service-row-latency">{svc.latency}</span>
                              </div>
                            ))}
                          </div>
                        </div>
                      );
                    })}
                  </div>
                )}
              </CardBody>
            </Card>
          </GridItem>

          <GridItem span={4}>
            <Card className="stat-card" isFullHeight>
              <CardTitle>
                <div className="section-title" style={{ margin: 0 }}>Quick Actions</div>
              </CardTitle>
              <CardBody>
                <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
                  {QUICK_ACTIONS.map((action) => (
                    <Button
                      key={action.label}
                      variant="plain"
                      className="action-card"
                      style={{ padding: 16, display: "flex", alignItems: "center", gap: 14, textAlign: "left", width: "100%", height: "auto" }}
                      onClick={() => handleQuickAction(action.key, action.href)}
                    >
                      <div className="action-card-icon" style={{ margin: 0, flexShrink: 0, width: 40, height: 40 }}>
                        {action.icon}
                      </div>
                      <div style={{ flex: 1, textAlign: "left" }}>
                        <div style={{ fontWeight: 600, fontSize: 14 }}>{action.label}</div>
                        <div style={{ fontSize: 12, color: "var(--pf-t--global--text--color--subtle)" }}>{action.desc}</div>
                      </div>
                      <div style={{ marginLeft: "auto", color: "var(--pf-t--global--text--color--subtle)" }}>
                        <ArrowRightIcon />
                      </div>
                    </Button>
                  ))}
                </div>
              </CardBody>
            </Card>
          </GridItem>

          <GridItem span={12}>
            <Card className="stat-card">
              <CardTitle>
                <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
                  <div className="section-title" style={{ margin: 0 }}>
                    Agentic Protocol Stack
                  </div>
                </div>
              </CardTitle>
              <CardBody>
                <div style={{ display: "flex", gap: 12, flexWrap: "wrap" }}>
                  {PROTOCOLS.map((p) => (
                    <Tooltip key={p.name} content={p.desc}>
                      <span className="protocol-pill">{p.name}</span>
                    </Tooltip>
                  ))}
                </div>
              </CardBody>
            </Card>
          </GridItem>

          <GridItem span={12}>
            <Card className="stat-card">
              <CardTitle>
                <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
                  <div className="section-title" style={{ margin: 0 }}>
                    Custom Resource Definitions ({CRDS.length})
                  </div>
                  <Button variant="link" icon={<PlusCircleIcon />} isInline onClick={() => setYamlModalOpen(true)}>
                    Create Resource
                  </Button>
                </div>
              </CardTitle>
              <CardBody>
                <Grid hasGutter>
                  {CRDS.map((crd) => (
                    <GridItem span={3} key={crd.name}>
                      <div className="crd-card">
                        <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 6 }}>
                          <span style={{ color: "var(--arcana-indigo)" }}>
                            <CubesIcon />
                          </span>
                          <div className="crd-card-name">{crd.name}</div>
                        </div>
                        <div className="crd-card-short">{crd.short}</div>
                        <div className="crd-card-desc">{crd.purpose}</div>
                      </div>
                    </GridItem>
                  ))}
                </Grid>
              </CardBody>
            </Card>
          </GridItem>
        </Grid>
      </PageSection>

      <Modal
        variant={ModalVariant.medium}
        isOpen={taskModalOpen}
        onClose={closeTaskModal}
        aria-labelledby="submit-task-title"
      >
        <ModalHeader title="Submit Task" labelId="submit-task-title" />
        <ModalBody>
          {taskError && (
            <Alert variant="danger" title="Task submission failed" isInline style={{ marginBottom: 16 }}>
              {taskError}
            </Alert>
          )}
          {taskResult && (
            <Alert variant="success" title="Task submitted" isInline style={{ marginBottom: 16 }}>
              <pre style={{ margin: 0, fontSize: 12, whiteSpace: "pre-wrap" }}>{taskResult}</pre>
            </Alert>
          )}
          <Form id="submit-task-form">
            <FormGroup label="Agent name" isRequired fieldId="task-agent">
              <TextInput
                id="task-agent"
                value={taskAgent}
                onChange={(_e, v) => setTaskAgent(v)}
                isRequired
              />
            </FormGroup>
            <FormGroup label="Input" fieldId="task-input">
              <TextArea
                id="task-input"
                value={taskInput}
                onChange={(_e, v) => setTaskInput(v)}
                rows={4}
              />
            </FormGroup>
            <FormGroup label="Model" fieldId="task-model">
              <FormSelect
                id="task-model"
                value={taskModel}
                onChange={(_e, v) => setTaskModel(v)}
                aria-label="Model"
              >
                {MODEL_OPTIONS.map((m) => (
                  <FormSelectOption key={m} value={m} label={m} />
                ))}
              </FormSelect>
            </FormGroup>
          </Form>
        </ModalBody>
        <ModalFooter>
          <Button
            variant="primary"
            onClick={handleSubmitTask}
            isDisabled={taskSubmitting}
            isLoading={taskSubmitting}
          >
            Submit
          </Button>
          <Button variant="link" onClick={closeTaskModal}>
            Cancel
          </Button>
        </ModalFooter>
      </Modal>

      <Modal
        variant={ModalVariant.medium}
        isOpen={searchModalOpen}
        onClose={closeSearchModal}
        aria-labelledby="search-knowledge-title"
      >
        <ModalHeader title="Search Knowledge" labelId="search-knowledge-title" />
        <ModalBody>
          {searchError && (
            <Alert variant="danger" title="Search failed" isInline style={{ marginBottom: 16 }}>
              {searchError}
            </Alert>
          )}
          {searchResult && (
            <Alert variant="success" title="Search results" isInline style={{ marginBottom: 16 }}>
              <pre style={{ margin: 0, fontSize: 12, whiteSpace: "pre-wrap", maxHeight: 300, overflow: "auto" }}>
                {searchResult}
              </pre>
            </Alert>
          )}
          <Form id="search-knowledge-form">
            <FormGroup label="Query" isRequired fieldId="search-query">
              <TextInput
                id="search-query"
                value={searchQuery}
                onChange={(_e, v) => setSearchQuery(v)}
                isRequired
              />
            </FormGroup>
          </Form>
        </ModalBody>
        <ModalFooter>
          <Button
            variant="primary"
            onClick={handleSearch}
            isDisabled={searchSubmitting}
            isLoading={searchSubmitting}
          >
            Search
          </Button>
          <Button variant="link" onClick={closeSearchModal}>
            Cancel
          </Button>
        </ModalFooter>
      </Modal>

      <Modal
        variant={ModalVariant.medium}
        isOpen={yamlModalOpen}
        onClose={() => setYamlModalOpen(false)}
        aria-labelledby="create-resource-title"
      >
        <ModalHeader
          title="ArcanaAgent YAML Template"
          description="Apply this Custom Resource to deploy an agent to the cluster."
          labelId="create-resource-title"
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
