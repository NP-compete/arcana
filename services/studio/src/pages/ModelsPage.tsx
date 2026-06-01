import { useCallback, useEffect, useState } from "react";
import {
  PageSection,
  Title,
  Content,
  Card,
  CardBody,
  CardTitle,
  Label,
  Button,
  Grid,
  GridItem,
  Spinner,
  Alert,
  AlertActionCloseButton,
  Modal,
  ModalBody,
  ModalFooter,
  ModalHeader,
  FormGroup,
  TextInput,
  FormSelect,
  FormSelectOption,
  DescriptionList,
  DescriptionListGroup,
  DescriptionListTerm,
  DescriptionListDescription,
  Progress,
  ProgressMeasureLocation,
} from "@patternfly/react-core";
import { Table, Thead, Tr, Th, Tbody, Td } from "@patternfly/react-table";
import { PlusCircleIcon, TrashIcon, ArrowCircleUpIcon } from "@patternfly/react-icons";
import { ShareBadge } from "../components/ShareBadge";

interface ModelCard {
  name: string;
  framework: string;
  base_model: string;
  task: string;
  metrics: Record<string, number>;
  serving: Record<string, unknown>;
  governance: Record<string, unknown>;
  environment: string;
  registered_at: string;
}

interface BudgetStatus {
  tokens_used: number;
  cost: number;
  remaining: number;
  budget_limit: number;
  period: string;
}

const envColor = (env: string): "green" | "blue" | "orange" | "grey" => {
  switch (env) {
    case "production": return "green";
    case "staging": return "orange";
    case "dev": return "blue";
    default: return "grey";
  }
};

export const ModelsPage = () => {
  const [models, setModels] = useState<ModelCard[]>([]);
  const [budget, setBudget] = useState<BudgetStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [modalOpen, setModalOpen] = useState(false);
  const [newName, setNewName] = useState("");
  const [newFramework, setNewFramework] = useState("pytorch");
  const [newBaseModel, setNewBaseModel] = useState("");
  const [newTask, setNewTask] = useState("");

  const fetchData = useCallback(async () => {
    try {
      const [modelsRes, budgetRes] = await Promise.allSettled([
        fetch("/api/v1/models"),
        fetch("/api/v1/budget"),
      ]);
      if (modelsRes.status === "fulfilled" && modelsRes.value.ok) {
        const data = await modelsRes.value.json();
        setModels(data.models || []);
      }
      if (budgetRes.status === "fulfilled" && budgetRes.value.ok) {
        const data = await budgetRes.value.json();
        setBudget(data);
      }
    } catch {
      setError("Failed to load models data");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const handleRegister = async () => {
    if (!newName.trim() || !newBaseModel.trim()) return;
    setError(null);
    try {
      const res = await fetch("/api/v1/models", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: newName.trim(),
          framework: newFramework,
          base_model: newBaseModel.trim(),
          task: newTask || "general",
        }),
      });
      if (!res.ok) {
        const data = await res.json();
        setError(data.detail || "Registration failed");
        return;
      }
      const card = await res.json();
      setModels((prev) => [...prev, card]);
      setModalOpen(false);
      setNewName("");
      setNewBaseModel("");
      setNewTask("");
    } catch {
      setError("Failed to register model");
    }
  };

  const handlePromote = async (name: string, env: string) => {
    try {
      const res = await fetch(`/api/v1/models/${name}/promote`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ environment: env }),
      });
      if (res.ok) {
        const updated = await res.json();
        setModels((prev) => prev.map((m) => (m.name === name ? updated : m)));
      }
    } catch {
      setError(`Failed to promote ${name}`);
    }
  };

  const handleDelete = async (name: string) => {
    try {
      await fetch(`/api/v1/models/${name}`, { method: "DELETE" });
      setModels((prev) => prev.filter((m) => m.name !== name));
    } catch {
      setError(`Failed to delete ${name}`);
    }
  };

  if (loading) {
    return (
      <PageSection hasBodyWrapper={false}>
        <div style={{ textAlign: "center", padding: 60 }}><Spinner size="xl" /></div>
      </PageSection>
    );
  }

  const budgetPct = budget ? Math.min(100, (budget.cost / budget.budget_limit) * 100) : 0;

  return (
    <>
      <PageSection hasBodyWrapper={false}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
          <div>
            <Title headingLevel="h1" size="2xl">Models</Title>
            <Content component="p" style={{ marginTop: 4, color: "var(--pf-t--global--text--color--subtle)" }}>
              LLM registry, serving, budget control, and fallback chains
            </Content>
          </div>
          <Button variant="primary" icon={<PlusCircleIcon />} onClick={() => setModalOpen(true)}>
            Register Model
          </Button>
        </div>
      </PageSection>

      {error && (
        <PageSection hasBodyWrapper={false} style={{ paddingTop: 0 }}>
          <Alert variant="danger" title={error} isInline actionClose={<AlertActionCloseButton onClose={() => setError(null)} />} />
        </PageSection>
      )}

      <PageSection hasBodyWrapper={false} style={{ paddingTop: 0 }}>
        <Grid hasGutter>
          <GridItem span={3}>
            <Card className="stat-card">
              <CardBody>
                <div style={{ fontSize: 28, fontWeight: 800 }}>{models.length}</div>
                <div style={{ fontSize: 13, color: "var(--pf-t--global--text--color--subtle)" }}>Registered Models</div>
              </CardBody>
            </Card>
          </GridItem>
          <GridItem span={3}>
            <Card className="stat-card">
              <CardBody>
                <div style={{ fontSize: 28, fontWeight: 800 }}>{budget ? budget.tokens_used.toLocaleString() : "—"}</div>
                <div style={{ fontSize: 13, color: "var(--pf-t--global--text--color--subtle)" }}>Tokens Used</div>
              </CardBody>
            </Card>
          </GridItem>
          <GridItem span={3}>
            <Card className="stat-card">
              <CardBody>
                <div style={{ fontSize: 28, fontWeight: 800 }}>${budget ? budget.cost.toFixed(2) : "—"}</div>
                <div style={{ fontSize: 13, color: "var(--pf-t--global--text--color--subtle)" }}>Total Cost</div>
              </CardBody>
            </Card>
          </GridItem>
          <GridItem span={3}>
            <Card className="stat-card">
              <CardBody>
                <div style={{ fontSize: 28, fontWeight: 800 }}>${budget ? budget.remaining.toFixed(2) : "—"}</div>
                <div style={{ fontSize: 13, color: "var(--pf-t--global--text--color--subtle)" }}>Budget Remaining</div>
              </CardBody>
            </Card>
          </GridItem>
        </Grid>
      </PageSection>

      {budget && (
        <PageSection hasBodyWrapper={false} style={{ paddingTop: 0 }}>
          <Card>
            <CardTitle>Budget Utilization</CardTitle>
            <CardBody>
              <Progress
                value={budgetPct}
                title={`$${budget.cost.toFixed(2)} of $${budget.budget_limit.toFixed(2)} (${budget.period})`}
                measureLocation={ProgressMeasureLocation.outside}
                variant={budgetPct > 80 ? "danger" : budgetPct > 50 ? "warning" : undefined}
              />
              <DescriptionList isHorizontal isCompact style={{ marginTop: 16 }}>
                <DescriptionListGroup>
                  <DescriptionListTerm>Fallback Chain</DescriptionListTerm>
                  <DescriptionListDescription>
                    <Label isCompact color="blue">gpt-4o-mini</Label>{" "}
                    <span style={{ color: "var(--pf-t--global--text--color--subtle)", margin: "0 4px" }}> → </span>
                    <Label isCompact color="grey">llama-3-8b</Label>
                  </DescriptionListDescription>
                </DescriptionListGroup>
              </DescriptionList>
            </CardBody>
          </Card>
        </PageSection>
      )}

      <PageSection hasBodyWrapper={false} style={{ paddingTop: 0 }}>
        <Card>
          <CardTitle>Model Registry</CardTitle>
          <CardBody>
            {models.length === 0 ? (
              <div className="arcana-empty-state">
                <div className="arcana-empty-icon">\uD83E\uDDE0</div>
                <Title headingLevel="h3">No models registered</Title>
                <Content component="p" style={{ marginTop: 8 }}>
                  Register your first model to get started with serving and evaluation.
                </Content>
              </div>
            ) : (
              <Table aria-label="Model registry" variant="compact">
                <Thead>
                  <Tr>
                    <Th>Name</Th>
                    <Th>Base Model</Th>
                    <Th>Framework</Th>
                    <Th>Task</Th>
                    <Th>Sharing</Th>
                    <Th>Environment</Th>
                    <Th>Registered</Th>
                    <Th>Actions</Th>
                  </Tr>
                </Thead>
                <Tbody>
                  {models.map((m) => (
                    <Tr key={m.name}>
                      <Td dataLabel="Name">
                        <span style={{ fontWeight: 700 }}>{m.name}</span>
                      </Td>
                      <Td dataLabel="Base Model">
                        <Label isCompact color="blue">{m.base_model}</Label>
                      </Td>
                      <Td dataLabel="Framework">{m.framework}</Td>
                      <Td dataLabel="Task">{m.task}</Td>
                      <Td dataLabel="Sharing">
                        <ShareBadge assetType="model" assetName={m.name} compact />
                      </Td>
                      <Td dataLabel="Environment">
                        <Label isCompact color={envColor(m.environment)}>{m.environment}</Label>
                      </Td>
                      <Td dataLabel="Registered">
                        {new Date(m.registered_at).toLocaleDateString()}
                      </Td>
                      <Td dataLabel="Actions">
                        {m.environment !== "production" && (
                          <Button
                            variant="plain"
                            icon={<ArrowCircleUpIcon />}
                            aria-label={`Promote ${m.name}`}
                            onClick={() =>
                              handlePromote(m.name, m.environment === "dev" ? "staging" : "production")
                            }
                          />
                        )}
                        <Button
                          variant="plain"
                          isDanger
                          icon={<TrashIcon />}
                          aria-label={`Delete ${m.name}`}
                          onClick={() => handleDelete(m.name)}
                        />
                      </Td>
                    </Tr>
                  ))}
                </Tbody>
              </Table>
            )}
          </CardBody>
        </Card>
      </PageSection>

      <Modal isOpen={modalOpen} onClose={() => setModalOpen(false)} variant="small" aria-label="Register model">
        <ModalHeader title="Register Model" />
        <ModalBody>
          <FormGroup label="Model Name" fieldId="model-name" isRequired>
            <TextInput id="model-name" value={newName} onChange={(_e, v) => setNewName(v)} placeholder="e.g. sentiment-v3" />
          </FormGroup>
          <FormGroup label="Base Model" fieldId="base-model" isRequired style={{ marginTop: 16 }}>
            <TextInput id="base-model" value={newBaseModel} onChange={(_e, v) => setNewBaseModel(v)} placeholder="e.g. gpt-4o, llama-3-70b" />
          </FormGroup>
          <FormGroup label="Framework" fieldId="framework" style={{ marginTop: 16 }}>
            <FormSelect id="framework" value={newFramework} onChange={(_e, v) => setNewFramework(v)}>
              <FormSelectOption value="pytorch" label="PyTorch" />
              <FormSelectOption value="transformers" label="Transformers (HuggingFace)" />
              <FormSelectOption value="vllm" label="vLLM" />
              <FormSelectOption value="tgi" label="TGI (Text Generation Inference)" />
              <FormSelectOption value="openai" label="OpenAI API" />
              <FormSelectOption value="anthropic" label="Anthropic API" />
              <FormSelectOption value="ollama" label="Ollama" />
            </FormSelect>
          </FormGroup>
          <FormGroup label="Task" fieldId="task" style={{ marginTop: 16 }}>
            <TextInput id="task" value={newTask} onChange={(_e, v) => setNewTask(v)} placeholder="e.g. text-generation, classification" />
          </FormGroup>
        </ModalBody>
        <ModalFooter>
          <Button variant="primary" onClick={handleRegister} isDisabled={!newName.trim() || !newBaseModel.trim()}>
            Register
          </Button>
          <Button variant="link" onClick={() => setModalOpen(false)}>Cancel</Button>
        </ModalFooter>
      </Modal>
    </>
  );
};
