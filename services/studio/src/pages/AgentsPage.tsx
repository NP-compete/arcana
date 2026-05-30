import {
  PageSection,
  Title,
  EmptyState,
  EmptyStateBody,
  EmptyStateActions,
  EmptyStateFooter,
  Button,
  Content,
  Divider,
} from "@patternfly/react-core";
import { PlusCircleIcon } from "@patternfly/react-icons";

export const AgentsPage = () => (
  <>
    <PageSection hasBodyWrapper={false}>
      <Title headingLevel="h1" size="2xl">Agents</Title>
      <Content component="p">
        Manage your AI agents. Each agent is a Kubernetes-native ArcanaAgent CR.
      </Content>
    </PageSection>
    <Divider />
    <PageSection hasBodyWrapper={false}>
      <EmptyState
        titleText="No agents deployed"
        headingLevel="h2"
        icon={PlusCircleIcon}
      >
        <EmptyStateBody>
          Create your first agent by applying an ArcanaAgent custom resource
          or using the form below.
        </EmptyStateBody>
        <EmptyStateFooter>
          <EmptyStateActions>
            <Button variant="primary">Create Agent</Button>
          </EmptyStateActions>
          <EmptyStateActions>
            <Button variant="link">View YAML example</Button>
          </EmptyStateActions>
        </EmptyStateFooter>
      </EmptyState>
    </PageSection>
  </>
);
