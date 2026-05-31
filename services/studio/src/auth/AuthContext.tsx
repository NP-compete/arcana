import { createContext, useContext, useState, useEffect, useCallback, useMemo, type ReactNode } from "react";

interface User {
  user_id: string;
  tenant: string;
  roles: string[];
  email?: string;
  auth_type: string;
}

type UserRole = "user" | "developer" | "data-engineer" | "sre" | "auditor" | "admin";

interface AuthState {
  user: User | null;
  token: string | null;
  loading: boolean;
  error: string | null;
  login: (token: string) => Promise<boolean>;
  loginAs: (role: string) => Promise<boolean>;
  logout: () => void;
  authHeaders: () => Record<string, string>;
  hasRole: (role: string) => boolean;
  isAtLeast: (minRole: UserRole) => boolean;
}

const AuthContext = createContext<AuthState | null>(null);

export const useAuth = () => {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
};

const STORAGE_KEY = "arcana_auth_token";
const ROLE_KEY = "arcana_auth_role";

const ROLE_INCLUDES: Record<string, Set<UserRole>> = {
  admin:           new Set(["user", "developer", "data-engineer", "sre", "auditor", "admin"]),
  developer:       new Set(["user", "developer"]),
  "data-engineer": new Set(["user", "data-engineer"]),
  sre:             new Set(["user", "sre"]),
  auditor:         new Set(["user", "auditor"]),
  user:            new Set(["user"]),
};

export const AuthProvider = ({ children }: { children: ReactNode }) => {
  const [user, setUser] = useState<User | null>(null);
  const [token, setToken] = useState<string | null>(null);
  const [selectedRole, setSelectedRole] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchMe = useCallback(async (authToken?: string, role?: string): Promise<User | null> => {
    try {
      const headers: Record<string, string> = { "Content-Type": "application/json" };
      if (authToken && authToken !== "open") {
        headers["Authorization"] = `Bearer ${authToken}`;
      }
      if (role) {
        headers["X-Arcana-Role"] = role;
      }
      const res = await fetch("/api/v1/auth/me", { headers });
      if (!res.ok) return null;
      const data = await res.json();
      if (data.user_id) return data as User;
      return null;
    } catch {
      return null;
    }
  }, []);

  useEffect(() => {
    const savedToken = localStorage.getItem(STORAGE_KEY);
    const savedRole = localStorage.getItem(ROLE_KEY);
    if (savedToken) {
      setToken(savedToken);
      if (savedRole) setSelectedRole(savedRole);
      fetchMe(savedToken, savedRole ?? undefined).then((u) => {
        if (u) {
          setUser(u);
        } else if (savedRole) {
          const ROLE_PERSONAS: Record<string, string> = {
            admin: "anonymous", developer: "alex", "data-engineer": "priya",
            sre: "jordan", auditor: "sam", user: "maya",
          };
          const allRoles = ROLE_INCLUDES[savedRole] ? Array.from(ROLE_INCLUDES[savedRole]) : [savedRole];
          setUser({
            user_id: ROLE_PERSONAS[savedRole] ?? savedRole,
            tenant: "default",
            roles: allRoles,
            email: `${ROLE_PERSONAS[savedRole] ?? savedRole}@arcana.local`,
            auth_type: "open",
          });
        } else {
          localStorage.removeItem(STORAGE_KEY);
          localStorage.removeItem(ROLE_KEY);
        }
        setLoading(false);
      });
    } else {
      setLoading(false);
    }
  }, [fetchMe]);

  const login = useCallback(async (apiKey: string): Promise<boolean> => {
    setError(null);
    setLoading(true);
    try {
      const u = await fetchMe(apiKey);
      if (u) {
        setUser(u);
        setToken(apiKey);
        localStorage.setItem(STORAGE_KEY, apiKey);
        setLoading(false);
        return true;
      }
      setError("Invalid API key");
      setLoading(false);
      return false;
    } catch {
      setError("Authentication failed");
      setLoading(false);
      return false;
    }
  }, [fetchMe]);

  const loginAs = useCallback(async (role: string): Promise<boolean> => {
    setError(null);
    setLoading(true);
    let u = await fetchMe("open", role);
    if (!u) {
      const ROLE_PERSONAS: Record<string, string> = {
        admin: "anonymous", developer: "alex", "data-engineer": "priya",
        sre: "jordan", auditor: "sam", user: "maya",
      };
      const allRoles = ROLE_INCLUDES[role] ? Array.from(ROLE_INCLUDES[role]) : [role];
      u = {
        user_id: ROLE_PERSONAS[role] ?? role,
        tenant: "default",
        roles: allRoles,
        email: `${ROLE_PERSONAS[role] ?? role}@arcana.local`,
        auth_type: "open",
      };
    }
    setUser(u);
    setToken("open");
    setSelectedRole(role);
    localStorage.setItem(STORAGE_KEY, "open");
    localStorage.setItem(ROLE_KEY, role);
    setLoading(false);
    return true;
  }, [fetchMe]);

  const logout = useCallback(() => {
    setUser(null);
    setToken(null);
    setSelectedRole(null);
    localStorage.removeItem(STORAGE_KEY);
    localStorage.removeItem(ROLE_KEY);
  }, []);

  const authHeaders = useCallback((): Record<string, string> => {
    const h: Record<string, string> = {};
    if (token && token !== "open") {
      h["Authorization"] = `Bearer ${token}`;
    }
    if (selectedRole) {
      h["X-Arcana-Role"] = selectedRole;
    }
    return h;
  }, [token, selectedRole]);

  const hasRole = useCallback((role: string): boolean => {
    return user?.roles?.includes(role) ?? false;
  }, [user]);

  const isAtLeast = useCallback((minRole: UserRole): boolean => {
    if (!user?.roles) return false;
    return user.roles.some((r) => ROLE_INCLUDES[r]?.has(minRole) ?? false);
  }, [user]);

  const value = useMemo(
    () => ({ user, token, loading, error, login, loginAs, logout, authHeaders, hasRole, isAtLeast }),
    [user, token, loading, error, login, loginAs, logout, authHeaders, hasRole, isAtLeast],
  );

  return (
    <AuthContext.Provider value={value}>
      {children}
    </AuthContext.Provider>
  );
};
