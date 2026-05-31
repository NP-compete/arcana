# Runbook: Service Unhealthy

## Symptoms

- Kubernetes readiness probe failing (`/readyz` returns non-200)
- Pod in `CrashLoopBackOff` or `NotReady` state
- Alerts from Prometheus for `up == 0` or high error rates
- Health check dashboard showing red

## Severity

- **P1** if user-facing services (api, studio, agui) are down
- **P2** if backend services (engine, mesh, operator) are degraded
- **P3** if non-critical services (finops, audit, gitops) are affected

## Investigation Steps

### 1. Check Pod Status

```bash
kubectl get pods -n arcana -l app.kubernetes.io/name=arcana-<SERVICE>
kubectl describe pod <POD_NAME> -n arcana
```

### 2. Check Logs

```bash
# Current container logs
kubectl logs <POD_NAME> -n arcana --tail=100

# Previous container logs (if restarting)
kubectl logs <POD_NAME> -n arcana --previous --tail=100
```

### 3. Check Events

```bash
kubectl get events -n arcana --sort-by='.lastTimestamp' | grep <SERVICE>
```

### 4. Check Resource Usage

```bash
kubectl top pod <POD_NAME> -n arcana
```

### 5. Check Dependencies

| Service | Dependencies |
|---------|-------------|
| api | PostgreSQL, all backend services |
| engine | PostgreSQL, Redis, Temporal |
| mesh | PostgreSQL, NATS |
| operator | Kubernetes API server |
| skills, ward, memory | PostgreSQL |
| codex-* | PostgreSQL, Redis |

```bash
# PostgreSQL connectivity
kubectl exec <POD_NAME> -n arcana -- wget -qO- http://localhost:PORT/readyz

# NATS connectivity
kubectl get pods -n arcana -l app=nats

# Redis connectivity
kubectl get pods -n arcana -l app=redis
```

## Common Causes & Fixes

### OOMKilled
Pod exceeded memory limits. Check for memory leaks or increase limits:
```bash
kubectl get pod <POD_NAME> -n arcana -o jsonpath='{.status.containerStatuses[0].lastState}'
# If OOMKilled, increase memory limit in values.yaml
```

### Database Connection Refused
PostgreSQL is down or connection pool exhausted:
```bash
kubectl get pods -n arcana -l app=postgres
kubectl logs postgres-0 -n arcana --tail=50
```

### CrashLoopBackOff
Service failing on startup. Check logs for the root cause:
```bash
kubectl logs <POD_NAME> -n arcana --previous
```

Common startup failures:
- Missing environment variables → check ConfigMap/Secret
- Database not ready → check PostgreSQL pod
- Port already in use → check for duplicate deployments

### ImagePullBackOff
Container image not found. Check image tag and registry access:
```bash
kubectl describe pod <POD_NAME> -n arcana | grep -A5 "Events"
```

## Remediation

### Restart the Service
```bash
kubectl rollout restart deployment arcana-<SERVICE> -n arcana
kubectl rollout status deployment arcana-<SERVICE> -n arcana
```

### Scale Up
```bash
kubectl scale deployment arcana-<SERVICE> -n arcana --replicas=3
```

### Rollback
```bash
kubectl rollout undo deployment arcana-<SERVICE> -n arcana
```

## Escalation

- **P1**: Page the on-call engineer. Slack: #arcana-incidents
- **P2**: Create a Jira ticket with P2 priority. Notify #arcana-ops
- **P3**: Create a Jira ticket. Address in next sprint

## Post-Incident

1. Update this runbook with any new failure modes discovered
2. Add monitoring for the root cause if not already covered
3. Write a post-incident review for P1/P2 incidents
