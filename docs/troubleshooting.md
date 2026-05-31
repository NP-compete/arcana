# Troubleshooting Guide

## Service Won't Start

### Port Already in Use
```
listen tcp :8080: bind: address already in use
```
Another process is using the port. Find and stop it:
```bash
lsof -i :8080
kill <PID>
```

### Missing Environment Variables
Services log missing config at startup. Check required variables:
```bash
kubectl logs <POD_NAME> -n arcana | head -20
```

Required for most services:
- `PORT` — defaults to service-specific port if unset
- `POSTGRES_HOST` — defaults to `postgres`

### Database Connection Failed
```
db: waiting for postgres...
```
The service retries PostgreSQL connection 30 times with 2s intervals. If it exhausts retries:
1. Check PostgreSQL is running: `kubectl get pods -n arcana -l app=postgres`
2. Verify credentials: `kubectl get secret arcana-platform-secret -n arcana -o yaml`
3. Check network policies allow the connection

## Health Check Failures

### Liveness Probe Failed (/healthz)
The service process is unresponsive. Kubernetes will restart it.
- Check for deadlocks in logs
- Check memory usage: `kubectl top pod <POD_NAME> -n arcana`

### Readiness Probe Failed (/readyz)
The service can't reach its dependencies (usually PostgreSQL).
- Check database connectivity
- Verify the `DB` config was passed to the server

## Authentication Issues

### API Key Rejected
```json
{"error": "invalid or expired API key"}
```
1. Verify the key format: must start with `ak-`
2. Check if the key is revoked: query the `api_keys` table
3. Check if the key is expired: `expires_at` column

### JWT Verification Failed
```json
{"error": "signature verification failed"}
```
1. Verify `JWT_SIGNING_KEY` matches between the token issuer and the API server
2. In production, `JWT_SIGNING_KEY` must be set explicitly (no fallback)
3. Check token expiration: decode the payload and check `exp` claim

### RBAC Permission Denied
```json
{"error": "insufficient permissions"}
```
Check the user's role and required scope:
- `admin` — all resources
- `developer` — agents, skills, models, blueprints
- `data-engineer` — connectors, codex, datasets
- `sre` — health, metrics, deployments
- `auditor` — audit logs (read-only)
- `user` — chat, agents (use only)

## High Latency

### Database Query Slow
Check the `arcana_db_query_duration_seconds` metric in Prometheus:
```promql
histogram_quantile(0.95, rate(arcana_db_query_duration_seconds_bucket[5m]))
```

If p95 > 100ms:
1. Check for missing indexes in PostgreSQL
2. Check connection pool saturation (max 25 open connections per service)
3. Look for N+1 query patterns in the handler code

### Request Timeouts
Default timeouts: read 15s, write 30s, idle 120s. If requests are timing out:
1. Check `arcana_http_request_duration_seconds` for the slow endpoint
2. Verify downstream services are responding
3. Check for resource contention (CPU throttling, memory pressure)

## Kubernetes Issues

### Pod Evicted
```
Status: Evicted
Reason: The node was low on resource: ephemeral-storage
```
Services use read-only root filesystems. If a service writes temp files:
1. Check for `/tmp` usage in the container
2. Add an emptyDir volume if needed

### PodDisruptionBudget Blocking Drain
```
Cannot evict pod as it would violate the pod's disruption budget
```
PDBs require `minAvailable: 1`. During node drain:
1. Ensure at least 2 replicas are running
2. Scale up before maintenance: `kubectl scale deployment arcana-<SVC> --replicas=3`

### ImagePullBackOff
1. Check if the image exists: `docker pull <IMAGE>:<TAG>`
2. Verify registry credentials if using a private registry
3. For local Kind clusters, ensure images are loaded: `make kind-load`

## Debugging Tips

### Structured Log Queries
All services output JSON logs. Filter by fields:
```bash
# Find errors for a specific service
kubectl logs <POD_NAME> -n arcana | jq 'select(.level == "error")'

# Find requests slower than 1s
kubectl logs <POD_NAME> -n arcana | jq 'select(.fields.duration_ms > 1000)'

# Trace a request by ID
kubectl logs <POD_NAME> -n arcana | jq 'select(.fields.request_id == "abc-123")'
```

### Prometheus Queries
```promql
# Error rate by service
rate(arcana_http_requests_total{status=~"5.."}[5m])

# Active connections
arcana_active_connections

# Auth failure rate
rate(arcana_auth_failures_total[5m])

# Rate limit hits
rate(arcana_rate_limit_hits_total[5m])
```
