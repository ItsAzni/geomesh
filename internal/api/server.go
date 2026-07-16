package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/itsazni/geomesh/internal/health"
	"github.com/itsazni/geomesh/internal/zone"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server provides an optional REST API for GeoMesh.
type Server struct {
	addr     string
	registry *zone.Registry
	store    *health.Store
	reloadFn func() error
	mux      *http.ServeMux
	httpSrv  *http.Server
}

// NewServer initializes a new API Server.
// addr: e.g., ":8080". reloadFn can be nil if reloading is not required.
func NewServer(addr string, reg *zone.Registry, store *health.Store, reloadFn func() error) *Server {
	s := &Server{
		addr:     addr,
		registry: reg,
		store:    store,
		reloadFn: reloadFn,
		mux:      http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("/api/status", s.loggingMiddleware(handleStatus))
	s.mux.Handle("/api/zones", s.loggingMiddleware(handleZones(s.registry)))
	s.mux.Handle("/api/health", s.loggingMiddleware(handleHealth(s.store)))
	s.mux.Handle("/api/reload", s.loggingMiddleware(handleReload(s.reloadFn)))
	s.mux.Handle("/metrics", promhttp.Handler())
}

// loggingMiddleware wraps an HTTP handler to log every API request.
func (s *Server) loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next(rw, r)
		elapsed := time.Since(start)
		slog.Info("api request",
			"method", r.Method,
			"path", r.URL.Path,
			"remote", r.RemoteAddr,
			"status", rw.statusCode,
			"duration_ms", elapsed.Milliseconds(),
		)
	}
}

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Handler returns the http.Handler for testing purposes (without starting the server).
func (s *Server) Handler() http.Handler {
	return s.mux
}

// Start launches the HTTP API server. This function blocks.
func (s *Server) Start() error {
	s.httpSrv = &http.Server{
		Addr:    s.addr,
		Handler: s.mux,
	}
	slog.Info("API server listening", "addr", s.addr)
	return s.httpSrv.ListenAndServe()
}

// Shutdown elegantly terminates the API server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpSrv != nil {
		slog.Info("API server shutting down")
		return s.httpSrv.Shutdown(ctx)
	}
	return nil
}
