from __future__ import annotations

from unittest.mock import AsyncMock, MagicMock, patch

import pytest
from fastapi.testclient import TestClient

from app.main import ModelRouter, _models, _store_lock, app

client = TestClient(app)


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _register_model(
    name: str = "test-model",
    framework: str = "transformers",
    base_model: str = "claude-sonnet-4-20250514",
    task: str = "text-generation",
    serving_config: dict | None = None,
) -> dict:
    """Register a model and return the response JSON."""
    # Clear any existing model with that name
    with _store_lock:
        _models.pop(name, None)

    return client.post(
        "/api/v1/models",
        json={
            "name": name,
            "framework": framework,
            "base_model": base_model,
            "task": task,
            "serving_config": serving_config or {"provider": "anthropic", "model_id": base_model},
        },
    ).json()


# ---------------------------------------------------------------------------
# Health probes
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
# Model registration
# ---------------------------------------------------------------------------


def test_register_model():
    with _store_lock:
        _models.pop("reg-test", None)

    resp = client.post(
        "/api/v1/models",
        json={
            "name": "reg-test",
            "framework": "transformers",
            "base_model": "claude-sonnet-4-20250514",
            "task": "text-generation",
            "serving_config": {"provider": "anthropic"},
        },
    )
    assert resp.status_code == 201
    data = resp.json()
    assert data["name"] == "reg-test"
    assert data["governance"]["approved"] is False

    # Cleanup
    with _store_lock:
        _models.pop("reg-test", None)


def test_register_duplicate_model():
    _register_model(name="dup-test")
    resp = client.post(
        "/api/v1/models",
        json={
            "name": "dup-test",
            "framework": "transformers",
            "base_model": "test",
            "task": "text-generation",
        },
    )
    assert resp.status_code == 409

    # Cleanup
    with _store_lock:
        _models.pop("dup-test", None)


# ---------------------------------------------------------------------------
# ModelRouter unit tests
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_model_router_anthropic():
    """ModelRouter correctly calls the Anthropic API."""
    router = ModelRouter()

    mock_response = MagicMock()
    mock_response.content = [MagicMock(text="Hello, world!")]
    mock_response.usage = MagicMock(input_tokens=10, output_tokens=5)

    mock_client = AsyncMock()
    mock_client.messages.create = AsyncMock(return_value=mock_response)

    with patch("app.main._anthropic_mod") as mock_mod:
        mock_mod.AsyncAnthropic.return_value = mock_client
        router._anthropic_client = None  # Force re-creation

        output, in_tok, out_tok, cost = await router.predict(
            "test-model",
            {"provider": "anthropic", "model_id": "claude-sonnet-4-20250514"},
            "Hello",
        )

    assert output == "Hello, world!"
    assert in_tok == 10
    assert out_tok == 5
    assert cost > 0


@pytest.mark.asyncio
async def test_model_router_anthropic_not_installed():
    """ModelRouter raises ImportError when anthropic is not installed."""
    router = ModelRouter()
    router._anthropic_client = None

    with patch("app.main._anthropic_mod", None):
        with pytest.raises(ImportError, match="anthropic package is required"):
            await router.predict(
                "test-model",
                {"provider": "anthropic", "model_id": "claude-sonnet-4-20250514"},
                "Hello",
            )


@pytest.mark.asyncio
async def test_model_router_openai_compat():
    """ModelRouter correctly calls OpenAI-compatible APIs."""
    router = ModelRouter()

    mock_resp = MagicMock()
    mock_resp.json.return_value = {
        "choices": [{"message": {"content": "Generated text"}}],
        "usage": {"prompt_tokens": 15, "completion_tokens": 8},
    }
    mock_resp.raise_for_status = MagicMock()

    mock_http = AsyncMock()
    mock_http.post = AsyncMock(return_value=mock_resp)
    router._http_client = mock_http

    output, in_tok, out_tok, cost = await router.predict(
        "gpt-4o-mini",
        {"provider": "openai", "model_id": "gpt-4o-mini", "base_url": "https://api.openai.com/v1"},
        "Hello",
    )

    assert output == "Generated text"
    assert in_tok == 15
    assert out_tok == 8
    assert cost > 0


@pytest.mark.asyncio
async def test_model_router_vllm_provider():
    """ModelRouter routes vllm provider to OpenAI-compatible endpoint."""
    router = ModelRouter()

    mock_resp = MagicMock()
    mock_resp.json.return_value = {
        "choices": [{"message": {"content": "vLLM output"}}],
        "usage": {"prompt_tokens": 5, "completion_tokens": 3},
    }
    mock_resp.raise_for_status = MagicMock()

    mock_http = AsyncMock()
    mock_http.post = AsyncMock(return_value=mock_resp)
    router._http_client = mock_http

    output, in_tok, out_tok, cost = await router.predict(
        "llama-model",
        {"provider": "vllm", "model_id": "llama-3-8b", "base_url": "http://vllm:8000/v1"},
        "test prompt",
    )

    assert output == "vLLM output"
    assert in_tok == 5
    assert out_tok == 3


@pytest.mark.asyncio
async def test_model_router_unsupported_provider():
    """ModelRouter raises ValueError for unsupported providers."""
    router = ModelRouter()

    with pytest.raises(ValueError, match="unsupported provider"):
        await router.predict(
            "test-model",
            {"provider": "unknown_provider", "model_id": "test"},
            "Hello",
        )


@pytest.mark.asyncio
async def test_model_router_cost_calculation_anthropic():
    """Verify Anthropic cost calculation uses correct rates."""
    router = ModelRouter()

    mock_response = MagicMock()
    mock_response.content = [MagicMock(text="output")]
    mock_response.usage = MagicMock(input_tokens=1_000_000, output_tokens=1_000_000)

    mock_client = AsyncMock()
    mock_client.messages.create = AsyncMock(return_value=mock_response)

    with patch("app.main._anthropic_mod") as mock_mod:
        mock_mod.AsyncAnthropic.return_value = mock_client
        router._anthropic_client = None

        _, _, _, cost = await router.predict(
            "test",
            {"provider": "anthropic", "model_id": "claude-sonnet-4-20250514"},
            "test",
        )

    # 1M input * 3.0/1M + 1M output * 15.0/1M = 3.0 + 15.0 = 18.0
    assert abs(cost - 18.0) < 0.01


@pytest.mark.asyncio
async def test_model_router_openai_missing_usage():
    """OpenAI-compatible response without usage data defaults to 0 tokens."""
    router = ModelRouter()

    mock_resp = MagicMock()
    mock_resp.json.return_value = {
        "choices": [{"message": {"content": "output"}}],
        # No "usage" key
    }
    mock_resp.raise_for_status = MagicMock()

    mock_http = AsyncMock()
    mock_http.post = AsyncMock(return_value=mock_resp)
    router._http_client = mock_http

    output, in_tok, out_tok, cost = await router.predict(
        "model",
        {"provider": "openai", "model_id": "gpt-4o-mini", "base_url": "https://api.openai.com/v1"},
        "test",
    )

    assert output == "output"
    assert in_tok == 0
    assert out_tok == 0
    assert cost == 0.0


# ---------------------------------------------------------------------------
# Predict endpoint integration tests (mocked router)
# ---------------------------------------------------------------------------


def test_predict_model_not_found():
    resp = client.post(
        "/api/v1/models/nonexistent/predict",
        json={"input": "hello"},
    )
    assert resp.status_code == 404


def test_predict_success_with_mocked_router():
    """Predict endpoint returns structured response from real router."""
    _register_model(name="predict-test")

    with patch.object(
        ModelRouter,
        "predict",
        new_callable=AsyncMock,
        return_value=("Generated output", 20, 10, 0.000045),
    ):
        resp = client.post(
            "/api/v1/models/predict-test/predict",
            json={"input": "What is AI?"},
        )

    assert resp.status_code == 200
    data = resp.json()
    assert data["model"] == "predict-test"
    assert data["output"] == "Generated output"
    assert data["tokens"]["input"] == 20
    assert data["tokens"]["output"] == 10
    assert data["tokens"]["total"] == 30
    assert data["cost_usd"] == 0.000045
    assert "budget_remaining" in data

    # Cleanup
    with _store_lock:
        _models.pop("predict-test", None)


def test_predict_fallback_on_failure():
    """When primary model fails, predict falls back to fallback chain model."""
    _register_model(name="primary-model")
    _register_model(name="gpt-4o-mini", serving_config={"provider": "openai", "model_id": "gpt-4o-mini"})

    call_count = 0

    async def mock_predict(model_name, model_info, input_text):
        nonlocal call_count
        call_count += 1
        if model_name == "primary-model":
            raise ConnectionError("service unavailable")
        return ("fallback output", 5, 3, 0.000024)

    with patch.object(ModelRouter, "predict", side_effect=mock_predict):
        resp = client.post(
            "/api/v1/models/primary-model/predict",
            json={"input": "test"},
        )

    assert resp.status_code == 200
    data = resp.json()
    assert data["output"] == "fallback output"
    assert data["model"] == "gpt-4o-mini"

    # Cleanup
    with _store_lock:
        _models.pop("primary-model", None)
        _models.pop("gpt-4o-mini", None)


def test_predict_all_fallbacks_fail():
    """When all models fail, predict returns 502."""
    _register_model(name="failing-model")

    async def mock_predict(model_name, model_info, input_text):
        raise ConnectionError("all backends down")

    with patch.object(ModelRouter, "predict", side_effect=mock_predict):
        resp = client.post(
            "/api/v1/models/failing-model/predict",
            json={"input": "test"},
        )

    assert resp.status_code == 502
    assert "inference failed" in resp.json()["detail"]

    # Cleanup
    with _store_lock:
        _models.pop("failing-model", None)


def test_predict_dict_input():
    """Predict handles dict input by converting to string."""
    _register_model(name="dict-input-test")

    with patch.object(
        ModelRouter,
        "predict",
        new_callable=AsyncMock,
        return_value=("output", 10, 5, 0.00001),
    ):
        resp = client.post(
            "/api/v1/models/dict-input-test/predict",
            json={"input": {"key": "value"}},
        )

    assert resp.status_code == 200
    assert resp.json()["output"] == "output"

    # Cleanup
    with _store_lock:
        _models.pop("dict-input-test", None)


# ---------------------------------------------------------------------------
# Budget endpoint
# ---------------------------------------------------------------------------


def test_budget_endpoint():
    resp = client.get("/api/v1/budget")
    assert resp.status_code == 200
    data = resp.json()
    assert "tokens_used" in data
    assert "cost" in data
    assert "remaining" in data
    assert "budget_limit" in data


# ---------------------------------------------------------------------------
# Fallback configuration
# ---------------------------------------------------------------------------


def test_configure_fallback():
    resp = client.post(
        "/api/v1/budget/fallback",
        json={
            "models": ["model-a", "model-b"],
            "thresholds": {"cost_per_token": 0.00002},
        },
    )
    assert resp.status_code == 200
    data = resp.json()
    assert data["models"] == ["model-a", "model-b"]
