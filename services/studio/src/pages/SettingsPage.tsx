import {
  PageSection,
  Title,
  Card,
  CardTitle,
  CardBody,
  DescriptionList,
  DescriptionListGroup,
  DescriptionListTerm,
  DescriptionListDescription,
  Label,
  Grid,
  GridItem,
  Content,
  Divider,
} from "@patternfly/react-core";

export const SettingsPage = () => (
  <>
    <PageSection hasBodyWrapper={false}>
      <Title headingLevel="h1" size="2xl">Settings</Title>
      <Content component="p">
        Platform configuration and connection details.
      </Content>
    </PageSection>
    <Divider />
    <PageSection hasBodyWrapper={false}>
      <Grid hasGutter>
        <GridItem span={6}>
          <Card>
            <CardTitle>Service Endpoints</CardTitle>
            <CardBody>
              <DescriptionList isHorizontal isCompact>
                <DescriptionListGroup>
                  <DescriptionListTerm>API Gateway</DescriptionListTerm>
                  <DescriptionListDescription>
                    <code>http://localhost:8080</code>
                  </DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                  <DescriptionListTerm>AG-UI Events</DescriptionListTerm>
                  <DescriptionListDescription>
                    <code>http://localhost:8084/events</code>
                  </DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                  <DescriptionListTerm>Temporal UI</DescriptionListTerm>
                  <DescriptionListDescription>
                    <a href="http://localhost:8233" target="_blank" rel="noopener">
                      http://localhost:8233
                    </a>
                  </DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                  <DescriptionListTerm>MinIO Console</DescriptionListTerm>
                  <DescriptionListDescription>
                    <a href="http://localhost:9001" target="_blank" rel="noopener">
                      http://localhost:9001
                    </a>
                  </DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                  <DescriptionListTerm>NATS Monitor</DescriptionListTerm>
                  <DescriptionListDescription>
                    <a href="http://localhost:8222" target="_blank" rel="noopener">
                      http://localhost:8222
                    </a>
                  </DescriptionListDescription>
                </DescriptionListGroup>
              </DescriptionList>
            </CardBody>
          </Card>
        </GridItem>
        <GridItem span={6}>
          <Card>
            <CardTitle>Cluster</CardTitle>
            <CardBody>
              <DescriptionList isHorizontal isCompact>
                <DescriptionListGroup>
                  <DescriptionListTerm>Context</DescriptionListTerm>
                  <DescriptionListDescription>
                    <Label isCompact>kind-arcana-dev</Label>
                  </DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                  <DescriptionListTerm>Kubernetes</DescriptionListTerm>
                  <DescriptionListDescription>v1.35.0</DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                  <DescriptionListTerm>Runtime</DescriptionListTerm>
                  <DescriptionListDescription>Kind (single node)</DescriptionListDescription>
                </DescriptionListGroup>
              </DescriptionList>
            </CardBody>
          </Card>
        </GridItem>
      </Grid>
    </PageSection>
  </>
);
