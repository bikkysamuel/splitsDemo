package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/bikkysamuel/splitsDemo/server/internal/auth"
	"github.com/bikkysamuel/splitsDemo/server/internal/platform"
)

// maxBodyBytes is the request body limit (doc 08).
const maxBodyBytes = 64 << 10

// publicRoutes need no access token: the health checks and the anonymous
// auth endpoints. Every other route requires one, so a new route is
// protected unless it is listed here (TestBearerAuthMatchesTheContract
// keeps this list and the contract's bearerAuth in step).
var publicRoutes = map[string]bool{
	"GET /healthz":                      true,
	"GET /readyz":                       true,
	"POST /v1/auth/signup":              true,
	"POST /v1/auth/verify-email":        true,
	"POST /v1/auth/verify-email/resend": true,
	"POST /v1/auth/signin":              true,
	"POST /v1/auth/refresh":             true,
}

// unverifiedRoutes are the protected routes a User whose email is not yet
// verified may call (FR-A2): only the launch check.
var unverifiedRoutes = map[string]bool{
	"GET /v1/me":            true,
	"POST /v1/auth/signout": true,
}

var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// withRequestID keeps the client's X-Request-ID when valid, or makes one,
// and puts it on the response and in the context for every log line.
func (s *Server) withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if !requestIDPattern.MatchString(id) {
			id = s.deps.IDs.New().String()
		}
		w.Header().Set("X-Request-ID", id)
		ctx := platform.WithRequestID(r.Context(), id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(ctx, clientIPKey{}, remoteIP(r))))
	})
}

// logRequests writes one line per request: route pattern (never the raw
// path, which may later carry tokens), status, duration and the User's ID
// when signed in. Logs carry IDs only (doc 08).
func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		info := &requestInfo{}
		next.ServeHTTP(rec, r.WithContext(context.WithValue(r.Context(), requestInfoKey{}, info)))
		attrs := []any{
			"method", r.Method,
			"pattern", info.pattern,
			"status", rec.status,
			"duration_ms", time.Since(start).Milliseconds(),
		}
		if info.userID != nil {
			attrs = append(attrs, "user_id", info.userID.String())
		}
		s.deps.Logger.InfoContext(r.Context(), "request", attrs...)
	})
}

// recoverPanics turns a handler panic into a 500 problem and a log line
// naming the panic's type only.
func (s *Server) recoverPanics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		defer func() {
			v := recover()
			if v == nil {
				return
			}
			if v == http.ErrAbortHandler { //nolint:errorlint // the sentinel is panicked as is
				panic(v)
			}
			s.deps.Logger.ErrorContext(r.Context(), "handler panicked",
				"method", r.Method, "panic_type", fmt.Sprintf("%T", v))
			if !rec.wroteHeader {
				writeProblem(w, problemInternal.problem(""))
			}
		}()
		next.ServeHTTP(rec, r)
	})
}

// limitBody refuses bodies over maxBodyBytes (doc 08): at once when the
// declared length is too big, otherwise when reading passes the limit.
func (s *Server) limitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength > maxBodyBytes {
			writeProblem(w, problemRequestTooLarge.problem(""))
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		next.ServeHTTP(w, r)
	})
}

// authenticate checks the Bearer token of every route not in publicRoutes
// (ADR-0011) and puts the Principal in the context. An unverified User may
// call unverifiedRoutes only.
func (s *Server) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, pattern := s.mux.Handler(r)
		if info, ok := r.Context().Value(requestInfoKey{}).(*requestInfo); ok {
			info.pattern = pattern
		}
		if pattern == "" || publicRoutes[pattern] {
			// Unknown routes fall through to the mux's 404 or 405.
			next.ServeHTTP(w, r)
			return
		}
		token, ok := bearerToken(r)
		if !ok {
			writeUnauthenticated(w)
			return
		}
		p, err := s.deps.Auth.Authenticate(r.Context(), token)
		if errors.Is(err, auth.ErrUnauthenticated) {
			writeUnauthenticated(w)
			return
		}
		if err != nil {
			s.responseError(w, r, err)
			return
		}
		noteUser(r.Context(), p.UserID)
		if !p.EmailVerified && !unverifiedRoutes[pattern] {
			writeProblem(w, problemEmailNotVerified.problem(""))
			return
		}
		next.ServeHTTP(w, r.WithContext(withPrincipal(r.Context(), p)))
	})
}

func bearerToken(r *http.Request) (string, bool) {
	scheme, token, ok := strings.Cut(r.Header.Get("Authorization"), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") || token == "" {
		return "", false
	}
	return token, true
}

func writeUnauthenticated(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	writeProblem(w, problemUnauthenticated.problem(""))
}

type clientIPKey struct{}

// clientIP is the address the request came from, for login throttling. The
// server runs locally without a proxy, so it is the TCP peer; trusting a
// forwarding header comes with a host (doc 08, before production).
func clientIP(ctx context.Context) string {
	ip, _ := ctx.Value(clientIPKey{}).(string)
	return ip
}

func remoteIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

type principalKey struct{}

func withPrincipal(ctx context.Context, p auth.Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

// principalFrom returns the signed-in User of a protected route.
func principalFrom(ctx context.Context) (auth.Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(auth.Principal)
	return p, ok
}

// requestInfo collects what inner middleware learns for the request log.
type requestInfo struct {
	pattern string
	userID  *platform.ID
}

type requestInfoKey struct{}

// noteUser puts the User's ID on the request's log line.
func noteUser(ctx context.Context, id platform.ID) {
	if info, ok := ctx.Value(requestInfoKey{}).(*requestInfo); ok {
		info.userID = &id
	}
}

// statusRecorder remembers the status a handler writes.
type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (r *statusRecorder) WriteHeader(status int) {
	if !r.wroteHeader {
		r.status, r.wroteHeader = status, true
	}
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	r.wroteHeader = true
	return r.ResponseWriter.Write(b)
}

func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }
