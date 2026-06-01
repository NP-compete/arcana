# Deployment Guide

Arcana deploys the same way across environments. One Helmfile, three overlays — dev, staging, prod. The platform scales from a single-node Kind cluster to a multi-replica production deployment.

## Quick Reference

```bash
cd deploy

helmfile -e dev apply       # local development
helmfile -e staging apply   # staging
helmfile -e prod apply      # production
```

That's it for most deployments. The rest of this doc covers what's happening underneath.

---

## Prerequisites

- [Helmfile](https://github.com/helmfile/helmfile) v0.160+
- [Helm](https://helm.sh/) v3.14+
- `kubectl` configured for the target cluster
- Container image registry access
- External Secrets Operator for staging/prod (see [secrets-management.md](secrets-management.md))

## Environment Scaling

Arcana automatically adjusts resources, replicas, and safety controls based on the target environment:

| | Dev | Staging | Prod |
|-|-----|---------|------|
| **Replicas** | 1 | 2 | 3 |
| **Autoscaling** | Off | 2-4 pods | 3-10 pods |
| **PDB** | Off | 1 min available | 2 min available |
| **Image pull** | IfNotPresent | Always | Always |
| **Log level** | debug | info | warn |
| **Memory** | 64-128Mi | 128-256Mi | 256-512Mi |
| **CPU** | 50-250m | 100-500m | 200-1000m |

## How It Works

```
deploy/
  helm/
    arcana-api/             # Per-service chart with base values.yaml
    arcana-engine/
    ...                     # 28 charts total
    overlays/
      values-dev.yaml       # Dev overrides
      values-staging.yaml   # Staging overrides
      values-prod.yaml      # Prod overrides
  helmfile.yaml             # Orchestrates all 28 releases
```

Helmfile merges values in order: the chart's `values.yaml` provides service-specific defaults (ports, host mappings, secrets), then the environment overlay applies resources, replicas, and autoscaling. Service-specific values take precedence.

---

## Deploying

### Development

```bash
cd deploy
helmfile -e dev apply
```

Uses `IfNotPresent` pull policy. Autoscaling and PDBs are disabled to minimize local resource usage.

### Staging

```bash
cd deploy
helmfile -e staging apply
```

Enables autoscaling (2-4 replicas per service based on CPU/memory targets). PDBs keep at least one pod available during rollouts.

### Production

```bash
cd deploy
helmfile -e prod apply
```

Aggressive autoscaling (3-10 replicas). PDBs require at least two pods available. Log level set to `warn`.

### Single Service

Deploy or update one service only:

```bash
helmfile -e staging -l name=arcana-api apply
```

### Preview Changes

See what would change before applying:

```bash
helmfile -e prod diff
```

### Full Reconciliation

Ensure cluster state matches desired state exactly:

```bash
helmfile -e prod sync
```

---

## Secrets

All environments use the `arcana-platform-secret` Kubernetes Secret. How it's populated differs.

**Development:** Applied from `deploy/backing/secrets.yaml` during `make dev`. Hardcoded credentials — never use outside local dev.

**Staging/Production:** Managed by External Secrets Operator (ESO), which syncs from your backend store (Vault, AWS Secrets Manager, GCP Secret Manager, or Azure Key Vault).

```bash
kubectl apply -f deploy/backing/external-secrets.yaml -n arcana
kubectl get secret arcana-platform-secret -n arcana
```

Force a secret refresh:

```bash
kubectl annotate externalsecret arcana-platform-secret \
  force-sync=$(date +%s) -n arcana --overwrite
```

See [secrets-management.md](secrets-management.md) for full ESO setup, rotation, and Vault configuration.

---

## Rollbacks

### Single Service

```bash
helm rollback arcana-api 0 -n arcana          # previous release
helm history arcana-api -n arcana              # list revisions
helm rollback arcana-api 3 -n arcana           # specific revision
```

### All Services

Re-apply with the previous image tag:

```bash
helmfile -e prod --set image.tag=v1.2.3-previous apply
```

### Emergency (kubectl)

If Helm is unavailable:

```bash
kubectl rollout undo deployment/arcana-api -n arcana
kubectl rollout status deployment/arcana-api -n arcana
```

---

## Blue/Green Deployment

For zero-downtime releases in production.

```bash
# Deploy new version alongside current
helmfile -e prod \
  --set fullnameOverride=arcana-api-green \
  --set image.tag=v2.0.0 \
  -l name=arcana-api apply

# Verify green
kubectl port-forward deployment/arcana-api-green 8090:8080 -n arcana
curl http://localhost:8090/healthz

# Switch traffic
kubectl patch service arcana-api -n arcana \
  -p '{"spec":{"selector":{"app.kubernetes.io/instance":"arcana-api-green"}}}'

# Teardown old version after verification
helmfile -e prod \
  --set fullnameOverride=arcana-api-blue \
  -l name=arcana-api destroy
```

---

## Monitoring

After any deployment:

```bash
# Check all pods
kubectl get pods -n arcana -l app.kubernetes.io/part-of=arcana

# Watch rollout progress
kubectl rollout status deployment -n arcana --timeout=300s

# Check for recent restarts
kubectl get pods -n arcana --sort-by='.status.containerStatuses[0].restartCount'
```

## Alerting

Prometheus alert rules in `deploy/backing/alerting-rules.yaml` cover:

| Category | Alerts |
|----------|--------|
| **Availability** | ServiceDown, HighErrorRate, HighLatencyP95/P99 |
| **Resources** | PodRestarting, PodOOMKilled, HighMemoryUsage, CPUThrottling |
| **Business** | AuthFailureSpike, RateLimitHitSpike, SlowQueries |
| **Infrastructure** | PostgresDown, RedisDown, PVCNearFull, CertExpiringSoon |

```bash
kubectl apply -f deploy/backing/alerting-rules.yaml
kubectl apply -f deploy/backing/alertmanager.yaml
```
