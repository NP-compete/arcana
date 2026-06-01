# ADR-001: Shared Server Package (pkg/server)

## Status

Accepted

## Context

Arcana runs 18+ Go services in production. All of them started HTTP servers using the bare `log.Fatal(http.ListenAndServe(...))` pattern. This caused several production-readiness issues:

- **No graceful shutdown**: Services terminated immediately on SIGTERM, dropping in-flight requests during Kubernetes rolling updates
- **No HTTP timeouts**: Servers were vulnerable to slowloris attacks and resource exhaustion from idle connections
- **No panic recovery**: An unhandled panic in any handler crashed the entire service
- **No request tracing**: No request IDs for correlating logs across the service mesh
- **No request logging**: No structured audit trail of HTTP requests
- **Inconsistent health checks**: Each service manually registered `/healthz` and `/readyz` with slightly different implementations

These gaps existed identically across all 18 services, making a shared solution the right approach.

## Decision

We created `pkg/server`, a shared Go package that wraps `net/http` with production-grade defaults. All Go services import this package and use `server.New(cfg).ListenAndServe()` instead of the bare HTTP server.

The package provides:
- Graceful shutdown with configurable drain period (default 30s)
- HTTP server timeouts (read 15s, write 30s, idle 120s)
- Middleware chain: recovery, request-ID, request logging, Prometheus metrics
- Request body size limits (default 10MB)
- Auto-registered `/healthz`, `/readyz`, `/metrics` endpoints

## Consequences

**Benefits:**
- Single place to fix server-level bugs or add features (e.g., TLS, circuit breakers)
- Consistent behavior across all services
- Kubernetes rolling updates now drain connections properly
- All services get structured JSON logging and Prometheus metrics automatically

**Risks:**
- Tight coupling: a bug in pkg/server affects all services simultaneously
- Middleware ordering is fixed — services can't customize the chain

**Mitigations:**
- Comprehensive test suite in `pkg/server/server_test.go`
- Services can still register custom middleware on individual handlers
- The package uses only stdlib + existing pkg/ dependencies (no new external deps)

## Alternatives Considered

1. **Per-service hardening**: Copy the same boilerplate into each `main.go`. Rejected because it would create 18 copies of identical code that drift over time.

2. **External framework (Echo, Chi, Gin)**: Adds a large dependency, changes the handler signature (`http.HandlerFunc` → framework-specific), and requires rewriting all existing handlers. Rejected as disproportionate.

3. **Sidecar proxy (Envoy/Istio)**: Handles timeouts and retries at the mesh layer. This is complementary, not a replacement — the application still needs graceful shutdown, panic recovery, and structured logging. Could be adopted later.
