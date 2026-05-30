package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

type ServiceHealth struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Latency string `json:"latency,omitempty"`
	Port    int    `json:"port"`
	Plane   string `json:"plane"`
}

type SystemHealth struct {
	Platform string          `json:"platform"`
	Version  string          `json:"version"`
	Uptime   string          `json:"uptime"`
	Services []ServiceHealth `json:"services"`
}

type ServiceRoute struct {
	Name    string
	EnvKey  string
	Default string
	Port    int
	Prefix  string
	Plane   string
}

var startTime = time.Now()

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func checkTCP(host string, port int, timeout time.Duration) (bool, time.Duration) {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), timeout)
	elapsed := time.Since(start)
	if err != nil {
		return false, elapsed
	}
	conn.Close()
	return true, elapsed
}

var backingServices = []struct {
	name  string
	host  string
	port  int
	plane string
}{
	{"PostgreSQL", "POSTGRES_HOST", 5432, "infra"},
	{"Redis", "REDIS_HOST", 6379, "infra"},
	{"Temporal", "TEMPORAL_HOST", 7233, "infra"},
	{"MinIO", "MINIO_HOST", 9000, "infra"},
	{"NATS", "NATS_HOST", 4222, "infra"},
}

var serviceRoutes = []ServiceRoute{
	// Agent Plane
	{Name: "engine", EnvKey: "ENGINE_HOST", Default: "arcana-engine", Port: 8081, Prefix: "/api/v1/tasks", Plane: "agent"},
	{Name: "blueprint", EnvKey: "BLUEPRINT_HOST", Default: "arcana-blueprint", Port: 8088, Prefix: "/api/v1/blueprints", Plane: "agent"},
	{Name: "oracle", EnvKey: "ORACLE_HOST", Default: "arcana-oracle", Port: 8089, Prefix: "/api/v1/predict", Plane: "agent"},
	{Name: "mesh", EnvKey: "MESH_HOST", Default: "arcana-mesh", Port: 8083, Prefix: "/api/v1/agents", Plane: "agent"},
	{Name: "memory", EnvKey: "MEMORY_HOST", Default: "arcana-memory", Port: 8087, Prefix: "/api/v1/memory", Plane: "agent"},

	// Data Plane
	{Name: "codex-router", EnvKey: "CODEX_ROUTER_HOST", Default: "arcana-codex-router", Port: 8090, Prefix: "/api/v1/search", Plane: "data"},
	{Name: "codex-ingestor", EnvKey: "CODEX_INGESTOR_HOST", Default: "arcana-codex-ingestor", Port: 8092, Prefix: "/api/v1/ingest", Plane: "data"},
	{Name: "connectors", EnvKey: "CONNECTORS_HOST", Default: "arcana-connectors", Port: 8094, Prefix: "/api/v1/connectors", Plane: "data"},
	{Name: "graph", EnvKey: "GRAPH_HOST", Default: "arcana-graph", Port: 8095, Prefix: "/api/v1/graph", Plane: "data"},

	// Tool Plane
	{Name: "tools", EnvKey: "TOOLS_HOST", Default: "arcana-tools", Port: 8096, Prefix: "/api/v1/tools", Plane: "tool"},
	{Name: "sandbox", EnvKey: "SANDBOX_HOST", Default: "arcana-sandbox", Port: 8097, Prefix: "/api/v1/exec", Plane: "tool"},

	// Model Plane
	{Name: "forge", EnvKey: "FORGE_HOST", Default: "arcana-forge", Port: 8098, Prefix: "/api/v1/experiments", Plane: "model"},
	{Name: "models", EnvKey: "MODELS_HOST", Default: "arcana-models", Port: 8099, Prefix: "/api/v1/models", Plane: "model"},

	// Govern Plane
	{Name: "ward", EnvKey: "WARD_HOST", Default: "arcana-ward", Port: 8086, Prefix: "/api/v1/check", Plane: "govern"},
	{Name: "audit", EnvKey: "AUDIT_HOST", Default: "arcana-audit", Port: 8100, Prefix: "/api/v1/audit", Plane: "govern"},

	// Quality Plane
	{Name: "probe", EnvKey: "PROBE_HOST", Default: "arcana-probe", Port: 8101, Prefix: "/api/v1/eval", Plane: "quality"},
	{Name: "annotate", EnvKey: "ANNOTATE_HOST", Default: "arcana-annotate", Port: 8102, Prefix: "/api/v1/annotations", Plane: "quality"},

	// Ops Plane
	{Name: "skills", EnvKey: "SKILLS_HOST", Default: "arcana-skills", Port: 8085, Prefix: "/api/v1/skills", Plane: "ops"},
	{Name: "scheduler", EnvKey: "SCHEDULER_HOST", Default: "arcana-scheduler", Port: 8103, Prefix: "/api/v1/scheduler", Plane: "ops"},
	{Name: "registry", EnvKey: "REGISTRY_HOST", Default: "arcana-registry", Port: 8104, Prefix: "/api/v1/catalog", Plane: "ops"},
	{Name: "finops", EnvKey: "FINOPS_HOST", Default: "arcana-finops", Port: 8105, Prefix: "/api/v1/costs", Plane: "ops"},
	{Name: "gitops", EnvKey: "GITOPS_HOST", Default: "arcana-gitops", Port: 8106, Prefix: "/api/v1/promotions", Plane: "ops"},
}

func cors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	type checkTarget struct {
		name  string
		host  string
		port  int
		plane string
	}

	var targets []checkTarget
	for _, bs := range backingServices {
		targets = append(targets, checkTarget{
			name:  bs.name,
			host:  envOr(bs.host, "localhost"),
			port:  bs.port,
			plane: bs.plane,
		})
	}
	for _, sr := range serviceRoutes {
		targets = append(targets, checkTarget{
			name:  sr.Name,
			host:  envOr(sr.EnvKey, sr.Default),
			port:  sr.Port,
			plane: sr.Plane,
		})
	}

	results := make([]ServiceHealth, len(targets))
	var wg sync.WaitGroup

	for i, t := range targets {
		wg.Add(1)
		go func(idx int, name, host string, port int, plane string) {
			defer wg.Done()
			ok, latency := checkTCP(host, port, 2*time.Second)
			status := "healthy"
			if !ok {
				status = "unreachable"
			}
			results[idx] = ServiceHealth{
				Name:    name,
				Status:  status,
				Latency: latency.Round(time.Millisecond).String(),
				Port:    port,
				Plane:   plane,
			}
		}(i, t.name, t.host, t.port, t.plane)
	}
	wg.Wait()

	resp := SystemHealth{
		Platform: "arcana",
		Version:  "0.1.0",
		Uptime:   time.Since(startTime).Round(time.Second).String(),
		Services: results,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(resp)
}

func routesHandler(w http.ResponseWriter, _ *http.Request) {
	type RouteInfo struct {
		Name   string `json:"name"`
		Prefix string `json:"prefix"`
		Plane  string `json:"plane"`
		Port   int    `json:"port"`
	}
	routes := make([]RouteInfo, len(serviceRoutes))
	for i, sr := range serviceRoutes {
		routes[i] = RouteInfo{Name: sr.Name, Prefix: sr.Prefix, Plane: sr.Plane, Port: sr.Port}
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(routes)
}

func makeProxy(host string, port int) http.Handler {
	target, _ := url.Parse(fmt.Sprintf("http://%s:%d", host, port))
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "service_unavailable",
			"message": fmt.Sprintf("upstream %s:%d unreachable: %v", host, port, err),
		})
	}
	return proxy
}

func main() {
	port := envOr("PORT", "8080")

	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})

	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})

	mux.HandleFunc("/api/v1/version", cors(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"name":"arcana","version":"0.1.0","services":28,"crds":16,"planes":8}`)
	}))

	mux.HandleFunc("/api/v1/health", cors(healthHandler))
	mux.HandleFunc("/api/v1/routes", cors(routesHandler))

	for _, sr := range serviceRoutes {
		host := envOr(sr.EnvKey, sr.Default)
		proxy := makeProxy(host, sr.Port)
		prefix := sr.Prefix
		mux.HandleFunc(prefix+"/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			proxy.ServeHTTP(w, r)
		})
		if !strings.HasSuffix(prefix, "/") {
			mux.HandleFunc(prefix, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Access-Control-Allow-Origin", "*")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				if r.Method == http.MethodOptions {
					w.WriteHeader(http.StatusNoContent)
					return
				}
				proxy.ServeHTTP(w, r)
			})
		}
	}

	log.Printf("arcana-api gateway starting on :%s", port)
	log.Printf("routing %d services across 8 planes", len(serviceRoutes))
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
