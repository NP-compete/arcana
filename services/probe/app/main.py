from __future__ import annotations

import asyncio
import os
import sys
import logging
import threading
import time
import uuid
from datetime import UTC, datetime
from enum import StrEnum
from pathlib import Path
from typing import Any

try:
    import anthropic  # noqa: F401 — optional; used by _judge_llm when JUDGE_LLM_PROVIDER=anthropic
except ImportError:
    anthropic = None  # type: ignore[assignment]

sys.path.insert(0, str(Path(__file__).resolve().parents[2]))

import structlog
from fastapi import Depends, FastAPI, HTTPException, Request
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel, Field

from _shared.auth import require_auth


def _cors_origins() -> list[str]:
    origins = os.getenv("CORS_ORIGINS", "*")
    if os.getenv("ARCANA_ENV") == "production" and origins == "*":
        raise RuntimeError("CORS_ORIGINS must be set in production")
    return [o.strip() for o in origins.split(",")]


app = FastAPI(title="Arcana Probe", version="0.1.0", description="Eval framework")
app.add_middleware(
    CORSMiddleware,
    allow_origins=_cors_origins(),
    allow_methods=["*"],
    allow_headers=["*"],
)

structlog.configure(
    processors=[
        structlog.processors.TimeStamper(fmt="iso"),
        structlog.processors.add_log_level,
        structlog.processors.JSONRenderer(),
    ],
    wrapper_class=structlog.make_filtering_bound_logger(logging.INFO),
)
log = structlog.get_logger(service="probe")

from opentelemetry import trace
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import Resource, SERVICE_NAME
from opentelemetry.instrumentation.fastapi import FastAPIInstrumentor

resource = Resource.create({SERVICE_NAME: "arcana-probe"})
provider = TracerProvider(resource=resource)
endpoint = os.environ.get("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")
provider.add_span_processor(BatchSpanProcessor(OTLPSpanExporter(endpoint=f"{endpoint}/v1/traces")))
trace.set_tracer_provider(provider)
FastAPIInstrumentor.instrument_app(app)


@app.middleware("http")
async def request_logging_middleware(request: Request, call_next):
    start = time.monotonic()
    response = await call_next(request)
    duration_ms = (time.monotonic() - start) * 1000
    log.info(
        "request",
        method=request.method,
        path=str(request.url.path),
        status=response.status_code,
        duration_ms=round(duration_ms, 2),
    )
    return response


@app.middleware("http")
async def request_id_middleware(request: Request, call_next):
    request_id = request.headers.get("x-request-id", str(uuid.uuid4()))
    response = await call_next(request)
    response.headers["x-request-id"] = request_id
    return response


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


async def _judge_llm(case: EvalCase) -> tuple[float, str]:
    """Use an LLM to judge output quality on a 0.0-1.0 scale."""
    input_text = case.input
    expected = case.expected or ""
    actual = case.metadata.get("actual", "")

    provider = os.getenv("JUDGE_LLM_PROVIDER", "anthropic")

    grading_prompt = (
        "You are an evaluation judge. Score the following AI output "
        "on a scale from 0.0 to 1.0.\n\n"
        f"Input: {input_text[:500]}\n"
        f"Expected output: {expected[:500]}\n"
        f"Actual output: {actual[:500]}\n\n"
        "Score 1.0 for perfect match in meaning (not necessarily exact text).\n"
        "Score 0.0 for completely wrong or irrelevant.\n"
        "Score intermediate values for partial correctness.\n\n"
        "Respond with ONLY a decimal number between 0.0 and 1.0, nothing else."
    )

    try:
        if provider == "anthropic":
            if anthropic is None:
                raise ImportError(
                    "anthropic package is required when JUDGE_LLM_PROVIDER=anthropic. "
                    "Install with: pip install anthropic"
                )
            client = anthropic.AsyncAnthropic()
            response = await client.messages.create(
                model=os.getenv("JUDGE_MODEL", "claude-sonnet-4-20250514"),
                max_tokens=10,
                messages=[{"role": "user", "content": grading_prompt}],
            )
            score_text = response.content[0].text.strip()
        else:
            import httpx

            base_url = os.getenv("JUDGE_API_URL", "http://localhost:11434/v1")
            async with httpx.AsyncClient(timeout=30.0) as http:
                resp = await http.post(
                    f"{base_url}/chat/completions",
                    json={
                        "model": os.getenv("JUDGE_MODEL", "gpt-4o-mini"),
                        "messages": [{"role": "user", "content": grading_prompt}],
                        "max_tokens": 10,
                    },
                    headers={
                        "Authorization": f"Bearer {os.getenv('JUDGE_API_KEY', '')}",
                    },
                )
                resp.raise_for_status()
                score_text = resp.json()["choices"][0]["message"]["content"].strip()

        score = float(score_text)
        return max(0.0, min(1.0, score)), "llm grading"
    except Exception as e:
        log.warning("llm_judge_fallback", error=str(e))
        # Fallback to heuristic when LLM is unreachable
        if not actual:
            return 0.0, f"llm fallback (no actual output): {e}"
        if expected and expected.lower() in actual.lower():
            return 0.8, f"llm fallback (substring match): {e}"
        return (
            min(1.0, len(actual) / max(len(expected), 1) * 0.5),
            f"llm fallback (length heuristic): {e}",
        )


def _judge_trajectory(case: EvalCase) -> tuple[float, str]:
    """Judge an agent trajectory based on efficiency and correctness."""
    input_text = case.input
    expected = case.expected or ""
    actual = case.metadata.get("actual", "")
    metadata = case.metadata

    steps = metadata.get("steps", [])
    max_steps = metadata.get("max_steps", 10)

    if not isinstance(steps, list):
        # Legacy format: steps is an int count
        step_count = int(steps) if isinstance(steps, (int, float)) else 0
        efficiency = max(0.0, 1.0 - (step_count / max(max_steps, 1)))
    else:
        step_count = len(steps)
        efficiency = max(0.0, 1.0 - (step_count / max(max_steps, 1)))

        # Check for repeated actions (sign of being stuck)
        if step_count > 2:
            actions = [s.get("action", "") for s in steps if isinstance(s, dict)]
            unique_ratio = len(set(actions)) / max(len(actions), 1)
            if unique_ratio < 0.5:
                efficiency *= 0.5  # Penalize repetitive behavior

    # Combine efficiency with output quality
    output_score = 1.0 if actual and actual.strip() else 0.0
    if expected and actual:
        if expected.lower() in actual.lower():
            output_score = 1.0
        else:
            output_score = 0.5

    score = (efficiency * 0.4) + (output_score * 0.6)
    detail = f"trajectory: efficiency={efficiency:.2f} output={output_score:.2f} steps={step_count}/{max_steps}"
    return score, detail


def _run_judge(judge: Judge, case: EvalCase) -> tuple[float, str]:
    if judge.tier == JudgeTier.DETERMINISTIC:
        return _judge_deterministic(case)
    if judge.tier == JudgeTier.SCRIPT:
        return _judge_script(case)
    if judge.tier == JudgeTier.LLM:
        # _judge_llm is async — run it in a new event loop since we are in a thread
        return asyncio.run(_judge_llm(case))
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
async def trigger_eval(req: RunEvalRequest, auth: dict = Depends(require_auth)) -> EvalRun:
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
async def list_runs(auth: dict = Depends(require_auth)) -> dict[str, Any]:
    with _store_lock:
        runs = list(_runs.values())
    return {"runs": runs, "total": len(runs)}


@app.get("/api/v1/eval/runs/{run_id}", response_model=EvalRun)
async def get_run(run_id: str, auth: dict = Depends(require_auth)) -> EvalRun:
    with _store_lock:
        run = _runs.get(run_id)
    if not run:
        raise HTTPException(status_code=404, detail="run not found")
    return run


@app.get("/api/v1/eval/runs/{run_id}/report")
async def get_report(run_id: str, auth: dict = Depends(require_auth)) -> dict[str, Any]:
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
async def compare_runs(req: CompareRequest, auth: dict = Depends(require_auth)) -> dict[str, Any]:
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
