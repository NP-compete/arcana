import { useState, useEffect } from "react";
import { Alert, Spinner } from "@patternfly/react-core";
import { useAuth } from "../auth/AuthContext";

interface RoleOption {
  role: string;
  title: string;
  persona: string;
  description: string;
  color: string;
  icon: string;
  capabilities: string[];
}

const ROLE_OPTIONS: RoleOption[] = [
  {
    role: "admin",
    title: "Administrator",
    persona: "Admin",
    description: "Full platform access",
    color: "#a855f7",
    icon: "A",
    capabilities: ["Agents", "Security", "Tenants", "Billing", "All Settings"],
  },
  {
    role: "developer",
    title: "Developer",
    persona: "Alex",
    description: "Build and deploy agents",
    color: "#5b8def",
    icon: "D",
    capabilities: ["Agents", "Skills", "Models", "Blueprints", "Evaluations"],
  },
  {
    role: "data-engineer",
    title: "Data Engineer",
    persona: "Priya",
    description: "Manage data pipelines",
    color: "#06b6d4",
    icon: "DE",
    capabilities: ["Connectors", "Knowledge Base", "Models", "MCP Servers"],
  },
  {
    role: "sre",
    title: "SRE / Platform",
    persona: "Jordan",
    description: "Operate and monitor",
    color: "#f59e0b",
    icon: "S",
    capabilities: ["Health", "Deployments", "FinOps", "Audit Trails"],
  },
  {
    role: "auditor",
    title: "Auditor",
    persona: "Sam",
    description: "Compliance and audit",
    color: "#ef4444",
    icon: "Au",
    capabilities: ["Audit Logs", "Compliance Reports", "Tenant Data"],
  },
  {
    role: "user",
    title: "Business User",
    persona: "Maya",
    description: "Use agents day-to-day",
    color: "#22c55e",
    icon: "U",
    capabilities: ["Chat", "Agents", "Dashboards", "Marketplace"],
  },
];

const VALUE_PROPS = [
  {
    icon: "🚀",
    title: "Deploy in minutes",
    desc: "From conversation to production agent — no custom glue code.",
  },
  {
    icon: "🛡️",
    title: "Enterprise governance",
    desc: "Per-agent guardrails, RBAC, OPA policies, immutable audit trail.",
  },
  {
    icon: "💰",
    title: "Built-in cost control",
    desc: "Token budgets, model fallback chains, team-level spending caps.",
  },
  {
    icon: "🔄",
    title: "Self-improving agents",
    desc: "Skills evolve, memory compacts, corrections become capabilities.",
  },
];

const LOGOS = [
  { name: "Kubernetes", abbr: "K8s" },
  { name: "Temporal", abbr: "⏱" },
  { name: "PostgreSQL", abbr: "PG" },
  { name: "Redis", abbr: "R" },
  { name: "Prometheus", abbr: "P" },
  { name: "OpenTelemetry", abbr: "OT" },
];

export const LoginPage = () => {
  const { login, loginAs, loading, error } = useAuth();
  const [apiKey, setApiKey] = useState("");
  const [localError, setLocalError] = useState<string | null>(null);
  const [mode, setMode] = useState<"roles" | "key">("roles");
  const [selectedRole, setSelectedRole] = useState<string | null>(null);
  const [hoveredRole, setHoveredRole] = useState<string | null>(null);
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    const t = setTimeout(() => setMounted(true), 50);
    return () => clearTimeout(t);
  }, []);

  const handleApiKeySubmit = async (e: React.SyntheticEvent<HTMLFormElement>) => {
    e.preventDefault();
    setLocalError(null);
    if (!apiKey.trim()) { setLocalError("API key is required"); return; }
    const ok = await login(apiKey.trim());
    if (!ok) setLocalError("Invalid API key");
  };

  const handleRoleLogin = async (role: string) => {
    setLocalError(null);
    setSelectedRole(role);
    const ok = await loginAs(role);
    if (!ok) setLocalError("Could not connect to platform");
    setSelectedRole(null);
  };

  const displayError = localError || error;

  return (
    <div style={{ minHeight: "100vh", background: "#030508", overflow: "hidden" }}>
      <style>{`
        @keyframes arcana-float { 0%,100%{transform:translateY(0)} 50%{transform:translateY(-20px)} }
        @keyframes arcana-pulse { 0%,100%{opacity:0.4} 50%{opacity:0.8} }
        @keyframes arcana-grid { 0%{transform:translate(0,0)} 100%{transform:translate(40px,40px)} }
        .arcana-login-role:hover { transform:translateY(-3px)!important; box-shadow:0 8px 24px rgba(0,0,0,0.4)!important }
        .arcana-login-role:active { transform:translateY(0)!important }
        .arcana-api-input:focus { border-color:rgba(91,141,239,0.6)!important; box-shadow:0 0 0 3px rgba(91,141,239,0.1)!important }
        .arcana-cta:hover { transform:translateY(-1px); box-shadow:0 6px 24px rgba(91,141,239,0.4)!important }
      `}</style>

      {/* Grid bg */}
      <div style={{
        position:"fixed",inset:0,
        backgroundImage:"linear-gradient(rgba(91,141,239,0.025) 1px,transparent 1px),linear-gradient(90deg,rgba(91,141,239,0.025) 1px,transparent 1px)",
        backgroundSize:"40px 40px", animation:"arcana-grid 20s linear infinite", pointerEvents:"none",
      }}/>

      {/* Orbs */}
      <div style={{ position:"fixed",width:600,height:600,borderRadius:"50%",background:"radial-gradient(circle,rgba(91,141,239,0.09) 0%,transparent 70%)",top:-200,left:-100,animation:"arcana-float 8s ease-in-out infinite",pointerEvents:"none" }}/>
      <div style={{ position:"fixed",width:500,height:500,borderRadius:"50%",background:"radial-gradient(circle,rgba(168,85,247,0.07) 0%,transparent 70%)",bottom:-100,right:50,animation:"arcana-float 10s ease-in-out infinite 2s",pointerEvents:"none" }}/>

      {/* ===== NAV BAR ===== */}
      <nav style={{
        display:"flex", alignItems:"center", justifyContent:"space-between",
        padding:"16px 48px", position:"relative", zIndex:10,
        borderBottom:"1px solid rgba(255,255,255,0.04)",
      }}>
        <div style={{ display:"flex", alignItems:"center", gap:12 }}>
          <div style={{
            width:36,height:36,borderRadius:10,
            background:"linear-gradient(135deg,#5b8def,#a855f7)",
            display:"flex",alignItems:"center",justifyContent:"center",
            fontSize:16,fontWeight:800,color:"#fff",
            boxShadow:"0 2px 12px rgba(91,141,239,0.3)",
          }}>A</div>
          <span style={{ fontSize:18,fontWeight:700,color:"#fff",letterSpacing:-0.3 }}>Arcana</span>
          <span style={{
            fontSize:9,fontWeight:700,color:"#667eea",letterSpacing:2,
            textTransform:"uppercase",background:"rgba(91,141,239,0.1)",
            padding:"3px 8px",borderRadius:6,marginLeft:2,
          }}>Platform</span>
        </div>
        <div style={{ display:"flex", alignItems:"center", gap:24 }}>
          <a href="https://github.com/NP-compete/arcana" target="_blank" rel="noreferrer" style={{ fontSize:13,color:"#5a6a7d",textDecoration:"none",fontWeight:500 }}>Docs</a>
          <a href="https://github.com/NP-compete/arcana" target="_blank" rel="noreferrer" style={{ fontSize:13,color:"#5a6a7d",textDecoration:"none",fontWeight:500 }}>GitHub</a>
          <a href="https://github.com/NP-compete/arcana/blob/main/CONTRIBUTING.md" target="_blank" rel="noreferrer" style={{ fontSize:13,color:"#5a6a7d",textDecoration:"none",fontWeight:500 }}>Contribute</a>
          <button onClick={() => document.getElementById("login-panel")?.scrollIntoView({ behavior:"smooth" })} style={{
            padding:"8px 20px",borderRadius:8,border:"1px solid rgba(91,141,239,0.3)",
            background:"rgba(91,141,239,0.08)",color:"#7ba4f0",fontSize:13,fontWeight:600,
            cursor:"pointer",transition:"all 0.2s",
          }}>Sign In</button>
        </div>
      </nav>

      {/* ===== HERO SECTION ===== */}
      <section style={{
        textAlign:"center", padding:"80px 48px 60px", position:"relative", zIndex:1,
        maxWidth:800, margin:"0 auto",
        opacity:mounted?1:0, transform:mounted?"translateY(0)":"translateY(20px)",
        transition:"all 0.8s cubic-bezier(0.16,1,0.3,1)",
      }}>
        <div style={{
          display:"inline-flex",alignItems:"center",gap:8,
          background:"rgba(91,141,239,0.08)",border:"1px solid rgba(91,141,239,0.15)",
          borderRadius:20,padding:"6px 16px",marginBottom:28,
        }}>
          <span style={{ width:6,height:6,borderRadius:"50%",background:"#22c55e",display:"inline-block" }}/>
          <span style={{ fontSize:12,color:"#7ba4f0",fontWeight:500 }}>Open Source &middot; Apache 2.0 &middot; v0.1.0</span>
        </div>

        <h1 style={{
          fontSize:56,fontWeight:800,lineHeight:1.1,margin:"0 0 20px",letterSpacing:-2,
          color:"#fff",
        }}>
          The operating system<br/>
          <span style={{
            background:"linear-gradient(135deg,#667eea 0%,#a855f7 40%,#06b6d4 100%)",
            WebkitBackgroundClip:"text",WebkitTextFillColor:"transparent",
          }}>for enterprise AI agents</span>
        </h1>

        <p style={{ fontSize:18,color:"#5a6a7d",lineHeight:1.7,margin:"0 auto 36px",maxWidth:560 }}>
          Build, deploy, govern, and continuously improve AI agents.
          One platform for every team — from marketing to engineering to compliance.
        </p>

        <div style={{ display:"flex",gap:12,justifyContent:"center",marginBottom:48 }}>
          <button className="arcana-cta" onClick={() => document.getElementById("login-panel")?.scrollIntoView({ behavior:"smooth" })} style={{
            padding:"14px 32px",borderRadius:12,border:"none",
            background:"linear-gradient(135deg,#5b8def,#a855f7)",
            color:"#fff",fontSize:15,fontWeight:600,cursor:"pointer",
            boxShadow:"0 4px 20px rgba(91,141,239,0.3)",transition:"all 0.25s ease",
          }}>Get Started</button>
          <a href="https://github.com/NP-compete/arcana" target="_blank" rel="noreferrer" style={{
            padding:"14px 32px",borderRadius:12,
            border:"1px solid rgba(255,255,255,0.1)",background:"rgba(255,255,255,0.03)",
            color:"#c5cdd8",fontSize:15,fontWeight:600,textDecoration:"none",
            display:"inline-flex",alignItems:"center",gap:8,transition:"all 0.25s ease",
          }}>
            <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor"><path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z"/></svg>
            View on GitHub
          </a>
        </div>

        {/* Tech stack bar */}
        <div style={{ display:"flex",alignItems:"center",justifyContent:"center",gap:24,flexWrap:"wrap" }}>
          <span style={{ fontSize:11,color:"#3a4252",fontWeight:500 }}>BUILT ON</span>
          {LOGOS.map(l => (
            <span key={l.name} style={{
              fontSize:11,color:"#4a5568",fontWeight:500,
              display:"flex",alignItems:"center",gap:6,
            }}>
              <span style={{ fontSize:14 }}>{l.abbr}</span> {l.name}
            </span>
          ))}
        </div>
      </section>

      {/* ===== VALUE PROPS ===== */}
      <section style={{
        display:"grid", gridTemplateColumns:"repeat(4,1fr)", gap:16,
        maxWidth:960, margin:"0 auto", padding:"0 48px 80px",
        position:"relative", zIndex:1,
      }}>
        {VALUE_PROPS.map((v,i) => (
          <div key={v.title} style={{
            padding:"24px 20px",borderRadius:16,
            background:"rgba(255,255,255,0.02)",border:"1px solid rgba(255,255,255,0.05)",
            opacity:mounted?1:0, transform:mounted?"translateY(0)":"translateY(15px)",
            transition:`all 0.6s cubic-bezier(0.16,1,0.3,1) ${0.4+i*0.1}s`,
          }}>
            <div style={{ fontSize:28,marginBottom:12 }}>{v.icon}</div>
            <div style={{ fontSize:14,fontWeight:600,color:"#e2e8f0",marginBottom:6 }}>{v.title}</div>
            <div style={{ fontSize:12,color:"#5a6272",lineHeight:1.6 }}>{v.desc}</div>
          </div>
        ))}
      </section>

      {/* ===== NUMBERS BAR ===== */}
      <section style={{
        display:"flex",justifyContent:"center",gap:48,padding:"40px 48px",
        borderTop:"1px solid rgba(255,255,255,0.04)",borderBottom:"1px solid rgba(255,255,255,0.04)",
        position:"relative",zIndex:1,
      }}>
        {[
          { n:"30+",l:"Microservices" },{ n:"16",l:"Custom CRDs" },{ n:"8",l:"Architecture Planes" },
          { n:"5",l:"Agent Protocols" },{ n:"28",l:"Helm Charts" },{ n:"99.95%",l:"Availability Target" },
        ].map(s => (
          <div key={s.l} style={{ textAlign:"center" }}>
            <div style={{
              fontSize:24,fontWeight:800,
              background:"linear-gradient(135deg,#5b8def,#a855f7)",
              WebkitBackgroundClip:"text",WebkitTextFillColor:"transparent",
            }}>{s.n}</div>
            <div style={{ fontSize:11,color:"#4a5568",marginTop:2,fontWeight:500 }}>{s.l}</div>
          </div>
        ))}
      </section>

      {/* ===== LOGIN PANEL ===== */}
      <section id="login-panel" style={{
        maxWidth:720, margin:"0 auto", padding:"80px 48px",
        position:"relative", zIndex:1,
      }}>
        {/* Section header */}
        <div style={{ textAlign:"center",marginBottom:40 }}>
          <h2 style={{ fontSize:32,fontWeight:800,color:"#fff",margin:"0 0 10px",letterSpacing:-0.5 }}>
            Get into the platform
          </h2>
          <p style={{ fontSize:15,color:"#5a6a7d",margin:0,maxWidth:420,marginLeft:"auto",marginRight:"auto" }}>
            Sign in with SSO, use an API key, or explore with a demo persona
          </p>
        </div>

        {displayError && <Alert variant="danger" isInline title={displayError} style={{ marginBottom:20,maxWidth:720,margin:"0 auto 20px" }}/>}

        {/* SSO — one-click, no key needed */}
        <div style={{
          background:"rgba(255,255,255,0.025)",borderRadius:20,
          border:"1px solid rgba(255,255,255,0.06)",padding:"24px 32px",
          backdropFilter:"blur(24px)",marginBottom:16,
          boxShadow:"0 8px 32px rgba(0,0,0,0.3),inset 0 1px 0 rgba(255,255,255,0.04)",
        }}>
          <button type="button" disabled={loading} className="arcana-cta" onClick={() => {
            window.location.href = "/auth/login";
          }} style={{
            width:"100%",padding:"15px 0",borderRadius:12,border:"none",
            background:"linear-gradient(135deg,#5b8def,#a855f7)",
            color:"#fff",fontSize:15,fontWeight:600,cursor:loading?"wait":"pointer",
            display:"flex",alignItems:"center",justifyContent:"center",gap:10,
            boxShadow:"0 4px 20px rgba(91,141,239,0.3)",transition:"all 0.25s ease",
            opacity:loading?0.7:1,
          }}>
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
            Continue with SSO
          </button>
          <p style={{ fontSize:11,color:"#3a4252",textAlign:"center",marginTop:8,marginBottom:0 }}>
            Redirects to your organization's identity provider (OIDC/SAML)
          </p>

          {/* API key expandable */}
          <div style={{ marginTop:16 }}>
            <button type="button" onClick={() => setMode(mode === "key" ? "roles" : "key")} style={{
              width:"100%",padding:"12px 0",borderRadius:10,
              border:"1px solid rgba(255,255,255,0.08)",
              background:"rgba(255,255,255,0.02)",
              color:"#5a6a7d",fontSize:13,fontWeight:500,cursor:"pointer",
              display:"flex",alignItems:"center",justifyContent:"center",gap:8,
              transition:"all 0.2s ease",
            }}>
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4"/></svg>
              {mode === "key" ? "Hide API key" : "Sign in with API key instead"}
              <svg width="12" height="12" viewBox="0 0 16 16" fill="none" style={{ transform:mode==="key"?"rotate(180deg)":"none",transition:"transform 0.2s" }}>
                <path d="M4 6L8 10L12 6" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round"/>
              </svg>
            </button>

            {mode === "key" && (
              <form onSubmit={handleApiKeySubmit} style={{ marginTop:16 }}>
                <div style={{ display:"flex",gap:10 }}>
                  <input id="arcana-api-key" className="arcana-api-input" type="password" value={apiKey}
                    onChange={e => setApiKey(e.target.value)} placeholder="ak-xxxx-xxxxxxxxxxxxxxxx" autoFocus
                    style={{
                      flex:1,padding:"12px 16px",borderRadius:10,
                      border:"1px solid rgba(255,255,255,0.08)",background:"rgba(0,0,0,0.25)",
                      color:"#fff",fontSize:13,fontFamily:"var(--pf-t--global--font--family--mono)",
                      outline:"none",boxSizing:"border-box",transition:"all 0.25s ease",
                    }}/>
                  <button type="submit" disabled={loading} style={{
                    padding:"12px 24px",borderRadius:10,border:"none",
                    background:"rgba(91,141,239,0.15)",color:"#7ba4f0",
                    fontSize:13,fontWeight:600,cursor:loading?"wait":"pointer",
                    transition:"all 0.2s ease",whiteSpace:"nowrap",
                  }}>{loading?<Spinner size="sm"/>:"Sign In"}</button>
                </div>
              </form>
            )}
          </div>
        </div>

        {/* Divider */}
        <div style={{ display:"flex",alignItems:"center",gap:16,margin:"0 0 16px" }}>
          <div style={{ flex:1,height:1,background:"rgba(255,255,255,0.06)" }}/>
          <span style={{ fontSize:11,color:"#3a4252",fontWeight:600,letterSpacing:1,textTransform:"uppercase" }}>
            or explore as
          </span>
          <div style={{ flex:1,height:1,background:"rgba(255,255,255,0.06)" }}/>
        </div>

        {/* Role cards */}
        <div style={{ display:"grid",gridTemplateColumns:"repeat(3,1fr)",gap:12 }}>
          {ROLE_OPTIONS.map((opt,idx) => {
            const isActive = selectedRole===opt.role;
            const isHovered = hoveredRole===opt.role;
            return (
              <button key={opt.role} type="button" disabled={loading}
                className="arcana-login-role"
                onClick={() => handleRoleLogin(opt.role)}
                onMouseEnter={() => setHoveredRole(opt.role)}
                onMouseLeave={() => setHoveredRole(null)}
                aria-label={`Sign in as ${opt.title}`}
                style={{
                  display:"flex",flexDirection:"column",alignItems:"center",
                  padding:"24px 16px 20px",borderRadius:16,
                  border:`1px solid ${isActive||isHovered?opt.color+"40":"rgba(255,255,255,0.05)"}`,
                  background:isActive?`${opt.color}10`:isHovered?"rgba(255,255,255,0.035)":"rgba(255,255,255,0.02)",
                  cursor:loading?"wait":"pointer",textAlign:"center",
                  transition:"all 0.25s cubic-bezier(0.16,1,0.3,1)",
                  opacity:mounted?(loading&&!isActive?0.35:1):0,
                  transform:mounted?"translateY(0)":"translateY(15px)",
                  transitionDelay:`${0.1+idx*0.05}s`,
                  backdropFilter:"blur(8px)",
                }}
              >
                {/* Avatar */}
                <div style={{
                  width:48,height:48,borderRadius:14,
                  background:`linear-gradient(135deg, ${opt.color}25, ${opt.color}10)`,
                  border:`2px solid ${isHovered?opt.color+"60":opt.color+"25"}`,
                  display:"flex",alignItems:"center",justifyContent:"center",
                  fontSize:18,fontWeight:800,color:opt.color,
                  marginBottom:12,transition:"all 0.25s ease",
                  boxShadow:isHovered?`0 0 20px ${opt.color}20`:"none",
                }}>
                  {isActive&&loading?<Spinner size="md"/>:opt.icon}
                </div>

                {/* Name + desc */}
                <div style={{ color:"#e2e8f0",fontSize:14,fontWeight:600,marginBottom:4 }}>
                  {opt.title}
                </div>
                <div style={{ color:"#4a5568",fontSize:11,marginBottom:12,lineHeight:1.4 }}>
                  {opt.description}
                </div>

                {/* Capability tags */}
                <div style={{ display:"flex",flexWrap:"wrap",gap:4,justifyContent:"center" }}>
                  {opt.capabilities.slice(0,2).map(cap => (
                    <span key={cap} style={{
                      fontSize:9,fontWeight:500,
                      color:isHovered?opt.color:"#4a5568",
                      background:isHovered?`${opt.color}0c`:"rgba(255,255,255,0.03)",
                      padding:"3px 8px",borderRadius:6,
                      border:`1px solid ${isHovered?opt.color+"18":"rgba(255,255,255,0.04)"}`,
                      transition:"all 0.25s ease",
                    }}>{cap}</span>
                  ))}
                  {opt.capabilities.length>2 && (
                    <span style={{ fontSize:9,color:opt.color,padding:"3px 4px",fontWeight:600 }}>+{opt.capabilities.length-2}</span>
                  )}
                </div>
              </button>
            );
          })}
        </div>
      </section>

      {/* ===== FOOTER ===== */}
      <footer style={{
        textAlign:"center",padding:"40px 48px",
        borderTop:"1px solid rgba(255,255,255,0.04)",
        position:"relative",zIndex:1,
      }}>
        <div style={{ display:"flex",justifyContent:"center",gap:24,marginBottom:16 }}>
          {["Documentation","GitHub","Changelog","Community","Contributing"].map(l => (
            <a key={l} href="#" style={{ fontSize:12,color:"#3a4252",textDecoration:"none",fontWeight:500 }}>{l}</a>
          ))}
        </div>
        <p style={{ fontSize:11,color:"#2a3242",margin:0 }}>
          Arcana Platform v0.1.0 &middot; Open Source (Apache 2.0) &middot; Kubernetes-native AI Agent Operating System
        </p>
      </footer>
    </div>
  );
};
