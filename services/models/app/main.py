from __future__ import annotations

import os
import sys
import logging
import threading
import time
import uuid
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

sys.path.insert(0, str(Path(__file__).resolve().parents[2]))

try:
    import anthropic as _anthropic_mod  # noqa: F401
except ImportError:
    _anthropic_mod = None  # type: ignore[assignment]

import httpx
import structlog
from fastapi import Depends, FastAPI, HTTPException, Request
from fastapi.middleware.cors import CORSMiddleware
from opentelemetry import trace
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.instrumentation.fastapi import FastAPIInstrumentor
from opentelemetry.sdk.resources import SERVICE_NAME, Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from pydantic import BaseModel, Field

from _shared.auth import require_auth


def _cors_origins() -> list[str]:
    origins = os.getenv("CORS_ORIGINS", "*")
    if os.getenv("ARCANA_ENV") == "production" and origins == "*":
        raise RuntimeError("CORS_ORIGINS must be set in production")
    return [o.strip() for o in origins.split(",")]


structlog.configure(
    processors=[
        structlog.processors.TimeStamper(fmt="iso"),
        structlog.processors.add_log_level,
        structlog.processors.JSONRenderer(),
    ],
    wrapper_class=structlog.make_filtering_bound_logger(logging.INFO),
)
log = structlog.get_logger(service="models")

app = FastAPI(title="Arcana Models", version="0.1.0", description="Model registry and serving")
app.add_middleware(
    CORSMiddleware,
    allow_origins=_cors_origins(),
    allow_methods=["*"],
    allow_headers=["*"],
)


resource = Resource.create({SERVICE_NAME: "arcana-models"})
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


class ModelCard(BaseModel):
    name: str
    framework: str
    base_model: str
    task: str
    metrics: dict[str, float] = Field(default_factory=dict)
    serving: dict[str, Any] = Field(default_factory=dict)
    governance: dict[str, Any] = Field(default_factory=dict)
    environment: str = "dev"
    registered_at: datetime = Field(default_factory=lambda: datetime.now(UTC))


class BudgetStatus(BaseModel):
    tokens_used: int
    cost: float
    remaining: float
    budget_limit: float
    period: str = "monthly"


class FallbackChain(BaseModel):
    models: list[str]
    thresholds: dict[str, float] = Field(default_factory=dict)


class RegisterModelRequest(BaseModel):
    name: str
    framework: str
    base_model: str
    task: str
    metrics: dict[str, float] = Field(default_factory=dict)
    serving_config: dict[str, Any] = Field(default_factory=dict)


class PredictRequest(BaseModel):
    input: str | dict[str, Any]
    params: dict[str, Any] = Field(default_factory=dict)


class PromoteRequest(BaseModel):
    environment: str


class FallbackConfigRequest(BaseModel):
    models: list[str]
    thresholds: dict[str, float] = Field(default_factory=dict)


_store_lock = threading.Lock()
_models: dict[str, ModelCard] = {}
_tokens_used = 0
_cost = 0.0
_budget_limit = 1000.0
_fallback_chain = FallbackChain(models=["gpt-4o-mini", "llama-3-8b"], thresholds={"cost_per_token": 0.00001})


class ModelRouter:
    """Routes inference requests to the appropriate LLM backend."""

    def __init__(self) -> None:
        self._anthropic_client: Any = None
        self._http_client: httpx.AsyncClient | None = None

    async def predict(
        self, model_name: str, model_info: dict[str, Any], input_text: str
    ) -> tuple[str, int, int, float]:
        """Returns (output, input_tokens, output_tokens, cost_usd)."""
        provider = model_info.get("provider", "anthropic")
        model_id = model_info.get("model_id", model_name)

        if provider == "anthropic":
            return await self._predict_anthropic(model_id, input_text)
        elif provider in ("openai", "vllm"):
            base_url = model_info.get("base_url") or os.getenv(
                "OPENAI_BASE_URL", "https://api.openai.com/v1"
            )
            api_key = os.getenv("OPENAI_API_KEY", "")
            return await self._predict_openai_compat(
                base_url, api_key, model_id, input_text
            )
        else:
            raise ValueError(f"unsupported provider: {provider}")

    async def _predict_anthropic(
        self, model_id: str, input_text: str
    ) -> tuple[str, int, int, float]:
        if _anthropic_mod is None:
            raise ImportError(
                "anthropic package is required for provider='anthropic'. "
                "Install with: pip install anthropic"
            )
        if not self._anthropic_client:
            self._anthropic_client = _anthropic_mod.AsyncAnthropic()

        response = await self._anthropic_client.messages.create(
            model=model_id,
            max_tokens=int(os.getenv("MAX_OUTPUT_TOKENS", "4096")),
            messages=[{"role": "user", "content": input_text}],
        )

        output = response.content[0].text
        input_tokens = response.usage.input_tokens
        output_tokens = response.usage.output_tokens

        # Cost calculation (approximate, per 1M tokens)
        cost_per_1m_input = {
            "claude-sonnet-4-20250514": 3.0,
            "claude-haiku-4-5-20251001": 0.80,
        }.get(model_id, 3.0)
        cost_per_1m_output = {
            "claude-sonnet-4-20250514": 15.0,
            "claude-haiku-4-5-20251001": 4.0,
        }.get(model_id, 15.0)
        cost = (
            input_tokens * cost_per_1m_input + output_tokens * cost_per_1m_output
        ) / 1_000_000

        return output, input_tokens, output_tokens, cost

    async def _predict_openai_compat(
        self,
        base_url: str,
        api_key: str,
        model_id: str,
        input_text: str,
    ) -> tuple[str, int, int, float]:
        if not self._http_client:
            self._http_client = httpx.AsyncClient(timeout=60.0)

        resp = await self._http_client.post(
            f"{base_url}/chat/completions",
            json={
                "model": model_id,
                "messages": [{"role": "user", "content": input_text}],
                "max_tokens": int(os.getenv("MAX_OUTPUT_TOKENS", "4096")),
            },
            headers={"Authorization": f"Bearer {api_key}"},
        )
        resp.raise_for_status()
        data = resp.json()

        output = data["choices"][0]["message"]["content"]
        usage = data.get("usage", {})
        input_tokens = usage.get("prompt_tokens", 0)
        output_tokens = usage.get("completion_tokens", 0)
        cost = (input_tokens + output_tokens) * 0.000003  # rough estimate

        return output, input_tokens, output_tokens, cost


_router = ModelRouter()


@app.get("/readyz")
async def readyz():
    return {"status": "ok"}

@app.get("/healthz")
async def healthz() -> dict[str, str]:
    return {"status": "ok"}


@app.post("/api/v1/models", response_model=ModelCard, status_code=201)
async def register_model(req: RegisterModelRequest, auth: dict = Depends(require_auth)) -> ModelCard:
    with _store_lock:
        if req.name in _models:
            raise HTTPException(status_code=409, detail="model already registered")
        card = ModelCard(
            name=req.name,
            framework=req.framework,
            base_model=req.base_model,
            task=req.task,
            metrics=req.metrics,
            serving=req.serving_config,
            governance={"approved": False, "owner": "platform"},
        )
        _models[req.name] = card
    return card


@app.get("/api/v1/models")
async def list_models(auth: dict = Depends(require_auth)) -> dict[str, Any]:
    with _store_lock:
        models = list(_models.values())
    return {"models": models, "total": len(models)}


@app.get("/api/v1/models/{name}", response_model=ModelCard)
async def get_model(name: str, auth: dict = Depends(require_auth)) -> ModelCard:
    with _store_lock:
        card = _models.get(name)
    if not card:
        raise HTTPException(status_code=404, detail="model not found")
    return card


@app.post("/api/v1/models/{name}/promote", response_model=ModelCard)
async def promote_model(name: str, req: PromoteRequest, auth: dict = Depends(require_auth)) -> ModelCard:
    with _store_lock:
        card = _models.get(name)
        if not card:
            raise HTTPException(status_code=404, detail="model not found")
        card.environment = req.environment
        card.governance["approved"] = True
        card.governance["promoted_at"] = datetime.now(UTC).isoformat()
    return card


@app.delete("/api/v1/models/{name}")
async def deregister_model(name: str, auth: dict = Depends(require_auth)) -> dict[str, str]:
    with _store_lock:
        if name not in _models:
            raise HTTPException(status_code=404, detail="model not found")
        del _models[name]
    return {"status": "deregistered", "name": name}


@app.post("/api/v1/models/{name}/predict")
async def predict(name: str, req: PredictRequest, auth: dict = Depends(require_auth)) -> dict[str, Any]:
    global _tokens_used, _cost
    with _store_lock:
        card = _models.get(name)
        if not card:
            raise HTTPException(status_code=404, detail="model not found")
        model_serving = dict(card.serving)

    input_text = req.input if isinstance(req.input, str) else str(req.input)

    # Check budget before inference
    if _tokens_used >= _budget_limit:
        # Try fallback chain
        fallback_found = False
        for fallback_name in _fallback_chain.models:
            with _store_lock:
                if fallback_name in _models:
                    name = fallback_name
                    card = _models[name]
                    model_serving = dict(card.serving)
                    fallback_found = True
                    break
        if not fallback_found:
            raise HTTPException(
                status_code=429,
                detail="token budget exceeded and no fallback available",
            )

    try:
        output, in_tok, out_tok, cost = await _router.predict(
            name, model_serving, input_text
        )
    except Exception as e:
        log.error("prediction_failed", model=name, error=str(e))
        # Try fallback chain
        output = None
        in_tok = out_tok = 0
        cost = 0.0
        for fallback_name in _fallback_chain.models:
            if fallback_name != name:
                with _store_lock:
                    fallback_card = _models.get(fallback_name)
                if fallback_card:
                    try:
                        output, in_tok, out_tok, cost = await _router.predict(
                            fallback_name, dict(fallback_card.serving), input_text
                        )
                        name = fallback_name
                        break
                    except Exception:
                        continue
        if output is None:
            raise HTTPException(
                status_code=502, detail=f"inference failed: {e}"
            ) from e

    with _store_lock:
        _tokens_used += in_tok + out_tok
        _cost += cost

    return {
        "model": name,
        "output": output,
        "tokens": {"input": in_tok, "output": out_tok, "total": in_tok + out_tok},
        "cost_usd": round(cost, 6),
        "budget_remaining": max(0, _budget_limit - _tokens_used),
    }


@app.get("/api/v1/budget", response_model=BudgetStatus)
async def get_budget(auth: dict = Depends(require_auth)) -> BudgetStatus:
    with _store_lock:
        return BudgetStatus(
            tokens_used=_tokens_used,
            cost=round(_cost, 4),
            remaining=round(_budget_limit - _cost, 4),
            budget_limit=_budget_limit,
        )


@app.post("/api/v1/budget/fallback", response_model=FallbackChain)
async def configure_fallback(req: FallbackConfigRequest, auth: dict = Depends(require_auth)) -> FallbackChain:
    global _fallback_chain
    with _store_lock:
        _fallback_chain = FallbackChain(models=req.models, thresholds=req.thresholds)
    return _fallback_chain
