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
import { CubesIcon } from "@patternfly/react-icons";

export const SkillsPage = () => (
  <>
    <PageSection hasBodyWrapper={false}>
      <Title headingLevel="h1" size="2xl">Skills</Title>
      <Content component="p">
        Browse and manage the 3-tier skill catalog (Planning, Functional, Atomic).
      </Content>
    </PageSection>
    <Divider />
    <PageSection hasBodyWrapper={false}>
      <EmptyState
        titleText="Skill registry is empty"
        headingLevel="h2"
        icon={CubesIcon}
      >
        <EmptyStateBody>
          Register your first skill to the ArcanaSkillRegistry. Skills are
          automatically evaluated and assigned quality badges.
        </EmptyStateBody>
        <EmptyStateFooter>
          <EmptyStateActions>
            <Button variant="primary">Register Skill</Button>
          </EmptyStateActions>
        </EmptyStateFooter>
      </EmptyState>
    </PageSection>
  </>
);
