import { useState, useEffect } from "react";
import { Alert, Spinner } from "@patternfly/react-core";
import { useAuth } from "../auth/AuthContext";

interface RoleOption {
  role: string;
  title: string;
  description: string;
  color: string;
  icon: string;
}

const ROLE_OPTIONS: RoleOption[] = [
  {
    role: "admin",
    title: "Admin",
    description: "Full platform access",
    color: "#a855f7",
    icon: "A",
  },
  {
    role: "developer",
    title: "Developer",
    description: "Build and deploy agents",
    color: "#5b8def",
    icon: "D",
  },
  {
    role: "data-engineer",
    title: "Data Engineer",
    description: "Manage pipelines & connectors",
    color: "#06b6d4",
    icon: "DE",
  },
  {
    role: "sre",
    title: "SRE",
    description: "Operate and monitor",
    color: "#f59e0b",
    icon: "S",
  },
  {
    role: "auditor",
    title: "Auditor",
    description: "Compliance and audit",
    color: "#ef4444",
    icon: "Au",
  },
  {
    role: "user",
    title: "User",
    description: "Use agents day-to-day",
    color: "#22c55e",
    icon: "U",
  },
];

export const LoginPage = () => {
  const { login, loginAs, loading, error } = useAuth();
  const [apiKey, setApiKey] = useState("");
  const [localError, setLocalError] = useState<string | null>(null);
  const [showApiKey, setShowApiKey] = useState(false);
  const [ssoAvailable, setSsoAvailable] = useState<boolean | null>(null);
  const [selectedRole, setSelectedRole] = useState<string | null>(null);
  const [hoveredRole, setHoveredRole] = useState<string | null>(null);
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    const t = setTimeout(() => setMounted(true), 50);
    return () => clearTimeout(t);
  }, []);

  useEffect(() => {
    fetch("/api/v1/enterprise/config")
      .then((r) => r.ok ? r.json() : null)
      .then((data) => {
        const ssoEnabled = data?.auth_mode === "oidc" || data?.auth_mode === "saml" || data?.sso_enabled === true;
        setSsoAvailable(ssoEnabled);
      })
      .catch(() => setSsoAvailable(false));
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
    if (!ok) setLocalError("Could not connect");
    setSelectedRole(null);
  };

  const displayError = localError || error;

  return (
    <div style={{ minHeight: "100vh", background: "#030508", overflow: "hidden" }}>
      <style>{`
        @keyframes arcana-float { 0%,100%{transform:translateY(0)} 50%{transform:translateY(-20px)} }
        .arcana-login-role:hover { transform:translateY(-3px)!important; box-shadow:0 8px 24px rgba(0,0,0,0.4)!important }
        .arcana-login-role:active { transform:translateY(0)!important }
        .arcana-api-input:focus { border-color:rgba(91,141,239,0.6)!important; box-shadow:0 0 0 3px rgba(91,141,239,0.1)!important }
        .arcana-cta:hover { transform:translateY(-1px); box-shadow:0 6px 24px rgba(91,141,239,0.4)!important }
      `}</style>

      {/* Subtle orbs */}
      <div style={{ position:"fixed",width:500,height:500,borderRadius:"50%",background:"radial-gradient(circle,rgba(91,141,239,0.07) 0%,transparent 70%)",top:-200,left:-100,animation:"arcana-float 8s ease-in-out infinite",pointerEvents:"none" }}/>
      <div style={{ position:"fixed",width:400,height:400,borderRadius:"50%",background:"radial-gradient(circle,rgba(168,85,247,0.05) 0%,transparent 70%)",bottom:-100,right:50,animation:"arcana-float 10s ease-in-out infinite 2s",pointerEvents:"none" }}/>

      {/* Nav */}
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
          }}>A</div>
          <span style={{ fontSize:18,fontWeight:700,color:"#fff",letterSpacing:-0.3 }}>Arcana</span>
        </div>
        <div style={{ display:"flex", alignItems:"center", gap:24 }}>
          <a href="https://github.com/NP-compete/arcana" target="_blank" rel="noreferrer" style={{ fontSize:13,color:"#5a6a7d",textDecoration:"none",fontWeight:500 }}>Docs</a>
          <a href="https://github.com/NP-compete/arcana" target="_blank" rel="noreferrer" style={{ fontSize:13,color:"#5a6a7d",textDecoration:"none",fontWeight:500 }}>GitHub</a>
          <button onClick={() => document.getElementById("login-panel")?.scrollIntoView({ behavior:"smooth" })} style={{
            padding:"8px 20px",borderRadius:8,border:"1px solid rgba(91,141,239,0.3)",
            background:"rgba(91,141,239,0.08)",color:"#7ba4f0",fontSize:13,fontWeight:600,
            cursor:"pointer",transition:"all 0.2s",
          }}>Sign In</button>
        </div>
      </nav>

      {/* Hero */}
      <section style={{
        textAlign:"center", padding:"100px 48px 60px", position:"relative", zIndex:1,
        maxWidth:700, margin:"0 auto",
        opacity:mounted?1:0, transform:mounted?"translateY(0)":"translateY(20px)",
        transition:"all 0.8s cubic-bezier(0.16,1,0.3,1)",
      }}>
        <div style={{
          display:"inline-flex",alignItems:"center",gap:8,
          background:"rgba(34,197,94,0.08)",border:"1px solid rgba(34,197,94,0.15)",
          borderRadius:20,padding:"6px 16px",marginBottom:28,
        }}>
          <span style={{ width:6,height:6,borderRadius:"50%",background:"#22c55e",display:"inline-block" }}/>
          <span style={{ fontSize:12,color:"#4ade80",fontWeight:500 }}>Open Source &middot; Apache 2.0</span>
        </div>

        <h1 style={{
          fontSize:52,fontWeight:800,lineHeight:1.1,margin:"0 0 20px",letterSpacing:-2,
          color:"#fff",
        }}>
          Deploy AI agents<br/>
          <span style={{
            background:"linear-gradient(135deg,#667eea 0%,#a855f7 100%)",
            WebkitBackgroundClip:"text",WebkitTextFillColor:"transparent",
          }}>like pushing code</span>
        </h1>

        <p style={{ fontSize:18,color:"#5a6a7d",lineHeight:1.7,margin:"0 auto 40px",maxWidth:480 }}>
          Name it. Pick a model. Add skills. Hit deploy.
          Your agent is running in seconds, not weeks.
        </p>

        <div style={{ display:"flex",gap:12,justifyContent:"center" }}>
          <button className="arcana-cta" onClick={() => document.getElementById("login-panel")?.scrollIntoView({ behavior:"smooth" })} style={{
            padding:"14px 32px",borderRadius:12,border:"none",
            background:"linear-gradient(135deg,#5b8def,#a855f7)",
            color:"#fff",fontSize:15,fontWeight:600,cursor:"pointer",
            boxShadow:"0 4px 20px rgba(91,141,239,0.3)",transition:"all 0.25s ease",
          }}>Get started free</button>
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
      </section>

      {/* How it works */}
      <section style={{
        display:"grid", gridTemplateColumns:"repeat(3,1fr)", gap:16,
        maxWidth:720, margin:"0 auto", padding:"0 48px 80px",
        position:"relative", zIndex:1,
      }}>
        {[
          { step: "1", title: "Name your agent", desc: "Give it a name and pick a model." },
          { step: "2", title: "Add skills", desc: "Attach search, code-gen, SQL — or build your own." },
          { step: "3", title: "Deploy", desc: "One click. It's running. Guardrails and monitoring included." },
        ].map((v, i) => (
          <div key={v.step} style={{
            padding:"28px 24px",borderRadius:16,
            background:"rgba(255,255,255,0.02)",border:"1px solid rgba(255,255,255,0.05)",
            opacity:mounted?1:0, transform:mounted?"translateY(0)":"translateY(15px)",
            transition:`all 0.6s cubic-bezier(0.16,1,0.3,1) ${0.4+i*0.1}s`,
          }}>
            <div style={{
              width:32,height:32,borderRadius:10,
              background:"rgba(91,141,239,0.1)",
              display:"flex",alignItems:"center",justifyContent:"center",
              fontSize:14,fontWeight:700,color:"#5b8def",marginBottom:12,
            }}>{v.step}</div>
            <div style={{ fontSize:15,fontWeight:600,color:"#e2e8f0",marginBottom:6 }}>{v.title}</div>
            <div style={{ fontSize:13,color:"#5a6272",lineHeight:1.6 }}>{v.desc}</div>
          </div>
        ))}
      </section>

      {/* Login panel */}
      <section id="login-panel" style={{
        maxWidth:640, margin:"0 auto", padding:"60px 48px 80px",
        position:"relative", zIndex:1,
      }}>
        <div style={{ textAlign:"center",marginBottom:36 }}>
          <h2 style={{ fontSize:28,fontWeight:700,color:"#fff",margin:"0 0 8px",letterSpacing:-0.5 }}>
            Sign in
          </h2>
          <p style={{ fontSize:14,color:"#5a6a7d",margin:0 }}>
            SSO, API key, or try a demo persona
          </p>
        </div>

        {displayError && <Alert variant="danger" isInline title={displayError} style={{ marginBottom:20 }}/>}

        {/* SSO + API key */}
        <div style={{
          background:"rgba(255,255,255,0.025)",borderRadius:16,
          border:"1px solid rgba(255,255,255,0.06)",padding:"20px 28px",
          marginBottom:16,
        }}>
          {ssoAvailable ? (
            <button type="button" disabled={loading} className="arcana-cta" onClick={() => {
              window.location.href = "/auth/login";
            }} style={{
              width:"100%",padding:"14px 0",borderRadius:10,border:"none",
              background:"linear-gradient(135deg,#5b8def,#a855f7)",
              color:"#fff",fontSize:15,fontWeight:600,cursor:loading?"wait":"pointer",
              display:"flex",alignItems:"center",justifyContent:"center",gap:10,
              boxShadow:"0 4px 20px rgba(91,141,239,0.3)",transition:"all 0.25s ease",
              opacity:loading?0.7:1,
            }}>
              Continue with SSO
            </button>
          ) : (
            <div style={{
              width:"100%",padding:"14px 0",borderRadius:10,
              border:"1px solid rgba(255,255,255,0.06)",
              background:"rgba(255,255,255,0.02)",
              display:"flex",alignItems:"center",justifyContent:"center",gap:8,
              color:"#4a5568",fontSize:14,fontWeight:500,
            }}>
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
              SSO not configured &mdash; use API key or a demo persona below
            </div>
          )}

          {/* API key toggle */}
          <div style={{ marginTop:12 }}>
            <button type="button" onClick={() => setShowApiKey(!showApiKey)} style={{
              width:"100%",padding:"10px 0",borderRadius:8,
              border:"1px solid rgba(255,255,255,0.06)",
              background:"transparent",
              color:"#5a6a7d",fontSize:13,fontWeight:500,cursor:"pointer",
              display:"flex",alignItems:"center",justifyContent:"center",gap:6,
            }}>
              {showApiKey ? "Hide API key" : "Use API key"}
            </button>
            {showApiKey && (
              <form onSubmit={handleApiKeySubmit} style={{ marginTop:12 }}>
                <div style={{ display:"flex",gap:8 }}>
                  <input className="arcana-api-input" type="password" value={apiKey}
                    onChange={e => setApiKey(e.target.value)} placeholder="ak-xxxx-xxxxxxxx" autoFocus
                    style={{
                      flex:1,padding:"10px 14px",borderRadius:8,
                      border:"1px solid rgba(255,255,255,0.08)",background:"rgba(0,0,0,0.25)",
                      color:"#fff",fontSize:13,fontFamily:"monospace",
                      outline:"none",boxSizing:"border-box",transition:"all 0.2s",
                    }}/>
                  <button type="submit" disabled={loading} style={{
                    padding:"10px 20px",borderRadius:8,border:"none",
                    background:"rgba(91,141,239,0.15)",color:"#7ba4f0",
                    fontSize:13,fontWeight:600,cursor:loading?"wait":"pointer",
                  }}>{loading?<Spinner size="sm"/>:"Sign In"}</button>
                </div>
              </form>
            )}
          </div>
        </div>

        {/* Divider */}
        <div style={{ display:"flex",alignItems:"center",gap:16,margin:"0 0 16px" }}>
          <div style={{ flex:1,height:1,background:"rgba(255,255,255,0.06)" }}/>
          <span style={{ fontSize:10,color:"#3a4252",fontWeight:600,letterSpacing:1,textTransform:"uppercase" }}>
            or try as
          </span>
          <div style={{ flex:1,height:1,background:"rgba(255,255,255,0.06)" }}/>
        </div>

        {/* Role cards — compact grid */}
        <div style={{ display:"grid",gridTemplateColumns:"repeat(3,1fr)",gap:10 }}>
          {ROLE_OPTIONS.map((opt,idx) => {
            const isActive = selectedRole===opt.role;
            const isHovered = hoveredRole===opt.role;
            return (
              <button key={opt.role} type="button" disabled={loading}
                className="arcana-login-role"
                onClick={() => handleRoleLogin(opt.role)}
                onMouseEnter={() => setHoveredRole(opt.role)}
                onMouseLeave={() => setHoveredRole(null)}
                style={{
                  display:"flex",flexDirection:"column",alignItems:"center",
                  padding:"20px 12px 16px",borderRadius:12,
                  border:`1px solid ${isActive||isHovered?opt.color+"40":"rgba(255,255,255,0.05)"}`,
                  background:isActive?`${opt.color}10`:"rgba(255,255,255,0.02)",
                  cursor:loading?"wait":"pointer",textAlign:"center",
                  transition:"all 0.2s",
                  opacity:mounted?(loading&&!isActive?0.35:1):0,
                  transform:mounted?"translateY(0)":"translateY(10px)",
                  transitionDelay:`${0.1+idx*0.04}s`,
                }}
              >
                <div style={{
                  width:40,height:40,borderRadius:12,
                  background:`${opt.color}15`,
                  border:`2px solid ${isHovered?opt.color+"50":opt.color+"20"}`,
                  display:"flex",alignItems:"center",justifyContent:"center",
                  fontSize:16,fontWeight:800,color:opt.color,
                  marginBottom:10,transition:"all 0.2s",
                }}>
                  {isActive&&loading?<Spinner size="sm"/>:opt.icon}
                </div>
                <div style={{ color:"#e2e8f0",fontSize:13,fontWeight:600,marginBottom:2 }}>
                  {opt.title}
                </div>
                <div style={{ color:"#4a5568",fontSize:11 }}>
                  {opt.description}
                </div>
              </button>
            );
          })}
        </div>
      </section>

      {/* Footer */}
      <footer style={{
        textAlign:"center",padding:"32px 48px",
        borderTop:"1px solid rgba(255,255,255,0.04)",
        position:"relative",zIndex:1,
      }}>
        <p style={{ fontSize:11,color:"#2a3242",margin:0 }}>
          Arcana &middot; Open Source (Apache 2.0) &middot; Deploy AI agents like pushing code
        </p>
      </footer>
    </div>
  );
};
