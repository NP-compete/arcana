# Arcana Architecture Overview

Arcana is a Kubernetes-native AI platform for building, deploying, governing, and improving AI agents and ML models. The platform is organized into five architectural planes, eight Phase 1 services, five agentic protocols, eight CRDs, and a set of backing infrastructure services.

## Five-Plane Architecture

Arcana separates concerns across five planes. Each plane owns a distinct responsibility; together they form the full agent lifecycle from user interaction to durable operations.

| Plane | Responsibility | Components |
|-------|----------------|------------|
| **Interact** | User-facing interfaces and streaming agent output | Studio UI, AG-UI server |
| **Agent** | Orchestration, reconciliation, and inter-agent routing | Engine, Operator, Mesh |
| **Data / Tool** | Skills, tool access, and data pipelines | Skills engine, MCP integrations |
| **Govern** | Guardrails, policy enforcement, FinOps | Ward, OPA constraints |
| **Ops** | Durable workflows, messaging, persistence | Temporal, NATS, PostgreSQL, Redis, MinIO |

```
┌─────────────────────────────────────────────────────────────────────┐
│                         INTERACT PLANE                              │
│   arcana-studio (:3000)  ·  arcana-agui (:8084)                     │
├─────────────────────────────────────────────────────────────────────┤
│                          AGENT PLANE                                │
│   arcana-engine (:8081)  ·  arcana-operator (:8082)                 │
│   arcana-mesh (:8083)                                               │
├─────────────────────────────────────────────────────────────────────┤
│                       DATA / TOOL PLANE                             │
│   arcana-skills (:8085)  ·  MCP tool access                         │
├─────────────────────────────────────────────────────────────────────┤
│                         GOVERN PLANE                                │
│   arcana-ward (:8086)  ·  OPA + Gatekeeper constraints              │
├─────────────────────────────────────────────────────────────────────┤
│                           OPS PLANE                                 │
│   arcana-api (:8080)  ·  PostgreSQL  ·  Redis  ·  Temporal          │
│   MinIO  ·  NATS                                                    │
└─────────────────────────────────────────────────────────────────────┘
```

## Phase 1 Services

Eight services ship in Phase 1. Five are Go binaries under `cmd/`, two are Python FastAPI services, and one is a TypeScript React app.

| Service | Language | Port | Plane | Role |
|---------|----------|------|-------|------|
| **arcana-api** | Go | 8080 | Ops / Interact | REST/GraphQL API gateway; entry point for external clients |
| **arcana-engine** | Go | 8081 | Agent | Agent orchestration engine (LangGraph); runs agent workflows |
| **arcana-operator** | Go | 8082 | Agent | Kubernetes operator; reconciles Arcana CRDs into cluster state |
| **arcana-mesh** | Go | 8083 | Agent | A2A + ACP mesh gateway; routes agent-to-agent communication |
| **arcana-agui** | Go | 8084 | Interact | AG-UI protocol server; streams agent events to clients via SSE |
| **arcana-skills** | Python | 8085 | Data / Tool | Skill engine; catalog, versioning, and execution of agent skills |
| **arcana-ward** | Python | 8086 | Govern | Guardrails pipeline; input/output filtering and policy checks |
| **arcana-studio** | TypeScript | 3000 | Interact | Web UI (React + PatternFly 6); operator console for agents and resources |

## Agentic Protocols

Arcana integrates five agentic protocols. Shared Go types live in `pkg/`.

| Protocol | Package | Purpose |
|----------|---------|---------|
| **MCP** (Model Context Protocol) | `pkg/mcp` | Tool access — agents invoke external tools and data sources through MCP servers |
| **A2A** (Agent-to-Agent) | `pkg/a2a` | Primary agent-to-agent communication; mesh exposes agent cards and routes messages |
| **ACP** (Agent Communication Protocol) | `pkg/acp` | Agent-to-agent adapter; bridges ACP-compatible agents into the Arcana mesh |
| **AG-UI** (Agent-User Interface) | `pkg/agui` | Agent-to-user streaming; SSE event stream for real-time UI updates |
| **ACS** (Agent Control) | — | Agent control plane; lifecycle management, run cancellation, and session governance |

```
  User / Studio
       │
       ▼ AG-UI (SSE)
  arcana-agui ──────────────────────────────┐
       │                                    │
       ▼                                    ▼
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

## Custom Resource Definitions

All eight CRDs are declared in `deploy/crds/` with Go types in `pkg/crds/`.

| CRD | Short Name | Scope | Purpose |
|-----|-----------|-------|---------|
| **ArcanaAgent** | `aag` | Namespaced | Agent lifecycle and configuration — model, skills, memory, guardrails |
| **ArcanaTenant** | `aten` | Cluster | Multi-tenant isolation — namespace mapping and resource quotas |
| **ArcanaSkillRegistry** | `askr` | Namespaced | Skill catalog and versioning — registers skills available to agents |
| **ArcanaEvalSuite** | `aes` | Namespaced | Skill evaluation pipelines — automated quality gates (advisory/warn/block) |
| **ArcanaRole** | `arole` | Namespaced | RBAC + ABAC policies — permissions for agents, skills, and resources |
| **ArcanaBudget** | `abud` | Namespaced | FinOps budgets — token and compute spend limits with alert thresholds |
| **ArcanaPromotion** | `aprom` | Namespaced | Environment promotion — dev → staging → prod with approval gates |
| **ArcanaBackupPolicy** | `abkp` | Namespaced | Backup scheduling — cron-based backups with retention and destinations |

The operator watches these CRDs and drives the desired state of agents, policies, and infrastructure within the cluster.

## Backing Services

Local development runs backing services via `deploy/compose/docker-compose.yaml`. Each service supports a specific platform capability.

| Service | Port(s) | Why It Is Needed |
|---------|---------|------------------|
| **PostgreSQL + pgvector** | 5432 | Primary datastore — agent metadata, skill definitions, vector embeddings for semantic memory |
| **Redis** | 6379 | Low-latency cache and short-TTL agent memory; session state and rate limiting |
| **Temporal** | 7233 (UI: 8233) | Durable workflow orchestration — long-running agent tasks, retries, and saga patterns |
| **MinIO** | 9000 (UI: 9001) | S3-compatible object storage — skill artifacts, model weights, eval datasets, backups |
| **NATS** | 4222 (monitor: 8222) | Event bus with JetStream — async agent events, mesh pub/sub, and cross-service messaging |

```
                         ┌──────────────┐
                         │ arcana-api   │
                         └──────┬───────┘
                                │
         ┌──────────────────────┼──────────────────────┐
         │                      │                      │
         ▼                      ▼                      ▼
  ┌─────────────┐       ┌─────────────┐       ┌─────────────┐
  │ PostgreSQL  │       │    Redis    │       │   Temporal  │
  │  + pgvector │       │   (cache)   │       │ (workflows) │
  └─────────────┘       └─────────────┘       └─────────────┘
         │                      │                      │
         └──────────────────────┼──────────────────────┘
                                │
         ┌──────────────────────┼──────────────────────┐
         │                      │                      │
         ▼                      ▼                      ▼
  ┌─────────────┐       ┌─────────────┐       ┌─────────────┐
  │    MinIO    │       │    NATS     │       │    Kind     │
  │  (objects)  │       │  (events)   │       │  (K8s CRDs) │
  └─────────────┘       └─────────────┘       └─────────────┘
```

## Repository Layout

```
arcana/
├── cmd/                    # Go service entrypoints (engine, operator, mesh, api, agui)
├── pkg/                    # Shared Go packages (mcp, a2a, acp, agui, crds, ward, temporal)
├── services/               # Non-Go services (skills, ward, studio)
├── deploy/
│   ├── kind/               # Kind cluster config
│   ├── compose/            # Backing service Compose file
│   ├── crds/               # CRD manifests (source of truth)
│   ├── helm/               # One chart per service
│   └── opa/                # Gatekeeper constraint templates
├── hack/                   # Developer scripts (new-branch.sh, etc.)
├── test/                   # E2E and integration tests
└── Makefile                # build, dev, test, lint targets
```
