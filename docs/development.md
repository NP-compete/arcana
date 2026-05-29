# Development Guide

This guide walks through first-time setup, the `make dev` workflow, adding new services and CRDs, running tests, and debugging.

## First-Time Setup

1. **Install prerequisites** (see [CONTRIBUTING.md](../CONTRIBUTING.md#development-setup)):
   - Go 1.23+, Python 3.11+, Node.js 22+
   - Kind 0.20+, Docker or Podman 4.0+, kubectl

2. **Clone and start the environment:**

   ```bash
   git clone https://github.com/NP-compete/arcana.git
   cd arcana
   make dev
   ```

3. **Verify everything is running:**

   ```bash
   make dev-status
   ```

4. **Set KUBECONFIG** (printed by `make dev`):

   ```bash
   export KUBECONFIG=$(pwd)/kubeconfig-arcana-dev
   ```

5. **Create a feature branch** before making changes:

   ```bash
   ./hack/new-branch.sh feat my-feature
   ```

## How `make dev` Works

`make dev` orchestrates four steps defined in the Makefile:

```
make dev
  ├── kind-up          Create Kind cluster (deploy/kind/cluster.yaml)
  ├── compose-up       Start backing services (deploy/compose/docker-compose.yaml)
  ├── crds-install     kubectl apply -f deploy/crds/
  └── build            Build all Go, Python, and TypeScript services
```

| Step | What happens |
|------|--------------|
| **Kind cluster** | Creates `arcana-dev` cluster; writes kubeconfig to `./kubeconfig-arcana-dev` |
| **Compose** | Starts PostgreSQL (pgvector), Redis, Temporal (+ UI), MinIO, NATS on localhost |
| **CRDs** | Applies all eight Arcana CRDs into the Kind cluster |
| **Build** | Compiles Go binaries to `bin/`, installs Python deps, builds Studio with Vite |

Tear down with `make dev-down` (stops Compose, deletes Kind cluster).

### Container Runtime

Arcana auto-detects Podman or Docker. Override explicitly:

```bash
CONTAINER_CMD=docker make dev
CONTAINER_CMD=podman make dev
```

## Adding a New Service

### Go Service

1. Create `cmd/{service}/main.go` with health endpoints (`/healthz`, `/readyz`).
2. Add a `Dockerfile` at `cmd/{service}/Dockerfile`.
3. Add `{service}` to `GO_SERVICES` in the Makefile.
4. Create Helm chart at `deploy/helm/arcana-{service}/`.
5. Add shared types to `pkg/` if needed.

```bash
# Verify build and tests
cd cmd/{service} && go build -o ../../bin/arcana-{service} .
make test-go
make lint
```

### Python Service

1. Create `services/{service}/` with FastAPI app in `app/main.py`.
2. Add `pyproject.toml` with `[dev]` extras for pytest and ruff.
3. Add tests under `services/{service}/tests/`.
4. Add a `Dockerfile` at `services/{service}/Dockerfile`.
5. Add `{service}` to `PYTHON_SERVICES` in the Makefile.
6. Create Helm chart at `deploy/helm/arcana-{service}/`.

```bash
cd services/{service} && python3 -m pip install -e ".[dev]"
python3 -m pytest tests/
ruff check .
```

### TypeScript Service

1. Create `services/{service}/` with Vite + React (follow `services/studio/` as reference).
2. Add `{service}` to `TS_SERVICES` in the Makefile.
3. Create Helm chart at `deploy/helm/arcana-{service}/`.

```bash
cd services/{service} && npm ci && npm run build
npm test
npm run lint
```

## Adding a New CRD

1. **Define the CRD manifest** in `deploy/crds/arcana-{name}.yaml`:
   - Set `group: arcana.io`, appropriate scope (Namespaced or Cluster)
   - Add a short name (e.g. `aag` for ArcanaAgent)
   - Define the OpenAPI v3 schema under `spec.versions[].schema`

2. **Add Go types** in `pkg/crds/` matching the schema fields.

3. **Install and verify:**

   ```bash
   make crds-install
   kubectl get crds | grep arcana
   kubectl explain arcanaagent.spec
   ```

4. **Update the operator** (`cmd/operator/`) to reconcile the new CRD when controller logic is ready.

5. **Add OPA constraints** under `deploy/opa/` if the CRD needs admission policy enforcement.

Keep `deploy/crds/*.yaml` and `pkg/crds/` in sync at all times.

## Running Tests

### All Tests

```bash
make test
```

### Per Language

| Command | What it runs |
|---------|--------------|
| `make test-go` | `go test ./...` in each `cmd/{service}` |
| `make test-python` | `pytest tests/` in `services/skills` and `services/ward` |
| `make test-studio` | `npm test` in `services/studio` |

### Lint and Format

```bash
make lint   # golangci-lint, ruff, eslint
make fmt    # gofmt, ruff format, prettier
```

## Debugging Tips

### kubectl

```bash
export KUBECONFIG=$(pwd)/kubeconfig-arcana-dev

# List Arcana CRDs and resources
kubectl get crds | grep arcana
kubectl get arcanaagents -A
kubectl get arcanatenants

# Inspect a resource
kubectl describe arcanaagent my-agent -n default
kubectl logs -l app=arcana-operator -f
```

### Compose Logs

```bash
# All backing services
docker compose -f deploy/compose/docker-compose.yaml logs -f

# Single service
docker compose -f deploy/compose/docker-compose.yaml logs -f postgres
docker compose -f deploy/compose/docker-compose.yaml logs -f temporal
```

Use `podman compose` instead of `docker compose` if running Podman.

### Port Forwarding

Forward a service running in Kind to localhost:

```bash
kubectl port-forward svc/arcana-api 8080:8080
kubectl port-forward svc/arcana-engine 8081:8081
```

Backing services are already exposed on localhost by Compose:

| Service | Address |
|---------|---------|
| PostgreSQL | `localhost:5432` |
| Redis | `localhost:6379` |
| Temporal gRPC | `localhost:7233` |
| Temporal UI | `localhost:8233` |
| MinIO API | `localhost:9000` |
| MinIO Console | `localhost:9001` |
| NATS | `localhost:4222` |

### Running a Service Locally

Build and run a Go service directly (outside Kind):

```bash
make build-go
PORT=8081 ./bin/arcana-engine
```

Run a Python service with hot reload:

```bash
cd services/skills
python3 -m pip install -e ".[dev]"
uvicorn app.main:app --reload --port 8085
```

Run Studio in dev mode:

```bash
cd services/studio
npm ci && npm run dev
```

### Common Issues

| Symptom | Fix |
|---------|-----|
| Kind cluster already exists | `make kind-down && make kind-up` |
| Compose port conflict | Stop conflicting services or change ports in `docker-compose.yaml` |
| CRDs not found | `make crds-install` |
| Go module errors | Run from repo root; ensure `go.work` includes all modules |
