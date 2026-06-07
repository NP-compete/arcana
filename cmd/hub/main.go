package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/NP-compete/arcana/pkg/server"
)

type HubAsset struct {
	ID          string   `json:"id"`
	Type        string   `json:"type"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Author      string   `json:"author"`
	Version     string   `json:"version"`
	Downloads   int      `json:"downloads"`
	Rating      float64  `json:"rating"`
	RatingCount int      `json:"rating_count"`
	Tags        []string `json:"tags"`
	Badge       string   `json:"badge"`
	PublishedAt string   `json:"published_at"`
}

type HubStore struct {
	mu     sync.RWMutex
	assets map[string]*HubAsset
}

func NewHubStore() *HubStore {
	hs := &HubStore{assets: make(map[string]*HubAsset)}
	hs.seedDefaults()
	return hs
}

func (hs *HubStore) seedDefaults() {
	defaults := []HubAsset{
		{Type: "skill", Name: "web-search", Description: "Search the web using multiple engines", Author: "arcana-team", Version: "1.0.0", Badge: "gold", Tags: []string{"search", "web", "research"}},
		{Type: "skill", Name: "code-review", Description: "Review code for bugs, style, and security", Author: "arcana-team", Version: "1.0.0", Badge: "gold", Tags: []string{"code", "review", "security"}},
		{Type: "skill", Name: "summarize", Description: "Summarize long documents into concise briefs", Author: "arcana-team", Version: "1.0.0", Badge: "silver", Tags: []string{"text", "summary", "nlp"}},
		{Type: "agent", Name: "research-assistant", Description: "Multi-step research pipeline with citations", Author: "arcana-team", Version: "1.0.0", Badge: "gold", Tags: []string{"research", "academic", "citations"}},
		{Type: "agent", Name: "support-triage", Description: "Customer support ticket triage and routing", Author: "community", Version: "1.0.0", Badge: "silver", Tags: []string{"support", "tickets", "triage"}},
		{Type: "hand", Name: "daily-researcher", Description: "Autonomous daily research briefing", Author: "arcana-team", Version: "1.0.0", Badge: "silver", Tags: []string{"research", "autonomous", "daily"}},
		{Type: "hand", Name: "lead-generator", Description: "Automated lead generation and enrichment", Author: "community", Version: "1.0.0", Badge: "bronze", Tags: []string{"sales", "leads", "automation"}},
		{Type: "blueprint", Name: "research-pipeline", Description: "3-agent research → review → publish pipeline", Author: "arcana-team", Version: "1.0.0", Badge: "gold", Tags: []string{"pipeline", "research", "multi-agent"}},
	}
	for i, a := range defaults {
		a.ID = fmt.Sprintf("hub-%d", i+1)
		a.Downloads = (i + 1) * 47
		a.Rating = 4.0 + float64(i%3)*0.3
		a.RatingCount = (i + 1) * 12
		a.PublishedAt = time.Now().AddDate(0, -(i + 1), 0).Format(time.RFC3339)
		hs.assets[a.ID] = &a
	}
}

func (hs *HubStore) List(assetType, query string) []HubAsset {
	hs.mu.RLock()
	defer hs.mu.RUnlock()
	result := make([]HubAsset, 0)
	for _, a := range hs.assets {
		if assetType != "" && a.Type != assetType {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(a.Name+a.Description), strings.ToLower(query)) {
			continue
		}
		result = append(result, *a)
	}
	return result
}

func (hs *HubStore) Get(id string) *HubAsset {
	hs.mu.RLock()
	defer hs.mu.RUnlock()
	return hs.assets[id]
}

func (hs *HubStore) Publish(asset HubAsset) {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	asset.ID = fmt.Sprintf("hub-%d", time.Now().UnixNano())
	asset.PublishedAt = time.Now().UTC().Format(time.RFC3339)
	if asset.Badge == "" {
		asset.Badge = "untested"
	}
	hs.assets[asset.ID] = &asset
}

func (hs *HubStore) Download(id string) bool {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	if a, ok := hs.assets[id]; ok {
		a.Downloads++
		return true
	}
	return false
}

func (hs *HubStore) Rate(id string, rating float64) bool {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	a, ok := hs.assets[id]
	if !ok {
		return false
	}
	totalRating := a.Rating * float64(a.RatingCount)
	a.RatingCount++
	a.Rating = (totalRating + rating) / float64(a.RatingCount)
	return true
}

func main() {
	httpSrv := server.New(server.Config{
		ServiceName: "hub",
		Port:        "8115",
	})

	store := NewHubStore()

	httpSrv.HandleFunc("/api/v1/hub/assets", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			assetType := r.URL.Query().Get("type")
			query := r.URL.Query().Get("q")
			assets := store.List(assetType, query)
			writeJSON(w, http.StatusOK, map[string]interface{}{"assets": assets, "total": len(assets)})
		case http.MethodPost:
			var asset HubAsset
			json.NewDecoder(r.Body).Decode(&asset)
			if asset.Name == "" || asset.Type == "" {
				writeError(w, http.StatusBadRequest, "name and type required")
				return
			}
			store.Publish(asset)
			log.Printf("hub: published %s/%s by %s", asset.Type, asset.Name, asset.Author)
			writeJSON(w, http.StatusCreated, asset)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}))

	httpSrv.HandleFunc("/api/v1/hub/assets/", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/v1/hub/assets/")
		id = strings.TrimSuffix(id, "/download")
		id = strings.TrimSuffix(id, "/rate")

		if strings.HasSuffix(r.URL.Path, "/download") {
			if store.Download(id) {
				writeJSON(w, http.StatusOK, map[string]string{"status": "downloaded"})
			} else {
				writeError(w, http.StatusNotFound, "asset not found")
			}
			return
		}

		if strings.HasSuffix(r.URL.Path, "/rate") {
			var req struct{ Rating float64 `json:"rating"` }
			json.NewDecoder(r.Body).Decode(&req)
			if store.Rate(id, req.Rating) {
				writeJSON(w, http.StatusOK, map[string]string{"status": "rated"})
			} else {
				writeError(w, http.StatusNotFound, "asset not found")
			}
			return
		}

		asset := store.Get(id)
		if asset == nil {
			writeError(w, http.StatusNotFound, "asset not found")
			return
		}
		writeJSON(w, http.StatusOK, asset)
	}))

	httpSrv.HandleFunc("/api/v1/hub/stats", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		assets := store.List("", "")
		totalDownloads := 0
		byType := map[string]int{}
		for _, a := range assets {
			totalDownloads += a.Downloads
			byType[a.Type]++
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"total_assets":    len(assets),
			"total_downloads": totalDownloads,
			"by_type":         byType,
		})
	}))

	log.Println("hub: ArcanaHub community marketplace ready")
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
