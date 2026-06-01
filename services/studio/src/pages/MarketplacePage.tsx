import { useState, useEffect, useCallback } from "react";
import {
  PageSection,
  Title,
  Content,
  Card,
  CardBody,
  Button,
  Label,
  SearchInput,
  Spinner,
  Alert,
  Divider,
  Modal,
  ModalHeader,
  ModalBody,
  ModalFooter,
  ModalVariant,
  Grid,
  GridItem,
  ToggleGroup,
  ToggleGroupItem,
} from "@patternfly/react-core";
import {
  CatalogIcon,
  StarIcon,
  OutlinedStarIcon,
  RocketIcon,
  DownloadIcon,
  CodeBranchIcon,
  RobotIcon,
  CubesIcon,
} from "@patternfly/react-icons";
import { useNavigate } from "react-router-dom";

/* ---------- types ---------- */

type Tab = "yours" | "community";
type ItemType = "agent" | "skill";
type QualityBadge = "gold" | "silver" | "bronze" | "untested";
type Category = "all" | "productivity" | "marketing" | "engineering" | "support" | "data";
type SortOption = "popular" | "recent" | "rating";

interface MarketplaceItem {
  name: string;
  type: ItemType;
  category: string;
  description: string;
  badge: QualityBadge;
  usage_count: number;
  rating: number;
  tier?: string;
}

interface OwnedAgent {
  name: string;
  status: string;
  capabilities: string[];
}

interface OwnedSkill {
  name: string;
  tier?: string;
  description?: string;
}

/* ---------- constants ---------- */

const CATEGORIES: { value: Category; label: string }[] = [
  { value: "all", label: "All" },
  { value: "productivity", label: "Productivity" },
  { value: "marketing", label: "Marketing" },
  { value: "engineering", label: "Engineering" },
  { value: "support", label: "Support" },
  { value: "data", label: "Data" },
];

const BADGE_COLORS: Record<QualityBadge, { label: string }> = {
  gold: { label: "Gold" },
  silver: { label: "Silver" },
  bronze: { label: "Bronze" },
  untested: { label: "Untested" },
};

const BADGE_PF_COLORS: Record<QualityBadge, "yellow" | "grey" | "orange" | "blue"> = {
  gold: "yellow",
  silver: "grey",
  bronze: "orange",
  untested: "blue",
};

const STATUS_DOT: Record<string, { color: string; label: string }> = {
  active: { color: "#22c55e", label: "Running" },
  busy: { color: "#f59e0b", label: "Busy" },
  idle: { color: "#8b95a5", label: "Sleeping" },
  offline: { color: "#ef4444", label: "Crashed" },
};

/* ---------- star rating ---------- */

function StarRating({
  rating,
  interactive = false,
  onRate,
}: {
  rating: number;
  interactive?: boolean;
  onRate?: (stars: number) => void;
}) {
  const stars = [];
  for (let i = 1; i <= 5; i++) {
    const filled = i <= Math.round(rating);
    stars.push(
      <span
        key={i}
        style={{
          cursor: interactive ? "pointer" : "default",
          color: filled ? "#fbbf24" : "#4a5568",
          fontSize: 14,
        }}
        onClick={interactive && onRate ? () => onRate(i) : undefined}
        role={interactive ? "button" : undefined}
        aria-label={interactive ? `Rate ${i} stars` : undefined}
      >
        {filled ? <StarIcon /> : <OutlinedStarIcon />}
      </span>,
    );
  }
  return <span style={{ display: "inline-flex", gap: 2 }}>{stars}</span>;
}

/* ---------- main component ---------- */

export const MarketplacePage = () => {
  const navigate = useNavigate();
  const [tab, setTab] = useState<Tab>("yours");

  /* ---- Yours tab state ---- */
  const [ownedAgents, setOwnedAgents] = useState<OwnedAgent[]>([]);
  const [ownedSkills, setOwnedSkills] = useState<OwnedSkill[]>([]);
  const [yoursLoading, setYoursLoading] = useState(true);
  const [yoursError, setYoursError] = useState<string | null>(null);
  const [yoursSearch, setYoursSearch] = useState("");
  const [yoursFilter, setYoursFilter] = useState<"all" | "agents" | "skills">("all");

  /* ---- Community tab state ---- */
  const [items, setItems] = useState<MarketplaceItem[]>([]);
  const [communityLoading, setCommunityLoading] = useState(false);
  const [fetchError, setFetchError] = useState<string | null>(null);
  const [search, setSearch] = useState("");
  const [category, setCategory] = useState<Category>("all");
  const [typeFilter, setTypeFilter] = useState<"all" | ItemType>("all");
  const [badgeFilter, setBadgeFilter] = useState<"all" | QualityBadge>("all");
  const [sortBy, setSortBy] = useState<SortOption>("popular");

  /* action state */
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [actionResult, setActionResult] = useState<{ name: string; message: string } | null>(null);

  /* rate modal */
  const [rateModalOpen, setRateModalOpen] = useState(false);
  const [rateTarget, setRateTarget] = useState<MarketplaceItem | null>(null);
  const [rateValue, setRateValue] = useState(0);
  const [ratingSubmitting, setRatingSubmitting] = useState(false);

  /* ---- fetch yours ---- */
  const fetchOwned = useCallback(async () => {
    setYoursLoading(true);
    setYoursError(null);
    try {
      const [agentsRes, skillsRes] = await Promise.allSettled([
        fetch("/api/v1/agents"),
        fetch("/api/v1/skills"),
      ]);
      if (agentsRes.status === "fulfilled" && agentsRes.value.ok) {
        const data = await agentsRes.value.json();
        setOwnedAgents((data.agents ?? []).map((a: Record<string, unknown>) => ({
          name: a.name as string,
          status: a.status as string,
          capabilities: (a.capabilities ?? []) as string[],
        })));
      }
      if (skillsRes.status === "fulfilled" && skillsRes.value.ok) {
        const data = await skillsRes.value.json();
        setOwnedSkills((data.skills ?? []).map((s: Record<string, unknown>) => ({
          name: s.name as string,
          tier: s.tier as string | undefined,
          description: s.description as string | undefined,
        })));
      }
    } catch (e) {
      setYoursError(e instanceof Error ? e.message : "Failed to load");
    } finally {
      setYoursLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchOwned();
  }, [fetchOwned]);

  /* ---- fetch community ---- */
  const fetchCommunity = useCallback(async () => {
    setCommunityLoading(true);
    setFetchError(null);
    try {
      const params = new URLSearchParams();
      if (category !== "all") params.set("category", category);
      if (typeFilter !== "all") params.set("type", typeFilter);
      if (search.trim()) params.set("q", search.trim());
      const qs = params.toString();
      const res = await fetch(`/api/v1/catalog${qs ? `?${qs}` : ""}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      setItems(data.items ?? []);
    } catch (e) {
      setFetchError(e instanceof Error ? e.message : "Failed to load");
    } finally {
      setCommunityLoading(false);
    }
  }, [category, typeFilter, search]);

  useEffect(() => {
    if (tab === "community") fetchCommunity();
  }, [tab, fetchCommunity]);

  /* ---- actions ---- */
  const handleDeploy = async (name: string) => {
    setActionLoading(name);
    setActionResult(null);
    try {
      const res = await fetch(`/api/v1/catalog/${encodeURIComponent(name)}/deploy`, { method: "POST" });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error((data as Record<string, string>).error ?? `HTTP ${res.status}`);
      }
      setActionResult({ name, message: `${name} deployed successfully` });
    } catch (e) {
      setActionResult({ name, message: e instanceof Error ? e.message : "Deploy failed" });
    } finally {
      setActionLoading(null);
    }
  };

  const handleInstall = async (name: string) => {
    setActionLoading(name);
    setActionResult(null);
    try {
      const res = await fetch(`/api/v1/catalog/${encodeURIComponent(name)}/install`, { method: "POST" });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error((data as Record<string, string>).error ?? `HTTP ${res.status}`);
      }
      setActionResult({ name, message: `${name} installed successfully` });
    } catch (e) {
      setActionResult({ name, message: e instanceof Error ? e.message : "Install failed" });
    } finally {
      setActionLoading(null);
    }
  };

  const handleFork = async (name: string) => {
    setActionLoading(name);
    try {
      setActionResult({ name, message: `${name} forked to your workspace` });
    } finally {
      setActionLoading(null);
    }
  };

  const openRateModal = (item: MarketplaceItem) => {
    setRateTarget(item);
    setRateValue(Math.round(item.rating));
    setRateModalOpen(true);
  };

  const submitRating = async () => {
    if (!rateTarget || rateValue < 1) return;
    setRatingSubmitting(true);
    try {
      const res = await fetch(`/api/v1/catalog/${encodeURIComponent(rateTarget.name)}/rate`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ rating: rateValue }),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setRateModalOpen(false);
      setActionResult({ name: rateTarget.name, message: `Rated ${rateTarget.name} ${rateValue} stars` });
      await fetchCommunity();
    } catch {
      /* ignore */
    } finally {
      setRatingSubmitting(false);
    }
  };

  /* ---- community sorting + filtering ---- */
  const sortedItems = [...items].sort((a, b) => {
    switch (sortBy) {
      case "popular": return b.usage_count - a.usage_count;
      case "rating": return b.rating - a.rating;
      default: return 0;
    }
  });

  const BADGE_RANK: Record<QualityBadge, number> = { gold: 3, silver: 2, bronze: 1, untested: 0 };
  const filteredItems = sortedItems.filter((item) => {
    if (badgeFilter === "all") return true;
    return BADGE_RANK[item.badge] >= BADGE_RANK[badgeFilter];
  });

  /* ---- yours filtering ---- */
  const yoursItems: Array<{ kind: "agent"; data: OwnedAgent } | { kind: "skill"; data: OwnedSkill }> = [
    ...(yoursFilter === "skills" ? [] : ownedAgents.map((a) => ({ kind: "agent" as const, data: a }))),
    ...(yoursFilter === "agents" ? [] : ownedSkills.map((s) => ({ kind: "skill" as const, data: s }))),
  ];
  const filteredYours = yoursSearch.trim()
    ? yoursItems.filter((item) => item.data.name.toLowerCase().includes(yoursSearch.toLowerCase()))
    : yoursItems;

  return (
    <>
      <PageSection hasBodyWrapper={false}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
          <div>
            <Title headingLevel="h1" size="2xl">Marketplace</Title>
            <Content component="p" style={{ marginTop: 4, color: "var(--pf-t--global--text--color--subtle)" }}>
              Your registry and the community catalog in one place
            </Content>
          </div>
        </div>

        {/* Yours / Community tabs */}
        <div style={{ marginTop: 20 }}>
          <div style={{ display: "flex", gap: 0, borderBottom: "1px solid var(--arcana-card-border)" }}>
            {([
              { key: "yours", label: `Yours (${ownedAgents.length + ownedSkills.length})` },
              { key: "community", label: "Community" },
            ] as const).map((t) => (
              <button
                key={t.key}
                type="button"
                onClick={() => setTab(t.key)}
                style={{
                  padding: "10px 24px",
                  fontSize: 14,
                  fontWeight: 600,
                  color: tab === t.key ? "var(--arcana-text)" : "var(--arcana-text-muted)",
                  background: "none",
                  border: "none",
                  borderBottom: tab === t.key ? "2px solid #5b8def" : "2px solid transparent",
                  cursor: "pointer",
                  transition: "all 0.15s",
                  marginBottom: -1,
                }}
              >
                {t.label}
              </button>
            ))}
          </div>
        </div>
      </PageSection>

      {/* ===== Yours tab ===== */}
      {tab === "yours" && (
        <PageSection hasBodyWrapper={false}>
          {yoursError && (
            <Alert variant="warning" title="Could not load registry" isInline style={{ marginBottom: 16 }}>
              {yoursError}
            </Alert>
          )}

          <div style={{ display: "flex", gap: 16, marginBottom: 20, alignItems: "center" }}>
            <SearchInput
              placeholder="Search your agents and skills..."
              value={yoursSearch}
              onChange={(_e, val) => setYoursSearch(val)}
              onClear={() => setYoursSearch("")}
              style={{ maxWidth: 280 }}
            />
            <ToggleGroup aria-label="Type filter">
              {([
                { value: "all", label: `All (${ownedAgents.length + ownedSkills.length})` },
                { value: "agents", label: `Agents (${ownedAgents.length})` },
                { value: "skills", label: `Skills (${ownedSkills.length})` },
              ] as const).map((opt) => (
                <ToggleGroupItem
                  key={opt.value}
                  text={opt.label}
                  buttonId={`yours-${opt.value}`}
                  isSelected={yoursFilter === opt.value}
                  onChange={() => setYoursFilter(opt.value)}
                />
              ))}
            </ToggleGroup>
          </div>

          {yoursLoading ? (
            <div style={{ textAlign: "center", padding: 60 }}>
              <Spinner size="xl" />
            </div>
          ) : filteredYours.length === 0 ? (
            <div style={{
              textAlign: "center",
              padding: "60px 40px",
              color: "var(--arcana-text-secondary)",
            }}>
              {yoursSearch.trim()
                ? `No results for "${yoursSearch}"`
                : "Nothing registered yet. Deploy an agent or add a skill to get started."}
            </div>
          ) : (
            <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
              {filteredYours.map((item) => {
                if (item.kind === "agent") {
                  const agent = item.data;
                  const dot = STATUS_DOT[agent.status] ?? STATUS_DOT.offline;
                  const skills = agent.capabilities.filter((c) => !c.startsWith("model:"));
                  const model = agent.capabilities.find((c) => c.startsWith("model:"))?.replace("model:", "");
                  return (
                    <div
                      key={`agent-${agent.name}`}
                      onClick={() => navigate(`/agents/${agent.name}`)}
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
                        width: 8, height: 8, borderRadius: "50%",
                        background: dot.color, flexShrink: 0,
                      }} />
                      <div style={{ flex: 1, minWidth: 0 }}>
                        <div style={{ fontSize: 14, fontWeight: 600, color: "var(--arcana-text)" }}>
                          {agent.name}
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

                const skill = item.data;
                return (
                  <div
                    key={`skill-${skill.name}`}
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
                        {skill.name}
                      </div>
                      <div style={{ fontSize: 12, color: "var(--arcana-text-muted)" }}>
                        Skill{skill.tier ? ` · ${skill.tier}` : ""}
                        {skill.description ? ` · ${skill.description}` : ""}
                      </div>
                    </div>
                    {skill.tier && <Label isCompact color="purple">{skill.tier}</Label>}
                  </div>
                );
              })}
            </div>
          )}
        </PageSection>
      )}

      {/* ===== Community tab ===== */}
      {tab === "community" && (
        <PageSection hasBodyWrapper={false}>
          {fetchError && (
            <Alert variant="warning" title="Could not load community catalog" isInline style={{ marginBottom: 16 }}>
              {fetchError}
            </Alert>
          )}
          {actionResult && (
            <Alert variant="info" title={actionResult.message} isInline style={{ marginBottom: 16 }} />
          )}

          <div style={{ display: "flex", gap: 24 }}>
            {/* Sidebar filters */}
            <div style={{ width: 200, flexShrink: 0 }}>
              <div style={{ marginBottom: 20 }}>
                <div style={{ fontWeight: 700, fontSize: 12, textTransform: "uppercase", letterSpacing: "0.5px", color: "var(--arcana-text-muted)", marginBottom: 10 }}>
                  Type
                </div>
                <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
                  {(["all", "agent", "skill"] as const).map((t) => (
                    <Button
                      key={t}
                      variant={typeFilter === t ? "primary" : "tertiary"}
                      size="sm"
                      isBlock
                      onClick={() => setTypeFilter(t)}
                      style={{ textAlign: "left", justifyContent: "flex-start" }}
                    >
                      {t === "all" ? "All Types" : t.charAt(0).toUpperCase() + t.slice(1) + "s"}
                    </Button>
                  ))}
                </div>
              </div>

              <div style={{ marginBottom: 20 }}>
                <div style={{ fontWeight: 700, fontSize: 12, textTransform: "uppercase", letterSpacing: "0.5px", color: "var(--arcana-text-muted)", marginBottom: 10 }}>
                  Badge
                </div>
                <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
                  {(["all", "gold", "silver", "bronze"] as const).map((b) => (
                    <Button
                      key={b}
                      variant={badgeFilter === b ? "primary" : "tertiary"}
                      size="sm"
                      isBlock
                      onClick={() => setBadgeFilter(b)}
                      style={{ textAlign: "left", justifyContent: "flex-start" }}
                    >
                      {b === "all" ? "Any Badge" : `${b.charAt(0).toUpperCase() + b.slice(1)}+`}
                    </Button>
                  ))}
                </div>
              </div>

              <div>
                <div style={{ fontWeight: 700, fontSize: 12, textTransform: "uppercase", letterSpacing: "0.5px", color: "var(--arcana-text-muted)", marginBottom: 10 }}>
                  Sort By
                </div>
                <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
                  {(["popular", "recent", "rating"] as const).map((s) => (
                    <Button
                      key={s}
                      variant={sortBy === s ? "primary" : "tertiary"}
                      size="sm"
                      isBlock
                      onClick={() => setSortBy(s)}
                      style={{ textAlign: "left", justifyContent: "flex-start" }}
                    >
                      {s.charAt(0).toUpperCase() + s.slice(1)}
                    </Button>
                  ))}
                </div>
              </div>
            </div>

            {/* Main content */}
            <div style={{ flex: 1 }}>
              <div style={{ display: "flex", gap: 12, marginBottom: 20, flexWrap: "wrap", alignItems: "center" }}>
                <SearchInput
                  placeholder="Search community..."
                  value={search}
                  onChange={(_e, val) => setSearch(val)}
                  onClear={() => setSearch("")}
                  style={{ maxWidth: 300 }}
                />
                <ToggleGroup aria-label="Category filter">
                  {CATEGORIES.map((cat) => (
                    <ToggleGroupItem
                      key={cat.value}
                      text={cat.label}
                      buttonId={`cat-${cat.value}`}
                      isSelected={category === cat.value}
                      onChange={() => setCategory(cat.value)}
                    />
                  ))}
                </ToggleGroup>
              </div>

              {communityLoading ? (
                <div style={{ textAlign: "center", padding: 60 }}>
                  <Spinner size="xl" />
                </div>
              ) : filteredItems.length === 0 ? (
                <div className="arcana-empty-state">
                  <div className="arcana-empty-icon"><CatalogIcon /></div>
                  <Title headingLevel="h2" size="xl" style={{ marginBottom: 8 }}>No items found</Title>
                  <Content component="p" style={{ maxWidth: 480, margin: "0 auto", color: "var(--pf-t--global--text--color--subtle)" }}>
                    Try adjusting your filters or search terms.
                  </Content>
                </div>
              ) : (
                <Grid hasGutter>
                  {filteredItems.map((item) => (
                    <GridItem span={4} key={`${item.type}-${item.name}`}>
                      <Card className="marketplace-card" isFullHeight>
                        <CardBody>
                          <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", marginBottom: 10 }}>
                            <div style={{ fontWeight: 700, fontSize: 15, color: "var(--arcana-text)" }}>
                              {item.name}
                            </div>
                            <div style={{ display: "flex", gap: 6 }}>
                              <Label isCompact color={item.type === "agent" ? "blue" : "purple"}>
                                {item.type === "agent" ? "Agent" : "Skill"}
                              </Label>
                              <Label isCompact color={BADGE_PF_COLORS[item.badge]}>
                                {BADGE_COLORS[item.badge].label}
                              </Label>
                            </div>
                          </div>

                          <Label isCompact color="grey" style={{ marginBottom: 8 }}>{item.category}</Label>
                          {item.tier && <Label isCompact color="teal" style={{ marginLeft: 6, marginBottom: 8 }}>{item.tier}</Label>}

                          <Content component="p" style={{
                            fontSize: 13, color: "var(--arcana-text-secondary)", marginBottom: 12, minHeight: 40,
                            display: "-webkit-box", WebkitLineClamp: 2, WebkitBoxOrient: "vertical", overflow: "hidden",
                          }}>
                            {item.description}
                          </Content>

                          <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 12 }}>
                            <StarRating rating={item.rating} />
                            <span style={{ fontSize: 12, color: "var(--arcana-text-muted)" }}>
                              {item.usage_count.toLocaleString()} uses
                            </span>
                          </div>

                          <div style={{ display: "flex", gap: 8 }}>
                            {item.type === "agent" ? (
                              <Button variant="primary" size="sm" icon={<RocketIcon />}
                                isLoading={actionLoading === item.name} isDisabled={actionLoading === item.name}
                                onClick={() => handleDeploy(item.name)} style={{ flex: 1 }}>
                                Deploy
                              </Button>
                            ) : (
                              <Button variant="primary" size="sm" icon={<DownloadIcon />}
                                isLoading={actionLoading === item.name} isDisabled={actionLoading === item.name}
                                onClick={() => handleInstall(item.name)} style={{ flex: 1 }}>
                                Install
                              </Button>
                            )}
                            <Button variant="secondary" size="sm" icon={<CodeBranchIcon />}
                              onClick={() => handleFork(item.name)} aria-label={`Fork ${item.name}`} />
                            <Button variant="secondary" size="sm" icon={<StarIcon />}
                              onClick={() => openRateModal(item)} aria-label={`Rate ${item.name}`} />
                          </div>
                        </CardBody>
                      </Card>
                    </GridItem>
                  ))}
                </Grid>
              )}
            </div>
          </div>
        </PageSection>
      )}

      {/* Rate modal */}
      <Modal variant={ModalVariant.small} isOpen={rateModalOpen} onClose={() => setRateModalOpen(false)} aria-labelledby="rate-modal-title">
        <ModalHeader title={`Rate ${rateTarget?.name ?? ""}`} labelId="rate-modal-title" />
        <ModalBody>
          <div style={{ textAlign: "center", padding: 16 }}>
            <div style={{ marginBottom: 16, fontSize: 14, color: "var(--arcana-text-secondary)" }}>
              How would you rate this {rateTarget?.type}?
            </div>
            <div style={{ fontSize: 28 }}>
              <StarRating rating={rateValue} interactive onRate={setRateValue} />
            </div>
          </div>
        </ModalBody>
        <ModalFooter>
          <Button variant="primary" onClick={submitRating} isDisabled={rateValue < 1 || ratingSubmitting} isLoading={ratingSubmitting}>
            Submit Rating
          </Button>
          <Button variant="link" onClick={() => setRateModalOpen(false)}>Cancel</Button>
        </ModalFooter>
      </Modal>
    </>
  );
};
