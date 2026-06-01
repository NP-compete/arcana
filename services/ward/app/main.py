from __future__ import annotations

import json
import os
import sys
import logging
import re
import threading
import time
import uuid
from collections import defaultdict
from datetime import UTC, datetime
from enum import StrEnum
from pathlib import Path
from typing import Any

sys.path.insert(0, str(Path(__file__).resolve().parents[2]))

import asyncpg
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


app = FastAPI(
    title="Arcana Ward",
    version="0.1.0",
    description="Guardrails, policy enforcement, and safety pipeline",
)
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
log = structlog.get_logger(service="ward")

from opentelemetry import trace
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import Resource, SERVICE_NAME
from opentelemetry.instrumentation.fastapi import FastAPIInstrumentor

resource = Resource.create({SERVICE_NAME: "arcana-ward"})
provider = TracerProvider(resource=resource)
endpoint = os.environ.get("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")
provider.add_span_processor(BatchSpanProcessor(OTLPSpanExporter(endpoint=f"{endpoint}/v1/traces")))
trace.set_tracer_provider(provider)
FastAPIInstrumentor.instrument_app(app)

# ---------------------------------------------------------------------------
# Database connection pool
# ---------------------------------------------------------------------------

pool: asyncpg.Pool | None = None


@app.on_event("startup")
async def startup_db():
    global pool
    password = os.getenv("POSTGRES_PASSWORD", "" if os.getenv("ARCANA_ENV") == "production" else "arcana-dev")
    if not password:
        raise RuntimeError("POSTGRES_PASSWORD required in production")
    try:
        pool = await asyncpg.create_pool(
            host=os.getenv("POSTGRES_HOST", "postgres"),
            port=int(os.getenv("POSTGRES_PORT", "5432")),
            user=os.getenv("POSTGRES_USER", "arcana"),
            password=password,
            database=os.getenv("POSTGRES_DB", "arcana"),
            min_size=2,
            max_size=10,
        )
        log.info("database_connected", host=os.getenv("POSTGRES_HOST", "postgres"))
    except Exception as e:
        log.warn("database_unavailable_using_memory", error=str(e))


@app.on_event("shutdown")
async def shutdown_db():
    if pool:
        await pool.close()


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


class Direction(StrEnum):
    INPUT = "input"
    OUTPUT = "output"


class Verdict(StrEnum):
    ALLOW = "allow"
    BLOCK = "block"
    WARN = "warn"
    REDACT = "redact"


class Severity(StrEnum):
    LOW = "low"
    MEDIUM = "medium"
    HIGH = "high"
    CRITICAL = "critical"


class RuleAction(StrEnum):
    BLOCK = "block"
    WARN = "warn"
    REDACT = "redact"


class LayerResult(BaseModel):
    layer: str
    verdict: Verdict
    passed: bool
    detail: str = ""
    latency_ms: float = 0.0


class GuardrailRule(BaseModel):
    id: str
    type: str
    pattern: str
    action: RuleAction
    severity: Severity
    agent_id: str = "*"
    created_at: datetime = Field(default_factory=lambda: datetime.now(UTC))


class GuardrailCheck(BaseModel):
    id: str
    text: str
    direction: Direction
    verdict: Verdict
    layer_results: list[LayerResult]
    agent_id: str
    caught_by_layer: str | None = None
    redacted_text: str | None = None


class CheckRequest(BaseModel):
    text: str
    agent_id: str
    direction: Direction
    context: dict[str, Any] = Field(default_factory=dict)


class CreateRuleRequest(BaseModel):
    type: str
    pattern: str
    action: RuleAction
    severity: Severity = Severity.MEDIUM
    agent_id: str = "*"


class GuardrailStats(BaseModel):
    checks_total: int
    blocked: int
    warned: int
    passed: int
    by_layer: dict[str, int]


_store_lock = threading.Lock()
_rules: dict[str, GuardrailRule] = {}
_stats = {"checks_total": 0, "blocked": 0, "warned": 0, "passed": 0, "by_layer": defaultdict(int)}
_rate_limits: dict[str, list[float]] = defaultdict(list)
_RATE_LIMIT = 100
_RATE_WINDOW_SEC = 60

_BLOCKED_PATTERNS = [
    re.compile(r"ignore\s+(all\s+)?previous\s+instructions", re.I),
    re.compile(r"jailbreak", re.I),
    re.compile(r"system\s+prompt", re.I),
]
_PII_PATTERN = re.compile(r"\b\d{3}-\d{2}-\d{4}\b|\b\d{16}\b")
_RISK_KEYWORDS = ["weapon", "explosive", "harmful", "illegal"]


def _seed_rules() -> None:
    defaults = [
        CreateRuleRequest(type="injection", pattern="ignore previous instructions", action=RuleAction.BLOCK, severity=Severity.HIGH),
        CreateRuleRequest(type="pii", pattern=r"\d{3}-\d{2}-\d{4}", action=RuleAction.REDACT, severity=Severity.MEDIUM),
        CreateRuleRequest(type="policy", pattern="confidential", action=RuleAction.WARN, severity=Severity.LOW),
    ]
    for req in defaults:
        rule_id = str(uuid.uuid4())
        _rules[rule_id] = GuardrailRule(id=rule_id, type=req.type, pattern=req.pattern, action=req.action, severity=req.severity)


_seed_rules()


def _layer_schema_validation(text: str, context: dict[str, Any]) -> LayerResult:
    start = time.perf_counter()
    passed = len(text) <= 32000 and text.strip() != ""
    detail = "schema valid" if passed else "text empty or exceeds max length"
    return LayerResult(
        layer="schema_validation",
        verdict=Verdict.ALLOW if passed else Verdict.BLOCK,
        passed=passed,
        detail=detail,
        latency_ms=(time.perf_counter() - start) * 1000,
    )


def _layer_policy_check(text: str, agent_id: str) -> LayerResult:
    start = time.perf_counter()
    for rule in _rules.values():
        if rule.agent_id not in (agent_id, "*"):
            continue
        if rule.type == "policy" and rule.pattern.lower() in text.lower():
            verdict = Verdict.BLOCK if rule.action == RuleAction.BLOCK else Verdict.WARN
            if rule.action == RuleAction.REDACT:
                verdict = Verdict.REDACT
            return LayerResult(
                layer="policy_check",
                verdict=verdict,
                passed=False,
                detail=f"policy rule matched: {rule.pattern}",
                latency_ms=(time.perf_counter() - start) * 1000,
            )
    return LayerResult(layer="policy_check", verdict=Verdict.ALLOW, passed=True, detail="no policy violations", latency_ms=(time.perf_counter() - start) * 1000)


def _layer_rate_limiting(agent_id: str) -> LayerResult:
    start = time.perf_counter()
    now = time.time()
    key = agent_id or "anonymous"
    window_start = now - _RATE_WINDOW_SEC
    _rate_limits[key] = [t for t in _rate_limits[key] if t > window_start]
    if len(_rate_limits[key]) >= _RATE_LIMIT:
        return LayerResult(
            layer="rate_limiting",
            verdict=Verdict.BLOCK,
            passed=False,
            detail=f"rate limit exceeded ({_RATE_LIMIT}/{_RATE_WINDOW_SEC}s)",
            latency_ms=(time.perf_counter() - start) * 1000,
        )
    _rate_limits[key].append(now)
    return LayerResult(layer="rate_limiting", verdict=Verdict.ALLOW, passed=True, detail="within rate limit", latency_ms=(time.perf_counter() - start) * 1000)


def _layer_pattern_prefilter(text: str, agent_id: str = "*") -> LayerResult:
    start = time.perf_counter()
    for rule in _rules.values():
        if rule.agent_id not in (agent_id, "*"):
            continue
        if rule.type in ("injection", "pii") and rule.pattern.lower() in text.lower():
            verdict = Verdict.BLOCK if rule.action == RuleAction.BLOCK else Verdict.REDACT if rule.action == RuleAction.REDACT else Verdict.WARN
            return LayerResult(
                layer="pattern_prefilter",
                verdict=verdict,
                passed=False,
                detail=f"custom rule matched: {rule.type}",
                latency_ms=(time.perf_counter() - start) * 1000,
            )
    for pat in _BLOCKED_PATTERNS:
        if pat.search(text):
            return LayerResult(
                layer="pattern_prefilter",
                verdict=Verdict.BLOCK,
                passed=False,
                detail="blocked pattern detected",
                latency_ms=(time.perf_counter() - start) * 1000,
            )
    if _PII_PATTERN.search(text):
        return LayerResult(
            layer="pattern_prefilter",
            verdict=Verdict.REDACT,
            passed=False,
            detail="PII detected",
            latency_ms=(time.perf_counter() - start) * 1000,
        )
    return LayerResult(layer="pattern_prefilter", verdict=Verdict.ALLOW, passed=True, detail="no patterns matched", latency_ms=(time.perf_counter() - start) * 1000)


def _layer_semantic_check(text: str) -> LayerResult:
    start = time.perf_counter()
    lower = text.lower()
    for kw in _RISK_KEYWORDS:
        if kw in lower:
            return LayerResult(
                layer="semantic_check",
                verdict=Verdict.WARN,
                passed=False,
                detail=f"semantic risk keyword: {kw}",
                latency_ms=(time.perf_counter() - start) * 1000,
            )
    return LayerResult(layer="semantic_check", verdict=Verdict.ALLOW, passed=True, detail="semantic check passed", latency_ms=(time.perf_counter() - start) * 1000)


def _layer_risk_chain(text: str, direction: Direction, context: dict[str, Any]) -> LayerResult:
    start = time.perf_counter()
    risk_score = context.get("risk_score", 0.0)
    if isinstance(risk_score, (int, float)) and risk_score > 0.8:
        return LayerResult(
            layer="risk_chain",
            verdict=Verdict.BLOCK,
            passed=False,
            detail=f"risk score {risk_score} exceeds threshold",
            latency_ms=(time.perf_counter() - start) * 1000,
        )
    if direction == Direction.OUTPUT and len(text) > 10000:
        return LayerResult(
            layer="risk_chain",
            verdict=Verdict.WARN,
            passed=False,
            detail="output length exceeds recommended limit",
            latency_ms=(time.perf_counter() - start) * 1000,
        )
    return LayerResult(layer="risk_chain", verdict=Verdict.ALLOW, passed=True, detail="risk chain passed", latency_ms=(time.perf_counter() - start) * 1000)


def _run_pipeline(text: str, agent_id: str, direction: Direction, context: dict[str, Any]) -> GuardrailCheck:
    layers = [
        _layer_schema_validation(text, context),
        _layer_policy_check(text, agent_id),
        _layer_rate_limiting(agent_id),
        _layer_pattern_prefilter(text, agent_id),
        _layer_semantic_check(text),
        _layer_risk_chain(text, direction, context),
    ]

    verdict = Verdict.ALLOW
    caught_by: str | None = None
    redacted_text: str | None = None

    for lr in layers:
        if lr.verdict == Verdict.BLOCK:
            verdict = Verdict.BLOCK
            caught_by = lr.layer
            break
        if lr.verdict == Verdict.REDACT and verdict != Verdict.BLOCK:
            verdict = Verdict.REDACT
            caught_by = lr.layer
            redacted_text = _PII_PATTERN.sub("[REDACTED]", text)
        if lr.verdict == Verdict.WARN and verdict == Verdict.ALLOW:
            verdict = Verdict.WARN
            caught_by = lr.layer

    return GuardrailCheck(
        id=str(uuid.uuid4()),
        text=text,
        direction=direction,
        verdict=verdict,
        layer_results=layers,
        agent_id=agent_id,
        caught_by_layer=caught_by,
        redacted_text=redacted_text,
    )


def _update_stats(check: GuardrailCheck) -> None:
    _stats["checks_total"] += 1
    if check.verdict == Verdict.BLOCK:
        _stats["blocked"] += 1
    elif check.verdict == Verdict.WARN:
        _stats["warned"] += 1
    else:
        _stats["passed"] += 1
    for lr in check.layer_results:
        if not lr.passed:
            _stats["by_layer"][lr.layer] += 1


# --- Agent-level guardrail rules (per-agent rule sets for GuardrailBuilderPage) ---

agent_rules_db: dict[str, list] = {}


@app.get("/healthz")
async def healthz() -> dict[str, str]:
    return {"status": "ok"}


@app.get("/readyz")
async def readyz() -> dict[str, str]:
    return {"status": "ok"}


@app.post("/api/v1/check", response_model=GuardrailCheck)
async def run_check(req: CheckRequest, auth: dict = Depends(require_auth)) -> GuardrailCheck:
    with _store_lock:
        check = _run_pipeline(req.text, req.agent_id, req.direction, req.context)
        _update_stats(check)
    return check


@app.get("/api/v1/rules")
async def list_rules(auth: dict = Depends(require_auth)) -> dict[str, Any]:
    with _store_lock:
        rules = list(_rules.values())
    return {"rules": rules, "total": len(rules)}


@app.post("/api/v1/rules", response_model=GuardrailRule, status_code=201)
async def create_rule(req: CreateRuleRequest, auth: dict = Depends(require_auth)) -> GuardrailRule:
    with _store_lock:
        rule_id = str(uuid.uuid4())
        rule = GuardrailRule(
            id=rule_id, type=req.type, pattern=req.pattern,
            action=req.action, severity=req.severity, agent_id=req.agent_id,
        )
        _rules[rule_id] = rule
    return rule


@app.get("/api/v1/rules/agent/{agent_id}")
async def list_agent_rules(agent_id: str, auth: dict = Depends(require_auth)) -> dict[str, Any]:
    with _store_lock:
        rules = [r for r in _rules.values() if r.agent_id in (agent_id, "*")]
    return {"rules": rules, "total": len(rules), "agent_id": agent_id}


@app.delete("/api/v1/rules/{rule_id}")
async def delete_rule(rule_id: str, auth: dict = Depends(require_auth)) -> dict[str, str]:
    with _store_lock:
        if rule_id not in _rules:
            raise HTTPException(status_code=404, detail="rule not found")
        del _rules[rule_id]
    return {"status": "deleted", "id": rule_id}


@app.get("/api/v1/stats", response_model=GuardrailStats)
async def get_stats(auth: dict = Depends(require_auth)) -> GuardrailStats:
    with _store_lock:
        return GuardrailStats(
            checks_total=_stats["checks_total"],
            blocked=_stats["blocked"],
            warned=_stats["warned"],
            passed=_stats["passed"],
            by_layer=dict(_stats["by_layer"]),
        )


# --- Agent-specific guardrail rules for GuardrailBuilderPage ---


@app.get("/api/v1/ward/agents/{name}/rules")
async def get_agent_rules(name: str, auth: dict = Depends(require_auth)) -> dict[str, Any]:
    """Get guardrail rules for an agent."""
    if pool:
        rows = await pool.fetch(
            "SELECT id, agent, type, config, action, severity, position, created_at"
            " FROM guardrail_rules WHERE agent = $1 ORDER BY position",
            name,
        )
        rules = [
            {
                "id": r["id"],
                "type": r["type"],
                "config": json.loads(r["config"]) if isinstance(r["config"], str) else r["config"],
                "action": r["action"],
                "severity": r["severity"],
                "position": r["position"],
            }
            for r in rows
        ]
        return {"agent": name, "rules": rules}
    rules = agent_rules_db.get(name, [])
    return {"agent": name, "rules": rules}


@app.put("/api/v1/ward/agents/{name}/rules")
async def set_agent_rules(name: str, request: Request, auth: dict = Depends(require_auth)) -> dict[str, Any]:
    """Set guardrail rules for an agent."""
    body = await request.json()
    rules = body.get("rules", [])
    if not isinstance(rules, list):
        raise HTTPException(status_code=400, detail="rules must be a list")
    if pool:
        async with pool.acquire() as conn:
            async with conn.transaction():
                await conn.execute(
                    "DELETE FROM guardrail_rules WHERE agent = $1", name
                )
                for idx, rule in enumerate(rules):
                    rule_type = rule.get("type", "custom") if isinstance(rule, dict) else "custom"
                    action = rule.get("action", "block") if isinstance(rule, dict) else "block"
                    severity = rule.get("severity", "medium") if isinstance(rule, dict) else "medium"
                    config = json.dumps(rule) if isinstance(rule, dict) else json.dumps({"value": rule})
                    await conn.execute(
                        "INSERT INTO guardrail_rules (id, agent, type, config, action, severity, position)"
                        " VALUES ($1, $2, $3, $4, $5, $6, $7)",
                        str(uuid.uuid4()), name, rule_type, config, action, severity, idx,
                    )
    else:
        agent_rules_db[name] = rules
    log.info("agent_rules_updated", agent=name, rule_count=len(rules))
    return {"agent": name, "rules": rules, "updated": True}


@app.post("/api/v1/ward/evaluate")
async def evaluate_input(request: Request, auth: dict = Depends(require_auth)) -> dict[str, Any]:
    """Test input against guardrail rules."""
    body = await request.json()
    text = body.get("text", "")
    rules = body.get("rules", [])

    if not isinstance(text, str):
        raise HTTPException(status_code=400, detail="text must be a string")
    if not isinstance(rules, list):
        raise HTTPException(status_code=400, detail="rules must be a list")

    results: list[dict[str, Any]] = []
    for rule in rules:
        rule_type = rule.get("type", "unknown") if isinstance(rule, dict) else "unknown"
        result: dict[str, Any] = {"rule": rule_type, "verdict": "pass", "details": ""}

        text_lower = text.lower()
        if rule_type == "pii" and any(
            p in text_lower for p in ["ssn", "credit card", "social security"]
        ):
            result["verdict"] = "block"
            result["details"] = "PII pattern detected"
        elif rule_type == "toxicity" and any(
            t in text_lower for t in ["hate", "kill", "attack"]
        ):
            result["verdict"] = "block"
            result["details"] = "Toxic content detected"
        elif rule_type == "prompt_injection" and any(
            p in text_lower
            for p in ["ignore previous", "jailbreak", "system prompt"]
        ):
            result["verdict"] = "block"
            result["details"] = "Prompt injection pattern detected"
        elif rule_type == "topic_restriction":
            blocked_topics = (
                rule.get("blocked_topics", []) if isinstance(rule, dict) else []
            )
            for topic in blocked_topics:
                if isinstance(topic, str) and topic.lower() in text_lower:
                    result["verdict"] = "block"
                    result["details"] = f"Restricted topic detected: {topic}"
                    break

        results.append(result)

    overall = (
        "pass" if all(r["verdict"] == "pass" for r in results) else "block"
    )
    return {"text": text, "results": results, "overall": overall}


@app.get("/api/v1/ward/stats")
async def ward_stats(auth: dict = Depends(require_auth)) -> dict[str, Any]:
    """Get overall ward statistics."""
    with _store_lock:
        total = _stats["checks_total"]
        blocked = _stats["blocked"]
        warned = _stats["warned"]
        passed = _stats["passed"]

    # Include rule count from DB when available.
    rule_count = 0
    if pool:
        row = await pool.fetchrow("SELECT COUNT(*) AS cnt FROM guardrail_rules")
        rule_count = row["cnt"] if row else 0

    if total == 0:
        return {
            "total_checks": 0,
            "blocked": 0,
            "warned": 0,
            "passed": 0,
            "block_rate": 0.0,
            "top_violations": [],
        }

    block_rate = blocked / total
    # Derive per-type violation counts from by_layer stats when available.
    by_layer = dict(_stats["by_layer"])
    top_violations = [
        {"type": layer, "count": count}
        for layer, count in sorted(by_layer.items(), key=lambda x: x[1], reverse=True)
    ]
    return {
        "total_checks": total,
        "blocked": blocked,
        "warned": warned,
        "passed": passed,
        "block_rate": round(block_rate, 4),
        "top_violations": top_violations,
    }
