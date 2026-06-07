package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/NP-compete/arcana/pkg/server"
)

type PeerConfig struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Endpoint string `json:"endpoint"`
	Secret   string `json:"secret"`
	Status   string `json:"status"`
}

type PeerMessage struct {
	FromPeer  string                 `json:"from_peer"`
	ToPeer    string                 `json:"to_peer"`
	AgentName string                 `json:"agent_name"`
	Payload   map[string]interface{} `json:"payload"`
	Signature string                 `json:"signature"`
	Timestamp string                 `json:"timestamp"`
}

type PeerStore struct {
	mu    sync.RWMutex
	peers map[string]*PeerConfig
}

func NewPeerStore() *PeerStore {
	return &PeerStore{peers: make(map[string]*PeerConfig)}
}

func main() {
	instanceID := os.Getenv("ARCANA_INSTANCE_ID")
	if instanceID == "" {
		instanceID = "arcana-local"
	}

	httpSrv := server.New(server.Config{
		ServiceName: "wire",
		Port:        "8114",
	})

	store := NewPeerStore()

	httpSrv.HandleFunc("/api/v1/wire/peers", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			store.mu.RLock()
			peers := make([]PeerConfig, 0)
			for _, p := range store.peers {
				safe := *p
				safe.Secret = "***"
				peers = append(peers, safe)
			}
			store.mu.RUnlock()
			writeJSON(w, http.StatusOK, map[string]interface{}{"peers": peers, "instance_id": instanceID})
		case http.MethodPost:
			var peer PeerConfig
			json.NewDecoder(r.Body).Decode(&peer)
			if peer.Name == "" || peer.Endpoint == "" || peer.Secret == "" {
				writeError(w, http.StatusBadRequest, "name, endpoint, and secret required")
				return
			}
			peer.ID = fmt.Sprintf("peer-%d", time.Now().UnixNano())
			peer.Status = "connected"
			store.mu.Lock()
			store.peers[peer.ID] = &peer
			store.mu.Unlock()
			log.Printf("wire: peer registered: %s (%s)", peer.Name, peer.Endpoint)
			writeJSON(w, http.StatusCreated, peer)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}))

	httpSrv.HandleFunc("/api/v1/wire/send", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "POST required")
			return
		}

		var msg PeerMessage
		json.NewDecoder(r.Body).Decode(&msg)

		store.mu.RLock()
		peer, ok := store.peers[msg.ToPeer]
		store.mu.RUnlock()
		if !ok {
			writeError(w, http.StatusNotFound, "peer not found")
			return
		}

		msg.FromPeer = instanceID
		msg.Timestamp = time.Now().UTC().Format(time.RFC3339)

		payload, _ := json.Marshal(msg.Payload)
		mac := hmac.New(sha256.New, []byte(peer.Secret))
		mac.Write(payload)
		msg.Signature = hex.EncodeToString(mac.Sum(nil))

		body, _ := json.Marshal(msg)
		client := &http.Client{Timeout: 10 * time.Second}
		req, _ := http.NewRequest("POST", peer.Endpoint+"/api/v1/wire/receive", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Arcana-Signature", msg.Signature)
		resp, err := client.Do(req)
		if err != nil {
			writeError(w, http.StatusBadGateway, fmt.Sprintf("peer unreachable: %v", err))
			return
		}
		defer resp.Body.Close()
		log.Printf("wire: sent message from %s to peer %s (agent=%s)", instanceID, peer.Name, msg.AgentName)
		writeJSON(w, http.StatusOK, map[string]string{"status": "sent", "signature": msg.Signature})
	}))

	httpSrv.HandleFunc("/api/v1/wire/receive", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "POST required")
			return
		}
		var msg PeerMessage
		json.NewDecoder(r.Body).Decode(&msg)
		log.Printf("wire: received message from peer %s (agent=%s)", msg.FromPeer, msg.AgentName)
		writeJSON(w, http.StatusOK, map[string]string{"status": "received"})
	}))

	httpSrv.ListenAndServe()
}

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", os.Getenv("CORS_ORIGIN"))
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
