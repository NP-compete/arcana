.PHONY: help dev dev-down dev-status build build-go build-python build-studio \
       test test-go test-python test-studio lint fmt \
       kind-up kind-down crds-install compose-up compose-down \
       docker-build podman-build clean

SHELL := /bin/bash
.DEFAULT_GOAL := help

# ──────────────────────────────────────────────
# Configuration
# ──────────────────────────────────────────────

CLUSTER_NAME    ?= arcana-dev
KIND_CONFIG     := deploy/kind/cluster.yaml
COMPOSE_FILE    := deploy/compose/docker-compose.yaml
KUBECONFIG      := $(shell pwd)/kubeconfig-$(CLUSTER_NAME)

GO_SERVICES     := engine operator mesh api agui
PYTHON_SERVICES := skills ward
TS_SERVICES     := studio

CONTAINER_RT    := $(shell command -v podman 2>/dev/null || command -v docker 2>/dev/null)
CONTAINER_CMD   := $(notdir $(CONTAINER_RT))
COMPOSE_CMD     := $(CONTAINER_CMD) compose

export KUBECONFIG

# ──────────────────────────────────────────────
# Help
# ──────────────────────────────────────────────

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
	  awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ──────────────────────────────────────────────
# Development (full stack)
# ──────────────────────────────────────────────

dev: kind-up compose-up crds-install build ## Start full dev environment (Kind + backing services + build)
	@echo ""
	@echo "╔══════════════════════════════════════════════╗"
	@echo "║          Arcana Dev Environment Ready        ║"
	@echo "╠══════════════════════════════════════════════╣"
	@echo "║  Kind cluster:  $(CLUSTER_NAME)              ║"
	@echo "║  KUBECONFIG:    $(KUBECONFIG)                ║"
	@echo "║  PostgreSQL:    localhost:5432                ║"
	@echo "║  Redis:         localhost:6379                ║"
	@echo "║  Temporal:      localhost:7233 (UI: 8233)    ║"
	@echo "║  MinIO:         localhost:9000 (UI: 9001)    ║"
	@echo "║  NATS:          localhost:4222                ║"
	@echo "╚══════════════════════════════════════════════╝"
	@echo ""
	@echo "Run 'make dev-status' to check service health."

dev-down: compose-down kind-down ## Tear down full dev environment
	@echo "Dev environment stopped."

dev-status: ## Check status of all dev services
	@echo "=== Kind Cluster ==="
	@kind get clusters 2>/dev/null | grep -q $(CLUSTER_NAME) && echo "✓ $(CLUSTER_NAME) running" || echo "✗ $(CLUSTER_NAME) not found"
	@echo ""
	@echo "=== Backing Services ==="
	@$(COMPOSE_CMD) -f $(COMPOSE_FILE) ps 2>/dev/null || echo "Compose services not running"
	@echo ""
	@echo "=== CRDs ==="
	@kubectl get crds 2>/dev/null | grep arcana || echo "No Arcana CRDs installed"

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
# Backing Services (Compose)
# ──────────────────────────────────────────────

compose-up: ## Start backing services (PostgreSQL, Redis, Temporal, MinIO, NATS)
	@$(COMPOSE_CMD) -f $(COMPOSE_FILE) up -d
	@echo "Waiting for services to be healthy..."
	@$(COMPOSE_CMD) -f $(COMPOSE_FILE) ps

compose-down: ## Stop backing services
	@$(COMPOSE_CMD) -f $(COMPOSE_FILE) down -v

# ──────────────────────────────────────────────
# CRDs
# ──────────────────────────────────────────────

crds-install: ## Install Arcana CRDs into Kind cluster
	@echo "Installing Arcana CRDs..."
	@kubectl apply -f deploy/crds/
	@echo "CRDs installed."

# ──────────────────────────────────────────────
# Build
# ──────────────────────────────────────────────

build: build-go build-python build-studio ## Build all services

build-go: ## Build Go services
	@echo "Building Go services..."
	@mkdir -p bin
	@for svc in $(GO_SERVICES); do \
	  echo "  → cmd/$$svc"; \
	  (cd cmd/$$svc && go build -o ../../bin/arcana-$$svc .) || exit 1; \
	done
	@echo "Go services built → bin/"

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
# Container Images
# ──────────────────────────────────────────────

REGISTRY ?= localhost:5001
TAG      ?= dev

docker-build: ## Build all container images with Docker
	@CONTAINER_CMD=docker $(MAKE) _container-build

podman-build: ## Build all container images with Podman
	@CONTAINER_CMD=podman $(MAKE) _container-build

_container-build:
	@for svc in $(GO_SERVICES); do \
	  echo "Building $(REGISTRY)/arcana-$$svc:$(TAG)"; \
	  $(CONTAINER_CMD) build -t $(REGISTRY)/arcana-$$svc:$(TAG) -f cmd/$$svc/Dockerfile .; \
	done
	@for svc in $(PYTHON_SERVICES); do \
	  echo "Building $(REGISTRY)/arcana-$$svc:$(TAG)"; \
	  $(CONTAINER_CMD) build -t $(REGISTRY)/arcana-$$svc:$(TAG) -f services/$$svc/Dockerfile .; \
	done
	@echo "Building $(REGISTRY)/arcana-studio:$(TAG)"
	@$(CONTAINER_CMD) build -t $(REGISTRY)/arcana-studio:$(TAG) -f services/studio/Dockerfile .

# ──────────────────────────────────────────────
# Clean
# ──────────────────────────────────────────────

clean: ## Remove build artifacts
	@rm -rf bin/ dist/
	@find . -type d -name __pycache__ -exec rm -rf {} + 2>/dev/null || true
	@find . -type d -name node_modules -exec rm -rf {} + 2>/dev/null || true
	@echo "Cleaned."
