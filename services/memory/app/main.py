"""Arcana Tri-Scope Memory service."""

from __future__ import annotations

import hashlib
import math
import threading
import uuid
from datetime import datetime, timezone
from typing import Any

from fastapi import FastAPI, HTTPException, Query
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel, Field


def utcnow() -> datetime:
    return datetime.now(timezone.utc)


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


EMBEDDING_DIM = 64


def text_to_embedding(text: str) -> list[float]:
    digest = hashlib.sha256(text.encode("utf-8")).digest()
    values = [((digest[i % len(digest)] / 255.0) * 2.0) - 1.0 for i in range(EMBEDDING_DIM)]
    norm = math.sqrt(sum(v * v for v in values)) or 1.0
    return [v / norm for v in values]


def cosine_similarity(a: list[float], b: list[float]) -> float:
    if len(a) != len(b):
        min_len = min(len(a), len(b))
        a = a[:min_len]
        b = b[:min_len]
    dot = sum(x * y for x, y in zip(a, b, strict=False))
    norm_a = math.sqrt(sum(x * x for x in a)) or 1.0
    norm_b = math.sqrt(sum(x * x for x in b)) or 1.0
    return dot / (norm_a * norm_b)


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
            expires_at=datetime.fromtimestamp(now.timestamp() + ttl, tz=timezone.utc),
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
        vec = embedding if embedding else text_to_embedding(content)
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
        query_vec = text_to_embedding(query)
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
                embedding=text_to_embedding(summary_content),
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
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)


@app.get("/readyz")
async def readyz():
    return {"status": "ok"}

@app.get("/healthz")
async def healthz() -> dict[str, str]:
    return {"status": "ok"}


@app.post("/api/v1/memory/short-term", response_model=ShortTermEntry, status_code=201)
async def store_short_term(req: ShortTermStoreRequest) -> ShortTermEntry:
    if req.ttl <= 0:
        raise HTTPException(status_code=400, detail="ttl must be positive")
    return store.store_short_term(req.agent_id, req.key, req.value, req.ttl)


@app.get("/api/v1/memory/short-term/{agent_id}", response_model=list[ShortTermEntry])
async def get_short_term(agent_id: str) -> list[ShortTermEntry]:
    return store.get_short_term(agent_id)


@app.post("/api/v1/memory/long-term", response_model=LongTermMemory, status_code=201)
async def store_long_term(req: LongTermStoreRequest) -> LongTermMemory:
    if not req.content.strip():
        raise HTTPException(status_code=400, detail="content is required")
    return store.store_long_term(req.agent_id, req.content, req.metadata, req.embedding)


@app.get("/api/v1/memory/long-term/{agent_id}/search", response_model=SearchResponse)
async def search_long_term(
    agent_id: str,
    query: str = Query(..., min_length=1),
    top_k: int = Query(default=5, ge=1, le=50),
) -> SearchResponse:
    results = store.search_long_term(agent_id, query, top_k)
    return SearchResponse(agent_id=agent_id, query=query, results=results)


@app.post("/api/v1/memory/skill/{skill_name}", response_model=SkillMemory)
async def append_skill(skill_name: str, req: SkillAppendRequest) -> SkillMemory:
    if not req.content.strip():
        raise HTTPException(status_code=400, detail="content is required")
    return store.append_skill(skill_name, req.content, req.metadata)


@app.get("/api/v1/memory/skill/{skill_name}", response_model=SkillMemory)
async def get_skill(skill_name: str) -> SkillMemory:
    skill = store.get_skill(skill_name)
    if skill is None:
        raise HTTPException(status_code=404, detail="skill memory not found")
    return skill


@app.post("/api/v1/memory/compact", response_model=CompactResponse)
async def compact_memory(req: CompactRequest) -> CompactResponse:
    try:
        return store.compact(req.agent_id)
    except ValueError as exc:
        raise HTTPException(status_code=404, detail=str(exc)) from exc
