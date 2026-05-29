import {
  Brand,
  Masthead,
  MastheadBrand,
  MastheadMain,
  MastheadContent,
  Nav,
  NavItem,
  NavList,
  Page,
  PageSection,
  PageSidebar,
  PageSidebarBody,
  Title,
  EmptyState,
  EmptyStateBody,
} from "@patternfly/react-core";

export const App = () => {
  return (
    <Page
      masthead={
        <Masthead>
          <MastheadMain>
            <MastheadBrand>
              <Brand alt="Arcana" heights={{ default: "36px" }}>
                <source srcSet="" />
              </Brand>
              <Title headingLevel="h1" size="lg">
                Arcana Studio
              </Title>
            </MastheadBrand>
          </MastheadMain>
          <MastheadContent>&nbsp;</MastheadContent>
        </Masthead>
      }
      sidebar={
        <PageSidebar>
          <PageSidebarBody>
            <Nav>
              <NavList>
                <NavItem isActive>Dashboard</NavItem>
                <NavItem>Agents</NavItem>
                <NavItem>Skills</NavItem>
                <NavItem>Models</NavItem>
                <NavItem>Evaluations</NavItem>
                <NavItem>Settings</NavItem>
              </NavList>
            </Nav>
          </PageSidebarBody>
        </PageSidebar>
      }
    >
      <PageSection>
        <EmptyState titleText="Welcome to Arcana" headingLevel="h2" icon={undefined}>
          <EmptyStateBody>
            Arcana Studio is initializing. Connect your first agent to get started.
          </EmptyStateBody>
        </EmptyState>
      </PageSection>
    </Page>
  );
};
