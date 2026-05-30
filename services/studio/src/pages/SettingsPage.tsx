import {
  PageSection,
  Title,
  Card,
  CardTitle,
  CardBody,
  Grid,
  GridItem,
  Content,
  Divider,
  Label,
  Button,
} from "@patternfly/react-core";
import {
  ExternalLinkAltIcon,
  ServerIcon,
  ClusterIcon,
} from "@patternfly/react-icons";

const ENDPOINTS = [
  { name: "API Gateway", url: "http://localhost:8080", port: 8080, internal: true },
  { name: "AG-UI Events", url: "http://localhost:8084/events", port: 8084, internal: true },
  { name: "Temporal UI", url: "http://localhost:8233", port: 8233, internal: false },
  { name: "MinIO Console", url: "http://localhost:9001", port: 9001, internal: false },
  { name: "NATS Monitor", url: "http://localhost:8222", port: 8222, internal: false },
];

const CLUSTER_INFO = [
  { label: "Context", value: "kind-arcana-dev" },
  { label: "Kubernetes", value: "v1.35.0" },
  { label: "Runtime", value: "Kind (single-node)" },
  { label: "Container Engine", value: "Docker / Podman (auto-detected)" },
  { label: "CRD Group", value: "arcana.io/v1alpha1" },
];

export const SettingsPage = () => (
  <>
    <PageSection hasBodyWrapper={false}>
      <Title headingLevel="h1" size="2xl">Settings</Title>
      <Content component="p" style={{ marginTop: 4 }}>
        Platform configuration, endpoints, and cluster information.
      </Content>
    </PageSection>
    <Divider />
    <PageSection hasBodyWrapper={false}>
      <Grid hasGutter>
        <GridItem span={7}>
          <Card className="stat-card" isFullHeight>
            <CardTitle>
              <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
                <ServerIcon /> Service Endpoints
              </div>
            </CardTitle>
            <CardBody>
              <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
                {ENDPOINTS.map((ep) => (
                  <div className="service-row" key={ep.name}>
                    <div className="service-row-name" style={{ flex: 1 }}>{ep.name}</div>
                    <Label color="grey" isCompact style={{ fontFamily: "var(--pf-t--global--font--family--mono)" }}>
                      :{ep.port}
                    </Label>
                    {ep.internal ? (
                      <Label color="blue" isCompact>Internal</Label>
                    ) : (
                      <Button
                        variant="link"
                        size="sm"
                        isInline
                        icon={<ExternalLinkAltIcon />}
                        component="a"
                        href={ep.url}
                        target="_blank"
                      >
                        Open
                      </Button>
                    )}
                  </div>
                ))}
              </div>
            </CardBody>
          </Card>
        </GridItem>

        <GridItem span={5}>
          <Card className="stat-card" isFullHeight>
            <CardTitle>
              <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
                <ClusterIcon /> Cluster
              </div>
            </CardTitle>
            <CardBody>
              <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
                {CLUSTER_INFO.map((item) => (
                  <div key={item.label}>
                    <div style={{ fontSize: 12, fontWeight: 600, color: "var(--pf-t--global--text--color--subtle)", textTransform: "uppercase", letterSpacing: 0.5, marginBottom: 4 }}>
                      {item.label}
                    </div>
                    <div style={{ fontWeight: 500, fontSize: 14 }}>{item.value}</div>
                  </div>
                ))}
              </div>
            </CardBody>
          </Card>
        </GridItem>
      </Grid>
    </PageSection>
  </>
);
