// Package httpapi adapts HTTP to the domain services. Routing, request
// decoding and response encoding are generated from api/openapi.yaml into
// apigen (ADR-0012); this package implements the generated strict-server
// interface and maps outcomes to problem+json errors (RFC 9457).
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/bikkysamuel/splitsDemo/server/internal/httpapi/apigen"
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

// Server serves the generated API routes.
type Server struct {
	handler http.Handler
	routes  []string
	deps    Deps
}

var _ apigen.StrictServerInterface = (*Server)(nil)

// New registers every route in the contract.
func New(deps Deps) *Server {
	s := &Server{deps: deps}
	mux := &routeRecorder{ServeMux: http.NewServeMux()}
	strict := apigen.NewStrictHandlerWithOptions(s, nil, apigen.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  s.requestError,
		ResponseErrorHandlerFunc: s.responseError,
	})
	s.handler = apigen.HandlerWithOptions(strict, apigen.StdHTTPServerOptions{
		BaseRouter:       mux,
		ErrorHandlerFunc: s.requestError,
	})
	s.routes = mux.patterns
	return s
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handler.ServeHTTP(w, r)
}

// Routes lists every registered route as "METHOD /path", in registration
// order, for the startup log (NFR-O3).
func (s *Server) Routes() []string {
	return append([]string(nil), s.routes...)
}

// GetHealthz reports liveness: the process is up and serving HTTP.
func (s *Server) GetHealthz(context.Context, apigen.GetHealthzRequestObject) (apigen.GetHealthzResponseObject, error) {
	return apigen.GetHealthz200JSONResponse{Status: apigen.HealthStatusStatusOk}, nil
}

// GetReadyz reports readiness: the database answers.
func (s *Server) GetReadyz(ctx context.Context, _ apigen.GetReadyzRequestObject) (apigen.GetReadyzResponseObject, error) {
	ctx, cancel := context.WithTimeout(ctx, readinessTimeout)
	defer cancel()
	if err := s.deps.Readiness.Ping(ctx); err != nil {
		// Database driver errors name the host, role and database; logs carry
		// a cause category only (doc 08).
		cause := "unreachable"
		if errors.Is(err, context.DeadlineExceeded) {
			cause = "timeout"
		}
		s.deps.Logger.WarnContext(ctx, "readiness check failed", "cause", cause)
		return apigen.GetReadyz503ApplicationProblemPlusJSONResponse{
			NotReadyApplicationProblemPlusJSONResponse: apigen.NotReadyApplicationProblemPlusJSONResponse(
				newProblem("not-ready", "Service not ready", http.StatusServiceUnavailable, "The database is unreachable."),
			),
		}, nil
	}
	return apigen.GetReadyz200JSONResponse{Status: apigen.ReadinessStatusStatusReady}, nil
}

// requestError answers a request the generated code could not decode. The
// decoding error stays out of the response and the logs: it may echo input.
func (s *Server) requestError(w http.ResponseWriter, r *http.Request, _ error) {
	s.deps.Logger.InfoContext(r.Context(), "invalid request", "method", r.Method, "pattern", r.Pattern)
	writeProblem(w, newProblem("invalid-request", "Invalid request", http.StatusBadRequest, ""))
}

// responseError answers when a handler fails unexpectedly.
func (s *Server) responseError(w http.ResponseWriter, r *http.Request, err error) {
	s.deps.Logger.ErrorContext(r.Context(), "handler failed", "method", r.Method, "pattern", r.Pattern, "error", err)
	writeProblem(w, newProblem("internal", "Internal error", http.StatusInternalServerError, ""))
}

func newProblem(slug, title string, status int, detail string) apigen.Problem {
	p := apigen.Problem{Type: problemTypeBase + slug, Title: title, Status: status}
	if detail != "" {
		p.Detail = &detail
	}
	return p
}

func writeProblem(w http.ResponseWriter, p apigen.Problem) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(p.Status)
	_ = json.NewEncoder(w).Encode(p)
}

// routeRecorder is the ServeMux the generated code registers routes on; it
// remembers each pattern for Routes.
type routeRecorder struct {
	*http.ServeMux
	patterns []string
}

func (m *routeRecorder) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	m.ServeMux.HandleFunc(pattern, handler)
	m.patterns = append(m.patterns, pattern)
}
