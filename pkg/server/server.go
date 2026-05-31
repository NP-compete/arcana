package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/NP-compete/arcana/pkg/common"
	"github.com/NP-compete/arcana/pkg/logger"
	"github.com/NP-compete/arcana/pkg/metrics"
	"github.com/NP-compete/arcana/pkg/tracing"
	"github.com/google/uuid"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type contextKey string

const RequestIDKey contextKey = "request_id"

type Config struct {
	ServiceName    string
	Port           string
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	IdleTimeout    time.Duration
	MaxHeaderBytes int
	MaxBodyBytes   int64
	DrainTimeout   time.Duration
	DB             *sql.DB
	TLSCertFile    string
	TLSKeyFile     string
}

func (c *Config) defaults() {
	if c.Port == "" {
		c.Port = os.Getenv("PORT")
	}
	if c.Port == "" {
		c.Port = "8080"
	}
	if c.ReadTimeout == 0 {
		c.ReadTimeout = 15 * time.Second
	}
	if c.WriteTimeout == 0 {
		c.WriteTimeout = 30 * time.Second
	}
	if c.IdleTimeout == 0 {
		c.IdleTimeout = 120 * time.Second
	}
	if c.MaxHeaderBytes == 0 {
		c.MaxHeaderBytes = 1 << 20 // 1MB
	}
	if c.MaxBodyBytes == 0 {
		c.MaxBodyBytes = 10 << 20 // 10MB
	}
	if c.DrainTimeout == 0 {
		c.DrainTimeout = 30 * time.Second
	}
	if c.TLSCertFile == "" {
		c.TLSCertFile = os.Getenv("TLS_CERT_FILE")
	}
	if c.TLSKeyFile == "" {
		c.TLSKeyFile = os.Getenv("TLS_KEY_FILE")
	}
}

type Server struct {
	cfg            Config
	mux            *http.ServeMux
	log            *logger.Logger
	httpSrv        *http.Server
	shutdownTracer func(context.Context) error
}

func New(cfg Config) *Server {
	cfg.defaults()
	mux := http.NewServeMux()
	log := logger.New(cfg.ServiceName)

	mux.HandleFunc("/healthz", common.HealthHandler)
	mux.HandleFunc("/readyz", common.ReadinessHandler(cfg.DB))
	mux.Handle("/metrics", metrics.Handler())

	shutdownTracer, err := tracing.Init(context.Background(), cfg.ServiceName)
	if err != nil {
		log.Warn("tracing init failed, continuing without tracing", "error", err.Error())
	}

	s := &Server{
		cfg:            cfg,
		mux:            mux,
		log:            log,
		shutdownTracer: shutdownTracer,
	}
	return s
}

func (s *Server) Handle(pattern string, handler http.Handler) {
	s.mux.Handle(pattern, handler)
}

func (s *Server) HandleFunc(pattern string, handler http.HandlerFunc) {
	s.mux.HandleFunc(pattern, handler)
}

func (s *Server) ListenAndServe() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	handler := s.buildMiddlewareChain(s.mux)

	s.httpSrv = &http.Server{
		Addr:           ":" + s.cfg.Port,
		Handler:        handler,
		ReadTimeout:    s.cfg.ReadTimeout,
		WriteTimeout:   s.cfg.WriteTimeout,
		IdleTimeout:    s.cfg.IdleTimeout,
		MaxHeaderBytes: s.cfg.MaxHeaderBytes,
	}

	errCh := make(chan error, 1)
	go func() {
		s.log.Info("server starting", "port", s.cfg.Port, "tls", s.cfg.TLSCertFile != "")
		var err error
		if s.cfg.TLSCertFile != "" && s.cfg.TLSKeyFile != "" {
			err = s.httpSrv.ListenAndServeTLS(s.cfg.TLSCertFile, s.cfg.TLSKeyFile)
		} else {
			err = s.httpSrv.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("server failed: %w", err)
	case <-ctx.Done():
		s.log.Info("shutdown signal received, draining connections", "timeout", s.cfg.DrainTimeout.String())
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.cfg.DrainTimeout)
	defer cancel()

	if err := s.httpSrv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown error: %w", err)
	}

	if s.shutdownTracer != nil {
		if err := s.shutdownTracer(shutdownCtx); err != nil {
			s.log.Warn("tracer shutdown error", "error", err.Error())
		}
	}

	s.log.Info("server stopped gracefully")
	return nil
}

func (s *Server) buildMiddlewareChain(handler http.Handler) http.Handler {
	h := handler
	h = s.metricsMiddleware(h)
	h = s.requestLoggingMiddleware(h)
	h = s.tracingMiddleware(h)
	h = s.requestIDMiddleware(h)
	h = s.bodySizeLimitMiddleware(h)
	h = s.recoveryMiddleware(h)
	h = s.securityHeadersMiddleware(h)
	return h
}

func (s *Server) securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "0")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) tracingMiddleware(next http.Handler) http.Handler {
	return otelhttp.NewHandler(next, s.cfg.ServiceName,
		otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
			return r.Method + " " + r.URL.Path
		}),
	)
}

func (s *Server) recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				stack := debug.Stack()
				s.log.Error("panic recovered",
					"error", fmt.Sprintf("%v", rec),
					"stack", string(stack),
					"method", r.Method,
					"path", r.URL.Path,
				)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "internal server error",
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (s *Server) requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = uuid.New().String()
		}
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), RequestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) requestLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &statusWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rw, r)
		duration := time.Since(start)

		reqID, _ := r.Context().Value(RequestIDKey).(string)
		s.log.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.statusCode,
			"duration_ms", duration.Milliseconds(),
			"request_id", reqID,
		)
	})
}

func (s *Server) metricsMiddleware(next http.Handler) http.Handler {
	return metrics.InstrumentHandler(next)
}

func (s *Server) bodySizeLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil && r.ContentLength > s.cfg.MaxBodyBytes {
			http.Error(w, `{"error":"request body too large"}`, http.StatusRequestEntityTooLarge)
			return
		}
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, s.cfg.MaxBodyBytes)
		}
		next.ServeHTTP(w, r)
	})
}

type statusWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(RequestIDKey).(string)
	return id
}
