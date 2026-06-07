import { useState } from "react";
import {
  Card,
  CardBody,
  CardTitle,
  Alert,
  Label,
  Button,
  Content,
  Grid,
  GridItem,
  Spinner,
} from "@patternfly/react-core";
import { Table, Thead, Tr, Th, Tbody, Td } from "@patternfly/react-table";

interface GenerativeUIComponent {
  type: "card" | "alert" | "table" | "label" | "button" | "text" | "grid" | "stat";
  props?: Record<string, unknown>;
  children?: GenerativeUIComponent[];
  content?: string;
}

interface GenerativeUIProps {
  spec: GenerativeUIComponent;
  onAction?: (action: string, data?: unknown) => void;
}

export const GenerativeUI = ({ spec, onAction }: GenerativeUIProps) => {
  return <RenderComponent component={spec} onAction={onAction} />;
};

const RenderComponent = ({
  component,
  onAction,
}: {
  component: GenerativeUIComponent;
  onAction?: (action: string, data?: unknown) => void;
}) => {
  const { type, props = {}, children = [], content = "" } = component;

  switch (type) {
    case "card":
      return (
        <Card>
          {props.title && <CardTitle>{String(props.title)}</CardTitle>}
          <CardBody>
            {content && <Content component="p">{content}</Content>}
            {children.map((child, i) => (
              <RenderComponent key={i} component={child} onAction={onAction} />
            ))}
          </CardBody>
        </Card>
      );

    case "alert":
      return (
        <Alert
          variant={(props.variant as "success" | "danger" | "warning" | "info") ?? "info"}
          title={String(props.title ?? content)}
          isInline
          style={{ marginBottom: 8 }}
        >
          {content && !props.title ? undefined : content}
        </Alert>
      );

    case "table": {
      const columns = (props.columns as string[]) ?? [];
      const rows = (props.rows as string[][]) ?? [];
      return (
        <Table aria-label="Generated table" variant="compact">
          <Thead>
            <Tr>
              {columns.map((col, i) => (
                <Th key={i}>{col}</Th>
              ))}
            </Tr>
          </Thead>
          <Tbody>
            {rows.map((row, ri) => (
              <Tr key={ri}>
                {row.map((cell, ci) => (
                  <Td key={ci} dataLabel={columns[ci] ?? ""}>{cell}</Td>
                ))}
              </Tr>
            ))}
          </Tbody>
        </Table>
      );
    }

    case "label":
      return (
        <Label
          color={(props.color as "blue" | "green" | "red" | "orange" | "grey") ?? "blue"}
          isCompact={Boolean(props.compact)}
          style={{ marginRight: 4 }}
        >
          {content}
        </Label>
      );

    case "button":
      return (
        <Button
          variant={(props.variant as "primary" | "secondary" | "link") ?? "primary"}
          onClick={() => onAction?.(String(props.action ?? "click"), props.data)}
          style={{ marginRight: 8, marginTop: 8 }}
        >
          {content}
        </Button>
      );

    case "text":
      return <Content component="p" style={{ marginBottom: 8 }}>{content}</Content>;

    case "grid":
      return (
        <Grid hasGutter>
          {children.map((child, i) => (
            <GridItem span={(props.span as 1 | 2 | 3 | 4 | 6 | 12) ?? 6} key={i}>
              <RenderComponent component={child} onAction={onAction} />
            </GridItem>
          ))}
        </Grid>
      );

    case "stat":
      return (
        <div style={{
          background: "rgba(255,255,255,0.04)",
          borderRadius: 8,
          padding: "16px",
          textAlign: "center",
        }}>
          <div style={{ fontSize: 28, fontWeight: 700 }}>{content}</div>
          <div style={{ fontSize: 12, color: "var(--pf-t--global--text--color--subtle)", marginTop: 4 }}>
            {String(props.label ?? "")}
          </div>
        </div>
      );

    default:
      return <Content component="p">[Unknown component: {type}]</Content>;
  }
};

export const GenerativeUIPreview = () => {
  const [spec] = useState<GenerativeUIComponent>({
    type: "card",
    props: { title: "Agent Report" },
    children: [
      { type: "alert", props: { variant: "success", title: "All systems operational" }, content: "" },
      {
        type: "grid",
        props: { span: 4 },
        children: [
          { type: "stat", content: "12", props: { label: "Agents" } },
          { type: "stat", content: "89%", props: { label: "Success Rate" } },
          { type: "stat", content: "$42", props: { label: "Cost Today" } },
        ],
      },
      {
        type: "table",
        props: {
          columns: ["Agent", "Status", "Tasks"],
          rows: [
            ["researcher", "active", "47"],
            ["support-bot", "idle", "342"],
          ],
        },
        content: "",
      },
      { type: "button", content: "View Details", props: { action: "navigate", data: "/agents" } },
    ],
  });

  return <GenerativeUI spec={spec} onAction={(a, d) => console.log("action:", a, d)} />;
};
