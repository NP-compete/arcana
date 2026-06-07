-- Arcana Platform Schema
-- Applied on first boot via postgres init ConfigMap or CNPG postInitApplicationSQL
-- All tables use IF NOT EXISTS for idempotent re-runs

-- Extensions
CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Additional databases
SELECT 'CREATE DATABASE arcana_skills OWNER arcana'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'arcana_skills')\gexec
SELECT 'CREATE DATABASE arcana_temporal OWNER arcana'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'arcana_temporal')\gexec
SELECT 'CREATE DATABASE arcana_temporal_visibility OWNER arcana'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'arcana_temporal_visibility')\gexec

-- ============================================================
-- API Gateway (arcana-api)
-- ============================================================

CREATE TABLE IF NOT EXISTS api_keys (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    key_hash VARCHAR(128) UNIQUE NOT NULL,
    prefix VARCHAR(32) NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    tenant VARCHAR(255) NOT NULL DEFAULT 'default',
    roles TEXT[] NOT NULL DEFAULT '{}',
    scopes TEXT[] NOT NULL DEFAULT '{}',
    rate_limit_ps INTEGER NOT NULL DEFAULT 100,
    revoked BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS tenants (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    namespace VARCHAR(255),
    resource_quota JSONB DEFAULT '{}',
    budget_limit DOUBLE PRECISION DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS audit_log (
    id BIGSERIAL PRIMARY KEY,
    actor VARCHAR(255) NOT NULL,
    tenant VARCHAR(255) NOT NULL DEFAULT 'default',
    action VARCHAR(255) NOT NULL,
    resource VARCHAR(512),
    detail TEXT,
    ip VARCHAR(64),
    entry_hash VARCHAR(128) NOT NULL,
    prev_hash VARCHAR(128) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_audit_actor ON audit_log(actor);
CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_log(action);
CREATE INDEX IF NOT EXISTS idx_audit_tenant ON audit_log(tenant);
CREATE INDEX IF NOT EXISTS idx_audit_created ON audit_log(created_at);

CREATE TABLE IF NOT EXISTS asset_sharing (
    asset_type VARCHAR(64) NOT NULL,
    asset_name VARCHAR(255) NOT NULL,
    owner VARCHAR(255) NOT NULL,
    tenant VARCHAR(255) NOT NULL DEFAULT 'default',
    visibility VARCHAR(32) NOT NULL DEFAULT 'private',
    shared_with TEXT[] DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (asset_type, asset_name)
);

-- ============================================================
-- Mesh (arcana-mesh)
-- ============================================================

CREATE TABLE IF NOT EXISTS agents (
    name VARCHAR(255) NOT NULL,
    tenant VARCHAR(255) NOT NULL DEFAULT 'default',
    agent_type VARCHAR(64) NOT NULL DEFAULT 'create_agent',
    capabilities TEXT[] DEFAULT '{}',
    protocols TEXT[] DEFAULT '{}',
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    deep_config JSONB,
    registered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant, name)
);

CREATE TABLE IF NOT EXISTS messages (
    id VARCHAR(64) PRIMARY KEY DEFAULT uuid_generate_v4()::text,
    tenant VARCHAR(255) NOT NULL DEFAULT 'default',
    from_agent VARCHAR(255) NOT NULL,
    to_agent VARCHAR(255) NOT NULL,
    payload JSONB DEFAULT '{}',
    protocol VARCHAR(64),
    delivered BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_messages_to ON messages(to_agent, delivered);
CREATE INDEX IF NOT EXISTS idx_messages_tenant ON messages(tenant);

CREATE TABLE IF NOT EXISTS delegations (
    id VARCHAR(64) PRIMARY KEY DEFAULT uuid_generate_v4()::text,
    tenant VARCHAR(255) NOT NULL DEFAULT 'default',
    from_agent VARCHAR(255) NOT NULL,
    to_agent VARCHAR(255) NOT NULL,
    task_type VARCHAR(128),
    payload JSONB DEFAULT '{}',
    result JSONB,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_delegations_tenant ON delegations(tenant);

-- Agent health events (health monitor writes state changes here)
CREATE TABLE IF NOT EXISTS agent_health_events (
    id BIGSERIAL PRIMARY KEY,
    tenant VARCHAR(255) NOT NULL DEFAULT 'default',
    agent_name VARCHAR(255) NOT NULL,
    event_type VARCHAR(32) NOT NULL,
    restart_count INTEGER DEFAULT 0,
    ready_replicas INTEGER DEFAULT 0,
    desired_replicas INTEGER DEFAULT 0,
    failure_reason TEXT DEFAULT '',
    pod_phase VARCHAR(32) DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_health_agent ON agent_health_events(tenant, agent_name);
CREATE INDEX IF NOT EXISTS idx_health_created ON agent_health_events(created_at);

-- Cached health columns on agents (fast reads without joining events)
ALTER TABLE agents ADD COLUMN IF NOT EXISTS restart_count INTEGER DEFAULT 0;
ALTER TABLE agents ADD COLUMN IF NOT EXISTS last_healthy_at TIMESTAMPTZ;
ALTER TABLE agents ADD COLUMN IF NOT EXISTS last_failure_at TIMESTAMPTZ;
ALTER TABLE agents ADD COLUMN IF NOT EXISTS last_failure_reason TEXT DEFAULT '';
ALTER TABLE agents ADD COLUMN IF NOT EXISTS pod_phase VARCHAR(32) DEFAULT '';

-- ============================================================
-- Engine (arcana-engine)
-- ============================================================

CREATE TABLE IF NOT EXISTS agent_tasks (
    id VARCHAR(64) PRIMARY KEY,
    agent VARCHAR(255) NOT NULL,
    input JSONB DEFAULT '{}',
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    result JSONB,
    model_config JSONB DEFAULT '{}',
    tokens_used INTEGER DEFAULT 0,
    cost_usd DOUBLE PRECISION DEFAULT 0,
    current_step INTEGER DEFAULT 0,
    error TEXT DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_tasks_agent ON agent_tasks(agent);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON agent_tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_created ON agent_tasks(created_at);

-- ============================================================
-- Blueprint (arcana-blueprint)
-- ============================================================

CREATE TABLE IF NOT EXISTS blueprints (
    name VARCHAR(255) PRIMARY KEY,
    description TEXT DEFAULT '',
    nodes JSONB NOT NULL DEFAULT '[]',
    edges JSONB DEFAULT '[]',
    fallback VARCHAR(64) DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- Skills (arcana-skills)
-- ============================================================

CREATE TABLE IF NOT EXISTS skills (
    name VARCHAR(255) PRIMARY KEY,
    type VARCHAR(32) NOT NULL DEFAULT 'functional',
    version VARCHAR(32) DEFAULT '1.0.0',
    description TEXT DEFAULT '',
    skill_md TEXT DEFAULT '',
    quality_badge VARCHAR(32) DEFAULT 'untested',
    source VARCHAR(32) DEFAULT 'manual',
    category VARCHAR(64) DEFAULT 'general',
    metadata JSONB DEFAULT '{}',
    memory JSONB DEFAULT '[]',
    usage_count INTEGER DEFAULT 0,
    rating DOUBLE PRECISION DEFAULT 0,
    status VARCHAR(32) DEFAULT 'active',
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_skills_type ON skills(type);
CREATE INDEX IF NOT EXISTS idx_skills_badge ON skills(quality_badge);

-- Annotations (arcana-annotate) — human corrections that feed skill crystallization
CREATE TABLE IF NOT EXISTS annotations (
    id BIGSERIAL PRIMARY KEY,
    tenant VARCHAR(255) NOT NULL DEFAULT 'default',
    agent_id VARCHAR(255) NOT NULL,
    topic VARCHAR(255) DEFAULT '',
    question TEXT NOT NULL,
    original_answer TEXT,
    corrected_answer TEXT NOT NULL,
    crystallized BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_annotations_agent ON annotations(tenant, agent_id);
CREATE INDEX IF NOT EXISTS idx_annotations_topic ON annotations(tenant, topic);
CREATE INDEX IF NOT EXISTS idx_annotations_uncrystallized ON annotations(tenant, topic, crystallized) WHERE crystallized = FALSE;

-- Eval results (arcana-probe) — skill evaluation history
CREATE TABLE IF NOT EXISTS eval_results (
    id BIGSERIAL PRIMARY KEY,
    skill_name VARCHAR(255) NOT NULL,
    tenant VARCHAR(255) NOT NULL DEFAULT 'default',
    run_id VARCHAR(64) NOT NULL,
    avg_score DOUBLE PRECISION DEFAULT 0,
    pass_rate DOUBLE PRECISION DEFAULT 0,
    badge VARCHAR(32) DEFAULT 'untested',
    test_count INTEGER DEFAULT 0,
    judge_scores JSONB DEFAULT '{}',
    regression BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_eval_skill ON eval_results(skill_name);
CREATE INDEX IF NOT EXISTS idx_eval_run ON eval_results(run_id);

-- ============================================================
-- Registry (arcana-registry)
-- ============================================================

CREATE TABLE IF NOT EXISTS catalog_entries (
    id VARCHAR(64) PRIMARY KEY,
    type VARCHAR(32) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT DEFAULT '',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(type, name)
);

CREATE TABLE IF NOT EXISTS package_versions (
    id VARCHAR(64) PRIMARY KEY,
    package_type VARCHAR(32) NOT NULL,
    package_name VARCHAR(255) NOT NULL,
    version VARCHAR(64) NOT NULL,
    author VARCHAR(255) DEFAULT '',
    digest VARCHAR(128) DEFAULT '',
    status VARCHAR(32) DEFAULT 'published',
    notes TEXT DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_pkg_versions ON package_versions(package_type, package_name);

CREATE TABLE IF NOT EXISTS approval_requests (
    id VARCHAR(64) PRIMARY KEY,
    package_type VARCHAR(32) NOT NULL,
    package_name VARCHAR(255) NOT NULL,
    version VARCHAR(64) NOT NULL,
    author VARCHAR(255) DEFAULT '',
    status VARCHAR(32) DEFAULT 'pending',
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_at TIMESTAMPTZ,
    reviewed_by VARCHAR(255) DEFAULT '',
    comment TEXT DEFAULT '',
    diff TEXT DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_approvals_status ON approval_requests(status);

-- ============================================================
-- FinOps (arcana-finops)
-- ============================================================

CREATE TABLE IF NOT EXISTS cost_events (
    id BIGSERIAL PRIMARY KEY,
    agent VARCHAR(255) NOT NULL,
    team VARCHAR(255) DEFAULT 'default',
    model VARCHAR(255) DEFAULT '',
    tokens INTEGER DEFAULT 0,
    cost_usd DOUBLE PRECISION DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_cost_agent ON cost_events(agent);
CREATE INDEX IF NOT EXISTS idx_cost_team ON cost_events(team);
CREATE INDEX IF NOT EXISTS idx_cost_created ON cost_events(created_at);

CREATE TABLE IF NOT EXISTS budgets (
    id VARCHAR(64) PRIMARY KEY,
    team VARCHAR(255) NOT NULL UNIQUE,
    daily_usd DOUBLE PRECISION DEFAULT 0,
    monthly_usd DOUBLE PRECISION DEFAULT 0,
    per_agent_daily_usd DOUBLE PRECISION DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- Memory (arcana-memory)
-- ============================================================

CREATE TABLE IF NOT EXISTS agent_memory (
    id VARCHAR(64) PRIMARY KEY,
    agent_id VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    scope VARCHAR(32) NOT NULL DEFAULT 'long_term',
    type VARCHAR(32) DEFAULT 'fact',
    embedding vector(1536),
    status VARCHAR(32) DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_memory_agent ON agent_memory(agent_id, scope);
CREATE INDEX IF NOT EXISTS idx_memory_status ON agent_memory(agent_id, status);

-- ============================================================
-- Ward (arcana-ward) — guardrail rules
-- ============================================================

CREATE TABLE IF NOT EXISTS guardrail_rules (
    id VARCHAR(64) PRIMARY KEY,
    agent VARCHAR(255) NOT NULL,
    type VARCHAR(64) NOT NULL,
    config JSONB DEFAULT '{}',
    action VARCHAR(32) DEFAULT 'block',
    severity VARCHAR(32) DEFAULT 'medium',
    position INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_guardrail_agent ON guardrail_rules(agent);

-- ============================================================
-- Scheduler (arcana-scheduler)
-- ============================================================

CREATE TABLE IF NOT EXISTS agent_schedules (
    agent_name VARCHAR(255) PRIMARY KEY,
    status VARCHAR(32) DEFAULT 'active',
    snapshot_path TEXT DEFAULT '',
    suspended_at TIMESTAMPTZ,
    resumed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
