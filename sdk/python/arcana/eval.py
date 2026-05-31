"""Evaluation API."""

from __future__ import annotations

from typing import Any

import httpx


class EvalAPI:
    """Run evaluations on the Arcana platform."""

    def __init__(self, http: httpx.Client) -> None:
        self._http = http

    def run(
        self,
        skill_ref: str,
        judges: list[str] | None = None,
    ) -> dict[str, Any]:
        """Start an evaluation run for a skill.

        Args:
            skill_ref: Name or reference of the skill to evaluate.
            judges: List of judge types (default: deterministic + llm).
        """
        if judges is None:
            judges = ["deterministic", "llm"]
        return self._http.post(
            "/api/v1/eval/run",
            json={"skill_ref": skill_ref, "judges": judges},
        ).raise_for_status().json()

    def report(self, run_id: str | None = None) -> dict[str, Any]:
        """Get evaluation report.

        Args:
            run_id: Specific run ID to retrieve. If ``None``, returns the latest.
        """
        params: dict[str, str] = {}
        if run_id is not None:
            params["run_id"] = run_id
        return self._http.get("/api/v1/eval/report", params=params).raise_for_status().json()
