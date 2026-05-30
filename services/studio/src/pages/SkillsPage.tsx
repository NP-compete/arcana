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
import { CubesIcon, PlusCircleIcon } from "@patternfly/react-icons";

const SKILL_TIERS = [
  {
    tier: "Planning",
    color: "#805ad5" as const,
    desc: "High-level orchestration skills that decompose goals into sub-tasks",
    examples: ["task-decompose", "multi-step-plan", "goal-refine"],
  },
  {
    tier: "Functional",
    color: "#3182ce" as const,
    desc: "Domain-specific capabilities like code generation or data analysis",
    examples: ["code-gen", "sql-query", "summarize", "translate"],
  },
  {
    tier: "Atomic",
    color: "#38a169" as const,
    desc: "Low-level tool wrappers: API calls, file I/O, database queries",
    examples: ["http-get", "file-read", "db-query", "shell-exec"],
  },
];

export const SkillsPage = () => (
  <>
    <PageSection hasBodyWrapper={false}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
        <div>
          <Title headingLevel="h1" size="2xl">Skills</Title>
          <Content component="p" style={{ marginTop: 4 }}>
            3-tier skill catalog with automated evaluation and quality badges.
          </Content>
        </div>
        <Button variant="primary" icon={<PlusCircleIcon />}>
          Register Skill
        </Button>
      </div>
    </PageSection>
    <Divider />
    <PageSection hasBodyWrapper={false}>
      <div className="arcana-empty-state" style={{ paddingBottom: 32 }}>
        <div className="arcana-empty-icon">
          <CubesIcon />
        </div>
        <Title headingLevel="h2" size="xl" style={{ marginBottom: 8 }}>
          Skill registry is empty
        </Title>
        <Content component="p" style={{ maxWidth: 480, margin: "0 auto", color: "var(--pf-t--global--text--color--subtle)" }}>
          Register your first skill to build your agent&apos;s capabilities.
          Skills are auto-evaluated and assigned Gold, Silver, or Bronze badges.
        </Content>
      </div>

      <div className="section-title">Skill Tier Architecture</div>
      <Grid hasGutter>
        {SKILL_TIERS.map((t) => (
          <GridItem span={4} key={t.tier}>
            <Card className="stat-card" isFullHeight>
              <CardBody>
                <div style={{ display: "flex", alignItems: "center", gap: 10, marginBottom: 12 }}>
                  <div style={{
                    width: 12, height: 12, borderRadius: 4,
                    background: t.color,
                  }} />
                  <span style={{ fontWeight: 700, fontSize: 16 }}>{t.tier}</span>
                </div>
                <Content component="p" style={{ fontSize: 13, marginBottom: 14, color: "var(--pf-t--global--text--color--subtle)" }}>
                  {t.desc}
                </Content>
                <div style={{ display: "flex", gap: 6, flexWrap: "wrap" }}>
                  {t.examples.map((e) => (
                    <Label color="grey" isCompact key={e}>{e}</Label>
                  ))}
                </div>
              </CardBody>
            </Card>
          </GridItem>
        ))}
      </Grid>
    </PageSection>
  </>
);
