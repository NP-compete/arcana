# Security Policy

Arcana runs AI agents in production. Security is not optional.

## Supported Versions

| Version | Supported |
|---------|-----------|
| 0.x (current) | Yes |

## Reporting a Vulnerability

**Do not open a public GitHub issue for security vulnerabilities.**

Email **security@arcana.io** with:

1. Description of the vulnerability
2. Steps to reproduce
3. Affected components (CRD, service, protocol)
4. Potential impact assessment

You will receive an acknowledgment within **48 hours** and a detailed response within **7 days**.

## What's In Scope

| Area | Examples |
|------|---------|
| **CRD validation** | Bypassing schema validation or admission webhooks |
| **Sandbox escapes** | Breaking out of gVisor/Kata isolation |
| **Tenant isolation** | Cross-tenant data access or resource manipulation |
| **Protocol handling** | Injection, replay, or spoofing in MCP/A2A/ACP |
| **Guardrail bypasses** | Circumventing Ward content filters or OPA policies |
| **Auth/authz** | RBAC/ABAC policy bypasses via ArcanaRole |
| **Secret handling** | Exposure of credentials, tokens, or API keys |
| **Supply chain** | Compromised dependencies or container images |

## Defense in Depth

Arcana implements security at every layer:

- **Agent Plane** — Sandboxed execution (gVisor/Kata), per-agent network policies
- **Govern Plane** — OPA constraint templates, KubeArmor runtime enforcement, Ward input/output filtering
- **Ops Plane** — mTLS between all services, secret rotation via External Secrets Operator
- **Data Plane** — Tenant-scoped data isolation, encrypted storage at rest

For implementation details:
- [Sandbox Security](docs/sandbox-security.md) — isolation architecture and controls
- [TLS Setup](docs/tls-setup.md) — mTLS between services
- [Secrets Management](docs/secrets-management.md) — ESO, Vault, rotation
