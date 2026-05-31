from __future__ import annotations

import os
import logging
import random
import time
import uuid
from datetime import UTC, datetime
from enum import StrEnum
from typing import Any

import structlog
from fastapi import FastAPI, HTTPException, Request, Response
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel, Field


class ConnectorType(StrEnum):
    GDRIVE = "gdrive"
    CONFLUENCE = "confluence"
    SLACK = "slack"
    NOTION = "notion"
    GITHUB = "github"
    GITLAB = "gitlab"
    JIRA = "jira"
    S3 = "s3"
    POSTGRES = "postgres"
    MYSQL = "mysql"
    SNOWFLAKE = "snowflake"
    EMAIL = "email"
    WEB = "web"
    FILE = "file"


TIER1_CONNECTORS: list[dict[str, str]] = [
    {"type": ConnectorType.GDRIVE, "description": "Google Drive documents and folders"},
    {"type": ConnectorType.CONFLUENCE, "description": "Atlassian Confluence wiki pages"},
    {"type": ConnectorType.SLACK, "description": "Slack channels and message history"},
    {"type": ConnectorType.NOTION, "description": "Notion pages and databases"},
    {"type": ConnectorType.GITHUB, "description": "GitHub repositories, issues, and PRs"},
    {"type": ConnectorType.GITLAB, "description": "GitLab projects, issues, and merge requests"},
    {"type": ConnectorType.JIRA, "description": "Jira issues and project boards"},
    {"type": ConnectorType.S3, "description": "Amazon S3 object storage buckets"},
    {"type": ConnectorType.POSTGRES, "description": "PostgreSQL relational database tables"},
    {"type": ConnectorType.MYSQL, "description": "MySQL relational database tables"},
    {"type": ConnectorType.SNOWFLAKE, "description": "Snowflake data warehouse tables and views"},
    {"type": ConnectorType.EMAIL, "description": "IMAP/SMTP email mailboxes"},
    {"type": ConnectorType.WEB, "description": "Web page crawling and RSS feeds"},
    {"type": ConnectorType.FILE, "description": "Local and network filesystem paths"},
]


class ConnectorAuth(BaseModel):
    method: str = Field(description="Authentication method, e.g. oauth2, api_key, basic")
    credentials_ref: str = Field(description="Reference to stored credentials secret")


class RegisterConnectorRequest(BaseModel):
    type: ConnectorType
    name: str = Field(min_length=1, max_length=128)
    config: dict[str, Any] = Field(default_factory=dict)
    auth: ConnectorAuth | None = None


class ConnectorInstance(BaseModel):
    name: str
    type: ConnectorType
    status: str
    last_sync: datetime | None = None
    documents_synced: int = 0
    config: dict[str, Any] = Field(default_factory=dict)


class SyncResult(BaseModel):
    name: str
    status: str
    documents_synced: int
    started_at: datetime
    completed_at: datetime
    message: str


class ConnectorHealth(BaseModel):
    name: str
    healthy: bool
    status: str
    last_check: datetime
    details: dict[str, Any] = Field(default_factory=dict)


_store: dict[str, ConnectorInstance] = {}


def _now() -> datetime:
    return datetime.now(UTC)


def _simulate_sync(connector: ConnectorInstance) -> ConnectorInstance:
    docs = random.randint(5, 50)
    connector.documents_synced += docs
    connector.last_sync = _now()
    connector.status = "active"
    _store[connector.name] = connector
    return connector


app = FastAPI(
    title="Arcana Connectors",
    version="0.1.0",
    description="Tier 1 data connector registry and sync orchestration",
)

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
log = structlog.get_logger(service="connectors")

from opentelemetry import trace
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import Resource, SERVICE_NAME
from opentelemetry.instrumentation.fastapi import FastAPIInstrumentor

resource = Resource.create({SERVICE_NAME: "arcana-connectors"})
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


@app.get("/readyz")
async def readyz():
    return {"status": "ok"}

@app.get("/healthz")
async def healthz() -> dict[str, str]:
    return {"status": "ok"}


@app.get("/api/v1/connectors")
async def list_connector_types() -> dict[str, Any]:
    return {"connectors": TIER1_CONNECTORS, "count": len(TIER1_CONNECTORS)}


@app.post("/api/v1/connectors", status_code=201)
async def register_connector(req: RegisterConnectorRequest) -> ConnectorInstance:
    if req.name in _store:
        raise HTTPException(status_code=409, detail=f"connector '{req.name}' already exists")
    instance = ConnectorInstance(
        name=req.name,
        type=req.type,
        status="registered",
        config=req.config,
        documents_synced=0,
    )
    _store[req.name] = instance
    return instance


@app.get("/api/v1/connectors/{name}")
async def get_connector(name: str) -> ConnectorInstance:
    instance = _store.get(name)
    if instance is None:
        raise HTTPException(status_code=404, detail=f"connector '{name}' not found")
    return instance


@app.post("/api/v1/connectors/{name}/sync")
async def trigger_sync(name: str) -> SyncResult:
    instance = _store.get(name)
    if instance is None:
        raise HTTPException(status_code=404, detail=f"connector '{name}' not found")

    started = _now()
    updated = _simulate_sync(instance)
    return SyncResult(
        name=name,
        status="completed",
        documents_synced=updated.documents_synced,
        started_at=started,
        completed_at=_now(),
        message=f"sync completed for {updated.type} connector",
    )


@app.delete("/api/v1/connectors/{name}", status_code=204)
async def remove_connector(name: str) -> Response:
    if name not in _store:
        raise HTTPException(status_code=404, detail=f"connector '{name}' not found")
    del _store[name]
    return Response(status_code=204)


@app.get("/api/v1/connectors/{name}/health")
async def connector_health(name: str) -> ConnectorHealth:
    instance = _store.get(name)
    if instance is None:
        raise HTTPException(status_code=404, detail=f"connector '{name}' not found")

    healthy = instance.status in {"registered", "active", "syncing"}
    return ConnectorHealth(
        name=name,
        healthy=healthy,
        status=instance.status,
        last_check=_now(),
        details={
            "type": instance.type,
            "last_sync": instance.last_sync.isoformat() if instance.last_sync else None,
            "documents_synced": instance.documents_synced,
        },
    )
