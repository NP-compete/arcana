from __future__ import annotations

import asyncio
import json
import os
import sys
import logging
import threading
import time
import uuid
from abc import ABC, abstractmethod
from datetime import UTC, datetime
from enum import StrEnum
from pathlib import Path
from typing import Any

sys.path.insert(0, str(Path(__file__).resolve().parents[2]))

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
log = structlog.get_logger(service="forge")

app = FastAPI(title="Arcana Forge", version="0.1.0", description="Fine-tuning pipeline")
app.add_middleware(
    CORSMiddleware,
    allow_origins=_cors_origins(),
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


def _utcnow() -> datetime:
    return datetime.now(UTC)


# ---------------------------------------------------------------------------
# Training backends
# ---------------------------------------------------------------------------

class TrainingBackend(ABC):
    @abstractmethod
    async def run(self, experiment: dict, dataset: dict, config: dict) -> None: ...


class SimulatedBackend(TrainingBackend):
    """Development mode: simulates training progress."""

    async def run(self, experiment: dict, dataset: dict, config: dict) -> None:
        epochs = config.get("epochs", 3)
        for epoch in range(epochs):
            for step in range(10):
                await asyncio.sleep(0.05)
                loss = max(0.1, 2.5 - (epoch * 0.5 + step * 0.05))
                experiment["metrics"].append({
                    "epoch": epoch + 1,
                    "step": step + 1,
                    "loss": round(loss, 4),
                    "learning_rate": config.get("learning_rate", 5e-5),
                    "timestamp": _utcnow().isoformat(),
                })
                if experiment["status"] == "cancelled":
                    return
        experiment["status"] = "completed"
        experiment["artifacts"] = {
            "checkpoint": f"s3://forge-artifacts/{experiment['id']}/checkpoint-final",
            "config": config,
        }


class K8sJobBackend(TrainingBackend):
    """Production mode: launches training as a Kubernetes Job."""

    async def run(self, experiment: dict, dataset: dict, config: dict) -> None:
        k8s_host = os.getenv("KUBERNETES_SERVICE_HOST")
        k8s_port = os.getenv("KUBERNETES_SERVICE_PORT")

        if not k8s_host:
            log.warn("k8s_not_available_falling_back_to_simulated")
            await SimulatedBackend().run(experiment, dataset, config)
            return

        token_path = "/var/run/secrets/kubernetes.io/serviceaccount/token"
        try:
            with open(token_path) as f:
                token = f.read().strip()
        except FileNotFoundError:
            log.warn("k8s_token_not_found_falling_back_to_simulated")
            await SimulatedBackend().run(experiment, dataset, config)
            return

        import httpx

        namespace = os.getenv("TRAINING_NAMESPACE", "arcana")
        job_name = f"forge-{experiment['id'][:8]}"
        image = os.getenv("TRAINING_IMAGE", "ghcr.io/np-compete/arcana-training:latest")

        job_spec = {
            "apiVersion": "batch/v1",
            "kind": "Job",
            "metadata": {"name": job_name, "namespace": namespace},
            "spec": {
                "backoffLimit": 1,
                "template": {
                    "spec": {
                        "restartPolicy": "Never",
                        "containers": [{
                            "name": "trainer",
                            "image": image,
                            "env": [
                                {"name": "EXPERIMENT_ID", "value": experiment["id"]},
                                {"name": "BASE_MODEL", "value": experiment.get("base_model", "")},
                                {"name": "DATASET_ID", "value": experiment.get("dataset_id", "")},
                                {"name": "CONFIG", "value": json.dumps(config)},
                            ],
                            "resources": {
                                "requests": {"memory": "4Gi", "cpu": "2"},
                                "limits": {"memory": "8Gi", "cpu": "4"},
                            },
                        }],
                    },
                },
            },
        }

        async with httpx.AsyncClient(
            base_url=f"https://{k8s_host}:{k8s_port}",
            verify=False,
            headers={"Authorization": f"Bearer {token}"},
            timeout=30.0,
        ) as http_client:
            resp = await http_client.post(
                f"/apis/batch/v1/namespaces/{namespace}/jobs",
                json=job_spec,
            )
            if resp.status_code not in (200, 201, 409):
                experiment["status"] = "failed"
                experiment["metrics"].append({"error": f"Failed to create K8s Job: HTTP {resp.status_code}"})
                return

            for _ in range(360):  # Max 30 minutes
                await asyncio.sleep(5)
                if experiment["status"] == "cancelled":
                    await http_client.delete(f"/apis/batch/v1/namespaces/{namespace}/jobs/{job_name}")
                    return

                resp = await http_client.get(f"/apis/batch/v1/namespaces/{namespace}/jobs/{job_name}")
                if resp.status_code != 200:
                    continue

                job_status = resp.json().get("status", {})
                if job_status.get("succeeded", 0) > 0:
                    experiment["status"] = "completed"
                    experiment["artifacts"] = {
                        "checkpoint": f"s3://forge-artifacts/{experiment['id']}/checkpoint-final",
                    }
                    return
                if job_status.get("failed", 0) > 0:
                    experiment["status"] = "failed"
                    return

            experiment["status"] = "timeout"


def _get_training_backend() -> TrainingBackend:
    mode = os.getenv("TRAINING_BACKEND", "simulated")
    if mode == "k8s":
        return K8sJobBackend()
    return SimulatedBackend()


# ---------------------------------------------------------------------------
# In-memory stores
# ---------------------------------------------------------------------------

_store_lock = threading.Lock()
_experiments: dict[str, Experiment] = {}
_datasets: dict[str, Dataset] = {}
_cancel_flags: dict[str, bool] = {}


def _run_experiment(exp_id: str) -> None:
    exp = _experiments.get(exp_id)
    if not exp:
        return
    exp.status = ExperimentStatus.RUNNING

    exp_dict: dict = {
        "id": exp.id,
        "status": "running",
        "metrics": [],
        "base_model": exp.base_model,
        "dataset_id": exp.dataset or "",
    }
    dataset_dict: dict = {}
    if exp.dataset and exp.dataset in _datasets:
        ds = _datasets[exp.dataset]
        dataset_dict = {"id": ds.id, "name": ds.name, "path": ds.path}

    config = dict(exp.config)
    backend = _get_training_backend()

    loop = asyncio.new_event_loop()
    try:
        loop.run_until_complete(backend.run(exp_dict, dataset_dict, config))
    finally:
        loop.close()

    with _store_lock:
        exp = _experiments.get(exp_id)
        if not exp:
            return
        if _cancel_flags.get(exp_id):
            exp.status = ExperimentStatus.CANCELLED
            return
        if exp_dict["status"] == "completed":
            exp.status = ExperimentStatus.COMPLETED
            exp.progress_pct = 100.0
            metrics_list = exp_dict.get("metrics", [])
            if metrics_list:
                last = metrics_list[-1]
                exp.metrics = {
                    "loss": last.get("loss", 0.0),
                    "learning_rate": last.get("learning_rate", 0.0),
                    "final_loss": last.get("loss", 0.0),
                }
            artifacts = exp_dict.get("artifacts", {})
            if isinstance(artifacts, dict) and "checkpoint" in artifacts:
                exp.artifacts = [artifacts["checkpoint"]]
        elif exp_dict["status"] in ("failed", "timeout"):
            exp.status = ExperimentStatus.FAILED
        else:
            exp.status = ExperimentStatus.COMPLETED
            exp.progress_pct = 100.0


@app.get("/readyz")
async def readyz():
    return {"status": "ok"}

@app.get("/healthz")
async def healthz() -> dict[str, str]:
    return {"status": "ok"}


@app.post("/api/v1/experiments", response_model=Experiment, status_code=201)
async def create_experiment(req: CreateExperimentRequest, auth: dict = Depends(require_auth)) -> Experiment:
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
async def list_experiments(auth: dict = Depends(require_auth)) -> dict[str, Any]:
    with _store_lock:
        exps = list(_experiments.values())
    return {"experiments": exps, "total": len(exps)}


@app.get("/api/v1/experiments/{exp_id}", response_model=Experiment)
async def get_experiment(exp_id: str, auth: dict = Depends(require_auth)) -> Experiment:
    with _store_lock:
        exp = _experiments.get(exp_id)
    if not exp:
        raise HTTPException(status_code=404, detail="experiment not found")
    return exp


@app.post("/api/v1/experiments/{exp_id}/cancel", response_model=Experiment)
async def cancel_experiment(exp_id: str, auth: dict = Depends(require_auth)) -> Experiment:
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
async def list_datasets(auth: dict = Depends(require_auth)) -> dict[str, Any]:
    with _store_lock:
        ds = list(_datasets.values())
    return {"datasets": ds, "total": len(ds)}


@app.post("/api/v1/datasets", response_model=Dataset, status_code=201)
async def register_dataset(req: RegisterDatasetRequest, auth: dict = Depends(require_auth)) -> Dataset:
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
