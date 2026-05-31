# Sandbox Security Architecture

## Overview

The Arcana sandbox service executes untrusted user code in isolated environments.
In production, each execution request spawns an ephemeral Kubernetes pod with
gVisor runtime isolation. In development environments without a Kubernetes
cluster, the service falls back to local process execution with a sanitized
environment.

## Architecture

### Production (Kubernetes Pod per Execution)

Each code execution request creates a short-lived pod with the following properties:

- **Ephemeral lifecycle**: Pod is created, runs the code, captures output, and is
  deleted. No state persists between executions.
- **gVisor runtime**: Pods use the `arcana-sandbox` RuntimeClass backed by the
  `runsc` handler. gVisor interposes a user-space kernel between the container
  and the host, blocking direct syscalls.
- **NetworkPolicy isolation**: The `sandbox-deny-all` NetworkPolicy blocks all
  ingress and egress traffic for pods labeled `arcana.io/sandbox: "true"`.
  Sandboxed code cannot make network requests.
- **No service account**: `AutomountServiceAccountToken` is set to `false`.
  Sandboxed code has no access to the Kubernetes API.
- **Read-only root filesystem**: The container filesystem is mounted read-only.
- **Non-root execution**: Pods run as UID/GID 65534 (nobody) with `runAsNonRoot`
  enforced.
- **Seccomp profile**: The `RuntimeDefault` seccomp profile is applied to
  restrict available syscalls.
- **Capability drop**: All Linux capabilities are dropped (`drop: ["ALL"]`).

### Dev Mode Fallback (Local Execution)

When the sandbox service is not running inside a Kubernetes cluster (e.g., local
development), it falls back to executing code via `exec.CommandContext` on the
host process. This mode applies the following mitigations:

- **SanitizeEnv**: Only `PATH`, `HOME`, `LANG`, and `TZ` are exposed to child
  processes. All secret-bearing variables (database credentials, API keys, HMAC
  secrets) are stripped.
- **Temporary working directory**: Each execution runs in a fresh temp directory
  that is deleted after completion.
- **Output limits**: stdout and stderr are capped at 1MB to prevent memory
  exhaustion.
- **Timeout enforcement**: Each execution has a configurable timeout (max 30s)
  enforced via `context.WithTimeout`.

**WARNING**: Local execution mode does NOT provide security isolation. It is
intended for development and testing only. Never expose the sandbox service to
untrusted input without Kubernetes pod isolation.

## Production Requirements

### Node Setup

1. Install gVisor (`runsc`) on cluster nodes designated for sandbox execution:
   https://gvisor.dev/docs/user_guide/install/

2. Label nodes that have gVisor installed:
   ```bash
   kubectl label node <node-name> sandbox.arcana.io/enabled=true
   ```

3. Apply the RuntimeClass and supporting resources:
   ```bash
   kubectl apply -f deploy/backing/sandbox-runtime.yaml
   ```

### RuntimeClass Selection

| Environment | RuntimeClass         | Handler | Description                    |
|-------------|----------------------|---------|--------------------------------|
| Production  | `arcana-sandbox`     | `runsc` | gVisor kernel-level isolation  |
| Development | `arcana-sandbox-dev` | `runc`  | Standard OCI runtime (no gVisor) |

Set the `SANDBOX_RUNTIME_CLASS` environment variable to select the RuntimeClass.
The default is `arcana-sandbox-dev`.

## Resource Limits

### Per-Pod Limits

| Resource | Request | Limit   |
|----------|---------|---------|
| CPU      | 50m     | 100m    |
| Memory   | 32Mi    | 64Mi    |

### Namespace-Wide Quotas

| Resource         | Limit  |
|------------------|--------|
| Max pods         | 50     |
| CPU requests     | 10     |
| Memory requests  | 4Gi    |
| CPU limits       | 20     |
| Memory limits    | 8Gi    |

### Per-Container LimitRange

| Resource | Default Request | Default Limit | Max    |
|----------|-----------------|---------------|--------|
| CPU      | 50m             | 100m          | 500m   |
| Memory   | 32Mi            | 64Mi          | 256Mi  |

## Security Controls Summary

| Control                          | Mechanism                                |
|----------------------------------|------------------------------------------|
| Process isolation                | gVisor (runsc) RuntimeClass              |
| Network isolation                | NetworkPolicy `sandbox-deny-all`         |
| No K8s API access                | `AutomountServiceAccountToken: false`    |
| Filesystem isolation             | `readOnlyRootFilesystem: true`           |
| Privilege restriction            | `runAsNonRoot`, UID 65534, `drop: ALL`   |
| Syscall filtering                | Seccomp `RuntimeDefault` profile         |
| Resource exhaustion prevention   | ResourceQuota + LimitRange               |
| Output size cap                  | 1MB log read limit                       |
| Execution timeout                | `ActiveDeadlineSeconds` on pod spec      |
| Pod cleanup                      | Deferred deletion after execution        |

## RBAC

The sandbox service's ServiceAccount is granted a Role with the minimum
permissions needed to manage ephemeral execution pods:

| Resource     | Verbs                                |
|--------------|--------------------------------------|
| `pods`       | `create`, `get`, `list`, `watch`, `delete` |
| `pods/log`   | `create`, `get`, `list`, `watch`, `delete` |

No cluster-wide permissions are granted. The Role and RoleBinding are scoped to
the release namespace.

## Supported Languages

| Language              | Container Image        | Command           |
|-----------------------|------------------------|--------------------|
| Python (`python`)     | `python:3.12-alpine`   | `python3 -c CODE`  |
| JavaScript (`js`)     | `node:20-alpine`       | `node -e CODE`     |
| Shell (`bash`, `sh`)  | `alpine:3.19`          | `sh -c CODE`       |
