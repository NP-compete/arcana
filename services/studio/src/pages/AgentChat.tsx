import { useCallback, useEffect, useRef, useState } from "react";
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
} from "@patternfly/chatbot/dist/dynamic/ChatbotHeader";
import { Label } from "@patternfly/react-core";

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

function formatSteps(steps: ChatStep[]): string {
  if (!steps || steps.length === 0) return "";
  const lines = steps.map((s) => {
    const icon = s.type === "action" ? "\u2699\ufe0f" : s.type === "result" ? "\u2705" : "\u274c";
    return `${icon} **${s.service}** — ${s.message}`;
  });
  return "\n\n---\n" + lines.join("\n\n");
}

interface AgentChatProps {
  agentName: string;
  capabilities: string[];
}

export const AgentChat = ({ agentName, capabilities }: AgentChatProps) => {
  const [messages, setMessages] = useState<ConversationMessage[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [sessionId, setSessionId] = useState<string | null>(null);
  const scrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollIntoView({ behavior: "smooth" });
    }
  }, [messages, isLoading]);

  const prompts = [
    { title: "Check status", message: "What's your current status?" },
    { title: "Run a task", message: `Run a task: process the latest data` },
    { title: "View costs", message: "How much have you spent so far?" },
    { title: "List tools", message: "What MCP tools are available?" },
  ];

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

        const botMsg: ConversationMessage = {
          id: `bot-${Date.now()}`,
          role: "bot",
          content: reply,
          steps: data.steps,
          timestamp: new Date().toLocaleTimeString(),
        };
        setMessages((prev) => [...prev, botMsg]);
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

  return (
    <div className="agent-chat-embedded">
      <Chatbot displayMode={ChatbotDisplayMode.default}>
        <ChatbotHeader>
          <ChatbotHeaderMain>
            <ChatbotHeaderTitle>
              <span style={{ fontSize: 15, fontWeight: 700, display: "flex", alignItems: "center", gap: 8 }}>
                <span
                  style={{
                    width: 24,
                    height: 24,
                    borderRadius: 6,
                    background: "var(--arcana-gradient, linear-gradient(135deg, #667eea, #764ba2))",
                    display: "inline-flex",
                    alignItems: "center",
                    justifyContent: "center",
                    color: "#fff",
                    fontSize: 12,
                    fontWeight: 800,
                  }}
                >
                  {agentName[0]?.toUpperCase()}
                </span>
                Chat with {agentName}
                <Label color="green" isCompact style={{ marginLeft: 4 }}>active</Label>
              </span>
            </ChatbotHeaderTitle>
          </ChatbotHeaderMain>
        </ChatbotHeader>

        <ChatbotContent>
          <MessageBox>
            {messages.length === 0 && (
              <ChatbotWelcomePrompt
                title={`Hello! I'm ${agentName}.`}
                description={`Capabilities: ${capabilities.join(", ") || "general"}. Ask me anything or give me a task.`}
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
          <ChatbotFootnote label={`Talking to ${agentName} — all traffic stays in-cluster.`} />
        </ChatbotFooter>
      </Chatbot>
    </div>
  );
};
