-- Reverse of 001_initial_schema.up.sql
-- Drop order respects foreign key dependencies (none currently, but future-proofed)

DROP TABLE IF EXISTS agent_schedules;
DROP TABLE IF EXISTS guardrail_rules;
DROP TABLE IF EXISTS agent_memory;
DROP TABLE IF EXISTS budgets;
DROP TABLE IF EXISTS cost_events;
DROP TABLE IF EXISTS approval_requests;
DROP TABLE IF EXISTS package_versions;
DROP TABLE IF EXISTS catalog_entries;
DROP TABLE IF EXISTS skills;
DROP TABLE IF EXISTS blueprints;
DROP TABLE IF EXISTS agent_tasks;
DROP TABLE IF EXISTS delegations;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS agents;
DROP TABLE IF EXISTS asset_sharing;
DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS tenants;
DROP TABLE IF EXISTS api_keys;
