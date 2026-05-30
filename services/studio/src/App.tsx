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
  Label,
} from "@patternfly/react-core";
import {
  HomeIcon,
  RobotIcon,
  CubesIcon,
  ChartBarIcon,
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
  { id: "evaluations", label: "Evaluations", icon: <ChartBarIcon /> },
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
        <Masthead className="arcana-masthead">
          <MastheadMain>
            <MastheadBrand data-codemods>
              <div className="arcana-logo">
                <div className="arcana-logo-icon">A</div>
                <span className="arcana-logo-text">Arcana</span>
                <span className="arcana-logo-badge">Studio</span>
              </div>
            </MastheadBrand>
          </MastheadMain>
          <MastheadContent>
            <div style={{ display: "flex", justifyContent: "flex-end", width: "100%", alignItems: "center", gap: 12 }}>
              <Label className="arcana-env-badge" isCompact>Dev Environment</Label>
            </div>
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
