from __future__ import annotations

import re
import threading
import time
import uuid
from collections import defaultdict
from datetime import UTC, datetime
from enum import StrEnum
from typing import Any

from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel, Field

app = FastAPI(
    title="Arcana Ward",
    version="0.1.0",
    description="Guardrails, policy enforcement, and safety pipeline",
)
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)


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


def _layer_pattern_prefilter(text: str) -> LayerResult:
    start = time.perf_counter()
    for rule in _rules.values():
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
        _layer_pattern_prefilter(text),
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


@app.get("/healthz")
async def healthz() -> dict[str, str]:
    return {"status": "ok"}


@app.get("/readyz")
async def readyz() -> dict[str, str]:
    return {"status": "ok"}


@app.post("/api/v1/check", response_model=GuardrailCheck)
async def run_check(req: CheckRequest) -> GuardrailCheck:
    with _store_lock:
        check = _run_pipeline(req.text, req.agent_id, req.direction, req.context)
        _update_stats(check)
    return check


@app.get("/api/v1/rules")
async def list_rules() -> dict[str, Any]:
    with _store_lock:
        rules = list(_rules.values())
    return {"rules": rules, "total": len(rules)}


@app.post("/api/v1/rules", response_model=GuardrailRule, status_code=201)
async def create_rule(req: CreateRuleRequest) -> GuardrailRule:
    with _store_lock:
        rule_id = str(uuid.uuid4())
        rule = GuardrailRule(id=rule_id, type=req.type, pattern=req.pattern, action=req.action, severity=req.severity)
        _rules[rule_id] = rule
    return rule


@app.delete("/api/v1/rules/{rule_id}")
async def delete_rule(rule_id: str) -> dict[str, str]:
    with _store_lock:
        if rule_id not in _rules:
            raise HTTPException(status_code=404, detail="rule not found")
        del _rules[rule_id]
    return {"status": "deleted", "id": rule_id}


@app.get("/api/v1/stats", response_model=GuardrailStats)
async def get_stats() -> GuardrailStats:
    with _store_lock:
        return GuardrailStats(
            checks_total=_stats["checks_total"],
            blocked=_stats["blocked"],
            warned=_stats["warned"],
            passed=_stats["passed"],
            by_layer=dict(_stats["by_layer"]),
        )
