"""Tests for the Arcana Connectors service with real connector plugin system."""
from __future__ import annotations

import sys
import types
from unittest.mock import AsyncMock, MagicMock, patch

import pytest
from fastapi.testclient import TestClient

# Ensure boto3 is importable for patching even if not installed
if "boto3" not in sys.modules:
    _mock_boto3 = types.ModuleType("boto3")
    _mock_boto3.client = MagicMock()
    sys.modules["boto3"] = _mock_boto3

from app.main import (
    ConnectorPlugin,
    PostgresConnector,
    S3Connector,
    WebConnector,
    _PLUGINS,
    app,
)

client = TestClient(app)


# ---------------------------------------------------------------------------
# Health / readiness probes
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
# Connector type registry
# ---------------------------------------------------------------------------

def test_list_connector_types():
    response = client.get("/api/v1/connectors")
    assert response.status_code == 200
    data = response.json()
    assert "connectors" in data
    assert data["count"] > 0
    types = [c["type"] for c in data["connectors"]]
    assert "s3" in types
    assert "postgres" in types
    assert "web" in types


# ---------------------------------------------------------------------------
# Connector CRUD
# ---------------------------------------------------------------------------

def test_register_connector():
    response = client.post(
        "/api/v1/connectors",
        json={"type": "s3", "name": "test-s3-crud", "config": {"bucket": "my-bucket"}},
    )
    assert response.status_code == 201
    data = response.json()
    assert data["name"] == "test-s3-crud"
    assert data["type"] == "s3"
    assert data["status"] == "registered"


def test_register_connector_duplicate():
    client.post(
        "/api/v1/connectors",
        json={"type": "s3", "name": "test-dup-connector", "config": {}},
    )
    response = client.post(
        "/api/v1/connectors",
        json={"type": "s3", "name": "test-dup-connector", "config": {}},
    )
    assert response.status_code == 409


def test_get_connector():
    client.post(
        "/api/v1/connectors",
        json={"type": "postgres", "name": "test-get-pg", "config": {}},
    )
    response = client.get("/api/v1/connectors/test-get-pg")
    assert response.status_code == 200
    assert response.json()["name"] == "test-get-pg"


def test_get_connector_not_found():
    response = client.get("/api/v1/connectors/nonexistent-xyz")
    assert response.status_code == 404


def test_delete_connector():
    client.post(
        "/api/v1/connectors",
        json={"type": "web", "name": "test-delete-conn", "config": {}},
    )
    response = client.delete("/api/v1/connectors/test-delete-conn")
    assert response.status_code == 204
    response = client.get("/api/v1/connectors/test-delete-conn")
    assert response.status_code == 404


def test_delete_connector_not_found():
    response = client.delete("/api/v1/connectors/nonexistent-xyz")
    assert response.status_code == 404


# ---------------------------------------------------------------------------
# Plugin system
# ---------------------------------------------------------------------------

def test_plugin_registry_has_expected_types():
    assert "s3" in _PLUGINS
    assert "postgres" in _PLUGINS
    assert "mysql" in _PLUGINS
    assert "web" in _PLUGINS
    assert "file" in _PLUGINS


def test_all_plugins_are_connector_plugin_subclasses():
    for name, cls in _PLUGINS.items():
        assert issubclass(cls, ConnectorPlugin), f"{name} is not a ConnectorPlugin subclass"


# ---------------------------------------------------------------------------
# S3 Connector
# ---------------------------------------------------------------------------

@pytest.mark.asyncio
async def test_s3_sync_success():
    mock_paginator = MagicMock()
    mock_paginator.paginate.return_value = [
        {"Contents": [{"Key": "file1.txt"}, {"Key": "file2.txt"}]},
        {"Contents": [{"Key": "file3.txt"}]},
    ]
    mock_s3 = MagicMock()
    mock_s3.get_paginator.return_value = mock_paginator

    with patch("boto3.client", return_value=mock_s3):
        connector = S3Connector()
        count, errors = await connector.sync({"bucket": "test-bucket", "prefix": "data/"})
        assert count == 3
        assert errors == []


@pytest.mark.asyncio
async def test_s3_sync_boto3_not_installed():
    connector = S3Connector()
    with patch.dict("sys.modules", {"boto3": None}):
        # Force reimport failure
        with patch("builtins.__import__", side_effect=ImportError("No module named 'boto3'")):
            count, errors = await connector.sync({"bucket": "test"})
            assert count == 0
            assert any("boto3" in e for e in errors)


@pytest.mark.asyncio
async def test_s3_sync_api_error():
    with patch("boto3.client", side_effect=Exception("access denied")):
        connector = S3Connector()
        count, errors = await connector.sync({"bucket": "bad-bucket"})
        assert count == 0
        assert len(errors) == 1
        assert "access denied" in errors[0]


@pytest.mark.asyncio
async def test_s3_health_check_success():
    mock_s3 = MagicMock()
    mock_s3.head_bucket.return_value = {}
    with patch("boto3.client", return_value=mock_s3):
        connector = S3Connector()
        result = await connector.health_check({"bucket": "test-bucket"})
        assert result is True


@pytest.mark.asyncio
async def test_s3_health_check_failure():
    mock_s3 = MagicMock()
    mock_s3.head_bucket.side_effect = Exception("not found")
    with patch("boto3.client", return_value=mock_s3):
        connector = S3Connector()
        result = await connector.health_check({"bucket": "bad-bucket"})
        assert result is False


# ---------------------------------------------------------------------------
# Postgres Connector
# ---------------------------------------------------------------------------

@pytest.mark.asyncio
async def test_postgres_sync_success():
    mock_conn = AsyncMock()
    mock_conn.fetch.return_value = [
        {"schemaname": "public", "tablename": "users"},
        {"schemaname": "public", "tablename": "orders"},
    ]
    mock_conn.fetchval.side_effect = [100, 250]

    with patch("asyncpg.connect", return_value=mock_conn):
        connector = PostgresConnector()
        count, errors = await connector.sync({
            "host": "localhost",
            "port": 5432,
            "user": "test",
            "password": "test",
            "database": "testdb",
        })
        assert count == 350
        assert errors == []
        mock_conn.close.assert_awaited_once()


@pytest.mark.asyncio
async def test_postgres_sync_connection_failure():
    with patch("asyncpg.connect", side_effect=Exception("connection refused")):
        connector = PostgresConnector()
        count, errors = await connector.sync({"host": "bad-host"})
        assert count == 0
        assert len(errors) == 1
        assert "connection refused" in errors[0]


@pytest.mark.asyncio
async def test_postgres_health_check_success():
    mock_conn = AsyncMock()
    with patch("asyncpg.connect", return_value=mock_conn):
        connector = PostgresConnector()
        result = await connector.health_check({"host": "localhost", "database": "testdb"})
        assert result is True
        mock_conn.close.assert_awaited_once()


@pytest.mark.asyncio
async def test_postgres_health_check_failure():
    with patch("asyncpg.connect", side_effect=Exception("timeout")):
        connector = PostgresConnector()
        result = await connector.health_check({"host": "bad-host"})
        assert result is False


# ---------------------------------------------------------------------------
# Web Connector
# ---------------------------------------------------------------------------

@pytest.mark.asyncio
async def test_web_sync_success():
    mock_response = MagicMock()
    mock_response.status_code = 200

    mock_client = AsyncMock()
    mock_client.get.return_value = mock_response
    mock_client.__aenter__ = AsyncMock(return_value=mock_client)
    mock_client.__aexit__ = AsyncMock(return_value=False)

    with patch("httpx.AsyncClient", return_value=mock_client):
        connector = WebConnector()
        count, errors = await connector.sync({"urls": ["https://example.com", "https://test.com"]})
        assert count == 2
        assert errors == []


@pytest.mark.asyncio
async def test_web_sync_partial_failure():
    mock_response_ok = MagicMock()
    mock_response_ok.status_code = 200
    mock_response_fail = MagicMock()
    mock_response_fail.status_code = 404

    mock_client = AsyncMock()
    mock_client.get.side_effect = [mock_response_ok, mock_response_fail]
    mock_client.__aenter__ = AsyncMock(return_value=mock_client)
    mock_client.__aexit__ = AsyncMock(return_value=False)

    with patch("httpx.AsyncClient", return_value=mock_client):
        connector = WebConnector()
        count, errors = await connector.sync({"urls": ["https://good.com", "https://bad.com"]})
        assert count == 1
        assert len(errors) == 1
        assert "404" in errors[0]


@pytest.mark.asyncio
async def test_web_sync_string_url():
    """URLs can be passed as a single string."""
    mock_response = MagicMock()
    mock_response.status_code = 200

    mock_client = AsyncMock()
    mock_client.get.return_value = mock_response
    mock_client.__aenter__ = AsyncMock(return_value=mock_client)
    mock_client.__aexit__ = AsyncMock(return_value=False)

    with patch("httpx.AsyncClient", return_value=mock_client):
        connector = WebConnector()
        count, errors = await connector.sync({"urls": "https://single.com"})
        assert count == 1
        assert errors == []


@pytest.mark.asyncio
async def test_web_health_check_always_true():
    connector = WebConnector()
    result = await connector.health_check({})
    assert result is True


# ---------------------------------------------------------------------------
# Sync endpoint using plugin system
# ---------------------------------------------------------------------------

def test_sync_uses_plugin_success():
    """Sync endpoint should delegate to the correct plugin."""
    # Register a connector
    client.post(
        "/api/v1/connectors",
        json={
            "type": "s3",
            "name": "test-sync-plugin",
            "config": {"bucket": "test-bucket"},
        },
    )

    mock_paginator = MagicMock()
    mock_paginator.paginate.return_value = [
        {"Contents": [{"Key": f"file{i}.txt"} for i in range(10)]},
    ]
    mock_s3 = MagicMock()
    mock_s3.get_paginator.return_value = mock_paginator

    with patch("boto3.client", return_value=mock_s3):
        response = client.post("/api/v1/connectors/test-sync-plugin/sync")
        assert response.status_code == 200
        data = response.json()
        assert data["status"] == "completed"
        assert data["documents_synced"] == 10


def test_sync_plugin_with_errors():
    """Sync endpoint should report errors from the plugin."""
    client.post(
        "/api/v1/connectors",
        json={
            "type": "s3",
            "name": "test-sync-errors",
            "config": {"bucket": "bad-bucket"},
        },
    )

    with patch("boto3.client", side_effect=Exception("access denied")):
        response = client.post("/api/v1/connectors/test-sync-errors/sync")
        assert response.status_code == 200
        data = response.json()
        assert data["status"] == "error"


def test_sync_unsupported_type():
    """Unsupported connector types should return unsupported status."""
    # Manually insert a connector with an unsupported type via CRUD
    # The ConnectorType enum constrains registration, so we test the fallback
    # by using a type that exists in the enum but not in _PLUGINS
    client.post(
        "/api/v1/connectors",
        json={"type": "snowflake", "name": "test-unsupported", "config": {}},
    )
    response = client.post("/api/v1/connectors/test-unsupported/sync")
    assert response.status_code == 200
    data = response.json()
    assert data["status"] == "unsupported"


def test_sync_not_found():
    response = client.post("/api/v1/connectors/nonexistent/sync")
    assert response.status_code == 404


# ---------------------------------------------------------------------------
# Health check endpoint using plugin system
# ---------------------------------------------------------------------------

def test_health_check_with_plugin():
    """Health check should delegate to the plugin's health_check method."""
    client.post(
        "/api/v1/connectors",
        json={
            "type": "web",
            "name": "test-health-plugin",
            "config": {},
        },
    )
    response = client.get("/api/v1/connectors/test-health-plugin/health")
    assert response.status_code == 200
    data = response.json()
    assert data["healthy"] is True


def test_health_check_not_found():
    response = client.get("/api/v1/connectors/nonexistent/health")
    assert response.status_code == 404
