package main

import (
	"fmt"
	"strings"
)

func GenerateDeepAgentConfig(agent *MeshAgent) map[string]string {
	cfg := make(map[string]string)
	cfg["PROMPT.md"] = generatePrompt(agent)
	cfg["agent.yaml"] = generateAgentYAML(agent)
	cfg["mcp.json"] = generateMCPJSON(agent)
	cfg["ui.yaml"] = generateUIYAML(agent)
	return cfg
}

func generatePrompt(agent *MeshAgent) string {
	dc := agent.DeepConfig

	if dc != nil && dc.SystemPrompt != "" {
		return dc.SystemPrompt
	}

	caps := strings.Join(agent.Capabilities, ", ")
	skillsList := ""
	for _, c := range agent.Capabilities {
		skillsList += fmt.Sprintf("  - %s\n", c)
	}

	prompt := fmt.Sprintf(`---
name: %s
description: >
  Arcana-deployed deep agent with capabilities: %s.
  Handles orchestration, delegation, and autonomous task execution.
model: gpt-4o
skills:
%s---

# %s

Today's date is {{current_date}}.

## Identity

You are **%s**, a deep agent deployed on the Arcana platform.
Your capabilities: %s.

## Behavior

- Delegate complex work to subagents when available.
- Use MCP tools for external service access.
- Track all work with TODO lists before starting.
- Always respond using proper Markdown formatting.
- Only use the tools you are given. Ground answers in tool observations.
`, agent.Name, caps, skillsList, agent.Name, agent.Name, caps)

	if dc != nil {
		if dc.WorldModel {
			prompt += `
## World Model (Oracle)

Before executing a tool call, predict the likely outcome using retrieved
documentation and historical call results. If confidence > 0.85, use the
prediction (speculative execution). If confidence < 0.50, proceed with
the actual call.
`
		}
		if dc.SkillGraph {
			prompt += `
## Skill Graph

You have a 3-tier skill graph: planning skills → functional skills → atomic skills.
When encountering a new capability need, synthesize a skill inline, test it,
and register it for future use.
`
		}
		if dc.HITLEnabled {
			prompt += `
## Human-in-the-Loop

For high-impact actions (data mutations, external API calls with side effects,
budget-exceeding operations), pause and request human approval before proceeding.
`
		}
		if dc.SelfImprove {
			prompt += `
## Self-Improvement

After completing tasks, crystallize successful patterns into reusable skills.
Annotate tool call sequences that worked well. These annotations are
automatically refined into registered skills.
`
		}
		if len(dc.SubAgents) > 0 {
			prompt += `
## Sub-Agent Delegation

You are an orchestrator. Break complex tasks into sub-tasks and delegate
to specialized subagents. Never do work yourself that a subagent can handle.
Route to the correct subagent based on intent classification.
`
		}
	}

	return prompt
}

func generateAgentYAML(agent *MeshAgent) string {
	dc := agent.DeepConfig
	memEnabled := true
	skillsEnabled := true
	if dc != nil {
		skillsEnabled = dc.SkillGraph
	}

	memPolicy := "tri-scope"
	temperature := 0.0
	maxTokens := 8192
	modelCallLimit := 50
	toolCallLimit := 200

	if dc != nil {
		if dc.MemoryPolicy != "" {
			memPolicy = dc.MemoryPolicy
		}
		if dc.Temperature > 0 {
			temperature = dc.Temperature
		}
		if dc.MaxTokens > 0 {
			maxTokens = dc.MaxTokens
		}
		if dc.ModelCallLimit > 0 {
			modelCallLimit = dc.ModelCallLimit
		}
		if dc.ToolCallLimit > 0 {
			toolCallLimit = dc.ToolCallLimit
		}
	}

	// Generate subagent config sections if defined
	subagentSection := ""
	if dc != nil && len(dc.SubAgents) > 0 {
		subagentSection = "\nsubagents:\n"
		for _, sa := range dc.SubAgents {
			subagentSection += fmt.Sprintf("  - name: %s\n    enabled: true\n", sa)
		}
	}

	yaml := fmt.Sprintf(`name: "%s"

model:
  max_output_tokens: %d

resolve_strategy: legacy

providers:
  openai:
    init_kwargs:
      temperature: %.1f

middleware:
  summarization_tool:
    enabled: true
  memory:
    enabled: %v
    namespaces:
      - "memories"
    policy: "%s"
  patch_tool_calls:
    enabled: true
  skills:
    enabled: %v
  model_call_limit:
    enabled: true
    run_limit: %d
  tool_call_limit:
    enabled: true
    run_limit: %d
  model_retry:
    enabled: true
    max_retries: 3
    backoff_factor: 2.0
    initial_delay: 1.0
  pii:
    enabled: true
    rules:
      - type: credit_card
        strategy: mask
      - type: ip
        strategy: redact

filesystem:
  backend:
    type: state
  permissions:
    - operations: [read, glob, grep, ls]
      paths: ["config/**", "docs/**"]
      mode: allow
    - operations: [write, edit]
      paths: ["reports/**", "memories/**"]
      mode: allow

cache:
  enabled: true
  model:
    enabled: true
    ttl: 600
  personalization:
    ttl: 120
  graph:
    ttl: 300

memory:
  consolidation:
    enabled: %v
  decay:
    enabled: true
    lambda: 0.05
  clustering:
    enabled: true
    threshold: 0.4
  relationship_extraction:
    enabled: true
  scheduler:
    interval_hours: 6
  max_inject: 20
%s
logging:
  level: INFO

server:
  host: "0.0.0.0"
  port: 5002
`, agent.Name, maxTokens, temperature, memEnabled, memPolicy, skillsEnabled,
		modelCallLimit, toolCallLimit, memEnabled, subagentSection)

	return yaml
}

func generateUIYAML(agent *MeshAgent) string {
	return fmt.Sprintf(`title: "%s"
description: "Arcana deep agent: %s"
theme:
  primary_color: "#EE0000"
  logo: ""
features:
  feedback: true
  history: true
  streaming: true
`, agent.Name, agent.Name)
}

func generateMCPJSON(agent *MeshAgent) string {
	return fmt.Sprintf(`{
    "mcpServers": {
        "arcana-tools": {
            "url": "http://arcana-tools.arcana.svc.cluster.local:8096/mcp",
            "transport": "streamable_http",
            "enabled": true,
            "auth": false,
            "timeout": 30
        }
    }
}
`)
}
