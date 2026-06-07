import { useState, useCallback, useRef, useEffect } from "react";
import {
  PageSection,
  Title,
  Card,
  CardBody,
  CardTitle,
  Content,
  Button,
  Grid,
  GridItem,
  Label,
  TextInput,
  Divider,
  Spinner,
  Alert,
  DescriptionList,
  DescriptionListGroup,
  DescriptionListTerm,
  DescriptionListDescription,
  ExpandableSection,
} from "@patternfly/react-core";
import {
  CheckCircleIcon,
  ExclamationCircleIcon,
  CubesIcon,
  ShieldAltIcon,
  DollarSignIcon,
  PaperPlaneIcon,
  TimesIcon,
} from "@patternfly/react-icons";

interface ToolCall {
  tool: string;
  status: string;
  duration_ms?: number;
}

interface GuardrailCheck {
  rule: string;
  status: string;
}

interface ChatMessage {
  role: "user" | "assistant";
  content: string;
  toolCalls?: ToolCall[];
  guardrails?: GuardrailCheck[];
  reasoning?: string[];
  tokensUsed?: number;
  costUsd?: number;
}

interface SessionInfo {
  id: string;
  agent_name: string;
  budget_limit: number;
  budget_used: number;
  tokens_used: number;
  message_count: number;
  status: string;
}

export const PlaygroundPage = () => {
  const [agentName, setAgentName] = useState("");
  const [session, setSession] = useState<SessionInfo | null>(null);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [input, setInput] = useState("");
  const [sending, setSending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [selectedMsg, setSelectedMsg] = useState<number | null>(null);
  const scrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    scrollRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  const startSession = useCallback(async () => {
    if (!agentName.trim()) return;
    setError(null);
    try {
      const res = await fetch("/api/v1/playground/start", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ agent_name: agentName.trim(), budget_limit: 1.0 }),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      setSession(data);
      setMessages([]);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to start session");
    }
  }, [agentName]);

  const sendMessage = useCallback(async () => {
    if (!input.trim() || !session) return;
    const userMsg: ChatMessage = { role: "user", content: input };
    setMessages((prev) => [...prev, userMsg]);
    setInput("");
    setSending(true);
    try {
      const res = await fetch(`/api/v1/playground/${session.id}/chat`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ message: input }),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      const botMsg: ChatMessage = {
        role: "assistant",
        content: data.reply,
        toolCalls: (data.tool_calls ?? []).map((t: Record<string, string>) => ({
          tool: t.tool ?? "unknown",
          status: t.status ?? "ok",
        })),
        guardrails: data.guardrails ?? [],
        reasoning: data.reasoning ?? [],
        tokensUsed: data.tokens_used ?? 0,
        costUsd: data.cost_usd ?? 0,
      };
      setMessages((prev) => [...prev, botMsg]);
      setSelectedMsg(messages.length + 1);
      setSession((prev) =>
        prev ? { ...prev, budget_used: data.budget_used, tokens_used: (prev.tokens_used ?? 0) + (data.tokens_used ?? 0), message_count: (prev.message_count ?? 0) + 1 } : prev
      );
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to send");
    } finally {
      setSending(false);
    }
  }, [input, session, messages.length]);

  const endSession = useCallback(async () => {
    if (!session) return;
    await fetch(`/api/v1/playground/${session.id}/end`, { method: "POST" });
    setSession(null);
    setMessages([]);
    setSelectedMsg(null);
  }, [session]);

  const inspectedMsg = selectedMsg !== null ? messages[selectedMsg] : null;

  if (!session) {
    return (
      <PageSection hasBodyWrapper={false}>
        <div style={{ maxWidth: 500, margin: "60px auto", textAlign: "center" }}>
          <Title headingLevel="h1" size="2xl" style={{ marginBottom: 8 }}>
            Playground
          </Title>
          <Content component="p" style={{ marginBottom: 24, color: "var(--pf-t--global--text--color--subtle)" }}>
            Test-drive an agent in a sandboxed environment. Ephemeral memory, read-only tools, $1 budget.
          </Content>
          {error && (
            <Alert variant="danger" title="Error" isInline style={{ marginBottom: 16 }}>
              {error}
            </Alert>
          )}
          <div style={{ display: "flex", gap: 8 }}>
            <TextInput
              aria-label="Agent name"
              placeholder="Enter agent name..."
              value={agentName}
              onChange={(_e, v) => setAgentName(v)}
              onKeyDown={(e) => e.key === "Enter" && startSession()}
            />
            <Button variant="primary" onClick={startSession} isDisabled={!agentName.trim()}>
              Start Session
            </Button>
          </div>
        </div>
      </PageSection>
    );
  }

  return (
    <>
      <PageSection hasBodyWrapper={false} style={{ paddingBottom: 0 }}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
          <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
            <Title headingLevel="h1" size="xl">
              Playground: {session.agent_name}
            </Title>
            <Label color="blue" isCompact>sandbox</Label>
          </div>
          <div style={{ display: "flex", alignItems: "center", gap: 16 }}>
            <span style={{ fontSize: 13, color: "var(--pf-t--global--text--color--subtle)" }}>
              <DollarSignIcon /> ${session.budget_used?.toFixed(4)} / ${session.budget_limit?.toFixed(2)}
            </span>
            <Button variant="secondary" icon={<TimesIcon />} onClick={endSession}>
              End Session
            </Button>
          </div>
        </div>
      </PageSection>
      <Divider />
      <PageSection hasBodyWrapper={false} style={{ flex: 1 }}>
        {error && (
          <Alert variant="danger" title="Error" isInline style={{ marginBottom: 16 }}>
            {error}
          </Alert>
        )}
        <Grid hasGutter style={{ height: "calc(100vh - 220px)" }}>
          {/* Chat Panel */}
          <GridItem span={7}>
            <Card style={{ height: "100%", display: "flex", flexDirection: "column" }}>
              <CardBody style={{ flex: 1, overflow: "auto", padding: "16px" }}>
                {messages.length === 0 && (
                  <div style={{ textAlign: "center", padding: 40, color: "var(--pf-t--global--text--color--subtle)" }}>
                    <Content component="p">Send a message to start testing the agent.</Content>
                  </div>
                )}
                {messages.map((msg, i) => (
                  <div
                    key={i}
                    onClick={() => msg.role === "assistant" && setSelectedMsg(i)}
                    style={{
                      marginBottom: 16,
                      padding: "12px 16px",
                      borderRadius: 8,
                      cursor: msg.role === "assistant" ? "pointer" : "default",
                      background: msg.role === "user"
                        ? "rgba(255,255,255,0.06)"
                        : selectedMsg === i
                        ? "rgba(102, 126, 234, 0.15)"
                        : "rgba(255,255,255,0.02)",
                      border: selectedMsg === i ? "1px solid rgba(102, 126, 234, 0.4)" : "1px solid transparent",
                    }}
                  >
                    <div style={{ fontSize: 11, color: "var(--pf-t--global--text--color--subtle)", marginBottom: 4 }}>
                      {msg.role === "user" ? "You" : session.agent_name}
                    </div>
                    <div style={{ whiteSpace: "pre-wrap" }}>{msg.content}</div>
                  </div>
                ))}
                {sending && (
                  <div style={{ textAlign: "center", padding: 12 }}>
                    <Spinner size="md" />
                  </div>
                )}
                <div ref={scrollRef} />
              </CardBody>
              <div style={{ padding: "12px 16px", borderTop: "1px solid rgba(255,255,255,0.08)" }}>
                <div style={{ display: "flex", gap: 8 }}>
                  <TextInput
                    aria-label="Message"
                    placeholder="Type a message..."
                    value={input}
                    onChange={(_e, v) => setInput(v)}
                    onKeyDown={(e) => e.key === "Enter" && !sending && sendMessage()}
                    isDisabled={sending}
                  />
                  <Button variant="primary" icon={<PaperPlaneIcon />} onClick={sendMessage} isDisabled={sending || !input.trim()}>
                    Send
                  </Button>
                </div>
              </div>
            </Card>
          </GridItem>

          {/* Inspector Panel */}
          <GridItem span={5}>
            <Card style={{ height: "100%", overflow: "auto" }}>
              <CardTitle>Inspector</CardTitle>
              <CardBody>
                {inspectedMsg && inspectedMsg.role === "assistant" ? (
                  <>
                    <ExpandableSection toggleText={`Reasoning (${inspectedMsg.reasoning?.length ?? 0})`} isIndented>
                      {(inspectedMsg.reasoning ?? []).map((r, i) => (
                        <div key={i} style={{ padding: "4px 0", fontSize: 13, color: "var(--pf-t--global--text--color--subtle)" }}>
                          {r}
                        </div>
                      ))}
                      {(!inspectedMsg.reasoning || inspectedMsg.reasoning.length === 0) && (
                        <Content component="p" style={{ fontSize: 13, color: "var(--pf-t--global--text--color--subtle)" }}>No reasoning trace.</Content>
                      )}
                    </ExpandableSection>

                    <Divider style={{ margin: "12px 0" }} />

                    <ExpandableSection toggleText={`Tool Calls (${inspectedMsg.toolCalls?.length ?? 0})`} isIndented>
                      {(inspectedMsg.toolCalls ?? []).map((t, i) => (
                        <div key={i} style={{ display: "flex", gap: 8, alignItems: "center", padding: "4px 0" }}>
                          <CubesIcon />
                          <span style={{ fontSize: 13 }}>{t.tool}</span>
                          <Label color={t.status === "ok" || t.status === "pass" ? "green" : "red"} isCompact>
                            {t.status}
                          </Label>
                        </div>
                      ))}
                      {(!inspectedMsg.toolCalls || inspectedMsg.toolCalls.length === 0) && (
                        <Content component="p" style={{ fontSize: 13, color: "var(--pf-t--global--text--color--subtle)" }}>No tool calls.</Content>
                      )}
                    </ExpandableSection>

                    <Divider style={{ margin: "12px 0" }} />

                    <ExpandableSection toggleText={`Guardrails (${inspectedMsg.guardrails?.length ?? 0})`} isIndented>
                      {(inspectedMsg.guardrails ?? []).map((g, i) => (
                        <div key={i} style={{ display: "flex", gap: 8, alignItems: "center", padding: "4px 0" }}>
                          <ShieldAltIcon />
                          <span style={{ fontSize: 13 }}>{g.rule}</span>
                          <Label color={g.status === "pass" ? "green" : "red"} isCompact>
                            {g.status === "pass" ? <CheckCircleIcon /> : <ExclamationCircleIcon />} {g.status}
                          </Label>
                        </div>
                      ))}
                    </ExpandableSection>

                    <Divider style={{ margin: "12px 0" }} />

                    <DescriptionList isHorizontal isCompact>
                      <DescriptionListGroup>
                        <DescriptionListTerm>Tokens</DescriptionListTerm>
                        <DescriptionListDescription>{inspectedMsg.tokensUsed ?? 0}</DescriptionListDescription>
                      </DescriptionListGroup>
                      <DescriptionListGroup>
                        <DescriptionListTerm>Cost</DescriptionListTerm>
                        <DescriptionListDescription>${(inspectedMsg.costUsd ?? 0).toFixed(6)}</DescriptionListDescription>
                      </DescriptionListGroup>
                    </DescriptionList>
                  </>
                ) : (
                  <Content component="p" style={{ color: "var(--pf-t--global--text--color--subtle)", padding: 20, textAlign: "center" }}>
                    Click an agent response to inspect its reasoning, tool calls, guardrails, and cost.
                  </Content>
                )}
              </CardBody>
            </Card>
          </GridItem>
        </Grid>
      </PageSection>
    </>
  );
};
