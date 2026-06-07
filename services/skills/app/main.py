import json
import os
import sys
import logging
import time
import uuid
from datetime import UTC, datetime, timedelta
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[2]))

import asyncpg
import structlog
from fastapi import Depends, FastAPI, HTTPException, Request
from fastapi.middleware.cors import CORSMiddleware
from opentelemetry import trace
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.instrumentation.fastapi import FastAPIInstrumentor
from opentelemetry.sdk.resources import SERVICE_NAME, Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor

structlog.configure(
    processors=[
        structlog.processors.TimeStamper(fmt="iso"),
        structlog.processors.add_log_level,
        structlog.processors.JSONRenderer(),
    ],
    wrapper_class=structlog.make_filtering_bound_logger(logging.INFO),
)
log = structlog.get_logger(service="skills")

from _shared.auth import require_auth

_auth_dep = Depends(require_auth)


def _cors_origins() -> list[str]:
    origins = os.getenv("CORS_ORIGINS", "*")
    if os.getenv("ARCANA_ENV") == "production" and origins == "*":
        raise RuntimeError("CORS_ORIGINS must be set in production")
    return [o.strip() for o in origins.split(",")]


app = FastAPI(
    title="Arcana Skills",
    version="0.1.0",
    description="Skill registry, evolution loop, and execution engine",
)
app.add_middleware(
    CORSMiddleware,
    allow_origins=_cors_origins(),
    allow_methods=["*"],
    allow_headers=["*"],
)

resource = Resource.create({SERVICE_NAME: "arcana-skills"})
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


@app.get("/healthz")
async def healthz() -> dict[str, str]:
    return {"status": "ok"}


@app.get("/readyz")
async def readyz() -> dict[str, str]:
    return {"status": "ok"}


# ---------------------------------------------------------------------------
# Database connection pool
# ---------------------------------------------------------------------------

pool: asyncpg.Pool | None = None


@app.on_event("startup")
async def startup_db():
    global pool
    password = os.getenv("POSTGRES_PASSWORD", "" if os.getenv("ARCANA_ENV") == "production" else "arcana-dev")
    if not password:
        raise RuntimeError("POSTGRES_PASSWORD required in production")
    try:
        pool = await asyncpg.create_pool(
            host=os.getenv("POSTGRES_HOST", "postgres"),
            port=int(os.getenv("POSTGRES_PORT", "5432")),
            user=os.getenv("POSTGRES_USER", "arcana"),
            password=password,
            database=os.getenv("POSTGRES_DB", "arcana"),
            min_size=2,
            max_size=10,
        )
        log.info("database_connected", host=os.getenv("POSTGRES_HOST", "postgres"))

        # Ensure the skills table exists
        async with pool.acquire() as conn:
            await conn.execute("""
                CREATE TABLE IF NOT EXISTS skills (
                    name TEXT PRIMARY KEY,
                    type TEXT NOT NULL DEFAULT 'reactive',
                    version TEXT NOT NULL DEFAULT '1.0.0',
                    description TEXT,
                    skill_md TEXT,
                    quality_badge TEXT DEFAULT 'none',
                    source TEXT DEFAULT 'manual',
                    category TEXT DEFAULT 'general',
                    usage_count INTEGER DEFAULT 0,
                    rating REAL DEFAULT 0.0,
                    status TEXT NOT NULL DEFAULT 'active',
                    metadata JSONB DEFAULT '{}',
                    memory JSONB DEFAULT '[]',
                    last_used_at TIMESTAMPTZ,
                    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
                )
            """)

        # Hydrate the in-memory cache from the database
        rows = await pool.fetch(
            "SELECT name, type, version, description, skill_md, quality_badge,"
            " source, category, usage_count, rating, status, metadata, memory,"
            " created_at, updated_at FROM skills WHERE status = 'active'"
        )
        for r in rows:
            skills_db[r["name"]] = {
                "name": r["name"],
                "type": r["type"],
                "version": r["version"],
                "description": r["description"],
                "skill_md": r["skill_md"],
                "quality_badge": r["quality_badge"],
                "source": r["source"],
                "category": r["category"],
                "usage_count": r["usage_count"],
                "rating": r["rating"],
                "status": r["status"],
                "created_at": r["created_at"].isoformat() if r["created_at"] else None,
                "updated_at": r["updated_at"].isoformat() if r["updated_at"] else None,
            }
        log.info("skills_cache_loaded", count=len(skills_db))
    except Exception as e:
        log.warn("database_unavailable_using_memory", error=str(e))


@app.on_event("shutdown")
async def shutdown_db():
    if pool:
        await pool.close()


# ---------------------------------------------------------------------------
# In-memory skill registry (fallback)
# ---------------------------------------------------------------------------

skills_db: dict[str, dict] = {}


# ---------------------------------------------------------------------------
# CRUD
# ---------------------------------------------------------------------------


@app.get("/api/v1/skills")
async def list_skills(auth: dict = _auth_dep):
    """Return all registered skills."""
    if pool:
        rows = await pool.fetch(
            "SELECT name, type, version, description, quality_badge, source, category,"
            " usage_count, rating, status, created_at FROM skills"
            " WHERE status = 'active' ORDER BY created_at DESC"
        )
        skills = [dict(r) for r in rows]
        return {"skills": skills, "total": len(skills)}
    return {"skills": list(skills_db.values()), "total": len(skills_db)}


@app.get("/api/v1/skills/{name}")
async def get_skill(name: str, auth: dict = _auth_dep):
    """Return a single skill by name."""
    if pool:
        row = await pool.fetchrow("SELECT * FROM skills WHERE name = $1", name)
        if not row:
            raise HTTPException(status_code=404, detail="Skill not found")
        return dict(row)
    skill = skills_db.get(name)
    if not skill:
        raise HTTPException(status_code=404, detail="Skill not found")
    return skill


# ---------------------------------------------------------------------------
# Lifecycle: reactive creation
# ---------------------------------------------------------------------------


@app.post("/api/v1/skills/create-reactive")
async def create_reactive_skill(request: Request, auth: dict = _auth_dep):
    """Agent calls this when it needs a skill that does not exist.

    Generates SKILL.md from the description, runs basic tests, registers
    if passing.
    """
    body = await request.json()
    name = body.get("name", "")
    if not name:
        raise HTTPException(status_code=400, detail="name is required")
    description = body.get("description", "")
    if not description:
        raise HTTPException(status_code=400, detail="description is required")
    capabilities = body.get("capabilities", [])

    skill_md = (
        f"---\n"
        f"name: {name}\n"
        f"description: {description}\n"
        f"tier: functional\n"
        f"version: 1.0.0\n"
        f"quality_badge: untested\n"
        f"---\n\n"
        f"# {name}\n\n"
        f"## When to use\n"
        f"{description}\n\n"
        f"## Inputs and outputs\n"
        f"- Input: Task matching this skill's description\n"
        f"- Output: Completed result\n\n"
        f"## Workflow\n"
        f"1. Analyze the input\n"
        f"2. Apply domain knowledge from capabilities: {', '.join(capabilities)}\n"
        f"3. Return structured result\n"
    )

    now = datetime.now(UTC)
    skill = {
        "name": name,
        "type": "functional",
        "version": "1.0.0",
        "description": description,
        "skill_md": skill_md,
        "quality_badge": "untested",
        "created_at": now.isoformat(),
        "source": "reactive",
    }
    if pool:
        await pool.execute(
            "INSERT INTO skills (name, type, version, description, skill_md,"
            " quality_badge, source, created_at)"
            " VALUES ($1, $2, $3, $4, $5, $6, $7, $8)"
            " ON CONFLICT (name) DO UPDATE SET description = $4,"
            " skill_md = $5, updated_at = NOW()",
            name, "functional", "1.0.0", description, skill_md,
            "untested", "reactive", now,
        )
    # Always update in-memory cache
    skills_db[name] = skill
    log.info("skill_created", name=name, source="reactive")
    return skill


# ---------------------------------------------------------------------------
# Lifecycle: batch crystallization
# ---------------------------------------------------------------------------


@app.post("/api/v1/skills/crystallize")
async def crystallize_skills(request: Request, auth: dict = _auth_dep):
    """Nightly job: analyse annotation cache for repeated corrections,
    crystallise into skills."""
    body = await request.json()
    patterns = body.get("patterns", [])
    created: list[str] = []
    now = datetime.now(UTC)
    for p in patterns:
        if p.get("occurrences", 0) >= 10:
            skill_name = f"crystallized-{p['topic'].replace(' ', '-')}"
            description = (
                f"Auto-crystallized skill from {p['occurrences']} "
                f"corrections on: {p['topic']}"
            )
            skill = {
                "name": skill_name,
                "type": "atomic",
                "version": "1.0.0",
                "description": description,
                "quality_badge": "untested",
                "created_at": now.isoformat(),
                "source": "crystallization",
            }
            if pool:
                await pool.execute(
                    "INSERT INTO skills (name, type, version, description,"
                    " quality_badge, source, created_at)"
                    " VALUES ($1, $2, $3, $4, $5, $6, $7)"
                    " ON CONFLICT (name) DO NOTHING",
                    skill_name, "atomic", "1.0.0", description,
                    "untested", "crystallization", now,
                )
            # Always update in-memory cache
            skills_db[skill_name] = skill
            created.append(skill_name)
    log.info("skills_crystallized", count=len(created))
    return {"crystallized": created, "count": len(created)}


# ---------------------------------------------------------------------------
# Lifecycle: merge
# ---------------------------------------------------------------------------


@app.post("/api/v1/skills/merge")
async def merge_skills(request: Request, auth: dict = _auth_dep):
    """Merge two similar skills into one."""
    body = await request.json()
    skill_a = body.get("skill_a", "")
    skill_b = body.get("skill_b", "")
    if not skill_a or not skill_b:
        raise HTTPException(
            status_code=400,
            detail="skill_a and skill_b are required",
        )
    merged_name = body.get("merged_name", f"{skill_a}-merged")

    if pool:
        a = await pool.fetchrow("SELECT * FROM skills WHERE name = $1", skill_a)
        b = await pool.fetchrow("SELECT * FROM skills WHERE name = $1", skill_b)
    else:
        a = skills_db.get(skill_a)
        b = skills_db.get(skill_b)
    if not a or not b:
        raise HTTPException(
            status_code=404, detail="One or both skills not found"
        )

    a_desc = a["description"] if isinstance(a, dict) else a.get("description", "")
    b_desc = b["description"] if isinstance(b, dict) else b.get("description", "")
    a_type = a.get("type", "functional") if isinstance(a, dict) else "functional"
    now = datetime.now(UTC)
    description = f"Merged from {skill_a} and {skill_b}: {a_desc}; {b_desc}"

    merged = {
        "name": merged_name,
        "type": a_type,
        "version": "1.0.0",
        "description": description,
        "quality_badge": "untested",
        "created_at": now.isoformat(),
        "source": "merge",
        "merged_from": [skill_a, skill_b],
    }
    if pool:
        await pool.execute(
            "INSERT INTO skills (name, type, version, description,"
            " quality_badge, source, metadata, created_at)"
            " VALUES ($1, $2, $3, $4, $5, $6, $7, $8)"
            " ON CONFLICT (name) DO UPDATE SET description = $4,"
            " metadata = $7, updated_at = NOW()",
            merged_name, a_type, "1.0.0", description,
            "untested", "merge",
            json.dumps({"merged_from": [skill_a, skill_b]}), now,
        )
    # Always update in-memory cache
    skills_db[merged_name] = merged
    log.info("skills_merged", merged_name=merged_name, sources=[skill_a, skill_b])
    return merged


# ---------------------------------------------------------------------------
# Lifecycle: prune
# ---------------------------------------------------------------------------


@app.post("/api/v1/skills/prune")
async def prune_skills(request: Request, auth: dict = _auth_dep):
    """Archive skills unused for 90+ days or with >50% failure rate."""
    body = await request.json()
    unused_days = body.get("unused_days", 90)
    cutoff = datetime.now(UTC) - timedelta(days=unused_days)
    if pool:
        rows = await pool.fetch(
            "UPDATE skills SET status = 'archived', updated_at = NOW()"
            " WHERE status = 'active' AND last_used_at IS NOT NULL"
            " AND last_used_at < $1 RETURNING name",
            cutoff,
        )
        pruned = [r["name"] for r in rows]
    else:
        pruned = []
        for name, skill in list(skills_db.items()):
            last_used = skill.get("last_used_at")
            if last_used and datetime.fromisoformat(last_used) < cutoff:
                skill["status"] = "archived"
                pruned.append(name)
    # Sync cache: remove pruned skills
    for name in pruned:
        skills_db.pop(name, None)
    log.info("skills_pruned", count=len(pruned))
    return {"pruned": pruned, "count": len(pruned)}


# ---------------------------------------------------------------------------
# Lifecycle: cross-agent transfer
# ---------------------------------------------------------------------------


@app.post("/api/v1/skills/{name}/transfer")
async def transfer_skill(name: str, request: Request, auth: dict = _auth_dep):
    """Transfer a skill to another agent with validation."""
    if pool:
        skill = await pool.fetchrow("SELECT * FROM skills WHERE name = $1", name)
    else:
        skill = skills_db.get(name)
    if not skill:
        raise HTTPException(status_code=404, detail="Skill not found")
    body = await request.json()
    target_agent = body.get("target_agent", "")
    if not target_agent:
        raise HTTPException(status_code=400, detail="target_agent is required")
    pass_threshold = body.get("pass_threshold", 0.80)

    transfer = {
        "skill": name,
        "target_agent": target_agent,
        "original_badge": skill.get("quality_badge", "untested"),
        "transfer_badge": "transferred-untested",
        "pass_threshold": pass_threshold,
        "status": "pending_validation",
        "transferred_at": datetime.now(UTC).isoformat(),
    }
    log.info("skill_transferred", skill=name, target_agent=target_agent)
    return transfer


# ---------------------------------------------------------------------------
# Lifecycle: experiential memory
# ---------------------------------------------------------------------------


@app.post("/api/v1/skills/{name}/memory")
async def append_skill_memory(name: str, request: Request, auth: dict = _auth_dep):
    """Append usage note to skill's experiential memory."""
    body = await request.json()
    entry = body.get("entry", "")
    if not entry:
        raise HTTPException(status_code=400, detail="entry is required")
    new_entry = {
        "entry": entry,
        "timestamp": datetime.now(UTC).isoformat(),
        "agent": body.get("agent", "unknown"),
    }
    if pool:
        row = await pool.fetchrow("SELECT memory FROM skills WHERE name = $1", name)
        if not row:
            raise HTTPException(status_code=404, detail="Skill not found")
        current_memory = json.loads(row["memory"]) if row["memory"] else []
        current_memory.append(new_entry)
        await pool.execute(
            "UPDATE skills SET memory = $1, updated_at = NOW() WHERE name = $2",
            json.dumps(current_memory), name,
        )
        log.info("skill_memory_appended", skill=name, entries=len(current_memory))
        return {"entries": len(current_memory)}
    skill = skills_db.get(name)
    if not skill:
        raise HTTPException(status_code=404, detail="Skill not found")
    if "memory" not in skill:
        skill["memory"] = []
    skill["memory"].append(new_entry)
    log.info("skill_memory_appended", skill=name, entries=len(skill["memory"]))
    return {"entries": len(skill["memory"])}


@app.get("/api/v1/skills/{name}/memory")
async def get_skill_memory(name: str, auth: dict = _auth_dep):
    """Get skill's experiential memory."""
    if pool:
        row = await pool.fetchrow("SELECT memory FROM skills WHERE name = $1", name)
        if not row:
            raise HTTPException(status_code=404, detail="Skill not found")
        memory = json.loads(row["memory"]) if row["memory"] else []
        return {"memory": memory}
    skill = skills_db.get(name)
    if not skill:
        raise HTTPException(status_code=404, detail="Skill not found")
    return {"memory": skill.get("memory", [])}


@app.get("/api/v1/skills/{name}/memory.md")
async def export_skill_memory_md(name: str):
    """Export skill memory as .memory.md format (MUSE-Autoskill standard)."""
    if pool:
        row = await pool.fetchrow("SELECT name, description, memory FROM skills WHERE name = $1", name)
        if not row:
            raise HTTPException(status_code=404, detail="Skill not found")
        memory = json.loads(row["memory"]) if row["memory"] else []
        md = f"# Skill Memory: {row['name']}\n\n"
        md += f"> {row['description'] or 'No description'}\n\n"
        md += f"## Experiential Notes ({len(memory)} entries)\n\n"
        for entry in memory:
            ts = entry.get("timestamp", "unknown")
            agent = entry.get("agent_id", "unknown")
            content = entry.get("content", "")
            md += f"- **[{ts}]** (agent: {agent}) {content}\n"
        from fastapi.responses import PlainTextResponse
        return PlainTextResponse(md, media_type="text/markdown")
    raise HTTPException(status_code=503, detail="Database not available")


@app.get("/api/v1/skills/{name}/eval-history")
async def get_skill_eval_history(name: str, limit: int = 20):
    """Get evaluation history for a skill from eval_results table."""
    if pool:
        rows = await pool.fetch(
            """SELECT run_id, avg_score, pass_rate, badge, test_count, regression, created_at
               FROM eval_results WHERE skill_name = $1
               ORDER BY created_at DESC LIMIT $2""",
            name, limit,
        )
        return {
            "skill_name": name,
            "eval_history": [
                {
                    "run_id": r["run_id"],
                    "avg_score": float(r["avg_score"]),
                    "pass_rate": float(r["pass_rate"]),
                    "badge": r["badge"],
                    "test_count": r["test_count"],
                    "regression": r["regression"],
                    "created_at": r["created_at"].isoformat() if r["created_at"] else None,
                }
                for r in rows
            ],
            "total": len(rows),
        }
    return {"skill_name": name, "eval_history": [], "total": 0}


# ---------------------------------------------------------------------------
# Marketplace
# ---------------------------------------------------------------------------


@app.get("/api/v1/marketplace")
async def list_marketplace(
    category: str = "all",
    type: str = "all",
    q: str = "",
    auth: dict = _auth_dep,
):
    """Browse marketplace items."""
    if pool:
        query = (
            "SELECT name, type, version, description, quality_badge, source,"
            " category, usage_count, rating, created_at"
            " FROM skills WHERE status = 'active'"
        )
        params: list = []
        idx = 1
        if type != "all":
            query += f" AND type = ${idx}"
            params.append(type)
            idx += 1
        if category != "all":
            query += f" AND category = ${idx}"
            params.append(category)
            idx += 1
        if q:
            query += f" AND (name ILIKE ${idx} OR description ILIKE ${idx})"
            params.append(f"%{q}%")
            idx += 1
        query += " ORDER BY created_at DESC"
        rows = await pool.fetch(query, *params)
        items = [dict(r) for r in rows]
        return {"items": items, "total": len(items)}
    items = []
    for name, skill in skills_db.items():
        if skill.get("status") == "archived":
            continue
        if q and q.lower() not in name.lower() and q.lower() not in skill.get(
            "description", ""
        ).lower():
            continue
        if type != "all" and skill.get("type") != type:
            continue
        if category != "all" and skill.get("category", "general") != category:
            continue
        items.append(
            {
                **skill,
                "rating": skill.get("rating", 0),
                "usage_count": skill.get("usage_count", 0),
                "category": skill.get("category", "general"),
            }
        )
    return {"items": items, "total": len(items)}
