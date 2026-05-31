"""Arcana SDK client — single entry point for all platform APIs."""

from __future__ import annotations

import os

import httpx

from arcana.agents import AgentAPI
from arcana.skills import SkillAPI
from arcana.models import ModelAPI
from arcana.eval import EvalAPI
from arcana.cost import CostAPI


class Client:
    """Top-level client for the Arcana AI Agent Platform.

    Usage::

        from arcana import Client

        client = Client()                         # reads ARCANA_API_URL / ARCANA_API_KEY
        client = Client("https://arcana.example.com", api_key="sk-...")

        agents = client.agents.list()
        client.agents.deploy("my-agent")
    """

    def __init__(
        self,
        base_url: str | None = None,
        api_key: str | None = None,
        timeout: float = 30.0,
    ) -> None:
        self.base_url = (base_url or os.getenv("ARCANA_API_URL", "http://localhost:8080")).rstrip("/")
        self.api_key = api_key or os.getenv("ARCANA_API_KEY", "")

        headers: dict[str, str] = {}
        if self.api_key:
            headers["Authorization"] = f"Bearer {self.api_key}"

        self._http = httpx.Client(
            base_url=self.base_url,
            headers=headers,
            timeout=timeout,
        )

        self.agents = AgentAPI(self._http)
        self.skills = SkillAPI(self._http)
        self.models = ModelAPI(self._http)
        self.eval = EvalAPI(self._http)
        self.cost = CostAPI(self._http)

    # -- convenience helpers --------------------------------------------------

    def health(self) -> dict:
        """Return platform health status."""
        return self._http.get("/api/v1/health").raise_for_status().json()

    def version(self) -> dict:
        """Return platform version information."""
        return self._http.get("/api/v1/version").raise_for_status().json()

    def close(self) -> None:
        """Close the underlying HTTP connection pool."""
        self._http.close()

    def __enter__(self) -> Client:
        return self

    def __exit__(self, *args: object) -> None:
        self.close()
