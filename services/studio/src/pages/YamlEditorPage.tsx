import { useState, useEffect, useCallback, useRef } from "react";
import {
  PageSection,
  Title,
  Content,
  Divider,
  Button,
  FormSelect,
  FormSelectOption,
  TextInput,
  TextArea,
  Form,
  FormGroup,
  Alert,
  Spinner,
  ToggleGroup,
  ToggleGroupItem,
  Slider,
} from "@patternfly/react-core";
import {
  SaveIcon,
  CheckCircleIcon,
  RocketIcon,
  PlusCircleIcon,
} from "@patternfly/react-icons";

/* ------------------------------------------------------------------ */
/*  Types                                                              */
/* ------------------------------------------------------------------ */

type PackageType = "agent" | "skill" | "model" | "mcp";
type ViewMode = "form" | "yaml";

/* ------------------------------------------------------------------ */
/*  Constants                                                          */
/* ------------------------------------------------------------------ */

const TYPE_OPTIONS: { value: PackageType; label: string }[] = [
  { value: "agent", label: "Agent" },
  { value: "skill", label: "Skill" },
  { value: "model", label: "Model" },
  { value: "mcp", label: "MCP Server" },
];

const PROVIDER_OPTIONS = [
  { value: "openai", label: "OpenAI" },
  { value: "anthropic", label: "Anthropic" },
  { value: "google", label: "Google" },
  { value: "vllm", label: "vLLM" },
];

const TIER_OPTIONS = [
  { value: "planning", label: "Planning" },
  { value: "functional", label: "Functional" },
  { value: "atomic", label: "Atomic" },
];

/* ------------------------------------------------------------------ */
/*  Helpers                                                            */
/* ------------------------------------------------------------------ */

/** Minimal YAML serializer for simple flat/nested objects. */
function objectToYaml(obj: Record<string, unknown>, indent = 0): string {
  const prefix = "  ".repeat(indent);
  const lines: string[] = [];
  for (const [key, val] of Object.entries(obj)) {
    if (val === null || val === undefined || val === "") continue;
    if (typeof val === "object" && !Array.isArray(val)) {
      lines.push(`${prefix}${key}:`);
      lines.push(objectToYaml(val as Record<string, unknown>, indent + 1));
    } else if (Array.isArray(val)) {
      if (val.length === 0) {
        lines.push(`${prefix}${key}: []`);
      } else {
        lines.push(`${prefix}${key}:`);
        for (const item of val) {
          if (typeof item === "object" && item !== null) {
            const nested = objectToYaml(item as Record<string, unknown>, indent + 2).trimStart();
            lines.push(`${prefix}  - ${nested}`);
          } else {
            lines.push(`${prefix}  - ${String(item)}`);
          }
        }
      }
    } else if (typeof val === "string" && val.includes("\n")) {
      lines.push(`${prefix}${key}: |`);
      for (const line of val.split("\n")) {
        lines.push(`${prefix}  ${line}`);
      }
    } else {
      lines.push(`${prefix}${key}: ${String(val)}`);
    }
  }
  return lines.join("\n");
}

/** Very simple YAML parser: handles flat keys, quoted values, and minimal nesting. */
function yamlToObject(yaml: string): Record<string, string> {
  const result: Record<string, string> = {};
  for (const line of yaml.split("\n")) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith("#")) continue;
    const colonIdx = trimmed.indexOf(":");
    if (colonIdx > 0) {
      const key = trimmed.slice(0, colonIdx).trim();
      let val = trimmed.slice(colonIdx + 1).trim();
      if ((val.startsWith('"') && val.endsWith('"')) || (val.startsWith("'") && val.endsWith("'"))) {
        val = val.slice(1, -1);
      }
      result[key] = val;
    }
  }
  return result;
}

/* ------------------------------------------------------------------ */
/*  Component                                                          */
/* ------------------------------------------------------------------ */

export const YamlEditorPage = () => {
  const [packageType, setPackageType] = useState<PackageType>("agent");
  const [packageName, setPackageName] = useState("");
  const [packages, setPackages] = useState<string[]>([]);
  const [viewMode, setViewMode] = useState<ViewMode>("form");

  /* Form fields */
  const [formName, setFormName] = useState("");
  const [formDescription, setFormDescription] = useState("");
  const [formProvider, setFormProvider] = useState("openai");
  const [formModelId, setFormModelId] = useState("");
  const [formTemperature, setFormTemperature] = useState(70);
  const [formMaxTokens, setFormMaxTokens] = useState("4096");
  const [formSystemPrompt, setFormSystemPrompt] = useState("");
  const [formTier, setFormTier] = useState("functional");
  const [formSkills, setFormSkills] = useState("");

  /* YAML */
  const [yamlText, setYamlText] = useState("");
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  /* State */
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [validating, setValidating] = useState(false);
  const [deploying, setDeploying] = useState(false);
  const [alert, setAlert] = useState<{ type: "success" | "danger" | "info"; message: string } | null>(null);

  /* Fetch packages list */
  const fetchPackages = useCallback(async () => {
    try {
      const res = await fetch(`/api/v1/catalog/${packageType}s`);
      if (!res.ok) return;
      const data = await res.json();
      const entries: { name: string }[] = data.entries ?? [];
      setPackages(entries.map((e) => e.name));
    } catch {
      setPackages([]);
    }
  }, [packageType]);

  useEffect(() => {
    fetchPackages();
  }, [fetchPackages]);

  /* Build form object from form state */
  const buildFormObject = useCallback((): Record<string, unknown> => {
    const obj: Record<string, unknown> = {
      name: formName,
      type: packageType,
      description: formDescription,
    };
    if (packageType === "agent") {
      obj.spec = {
        provider: formProvider,
        model_id: formModelId,
        temperature: (formTemperature / 100).toFixed(2),
        max_tokens: formMaxTokens,
        system_prompt: formSystemPrompt,
        skills: formSkills.split(",").map((s) => s.trim()).filter(Boolean),
      };
    } else if (packageType === "skill") {
      obj.tier = formTier;
    } else if (packageType === "model") {
      obj.provider = formProvider;
      obj.model_id = formModelId;
      obj.temperature = (formTemperature / 100).toFixed(2);
      obj.max_tokens = formMaxTokens;
    }
    return obj;
  }, [formName, formDescription, packageType, formProvider, formModelId, formTemperature, formMaxTokens, formSystemPrompt, formSkills, formTier]);

  /* Sync form to YAML when switching to YAML view */
  const syncFormToYaml = useCallback(() => {
    setYamlText(objectToYaml(buildFormObject()));
  }, [buildFormObject]);

  /* Sync YAML to form when switching to form view */
  const syncYamlToForm = useCallback(() => {
    const parsed = yamlToObject(yamlText);
    if (parsed.name) setFormName(parsed.name);
    if (parsed.description) setFormDescription(parsed.description);
    if (parsed.provider) setFormProvider(parsed.provider);
    if (parsed.model_id) setFormModelId(parsed.model_id);
    if (parsed.tier) setFormTier(parsed.tier);
    if (parsed.system_prompt) setFormSystemPrompt(parsed.system_prompt);
  }, [yamlText]);

  const handleViewChange = (mode: ViewMode) => {
    if (mode === "yaml" && viewMode === "form") {
      syncFormToYaml();
    } else if (mode === "form" && viewMode === "yaml") {
      syncYamlToForm();
    }
    setViewMode(mode);
  };

  /* Load existing package */
  const loadPackage = async (name: string) => {
    if (!name) return;
    setLoading(true);
    setAlert(null);
    try {
      const res = await fetch(`/api/v1/catalog/${packageType}/${encodeURIComponent(name)}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      setFormName(data.name ?? name);
      setFormDescription(data.description ?? "");
      if (data.metadata?.provider) setFormProvider(data.metadata.provider);
      if (data.metadata?.model_id) setFormModelId(data.metadata.model_id);
      if (data.metadata?.tier) setFormTier(data.metadata.tier);
      if (data.metadata?.system_prompt) setFormSystemPrompt(data.metadata.system_prompt);
      if (data.metadata?.skills) setFormSkills(Array.isArray(data.metadata.skills) ? data.metadata.skills.join(", ") : "");
      setYamlText(objectToYaml(data));
      setPackageName(name);
    } catch (e) {
      setAlert({ type: "danger", message: e instanceof Error ? e.message : "Failed to load" });
    } finally {
      setLoading(false);
    }
  };

  const handleNew = () => {
    setFormName("");
    setFormDescription("");
    setFormProvider("openai");
    setFormModelId("");
    setFormTemperature(70);
    setFormMaxTokens("4096");
    setFormSystemPrompt("");
    setFormTier("functional");
    setFormSkills("");
    setYamlText("");
    setPackageName("");
    setAlert(null);
  };

  /* Save */
  const handleSave = async () => {
    const name = formName.trim();
    if (!name) {
      setAlert({ type: "danger", message: "Package name is required." });
      return;
    }
    setSaving(true);
    setAlert(null);
    try {
      const body = viewMode === "yaml" ? yamlText : JSON.stringify(buildFormObject());
      const contentType = viewMode === "yaml" ? "text/yaml" : "application/json";
      const res = await fetch(`/api/v1/catalog/${packageType}/${encodeURIComponent(name)}`, {
        method: "PUT",
        headers: { "Content-Type": contentType },
        body,
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error((data as Record<string, string>).error ?? `HTTP ${res.status}`);
      }
      setAlert({ type: "success", message: `"${name}" saved successfully.` });
      await fetchPackages();
    } catch (e) {
      setAlert({ type: "danger", message: e instanceof Error ? e.message : "Save failed" });
    } finally {
      setSaving(false);
    }
  };

  /* Validate */
  const handleValidate = async () => {
    setValidating(true);
    setAlert(null);
    try {
      const body = viewMode === "yaml" ? yamlText : JSON.stringify(buildFormObject());
      const contentType = viewMode === "yaml" ? "text/yaml" : "application/json";
      const res = await fetch("/api/v1/catalog/validate", {
        method: "POST",
        headers: { "Content-Type": contentType },
        body,
      });
      const data = await res.json().catch(() => ({}));
      if (res.ok) {
        setAlert({ type: "success", message: "Validation passed." });
      } else {
        setAlert({ type: "danger", message: (data as Record<string, string>).error ?? "Validation failed." });
      }
    } catch (e) {
      setAlert({ type: "danger", message: e instanceof Error ? e.message : "Validation error" });
    } finally {
      setValidating(false);
    }
  };

  /* Deploy */
  const handleDeploy = async () => {
    setDeploying(true);
    setAlert(null);
    try {
      const endpoint = packageType === "agent" ? "/api/v1/agents/register" : `/api/v1/${packageType}s`;
      const res = await fetch(endpoint, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(buildFormObject()),
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error((data as Record<string, string>).error ?? `HTTP ${res.status}`);
      }
      setAlert({ type: "success", message: `"${formName}" deployed.` });
    } catch (e) {
      setAlert({ type: "danger", message: e instanceof Error ? e.message : "Deploy failed" });
    } finally {
      setDeploying(false);
    }
  };

  /* Line numbers for YAML view */
  const lineCount = yamlText.split("\n").length;
  const lineNumbers = Array.from({ length: Math.max(lineCount, 20) }, (_, i) => i + 1).join("\n");

  return (
    <>
      {/* Top bar */}
      <PageSection hasBodyWrapper={false}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
          <div>
            <Title headingLevel="h1" size="2xl">YAML Editor</Title>
            <Content component="p" style={{ marginTop: 4, color: "var(--pf-t--global--text--color--subtle)" }}>
              Edit package configurations with live form/YAML toggle.
            </Content>
          </div>
        </div>
      </PageSection>
      <Divider />

      {alert && (
        <PageSection hasBodyWrapper={false} style={{ paddingBottom: 0 }}>
          <Alert variant={alert.type} title={alert.message} isInline />
        </PageSection>
      )}

      {/* Selector bar */}
      <PageSection hasBodyWrapper={false} style={{ paddingBottom: 0 }}>
        <div style={{
          display: "flex", gap: 12, alignItems: "flex-end", flexWrap: "wrap",
          padding: "16px 20px",
          background: "rgba(255,255,255,0.03)",
          borderRadius: 10,
          border: "1px solid rgba(255,255,255,0.06)",
        }}>
          <div>
            <div style={{ fontSize: 11, color: "#8b95a5", marginBottom: 4, fontWeight: 600 }}>Package Type</div>
            <FormSelect
              value={packageType}
              onChange={(_e, v) => { setPackageType(v as PackageType); setPackageName(""); handleNew(); }}
              aria-label="Package type"
              style={{ width: 140 }}
            >
              {TYPE_OPTIONS.map((t) => <FormSelectOption key={t.value} value={t.value} label={t.label} />)}
            </FormSelect>
          </div>
          <div>
            <div style={{ fontSize: 11, color: "#8b95a5", marginBottom: 4, fontWeight: 600 }}>Package Name</div>
            <FormSelect
              value={packageName}
              onChange={(_e, v) => { setPackageName(v); if (v) loadPackage(v); }}
              aria-label="Package name"
              style={{ width: 220 }}
            >
              <FormSelectOption value="" label="-- Select --" />
              {packages.map((p) => <FormSelectOption key={p} value={p} label={p} />)}
            </FormSelect>
          </div>
          <Button variant="secondary" icon={<PlusCircleIcon />} onClick={handleNew}>New</Button>
          <div style={{ flex: 1 }} />
          <ToggleGroup aria-label="View mode">
            <ToggleGroupItem
              text="Form View"
              buttonId="view-form"
              isSelected={viewMode === "form"}
              onChange={() => handleViewChange("form")}
            />
            <ToggleGroupItem
              text="YAML View"
              buttonId="view-yaml"
              isSelected={viewMode === "yaml"}
              onChange={() => handleViewChange("yaml")}
            />
          </ToggleGroup>
        </div>
      </PageSection>

      {/* Editor */}
      <PageSection hasBodyWrapper={false}>
        {loading ? (
          <div style={{ textAlign: "center", padding: 40 }}>
            <Spinner size="xl" />
          </div>
        ) : viewMode === "form" ? (
          <div style={{
            background: "rgba(255,255,255,0.03)",
            borderRadius: 12,
            padding: 24,
            border: "1px solid rgba(255,255,255,0.06)",
          }}>
            <Form>
              <FormGroup label="Name" isRequired fieldId="editor-name">
                <TextInput id="editor-name" value={formName} onChange={(_e, v) => setFormName(v)} isRequired />
              </FormGroup>
              <FormGroup label="Description" fieldId="editor-desc">
                <TextArea id="editor-desc" value={formDescription} onChange={(_e, v) => setFormDescription(v)} rows={2} />
              </FormGroup>

              {packageType === "agent" && (
                <>
                  <FormGroup label="Provider" fieldId="editor-provider">
                    <FormSelect id="editor-provider" value={formProvider} onChange={(_e, v) => setFormProvider(v)} aria-label="Provider">
                      {PROVIDER_OPTIONS.map((p) => <FormSelectOption key={p.value} value={p.value} label={p.label} />)}
                    </FormSelect>
                  </FormGroup>
                  <FormGroup label="Model ID" fieldId="editor-model">
                    <TextInput id="editor-model" value={formModelId} onChange={(_e, v) => setFormModelId(v)} placeholder="e.g. gpt-4o" />
                  </FormGroup>
                  <FormGroup label={`Temperature: ${(formTemperature / 100).toFixed(2)}`} fieldId="editor-temp">
                    <Slider
                      id="editor-temp"
                      value={formTemperature}
                      onChange={(_e, v) => setFormTemperature(v)}
                      max={100} min={0} showTicks={false}
                      aria-label="Temperature"
                    />
                  </FormGroup>
                  <FormGroup label="Max Tokens" fieldId="editor-tokens">
                    <TextInput id="editor-tokens" value={formMaxTokens} onChange={(_e, v) => setFormMaxTokens(v)} type="number" />
                  </FormGroup>
                  <FormGroup label="System Prompt" fieldId="editor-prompt">
                    <TextArea
                      id="editor-prompt"
                      value={formSystemPrompt}
                      onChange={(_e, v) => setFormSystemPrompt(v)}
                      rows={5}
                      style={{ fontFamily: "'JetBrains Mono', 'Fira Code', monospace", fontSize: 13 }}
                    />
                  </FormGroup>
                  <FormGroup label="Skills (comma-separated)" fieldId="editor-skills">
                    <TextInput id="editor-skills" value={formSkills} onChange={(_e, v) => setFormSkills(v)} placeholder="e.g. code-gen, sql-query" />
                  </FormGroup>
                </>
              )}

              {packageType === "skill" && (
                <FormGroup label="Tier" fieldId="editor-tier">
                  <FormSelect id="editor-tier" value={formTier} onChange={(_e, v) => setFormTier(v)} aria-label="Tier">
                    {TIER_OPTIONS.map((t) => <FormSelectOption key={t.value} value={t.value} label={t.label} />)}
                  </FormSelect>
                </FormGroup>
              )}

              {packageType === "model" && (
                <>
                  <FormGroup label="Provider" fieldId="editor-model-provider">
                    <FormSelect id="editor-model-provider" value={formProvider} onChange={(_e, v) => setFormProvider(v)} aria-label="Provider">
                      {PROVIDER_OPTIONS.map((p) => <FormSelectOption key={p.value} value={p.value} label={p.label} />)}
                    </FormSelect>
                  </FormGroup>
                  <FormGroup label="Model ID" fieldId="editor-model-id">
                    <TextInput id="editor-model-id" value={formModelId} onChange={(_e, v) => setFormModelId(v)} placeholder="e.g. gpt-4o" />
                  </FormGroup>
                </>
              )}
            </Form>
          </div>
        ) : (
          /* YAML View */
          <div style={{
            display: "flex",
            background: "#0d0f14",
            borderRadius: 12,
            border: "1px solid rgba(255,255,255,0.08)",
            overflow: "hidden",
            minHeight: 400,
          }}>
            {/* Line numbers */}
            <pre style={{
              margin: 0,
              padding: "16px 12px",
              background: "rgba(255,255,255,0.03)",
              color: "#4a5568",
              fontSize: 13,
              fontFamily: "'JetBrains Mono', 'Fira Code', 'Consolas', monospace",
              lineHeight: "1.6",
              textAlign: "right",
              userSelect: "none",
              minWidth: 40,
              borderRight: "1px solid rgba(255,255,255,0.06)",
            }}>
              {lineNumbers}
            </pre>
            {/* Textarea */}
            <textarea
              ref={textareaRef}
              value={yamlText}
              onChange={(e) => setYamlText(e.target.value)}
              spellCheck={false}
              style={{
                flex: 1,
                margin: 0,
                padding: 16,
                background: "transparent",
                color: "#c5cdd8",
                fontSize: 13,
                fontFamily: "'JetBrains Mono', 'Fira Code', 'Consolas', monospace",
                lineHeight: "1.6",
                border: "none",
                outline: "none",
                resize: "none",
                minHeight: 400,
              }}
            />
          </div>
        )}
      </PageSection>

      {/* Bottom bar */}
      <PageSection hasBodyWrapper={false} style={{ paddingTop: 0 }}>
        <div style={{ display: "flex", gap: 12 }}>
          <Button variant="primary" icon={<SaveIcon />} onClick={handleSave} isLoading={saving} isDisabled={saving}>
            Save
          </Button>
          <Button variant="secondary" icon={<CheckCircleIcon />} onClick={handleValidate} isLoading={validating} isDisabled={validating}>
            Validate
          </Button>
          <Button variant="secondary" icon={<RocketIcon />} onClick={handleDeploy} isLoading={deploying} isDisabled={deploying}>
            Deploy
          </Button>
        </div>
      </PageSection>
    </>
  );
};
