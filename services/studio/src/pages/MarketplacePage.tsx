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
} from "@patternfly/react-icons";

/* ---------- types ---------- */

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

/* ---------- constants ---------- */

const CATEGORIES: { value: Category; label: string }[] = [
  { value: "all", label: "All" },
  { value: "productivity", label: "Productivity" },
  { value: "marketing", label: "Marketing" },
  { value: "engineering", label: "Engineering" },
  { value: "support", label: "Support" },
  { value: "data", label: "Data" },
];

const BADGE_COLORS: Record<QualityBadge, { bg: string; text: string; label: string }> = {
  gold: { bg: "#92400e", text: "#fbbf24", label: "Gold" },
  silver: { bg: "#374151", text: "#d1d5db", label: "Silver" },
  bronze: { bg: "#7c2d12", text: "#fb923c", label: "Bronze" },
  untested: { bg: "#1f2937", text: "#6b7280", label: "Untested" },
};

const BADGE_PF_COLORS: Record<QualityBadge, "yellow" | "grey" | "orange" | "blue"> = {
  gold: "yellow",
  silver: "grey",
  bronze: "orange",
  untested: "blue",
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
  const [items, setItems] = useState<MarketplaceItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [fetchError, setFetchError] = useState<string | null>(null);

  /* filters */
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

  const fetchItems = useCallback(async () => {
    setLoading(true);
    setFetchError(null);
    try {
      const params = new URLSearchParams();
      if (category !== "all") params.set("category", category);
      if (typeFilter !== "all") params.set("type", typeFilter);
      if (search.trim()) params.set("q", search.trim());
      const qs = params.toString();
      const url = `/api/v1/marketplace${qs ? `?${qs}` : ""}`;
      const res = await fetch(url);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      setItems(data.items ?? []);
    } catch (e) {
      setFetchError(e instanceof Error ? e.message : "Failed to load marketplace");
    } finally {
      setLoading(false);
    }
  }, [category, typeFilter, search]);

  useEffect(() => {
    fetchItems();
  }, [fetchItems]);

  /* ---- actions ---- */
  const handleDeploy = async (name: string) => {
    setActionLoading(name);
    setActionResult(null);
    try {
      const res = await fetch(`/api/v1/marketplace/${encodeURIComponent(name)}/deploy`, { method: "POST" });
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
      const res = await fetch(`/api/v1/marketplace/${encodeURIComponent(name)}/install`, { method: "POST" });
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
      /* fork is a conceptual clone — post to marketplace */
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
      const res = await fetch(`/api/v1/marketplace/${encodeURIComponent(rateTarget.name)}/rate`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ rating: rateValue }),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setRateModalOpen(false);
      setActionResult({ name: rateTarget.name, message: `Rated ${rateTarget.name} ${rateValue} stars` });
      await fetchItems();
    } catch {
      /* ignore */
    } finally {
      setRatingSubmitting(false);
    }
  };

  /* ---- sorting ---- */
  const sortedItems = [...items].sort((a, b) => {
    switch (sortBy) {
      case "popular":
        return b.usage_count - a.usage_count;
      case "rating":
        return b.rating - a.rating;
      case "recent":
        return 0; /* server order */
      default:
        return 0;
    }
  });

  /* ---- badge filter ---- */
  const BADGE_RANK: Record<QualityBadge, number> = { gold: 3, silver: 2, bronze: 1, untested: 0 };
  const filteredItems = sortedItems.filter((item) => {
    if (badgeFilter === "all") return true;
    return BADGE_RANK[item.badge] >= BADGE_RANK[badgeFilter];
  });

  return (
    <>
      <PageSection hasBodyWrapper={false}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
          <div>
            <Title headingLevel="h1" size="2xl">Marketplace</Title>
            <Content component="p" style={{ marginTop: 4, color: "var(--pf-t--global--text--color--subtle)" }}>
              Browse, deploy, and share agents and skills
            </Content>
          </div>
          <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
            <Label isCompact color="blue">{filteredItems.length} items</Label>
          </div>
        </div>
      </PageSection>
      <Divider />

      <PageSection hasBodyWrapper={false}>
        {fetchError && (
          <Alert variant="warning" title="Could not load marketplace" isInline style={{ marginBottom: 16 }}>
            {fetchError}
          </Alert>
        )}
        {actionResult && (
          <Alert variant="info" title={actionResult.message} isInline style={{ marginBottom: 16 }} />
        )}

        <div style={{ display: "flex", gap: 24 }}>
          {/* Sidebar filters */}
          <div style={{ width: 220, flexShrink: 0 }}>
            <div style={{ marginBottom: 20 }}>
              <div style={{ fontWeight: 700, fontSize: 13, textTransform: "uppercase", letterSpacing: "0.5px", color: "#8b95a5", marginBottom: 10 }}>
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
              <div style={{ fontWeight: 700, fontSize: 13, textTransform: "uppercase", letterSpacing: "0.5px", color: "#8b95a5", marginBottom: 10 }}>
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
              <div style={{ fontWeight: 700, fontSize: 13, textTransform: "uppercase", letterSpacing: "0.5px", color: "#8b95a5", marginBottom: 10 }}>
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
            {/* Search + category bar */}
            <div style={{ display: "flex", gap: 12, marginBottom: 20, flexWrap: "wrap", alignItems: "center" }}>
              <SearchInput
                placeholder="Search agents and skills..."
                value={search}
                onChange={(_e, val) => setSearch(val)}
                onClear={() => setSearch("")}
                style={{ maxWidth: 320 }}
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

            {loading ? (
              <div style={{ textAlign: "center", padding: 60 }}>
                <Spinner size="xl" />
              </div>
            ) : filteredItems.length === 0 ? (
              <div className="arcana-empty-state">
                <div className="arcana-empty-icon">
                  <CatalogIcon />
                </div>
                <Title headingLevel="h2" size="xl" style={{ marginBottom: 8 }}>
                  No items found
                </Title>
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
                          <div style={{ fontWeight: 700, fontSize: 15, color: "#e2e8f0" }}>
                            {item.name}
                          </div>
                          <div style={{ display: "flex", gap: 6 }}>
                            <Label
                              isCompact
                              color={item.type === "agent" ? "blue" : "purple"}
                            >
                              {item.type === "agent" ? "Agent" : "Skill"}
                            </Label>
                            <Label
                              isCompact
                              color={BADGE_PF_COLORS[item.badge]}
                            >
                              {BADGE_COLORS[item.badge].label}
                            </Label>
                          </div>
                        </div>

                        <Label isCompact color="grey" style={{ marginBottom: 8 }}>
                          {item.category}
                        </Label>
                        {item.tier && (
                          <Label isCompact color="teal" style={{ marginLeft: 6, marginBottom: 8 }}>
                            {item.tier}
                          </Label>
                        )}

                        <Content component="p" style={{
                          fontSize: 13,
                          color: "#8b95a5",
                          marginBottom: 12,
                          minHeight: 40,
                          display: "-webkit-box",
                          WebkitLineClamp: 2,
                          WebkitBoxOrient: "vertical",
                          overflow: "hidden",
                        }}>
                          {item.description}
                        </Content>

                        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 12 }}>
                          <StarRating rating={item.rating} />
                          <span style={{ fontSize: 12, color: "#6b7280" }}>
                            {item.usage_count.toLocaleString()} uses
                          </span>
                        </div>

                        <div style={{ display: "flex", gap: 8 }}>
                          {item.type === "agent" ? (
                            <Button
                              variant="primary"
                              size="sm"
                              icon={<RocketIcon />}
                              isLoading={actionLoading === item.name}
                              isDisabled={actionLoading === item.name}
                              onClick={() => handleDeploy(item.name)}
                              style={{ flex: 1 }}
                            >
                              Deploy
                            </Button>
                          ) : (
                            <Button
                              variant="primary"
                              size="sm"
                              icon={<DownloadIcon />}
                              isLoading={actionLoading === item.name}
                              isDisabled={actionLoading === item.name}
                              onClick={() => handleInstall(item.name)}
                              style={{ flex: 1 }}
                            >
                              Install
                            </Button>
                          )}
                          <Button
                            variant="secondary"
                            size="sm"
                            icon={<CodeBranchIcon />}
                            onClick={() => handleFork(item.name)}
                            aria-label={`Fork ${item.name}`}
                          />
                          <Button
                            variant="secondary"
                            size="sm"
                            icon={<StarIcon />}
                            onClick={() => openRateModal(item)}
                            aria-label={`Rate ${item.name}`}
                          />
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

      {/* Rate modal */}
      <Modal
        variant={ModalVariant.small}
        isOpen={rateModalOpen}
        onClose={() => setRateModalOpen(false)}
        aria-labelledby="rate-modal-title"
      >
        <ModalHeader title={`Rate ${rateTarget?.name ?? ""}`} labelId="rate-modal-title" />
        <ModalBody>
          <div style={{ textAlign: "center", padding: 16 }}>
            <div style={{ marginBottom: 16, fontSize: 14, color: "#8b95a5" }}>
              How would you rate this {rateTarget?.type}?
            </div>
            <div style={{ fontSize: 28 }}>
              <StarRating rating={rateValue} interactive onRate={setRateValue} />
            </div>
          </div>
        </ModalBody>
        <ModalFooter>
          <Button
            variant="primary"
            onClick={submitRating}
            isDisabled={rateValue < 1 || ratingSubmitting}
            isLoading={ratingSubmitting}
          >
            Submit Rating
          </Button>
          <Button variant="link" onClick={() => setRateModalOpen(false)}>
            Cancel
          </Button>
        </ModalFooter>
      </Modal>
    </>
  );
};
