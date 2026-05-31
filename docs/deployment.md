# Deployment Guide

This guide covers deploying Arcana to development, staging, and production environments using Helmfile.

## Prerequisites

- [Helmfile](https://github.com/helmfile/helmfile) v0.160+
- [Helm](https://helm.sh/) v3.14+
- `kubectl` configured for the target cluster
- Access to the container image registry
- External Secrets Operator installed in staging/production (see [secrets-management.md](secrets-management.md))

## Environment Overview

| Environment | Replicas | Autoscaling | PDB | Image Tag | Log Level |
|-------------|----------|-------------|-----|-----------|-----------|
| **dev** | 1 | Disabled | Disabled | `dev` | `debug` |
| **staging** | 2 | 2-4 pods | 1 min available | `staging` | `info` |
| **prod** | 3 | 3-10 pods | 2 min available | `latest` | `warn` |

Resource allocations scale with environment:

| Environment | Memory Request | Memory Limit | CPU Request | CPU Limit |
|-------------|---------------|--------------|-------------|-----------|
| **dev** | 64Mi | 128Mi | 50m | 250m |
| **staging** | 128Mi | 256Mi | 100m | 500m |
| **prod** | 256Mi | 512Mi | 200m | 1000m |

## Directory Structure

```
deploy/
  helm/
    arcana-api/             # Per-service chart with base values.yaml
    arcana-engine/
    ...                     # 28 charts total
    overlays/
      values-dev.yaml       # Environment-specific overrides
      values-staging.yaml
      values-prod.yaml
  helmfile.yaml             # Orchestrates all 28 releases
```

Helmfile merges values in order: the chart's `values.yaml` provides service-specific defaults (ports, host mappings, secrets), then the environment overlay applies resource limits, replica counts, and autoscaling settings. Service-specific values always take precedence where they do not conflict.

## Deploying

### Development

```bash
cd deploy
helmfile -e dev apply
```

Development uses `IfNotPresent` pull policy, so images are only pulled when missing. Autoscaling and pod disruption budgets are disabled to reduce local resource usage.

### Staging

```bash
cd deploy
helmfile -e staging apply
```

Staging uses `Always` pull policy and enables autoscaling (2-4 replicas per service based on CPU/memory targets). PDBs ensure at least one pod remains available during rollouts.

### Production

```bash
cd deploy
helmfile -e prod apply
```

Production uses `Always` pull policy with aggressive autoscaling (3-10 replicas). PDBs require at least two pods available at all times. Log level is set to `warn` to reduce noise.

### Deploying a Single Service

To deploy or update only one service:

```bash
cd deploy
helmfile -e staging -l name=arcana-api apply
```

### Diff Before Applying

Preview what will change before applying:

```bash
cd deploy
helmfile -e prod diff
```

### Syncing (Full Reconciliation)

To ensure the cluster state matches the desired state exactly:

```bash
cd deploy
helmfile -e prod sync
```

## Secret Injection

Secrets are injected differently per environment. All environments use the `arcana-platform-secret` Kubernetes Secret, but how that secret is populated varies.

### Development

Secrets are applied directly from `deploy/backing/secrets.yaml` during `make dev`. These contain hardcoded development credentials and must never be used outside local development.

### Staging and Production

Secrets are managed by External Secrets Operator (ESO), which synchronizes secrets from a backend store (HashiCorp Vault, AWS Secrets Manager, GCP Secret Manager, or Azure Key Vault) into the `arcana-platform-secret` Kubernetes Secret.

```bash
# Apply the ExternalSecret resource
kubectl apply -f deploy/backing/external-secrets.yaml -n arcana

# Verify the secret was created
kubectl get secret arcana-platform-secret -n arcana

# Force a secret refresh
kubectl annotate externalsecret arcana-platform-secret \
  force-sync=$(date +%s) -n arcana --overwrite
```

See [secrets-management.md](secrets-management.md) for full ESO setup, secret rotation, and Vault configuration.

### Per-Service Secrets

Individual services reference secrets through `envFromSecrets` in their chart values:

```yaml
envFromSecrets:
  - arcana-platform-secret
```

To add a service-specific secret, create an additional ExternalSecret and add its name to the service's `envFromSecrets` list.

## Rollback Procedures

### Rolling Back a Single Service

Helm maintains a release history. To roll back to the previous version:

```bash
helm rollback arcana-api 0 -n arcana
```

The `0` revision means "previous release." To roll back to a specific revision:

```bash
# List release history
helm history arcana-api -n arcana

# Roll back to revision 3
helm rollback arcana-api 3 -n arcana
```

### Rolling Back All Services

If a full Helmfile apply needs to be reverted:

```bash
cd deploy

# Re-apply with the previous image tag
helmfile -e prod \
  --set image.tag=v1.2.3-previous \
  apply
```

### Emergency Rollback via kubectl

If Helm is unavailable, roll back a deployment directly:

```bash
# Roll back to the previous ReplicaSet
kubectl rollout undo deployment/arcana-api -n arcana

# Roll back to a specific revision
kubectl rollout undo deployment/arcana-api --to-revision=2 -n arcana

# Check rollout status
kubectl rollout status deployment/arcana-api -n arcana
```

## Blue/Green Deployment

Arcana supports blue/green deployments for zero-downtime releases in production. This pattern runs two identical environments (blue and green) and switches traffic between them.

### Setup

Label the current production deployment as "blue":

```bash
# Tag the current release
kubectl label deployment arcana-api slot=blue -n arcana
```

### Deploy the Green Slot

Deploy the new version alongside the current one using a separate release name:

```bash
cd deploy

# Deploy green slot with new image tag
helmfile -e prod \
  --set fullnameOverride=arcana-api-green \
  --set image.tag=v2.0.0 \
  -l name=arcana-api \
  apply
```

### Verify Green

Run smoke tests against the green deployment:

```bash
# Port-forward to green and test
kubectl port-forward deployment/arcana-api-green 8090:8080 -n arcana
curl http://localhost:8090/healthz
curl http://localhost:8090/readyz
```

### Switch Traffic

Update the Service selector to point to the green deployment:

```bash
kubectl patch service arcana-api -n arcana \
  -p '{"spec":{"selector":{"app.kubernetes.io/instance":"arcana-api-green"}}}'
```

### Teardown Blue

After verifying green is healthy in production:

```bash
helmfile -e prod \
  --set fullnameOverride=arcana-api-blue \
  -l name=arcana-api \
  destroy
```

### Rollback to Blue

If green has issues, switch traffic back:

```bash
kubectl patch service arcana-api -n arcana \
  -p '{"spec":{"selector":{"app.kubernetes.io/instance":"arcana-api"}}}'
```

## Monitoring Deployments

After any deployment, verify health:

```bash
# Check all pods are running
kubectl get pods -n arcana -l app.kubernetes.io/part-of=arcana

# Check for recent restarts
kubectl get pods -n arcana --sort-by='.status.containerStatuses[0].restartCount'

# Watch rollout progress
kubectl rollout status deployment -n arcana --timeout=300s

# Check Prometheus alerts
kubectl port-forward svc/prometheus 9090:9090 -n arcana
# Open http://localhost:9090/alerts in a browser

# Check AlertManager
kubectl port-forward svc/alertmanager 9093:9093 -n arcana
# Open http://localhost:9093 in a browser
```

## Alerting

Prometheus alert rules are defined in `deploy/backing/alerting-rules.yaml` and cover four categories:

| Category | Alerts |
|----------|--------|
| **Availability** | ServiceDown, HighErrorRate, HighLatencyP95, HighLatencyP99 |
| **Resources** | PodRestarting, PodOOMKilled, HighMemoryUsage, HighCPUThrottling |
| **Business** | AuthFailureSpike, RateLimitHitSpike, DatabaseSlowQueries |
| **Infrastructure** | PostgresDown, RedisDown, PVCNearFull, CertificateExpiringSoon |

AlertManager routes critical alerts (severity=critical) and warning alerts (severity=warning) to separate webhook receivers. Critical alerts suppress matching warning alerts via inhibition rules.

To apply alert rules and AlertManager:

```bash
kubectl apply -f deploy/backing/alerting-rules.yaml
kubectl apply -f deploy/backing/alertmanager.yaml
```
