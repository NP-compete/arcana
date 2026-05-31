import { useState, useEffect, useCallback } from "react";
import {
  PageSection,
  Title,
  Card,
  CardTitle,
  CardBody,
  Grid,
  GridItem,
  Content,
  Divider,
  Label,
  Button,
  Tabs,
  Tab,
  TabTitleText,
  DescriptionList,
  DescriptionListGroup,
  DescriptionListTerm,
  DescriptionListDescription,
  Alert,
  Spinner,
  TextInput,
  FormGroup,
  Form,
  FormSelect,
  FormSelectOption,
  Checkbox,
} from "@patternfly/react-core";
import { Table, Thead, Tr, Th, Tbody, Td } from "@patternfly/react-table";
import {
  ExternalLinkAltIcon,
  ServerIcon,
  ClusterIcon,
  KeyIcon,
  UsersIcon,
  ShieldAltIcon,
  ListIcon,
} from "@patternfly/react-icons";

const ENDPOINTS = [
  { name: "API Gateway", url: "http://localhost:8080", port: 8080, internal: true },
  { name: "AG-UI Events", url: "http://localhost:8084/events", port: 8084, internal: true },
  { name: "Temporal UI", url: "http://localhost:8233", port: 8233, internal: false },
  { name: "MinIO Console", url: "http://localhost:9001", port: 9001, internal: false },
  { name: "NATS Monitor", url: "http://localhost:8222", port: 8222, internal: false },
];

const CLUSTER_INFO = [
  { label: "Context", value: "kind-arcana-dev" },
  { label: "Kubernetes", value: "v1.35.0" },
  { label: "Runtime", value: "Kind (single-node)" },
  { label: "Container Engine", value: "Docker / Podman (auto-detected)" },
  { label: "CRD Group", value: "arcana.io/v1alpha1" },
];

interface APIKeyData {
  id: string;
  name: string;
  prefix: string;
  user_id: string;
  tenant: string;
  roles: string[];
  scopes: string[];
  rate_limit_per_second: number;
  created_at: string;
  last_used_at: string;
  revoked: boolean;
}

interface TenantData {
  id: string;
  name: string;
  namespace: string;
  status: string;
  max_agents: number;
  max_models: number;
  budget_limit_usd: number;
  created_at: string;
}

interface AuditEntryData {
  id: string;
  timestamp: string;
  actor: string;
  tenant: string;
  action: string;
  resource: string;
  detail: string;
  ip: string;
}

interface ComplianceCheck {
  framework: string;
  control: string;
  status: string;
  evidence: string;
}

interface EnterpriseConfig {
  auth_mode: string;
  rbac_enabled: boolean;
  rate_limiting_enabled: boolean;
  audit_enabled: boolean;
  multi_tenancy: boolean;
  compliance_frameworks: string[];
  api_key_count: number;
  tenant_count: number;
  roles: string[];
}

const PlatformTab = () => (
  <Grid hasGutter>
    <GridItem span={7}>
      <Card className="stat-card" isFullHeight>
        <CardTitle>
          <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
            <ServerIcon /> Service Endpoints
          </div>
        </CardTitle>
        <CardBody>
          <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
            {ENDPOINTS.map((ep) => (
              <div className="service-row" key={ep.name}>
                <div className="service-row-name" style={{ flex: 1 }}>{ep.name}</div>
                <Label color="grey" isCompact style={{ fontFamily: "var(--pf-t--global--font--family--mono)" }}>
                  :{ep.port}
                </Label>
                {ep.internal ? (
                  <Label color="blue" isCompact>Internal</Label>
                ) : (
                  <Button
                    variant="link"
                    size="sm"
                    isInline
                    icon={<ExternalLinkAltIcon />}
                    component="a"
                    href={ep.url}
                    target="_blank"
                  >
                    Open
                  </Button>
                )}
              </div>
            ))}
          </div>
        </CardBody>
      </Card>
    </GridItem>

    <GridItem span={5}>
      <Card className="stat-card" isFullHeight>
        <CardTitle>
          <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
            <ClusterIcon /> Cluster
          </div>
        </CardTitle>
        <CardBody>
          <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
            {CLUSTER_INFO.map((item) => (
              <div key={item.label}>
                <div style={{ fontSize: 12, fontWeight: 600, color: "var(--pf-t--global--text--color--subtle)", textTransform: "uppercase", letterSpacing: 0.5, marginBottom: 4 }}>
                  {item.label}
                </div>
                <div style={{ fontWeight: 500, fontSize: 14 }}>{item.value}</div>
              </div>
            ))}
          </div>
        </CardBody>
      </Card>
    </GridItem>
  </Grid>
);

const SecurityTab = () => {
  const [config, setConfig] = useState<EnterpriseConfig | null>(null);
  const [keys, setKeys] = useState<APIKeyData[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchData = useCallback(async () => {
    try {
      const [configRes, keysRes] = await Promise.all([
        fetch("/api/v1/enterprise/config"),
        fetch("/api/v1/auth/keys"),
      ]);
      if (configRes.ok) setConfig(await configRes.json());
      if (keysRes.ok) {
        const kd = await keysRes.json();
        setKeys(kd.keys ?? []);
      }
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  if (loading) return <Spinner size="lg" />;

  return (
    <Grid hasGutter>
      <GridItem span={5}>
        <Card>
          <CardTitle><KeyIcon /> Authentication & RBAC</CardTitle>
          <CardBody>
            {config && (
              <DescriptionList isHorizontal isCompact>
                <DescriptionListGroup>
                  <DescriptionListTerm>Auth Mode</DescriptionListTerm>
                  <DescriptionListDescription>
                    <Label color={config.auth_mode === "open" ? "orange" : "green"} isCompact>
                      {config.auth_mode}
                    </Label>
                  </DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                  <DescriptionListTerm>RBAC</DescriptionListTerm>
                  <DescriptionListDescription>
                    <Label color="green" isCompact>enabled</Label>
                  </DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                  <DescriptionListTerm>Rate Limiting</DescriptionListTerm>
                  <DescriptionListDescription>
                    <Label color="green" isCompact>enabled</Label>
                  </DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                  <DescriptionListTerm>Roles</DescriptionListTerm>
                  <DescriptionListDescription>
                    {config.roles.map(r => <Label key={r} color="blue" isCompact style={{ marginRight: 4 }}>{r}</Label>)}
                  </DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                  <DescriptionListTerm>API Keys</DescriptionListTerm>
                  <DescriptionListDescription>{config.api_key_count}</DescriptionListDescription>
                </DescriptionListGroup>
              </DescriptionList>
            )}
          </CardBody>
        </Card>
      </GridItem>

      <GridItem span={7}>
        <Card>
          <CardTitle>API Keys</CardTitle>
          <CardBody>
            <Table variant="compact">
              <Thead><Tr>
                <Th>Name</Th><Th>Prefix</Th><Th>Roles</Th><Th>Rate</Th><Th>Status</Th>
              </Tr></Thead>
              <Tbody>
                {keys.map(k => (
                  <Tr key={k.id}>
                    <Td>{k.name}</Td>
                    <Td><code>{k.prefix}</code></Td>
                    <Td>{k.roles?.join(", ")}</Td>
                    <Td>{k.rate_limit_per_second}/s</Td>
                    <Td>
                      <Label color={k.revoked ? "red" : "green"} isCompact>
                        {k.revoked ? "revoked" : "active"}
                      </Label>
                    </Td>
                  </Tr>
                ))}
              </Tbody>
            </Table>
          </CardBody>
        </Card>
      </GridItem>
    </Grid>
  );
};

const TenantsTab = () => {
  const [tenants, setTenants] = useState<TenantData[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetch("/api/v1/tenants")
      .then(r => r.json())
      .then(d => setTenants(d.tenants ?? []))
      .finally(() => setLoading(false));
  }, []);

  if (loading) return <Spinner size="lg" />;

  return (
    <Card>
      <CardTitle><UsersIcon /> Tenants</CardTitle>
      <CardBody>
        <Table variant="compact">
          <Thead><Tr>
            <Th>ID</Th><Th>Name</Th><Th>Namespace</Th><Th>Status</Th>
            <Th>Max Agents</Th><Th>Max Models</Th><Th>Budget Limit</Th>
          </Tr></Thead>
          <Tbody>
            {tenants.map(t => (
              <Tr key={t.id}>
                <Td>{t.id}</Td>
                <Td>{t.name}</Td>
                <Td><code>{t.namespace}</code></Td>
                <Td><Label color="green" isCompact>{t.status}</Label></Td>
                <Td>{t.max_agents}</Td>
                <Td>{t.max_models}</Td>
                <Td>${t.budget_limit_usd?.toLocaleString()}</Td>
              </Tr>
            ))}
          </Tbody>
        </Table>
      </CardBody>
    </Card>
  );
};

const AuditTab = () => {
  const [entries, setEntries] = useState<AuditEntryData[]>([]);
  const [stats, setStats] = useState<Record<string, unknown> | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    Promise.all([
      fetch("/api/v1/enterprise/audit").then(r => r.json()),
      fetch("/api/v1/enterprise/audit/stats").then(r => r.json()),
    ]).then(([auditData, statsData]) => {
      setEntries(auditData.entries ?? []);
      setStats(statsData);
    }).finally(() => setLoading(false));
  }, []);

  if (loading) return <Spinner size="lg" />;

  return (
    <Grid hasGutter>
      <GridItem span={4}>
        <Card>
          <CardTitle>Audit Stats</CardTitle>
          <CardBody>
            {stats && (
              <DescriptionList isCompact>
                <DescriptionListGroup>
                  <DescriptionListTerm>Total Entries</DescriptionListTerm>
                  <DescriptionListDescription>{(stats as { total_entries: number }).total_entries}</DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                  <DescriptionListTerm>Chain Integrity</DescriptionListTerm>
                  <DescriptionListDescription>
                    <Label color={(stats as { chain_intact: boolean }).chain_intact ? "green" : "red"} isCompact>
                      {(stats as { chain_intact: boolean }).chain_intact ? "intact" : "broken"}
                    </Label>
                  </DescriptionListDescription>
                </DescriptionListGroup>
              </DescriptionList>
            )}
          </CardBody>
        </Card>
      </GridItem>
      <GridItem span={8}>
        <Card>
          <CardTitle><ListIcon /> Recent Audit Entries</CardTitle>
          <CardBody>
            <Table variant="compact">
              <Thead><Tr>
                <Th>Time</Th><Th>Actor</Th><Th>Action</Th><Th>Resource</Th><Th>Detail</Th>
              </Tr></Thead>
              <Tbody>
                {entries.slice(0, 20).map(e => (
                  <Tr key={e.id}>
                    <Td style={{ fontSize: 12 }}>{new Date(e.timestamp).toLocaleTimeString()}</Td>
                    <Td>{e.actor}</Td>
                    <Td><Label isCompact color="blue">{e.action}</Label></Td>
                    <Td>{e.resource}</Td>
                    <Td style={{ fontSize: 12, maxWidth: 200, overflow: "hidden", textOverflow: "ellipsis" }}>{e.detail}</Td>
                  </Tr>
                ))}
              </Tbody>
            </Table>
          </CardBody>
        </Card>
      </GridItem>
    </Grid>
  );
};

interface RoleData {
  name: string;
  description: string;
  permissions: { resource: string; verbs: string[] }[];
}

const VERB_SHORT: Record<string, string> = { create: "C", read: "R", update: "U", delete: "D" };
const VERB_COLORS: Record<string, string> = {
  C: "#22c55e", R: "#5b8def", U: "#f59e0b", D: "#ef4444",
};

const ROLE_ORDER = ["admin", "developer", "data-engineer", "sre", "auditor", "user"];
const ROLE_DISPLAY: Record<string, { color: string; icon: string }> = {
  admin: { color: "#a855f7", icon: "A" },
  developer: { color: "#5b8def", icon: "D" },
  "data-engineer": { color: "#06b6d4", icon: "DE" },
  sre: { color: "#f59e0b", icon: "S" },
  auditor: { color: "#ef4444", icon: "Au" },
  user: { color: "#22c55e", icon: "U" },
};

const RBACTab = () => {
  const [roles, setRoles] = useState<RoleData[]>([]);
  const [loading, setLoading] = useState(true);
  const [view, setView] = useState<"matrix" | "list">("matrix");

  useEffect(() => {
    fetch("/api/v1/auth/roles")
      .then((r) => r.json())
      .then((d) => {
        const sorted = (d.roles ?? []).sort(
          (a: RoleData, b: RoleData) => ROLE_ORDER.indexOf(a.name) - ROLE_ORDER.indexOf(b.name),
        );
        setRoles(sorted);
      })
      .finally(() => setLoading(false));
  }, []);

  if (loading) return <Spinner size="lg" />;

  const allResources = Array.from(
    new Set(roles.flatMap((r) => r.permissions.map((p) => p.resource))),
  ).sort((a, b) => {
    if (a === "*") return -1;
    if (b === "*") return 1;
    return a.localeCompare(b);
  });

  const getVerbs = (role: RoleData, resource: string): string[] => {
    const wildcard = role.permissions.find((p) => p.resource === "*");
    if (wildcard) return wildcard.verbs;
    const perm = role.permissions.find((p) => p.resource === resource);
    return perm ? perm.verbs : [];
  };

  return (
    <Grid hasGutter>
      <GridItem span={12}>
        <Card>
          <CardTitle>
            <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between" }}>
              <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
                <ShieldAltIcon /> Role-Based Access Control
              </div>
              <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
                <span style={{ fontSize: 12, color: "var(--pf-t--global--text--color--subtle)" }}>
                  {roles.length} roles · {allResources.length} resources
                </span>
                <Button
                  variant={view === "matrix" ? "primary" : "secondary"}
                  size="sm"
                  onClick={() => setView("matrix")}
                >
                  Matrix
                </Button>
                <Button
                  variant={view === "list" ? "primary" : "secondary"}
                  size="sm"
                  onClick={() => setView("list")}
                >
                  By Role
                </Button>
              </div>
            </div>
          </CardTitle>
          <CardBody>
            <div style={{ display: "flex", gap: 8, marginBottom: 16, flexWrap: "wrap" }}>
              {Object.entries(VERB_SHORT).map(([verb, short]) => (
                <span key={verb} style={{ display: "flex", alignItems: "center", gap: 4, fontSize: 12 }}>
                  <span
                    style={{
                      width: 20, height: 20, borderRadius: 4,
                      background: `${VERB_COLORS[short]}22`, color: VERB_COLORS[short],
                      display: "inline-flex", alignItems: "center", justifyContent: "center",
                      fontWeight: 700, fontSize: 10,
                    }}
                  >
                    {short}
                  </span>
                  {verb}
                </span>
              ))}
              <span style={{ display: "flex", alignItems: "center", gap: 4, fontSize: 12 }}>
                <span style={{
                  width: 20, height: 20, borderRadius: 4, background: "rgba(128,128,128,0.1)",
                  display: "inline-flex", alignItems: "center", justifyContent: "center",
                  fontSize: 10, color: "#666",
                }}>—</span>
                no access
              </span>
            </div>

            {view === "matrix" ? (
              <div style={{ overflowX: "auto" }}>
                <table style={{ width: "100%", borderCollapse: "collapse", fontSize: 13 }}>
                  <thead>
                    <tr style={{ borderBottom: "2px solid var(--pf-t--global--border--color--default)" }}>
                      <th style={{ textAlign: "left", padding: "8px 12px", fontWeight: 600, position: "sticky", left: 0, background: "var(--pf-t--global--background--color--primary--default)", zIndex: 1, minWidth: 140 }}>
                        Resource
                      </th>
                      {roles.map((role) => {
                        const rd = ROLE_DISPLAY[role.name] ?? { color: "#888", icon: "?" };
                        return (
                          <th key={role.name} style={{ textAlign: "center", padding: "8px 6px", minWidth: 90 }}>
                            <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: 4 }}>
                              <span
                                style={{
                                  width: 28, height: 28, borderRadius: 8,
                                  background: `${rd.color}22`, border: `1px solid ${rd.color}44`,
                                  display: "flex", alignItems: "center", justifyContent: "center",
                                  fontSize: 11, fontWeight: 700, color: rd.color,
                                }}
                              >
                                {rd.icon}
                              </span>
                              <span style={{ fontSize: 11, fontWeight: 600 }}>{role.name}</span>
                            </div>
                          </th>
                        );
                      })}
                    </tr>
                  </thead>
                  <tbody>
                    {allResources.map((res, idx) => (
                      <tr
                        key={res}
                        style={{
                          borderBottom: "1px solid var(--pf-t--global--border--color--default)",
                          background: idx % 2 === 0 ? "transparent" : "rgba(0,0,0,0.02)",
                        }}
                      >
                        <td style={{
                          padding: "6px 12px", fontWeight: 500, fontFamily: "var(--pf-t--global--font--family--mono)",
                          fontSize: 12, position: "sticky", left: 0,
                          background: idx % 2 === 0 ? "var(--pf-t--global--background--color--primary--default)" : "var(--pf-t--global--background--color--secondary--default)",
                          zIndex: 1,
                        }}>
                          {res === "*" ? "* (all)" : res}
                        </td>
                        {roles.map((role) => {
                          const verbs = getVerbs(role, res);
                          return (
                            <td key={role.name} style={{ textAlign: "center", padding: "6px 4px" }}>
                              {verbs.length > 0 ? (
                                <div style={{ display: "flex", gap: 2, justifyContent: "center" }}>
                                  {["create", "read", "update", "delete"].map((v) => {
                                    const has = verbs.includes(v) || verbs.includes("*");
                                    const short = VERB_SHORT[v];
                                    return (
                                      <span
                                        key={v}
                                        title={has ? v : `no ${v}`}
                                        style={{
                                          width: 18, height: 18, borderRadius: 3,
                                          display: "inline-flex", alignItems: "center", justifyContent: "center",
                                          fontSize: 9, fontWeight: 700,
                                          background: has ? `${VERB_COLORS[short]}18` : "transparent",
                                          color: has ? VERB_COLORS[short] : "rgba(128,128,128,0.25)",
                                          border: has ? `1px solid ${VERB_COLORS[short]}44` : "1px solid transparent",
                                        }}
                                      >
                                        {short}
                                      </span>
                                    );
                                  })}
                                </div>
                              ) : (
                                <span style={{ color: "rgba(128,128,128,0.3)", fontSize: 12 }}>—</span>
                              )}
                            </td>
                          );
                        })}
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : (
              <Grid hasGutter>
                {roles.map((role) => {
                  const rd = ROLE_DISPLAY[role.name] ?? { color: "#888", icon: "?" };
                  return (
                    <GridItem span={6} key={role.name}>
                      <Card>
                        <CardTitle>
                          <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
                            <span
                              style={{
                                width: 32, height: 32, borderRadius: 8,
                                background: `${rd.color}22`, border: `1px solid ${rd.color}44`,
                                display: "flex", alignItems: "center", justifyContent: "center",
                                fontSize: 13, fontWeight: 700, color: rd.color,
                              }}
                            >
                              {rd.icon}
                            </span>
                            <div>
                              <div style={{ fontWeight: 600, fontSize: 15 }}>{role.name}</div>
                              <div style={{ fontSize: 12, color: "var(--pf-t--global--text--color--subtle)", fontWeight: 400 }}>
                                {role.description}
                              </div>
                            </div>
                          </div>
                        </CardTitle>
                        <CardBody>
                          <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
                            {role.permissions.map((p) => (
                              <div
                                key={p.resource}
                                style={{
                                  display: "flex", alignItems: "center", justifyContent: "space-between",
                                  padding: "4px 8px", borderRadius: 6,
                                  background: "rgba(0,0,0,0.03)", fontSize: 13,
                                }}
                              >
                                <code style={{ fontWeight: 500, fontSize: 12 }}>
                                  {p.resource === "*" ? "* (all resources)" : p.resource}
                                </code>
                                <div style={{ display: "flex", gap: 3 }}>
                                  {p.verbs.map((v) => {
                                    const short = VERB_SHORT[v] ?? v[0].toUpperCase();
                                    const color = VERB_COLORS[short] ?? "#888";
                                    return (
                                      <span
                                        key={v}
                                        title={v}
                                        style={{
                                          width: 20, height: 20, borderRadius: 4,
                                          display: "inline-flex", alignItems: "center", justifyContent: "center",
                                          fontSize: 10, fontWeight: 700,
                                          background: `${color}18`, color, border: `1px solid ${color}44`,
                                        }}
                                      >
                                        {short}
                                      </span>
                                    );
                                  })}
                                </div>
                              </div>
                            ))}
                          </div>
                          <div style={{ marginTop: 10, fontSize: 11, color: "var(--pf-t--global--text--color--subtle)" }}>
                            {role.permissions.length} resource{role.permissions.length !== 1 ? "s" : ""}
                            {" · "}
                            {role.permissions.reduce((s, p) => s + p.verbs.length, 0)} total permissions
                          </div>
                        </CardBody>
                      </Card>
                    </GridItem>
                  );
                })}
              </Grid>
            )}
          </CardBody>
        </Card>
      </GridItem>
    </Grid>
  );
};

const ComplianceTab = () => {
  const [framework, setFramework] = useState("soc2");
  const [report, setReport] = useState<{ framework: string; total_checks: number; passed: number; checks: ComplianceCheck[] } | null>(null);
  const [loading, setLoading] = useState(false);

  const runReport = async () => {
    setLoading(true);
    try {
      const res = await fetch(`/api/v1/compliance?framework=${framework}`);
      if (res.ok) setReport(await res.json());
    } finally {
      setLoading(false);
    }
  };

  return (
    <Grid hasGutter>
      <GridItem span={12}>
        <Card>
          <CardTitle><ShieldAltIcon /> Compliance Report</CardTitle>
          <CardBody>
            <div style={{ display: "flex", gap: 12, alignItems: "flex-end", marginBottom: 16 }}>
              <FormGroup label="Framework" fieldId="compliance-framework">
                <FormSelect
                  id="compliance-framework"
                  value={framework}
                  onChange={(_e, v) => setFramework(v)}
                  style={{ width: 200 }}
                >
                  <FormSelectOption value="soc2" label="SOC 2 Type II" />
                  <FormSelectOption value="gdpr" label="GDPR" />
                  <FormSelectOption value="hipaa" label="HIPAA" />
                  <FormSelectOption value="iso27001" label="ISO 27001" />
                  <FormSelectOption value="euaiact" label="EU AI Act" />
                </FormSelect>
              </FormGroup>
              <Button onClick={runReport} isLoading={loading} isDisabled={loading}>
                Generate Report
              </Button>
            </div>

            {report && (
              <>
                <Alert
                  variant={report.passed === report.total_checks ? "success" : "warning"}
                  isInline
                  title={`${report.passed}/${report.total_checks} checks passed for ${report.framework.toUpperCase()}`}
                  style={{ marginBottom: 16 }}
                />
                <Table variant="compact">
                  <Thead><Tr>
                    <Th>Control</Th><Th>Status</Th><Th>Evidence</Th>
                  </Tr></Thead>
                  <Tbody>
                    {report.checks.map((c, i) => (
                      <Tr key={i}>
                        <Td style={{ fontWeight: 600 }}>{c.control}</Td>
                        <Td>
                          <Label
                            isCompact
                            color={c.status === "pass" ? "green" : c.status === "partial" ? "orange" : c.status === "fail" ? "red" : "grey"}
                          >
                            {c.status}
                          </Label>
                        </Td>
                        <Td style={{ fontSize: 13 }}>{c.evidence}</Td>
                      </Tr>
                    ))}
                  </Tbody>
                </Table>
              </>
            )}
          </CardBody>
        </Card>
      </GridItem>
    </Grid>
  );
};

export const SettingsPage = () => {
  const [activeTab, setActiveTab] = useState(0);

  return (
    <>
      <PageSection hasBodyWrapper={false}>
        <Title headingLevel="h1" size="2xl">Settings</Title>
        <Content component="p" style={{ marginTop: 4 }}>
          Platform configuration, security, multi-tenancy, and compliance.
        </Content>
      </PageSection>
      <Divider />
      <PageSection hasBodyWrapper={false}>
        <Tabs activeKey={activeTab} onSelect={(_e, k) => setActiveTab(k as number)}>
          <Tab eventKey={0} title={<TabTitleText>Platform</TabTitleText>}>
            <div style={{ marginTop: 16 }}><PlatformTab /></div>
          </Tab>
          <Tab eventKey={1} title={<TabTitleText>RBAC</TabTitleText>}>
            <div style={{ marginTop: 16 }}><RBACTab /></div>
          </Tab>
          <Tab eventKey={2} title={<TabTitleText>Security & Auth</TabTitleText>}>
            <div style={{ marginTop: 16 }}><SecurityTab /></div>
          </Tab>
          <Tab eventKey={3} title={<TabTitleText>Tenants</TabTitleText>}>
            <div style={{ marginTop: 16 }}><TenantsTab /></div>
          </Tab>
          <Tab eventKey={4} title={<TabTitleText>Audit Log</TabTitleText>}>
            <div style={{ marginTop: 16 }}><AuditTab /></div>
          </Tab>
          <Tab eventKey={5} title={<TabTitleText>Compliance</TabTitleText>}>
            <div style={{ marginTop: 16 }}><ComplianceTab /></div>
          </Tab>
        </Tabs>
      </PageSection>
    </>
  );
};
