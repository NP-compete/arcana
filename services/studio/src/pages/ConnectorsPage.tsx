import { useCallback, useEffect, useState } from "react";
import {
  PageSection,
  Title,
  Content,
  Card,
  CardBody,
  Grid,
  GridItem,
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
  Spinner,
  SearchInput,
} from "@patternfly/react-core";
import { Table, Thead, Tr, Th, Tbody, Td } from "@patternfly/react-table";
import {
  PlusCircleIcon,
  SyncAltIcon,
  TrashIcon,
} from "@patternfly/react-icons";
import { ShareBadge } from "../components/ShareBadge";

interface ConnectorType {
  type: string;
  description: string;
}

interface ConnectorInstance {
  name: string;
  type: string;
  status: string;
  last_sync: string | null;
  documents_synced: number;
}

const CONNECTOR_ICONS: Record<string, string> = {
  gdrive: "\uD83D\uDCC1",
  confluence: "\uD83D\uDCD6",
  slack: "\uD83D\uDCAC",
  notion: "\uD83D\uDDD2\uFE0F",
  github: "\uD83D\uDC19",
  gitlab: "\uD83E\uDD8A",
  jira: "\uD83D\uDCCB",
  s3: "\uD83E\uDEA3",
  postgres: "\uD83D\uDC18",
  mysql: "\uD83D\uDDC4\uFE0F",
  snowflake: "\u2744\uFE0F",
  email: "\uD83D\uDCE7",
  web: "\uD83C\uDF10",
  file: "\uD83D\uDCC4",
  mailchimp: "\uD83D\uDCE8",
  hubspot: "\uD83E\uDDE2",
  salesforce: "\u2601\uFE0F",
  bigquery: "\uD83D\uDCCA",
};

const statusColor = (s: string): "green" | "blue" | "orange" | "grey" | "red" => {
  switch (s) {
    case "active": return "green";
    case "registered": return "blue";
    case "syncing": return "orange";
    case "error": return "red";
    default: return "grey";
  }
};

export const ConnectorsPage = () => {
  const [types, setTypes] = useState<ConnectorType[]>([]);
  const [instances, setInstances] = useState<ConnectorInstance[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState("");
  const [modalOpen, setModalOpen] = useState(false);
  const [newName, setNewName] = useState("");
  const [newType, setNewType] = useState("");
  const [syncing, setSyncing] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    try {
      const typesRes = await fetch("/api/v1/connectors");
      const typesData = await typesRes.json();
      setTypes(typesData.connectors || []);

      const instanceList: ConnectorInstance[] = [];
      if (typesData.instances) {
        for (const inst of typesData.instances) {
          instanceList.push(inst);
        }
      }
      setInstances(instanceList);
    } catch {
      setError("Failed to load connectors");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const handleRegister = async () => {
    if (!newName.trim() || !newType) return;
    setError(null);
    try {
      const res = await fetch("/api/v1/connectors", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ type: newType, name: newName.trim() }),
      });
      if (!res.ok) {
        const data = await res.json();
        setError(data.detail || "Registration failed");
        return;
      }
      const inst = await res.json();
      setInstances((prev) => [...prev, inst]);
      setModalOpen(false);
      setNewName("");
      setNewType("");
    } catch {
      setError("Failed to register connector");
    }
  };

  const handleSync = async (name: string) => {
    setSyncing(name);
    try {
      const res = await fetch(`/api/v1/connectors/${name}/sync`, { method: "POST" });
      if (res.ok) {
        const updated = await fetch(`/api/v1/connectors/${name}`);
        if (updated.ok) {
          const inst = await updated.json();
          setInstances((prev) => prev.map((i) => (i.name === name ? inst : i)));
        }
      }
    } catch {
      setError(`Sync failed for ${name}`);
    } finally {
      setSyncing(null);
    }
  };

  const handleDelete = async (name: string) => {
    try {
      await fetch(`/api/v1/connectors/${name}`, { method: "DELETE" });
      setInstances((prev) => prev.filter((i) => i.name !== name));
    } catch {
      setError(`Failed to delete ${name}`);
    }
  };

  const filteredTypes = types.filter(
    (t) =>
      !filter ||
      t.type.toLowerCase().includes(filter.toLowerCase()) ||
      t.description.toLowerCase().includes(filter.toLowerCase()),
  );

  if (loading) {
    return (
      <PageSection hasBodyWrapper={false}>
        <div style={{ textAlign: "center", padding: 60 }}>
          <Spinner size="xl" />
        </div>
      </PageSection>
    );
  }

  return (
    <>
      <PageSection hasBodyWrapper={false}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
          <div>
            <Title headingLevel="h1" size="2xl">Connectors</Title>
            <Content component="p" style={{ marginTop: 4, color: "var(--pf-t--global--text--color--subtle)" }}>
              Data sources and tool integrations for your agents
            </Content>
          </div>
          <Button
            variant="primary"
            icon={<PlusCircleIcon />}
            onClick={() => setModalOpen(true)}
          >
            Add Connector
          </Button>
        </div>
      </PageSection>

      {error && (
        <PageSection hasBodyWrapper={false} style={{ paddingTop: 0 }}>
          <Alert variant="danger" title={error} isInline actionClose={<AlertActionCloseButton onClose={() => setError(null)} />} />
        </PageSection>
      )}

      {instances.length > 0 && (
        <PageSection hasBodyWrapper={false} style={{ paddingTop: 0 }}>
          <Card>
            <CardBody>
              <Title headingLevel="h3" size="lg" style={{ marginBottom: 16 }}>Active Connectors</Title>
              <Table aria-label="Active connectors" variant="compact">
                <Thead>
                  <Tr>
                    <Th>Name</Th>
                    <Th>Type</Th>
                    <Th>Status</Th>
                    <Th>Sharing</Th>
                    <Th>Docs Synced</Th>
                    <Th>Last Sync</Th>
                    <Th>Actions</Th>
                  </Tr>
                </Thead>
                <Tbody>
                  {instances.map((inst) => (
                    <Tr key={inst.name}>
                      <Td dataLabel="Name">
                        <span style={{ fontWeight: 600 }}>
                          {CONNECTOR_ICONS[inst.type] || "\uD83D\uDD0C"} {inst.name}
                        </span>
                      </Td>
                      <Td dataLabel="Type">
                        <Label isCompact color="blue">{inst.type}</Label>
                      </Td>
                      <Td dataLabel="Status">
                        <Label isCompact color={statusColor(inst.status)}>{inst.status}</Label>
                      </Td>
                      <Td dataLabel="Sharing">
                        <ShareBadge assetType="connector" assetName={inst.name} compact />
                      </Td>
                      <Td dataLabel="Docs Synced">{inst.documents_synced}</Td>
                      <Td dataLabel="Last Sync">
                        {inst.last_sync ? new Date(inst.last_sync).toLocaleString() : "—"}
                      </Td>
                      <Td dataLabel="Actions">
                        <Button
                          variant="plain"
                          icon={<SyncAltIcon />}
                          aria-label={`Sync ${inst.name}`}
                          isLoading={syncing === inst.name}
                          onClick={() => handleSync(inst.name)}
                        />
                        <Button
                          variant="plain"
                          isDanger
                          icon={<TrashIcon />}
                          aria-label={`Delete ${inst.name}`}
                          onClick={() => handleDelete(inst.name)}
                        />
                      </Td>
                    </Tr>
                  ))}
                </Tbody>
              </Table>
            </CardBody>
          </Card>
        </PageSection>
      )}

      <PageSection hasBodyWrapper={false} style={{ paddingTop: instances.length > 0 ? 0 : undefined }}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 16 }}>
          <Title headingLevel="h3" size="lg">Available Connector Types</Title>
          <SearchInput
            placeholder="Filter connectors..."
            value={filter}
            onChange={(_e, val) => setFilter(val)}
            onClear={() => setFilter("")}
            style={{ maxWidth: 280 }}
          />
        </div>
        <Grid hasGutter>
          {filteredTypes.map((ct) => (
            <GridItem span={4} key={ct.type}>
              <Card
                isClickable
                onClick={() => {
                  setNewType(ct.type);
                  setNewName(`my-${ct.type}`);
                  setModalOpen(true);
                }}
                className="action-card"
                style={{ height: "100%" }}
              >
                <CardBody>
                  <div style={{ display: "flex", alignItems: "flex-start", gap: 12 }}>
                    <div
                      className="action-card-icon"
                      style={{ margin: 0, flexShrink: 0, fontSize: 24, width: 44, height: 44 }}
                    >
                      {CONNECTOR_ICONS[ct.type] || "\uD83D\uDD0C"}
                    </div>
                    <div>
                      <div style={{ fontWeight: 700, fontSize: 15, marginBottom: 4 }}>
                        {ct.type}
                      </div>
                      <div style={{ fontSize: 13, color: "var(--pf-t--global--text--color--subtle)" }}>
                        {ct.description}
                      </div>
                    </div>
                  </div>
                </CardBody>
              </Card>
            </GridItem>
          ))}
        </Grid>
      </PageSection>

      <Modal
        isOpen={modalOpen}
        onClose={() => setModalOpen(false)}
        variant="small"
        aria-label="Add connector"
      >
        <ModalHeader title="Add Connector" />
        <ModalBody>
          <FormGroup label="Connector Type" fieldId="conn-type" isRequired>
            <FormSelect
              id="conn-type"
              value={newType}
              onChange={(_e, val) => setNewType(val)}
            >
              <FormSelectOption value="" label="Select a type..." isPlaceholder />
              {types.map((t) => (
                <FormSelectOption key={t.type} value={t.type} label={`${CONNECTOR_ICONS[t.type] || ""} ${t.type} — ${t.description}`} />
              ))}
            </FormSelect>
          </FormGroup>
          <FormGroup label="Instance Name" fieldId="conn-name" isRequired style={{ marginTop: 16 }}>
            <TextInput
              id="conn-name"
              value={newName}
              onChange={(_e, val) => setNewName(val)}
              placeholder="e.g. brand-guidelines-drive"
            />
          </FormGroup>
        </ModalBody>
        <ModalFooter>
          <Button
            variant="primary"
            onClick={handleRegister}
            isDisabled={!newName.trim() || !newType}
          >
            Register
          </Button>
          <Button variant="link" onClick={() => setModalOpen(false)}>
            Cancel
          </Button>
        </ModalFooter>
      </Modal>
    </>
  );
};
