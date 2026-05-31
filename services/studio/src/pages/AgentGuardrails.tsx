import { useCallback, useEffect, useState } from "react";
import {
  Card,
  CardBody,
  CardTitle,
  Label,
  Button,
  Modal,
  ModalBody,
  ModalFooter,
  ModalHeader,
  FormGroup,
  TextInput,
  FormSelect,
  FormSelectOption,
  Alert,
  AlertActionCloseButton,
} from "@patternfly/react-core";
import { Table, Thead, Tr, Th, Tbody, Td } from "@patternfly/react-table";
import { PlusCircleIcon, TrashIcon, ShieldAltIcon } from "@patternfly/react-icons";

interface GuardrailRule {
  id: string;
  type: string;
  pattern: string;
  action: string;
  severity: string;
  agent_id: string;
  created_at: string;
}

const RULE_TYPES = [
  { value: "policy", label: "Policy — block/warn on keyword match" },
  { value: "injection", label: "Injection — prompt injection detection" },
  { value: "pii", label: "PII — personal data detection" },
  { value: "brand", label: "Brand — brand tone enforcement" },
  { value: "competitor", label: "Competitor — competitor mention blocking" },
  { value: "content", label: "Content — content safety filter" },
];

const actionColor = (a: string): "red" | "orange" | "blue" => {
  switch (a) {
    case "block": return "red";
    case "warn": return "orange";
    case "redact": return "blue";
    default: return "orange";
  }
};

const severityColor = (s: string): "red" | "orange" | "blue" | "grey" => {
  switch (s) {
    case "critical": return "red";
    case "high": return "red";
    case "medium": return "orange";
    case "low": return "blue";
    default: return "grey";
  }
};

interface AgentGuardrailsProps {
  agentName: string;
}

export const AgentGuardrails = ({ agentName }: AgentGuardrailsProps) => {
  const [rules, setRules] = useState<GuardrailRule[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [modalOpen, setModalOpen] = useState(false);
  const [newType, setNewType] = useState("policy");
  const [newPattern, setNewPattern] = useState("");
  const [newAction, setNewAction] = useState("block");
  const [newSeverity, setNewSeverity] = useState("medium");

  const fetchRules = useCallback(async () => {
    try {
      const res = await fetch(`/api/v1/rules/agent/${agentName}`);
      const data = await res.json();
      setRules(data.rules || []);
    } catch {
      setError("Failed to load guardrail rules");
    } finally {
      setLoading(false);
    }
  }, [agentName]);

  useEffect(() => {
    fetchRules();
  }, [fetchRules]);

  const handleAdd = async () => {
    if (!newPattern.trim()) return;
    setError(null);
    try {
      const res = await fetch("/api/v1/rules", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          type: newType,
          pattern: newPattern.trim(),
          action: newAction,
          severity: newSeverity,
          agent_id: agentName,
        }),
      });
      if (!res.ok) {
        setError("Failed to create rule");
        return;
      }
      const rule = await res.json();
      setRules((prev) => [...prev, rule]);
      setModalOpen(false);
      setNewPattern("");
    } catch {
      setError("Failed to create rule");
    }
  };

  const handleDelete = async (ruleId: string) => {
    try {
      await fetch(`/api/v1/rules/${ruleId}`, { method: "DELETE" });
      setRules((prev) => prev.filter((r) => r.id !== ruleId));
    } catch {
      setError("Failed to delete rule");
    }
  };

  const agentRules = rules.filter((r) => r.agent_id === agentName);
  const globalRules = rules.filter((r) => r.agent_id === "*");

  return (
    <>
      <Card>
        <CardTitle>
          <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
            <span style={{ display: "flex", alignItems: "center", gap: 8 }}>
              <ShieldAltIcon /> Guardrails
              <Label isCompact color="green">{rules.length} active</Label>
            </span>
            <Button
              variant="secondary"
              icon={<PlusCircleIcon />}
              size="sm"
              onClick={() => setModalOpen(true)}
            >
              Add Guardrail
            </Button>
          </div>
        </CardTitle>
        <CardBody>
          {error && (
            <Alert
              variant="danger"
              title={error}
              isInline
              style={{ marginBottom: 16 }}
              actionClose={<AlertActionCloseButton onClose={() => setError(null)} />}
            />
          )}

          {loading ? (
            <div style={{ padding: 16, textAlign: "center", color: "var(--pf-t--global--text--color--subtle)" }}>
              Loading rules...
            </div>
          ) : rules.length === 0 ? (
            <div style={{ padding: 16, textAlign: "center", color: "var(--pf-t--global--text--color--subtle)" }}>
              No guardrail rules configured. Add one to protect this agent.
            </div>
          ) : (
            <>
              {agentRules.length > 0 && (
                <>
                  <div style={{ fontSize: 13, fontWeight: 700, marginBottom: 8, color: "var(--pf-t--global--text--color--subtle)" }}>
                    AGENT-SPECIFIC RULES ({agentRules.length})
                  </div>
                  <Table aria-label="Agent guardrail rules" variant="compact">
                    <Thead>
                      <Tr>
                        <Th>Type</Th>
                        <Th>Pattern</Th>
                        <Th>Action</Th>
                        <Th>Severity</Th>
                        <Th />
                      </Tr>
                    </Thead>
                    <Tbody>
                      {agentRules.map((r) => (
                        <Tr key={r.id}>
                          <Td dataLabel="Type">
                            <Label isCompact color="purple">{r.type}</Label>
                          </Td>
                          <Td dataLabel="Pattern">
                            <code style={{ fontSize: 13 }}>{r.pattern}</code>
                          </Td>
                          <Td dataLabel="Action">
                            <Label isCompact color={actionColor(r.action)}>{r.action}</Label>
                          </Td>
                          <Td dataLabel="Severity">
                            <Label isCompact color={severityColor(r.severity)}>{r.severity}</Label>
                          </Td>
                          <Td dataLabel="Actions">
                            <Button
                              variant="plain"
                              isDanger
                              icon={<TrashIcon />}
                              aria-label={`Delete rule ${r.id}`}
                              onClick={() => handleDelete(r.id)}
                              size="sm"
                            />
                          </Td>
                        </Tr>
                      ))}
                    </Tbody>
                  </Table>
                </>
              )}

              {globalRules.length > 0 && (
                <>
                  <div style={{ fontSize: 13, fontWeight: 700, marginTop: agentRules.length > 0 ? 24 : 0, marginBottom: 8, color: "var(--pf-t--global--text--color--subtle)" }}>
                    GLOBAL RULES ({globalRules.length})
                  </div>
                  <Table aria-label="Global guardrail rules" variant="compact">
                    <Thead>
                      <Tr>
                        <Th>Type</Th>
                        <Th>Pattern</Th>
                        <Th>Action</Th>
                        <Th>Severity</Th>
                      </Tr>
                    </Thead>
                    <Tbody>
                      {globalRules.map((r) => (
                        <Tr key={r.id}>
                          <Td dataLabel="Type">
                            <Label isCompact color="grey">{r.type}</Label>
                          </Td>
                          <Td dataLabel="Pattern">
                            <code style={{ fontSize: 13 }}>{r.pattern}</code>
                          </Td>
                          <Td dataLabel="Action">
                            <Label isCompact color={actionColor(r.action)}>{r.action}</Label>
                          </Td>
                          <Td dataLabel="Severity">
                            <Label isCompact color={severityColor(r.severity)}>{r.severity}</Label>
                          </Td>
                        </Tr>
                      ))}
                    </Tbody>
                  </Table>
                </>
              )}
            </>
          )}
        </CardBody>
      </Card>

      <Modal isOpen={modalOpen} onClose={() => setModalOpen(false)} variant="small" aria-label="Add guardrail">
        <ModalHeader title="Add Guardrail Rule" />
        <ModalBody>
          <FormGroup label="Rule Type" fieldId="rule-type" isRequired>
            <FormSelect id="rule-type" value={newType} onChange={(_e, v) => setNewType(v)}>
              {RULE_TYPES.map((t) => (
                <FormSelectOption key={t.value} value={t.value} label={t.label} />
              ))}
            </FormSelect>
          </FormGroup>
          <FormGroup label="Pattern" fieldId="rule-pattern" isRequired style={{ marginTop: 16 }}>
            <TextInput
              id="rule-pattern"
              value={newPattern}
              onChange={(_e, v) => setNewPattern(v)}
              placeholder="e.g. competitor_name, credit card number, DROP TABLE"
            />
          </FormGroup>
          <FormGroup label="Action" fieldId="rule-action" style={{ marginTop: 16 }}>
            <FormSelect id="rule-action" value={newAction} onChange={(_e, v) => setNewAction(v)}>
              <FormSelectOption value="block" label="Block — reject the request entirely" />
              <FormSelectOption value="warn" label="Warn — allow but flag for review" />
              <FormSelectOption value="redact" label="Redact — remove matched content" />
            </FormSelect>
          </FormGroup>
          <FormGroup label="Severity" fieldId="rule-severity" style={{ marginTop: 16 }}>
            <FormSelect id="rule-severity" value={newSeverity} onChange={(_e, v) => setNewSeverity(v)}>
              <FormSelectOption value="low" label="Low" />
              <FormSelectOption value="medium" label="Medium" />
              <FormSelectOption value="high" label="High" />
              <FormSelectOption value="critical" label="Critical" />
            </FormSelect>
          </FormGroup>
          <div style={{ marginTop: 16, padding: 12, background: "var(--pf-t--global--background--color--secondary--default, #f0f0f0)", borderRadius: 8, fontSize: 13 }}>
            This rule applies only to <strong>{agentName}</strong>. Global rules (applied to all agents) can be managed from Settings.
          </div>
        </ModalBody>
        <ModalFooter>
          <Button variant="primary" onClick={handleAdd} isDisabled={!newPattern.trim()}>
            Add Rule
          </Button>
          <Button variant="link" onClick={() => setModalOpen(false)}>Cancel</Button>
        </ModalFooter>
      </Modal>
    </>
  );
};
