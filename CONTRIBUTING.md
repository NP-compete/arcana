# Contributing to Arcana

We're building the deployment platform for AI agents. Contributions make it better for everyone.

## Your First Contribution

Not sure where to start? Here are some good entry points:

1. **Pick a `good-first-issue`** — [browse open issues](https://github.com/NP-compete/arcana/labels/good-first-issue)
2. **Improve docs** — fix a typo, clarify a confusing section, add an example
3. **Add a test** — find untested code and cover it
4. **Try the platform** — run `make dev`, deploy an agent, and file issues for anything that's confusing

## Setup

### Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.23+ | Go services (`cmd/`) and shared packages (`pkg/`) |
| Python | 3.11+ | Python services (`services/skills`, `services/ward`) |
| Node.js | 22+ | Studio UI (`services/studio`) |
| Kind | 0.20+ | Local Kubernetes cluster |
| Docker or Podman | 4.0+ | Backing services and container builds |
| kubectl | latest | Cluster interaction |

### Get Running

```bash
git clone https://github.com/NP-compete/arcana.git
cd arcana
make dev          # creates cluster, starts services, builds everything
make dev-status   # verify health
```

Arcana auto-detects Podman or Docker. Override with `CONTAINER_CMD=docker make dev`.

See the [Development Guide](docs/development.md) for a deeper walkthrough.

## Workflow

### 1. Create a Branch

```bash
./hack/new-branch.sh feat add-skill-versioning
# Creates: feat/add-skill-versioning
```

Branch naming: `{type}/{slug}`

| Type | Use for |
|------|---------|
| `feat` | New features |
| `fix` | Bug fixes |
| `refactor` | Code restructuring |
| `docs` | Documentation |
| `chore` | Maintenance, deps, CI |
| `scaffold` | New services or infrastructure |

### 2. Make Your Changes

Write code, write tests, run the full check:

```bash
make test   # all tests (Go + Python + TypeScript)
make lint   # all linters
```

### 3. Commit

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
type(scope): description
```

**Scopes:** `engine`, `operator`, `mesh`, `api`, `agui`, `skills`, `ward`, `studio`, `helm`, `crds`, `opa`, `ci`

```
feat(skills): add versioning support
fix(operator): reconcile ArcanaAgent status correctly
docs(architecture): clarify five-plane layout
chore(ci): bump golangci-lint version
```

### 4. Open a PR

```bash
git push -u origin HEAD
gh pr create
```

- Keep PRs small and focused — one concern per PR
- Use the [PR template](.github/PULL_REQUEST_TEMPLATE.md)
- Ensure `make test` and `make lint` pass
- Never push directly to `main`

## Code Style

| Language | Tools |
|----------|-------|
| Go | `gofmt` + `golangci-lint` |
| Python | `ruff` (lint + format) |
| TypeScript | `eslint` + `prettier` |

```bash
make fmt    # format all code
make lint   # lint all code
```

## Testing

New features must include tests.

| Target | Scope |
|--------|-------|
| `make test-go` | Go services in `cmd/` |
| `make test-python` | Python services |
| `make test-studio` | Studio UI |

## CRDs

Manifests live in `deploy/crds/`. Go types in `pkg/crds/`. **Keep them in sync** — when you change a CRD schema, update the Go types.

```bash
make crds-install
```

## Helm Charts

One chart per service under `deploy/helm/{service}/`. When adding a new service, scaffold a matching chart.

## Code of Conduct

We follow the [Contributor Covenant](CODE_OF_CONDUCT.md). Be respectful, be constructive, be kind.

## Questions?

Open a [discussion](https://github.com/NP-compete/arcana/discussions) or reach out on [Discord](https://discord.gg/arcana).
