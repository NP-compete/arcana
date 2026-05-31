.PHONY: help dev dev-down dev-status build build-go build-python build-studio \
       test test-go test-python test-studio lint fmt \
       kind-up kind-down crds-install \
       docker-build kind-load ingress-install backing-deploy helm-deploy deploy \
       clean

SHELL := /bin/bash
.DEFAULT_GOAL := help

# ──────────────────────────────────────────────
# Configuration
# ──────────────────────────────────────────────

CLUSTER_NAME    ?= arcana-dev
KIND_CONFIG     := deploy/kind/cluster.yaml
KUBECONFIG      := $(shell pwd)/kubeconfig-$(CLUSTER_NAME)
NAMESPACE       := arcana

GO_SERVICES     := engine operator mesh api agui blueprint oracle \
                   codex-router codex-searcher codex-ingestor codex-scorer \
                   tools sandbox audit scheduler registry finops gitops
GO_CLI          := cli
PYTHON_SERVICES := skills ward memory connectors graph forge models probe annotate
TS_SERVICES     := studio
ALL_IMAGES      := $(addprefix arcana-,$(GO_SERVICES) $(PYTHON_SERVICES) $(TS_SERVICES))

TAG             ?= dev

CONTAINER_RT    := $(shell command -v podman 2>/dev/null || command -v docker 2>/dev/null)
CONTAINER_CMD   := $(notdir $(CONTAINER_RT))

export KUBECONFIG
export KIND_EXPERIMENTAL_PROVIDER=podman

# ──────────────────────────────────────────────
# Help
# ──────────────────────────────────────────────

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
	  awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ──────────────────────────────────────────────
# Full Stack (Kind-only, no Compose)
# ──────────────────────────────────────────────

dev: kind-up ingress-install crds-install docker-build kind-load backing-deploy helm-deploy ## One command: Kind + build + deploy everything
	@echo ""
	@echo "╔════════════════════════════════════════════════════════════╗"
	@echo "║             Arcana Platform Running on Kind                ║"
	@echo "╠════════════════════════════════════════════════════════════╣"
	@echo "║  Cluster:     $(CLUSTER_NAME)                              ║"
	@echo "║  Namespace:   $(NAMESPACE)                                 ║"
	@echo "║  KUBECONFIG:  $(KUBECONFIG)                                ║"
	@echo "║                                                            ║"
	@echo "║  Studio UI:   http://arcana.localhost.me:8080                ║"
	@echo "║  API:         http://arcana.localhost.me:8080/api/v1/health║"
	@echo "║  AG-UI:       http://arcana.localhost.me:8080/events       ║"
	@echo "╚════════════════════════════════════════════════════════════╝"
	@echo ""
	@echo "Run 'make dev-status' to check pod health."

dev-down: kind-down ## Tear down entire dev environment
	@echo "Dev environment destroyed."

dev-clean: dev-down clean dev ## Destroy, clean, and rebuild everything from scratch

dev-status: ## Check status of all dev services
	@echo "=== Kind Cluster ==="
	@kind get clusters 2>/dev/null | grep -q $(CLUSTER_NAME) && echo "✓ $(CLUSTER_NAME) running" || echo "✗ $(CLUSTER_NAME) not found"
	@echo ""
	@echo "=== Pods ==="
	@kubectl get pods -n $(NAMESPACE) -o wide 2>/dev/null || echo "No pods found"
	@echo ""
	@echo "=== Services ==="
	@kubectl get svc -n $(NAMESPACE) 2>/dev/null || echo "No services found"
	@echo ""
	@echo "=== CRDs ==="
	@kubectl get crds 2>/dev/null | grep arcana || echo "No Arcana CRDs installed"
	@echo ""
	@echo "=== Ingress ==="
	@kubectl get ingress -n $(NAMESPACE) 2>/dev/null || echo "No ingress found"

# ──────────────────────────────────────────────
# Kind Cluster
# ──────────────────────────────────────────────

kind-up: ## Create Kind cluster
	@if kind get clusters 2>/dev/null | grep -q $(CLUSTER_NAME); then \
	  echo "Kind cluster '$(CLUSTER_NAME)' already exists."; \
	else \
	  echo "Creating Kind cluster '$(CLUSTER_NAME)'..."; \
	  kind create cluster --name $(CLUSTER_NAME) --config $(KIND_CONFIG) --kubeconfig $(KUBECONFIG) \
	    && echo "Kind cluster created. KUBECONFIG=$(KUBECONFIG)" \
	    || { echo "ERROR: Kind cluster creation failed."; exit 1; }; \
	fi

kind-down: ## Delete Kind cluster
	@kind delete cluster --name $(CLUSTER_NAME) 2>/dev/null || true
	@rm -f $(KUBECONFIG)

# ──────────────────────────────────────────────
# CRDs
# ──────────────────────────────────────────────

crds-install: ## Install Arcana CRDs into Kind cluster
	@echo "Installing Arcana CRDs..."
	@kubectl apply -f deploy/crds/
	@echo "CRDs installed."

# ──────────────────────────────────────────────
# Container Images
# ──────────────────────────────────────────────

docker-build: ## Build all container images
	@echo "Building container images with $(CONTAINER_CMD)..."
	@for svc in $(GO_SERVICES); do \
	  echo "  → arcana-$$svc"; \
	  $(CONTAINER_CMD) build -t arcana-$$svc:$(TAG) -f cmd/$$svc/Dockerfile . || exit 1; \
	  if [ "$(CONTAINER_CMD)" = "podman" ]; then $(CONTAINER_CMD) tag localhost/arcana-$$svc:$(TAG) arcana-$$svc:$(TAG) 2>/dev/null || true; fi; \
	done
	@for svc in $(PYTHON_SERVICES); do \
	  echo "  → arcana-$$svc"; \
	  $(CONTAINER_CMD) build -t arcana-$$svc:$(TAG) -f services/$$svc/Dockerfile . || exit 1; \
	  if [ "$(CONTAINER_CMD)" = "podman" ]; then $(CONTAINER_CMD) tag localhost/arcana-$$svc:$(TAG) arcana-$$svc:$(TAG) 2>/dev/null || true; fi; \
	done
	@echo "  → arcana-studio"
	@$(CONTAINER_CMD) build -t arcana-studio:$(TAG) -f services/studio/Dockerfile . || exit 1
	@if [ "$(CONTAINER_CMD)" = "podman" ]; then $(CONTAINER_CMD) tag localhost/arcana-studio:$(TAG) arcana-studio:$(TAG) 2>/dev/null || true; fi
	@echo "All images built."

kind-load: ## Load images into Kind cluster
	@echo "Loading images into Kind cluster '$(CLUSTER_NAME)'..."
	@if [ "$(CONTAINER_CMD)" = "podman" ]; then \
	  for img in $(ALL_IMAGES); do \
	    echo "  → $$img:$(TAG)"; \
	    $(CONTAINER_CMD) save localhost/$$img:$(TAG) -o /tmp/$$img.tar \
	      && kind load image-archive /tmp/$$img.tar --name $(CLUSTER_NAME) \
	      && rm -f /tmp/$$img.tar || exit 1; \
	  done; \
	else \
	  for img in $(ALL_IMAGES); do \
	    echo "  → $$img:$(TAG)"; \
	    kind load docker-image $$img:$(TAG) --name $(CLUSTER_NAME) || exit 1; \
	  done; \
	fi
	@echo "All images loaded."

# ──────────────────────────────────────────────
# Ingress Controller
# ──────────────────────────────────────────────

ingress-install: ## Install Nginx Ingress Controller into Kind
	@echo "Waiting for Kubernetes API server to be ready..."
	@for i in $$(seq 1 30); do kubectl get nodes >/dev/null 2>&1 && break || sleep 2; done
	@echo "Installing Nginx Ingress Controller..."
	@kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.12.0/deploy/static/provider/kind/deploy.yaml --validate=false
	@echo "Waiting for ingress controller to be ready..."
	@kubectl wait --namespace ingress-nginx \
	  --for=condition=ready pod \
	  --selector=app.kubernetes.io/component=controller \
	  --timeout=180s 2>/dev/null || echo "Warning: ingress controller not ready yet, continuing..."

# ──────────────────────────────────────────────
# Backing Services (K8s)
# ──────────────────────────────────────────────

backing-deploy: ## Deploy backing services (PostgreSQL, Redis, Temporal, MinIO, NATS, monitoring, policies) to Kind
	@echo "Deploying backing services to namespace '$(NAMESPACE)'..."
	@kubectl apply -f deploy/backing/namespace.yaml
	@kubectl apply -f deploy/backing/secrets.yaml
	@kubectl apply -f deploy/backing/external-db.yaml
	@kubectl apply -f deploy/backing/postgres.yaml
	@kubectl apply -f deploy/backing/redis.yaml
	@kubectl apply -f deploy/backing/nats.yaml
	@kubectl apply -f deploy/backing/minio.yaml
	@echo "Waiting for core services to be ready..."
	@kubectl wait -n $(NAMESPACE) --for=condition=ready pod -l app.kubernetes.io/name=postgres --timeout=120s 2>/dev/null || true
	@kubectl wait -n $(NAMESPACE) --for=condition=ready pod -l app.kubernetes.io/name=redis --timeout=60s 2>/dev/null || true
	@echo "Deploying Temporal (depends on PostgreSQL)..."
	@kubectl apply -f deploy/backing/temporal.yaml
	@echo "Deploying monitoring and observability..."
	@kubectl apply -f deploy/backing/monitoring.yaml 2>/dev/null || true
	@kubectl apply -f deploy/backing/alerting-rules.yaml 2>/dev/null || true
	@kubectl apply -f deploy/backing/alertmanager.yaml 2>/dev/null || true
	@kubectl apply -f deploy/backing/jaeger.yaml 2>/dev/null || true
	@echo "Deploying network policies..."
	@kubectl apply -f deploy/backing/network-policies.yaml 2>/dev/null || true
	@kubectl apply -f deploy/backing/egress-policies.yaml 2>/dev/null || true
	@echo "Deploying sandbox runtime..."
	@kubectl apply -f deploy/backing/sandbox-runtime.yaml 2>/dev/null || true
	@echo "Backing services deployed."

# ──────────────────────────────────────────────
# Helm Deploy (Arcana Services)
# ──────────────────────────────────────────────

helm-deploy: ## Deploy all Arcana services via Helm
	@echo "Deploying Arcana services via Helm..."
	@for svc in $(GO_SERVICES) $(PYTHON_SERVICES) $(TS_SERVICES); do \
	  echo "  → arcana-$$svc"; \
	  helm upgrade --install arcana-$$svc deploy/helm/arcana-$$svc/ \
	    --namespace $(NAMESPACE) --create-namespace \
	    --set image.tag=$(TAG) \
	    --wait --timeout 120s 2>/dev/null \
	    || helm upgrade --install arcana-$$svc deploy/helm/arcana-$$svc/ \
	         --namespace $(NAMESPACE) --create-namespace \
	         --set image.tag=$(TAG) || exit 1; \
	done
	@echo "Deploying Ingress rules..."
	@kubectl apply -f deploy/backing/ingress.yaml
	@echo "All services deployed."

helm-uninstall: ## Uninstall all Arcana Helm releases
	@for svc in $(GO_SERVICES) $(PYTHON_SERVICES) $(TS_SERVICES); do \
	  helm uninstall arcana-$$svc --namespace $(NAMESPACE) 2>/dev/null || true; \
	done

# ──────────────────────────────────────────────
# Build (local binaries, for dev without K8s)
# ──────────────────────────────────────────────

build: build-go build-cli build-python build-studio ## Build all services locally

build-go: ## Build Go services
	@echo "Building Go services..."
	@mkdir -p bin
	@for svc in $(GO_SERVICES); do \
	  echo "  → cmd/$$svc"; \
	  (cd cmd/$$svc && go build -o ../../bin/arcana-$$svc .) || exit 1; \
	done
	@echo "Go services built → bin/"

build-cli: ## Build Arcana CLI
	@echo "Building CLI..."
	@mkdir -p bin
	@(cd cmd/cli && go build -o ../../bin/arcana .) || exit 1
	@echo "CLI built → bin/arcana"

build-python: ## Build Python services (install deps in venvs)
	@echo "Building Python services..."
	@for svc in $(PYTHON_SERVICES); do \
	  echo "  → services/$$svc"; \
	  (cd services/$$svc \
	    && python3 -m venv .venv \
	    && .venv/bin/pip install -e ".[dev]" --quiet) || exit 1; \
	done
	@echo "Python services ready."

build-studio: ## Build Studio (TypeScript)
	@echo "Building Studio..."
	@(cd services/studio && npm ci --silent && npm run build)
	@echo "Studio built."

# ──────────────────────────────────────────────
# Test
# ──────────────────────────────────────────────

test: test-go test-python test-studio ## Run all tests

test-go: ## Run Go tests
	@echo "Running Go tests..."
	@for svc in $(GO_SERVICES); do \
	  (cd cmd/$$svc && go test ./...) || exit 1; \
	done

test-python: ## Run Python tests
	@echo "Running Python tests..."
	@for svc in $(PYTHON_SERVICES); do \
	  (cd services/$$svc && .venv/bin/python -m pytest tests/) || exit 1; \
	done

test-studio: ## Run Studio tests
	@(cd services/studio && npm test)

# ──────────────────────────────────────────────
# Lint & Format
# ──────────────────────────────────────────────

lint: ## Lint all code
	@echo "Linting Go..."
	@golangci-lint run ./cmd/... ./pkg/...
	@echo "Linting Python..."
	@ruff check services/skills/ services/ward/
	@echo "Linting TypeScript..."
	@cd services/studio && npm run lint

fmt: ## Format all code
	@gofmt -w cmd/ pkg/
	@ruff format services/skills/ services/ward/
	@cd services/studio && npm run fmt

# ──────────────────────────────────────────────
# Clean
# ──────────────────────────────────────────────

clean: ## Remove build artifacts
	@rm -rf bin/ dist/
	@find . -type d -name __pycache__ -exec rm -rf {} + 2>/dev/null || true
	@find . -type d -name node_modules -exec rm -rf {} + 2>/dev/null || true
	@echo "Cleaned."
