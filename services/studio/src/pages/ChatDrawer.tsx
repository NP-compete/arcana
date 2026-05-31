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
import { Button } from "@patternfly/react-core";
import { TimesIcon } from "@patternfly/react-icons";
import { useAuth } from "../auth/AuthContext";
import {
  detectCommand,
  executePlatformCommand,
} from "./platformCommands";

interface ConversationMessage {
  id: string;
  role: "user" | "bot";
  content: string;
  /** Agent name produced by a deploy/suspend/resume command, for the nav button. */
  agentName?: string;
  timestamp: string;
}

const WELCOME_PROMPTS = [
  {
    title: "Deploy an agent",
    message: "Deploy an agent called my-assistant",
  },
  {
    title: "Show costs",
    message: "Show me the current platform costs",
  },
  {
    title: "Check system status",
    message: "What is the system health status?",
  },
  {
    title: "List skills",
    message: "List all available skills",
  },
];

interface ChatDrawerProps {
  isOpen: boolean;
  onClose: () => void;
}

export const ChatDrawer = ({ isOpen, onClose }: ChatDrawerProps) => {
  const navigate = useNavigate();
  const { authHeaders } = useAuth();
  const [messages, setMessages] = useState<ConversationMessage[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [sessionId, setSessionId] = useState<string | null>(null);
  const scrollToBottomRef = useRef<HTMLDivElement>(null);
  const abortRef = useRef<AbortController | null>(null);

  useEffect(() => {
    if (scrollToBottomRef.current) {
      scrollToBottomRef.current.scrollIntoView({ behavior: "smooth" });
    }
  }, [messages, isLoading]);

  // Abort any in-flight SSE stream when the drawer closes.
  useEffect(() => {
    if (!isOpen && abortRef.current) {
      abortRef.current.abort();
      abortRef.current = null;
    }
  }, [isOpen]);

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

        if (!res.ok) {
          throw new Error(`HTTP ${res.status}`);
        }

        const contentType = res.headers.get("content-type") ?? "";
        if (contentType.includes("text/event-stream") && res.body) {
          const reader = res.body.getReader();
          const decoder = new TextDecoder();
          let accumulated = "";

          while (true) {
            const { done, value } = await reader.read();
            if (done) break;
            const chunk = decoder.decode(value, { stream: true });

            // Parse SSE lines
            const lines = chunk.split("\n");
            for (const line of lines) {
              if (line.startsWith("data: ")) {
                const payload = line.slice(6);
                if (payload === "[DONE]") break;
                try {
                  const parsed = JSON.parse(payload);
                  if (parsed.session_id) setSessionId(parsed.session_id);
                  if (parsed.token) {
                    accumulated += parsed.token;
                    setMessages((prev) =>
                      prev.map((m) => (m.id === botId ? { ...m, content: accumulated } : m)),
                    );
                  }
                  if (parsed.content) {
                    accumulated += parsed.content;
                    setMessages((prev) =>
                      prev.map((m) => (m.id === botId ? { ...m, content: accumulated } : m)),
                    );
                  }
                } catch {
                  // Non-JSON SSE line, treat as plain text token
                  if (payload.trim()) {
                    accumulated += payload;
                    setMessages((prev) =>
                      prev.map((m) => (m.id === botId ? { ...m, content: accumulated } : m)),
                    );
                  }
                }
              }
            }
          }

          // If nothing was streamed, show a fallback
          if (!accumulated) {
            setMessages((prev) =>
              prev.map((m) => (m.id === botId ? { ...m, content: "Done." } : m)),
            );
          }
        } else {
          // Non-streaming JSON fallback
          const data = await res.json();
          if (data.session_id) setSessionId(data.session_id);
          setMessages((prev) =>
            prev.map((m) =>
              m.id === botId ? { ...m, content: data.reply ?? data.content ?? "Done." } : m,
            ),
          );
        }
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
    [sessionId, authHeaders],
  );

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
          const botMsg: ConversationMessage = {
            id: `bot-${Date.now()}`,
            role: "bot",
            content: result.markdown,
            agentName: result.agentName,
            timestamp: new Date().toLocaleTimeString(),
          };
          setMessages((prev) => [...prev, botMsg]);
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
        // Not a platform command — stream via AG-UI SSE endpoint
        try {
          await streamAgentResponse(text);
        } finally {
          setIsLoading(false);
        }
      }
    },
    [isLoading, authHeaders, streamAgentResponse],
  );

  if (!isOpen) return null;

  return (
    <div className="arcana-chat-overlay">
      <Chatbot displayMode={ChatbotDisplayMode.default}>
        <ChatbotHeader>
          <ChatbotHeaderMain>
            <ChatbotHeaderTitle>
              <span
                style={{
                  fontSize: 16,
                  fontWeight: 700,
                  display: "flex",
                  alignItems: "center",
                  gap: 8,
                }}
              >
                <span
                  style={{
                    width: 28,
                    height: 28,
                    borderRadius: 8,
                    background: "var(--arcana-gradient, linear-gradient(135deg, #667eea, #764ba2))",
                    display: "inline-flex",
                    alignItems: "center",
                    justifyContent: "center",
                    color: "#fff",
                    fontSize: 14,
                    fontWeight: 800,
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
              aria-label="Close chat"
              onClick={onClose}
              icon={<TimesIcon />}
            />
          </ChatbotHeaderActions>
        </ChatbotHeader>

        <ChatbotContent>
          <MessageBox>
            {messages.length === 0 && (
              <ChatbotWelcomePrompt
                title="Hello! I'm Arcana."
                description="Tell me what you need in plain English — deploy agents, check costs, view status, list skills and models."
                prompts={WELCOME_PROMPTS.map((p) => ({
                  title: p.title,
                  message: p.message,
                  onClick: () => handleSend(p.message),
                }))}
              />
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
                      onClick={() => {
                        onClose();
                        navigate(`/agents/${msg.agentName}`);
                      }}
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
              <Message
                role="bot"
                name="Arcana"
                content="Working on it..."
                isLoading
              />
            )}

            <div ref={scrollToBottomRef} />
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
    </div>
  );
};
