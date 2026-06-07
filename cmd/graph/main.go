package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/NP-compete/arcana/pkg/server"
)

type GraphNode struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
}

type GraphEdge struct {
	From     string                 `json:"from"`
	To       string                 `json:"to"`
	Type     string                 `json:"type"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

type GraphStore struct {
	mu       sync.RWMutex
	nodes    map[string]*GraphNode
	edges    []GraphEdge
	neo4jURL string
	neo4jAuth string
	client   *http.Client
}

func NewGraphStore() *GraphStore {
	neo4jURL := os.Getenv("NEO4J_URL")
	if neo4jURL == "" {
		neo4jURL = "http://neo4j.arcana.svc.cluster.local:7474"
	}
	return &GraphStore{
		nodes:    make(map[string]*GraphNode),
		edges:    make([]GraphEdge, 0),
		neo4jURL: neo4jURL,
		neo4jAuth: os.Getenv("NEO4J_AUTH"),
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (gs *GraphStore) AddNode(node GraphNode) {
	gs.mu.Lock()
	gs.nodes[node.ID] = &node
	gs.mu.Unlock()

	gs.execCypher(fmt.Sprintf(
		"MERGE (n:%s {id: $id}) SET n += $props",
		node.Type,
	), map[string]interface{}{
		"id":    node.ID,
		"props": node.Properties,
	})
}

func (gs *GraphStore) AddEdge(edge GraphEdge) {
	gs.mu.Lock()
	gs.edges = append(gs.edges, edge)
	gs.mu.Unlock()

	gs.execCypher(fmt.Sprintf(
		"MATCH (a {id: $from}), (b {id: $to}) MERGE (a)-[r:%s]->(b) SET r += $props",
		edge.Type,
	), map[string]interface{}{
		"from":  edge.From,
		"to":    edge.To,
		"props": edge.Properties,
	})
}

func (gs *GraphStore) Query(cypher string, params map[string]interface{}) (interface{}, error) {
	result, err := gs.execCypher(cypher, params)
	if err != nil {
		gs.mu.RLock()
		defer gs.mu.RUnlock()
		return map[string]interface{}{
			"nodes": len(gs.nodes),
			"edges": len(gs.edges),
			"note":  "Neo4j unavailable, returning in-memory stats",
		}, nil
	}
	return result, nil
}

func (gs *GraphStore) GetNode(id string) *GraphNode {
	gs.mu.RLock()
	defer gs.mu.RUnlock()
	return gs.nodes[id]
}

func (gs *GraphStore) FindNeighbors(nodeID string) []GraphNode {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	neighbors := make(map[string]bool)
	for _, e := range gs.edges {
		if e.From == nodeID {
			neighbors[e.To] = true
		}
		if e.To == nodeID {
			neighbors[e.From] = true
		}
	}

	result := make([]GraphNode, 0)
	for id := range neighbors {
		if n, ok := gs.nodes[id]; ok {
			result = append(result, *n)
		}
	}
	return result
}

func (gs *GraphStore) FindPath(from, to string) []string {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	adj := make(map[string][]string)
	for _, e := range gs.edges {
		adj[e.From] = append(adj[e.From], e.To)
		adj[e.To] = append(adj[e.To], e.From)
	}

	visited := map[string]bool{from: true}
	queue := [][]string{{from}}

	for len(queue) > 0 {
		path := queue[0]
		queue = queue[1:]
		current := path[len(path)-1]

		if current == to {
			return path
		}

		for _, next := range adj[current] {
			if !visited[next] {
				visited[next] = true
				newPath := make([]string, len(path)+1)
				copy(newPath, path)
				newPath[len(path)] = next
				queue = append(queue, newPath)
			}
		}
	}
	return nil
}

func (gs *GraphStore) Stats() map[string]interface{} {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	typeCounts := make(map[string]int)
	for _, n := range gs.nodes {
		typeCounts[n.Type]++
	}

	edgeTypeCounts := make(map[string]int)
	for _, e := range gs.edges {
		edgeTypeCounts[e.Type]++
	}

	return map[string]interface{}{
		"total_nodes":    len(gs.nodes),
		"total_edges":    len(gs.edges),
		"node_types":     typeCounts,
		"edge_types":     edgeTypeCounts,
		"neo4j_connected": gs.neo4jAuth != "",
	}
}

func (gs *GraphStore) execCypher(cypher string, params map[string]interface{}) (interface{}, error) {
	if gs.neo4jAuth == "" {
		return nil, fmt.Errorf("neo4j not configured")
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"statements": []map[string]interface{}{
			{"statement": cypher, "parameters": params},
		},
	})

	req, _ := http.NewRequest("POST", gs.neo4jURL+"/db/neo4j/tx/commit", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+gs.neo4jAuth)

	resp, err := gs.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var result interface{}
	json.Unmarshal(body, &result)
	return result, nil
}

func main() {
	httpSrv := server.New(server.Config{
		ServiceName: "graph",
		Port:        "8095",
	})

	store := NewGraphStore()

	httpSrv.HandleFunc("/api/v1/graph/nodes", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			var node GraphNode
			json.NewDecoder(r.Body).Decode(&node)
			if node.ID == "" || node.Type == "" {
				writeError(w, http.StatusBadRequest, "id and type required")
				return
			}
			store.AddNode(node)
			writeJSON(w, http.StatusCreated, node)
		case http.MethodGet:
			id := r.URL.Query().Get("id")
			if id != "" {
				node := store.GetNode(id)
				if node == nil {
					writeError(w, http.StatusNotFound, "node not found")
					return
				}
				writeJSON(w, http.StatusOK, node)
				return
			}
			writeJSON(w, http.StatusOK, store.Stats())
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}))

	httpSrv.HandleFunc("/api/v1/graph/edges", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "POST required")
			return
		}
		var edge GraphEdge
		json.NewDecoder(r.Body).Decode(&edge)
		if edge.From == "" || edge.To == "" || edge.Type == "" {
			writeError(w, http.StatusBadRequest, "from, to, and type required")
			return
		}
		store.AddEdge(edge)
		writeJSON(w, http.StatusCreated, edge)
	}))

	httpSrv.HandleFunc("/api/v1/graph/neighbors", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if id == "" {
			writeError(w, http.StatusBadRequest, "id required")
			return
		}
		neighbors := store.FindNeighbors(id)
		writeJSON(w, http.StatusOK, map[string]interface{}{"node": id, "neighbors": neighbors})
	}))

	httpSrv.HandleFunc("/api/v1/graph/path", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		from := r.URL.Query().Get("from")
		to := r.URL.Query().Get("to")
		if from == "" || to == "" {
			writeError(w, http.StatusBadRequest, "from and to required")
			return
		}
		path := store.FindPath(from, to)
		writeJSON(w, http.StatusOK, map[string]interface{}{"from": from, "to": to, "path": path})
	}))

	httpSrv.HandleFunc("/api/v1/graph/query", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "POST required")
			return
		}
		var req struct {
			Cypher string                 `json:"cypher"`
			Params map[string]interface{} `json:"params"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		result, err := store.Query(req.Cypher, req.Params)
		if err != nil {
			writeJSON(w, http.StatusOK, result)
			return
		}
		writeJSON(w, http.StatusOK, result)
	}))

	httpSrv.HandleFunc("/api/v1/graph/stats", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, store.Stats())
	}))

	log.Println("graph: knowledge graph service ready (Neo4j HTTP + in-memory fallback)")
	httpSrv.ListenAndServe()
}

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
