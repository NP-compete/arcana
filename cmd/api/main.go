package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
)

type ServiceHealth struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Latency string `json:"latency,omitempty"`
	Port    int    `json:"port"`
}

type SystemHealth struct {
	Platform string          `json:"platform"`
	Version  string          `json:"version"`
	Uptime   string          `json:"uptime"`
	Services []ServiceHealth `json:"services"`
}

var startTime = time.Now()

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

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	services := []struct {
		name string
		host string
		port int
	}{
		{"PostgreSQL", "localhost", 5432},
		{"Redis", "localhost", 6379},
		{"Temporal", "localhost", 7233},
		{"MinIO", "localhost", 9000},
		{"NATS", "localhost", 4222},
	}

	results := make([]ServiceHealth, len(services))
	var wg sync.WaitGroup

	for i, svc := range services {
		wg.Add(1)
		go func(idx int, name, host string, port int) {
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
			}
		}(i, svc.name, svc.host, svc.port)
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

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})

	http.HandleFunc("/api/v1/version", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		fmt.Fprint(w, `{"name":"arcana","version":"0.1.0"}`)
	})

	http.HandleFunc("/api/v1/health", healthHandler)

	log.Printf("arcana-api starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
