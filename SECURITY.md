# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| 0.x (current) | Yes |

## Reporting a Vulnerability

**Do not open a public GitHub issue for security vulnerabilities.**

To report a vulnerability, email **security@arcana.io** with:

1. Description of the vulnerability
2. Steps to reproduce
3. Affected components (CRD, service, protocol)
4. Potential impact assessment

You will receive an acknowledgment within 48 hours and a detailed response within 7 days.

## Scope

The following components are in scope for security reports:

- **CRD validation** — bypassing schema validation or admission webhooks
- **Agent sandbox escapes** — breaking out of gVisor/Kata isolation
- **Multi-tenant isolation** — cross-tenant data access or resource manipulation
- **MCP/A2A/ACP protocol handling** — injection, replay, or spoofing attacks
- **Guardrail bypasses** — circumventing `arcana-ward` content filters or OPA policies
- **Authentication/authorization** — RBAC/ABAC policy bypasses via `ArcanaRole`
- **Secret handling** — exposure of credentials, tokens, or API keys
- **Supply chain** — compromised dependencies or container images

## Security Architecture

Arcana implements defense-in-depth across its five planes:

- **Agent Plane**: Sandboxed execution (gVisor/Kata), per-agent network policies
- **Govern Plane**: OPA constraint templates, KubeArmor runtime enforcement, Ward input/output filtering
- **Ops Plane**: TLS between all services, secret rotation via external-secrets-operator
- **Data Plane**: Tenant-scoped data isolation, encrypted storage at rest

See [`docs/sandbox-security.md`](docs/sandbox-security.md), [`docs/tls-setup.md`](docs/tls-setup.md), and [`docs/secrets-management.md`](docs/secrets-management.md) for implementation details.
