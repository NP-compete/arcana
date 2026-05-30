from __future__ import annotations

import re
import uuid
from collections import defaultdict
from enum import StrEnum
from typing import Any

from fastapi import FastAPI, HTTPException, Query
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel, Field


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


_nodes: dict[str, GraphNode] = {}
_edges: dict[str, GraphEdge] = {}


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


def _create_node(req: CreateNodeRequest) -> GraphNode:
    node_id = str(uuid.uuid4())
    node = GraphNode(id=node_id, type=req.type, name=req.name, properties=req.properties)
    _nodes[node_id] = node
    return node


def _create_edge(req: CreateEdgeRequest) -> GraphEdge:
    if req.from_id not in _nodes:
        raise HTTPException(status_code=404, detail=f"from node '{req.from_id}' not found")
    if req.to_id not in _nodes:
        raise HTTPException(status_code=404, detail=f"to node '{req.to_id}' not found")
    edge_id = str(uuid.uuid4())
    edge = GraphEdge(id=edge_id, from_id=req.from_id, to_id=req.to_id, type=req.type, properties=req.properties)
    _edges[edge_id] = edge
    return edge


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
    nodes = list(_nodes.values())
    edges = list(_edges.values())

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
        edges = [e for e in edges if e.from_id in anchor_ids or e.to_id in anchor_ids]
        expanded = anchor_ids | {e.from_id for e in edges} | {e.to_id for e in edges}
        nodes = [_nodes[nid] for nid in expanded if nid in _nodes]

    return nodes, edges


_seed_graph()

app = FastAPI(
    title="Arcana Graph",
    version="0.1.0",
    description="Knowledge graph service for entities, relationships, and traversal",
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


@app.post("/api/v1/nodes", status_code=201)
async def create_node(req: CreateNodeRequest) -> GraphNode:
    return _create_node(req)


@app.get("/api/v1/nodes")
async def list_nodes(type: NodeType | None = Query(default=None)) -> dict[str, Any]:
    nodes = list(_nodes.values())
    if type is not None:
        nodes = [n for n in nodes if n.type == type]
    return {"nodes": nodes, "count": len(nodes)}


@app.get("/api/v1/nodes/{node_id}")
async def get_node(node_id: str) -> GraphNode:
    node = _nodes.get(node_id)
    if node is None:
        raise HTTPException(status_code=404, detail=f"node '{node_id}' not found")
    return node


@app.post("/api/v1/edges", status_code=201)
async def create_edge(req: CreateEdgeRequest) -> GraphEdge:
    return _create_edge(req)


@app.get("/api/v1/edges")
async def list_edges(
    type: EdgeType | None = Query(default=None),
    from_id: str | None = Query(default=None),
    to_id: str | None = Query(default=None),
) -> dict[str, Any]:
    edges = list(_edges.values())
    if type is not None:
        edges = [e for e in edges if e.type == type]
    if from_id is not None:
        edges = [e for e in edges if e.from_id == from_id]
    if to_id is not None:
        edges = [e for e in edges if e.to_id == to_id]
    return {"edges": edges, "count": len(edges)}


@app.post("/api/v1/query")
async def graph_query(req: GraphQueryRequest) -> GraphQueryResult:
    parsed = _parse_natural_language_query(req.query)
    nodes, edges = _execute_query(parsed)
    return GraphQueryResult(query=req.query, parsed=parsed, nodes=nodes, edges=edges)


@app.get("/api/v1/graph/stats")
async def graph_stats() -> GraphStats:
    node_counts: dict[str, int] = defaultdict(int)
    edge_counts: dict[str, int] = defaultdict(int)
    for node in _nodes.values():
        node_counts[node.type.value] += 1
    for edge in _edges.values():
        edge_counts[edge.type.value] += 1
    return GraphStats(
        node_counts=dict(node_counts),
        edge_counts=dict(edge_counts),
        total_nodes=len(_nodes),
        total_edges=len(_edges),
    )
