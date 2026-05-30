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

const QUICK_ACTIONS = [
  { label: "Deploy Agent", desc: "Create an ArcanaAgent CR", icon: <RobotIcon /> },
  { label: "Register Skill", desc: "Add to the skill catalog", icon: <CubesIcon /> },
  { label: "Run Evaluation", desc: "Test a skill with judges", icon: <CheckCircleIcon /> },
  { label: "View Workflows", desc: "Open Temporal UI", icon: <ClusterIcon />, href: "http://localhost:8233" },
  { label: "Submit Task", desc: "Run an agent task via engine", icon: <AutomationIcon /> },
  { label: "Search Knowledge", desc: "Query the Codex search engine", icon: <SearchIcon /> },
];

function groupByPlane(services: ServiceHealth[]): Record<string, ServiceHealth[]> {
  const groups: Record<string, ServiceHealth[]> = {};
  for (const svc of services) {
    const plane = svc.plane || "unknown";
    if (!groups[plane]) groups[plane] = [];
    groups[plane].push(svc);
  }
  return groups;
}

export const DashboardPage = () => {
  const { health, error, loading, refresh } = useHealth(5000);

  const healthyCount = health?.services.filter((s) => s.status === "healthy").length ?? 0;
  const totalCount = health?.services.length ?? 0;
  const allHealthy = healthyCount === totalCount && totalCount > 0;
  const planeGroups = health ? groupByPlane(health.services) : {};
  const planeOrder = ["infra", "agent", "data", "tool", "model", "govern", "quality", "ops"];

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
                    <div
                      key={action.label}
                      className="action-card"
                      style={{ padding: 16, display: "flex", alignItems: "center", gap: 14, textAlign: "left" }}
                      onClick={() => action.href && window.open(action.href, "_blank")}
                    >
                      <div className="action-card-icon" style={{ margin: 0, flexShrink: 0, width: 40, height: 40 }}>
                        {action.icon}
                      </div>
                      <div>
                        <div style={{ fontWeight: 600, fontSize: 14 }}>{action.label}</div>
                        <div style={{ fontSize: 12, color: "var(--pf-t--global--text--color--subtle)" }}>{action.desc}</div>
                      </div>
                      <div style={{ marginLeft: "auto", color: "var(--pf-t--global--text--color--subtle)" }}>
                        <ArrowRightIcon />
                      </div>
                    </div>
                  ))}
                </div>
              </CardBody>
            </Card>
          </GridItem>

          <GridItem span={12}>
            <Card className="stat-card">
              <CardTitle>
                <div className="section-title" style={{ margin: 0 }}>Agentic Protocol Stack</div>
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
                  <Button variant="link" icon={<PlusCircleIcon />} isInline>
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
    </>
  );
};
