# Secrets Management

Arcana never stores secrets in code or config files. In development, secrets live in Kubernetes Secrets applied during setup. In production, External Secrets Operator syncs secrets from your vault of choice — zero manual handling.

## Development

The file `deploy/backing/secrets.yaml` contains hardcoded credentials for local development only. These are applied automatically by `make dev`.

**Never use these credentials in production.**

## Production Setup

Production uses the [External Secrets Operator (ESO)](https://external-secrets.io/) to pull secrets from a backend store.

### Prerequisites

1. Install ESO in your cluster:
   ```bash
   helm repo add external-secrets https://charts.external-secrets.io
   helm install external-secrets external-secrets/external-secrets \
     -n external-secrets-system --create-namespace
   ```

2. Configure your secrets backend (choose one):
   - **HashiCorp Vault** (default template in `deploy/backing/external-secrets.yaml`)
   - **AWS Secrets Manager**
   - **GCP Secret Manager**
   - **Azure Key Vault**

### Required Secrets

Populate the following secrets in your backend under the path `arcana/platform`:

| Key | Description | Requirements |
|-----|-------------|--------------|
| `postgres_user` | PostgreSQL username | — |
| `postgres_password` | PostgreSQL password | Min 16 chars, mixed case + digits + symbols |
| `postgres_db` | PostgreSQL database name | — |
| `minio_root_user` | MinIO access key | — |
| `minio_root_password` | MinIO secret key | Min 16 chars |
| `jwt_signing_key` | JWT HMAC-SHA256 signing key | Min 32 chars, cryptographically random |
| `audit_hmac_key` | Audit log tamper-detection key | Min 32 chars, cryptographically random |
| `admin_api_key` | Bootstrap admin API key | Format: `ak-admin-{random}` |
| `encryption_key` | Data-at-rest encryption key | Exactly 32 bytes |

### Generating Secrets

```bash
# Generate a cryptographically random 32-byte key
openssl rand -base64 32

# Generate a 48-char API key
echo "ak-admin-$(openssl rand -hex 20)"
```

### Deploying

```bash
# Apply the ExternalSecret (instead of secrets.yaml)
kubectl apply -f deploy/backing/external-secrets.yaml

# Verify the secret was created
kubectl get secret arcana-platform-secret -n arcana

# Set ARCANA_ENV to enable fail-closed behavior
# (API server will refuse to start without JWT_SIGNING_KEY)
kubectl set env deployment/arcana-api ARCANA_ENV=production -n arcana
```

### Secret Rotation

ESO refreshes secrets on the interval specified in `spec.refreshInterval` (default: 1 hour). To force an immediate refresh:

```bash
kubectl annotate externalsecret arcana-platform-secret \
  force-sync=$(date +%s) -n arcana --overwrite
```

### Vault-Specific Setup

1. Enable the Kubernetes auth method:
   ```bash
   vault auth enable kubernetes
   vault write auth/kubernetes/config \
     kubernetes_host="https://$KUBERNETES_HOST:6443"
   ```

2. Create a policy for Arcana:
   ```bash
   vault policy write arcana - <<EOF
   path "secret/data/arcana/*" {
     capabilities = ["read"]
   }
   EOF
   ```

3. Bind the service account:
   ```bash
   vault write auth/kubernetes/role/arcana \
     bound_service_account_names=arcana-vault-auth \
     bound_service_account_namespaces=arcana \
     policies=arcana \
     ttl=1h
   ```
