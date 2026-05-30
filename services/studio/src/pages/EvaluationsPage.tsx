import {
  PageSection,
  Title,
  Card,
  CardBody,
  Grid,
  GridItem,
  Content,
  Divider,
  Label,
} from "@patternfly/react-core";
import { ChartBarIcon } from "@patternfly/react-icons";

const JUDGE_TIERS = [
  { tier: "Deterministic", desc: "Exact match, regex, JSON schema validators", color: "#38a169", badge: "Fastest" },
  { tier: "Script", desc: "Custom Python/Go scoring functions", color: "#3182ce", badge: "Flexible" },
  { tier: "LLM", desc: "GPT-4o / Claude judge with rubric scoring", color: "#805ad5", badge: "Nuanced" },
  { tier: "Trajectory", desc: "Full agent trace analysis across steps", color: "#d69e2e", badge: "Deepest" },
];

const BADGES = [
  { name: "Gold", color: "#d69e2e", bg: "#fefcbf", criteria: ">95% pass, 5+ security judges" },
  { name: "Silver", color: "#718096", bg: "#e2e8f0", criteria: ">85% pass, 3+ judges" },
  { name: "Bronze", color: "#c05621", bg: "#feebc8", criteria: ">70% pass, basic coverage" },
  { name: "Untested", color: "#a0aec0", bg: "#f7fafc", criteria: "No eval suite configured" },
];

export const EvaluationsPage = () => (
  <>
    <PageSection hasBodyWrapper={false}>
      <Title headingLevel="h1" size="2xl">Evaluations</Title>
      <Content component="p" style={{ marginTop: 4 }}>
        Three-condition testing with a 4-tier judge pipeline and quality badges.
      </Content>
    </PageSection>
    <Divider />
    <PageSection hasBodyWrapper={false}>
      <div className="arcana-empty-state" style={{ paddingBottom: 32 }}>
        <div className="arcana-empty-icon">
          <ChartBarIcon />
        </div>
        <Title headingLevel="h2" size="xl" style={{ marginBottom: 8 }}>
          No evaluation suites yet
        </Title>
        <Content component="p" style={{ maxWidth: 480, margin: "0 auto", color: "var(--pf-t--global--text--color--subtle)" }}>
          Create an ArcanaEvalSuite CR to define test cases, judges, and quality gates for your skills.
        </Content>
      </div>

      <div className="section-title">Judge Pipeline</div>
      <Grid hasGutter>
        {JUDGE_TIERS.map((j, i) => (
          <GridItem span={3} key={j.tier}>
            <Card className="stat-card" isFullHeight>
              <CardBody>
                <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 12 }}>
                  <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
                    <div style={{
                      width: 28, height: 28, borderRadius: 8,
                      background: `${j.color}15`, color: j.color,
                      display: "flex", alignItems: "center", justifyContent: "center",
                      fontWeight: 800, fontSize: 14,
                    }}>
                      {i + 1}
                    </div>
                    <span style={{ fontWeight: 700, fontSize: 14 }}>{j.tier}</span>
                  </div>
                  <Label isCompact color="blue">{j.badge}</Label>
                </div>
                <Content component="p" style={{ fontSize: 13, color: "var(--pf-t--global--text--color--subtle)" }}>
                  {j.desc}
                </Content>
              </CardBody>
            </Card>
          </GridItem>
        ))}
      </Grid>

      <div className="section-title" style={{ marginTop: 32 }}>Quality Badges</div>
      <div style={{ display: "flex", gap: 16, flexWrap: "wrap" }}>
        {BADGES.map((b) => (
          <Card className="stat-card" key={b.name} style={{ flex: 1, minWidth: 200 }}>
            <CardBody>
              <div style={{ display: "flex", alignItems: "center", gap: 10, marginBottom: 8 }}>
                <div style={{
                  width: 24, height: 24, borderRadius: "50%",
                  background: b.bg, border: `2px solid ${b.color}`,
                }} />
                <span style={{ fontWeight: 700, fontSize: 16, color: b.color }}>{b.name}</span>
              </div>
              <Content component="p" style={{ fontSize: 12, color: "var(--pf-t--global--text--color--subtle)" }}>
                {b.criteria}
              </Content>
            </CardBody>
          </Card>
        ))}
      </div>
    </PageSection>
  </>
);
