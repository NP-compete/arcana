export const ARCANA_AGENT_YAML = `apiVersion: arcana.io/v1alpha1
kind: ArcanaAgent
metadata:
  name: my-agent
  namespace: arcana-system
spec:
  model: gpt-4o
  skills:
    - search
    - summarize
  memory:
    backend: pgvector
    ttl: 24h
  budget:
    maxTokensPerTurn: 4096
    routingStrategy: baar
  sandbox:
    runtime: gvisor
status:
  phase: Pending
`;
