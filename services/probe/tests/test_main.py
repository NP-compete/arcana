from __future__ import annotations

import asyncio
from unittest.mock import AsyncMock, MagicMock, patch

import pytest
from fastapi.testclient import TestClient

from app.main import (
    EvalCase,
    _judge_deterministic,
    _judge_llm,
    _judge_script,
    _judge_trajectory,
    _run_judge,
    Judge,
    JudgeTier,
    app,
)

client = TestClient(app)


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
# _judge_deterministic
# ---------------------------------------------------------------------------


def test_judge_deterministic_exact_match():
    case = EvalCase(id="1", input="hello world", expected="hello world")
    score, detail = _judge_deterministic(case)
    assert score == 1.0


def test_judge_deterministic_substring_match():
    case = EvalCase(id="1", input="hello world, how are you?", expected="hello world")
    score, _ = _judge_deterministic(case)
    assert score == 1.0


def test_judge_deterministic_no_match():
    case = EvalCase(id="1", input="hello world", expected="something else")
    score, _ = _judge_deterministic(case)
    assert score == 0.0


def test_judge_deterministic_no_expected():
    case = EvalCase(id="1", input="hello world")
    score, detail = _judge_deterministic(case)
    assert score == 0.5
    assert "no expected" in detail


# ---------------------------------------------------------------------------
# _judge_script
# ---------------------------------------------------------------------------


def test_judge_script_valid_input():
    case = EvalCase(id="1", input="valid input")
    score, _ = _judge_script(case)
    assert score == 1.0


def test_judge_script_error_input():
    case = EvalCase(id="1", input="ERROR: something broke")
    score, _ = _judge_script(case)
    assert score == 0.0


def test_judge_script_empty_input():
    case = EvalCase(id="1", input="")
    score, _ = _judge_script(case)
    assert score == 0.0


# ---------------------------------------------------------------------------
# _judge_llm — async, tested with mocked LLM backends
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_judge_llm_anthropic_success():
    """LLM judge returns valid score from Anthropic provider."""
    case = EvalCase(
        id="1",
        input="What is 2+2?",
        expected="4",
        metadata={"actual": "The answer is 4."},
    )

    mock_response = MagicMock()
    mock_response.content = [MagicMock(text="0.95")]

    mock_client = AsyncMock()
    mock_client.messages.create = AsyncMock(return_value=mock_response)

    with patch("app.main.anthropic") as mock_anthropic:
        mock_anthropic.AsyncAnthropic.return_value = mock_client
        score, detail = await _judge_llm(case)

    assert score == 0.95
    assert detail == "llm grading"


@pytest.mark.asyncio
async def test_judge_llm_anthropic_clamps_score():
    """Score from LLM is clamped to 0.0-1.0 range."""
    case = EvalCase(
        id="1",
        input="test",
        expected="answer",
        metadata={"actual": "response"},
    )

    mock_response = MagicMock()
    mock_response.content = [MagicMock(text="1.5")]

    mock_client = AsyncMock()
    mock_client.messages.create = AsyncMock(return_value=mock_response)

    with patch("app.main.anthropic") as mock_anthropic:
        mock_anthropic.AsyncAnthropic.return_value = mock_client
        score, _ = await _judge_llm(case)

    assert score == 1.0


@pytest.mark.asyncio
async def test_judge_llm_anthropic_negative_score():
    """Negative score from LLM is clamped to 0.0."""
    case = EvalCase(
        id="1",
        input="test",
        expected="answer",
        metadata={"actual": "response"},
    )

    mock_response = MagicMock()
    mock_response.content = [MagicMock(text="-0.5")]

    mock_client = AsyncMock()
    mock_client.messages.create = AsyncMock(return_value=mock_response)

    with patch("app.main.anthropic") as mock_anthropic:
        mock_anthropic.AsyncAnthropic.return_value = mock_client
        score, _ = await _judge_llm(case)

    assert score == 0.0


@pytest.mark.asyncio
async def test_judge_llm_openai_compat_success():
    """LLM judge with OpenAI-compatible provider returns valid score."""
    case = EvalCase(
        id="1",
        input="What is Python?",
        expected="A programming language",
        metadata={"actual": "Python is a programming language."},
    )

    mock_resp = MagicMock()
    mock_resp.json.return_value = {
        "choices": [{"message": {"content": "0.85"}}],
    }
    mock_resp.raise_for_status = MagicMock()

    mock_http = AsyncMock()
    mock_http.post = AsyncMock(return_value=mock_resp)
    mock_http.__aenter__ = AsyncMock(return_value=mock_http)
    mock_http.__aexit__ = AsyncMock(return_value=False)

    with (
        patch.dict("os.environ", {"JUDGE_LLM_PROVIDER": "openai"}),
        patch("httpx.AsyncClient", return_value=mock_http),
    ):
        score, detail = await _judge_llm(case)

    assert score == 0.85
    assert detail == "llm grading"


@pytest.mark.asyncio
async def test_judge_llm_fallback_no_actual():
    """Fallback returns 0.0 when there is no actual output."""
    case = EvalCase(
        id="1",
        input="test",
        expected="answer",
        metadata={},  # no "actual" key
    )

    with patch("app.main.anthropic") as mock_anthropic:
        mock_anthropic.AsyncAnthropic.side_effect = Exception("connection refused")
        score, detail = await _judge_llm(case)

    assert score == 0.0
    assert "fallback" in detail


@pytest.mark.asyncio
async def test_judge_llm_fallback_substring_match():
    """Fallback returns 0.8 when expected is substring of actual."""
    case = EvalCase(
        id="1",
        input="test",
        expected="answer",
        metadata={"actual": "The answer is correct."},
    )

    with patch("app.main.anthropic") as mock_anthropic:
        mock_anthropic.AsyncAnthropic.side_effect = Exception("connection refused")
        score, detail = await _judge_llm(case)

    assert score == 0.8
    assert "substring match" in detail


@pytest.mark.asyncio
async def test_judge_llm_fallback_length_heuristic():
    """Fallback returns length-based heuristic when no match."""
    case = EvalCase(
        id="1",
        input="test",
        expected="expected output",
        metadata={"actual": "something completely different and long enough"},
    )

    with patch("app.main.anthropic") as mock_anthropic:
        mock_anthropic.AsyncAnthropic.side_effect = Exception("connection refused")
        score, detail = await _judge_llm(case)

    assert 0.0 <= score <= 1.0
    assert "length heuristic" in detail


@pytest.mark.asyncio
async def test_judge_llm_anthropic_not_installed():
    """Fallback when anthropic package is not installed."""
    case = EvalCase(
        id="1",
        input="test",
        expected="answer",
        metadata={"actual": "some output"},
    )

    with patch("app.main.anthropic", None):
        score, detail = await _judge_llm(case)

    # Should hit fallback because ImportError is caught
    assert 0.0 <= score <= 1.0
    assert "fallback" in detail


# ---------------------------------------------------------------------------
# _judge_trajectory
# ---------------------------------------------------------------------------


def test_judge_trajectory_few_steps_with_output():
    """Few steps + correct output = high score."""
    case = EvalCase(
        id="1",
        input="task",
        expected="result",
        metadata={
            "actual": "The result is here.",
            "steps": [{"action": "search"}, {"action": "analyze"}],
            "max_steps": 10,
        },
    )
    score, detail = _judge_trajectory(case)
    # efficiency = 1.0 - (2/10) = 0.8, output = 1.0 (substring match)
    # score = 0.8*0.4 + 1.0*0.6 = 0.32 + 0.60 = 0.92
    assert abs(score - 0.92) < 0.01
    assert "trajectory" in detail


def test_judge_trajectory_max_steps_reached():
    """Max steps reached = efficiency 0."""
    case = EvalCase(
        id="1",
        input="task",
        expected="result",
        metadata={
            "actual": "the result",
            "steps": [{"action": f"step_{i}"} for i in range(10)],
            "max_steps": 10,
        },
    )
    score, _ = _judge_trajectory(case)
    # efficiency = 0.0, output = 1.0 (substring match)
    # score = 0.0*0.4 + 1.0*0.6 = 0.6
    assert abs(score - 0.6) < 0.01


def test_judge_trajectory_repetitive_actions():
    """Repetitive actions penalize efficiency."""
    case = EvalCase(
        id="1",
        input="task",
        expected="result",
        metadata={
            "actual": "the result",
            "steps": [
                {"action": "retry"},
                {"action": "retry"},
                {"action": "retry"},
                {"action": "retry"},
            ],
            "max_steps": 10,
        },
    )
    score, _ = _judge_trajectory(case)
    # step_count=4, efficiency = 1.0 - (4/10) = 0.6
    # unique_ratio = 1/4 = 0.25 < 0.5 -> efficiency *= 0.5 -> 0.3
    # output = 1.0 (substring match)
    # score = 0.3*0.4 + 1.0*0.6 = 0.12 + 0.60 = 0.72
    assert abs(score - 0.72) < 0.01


def test_judge_trajectory_no_output():
    """No actual output = output_score 0."""
    case = EvalCase(
        id="1",
        input="task",
        expected="result",
        metadata={
            "actual": "",
            "steps": [{"action": "search"}],
            "max_steps": 10,
        },
    )
    score, _ = _judge_trajectory(case)
    # efficiency = 0.9, output = 0.0 (empty actual)
    # score = 0.9*0.4 + 0.0*0.6 = 0.36
    assert abs(score - 0.36) < 0.01


def test_judge_trajectory_legacy_int_steps():
    """Legacy format: steps is an integer, not a list."""
    case = EvalCase(
        id="1",
        input="task",
        expected="result",
        metadata={
            "actual": "result here",
            "steps": 3,
            "max_steps": 10,
        },
    )
    score, _ = _judge_trajectory(case)
    # efficiency = 1.0 - (3/10) = 0.7, output = 1.0
    # score = 0.7*0.4 + 1.0*0.6 = 0.28 + 0.60 = 0.88
    assert abs(score - 0.88) < 0.01


def test_judge_trajectory_no_expected_with_output():
    """No expected but actual output provided = output_score 1.0 (non-empty)."""
    case = EvalCase(
        id="1",
        input="task",
        metadata={
            "actual": "some output",
            "steps": [],
            "max_steps": 10,
        },
    )
    score, _ = _judge_trajectory(case)
    # efficiency = 1.0 - (0/10) = 1.0, output = 1.0 (non-empty, no expected)
    # score = 1.0*0.4 + 1.0*0.6 = 1.0
    assert abs(score - 1.0) < 0.01


def test_judge_trajectory_partial_match():
    """Expected does not appear in actual = output_score 0.5."""
    case = EvalCase(
        id="1",
        input="task",
        expected="apples",
        metadata={
            "actual": "oranges and bananas",
            "steps": [{"action": "search"}],
            "max_steps": 10,
        },
    )
    score, _ = _judge_trajectory(case)
    # efficiency = 0.9, output = 0.5 (no substring match but both non-empty)
    # score = 0.9*0.4 + 0.5*0.6 = 0.36 + 0.30 = 0.66
    assert abs(score - 0.66) < 0.01


# ---------------------------------------------------------------------------
# _run_judge dispatching
# ---------------------------------------------------------------------------


def test_run_judge_deterministic():
    judge = Judge(name="test", tier=JudgeTier.DETERMINISTIC)
    case = EvalCase(id="1", input="hello", expected="hello")
    score, _ = _run_judge(judge, case)
    assert score == 1.0


def test_run_judge_script():
    judge = Judge(name="test", tier=JudgeTier.SCRIPT)
    case = EvalCase(id="1", input="valid input")
    score, _ = _run_judge(judge, case)
    assert score == 1.0


def test_run_judge_llm_dispatches_async():
    """_run_judge with LLM tier calls asyncio.run for the async judge."""
    judge = Judge(name="test", tier=JudgeTier.LLM)
    case = EvalCase(
        id="1",
        input="test",
        expected="answer",
        metadata={"actual": "the answer"},
    )

    mock_response = MagicMock()
    mock_response.content = [MagicMock(text="0.75")]

    mock_client = AsyncMock()
    mock_client.messages.create = AsyncMock(return_value=mock_response)

    with patch("app.main.anthropic") as mock_anthropic:
        mock_anthropic.AsyncAnthropic.return_value = mock_client
        score, _ = _run_judge(judge, case)

    assert score == 0.75


def test_run_judge_trajectory():
    judge = Judge(name="test", tier=JudgeTier.TRAJECTORY)
    case = EvalCase(
        id="1",
        input="task",
        expected="result",
        metadata={
            "actual": "result output",
            "steps": [{"action": "do"}],
            "max_steps": 10,
        },
    )
    score, _ = _run_judge(judge, case)
    assert score > 0.0


# ---------------------------------------------------------------------------
# Full eval run via API (mocked LLM)
# ---------------------------------------------------------------------------


def test_trigger_eval_and_get_result():
    """End-to-end eval run completes and returns results."""
    mock_response = MagicMock()
    mock_response.content = [MagicMock(text="0.9")]

    mock_client = AsyncMock()
    mock_client.messages.create = AsyncMock(return_value=mock_response)

    with patch("app.main.anthropic") as mock_anthropic:
        mock_anthropic.AsyncAnthropic.return_value = mock_client

        response = client.post(
            "/api/v1/eval/run",
            json={
                "skill_ref": "test-skill",
                "cases": [
                    {
                        "id": "case-1",
                        "input": "What is 2+2?",
                        "expected": "4",
                        "metadata": {"actual": "4"},
                    }
                ],
            },
        )

    assert response.status_code == 201
    data = response.json()
    run_id = data["run_id"]
    assert data["status"] in ("pending", "running", "completed")

    # Wait for the background thread to finish
    import time

    for _ in range(20):
        time.sleep(0.1)
        get_resp = client.get(f"/api/v1/eval/runs/{run_id}")
        if get_resp.json()["status"] == "completed":
            break

    result = client.get(f"/api/v1/eval/runs/{run_id}").json()
    assert result["status"] == "completed"
    assert len(result["results"]) == 1
    assert result["badge"] is not None
