import {
  PageSection,
  Title,
  EmptyState,
  EmptyStateBody,
  Content,
  Divider,
} from "@patternfly/react-core";
import { SearchIcon } from "@patternfly/react-icons";

export const EvaluationsPage = () => (
  <>
    <PageSection hasBodyWrapper={false}>
      <Title headingLevel="h1" size="2xl">Evaluations</Title>
      <Content component="p">
        Three-condition skill evaluation with 4-tier judge pipeline and quality badges.
      </Content>
    </PageSection>
    <Divider />
    <PageSection hasBodyWrapper={false}>
      <EmptyState
        titleText="No evaluation suites configured"
        headingLevel="h2"
        icon={SearchIcon}
      >
        <EmptyStateBody>
          Create an ArcanaEvalSuite CR to define evaluation cases and judges
          for your skills.
        </EmptyStateBody>
      </EmptyState>
    </PageSection>
  </>
);
