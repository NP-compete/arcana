import { useCallback, useEffect, useState } from "react";
import {
  PageSection,
  Title,
  Content,
  Card,
  CardBody,
  CardTitle,
  Label,
  Spinner,
  Alert,
  AlertActionCloseButton,
  ExpandableSection,
  Grid,
  GridItem,
  DescriptionList,
  DescriptionListGroup,
  DescriptionListTerm,
  DescriptionListDescription,
  SearchInput,
} from "@patternfly/react-core";
import { ShareBadge } from "../components/ShareBadge";

interface ToolSchema {
  name: string;
  server: string;
  description: string;
  input_schema: Record<string, unknown>;
}

interface ToolServer {
  name: string;
  tools: ToolSchema[];
  protocol_version: string;
}

const SERVER_ICONS: Record<string, string> = {
  filesystem: "\uD83D\uDCC2",
  web: "\uD83C\uDF10",
  database: "\uD83D\uDDC4\uFE0F",
  mailchimp: "\uD83D\uDCE8",
  gdrive: "\uD83D\uDCC1",
  github: "\uD83D\uDC19",
  gitlab: "\uD83E\uDD8A",
  slack: "\uD83D\uDCAC",
  snowflake: "\u2744\uFE0F",
  codex: "\uD83D\uDD0D",
};

export const McpServersPage = () => {
  const [servers, setServers] = useState<ToolServer[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [expandedServers, setExpandedServers] = useState<Set<string>>(new Set());
  const [filter, setFilter] = useState("");

  const fetchServers = useCallback(async () => {
    try {
      const res = await fetch("/api/v1/tools");
      const data = await res.json();
      setServers(data.servers || []);
    } catch {
      setError("Failed to load MCP servers");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchServers();
  }, [fetchServers]);

  const toggleExpand = (name: string) => {
    setExpandedServers((prev) => {
      const next = new Set(prev);
      if (next.has(name)) next.delete(name);
      else next.add(name);
      return next;
    });
  };

  const totalTools = servers.reduce((sum, s) => sum + s.tools.length, 0);

  const filteredServers = servers.filter(
    (s) =>
      !filter ||
      s.name.toLowerCase().includes(filter.toLowerCase()) ||
      s.tools.some((t) => t.name.toLowerCase().includes(filter.toLowerCase()) || t.description.toLowerCase().includes(filter.toLowerCase())),
  );

  if (loading) {
    return (
      <PageSection hasBodyWrapper={false}>
        <div style={{ textAlign: "center", padding: 60 }}><Spinner size="xl" /></div>
      </PageSection>
    );
  }

  return (
    <>
      <PageSection hasBodyWrapper={false}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start" }}>
          <div>
            <Title headingLevel="h1" size="2xl">MCP Servers</Title>
            <Content component="p" style={{ marginTop: 4, color: "var(--pf-t--global--text--color--subtle)" }}>
              Model Context Protocol servers provide tools to your agents
            </Content>
          </div>
          <SearchInput
            placeholder="Filter servers or tools..."
            value={filter}
            onChange={(_e, val) => setFilter(val)}
            onClear={() => setFilter("")}
            style={{ maxWidth: 280 }}
          />
        </div>
      </PageSection>

      {error && (
        <PageSection hasBodyWrapper={false} style={{ paddingTop: 0 }}>
          <Alert variant="danger" title={error} isInline actionClose={<AlertActionCloseButton onClose={() => setError(null)} />} />
        </PageSection>
      )}

      <PageSection hasBodyWrapper={false} style={{ paddingTop: 0 }}>
        <Grid hasGutter>
          <GridItem span={3}>
            <Card className="stat-card">
              <CardBody>
                <div style={{ fontSize: 28, fontWeight: 800 }}>{servers.length}</div>
                <div style={{ fontSize: 13, color: "var(--pf-t--global--text--color--subtle)" }}>MCP Servers</div>
              </CardBody>
            </Card>
          </GridItem>
          <GridItem span={3}>
            <Card className="stat-card">
              <CardBody>
                <div style={{ fontSize: 28, fontWeight: 800 }}>{totalTools}</div>
                <div style={{ fontSize: 13, color: "var(--pf-t--global--text--color--subtle)" }}>Total Tools</div>
              </CardBody>
            </Card>
          </GridItem>
          <GridItem span={3}>
            <Card className="stat-card">
              <CardBody>
                <div style={{ fontSize: 28, fontWeight: 800 }}>MCP</div>
                <div style={{ fontSize: 13, color: "var(--pf-t--global--text--color--subtle)" }}>Protocol</div>
              </CardBody>
            </Card>
          </GridItem>
          <GridItem span={3}>
            <Card className="stat-card">
              <CardBody>
                <div style={{ fontSize: 28, fontWeight: 800 }}>2024-11-05</div>
                <div style={{ fontSize: 13, color: "var(--pf-t--global--text--color--subtle)" }}>Protocol Version</div>
              </CardBody>
            </Card>
          </GridItem>
        </Grid>
      </PageSection>

      <PageSection hasBodyWrapper={false} style={{ paddingTop: 0 }}>
        {filteredServers.map((server) => (
          <Card key={server.name} style={{ marginBottom: 16 }}>
            <CardBody>
              <ExpandableSection
                toggleText={
                  `${SERVER_ICONS[server.name] || "\uD83D\uDD0C"} ${server.name} — ${server.tools.length} tool${server.tools.length !== 1 ? "s" : ""}`
                }
                isExpanded={expandedServers.has(server.name)}
                onToggle={() => toggleExpand(server.name)}
              >
                <div style={{ display: "flex", gap: 8, marginBottom: 16 }}>
                  <Label isCompact color="blue">MCP {server.protocol_version}</Label>
                  <Label isCompact color="green">{server.tools.length} tools</Label>
                  <ShareBadge assetType="mcp" assetName={server.name} compact />
                </div>
                {server.tools.map((tool) => (
                  <Card key={tool.name} variant="secondary" style={{ marginBottom: 12 }}>
                    <CardTitle>
                      <span style={{ fontFamily: "monospace", fontSize: 14, fontWeight: 700 }}>
                        {tool.name}
                      </span>
                    </CardTitle>
                    <CardBody>
                      <div style={{ fontSize: 14, marginBottom: 12 }}>{tool.description}</div>
                      <DescriptionList isHorizontal isCompact>
                        <DescriptionListGroup>
                          <DescriptionListTerm>Server</DescriptionListTerm>
                          <DescriptionListDescription>
                            <Label isCompact color="blue">{tool.server}</Label>
                          </DescriptionListDescription>
                        </DescriptionListGroup>
                        {tool.input_schema && (
                          <DescriptionListGroup>
                            <DescriptionListTerm>Parameters</DescriptionListTerm>
                            <DescriptionListDescription>
                              {Object.keys(
                                (tool.input_schema as Record<string, unknown>)?.properties as Record<string, unknown> || {},
                              ).map((param) => (
                                <Label key={param} isCompact color="grey" style={{ marginRight: 4 }}>
                                  {param}
                                </Label>
                              ))}
                            </DescriptionListDescription>
                          </DescriptionListGroup>
                        )}
                      </DescriptionList>
                    </CardBody>
                  </Card>
                ))}
              </ExpandableSection>
            </CardBody>
          </Card>
        ))}
      </PageSection>
    </>
  );
};
