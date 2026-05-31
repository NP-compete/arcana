import { useState, useCallback, useRef, useMemo, type DragEvent } from "react";
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  Panel,
  useNodesState,
  useEdgesState,
  addEdge,
  type Node,
  type Edge,
  type Connection,
  Handle,
  Position,
  type NodeProps,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import {
  PageSection,
  Title,
  Button,
  TextInput,
  FormGroup,
  FormSelect,
  FormSelectOption,
  TextArea,
  Label,
  Alert,
  Modal,
  ModalHeader,
  ModalBody,
  ModalFooter,
  ModalVariant,
  Spinner,
} from "@patternfly/react-core";
import {
  RobotIcon,
  CogIcon,
  CodeIcon,
  FilterIcon,
  UserIcon,
  DatabaseIcon,
  SaveIcon,
  FolderOpenIcon,
  PlayIcon,
  FileExportIcon,
  TrashIcon,
} from "@patternfly/react-icons";

/* ---------- types ---------- */

type FlowNodeType = "agent" | "tool" | "condition" | "hitl" | "rag" | "code";

interface FlowNodeData extends Record<string, unknown> {
  label: string;
  nodeType: FlowNodeType;
  config: Record<string, unknown>;
}

type FlowNode = Node<FlowNodeData>;

/* ---------- palette ---------- */

interface PaletteItem {
  type: FlowNodeType;
  label: string;
  color: string;
  icon: React.ReactNode;
}

const PALETTE: PaletteItem[] = [
  { type: "agent", label: "Agent", color: "#4a90d9", icon: <RobotIcon /> },
  { type: "tool", label: "Tool", color: "#48bb78", icon: <CogIcon /> },
  { type: "condition", label: "Condition", color: "#ed8936", icon: <FilterIcon /> },
  { type: "hitl", label: "HITL", color: "#9f7aea", icon: <UserIcon /> },
  { type: "rag", label: "RAG", color: "#4fd1c5", icon: <DatabaseIcon /> },
  { type: "code", label: "Code", color: "#718096", icon: <CodeIcon /> },
];

const NODE_COLORS: Record<FlowNodeType, string> = {
  agent: "#4a90d9",
  tool: "#48bb78",
  condition: "#ed8936",
  hitl: "#9f7aea",
  rag: "#4fd1c5",
  code: "#718096",
};

/* ---------- custom nodes ---------- */

function configSummary(nodeType: FlowNodeType, config: Record<string, unknown>): string {
  switch (nodeType) {
    case "agent":
      return (config.model as string) || "No model";
    case "tool":
      return (config.toolName as string) || "No tool selected";
    case "condition":
      return (config.expression as string) || "No expression";
    case "hitl":
      return (config.assignee as string) || "No assignee";
    case "rag":
      return `top-k: ${config.topK ?? 5}`;
    case "code":
      return (config.language as string) || "python";
    default:
      return "";
  }
}

function paletteIconForType(nodeType: FlowNodeType): React.ReactNode {
  const item = PALETTE.find((p) => p.type === nodeType);
  return item?.icon ?? <CogIcon />;
}

function FlowCustomNode({ data, selected }: NodeProps<FlowNode>) {
  const color = NODE_COLORS[data.nodeType] ?? "#718096";
  return (
    <div
      className={`flow-node flow-node-${data.nodeType}`}
      style={{
        borderLeft: `4px solid ${color}`,
        outline: selected ? `2px solid ${color}` : "none",
        outlineOffset: 2,
      }}
    >
      <Handle type="target" position={Position.Top} style={{ background: color }} />
      <div className="flow-node-header">
        <span className="flow-node-icon" style={{ color }}>{paletteIconForType(data.nodeType)}</span>
        <span className="flow-node-label">{data.label}</span>
      </div>
      <div className="flow-node-summary">{configSummary(data.nodeType, data.config)}</div>
      <Handle type="source" position={Position.Bottom} style={{ background: color }} />
    </div>
  );
}

const NODE_TYPE_MAP = {
  agent: FlowCustomNode,
  tool: FlowCustomNode,
  condition: FlowCustomNode,
  hitl: FlowCustomNode,
  rag: FlowCustomNode,
  code: FlowCustomNode,
};

/* ---------- default configs ---------- */

function defaultConfig(nodeType: FlowNodeType): Record<string, unknown> {
  switch (nodeType) {
    case "agent":
      return { model: "claude-sonnet-4", tools: [] };
    case "tool":
      return { server: "", toolName: "" };
    case "condition":
      return { expression: "" };
    case "hitl":
      return { assignee: "", timeout: "30m" };
    case "rag":
      return { query: "", topK: 5 };
    case "code":
      return { language: "python", snippet: "" };
    default:
      return {};
  }
}

/* ---------- YAML export ---------- */

function toYaml(nodes: FlowNode[], edges: Edge[]): string {
  const lines: string[] = ["nodes:"];
  for (const n of nodes) {
    lines.push(`  - id: ${n.id}`);
    lines.push(`    type: ${n.data.nodeType}`);
    const cfg = n.data.config;
    for (const [k, v] of Object.entries(cfg)) {
      if (v === "" || v === undefined || v === null) continue;
      if (Array.isArray(v)) {
        if (v.length === 0) continue;
        lines.push(`    ${k}: [${v.join(", ")}]`);
      } else {
        lines.push(`    ${k}: ${String(v)}`);
      }
    }
  }
  lines.push("edges:");
  for (const e of edges) {
    lines.push(`  - from: ${e.source}`);
    lines.push(`    to: ${e.target}`);
  }
  return lines.join("\n");
}

/* ---------- properties panel ---------- */

interface PropertiesPanelProps {
  node: FlowNode;
  onUpdate: (id: string, data: FlowNodeData) => void;
  onClose: () => void;
}

function PropertiesPanel({ node, onUpdate, onClose }: PropertiesPanelProps) {
  const { data } = node;
  const updateConfig = (key: string, value: unknown) => {
    onUpdate(node.id, {
      ...data,
      config: { ...data.config, [key]: value },
    });
  };

  return (
    <div className="flow-properties">
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 16 }}>
        <Title headingLevel="h4" size="md">Properties</Title>
        <Button variant="plain" size="sm" onClick={onClose} aria-label="Close properties">
          &times;
        </Button>
      </div>
      <FormGroup label="Label" fieldId="node-label">
        <TextInput
          id="node-label"
          value={data.label}
          onChange={(_e, v) => onUpdate(node.id, { ...data, label: v })}
        />
      </FormGroup>
      <div style={{ marginTop: 8 }}>
        <Label color="blue" isCompact>{data.nodeType}</Label>
      </div>
      <div style={{ marginTop: 16 }}>
        {data.nodeType === "agent" && (
          <>
            <FormGroup label="Model" fieldId="agent-model" style={{ marginBottom: 12 }}>
              <FormSelect
                id="agent-model"
                value={(data.config.model as string) ?? "claude-sonnet-4"}
                onChange={(_e, v) => updateConfig("model", v)}
              >
                <FormSelectOption value="claude-sonnet-4" label="claude-sonnet-4" />
                <FormSelectOption value="claude-opus-4" label="claude-opus-4" />
                <FormSelectOption value="gpt-4o" label="gpt-4o" />
                <FormSelectOption value="gpt-4o-mini" label="gpt-4o-mini" />
              </FormSelect>
            </FormGroup>
            <FormGroup label="Tools (comma-separated)" fieldId="agent-tools">
              <TextInput
                id="agent-tools"
                value={Array.isArray(data.config.tools) ? (data.config.tools as string[]).join(", ") : ""}
                onChange={(_e, v) => updateConfig("tools", v.split(",").map((s) => s.trim()).filter(Boolean))}
                placeholder="web_search, code_exec"
              />
            </FormGroup>
          </>
        )}
        {data.nodeType === "tool" && (
          <>
            <FormGroup label="MCP Server" fieldId="tool-server" style={{ marginBottom: 12 }}>
              <TextInput
                id="tool-server"
                value={(data.config.server as string) ?? ""}
                onChange={(_e, v) => updateConfig("server", v)}
                placeholder="mcp-server-name"
              />
            </FormGroup>
            <FormGroup label="Tool Name" fieldId="tool-name">
              <TextInput
                id="tool-name"
                value={(data.config.toolName as string) ?? ""}
                onChange={(_e, v) => updateConfig("toolName", v)}
                placeholder="web_search"
              />
            </FormGroup>
          </>
        )}
        {data.nodeType === "condition" && (
          <FormGroup label="Expression" fieldId="cond-expr">
            <TextArea
              id="cond-expr"
              value={(data.config.expression as string) ?? ""}
              onChange={(_e, v) => updateConfig("expression", v)}
              placeholder='result.status == "success"'
              rows={3}
            />
          </FormGroup>
        )}
        {data.nodeType === "hitl" && (
          <>
            <FormGroup label="Assignee" fieldId="hitl-assignee" style={{ marginBottom: 12 }}>
              <TextInput
                id="hitl-assignee"
                value={(data.config.assignee as string) ?? ""}
                onChange={(_e, v) => updateConfig("assignee", v)}
                placeholder="team-lead"
              />
            </FormGroup>
            <FormGroup label="Timeout" fieldId="hitl-timeout">
              <TextInput
                id="hitl-timeout"
                value={(data.config.timeout as string) ?? "30m"}
                onChange={(_e, v) => updateConfig("timeout", v)}
                placeholder="30m"
              />
            </FormGroup>
          </>
        )}
        {data.nodeType === "rag" && (
          <>
            <FormGroup label="Query Template" fieldId="rag-query" style={{ marginBottom: 12 }}>
              <TextArea
                id="rag-query"
                value={(data.config.query as string) ?? ""}
                onChange={(_e, v) => updateConfig("query", v)}
                placeholder="Find relevant documents about..."
                rows={3}
              />
            </FormGroup>
            <FormGroup label="Top-K" fieldId="rag-topk">
              <TextInput
                id="rag-topk"
                type="number"
                value={String(data.config.topK ?? 5)}
                onChange={(_e, v) => updateConfig("topK", parseInt(v) || 5)}
              />
            </FormGroup>
          </>
        )}
        {data.nodeType === "code" && (
          <>
            <FormGroup label="Language" fieldId="code-lang" style={{ marginBottom: 12 }}>
              <FormSelect
                id="code-lang"
                value={(data.config.language as string) ?? "python"}
                onChange={(_e, v) => updateConfig("language", v)}
              >
                <FormSelectOption value="python" label="Python" />
                <FormSelectOption value="javascript" label="JavaScript" />
                <FormSelectOption value="bash" label="Bash" />
                <FormSelectOption value="sql" label="SQL" />
              </FormSelect>
            </FormGroup>
            <FormGroup label="Code Snippet" fieldId="code-snippet">
              <TextArea
                id="code-snippet"
                value={(data.config.snippet as string) ?? ""}
                onChange={(_e, v) => updateConfig("snippet", v)}
                placeholder="# your code here"
                rows={6}
                style={{ fontFamily: "monospace", fontSize: 12 }}
              />
            </FormGroup>
          </>
        )}
      </div>
    </div>
  );
}

/* ---------- load modal ---------- */

interface BlueprintListItem {
  name: string;
  created_at?: string;
}

/* ---------- main component ---------- */

let nodeIdCounter = 0;
function nextNodeId(type: string): string {
  nodeIdCounter += 1;
  return `${type}-${nodeIdCounter}`;
}

export const FlowBuilderPage = () => {
  const reactFlowWrapper = useRef<HTMLDivElement>(null);
  const [nodes, setNodes, onNodesChange] = useNodesState<FlowNode>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);
  const [selectedNode, setSelectedNode] = useState<FlowNode | null>(null);
  const [blueprintName, setBlueprintName] = useState("my-pipeline");

  /* save / load / execute state */
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [saveSuccess, setSaveSuccess] = useState<string | null>(null);
  const [executing, setExecuting] = useState(false);
  const [execResult, setExecResult] = useState<string | null>(null);
  const [execError, setExecError] = useState<string | null>(null);

  /* load modal */
  const [loadModalOpen, setLoadModalOpen] = useState(false);
  const [blueprintList, setBlueprintList] = useState<BlueprintListItem[]>([]);
  const [loadingList, setLoadingList] = useState(false);

  const nodeTypes = useMemo(() => NODE_TYPE_MAP, []);

  /* ---- connections ---- */
  const onConnect = useCallback(
    (params: Connection) => {
      setEdges((eds) => addEdge({ ...params, animated: true, style: { stroke: "#4a5568" } }, eds));
    },
    [setEdges],
  );

  /* ---- node selection ---- */
  const onNodeClick = useCallback(
    (_event: React.MouseEvent, node: FlowNode) => {
      setSelectedNode(node);
    },
    [],
  );

  const onPaneClick = useCallback(() => {
    setSelectedNode(null);
  }, []);

  /* ---- drag and drop ---- */
  const onDragOver = useCallback((event: DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    event.dataTransfer.dropEffect = "move";
  }, []);

  const onDrop = useCallback(
    (event: DragEvent<HTMLDivElement>) => {
      event.preventDefault();
      const typeStr = event.dataTransfer.getData("application/reactflow");
      if (!typeStr) return;
      const nodeType = typeStr as FlowNodeType;

      const wrapper = reactFlowWrapper.current;
      if (!wrapper) return;
      const bounds = wrapper.getBoundingClientRect();
      const position = {
        x: event.clientX - bounds.left - 80,
        y: event.clientY - bounds.top - 20,
      };

      const paletteItem = PALETTE.find((p) => p.type === nodeType);
      const newNode: FlowNode = {
        id: nextNodeId(nodeType),
        type: nodeType,
        position,
        data: {
          label: paletteItem?.label ?? nodeType,
          nodeType,
          config: defaultConfig(nodeType),
        },
      };
      setNodes((nds) => [...nds, newNode]);
    },
    [setNodes],
  );

  /* ---- update node data ---- */
  const updateNodeData = useCallback(
    (id: string, newData: FlowNodeData) => {
      setNodes((nds) =>
        nds.map((n) => (n.id === id ? { ...n, data: newData } : n)),
      );
      setSelectedNode((prev) => (prev && prev.id === id ? { ...prev, data: newData } : prev));
    },
    [setNodes],
  );

  /* ---- save blueprint ---- */
  const handleSave = async () => {
    if (!blueprintName.trim()) {
      setSaveError("Blueprint name is required");
      return;
    }
    setSaving(true);
    setSaveError(null);
    setSaveSuccess(null);
    try {
      const yaml = toYaml(nodes, edges);
      const res = await fetch("/api/v1/blueprints", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: blueprintName.trim(), yaml }),
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error((data as Record<string, string>).error ?? `HTTP ${res.status}`);
      }
      setSaveSuccess("Blueprint saved");
    } catch (e) {
      setSaveError(e instanceof Error ? e.message : "Save failed");
    } finally {
      setSaving(false);
    }
  };

  /* ---- load blueprints list ---- */
  const openLoadModal = async () => {
    setLoadModalOpen(true);
    setLoadingList(true);
    try {
      const res = await fetch("/api/v1/blueprints");
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      setBlueprintList(data.blueprints ?? []);
    } catch {
      setBlueprintList([]);
    } finally {
      setLoadingList(false);
    }
  };

  const loadBlueprint = async (name: string) => {
    try {
      const res = await fetch(`/api/v1/blueprints/${encodeURIComponent(name)}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      const yamlStr = data.yaml ?? "";
      /* basic YAML parse — nodes/edges */
      const parsed = parseSimpleYaml(yamlStr);
      setNodes(parsed.nodes);
      setEdges(parsed.edges);
      setBlueprintName(name);
      setSelectedNode(null);
      setLoadModalOpen(false);
    } catch {
      /* silently fail — user sees empty canvas */
      setLoadModalOpen(false);
    }
  };

  /* ---- execute ---- */
  const handleExecute = async () => {
    if (!blueprintName.trim()) return;
    setExecuting(true);
    setExecError(null);
    setExecResult(null);
    try {
      const res = await fetch(`/api/v1/blueprints/${encodeURIComponent(blueprintName.trim())}/execute`, {
        method: "POST",
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error((data as Record<string, string>).error ?? `HTTP ${res.status}`);
      }
      const data = await res.json();
      setExecResult(data.status ?? "Execution started");
    } catch (e) {
      setExecError(e instanceof Error ? e.message : "Execution failed");
    } finally {
      setExecuting(false);
    }
  };

  /* ---- export YAML ---- */
  const handleExportYaml = () => {
    const yaml = toYaml(nodes, edges);
    const blob = new Blob([yaml], { type: "text/yaml" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `${blueprintName || "blueprint"}.yaml`;
    a.click();
    URL.revokeObjectURL(url);
  };

  /* ---- clear ---- */
  const handleClear = () => {
    setNodes([]);
    setEdges([]);
    setSelectedNode(null);
  };

  return (
    <PageSection hasBodyWrapper={false} style={{ padding: 0, height: "calc(100vh - 76px)" }}>
      <div style={{ display: "flex", height: "100%" }}>
        {/* Left sidebar - palette */}
        <div className="flow-sidebar">
          <div style={{ padding: "16px 12px 8px", fontWeight: 700, fontSize: 13, textTransform: "uppercase", letterSpacing: "0.5px", color: "#8b95a5" }}>
            Node Palette
          </div>
          {PALETTE.map((item) => (
            <div
              key={item.type}
              className="flow-palette-item"
              draggable
              onDragStart={(e) => {
                e.dataTransfer.setData("application/reactflow", item.type);
                e.dataTransfer.effectAllowed = "move";
              }}
              style={{ borderLeftColor: item.color }}
            >
              <span style={{ color: item.color, display: "flex", alignItems: "center", fontSize: 14 }}>
                {item.icon}
              </span>
              <span>{item.label}</span>
            </div>
          ))}
        </div>

        {/* Center - canvas */}
        <div style={{ flex: 1, position: "relative" }} ref={reactFlowWrapper}>
          {(saveError || saveSuccess || execError || execResult) && (
            <div style={{ position: "absolute", top: 56, left: "50%", transform: "translateX(-50%)", zIndex: 10, minWidth: 300 }}>
              {saveError && <Alert variant="danger" title={saveError} isInline style={{ marginBottom: 4 }} />}
              {saveSuccess && <Alert variant="success" title={saveSuccess} isInline style={{ marginBottom: 4 }} />}
              {execError && <Alert variant="danger" title={execError} isInline style={{ marginBottom: 4 }} />}
              {execResult && <Alert variant="info" title={execResult} isInline style={{ marginBottom: 4 }} />}
            </div>
          )}
          <ReactFlow
            nodes={nodes}
            edges={edges}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onConnect={onConnect}
            onNodeClick={onNodeClick}
            onPaneClick={onPaneClick}
            onDrop={onDrop}
            onDragOver={onDragOver}
            nodeTypes={nodeTypes}
            fitView
            style={{ background: "#0f1117" }}
            defaultEdgeOptions={{ animated: true, style: { stroke: "#4a5568" } }}
          >
            <Background color="#2d3348" gap={20} />
            <Controls
              style={{ background: "#1e2130", border: "1px solid #2d3348", borderRadius: 8 }}
            />
            <MiniMap
              style={{ background: "#1a1d2e", border: "1px solid #2d3348", borderRadius: 8 }}
              nodeColor={(n: FlowNode) => NODE_COLORS[(n.data?.nodeType as FlowNodeType)] ?? "#718096"}
              maskColor="rgba(0,0,0,0.6)"
            />
            <Panel position="top-center">
              <div className="flow-toolbar">
                <TextInput
                  value={blueprintName}
                  onChange={(_e, v) => setBlueprintName(v)}
                  placeholder="Blueprint name"
                  style={{ width: 180, background: "#161822", border: "1px solid #2d3348", color: "#e2e8f0", borderRadius: 6 }}
                  aria-label="Blueprint name"
                />
                <Button variant="primary" size="sm" icon={<SaveIcon />} onClick={handleSave} isLoading={saving} isDisabled={saving}>
                  Save
                </Button>
                <Button variant="secondary" size="sm" icon={<FolderOpenIcon />} onClick={openLoadModal}>
                  Load
                </Button>
                <Button variant="secondary" size="sm" icon={<PlayIcon />} onClick={handleExecute} isLoading={executing} isDisabled={executing}>
                  Execute
                </Button>
                <Button variant="secondary" size="sm" icon={<FileExportIcon />} onClick={handleExportYaml}>
                  YAML
                </Button>
                <Button variant="danger" size="sm" icon={<TrashIcon />} onClick={handleClear}>
                  Clear
                </Button>
              </div>
            </Panel>
          </ReactFlow>
        </div>

        {/* Right sidebar - properties */}
        {selectedNode && (
          <PropertiesPanel
            node={selectedNode}
            onUpdate={updateNodeData}
            onClose={() => setSelectedNode(null)}
          />
        )}
      </div>

      {/* Load modal */}
      <Modal
        variant={ModalVariant.small}
        isOpen={loadModalOpen}
        onClose={() => setLoadModalOpen(false)}
        aria-labelledby="load-blueprint-title"
      >
        <ModalHeader title="Load Blueprint" labelId="load-blueprint-title" />
        <ModalBody>
          {loadingList ? (
            <div style={{ textAlign: "center", padding: 24 }}><Spinner size="lg" /></div>
          ) : blueprintList.length === 0 ? (
            <div style={{ textAlign: "center", padding: 24, color: "#8b95a5" }}>No blueprints found</div>
          ) : (
            <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
              {blueprintList.map((bp) => (
                <Button key={bp.name} variant="secondary" isBlock onClick={() => loadBlueprint(bp.name)}>
                  {bp.name}
                </Button>
              ))}
            </div>
          )}
        </ModalBody>
        <ModalFooter>
          <Button variant="link" onClick={() => setLoadModalOpen(false)}>Cancel</Button>
        </ModalFooter>
      </Modal>
    </PageSection>
  );
};

/* ---------- simple YAML parser for blueprint format ---------- */

function parseSimpleYaml(yaml: string): { nodes: FlowNode[]; edges: Edge[] } {
  const nodes: FlowNode[] = [];
  const edges: Edge[] = [];

  const lines = yaml.split("\n");
  let section: "none" | "nodes" | "edges" = "none";
  let currentNode: Record<string, unknown> | null = null;

  for (const raw of lines) {
    const line = raw.trimEnd();
    if (line === "nodes:") {
      section = "nodes";
      continue;
    }
    if (line === "edges:") {
      /* flush last node */
      if (currentNode) flushNode(currentNode, nodes);
      currentNode = null;
      section = "edges";
      continue;
    }

    if (section === "nodes") {
      const idMatch = line.match(/^\s+-\s+id:\s+(.+)/);
      if (idMatch) {
        if (currentNode) flushNode(currentNode, nodes);
        currentNode = { id: idMatch[1].trim() };
        continue;
      }
      const kvMatch = line.match(/^\s+(\w+):\s+(.+)/);
      if (kvMatch && currentNode) {
        const key = kvMatch[1];
        let val: unknown = kvMatch[2].trim();
        /* parse array syntax [a, b] */
        const arrMatch = (val as string).match(/^\[(.+)\]$/);
        if (arrMatch) {
          val = arrMatch[1].split(",").map((s) => s.trim()).filter(Boolean);
        }
        /* parse numbers */
        if (typeof val === "string" && /^\d+$/.test(val)) {
          val = parseInt(val);
        }
        currentNode[key] = val;
      }
    }

    if (section === "edges") {
      const fromMatch = line.match(/^\s+-\s+from:\s+(.+)/);
      if (fromMatch) {
        edges.push({ id: `e-${edges.length}`, source: fromMatch[1].trim(), target: "" });
        continue;
      }
      const toMatch = line.match(/^\s+to:\s+(.+)/);
      if (toMatch && edges.length > 0) {
        edges[edges.length - 1].target = toMatch[1].trim();
      }
    }
  }
  /* flush last node */
  if (currentNode) flushNode(currentNode, nodes);

  return { nodes, edges: edges.filter((e) => e.target !== "") };
}

function flushNode(raw: Record<string, unknown>, nodes: FlowNode[]) {
  const id = raw.id as string;
  const nodeType = (raw.type as FlowNodeType) ?? "agent";
  const config: Record<string, unknown> = {};
  for (const [k, v] of Object.entries(raw)) {
    if (k === "id" || k === "type") continue;
    config[k] = v;
  }
  nodes.push({
    id,
    type: nodeType,
    position: { x: 100 + nodes.length * 200, y: 100 + nodes.length * 80 },
    data: {
      label: nodeType.charAt(0).toUpperCase() + nodeType.slice(1),
      nodeType,
      config,
    },
  });
}
