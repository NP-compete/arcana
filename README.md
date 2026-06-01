<div align="center">

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="https://raw.githubusercontent.com/NP-compete/arcana/main/docs/assets/logo-dark.svg">
  <source media="(prefers-color-scheme: light)" srcset="https://raw.githubusercontent.com/NP-compete/arcana/main/docs/assets/logo-light.svg">
  <img alt="Arcana" src="https://raw.githubusercontent.com/NP-compete/arcana/main/docs/assets/logo-light.svg" width="400">
</picture>

### The Heroku for AI Agents

Define your agent in YAML. Deploy with one command. Arcana handles orchestration, memory, guardrails, cost controls, multi-agent routing, and production infrastructure.

[![Build](https://github.com/NP-compete/arcana/actions/workflows/ci.yaml/badge.svg)](https://github.com/NP-compete/arcana/actions/workflows/ci.yaml)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Python](https://img.shields.io/badge/Python-3.11+-3776AB?logo=python&logoColor=white)](https://python.org)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-native-326CE5?logo=kubernetes&logoColor=white)](#how-it-works)
[![Protocols](https://img.shields.io/badge/Protocols-MCP%20%7C%20A2A%20%7C%20ACP%20%7C%20AG--UI-orange)](#protocol-support)

[Getting Started](#getting-started) · [How It Works](#how-it-works) · [Examples](#examples) · [Docs](docs/) · [Roadmap](ROADMAP.md) · [Contributing](CONTRIBUTING.md)

</div>

---

## Why Arcana?

Building an AI agent takes an afternoon. Getting it to production takes months.

You need orchestration, vector storage, guardrails, monitoring, cost controls, sandboxed execution, multi-agent routing, environment promotion, and a deployment pipeline to hold it all together. Each piece has its own API, its own lifecycle, and its own failure modes. You spend more time on infrastructure than on the agent itself.

**Arcana is the platform that makes all of that disappear.**

```yaml
apiVersion: arcana.io/v1alpha1
kind: ArcanaAgent
metadata:
  name: code-reviewer
spec:
  model: claude-sonnet-4-20250514
  skills: [github-pr-review, static-analysis, security-scan]
  memory:
    backend: pgvector
    ttl: 24h
  budget:
    maxTokensPerTurn: 8000
  sandbox:
    runtime: gvisor
```

```bash
kubectl apply -f code-reviewer.yaml
```

That's it. One file. One command. Your agent is running with memory, guardrails, cost controls, and sandboxed execution — managed by a Kubernetes operator, not a pile of shell scripts.

---

## What You Get

| You define | Arcana handles |
|-----------|---------------|
| Agent model and skills | Orchestration engine (LangGraph), lifecycle management, health checks |
| `memory: pgvector` | Vector storage, semantic search, TTL-based cleanup |
| `budget: maxTokens` | Real-time token tracking, spend alerts, automatic cutoff |
| `sandbox: gvisor` | gVisor/Kata isolation, network policies, read-only filesystem |
| `skills: [...]` | Skill catalog with versioning, MCP tool access, execution pipeline |
| Multi-agent team | A2A/ACP mesh routing, agent cards, inter-agent communication |
| Quality gates | Automated eval suites — advisory, warn, or block on quality drops |
| Environment promotion | `dev → staging → prod` with approval gates, one CRD |

Think of it as the difference between running your own servers and pushing to Heroku. Same power, fraction of the work.

---

## Getting Started

### Prerequisites

- [Kind](https://kind.sigs.k8s.io/) >= 0.20 (or any Kubernetes cluster)
- [Docker](https://docs.docker.com/get-docker/) or [Podman](https://podman.io/) >= 4.0
- Go >= 1.23, Python >= 3.11, Node.js >= 22

### 60-second setup

```bash
git clone https://github.com/NP-compete/arcana.git
cd arcana
make dev
```

This creates a local Kubernetes cluster, starts all backing services (PostgreSQL, Redis, Temporal, NATS, MinIO), installs 16 CRDs, and builds every service. One command.

```bash
make dev-status   # verify everything is healthy
```

### Deploy your first agent

```bash
kubectl apply -f examples/code-review-agent.yaml
kubectl get arcanaagents
```

Your agent is live. It has a model, skills, memory, budget limits, and a sandbox — all configured from that single YAML file.

> **Want a deeper walkthrough?** See the [Getting Started Guide](docs/getting-started.md) for a step-by-step tutorial.

---

## How It Works

Arcana organizes everything into five planes. You interact with the top layer. The platform manages the rest.

```
   You write YAML                    Arcana handles everything below
─────────────────────────────────────────────────────────────────────

   ┌─────────────────────────────────────────────────────────────┐
   │  INTERACT       Studio UI  ·  AG-UI streaming  ·  REST API │
   ├─────────────────────────────────────────────────────────────┤
   │  AGENT          Orchestration  ·  CRD operator  ·  Mesh    │
   ├─────────────────────────────────────────────────────────────┤
   │  DATA / TOOLS   Skills  ·  RAG pipeline  ·  MCP  ·  Sandbox│
   ├─────────────────────────────────────────────────────────────┤
   │  GOVERN         Guardrails  ·  OPA  ·  KubeArmor  ·  RBAC │
   ├─────────────────────────────────────────────────────────────┤
   │  OPS            PostgreSQL  ·  Redis  ·  Temporal  ·  NATS │
   └─────────────────────────────────────────────────────────────┘
```

**Interact** — The web UI (Studio), real-time streaming (AG-UI), and API gateway. What users and developers see.

**Agent** — The orchestration engine runs agents on LangGraph. The operator reconciles CRDs into cluster state. The mesh routes agent-to-agent communication via A2A and ACP protocols.

**Data / Tools** — Skills catalog with versioning. RAG pipeline (ingest, route, search, score). MCP tool integrations. Sandboxed code execution with gVisor.

**Govern** — Guardrails pipeline filters inputs and outputs. OPA enforces policies. KubeArmor locks down runtime behavior. Budgets track and cap spend.

**Ops** — The boring but critical stuff: durable workflows (Temporal), event streaming (NATS), persistence (PostgreSQL + pgvector), caching (Redis), object storage (MinIO).

> **Deep dive:** [Architecture documentation](docs/architecture.md) covers every service, protocol, and CRD in detail.

---

## Protocol Support

Arcana natively speaks five agentic protocols — so your agents can talk to tools, each other, and users without custom integration code.

| Protocol | What it does | Why it matters |
|----------|-------------|---------------|
| **MCP** | Tool access | Agents invoke external tools through a standard interface |
| **A2A** | Agent-to-agent | Agents discover and communicate with each other |
| **ACP** | Agent communication | Bridges ACP-compatible agents into the Arcana mesh |
| **AG-UI** | Agent-to-user | Real-time streaming from agent to UI via SSE |
| **ACS** | Agent control | Lifecycle management, run cancellation, session governance |

---

## 16 CRDs — Your Entire Agent Lifecycle in Kubernetes

Everything is a CRD. No external dashboards, no separate APIs, no config files scattered across services.

### Core
| CRD | Purpose |
|-----|---------|
| `ArcanaAgent` | Agent configuration — model, skills, memory, guardrails, sandbox |
| `ArcanaTenant` | Multi-tenant isolation with resource quotas |
| `ArcanaModel` | Model registry — provider, version, routing |

### Skills & Data
| CRD | Purpose |
|-----|---------|
| `ArcanaSkillRegistry` | Skill catalog with versioning |
| `ArcanaConnector` | External data source connections |
| `ArcanaCodex` | RAG knowledge base configuration |
| `ArcanaDataset` | Training and evaluation datasets |

### Governance
| CRD | Purpose |
|-----|---------|
| `ArcanaRole` | RBAC + ABAC policies |
| `ArcanaGuardrail` | Input/output filtering and safety rules |
| `ArcanaBudget` | Token and compute spend limits with alerts |

### Operations
| CRD | Purpose |
|-----|---------|
| `ArcanaEvalSuite` | Automated quality gates — advisory / warn / block |
| `ArcanaExperiment` | A/B testing and canary rollouts for agents |
| `ArcanaPromotion` | Environment promotion: dev → staging → prod |
| `ArcanaBlueprint` | Reusable agent templates |
| `ArcanaBackupPolicy` | Scheduled backups with retention |
| `ArcanaPlatform` | Platform-wide defaults and configuration |

---

## Examples

Ready-to-deploy configurations in [`examples/`](examples/):

| Example | What it shows |
|---------|--------------|
| [`code-review-agent.yaml`](examples/code-review-agent.yaml) | Single agent with GitHub PR skills, static analysis, and security scanning |
| [`rag-pipeline.yaml`](examples/rag-pipeline.yaml) | End-to-end RAG: document ingestion → semantic search → response generation |
| [`multi-agent-team.yaml`](examples/multi-agent-team.yaml) | Multi-agent team (planner + researcher + writer) with A2A mesh routing |
| [`budget-and-eval.yaml`](examples/budget-and-eval.yaml) | FinOps budget limits + automated evaluation suite with quality gates |

---

## How It Compares

| | Arcana | kagent | LangGraph Platform | HuggingFace Agents |
|-|--------|--------|--------------------|---------------------|
| **Deploy an agent** | `kubectl apply` one YAML | CRDs for agents + tools | Helm chart | No K8s support |
| **Multi-agent routing** | A2A + ACP mesh | A2A support | Supervisor pattern | Sequential only |
| **Protocols** | MCP, A2A, ACP, AG-UI, ACS | MCP, A2A | MCP (partial) | MCP (partial) |
| **Guardrails** | OPA + KubeArmor + Ward | None built-in | None built-in | None built-in |
| **Cost controls** | ArcanaBudget CRD | None built-in | Usage tracking | None built-in |
| **Eval & quality gates** | ArcanaEvalSuite | None built-in | LangSmith (separate) | None built-in |
| **Sandboxed execution** | gVisor / Kata per agent | None built-in | None built-in | None built-in |
| **Multi-tenancy** | ArcanaTenant CRD | Namespace-based | Not supported | N/A |
| **Env promotion** | `dev → staging → prod` CRD | None built-in | None built-in | N/A |
| **CRDs** | 16 | ~3 | 0 (Helm only) | 0 |

---

## Project Structure

```
arcana/
├── cmd/                    # 19 Go service entrypoints
│   ├── engine/             # Agent orchestration (LangGraph)
│   ├── operator/           # Kubernetes CRD controller
│   ├── mesh/               # A2A + ACP mesh gateway
│   ├── api/                # REST/GraphQL API gateway
│   ├── agui/               # AG-UI SSE streaming
│   ├── codex-*/            # RAG pipeline services
│   └── ...
├── pkg/                    # Shared Go packages
├── services/               # Non-Go services
│   ├── skills/             # Skill engine (Python/FastAPI)
│   ├── ward/               # Guardrails pipeline (Python/FastAPI)
│   ├── studio/             # Web UI (React + PatternFly 6)
│   └── ...
├── deploy/
│   ├── crds/               # 16 CRD manifests
│   ├── helm/               # Helm chart per service
│   ├── compose/            # Backing services (Compose)
│   └── kind/               # Local dev cluster config
├── examples/               # Ready-to-deploy agent configs
├── docs/                   # Architecture, deployment, security
├── e2e/                    # Playwright end-to-end tests
└── Makefile                # 25+ dev/build/test/deploy targets
```

## Quick Reference

| Command | What it does |
|---------|-------------|
| `make dev` | Full dev environment — Kind cluster + backing services + build + deploy |
| `make dev-down` | Tear everything down |
| `make dev-status` | Health check all services |
| `make build` | Build all services |
| `make test` | Run all tests (Go + Python + TypeScript) |
| `make lint` | Lint all code |
| `make crds-install` | Install CRDs into the cluster |
| `make docker-build` | Build container images |

---

## Documentation

| Doc | What's in it |
|-----|-------------|
| [Getting Started](docs/getting-started.md) | Step-by-step tutorial — first agent in 5 minutes |
| [Architecture](docs/architecture.md) | Five-plane architecture, services, protocols, CRDs |
| [Deployment](docs/deployment.md) | Helmfile-based deployment for dev/staging/prod |
| [Development](docs/development.md) | Local setup, adding services, adding CRDs |
| [Sandbox Security](docs/sandbox-security.md) | gVisor isolation, network policies, resource limits |
| [Secrets Management](docs/secrets-management.md) | External Secrets Operator, Vault integration |
| [TLS Setup](docs/tls-setup.md) | mTLS between services via cert-manager |
| [Troubleshooting](docs/troubleshooting.md) | Common issues and fixes |
| [Disaster Recovery](docs/runbooks/disaster-recovery.md) | Backup/restore procedures, failover |

---

## Roadmap

See [ROADMAP.md](ROADMAP.md) for what's coming next.

**Phase 1 (current):** Core platform — 8 services, 16 CRDs, 5 protocols, Studio UI.

**Phase 2:** CLI (`arcana deploy`), hosted control plane, marketplace for skills and blueprints.

**Phase 3:** Multi-cluster federation, edge deployment, managed offering.

---

## Contributing

We welcome contributions. See [CONTRIBUTING.md](CONTRIBUTING.md) for setup, branch naming, and PR conventions.

Look for issues labeled [`good-first-issue`](https://github.com/NP-compete/arcana/labels/good-first-issue) and [`help-wanted`](https://github.com/NP-compete/arcana/labels/help-wanted).

## Security

See [SECURITY.md](SECURITY.md) for vulnerability disclosure. Do not open public issues for security bugs.

## License

Apache 2.0 — see [LICENSE](LICENSE).
