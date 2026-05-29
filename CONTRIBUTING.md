# Contributing to Arcana

Thank you for contributing to Arcana. This guide covers setup, workflow, and conventions for the monorepo.

## Development Setup

### Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.23+ | Go services (`cmd/`) and shared packages (`pkg/`) |
| Python | 3.11+ | Python services (`services/skills`, `services/ward`) |
| Node.js | 22+ | Studio UI (`services/studio`) |
| Kind | 0.20+ | Local Kubernetes cluster |
| Docker or Podman | 4.0+ | Backing services and container builds |
| kubectl | latest | Cluster interaction |

Clone the repository and start the full development environment:

```bash
git clone https://github.com/NP-compete/arcana.git
cd arcana
make dev
```

`make dev` creates a Kind cluster, starts backing services (PostgreSQL, Redis, Temporal, MinIO, NATS) via Compose, installs CRDs, and builds all services. Use `make dev-status` to verify health and `make dev-down` to tear everything down.

Arcana auto-detects Podman or Docker. Override with `CONTAINER_CMD=docker make dev` or `CONTAINER_CMD=podman make dev`.

## Branch Naming

Use the pattern `{type}/{slug}`:

| Type | Use for |
|------|---------|
| `feat` | New features |
| `fix` | Bug fixes |
| `refactor` | Code restructuring without behavior change |
| `docs` | Documentation only |
| `chore` | Maintenance, deps, CI |
| `scaffold` | New service or infrastructure scaffolding |

The slug should be kebab-case and describe the change (e.g. `add-skill-versioning`).

Create branches with the helper script:

```bash
./hack/new-branch.sh feat add-skill-versioning
# Creates branch: feat/add-skill-versioning
```

## Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
type(scope): description
```

**Types:** `feat`, `fix`, `refactor`, `docs`, `chore`, `scaffold`, `test`

**Scopes:** `engine`, `operator`, `mesh`, `api`, `agui`, `skills`, `ward`, `studio`, `helm`, `crds`, `opa`, `ci`

Examples:

```
feat(skills): add versioning support
fix(operator): reconcile ArcanaAgent status correctly
docs(architecture): document five-plane layout
chore(ci): bump golangci-lint version
```

## Pull Requests

- **Never push directly to `main`.** All changes go through pull requests.
- Keep PRs **small and focused** — one concern per PR.
- Use the [PR template](.github/PULL_REQUEST_TEMPLATE.md) when creating PRs.
- Run `make test` and `make lint` before opening a PR.
- Ensure the checklist in the PR template is complete.

```bash
git push -u origin HEAD
gh pr create
```

## Code Style

| Language | Tools |
|----------|-------|
| Go | `gofmt` + `golangci-lint` |
| Python | `ruff` (lint + format) |
| TypeScript | `eslint` + `prettier` |

Run formatting and linting across the repo:

```bash
make fmt    # Format all code
make lint   # Lint all code
```

## Testing

Run the full test suite before pushing:

```bash
make test
```

New features must include tests. Per-language targets:

| Target | Scope |
|--------|-------|
| `make test-go` | Go services in `cmd/` |
| `make test-python` | Python services in `services/skills`, `services/ward` |
| `make test-studio` | Studio UI tests |

## CRDs

All CRD manifests live under `deploy/crds/`. Go types live in `pkg/crds/`. **Keep them in sync** — when you change a CRD schema, update the corresponding Go types.

Install CRDs into the local Kind cluster:

```bash
make crds-install
```

## Helm Charts

One Helm chart per service under `deploy/helm/{service}/`:

| Chart | Service |
|-------|---------|
| `arcana-api` | API gateway |
| `arcana-engine` | Agent orchestration engine |
| `arcana-operator` | Kubernetes operator |
| `arcana-mesh` | A2A/ACP mesh gateway |
| `arcana-agui` | AG-UI protocol server |
| `arcana-skills` | Skill engine |
| `arcana-ward` | Guardrails pipeline |
| `arcana-studio` | Web UI |

When adding a new service, scaffold a matching chart under `deploy/helm/`.
