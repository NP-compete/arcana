import { useCallback, useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import Chatbot, {
  ChatbotDisplayMode,
} from "@patternfly/chatbot/dist/dynamic/Chatbot";
import ChatbotContent from "@patternfly/chatbot/dist/dynamic/ChatbotContent";
import ChatbotWelcomePrompt from "@patternfly/chatbot/dist/dynamic/ChatbotWelcomePrompt";
import ChatbotFooter, {
  ChatbotFootnote,
} from "@patternfly/chatbot/dist/dynamic/ChatbotFooter";
import MessageBar from "@patternfly/chatbot/dist/dynamic/MessageBar";
import MessageBox from "@patternfly/chatbot/dist/dynamic/MessageBox";
import Message from "@patternfly/chatbot/dist/dynamic/Message";
import ChatbotHeader, {
  ChatbotHeaderMain,
  ChatbotHeaderTitle,
  ChatbotHeaderActions,
} from "@patternfly/chatbot/dist/dynamic/ChatbotHeader";
import ChatbotConversationHistoryNav, {
  type Conversation,
} from "@patternfly/chatbot/dist/dynamic/ChatbotConversationHistoryNav";
import { Button, Spinner } from "@patternfly/react-core";
import { PlusCircleIcon } from "@patternfly/react-icons";
import { useAuth } from "../auth/AuthContext";
import {
  detectCommand,
  executePlatformCommand,
} from "./platformCommands";

/* ---------- types ---------- */

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

/* ---------- welcome prompts ---------- */

const WELCOME_PROMPTS = [
  { title: "Deploy an agent", message: "Deploy an agent called my-assistant" },
  { title: "Show costs", message: "Show me the current platform costs" },
  { title: "Check system status", message: "What is the system health status?" },
  { title: "List skills", message: "List all available skills" },
  { title: "List models", message: "Show me all registered models" },
  { title: "View audit logs", message: "Show recent audit events" },
];

/* ---------- component ---------- */

export const PlatformChatPage = () => {
  const navigate = useNavigate();
  const { authHeaders } = useAuth();

  const [messages, setMessages] = useState<ConversationMessage[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [sessionId, setSessionId] = useState<string | null>(null);
  const [sessions, setSessions] = useState<ChatSession[]>([]);
  const [sessionsLoading, setSessionsLoading] = useState(true);
  const [isHistoryOpen, setIsHistoryOpen] = useState(true);
  const scrollRef = useRef<HTMLDivElement>(null);
  const abortRef = useRef<AbortController | null>(null);

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollIntoView({ behavior: "smooth" });
    }
  }, [messages, isLoading]);

  /* ----- load session list ----- */
  const loadSessions = useCallback(async () => {
    setSessionsLoading(true);
    try {
      const res = await fetch("/api/v1/chat/sessions", {
        headers: { ...authHeaders() },
      });
      if (res.ok) {
        const data = await res.json();
        const list: ChatSession[] = Array.isArray(data) ? data : data.sessions ?? [];
        setSessions(list);
      }
    } catch {
      /* best-effort */
    } finally {
      setSessionsLoading(false);
    }
  }, [authHeaders]);

  useEffect(() => {
    loadSessions();
  }, [loadSessions]);

  /* ----- load messages for a session ----- */
  const loadSessionMessages = useCallback(
    async (id: string) => {
      try {
        const res = await fetch(`/api/v1/chat/sessions/${id}/messages`, {
          headers: { ...authHeaders() },
        });
        if (!res.ok) return;
        const data = await res.json();
        const msgs: Array<{ role: string; content: string; timestamp?: string }> =
          Array.isArray(data) ? data : data.messages ?? [];
        setMessages(
          msgs.map((m, i) => ({
            id: `${id}-${i}`,
            role: m.role === "user" ? "user" : "bot",
            content: m.content,
            timestamp: m.timestamp ?? "",
          })),
        );
        setSessionId(id);
      } catch {
        /* best-effort */
      }
    },
    [authHeaders],
  );

  /* ----- new conversation ----- */
  const startNewConversation = useCallback(() => {
    if (abortRef.current) {
      abortRef.current.abort();
      abortRef.current = null;
    }
    setMessages([]);
    setSessionId(null);
    setIsLoading(false);
  }, []);

  /* ----- SSE streaming ----- */
  const streamAgentResponse = useCallback(
    async (text: string) => {
      const controller = new AbortController();
      abortRef.current = controller;

      const botId = `bot-${Date.now()}`;
      setMessages((prev) => [
        ...prev,
        { id: botId, role: "bot", content: "", timestamp: new Date().toLocaleTimeString() },
      ]);

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

        const contentType = res.headers.get("content-type") ?? "";
        if (contentType.includes("text/event-stream") && res.body) {
          const reader = res.body.getReader();
          const decoder = new TextDecoder();
          let accumulated = "";

          while (true) {
            const { done, value } = await reader.read();
            if (done) break;
            const chunk = decoder.decode(value, { stream: true });
            const lines = chunk.split("\n");
            for (const line of lines) {
              if (line.startsWith("data: ")) {
                const dataPayload = line.slice(6);
                if (dataPayload === "[DONE]") break;
                try {
                  const parsed = JSON.parse(dataPayload);
                  if (parsed.session_id) setSessionId(parsed.session_id);
                  const token = parsed.token ?? parsed.content ?? "";
                  if (token) {
                    accumulated += token;
                    setMessages((prev) =>
                      prev.map((m) => (m.id === botId ? { ...m, content: accumulated } : m)),
                    );
                  }
                } catch {
                  if (dataPayload.trim()) {
                    accumulated += dataPayload;
                    setMessages((prev) =>
                      prev.map((m) => (m.id === botId ? { ...m, content: accumulated } : m)),
                    );
                  }
                }
              }
            }
          }

          if (!accumulated) {
            setMessages((prev) =>
              prev.map((m) => (m.id === botId ? { ...m, content: "Done." } : m)),
            );
          }
        } else {
          const data = await res.json();
          if (data.session_id) setSessionId(data.session_id);
          setMessages((prev) =>
            prev.map((m) =>
              m.id === botId ? { ...m, content: data.reply ?? data.content ?? "Done." } : m,
            ),
          );
        }

        // Refresh sidebar after a conversation
        loadSessions();
      } catch (err) {
        if (err instanceof DOMException && err.name === "AbortError") return;
        setMessages((prev) =>
          prev.map((m) =>
            m.id === botId
              ? {
                  ...m,
                  content:
                    "Sorry, I couldn't reach the Arcana API. Is the platform running? Try `make dev` to start.",
                }
              : m,
          ),
        );
      }
    },
    [sessionId, authHeaders, loadSessions],
  );

  /* ----- send handler ----- */
  const handleSend = useCallback(
    async (message: string | number) => {
      const text = String(message);
      if (!text.trim() || isLoading) return;

      const userMsg: ConversationMessage = {
        id: `user-${Date.now()}`,
        role: "user",
        content: text,
        timestamp: new Date().toLocaleTimeString(),
      };
      setMessages((prev) => [...prev, userMsg]);
      setIsLoading(true);

      const action = detectCommand(text);
      if (action) {
        try {
          const result = await executePlatformCommand(action, text, authHeaders());
          setMessages((prev) => [
            ...prev,
            {
              id: `bot-${Date.now()}`,
              role: "bot",
              content: result.markdown,
              agentName: result.agentName,
              timestamp: new Date().toLocaleTimeString(),
            },
          ]);
        } catch {
          setMessages((prev) => [
            ...prev,
            {
              id: `err-${Date.now()}`,
              role: "bot",
              content: "❌ **Command failed.** Could not reach the Arcana API.",
              timestamp: new Date().toLocaleTimeString(),
            },
          ]);
        } finally {
          setIsLoading(false);
        }
      } else {
        try {
          await streamAgentResponse(text);
        } finally {
          setIsLoading(false);
        }
      }
    },
    [isLoading, authHeaders, streamAgentResponse],
  );

  /* ----- build conversation list for sidebar ----- */
  const conversations: Conversation[] = sessions.map((s) => ({
    id: s.id,
    text: s.title || `Session ${s.id.slice(0, 8)}`,
    onSelect: () => loadSessionMessages(s.id),
  }));

  /* ----- render ----- */
  return (
    <div className="platform-chat-page">
      <ChatbotConversationHistoryNav
        displayMode={ChatbotDisplayMode.fullscreen}
        onDrawerToggle={() => setIsHistoryOpen((prev) => !prev)}
        isDrawerOpen={isHistoryOpen}
        setIsDrawerOpen={setIsHistoryOpen}
        activeItemId={sessionId ?? undefined}
        onSelectActiveItem={(_e, itemId) => {
          if (typeof itemId === "string") {
            loadSessionMessages(itemId);
          }
        }}
        conversations={conversations}
        onNewChat={startNewConversation}
        newChatButtonText="New Conversation"
        searchInputPlaceholder="Search conversations..."
        searchInputAriaLabel="Search conversations"
        isLoading={sessionsLoading}
        drawerContent={
          <Chatbot displayMode={ChatbotDisplayMode.fullscreen}>
            <ChatbotHeader>
              <ChatbotHeaderMain>
                <ChatbotHeaderTitle>
                  <span
                    style={{
                      fontSize: 17,
                      fontWeight: 700,
                      display: "flex",
                      alignItems: "center",
                      gap: 10,
                    }}
                  >
                    <span
                      style={{
                        width: 32,
                        height: 32,
                        borderRadius: 8,
                        background:
                          "var(--arcana-gradient, linear-gradient(135deg, #667eea, #764ba2))",
                        display: "inline-flex",
                        alignItems: "center",
                        justifyContent: "center",
                        color: "#fff",
                        fontSize: 16,
                        fontWeight: 800,
                        flexShrink: 0,
                      }}
                    >
                      A
                    </span>
                    Arcana Chat
                  </span>
                </ChatbotHeaderTitle>
              </ChatbotHeaderMain>
              <ChatbotHeaderActions>
                <Button
                  variant="plain"
                  aria-label="New conversation"
                  onClick={startNewConversation}
                  icon={<PlusCircleIcon />}
                />
              </ChatbotHeaderActions>
            </ChatbotHeader>

            <ChatbotContent>
              <MessageBox>
                {messages.length === 0 && !sessionsLoading && (
                  <ChatbotWelcomePrompt
                    title="Hello! I'm Arcana."
                    description="Your platform command center. Deploy agents, check costs, view audit logs, and more — all in plain English."
                    prompts={WELCOME_PROMPTS.map((p) => ({
                      title: p.title,
                      message: p.message,
                      onClick: () => handleSend(p.message),
                    }))}
                  />
                )}

                {messages.length === 0 && sessionsLoading && (
                  <div
                    style={{
                      display: "flex",
                      alignItems: "center",
                      justifyContent: "center",
                      height: 200,
                    }}
                  >
                    <Spinner size="lg" />
                  </div>
                )}

                {messages.map((msg) => (
                  <div key={msg.id}>
                    <Message
                      role={msg.role}
                      content={msg.content}
                      name={msg.role === "user" ? "You" : "Arcana"}
                      timestamp={msg.timestamp}
                    />
                    {msg.role === "bot" && msg.agentName && (
                      <div style={{ marginLeft: 56, marginTop: -8, marginBottom: 12 }}>
                        <Button
                          variant="link"
                          onClick={() => navigate(`/agents/${msg.agentName}`)}
                          style={{
                            background:
                              "var(--arcana-gradient, linear-gradient(135deg, #667eea, #764ba2))",
                            color: "#fff",
                            border: "none",
                            borderRadius: 6,
                            padding: "6px 16px",
                            fontSize: 13,
                            fontWeight: 600,
                          }}
                        >
                          Open {msg.agentName} &rarr;
                        </Button>
                      </div>
                    )}
                  </div>
                ))}

                {isLoading && (
                  <Message role="bot" name="Arcana" content="Working on it..." isLoading />
                )}

                <div ref={scrollRef} />
              </MessageBox>
            </ChatbotContent>

            <ChatbotFooter>
              <MessageBar
                onSendMessage={handleSend}
                hasAttachButton={false}
                hasMicrophoneButton={false}
              />
              <ChatbotFootnote label="Arcana Chat — no data leaves the cluster." />
            </ChatbotFooter>
          </Chatbot>
        }
      />
    </div>
  );
};
