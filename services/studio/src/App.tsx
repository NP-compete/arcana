import { useState } from "react";
import {
  BrowserRouter,
  Routes,
  Route,
  useNavigate,
  useLocation,
  useParams,
  Navigate,
} from "react-router-dom";
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
  Button,
  Spinner,
} from "@patternfly/react-core";
import {
  HomeIcon,
  RobotIcon,
  CubesIcon,
  ChartBarIcon,
  CogIcon,
  PluggedIcon,
  CodeIcon,
  BrainIcon,
  UserIcon,
  SignOutAltIcon,
  ProjectDiagramIcon,
  ShoppingCartIcon,
  UsersIcon,
  MoneyBillIcon,
  ClipboardCheckIcon,
  CommentsIcon,
  ShieldAltIcon,
  PlusCircleIcon,
  CheckCircleIcon,
  EditIcon,
} from "@patternfly/react-icons";
import { AuthProvider, useAuth } from "./auth/AuthContext";
import { LoginPage } from "./pages/LoginPage";
import { DashboardPage } from "./pages/DashboardPage";
import { ChatDrawer } from "./pages/ChatDrawer";
import { AgentsPage } from "./pages/AgentsPage";
import { AgentDetailView } from "./pages/AgentDetailView";
import { ConnectorsPage } from "./pages/ConnectorsPage";
import { McpServersPage } from "./pages/McpServersPage";
import { ModelsPage } from "./pages/ModelsPage";
import { SkillsPage } from "./pages/SkillsPage";
import { EvaluationsPage } from "./pages/EvaluationsPage";
import { SettingsPage } from "./pages/SettingsPage";
import { AgentChatPage } from "./pages/AgentChatPage";
import { FlowBuilderPage } from "./pages/FlowBuilderPage";
import { MarketplacePage } from "./pages/MarketplacePage";
import { OrgChartPage } from "./pages/OrgChartPage";
import { FinOpsDashboardPage } from "./pages/FinOpsDashboardPage";
import { AuditExplorerPage } from "./pages/AuditExplorerPage";
import { PlatformChatPage } from "./pages/PlatformChatPage";
import { GuardrailBuilderPage } from "./pages/GuardrailBuilderPage";
import { ApprovalsPage } from "./pages/ApprovalsPage";
import { BuildHubPage } from "./pages/BuildHubPage";
import { YamlEditorPage } from "./pages/YamlEditorPage";
import { CommandPalette } from "./components/CommandPalette";
import { ThemeToggle } from "./components/ThemeToggle";

interface NavEntry {
  path: string;
  label: string;
  userLabel?: string;
  icon: React.ReactNode;
  allowedRoles: string[];
  group?: string;
  userGroup?: string;
}

const TECH_ROLES = new Set(["developer", "data-engineer", "sre", "auditor", "admin"]);

const NAV_ITEMS: NavEntry[] = [
  // Core
  { path: "/", label: "Overview", userLabel: "Home", icon: <HomeIcon />, allowedRoles: ["user", "developer", "data-engineer", "sre", "auditor", "admin"] },
  { path: "/agents", label: "Agents", userLabel: "My Agents", icon: <RobotIcon />, allowedRoles: ["user", "developer", "data-engineer", "sre", "admin"] },
  { path: "/chat", label: "Chat", icon: <CommentsIcon />, allowedRoles: ["user", "developer", "data-engineer", "sre", "auditor", "admin"] },

  // Build
  { path: "/build", label: "New Agent", icon: <PlusCircleIcon />, allowedRoles: ["developer", "admin"], group: "Build" },
  { path: "/skills", label: "Skills", userLabel: "Capabilities", icon: <CubesIcon />, allowedRoles: ["developer", "admin"], group: "Build" },
  { path: "/models", label: "Models", userLabel: "AI Models", icon: <BrainIcon />, allowedRoles: ["developer", "data-engineer", "admin"], group: "Build" },
  { path: "/flow-builder", label: "Flows", userLabel: "Workflows", icon: <ProjectDiagramIcon />, allowedRoles: ["developer", "admin"], group: "Build" },
  { path: "/mcp", label: "MCP Servers", userLabel: "Integrations", icon: <CodeIcon />, allowedRoles: ["developer", "data-engineer", "admin"], group: "Build" },
  { path: "/connectors", label: "Connectors", userLabel: "Data Sources", icon: <PluggedIcon />, allowedRoles: ["developer", "data-engineer", "admin"], group: "Build" },

  // Discover
  { path: "/marketplace", label: "Marketplace", userLabel: "Browse & Install", icon: <ShoppingCartIcon />, allowedRoles: ["user", "developer", "data-engineer", "sre", "admin"], group: "Discover" },

  // Operate
  { path: "/guardrails", label: "Guardrails", userLabel: "Safety Rules", icon: <ShieldAltIcon />, allowedRoles: ["admin", "developer"], group: "Operate" },
  { path: "/evaluations", label: "Evaluations", userLabel: "Quality Checks", icon: <ChartBarIcon />, allowedRoles: ["developer", "admin"], group: "Operate" },
  { path: "/finops", label: "Usage & Costs", icon: <MoneyBillIcon />, allowedRoles: ["admin", "sre"], group: "Operate" },
  { path: "/audit", label: "Audit Log", userLabel: "Activity Log", icon: <ClipboardCheckIcon />, allowedRoles: ["auditor", "admin"], group: "Operate" },
  { path: "/approvals", label: "Approvals", icon: <CheckCircleIcon />, allowedRoles: ["admin"], group: "Operate" },
];

const ROLE_COLORS: Record<string, string> = {
  admin: "purple",
  developer: "blue",
  "data-engineer": "cyan",
  sre: "gold",
  auditor: "red",
  user: "green",
};

const ShellLayout = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { user, logout, hasRole } = useAuth();
  const [chatOpen, setChatOpen] = useState(false);

  const isTechUser = user?.roles?.some((r) => TECH_ROLES.has(r)) ?? false;

  const getLabel = (item: NavEntry) => isTechUser ? item.label : (item.userLabel ?? item.label);
  const getGroup = (item: NavEntry) => isTechUser ? item.group : (item.userGroup ?? item.group);

  const visibleNav = NAV_ITEMS.filter((item) =>
    item.allowedRoles.some((r) => hasRole(r)),
  );

  const canAccess = (path: string) => {
    const entry = NAV_ITEMS.find((n) => n.path === path);
    if (!entry) return false;
    return entry.allowedRoles.some((r) => hasRole(r));
  };

  const activeItem = visibleNav.find(
    (item) =>
      item.path === "/"
        ? location.pathname === "/"
        : location.pathname.startsWith(item.path),
  )?.path ?? "/";

  return (
    <>
      <Page
        masthead={
          <Masthead className="arcana-masthead">
            <MastheadMain>
              <MastheadBrand data-codemods>
                <div
                  className="arcana-logo"
                  style={{ cursor: "pointer" }}
                  onClick={() => navigate("/")}
                >
                  <div className="arcana-logo-icon">A</div>
                  <span className="arcana-logo-text">Arcana</span>
                </div>
              </MastheadBrand>
            </MastheadMain>
            <MastheadContent>
              <div
                style={{
                  display: "flex",
                  justifyContent: "flex-end",
                  width: "100%",
                  alignItems: "center",
                  gap: 12,
                }}
              >
                <Label className="arcana-env-badge" isCompact>
                  Dev Environment
                </Label>
                <ThemeToggle />
                {user && (
                  <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
                    <div
                      style={{
                        display: "flex",
                        alignItems: "center",
                        gap: 6,
                        background: "rgba(255,255,255,0.08)",
                        borderRadius: 20,
                        padding: "4px 12px 4px 8px",
                      }}
                    >
                      <UserIcon style={{ color: "#8b95a5", fontSize: 13 }} />
                      <span style={{ color: "#c5cdd8", fontSize: 13, fontWeight: 500 }}>
                        {user.user_id}
                      </span>
                      <Label isCompact color={(ROLE_COLORS[user.roles?.[0]] ?? "blue") as any} style={{ marginLeft: 4 }}>
                        {user.roles?.[0] ?? "user"}
                      </Label>
                    </div>
                    <Button
                      variant="plain"
                      aria-label="Sign out"
                      onClick={() => { logout(); navigate("/"); }}
                      style={{ color: "#8b95a5", padding: 4 }}
                    >
                      <SignOutAltIcon />
                    </Button>
                  </div>
                )}
              </div>
            </MastheadContent>
          </Masthead>
        }
        sidebar={
          <PageSidebar>
            <PageSidebarBody style={{ display: "flex", flexDirection: "column", height: "100%" }}>
              <Nav
                onSelect={(_e, result) => {
                  const target = result.itemId as string;
                  if (target) navigate(target);
                }}
                style={{ flex: 1 }}
              >
                <NavList>
                  {(() => {
                    const ungrouped = visibleNav.filter((item) => !getGroup(item));
                    const groups = [...new Set(visibleNav.filter((item) => getGroup(item)).map((item) => getGroup(item)!))];
                    return (
                      <>
                        {ungrouped.map((item) => (
                          <NavItem
                            key={item.path}
                            itemId={item.path}
                            isActive={activeItem === item.path}
                            icon={item.icon}
                          >
                            {getLabel(item)}
                          </NavItem>
                        ))}
                        {groups.map((group) => (
                          <li key={group}>
                            <div className="arcana-nav-group-label">{group}</div>
                            {visibleNav
                              .filter((item) => getGroup(item) === group)
                              .map((item) => (
                                <NavItem
                                  key={item.path}
                                  itemId={item.path}
                                  isActive={activeItem === item.path}
                                  icon={item.icon}
                                >
                                  {getLabel(item)}
                                </NavItem>
                              ))}
                          </li>
                        ))}
                      </>
                    );
                  })()}
                </NavList>
              </Nav>
              {hasRole("sre") || hasRole("auditor") || hasRole("admin") ? (
                <div className="arcana-sidebar-footer">
                  <button
                    type="button"
                    className={`arcana-settings-btn${activeItem === "/settings" ? " active" : ""}`}
                    onClick={() => navigate("/settings")}
                  >
                    <CogIcon /> Settings
                  </button>
                </div>
              ) : null}
            </PageSidebarBody>
          </PageSidebar>
        }
      >
        <Routes>
          <Route path="/" element={<DashboardPage />} />
          <Route path="/chat" element={canAccess("/chat") ? <PlatformChatPage /> : <Navigate to="/" replace />} />
          <Route path="/org-chart" element={canAccess("/org-chart") ? <OrgChartPage /> : <Navigate to="/" replace />} />
          <Route path="/agents" element={<AgentsPage />} />
          <Route path="/agents/:name" element={<AgentDetailRoute />} />
          <Route path="/connectors" element={canAccess("/connectors") ? <ConnectorsPage /> : <Navigate to="/" replace />} />
          <Route path="/mcp" element={canAccess("/mcp") ? <McpServersPage /> : <Navigate to="/" replace />} />
          <Route path="/models" element={canAccess("/models") ? <ModelsPage /> : <Navigate to="/" replace />} />
          <Route path="/skills" element={canAccess("/skills") ? <SkillsPage /> : <Navigate to="/" replace />} />
          <Route path="/flow-builder" element={canAccess("/flow-builder") ? <FlowBuilderPage /> : <Navigate to="/" replace />} />
          <Route path="/marketplace" element={canAccess("/marketplace") ? <MarketplacePage /> : <Navigate to="/" replace />} />
          <Route path="/evaluations" element={canAccess("/evaluations") ? <EvaluationsPage /> : <Navigate to="/" replace />} />
          <Route path="/guardrails" element={canAccess("/guardrails") ? <GuardrailBuilderPage /> : <Navigate to="/" replace />} />
          <Route path="/finops" element={canAccess("/finops") ? <FinOpsDashboardPage /> : <Navigate to="/" replace />} />
          <Route path="/build" element={canAccess("/build") ? <BuildHubPage /> : <Navigate to="/" replace />} />
          <Route path="/editor" element={canAccess("/editor") ? <YamlEditorPage /> : <Navigate to="/" replace />} />
          <Route path="/audit" element={canAccess("/audit") ? <AuditExplorerPage /> : <Navigate to="/" replace />} />
          <Route path="/approvals" element={canAccess("/approvals") ? <ApprovalsPage /> : <Navigate to="/" replace />} />
          <Route path="/settings" element={canAccess("/settings") ? <SettingsPage /> : <Navigate to="/" replace />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </Page>

      <CommandPalette />
      <ChatDrawer isOpen={chatOpen} onClose={() => setChatOpen(false)} />

      {!chatOpen && (
        <button
          className="arcana-chat-fab"
          onClick={() => setChatOpen(true)}
          aria-label="Open Arcana Chat"
        >
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
            <path
              d="M20 2H4C2.9 2 2 2.9 2 4V22L6 18H20C21.1 18 22 17.1 22 16V4C22 2.9 21.1 2 20 2ZM20 16H5.17L4 17.17V4H20V16Z"
              fill="currentColor"
            />
            <path d="M7 9H17V11H7V9ZM7 6H17V8H7V6ZM7 12H14V14H7V12Z" fill="currentColor" />
          </svg>
        </button>
      )}
    </>
  );
};

const AuthGate = () => {
  const { user, loading } = useAuth();

  if (loading) {
    return (
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          height: "100vh",
          background: "linear-gradient(135deg, #0f1117 0%, #1a1d2e 50%, #0d2137 100%)",
          flexDirection: "column",
          gap: 16,
        }}
      >
        <Spinner size="xl" />
        <span style={{ color: "#8b95a5", fontSize: 14 }}>Connecting to Arcana...</span>
      </div>
    );
  }

  if (!user) return <LoginPage />;

  return (
    <Routes>
      <Route path="/agents/:name/chat" element={<AgentChatRoute />} />
      <Route path="*" element={<ShellLayout />} />
    </Routes>
  );
};

const AgentDetailRoute = () => {
  const navigate = useNavigate();
  const { name } = useParams();
  if (!name) return <Navigate to="/agents" replace />;
  return <AgentDetailView agentName={name} onBack={() => navigate("/agents")} />;
};

const AgentChatRoute = () => {
  const { name } = useParams();
  if (!name) return <Navigate to="/agents" replace />;
  return <AgentChatPage agentName={name} />;
};

export const App = () => (
  <BrowserRouter>
    <AuthProvider>
      <AuthGate />
    </AuthProvider>
  </BrowserRouter>
);
