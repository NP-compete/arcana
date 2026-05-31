"""Agent management API."""

from __future__ import annotations

from typing import Any

import httpx


class AgentAPI:
    """Manage agents on the Arcana platform."""

    def __init__(self, http: httpx.Client) -> None:
        self._http = http

    def list(self) -> dict[str, Any]:
        """List all registered agents."""
        return self._http.get("/api/v1/agents").raise_for_status().json()

    def deploy(self, name: str, agent_type: str = "create_deep_agent", **kwargs: Any) -> dict[str, Any]:
        """Register and deploy a new agent.

        Args:
            name: Agent name (used as identifier).
            agent_type: One of ``create_agent`` or ``create_deep_agent``.
            **kwargs: Additional fields forwarded to the registration payload
                      (e.g. ``model``, ``capabilities``, ``protocols``).
        """
        payload: dict[str, Any] = {"name": name, "agent_type": agent_type, **kwargs}
        return self._http.post("/api/v1/agents/register", json=payload).raise_for_status().json()

    def status(self, name: str) -> dict[str, Any]:
        """Get detailed status of an agent."""
        return self._http.get(f"/api/v1/agents/{name}/detail").raise_for_status().json()

    def suspend(self, name: str) -> dict[str, Any]:
        """Suspend a running agent."""
        return self._http.post(f"/api/v1/agents/suspend/{name}").raise_for_status().json()

    def resume(self, name: str) -> dict[str, Any]:
        """Resume a suspended agent."""
        return self._http.post(f"/api/v1/agents/resume/{name}").raise_for_status().json()

    def delete(self, name: str) -> dict[str, Any]:
        """Delete an agent."""
        resp = self._http.delete(f"/api/v1/agents/{name}").raise_for_status()
        if resp.content:
            return resp.json()
        return {"status": "deleted"}

    def run(self, name: str, message: str) -> dict[str, Any]:
        """Send a chat message to an agent."""
        return self._http.post(
            f"/api/v1/agents/{name}/chat",
            json={"message": message},
        ).raise_for_status().json()
