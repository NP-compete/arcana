from __future__ import annotations

import threading
import time
import uuid
from datetime import UTC, datetime
from enum import StrEnum
from typing import Any

from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel, Field

app = FastAPI(title="Arcana Probe", version="0.1.0", description="Eval framework")
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)


class EvalStatus(StrEnum):
    PENDING = "pending"
    RUNNING = "running"
    COMPLETED = "completed"
    FAILED = "failed"


class BadgeLevel(StrEnum):
    GOLD = "gold"
    SILVER = "silver"
    BRONZE = "bronze"
    UNTESTED = "untested"
    FAILED = "failed"


class JudgeTier(StrEnum):
    DETERMINISTIC = "deterministic"
    SCRIPT = "script"
    LLM = "llm"
    TRAJECTORY = "trajectory"


class EvalCase(BaseModel):
    id: str
    input: str
    expected: str | None = None
    metadata: dict[str, Any] = Field(default_factory=dict)


class Judge(BaseModel):
    name: str
    tier: JudgeTier
    weight: float = 1.0
    config: dict[str, Any] = Field(default_factory=dict)


class EvalResult(BaseModel):
    case_id: str
    passed: bool
    score: float
    judge_scores: dict[str, float] = Field(default_factory=dict)
    detail: str = ""


class Badge(BaseModel):
    level: BadgeLevel
    criteria_met: list[str] = Field(default_factory=list)
    criteria_failed: list[str] = Field(default_factory=list)


class EvalRun(BaseModel):
    run_id: str
    skill_ref: str
    status: EvalStatus
    cases: list[EvalCase]
    judges: list[Judge]
    results: list[EvalResult] = Field(default_factory=list)
    badge: Badge | None = None
    regression: bool = False
    started_at: datetime = Field(default_factory=lambda: datetime.now(UTC))
    completed_at: datetime | None = None


class RunEvalRequest(BaseModel):
    skill_ref: str
    cases: list[EvalCase]
    judges: list[Judge] = Field(default_factory=list)
    settings: dict[str, Any] = Field(default_factory=dict)


class CompareRequest(BaseModel):
    run_ids: list[str]
    baseline_run_id: str | None = None


_store_lock = threading.Lock()
_runs: dict[str, EvalRun] = {}


def _default_judges() -> list[Judge]:
    return [
        Judge(name="exact_match", tier=JudgeTier.DETERMINISTIC, weight=0.3),
        Judge(name="script_validator", tier=JudgeTier.SCRIPT, weight=0.2),
        Judge(name="llm_grader", tier=JudgeTier.LLM, weight=0.3),
        Judge(name="trajectory_analyzer", tier=JudgeTier.TRAJECTORY, weight=0.2),
    ]


def _judge_deterministic(case: EvalCase) -> tuple[float, str]:
    if case.expected is None:
        return 0.5, "no expected output for deterministic check"
    passed = case.input.strip().lower() == case.expected.strip().lower() or case.expected.lower() in case.input.lower()
    return (1.0 if passed else 0.0), "exact/substring match"


def _judge_script(case: EvalCase) -> tuple[float, str]:
    score = 1.0 if len(case.input) > 0 and not case.input.startswith("ERROR") else 0.0
    return score, "script validation"


def _judge_llm(case: EvalCase) -> tuple[float, str]:
    score = min(1.0, len(case.input) / 100.0) if case.input else 0.0
    return score, "simulated LLM grading"


def _judge_trajectory(case: EvalCase) -> tuple[float, str]:
    steps = case.metadata.get("steps", 1)
    score = 1.0 if isinstance(steps, int) and steps <= 5 else 0.6
    return score, "trajectory step count check"


def _run_judge(judge: Judge, case: EvalCase) -> tuple[float, str]:
    if judge.tier == JudgeTier.DETERMINISTIC:
        return _judge_deterministic(case)
    if judge.tier == JudgeTier.SCRIPT:
        return _judge_script(case)
    if judge.tier == JudgeTier.LLM:
        return _judge_llm(case)
    return _judge_trajectory(case)


def _compute_badge(avg_score: float, results: list[EvalResult]) -> Badge:
    pass_rate = sum(1 for r in results if r.passed) / max(len(results), 1)
    criteria_met: list[str] = []
    criteria_failed: list[str] = []

    if avg_score >= 0.9:
        criteria_met.append("avg_score >= 0.9")
    else:
        criteria_failed.append("avg_score >= 0.9")

    if pass_rate >= 0.95:
        criteria_met.append("pass_rate >= 0.95")
    else:
        criteria_failed.append("pass_rate >= 0.95")

    if pass_rate >= 0.8:
        criteria_met.append("pass_rate >= 0.8")
    else:
        criteria_failed.append("pass_rate >= 0.8")

    if avg_score >= 0.9 and pass_rate >= 0.95:
        level = BadgeLevel.GOLD
    elif avg_score >= 0.75 and pass_rate >= 0.8:
        level = BadgeLevel.SILVER
    elif avg_score >= 0.5:
        level = BadgeLevel.BRONZE
    elif len(results) == 0:
        level = BadgeLevel.UNTESTED
    else:
        level = BadgeLevel.FAILED

    return Badge(level=level, criteria_met=criteria_met, criteria_failed=criteria_failed)


def _execute_run(run_id: str) -> None:
    time.sleep(0.05)
    with _store_lock:
        run = _runs.get(run_id)
        if not run:
            return
        run.status = EvalStatus.RUNNING

    judges = run.judges if run.judges else _default_judges()
    results: list[EvalResult] = []

    for case in run.cases:
        judge_scores: dict[str, float] = {}
        weighted_sum = 0.0
        total_weight = 0.0
        for judge in judges:
            score, detail = _run_judge(judge, case)
            judge_scores[judge.name] = score
            weighted_sum += score * judge.weight
            total_weight += judge.weight
        avg = weighted_sum / max(total_weight, 0.001)
        passed = avg >= 0.7
        results.append(EvalResult(case_id=case.id, passed=passed, score=avg, judge_scores=judge_scores, detail=detail))

    avg_score = sum(r.score for r in results) / max(len(results), 1)
    badge = _compute_badge(avg_score, results)

    with _store_lock:
        run = _runs.get(run_id)
        if run:
            run.results = results
            run.badge = badge
            run.status = EvalStatus.COMPLETED
            run.completed_at = datetime.now(UTC)


@app.get("/readyz")
async def readyz():
    return {"status": "ok"}

@app.get("/healthz")
async def healthz() -> dict[str, str]:
    return {"status": "ok"}


@app.post("/api/v1/eval/run", response_model=EvalRun, status_code=201)
async def trigger_eval(req: RunEvalRequest) -> EvalRun:
    run_id = str(uuid.uuid4())
    judges = req.judges if req.judges else _default_judges()
    run = EvalRun(
        run_id=run_id,
        skill_ref=req.skill_ref,
        status=EvalStatus.PENDING,
        cases=req.cases,
        judges=judges,
    )
    with _store_lock:
        _runs[run_id] = run
    threading.Thread(target=_execute_run, args=(run_id,), daemon=True).start()
    return run


@app.get("/api/v1/eval/runs")
async def list_runs() -> dict[str, Any]:
    with _store_lock:
        runs = list(_runs.values())
    return {"runs": runs, "total": len(runs)}


@app.get("/api/v1/eval/runs/{run_id}", response_model=EvalRun)
async def get_run(run_id: str) -> EvalRun:
    with _store_lock:
        run = _runs.get(run_id)
    if not run:
        raise HTTPException(status_code=404, detail="run not found")
    return run


@app.get("/api/v1/eval/runs/{run_id}/report")
async def get_report(run_id: str) -> dict[str, Any]:
    with _store_lock:
        run = _runs.get(run_id)
    if not run:
        raise HTTPException(status_code=404, detail="run not found")
    avg_score = sum(r.score for r in run.results) / max(len(run.results), 1)
    pass_rate = sum(1 for r in run.results if r.passed) / max(len(run.results), 1)
    return {
        "run_id": run.run_id,
        "skill_ref": run.skill_ref,
        "status": run.status,
        "summary": {
            "total_cases": len(run.cases),
            "passed": sum(1 for r in run.results if r.passed),
            "failed": sum(1 for r in run.results if not r.passed),
            "avg_score": round(avg_score, 4),
            "pass_rate": round(pass_rate, 4),
        },
        "badge": run.badge,
        "results": run.results,
        "judges": run.judges,
        "regression": run.regression,
    }


@app.post("/api/v1/eval/compare")
async def compare_runs(req: CompareRequest) -> dict[str, Any]:
    comparisons = []
    baseline_score = None
    if req.baseline_run_id:
        with _store_lock:
            baseline = _runs.get(req.baseline_run_id)
        if baseline and baseline.results:
            baseline_score = sum(r.score for r in baseline.results) / len(baseline.results)

    for run_id in req.run_ids:
        with _store_lock:
            run = _runs.get(run_id)
        if not run:
            continue
        avg = sum(r.score for r in run.results) / max(len(run.results), 1) if run.results else 0.0
        regression = baseline_score is not None and avg < baseline_score - 0.05
        comparisons.append({
            "run_id": run_id,
            "skill_ref": run.skill_ref,
            "avg_score": round(avg, 4),
            "badge": run.badge,
            "regression": regression,
        })
    return {"comparisons": comparisons, "baseline_run_id": req.baseline_run_id}
