# Roadmap

Arcana is building the deployment platform for AI agents. Here's where we're headed.

## Phase 1 — Core Platform (current)

The foundation. Ship the platform that makes deploying AI agents as simple as deploying a web app.

| Area | Status | Description |
|------|--------|-------------|
| **Kubernetes Operator** | Done | Reconciles 16 CRDs into running infrastructure |
| **Agent Engine** | Done | LangGraph-based orchestration with tool calls, memory, and streaming |
| **A2A/ACP Mesh** | Done | Agent-to-agent discovery and routing |
| **Skill Engine** | Done | Skill catalog with versioning and MCP tool access |
| **Guardrails (Ward)** | Done | Input/output filtering pipeline with policy enforcement |
| **Studio UI** | Done | React + PatternFly 6 operator console |
| **AG-UI Streaming** | Done | Real-time agent-to-user SSE streaming |
| **API Gateway** | Done | REST/GraphQL entry point |
| **RAG Pipeline** | Done | Ingest → embed → search → score pipeline (codex-*) |
| **Sandbox Execution** | Done | gVisor/Kata isolated code execution per agent |
| **FinOps Budgets** | Done | Token and compute spend limits via CRD |
| **Eval Suites** | Done | Automated quality gates — advisory / warn / block |
| **Environment Promotion** | Done | dev → staging → prod with approval gates |
| **Secrets Management** | Done | External Secrets Operator + Vault integration |
| **mTLS** | Done | Service-to-service TLS via cert-manager |
| **Alerting** | Done | Prometheus alert rules + AlertManager |

## Phase 2 — Developer Experience

Make Arcana feel less like infrastructure and more like a product.

| Area | Status | Description |
|------|--------|-------------|
| **`arcana` CLI** | Planned | `arcana deploy`, `arcana logs`, `arcana status` — no kubectl needed |
| **Agent Marketplace** | Planned | Share and discover agent blueprints and skill packs |
| **One-click Templates** | Planned | "Deploy a customer support agent" with zero YAML |
| **GitOps Integration** | Planned | Push to main → agent deploys automatically |
| **Hosted Control Plane** | Planned | Arcana Cloud — managed platform, bring your own cluster |
| **Dashboard Improvements** | Planned | Cost analytics, agent performance trends, team views |
| **Plugin SDK** | Planned | Build custom skills with a typed SDK (Go + Python) |
| **LLM Observability** | Planned | Trace every token through the system — prompts, completions, latency |

## Phase 3 — Scale

Production at scale for teams and enterprises.

| Area | Status | Description |
|------|--------|-------------|
| **Multi-Cluster Federation** | Planned | Deploy agents across multiple clusters with unified management |
| **Edge Deployment** | Planned | Run agents on edge nodes for low-latency use cases |
| **Managed Offering** | Planned | Fully managed Arcana — no cluster to operate |
| **Enterprise SSO** | Planned | SAML/OIDC integration for team management |
| **Audit & Compliance** | Planned | SOC 2 / HIPAA compliance toolkit |
| **Custom Model Hosting** | Planned | Deploy and serve fine-tuned models alongside agents |
| **Agent Analytics** | Planned | User satisfaction scores, conversation quality metrics |

---

## How to Influence the Roadmap

- **Upvote existing issues** — the most-requested features get prioritized
- **Open a feature request** — [use the template](https://github.com/NP-compete/arcana/issues/new?template=feature_request.yml)
- **Start a discussion** — [GitHub Discussions](https://github.com/NP-compete/arcana/discussions)
- **Contribute** — the best way to get a feature shipped is to build it. See [CONTRIBUTING.md](CONTRIBUTING.md).
