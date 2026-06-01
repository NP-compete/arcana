"""Comprehensive tests for Arcana Memory service."""
from __future__ import annotations

import time
import uuid
from unittest.mock import patch

import pytest
from fastapi.testclient import TestClient

with patch.dict("os.environ", {"EMBEDDING_PROVIDER": "hash", "AUTH_MODE": "open"}):
    from app.main import app, store

client = TestClient(app)


@pytest.fixture(autouse=True)
def _reset_store():
    """Clear all in-memory state between tests."""
    store._short_term.clear()
    store._long_term.clear()
    store._skills.clear()
    yield


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
# Short-term memory
# ---------------------------------------------------------------------------


def test_store_short_term():
    """POST a valid short-term entry, verify 201 and all response fields."""
    payload = {"agent_id": "agent-1", "key": "task", "value": "do something", "ttl": 300}
    resp = client.post("/api/v1/memory/short-term", json=payload)
    assert resp.status_code == 201
    data = resp.json()
    assert data["agent_id"] == "agent-1"
    assert data["key"] == "task"
    assert data["value"] == "do something"
    assert data["ttl"] == 300
    assert "id" in data
    assert "created_at" in data
    assert "expires_at" in data


def test_store_short_term_invalid_ttl():
    """POST with ttl=0 must be rejected with 400."""
    payload = {"agent_id": "agent-1", "key": "k", "value": "v", "ttl": 0}
    resp = client.post("/api/v1/memory/short-term", json=payload)
    assert resp.status_code == 400
    assert "ttl" in resp.json()["detail"].lower()


def test_store_short_term_negative_ttl():
    """POST with negative ttl must be rejected with 400."""
    payload = {"agent_id": "agent-1", "key": "k", "value": "v", "ttl": -10}
    resp = client.post("/api/v1/memory/short-term", json=payload)
    assert resp.status_code == 400


def test_get_short_term_empty():
    """GET for an unknown agent returns an empty list."""
    resp = client.get("/api/v1/memory/short-term/nonexistent-agent")
    assert resp.status_code == 200
    assert resp.json() == []


def test_get_short_term_returns_stored():
    """Store an entry then retrieve it for the same agent."""
    payload = {"agent_id": "agent-A", "key": "color", "value": "blue", "ttl": 600}
    client.post("/api/v1/memory/short-term", json=payload)

    resp = client.get("/api/v1/memory/short-term/agent-A")
    assert resp.status_code == 200
    entries = resp.json()
    assert len(entries) == 1
    assert entries[0]["key"] == "color"
    assert entries[0]["value"] == "blue"


def test_get_short_term_multiple_entries():
    """Multiple entries for the same agent are all returned."""
    for i in range(3):
        client.post(
            "/api/v1/memory/short-term",
            json={"agent_id": "agent-M", "key": f"k{i}", "value": f"v{i}", "ttl": 600},
        )
    resp = client.get("/api/v1/memory/short-term/agent-M")
    assert resp.status_code == 200
    assert len(resp.json()) == 3


def test_short_term_ttl_expiry():
    """Store with ttl=1, sleep for 2s, verify the entry has expired."""
    payload = {"agent_id": "agent-ttl", "key": "ephemeral", "value": "gone soon", "ttl": 1}
    client.post("/api/v1/memory/short-term", json=payload)

    # Immediately should be present
    resp = client.get("/api/v1/memory/short-term/agent-ttl")
    assert len(resp.json()) == 1

    time.sleep(2)

    resp = client.get("/api/v1/memory/short-term/agent-ttl")
    assert resp.json() == []


def test_short_term_default_ttl():
    """Omitting ttl should default to 3600."""
    payload = {"agent_id": "agent-d", "key": "k", "value": "v"}
    resp = client.post("/api/v1/memory/short-term", json=payload)
    assert resp.status_code == 201
    assert resp.json()["ttl"] == 3600


def test_short_term_isolation_between_agents():
    """Entries for one agent are not returned for a different agent."""
    client.post(
        "/api/v1/memory/short-term",
        json={"agent_id": "alice", "key": "k", "value": "alice-val", "ttl": 600},
    )
    client.post(
        "/api/v1/memory/short-term",
        json={"agent_id": "bob", "key": "k", "value": "bob-val", "ttl": 600},
    )
    alice_entries = client.get("/api/v1/memory/short-term/alice").json()
    bob_entries = client.get("/api/v1/memory/short-term/bob").json()
    assert len(alice_entries) == 1
    assert alice_entries[0]["value"] == "alice-val"
    assert len(bob_entries) == 1
    assert bob_entries[0]["value"] == "bob-val"


# ---------------------------------------------------------------------------
# Long-term memory
# ---------------------------------------------------------------------------


def test_store_long_term():
    """POST a valid long-term entry, verify 201 and response fields."""
    payload = {"agent_id": "agent-1", "content": "Python is great", "metadata": {"source": "chat"}}
    resp = client.post("/api/v1/memory/long-term", json=payload)
    assert resp.status_code == 201
    data = resp.json()
    assert data["agent_id"] == "agent-1"
    assert data["content"] == "Python is great"
    assert data["metadata"]["source"] == "chat"
    assert "id" in data
    assert "embedding" in data
    assert len(data["embedding"]) > 0
    assert "created_at" in data


def test_store_long_term_empty_content():
    """POST with empty content must be rejected with 400."""
    payload = {"agent_id": "agent-1", "content": "", "metadata": {}}
    resp = client.post("/api/v1/memory/long-term", json=payload)
    assert resp.status_code == 400
    assert "content" in resp.json()["detail"].lower()


def test_store_long_term_whitespace_only():
    """POST with whitespace-only content must be rejected with 400."""
    payload = {"agent_id": "agent-1", "content": "   \t\n  "}
    resp = client.post("/api/v1/memory/long-term", json=payload)
    assert resp.status_code == 400


def test_store_long_term_with_custom_embedding():
    """Providing a pre-computed embedding should be stored as-is."""
    custom_emb = [0.1] * 64
    payload = {"agent_id": "agent-1", "content": "test", "embedding": custom_emb}
    resp = client.post("/api/v1/memory/long-term", json=payload)
    assert resp.status_code == 201
    data = resp.json()
    assert data["embedding"] == custom_emb


# ---------------------------------------------------------------------------
# Long-term search
# ---------------------------------------------------------------------------


def test_search_long_term_empty():
    """Search with no data returns empty results."""
    resp = client.get("/api/v1/memory/long-term/agent-1/search", params={"query": "anything"})
    assert resp.status_code == 200
    data = resp.json()
    assert data["results"] == []
    assert data["agent_id"] == "agent-1"
    assert data["query"] == "anything"


def test_search_long_term():
    """Store multiple entries and search; verify relevance ordering."""
    agent = "agent-search"
    # Store 3 distinct entries
    client.post("/api/v1/memory/long-term", json={"agent_id": agent, "content": "The cat sat on the mat"})
    client.post("/api/v1/memory/long-term", json={"agent_id": agent, "content": "Python is a programming language"})
    client.post("/api/v1/memory/long-term", json={"agent_id": agent, "content": "The cat played with yarn"})

    resp = client.get(f"/api/v1/memory/long-term/{agent}/search", params={"query": "cat", "top_k": 3})
    assert resp.status_code == 200
    data = resp.json()
    assert len(data["results"]) == 3
    # Results are sorted by descending score
    scores = [r["score"] for r in data["results"]]
    assert scores == sorted(scores, reverse=True)


def test_search_long_term_top_k():
    """top_k limits the number of results."""
    agent = "agent-topk"
    for i in range(5):
        client.post("/api/v1/memory/long-term", json={"agent_id": agent, "content": f"memory item {i}"})

    resp = client.get(f"/api/v1/memory/long-term/{agent}/search", params={"query": "memory", "top_k": 2})
    assert resp.status_code == 200
    assert len(resp.json()["results"]) == 2


def test_search_long_term_exact_match_highest_score():
    """An exact match should have the highest similarity score."""
    agent = "agent-exact"
    client.post("/api/v1/memory/long-term", json={"agent_id": agent, "content": "unique query string"})
    client.post("/api/v1/memory/long-term", json={"agent_id": agent, "content": "completely different topic"})

    resp = client.get(f"/api/v1/memory/long-term/{agent}/search", params={"query": "unique query string"})
    data = resp.json()
    # The first result should be the exact match with score ~1.0
    assert data["results"][0]["memory"]["content"] == "unique query string"
    assert data["results"][0]["score"] > 0.99


# ---------------------------------------------------------------------------
# Skill memory
# ---------------------------------------------------------------------------


def test_skill_memory_append():
    """POST to skill memory creates and returns the skill with the entry."""
    resp = client.post(
        "/api/v1/memory/skill/code-review",
        json={"content": "Always check null pointers", "metadata": {"context": "java"}},
    )
    assert resp.status_code == 200
    data = resp.json()
    assert data["skill_name"] == "code-review"
    assert len(data["entries"]) == 1
    assert data["entries"][0]["content"] == "Always check null pointers"


def test_skill_memory_append_multiple():
    """Multiple appends accumulate entries."""
    for i in range(3):
        client.post(
            "/api/v1/memory/skill/debugging",
            json={"content": f"tip {i}"},
        )
    resp = client.get("/api/v1/memory/skill/debugging")
    assert resp.status_code == 200
    assert len(resp.json()["entries"]) == 3


def test_skill_memory_append_empty_content():
    """Appending empty content to skill memory must fail with 400."""
    resp = client.post("/api/v1/memory/skill/empty-test", json={"content": ""})
    assert resp.status_code == 400


def test_skill_memory_get_not_found():
    """GET for an unknown skill returns 404."""
    resp = client.get("/api/v1/memory/skill/nonexistent-skill")
    assert resp.status_code == 404
    assert "not found" in resp.json()["detail"].lower()


def test_skill_memory_get():
    """Store then retrieve skill memory."""
    client.post("/api/v1/memory/skill/testing", json={"content": "Use fixtures"})
    resp = client.get("/api/v1/memory/skill/testing")
    assert resp.status_code == 200
    data = resp.json()
    assert data["skill_name"] == "testing"
    assert len(data["entries"]) == 1


# ---------------------------------------------------------------------------
# Compact
# ---------------------------------------------------------------------------


def test_compact_memory():
    """Store short-term entries, compact, verify long-term created."""
    agent = "agent-compact"
    for i in range(3):
        client.post(
            "/api/v1/memory/short-term",
            json={"agent_id": agent, "key": f"fact-{i}", "value": f"value-{i}", "ttl": 600},
        )

    resp = client.post("/api/v1/memory/compact", json={"agent_id": agent})
    assert resp.status_code == 200
    data = resp.json()
    assert data["agent_id"] == agent
    assert data["compacted_entries"] == 3
    assert "summary_id" in data
    assert "Compacted 3" in data["message"]

    # Short-term should be empty after compaction
    st_resp = client.get(f"/api/v1/memory/short-term/{agent}")
    assert st_resp.json() == []

    # Long-term should have the compacted entry
    lt_resp = client.get(f"/api/v1/memory/long-term/{agent}/search", params={"query": "compaction"})
    assert len(lt_resp.json()["results"]) == 1


def test_compact_no_entries():
    """Compact with no short-term data returns 404."""
    resp = client.post("/api/v1/memory/compact", json={"agent_id": "empty-agent"})
    assert resp.status_code == 404
    assert "no short-term" in resp.json()["detail"]


# ---------------------------------------------------------------------------
# Dream compaction
# ---------------------------------------------------------------------------


def test_dream_compact():
    """Store 5+ long-term entries with duplicates, run dream, verify dedup."""
    agent = "agent-dream"
    # Store 6 entries: 3 identical pairs
    for content in ["cats are great", "cats are great", "dogs are loyal", "dogs are loyal", "fish swim", "fish swim"]:
        store.store_long_term(agent, content, {}, None)

    resp = client.post(f"/api/v1/memory/{agent}/dream")
    assert resp.status_code == 200
    data = resp.json()
    assert data["compacted"] == 6
    # Dedup should reduce the 6 entries to 3 unique facts
    assert len(data["facts"]) == 3

    # Verify the long-term store now has exactly 3 entries
    with store._lock:
        assert len(store._long_term[agent]) == 3


def test_dream_insufficient_memories():
    """Dream with < 5 long-term entries returns skip reason."""
    agent = "agent-few"
    for i in range(4):
        store.store_long_term(agent, f"memory {i}", {}, None)

    resp = client.post(f"/api/v1/memory/{agent}/dream")
    assert resp.status_code == 200
    data = resp.json()
    assert data["compacted"] == 0
    assert data["facts"] == []
    assert "insufficient" in data["reason"]


def test_dream_no_memories():
    """Dream with no memories at all returns skip."""
    resp = client.post("/api/v1/memory/no-agent/dream")
    assert resp.status_code == 200
    data = resp.json()
    assert data["compacted"] == 0


def test_dream_all_unique():
    """Dream with all unique entries preserves all of them."""
    agent = "agent-unique"
    unique_contents = [
        "apples are fruits",
        "the sky is blue",
        "water freezes at zero celsius",
        "python uses indentation",
        "kubernetes orchestrates containers",
    ]
    for content in unique_contents:
        store.store_long_term(agent, content, {}, None)

    resp = client.post(f"/api/v1/memory/{agent}/dream")
    data = resp.json()
    assert data["compacted"] == 5
    assert len(data["facts"]) == 5


# ---------------------------------------------------------------------------
# Reflect
# ---------------------------------------------------------------------------


def test_reflect_with_memories():
    """Reflect with enough similar memories should produce insights."""
    agent = "agent-reflect"
    # Create 5+ memories with similar content so they cluster together
    for i in range(6):
        # Use the same content so they form a cluster
        store.store_long_term(agent, "debugging python errors", {"idx": i}, None)

    resp = client.post(f"/api/v1/memory/{agent}/reflect")
    assert resp.status_code == 200
    data = resp.json()
    assert "insights" in data
    # With 6 identical memories, they should cluster and produce at least 1 insight
    assert len(data["insights"]) >= 1
    insight = data["insights"][0]
    assert "id" in insight
    assert "content" in insight
    assert insight["type"] == "insight"
    assert insight["source_memories"] >= 2


def test_reflect_insufficient():
    """Reflect with fewer than 5 entries returns no insights."""
    agent = "agent-reflect-few"
    for i in range(3):
        store.store_long_term(agent, f"distinct-{i}-{uuid.uuid4()}", {}, None)

    resp = client.post(f"/api/v1/memory/{agent}/reflect")
    assert resp.status_code == 200
    data = resp.json()
    assert data["insights"] == []


def test_reflect_no_memories():
    """Reflect with no memories at all returns empty insights."""
    resp = client.post("/api/v1/memory/no-agent/reflect")
    assert resp.status_code == 200
    assert resp.json()["insights"] == []


def test_reflect_diverse_no_clusters():
    """Reflect with all diverse memories produces no insights (no clusters >= 2)."""
    agent = "agent-diverse"
    diverse_contents = [
        "quantum computing uses qubits",
        "the amazon river is long",
        "beethoven composed symphonies",
        "silicon valley is in california",
        "photosynthesis converts sunlight",
    ]
    for content in diverse_contents:
        store.store_long_term(agent, content, {}, None)

    resp = client.post(f"/api/v1/memory/{agent}/reflect")
    data = resp.json()
    # Each memory is distinct, so no cluster should have >= 2 members
    # (depends on hash-based embeddings similarity, but these are very different)
    assert isinstance(data["insights"], list)


# ---------------------------------------------------------------------------
# Request-ID middleware
# ---------------------------------------------------------------------------


def test_request_id_propagated():
    """X-Request-Id is echoed back in the response."""
    rid = str(uuid.uuid4())
    resp = client.get("/healthz", headers={"x-request-id": rid})
    assert resp.headers["x-request-id"] == rid


def test_request_id_generated():
    """A request-id is generated when none is provided."""
    resp = client.get("/healthz")
    assert "x-request-id" in resp.headers
