# Development Guide

## Getting Running

```bash
git clone https://github.com/NP-compete/arcana.git
cd arcana
make dev
```

Done. This creates a Kind cluster, starts all backing services, installs CRDs, and builds everything. Takes 2-3 minutes.

```bash
make dev-status                           # verify health
export KUBECONFIG=$(pwd)/kubeconfig-arcana-dev   # set kubeconfig
```

Tear down with `make dev-down`.

### What `make dev` Does

```
make dev
  ├── kind-up          Create Kind cluster (deploy/kind/cluster.yaml)
  ├── compose-up       Start PostgreSQL, Redis, Temporal, MinIO, NATS
  ├── crds-install     kubectl apply -f deploy/crds/
  └── build            Build all Go, Python, and TypeScript services
```

### Container Runtime

Arcana auto-detects Podman or Docker. Override:

```bash
CONTAINER_CMD=docker make dev
CONTAINER_CMD=podman make dev
```

---

## Creating a Branch

```bash
./hack/new-branch.sh feat my-feature
# Creates: feat/my-feature
```

---

## Adding a New Service

### Go Service

1. Create `cmd/{service}/main.go` with `/healthz` and `/readyz` endpoints.
2. Add `Dockerfile` at `cmd/{service}/Dockerfile`.
3. Add `{service}` to `GO_SERVICES` in the Makefile.
4. Create Helm chart at `deploy/helm/arcana-{service}/`.
5. Add shared types to `pkg/` if needed.

```bash
cd cmd/{service} && go build -o ../../bin/arcana-{service} .
make test-go && make lint
```

### Python Service

1. Create `services/{service}/` with FastAPI app in `app/main.py`.
2. Add `pyproject.toml` with `[dev]` extras.
3. Add tests under `services/{service}/tests/`.
4. Add `Dockerfile` at `services/{service}/Dockerfile`.
5. Add `{service}` to `PYTHON_SERVICES` in the Makefile.
6. Create Helm chart at `deploy/helm/arcana-{service}/`.

```bash
cd services/{service}
python3 -m pip install -e ".[dev]"
python3 -m pytest tests/
ruff check .
```

### TypeScript Service

1. Create `services/{service}/` (follow `services/studio/` as reference).
2. Add `{service}` to `TS_SERVICES` in the Makefile.
3. Create Helm chart at `deploy/helm/arcana-{service}/`.

```bash
cd services/{service}
npm ci && npm run build && npm test && npm run lint
```

---

## Adding a New CRD

1. Define the manifest in `deploy/crds/arcana-{name}.yaml`:
   - Group: `arcana.io`
   - Add a short name (e.g. `aag` for ArcanaAgent)
   - Define OpenAPI v3 schema

2. Add Go types in `pkg/crds/`.

3. Install and verify:
   ```bash
   make crds-install
   kubectl get crds | grep arcana
   kubectl explain arcanaagent.spec
   ```

4. Update the operator (`cmd/operator/`) to reconcile the new CRD.

5. Add OPA constraints in `deploy/opa/` if needed.

**Keep `deploy/crds/` and `pkg/crds/` in sync at all times.**

---

## Running Tests

```bash
make test        # everything
make test-go     # Go services
make test-python # Python services
make test-studio # Studio UI
```

### Lint and Format

```bash
make lint   # golangci-lint, ruff, eslint
make fmt    # gofmt, ruff format, prettier
```

---

## Debugging

### Kubernetes

```bash
export KUBECONFIG=$(pwd)/kubeconfig-arcana-dev

kubectl get crds | grep arcana
kubectl get arcanaagents -A
kubectl describe arcanaagent my-agent -n default
kubectl logs -l app=arcana-operator -f
```

### Compose Logs

```bash
docker compose -f deploy/compose/docker-compose.yaml logs -f
docker compose -f deploy/compose/docker-compose.yaml logs -f postgres
```

### Port Forwarding

```bash
kubectl port-forward svc/arcana-api 8080:8080
kubectl port-forward svc/arcana-engine 8081:8081
```

Backing services are already on localhost via Compose:

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

```bash
# Go service
make build-go
PORT=8081 ./bin/arcana-engine

# Python service with hot reload
cd services/skills
python3 -m pip install -e ".[dev]"
uvicorn app.main:app --reload --port 8085

# Studio in dev mode
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
