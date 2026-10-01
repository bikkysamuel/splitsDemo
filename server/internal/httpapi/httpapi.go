// Package httpapi adapts HTTP to the domain services: routing, request
// decoding, problem+json errors (RFC 9457) and, later, auth and idempotency.
package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// problemTypeBase prefixes every problem+json type URI (doc 07).
const problemTypeBase = "https://splits.dev/problems/"

const readinessTimeout = 2 * time.Second

// Readiness reports whether the server's dependencies can serve requests.
type Readiness interface {
	Ping(ctx context.Context) error
}

// Deps are the collaborators the HTTP layer calls.
type Deps struct {
	Logger    *slog.Logger
	Readiness Readiness
}

// Server routes HTTP requests to the domain services.
type Server struct {
	mux    *http.ServeMux
	routes []string
	deps   Deps
}

// New registers every route.
func New(deps Deps) *Server {
	s := &Server{mux: http.NewServeMux(), deps: deps}
	s.handle("GET /healthz", s.healthz)
	s.handle("GET /readyz", s.readyz)
	return s
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// Routes lists every registered route as "METHOD /path", in registration
// order, for the startup log (NFR-O3).
func (s *Server) Routes() []string {
	return append([]string(nil), s.routes...)
}

func (s *Server) handle(pattern string, h http.HandlerFunc) {
	s.mux.HandleFunc(pattern, h)
	s.routes = append(s.routes, pattern)
}

// healthz reports liveness: the process is up and serving HTTP.
func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// readyz reports readiness: the database answers.
func (s *Server) readyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), readinessTimeout)
	defer cancel()
	if err := s.deps.Readiness.Ping(ctx); err != nil {
		s.deps.Logger.WarnContext(ctx, "readiness check failed", "error", err)
		writeProblem(w, problem{
			Type:   problemTypeBase + "not-ready",
			Title:  "Service not ready",
			Status: http.StatusServiceUnavailable,
			Detail: "The database is unreachable.",
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

// problem is an RFC 9457 problem details object.
type problem struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail,omitempty"`
}

func writeProblem(w http.ResponseWriter, p problem) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(p.Status)
	_ = json.NewEncoder(w).Encode(p)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
