import { useCallback, useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { detectCommand, executePlatformCommand } from "./platformCommands";

interface ConversationMessage {
  id: string;
  role: "user" | "bot";
  content: string;
  agentName?: string;
  timestamp: string;
}

const QUICK = [
  { label: "Deploy agent", msg: "Deploy an agent called my-assistant" },
  { label: "System status", msg: "What is the system health status?" },
  { label: "List skills", msg: "List all available skills" },
];

interface ChatDrawerProps {
  isOpen: boolean;
  onClose: () => void;
}

export const ChatDrawer = ({ isOpen, onClose }: ChatDrawerProps) => {
  const navigate = useNavigate();
  const { authHeaders } = useAuth();
  const [messages, setMessages] = useState<ConversationMessage[]>([]);
  const [input, setInput] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [sessionId, setSessionId] = useState<string | null>(null);
  const scrollRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const abortRef = useRef<AbortController | null>(null);

  useEffect(() => { scrollRef.current?.scrollIntoView({ behavior: "smooth" }); }, [messages, isLoading]);

  useEffect(() => {
    if (!isOpen && abortRef.current) { abortRef.current.abort(); abortRef.current = null; }
    if (isOpen) setTimeout(() => inputRef.current?.focus(), 100);
  }, [isOpen]);

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
    } catch (err) {
      if (err instanceof DOMException && err.name === "AbortError") return;
      setMessages((prev) => prev.map((m) => m.id === botId ? { ...m, content: "Could not reach the API." } : m));
    }
  }, [sessionId, authHeaders]);

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
        setMessages((prev) => [...prev, { id: `err-${Date.now()}`, role: "bot", content: "Command failed.", timestamp: new Date().toLocaleTimeString() }]);
      } finally { setIsLoading(false); }
    } else {
      try { await streamResponse(text); } finally { setIsLoading(false); }
    }
  }, [isLoading, authHeaders, streamResponse]);

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); send(input); }
  };

  if (!isOpen) return null;

  return (
    <div className="arcana-chat-overlay">
      <div style={{ display: "flex", flexDirection: "column", height: "100%", background: "var(--arcana-bg-secondary)", borderRadius: 16, overflow: "hidden" }}>
        {/* Header */}
        <div style={{
          padding: "14px 18px", display: "flex", justifyContent: "space-between", alignItems: "center",
          borderBottom: "1px solid var(--arcana-card-border)",
        }}>
          <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
            <div style={{
              width: 26, height: 26, borderRadius: 7,
              background: "linear-gradient(135deg, #5b8def, #a855f7)",
              display: "flex", alignItems: "center", justifyContent: "center",
              color: "#fff", fontSize: 12, fontWeight: 800,
            }}>A</div>
            <span style={{ fontSize: 14, fontWeight: 600, color: "var(--arcana-text)" }}>Arcana</span>
          </div>
          <button type="button" onClick={onClose} style={{
            background: "none", border: "none", cursor: "pointer",
            color: "var(--arcana-text-muted)", fontSize: 18, padding: 4, lineHeight: 1,
          }} aria-label="Close chat">
            &times;
          </button>
        </div>

        {/* Messages */}
        <div style={{ flex: 1, overflowY: "auto", padding: "16px 16px" }}>
          {messages.length === 0 && (
            <div style={{ textAlign: "center", paddingTop: 40 }}>
              <div style={{
                width: 40, height: 40, borderRadius: 12, margin: "0 auto 14px",
                background: "linear-gradient(135deg, #5b8def, #a855f7)",
                display: "flex", alignItems: "center", justifyContent: "center",
                color: "#fff", fontSize: 18, fontWeight: 800,
              }}>A</div>
              <div style={{ fontSize: 15, fontWeight: 600, color: "var(--arcana-text)", marginBottom: 6 }}>
                How can I help?
              </div>
              <div style={{ fontSize: 12, color: "var(--arcana-text-muted)", marginBottom: 20 }}>
                Ask anything about the platform.
              </div>
              <div style={{ display: "flex", flexDirection: "column", gap: 6, alignItems: "center" }}>
                {QUICK.map((s) => (
                  <button key={s.label} type="button" onClick={() => send(s.msg)} style={{
                    padding: "7px 14px", borderRadius: 16, fontSize: 12, fontWeight: 500,
                    border: "1px solid var(--arcana-card-border)", background: "var(--arcana-card-bg)",
                    color: "var(--arcana-text-secondary)", cursor: "pointer", transition: "all 0.15s",
                  }}>
                    {s.label}
                  </button>
                ))}
              </div>
            </div>
          )}

          {messages.map((msg) => (
            <div key={msg.id} style={{
              display: "flex", gap: 8, marginBottom: 14,
              flexDirection: msg.role === "user" ? "row-reverse" : "row",
            }}>
              {msg.role === "bot" && (
                <div style={{
                  width: 24, height: 24, borderRadius: 6, flexShrink: 0,
                  background: "linear-gradient(135deg, #5b8def, #a855f7)",
                  display: "flex", alignItems: "center", justifyContent: "center",
                  color: "#fff", fontSize: 10, fontWeight: 800, marginTop: 2,
                }}>A</div>
              )}
              <div style={{
                maxWidth: "80%", padding: "10px 14px",
                borderRadius: msg.role === "user" ? "14px 14px 4px 14px" : "14px 14px 14px 4px",
                background: msg.role === "user" ? "rgba(91,141,239,0.15)" : "var(--arcana-card-bg)",
                border: msg.role === "user" ? "none" : "1px solid var(--arcana-card-border)",
                color: "var(--arcana-text)", fontSize: 13, lineHeight: 1.5,
                whiteSpace: "pre-wrap", wordBreak: "break-word",
              }}>
                {msg.content}
                {msg.agentName && (
                  <div style={{ marginTop: 6 }}>
                    <button type="button" onClick={() => { onClose(); navigate(`/agents/${msg.agentName}`); }} style={{
                      padding: "4px 10px", borderRadius: 4, border: "none", fontSize: 11, fontWeight: 600,
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
            <div style={{ display: "flex", gap: 8, marginBottom: 14 }}>
              <div style={{
                width: 24, height: 24, borderRadius: 6, flexShrink: 0,
                background: "linear-gradient(135deg, #5b8def, #a855f7)",
                display: "flex", alignItems: "center", justifyContent: "center",
                color: "#fff", fontSize: 10, fontWeight: 800,
              }}>A</div>
              <div style={{
                padding: "10px 14px", borderRadius: "14px 14px 14px 4px",
                background: "var(--arcana-card-bg)", border: "1px solid var(--arcana-card-border)",
              }}>
                <span className="arcana-typing-dots"><span /><span /><span /></span>
              </div>
            </div>
          )}

          <div ref={scrollRef} />
        </div>

        {/* Input */}
        <div style={{ padding: "12px 14px 16px", borderTop: "1px solid var(--arcana-card-border)" }}>
          <div style={{ display: "flex", gap: 8, alignItems: "flex-end" }}>
            <textarea
              ref={inputRef}
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="Ask something..."
              rows={1}
              style={{
                flex: 1, padding: "10px 14px", borderRadius: 10, resize: "none",
                border: "1px solid var(--arcana-card-border)", background: "var(--arcana-input-bg)",
                color: "var(--arcana-text)", fontSize: 13, lineHeight: 1.4,
                outline: "none", fontFamily: "inherit", minHeight: 38, maxHeight: 80,
              }}
              onFocus={(e) => { e.currentTarget.style.borderColor = "rgba(91,141,239,0.5)"; }}
              onBlur={(e) => { e.currentTarget.style.borderColor = "var(--arcana-card-border)"; }}
            />
            <button type="button" onClick={() => send(input)} disabled={isLoading || !input.trim()} style={{
              width: 38, height: 38, borderRadius: 10, border: "none",
              background: input.trim() ? "linear-gradient(135deg, #5b8def, #a855f7)" : "var(--arcana-card-bg)",
              color: input.trim() ? "#fff" : "var(--arcana-text-muted)",
              cursor: input.trim() && !isLoading ? "pointer" : "default",
              display: "flex", alignItems: "center", justifyContent: "center",
              transition: "all 0.15s", flexShrink: 0,
            }}>
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <line x1="22" y1="2" x2="11" y2="13" /><polygon points="22 2 15 22 11 13 2 9 22 2" />
              </svg>
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};
