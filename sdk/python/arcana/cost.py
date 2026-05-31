"""Cost and budget management API."""

from __future__ import annotations

from typing import Any

import httpx


class CostAPI:
    """Query costs and manage budgets on the Arcana platform."""

    def __init__(self, http: httpx.Client) -> None:
        self._http = http

    def report(
        self,
        agent: str | None = None,
        since: str | None = None,
    ) -> dict[str, Any]:
        """Get a cost report.

        Args:
            agent: Filter by agent name.
            since: Start date in ``YYYY-MM-DD`` format.
        """
        params: dict[str, str] = {}
        if agent is not None:
            params["agent"] = agent
        if since is not None:
            params["since"] = since
        return self._http.get("/api/v1/costs", params=params).raise_for_status().json()

    def budget_set(self, agent: str, budget: str) -> dict[str, Any]:
        """Set the daily budget for an agent.

        Args:
            agent: Agent name.
            budget: Budget string (e.g. ``"$20/day"``).
        """
        return self._http.post(
            "/api/v1/budget",
            json={"agent": agent, "budget": budget},
        ).raise_for_status().json()

    def budget_get(self, agent: str) -> dict[str, Any]:
        """Get the budget for an agent."""
        return self._http.get("/api/v1/budget", params={"agent": agent}).raise_for_status().json()
