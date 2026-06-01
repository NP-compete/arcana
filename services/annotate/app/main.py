from __future__ import annotations

import json
import os
import sys
import logging
import threading
import time
import uuid
from collections import defaultdict
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

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
from pydantic import BaseModel, Field

from _shared.auth import require_auth

_auth_dep = Depends(require_auth)
from _shared.embeddings import embed_text, cosine_similarity


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
log = structlog.get_logger(service="annotate")

app = FastAPI(title="Arcana Annotate", version="0.1.0", description="Annotation loop")
app.add_middleware(
    CORSMiddleware,
    allow_origins=_cors_origins(),
    allow_methods=["*"],
    allow_headers=["*"],
)


resource = Resource.create({SERVICE_NAME: "arcana-annotate"})
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


class Annotation(BaseModel):
    id: str
    question: str
    original: str
    corrected: str
    agent_id: str
    user_id: str
    embedding: list[float] = Field(default_factory=list)
    topic: str = "general"
    crystallized: bool = False
    created_at: datetime = Field(default_factory=lambda: datetime.now(UTC))


class CrystallizationCandidate(BaseModel):
    topic: str
    agent_id: str
    correction_count: int
    sample_questions: list[str]
    ready: bool


class SubmitAnnotationRequest(BaseModel):
    question: str
    original_answer: str
    corrected_answer: str
    agent_id: str
    user_id: str
    topic: str = "general"


class SearchRequest(BaseModel):
    query: str
    similarity_threshold: float = 0.5


class CrystallizeRequest(BaseModel):
    agent_id: str | None = None
    topic: str | None = None
    min_corrections: int = 3


_store_lock = threading.Lock()
_annotations: dict[str, Annotation] = {}
_CRYSTALLIZATION_THRESHOLD = 3

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

        # Ensure the annotations table exists
        async with pool.acquire() as conn:
            await conn.execute("""
                CREATE TABLE IF NOT EXISTS annotations (
                    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                    agent_id TEXT,
                    user_id TEXT,
                    topic TEXT,
                    question TEXT NOT NULL,
                    original_answer TEXT NOT NULL,
                    corrected_answer TEXT NOT NULL,
                    embedding REAL[],
                    crystallized BOOLEAN DEFAULT FALSE,
                    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
                )
            """)

        # Hydrate in-memory cache from database
        rows = await pool.fetch(
            "SELECT id, agent_id, user_id, topic, question, original_answer,"
            " corrected_answer, embedding, crystallized, created_at"
            " FROM annotations ORDER BY created_at DESC"
        )
        for r in rows:
            ann_id = str(r["id"])
            _annotations[ann_id] = Annotation(
                id=ann_id,
                question=r["question"],
                original=r["original_answer"],
                corrected=r["corrected_answer"],
                agent_id=r["agent_id"] or "",
                user_id=r["user_id"] or "",
                embedding=list(r["embedding"]) if r["embedding"] else [],
                topic=r["topic"] or "general",
                crystallized=r["crystallized"] or False,
                created_at=r["created_at"],
            )
        log.info("annotations_cache_loaded", count=len(_annotations))
    except Exception as e:
        log.warn("database_unavailable_using_memory", error=str(e))


@app.on_event("shutdown")
async def shutdown_db():
    if pool:
        await pool.close()


@app.get("/readyz")
async def readyz():
    return {"status": "ok"}

@app.get("/healthz")
async def healthz() -> dict[str, str]:
    return {"status": "ok"}


@app.post("/api/v1/annotations", response_model=Annotation, status_code=201)
async def submit_annotation(req: SubmitAnnotationRequest, auth: dict = _auth_dep) -> Annotation:
    ann_id = str(uuid.uuid4())
    embedding = embed_text(req.question + " " + req.corrected_answer)
    ann = Annotation(
        id=ann_id,
        question=req.question,
        original=req.original_answer,
        corrected=req.corrected_answer,
        agent_id=req.agent_id,
        user_id=req.user_id,
        embedding=embedding,
        topic=req.topic,
    )
    if pool:
        await pool.execute(
            "INSERT INTO annotations (id, agent_id, user_id, topic, question,"
            " original_answer, corrected_answer, embedding)"
            " VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
            uuid.UUID(ann_id), req.agent_id, req.user_id, req.topic,
            req.question, req.original_answer, req.corrected_answer, embedding,
        )
    with _store_lock:
        _annotations[ann_id] = ann
    return ann


@app.get("/api/v1/annotations")
async def list_annotations(agent: str | None = None, topic: str | None = None, auth: dict = _auth_dep) -> dict[str, Any]:
    # Use in-memory cache (kept in sync with DB)
    with _store_lock:
        items = list(_annotations.values())
    if agent:
        items = [a for a in items if a.agent_id == agent]
    if topic:
        items = [a for a in items if a.topic == topic]
    return {"annotations": items, "total": len(items)}


@app.post("/api/v1/annotations/search")
async def search_annotations(req: SearchRequest, auth: dict = _auth_dep) -> dict[str, Any]:
    query_emb = embed_text(req.query)
    matches = []
    with _store_lock:
        for ann in _annotations.values():
            if not ann.embedding:
                continue
            sim = cosine_similarity(query_emb, ann.embedding)
            if sim >= req.similarity_threshold:
                matches.append({"annotation": ann, "similarity": round(sim, 4)})
    matches.sort(key=lambda m: m["similarity"], reverse=True)
    return {"matches": matches, "total": len(matches)}


@app.get("/api/v1/annotations/stats")
async def annotation_stats(auth: dict = _auth_dep) -> dict[str, Any]:
    with _store_lock:
        total = len(_annotations)
        crystallized_count = sum(1 for a in _annotations.values() if getattr(a, "crystallized", False))
        unique_topics = len({a.topic for a in _annotations.values()})

        by_agent: dict[str, int] = defaultdict(int)
        by_topic: dict[str, int] = defaultdict(int)
        for ann in _annotations.values():
            by_agent[ann.agent_id] += 1
            by_topic[ann.topic] += 1

        candidates: list[CrystallizationCandidate] = []
        topic_agent_counts: dict[tuple[str, str], list[str]] = defaultdict(list)
        for ann in _annotations.values():
            topic_agent_counts[(ann.topic, ann.agent_id)].append(ann.question)

        for (topic, agent_id), questions in topic_agent_counts.items():
            count = len(questions)
            candidates.append(CrystallizationCandidate(
                topic=topic,
                agent_id=agent_id,
                correction_count=count,
                sample_questions=questions[:3],
                ready=count >= _CRYSTALLIZATION_THRESHOLD,
            ))

    return {
        "total_annotations": total,
        "total": total,
        "crystallized": crystallized_count,
        "unique_topics": unique_topics,
        "cache_hit_rate": 0.0,
        "by_agent": dict(by_agent),
        "by_topic": dict(by_topic),
        "crystallization_candidates": [c for c in candidates if c.ready],
    }


@app.post("/api/v1/annotations/crystallize")
async def crystallize(req: CrystallizeRequest, auth: dict = _auth_dep) -> dict[str, Any]:
    crystallized_ids: list[str] = []
    with _store_lock:
        grouped: dict[tuple[str, str], list[Annotation]] = defaultdict(list)
        for ann in _annotations.values():
            if req.agent_id and ann.agent_id != req.agent_id:
                continue
            if req.topic and ann.topic != req.topic:
                continue
            grouped[(ann.topic, ann.agent_id)].append(ann)

        crystallized = []
        for (topic, agent_id), anns in grouped.items():
            if len(anns) < req.min_corrections:
                continue
            skill_name = f"crystallized-{topic}-{agent_id}".replace(" ", "-").lower()
            crystallized.append({
                "skill_name": skill_name,
                "topic": topic,
                "agent_id": agent_id,
                "corrections_used": len(anns),
                "status": "created",
                "examples": [{"question": a.question, "answer": a.corrected} for a in anns[:5]],
            })
            # Mark annotations as crystallized
            for ann in anns:
                ann.crystallized = True
                crystallized_ids.append(ann.id)

    # Persist crystallized status to database
    if pool and crystallized_ids:
        await pool.execute(
            "UPDATE annotations SET crystallized = TRUE WHERE id = ANY($1::uuid[])",
            [uuid.UUID(aid) for aid in crystallized_ids],
        )

    if not crystallized:
        raise HTTPException(status_code=404, detail="no candidates meet crystallization threshold")
    return {"crystallized_skills": crystallized, "total": len(crystallized)}
