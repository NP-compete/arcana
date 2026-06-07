import { useState, useEffect, useCallback } from "react";

export interface AgentHealthSummary {
  agent_name: string;
  status: string;
  restart_count: number;
  last_healthy_at?: string;
  last_failure_at?: string;
  last_failure_reason?: string;
  pod_phase: string;
  ready_replicas: number;
  desired_replicas: number;
}

export interface AgentHealthEvent {
  id: number;
  event_type: string;
  restart_count: number;
  ready_replicas: number;
  desired_replicas: number;
  failure_reason: string;
  pod_phase: string;
  created_at: string;
}

export interface AgentHealthData {
  summary: AgentHealthSummary;
  events: AgentHealthEvent[];
}

export interface AgentsHealthOverview {
  total_agents: number;
  healthy_agents: number;
  unhealthy_agents: number;
  total_restarts: number;
}

export function useAgentHealth(agentName: string, intervalMs = 15000) {
  const [health, setHealth] = useState<AgentHealthData | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchHealth = useCallback(async () => {
    try {
      const res = await fetch(`/api/v1/agents/${agentName}/health`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data: AgentHealthData = await res.json();
      setHealth(data);
      setError(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Unknown error");
    } finally {
      setLoading(false);
    }
  }, [agentName]);

  useEffect(() => {
    fetchHealth();
    const id = setInterval(fetchHealth, intervalMs);
    return () => clearInterval(id);
  }, [fetchHealth, intervalMs]);

  return { health, error, loading, refresh: fetchHealth };
}

export function useAgentsHealthOverview(intervalMs = 30000) {
  const [overview, setOverview] = useState<AgentsHealthOverview | null>(null);
  const [error, setError] = useState<string | null>(null);

  const fetchOverview = useCallback(async () => {
    try {
      const res = await fetch("/api/v1/agents/health");
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data: AgentsHealthOverview = await res.json();
      setOverview(data);
      setError(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Unknown error");
    }
  }, []);

  useEffect(() => {
    fetchOverview();
    const id = setInterval(fetchOverview, intervalMs);
    return () => clearInterval(id);
  }, [fetchOverview, intervalMs]);

  return { overview, error, refresh: fetchOverview };
}
