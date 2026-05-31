import { useCallback, useEffect, useState } from "react";
import {
  Card,
  CardBody,
  CardTitle,
  Label,
  Button,
  Tabs,
  Tab,
  TabTitleText,
  Alert,
  AlertActionCloseButton,
  TextInput,
  FormGroup,
  Spinner,
} from "@patternfly/react-core";
import { Table, Thead, Tr, Th, Tbody, Td } from "@patternfly/react-table";
import { MemoryIcon, SearchIcon, CompressIcon, PlusCircleIcon } from "@patternfly/react-icons";

interface ShortTermEntry {
  id: string;
  agent_id: string;
  key: string;
  value: unknown;
  ttl: number;
  created_at: string;
  expires_at: string;
}

interface LongTermMemory {
  id: string;
  agent_id: string;
  content: string;
  metadata: Record<string, unknown>;
  created_at: string;
}

interface SearchResult {
  memory: LongTermMemory;
  score: number;
}

interface AgentMemoryProps {
  agentName: string;
}

export const AgentMemory = ({ agentName }: AgentMemoryProps) => {
  const [activeTab, setActiveTab] = useState(0);
  const [shortTerm, setShortTerm] = useState<ShortTermEntry[]>([]);
  const [longTerm, setLongTerm] = useState<SearchResult[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const [searching, setSearching] = useState(false);
  const [compacting, setCompacting] = useState(false);
  const [storeKey, setStoreKey] = useState("");
  const [storeValue, setStoreValue] = useState("");

  const fetchShortTerm = useCallback(async () => {
    try {
      const res = await fetch(`/api/v1/memory/short-term/${agentName}`);
      const data = await res.json();
      setShortTerm(Array.isArray(data) ? data : []);
    } catch {
      setError("Failed to load short-term memory");
    }
  }, [agentName]);

  const searchLongTerm = useCallback(async (query: string) => {
    setSearching(true);
    try {
      const q = encodeURIComponent(query || "conversation");
      const res = await fetch(`/api/v1/memory/long-term/${agentName}/search?query=${q}&top_k=10`);
      const data = await res.json();
      setLongTerm(data.results || []);
    } catch {
      setError("Failed to search long-term memory");
    } finally {
      setSearching(false);
    }
  }, [agentName]);

  useEffect(() => {
    (async () => {
      setLoading(true);
      await fetchShortTerm();
      await searchLongTerm("conversation");
      setLoading(false);
    })();
  }, [fetchShortTerm, searchLongTerm]);

  const handleCompact = async () => {
    setCompacting(true);
    setError(null);
    try {
      const res = await fetch("/api/v1/memory/compact", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ agent_id: agentName }),
      });
      const data = await res.json();
      if (!res.ok) {
        setError(data.detail || "Compaction failed");
      } else {
        await fetchShortTerm();
        await searchLongTerm("conversation");
      }
    } catch {
      setError("Compaction failed");
    } finally {
      setCompacting(false);
    }
  };

  const handleStore = async () => {
    if (!storeKey.trim() || !storeValue.trim()) return;
    try {
      await fetch("/api/v1/memory/short-term", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          agent_id: agentName,
          key: storeKey.trim(),
          value: storeValue.trim(),
          ttl: 3600,
        }),
      });
      setStoreKey("");
      setStoreValue("");
      await fetchShortTerm();
    } catch {
      setError("Failed to store memory");
    }
  };

  const handleSearch = () => {
    if (searchQuery.trim()) searchLongTerm(searchQuery.trim());
  };

  if (loading) {
    return (
      <Card>
        <CardBody>
          <div style={{ textAlign: "center", padding: 24 }}><Spinner size="lg" /></div>
        </CardBody>
      </Card>
    );
  }

  return (
    <Card>
      <CardTitle>
        <span style={{ display: "flex", alignItems: "center", gap: 8 }}>
          <MemoryIcon /> Memory
          <Label isCompact color="blue">{shortTerm.length} short-term</Label>
          <Label isCompact color="purple">{longTerm.length} long-term</Label>
        </span>
      </CardTitle>
      <CardBody>
        {error && (
          <Alert
            variant="warning"
            title={error}
            isInline
            style={{ marginBottom: 16 }}
            actionClose={<AlertActionCloseButton onClose={() => setError(null)} />}
          />
        )}

        <Tabs activeKey={activeTab} onSelect={(_e, k) => setActiveTab(k as number)}>
          <Tab eventKey={0} title={<TabTitleText>Short-Term ({shortTerm.length})</TabTitleText>}>
            <div style={{ marginTop: 16 }}>
              <div style={{ display: "flex", gap: 8, marginBottom: 16, alignItems: "flex-end" }}>
                <FormGroup label="Key" fieldId="mem-key" style={{ flex: 1 }}>
                  <TextInput id="mem-key" value={storeKey} onChange={(_e, v) => setStoreKey(v)} placeholder="e.g. preference" />
                </FormGroup>
                <FormGroup label="Value" fieldId="mem-val" style={{ flex: 2 }}>
                  <TextInput id="mem-val" value={storeValue} onChange={(_e, v) => setStoreValue(v)} placeholder="e.g. prefers formal tone" />
                </FormGroup>
                <Button
                  variant="secondary"
                  icon={<PlusCircleIcon />}
                  onClick={handleStore}
                  isDisabled={!storeKey.trim() || !storeValue.trim()}
                  size="sm"
                >
                  Store
                </Button>
                <Button
                  variant="secondary"
                  icon={<CompressIcon />}
                  onClick={handleCompact}
                  isLoading={compacting}
                  isDisabled={shortTerm.length === 0}
                  size="sm"
                >
                  Compact
                </Button>
              </div>

              {shortTerm.length === 0 ? (
                <div style={{ padding: 16, textAlign: "center", color: "var(--pf-t--global--text--color--subtle)", fontSize: 13 }}>
                  No short-term memories. Chat with this agent to start building history.
                </div>
              ) : (
                <Table aria-label="Short-term memory" variant="compact">
                  <Thead>
                    <Tr>
                      <Th>Key</Th>
                      <Th>Value</Th>
                      <Th>Expires</Th>
                    </Tr>
                  </Thead>
                  <Tbody>
                    {shortTerm.map((e) => (
                      <Tr key={e.id}>
                        <Td dataLabel="Key">
                          <code style={{ fontSize: 12 }}>{e.key}</code>
                        </Td>
                        <Td dataLabel="Value">
                          <span style={{ fontSize: 13, maxWidth: 400, display: "inline-block", overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                            {typeof e.value === "string" ? e.value : JSON.stringify(e.value)}
                          </span>
                        </Td>
                        <Td dataLabel="Expires">
                          <span style={{ fontSize: 12, color: "var(--pf-t--global--text--color--subtle)" }}>
                            {new Date(e.expires_at).toLocaleTimeString()}
                          </span>
                        </Td>
                      </Tr>
                    ))}
                  </Tbody>
                </Table>
              )}
            </div>
          </Tab>

          <Tab eventKey={1} title={<TabTitleText>Long-Term ({longTerm.length})</TabTitleText>}>
            <div style={{ marginTop: 16 }}>
              <div style={{ display: "flex", gap: 8, marginBottom: 16 }}>
                <TextInput
                  value={searchQuery}
                  onChange={(_e, v) => setSearchQuery(v)}
                  placeholder="Search agent memories..."
                  style={{ flex: 1 }}
                  onKeyDown={(e) => e.key === "Enter" && handleSearch()}
                />
                <Button
                  variant="secondary"
                  icon={<SearchIcon />}
                  onClick={handleSearch}
                  isLoading={searching}
                  size="sm"
                >
                  Search
                </Button>
              </div>

              {longTerm.length === 0 ? (
                <div style={{ padding: 16, textAlign: "center", color: "var(--pf-t--global--text--color--subtle)", fontSize: 13 }}>
                  No long-term memories yet. Interactions with action steps are automatically stored.
                </div>
              ) : (
                <Table aria-label="Long-term memory" variant="compact">
                  <Thead>
                    <Tr>
                      <Th>Content</Th>
                      <Th>Type</Th>
                      <Th>Score</Th>
                      <Th>Stored</Th>
                    </Tr>
                  </Thead>
                  <Tbody>
                    {longTerm.map((r) => (
                      <Tr key={r.memory.id}>
                        <Td dataLabel="Content">
                          <span style={{ fontSize: 13, maxWidth: 500, display: "inline-block", overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                            {r.memory.content}
                          </span>
                        </Td>
                        <Td dataLabel="Type">
                          <Label isCompact color="blue">
                            {(r.memory.metadata?.type as string) || "general"}
                          </Label>
                        </Td>
                        <Td dataLabel="Score">
                          <span style={{ fontSize: 12, fontFamily: "monospace" }}>
                            {r.score.toFixed(3)}
                          </span>
                        </Td>
                        <Td dataLabel="Stored">
                          <span style={{ fontSize: 12, color: "var(--pf-t--global--text--color--subtle)" }}>
                            {new Date(r.memory.created_at).toLocaleString()}
                          </span>
                        </Td>
                      </Tr>
                    ))}
                  </Tbody>
                </Table>
              )}
            </div>
          </Tab>
        </Tabs>
      </CardBody>
    </Card>
  );
};
