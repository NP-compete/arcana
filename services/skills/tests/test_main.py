"""Comprehensive tests for Arcana Skills service."""
from __future__ import annotations

import uuid
from datetime import UTC, datetime, timedelta
from unittest.mock import patch

import pytest
from fastapi.testclient import TestClient

with patch.dict("os.environ", {"AUTH_MODE": "open", "EMBEDDING_PROVIDER": "hash"}):
    from app.main import app, skills_db

client = TestClient(app)


@pytest.fixture(autouse=True)
def _reset_store():
    """Clear all in-memory skill state between tests."""
    skills_db.clear()
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
# List skills
# ---------------------------------------------------------------------------


def test_list_skills_empty():
    """With no skills registered, list returns empty."""
    resp = client.get("/api/v1/skills")
    assert resp.status_code == 200
    data = resp.json()
    assert data["skills"] == []
    assert data["total"] == 0


def test_list_skills_after_creation():
    """List reflects skills created via the reactive endpoint."""
    client.post(
        "/api/v1/skills/create-reactive",
        json={"name": "skill-alpha", "description": "Alpha skill"},
    )
    client.post(
        "/api/v1/skills/create-reactive",
        json={"name": "skill-beta", "description": "Beta skill"},
    )
    resp = client.get("/api/v1/skills")
    assert resp.status_code == 200
    data = resp.json()
    assert data["total"] == 2
    names = {s["name"] for s in data["skills"]}
    assert names == {"skill-alpha", "skill-beta"}


# ---------------------------------------------------------------------------
# Create reactive skill
# ---------------------------------------------------------------------------


def test_create_reactive_skill():
    """POST a description and verify the skill is created with markdown."""
    resp = client.post(
        "/api/v1/skills/create-reactive",
        json={
            "name": "code-review",
            "description": "Reviews code for bugs and style issues",
            "capabilities": ["lint", "style-check"],
        },
    )
    assert resp.status_code == 200
    data = resp.json()
    assert data["name"] == "code-review"
    assert data["type"] == "functional"
    assert data["version"] == "1.0.0"
    assert data["quality_badge"] == "untested"
    assert data["source"] == "reactive"
    assert data["description"] == "Reviews code for bugs and style issues"
    # Verify SKILL.md was generated
    assert "skill_md" in data
    assert "# code-review" in data["skill_md"]
    assert "lint" in data["skill_md"]
    assert "style-check" in data["skill_md"]


def test_create_reactive_skill_missing_name():
    """Creating a skill without a name returns 400."""
    resp = client.post(
        "/api/v1/skills/create-reactive",
        json={"name": "", "description": "some desc"},
    )
    assert resp.status_code == 400
    assert "name" in resp.json()["detail"].lower()


def test_create_reactive_skill_missing_description():
    """Creating a skill without a description returns 400."""
    resp = client.post(
        "/api/v1/skills/create-reactive",
        json={"name": "no-desc"},
    )
    assert resp.status_code == 400
    assert "description" in resp.json()["detail"].lower()


def test_create_reactive_skill_upsert():
    """Creating a skill with an existing name updates it (upsert behavior)."""
    client.post(
        "/api/v1/skills/create-reactive",
        json={"name": "updatable", "description": "Version 1"},
    )
    client.post(
        "/api/v1/skills/create-reactive",
        json={"name": "updatable", "description": "Version 2"},
    )
    resp = client.get("/api/v1/skills/updatable")
    assert resp.status_code == 200
    assert resp.json()["description"] == "Version 2"


# ---------------------------------------------------------------------------
# Get skill
# ---------------------------------------------------------------------------


def test_get_skill():
    """Create a skill, then retrieve it by name."""
    client.post(
        "/api/v1/skills/create-reactive",
        json={"name": "lookup-skill", "description": "For lookup testing"},
    )
    resp = client.get("/api/v1/skills/lookup-skill")
    assert resp.status_code == 200
    data = resp.json()
    assert data["name"] == "lookup-skill"
    assert data["description"] == "For lookup testing"


def test_get_skill_not_found():
    """GET for a non-existent skill returns 404."""
    resp = client.get("/api/v1/skills/nonexistent-skill")
    assert resp.status_code == 404
    assert "not found" in resp.json()["detail"].lower()


# ---------------------------------------------------------------------------
# Marketplace
# ---------------------------------------------------------------------------


def test_marketplace_listing():
    """Marketplace returns registered skills."""
    client.post(
        "/api/v1/skills/create-reactive",
        json={"name": "market-skill", "description": "A marketplace skill"},
    )
    resp = client.get("/api/v1/marketplace")
    assert resp.status_code == 200
    data = resp.json()
    assert data["total"] >= 1
    names = [item["name"] for item in data["items"]]
    assert "market-skill" in names


def test_marketplace_empty():
    """Empty marketplace returns zero items."""
    resp = client.get("/api/v1/marketplace")
    assert resp.status_code == 200
    assert resp.json()["total"] == 0
    assert resp.json()["items"] == []


def test_marketplace_filter_by_type():
    """Marketplace type filter works."""
    client.post(
        "/api/v1/skills/create-reactive",
        json={"name": "func-skill", "description": "Functional"},
    )
    # Reactive skills have type="functional"
    resp = client.get("/api/v1/marketplace", params={"type": "functional"})
    assert resp.status_code == 200
    assert resp.json()["total"] >= 1

    # Filtering by a non-matching type returns empty
    resp = client.get("/api/v1/marketplace", params={"type": "atomic"})
    assert resp.status_code == 200
    assert resp.json()["total"] == 0


def test_marketplace_filter_by_category():
    """Marketplace category filter works (default category is 'general')."""
    client.post(
        "/api/v1/skills/create-reactive",
        json={"name": "cat-skill", "description": "Categorized skill"},
    )
    resp = client.get("/api/v1/marketplace", params={"category": "general"})
    assert resp.status_code == 200
    assert resp.json()["total"] >= 1

    resp = client.get("/api/v1/marketplace", params={"category": "specialized"})
    assert resp.status_code == 200
    assert resp.json()["total"] == 0


def test_marketplace_search():
    """Marketplace text search filters by name and description."""
    client.post(
        "/api/v1/skills/create-reactive",
        json={"name": "python-linter", "description": "Lints Python code for style"},
    )
    client.post(
        "/api/v1/skills/create-reactive",
        json={"name": "java-formatter", "description": "Formats Java code"},
    )

    resp = client.get("/api/v1/marketplace", params={"q": "python"})
    assert resp.status_code == 200
    data = resp.json()
    assert data["total"] == 1
    assert data["items"][0]["name"] == "python-linter"


def test_marketplace_excludes_archived():
    """Archived skills should not appear in marketplace."""
    skills_db["archived-skill"] = {
        "name": "archived-skill",
        "description": "old skill",
        "type": "functional",
        "status": "archived",
    }
    resp = client.get("/api/v1/marketplace")
    assert resp.status_code == 200
    names = [item["name"] for item in resp.json()["items"]]
    assert "archived-skill" not in names


# ---------------------------------------------------------------------------
# Skill memory (experiential)
# ---------------------------------------------------------------------------


def test_skill_memory_append():
    """Append an experiential memory entry to a skill."""
    # First create the skill
    client.post(
        "/api/v1/skills/create-reactive",
        json={"name": "mem-skill", "description": "Skill with memory"},
    )
    resp = client.post(
        "/api/v1/skills/mem-skill/memory",
        json={"entry": "This skill works well with large files", "agent": "agent-1"},
    )
    assert resp.status_code == 200
    data = resp.json()
    assert data["entries"] == 1


def test_skill_memory_append_multiple():
    """Multiple memory entries accumulate."""
    client.post(
        "/api/v1/skills/create-reactive",
        json={"name": "multi-mem", "description": "Multi-memory skill"},
    )
    for i in range(3):
        client.post(
            "/api/v1/skills/multi-mem/memory",
            json={"entry": f"Memory entry {i}"},
        )
    resp = client.get("/api/v1/skills/multi-mem/memory")
    assert resp.status_code == 200
    assert len(resp.json()["memory"]) == 3


def test_skill_memory_append_empty_entry():
    """Appending an empty entry should fail with 400."""
    client.post(
        "/api/v1/skills/create-reactive",
        json={"name": "empty-mem", "description": "Test empty entry"},
    )
    resp = client.post(
        "/api/v1/skills/empty-mem/memory",
        json={"entry": ""},
    )
    assert resp.status_code == 400


def test_skill_memory_append_not_found():
    """Appending memory to a non-existent skill returns 404."""
    resp = client.post(
        "/api/v1/skills/ghost-skill/memory",
        json={"entry": "some memory"},
    )
    assert resp.status_code == 404


def test_skill_memory_get():
    """Retrieve skill's experiential memory."""
    client.post(
        "/api/v1/skills/create-reactive",
        json={"name": "get-mem", "description": "Get memory test"},
    )
    client.post(
        "/api/v1/skills/get-mem/memory",
        json={"entry": "First observation", "agent": "observer"},
    )
    resp = client.get("/api/v1/skills/get-mem/memory")
    assert resp.status_code == 200
    data = resp.json()
    assert len(data["memory"]) == 1
    assert data["memory"][0]["entry"] == "First observation"
    assert data["memory"][0]["agent"] == "observer"
    assert "timestamp" in data["memory"][0]


def test_skill_memory_get_not_found():
    """GET memory for a non-existent skill returns 404."""
    resp = client.get("/api/v1/skills/ghost-skill/memory")
    assert resp.status_code == 404


# ---------------------------------------------------------------------------
# Merge skills
# ---------------------------------------------------------------------------


def test_merge_skills():
    """Merge two skills into one."""
    client.post(
        "/api/v1/skills/create-reactive",
        json={"name": "skill-x", "description": "Does X"},
    )
    client.post(
        "/api/v1/skills/create-reactive",
        json={"name": "skill-y", "description": "Does Y"},
    )
    resp = client.post(
        "/api/v1/skills/merge",
        json={"skill_a": "skill-x", "skill_b": "skill-y", "merged_name": "skill-xy"},
    )
    assert resp.status_code == 200
    data = resp.json()
    assert data["name"] == "skill-xy"
    assert "Does X" in data["description"]
    assert "Does Y" in data["description"]
    assert data["source"] == "merge"
    assert data["merged_from"] == ["skill-x", "skill-y"]

    # Verify merged skill is retrievable
    get_resp = client.get("/api/v1/skills/skill-xy")
    assert get_resp.status_code == 200


def test_merge_skills_default_name():
    """Merging without merged_name uses default naming."""
    client.post("/api/v1/skills/create-reactive", json={"name": "a1", "description": "A"})
    client.post("/api/v1/skills/create-reactive", json={"name": "b1", "description": "B"})
    resp = client.post(
        "/api/v1/skills/merge",
        json={"skill_a": "a1", "skill_b": "b1"},
    )
    assert resp.status_code == 200
    assert resp.json()["name"] == "a1-merged"


def test_merge_skills_missing_params():
    """Merge with missing skill names returns 400."""
    resp = client.post(
        "/api/v1/skills/merge",
        json={"skill_a": "", "skill_b": "something"},
    )
    assert resp.status_code == 400


def test_merge_skills_not_found():
    """Merge with non-existent skills returns 404."""
    client.post("/api/v1/skills/create-reactive", json={"name": "exists", "description": "I exist"})
    resp = client.post(
        "/api/v1/skills/merge",
        json={"skill_a": "exists", "skill_b": "does-not-exist"},
    )
    assert resp.status_code == 404


# ---------------------------------------------------------------------------
# Prune skills
# ---------------------------------------------------------------------------


def test_prune_skills():
    """Archive skills that have not been used in 90+ days."""
    old_time = (datetime.now(UTC) - timedelta(days=100)).isoformat()
    skills_db["old-skill"] = {
        "name": "old-skill",
        "description": "Very old",
        "type": "functional",
        "status": "active",
        "last_used_at": old_time,
    }
    skills_db["new-skill"] = {
        "name": "new-skill",
        "description": "Recently used",
        "type": "functional",
        "status": "active",
        "last_used_at": datetime.now(UTC).isoformat(),
    }

    resp = client.post("/api/v1/skills/prune", json={"unused_days": 90})
    assert resp.status_code == 200
    data = resp.json()
    assert "old-skill" in data["pruned"]
    assert "new-skill" not in data["pruned"]
    assert data["count"] == 1

    # Old skill should be removed from cache
    assert "old-skill" not in skills_db
    # New skill should remain
    assert "new-skill" in skills_db


def test_prune_skills_none_to_prune():
    """Prune with no old skills returns empty list."""
    resp = client.post("/api/v1/skills/prune", json={"unused_days": 90})
    assert resp.status_code == 200
    data = resp.json()
    assert data["pruned"] == []
    assert data["count"] == 0


def test_prune_skills_custom_days():
    """Custom unused_days threshold is respected."""
    recent_time = (datetime.now(UTC) - timedelta(days=5)).isoformat()
    skills_db["borderline"] = {
        "name": "borderline",
        "description": "Borderline old",
        "type": "functional",
        "status": "active",
        "last_used_at": recent_time,
    }

    # With 3-day threshold, the skill should be pruned
    resp = client.post("/api/v1/skills/prune", json={"unused_days": 3})
    assert resp.status_code == 200
    assert "borderline" in resp.json()["pruned"]


def test_prune_skips_no_last_used():
    """Skills without last_used_at are not pruned."""
    skills_db["never-used"] = {
        "name": "never-used",
        "description": "Never used but not pruned",
        "type": "functional",
        "status": "active",
    }
    resp = client.post("/api/v1/skills/prune", json={"unused_days": 1})
    assert resp.status_code == 200
    assert "never-used" not in resp.json()["pruned"]


# ---------------------------------------------------------------------------
# Transfer skill
# ---------------------------------------------------------------------------


def test_transfer_skill():
    """Transfer a skill to another agent."""
    client.post(
        "/api/v1/skills/create-reactive",
        json={"name": "transferable", "description": "Can be transferred"},
    )
    resp = client.post(
        "/api/v1/skills/transferable/transfer",
        json={"target_agent": "agent-B", "pass_threshold": 0.9},
    )
    assert resp.status_code == 200
    data = resp.json()
    assert data["skill"] == "transferable"
    assert data["target_agent"] == "agent-B"
    assert data["pass_threshold"] == 0.9
    assert data["status"] == "pending_validation"
    assert data["transfer_badge"] == "transferred-untested"
    assert "transferred_at" in data


def test_transfer_skill_not_found():
    """Transferring a non-existent skill returns 404."""
    resp = client.post(
        "/api/v1/skills/ghost/transfer",
        json={"target_agent": "agent-C"},
    )
    assert resp.status_code == 404


def test_transfer_skill_missing_target():
    """Transfer without target_agent returns 400."""
    client.post(
        "/api/v1/skills/create-reactive",
        json={"name": "no-target", "description": "Needs a target"},
    )
    resp = client.post(
        "/api/v1/skills/no-target/transfer",
        json={"target_agent": ""},
    )
    assert resp.status_code == 400
    assert "target_agent" in resp.json()["detail"].lower()


# ---------------------------------------------------------------------------
# Crystallize skills
# ---------------------------------------------------------------------------


def test_crystallize():
    """Crystallize skills from annotation patterns with sufficient occurrences."""
    resp = client.post(
        "/api/v1/skills/crystallize",
        json={
            "patterns": [
                {"topic": "null checks", "occurrences": 15},
                {"topic": "error handling", "occurrences": 10},
                {"topic": "rare pattern", "occurrences": 3},  # below threshold
            ],
        },
    )
    assert resp.status_code == 200
    data = resp.json()
    assert data["count"] == 2  # only patterns with >= 10 occurrences
    assert "crystallized-null-checks" in data["crystallized"]
    assert "crystallized-error-handling" in data["crystallized"]
    assert "crystallized-rare-pattern" not in data["crystallized"]

    # Verify the crystallized skills are in the registry
    for name in data["crystallized"]:
        assert name in skills_db
        assert skills_db[name]["source"] == "crystallization"
        assert skills_db[name]["type"] == "atomic"


def test_crystallize_no_patterns():
    """Crystallize with empty patterns creates nothing."""
    resp = client.post(
        "/api/v1/skills/crystallize",
        json={"patterns": []},
    )
    assert resp.status_code == 200
    assert resp.json()["count"] == 0
    assert resp.json()["crystallized"] == []


def test_crystallize_below_threshold():
    """Patterns with < 10 occurrences are not crystallized."""
    resp = client.post(
        "/api/v1/skills/crystallize",
        json={
            "patterns": [
                {"topic": "minor", "occurrences": 9},
                {"topic": "tiny", "occurrences": 1},
            ],
        },
    )
    assert resp.status_code == 200
    assert resp.json()["count"] == 0


# ---------------------------------------------------------------------------
# Request-ID middleware
# ---------------------------------------------------------------------------


def test_request_id_propagated():
    """X-Request-Id header is echoed back."""
    rid = str(uuid.uuid4())
    resp = client.get("/healthz", headers={"x-request-id": rid})
    assert resp.headers["x-request-id"] == rid


def test_request_id_generated():
    """A request-id is generated when none is provided."""
    resp = client.get("/healthz")
    assert "x-request-id" in resp.headers
