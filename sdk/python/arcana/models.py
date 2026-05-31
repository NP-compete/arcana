"""Model management API."""

from __future__ import annotations

from typing import Any

import httpx


class ModelAPI:
    """Manage models on the Arcana platform."""

    def __init__(self, http: httpx.Client) -> None:
        self._http = http

    def list(self) -> dict[str, Any]:
        """List all registered models."""
        return self._http.get("/api/v1/models").raise_for_status().json()

    def train(self, experiment_name: str, **kwargs: Any) -> dict[str, Any]:
        """Start a training experiment."""
        payload: dict[str, Any] = {"name": experiment_name, **kwargs}
        return self._http.post("/api/v1/experiments", json=payload).raise_for_status().json()

    def promote(self, model: str, target: str = "production", auto_approve: bool = False) -> dict[str, Any]:
        """Promote a model to a target environment."""
        return self._http.post(
            "/api/v1/promotions",
            json={"model": model, "target": target, "auto_approve": auto_approve},
        ).raise_for_status().json()

    def serve(self, model: str) -> dict[str, Any]:
        """Start serving a model."""
        return self._http.post(
            "/api/v1/models/serve",
            json={"model": model},
        ).raise_for_status().json()
