# Disaster Recovery Runbook

**Service:** Arcana Platform
**Last Updated:** 2026-05-31
**Owner:** Platform SRE Team

---

## 1. RTO/RPO Targets

| Metric | Target | Justification |
|--------|--------|---------------|
| **RPO** (Recovery Point Objective) | 24 hours | Daily backups at 02:00 UTC; maximum data loss is 24 hours |
| **RTO** (Recovery Time Objective) | 1 hour | Restore from backup + validate + restart services |

---

## 2. Backup Schedule

| Parameter | Value |
|-----------|-------|
| Frequency | Daily at 02:00 UTC |
| Retention | 7 days (last 7 backups kept) |
| Method | `pg_dumpall` compressed with gzip |
| Storage | PVC `backup-data` (5Gi) mounted at `/backups/` |
| CronJob | `arcana-db-backup` in namespace `arcana` |
| Verification | Weekly on Sunday at 04:00 UTC via `arcana-backup-verify` CronJob |

### Backup File Naming

```
/backups/arcana_YYYYMMDD_HHMMSS.sql.gz
```

### Verify Backups Are Running

```bash
# Check recent backup jobs
kubectl get jobs -n arcana -l app.kubernetes.io/name=db-backup --sort-by=.metadata.creationTimestamp

# Check latest backup file
kubectl exec -n arcana deploy/postgres -- ls -lht /backups/ | head -5

# Check verification job status
kubectl get jobs -n arcana -l app.kubernetes.io/name=backup-verify --sort-by=.metadata.creationTimestamp
```

---

## 3. Restore Procedure

### Prerequisites

- `kubectl` configured with cluster access
- Credentials for the `arcana` namespace (RBAC: `admin` or `cluster-admin`)
- Access to the backup PVC or an off-cluster copy of the backup file

### Step 1: Identify the Backup to Restore

```bash
# List available backups (newest first)
kubectl exec -n arcana deploy/postgres -- ls -lt /backups/*.sql.gz

# Pick the backup to restore
BACKUP_FILE="/backups/arcana_20260530_020000.sql.gz"
```

### Step 2: Scale Down Application Services

Stop all services that write to the database to prevent conflicts during restore.

```bash
# Scale down all Arcana deployments except postgres
for deploy in $(kubectl get deploy -n arcana -o name | grep -v postgres); do
  kubectl scale "$deploy" -n arcana --replicas=0
done

# Verify all pods are terminated
kubectl get pods -n arcana -l app.kubernetes.io/part-of=arcana --field-selector=status.phase=Running
```

### Step 3: Restore the Database

```bash
# Drop and recreate the database
kubectl exec -n arcana deploy/postgres -- psql -U arcana -d postgres -c "DROP DATABASE IF EXISTS arcana;"
kubectl exec -n arcana deploy/postgres -- psql -U arcana -d postgres -c "CREATE DATABASE arcana;"

# Restore from backup
kubectl exec -n arcana deploy/postgres -- sh -c "gunzip -c ${BACKUP_FILE} | psql -U arcana -d arcana -q"
```

### Step 4: Verify the Restore

```bash
# Check table counts
kubectl exec -n arcana deploy/postgres -- psql -U arcana -d arcana -c "
SELECT schemaname, tablename
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY tablename;
"

# Verify critical tables have data
for table in agents api_keys audit_log tenants; do
  echo "--- $table ---"
  kubectl exec -n arcana deploy/postgres -- psql -U arcana -d arcana -t -c "SELECT count(*) FROM $table;"
done
```

### Step 5: Restart Application Services

```bash
# Scale services back up
for deploy in $(kubectl get deploy -n arcana -o name | grep -v postgres); do
  kubectl scale "$deploy" -n arcana --replicas=1
done

# Wait for rollout
kubectl rollout status deploy -n arcana --timeout=300s
```

### Step 6: Run Smoke Tests

```bash
# Run the in-cluster smoke test
kubectl delete job arcana-smoke-test -n arcana --ignore-not-found
kubectl apply -f deploy/tests/smoke-test-job.yaml
kubectl wait --for=condition=complete job/arcana-smoke-test -n arcana --timeout=120s
kubectl logs job/arcana-smoke-test -n arcana
```

---

## 4. Verification Checklist

After restore, verify each of these before declaring recovery complete:

- [ ] All database tables present with expected row counts
- [ ] Audit log chain integrity intact (hash chain unbroken)
- [ ] All 28 service health checks pass (`/healthz` returns 200)
- [ ] All 28 service readiness checks pass (`/readyz` returns 200)
- [ ] API gateway can proxy requests to backend services
- [ ] At least one agent can be registered via the mesh service
- [ ] At least one task can be submitted via the engine service
- [ ] Authentication works in the configured auth mode (open/apikey/jwt)
- [ ] Smoke test job completes successfully

---

## 5. Failover Procedure

### Active-Passive Failover

If the primary cluster is unrecoverable:

1. **Activate standby cluster** — switch DNS or load balancer to point to the standby.
2. **Restore latest backup** — follow the restore procedure above on the standby cluster.
3. **Verify** — run the smoke tests and verification checklist.
4. **Update DNS** — point `arcana.example.com` to the standby cluster's ingress.

```bash
# Example: update Route53 (replace with your DNS provider)
aws route53 change-resource-record-sets \
  --hosted-zone-id Z1234567890 \
  --change-batch '{
    "Changes": [{
      "Action": "UPSERT",
      "ResourceRecordSet": {
        "Name": "arcana.example.com",
        "Type": "CNAME",
        "TTL": 60,
        "ResourceRecords": [{"Value": "standby-ingress.example.com"}]
      }
    }]
  }'
```

### Cluster Recovery (Primary)

Once the primary cluster is restored:

1. Restore the latest backup to the primary.
2. Verify data integrity on the primary.
3. Switch DNS back to the primary cluster.
4. Monitor for 24 hours before considering the incident resolved.

---

## 6. Communication Plan

### Escalation Path

| Severity | Who to Notify | Channel | SLA |
|----------|--------------|---------|-----|
| P1 (full outage) | On-call SRE, Engineering Lead, VP Engineering | PagerDuty, Slack `#arcana-incidents` | 15 min acknowledgment |
| P2 (partial outage) | On-call SRE, Engineering Lead | Slack `#arcana-incidents` | 30 min acknowledgment |
| P3 (degraded) | On-call SRE | Slack `#arcana-ops` | 1 hour acknowledgment |

### Notification Templates

**Incident Start:**
```
INCIDENT: Arcana Platform [P1/P2/P3]
Status: Investigating
Impact: [describe user-facing impact]
Start Time: [UTC timestamp]
Next Update: [ETA]
```

**Recovery Complete:**
```
RESOLVED: Arcana Platform [P1/P2/P3]
Duration: [X hours Y minutes]
Root Cause: [brief description]
Data Loss: [none / describe what was lost]
Post-mortem: [link to follow]
```

---

## 7. Post-Recovery Steps

After the platform is restored and verified:

1. **Data integrity audit** — verify audit log hash chain is intact:
   ```bash
   curl -s http://arcana-api:8080/api/v1/enterprise/audit/stats | jq '.chain_intact'
   ```

2. **Notify stakeholders** — send the "Recovery Complete" notification.

3. **Monitor closely** — watch for 24 hours:
   - Service health dashboards
   - Error rates in application logs
   - Database connection pool usage
   - Audit log entries are being written correctly

4. **Schedule post-mortem** — within 48 hours of resolution:
   - Timeline of events
   - Root cause analysis
   - What worked, what did not
   - Action items to prevent recurrence

5. **Update this runbook** — incorporate any lessons learned.

6. **Verify backup schedule** — confirm the daily backup CronJob is running and the next backup completes successfully.
