import { useState, useEffect, useCallback } from "react";
import {
  PageSection,
  Title,
  Content,
  Label,
  SearchInput,
  Spinner,
  Alert,
  Divider,
  ToggleGroup,
  ToggleGroupItem,
} from "@patternfly/react-core";
import {
  RobotIcon,
  CubesIcon,
} from "@patternfly/react-icons";
import { useNavigate } from "react-router-dom";

type RegistryFilter = "all" | "agents" | "skills";

interface RegistryAgent {
  kind: "agent";
  name: string;
  status: string;
  capabilities: string[];
  registered_at?: string;
}

interface RegistrySkill {
  kind: "skill";
  name: string;
  tier?: string;
  description?: string;
}

type RegistryItem = RegistryAgent | RegistrySkill;

const STATUS_DOT: Record<string, { color: string; label: string }> = {
  active: { color: "#22c55e", label: "Running" },
  busy: { color: "#f59e0b", label: "Busy" },
  idle: { color: "#8b95a5", label: "Sleeping" },
  offline: { color: "#ef4444", label: "Crashed" },
};

export const RegistryPage = () => {
  const navigate = useNavigate();
  const [agents, setAgents] = useState<RegistryAgent[]>([]);
  const [skills, setSkills] = useState<RegistrySkill[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [filter, setFilter] = useState<RegistryFilter>("all");
  const [search, setSearch] = useState("");

  const fetchAll = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [agentsRes, skillsRes] = await Promise.allSettled([
        fetch("/api/v1/agents"),
        fetch("/api/v1/skills"),
      ]);

      const agentList: RegistryAgent[] = [];
      if (agentsRes.status === "fulfilled" && agentsRes.value.ok) {
        const data = await agentsRes.value.json();
        for (const a of data.agents ?? []) {
          agentList.push({
            kind: "agent",
            name: a.name,
            status: a.status,
            capabilities: a.capabilities ?? [],
            registered_at: a.registered_at,
          });
        }
      }

      const skillList: RegistrySkill[] = [];
      if (skillsRes.status === "fulfilled" && skillsRes.value.ok) {
        const data = await skillsRes.value.json();
        for (const s of data.skills ?? []) {
          skillList.push({
            kind: "skill",
            name: s.name,
            tier: s.tier,
            description: s.description,
          });
        }
      }

      setAgents(agentList);
      setSkills(skillList);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load registry");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchAll();
  }, [fetchAll]);

  const allItems: RegistryItem[] = [
    ...(filter === "skills" ? [] : agents),
    ...(filter === "agents" ? [] : skills),
  ];

  const filtered = search.trim()
    ? allItems.filter((item) =>
        item.name.toLowerCase().includes(search.toLowerCase()),
      )
    : allItems;

  const agentCount = agents.length;
  const skillCount = skills.length;

  return (
    <>
      <PageSection hasBodyWrapper={false}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
          <div>
            <Title headingLevel="h1" size="2xl">Registry</Title>
            <Content component="p" style={{ marginTop: 4, color: "var(--pf-t--global--text--color--subtle)" }}>
              All agents and skills registered on the platform
            </Content>
          </div>
          <div style={{ display: "flex", gap: 8 }}>
            <Label isCompact color="blue">{agentCount} agent{agentCount !== 1 ? "s" : ""}</Label>
            <Label isCompact color="purple">{skillCount} skill{skillCount !== 1 ? "s" : ""}</Label>
          </div>
        </div>
      </PageSection>
      <Divider />
      <PageSection hasBodyWrapper={false}>
        {error && (
          <Alert variant="warning" title="Could not load registry" isInline style={{ marginBottom: 16 }}>
            {error}
          </Alert>
        )}

        {/* Filters */}
        <div style={{ display: "flex", gap: 16, marginBottom: 20, alignItems: "center" }}>
          <SearchInput
            placeholder="Search..."
            value={search}
            onChange={(_e, val) => setSearch(val)}
            onClear={() => setSearch("")}
            style={{ maxWidth: 280 }}
          />
          <ToggleGroup aria-label="Type filter">
            {([
              { value: "all", label: `All (${agentCount + skillCount})` },
              { value: "agents", label: `Agents (${agentCount})` },
              { value: "skills", label: `Skills (${skillCount})` },
            ] as const).map((opt) => (
              <ToggleGroupItem
                key={opt.value}
                text={opt.label}
                buttonId={`reg-${opt.value}`}
                isSelected={filter === opt.value}
                onChange={() => setFilter(opt.value)}
              />
            ))}
          </ToggleGroup>
        </div>

        {loading ? (
          <div style={{ textAlign: "center", padding: 60 }}>
            <Spinner size="xl" />
          </div>
        ) : filtered.length === 0 ? (
          <div style={{
            textAlign: "center",
            padding: "60px 40px",
            color: "var(--arcana-text-secondary)",
          }}>
            {search.trim()
              ? `No results for "${search}"`
              : "Nothing registered yet. Deploy an agent or add a skill to get started."}
          </div>
        ) : (
          <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
            {filtered.map((item) => {
              if (item.kind === "agent") {
                const dot = STATUS_DOT[item.status] ?? STATUS_DOT.offline;
                const skills = item.capabilities.filter((c) => !c.startsWith("model:"));
                const model = item.capabilities.find((c) => c.startsWith("model:"))?.replace("model:", "");
                return (
                  <div
                    key={`agent-${item.name}`}
                    onClick={() => navigate(`/agents/${item.name}`)}
                    style={{
                      display: "flex",
                      alignItems: "center",
                      gap: 14,
                      padding: "14px 18px",
                      background: "var(--arcana-card-bg)",
                      borderRadius: 10,
                      border: "1px solid var(--arcana-card-border)",
                      cursor: "pointer",
                      transition: "background 0.15s",
                    }}
                    onMouseEnter={(e) => { (e.currentTarget as HTMLDivElement).style.background = "var(--arcana-hover-bg)"; }}
                    onMouseLeave={(e) => { (e.currentTarget as HTMLDivElement).style.background = "var(--arcana-card-bg)"; }}
                  >
                    <RobotIcon style={{ color: "#5b8def", fontSize: 16, flexShrink: 0 }} />
                    <span style={{
                      width: 8,
                      height: 8,
                      borderRadius: "50%",
                      background: dot.color,
                      flexShrink: 0,
                    }} />
                    <div style={{ flex: 1, minWidth: 0 }}>
                      <div style={{ fontSize: 14, fontWeight: 600, color: "var(--arcana-text)" }}>
                        {item.name}
                      </div>
                      <div style={{ fontSize: 12, color: "var(--arcana-text-muted)" }}>
                        Agent &middot; {model ?? "—"} &middot; {dot.label}
                      </div>
                    </div>
                    <div style={{ display: "flex", gap: 4, flexWrap: "wrap" }}>
                      {skills.slice(0, 3).map((s) => (
                        <Label key={s} isCompact color="grey">{s}</Label>
                      ))}
                      {skills.length > 3 && <Label isCompact color="grey">+{skills.length - 3}</Label>}
                    </div>
                  </div>
                );
              }

              return (
                <div
                  key={`skill-${item.name}`}
                  style={{
                    display: "flex",
                    alignItems: "center",
                    gap: 14,
                    padding: "14px 18px",
                    background: "var(--arcana-card-bg)",
                    borderRadius: 10,
                    border: "1px solid var(--arcana-card-border)",
                    transition: "background 0.15s",
                  }}
                  onMouseEnter={(e) => { (e.currentTarget as HTMLDivElement).style.background = "var(--arcana-hover-bg)"; }}
                  onMouseLeave={(e) => { (e.currentTarget as HTMLDivElement).style.background = "var(--arcana-card-bg)"; }}
                >
                  <CubesIcon style={{ color: "#a855f7", fontSize: 16, flexShrink: 0 }} />
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ fontSize: 14, fontWeight: 600, color: "var(--arcana-text)" }}>
                      {item.name}
                    </div>
                    <div style={{ fontSize: 12, color: "var(--arcana-text-muted)" }}>
                      Skill{item.tier ? ` · ${item.tier}` : ""}
                      {item.description ? ` · ${item.description}` : ""}
                    </div>
                  </div>
                  {item.tier && <Label isCompact color="purple">{item.tier}</Label>}
                </div>
              );
            })}
          </div>
        )}
      </PageSection>
    </>
  );
};
