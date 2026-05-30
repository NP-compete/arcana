import { useState, useEffect, useCallback } from "react";

export interface ServiceHealth {
  name: string;
  status: string;
  latency: string;
  port: number;
  plane: string;
}

export interface SystemHealth {
  platform: string;
  version: string;
  uptime: string;
  services: ServiceHealth[];
}

export function useHealth(intervalMs = 10000) {
  const [health, setHealth] = useState<SystemHealth | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchHealth = useCallback(async () => {
    try {
      const res = await fetch("/api/v1/health");
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data: SystemHealth = await res.json();
      setHealth(data);
      setError(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Unknown error");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchHealth();
    const id = setInterval(fetchHealth, intervalMs);
    return () => clearInterval(id);
  }, [fetchHealth, intervalMs]);

  return { health, error, loading, refresh: fetchHealth };
}
