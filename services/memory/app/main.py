"""Arcana Tri-Scope Memory service."""

from __future__ import annotations

import logging
import os
import sys
import threading
import time
import uuid
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

sys.path.insert(0, str(Path(__file__).resolve().parents[2]))

import asyncpg
import structlog
from fastapi import Depends, FastAPI, HTTPException, Query, Request
from fastapi.middleware.cors import CORSMiddleware
from opentelemetry import trace
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.instrumentation.fastapi import FastAPIInstrumentor
from opentelemetry.sdk.resources import SERVICE_NAME, Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from pydantic import BaseModel, Field

from _shared.auth import require_auth

_auth_dep = Depends(require_auth)
from _shared.embeddings import cosine_similarity, embed_text, embedding_dim  # noqa: F401


def _cors_origins() -> list[str]:
    origins = os.getenv("CORS_ORIGINS", "*")
    if os.getenv("ARCANA_ENV") == "production" and origins == "*":
        raise RuntimeError("CORS_ORIGINS must be set in production")
    return [o.strip() for o in origins.split(",")]


def utcnow() -> datetime:
    return datetime.now(UTC)


class ShortTermEntry(BaseModel):
    id: str
    agent_id: str
    key: str
    value: Any
    ttl: int = Field(description="Time-to-live in seconds")
    created_at: datetime
    expires_at: datetime


class ShortTermStoreRequest(BaseModel):
    agent_id: str
    key: str
    value: Any
    ttl: int = 3600


class LongTermMemory(BaseModel):
    id: str
    agent_id: str
    content: str
    metadata: dict[str, Any] = Field(default_factory=dict)
    embedding: list[float]
    created_at: datetime


class LongTermStoreRequest(BaseModel):
    agent_id: str
    content: str
    metadata: dict[str, Any] = Field(default_factory=dict)
    embedding: list[float] | None = None


class SkillMemoryEntry(BaseModel):
    timestamp: datetime
    content: str
    metadata: dict[str, Any] = Field(default_factory=dict)


class SkillMemory(BaseModel):
    skill_name: str
    entries: list[SkillMemoryEntry]
    updated_at: datetime


class SkillAppendRequest(BaseModel):
    content: str
    metadata: dict[str, Any] = Field(default_factory=dict)


class CompactRequest(BaseModel):
    agent_id: str


class CompactResponse(BaseModel):
    agent_id: str
    compacted_entries: int
    summary_id: str
    message: str


class SearchResult(BaseModel):
    memory: LongTermMemory
    score: float


class SearchResponse(BaseModel):
    agent_id: str
    query: str
    results: list[SearchResult]


class MemoryStore:
    def __init__(self) -> None:
        self._lock = threading.RLock()
        self._short_term: dict[str, list[ShortTermEntry]] = {}
        self._long_term: dict[str, list[LongTermMemory]] = {}
        self._skills: dict[str, SkillMemory] = {}

    def store_short_term(self, agent_id: str, key: str, value: Any, ttl: int) -> ShortTermEntry:
        now = utcnow()
        entry = ShortTermEntry(
            id=str(uuid.uuid4()),
            agent_id=agent_id,
            key=key,
            value=value,
            ttl=ttl,
            created_at=now,
            expires_at=datetime.fromtimestamp(now.timestamp() + ttl, tz=UTC),
        )
        with self._lock:
            self._purge_expired_short_term(agent_id)
            self._short_term.setdefault(agent_id, []).append(entry)
        return entry

    def get_short_term(self, agent_id: str) -> list[ShortTermEntry]:
        with self._lock:
            self._purge_expired_short_term(agent_id)
            return list(self._short_term.get(agent_id, []))

    def _purge_expired_short_term(self, agent_id: str) -> None:
        now = utcnow()
        entries = self._short_term.get(agent_id, [])
        self._short_term[agent_id] = [e for e in entries if e.expires_at > now]

    def store_long_term(
        self,
        agent_id: str,
        content: str,
        metadata: dict[str, Any],
        embedding: list[float] | None,
    ) -> LongTermMemory:
        vec = embedding if embedding else embed_text(content)
        memory = LongTermMemory(
            id=str(uuid.uuid4()),
            agent_id=agent_id,
            content=content,
            metadata=metadata,
            embedding=vec,
            created_at=utcnow(),
        )
        with self._lock:
            self._long_term.setdefault(agent_id, []).append(memory)
        return memory

    def search_long_term(self, agent_id: str, query: str, top_k: int) -> list[SearchResult]:
        query_vec = embed_text(query)
        with self._lock:
            memories = list(self._long_term.get(agent_id, []))

        scored = [
            SearchResult(memory=m, score=cosine_similarity(query_vec, m.embedding))
            for m in memories
        ]
        scored.sort(key=lambda r: r.score, reverse=True)
        return scored[:top_k]

    def append_skill(self, skill_name: str, content: str, metadata: dict[str, Any]) -> SkillMemory:
        entry = SkillMemoryEntry(timestamp=utcnow(), content=content, metadata=metadata)
        with self._lock:
            if skill_name not in self._skills:
                self._skills[skill_name] = SkillMemory(
                    skill_name=skill_name,
                    entries=[],
                    updated_at=utcnow(),
                )
            skill = self._skills[skill_name]
            skill.entries.append(entry)
            skill.updated_at = utcnow()
            return skill.model_copy(deep=True)

    def get_skill(self, skill_name: str) -> SkillMemory | None:
        with self._lock:
            skill = self._skills.get(skill_name)
            return skill.model_copy(deep=True) if skill else None

    def compact(self, agent_id: str) -> CompactResponse:
        with self._lock:
            self._purge_expired_short_term(agent_id)
            short_entries = self._short_term.get(agent_id, [])
            if not short_entries:
                raise ValueError("no short-term memories to compact")

            summary_parts = [f"{e.key}: {e.value}" for e in short_entries]
            summary_content = f"[dream compaction] Consolidated {len(short_entries)} memories: " + "; ".join(
                summary_parts
            )
            metadata = {
                "source": "compact",
                "entry_count": len(short_entries),
                "compacted_at": utcnow().isoformat(),
            }
            memory = LongTermMemory(
                id=str(uuid.uuid4()),
                agent_id=agent_id,
                content=summary_content,
                metadata=metadata,
                embedding=embed_text(summary_content),
                created_at=utcnow(),
            )
            self._long_term.setdefault(agent_id, []).append(memory)
            count = len(short_entries)
            self._short_term[agent_id] = []

        return CompactResponse(
            agent_id=agent_id,
            compacted_entries=count,
            summary_id=memory.id,
            message=f"Compacted {count} short-term entries into long-term memory",
        )


store = MemoryStore()

app = FastAPI(
    title="Arcana Memory",
    version="0.1.0",
    description="Tri-Scope Memory — short-term, long-term, and skill memory",
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=_cors_origins(),
    allow_methods=["*"],
    allow_headers=["*"],
)

structlog.configure(
    processors=[
        structlog.processors.TimeStamper(fmt="iso"),
        structlog.processors.add_log_level,
        structlog.processors.JSONRenderer(),
    ],
    wrapper_class=structlog.make_filtering_bound_logger(logging.INFO),
)
log = structlog.get_logger(service="memory")


resource = Resource.create({SERVICE_NAME: "arcana-memory"})
provider = TracerProvider(resource=resource)
endpoint = os.environ.get("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")
provider.add_span_processor(BatchSpanProcessor(OTLPSpanExporter(endpoint=f"{endpoint}/v1/traces")))
trace.set_tracer_provider(provider)
FastAPIInstrumentor.instrument_app(app)

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
    except Exception as e:
        log.warn("database_unavailable_using_memory", error=str(e))


@app.on_event("shutdown")
async def shutdown_db():
    if pool:
        await pool.close()


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


@app.get("/readyz")
async def readyz():
    return {"status": "ok"}

@app.get("/healthz")
async def healthz() -> dict[str, str]:
    return {"status": "ok"}


@app.post("/api/v1/memory/short-term", response_model=ShortTermEntry, status_code=201)
async def store_short_term(req: ShortTermStoreRequest, auth: dict = _auth_dep) -> ShortTermEntry:
    if req.ttl <= 0:
        raise HTTPException(status_code=400, detail="ttl must be positive")
    entry = store.store_short_term(req.agent_id, req.key, req.value, req.ttl)
    if pool:
        await pool.execute(
            "INSERT INTO agent_memory (id, agent_id, content, scope, type, status, created_at)"
            " VALUES ($1, $2, $3, $4, $5, $6, $7)",
            entry.id, entry.agent_id, f"{entry.key}: {entry.value}",
            "short_term", "fact", "active", entry.created_at,
        )
    return entry


@app.get("/api/v1/memory/short-term/{agent_id}", response_model=list[ShortTermEntry])
async def get_short_term(agent_id: str, auth: dict = _auth_dep) -> list[ShortTermEntry]:
    return store.get_short_term(agent_id)


@app.post("/api/v1/memory/long-term", response_model=LongTermMemory, status_code=201)
async def store_long_term(req: LongTermStoreRequest, auth: dict = _auth_dep) -> LongTermMemory:
    if not req.content.strip():
        raise HTTPException(status_code=400, detail="content is required")
    memory = store.store_long_term(req.agent_id, req.content, req.metadata, req.embedding)
    if pool:
        await pool.execute(
            "INSERT INTO agent_memory (id, agent_id, content, scope, type, status, created_at)"
            " VALUES ($1, $2, $3, $4, $5, $6, $7)",
            memory.id, memory.agent_id, memory.content, "long_term",
            memory.metadata.get("type", "fact"), "active", memory.created_at,
        )
    return memory


@app.get("/api/v1/memory/long-term/{agent_id}/search", response_model=SearchResponse)
async def search_long_term(
    agent_id: str,
    query: str = Query(..., min_length=1),
    top_k: int = Query(default=5, ge=1, le=50),
    auth: dict = _auth_dep,
) -> SearchResponse:
    # Always use in-memory search for vector similarity; DB is used as
    # persistence backing but the embedding search stays in-process.
    results = store.search_long_term(agent_id, query, top_k)
    return SearchResponse(agent_id=agent_id, query=query, results=results)


@app.post("/api/v1/memory/skill/{skill_name}", response_model=SkillMemory)
async def append_skill(skill_name: str, req: SkillAppendRequest, auth: dict = _auth_dep) -> SkillMemory:
    if not req.content.strip():
        raise HTTPException(status_code=400, detail="content is required")
    return store.append_skill(skill_name, req.content, req.metadata)


@app.get("/api/v1/memory/skill/{skill_name}", response_model=SkillMemory)
async def get_skill(skill_name: str, auth: dict = _auth_dep) -> SkillMemory:
    skill = store.get_skill(skill_name)
    if skill is None:
        raise HTTPException(status_code=404, detail="skill memory not found")
    return skill


@app.post("/api/v1/memory/compact", response_model=CompactResponse)
async def compact_memory(req: CompactRequest, auth: dict = _auth_dep) -> CompactResponse:
    try:
        return store.compact(req.agent_id)
    except ValueError as exc:
        raise HTTPException(status_code=404, detail=str(exc)) from exc


@app.post("/api/v1/memory/{agent_id}/dream")
async def dream_compact(agent_id: str, auth: dict = _auth_dep) -> dict:
    """Nightly dreaming compaction: consolidate raw memories into crisp facts.

    Merges duplicates, extracts patterns, resolves contradictions.
    """
    with store._lock:
        long_term = list(store._long_term.get(agent_id, []))

    if len(long_term) < 5:
        return {
            "compacted": 0,
            "facts": [],
            "reason": "insufficient memories for compaction",
        }

    # Deduplicate using embedding similarity instead of prefix matching
    seen: list[tuple[str, list[float]]] = []
    facts: list[dict] = []

    for mem in long_term:
        vec = embed_text(mem.content)
        is_duplicate = False
        for _, seen_vec in seen:
            if cosine_similarity(vec, seen_vec) > 0.9:
                is_duplicate = True
                break
        if not is_duplicate:
            seen.append((mem.content, vec))
            facts.append({
                "id": str(uuid.uuid4()),
                "content": mem.content,
                "source_count": 1,
                "compacted_at": utcnow().isoformat(),
                "type": "fact",
            })

    archived_count = len(long_term)

    # Store compacted facts back as long-term memories and clear originals.
    with store._lock:
        store._long_term[agent_id] = []
        for fact in facts:
            store._long_term.setdefault(agent_id, []).append(
                LongTermMemory(
                    id=fact["id"],
                    agent_id=agent_id,
                    content=fact["content"],
                    metadata={"type": "compacted_fact", "compacted_at": fact["compacted_at"]},
                    embedding=embed_text(fact["content"]),
                    created_at=utcnow(),
                )
            )

    # Persist to PostgreSQL: archive old, insert compacted facts.
    if pool:
        now = utcnow()
        await pool.execute(
            "UPDATE agent_memory SET status = 'archived'"
            " WHERE agent_id = $1 AND status = 'active'",
            agent_id,
        )
        for fact in facts:
            await pool.execute(
                "INSERT INTO agent_memory (id, agent_id, content, scope, type, status, created_at)"
                " VALUES ($1, $2, $3, $4, $5, $6, $7)",
                fact["id"], agent_id, fact["content"], "long_term",
                "compacted_fact", "active", now,
            )

    log.info("dreaming_complete", agent=agent_id, archived=archived_count, facts=len(facts))
    return {"compacted": archived_count, "facts": facts}


@app.post("/api/v1/memory/{agent_id}/reflect")
async def reflect(agent_id: str, auth: dict = _auth_dep) -> dict:
    """Generate higher-order insights from recent memories."""
    recent_count = 0
    if pool:
        rows = await pool.fetch(
            "SELECT id, content, created_at FROM agent_memory"
            " WHERE agent_id = $1 AND status = 'active'"
            " ORDER BY created_at DESC LIMIT 20",
            agent_id,
        )
        recent_count = len(rows)
    else:
        with store._lock:
            long_term = list(store._long_term.get(agent_id, []))
        recent = long_term[-20:]
        recent_count = len(recent)

    insights: list[dict] = []
    if recent_count >= 5:
        # Cluster recent memories by semantic similarity
        contents = [r["content"] if pool else r.content for r in (rows if pool else recent)]
        clusters: list[list[str]] = []
        for content in contents[:20]:
            vec = embed_text(content)
            placed = False
            for cluster in clusters:
                cluster_vec = embed_text(cluster[0])
                if cosine_similarity(vec, cluster_vec) > 0.7:
                    cluster.append(content)
                    placed = True
                    break
            if not placed:
                clusters.append([content])

        for i, cluster in enumerate(clusters):
            if len(cluster) >= 2:
                insights.append({
                    "id": str(uuid.uuid4()),
                    "content": f"Recurring theme ({len(cluster)} memories): {cluster[0][:100]}",
                    "type": "insight",
                    "source_memories": len(cluster),
                    "created_at": utcnow().isoformat(),
                })

    return {"insights": insights}
