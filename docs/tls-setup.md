# mTLS Setup for Arcana Services

This document describes how to enable mutual TLS (mTLS) between Arcana services
using cert-manager and the internal CA infrastructure.

## Prerequisites

- Kubernetes cluster (OpenShift or vanilla k8s)
- Helm 3.x
- kubectl configured for your cluster
- The `arcana` namespace must exist

## 1. Install cert-manager

cert-manager automates certificate issuance and renewal.

```bash
# Add the Jetstack Helm repository
helm repo add jetstack https://charts.jetstack.io
helm repo update

# Install cert-manager with CRDs
helm install cert-manager jetstack/cert-manager \
  --namespace cert-manager \
  --create-namespace \
  --set crds.enabled=true \
  --version v1.17.2
```

Verify the installation:

```bash
kubectl get pods -n cert-manager
# All three pods (cert-manager, cert-manager-cainjector, cert-manager-webhook)
# should be Running.
```

## 2. Apply the Internal CA

The internal CA creates a self-signed root certificate and an issuer that signs
service certificates against it. All resources live in `deploy/backing/internal-ca.yaml`.

```bash
kubectl apply -f deploy/backing/internal-ca.yaml
```

This creates:

| Resource | Kind | Purpose |
|----------|------|---------|
| `arcana-selfsigned` | ClusterIssuer | Bootstrap issuer for the CA certificate |
| `arcana-internal-ca` | Certificate | Self-signed CA certificate (10-year lifetime) |
| `arcana-internal-issuer` | Issuer | Signs service certificates using the CA |
| `arcana-service-tls` | Certificate | Wildcard cert for `*.arcana.svc.cluster.local` |

Verify the CA is ready:

```bash
kubectl get certificate -n arcana
# Both arcana-internal-ca and arcana-service-tls should show READY=True.

kubectl get secret arcana-service-tls -n arcana
# Should exist and contain tls.crt, tls.key, and ca.crt.
```

## 3. Enable TLS per Service via Helm Values

Every Arcana Helm chart supports optional TLS through the `tls` values block.
TLS is disabled by default.

### Enable for a single service

Override `tls.enabled` when installing or upgrading the chart:

```bash
helm upgrade --install arcana-engine deploy/helm/arcana-engine \
  --namespace arcana \
  --set tls.enabled=true
```

### Enable for all services

Create a shared values override file:

```yaml
# tls-values.yaml
tls:
  enabled: true
  secretName: arcana-service-tls
  mountPath: /etc/tls
```

Apply it to every chart:

```bash
for chart in deploy/helm/arcana-*; do
  name=$(basename "$chart")
  helm upgrade --install "$name" "$chart" \
    --namespace arcana \
    -f tls-values.yaml
done
```

### Values reference

| Key | Default | Description |
|-----|---------|-------------|
| `tls.enabled` | `false` | Mount TLS certificates into the pod |
| `tls.secretName` | `arcana-service-tls` | Kubernetes Secret containing `tls.crt`, `tls.key`, and `ca.crt` |
| `tls.mountPath` | `/etc/tls` | Directory where certificates are mounted |

When enabled, the pod receives:

- `/etc/tls/tls.crt` -- the server certificate
- `/etc/tls/tls.key` -- the private key
- `/etc/tls/ca.crt` -- the CA certificate (for client verification)

### Configure the application

Go services using `pkg/server` read TLS configuration from environment variables.
Set these in the chart's `env` block:

```yaml
env:
  TLS_CERT_FILE: "/etc/tls/tls.crt"
  TLS_KEY_FILE: "/etc/tls/tls.key"
```

Python and TypeScript services should read the same environment variables in
their startup configuration.

## 4. Verify TLS is Working

### Check the certificate is mounted

```bash
# Pick any pod with TLS enabled
POD=$(kubectl get pods -n arcana -l app.kubernetes.io/name=arcana-engine -o jsonpath='{.items[0].metadata.name}')

kubectl exec -n arcana "$POD" -- ls /etc/tls/
# Should list: ca.crt  tls.crt  tls.key
```

### Test the TLS endpoint

Port-forward to the service and connect with openssl:

```bash
kubectl port-forward -n arcana svc/arcana-engine 8443:8081 &

openssl s_client -connect localhost:8443 \
  -CAfile <(kubectl get secret arcana-service-tls -n arcana -o jsonpath='{.data.ca\.crt}' | base64 -d)
```

You should see a valid certificate chain with the subject
`CN=*.arcana.svc.cluster.local` signed by `CN=arcana-internal-ca`.

### Verify from another pod

From inside the cluster, test that one service can reach another over TLS:

```bash
kubectl run tls-test --rm -it --image=curlimages/curl -- \
  curl --cacert /etc/tls/ca.crt \
       https://arcana-engine.arcana.svc.cluster.local:8081/healthz
```

## Certificate Renewal

cert-manager handles renewal automatically. The service certificate renews
30 days before expiry (configurable via `renewBefore` in the Certificate spec).
The CA certificate renews 1 year before its 10-year expiry.

Monitor certificate status:

```bash
kubectl get certificate -n arcana -w
```

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| Certificate stuck in `False` READY state | cert-manager cannot reach the issuer | Check `kubectl describe certificate -n arcana` for events |
| Pod fails to start with TLS errors | Secret not yet created | Wait for cert-manager to issue the certificate, then restart the pod |
| `x509: certificate signed by unknown authority` | Client does not trust the internal CA | Mount `ca.crt` and pass it to your HTTP client's CA pool |
| Connection refused on TLS port | `TLS_CERT_FILE` / `TLS_KEY_FILE` env vars not set | Verify the env block in your Helm values |
