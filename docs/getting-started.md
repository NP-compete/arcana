# Getting Started

This guide walks you from zero to a running AI agent in about 5 minutes.

## Prerequisites

You need these installed:

| Tool | Version | Install |
|------|---------|---------|
| Go | 1.23+ | [go.dev/dl](https://go.dev/dl/) |
| Python | 3.11+ | [python.org](https://python.org) |
| Node.js | 22+ | [nodejs.org](https://nodejs.org) |
| Kind | 0.20+ | [kind.sigs.k8s.io](https://kind.sigs.k8s.io/) |
| Docker or Podman | 4.0+ | [docker.com](https://docs.docker.com/get-docker/) or [podman.io](https://podman.io/) |
| kubectl | latest | [kubernetes.io](https://kubernetes.io/docs/tasks/tools/) |

## Step 1: Start the Platform

```bash
git clone https://github.com/NP-compete/arcana.git
cd arcana
make dev
```

This does everything:
- Creates a local Kubernetes cluster (`arcana-dev`) via Kind
- Starts backing services (PostgreSQL, Redis, Temporal, NATS, MinIO) via Compose
- Installs all 16 Arcana CRDs
- Builds every service (Go, Python, TypeScript)

Takes 2-3 minutes on first run. Subsequent runs are faster.

Verify everything is healthy:

```bash
make dev-status
```

Set your kubeconfig (printed by `make dev`):

```bash
export KUBECONFIG=$(pwd)/kubeconfig-arcana-dev
```

## Step 2: Deploy Your First Agent

Arcana ships with example configurations. Let's deploy a code review agent:

```bash
kubectl apply -f examples/code-review-agent.yaml
```

Check the agent status:

```bash
kubectl get arcanaagents
```

```
NAME            MODEL                    STATUS   AGE
code-reviewer   claude-sonnet-4-20250514   Ready    12s
```

That's it. The operator picked up your YAML, configured the engine, registered the agent on the mesh, and wired up its skills, memory, and guardrails.

## Step 3: Explore What You Deployed

Let's look at what the agent YAML actually created:

```bash
kubectl describe arcanaagent code-reviewer
```

The agent has:
- **Model**: `claude-sonnet-4-20250514` — which LLM powers it
- **Skills**: `github-pr-review`, `static-analysis`, `security-scan` — what it can do
- **Memory**: pgvector backend with 24h TTL — conversation history with semantic search
- **Budget**: 8000 max tokens per turn — automatic cost controls
- **Sandbox**: gVisor runtime — isolated code execution

All of this was set up from one YAML file. No manual configuration, no connecting services, no deployment scripts.

## Step 4: Try More Examples

### RAG Pipeline

Deploy a complete retrieval-augmented generation pipeline:

```bash
kubectl apply -f examples/rag-pipeline.yaml
```

This creates an `ArcanaCodex` (knowledge base) with document ingestion, vector embeddings, and semantic search — all from a CRD.

### Multi-Agent Team

Deploy a team of agents that collaborate:

```bash
kubectl apply -f examples/multi-agent-team.yaml
```

This creates three agents (planner, researcher, writer) connected via the A2A mesh. They discover each other through agent cards and route messages automatically.

### Budget + Eval Gates

Deploy an agent with cost controls and quality gates:

```bash
kubectl apply -f examples/budget-and-eval.yaml
```

This creates an `ArcanaBudget` (token spend limits with alerts) and an `ArcanaEvalSuite` (automated quality checks that can warn or block deployments).

## Step 5: Open Studio

The Arcana Studio UI gives you a visual dashboard for managing agents:

```bash
cd services/studio
npm ci && npm run dev
```

Open [http://localhost:3000](http://localhost:3000) in your browser.

From Studio you can:
- See all running agents and their status
- Monitor token usage and costs
- View agent conversation logs
- Manage skills and knowledge bases

## What's Next

| Want to... | Read... |
|-----------|---------|
| Understand how the platform works | [Architecture](architecture.md) |
| Add a new service or CRD | [Development Guide](development.md) |
| Deploy to staging or production | [Deployment Guide](deployment.md) |
| Lock down security | [Sandbox Security](sandbox-security.md), [Secrets](secrets-management.md), [TLS](tls-setup.md) |
| Contribute | [Contributing Guide](../CONTRIBUTING.md) |

## Tearing Down

When you're done:

```bash
make dev-down
```

This stops all Compose services and deletes the Kind cluster. Clean slate.
