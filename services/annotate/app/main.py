from __future__ import annotations

import os
import hashlib
import logging
import threading
import time
import uuid
from collections import defaultdict
from datetime import UTC, datetime
from typing import Any

import structlog
from fastapi import FastAPI, HTTPException, Request
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel, Field

app = FastAPI(title="Arcana Annotate", version="0.1.0", description="Annotation loop")
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
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
log = structlog.get_logger(service="annotate")

from opentelemetry import trace
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import Resource, SERVICE_NAME
from opentelemetry.instrumentation.fastapi import FastAPIInstrumentor

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

# --- Database persistence ---
import psycopg2
import psycopg2.extras

_db_pool = None

def _get_db():
    global _db_pool
    if _db_pool is not None:
        return _db_pool
    try:
        _db_pool = psycopg2.connect(
            host=os.environ.get("POSTGRES_HOST", "localhost"),
            port=int(os.environ.get("POSTGRES_PORT", "5432")),
            dbname=os.environ.get("POSTGRES_DB", "arcana"),
            user=os.environ.get("POSTGRES_USER", "arcana"),
            password=os.environ.get("POSTGRES_PASSWORD", "arcana-dev"),
        )
        _db_pool.autocommit = True
        return _db_pool
    except Exception as e:
        log.warn("db_connect_failed", error=str(e))
        return None

def _db_insert_annotation(ann: Annotation):
    db = _get_db()
    if not db:
        return
    try:
        with db.cursor() as cur:
            cur.execute(
                """INSERT INTO annotations (tenant, agent_id, topic, question, original_answer, corrected_answer)
                   VALUES (%s, %s, %s, %s, %s, %s)""",
                ("default", ann.agent_id, ann.topic, ann.question, ann.original, ann.corrected),
            )
    except Exception as e:
        log.warn("db_insert_annotation_failed", error=str(e))

def _db_mark_crystallized(topic: str, agent_id: str):
    db = _get_db()
    if not db:
        return
    try:
        with db.cursor() as cur:
            cur.execute(
                "UPDATE annotations SET crystallized = TRUE WHERE tenant = 'default' AND topic = %s AND agent_id = %s AND crystallized = FALSE",
                (topic, agent_id),
            )
    except Exception as e:
        log.warn("db_mark_crystallized_failed", error=str(e))

_SKILLS_HOST = os.environ.get("SKILLS_HOST", "arcana-skills.arcana.svc.cluster.local")
_SKILLS_PORT = os.environ.get("SKILLS_PORT", "8085")

def _notify_skills_service(skill_data: dict):
    """Call skills service to create a crystallized skill."""
    import httpx
    url = f"http://{_SKILLS_HOST}:{_SKILLS_PORT}/api/v1/skills/crystallize"
    try:
        resp = httpx.post(url, json=skill_data, timeout=10.0)
        log.info("skills_crystallize_callback", status=resp.status_code, skill=skill_data.get("skill_name"))
    except Exception as e:
        log.warn("skills_crystallize_callback_failed", error=str(e))


def _simple_embedding(text: str) -> list[float]:
    h = hashlib.sha256(text.encode()).digest()
    return [b / 255.0 for b in h[:16]]


def _cosine_similarity(a: list[float], b: list[float]) -> float:
    if not a or not b or len(a) != len(b):
        return 0.0
    dot = sum(x * y for x, y in zip(a, b))
    norm_a = sum(x * x for x in a) ** 0.5
    norm_b = sum(x * x for x in b) ** 0.5
    if norm_a == 0 or norm_b == 0:
        return 0.0
    return dot / (norm_a * norm_b)


@app.get("/readyz")
async def readyz():
    return {"status": "ok"}

@app.get("/healthz")
async def healthz() -> dict[str, str]:
    return {"status": "ok"}


@app.post("/api/v1/annotations", response_model=Annotation, status_code=201)
async def submit_annotation(req: SubmitAnnotationRequest) -> Annotation:
    ann_id = str(uuid.uuid4())
    embedding = _simple_embedding(req.question + req.corrected_answer)
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
    with _store_lock:
        _annotations[ann_id] = ann
    _db_insert_annotation(ann)
    return ann


@app.get("/api/v1/annotations")
async def list_annotations(agent: str | None = None, topic: str | None = None) -> dict[str, Any]:
    with _store_lock:
        items = list(_annotations.values())
    if agent:
        items = [a for a in items if a.agent_id == agent]
    if topic:
        items = [a for a in items if a.topic == topic]
    return {"annotations": items, "total": len(items)}


@app.post("/api/v1/annotations/search")
async def search_annotations(req: SearchRequest) -> dict[str, Any]:
    query_emb = _simple_embedding(req.query)
    matches = []
    with _store_lock:
        for ann in _annotations.values():
            sim = _cosine_similarity(query_emb, ann.embedding)
            if sim >= req.similarity_threshold:
                matches.append({"annotation": ann, "similarity": round(sim, 4)})
    matches.sort(key=lambda m: m["similarity"], reverse=True)
    return {"matches": matches, "total": len(matches)}


@app.get("/api/v1/annotations/stats")
async def annotation_stats() -> dict[str, Any]:
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
async def crystallize(req: CrystallizeRequest) -> dict[str, Any]:
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
            for ann in anns:
                ann.crystallized = True
            _db_mark_crystallized(topic, agent_id)

    if not crystallized:
        raise HTTPException(status_code=404, detail="no candidates meet crystallization threshold")

    for skill in crystallized:
        _notify_skills_service({
            "patterns": [{"pattern": skill["skill_name"], "occurrences": skill["corrections_used"]}],
            "examples": skill["examples"],
        })

    return {"crystallized_skills": crystallized, "total": len(crystallized)}
