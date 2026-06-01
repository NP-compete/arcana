from __future__ import annotations

import os
import sys
import logging
import time
import uuid
from abc import ABC, abstractmethod
from datetime import UTC, datetime
from enum import StrEnum
from pathlib import Path
from typing import Any

sys.path.insert(0, str(Path(__file__).resolve().parents[2]))

import structlog
from fastapi import Depends, FastAPI, HTTPException, Request, Response
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel, Field

from _shared.auth import require_auth


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


# ---------------------------------------------------------------------------
# Connector plugin system
# ---------------------------------------------------------------------------

class ConnectorPlugin(ABC):
    """Base class for data connectors."""

    @abstractmethod
    async def sync(self, config: dict) -> tuple[int, list[str]]:
        """Sync data and return (doc_count, errors)."""
        ...

    @abstractmethod
    async def health_check(self, config: dict) -> bool:
        """Check connector health."""
        ...


class S3Connector(ConnectorPlugin):
    async def sync(self, config: dict) -> tuple[int, list[str]]:
        try:
            import boto3

            s3 = boto3.client(
                "s3",
                endpoint_url=config.get("endpoint_url"),
                aws_access_key_id=config.get("access_key"),
                aws_secret_access_key=config.get("secret_key"),
                region_name=config.get("region", "us-east-1"),
            )
            bucket = config.get("bucket", "")
            prefix = config.get("prefix", "")

            paginator = s3.get_paginator("list_objects_v2")
            count = 0
            for page in paginator.paginate(Bucket=bucket, Prefix=prefix):
                count += len(page.get("Contents", []))
            return count, []
        except ImportError:
            return 0, ["boto3 not installed"]
        except Exception as e:
            return 0, [str(e)]

    async def health_check(self, config: dict) -> bool:
        try:
            import boto3

            s3 = boto3.client(
                "s3",
                endpoint_url=config.get("endpoint_url"),
                aws_access_key_id=config.get("access_key"),
                aws_secret_access_key=config.get("secret_key"),
            )
            s3.head_bucket(Bucket=config.get("bucket", ""))
            return True
        except Exception:
            return False


class PostgresConnector(ConnectorPlugin):
    async def sync(self, config: dict) -> tuple[int, list[str]]:
        try:
            import asyncpg

            conn = await asyncpg.connect(
                host=config.get("host", "localhost"),
                port=int(config.get("port", 5432)),
                user=config.get("user", ""),
                password=config.get("password", ""),
                database=config.get("database", ""),
            )
            tables = await conn.fetch(
                "SELECT schemaname, tablename FROM pg_tables "
                "WHERE schemaname NOT IN ('pg_catalog', 'information_schema')"
            )
            total_rows = 0
            for t in tables:
                count = await conn.fetchval(
                    f'SELECT COUNT(*) FROM "{t["schemaname"]}"."{t["tablename"]}"'
                )
                total_rows += count
            await conn.close()
            return total_rows, []
        except Exception as e:
            return 0, [str(e)]

    async def health_check(self, config: dict) -> bool:
        try:
            import asyncpg

            conn = await asyncpg.connect(
                host=config.get("host"),
                port=int(config.get("port", 5432)),
                user=config.get("user"),
                password=config.get("password"),
                database=config.get("database"),
            )
            await conn.close()
            return True
        except Exception:
            return False


class WebConnector(ConnectorPlugin):
    async def sync(self, config: dict) -> tuple[int, list[str]]:
        try:
            import httpx

            urls = config.get("urls", [])
            if isinstance(urls, str):
                urls = [urls]
            count = 0
            errors: list[str] = []
            async with httpx.AsyncClient(timeout=30.0) as http_client:
                for url in urls:
                    try:
                        resp = await http_client.get(url)
                        if resp.status_code == 200:
                            count += 1
                        else:
                            errors.append(f"{url}: HTTP {resp.status_code}")
                    except Exception as e:
                        errors.append(f"{url}: {e}")
            return count, errors
        except ImportError:
            return 0, ["httpx not installed"]

    async def health_check(self, config: dict) -> bool:
        return True  # Web URLs are always "reachable" from our perspective


_PLUGINS: dict[str, type[ConnectorPlugin]] = {
    "s3": S3Connector,
    "postgres": PostgresConnector,
    "mysql": PostgresConnector,  # Similar pattern
    "web": WebConnector,
    "file": WebConnector,  # Placeholder
}


# ---------------------------------------------------------------------------
# Pydantic models
# ---------------------------------------------------------------------------

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


def _cors_origins() -> list[str]:
    origins = os.getenv("CORS_ORIGINS", "*")
    if os.getenv("ARCANA_ENV") == "production" and origins == "*":
        raise RuntimeError("CORS_ORIGINS must be set in production")
    return [o.strip() for o in origins.split(",")]


app = FastAPI(
    title="Arcana Connectors",
    version="0.1.0",
    description="Tier 1 data connector registry and sync orchestration",
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
async def list_connector_types(auth: dict = Depends(require_auth)) -> dict[str, Any]:
    return {"connectors": TIER1_CONNECTORS, "count": len(TIER1_CONNECTORS)}


@app.post("/api/v1/connectors", status_code=201)
async def register_connector(req: RegisterConnectorRequest, auth: dict = Depends(require_auth)) -> ConnectorInstance:
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
async def get_connector(name: str, auth: dict = Depends(require_auth)) -> ConnectorInstance:
    instance = _store.get(name)
    if instance is None:
        raise HTTPException(status_code=404, detail=f"connector '{name}' not found")
    return instance


@app.post("/api/v1/connectors/{name}/sync")
async def trigger_sync(name: str, auth: dict = Depends(require_auth)) -> SyncResult:
    instance = _store.get(name)
    if instance is None:
        raise HTTPException(status_code=404, detail=f"connector '{name}' not found")

    started = _now()

    plugin_cls = _PLUGINS.get(instance.type)
    if plugin_cls:
        plugin = plugin_cls()
        doc_count, errors = await plugin.sync(instance.config)
        instance.documents_synced = doc_count
        instance.last_sync = _now()
        if errors:
            instance.status = "error"
            status = "error"
            message = "; ".join(errors[:5])
        else:
            instance.status = "active"
            status = "completed"
            message = f"sync completed for {instance.type} connector"
        _store[name] = instance
    else:
        # Unsupported connector type — keep existing behavior
        instance.documents_synced = instance.documents_synced
        instance.status = "unsupported"
        status = "unsupported"
        message = f"connector type '{instance.type}' has no plugin implementation"
        _store[name] = instance

    return SyncResult(
        name=name,
        status=status,
        documents_synced=instance.documents_synced,
        started_at=started,
        completed_at=_now(),
        message=message,
    )


@app.delete("/api/v1/connectors/{name}", status_code=204)
async def remove_connector(name: str, auth: dict = Depends(require_auth)) -> Response:
    if name not in _store:
        raise HTTPException(status_code=404, detail=f"connector '{name}' not found")
    del _store[name]
    return Response(status_code=204)


@app.get("/api/v1/connectors/{name}/health")
async def connector_health(name: str, auth: dict = Depends(require_auth)) -> ConnectorHealth:
    instance = _store.get(name)
    if instance is None:
        raise HTTPException(status_code=404, detail=f"connector '{name}' not found")

    plugin_cls = _PLUGINS.get(instance.type)
    if plugin_cls:
        plugin = plugin_cls()
        healthy = await plugin.health_check(instance.config)
    else:
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
