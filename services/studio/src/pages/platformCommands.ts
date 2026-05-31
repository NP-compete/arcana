/**
 * Platform command detection and execution for Arcana Chat.
 *
 * Routes natural-language messages to the correct API endpoint
 * and formats the response as markdown for the chatbot.
 */

export type PlatformAction =
  | "deploy_agent"
  | "list_agents"
  | "show_costs"
  | "show_audit"
  | "show_status"
  | "list_skills"
  | "list_models"
  | "suspend_agent"
  | "resume_agent";

interface CommandPattern {
  pattern: RegExp;
  action: PlatformAction;
}

const PLATFORM_COMMANDS: CommandPattern[] = [
  { pattern: /deploy|create.*agent/i, action: "deploy_agent" },
  { pattern: /list.*agents?|show.*agents?/i, action: "list_agents" },
  { pattern: /cost|spend|budget|finops/i, action: "show_costs" },
  { pattern: /audit|log|history/i, action: "show_audit" },
  { pattern: /status|health/i, action: "show_status" },
  { pattern: /skill|skills/i, action: "list_skills" },
  { pattern: /model|models/i, action: "list_models" },
  { pattern: /suspend|pause/i, action: "suspend_agent" },
  { pattern: /resume|wake/i, action: "resume_agent" },
];

export interface PlatformCommandResult {
  action: PlatformAction;
  markdown: string;
  /** Agent name extracted from the message, if applicable. */
  agentName?: string;
}

/**
 * Detect whether a message matches a known platform command.
 * Returns the first matching action or null.
 */
export function detectCommand(message: string): PlatformAction | null {
  for (const cmd of PLATFORM_COMMANDS) {
    if (cmd.pattern.test(message)) {
      return cmd.action;
    }
  }
  return null;
}

/**
 * Extract an agent name from a natural-language message.
 * Looks for patterns like "deploy <name>", "create agent <name>",
 * "suspend <name>", "resume <name>", or falls back to the last word.
 */
function extractAgentName(message: string): string {
  // "deploy an agent called foo-bar"
  const calledMatch = message.match(/(?:called|named)\s+([a-zA-Z0-9_-]+)/i);
  if (calledMatch) return calledMatch[1].toLowerCase();

  // "deploy foo-bar" / "create agent foo-bar"
  const deployMatch = message.match(
    /(?:deploy|create|suspend|pause|resume|wake)\s+(?:an?\s+)?(?:agent\s+)?([a-zA-Z0-9_-]+)/i,
  );
  if (deployMatch) return deployMatch[1].toLowerCase();

  // fallback: last word that looks like a valid name
  const words = message.trim().split(/\s+/);
  const last = words[words.length - 1];
  if (/^[a-zA-Z0-9_-]+$/.test(last)) return last.toLowerCase();

  return "unnamed-agent";
}

/**
 * Execute a platform command against the Arcana API and return formatted markdown.
 */
export async function executePlatformCommand(
  action: PlatformAction,
  message: string,
  authHeaders: Record<string, string>,
): Promise<PlatformCommandResult> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...authHeaders,
  };

  switch (action) {
    case "deploy_agent": {
      const agentName = extractAgentName(message);
      try {
        const res = await fetch("/api/v1/agents/register", {
          method: "POST",
          headers,
          body: JSON.stringify({ name: agentName }),
        });
        if (!res.ok) {
          const err = await res.text().catch(() => "Unknown error");
          return {
            action,
            agentName,
            markdown: `❌ **Deploy failed** for \`${agentName}\`:\n\n\`\`\`\n${err}\n\`\`\``,
          };
        }
        return {
          action,
          agentName,
          markdown:
            `✅ **Agent \`${agentName}\` deployed successfully.**\n\n` +
            `The agent has been registered and is starting up in namespace \`arcana-agent-${agentName}\`.`,
        };
      } catch {
        return {
          action,
          agentName,
          markdown: `❌ **Failed to reach the API.** Is the platform running?`,
        };
      }
    }

    case "list_agents": {
      try {
        const res = await fetch("/api/v1/agents", { headers });
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const data = await res.json();
        const agents: Array<{ name: string; status?: string }> =
          Array.isArray(data) ? data : data.agents ?? [];
        if (agents.length === 0) {
          return {
            action,
            markdown: "\U0001F4CA **No agents registered.** Deploy one to get started.",
          };
        }
        const rows = agents
          .map((a) => {
            const badge = a.status === "running" ? "\U0001F7E2" : a.status === "stopped" ? "\U0001F534" : "\U0001F7E1";
            return `| ${badge} | \`${a.name}\` | ${a.status ?? "unknown"} |`;
          })
          .join("\n");
        return {
          action,
          markdown:
            `\U0001F4CA **${agents.length} agent(s) registered:**\n\n` +
            `| | Name | Status |\n|---|---|---|\n${rows}`,
        };
      } catch {
        return { action, markdown: "❌ **Could not fetch agents.** Is the API reachable?" };
      }
    }

    case "show_costs": {
      try {
        const res = await fetch("/api/v1/finops/summary", { headers });
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const data = await res.json();
        const total = data.total_cost ?? data.totalCost ?? "N/A";
        const period = data.period ?? "current period";
        return {
          action,
          markdown:
            `\U0001F4CA **Cost Summary (${period})**\n\n` +
            `| Metric | Value |\n|---|---|\n` +
            `| Total Cost | $${total} |\n` +
            `| Token Usage | ${data.token_usage ?? data.tokenUsage ?? "N/A"} |\n` +
            `| Active Agents | ${data.active_agents ?? data.activeAgents ?? "N/A"} |\n\n` +
            `View full breakdown in **FinOps**.`,
        };
      } catch {
        return { action, markdown: "❌ **Could not fetch cost data.** Is the FinOps service running?" };
      }
    }

    case "show_audit": {
      try {
        const res = await fetch("/api/v1/enterprise/audit?limit=5", { headers });
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const data = await res.json();
        const events: Array<{ action?: string; user?: string; timestamp?: string; detail?: string }> =
          Array.isArray(data) ? data : data.events ?? [];
        if (events.length === 0) {
          return { action, markdown: "\U0001F4CA **No audit events found.**" };
        }
        const rows = events
          .map(
            (e) =>
              `| ${e.timestamp ?? "-"} | ${e.user ?? "-"} | ${e.action ?? "-"} | ${e.detail ?? "-"} |`,
          )
          .join("\n");
        return {
          action,
          markdown:
            `\U0001F4CA **Recent Audit Events:**\n\n` +
            `| Time | User | Action | Detail |\n|---|---|---|---|\n${rows}`,
        };
      } catch {
        return { action, markdown: "❌ **Could not fetch audit logs.**" };
      }
    }

    case "show_status": {
      try {
        const res = await fetch("/api/v1/health", { headers });
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const data = await res.json();
        const services: Array<{ name: string; status: string }> =
          data.services ?? [];
        if (services.length === 0) {
          const overall = data.status ?? "unknown";
          return {
            action,
            markdown: `✅ **Platform status: ${overall}**`,
          };
        }
        const rows = services
          .map((s) => {
            const icon = s.status === "healthy" ? "\U0001F7E2" : s.status === "degraded" ? "\U0001F7E1" : "\U0001F534";
            return `| ${icon} | ${s.name} | ${s.status} |`;
          })
          .join("\n");
        return {
          action,
          markdown:
            `✅ **Platform Health:**\n\n` +
            `| | Service | Status |\n|---|---|---|\n${rows}`,
        };
      } catch {
        return { action, markdown: "❌ **Could not reach the health endpoint.**" };
      }
    }

    case "list_skills": {
      try {
        const res = await fetch("/api/v1/skills", { headers });
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const data = await res.json();
        const skills: Array<{ name: string; description?: string }> =
          Array.isArray(data) ? data : data.skills ?? [];
        if (skills.length === 0) {
          return { action, markdown: "\U0001F4CA **No skills registered.**" };
        }
        const rows = skills
          .map((s) => `| \`${s.name}\` | ${s.description ?? "-"} |`)
          .join("\n");
        return {
          action,
          markdown:
            `\U0001F4CA **${skills.length} skill(s) available:**\n\n` +
            `| Name | Description |\n|---|---|\n${rows}`,
        };
      } catch {
        return { action, markdown: "❌ **Could not fetch skills.**" };
      }
    }

    case "list_models": {
      try {
        const res = await fetch("/api/v1/models", { headers });
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const data = await res.json();
        const models: Array<{ name: string; provider?: string; status?: string }> =
          Array.isArray(data) ? data : data.models ?? [];
        if (models.length === 0) {
          return { action, markdown: "\U0001F4CA **No models registered.**" };
        }
        const rows = models
          .map(
            (m) =>
              `| \`${m.name}\` | ${m.provider ?? "-"} | ${m.status ?? "-"} |`,
          )
          .join("\n");
        return {
          action,
          markdown:
            `\U0001F4CA **${models.length} model(s) available:**\n\n` +
            `| Name | Provider | Status |\n|---|---|---|\n${rows}`,
        };
      } catch {
        return { action, markdown: "❌ **Could not fetch models.**" };
      }
    }

    case "suspend_agent": {
      const agentName = extractAgentName(message);
      try {
        const res = await fetch(`/api/v1/agents/${agentName}/suspend`, {
          method: "POST",
          headers,
        });
        if (!res.ok) {
          return {
            action,
            agentName,
            markdown: `❌ **Failed to suspend \`${agentName}\`.** (HTTP ${res.status})`,
          };
        }
        return {
          action,
          agentName,
          markdown: `✅ **Agent \`${agentName}\` has been suspended.**`,
        };
      } catch {
        return { action, agentName, markdown: "❌ **Could not reach the API.**" };
      }
    }

    case "resume_agent": {
      const agentName = extractAgentName(message);
      try {
        const res = await fetch(`/api/v1/agents/${agentName}/resume`, {
          method: "POST",
          headers,
        });
        if (!res.ok) {
          return {
            action,
            agentName,
            markdown: `❌ **Failed to resume \`${agentName}\`.** (HTTP ${res.status})`,
          };
        }
        return {
          action,
          agentName,
          markdown: `✅ **Agent \`${agentName}\` has been resumed.**`,
        };
      } catch {
        return { action, agentName, markdown: "❌ **Could not reach the API.**" };
      }
    }
  }
}
