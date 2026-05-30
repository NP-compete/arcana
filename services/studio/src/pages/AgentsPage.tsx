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
} from "@patternfly/react-core";
import {
  RobotIcon,
  PlusCircleIcon,
  CodeIcon,
  CubesIcon,
  AutomationIcon,
} from "@patternfly/react-icons";

const AGENT_TEMPLATES = [
  {
    name: "Conversational Agent",
    desc: "Chat-based agent with memory, skill execution, and guardrails",
    model: "gpt-4o",
    skills: ["search", "code-gen", "summarize"],
    icon: <RobotIcon />,
  },
  {
    name: "Code Assistant",
    desc: "Specialized for code review, generation, and refactoring",
    model: "claude-sonnet",
    skills: ["code-review", "refactor", "test-gen"],
    icon: <CodeIcon />,
  },
  {
    name: "Data Pipeline Agent",
    desc: "Orchestrates ETL workflows and data quality checks",
    model: "gpt-4o-mini",
    skills: ["sql-gen", "schema-validate", "data-profile"],
    icon: <CubesIcon />,
  },
  {
    name: "Ops Automation Agent",
    desc: "SRE agent for incident response and infrastructure management",
    model: "claude-sonnet",
    skills: ["k8s-ops", "log-analyze", "runbook-exec"],
    icon: <AutomationIcon />,
  },
];

export const AgentsPage = () => (
  <>
    <PageSection hasBodyWrapper={false}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
        <div>
          <Title headingLevel="h1" size="2xl">Agents</Title>
          <Content component="p" style={{ marginTop: 4 }}>
            Deploy and manage Kubernetes-native AI agents.
          </Content>
        </div>
        <Button variant="primary" icon={<PlusCircleIcon />}>
          Deploy Agent
        </Button>
      </div>
    </PageSection>
    <Divider />
    <PageSection hasBodyWrapper={false}>
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

      <div className="section-title">Agent Templates</div>
      <Grid hasGutter>
        {AGENT_TEMPLATES.map((t) => (
          <GridItem span={6} key={t.name}>
            <Card className="stat-card" style={{ cursor: "pointer" }}>
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
                  <Button variant="secondary" size="sm" style={{ alignSelf: "center" }}>
                    Deploy
                  </Button>
                </div>
              </CardBody>
            </Card>
          </GridItem>
        ))}
      </Grid>

      <div style={{ marginTop: 24, textAlign: "center" }}>
        <Button variant="link" icon={<CodeIcon />}>
          View YAML example for custom agent
        </Button>
      </div>
    </PageSection>
  </>
);
