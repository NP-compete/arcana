# Architecture

Arcana is organized into five planes. Each plane owns one responsibility. Together, they cover the full lifecycle of an AI agent — from the YAML you write to the production infrastructure that keeps it running.

You interact with the top layer. Arcana manages everything underneath.

```
   Your agent YAML
        │
        ▼
┌─────────────────────────────────────────────────────────────────┐
│                         INTERACT PLANE                          │
│   Studio UI (:3000)  ·  AG-UI streaming (:8084)                 │
│   What users and developers see                                 │
├─────────────────────────────────────────────────────────────────┤
│                          AGENT PLANE                            │
│   Engine (:8081)  ·  Operator (:8082)  ·  Mesh (:8083)          │
│   Orchestration, reconciliation, inter-agent routing            │
├─────────────────────────────────────────────────────────────────┤
│                       DATA / TOOL PLANE                         │
│   Skills (:8085)  ·  RAG pipeline  ·  MCP  ·  Sandbox           │
│   What agents can use                                           │
├─────────────────────────────────────────────────────────────────┤
│                         GOVERN PLANE                            │
│   Ward (:8086)  ·  OPA  ·  KubeArmor                            │
│   What agents are allowed to do                                 │
├─────────────────────────────────────────────────────────────────┤
│                           OPS PLANE                             │
│   PostgreSQL  ·  Redis  ·  Temporal  ·  NATS  ·  MinIO           │
│   The boring infrastructure that makes it all work              │
└─────────────────────────────────────────────────────────────────┘
```

---

## What Each Plane Does For You

### Interact Plane — What Users See

| Service | Language | Port | What it does |
|---------|----------|------|-------------|
| **arcana-studio** | TypeScript | 3000 | Web UI built with React + PatternFly 6. Manage agents, view logs, monitor costs. |
| **arcana-agui** | Go | 8084 | AG-UI protocol server. Streams agent events to the browser via SSE in real time. |

When a user talks to an agent, Studio connects to the AG-UI server. AG-UI streams events (thinking, tool calls, responses) back to the UI as they happen. No polling, no websocket complexity — just SSE.

### Agent Plane — The Brains

| Service | Language | Port | What it does |
|---------|----------|------|-------------|
| **arcana-engine** | Go | 8081 | Runs agent workflows on LangGraph. Handles tool calls, memory retrieval, and response generation. |
| **arcana-operator** | Go | 8082 | Kubernetes operator. Watches Arcana CRDs and reconciles them into running infrastructure. |
| **arcana-mesh** | Go | 8083 | A2A + ACP mesh gateway. Routes messages between agents. Exposes agent cards for discovery. |

The engine is the runtime. The operator is the control plane. The mesh is the network.

When you `kubectl apply` an `ArcanaAgent`, the operator picks it up, configures the engine, registers the agent on the mesh, and wires up its skills, memory, and guardrails. You don't touch any of these services directly.

### Data / Tool Plane — What Agents Can Use

| Service | Language | Port | What it does |
|---------|----------|------|-------------|
| **arcana-skills** | Python | 8085 | Skill catalog with versioning. Agents invoke skills; skills invoke tools. |
| **codex-ingestor** | Go | — | Ingests documents into the RAG pipeline. |
| **codex-router** | Go | — | Routes queries to the right knowledge base. |
| **codex-searcher** | Go | — | Semantic search over vector embeddings. |
| **codex-scorer** | Go | — | Re-ranks search results for relevance. |
| **arcana-tools** | Go | — | MCP server integrations. Bridges external tools into the platform. |
| **arcana-sandbox** | Go | — | Executes untrusted code in gVisor/Kata isolated pods. |

Skills are the building blocks. A skill might call a GitHub API, run a static analysis tool, or search a knowledge base. Skills go through the guardrails pipeline before results reach the agent.

The RAG pipeline (codex-*) handles the full retrieval lifecycle: ingest documents, chunk and embed them, search semantically, and score results. All from an `ArcanaCodex` CRD.

### Govern Plane — What Agents Are Allowed To Do

| Service | Language | Port | What it does |
|---------|----------|------|-------------|
| **arcana-ward** | Python | 8086 | Guardrails pipeline. Filters agent inputs and outputs against safety rules. |
| **OPA** | — | — | Open Policy Agent. Enforces fine-grained policies on CRD admission. |
| **KubeArmor** | — | — | Runtime security enforcement. Locks down container behavior. |

Every agent interaction passes through Ward. Ward checks inputs against the `ArcanaGuardrail` rules before the agent sees them, and checks outputs before they reach the user. OPA handles cluster-level policy enforcement. KubeArmor prevents container escapes.

Budgets are enforced here too. The `ArcanaBudget` CRD sets token and compute spend limits. When an agent approaches its limit, it gets an alert. When it hits the limit, it stops.

### Ops Plane — The Infrastructure

| Service | Port(s) | What it does for you |
|---------|---------|---------------------|
| **arcana-api** | 8080 | REST/GraphQL gateway. Entry point for external clients. |
| **PostgreSQL + pgvector** | 5432 | Agent metadata, skill definitions, vector embeddings. |
| **Redis** | 6379 | Low-latency cache, session state, rate limiting. |
| **Temporal** | 7233 | Durable workflows — long-running agent tasks, retries, sagas. |
| **NATS JetStream** | 4222 | Event bus — async agent events, mesh pub/sub, cross-service messaging. |
| **MinIO** | 9000 | S3-compatible object storage — skill artifacts, model weights, eval datasets. |

You don't configure these individually. `make dev` starts all of them. In production, Helmfile deploys them with environment-appropriate settings.

---

## Agentic Protocols

Arcana integrates five protocols so agents can talk to tools, each other, and users without custom integration code.

| Protocol | Package | What it does |
|----------|---------|-------------|
| **MCP** (Model Context Protocol) | `pkg/mcp` | Agents invoke external tools and data sources through MCP servers. |
| **A2A** (Agent-to-Agent) | `pkg/a2a` | Agents discover each other via agent cards and exchange messages through the mesh. |
| **ACP** (Agent Communication Protocol) | `pkg/acp` | Bridges ACP-compatible agents into the Arcana mesh. |
| **AG-UI** (Agent-User Interface) | `pkg/agui` | Real-time streaming from agent to browser via SSE events. |
| **ACS** (Agent Control) | — | Lifecycle management — start, stop, cancel runs, manage sessions. |

```
  User / Studio
       │
       ▼ AG-UI (SSE)
  arcana-agui
       │
       ▼
  arcana-engine ◄──── A2A / ACP ────► arcana-mesh
       │                                    │
       ▼ MCP                                │
  External Tools                     Other Agents
       │
       ▼
  arcana-skills ──► arcana-ward (guardrails)
       │
       ▼ ACS (control)
  arcana-operator (CRD reconciliation)
```

---

## Custom Resource Definitions

Arcana ships 16 CRDs that cover the full agent lifecycle. The operator watches these and drives cluster state.

All CRD manifests live in `deploy/crds/`. Go types live in `pkg/crds/`.

| CRD | Short Name | Scope | What it does |
|-----|-----------|-------|-------------|
| **ArcanaAgent** | `aag` | Namespaced | Agent configuration — model, skills, memory, guardrails, sandbox |
| **ArcanaTenant** | `aten` | Cluster | Multi-tenant isolation — namespace mapping and resource quotas |
| **ArcanaModel** | `amod` | Namespaced | Model registry — provider, version, routing |
| **ArcanaSkillRegistry** | `askr` | Namespaced | Skill catalog with versioning |
| **ArcanaConnector** | `acon` | Namespaced | External data source connections |
| **ArcanaCodex** | `acdx` | Namespaced | RAG knowledge base configuration |
| **ArcanaDataset** | `adset` | Namespaced | Training and evaluation datasets |
| **ArcanaRole** | `arole` | Namespaced | RBAC + ABAC policies for agents and resources |
| **ArcanaGuardrail** | `agrd` | Namespaced | Input/output filtering and safety constraints |
| **ArcanaBudget** | `abud` | Namespaced | Token and compute spend limits with alert thresholds |
| **ArcanaEvalSuite** | `aes` | Namespaced | Automated eval pipelines — advisory / warn / block quality gates |
| **ArcanaExperiment** | `aexp` | Namespaced | A/B testing and canary experiments |
| **ArcanaPromotion** | `aprom` | Namespaced | Environment promotion: dev → staging → prod with approvals |
| **ArcanaBlueprint** | `abp` | Namespaced | Reusable agent templates |
| **ArcanaBackupPolicy** | `abkp` | Namespaced | Cron-based backups with retention |
| **ArcanaPlatform** | `aplat` | Cluster | Platform-wide configuration and defaults |

---

## Repository Layout

```
arcana/
├── cmd/                    # Go service entrypoints (engine, operator, mesh, api, agui, codex-*)
├── pkg/                    # Shared Go packages (mcp, a2a, acp, agui, crds, ward, temporal)
├── services/               # Non-Go services (skills, ward, studio, memory, forge, ...)
├── deploy/
│   ├── kind/               # Kind cluster config
│   ├── compose/            # Backing service Compose file
│   ├── crds/               # CRD manifests (source of truth)
│   ├── helm/               # One Helm chart per service (28 charts)
│   └── opa/                # OPA constraint templates
├── examples/               # Ready-to-deploy agent configurations
├── e2e/                    # Playwright end-to-end tests
├── docs/                   # You are here
├── hack/                   # Developer scripts
└── Makefile                # build, dev, test, lint, deploy targets
```
