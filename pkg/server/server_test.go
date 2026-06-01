package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHealthEndpointsRegistered(t *testing.T) {
	srv := New(Config{ServiceName: "test", Port: "0"})

	for _, path := range []string{"/healthz", "/readyz", "/metrics"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		srv.mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("%s: got %d, want 200", path, rec.Code)
		}
	}
}

func TestPanicRecovery(t *testing.T) {
	srv := New(Config{ServiceName: "test", Port: "0"})
	srv.HandleFunc("/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	handler := srv.buildMiddlewareChain(srv.mux)
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("panic handler: got %d, want 500", rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["error"] != "internal server error" {
		t.Errorf("error body: got %q, want %q", body["error"], "internal server error")
	}
}

func TestRequestIDPropagation(t *testing.T) {
	srv := New(Config{ServiceName: "test", Port: "0"})

	var capturedID string
	srv.HandleFunc("/id", func(w http.ResponseWriter, r *http.Request) {
		capturedID = RequestID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := srv.buildMiddlewareChain(srv.mux)

	t.Run("generates ID when missing", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/id", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		respID := rec.Header().Get("X-Request-ID")
		if respID == "" {
			t.Error("expected X-Request-ID header to be set")
		}
		if capturedID == "" {
			t.Error("expected request ID in context")
		}
		if respID != capturedID {
			t.Errorf("header ID %q != context ID %q", respID, capturedID)
		}
	})

	t.Run("preserves existing ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/id", nil)
		req.Header.Set("X-Request-ID", "custom-123")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Header().Get("X-Request-ID") != "custom-123" {
			t.Errorf("expected preserved ID, got %q", rec.Header().Get("X-Request-ID"))
		}
		if capturedID != "custom-123" {
			t.Errorf("context ID: got %q, want %q", capturedID, "custom-123")
		}
	})
}

func TestBodySizeLimit(t *testing.T) {
	srv := New(Config{ServiceName: "test", Port: "0", MaxBodyBytes: 100})
	srv.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "body too large", http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	handler := srv.buildMiddlewareChain(srv.mux)

	t.Run("allows small body", func(t *testing.T) {
		body := bytes.NewReader(make([]byte, 50))
		req := httptest.NewRequest(http.MethodPost, "/upload", body)
		req.ContentLength = 50
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("small body: got %d, want 200", rec.Code)
		}
	})

	t.Run("rejects oversized content-length", func(t *testing.T) {
		body := bytes.NewReader(make([]byte, 200))
		req := httptest.NewRequest(http.MethodPost, "/upload", body)
		req.ContentLength = 200
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusRequestEntityTooLarge {
			t.Errorf("large body: got %d, want 413", rec.Code)
		}
	})
}

func TestRequestLogging(t *testing.T) {
	srv := New(Config{ServiceName: "test-svc", Port: "0"})
	srv.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := srv.buildMiddlewareChain(srv.mux)
	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("got %d, want 200", rec.Code)
	}
}

func TestGracefulShutdown(t *testing.T) {
	srv := New(Config{ServiceName: "test", Port: "0", DrainTimeout: 2 * time.Second})

	handler := srv.buildMiddlewareChain(srv.mux)
	httpSrv := &http.Server{
		Addr:    ":0",
		Handler: handler,
	}

	go func() {
		_ = httpSrv.ListenAndServe()
	}()

	time.Sleep(50 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := httpSrv.Shutdown(ctx); err != nil {
		t.Errorf("shutdown failed: %v", err)
	}
}

func TestConfigDefaults(t *testing.T) {
	cfg := Config{ServiceName: "test"}
	cfg.defaults()

	if cfg.ReadTimeout != 15*time.Second {
		t.Errorf("ReadTimeout: got %v, want 15s", cfg.ReadTimeout)
	}
	if cfg.WriteTimeout != 30*time.Second {
		t.Errorf("WriteTimeout: got %v, want 30s", cfg.WriteTimeout)
	}
	if cfg.IdleTimeout != 120*time.Second {
		t.Errorf("IdleTimeout: got %v, want 120s", cfg.IdleTimeout)
	}
	if cfg.MaxHeaderBytes != 1<<20 {
		t.Errorf("MaxHeaderBytes: got %d, want 1MB", cfg.MaxHeaderBytes)
	}
	if cfg.MaxBodyBytes != 10<<20 {
		t.Errorf("MaxBodyBytes: got %d, want 10MB", cfg.MaxBodyBytes)
	}
	if cfg.DrainTimeout != 30*time.Second {
		t.Errorf("DrainTimeout: got %v, want 30s", cfg.DrainTimeout)
	}
}

func TestMiddlewareChainOrder(t *testing.T) {
	srv := New(Config{ServiceName: "test", Port: "0"})

	var order []string
	srv.HandleFunc("/order", func(w http.ResponseWriter, r *http.Request) {
		if RequestID(r.Context()) != "" {
			order = append(order, "handler-with-id")
		}
		w.WriteHeader(http.StatusOK)
	})

	handler := srv.buildMiddlewareChain(srv.mux)
	req := httptest.NewRequest(http.MethodGet, "/order", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if len(order) == 0 || !strings.Contains(order[0], "handler-with-id") {
		t.Error("handler should have received request ID from middleware")
	}
}

func TestRateLimiting_AllowsNormalTraffic(t *testing.T) {
	srv := New(Config{ServiceName: "test", Port: "0", RateLimitPerSecond: 50})
	srv.HandleFunc("/ok", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := srv.buildMiddlewareChain(srv.mux)

	// Send a small number of requests well within the limit
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/ok", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("request %d: got %d, want 200", i, rec.Code)
		}
	}
}

func TestRateLimiting_BlocksExcessTraffic(t *testing.T) {
	// Set a very low rate limit so we can exhaust the bucket quickly
	srv := New(Config{ServiceName: "test", Port: "0", RateLimitPerSecond: 3})
	srv.HandleFunc("/limited", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := srv.buildMiddlewareChain(srv.mux)

	var blocked int
	// Send more requests than the bucket capacity in rapid succession
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/limited", nil)
		req.RemoteAddr = "10.0.0.2:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code == http.StatusTooManyRequests {
			blocked++
		}
	}

	if blocked == 0 {
		t.Error("expected at least one request to be rate limited (429)")
	}
}

func TestRateLimiting_Disabled(t *testing.T) {
	// Negative RateLimitPerSecond disables rate limiting
	srv := New(Config{ServiceName: "test", Port: "0", RateLimitPerSecond: -1})
	srv.HandleFunc("/nolimit", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := srv.buildMiddlewareChain(srv.mux)

	for i := 0; i < 100; i++ {
		req := httptest.NewRequest(http.MethodGet, "/nolimit", nil)
		req.RemoteAddr = "10.0.0.3:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("request %d: got %d, want 200 (rate limiting should be disabled)", i, rec.Code)
		}
	}
}

func TestRateLimiting_RetryAfterHeader(t *testing.T) {
	srv := New(Config{ServiceName: "test", Port: "0", RateLimitPerSecond: 2})
	srv.HandleFunc("/retry", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := srv.buildMiddlewareChain(srv.mux)

	// Exhaust the bucket
	for i := 0; i < 20; i++ {
		req := httptest.NewRequest(http.MethodGet, "/retry", nil)
		req.RemoteAddr = "10.0.0.4:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code == http.StatusTooManyRequests {
			retryAfter := rec.Header().Get("Retry-After")
			if retryAfter == "" {
				t.Error("429 response missing Retry-After header")
			}
			if retryAfter != "1" {
				t.Errorf("Retry-After: got %q, want %q", retryAfter, "1")
			}

			// Verify response body contains error message
			var body map[string]string
			if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
				t.Fatalf("decode 429 body: %v", err)
			}
			if body["error"] != "rate limit exceeded" {
				t.Errorf("error message: got %q, want %q", body["error"], "rate limit exceeded")
			}
			return
		}
	}
	t.Error("never received a 429 response")
}

func TestRateLimiting_PerIP(t *testing.T) {
	// Small bucket so one IP exhausts quickly
	srv := New(Config{ServiceName: "test", Port: "0", RateLimitPerSecond: 3})
	srv.HandleFunc("/perip", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := srv.buildMiddlewareChain(srv.mux)

	// Exhaust the bucket for IP-A
	var ipABlocked int
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/perip", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code == http.StatusTooManyRequests {
			ipABlocked++
		}
	}

	if ipABlocked == 0 {
		t.Error("expected IP-A to be rate limited")
	}

	// IP-B should still have its own fresh bucket
	var ipBBlocked int
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/perip", nil)
		req.RemoteAddr = "192.168.1.2:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code == http.StatusTooManyRequests {
			ipBBlocked++
		}
	}

	if ipBBlocked > 0 {
		t.Error("IP-B should not be rate limited - it has a separate bucket from IP-A")
	}
}
