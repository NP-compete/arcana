"""Comprehensive tests for the Arcana Annotate service."""
from __future__ import annotations

import os

os.environ["AUTH_MODE"] = "open"
os.environ["EMBEDDING_PROVIDER"] = "hash"

from fastapi.testclient import TestClient

from app.main import app, _annotations, _store_lock

client = TestClient(app)


def _clear_store() -> None:
    """Reset the in-memory annotation store between tests."""
    with _store_lock:
        _annotations.clear()


def _make_annotation(
    question: str = "What is Python?",
    original: str = "A snake",
    corrected: str = "A programming language",
    agent_id: str = "agent-1",
    user_id: str = "user-1",
    topic: str = "general",
) -> dict:
    return {
        "question": question,
        "original_answer": original,
        "corrected_answer": corrected,
        "agent_id": agent_id,
        "user_id": user_id,
        "topic": topic,
    }


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
# Submit annotation
# ---------------------------------------------------------------------------


def test_submit_annotation():
    _clear_store()
    payload = _make_annotation()
    response = client.post("/api/v1/annotations", json=payload)
    assert response.status_code == 201

    body = response.json()
    assert body["question"] == payload["question"]
    assert body["original"] == payload["original_answer"]
    assert body["corrected"] == payload["corrected_answer"]
    assert body["agent_id"] == payload["agent_id"]
    assert body["user_id"] == payload["user_id"]
    assert body["topic"] == "general"
    assert body["crystallized"] is False
    assert "id" in body
    assert isinstance(body["embedding"], list)
    assert len(body["embedding"]) > 0


def test_submit_annotation_missing_fields():
    _clear_store()
    # Missing required 'question' field
    payload = {
        "original_answer": "A snake",
        "corrected_answer": "A programming language",
        "agent_id": "agent-1",
        "user_id": "user-1",
    }
    response = client.post("/api/v1/annotations", json=payload)
    assert response.status_code == 422


# ---------------------------------------------------------------------------
# List annotations
# ---------------------------------------------------------------------------


def test_list_annotations():
    _clear_store()
    # Submit several annotations
    for i in range(5):
        payload = _make_annotation(question=f"Question {i}", agent_id=f"agent-{i % 2}")
        resp = client.post("/api/v1/annotations", json=payload)
        assert resp.status_code == 201

    response = client.get("/api/v1/annotations")
    assert response.status_code == 200
    body = response.json()
    assert body["total"] == 5
    assert len(body["annotations"]) == 5


def test_list_filter_by_agent():
    _clear_store()
    client.post("/api/v1/annotations", json=_make_annotation(agent_id="alpha"))
    client.post("/api/v1/annotations", json=_make_annotation(agent_id="alpha"))
    client.post("/api/v1/annotations", json=_make_annotation(agent_id="beta"))

    response = client.get("/api/v1/annotations", params={"agent": "alpha"})
    assert response.status_code == 200
    body = response.json()
    assert body["total"] == 2
    for ann in body["annotations"]:
        assert ann["agent_id"] == "alpha"


def test_list_filter_by_topic():
    _clear_store()
    client.post("/api/v1/annotations", json=_make_annotation(topic="python"))
    client.post("/api/v1/annotations", json=_make_annotation(topic="python"))
    client.post("/api/v1/annotations", json=_make_annotation(topic="rust"))

    response = client.get("/api/v1/annotations", params={"topic": "python"})
    assert response.status_code == 200
    body = response.json()
    assert body["total"] == 2
    for ann in body["annotations"]:
        assert ann["topic"] == "python"


# ---------------------------------------------------------------------------
# Search annotations
# ---------------------------------------------------------------------------


def test_search_annotations():
    _clear_store()
    # Submit annotations with distinct content so embeddings differ
    client.post(
        "/api/v1/annotations",
        json=_make_annotation(
            question="How to sort a list in Python?",
            corrected="Use sorted() or list.sort()",
        ),
    )
    client.post(
        "/api/v1/annotations",
        json=_make_annotation(
            question="How to reverse a list in Python?",
            corrected="Use reversed() or list.reverse()",
        ),
    )
    client.post(
        "/api/v1/annotations",
        json=_make_annotation(
            question="What is the capital of France?",
            corrected="Paris is the capital of France",
            topic="geography",
        ),
    )

    # Search with a low threshold to get results
    response = client.post(
        "/api/v1/annotations/search",
        json={"query": "sort a list", "similarity_threshold": 0.0},
    )
    assert response.status_code == 200
    body = response.json()
    assert body["total"] >= 1
    for match in body["matches"]:
        assert "annotation" in match
        assert "similarity" in match
        assert isinstance(match["similarity"], float)

    # Results should be sorted by similarity descending
    similarities = [m["similarity"] for m in body["matches"]]
    assert similarities == sorted(similarities, reverse=True)


def test_search_threshold():
    _clear_store()
    client.post(
        "/api/v1/annotations",
        json=_make_annotation(question="alpha beta gamma"),
    )
    client.post(
        "/api/v1/annotations",
        json=_make_annotation(question="delta epsilon zeta"),
    )

    # Very high threshold should return fewer (possibly zero) results
    high = client.post(
        "/api/v1/annotations/search",
        json={"query": "completely unrelated query xyz", "similarity_threshold": 0.99},
    )
    low = client.post(
        "/api/v1/annotations/search",
        json={"query": "completely unrelated query xyz", "similarity_threshold": 0.0},
    )
    assert high.status_code == 200
    assert low.status_code == 200
    assert high.json()["total"] <= low.json()["total"]


# ---------------------------------------------------------------------------
# Stats
# ---------------------------------------------------------------------------


def test_stats():
    _clear_store()
    client.post("/api/v1/annotations", json=_make_annotation(agent_id="a1", topic="t1"))
    client.post("/api/v1/annotations", json=_make_annotation(agent_id="a1", topic="t2"))
    client.post("/api/v1/annotations", json=_make_annotation(agent_id="a2", topic="t1"))

    response = client.get("/api/v1/annotations/stats")
    assert response.status_code == 200
    body = response.json()
    assert body["total"] == 3
    assert body["total_annotations"] == 3
    assert body["unique_topics"] == 2
    assert body["by_agent"]["a1"] == 2
    assert body["by_agent"]["a2"] == 1
    assert body["by_topic"]["t1"] == 2
    assert body["by_topic"]["t2"] == 1


# ---------------------------------------------------------------------------
# Crystallization candidates
# ---------------------------------------------------------------------------


def test_crystallization_candidates():
    _clear_store()
    # Submit 4 annotations for the same agent+topic (above threshold of 3)
    for i in range(4):
        client.post(
            "/api/v1/annotations",
            json=_make_annotation(
                question=f"Repeated question {i}",
                agent_id="bot-x",
                topic="devops",
            ),
        )
    # Submit 1 annotation for a different pair (below threshold)
    client.post(
        "/api/v1/annotations",
        json=_make_annotation(agent_id="bot-y", topic="security"),
    )

    response = client.get("/api/v1/annotations/stats")
    assert response.status_code == 200
    body = response.json()
    candidates = body["crystallization_candidates"]
    # Only (devops, bot-x) should be ready
    assert len(candidates) >= 1
    ready = [c for c in candidates if c["ready"]]
    assert any(c["topic"] == "devops" and c["agent_id"] == "bot-x" for c in ready)
    # bot-y/security should NOT appear in ready candidates
    assert not any(c["topic"] == "security" and c["agent_id"] == "bot-y" for c in ready)


# ---------------------------------------------------------------------------
# Crystallize
# ---------------------------------------------------------------------------


def test_crystallize():
    _clear_store()
    # Submit enough annotations to meet the default min_corrections=3
    for i in range(4):
        client.post(
            "/api/v1/annotations",
            json=_make_annotation(
                question=f"How do I deploy service {i}?",
                corrected=f"Use helm chart version {i}",
                agent_id="deploy-bot",
                topic="deployment",
            ),
        )

    response = client.post(
        "/api/v1/annotations/crystallize",
        json={"agent_id": "deploy-bot", "topic": "deployment", "min_corrections": 3},
    )
    assert response.status_code == 200
    body = response.json()
    assert body["total"] >= 1
    skill = body["crystallized_skills"][0]
    assert skill["topic"] == "deployment"
    assert skill["agent_id"] == "deploy-bot"
    assert skill["corrections_used"] == 4
    assert skill["status"] == "created"
    assert len(skill["examples"]) <= 5

    # Verify annotations are now marked as crystallized
    stats = client.get("/api/v1/annotations/stats").json()
    assert stats["crystallized"] == 4


def test_crystallize_no_candidates():
    _clear_store()
    # Submit fewer than the threshold
    client.post("/api/v1/annotations", json=_make_annotation(agent_id="lone-bot"))

    response = client.post(
        "/api/v1/annotations/crystallize",
        json={"min_corrections": 5},
    )
    assert response.status_code == 404
    assert "no candidates" in response.json()["detail"]
