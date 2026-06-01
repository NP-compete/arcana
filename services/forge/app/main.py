from __future__ import annotations

import os
import logging
import threading
import time
import uuid
from datetime import UTC, datetime
from enum import StrEnum
from typing import Any

import structlog
from fastapi import FastAPI, HTTPException, Request
from fastapi.middleware.cors import CORSMiddleware
from opentelemetry import trace
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.instrumentation.fastapi import FastAPIInstrumentor
from opentelemetry.sdk.resources import SERVICE_NAME, Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from pydantic import BaseModel, Field


structlog.configure(
    processors=[
        structlog.processors.TimeStamper(fmt="iso"),
        structlog.processors.add_log_level,
        structlog.processors.JSONRenderer(),
    ],
    wrapper_class=structlog.make_filtering_bound_logger(logging.INFO),
)
log = structlog.get_logger(service="forge")

app = FastAPI(title="Arcana Forge", version="0.1.0", description="Fine-tuning pipeline")
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)


resource = Resource.create({SERVICE_NAME: "arcana-forge"})
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


class TrainingMethod(StrEnum):
    LORA = "lora"
    QLORA = "qlora"
    FULL = "full"
    DPO = "dpo"


class ExperimentStatus(StrEnum):
    PENDING = "pending"
    RUNNING = "running"
    COMPLETED = "completed"
    FAILED = "failed"
    CANCELLED = "cancelled"


class Dataset(BaseModel):
    id: str
    name: str
    path: str
    format: str
    size: int
    schema: dict[str, Any] = Field(default_factory=dict)
    created_at: datetime


class Experiment(BaseModel):
    id: str
    name: str
    base_model: str
    method: TrainingMethod
    status: ExperimentStatus
    progress_pct: float
    metrics: dict[str, float] = Field(default_factory=dict)
    config: dict[str, Any] = Field(default_factory=dict)
    dataset: str | None = None
    artifacts: list[str] = Field(default_factory=list)


class CreateExperimentRequest(BaseModel):
    name: str
    base_model: str
    dataset: str
    method: TrainingMethod
    config: dict[str, Any] = Field(default_factory=dict)


class RegisterDatasetRequest(BaseModel):
    name: str
    path: str
    format: str
    size: int
    schema: dict[str, Any] = Field(default_factory=dict)


_store_lock = threading.Lock()
_experiments: dict[str, Experiment] = {}
_datasets: dict[str, Dataset] = {}
_cancel_flags: dict[str, bool] = {}


def _run_experiment(exp_id: str) -> None:
    steps = 20
    for i in range(1, steps + 1):
        time.sleep(0.05)
        with _store_lock:
            if _cancel_flags.get(exp_id):
                exp = _experiments.get(exp_id)
                if exp:
                    exp.status = ExperimentStatus.CANCELLED
                return
            exp = _experiments.get(exp_id)
            if not exp:
                return
            exp.status = ExperimentStatus.RUNNING
            exp.progress_pct = (i / steps) * 100
            exp.metrics = {
                "loss": max(0.1, 2.5 - (i * 0.1)),
                "learning_rate": exp.config.get("learning_rate", 2e-5),
                "epoch": i / (steps / 3),
            }
    with _store_lock:
        exp = _experiments.get(exp_id)
        if exp and exp.status != ExperimentStatus.CANCELLED:
            exp.status = ExperimentStatus.COMPLETED
            exp.progress_pct = 100.0
            exp.metrics["final_loss"] = exp.metrics.get("loss", 0.1)
            exp.artifacts = [f"s3://forge-artifacts/{exp_id}/checkpoint-final"]


@app.get("/readyz")
async def readyz():
    return {"status": "ok"}

@app.get("/healthz")
async def healthz() -> dict[str, str]:
    return {"status": "ok"}


@app.post("/api/v1/experiments", response_model=Experiment, status_code=201)
async def create_experiment(req: CreateExperimentRequest) -> Experiment:
    with _store_lock:
        if req.dataset not in _datasets:
            raise HTTPException(status_code=404, detail="dataset not found")
        exp_id = str(uuid.uuid4())
        exp = Experiment(
            id=exp_id,
            name=req.name,
            base_model=req.base_model,
            method=req.method,
            status=ExperimentStatus.PENDING,
            progress_pct=0.0,
            config=req.config,
            dataset=req.dataset,
        )
        _experiments[exp_id] = exp
        _cancel_flags[exp_id] = False
    threading.Thread(target=_run_experiment, args=(exp_id,), daemon=True).start()
    return exp


@app.get("/api/v1/experiments")
async def list_experiments() -> dict[str, Any]:
    with _store_lock:
        exps = list(_experiments.values())
    return {"experiments": exps, "total": len(exps)}


@app.get("/api/v1/experiments/{exp_id}", response_model=Experiment)
async def get_experiment(exp_id: str) -> Experiment:
    with _store_lock:
        exp = _experiments.get(exp_id)
    if not exp:
        raise HTTPException(status_code=404, detail="experiment not found")
    return exp


@app.post("/api/v1/experiments/{exp_id}/cancel", response_model=Experiment)
async def cancel_experiment(exp_id: str) -> Experiment:
    with _store_lock:
        exp = _experiments.get(exp_id)
        if not exp:
            raise HTTPException(status_code=404, detail="experiment not found")
        if exp.status in (ExperimentStatus.COMPLETED, ExperimentStatus.CANCELLED):
            raise HTTPException(status_code=409, detail=f"experiment already {exp.status}")
        _cancel_flags[exp_id] = True
        exp.status = ExperimentStatus.CANCELLED
    return exp


@app.get("/api/v1/datasets")
async def list_datasets() -> dict[str, Any]:
    with _store_lock:
        ds = list(_datasets.values())
    return {"datasets": ds, "total": len(ds)}


@app.post("/api/v1/datasets", response_model=Dataset, status_code=201)
async def register_dataset(req: RegisterDatasetRequest) -> Dataset:
    with _store_lock:
        ds_id = str(uuid.uuid4())
        ds = Dataset(
            id=ds_id,
            name=req.name,
            path=req.path,
            format=req.format,
            size=req.size,
            schema=req.schema,
            created_at=datetime.now(UTC),
        )
        _datasets[ds_id] = ds
    return ds
