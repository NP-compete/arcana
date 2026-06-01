import { useCallback, useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Spinner, Alert, Label } from "@patternfly/react-core";
import { ArrowLeftIcon } from "@patternfly/react-icons";

interface ChatStep {
  type: "action" | "result" | "error";
  service: string;
  message: string;
}

interface ConversationMessage {
  id: string;
  role: "user" | "bot";
  content: string;
  steps?: ChatStep[];
  timestamp: string;
}

interface AgentInfo {
  name: string;
  capabilities: string[];
  protocols: string[];
  status: string;
}

const STATUS_LABEL: Record<string, string> = {
  active: "Running",
  busy: "Busy",
  idle: "Sleeping",
  offline: "Crashed",
};

const STATUS_COLOR: Record<string, string> = {
  active: "#22c55e",
  busy: "#f59e0b",
  idle: "#8b95a5",
  offline: "#ef4444",
};

function formatSteps(steps: ChatStep[]): string {
  if (!steps?.length) return "";
  return "\n\n---\n" + steps.map((s) => {
    const icon = s.type === "action" ? "⚙️" : s.type === "result" ? "✅" : "❌";
    return `${icon} **${s.service}** — ${s.message}`;
  }).join("\n\n");
}

interface AgentChatPageProps {
  agentName: string;
}

export const AgentChatPage = ({ agentName }: AgentChatPageProps) => {
  const navigate = useNavigate();
  const [agentInfo, setAgentInfo] = useState<AgentInfo | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [messages, setMessages] = useState<ConversationMessage[]>([]);
  const [input, setInput] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [sessionId, setSessionId] = useState<string | null>(null);
  const scrollRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    (async () => {
      try {
        const res = await fetch(`/api/v1/agents/${agentName}`);
        if (!res.ok) throw new Error(`Agent not found (${res.status})`);
        setAgentInfo(await res.json());
      } catch (e) {
        setError(e instanceof Error ? e.message : "Failed to load agent");
      } finally {
        setLoading(false);
      }
    })();
  }, [agentName]);

  useEffect(() => { scrollRef.current?.scrollIntoView({ behavior: "smooth" }); }, [messages, isLoading]);
  useEffect(() => { if (!loading && agentInfo) inputRef.current?.focus(); }, [loading, agentInfo]);

  const send = useCallback(async (text: string) => {
    if (!text.trim() || isLoading) return;
    setMessages((prev) => [...prev, { id: `user-${Date.now()}`, role: "user", content: text, timestamp: new Date().toLocaleTimeString() }]);
    setInput("");
    setIsLoading(true);

    try {
      const payload: Record<string, string> = { message: text };
      if (sessionId) payload.session_id = sessionId;
      const res = await fetch(`/api/v1/agents/${agentName}/chat`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      const data = await res.json();
      if (data.session_id) setSessionId(data.session_id);
      const reply = (data.reply ?? "Done.") + formatSteps(data.steps ?? []);
      setMessages((prev) => [...prev, { id: `bot-${Date.now()}`, role: "bot", content: reply, steps: data.steps, timestamp: new Date().toLocaleTimeString() }]);
    } catch {
      setMessages((prev) => [...prev, { id: `err-${Date.now()}`, role: "bot", content: "Could not reach the agent. Is the platform running?", timestamp: new Date().toLocaleTimeString() }]);
    } finally {
      setIsLoading(false);
    }
  }, [isLoading, agentName, sessionId]);

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); send(input); }
  };

  if (loading) {
    return (
      <div style={{ display: "flex", alignItems: "center", justifyContent: "center", height: "100vh", background: "var(--arcana-bg)" }}>
        <Spinner size="xl" />
      </div>
    );
  }

  if (error || !agentInfo) {
    return (
      <div style={{ padding: 32, background: "var(--arcana-bg)", minHeight: "100vh" }}>
        <button type="button" onClick={() => navigate("/agents")} style={{
          display: "flex", alignItems: "center", gap: 6, background: "none", border: "none",
          color: "var(--arcana-text-secondary)", cursor: "pointer", fontSize: 14, marginBottom: 16,
        }}>
          <ArrowLeftIcon /> Back to Agents
        </button>
        <Alert variant="danger" title="Agent not found" isInline>{error}</Alert>
      </div>
    );
  }

  const initial = agentName[0]?.toUpperCase() ?? "A";
  const statusLabel = STATUS_LABEL[agentInfo.status] ?? agentInfo.status;
  const statusColor = STATUS_COLOR[agentInfo.status] ?? "#8b95a5";

  const SUGGESTIONS = [
    { label: "Check status", msg: "What's your current status?" },
    { label: "Run a task", msg: "Send the weekly newsletter to all subscribers" },
    { label: "List tools", msg: "What tools are available to you?" },
    { label: "View costs", msg: "How much have you spent so far?" },
  ];

  return (
    <div style={{ display: "flex", flexDirection: "column", height: "100vh", background: "var(--arcana-bg)" }}>
      {/* Header */}
      <div style={{
        padding: "12px 20px", borderBottom: "1px solid var(--arcana-card-border)",
        display: "flex", alignItems: "center", gap: 12, background: "var(--arcana-bg-secondary)",
      }}>
        <button type="button" onClick={() => navigate(`/agents/${agentName}`)} style={{
          background: "none", border: "none", cursor: "pointer", color: "var(--arcana-text-muted)", padding: 4,
        }}>
          <ArrowLeftIcon />
        </button>
        <div style={{
          width: 30, height: 30, borderRadius: 8,
          background: "linear-gradient(135deg, #5b8def, #a855f7)",
          display: "flex", alignItems: "center", justifyContent: "center",
          color: "#fff", fontSize: 14, fontWeight: 800,
        }}>{initial}</div>
        <div>
          <div style={{ fontSize: 15, fontWeight: 600, color: "var(--arcana-text)" }}>{agentName}</div>
          <div style={{ display: "flex", alignItems: "center", gap: 6 }}>
            <span style={{ width: 6, height: 6, borderRadius: "50%", background: statusColor, display: "inline-block" }} />
            <span style={{ fontSize: 11, color: "var(--arcana-text-muted)" }}>{statusLabel}</span>
          </div>
        </div>
      </div>

      {/* Messages */}
      <div style={{ flex: 1, overflowY: "auto", padding: "24px 0" }}>
        <div style={{ maxWidth: 720, margin: "0 auto", padding: "0 24px" }}>
          {messages.length === 0 && (
            <div style={{ textAlign: "center", paddingTop: 60 }}>
              <div style={{
                width: 52, height: 52, borderRadius: 14, margin: "0 auto 18px",
                background: "linear-gradient(135deg, #5b8def, #a855f7)",
                display: "flex", alignItems: "center", justifyContent: "center",
                color: "#fff", fontSize: 24, fontWeight: 800,
              }}>{initial}</div>
              <h2 style={{ fontSize: 20, fontWeight: 600, color: "var(--arcana-text)", margin: "0 0 6px" }}>
                Talk to {agentName}
              </h2>
              <p style={{ fontSize: 13, color: "var(--arcana-text-muted)", margin: "0 0 24px" }}>
                Give it a task, ask a question, or explore what it can do.
              </p>
              <div style={{ display: "flex", gap: 8, justifyContent: "center", flexWrap: "wrap" }}>
                {SUGGESTIONS.map((s) => (
                  <button key={s.label} type="button" onClick={() => send(s.msg)} style={{
                    padding: "8px 16px", borderRadius: 20, fontSize: 13, fontWeight: 500,
                    border: "1px solid var(--arcana-card-border)", background: "var(--arcana-card-bg)",
                    color: "var(--arcana-text-secondary)", cursor: "pointer", transition: "all 0.15s",
                  }}
                  onMouseEnter={(e) => { (e.currentTarget as HTMLButtonElement).style.borderColor = "rgba(91,141,239,0.4)"; }}
                  onMouseLeave={(e) => { (e.currentTarget as HTMLButtonElement).style.borderColor = "var(--arcana-card-border)"; }}
                  >
                    {s.label}
                  </button>
                ))}
              </div>
            </div>
          )}

          {messages.map((msg) => (
            <div key={msg.id} style={{
              display: "flex", gap: 12, marginBottom: 20,
              flexDirection: msg.role === "user" ? "row-reverse" : "row",
            }}>
              {msg.role === "bot" && (
                <div style={{
                  width: 28, height: 28, borderRadius: 8, flexShrink: 0,
                  background: "linear-gradient(135deg, #5b8def, #a855f7)",
                  display: "flex", alignItems: "center", justifyContent: "center",
                  color: "#fff", fontSize: 12, fontWeight: 800, marginTop: 2,
                }}>{initial}</div>
              )}
              <div style={{
                maxWidth: "75%", padding: "12px 16px",
                borderRadius: msg.role === "user" ? "16px 16px 4px 16px" : "16px 16px 16px 4px",
                background: msg.role === "user" ? "rgba(91,141,239,0.15)" : "var(--arcana-card-bg)",
                border: msg.role === "user" ? "none" : "1px solid var(--arcana-card-border)",
                color: "var(--arcana-text)", fontSize: 14, lineHeight: 1.6,
                whiteSpace: "pre-wrap", wordBreak: "break-word",
              }}>
                {msg.content}
              </div>
            </div>
          ))}

          {isLoading && (
            <div style={{ display: "flex", gap: 12, marginBottom: 20 }}>
              <div style={{
                width: 28, height: 28, borderRadius: 8, flexShrink: 0,
                background: "linear-gradient(135deg, #5b8def, #a855f7)",
                display: "flex", alignItems: "center", justifyContent: "center",
                color: "#fff", fontSize: 12, fontWeight: 800,
              }}>{initial}</div>
              <div style={{
                padding: "12px 16px", borderRadius: "16px 16px 16px 4px",
                background: "var(--arcana-card-bg)", border: "1px solid var(--arcana-card-border)",
              }}>
                <span className="arcana-typing-dots"><span /><span /><span /></span>
              </div>
            </div>
          )}

          <div ref={scrollRef} />
        </div>
      </div>

      {/* Input */}
      <div style={{ padding: "16px 24px 20px", borderTop: "1px solid var(--arcana-card-border)", background: "var(--arcana-bg-secondary)" }}>
        <div style={{ maxWidth: 720, margin: "0 auto", display: "flex", gap: 10, alignItems: "flex-end" }}>
          <textarea
            ref={inputRef}
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={`Message ${agentName}...`}
            rows={1}
            style={{
              flex: 1, padding: "12px 16px", borderRadius: 12, resize: "none",
              border: "1px solid var(--arcana-card-border)", background: "var(--arcana-input-bg)",
              color: "var(--arcana-text)", fontSize: 14, lineHeight: 1.5,
              outline: "none", fontFamily: "inherit", minHeight: 44, maxHeight: 120,
            }}
            onFocus={(e) => { e.currentTarget.style.borderColor = "rgba(91,141,239,0.5)"; }}
            onBlur={(e) => { e.currentTarget.style.borderColor = "var(--arcana-card-border)"; }}
          />
          <button type="button" onClick={() => send(input)} disabled={isLoading || !input.trim()} style={{
            width: 44, height: 44, borderRadius: 12, border: "none",
            background: input.trim() ? "linear-gradient(135deg, #5b8def, #a855f7)" : "var(--arcana-card-bg)",
            color: input.trim() ? "#fff" : "var(--arcana-text-muted)",
            cursor: input.trim() && !isLoading ? "pointer" : "default",
            display: "flex", alignItems: "center", justifyContent: "center",
            transition: "all 0.15s", flexShrink: 0,
          }}>
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <line x1="22" y1="2" x2="11" y2="13" /><polygon points="22 2 15 22 11 13 2 9 22 2" />
            </svg>
          </button>
        </div>
        <div style={{ maxWidth: 720, margin: "6px auto 0", fontSize: 11, color: "var(--arcana-text-muted)", textAlign: "center" }}>
          All traffic stays in-cluster
        </div>
      </div>
    </div>
  );
};
