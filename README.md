# Arcana

**Kubernetes-native AI platform for building, deploying, governing, and improving AI agents and ML models.**

Arcana unifies agent orchestration (LangGraph, MCP, A2A, ACP, AG-UI), skill management, guardrails, evaluation, and FinOps into a single platform — controlled entirely through Kubernetes CRDs.

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Interact Plane                       │
│   arcana-studio (PF6)  ·  arcana-agui  ·  arcana-chat  │
├─────────────────────────────────────────────────────────┤
│                     Agent Plane                         │
│   arcana-engine  ·  arcana-operator  ·  arcana-mesh     │
├─────────────────────────────────────────────────────────┤
│                   Data / Tool Plane                     │
│   arcana-skills  ·  codex-*  ·  arcana-oracle           │
├─────────────────────────────────────────────────────────┤
│                    Govern Plane                         │
│   arcana-ward  ·  arcana-probe  ·  OPA + KubeArmor     │
├─────────────────────────────────────────────────────────┤
│                      Ops Plane                          │
│   Temporal  ·  NATS  ·  PostgreSQL  ·  Redis  ·  MinIO │
└─────────────────────────────────────────────────────────┘
```

## Quick Start

### Prerequisites

- [Kind](https://kind.sigs.k8s.io/) >= 0.20
- [Docker](https://docs.docker.com/get-docker/) or [Podman](https://podman.io/) >= 4.0
- Go >= 1.23
- Python >= 3.11
- Node.js >= 22
- kubectl

### Start Development Environment

```bash
# Clone
git clone https://github.com/NP-compete/arcana.git
cd arcana

# Start everything: Kind cluster + backing services + build all
make dev

# Check status
make dev-status

# Stop everything
make dev-down
```

### Container Runtime

Arcana auto-detects your container runtime. It prefers Podman if available, falls back to Docker.

```bash
# Force Docker
CONTAINER_CMD=docker make dev

# Force Podman
CONTAINER_CMD=podman make dev
```

## Repository Structure

```
arcana/
├── cmd/                    # Go service entrypoints
│   ├── engine/             # Agent orchestration engine
│   ├── operator/           # Kubernetes operator (CRD controller)
│   ├── mesh/               # A2A + ACP mesh gateway
│   ├── api/                # REST/GraphQL API gateway
│   └── agui/               # AG-UI protocol server (SSE)
├── pkg/                    # Shared Go packages
│   ├── common/             # Health checks, logging, config
│   └── crds/               # CRD Go types
├── services/               # Non-Go services
│   ├── skills/             # Skill engine (Python/FastAPI)
│   ├── ward/               # Guardrails pipeline (Python/FastAPI)
│   └── studio/             # Web UI (React + PatternFly 6)
├── deploy/
│   ├── kind/               # Kind cluster config
│   ├── compose/            # Docker/Podman Compose for backing services
│   ├── crds/               # CRD manifests
│   └── helm/               # Helm charts (per service)
├── hack/                   # Developer scripts
├── test/                   # E2E and integration tests
├── .github/                # CI workflows + PR template
├── Makefile                # All build/dev/test commands
└── go.work                 # Go workspace
```

## CRDs

| CRD | Short Name | Scope | Purpose |
|-----|-----------|-------|---------|
| `ArcanaAgent` | `aag` | Namespaced | Agent lifecycle and configuration |
| `ArcanaTenant` | `aten` | Cluster | Multi-tenant isolation |
| `ArcanaSkillRegistry` | `askr` | Namespaced | Skill catalog and versioning |
| `ArcanaEvalSuite` | `aes` | Namespaced | Skill evaluation pipelines |
| `ArcanaRole` | `arole` | Namespaced | RBAC + ABAC policies |
| `ArcanaBudget` | `abud` | Namespaced | FinOps token/compute budgets |

## Development Workflow

**Never push directly to `main`.** All changes go through PRs.

```bash
# Create a new branch
./hack/new-branch.sh feat add-skill-versioning

# Make changes, commit
git add . && git commit -m "feat(skills): add versioning support"

# Push and create PR
git push -u origin HEAD
gh pr create --title "feat(skills): add versioning" --body "..."
```

## Make Targets

| Target | Description |
|--------|-------------|
| `make dev` | Full dev env (Kind + compose + build) |
| `make dev-down` | Tear down dev env |
| `make dev-status` | Health check all services |
| `make build` | Build all services |
| `make test` | Run all tests |
| `make lint` | Lint all code |
| `make docker-build` | Build container images (Docker) |
| `make podman-build` | Build container images (Podman) |
| `make crds-install` | Install CRDs into cluster |
| `make clean` | Remove build artifacts |

## License

Apache 2.0
