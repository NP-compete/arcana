from __future__ import annotations

import os
import sys
import logging
import re
import time
import uuid
from collections import defaultdict
from datetime import UTC, datetime
from enum import StrEnum
from pathlib import Path
from typing import Any

sys.path.insert(0, str(Path(__file__).resolve().parents[2]))

import networkx as nx
import structlog
from fastapi import Depends, FastAPI, HTTPException, Query, Request, Response
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel, Field

from _shared.auth import require_auth


# ---------------------------------------------------------------------------
# Enums
# ---------------------------------------------------------------------------

class NodeType(StrEnum):
    PERSON = "Person"
    TEAM = "Team"
    PROJECT = "Project"
    DOCUMENT = "Document"
    CONCEPT = "Concept"
    AGENT = "Agent"
    SKILL = "Skill"
    TOOL = "Tool"
    MODEL = "Model"


class EdgeType(StrEnum):
    AUTHORED = "authored"
    REFERENCES = "references"
    SUPERSEDES = "supersedes"
    DEPENDS_ON = "depends_on"
    MEMBER_OF = "member_of"
    RELATED_TO = "related_to"
    OWNS = "owns"
    USES = "uses"
    INVOKES = "invokes"


# ---------------------------------------------------------------------------
# Pydantic models
# ---------------------------------------------------------------------------

class GraphNode(BaseModel):
    id: str
    type: NodeType
    name: str
    properties: dict[str, Any] = Field(default_factory=dict)


class GraphEdge(BaseModel):
    id: str
    from_id: str
    to_id: str
    type: EdgeType
    properties: dict[str, Any] = Field(default_factory=dict)


class CreateNodeRequest(BaseModel):
    type: NodeType
    name: str = Field(min_length=1, max_length=256)
    properties: dict[str, Any] = Field(default_factory=dict)


class CreateEdgeRequest(BaseModel):
    from_id: str
    to_id: str
    type: EdgeType
    properties: dict[str, Any] = Field(default_factory=dict)


class GraphQueryRequest(BaseModel):
    query: str = Field(min_length=1, description="Natural language graph query")


class GraphQueryResult(BaseModel):
    query: str
    parsed: dict[str, Any]
    nodes: list[GraphNode]
    edges: list[GraphEdge]


class GraphStats(BaseModel):
    node_counts: dict[str, int]
    edge_counts: dict[str, int]
    total_nodes: int
    total_edges: int
    density: float
    connected_components: int


# ---------------------------------------------------------------------------
# NetworkX graph store
# ---------------------------------------------------------------------------

_graph = nx.DiGraph()


def _utcnow() -> datetime:
    return datetime.now(UTC)


def _node_to_model(node_id: str) -> GraphNode:
    """Convert a networkx node to a GraphNode model."""
    data = _graph.nodes[node_id]
    return GraphNode(
        id=node_id,
        type=data["type"],
        name=data["name"],
        properties=data.get("properties", {}),
    )


def _edge_to_model(from_id: str, to_id: str, key: str) -> GraphEdge:
    """Convert a networkx edge to a GraphEdge model."""
    data = _graph.edges[from_id, to_id, key]
    return GraphEdge(
        id=data["id"],
        from_id=from_id,
        to_id=to_id,
        type=data["relation"],
        properties=data.get("properties", {}),
    )


# Use a MultiDiGraph so we can have multiple edges between the same pair of nodes
_graph = nx.MultiDiGraph()

# Edge ID to (from_id, to_id, key) mapping for O(1) lookup by edge ID
_edge_index: dict[str, tuple[str, str, int]] = {}


def _create_node(req: CreateNodeRequest) -> GraphNode:
    node_id = str(uuid.uuid4())
    _graph.add_node(
        node_id,
        name=req.name,
        type=req.type,
        properties=dict(req.properties),
        created_at=_utcnow().isoformat(),
    )
    return GraphNode(id=node_id, type=req.type, name=req.name, properties=req.properties)


def _create_edge(req: CreateEdgeRequest) -> GraphEdge:
    if not _graph.has_node(req.from_id):
        raise HTTPException(status_code=404, detail=f"from node '{req.from_id}' not found")
    if not _graph.has_node(req.to_id):
        raise HTTPException(status_code=404, detail=f"to node '{req.to_id}' not found")
    edge_id = str(uuid.uuid4())
    key = _graph.add_edge(
        req.from_id,
        req.to_id,
        id=edge_id,
        relation=req.type,
        properties=dict(req.properties),
        created_at=_utcnow().isoformat(),
    )
    _edge_index[edge_id] = (req.from_id, req.to_id, key)
    return GraphEdge(id=edge_id, from_id=req.from_id, to_id=req.to_id, type=req.type, properties=req.properties)


def _get_all_nodes(node_type: NodeType | None = None) -> list[GraphNode]:
    """Return all nodes, optionally filtered by type."""
    result = []
    for node_id, data in _graph.nodes(data=True):
        if node_type is not None and data.get("type") != node_type:
            continue
        result.append(GraphNode(
            id=node_id,
            type=data["type"],
            name=data["name"],
            properties=data.get("properties", {}),
        ))
    return result


def _get_all_edges(
    edge_type: EdgeType | None = None,
    from_id: str | None = None,
    to_id: str | None = None,
) -> list[GraphEdge]:
    """Return all edges, optionally filtered."""
    result = []
    for u, v, key, data in _graph.edges(keys=True, data=True):
        if edge_type is not None and data.get("relation") != edge_type:
            continue
        if from_id is not None and u != from_id:
            continue
        if to_id is not None and v != to_id:
            continue
        result.append(GraphEdge(
            id=data["id"],
            from_id=u,
            to_id=v,
            type=data["relation"],
            properties=data.get("properties", {}),
        ))
    return result


# ---------------------------------------------------------------------------
# Query engine (keyword extraction, replacing regex NLP)
# ---------------------------------------------------------------------------

def _parse_natural_language_query(query: str) -> dict[str, Any]:
    q = query.lower().strip()
    parsed: dict[str, Any] = {"raw": query, "intent": "traverse"}

    type_match = re.search(r"\b(person|team|project|document|concept|agent|skill|tool|model)s?\b", q)
    if type_match:
        parsed["node_type"] = type_match.group(1).capitalize()

    edge_match = re.search(
        r"\b(authored|references|supersedes|depends_on|member_of|related_to|owns|uses|invokes)\b", q
    )
    if edge_match:
        parsed["edge_type"] = edge_match.group(1)

    name_match = re.search(r"(?:named|called|name)\s+['\"]?([^'\"]+?)['\"]?(?:\s|$)", q)
    if name_match:
        parsed["name_contains"] = name_match.group(1).strip()

    if "connected" in q or "related" in q:
        parsed["intent"] = "neighborhood"

    return parsed


def _execute_query(parsed: dict[str, Any]) -> tuple[list[GraphNode], list[GraphEdge]]:
    nodes = _get_all_nodes()
    edges = _get_all_edges()

    if node_type := parsed.get("node_type"):
        nodes = [n for n in nodes if n.type.value == node_type]

    if name_contains := parsed.get("name_contains"):
        needle = name_contains.lower()
        nodes = [n for n in nodes if needle in n.name.lower()]

    if edge_type := parsed.get("edge_type"):
        edges = [e for e in edges if e.type.value == edge_type]
        node_ids = {e.from_id for e in edges} | {e.to_id for e in edges}
        if parsed.get("intent") != "traverse" or parsed.get("node_type") or parsed.get("name_contains"):
            nodes = [n for n in nodes if n.id in node_ids]

    if parsed.get("intent") == "neighborhood" and nodes:
        anchor_ids = {n.id for n in nodes}
        # Use networkx to find neighbors
        expanded = set(anchor_ids)
        for nid in anchor_ids:
            if _graph.has_node(nid):
                expanded.update(nx.descendants(_graph, nid))
                expanded.update(nx.ancestors(_graph, nid))
        # But limit to direct neighbors for the edge list
        edges = [e for e in _get_all_edges() if e.from_id in anchor_ids or e.to_id in anchor_ids]
        neighbor_ids = anchor_ids | {e.from_id for e in edges} | {e.to_id for e in edges}
        nodes = [_node_to_model(nid) for nid in neighbor_ids if _graph.has_node(nid)]

    return nodes, edges


# ---------------------------------------------------------------------------
# Seed graph (configurable via environment)
# ---------------------------------------------------------------------------

def _seed_graph() -> None:
    seed_nodes = [
        CreateNodeRequest(type=NodeType.PROJECT, name="Arcana Platform", properties={"domain": "ai-platform"}),
        CreateNodeRequest(type=NodeType.TEAM, name="Platform Engineering", properties={"size": 12}),
        CreateNodeRequest(type=NodeType.AGENT, name="codex-router", properties={"port": 8090}),
        CreateNodeRequest(type=NodeType.SKILL, name="semantic-search", properties={"version": "1.0"}),
        CreateNodeRequest(type=NodeType.CONCEPT, name="fusion-profiles", properties={"count": 4}),
    ]
    created = [_create_node(n) for n in seed_nodes]
    _create_edge(CreateEdgeRequest(from_id=created[1].id, to_id=created[0].id, type=EdgeType.OWNS))
    _create_edge(CreateEdgeRequest(from_id=created[2].id, to_id=created[3].id, type=EdgeType.USES))
    _create_edge(CreateEdgeRequest(from_id=created[2].id, to_id=created[4].id, type=EdgeType.REFERENCES))


if os.getenv("ARCANA_SEED_GRAPH", "false").lower() == "true":
    _seed_graph()


# ---------------------------------------------------------------------------
# Application setup
# ---------------------------------------------------------------------------

def _cors_origins() -> list[str]:
    origins = os.getenv("CORS_ORIGINS", "*")
    if os.getenv("ARCANA_ENV") == "production" and origins == "*":
        raise RuntimeError("CORS_ORIGINS must be set in production")
    return [o.strip() for o in origins.split(",")]


app = FastAPI(
    title="Arcana Graph",
    version="0.1.0",
    description="Knowledge graph service for entities, relationships, and traversal",
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
log = structlog.get_logger(service="graph")

from opentelemetry import trace
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import Resource, SERVICE_NAME
from opentelemetry.instrumentation.fastapi import FastAPIInstrumentor

resource = Resource.create({SERVICE_NAME: "arcana-graph"})
provider = TracerProvider(resource=resource)
endpoint = os.environ.get("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")
provider.add_span_processor(BatchSpanProcessor(OTLPSpanExporter(endpoint=f"{endpoint}/v1/traces")))
trace.set_tracer_provider(provider)
FastAPIInstrumentor.instrument_app(app)


# ---------------------------------------------------------------------------
# Middleware
# ---------------------------------------------------------------------------

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


# ---------------------------------------------------------------------------
# Probes
# ---------------------------------------------------------------------------

@app.get("/readyz")
async def readyz():
    return {"status": "ok"}

@app.get("/healthz")
async def healthz() -> dict[str, str]:
    return {"status": "ok"}


# ---------------------------------------------------------------------------
# Node endpoints
# ---------------------------------------------------------------------------

@app.post("/api/v1/nodes", status_code=201)
async def create_node(req: CreateNodeRequest, auth: dict = Depends(require_auth)) -> GraphNode:
    return _create_node(req)


@app.get("/api/v1/nodes")
async def list_nodes(type: NodeType | None = Query(default=None), auth: dict = Depends(require_auth)) -> dict[str, Any]:
    nodes = _get_all_nodes(node_type=type)
    return {"nodes": nodes, "count": len(nodes)}


@app.get("/api/v1/nodes/{node_id}")
async def get_node(node_id: str, auth: dict = Depends(require_auth)) -> GraphNode:
    if not _graph.has_node(node_id):
        raise HTTPException(status_code=404, detail=f"node '{node_id}' not found")
    return _node_to_model(node_id)


@app.delete("/api/v1/nodes/{node_id}", status_code=204)
async def delete_node(node_id: str, auth: dict = Depends(require_auth)) -> Response:
    if not _graph.has_node(node_id):
        raise HTTPException(status_code=404, detail=f"node '{node_id}' not found")
    # Remove edge index entries for edges connected to this node
    edges_to_remove = []
    for u, v, key, data in _graph.edges(keys=True, data=True):
        if u == node_id or v == node_id:
            edge_id = data.get("id")
            if edge_id and edge_id in _edge_index:
                edges_to_remove.append(edge_id)
    for eid in edges_to_remove:
        del _edge_index[eid]
    _graph.remove_node(node_id)
    return Response(status_code=204)


@app.get("/api/v1/nodes/{node_id}/neighbors")
async def get_neighbors(
    node_id: str,
    depth: int = Query(default=1, ge=1, le=10),
    auth: dict = Depends(require_auth),
) -> dict[str, Any]:
    if not _graph.has_node(node_id):
        raise HTTPException(status_code=404, detail=f"node '{node_id}' not found")

    # Use ego_graph on the undirected view for neighbor discovery
    undirected = _graph.to_undirected()
    ego = nx.ego_graph(undirected, node_id, radius=depth)
    neighbor_ids = set(ego.nodes()) - {node_id}

    nodes = [_node_to_model(nid) for nid in neighbor_ids if _graph.has_node(nid)]
    return {"nodes": nodes, "count": len(nodes)}


# ---------------------------------------------------------------------------
# Edge endpoints
# ---------------------------------------------------------------------------

@app.post("/api/v1/edges", status_code=201)
async def create_edge(req: CreateEdgeRequest, auth: dict = Depends(require_auth)) -> GraphEdge:
    return _create_edge(req)


@app.get("/api/v1/edges")
async def list_edges(
    type: EdgeType | None = Query(default=None),
    from_id: str | None = Query(default=None),
    to_id: str | None = Query(default=None),
    auth: dict = Depends(require_auth),
) -> dict[str, Any]:
    edges = _get_all_edges(edge_type=type, from_id=from_id, to_id=to_id)
    return {"edges": edges, "count": len(edges)}


@app.delete("/api/v1/edges/{edge_id}", status_code=204)
async def delete_edge(edge_id: str, auth: dict = Depends(require_auth)) -> Response:
    if edge_id not in _edge_index:
        raise HTTPException(status_code=404, detail=f"edge '{edge_id}' not found")
    from_id, to_id, key = _edge_index[edge_id]
    _graph.remove_edge(from_id, to_id, key=key)
    del _edge_index[edge_id]
    return Response(status_code=204)


# ---------------------------------------------------------------------------
# Graph traversal endpoints
# ---------------------------------------------------------------------------

@app.get("/api/v1/graph/path/{source_id}/{target_id}")
async def shortest_path(source_id: str, target_id: str, auth: dict = Depends(require_auth)) -> dict[str, Any]:
    if not _graph.has_node(source_id):
        raise HTTPException(status_code=404, detail=f"source node '{source_id}' not found")
    if not _graph.has_node(target_id):
        raise HTTPException(status_code=404, detail=f"target node '{target_id}' not found")
    try:
        path = nx.shortest_path(_graph, source_id, target_id)
    except nx.NetworkXNoPath:
        raise HTTPException(status_code=404, detail="no path found between the specified nodes")
    nodes = [_node_to_model(nid) for nid in path]
    return {"path": path, "nodes": nodes, "length": len(path) - 1}


# ---------------------------------------------------------------------------
# Query endpoint
# ---------------------------------------------------------------------------

@app.post("/api/v1/query")
async def graph_query(req: GraphQueryRequest, auth: dict = Depends(require_auth)) -> GraphQueryResult:
    parsed = _parse_natural_language_query(req.query)
    nodes, edges = _execute_query(parsed)
    return GraphQueryResult(query=req.query, parsed=parsed, nodes=nodes, edges=edges)


# ---------------------------------------------------------------------------
# Stats endpoint
# ---------------------------------------------------------------------------

@app.get("/api/v1/graph/stats")
async def graph_stats(auth: dict = Depends(require_auth)) -> GraphStats:
    node_counts: dict[str, int] = defaultdict(int)
    edge_counts: dict[str, int] = defaultdict(int)
    for _, data in _graph.nodes(data=True):
        node_counts[data.get("type", "unknown")] += 1
    for _, _, data in _graph.edges(data=True):
        relation = data.get("relation", "unknown")
        if hasattr(relation, "value"):
            relation = relation.value
        edge_counts[relation] += 1

    n = len(_graph.nodes)
    e = len(_graph.edges)
    density = nx.density(_graph) if n > 0 else 0.0

    # Connected components on the undirected view
    if n > 0:
        undirected = _graph.to_undirected()
        components = nx.number_connected_components(undirected)
    else:
        components = 0

    return GraphStats(
        node_counts=dict(node_counts),
        edge_counts=dict(edge_counts),
        total_nodes=n,
        total_edges=e,
        density=round(density, 6),
        connected_components=components,
    )
