"""Tests for the Arcana Forge service with real training orchestration."""
from __future__ import annotations

import time
from unittest.mock import AsyncMock, MagicMock, patch

import pytest
from fastapi.testclient import TestClient

from app.main import (
    K8sJobBackend,
    SimulatedBackend,
    TrainingBackend,
    _get_training_backend,
    app,
)

client = TestClient(app)


# ---------------------------------------------------------------------------
# Health / readiness probes
# ---------------------------------------------------------------------------

def test_healthz():
    response = client.get("/healthz")
    assert response.status_code == 200
    assert response.json() == {"status": "ok"}


def test_readyz():
    response = client.get("/readyz")
    assert response.status_code == 200
    assert response.json() == {"status": "ok"}


# ---------------------------------------------------------------------------
# Training backend selection
# ---------------------------------------------------------------------------

def test_get_training_backend_default_is_simulated():
    with patch.dict("os.environ", {}, clear=False):
        # Remove TRAINING_BACKEND if present
        import os
        os.environ.pop("TRAINING_BACKEND", None)
        backend = _get_training_backend()
        assert isinstance(backend, SimulatedBackend)


def test_get_training_backend_k8s():
    with patch.dict("os.environ", {"TRAINING_BACKEND": "k8s"}):
        backend = _get_training_backend()
        assert isinstance(backend, K8sJobBackend)


def test_get_training_backend_simulated_explicit():
    with patch.dict("os.environ", {"TRAINING_BACKEND": "simulated"}):
        backend = _get_training_backend()
        assert isinstance(backend, SimulatedBackend)


def test_all_backends_are_training_backend_subclasses():
    assert issubclass(SimulatedBackend, TrainingBackend)
    assert issubclass(K8sJobBackend, TrainingBackend)


# ---------------------------------------------------------------------------
# Simulated backend
# ---------------------------------------------------------------------------

@pytest.mark.asyncio
async def test_simulated_backend_runs_to_completion():
    backend = SimulatedBackend()
    experiment = {
        "id": "test-123",
        "status": "pending",
        "metrics": [],
        "base_model": "test-model",
        "dataset_id": "ds-1",
    }
    config = {"epochs": 2, "learning_rate": 1e-4}
    dataset = {}

    await backend.run(experiment, dataset, config)

    assert experiment["status"] == "completed"
    assert len(experiment["metrics"]) == 20  # 2 epochs * 10 steps
    assert "artifacts" in experiment
    assert "checkpoint" in experiment["artifacts"]


@pytest.mark.asyncio
async def test_simulated_backend_cancellation():
    backend = SimulatedBackend()
    experiment = {
        "id": "test-cancel",
        "status": "cancelled",  # Already cancelled
        "metrics": [],
        "base_model": "test-model",
        "dataset_id": "ds-1",
    }
    config = {"epochs": 3}
    dataset = {}

    await backend.run(experiment, dataset, config)

    # Should exit early due to cancellation — at most one metric recorded
    # before the cancellation check fires
    assert experiment["status"] == "cancelled"
    assert len(experiment["metrics"]) <= 1
    # 3 epochs * 10 steps = 30 total — we should be far short of that
    assert len(experiment["metrics"]) < 30


@pytest.mark.asyncio
async def test_simulated_backend_metrics_structure():
    backend = SimulatedBackend()
    experiment = {
        "id": "test-metrics",
        "status": "pending",
        "metrics": [],
        "base_model": "test-model",
        "dataset_id": "ds-1",
    }
    config = {"epochs": 1, "learning_rate": 3e-5}
    dataset = {}

    await backend.run(experiment, dataset, config)

    for m in experiment["metrics"]:
        assert "epoch" in m
        assert "step" in m
        assert "loss" in m
        assert "learning_rate" in m
        assert "timestamp" in m
        assert m["learning_rate"] == 3e-5


@pytest.mark.asyncio
async def test_simulated_backend_loss_decreases():
    backend = SimulatedBackend()
    experiment = {
        "id": "test-loss",
        "status": "pending",
        "metrics": [],
        "base_model": "test-model",
        "dataset_id": "ds-1",
    }
    config = {"epochs": 3}
    dataset = {}

    await backend.run(experiment, dataset, config)

    losses = [m["loss"] for m in experiment["metrics"]]
    assert losses[0] > losses[-1], "Loss should decrease over training"


# ---------------------------------------------------------------------------
# K8s backend fallback behavior
# ---------------------------------------------------------------------------

@pytest.mark.asyncio
async def test_k8s_backend_falls_back_without_k8s_host():
    """Without KUBERNETES_SERVICE_HOST, K8sJobBackend falls back to simulated."""
    backend = K8sJobBackend()
    experiment = {
        "id": "test-k8s-fallback",
        "status": "pending",
        "metrics": [],
        "base_model": "test-model",
        "dataset_id": "ds-1",
    }
    config = {"epochs": 1}
    dataset = {}

    with patch.dict("os.environ", {}, clear=False):
        import os
        os.environ.pop("KUBERNETES_SERVICE_HOST", None)
        os.environ.pop("KUBERNETES_SERVICE_PORT", None)
        await backend.run(experiment, dataset, config)

    # Should complete via simulated fallback
    assert experiment["status"] == "completed"
    assert len(experiment["metrics"]) > 0


# ---------------------------------------------------------------------------
# Dataset CRUD
# ---------------------------------------------------------------------------

def test_register_dataset():
    response = client.post(
        "/api/v1/datasets",
        json={
            "name": "test-dataset",
            "path": "s3://data/test.jsonl",
            "format": "jsonl",
            "size": 1000,
        },
    )
    assert response.status_code == 201
    data = response.json()
    assert data["name"] == "test-dataset"
    assert data["format"] == "jsonl"
    assert "id" in data


def test_list_datasets():
    response = client.get("/api/v1/datasets")
    assert response.status_code == 200
    data = response.json()
    assert "datasets" in data
    assert "total" in data


# ---------------------------------------------------------------------------
# Experiment CRUD
# ---------------------------------------------------------------------------

def _create_dataset() -> str:
    """Helper to register a dataset and return its id."""
    response = client.post(
        "/api/v1/datasets",
        json={
            "name": f"ds-{time.monotonic_ns()}",
            "path": "s3://data/train.jsonl",
            "format": "jsonl",
            "size": 5000,
        },
    )
    return response.json()["id"]


def test_create_experiment():
    ds_id = _create_dataset()
    response = client.post(
        "/api/v1/experiments",
        json={
            "name": "test-exp",
            "base_model": "meta-llama/Llama-3-8b",
            "dataset": ds_id,
            "method": "lora",
            "config": {"epochs": 1, "learning_rate": 2e-5},
        },
    )
    assert response.status_code == 201
    data = response.json()
    assert data["name"] == "test-exp"
    # The background thread starts immediately, so status may already be "running"
    assert data["status"] in ("pending", "running")
    assert "id" in data


def test_create_experiment_dataset_not_found():
    response = client.post(
        "/api/v1/experiments",
        json={
            "name": "test-exp-no-ds",
            "base_model": "test-model",
            "dataset": "nonexistent-dataset-id",
            "method": "lora",
        },
    )
    assert response.status_code == 404


def test_list_experiments():
    response = client.get("/api/v1/experiments")
    assert response.status_code == 200
    data = response.json()
    assert "experiments" in data
    assert "total" in data


def test_get_experiment_not_found():
    response = client.get("/api/v1/experiments/nonexistent-id")
    assert response.status_code == 404


def test_cancel_experiment():
    ds_id = _create_dataset()
    create_resp = client.post(
        "/api/v1/experiments",
        json={
            "name": "test-cancel-exp",
            "base_model": "test-model",
            "dataset": ds_id,
            "method": "qlora",
        },
    )
    exp_id = create_resp.json()["id"]

    # Wait a tiny bit for the thread to start
    time.sleep(0.1)

    cancel_resp = client.post(f"/api/v1/experiments/{exp_id}/cancel")
    assert cancel_resp.status_code == 200
    assert cancel_resp.json()["status"] == "cancelled"


def test_cancel_experiment_not_found():
    response = client.post("/api/v1/experiments/nonexistent/cancel")
    assert response.status_code == 404
