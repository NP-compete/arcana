"""
Arcana Integration Adapter SDK

Build custom connectors for proprietary systems in ~200 lines.
Implement the ArcanaAdapter class and register with the platform.

Example:
    class MyAPIAdapter(ArcanaAdapter):
        name = "my-api"
        description = "Custom API connector"

        async def connect(self, config):
            self.client = httpx.AsyncClient(base_url=config["base_url"])

        async def sync(self, since=None):
            resp = await self.client.get("/data", params={"since": since})
            return [Document(id=d["id"], content=d["text"]) for d in resp.json()]

        async def health_check(self):
            resp = await self.client.get("/health")
            return resp.status_code == 200

    if __name__ == "__main__":
        run_adapter(MyAPIAdapter)
"""
from __future__ import annotations

import asyncio
import os
from abc import ABC, abstractmethod
from dataclasses import dataclass, field
from datetime import datetime, UTC
from typing import Any

import httpx
from fastapi import FastAPI
from pydantic import BaseModel


@dataclass
class Document:
    id: str
    content: str
    metadata: dict[str, Any] = field(default_factory=dict)
    source: str = ""
    updated_at: datetime = field(default_factory=lambda: datetime.now(UTC))


class AdapterConfig(BaseModel):
    name: str
    config: dict[str, Any] = {}
    auth: dict[str, str] = {}


class ArcanaAdapter(ABC):
    name: str = "unnamed"
    description: str = ""
    version: str = "1.0.0"

    @abstractmethod
    async def connect(self, config: dict[str, Any]) -> None:
        """Initialize connection to the external system."""

    @abstractmethod
    async def sync(self, since: str | None = None) -> list[Document]:
        """Fetch documents from the external system."""

    @abstractmethod
    async def health_check(self) -> bool:
        """Check if the external system is reachable."""

    async def disconnect(self) -> None:
        """Clean up resources."""

    def capabilities(self) -> list[str]:
        return ["sync", "health_check"]


def run_adapter(adapter_class: type[ArcanaAdapter], port: int = 9200):
    """Run an adapter as a standalone FastAPI service."""
    app = FastAPI(
        title=f"Arcana Adapter: {adapter_class.name}",
        version=adapter_class.version,
    )

    adapter = adapter_class()
    connected = False

    @app.post("/connect")
    async def connect(config: AdapterConfig):
        nonlocal connected
        await adapter.connect(config.config)
        connected = True
        return {"status": "connected", "adapter": adapter.name}

    @app.post("/sync")
    async def sync(since: str | None = None):
        if not connected:
            return {"error": "not connected"}, 400
        docs = await adapter.sync(since)
        return {
            "documents": [
                {"id": d.id, "content": d.content, "metadata": d.metadata}
                for d in docs
            ],
            "count": len(docs),
        }

    @app.get("/health")
    async def health():
        if not connected:
            return {"healthy": False, "reason": "not connected"}
        healthy = await adapter.health_check()
        return {"healthy": healthy, "adapter": adapter.name}

    @app.get("/info")
    async def info():
        return {
            "name": adapter.name,
            "description": adapter.description,
            "version": adapter.version,
            "capabilities": adapter.capabilities(),
            "connected": connected,
        }

    @app.post("/disconnect")
    async def disconnect():
        nonlocal connected
        await adapter.disconnect()
        connected = False
        return {"status": "disconnected"}

    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=port)
