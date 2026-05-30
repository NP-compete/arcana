import {
  PageSection,
  Title,
  Card,
  CardTitle,
  CardBody,
  Grid,
  GridItem,
  Label,
  Flex,
  FlexItem,
  Spinner,
  DescriptionList,
  DescriptionListGroup,
  DescriptionListTerm,
  DescriptionListDescription,
  Alert,
  Divider,
  Content,
} from "@patternfly/react-core";
import {
  CheckCircleIcon,
  ExclamationCircleIcon,
  DatabaseIcon,
  ServerIcon,
  ClusterIcon,
  StorageDomainIcon,
  CloudUploadAltIcon,
} from "@patternfly/react-icons";
import { useHealth } from "../hooks/useHealth";

const SERVICE_ICONS: Record<string, React.ReactNode> = {
  PostgreSQL: <DatabaseIcon />,
  Redis: <ServerIcon />,
  Temporal: <ClusterIcon />,
  MinIO: <StorageDomainIcon />,
  NATS: <CloudUploadAltIcon />,
};

const CRDS = [
  { name: "ArcanaAgent", short: "aag", purpose: "Agent lifecycle & configuration" },
  { name: "ArcanaTenant", short: "aten", purpose: "Multi-tenant isolation" },
  { name: "ArcanaSkillRegistry", short: "askr", purpose: "Skill catalog & versioning" },
  { name: "ArcanaEvalSuite", short: "aes", purpose: "Skill evaluation pipelines" },
  { name: "ArcanaRole", short: "arole", purpose: "RBAC + ABAC policies" },
  { name: "ArcanaBudget", short: "abud", purpose: "FinOps token/compute budgets" },
  { name: "ArcanaBackupPolicy", short: "abkp", purpose: "Backup scheduling & retention" },
  { name: "ArcanaPromotion", short: "aprom", purpose: "Environment promotion gates" },
];

export const DashboardPage = () => {
  const { health, error, loading } = useHealth(5000);

  const healthyCount = health?.services.filter((s) => s.status === "healthy").length ?? 0;
  const totalCount = health?.services.length ?? 0;

  return (
    <>
      <PageSection hasBodyWrapper={false}>
        <Flex>
          <FlexItem>
            <Title headingLevel="h1" size="2xl">
              Dashboard
            </Title>
          </FlexItem>
          <FlexItem align={{ default: "alignRight" }}>
            {health && (
              <Label color="blue" isCompact>
                v{health.version} &middot; up {health.uptime}
              </Label>
            )}
          </FlexItem>
        </Flex>
        <Content component="p">
          Platform health and infrastructure overview.
        </Content>
      </PageSection>

      <Divider />

      {error && (
        <PageSection hasBodyWrapper={false}>
          <Alert variant="warning" title="API unreachable" isInline>
            Cannot reach arcana-api. Start it with: <code>./bin/arcana-api</code>
          </Alert>
        </PageSection>
      )}

      <PageSection hasBodyWrapper={false}>
        <Grid hasGutter>
          <GridItem span={8}>
            <Card isFullHeight>
              <CardTitle>
                <Flex>
                  <FlexItem>Backing Services</FlexItem>
                  <FlexItem align={{ default: "alignRight" }}>
                    {loading ? (
                      <Spinner size="md" />
                    ) : (
                      <Label color={healthyCount === totalCount ? "green" : "red"}>
                        {healthyCount}/{totalCount} healthy
                      </Label>
                    )}
                  </FlexItem>
                </Flex>
              </CardTitle>
              <CardBody>
                {health && (
                  <DescriptionList isHorizontal>
                    {health.services.map((svc) => (
                      <DescriptionListGroup key={svc.name}>
                        <DescriptionListTerm>
                          <Flex spaceItems={{ default: "spaceItemsSm" }}>
                            <FlexItem>{SERVICE_ICONS[svc.name] ?? <ServerIcon />}</FlexItem>
                            <FlexItem>{svc.name}</FlexItem>
                          </Flex>
                        </DescriptionListTerm>
                        <DescriptionListDescription>
                          <Flex spaceItems={{ default: "spaceItemsMd" }}>
                            <FlexItem>
                              {svc.status === "healthy" ? (
                                <Label color="green" icon={<CheckCircleIcon />} isCompact>
                                  Healthy
                                </Label>
                              ) : (
                                <Label color="red" icon={<ExclamationCircleIcon />} isCompact>
                                  Unreachable
                                </Label>
                              )}
                            </FlexItem>
                            <FlexItem>
                              <Label color="grey" isCompact>
                                :{svc.port}
                              </Label>
                            </FlexItem>
                            <FlexItem>
                              <Label color="grey" isCompact>
                                {svc.latency}
                              </Label>
                            </FlexItem>
                          </Flex>
                        </DescriptionListDescription>
                      </DescriptionListGroup>
                    ))}
                  </DescriptionList>
                )}
              </CardBody>
            </Card>
          </GridItem>

          <GridItem span={4}>
            <Card isFullHeight>
              <CardTitle>Platform</CardTitle>
              <CardBody>
                <DescriptionList>
                  <DescriptionListGroup>
                    <DescriptionListTerm>Version</DescriptionListTerm>
                    <DescriptionListDescription>
                      {health?.version ?? "—"}
                    </DescriptionListDescription>
                  </DescriptionListGroup>
                  <DescriptionListGroup>
                    <DescriptionListTerm>Uptime</DescriptionListTerm>
                    <DescriptionListDescription>
                      {health?.uptime ?? "—"}
                    </DescriptionListDescription>
                  </DescriptionListGroup>
                  <DescriptionListGroup>
                    <DescriptionListTerm>Cluster</DescriptionListTerm>
                    <DescriptionListDescription>
                      <Label color="green" isCompact icon={<CheckCircleIcon />}>
                        arcana-dev
                      </Label>
                    </DescriptionListDescription>
                  </DescriptionListGroup>
                  <DescriptionListGroup>
                    <DescriptionListTerm>CRDs</DescriptionListTerm>
                    <DescriptionListDescription>
                      <Label color="blue" isCompact>{CRDS.length} installed</Label>
                    </DescriptionListDescription>
                  </DescriptionListGroup>
                  <DescriptionListGroup>
                    <DescriptionListTerm>Protocols</DescriptionListTerm>
                    <DescriptionListDescription>
                      <Flex spaceItems={{ default: "spaceItemsXs" }}>
                        {["MCP", "A2A", "ACP", "AG-UI", "ACS"].map((p) => (
                          <FlexItem key={p}>
                            <Label color="purple" isCompact>{p}</Label>
                          </FlexItem>
                        ))}
                      </Flex>
                    </DescriptionListDescription>
                  </DescriptionListGroup>
                </DescriptionList>
              </CardBody>
            </Card>
          </GridItem>

          <GridItem span={12}>
            <Card>
              <CardTitle>Custom Resource Definitions</CardTitle>
              <CardBody>
                <Grid hasGutter>
                  {CRDS.map((crd) => (
                    <GridItem span={3} key={crd.name}>
                      <Card isCompact isPlain>
                        <CardBody>
                          <Flex direction={{ default: "column" }} spaceItems={{ default: "spaceItemsXs" }}>
                            <FlexItem>
                              <strong>{crd.name}</strong>
                            </FlexItem>
                            <FlexItem>
                              <Label color="grey" isCompact>{crd.short}</Label>
                            </FlexItem>
                            <FlexItem>
                              <Content component="small">{crd.purpose}</Content>
                            </FlexItem>
                          </Flex>
                        </CardBody>
                      </Card>
                    </GridItem>
                  ))}
                </Grid>
              </CardBody>
            </Card>
          </GridItem>
        </Grid>
      </PageSection>
    </>
  );
};
