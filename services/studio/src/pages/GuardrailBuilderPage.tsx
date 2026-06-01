import { useState, useEffect, useCallback, useRef } from "react";
import {
  PageSection,
  Title,
  Content,
  Divider,
  Card,
  CardBody,
  Button,
  Label,
  FormGroup,
  FormSelect,
  FormSelectOption,
  TextInput,
  TextArea,
  Checkbox,
  Alert,
  Spinner,
  Modal,
  ModalHeader,
  ModalBody,
  ModalFooter,
  ModalVariant,
} from "@patternfly/react-core";
import {
  ShieldAltIcon,
  TrashIcon,
  GripVerticalIcon,
  PlusCircleIcon,
  ExportIcon,
  ImportIcon,
  FlaskIcon,
  SaveIcon,
  ExclamationTriangleIcon,
  LockIcon,
  BanIcon,
  FilterIcon,
  TagIcon,
  CogIcon,
  CodeIcon,
  OutlinedMoneyBillAltIcon,
} from "@patternfly/react-icons";

/* ------------------------------------------------------------------ */
/*  Types                                                              */
/* ------------------------------------------------------------------ */

type RuleAction = "block" | "warn" | "log" | "escalate";
type RuleSeverity = "low" | "medium" | "high" | "critical";

interface RuleTypeDefinition {
  key: string;
  label: string;
  description: string;
  color: string;
  icon: React.ReactNode;
}

interface PiiConfig {
  ssn: boolean;
  cc: boolean;
  phone: boolean;
  email: boolean;
  address: boolean;
}

interface ToxicityConfig {
  sensitivity: "low" | "medium" | "high";
}

interface BrandToneConfig {
  guidelines: string;
  competitors: string[];
}

interface TopicRestrictionConfig {
  topics: string[];
}

interface PromptInjectionConfig {
  canaryTokens: boolean;
}

interface OutputValidationConfig {
  jsonSchema: string;
  maxLength: number;
}

interface CostGuardConfig {
  budgetThreshold: number;
}

interface CustomRuleConfig {
  regex: string;
  matchAction: string;
}

type RuleConfig =
  | PiiConfig
  | ToxicityConfig
  | BrandToneConfig
  | TopicRestrictionConfig
  | PromptInjectionConfig
  | OutputValidationConfig
  | CostGuardConfig
  | CustomRuleConfig;

interface GuardrailRule {
  id: string;
  type: string;
  action: RuleAction;
  severity: RuleSeverity;
  config: RuleConfig;
  enabled: boolean;
}

interface EvaluationResult {
  rule_id: string;
  rule_type: string;
  triggered: boolean;
  action: string;
  details: string;
}

/* ------------------------------------------------------------------ */
/*  Constants                                                          */
/* ------------------------------------------------------------------ */

const RULE_TYPES: RuleTypeDefinition[] = [
  { key: "pii", label: "PII Detection", description: "Block SSN, credit cards, phones", color: "#e53e3e", icon: <LockIcon /> },
  { key: "toxicity", label: "Toxicity Filter", description: "Hate speech, harassment", color: "#d69e2e", icon: <ExclamationTriangleIcon /> },
  { key: "brand_tone", label: "Brand Tone", description: "Off-brand language, competitor mentions", color: "#805ad5", icon: <TagIcon /> },
  { key: "topic_restriction", label: "Topic Restriction", description: "Political, medical, legal", color: "#3182ce", icon: <BanIcon /> },
  { key: "prompt_injection", label: "Prompt Injection", description: "Jailbreak attempts, instruction override", color: "#e53e3e", icon: <ShieldAltIcon /> },
  { key: "output_validation", label: "Output Validation", description: "JSON schema, max length", color: "#38a169", icon: <FilterIcon /> },
  { key: "cost_guard", label: "Cost Guard", description: "Token budget exceeded", color: "#ed8936", icon: <OutlinedMoneyBillAltIcon /> },
  { key: "custom", label: "Custom Rule", description: "Regex pattern match", color: "#718096", icon: <CodeIcon /> },
];

const DEFAULT_CONFIGS: Record<string, RuleConfig> = {
  pii: { ssn: true, cc: true, phone: true, email: false, address: false } as PiiConfig,
  toxicity: { sensitivity: "medium" } as ToxicityConfig,
  brand_tone: { guidelines: "", competitors: [] } as BrandToneConfig,
  topic_restriction: { topics: [] } as TopicRestrictionConfig,
  prompt_injection: { canaryTokens: true } as PromptInjectionConfig,
  output_validation: { jsonSchema: "", maxLength: 4096 } as OutputValidationConfig,
  cost_guard: { budgetThreshold: 10000 } as CostGuardConfig,
  custom: { regex: "", matchAction: "flag" } as CustomRuleConfig,
};

const ACTION_OPTIONS: { value: RuleAction; label: string; color: string }[] = [
  { value: "block", label: "Block", color: "#e53e3e" },
  { value: "warn", label: "Warn", color: "#d69e2e" },
  { value: "log", label: "Log", color: "#3182ce" },
  { value: "escalate", label: "Escalate", color: "#805ad5" },
];

const SEVERITY_OPTIONS: { value: RuleSeverity; label: string; color: string }[] = [
  { value: "low", label: "Low", color: "#3182ce" },
  { value: "medium", label: "Medium", color: "#d69e2e" },
  { value: "high", label: "High", color: "#ed8936" },
  { value: "critical", label: "Critical", color: "#e53e3e" },
];

const actionLabelColor = (a: RuleAction): "red" | "yellow" | "blue" | "purple" => {
  switch (a) {
    case "block": return "red";
    case "warn": return "yellow";
    case "log": return "blue";
    case "escalate": return "purple";
  }
};

const severityLabelColor = (s: RuleSeverity): "blue" | "yellow" | "orange" | "red" => {
  switch (s) {
    case "low": return "blue";
    case "medium": return "yellow";
    case "high": return "orange";
    case "critical": return "red";
  }
};

const ruleTypeColor = (type: string): string => {
  return RULE_TYPES.find((rt) => rt.key === type)?.color ?? "#718096";
};

const ruleTypeLabel = (type: string): string => {
  return RULE_TYPES.find((rt) => rt.key === type)?.label ?? type;
};

const configSummary = (rule: GuardrailRule): string => {
  const c = rule.config;
  switch (rule.type) {
    case "pii": {
      const p = c as PiiConfig;
      const items = [];
      if (p.ssn) items.push("SSN");
      if (p.cc) items.push("CC");
      if (p.phone) items.push("Phone");
      if (p.email) items.push("Email");
      if (p.address) items.push("Address");
      return items.length > 0 ? items.join(", ") : "None selected";
    }
    case "toxicity":
      return `Sensitivity: ${(c as ToxicityConfig).sensitivity}`;
    case "brand_tone": {
      const bt = c as BrandToneConfig;
      const parts = [];
      if (bt.guidelines) parts.push("Guidelines set");
      if (bt.competitors.length > 0) parts.push(`${bt.competitors.length} competitors`);
      return parts.length > 0 ? parts.join(", ") : "Not configured";
    }
    case "topic_restriction": {
      const tr = c as TopicRestrictionConfig;
      return tr.topics.length > 0 ? tr.topics.join(", ") : "No topics";
    }
    case "prompt_injection": {
      const pi = c as PromptInjectionConfig;
      return pi.canaryTokens ? "Canary tokens enabled" : "Canary tokens disabled";
    }
    case "output_validation": {
      const ov = c as OutputValidationConfig;
      const parts = [];
      if (ov.jsonSchema) parts.push("Schema set");
      parts.push(`Max ${ov.maxLength} chars`);
      return parts.join(", ");
    }
    case "cost_guard":
      return `Budget: ${(c as CostGuardConfig).budgetThreshold} tokens`;
    case "custom": {
      const cr = c as CustomRuleConfig;
      return cr.regex ? `/${cr.regex}/` : "No pattern";
    }
    default:
      return "";
  }
};

let ruleIdCounter = 0;
const generateRuleId = (): string => {
  ruleIdCounter += 1;
  return `rule-${Date.now()}-${ruleIdCounter}`;
};

/* ------------------------------------------------------------------ */
/*  Component                                                          */
/* ------------------------------------------------------------------ */

export const GuardrailBuilderPage = () => {
  const [agents, setAgents] = useState<string[]>([]);
  const [selectedAgent, setSelectedAgent] = useState("");
  const [rules, setRules] = useState<GuardrailRule[]>([]);
  const [selectedRuleId, setSelectedRuleId] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saveMessage, setSaveMessage] = useState<{ type: "success" | "danger"; text: string } | null>(null);

  const [testModalOpen, setTestModalOpen] = useState(false);
  const [testInput, setTestInput] = useState("");
  const [testResults, setTestResults] = useState<EvaluationResult[] | null>(null);
  const [testRunning, setTestRunning] = useState(false);

  const [importExportModalOpen, setImportExportModalOpen] = useState(false);
  const [importExportMode, setImportExportMode] = useState<"import" | "export">("export");
  const [importJson, setImportJson] = useState("");
  const [importError, setImportError] = useState<string | null>(null);

  const [dragIndex, setDragIndex] = useState<number | null>(null);
  const [dragOverIndex, setDragOverIndex] = useState<number | null>(null);
  const dragItemRef = useRef<number | null>(null);

  const selectedRule = rules.find((r) => r.id === selectedRuleId) ?? null;

  /* ---- Fetch agents ---- */
  useEffect(() => {
    fetch("/api/v1/agents")
      .then((r) => r.json())
      .then((d) => {
        const names: string[] = (d.agents ?? []).map((a: { name: string }) => a.name);
        setAgents(names);
        if (names.length > 0 && !selectedAgent) {
          setSelectedAgent(names[0]);
        }
      })
      .catch(() => {});
  }, []);

  /* ---- Fetch rules for selected agent ---- */
  const fetchRules = useCallback(async () => {
    if (!selectedAgent) return;
    setLoading(true);
    setSaveMessage(null);
    try {
      const res = await fetch(`/api/v1/ward/agents/${encodeURIComponent(selectedAgent)}/rules`);
      if (res.ok) {
        const data = await res.json();
        setRules(data.rules ?? []);
      } else {
        setRules([]);
      }
    } catch {
      setRules([]);
    } finally {
      setLoading(false);
    }
  }, [selectedAgent]);

  useEffect(() => {
    fetchRules();
  }, [fetchRules]);

  /* ---- Save rules ---- */
  const handleSave = async () => {
    if (!selectedAgent) return;
    setSaving(true);
    setSaveMessage(null);
    try {
      const res = await fetch(`/api/v1/ward/agents/${encodeURIComponent(selectedAgent)}/rules`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ rules }),
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data.detail ?? data.error ?? `HTTP ${res.status}`);
      }
      setSaveMessage({ type: "success", text: "Rules saved successfully" });
    } catch (e) {
      setSaveMessage({ type: "danger", text: e instanceof Error ? e.message : "Failed to save rules" });
    } finally {
      setSaving(false);
    }
  };

  /* ---- Test rules ---- */
  const handleTest = async () => {
    if (!testInput.trim()) return;
    setTestRunning(true);
    setTestResults(null);
    try {
      const res = await fetch("/api/v1/ward/evaluate", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ input: testInput, rules }),
      });
      if (res.ok) {
        const data = await res.json();
        setTestResults(data.results ?? []);
      } else {
        setTestResults([]);
      }
    } catch {
      setTestResults([]);
    } finally {
      setTestRunning(false);
    }
  };

  /* ---- Add rule from palette ---- */
  const addRule = (typeKey: string) => {
    const newRule: GuardrailRule = {
      id: generateRuleId(),
      type: typeKey,
      action: "block",
      severity: "medium",
      config: structuredClone(DEFAULT_CONFIGS[typeKey]) ?? {},
      enabled: true,
    };
    setRules((prev) => [...prev, newRule]);
    setSelectedRuleId(newRule.id);
  };

  /* ---- Delete rule ---- */
  const deleteRule = (id: string) => {
    setRules((prev) => prev.filter((r) => r.id !== id));
    if (selectedRuleId === id) setSelectedRuleId(null);
  };

  /* ---- Update rule config ---- */
  const updateRule = (id: string, patch: Partial<GuardrailRule>) => {
    setRules((prev) =>
      prev.map((r) => (r.id === id ? { ...r, ...patch } : r)),
    );
  };

  const updateConfig = (id: string, configPatch: Partial<RuleConfig>) => {
    setRules((prev) =>
      prev.map((r) =>
        r.id === id ? { ...r, config: { ...r.config, ...configPatch } } : r,
      ),
    );
  };

  /* ---- Drag-and-drop reorder ---- */
  const handleDragStart = (index: number) => {
    dragItemRef.current = index;
    setDragIndex(index);
  };

  const handleDragOver = (e: React.DragEvent, index: number) => {
    e.preventDefault();
    setDragOverIndex(index);
  };

  const handleDrop = (index: number) => {
    const from = dragItemRef.current;
    if (from === null || from === index) {
      setDragIndex(null);
      setDragOverIndex(null);
      return;
    }
    setRules((prev) => {
      const updated = [...prev];
      const [moved] = updated.splice(from, 1);
      updated.splice(index, 0, moved);
      return updated;
    });
    setDragIndex(null);
    setDragOverIndex(null);
    dragItemRef.current = null;
  };

  const handleDragEnd = () => {
    setDragIndex(null);
    setDragOverIndex(null);
    dragItemRef.current = null;
  };

  /* ---- Import / Export ---- */
  const handleExport = () => {
    setImportExportMode("export");
    setImportJson(JSON.stringify(rules, null, 2));
    setImportError(null);
    setImportExportModalOpen(true);
  };

  const handleImportOpen = () => {
    setImportExportMode("import");
    setImportJson("");
    setImportError(null);
    setImportExportModalOpen(true);
  };

  const handleImportApply = () => {
    try {
      const parsed = JSON.parse(importJson);
      if (!Array.isArray(parsed)) {
        setImportError("JSON must be an array of rule objects");
        return;
      }
      setRules(parsed);
      setImportExportModalOpen(false);
      setSaveMessage({ type: "success", text: "Rules imported. Remember to save." });
    } catch {
      setImportError("Invalid JSON");
    }
  };

  /* ---- Config editor ---- */
  const renderConfigEditor = (rule: GuardrailRule) => {
    const c = rule.config;
    switch (rule.type) {
      case "pii": {
        const pii = c as PiiConfig;
        return (
          <>
            <FormGroup label="Detect" fieldId="pii-checks">
              {(["ssn", "cc", "phone", "email", "address"] as const).map((field) => (
                <Checkbox
                  key={field}
                  id={`pii-${field}`}
                  label={field.toUpperCase()}
                  isChecked={pii[field]}
                  onChange={(_e, checked) => updateConfig(rule.id, { [field]: checked })}
                  style={{ marginBottom: 4 }}
                />
              ))}
            </FormGroup>
          </>
        );
      }
      case "toxicity": {
        const tox = c as ToxicityConfig;
        return (
          <FormGroup label="Sensitivity" fieldId="tox-sensitivity">
            <div style={{ display: "flex", gap: 8, marginTop: 8 }}>
              {(["low", "medium", "high"] as const).map((level) => (
                <button
                  key={level}
                  type="button"
                  onClick={() => updateConfig(rule.id, { sensitivity: level })}
                  style={{
                    flex: 1,
                    padding: "8px 12px",
                    borderRadius: 6,
                    border: tox.sensitivity === level
                      ? "2px solid #5b8def"
                      : "1px solid rgba(255,255,255,0.12)",
                    background: tox.sensitivity === level
                      ? "rgba(91,141,239,0.12)"
                      : "rgba(255,255,255,0.04)",
                    color: tox.sensitivity === level ? "#5b8def" : "#8b95a5",
                    cursor: "pointer",
                    fontSize: 13,
                    fontWeight: 600,
                    textTransform: "capitalize",
                  }}
                >
                  {level}
                </button>
              ))}
            </div>
          </FormGroup>
        );
      }
      case "brand_tone": {
        const bt = c as BrandToneConfig;
        return (
          <>
            <FormGroup label="Brand Guidelines" fieldId="brand-guidelines">
              <TextArea
                id="brand-guidelines"
                value={bt.guidelines}
                onChange={(_e, v) => updateConfig(rule.id, { guidelines: v })}
                rows={3}
                placeholder="Describe your brand voice..."
              />
            </FormGroup>
            <FormGroup label="Competitor Names" fieldId="brand-competitors" style={{ marginTop: 12 }}>
              <TextInput
                id="brand-competitors"
                value={bt.competitors.join(", ")}
                onChange={(_e, v) =>
                  updateConfig(rule.id, {
                    competitors: v.split(",").map((s) => s.trim()).filter(Boolean),
                  })
                }
                placeholder="Comma-separated list"
              />
            </FormGroup>
          </>
        );
      }
      case "topic_restriction": {
        const tr = c as TopicRestrictionConfig;
        return (
          <FormGroup label="Restricted Topics" fieldId="topic-tags">
            <TextInput
              id="topic-tags"
              value={tr.topics.join(", ")}
              onChange={(_e, v) =>
                updateConfig(rule.id, {
                  topics: v.split(",").map((s) => s.trim()).filter(Boolean),
                })
              }
              placeholder="political, medical, legal"
            />
          </FormGroup>
        );
      }
      case "prompt_injection": {
        const pi = c as PromptInjectionConfig;
        return (
          <FormGroup label="Options" fieldId="pi-canary">
            <Checkbox
              id="pi-canary"
              label="Enable canary tokens"
              isChecked={pi.canaryTokens}
              onChange={(_e, checked) => updateConfig(rule.id, { canaryTokens: checked })}
            />
          </FormGroup>
        );
      }
      case "output_validation": {
        const ov = c as OutputValidationConfig;
        return (
          <>
            <FormGroup label="JSON Schema" fieldId="ov-schema">
              <TextArea
                id="ov-schema"
                value={ov.jsonSchema}
                onChange={(_e, v) => updateConfig(rule.id, { jsonSchema: v })}
                rows={4}
                placeholder='{"type": "object", ...}'
                style={{ fontFamily: "var(--pf-t--global--font--family--mono)", fontSize: 12 }}
              />
            </FormGroup>
            <FormGroup label="Max Output Length" fieldId="ov-maxlen" style={{ marginTop: 12 }}>
              <TextInput
                id="ov-maxlen"
                type="number"
                value={String(ov.maxLength)}
                onChange={(_e, v) => updateConfig(rule.id, { maxLength: parseInt(v, 10) || 0 })}
              />
            </FormGroup>
          </>
        );
      }
      case "cost_guard": {
        const cg = c as CostGuardConfig;
        return (
          <FormGroup label="Token Budget Threshold" fieldId="cg-budget">
            <TextInput
              id="cg-budget"
              type="number"
              value={String(cg.budgetThreshold)}
              onChange={(_e, v) => updateConfig(rule.id, { budgetThreshold: parseInt(v, 10) || 0 })}
            />
          </FormGroup>
        );
      }
      case "custom": {
        const cr = c as CustomRuleConfig;
        return (
          <>
            <FormGroup label="Regex Pattern" fieldId="cr-regex">
              <TextInput
                id="cr-regex"
                value={cr.regex}
                onChange={(_e, v) => updateConfig(rule.id, { regex: v })}
                placeholder="\\b(badword)\\b"
                style={{ fontFamily: "var(--pf-t--global--font--family--mono)", fontSize: 12 }}
              />
            </FormGroup>
            <FormGroup label="Match Action" fieldId="cr-match-action" style={{ marginTop: 12 }}>
              <TextInput
                id="cr-match-action"
                value={cr.matchAction}
                onChange={(_e, v) => updateConfig(rule.id, { matchAction: v })}
                placeholder="flag"
              />
            </FormGroup>
          </>
        );
      }
      default:
        return null;
    }
  };

  return (
    <>
      <PageSection hasBodyWrapper={false}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
          <div>
            <Title headingLevel="h1" size="2xl">
              Guardrail Builder
            </Title>
            <Content component="p" style={{ marginTop: 4 }}>
              Visual editor for Ward guardrail rules. Drag rules from the palette, configure, and deploy.
            </Content>
          </div>
          <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
            <FormSelect
              value={selectedAgent}
              onChange={(_e, v) => setSelectedAgent(v)}
              aria-label="Select agent"
              style={{ width: 200 }}
            >
              {agents.length === 0 && <FormSelectOption value="" label="No agents" isDisabled />}
              {agents.map((a) => (
                <FormSelectOption key={a} value={a} label={a} />
              ))}
            </FormSelect>
            <Button
              variant="secondary"
              icon={<ImportIcon />}
              onClick={handleImportOpen}
            >
              Import
            </Button>
            <Button
              variant="secondary"
              icon={<ExportIcon />}
              onClick={handleExport}
            >
              Export
            </Button>
            <Button
              variant="secondary"
              icon={<FlaskIcon />}
              onClick={() => { setTestInput(""); setTestResults(null); setTestModalOpen(true); }}
            >
              Test Rules
            </Button>
            <Button
              variant="primary"
              icon={<SaveIcon />}
              onClick={handleSave}
              isDisabled={saving || !selectedAgent}
              isLoading={saving}
            >
              Save Rules
            </Button>
          </div>
        </div>
      </PageSection>
      <Divider />

      {saveMessage && (
        <PageSection hasBodyWrapper={false} style={{ paddingBottom: 0 }}>
          <Alert
            variant={saveMessage.type}
            title={saveMessage.text}
            isInline
            timeout={5000}
            onTimeout={() => setSaveMessage(null)}
          />
        </PageSection>
      )}

      <PageSection hasBodyWrapper={false} style={{ padding: 0 }}>
        {loading ? (
          <div style={{ textAlign: "center", padding: 60 }}>
            <Spinner size="xl" />
          </div>
        ) : (
          <div style={{ display: "flex", height: "calc(100vh - 200px)", minHeight: 500 }}>
            {/* ---- LEFT PANEL: Rule Palette ---- */}
            <div
              style={{
                width: 250,
                flexShrink: 0,
                borderRight: "1px solid rgba(255,255,255,0.08)",
                padding: 16,
                overflowY: "auto",
                background: "rgba(255,255,255,0.02)",
              }}
            >
              <div
                style={{
                  fontSize: 11,
                  fontWeight: 700,
                  letterSpacing: "1.5px",
                  color: "#8b95a5",
                  marginBottom: 12,
                  textTransform: "uppercase",
                }}
              >
                Rule Palette
              </div>
              {RULE_TYPES.map((rt) => (
                <button
                  key={rt.key}
                  type="button"
                  onClick={() => addRule(rt.key)}
                  style={{
                    display: "flex",
                    alignItems: "flex-start",
                    gap: 10,
                    width: "100%",
                    padding: "10px 12px",
                    marginBottom: 6,
                    borderRadius: 8,
                    border: "1px solid rgba(255,255,255,0.08)",
                    background: "rgba(255,255,255,0.04)",
                    cursor: "pointer",
                    textAlign: "left",
                    transition: "all 0.15s",
                    color: "#c5cdd8",
                  }}
                  onMouseEnter={(e) => {
                    e.currentTarget.style.borderColor = rt.color;
                    e.currentTarget.style.background = `${rt.color}10`;
                  }}
                  onMouseLeave={(e) => {
                    e.currentTarget.style.borderColor = "rgba(255,255,255,0.08)";
                    e.currentTarget.style.background = "rgba(255,255,255,0.04)";
                  }}
                >
                  <div
                    style={{
                      width: 28,
                      height: 28,
                      borderRadius: 6,
                      background: `${rt.color}18`,
                      color: rt.color,
                      display: "flex",
                      alignItems: "center",
                      justifyContent: "center",
                      flexShrink: 0,
                      fontSize: 13,
                    }}
                  >
                    {rt.icon}
                  </div>
                  <div>
                    <div style={{ fontSize: 13, fontWeight: 600 }}>{rt.label}</div>
                    <div style={{ fontSize: 11, color: "#6b7585", marginTop: 2 }}>{rt.description}</div>
                  </div>
                </button>
              ))}
              <div style={{ marginTop: 16, padding: "10px 12px", borderRadius: 8, background: "rgba(91,141,239,0.06)", fontSize: 12, color: "#6b7585", lineHeight: 1.5 }}>
                <PlusCircleIcon style={{ marginRight: 4 }} />
                Click a rule type to add it to the pipeline
              </div>
            </div>

            {/* ---- CENTER PANEL: Rule Pipeline ---- */}
            <div
              style={{
                flex: 1,
                padding: 20,
                overflowY: "auto",
                background: "rgba(255,255,255,0.01)",
              }}
            >
              <div
                style={{
                  fontSize: 11,
                  fontWeight: 700,
                  letterSpacing: "1.5px",
                  color: "#8b95a5",
                  marginBottom: 12,
                  textTransform: "uppercase",
                }}
              >
                Rule Pipeline ({rules.length} rules)
              </div>
              {rules.length === 0 ? (
                <div
                  style={{
                    padding: 48,
                    textAlign: "center",
                    borderRadius: 12,
                    border: "2px dashed rgba(255,255,255,0.08)",
                    color: "#6b7585",
                  }}
                >
                  <ShieldAltIcon style={{ fontSize: 32, marginBottom: 12, opacity: 0.3 }} />
                  <div style={{ fontSize: 14, fontWeight: 500 }}>No rules in pipeline</div>
                  <div style={{ fontSize: 12, marginTop: 4 }}>
                    Click rule types in the palette to add them here
                  </div>
                </div>
              ) : (
                <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
                  {rules.map((rule, index) => (
                    <div
                      key={rule.id}
                      draggable
                      onDragStart={() => handleDragStart(index)}
                      onDragOver={(e) => handleDragOver(e, index)}
                      onDrop={() => handleDrop(index)}
                      onDragEnd={handleDragEnd}
                      onClick={() => setSelectedRuleId(rule.id)}
                      style={{
                        display: "flex",
                        alignItems: "center",
                        gap: 10,
                        padding: "10px 14px",
                        borderRadius: 8,
                        border: selectedRuleId === rule.id
                          ? `2px solid ${ruleTypeColor(rule.type)}`
                          : "1px solid rgba(255,255,255,0.08)",
                        borderLeft: `4px solid ${ruleTypeColor(rule.type)}`,
                        background: selectedRuleId === rule.id
                          ? `${ruleTypeColor(rule.type)}08`
                          : dragOverIndex === index
                            ? "rgba(91,141,239,0.08)"
                            : "rgba(255,255,255,0.03)",
                        cursor: "pointer",
                        opacity: dragIndex === index ? 0.4 : 1,
                        transition: "all 0.15s",
                      }}
                    >
                      <div
                        style={{
                          cursor: "grab",
                          color: "#4a5568",
                          display: "flex",
                          alignItems: "center",
                          flexShrink: 0,
                        }}
                      >
                        <GripVerticalIcon />
                      </div>
                      <div
                        style={{
                          width: 26,
                          height: 26,
                          borderRadius: 6,
                          background: `${ruleTypeColor(rule.type)}18`,
                          color: ruleTypeColor(rule.type),
                          display: "flex",
                          alignItems: "center",
                          justifyContent: "center",
                          fontSize: 12,
                          flexShrink: 0,
                        }}
                      >
                        {RULE_TYPES.find((rt) => rt.key === rule.type)?.icon}
                      </div>
                      <div style={{ flex: 1, minWidth: 0 }}>
                        <div style={{ fontSize: 13, fontWeight: 600, color: "#e2e8f0" }}>
                          {ruleTypeLabel(rule.type)}
                        </div>
                        <div
                          style={{
                            fontSize: 11,
                            color: "#6b7585",
                            overflow: "hidden",
                            textOverflow: "ellipsis",
                            whiteSpace: "nowrap",
                          }}
                        >
                          {configSummary(rule)}
                        </div>
                      </div>
                      <Label isCompact color={actionLabelColor(rule.action)}>
                        {rule.action}
                      </Label>
                      <Button
                        variant="plain"
                        icon={<TrashIcon />}
                        aria-label={`Delete rule ${rule.id}`}
                        onClick={(e) => {
                          e.stopPropagation();
                          deleteRule(rule.id);
                        }}
                        isDanger
                        size="sm"
                      />
                    </div>
                  ))}
                </div>
              )}
            </div>

            {/* ---- RIGHT PANEL: Rule Configuration ---- */}
            <div
              style={{
                width: 300,
                flexShrink: 0,
                borderLeft: "1px solid rgba(255,255,255,0.08)",
                padding: 16,
                overflowY: "auto",
                background: "rgba(255,255,255,0.02)",
              }}
            >
              <div
                style={{
                  fontSize: 11,
                  fontWeight: 700,
                  letterSpacing: "1.5px",
                  color: "#8b95a5",
                  marginBottom: 12,
                  textTransform: "uppercase",
                }}
              >
                Configuration
              </div>
              {selectedRule ? (
                <Card
                  style={{
                    background: "rgba(255,255,255,0.03)",
                    border: "1px solid rgba(255,255,255,0.08)",
                    borderRadius: 10,
                  }}
                >
                  <CardBody>
                    <div
                      style={{
                        display: "flex",
                        alignItems: "center",
                        gap: 8,
                        marginBottom: 16,
                      }}
                    >
                      <div
                        style={{
                          width: 28,
                          height: 28,
                          borderRadius: 6,
                          background: `${ruleTypeColor(selectedRule.type)}18`,
                          color: ruleTypeColor(selectedRule.type),
                          display: "flex",
                          alignItems: "center",
                          justifyContent: "center",
                          fontSize: 13,
                        }}
                      >
                        {RULE_TYPES.find((rt) => rt.key === selectedRule.type)?.icon}
                      </div>
                      <span style={{ fontWeight: 700, fontSize: 15, color: "#e2e8f0" }}>
                        {ruleTypeLabel(selectedRule.type)}
                      </span>
                    </div>

                    {renderConfigEditor(selectedRule)}

                    <Divider style={{ margin: "16px 0" }} />

                    <FormGroup label="Action" fieldId="rule-action">
                      <div style={{ display: "flex", gap: 6, flexWrap: "wrap" }}>
                        {ACTION_OPTIONS.map((opt) => (
                          <button
                            key={opt.value}
                            type="button"
                            onClick={() => updateRule(selectedRule.id, { action: opt.value })}
                            style={{
                              padding: "6px 14px",
                              borderRadius: 6,
                              border: selectedRule.action === opt.value
                                ? `2px solid ${opt.color}`
                                : "1px solid rgba(255,255,255,0.12)",
                              background: selectedRule.action === opt.value
                                ? `${opt.color}15`
                                : "rgba(255,255,255,0.04)",
                              color: selectedRule.action === opt.value ? opt.color : "#8b95a5",
                              cursor: "pointer",
                              fontSize: 12,
                              fontWeight: 600,
                            }}
                          >
                            {opt.label}
                          </button>
                        ))}
                      </div>
                    </FormGroup>

                    <FormGroup label="Severity" fieldId="rule-severity" style={{ marginTop: 12 }}>
                      <div style={{ display: "flex", gap: 6, flexWrap: "wrap" }}>
                        {SEVERITY_OPTIONS.map((opt) => (
                          <button
                            key={opt.value}
                            type="button"
                            onClick={() => updateRule(selectedRule.id, { severity: opt.value })}
                            style={{
                              padding: "6px 14px",
                              borderRadius: 6,
                              border: selectedRule.severity === opt.value
                                ? `2px solid ${opt.color}`
                                : "1px solid rgba(255,255,255,0.12)",
                              background: selectedRule.severity === opt.value
                                ? `${opt.color}15`
                                : "rgba(255,255,255,0.04)",
                              color: selectedRule.severity === opt.value ? opt.color : "#8b95a5",
                              cursor: "pointer",
                              fontSize: 12,
                              fontWeight: 600,
                            }}
                          >
                            {opt.label}
                          </button>
                        ))}
                      </div>
                    </FormGroup>
                  </CardBody>
                </Card>
              ) : (
                <div
                  style={{
                    padding: 32,
                    textAlign: "center",
                    color: "#6b7585",
                    borderRadius: 10,
                    border: "1px dashed rgba(255,255,255,0.08)",
                  }}
                >
                  <CogIcon style={{ fontSize: 28, marginBottom: 8, opacity: 0.3 }} />
                  <div style={{ fontSize: 13 }}>Select a rule to configure</div>
                </div>
              )}
            </div>
          </div>
        )}
      </PageSection>

      {/* ---- Test Rules Modal ---- */}
      <Modal
        variant={ModalVariant.medium}
        isOpen={testModalOpen}
        onClose={() => setTestModalOpen(false)}
        aria-labelledby="test-rules-title"
      >
        <ModalHeader title="Test Guardrail Rules" labelId="test-rules-title" />
        <ModalBody>
          <FormGroup label="Test Input" fieldId="test-input" isRequired>
            <TextArea
              id="test-input"
              value={testInput}
              onChange={(_e, v) => setTestInput(v)}
              rows={4}
              placeholder="Enter text to test against the rule pipeline..."
            />
          </FormGroup>
          {testResults !== null && (
            <div style={{ marginTop: 16 }}>
              <div
                style={{
                  fontSize: 12,
                  fontWeight: 700,
                  letterSpacing: "1px",
                  color: "#8b95a5",
                  marginBottom: 8,
                  textTransform: "uppercase",
                }}
              >
                Results ({testResults.length} rules evaluated)
              </div>
              {testResults.length === 0 ? (
                <Alert variant="info" title="No rules were triggered" isInline />
              ) : (
                testResults.map((r, i) => (
                  <div
                    key={i}
                    style={{
                      padding: "8px 12px",
                      marginBottom: 4,
                      borderRadius: 6,
                      background: r.triggered ? "rgba(229,62,62,0.08)" : "rgba(56,161,105,0.08)",
                      border: `1px solid ${r.triggered ? "rgba(229,62,62,0.2)" : "rgba(56,161,105,0.2)"}`,
                      display: "flex",
                      alignItems: "center",
                      gap: 8,
                    }}
                  >
                    <Label isCompact color={r.triggered ? "red" : "green"}>
                      {r.triggered ? "TRIGGERED" : "PASS"}
                    </Label>
                    <span style={{ fontSize: 13, fontWeight: 500, color: "#e2e8f0" }}>
                      {ruleTypeLabel(r.rule_type)}
                    </span>
                    {r.triggered && (
                      <Label isCompact color={actionLabelColor(r.action as RuleAction)}>
                        {r.action}
                      </Label>
                    )}
                    <span style={{ fontSize: 12, color: "#6b7585", marginLeft: "auto" }}>
                      {r.details}
                    </span>
                  </div>
                ))
              )}
            </div>
          )}
        </ModalBody>
        <ModalFooter>
          <Button
            variant="primary"
            onClick={handleTest}
            isDisabled={testRunning || !testInput.trim()}
            isLoading={testRunning}
            icon={<FlaskIcon />}
          >
            Run Test
          </Button>
          <Button variant="link" onClick={() => setTestModalOpen(false)}>
            Close
          </Button>
        </ModalFooter>
      </Modal>

      {/* ---- Import / Export Modal ---- */}
      <Modal
        variant={ModalVariant.medium}
        isOpen={importExportModalOpen}
        onClose={() => setImportExportModalOpen(false)}
        aria-labelledby="import-export-title"
      >
        <ModalHeader
          title={importExportMode === "export" ? "Export Rules" : "Import Rules"}
          labelId="import-export-title"
        />
        <ModalBody>
          {importError && (
            <Alert variant="danger" title={importError} isInline style={{ marginBottom: 12 }} />
          )}
          <FormGroup
            label={importExportMode === "export" ? "Rule JSON (copy below)" : "Paste rule JSON"}
            fieldId="ie-json"
          >
            <TextArea
              id="ie-json"
              value={importJson}
              onChange={(_e, v) => setImportJson(v)}
              rows={12}
              readOnly={importExportMode === "export"}
              style={{ fontFamily: "var(--pf-t--global--font--family--mono)", fontSize: 12 }}
            />
          </FormGroup>
        </ModalBody>
        <ModalFooter>
          {importExportMode === "import" && (
            <Button variant="primary" onClick={handleImportApply}>
              Apply Import
            </Button>
          )}
          <Button variant="link" onClick={() => setImportExportModalOpen(false)}>
            Close
          </Button>
        </ModalFooter>
      </Modal>
    </>
  );
};
