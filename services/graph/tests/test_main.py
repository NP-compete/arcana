"""Tests for the Arcana Graph service with networkx-backed graph operations."""
from __future__ import annotations

import pytest
from fastapi.testclient import TestClient

from app.main import _graph, app

client = TestClient(app)


def _reset_graph():
    """Clear the graph between tests for isolation."""
    _graph.clear()


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
# Node CRUD
# ---------------------------------------------------------------------------

class TestNodeCRUD:
    def setup_method(self):
        _reset_graph()

    def test_create_node(self):
        response = client.post(
            "/api/v1/nodes",
            json={"type": "Person", "name": "Alice", "properties": {"role": "engineer"}},
        )
        assert response.status_code == 201
        data = response.json()
        assert data["name"] == "Alice"
        assert data["type"] == "Person"
        assert data["properties"]["role"] == "engineer"
        assert "id" in data

    def test_get_node(self):
        create_resp = client.post(
            "/api/v1/nodes",
            json={"type": "Project", "name": "Arcana"},
        )
        node_id = create_resp.json()["id"]

        response = client.get(f"/api/v1/nodes/{node_id}")
        assert response.status_code == 200
        assert response.json()["name"] == "Arcana"

    def test_get_node_not_found(self):
        response = client.get("/api/v1/nodes/nonexistent-id")
        assert response.status_code == 404

    def test_list_nodes(self):
        client.post("/api/v1/nodes", json={"type": "Person", "name": "Bob"})
        client.post("/api/v1/nodes", json={"type": "Team", "name": "Platform"})

        response = client.get("/api/v1/nodes")
        assert response.status_code == 200
        data = response.json()
        assert data["count"] == 2

    def test_list_nodes_filter_by_type(self):
        client.post("/api/v1/nodes", json={"type": "Person", "name": "Charlie"})
        client.post("/api/v1/nodes", json={"type": "Team", "name": "Backend"})
        client.post("/api/v1/nodes", json={"type": "Person", "name": "Diana"})

        response = client.get("/api/v1/nodes", params={"type": "Person"})
        assert response.status_code == 200
        data = response.json()
        assert data["count"] == 2
        for node in data["nodes"]:
            assert node["type"] == "Person"

    def test_delete_node(self):
        create_resp = client.post(
            "/api/v1/nodes",
            json={"type": "Concept", "name": "ToDelete"},
        )
        node_id = create_resp.json()["id"]

        response = client.delete(f"/api/v1/nodes/{node_id}")
        assert response.status_code == 204

        response = client.get(f"/api/v1/nodes/{node_id}")
        assert response.status_code == 404

    def test_delete_node_not_found(self):
        response = client.delete("/api/v1/nodes/nonexistent-id")
        assert response.status_code == 404

    def test_delete_node_removes_connected_edges(self):
        n1 = client.post("/api/v1/nodes", json={"type": "Person", "name": "A"}).json()
        n2 = client.post("/api/v1/nodes", json={"type": "Person", "name": "B"}).json()
        client.post(
            "/api/v1/edges",
            json={"from_id": n1["id"], "to_id": n2["id"], "type": "related_to"},
        )

        # Delete node A — edge should be removed
        client.delete(f"/api/v1/nodes/{n1['id']}")

        edges = client.get("/api/v1/edges").json()
        assert edges["count"] == 0


# ---------------------------------------------------------------------------
# Edge CRUD
# ---------------------------------------------------------------------------

class TestEdgeCRUD:
    def setup_method(self):
        _reset_graph()

    def _create_two_nodes(self) -> tuple[str, str]:
        n1 = client.post("/api/v1/nodes", json={"type": "Person", "name": "X"}).json()
        n2 = client.post("/api/v1/nodes", json={"type": "Project", "name": "Y"}).json()
        return n1["id"], n2["id"]

    def test_create_edge(self):
        n1_id, n2_id = self._create_two_nodes()
        response = client.post(
            "/api/v1/edges",
            json={"from_id": n1_id, "to_id": n2_id, "type": "owns"},
        )
        assert response.status_code == 201
        data = response.json()
        assert data["from_id"] == n1_id
        assert data["to_id"] == n2_id
        assert data["type"] == "owns"

    def test_create_edge_from_node_not_found(self):
        n2 = client.post("/api/v1/nodes", json={"type": "Person", "name": "Z"}).json()
        response = client.post(
            "/api/v1/edges",
            json={"from_id": "nonexistent", "to_id": n2["id"], "type": "owns"},
        )
        assert response.status_code == 404

    def test_create_edge_to_node_not_found(self):
        n1 = client.post("/api/v1/nodes", json={"type": "Person", "name": "Z"}).json()
        response = client.post(
            "/api/v1/edges",
            json={"from_id": n1["id"], "to_id": "nonexistent", "type": "owns"},
        )
        assert response.status_code == 404

    def test_list_edges(self):
        n1_id, n2_id = self._create_two_nodes()
        client.post("/api/v1/edges", json={"from_id": n1_id, "to_id": n2_id, "type": "owns"})

        response = client.get("/api/v1/edges")
        assert response.status_code == 200
        assert response.json()["count"] == 1

    def test_list_edges_filter_by_type(self):
        n1_id, n2_id = self._create_two_nodes()
        client.post("/api/v1/edges", json={"from_id": n1_id, "to_id": n2_id, "type": "owns"})
        client.post("/api/v1/edges", json={"from_id": n1_id, "to_id": n2_id, "type": "uses"})

        response = client.get("/api/v1/edges", params={"type": "owns"})
        assert response.status_code == 200
        assert response.json()["count"] == 1

    def test_list_edges_filter_by_from_id(self):
        n1_id, n2_id = self._create_two_nodes()
        client.post("/api/v1/edges", json={"from_id": n1_id, "to_id": n2_id, "type": "owns"})

        response = client.get("/api/v1/edges", params={"from_id": n1_id})
        assert response.status_code == 200
        assert response.json()["count"] == 1

        response = client.get("/api/v1/edges", params={"from_id": "other-id"})
        assert response.json()["count"] == 0

    def test_delete_edge(self):
        n1_id, n2_id = self._create_two_nodes()
        edge = client.post(
            "/api/v1/edges",
            json={"from_id": n1_id, "to_id": n2_id, "type": "owns"},
        ).json()

        response = client.delete(f"/api/v1/edges/{edge['id']}")
        assert response.status_code == 204

        response = client.get("/api/v1/edges")
        assert response.json()["count"] == 0

    def test_delete_edge_not_found(self):
        response = client.delete("/api/v1/edges/nonexistent-id")
        assert response.status_code == 404


# ---------------------------------------------------------------------------
# Graph stats
# ---------------------------------------------------------------------------

class TestGraphStats:
    def setup_method(self):
        _reset_graph()

    def test_stats_empty_graph(self):
        response = client.get("/api/v1/graph/stats")
        assert response.status_code == 200
        data = response.json()
        assert data["total_nodes"] == 0
        assert data["total_edges"] == 0
        assert data["density"] == 0.0

    def test_stats_with_data(self):
        n1 = client.post("/api/v1/nodes", json={"type": "Person", "name": "A"}).json()
        n2 = client.post("/api/v1/nodes", json={"type": "Project", "name": "B"}).json()
        client.post("/api/v1/edges", json={"from_id": n1["id"], "to_id": n2["id"], "type": "owns"})

        response = client.get("/api/v1/graph/stats")
        data = response.json()
        assert data["total_nodes"] == 2
        assert data["total_edges"] == 1
        assert data["density"] > 0
        assert "Person" in data["node_counts"]
        assert "Project" in data["node_counts"]
        assert "owns" in data["edge_counts"]


# ---------------------------------------------------------------------------
# Graph query (keyword-based)
# ---------------------------------------------------------------------------

class TestGraphQuery:
    def setup_method(self):
        _reset_graph()

    def test_query_by_node_type(self):
        client.post("/api/v1/nodes", json={"type": "Person", "name": "Alice"})
        client.post("/api/v1/nodes", json={"type": "Team", "name": "Platform"})

        response = client.post(
            "/api/v1/query",
            json={"query": "find all person nodes"},
        )
        assert response.status_code == 200
        data = response.json()
        assert len(data["nodes"]) == 1
        assert data["nodes"][0]["name"] == "Alice"

    def test_query_by_name(self):
        client.post("/api/v1/nodes", json={"type": "Project", "name": "Arcana Platform"})
        client.post("/api/v1/nodes", json={"type": "Project", "name": "Other Project"})

        response = client.post(
            "/api/v1/query",
            json={"query": "find nodes named 'Arcana'"},
        )
        assert response.status_code == 200
        data = response.json()
        assert len(data["nodes"]) == 1
        assert "Arcana" in data["nodes"][0]["name"]

    def test_query_neighborhood(self):
        n1 = client.post("/api/v1/nodes", json={"type": "Person", "name": "Alice"}).json()
        n2 = client.post("/api/v1/nodes", json={"type": "Project", "name": "Proj"}).json()
        client.post(
            "/api/v1/edges",
            json={"from_id": n1["id"], "to_id": n2["id"], "type": "owns"},
        )

        response = client.post(
            "/api/v1/query",
            json={"query": "what is connected to person nodes"},
        )
        assert response.status_code == 200
        data = response.json()
        # Should return both nodes (Alice + Proj) since they are connected
        assert len(data["nodes"]) == 2

    def test_query_by_edge_type(self):
        n1 = client.post("/api/v1/nodes", json={"type": "Person", "name": "Bob"}).json()
        n2 = client.post("/api/v1/nodes", json={"type": "Document", "name": "Doc1"}).json()
        client.post(
            "/api/v1/edges",
            json={"from_id": n1["id"], "to_id": n2["id"], "type": "authored"},
        )

        response = client.post(
            "/api/v1/query",
            json={"query": "find authored edges"},
        )
        assert response.status_code == 200
        data = response.json()
        assert len(data["edges"]) == 1
        assert data["edges"][0]["type"] == "authored"


# ---------------------------------------------------------------------------
# Shortest path
# ---------------------------------------------------------------------------

class TestShortestPath:
    def setup_method(self):
        _reset_graph()

    def test_shortest_path(self):
        n1 = client.post("/api/v1/nodes", json={"type": "Person", "name": "A"}).json()
        n2 = client.post("/api/v1/nodes", json={"type": "Person", "name": "B"}).json()
        n3 = client.post("/api/v1/nodes", json={"type": "Person", "name": "C"}).json()
        client.post("/api/v1/edges", json={"from_id": n1["id"], "to_id": n2["id"], "type": "related_to"})
        client.post("/api/v1/edges", json={"from_id": n2["id"], "to_id": n3["id"], "type": "related_to"})

        response = client.get(
            f"/api/v1/graph/path/{n1['id']}/{n3['id']}",
        )
        assert response.status_code == 200
        data = response.json()
        assert len(data["path"]) == 3
        assert data["path"][0] == n1["id"]
        assert data["path"][-1] == n3["id"]

    def test_shortest_path_no_path(self):
        n1 = client.post("/api/v1/nodes", json={"type": "Person", "name": "Isolated1"}).json()
        n2 = client.post("/api/v1/nodes", json={"type": "Person", "name": "Isolated2"}).json()

        response = client.get(f"/api/v1/graph/path/{n1['id']}/{n2['id']}")
        assert response.status_code == 404

    def test_shortest_path_node_not_found(self):
        response = client.get("/api/v1/graph/path/nonexistent1/nonexistent2")
        assert response.status_code == 404


# ---------------------------------------------------------------------------
# Neighbors
# ---------------------------------------------------------------------------

class TestNeighbors:
    def setup_method(self):
        _reset_graph()

    def test_neighbors(self):
        n1 = client.post("/api/v1/nodes", json={"type": "Person", "name": "Center"}).json()
        n2 = client.post("/api/v1/nodes", json={"type": "Project", "name": "P1"}).json()
        n3 = client.post("/api/v1/nodes", json={"type": "Project", "name": "P2"}).json()
        client.post("/api/v1/edges", json={"from_id": n1["id"], "to_id": n2["id"], "type": "owns"})
        client.post("/api/v1/edges", json={"from_id": n1["id"], "to_id": n3["id"], "type": "owns"})

        response = client.get(f"/api/v1/nodes/{n1['id']}/neighbors")
        assert response.status_code == 200
        data = response.json()
        assert len(data["nodes"]) == 2

    def test_neighbors_with_depth(self):
        n1 = client.post("/api/v1/nodes", json={"type": "Person", "name": "Root"}).json()
        n2 = client.post("/api/v1/nodes", json={"type": "Person", "name": "L1"}).json()
        n3 = client.post("/api/v1/nodes", json={"type": "Person", "name": "L2"}).json()
        client.post("/api/v1/edges", json={"from_id": n1["id"], "to_id": n2["id"], "type": "related_to"})
        client.post("/api/v1/edges", json={"from_id": n2["id"], "to_id": n3["id"], "type": "related_to"})

        # Depth 1 should only return n2
        response = client.get(f"/api/v1/nodes/{n1['id']}/neighbors", params={"depth": 1})
        assert response.status_code == 200
        assert len(response.json()["nodes"]) == 1

        # Depth 2 should return n2 and n3
        response = client.get(f"/api/v1/nodes/{n1['id']}/neighbors", params={"depth": 2})
        assert response.status_code == 200
        assert len(response.json()["nodes"]) == 2

    def test_neighbors_node_not_found(self):
        response = client.get("/api/v1/nodes/nonexistent/neighbors")
        assert response.status_code == 404
