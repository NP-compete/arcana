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
} from "@patternfly/react-icons";
import { useHealth } from "../hooks/useHealth";

const SERVICE_META: Record<string, { icon: React.ReactNode; color: string; desc: string }> = {
  PostgreSQL: { icon: <DatabaseIcon />, color: "#3182ce", desc: "pgvector-enabled relational store" },
  Redis: { icon: <ServerIcon />, color: "#e53e3e", desc: "In-memory cache & pub/sub" },
  Temporal: { icon: <ClusterIcon />, color: "#805ad5", desc: "Durable workflow execution" },
  MinIO: { icon: <StorageDomainIcon />, color: "#d69e2e", desc: "S3-compatible object store" },
  NATS: { icon: <CloudUploadAltIcon />, color: "#38a169", desc: "JetStream event backbone" },
};

const PROTOCOLS = [
  { name: "MCP", desc: "Model Context Protocol — tool access" },
  { name: "A2A", desc: "Agent-to-Agent — primary mesh" },
  { name: "ACP", desc: "Agent Communication Protocol — IBM adapter" },
  { name: "AG-UI", desc: "Agent-User Interaction — streaming" },
  { name: "ACS", desc: "Agent Control Standard — lifecycle" },
];

const CRDS = [
  { name: "ArcanaAgent", short: "aag", purpose: "Agent lifecycle & config", icon: <RobotIcon /> },
  { name: "ArcanaTenant", short: "aten", purpose: "Multi-tenant isolation", icon: <ServerIcon /> },
  { name: "ArcanaSkillRegistry", short: "askr", purpose: "Skill catalog & versioning", icon: <CubesIcon /> },
  { name: "ArcanaEvalSuite", short: "aes", purpose: "Skill evaluation pipelines", icon: <CheckCircleIcon /> },
  { name: "ArcanaRole", short: "arole", purpose: "RBAC + ABAC policies", icon: <ClusterIcon /> },
  { name: "ArcanaBudget", short: "abud", purpose: "FinOps token/compute budgets", icon: <StorageDomainIcon /> },
  { name: "ArcanaBackupPolicy", short: "abkp", purpose: "Backup scheduling & retention", icon: <DatabaseIcon /> },
  { name: "ArcanaPromotion", short: "aprom", purpose: "Environment promotion gates", icon: <ArrowRightIcon /> },
];

const QUICK_ACTIONS = [
  { label: "Deploy Agent", desc: "Create an ArcanaAgent CR", icon: <RobotIcon /> },
  { label: "Register Skill", desc: "Add to the skill catalog", icon: <CubesIcon /> },
  { label: "Run Evaluation", desc: "Test a skill with judges", icon: <CheckCircleIcon /> },
  { label: "View Workflows", desc: "Open Temporal UI", icon: <ClusterIcon />, href: "http://localhost:8233" },
];

export const DashboardPage = () => {
  const { health, error, loading, refresh } = useHealth(5000);

  const healthyCount = health?.services.filter((s) => s.status === "healthy").length ?? 0;
  const totalCount = health?.services.length ?? 0;
  const allHealthy = healthyCount === totalCount && totalCount > 0;

  return (
    <>
      {/* Welcome banner */}
      <PageSection hasBodyWrapper={false} style={{ paddingBottom: 0 }}>
        <div className="welcome-banner">
          <h2>Welcome to Arcana Studio</h2>
          <p>
            Your Kubernetes-native AI platform is running. Build agents, manage skills,
            enforce guardrails, and monitor everything from one place.
          </p>
        </div>
      </PageSection>

      {error && (
        <PageSection hasBodyWrapper={false} style={{ paddingBottom: 0 }}>
          <Alert variant="warning" title="API unreachable" isInline isExpandable>
            <p>Cannot reach arcana-api at <code>localhost:8080</code>.</p>
            <p>Start it with: <code>./bin/arcana-api</code></p>
          </Alert>
        </PageSection>
      )}

      {/* Stat cards */}
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
                    <div className="stat-card-value" style={{ color: "var(--arcana-indigo)" }}>
                      {CRDS.length}
                    </div>
                    <div className="stat-card-label">CRDs Installed</div>
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
                    <div className="stat-card-value" style={{ color: "var(--arcana-purple)" }}>
                      {PROTOCOLS.length}
                    </div>
                    <div className="stat-card-label">Protocols Active</div>
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

      {/* Main content */}
      <PageSection hasBodyWrapper={false}>
        <Grid hasGutter>
          {/* Service health */}
          <GridItem span={8}>
            <Card className="stat-card" isFullHeight>
              <CardTitle>
                <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
                  <div className="section-title" style={{ margin: 0 }}>Infrastructure Health</div>
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
                  <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
                    {health?.services.map((svc) => {
                      const meta = SERVICE_META[svc.name] ?? { icon: <ServerIcon />, color: "#718096", desc: "" };
                      return (
                        <div className="service-row" key={svc.name}>
                          <div className="service-row-icon" style={{ background: `${meta.color}15`, color: meta.color }}>
                            {meta.icon}
                          </div>
                          <div style={{ flex: 1 }}>
                            <div className="service-row-name">{svc.name}</div>
                            <div style={{ fontSize: 12, color: "var(--pf-t--global--text--color--subtle)" }}>{meta.desc}</div>
                          </div>
                          <span className={`health-dot ${svc.status === "healthy" ? "health-dot-healthy" : "health-dot-down"}`} />
                          <span className="service-row-port">:{svc.port}</span>
                          <span className="service-row-latency">{svc.latency}</span>
                        </div>
                      );
                    })}
                  </div>
                )}
              </CardBody>
            </Card>
          </GridItem>

          {/* Quick actions */}
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

          {/* Protocols */}
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

          {/* CRDs */}
          <GridItem span={12}>
            <Card className="stat-card">
              <CardTitle>
                <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
                  <div className="section-title" style={{ margin: 0 }}>Custom Resource Definitions</div>
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
                          <span style={{ color: "var(--arcana-indigo)" }}>{crd.icon}</span>
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
