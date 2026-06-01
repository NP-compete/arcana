import { useCallback, useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Spinner } from "@patternfly/react-core";
import { useAuth } from "../auth/AuthContext";
import { detectCommand, executePlatformCommand } from "./platformCommands";

interface ConversationMessage {
  id: string;
  role: "user" | "bot";
  content: string;
  agentName?: string;
  timestamp: string;
}

interface ChatSession {
  id: string;
  title: string;
  updated_at: string;
}

const SUGGESTIONS = [
  { label: "Deploy an agent", msg: "Deploy an agent called my-assistant" },
  { label: "System status", msg: "What is the system health status?" },
  { label: "List skills", msg: "List all available skills" },
  { label: "Show costs", msg: "Show me the current platform costs" },
];

export const PlatformChatPage = () => {
  const navigate = useNavigate();
  const { authHeaders } = useAuth();

  const [messages, setMessages] = useState<ConversationMessage[]>([]);
  const [input, setInput] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [sessionId, setSessionId] = useState<string | null>(null);
  const [sessions, setSessions] = useState<ChatSession[]>([]);
  const [sessionsLoading, setSessionsLoading] = useState(true);
  const [sidebarOpen, setSidebarOpen] = useState(true);
  const scrollRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const abortRef = useRef<AbortController | null>(null);

  useEffect(() => {
    scrollRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages, isLoading]);

  useEffect(() => {
    inputRef.current?.focus();
  }, []);

  const loadSessions = useCallback(async () => {
    setSessionsLoading(true);
    try {
      const res = await fetch("/api/v1/chat/sessions", { headers: { ...authHeaders() } });
      if (res.ok) {
        const data = await res.json();
        setSessions(Array.isArray(data) ? data : data.sessions ?? []);
      }
    } catch { /* best-effort */ }
    finally { setSessionsLoading(false); }
  }, [authHeaders]);

  useEffect(() => { loadSessions(); }, [loadSessions]);

  const loadSessionMessages = useCallback(async (id: string) => {
    try {
      const res = await fetch(`/api/v1/chat/sessions/${id}/messages`, { headers: { ...authHeaders() } });
      if (!res.ok) return;
      const data = await res.json();
      const msgs: Array<{ role: string; content: string; timestamp?: string }> =
        Array.isArray(data) ? data : data.messages ?? [];
      setMessages(msgs.map((m, i) => ({
        id: `${id}-${i}`,
        role: m.role === "user" ? "user" : "bot",
        content: m.content,
        timestamp: m.timestamp ?? "",
      })));
      setSessionId(id);
    } catch { /* best-effort */ }
  }, [authHeaders]);

  const startNew = useCallback(() => {
    abortRef.current?.abort();
    abortRef.current = null;
    setMessages([]);
    setSessionId(null);
    setIsLoading(false);
    inputRef.current?.focus();
  }, []);

  const streamResponse = useCallback(async (text: string) => {
    const controller = new AbortController();
    abortRef.current = controller;
    const botId = `bot-${Date.now()}`;
    setMessages((prev) => [...prev, { id: botId, role: "bot", content: "", timestamp: new Date().toLocaleTimeString() }]);

    try {
      const payload: Record<string, string> = { message: text };
      if (sessionId) payload.session_id = sessionId;
      const res = await fetch("/api/v1/chat/stream", {
        method: "POST",
        headers: { "Content-Type": "application/json", ...authHeaders() },
        body: JSON.stringify(payload),
        signal: controller.signal,
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);

      const ct = res.headers.get("content-type") ?? "";
      if (ct.includes("text/event-stream") && res.body) {
        const reader = res.body.getReader();
        const decoder = new TextDecoder();
        let acc = "";
        while (true) {
          const { done, value } = await reader.read();
          if (done) break;
          for (const line of decoder.decode(value, { stream: true }).split("\n")) {
            if (!line.startsWith("data: ")) continue;
            const d = line.slice(6);
            if (d === "[DONE]") break;
            try {
              const p = JSON.parse(d);
              if (p.session_id) setSessionId(p.session_id);
              const tok = p.token ?? p.content ?? "";
              if (tok) { acc += tok; setMessages((prev) => prev.map((m) => m.id === botId ? { ...m, content: acc } : m)); }
            } catch {
              if (d.trim()) { acc += d; setMessages((prev) => prev.map((m) => m.id === botId ? { ...m, content: acc } : m)); }
            }
          }
        }
        if (!acc) setMessages((prev) => prev.map((m) => m.id === botId ? { ...m, content: "Done." } : m));
      } else {
        const data = await res.json();
        if (data.session_id) setSessionId(data.session_id);
        setMessages((prev) => prev.map((m) => m.id === botId ? { ...m, content: data.reply ?? data.content ?? "Done." } : m));
      }
      loadSessions();
    } catch (err) {
      if (err instanceof DOMException && err.name === "AbortError") return;
      setMessages((prev) => prev.map((m) => m.id === botId ? { ...m, content: "Could not reach the Arcana API. Is the platform running?" } : m));
    }
  }, [sessionId, authHeaders, loadSessions]);

  const send = useCallback(async (text: string) => {
    if (!text.trim() || isLoading) return;
    setMessages((prev) => [...prev, { id: `user-${Date.now()}`, role: "user", content: text, timestamp: new Date().toLocaleTimeString() }]);
    setInput("");
    setIsLoading(true);

    const action = detectCommand(text);
    if (action) {
      try {
        const result = await executePlatformCommand(action, text, authHeaders());
        setMessages((prev) => [...prev, { id: `bot-${Date.now()}`, role: "bot", content: result.markdown, agentName: result.agentName, timestamp: new Date().toLocaleTimeString() }]);
      } catch {
        setMessages((prev) => [...prev, { id: `err-${Date.now()}`, role: "bot", content: "Command failed. Could not reach the API.", timestamp: new Date().toLocaleTimeString() }]);
      } finally { setIsLoading(false); }
    } else {
      try { await streamResponse(text); } finally { setIsLoading(false); }
    }
  }, [isLoading, authHeaders, streamResponse]);

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      send(input);
    }
  };

  return (
    <div style={{ display: "flex", height: "calc(100vh - 76px)", overflow: "hidden" }}>
      {/* Sidebar */}
      {sidebarOpen && (
        <div style={{
          width: 260, flexShrink: 0, borderRight: "1px solid var(--arcana-card-border)",
          background: "var(--arcana-bg-secondary)", display: "flex", flexDirection: "column",
        }}>
          <div style={{ padding: "16px 16px 12px", display: "flex", justifyContent: "space-between", alignItems: "center" }}>
            <span style={{ fontSize: 13, fontWeight: 600, color: "var(--arcana-text)" }}>Conversations</span>
            <button type="button" onClick={startNew} style={{
              fontSize: 12, fontWeight: 600, color: "#5b8def", background: "none", border: "none", cursor: "pointer",
            }}>+ New</button>
          </div>
          <div style={{ flex: 1, overflowY: "auto", padding: "0 8px" }}>
            {sessionsLoading ? (
              <div style={{ textAlign: "center", padding: 24 }}><Spinner size="md" /></div>
            ) : sessions.length === 0 ? (
              <div style={{ padding: "24px 12px", fontSize: 12, color: "var(--arcana-text-muted)", textAlign: "center" }}>
                No conversations yet
              </div>
            ) : sessions.map((s) => (
              <button key={s.id} type="button" onClick={() => loadSessionMessages(s.id)} style={{
                display: "block", width: "100%", textAlign: "left", padding: "10px 12px",
                borderRadius: 8, border: "none", cursor: "pointer", marginBottom: 2,
                background: sessionId === s.id ? "rgba(91,141,239,0.1)" : "transparent",
                color: sessionId === s.id ? "var(--arcana-text)" : "var(--arcana-text-secondary)",
                fontSize: 13, fontWeight: sessionId === s.id ? 600 : 400,
                overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap",
                transition: "background 0.15s",
              }}>
                {s.title || `Session ${s.id.slice(0, 8)}`}
              </button>
            ))}
          </div>
        </div>
      )}

      {/* Main chat area */}
      <div style={{ flex: 1, display: "flex", flexDirection: "column", minWidth: 0 }}>
        {/* Header */}
        <div style={{
          padding: "12px 20px", borderBottom: "1px solid var(--arcana-card-border)",
          display: "flex", alignItems: "center", gap: 12,
        }}>
          <button type="button" onClick={() => setSidebarOpen(!sidebarOpen)} style={{
            background: "none", border: "none", cursor: "pointer", color: "var(--arcana-text-muted)",
            fontSize: 16, padding: 4,
          }}>
            {sidebarOpen ? "☰" : "☰"}
          </button>
          <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
            <div style={{
              width: 28, height: 28, borderRadius: 8,
              background: "linear-gradient(135deg, #5b8def, #a855f7)",
              display: "flex", alignItems: "center", justifyContent: "center",
              color: "#fff", fontSize: 13, fontWeight: 800,
            }}>A</div>
            <span style={{ fontSize: 15, fontWeight: 600, color: "var(--arcana-text)" }}>Arcana</span>
          </div>
        </div>

        {/* Messages */}
        <div style={{ flex: 1, overflowY: "auto", padding: "24px 0" }}>
          <div style={{ maxWidth: 720, margin: "0 auto", padding: "0 24px" }}>
            {messages.length === 0 && !sessionsLoading && (
              <div style={{ textAlign: "center", paddingTop: 80 }}>
                <div style={{
                  width: 48, height: 48, borderRadius: 14, margin: "0 auto 20px",
                  background: "linear-gradient(135deg, #5b8def, #a855f7)",
                  display: "flex", alignItems: "center", justifyContent: "center",
                  color: "#fff", fontSize: 22, fontWeight: 800,
                }}>A</div>
                <h2 style={{ fontSize: 22, fontWeight: 600, color: "var(--arcana-text)", margin: "0 0 8px" }}>
                  What can I help you with?
                </h2>
                <p style={{ fontSize: 14, color: "var(--arcana-text-muted)", margin: "0 0 28px" }}>
                  Deploy agents, check costs, manage skills — in plain English.
                </p>
                <div style={{ display: "flex", gap: 8, justifyContent: "center", flexWrap: "wrap" }}>
                  {SUGGESTIONS.map((s) => (
                    <button key={s.label} type="button" onClick={() => send(s.msg)} style={{
                      padding: "8px 16px", borderRadius: 20, fontSize: 13, fontWeight: 500,
                      border: "1px solid var(--arcana-card-border)", background: "var(--arcana-card-bg)",
                      color: "var(--arcana-text-secondary)", cursor: "pointer", transition: "all 0.15s",
                    }}
                    onMouseEnter={(e) => { (e.currentTarget as HTMLButtonElement).style.borderColor = "rgba(91,141,239,0.4)"; (e.currentTarget as HTMLButtonElement).style.color = "var(--arcana-text)"; }}
                    onMouseLeave={(e) => { (e.currentTarget as HTMLButtonElement).style.borderColor = "var(--arcana-card-border)"; (e.currentTarget as HTMLButtonElement).style.color = "var(--arcana-text-secondary)"; }}
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
                  }}>A</div>
                )}
                <div style={{
                  maxWidth: "75%",
                  padding: "12px 16px",
                  borderRadius: msg.role === "user" ? "16px 16px 4px 16px" : "16px 16px 16px 4px",
                  background: msg.role === "user" ? "rgba(91,141,239,0.15)" : "var(--arcana-card-bg)",
                  border: msg.role === "user" ? "none" : "1px solid var(--arcana-card-border)",
                  color: "var(--arcana-text)",
                  fontSize: 14, lineHeight: 1.6,
                  whiteSpace: "pre-wrap", wordBreak: "break-word",
                }}>
                  {msg.content}
                  {msg.agentName && (
                    <div style={{ marginTop: 8 }}>
                      <button type="button" onClick={() => navigate(`/agents/${msg.agentName}`)} style={{
                        padding: "6px 14px", borderRadius: 6, border: "none", fontSize: 12, fontWeight: 600,
                        background: "linear-gradient(135deg, #5b8def, #a855f7)", color: "#fff", cursor: "pointer",
                      }}>
                        Open {msg.agentName}
                      </button>
                    </div>
                  )}
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
                }}>A</div>
                <div style={{
                  padding: "12px 16px", borderRadius: "16px 16px 16px 4px",
                  background: "var(--arcana-card-bg)", border: "1px solid var(--arcana-card-border)",
                  color: "var(--arcana-text-muted)", fontSize: 14,
                  display: "flex", alignItems: "center", gap: 8,
                }}>
                  <span className="arcana-typing-dots">
                    <span /><span /><span />
                  </span>
                </div>
              </div>
            )}

            <div ref={scrollRef} />
          </div>
        </div>

        {/* Input */}
        <div style={{
          padding: "16px 24px 20px",
          borderTop: "1px solid var(--arcana-card-border)",
        }}>
          <div style={{
            maxWidth: 720, margin: "0 auto",
            display: "flex", gap: 10, alignItems: "flex-end",
          }}>
            <textarea
              ref={inputRef}
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="Ask Arcana anything..."
              rows={1}
              style={{
                flex: 1, padding: "12px 16px", borderRadius: 12, resize: "none",
                border: "1px solid var(--arcana-card-border)", background: "var(--arcana-input-bg)",
                color: "var(--arcana-text)", fontSize: 14, lineHeight: 1.5,
                outline: "none", fontFamily: "inherit",
                transition: "border-color 0.15s",
                minHeight: 44, maxHeight: 120,
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
            All data stays in your cluster
          </div>
        </div>
      </div>
    </div>
  );
};
