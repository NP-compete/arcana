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
import {
  Button,
  Label,
  Spinner,
  Alert,
} from "@patternfly/react-core";
import {
  ArrowLeftIcon,
  InfoCircleIcon,
} from "@patternfly/react-icons";

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

function formatSteps(steps: ChatStep[]): string {
  if (!steps || steps.length === 0) return "";
  const lines = steps.map((s) => {
    const icon = s.type === "action" ? "\u2699\ufe0f" : s.type === "result" ? "\u2705" : "\u274c";
    return `${icon} **${s.service}** — ${s.message}`;
  });
  return "\n\n---\n" + lines.join("\n\n");
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
  const [isLoading, setIsLoading] = useState(false);
  const [sessionId, setSessionId] = useState<string | null>(null);
  const scrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    (async () => {
      try {
        const res = await fetch(`/api/v1/agents/${agentName}`);
        if (!res.ok) throw new Error(`Agent not found (${res.status})`);
        const data = await res.json();
        setAgentInfo(data);
      } catch (e) {
        setError(e instanceof Error ? e.message : "Failed to load agent");
      } finally {
        setLoading(false);
      }
    })();
  }, [agentName]);

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollIntoView({ behavior: "smooth" });
    }
  }, [messages, isLoading]);

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

        setMessages((prev) => [
          ...prev,
          {
            id: `bot-${Date.now()}`,
            role: "bot",
            content: reply,
            steps: data.steps,
            timestamp: new Date().toLocaleTimeString(),
          },
        ]);
      } catch {
        setMessages((prev) => [
          ...prev,
          {
            id: `err-${Date.now()}`,
            role: "bot",
            content: "Sorry, I couldn't reach the agent. Is the platform running?",
            timestamp: new Date().toLocaleTimeString(),
          },
        ]);
      } finally {
        setIsLoading(false);
      }
    },
    [isLoading, agentName, sessionId],
  );

  if (loading) {
    return (
      <div className="agent-chat-fullpage">
        <div style={{ display: "flex", alignItems: "center", justifyContent: "center", height: "100%" }}>
          <Spinner size="xl" />
        </div>
      </div>
    );
  }

  if (error || !agentInfo) {
    return (
      <div className="agent-chat-fullpage" style={{ padding: 32 }}>
        <Button variant="link" icon={<ArrowLeftIcon />} onClick={() => navigate("/agents")}>
          Back to Agents
        </Button>
        <Alert variant="danger" title="Agent not found" isInline style={{ marginTop: 16 }}>
          {error}
        </Alert>
      </div>
    );
  }

  const caps = agentInfo.capabilities?.join(", ") || "general";

  const prompts = [
    { title: "Check status", message: "What's your current status?" },
    { title: "Run a task", message: "Send the weekly newsletter to all subscribers" },
    { title: "Search knowledge", message: "Search for brand guidelines" },
    { title: "View costs", message: "How much have you spent so far?" },
    { title: "List MCP tools", message: "What tools are available to you?" },
    { title: "Guardrail check", message: "Is this content safe to send?" },
  ];

  return (
    <div className="agent-chat-fullpage">
      <Chatbot displayMode={ChatbotDisplayMode.default}>
        <ChatbotHeader>
          <ChatbotHeaderMain>
            <ChatbotHeaderTitle>
              <span style={{ display: "flex", alignItems: "center", gap: 10 }}>
                <button
                  onClick={() => navigate(`/agents/${agentName}`)}
                  className="agent-chat-back"
                  aria-label="Back to agent detail"
                >
                  <ArrowLeftIcon />
                </button>
                <span
                  style={{
                    width: 32,
                    height: 32,
                    borderRadius: 8,
                    background: "var(--arcana-gradient, linear-gradient(135deg, #667eea, #764ba2))",
                    display: "inline-flex",
                    alignItems: "center",
                    justifyContent: "center",
                    color: "#fff",
                    fontSize: 16,
                    fontWeight: 800,
                    flexShrink: 0,
                  }}
                >
                  {agentName[0]?.toUpperCase()}
                </span>
                <span style={{ fontSize: 17, fontWeight: 700 }}>{agentName}</span>
                <Label color="green" isCompact>{agentInfo.status}</Label>
                <Label color="blue" isCompact>{caps}</Label>
              </span>
            </ChatbotHeaderTitle>
          </ChatbotHeaderMain>
          <ChatbotHeaderActions>
            <Button
              variant="plain"
              aria-label="Agent details"
              icon={<InfoCircleIcon />}
              onClick={() => navigate(`/agents/${agentName}`)}
            />
          </ChatbotHeaderActions>
        </ChatbotHeader>

        <ChatbotContent>
          <MessageBox>
            {messages.length === 0 && (
              <ChatbotWelcomePrompt
                title={`Hello! I'm ${agentName}.`}
                description={`I'm an AI agent with ${caps} capabilities. Give me a task, ask me a question, or explore what I can do.`}
                prompts={prompts.map((p) => ({
                  title: p.title,
                  message: p.message,
                  onClick: () => handleSend(p.message),
                }))}
              />
            )}

            {messages.map((msg) => (
              <Message
                key={msg.id}
                role={msg.role}
                content={msg.content}
                name={msg.role === "user" ? "You" : agentName}
                timestamp={msg.timestamp}
              />
            ))}

            {isLoading && (
              <Message role="bot" name={agentName} content="Processing..." isLoading />
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
          <ChatbotFootnote label={`Talking to ${agentName} in namespace arcana-agent-${agentName} — all traffic stays in-cluster.`} />
        </ChatbotFooter>
      </Chatbot>
    </div>
  );
};
