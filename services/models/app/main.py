from __future__ import annotations

import threading
import uuid
from datetime import UTC, datetime
from typing import Any

from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel, Field

app = FastAPI(title="Arcana Models", version="0.1.0", description="Model registry and serving")
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)


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


@app.get("/readyz")
async def readyz():
    return {"status": "ok"}

@app.get("/healthz")
async def healthz() -> dict[str, str]:
    return {"status": "ok"}


@app.post("/api/v1/models", response_model=ModelCard, status_code=201)
async def register_model(req: RegisterModelRequest) -> ModelCard:
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
async def list_models() -> dict[str, Any]:
    with _store_lock:
        models = list(_models.values())
    return {"models": models, "total": len(models)}


@app.get("/api/v1/models/{name}", response_model=ModelCard)
async def get_model(name: str) -> ModelCard:
    with _store_lock:
        card = _models.get(name)
    if not card:
        raise HTTPException(status_code=404, detail="model not found")
    return card


@app.post("/api/v1/models/{name}/promote", response_model=ModelCard)
async def promote_model(name: str, req: PromoteRequest) -> ModelCard:
    with _store_lock:
        card = _models.get(name)
        if not card:
            raise HTTPException(status_code=404, detail="model not found")
        card.environment = req.environment
        card.governance["approved"] = True
        card.governance["promoted_at"] = datetime.now(UTC).isoformat()
    return card


@app.delete("/api/v1/models/{name}")
async def deregister_model(name: str) -> dict[str, str]:
    with _store_lock:
        if name not in _models:
            raise HTTPException(status_code=404, detail="model not found")
        del _models[name]
    return {"status": "deregistered", "name": name}


@app.post("/api/v1/models/{name}/predict")
async def predict(name: str, req: PredictRequest) -> dict[str, Any]:
    global _tokens_used, _cost
    with _store_lock:
        card = _models.get(name)
        if not card:
            raise HTTPException(status_code=404, detail="model not found")

    input_text = req.input if isinstance(req.input, str) else str(req.input)
    tokens = max(1, len(input_text.split()) * 2)
    cost = tokens * _fallback_chain.thresholds.get("cost_per_token", 0.00001)

    with _store_lock:
        _tokens_used += tokens
        _cost += cost

    return {
        "model": name,
        "output": f"[{name}] Generated response for: {input_text[:200]}",
        "tokens_used": tokens,
        "cost_usd": round(cost, 6),
        "params": req.params,
    }


@app.get("/api/v1/budget", response_model=BudgetStatus)
async def get_budget() -> BudgetStatus:
    with _store_lock:
        return BudgetStatus(
            tokens_used=_tokens_used,
            cost=round(_cost, 4),
            remaining=round(_budget_limit - _cost, 4),
            budget_limit=_budget_limit,
        )


@app.post("/api/v1/budget/fallback", response_model=FallbackChain)
async def configure_fallback(req: FallbackConfigRequest) -> FallbackChain:
    global _fallback_chain
    with _store_lock:
        _fallback_chain = FallbackChain(models=req.models, thresholds=req.thresholds)
    return _fallback_chain
