# Arcana Roadmap

Arcana is evolving from a Kubernetes-native AI platform into a full **Agent Operating System** -- autonomous agents that work for users 24/7, not just respond to prompts. This roadmap captures the journey from what ships today to the fully realized vision.

---

## Phase 1: Core Platform -- Done

**Status:** Shipped (v0.1 -- June 2026)

The foundation is live. 15 Go services, 9 Python services, React dashboard (Studio), 16 CRDs, 28 Helm charts. Kubernetes-native. CRD-driven agent lifecycle. LangGraph orchestration. OPA guardrails. Multi-tenant.

| Capability | Status | Details |
|------------|--------|---------|
| CRD-driven agent lifecycle | Done | `ArcanaAgent`, `ArcanaSkillRegistry`, `ArcanaBudget`, `ArcanaTenant` and 12 more CRDs |
| LangGraph orchestration | Done | `arcana-engine` with multi-step agent execution |
| A2A + ACP agent mesh | Done | `arcana-mesh` with protocol-native routing |
| MCP tool integration | Done | `arcana-bridge` connects agents to external tool servers |
| OPA guardrail pipeline | Done | `arcana-ward` with policy-as-code enforcement |
| React dashboard (Studio) | Done | `arcana-studio` with PatternFly 6, agent management, conversation replay |
| AG-UI streaming | Done | `arcana-agui` with SSE streaming to frontends |
| RAG pipeline | Done | `arcana-indexer` + `arcana-retriever` with pgvector |
| Multi-tenant isolation | Done | `ArcanaTenant` CRD with resource quotas |
| Eval + promotion | Done | `ArcanaEvalSuite` and `ArcanaPromotion` CRDs |
| Helm deployment | Done | 28 charts with environment overlays |

---

## Phase 2: Developer Experience -- Planned (Q3 2026)

**Goal:** Go from `git clone` to running agents in under 60 seconds. Make Arcana the easiest agent platform to adopt by meeting developers where they already are -- their IDE, their existing OpenAI code, their laptop.

### 2.1 Single-Binary Local Mode

`arcana start` boots all services in one process for local development. No Kubernetes required. No Docker required. Just the binary.

- Single-process mode: all 24 services run in-process with goroutine isolation
- SQLite fallback when PostgreSQL is unavailable (automatic detection)
- Embedded dashboard: React build served from the binary at `localhost:7777`
- `docker-compose up` alternative for users who prefer containers (full platform in < 30 seconds)
- Feature flags control which services start (`--without ward,finops` to skip optional planes)

### 2.2 OpenAI-Compatible API

Drop-in replacement at `/v1/chat/completions` and `/v1/models`. Users point existing OpenAI tools, libraries, and scripts at Arcana with zero code changes.

- Streaming support via SSE (matching OpenAI's `stream: true` behavior)
- Model name routes to agents: `"model": "researcher"` routes to the researcher agent
- Standard models pass through to configured providers: `"model": "gpt-4o"` proxies to OpenAI
- Function calling and tool use mapped to Arcana skills
- Compatible with OpenAI Python SDK, LangChain, LlamaIndex, and any OpenAI-compatible client

### 2.3 MCP Server Mode

`arcana mcp` exposes agents as MCP tools for seamless IDE integration. Every agent becomes a tool that any MCP client can call.

- Agents callable from VS Code, Cursor, Claude Desktop, and any MCP-compatible client
- `arcana mcp` starts the MCP server on stdio (standard MCP transport)
- Agent parameters mapped to MCP tool input schemas
- Streaming responses forwarded as MCP tool output
- Works with both local mode and remote Arcana clusters

### 2.4 25+ LLM Provider Routing

Three native drivers (Anthropic, Gemini, OpenAI-compatible) covering 25+ providers out of the box. No plugins, no adapters -- just add an API key.

- **Native Anthropic driver:** Claude model family with native features (extended thinking, prompt caching, citations)
- **Native Gemini driver:** Gemini model family with native features (grounding, code execution)
- **OpenAI-compatible driver covering 20+ providers:** Groq, DeepSeek, Together, Mistral, Fireworks, Cohere, Perplexity, xAI, Cerebras, SambaNova, HuggingFace, Replicate, Ollama, vLLM, LM Studio, and any OpenAI-compatible endpoint
- **Task complexity scoring:** route simple tasks to cheap/fast models, complex reasoning to powerful models
- Automatic fallback chains with configurable priority
- Per-model cost tracking with real pricing data
- Alias support: `fast` -> `claude-haiku-4-5-20251001`, `smart` -> `claude-sonnet-4-20250514`
- Provider health detection: check API key presence and endpoint availability without leaking secrets
- Extend `ArcanaBudget` CRD with per-model cost limits

### 2.5 Unified CLI

Full command suite for every Arcana operation, no `kubectl` required for development workflows.

- `arcana init` -- interactive setup wizard (provider API keys, database, first agent)
- `arcana start` -- boot all services, dashboard live at localhost
- `arcana agent spawn researcher` -- create agent from template
- `arcana agent list` / `arcana agent chat researcher` -- interactive chat
- `arcana hand activate lead` / `arcana hand pause lead` -- Hand lifecycle
- `arcana workflow run my-pipeline` -- trigger workflow
- `arcana channel setup telegram` -- guided channel setup
- `arcana migrate --from langchain` -- import from other frameworks

**Files to modify:** `cmd/cli/`, new `cmd/standalone/`, `services/models/`, `cmd/api/`, new `pkg/providers/`, `deploy/compose/`

---

## Phase 3: Autonomous Agents -- Planned (Q3-Q4 2026)

**Goal:** Agents that work without prompting. Scheduled, continuous, event-driven. This is the transition from "chatbot framework" to "agent operating system."

### 3.1 Agent Scheduler

Cron-based scheduling, continuous mode, proactive triggers. Agents run autonomously on schedules without user prompting.

- Periodic mode: run daily at 6 AM, every 4 hours, etc.
- Continuous mode: agent loop that runs until killed (background worker pattern)
- Proactive triggers: run when an event pattern matches (webhook, queue message, file change)
- Oneshot mode: run once and terminate (batch job pattern)
- New CRD field -- `spec.schedule`:
  ```yaml
  spec:
    schedule:
      mode: periodic  # oneshot | periodic | continuous | proactive
      cron: "0 6 * * *"
      timezone: America/New_York
  ```
- Implement in `cmd/scheduler` service (currently a stub)

### 3.2 Hands -- Autonomous Capability Packages

Pre-built autonomous agents bundled with everything they need: system prompt, skills, tools, guardrails, and dashboard metrics. Each Hand ships as a self-contained package with a `HAND.toml` manifest.

- Define `HAND.toml` manifest format:
  ```toml
  [hand]
  name = "researcher"
  version = "1.0.0"
  description = "Deep autonomous researcher"

  [tools]
  required = ["web_search", "web_fetch", "file_write"]

  [settings]
  max_sources = 10
  output_format = "markdown"

  [dashboard]
  metrics = ["reports_generated", "sources_analyzed", "accuracy_score"]
  ```
- New CRD: `ArcanaHand` with lifecycle management (activate, pause, resume, status)
- Hand state persistence across restarts

**Initial Hands:**

| Hand | Purpose | Key Capabilities |
|------|---------|-----------------|
| **Researcher** | Deep autonomous research | Multi-source cross-referencing, credibility evaluation, cited reports with confidence scores |
| **Lead Generator** | Daily prospect discovery | ICP matching, enrichment from public data, scoring 0-100, automatic deduplication |
| **Collector** | OSINT-grade continuous monitoring | Change detection, sentiment tracking, knowledge graph construction, critical alerts |
| **Content Creator** | Multi-format content generation | Approval queues, scheduling, performance tracking, platform-native formatting |
| **Browser** | Web automation via Playwright bridge | Session persistence, visual page understanding, mandatory purchase approval gates |

### 3.3 Enhanced Workflow Engine

Step modes beyond sequential execution. Workflows become real programs with branching, parallelism, and iteration.

- **Step modes:** parallel, conditional, loop (in addition to existing sequential)
- **Output variables:** `output_var` for passing named outputs between steps
- **Loop termination:** `until` condition evaluated after each iteration
- **Error modes per step:** fail (halt workflow), skip (continue), retry (with `max_retries`)
- **Prompt templates** with variable interpolation: `{{input}}`, `{{step.research.output}}`
- **Workflow triggers:** manual, scheduled, event-driven, webhook
- Example:
  ```yaml
  steps:
    - name: research
      agent: researcher
      prompt: "Research {{input.topic}}"
      output_var: findings
    - name: review
      agent: reviewer
      prompt: "Verify these findings: {{step.research.output}}"
      error_mode: retry
      max_retries: 2
    - name: publish
      mode: conditional
      condition: "{{step.review.output.approved}} == true"
      agent: publisher
      prompt: "Publish: {{step.research.output}}"
  ```

**Files to modify:** `cmd/scheduler/`, `pkg/temporal/workflows.go`, new CRDs in `deploy/crds/`

---

## Phase 4: Channel Adapters -- Planned (Q4 2026)

**Goal:** Agents reachable on every platform users already use. Meet users where they are -- messaging apps, email, social platforms, enterprise tools.

### 4.1 Channel Adapter Framework

Define the `ChannelAdapter` interface in `pkg/channels/`:

```go
type ChannelAdapter interface {
    Start(ctx context.Context) error
    Stop() error
    SendMessage(channelID, text string) error
    OnMessage(handler func(msg IncomingMessage))
}
```

- Per-channel model overrides (use cheap model for Telegram, powerful for Slack)
- DM vs group policies per channel
- Per-user rate limiting
- Output formatting: Markdown -> Telegram HTML, Slack mrkdwn, plain text (automatic per-channel)
- Message splitting for platform character limits (4096 for Telegram, 40000 for Slack, 2000 for Discord)

### 4.2 Core Adapters (Priority 1)

| Adapter | Protocol | Priority |
|---------|----------|----------|
| **Telegram** | Bot API long-polling | P0 |
| **Slack** | Socket Mode WebSocket | P0 |
| **Discord** | Gateway WebSocket | P0 |
| **WhatsApp** | Cloud API webhook | P0 |
| **Email** | IMAP/SMTP | P1 |
| **Matrix** | Client-Server API | P1 |
| **Webhook** | Generic HTTP | P1 |

### 4.3 Enterprise Adapters (Priority 2)

Microsoft Teams, Mattermost, Google Chat, Webex, Feishu/Lark, Rocket.Chat, Zulip, XMPP

### 4.4 Social Adapters (Priority 3)

Mastodon, Bluesky, Reddit, LinkedIn, Twitch, LINE, Viber, Facebook Messenger

### 4.5 Agent Router

- Route incoming messages to the right agent based on channel, user, keywords
- Multi-agent channels: different agents for different topics in one Slack workspace
- Canonical sessions: same user across Telegram + Slack sees continuous memory (identity resolution)

**New service:** `cmd/channels/` with adapter registry, bridge manager, message router

---

## Phase 5: Scale -- Planned (Q4 2026 - Q1 2027)

**Goal:** Production-grade security hardening, advanced memory, observability, and the performance characteristics needed for enterprise deployment. Defense in depth with independently testable security layers.

### 5.1 Security Hardening

#### 5.1.1 WASM Dual-Metered Sandbox
- Run untrusted tool code in WebAssembly with fuel metering + epoch interruption
- Watchdog thread kills runaway code (dual metering: instruction count + wall clock)
- Complement to existing gVisor/Kata sandboxing -- WASM for tool plugins, gVisor for full agent sandboxing
- Integrate Wasmtime runtime into `cmd/sandbox`

#### 5.1.2 Merkle Hash-Chain Audit Trail
- Every agent action cryptographically linked to the previous one
- Tamper-evident: modify one entry and the chain breaks
- Extend `cmd/audit` service with hash-chain verification
- Store `entry_hash` and `prev_hash` in audit_log table (schema already has these fields)

#### 5.1.3 Information Flow Taint Tracking
- Label data as it flows through agents (PII, secrets, external-input)
- Track taint propagation from source to sink
- Prevent tainted data from reaching unauthorized outputs
- Add `TaintLabel` and `TaintSet` types to `pkg/common/`

#### 5.1.4 Ed25519 Signed Agent Manifests
- Ed25519 signing of agent definitions
- Verify agent identity and capability set have not been tampered with
- New field in `ArcanaAgent` CRD: `spec.manifest.signature`

#### 5.1.5 SSRF Protection
- Block requests to private IPs (RFC 1918, link-local), cloud metadata endpoints (169.254.169.254)
- DNS rebinding protection with re-resolution checks
- Add to `pkg/server/` middleware chain

#### 5.1.6 Secret Zeroization
- Auto-wipe API keys from memory when no longer needed
- Use Go's `memguard` or custom zero-on-drop wrapper
- Apply to all credential handling in `pkg/db/`, `cmd/mesh/k8s.go`

#### 5.1.7 Prompt Injection Scanner
- Detect override attempts ("ignore previous instructions" and variations)
- Data exfiltration pattern detection (encoded outputs, URL injection)
- Shell reference injection detection in skills
- Integrate into `services/ward/` guardrail pipeline

#### 5.1.8 Loop Guard
- SHA256-based tool call loop detection with circuit breaker
- Detect ping-pong patterns between agents (A calls B calls A)
- Configurable cycle threshold and cooldown period
- Add to `cmd/engine/` agent execution loop

#### 5.1.9 Session Repair
- Multi-phase message history validation (schema, ordering, reference integrity)
- Automatic recovery from corruption (truncate, replay, rebuild)
- Add to `cmd/api/` conversation session management

#### 5.1.10 Path Traversal Prevention
- Canonicalization of all file paths before access
- Symlink escape prevention (resolve then verify containment)
- Apply to skill file access, tool outputs, and any user-supplied paths

### 5.2 Advanced Memory

| Capability | Details |
|------------|---------|
| **Knowledge Graph Memory** | Entity and relation extraction from conversations. Cross-agent knowledge sharing. Temporal decay and relevance scoring. Production graph DB (Neo4j or Postgres ltree) replacing current in-memory NetworkX. |
| **Canonical Sessions** | Same user across multiple channels shares one memory. Identity resolution linking platform-specific user IDs to canonical identities. |
| **Memory Compaction** | LLM-based compaction preserving conversation structure (not just prefix dedup). Automatic archival of stale memories. Extend existing `dream_compact` endpoint with LLM integration. |
| **Vector Search at Scale** | pgvector or dedicated vector DB for 1M+ embeddings per agent. Hybrid search: vector similarity + keyword matching + graph traversal. |

### 5.3 Observability and FinOps

| Capability | Details |
|------------|---------|
| **Real-Time Agent Dashboard** | Live agent status, current task, token consumption. Conversation replay with step-by-step tool calls. Performance metrics: latency, success rate, cost per task. |
| **Cost-Aware Metering** | Per-agent, per-model, per-tenant cost tracking. Budget alerts and automatic throttling. Cost anomaly detection. |
| **Evaluation Pipeline** | Continuous eval: run quality checks on agent outputs. A/B testing: compare model versions on same inputs. Regression detection: alert when quality drops. |

**Files to modify:** `cmd/sandbox/`, `cmd/audit/`, `services/ward/`, `pkg/server/`, `pkg/common/`, `services/memory/`, `services/graph/`, `services/studio/`, `cmd/finops/`, `services/probe/`, new `pkg/security/`

---

## Phase 6: Desktop Application -- Planned (2027)

**Goal:** Native desktop app with system tray, notifications, and global shortcuts. Arcana running locally should feel like a native app, not a terminal process.

### 6.1 Native Desktop App

- System tray with quick access (show dashboard, agent status, quit)
- Desktop notifications for agent completions, alerts, and Hand reports
- Global keyboard shortcuts (summon chat overlay, quick-command palette)
- Single-instance enforcement (second launch focuses the existing window)
- Auto-updater with signed releases (Ed25519 signature verification)

### 6.2 Cross-Platform

- macOS, Windows, Linux
- Boots kernel in-process -- no separate daemon needed
- WebView pointing at embedded dashboard (same React build as local mode)
- Platform-native system tray integration and notification APIs

**New directory:** `desktop/` using Tauri 2.0 (Rust + WebView) or Wails (Go + WebView)

---

## Phase 7: Ecosystem -- Planned (2027)

**Goal:** A thriving ecosystem of community-contributed skills, Hands, and agents. Easy migration from competing frameworks. Cross-instance agent collaboration.

### 7.1 ArcanaHub Marketplace

Public marketplace for community skills, Hands, and agent templates.

- **Publish:** package and upload skills/Hands with `arcana publish`
- **Discover:** search and browse by category, rating, download count
- **Install:** `arcana install researcher-hand` with dependency resolution
- **Rate:** community ratings and reviews
- **Organization-private skills:** enterprise customers publish to private registries
- Skill verification with SHA256 checksums
- Versioning and dependency resolution
- Hot-reload: install skills without restarting agents

### 7.2 Migration Engine

One command to import agents, memory, and configs from other frameworks.

- `arcana migrate --from langchain` -- import LangChain agents and tools
- `arcana migrate --from crewai` -- import CrewAI crew definitions
- `arcana migrate --from autogen` -- import AutoGen agent configs
- Convert foreign skill formats to Arcana's `SKILL.md`
- Import memory stores, vector databases, and conversation history
- Validation report showing what mapped cleanly and what needs manual review

### 7.3 Peer-to-Peer Agent Networking

Wire protocol for cross-instance agent communication. Agents on different Arcana instances can discover and collaborate.

- HMAC-SHA256 mutual authentication between instances
- Agent discovery via DNS-SD or explicit peering configuration
- Cross-instance task delegation with result callbacks
- Bandwidth and request rate controls per peer
- Audit trail for all cross-instance interactions

**Files to modify:** `services/skills/`, new `cmd/marketplace/`, new `pkg/migrate/`, new `pkg/p2p/`

---

## Priority Summary

| Phase | Timeline | Impact | Effort |
|-------|----------|--------|--------|
| 1. Core Platform | Done | Foundation -- everything builds on this | -- |
| 2. Developer Experience | Q3 2026 | Critical -- developer adoption depends on DX | High |
| 3. Autonomous Agents | Q3-Q4 2026 | Critical -- without this, Arcana is a chatbot framework | High |
| 4. Channel Adapters | Q4 2026 | High -- agents need to reach users where they are | High |
| 5. Scale | Q4 2026 - Q1 2027 | Critical -- enterprise security, memory, and observability | High |
| 6. Desktop App | 2027 | Medium -- native experience for consumer and prosumer use | Medium |
| 7. Ecosystem | 2027 | High -- marketplace and migration drive adoption | High |

---

## Key Architectural Decisions

1. **Single binary vs microservices?** Arcana is currently 28 Kubernetes services. For developer adoption, a single-binary local mode is essential. Approach: embed all services in one Go binary with feature flags for local mode, keep microservices for production K8s deployment.

2. **Channel adapter language?** Go (consistent with backend) vs Python (faster to write, more SDK support for social platforms). Recommendation: Go for core adapters (Telegram, Slack, Discord), Python for long-tail social adapters via a sidecar pattern.

3. **WASM sandbox vs gVisor?** WASM is lighter (millisecond startup, KB memory) but limited to compiled languages. gVisor handles arbitrary containers. Approach: support both -- WASM for tool plugins, gVisor for full agent sandboxing.

4. **Desktop framework?** Tauri (Rust + WebView, 10MB) vs Electron (JS, 200MB) vs Wails (Go + WebView). Given Arcana's Go backend, Wails offers the tightest integration. Tauri is the fallback if WebView features are insufficient.

5. **Marketplace backend?** Self-hosted registry vs GitHub-based (like Homebrew taps). Recommendation: GitHub-based for v1, dedicated registry for v2 once volume justifies the infrastructure.

6. **P2P agent protocol?** gRPC vs custom wire protocol vs HTTP/2. Recommendation: gRPC with HMAC-SHA256 interceptor for mutual auth. Familiar tooling, strong streaming support, automatic code generation.

---

## How to Influence the Roadmap

This roadmap is a living document. If you have use cases, feature requests, or architectural feedback:

1. **Open a Discussion** -- use the [GitHub Discussions](https://github.com/NP-compete/arcana/discussions) board with the `roadmap` label
2. **File an Issue** -- for specific feature requests, open an issue with the `enhancement` label
3. **Submit a PR** -- for smaller features or improvements, PRs are welcome (see [CONTRIBUTING.md](../CONTRIBUTING.md))
4. **Join the Community** -- roadmap priorities are discussed in community calls (schedule in Discussions)

Phase priorities shift based on community demand. If enough users need a feature, it moves up.
