"""Skill management API."""

from __future__ import annotations

from typing import Any

import httpx


class SkillAPI:
    """Manage skills on the Arcana platform."""

    def __init__(self, http: httpx.Client) -> None:
        self._http = http

    def list(self) -> dict[str, Any]:
        """List all registered skills."""
        return self._http.get("/api/v1/skills").raise_for_status().json()

    def create(self, name: str, description: str = "", version: str = "1.0.0") -> dict[str, Any]:
        """Register a new skill in the catalog."""
        payload = {
            "name": name,
            "type": "skills",
            "version": version,
            "description": description,
        }
        return self._http.post("/api/v1/catalog/skills", json=payload).raise_for_status().json()

    def merge(self, source: str, target: str) -> dict[str, Any]:
        """Merge one skill into another."""
        return self._http.post(
            "/api/v1/skills/merge",
            json={"source": source, "target": target},
        ).raise_for_status().json()

    def prune(self) -> dict[str, Any]:
        """Remove unused skills."""
        return self._http.post("/api/v1/skills/prune").raise_for_status().json()

    def transfer(self, skill: str, agent: str) -> dict[str, Any]:
        """Transfer a skill to an agent."""
        return self._http.post(
            "/api/v1/skills/transfer",
            json={"skill": skill, "agent": agent},
        ).raise_for_status().json()
