import { useState } from "react";
import {
  Masthead,
  MastheadMain,
  MastheadBrand,
  MastheadContent,
  Nav,
  NavItem,
  NavList,
  Page,
  PageSidebar,
  PageSidebarBody,
  Title,
  Label,
  Flex,
  FlexItem,
} from "@patternfly/react-core";
import {
  HomeIcon,
  RobotIcon,
  CubesIcon,
  SearchIcon,
  CogIcon,
} from "@patternfly/react-icons";
import { DashboardPage } from "./pages/DashboardPage";
import { AgentsPage } from "./pages/AgentsPage";
import { SkillsPage } from "./pages/SkillsPage";
import { EvaluationsPage } from "./pages/EvaluationsPage";
import { SettingsPage } from "./pages/SettingsPage";

type NavPage = "dashboard" | "agents" | "skills" | "evaluations" | "settings";

const NAV_ITEMS: { id: NavPage; label: string; icon: React.ReactNode }[] = [
  { id: "dashboard", label: "Dashboard", icon: <HomeIcon /> },
  { id: "agents", label: "Agents", icon: <RobotIcon /> },
  { id: "skills", label: "Skills", icon: <CubesIcon /> },
  { id: "evaluations", label: "Evaluations", icon: <SearchIcon /> },
  { id: "settings", label: "Settings", icon: <CogIcon /> },
];

const PAGE_MAP: Record<NavPage, React.ReactNode> = {
  dashboard: <DashboardPage />,
  agents: <AgentsPage />,
  skills: <SkillsPage />,
  evaluations: <EvaluationsPage />,
  settings: <SettingsPage />,
};

export const App = () => {
  const [activePage, setActivePage] = useState<NavPage>("dashboard");

  return (
    <Page
      masthead={
        <Masthead>
          <MastheadMain>
            <MastheadBrand data-codemods>
              <Flex spaceItems={{ default: "spaceItemsSm" }} alignItems={{ default: "alignItemsCenter" }}>
                <FlexItem>
                  <Title headingLevel="h1" size="lg">
                    Arcana
                  </Title>
                </FlexItem>
                <FlexItem>
                  <Label color="purple" isCompact>Studio</Label>
                </FlexItem>
              </Flex>
            </MastheadBrand>
          </MastheadMain>
          <MastheadContent>
            <Flex justifyContent={{ default: "justifyContentFlexEnd" }} style={{ width: "100%" }}>
              <FlexItem>
                <Label color="blue" isCompact>Dev</Label>
              </FlexItem>
            </Flex>
          </MastheadContent>
        </Masthead>
      }
      sidebar={
        <PageSidebar>
          <PageSidebarBody>
            <Nav onSelect={(_e, result) => setActivePage(result.itemId as NavPage)}>
              <NavList>
                {NAV_ITEMS.map((item) => (
                  <NavItem
                    key={item.id}
                    itemId={item.id}
                    isActive={activePage === item.id}
                    icon={item.icon}
                  >
                    {item.label}
                  </NavItem>
                ))}
              </NavList>
            </Nav>
          </PageSidebarBody>
        </PageSidebar>
      }
    >
      {PAGE_MAP[activePage]}
    </Page>
  );
};
