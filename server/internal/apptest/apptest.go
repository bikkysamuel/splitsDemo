// Package apptest starts the fully wired server against real Postgres for
// HTTP-seam tests (doc 09). Each server gets its own schema, and every
// response is validated against api/openapi.yaml.
package apptest

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/bikkysamuel/splitsDemo/server/internal/app"
	"github.com/bikkysamuel/splitsDemo/server/internal/auth"
	"github.com/bikkysamuel/splitsDemo/server/internal/pgtest"
	"github.com/bikkysamuel/splitsDemo/server/internal/platform"
)

// Server is a running test server.
type Server struct {
	// URL is the base URL, without a trailing slash.
	URL string
	// Clock is the server's clock; Advance it to expire codes and tokens.
	Clock *platform.FakeClock
	// DatabaseURL reaches the server's schema directly, for checking what is
	// stored (never for setting up state: use the API).
	DatabaseURL string
	proxy       *pgtest.Proxy
	logs        *syncBuffer
}

// Start migrates a fresh schema, wires the server and serves it over HTTP
// until the test ends. Database traffic goes through a proxy so the test can
// cut it with CutDatabase.
func Start(t testing.TB) *Server {
	t.Helper()
	databaseURL := pgtest.NewSchema(t)
	u, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatalf("parse database URL: %v", err)
	}
	proxy := pgtest.NewProxy(t, u.Host)
	u.Host = proxy.Addr()

	cfg := platform.Config{
		AppEnv:      platform.EnvDevelopment,
		DatabaseURL: u.String(),
		OTPMode:     platform.OTPModeFixed,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	clock := platform.NewFakeClock(time.Now().UTC().Truncate(time.Millisecond))
	logs := &syncBuffer{}
	a, err := app.New(ctx, cfg, platform.NewLogger(io.MultiWriter(t.Output(), logs)),
		app.WithClock(clock), app.WithPasswordParams(auth.TestPasswordParams))
	if err != nil {
		t.Fatalf("start app: %v", err)
	}
	t.Cleanup(a.Close)

	ts := httptest.NewServer(a.Handler())
	t.Cleanup(ts.Close)
	return &Server{URL: ts.URL, Clock: clock, DatabaseURL: databaseURL, proxy: proxy, logs: logs}
}

// Response is an HTTP response with its body fully read.
type Response struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

// JSON decodes the body into v, failing the test if it can't.
func (r Response) JSON(t testing.TB, v any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, v); err != nil {
		t.Fatalf("decode response body: %v\nbody: %s", err, r.Body)
	}
}

// ProblemType returns the slug of a problem+json response's type, or "".
func (r Response) ProblemType(t testing.TB) string {
	t.Helper()
	var p struct {
		Type string `json:"type"`
	}
	if json.Unmarshal(r.Body, &p) != nil {
		return ""
	}
	const base = "https://splits.dev/problems/"
	if len(p.Type) <= len(base) || p.Type[:len(base)] != base {
		return ""
	}
	return p.Type[len(base):]
}

// Get requests path on the server and checks the response against
// api/openapi.yaml.
func (s *Server) Get(t testing.TB, path string, headers ...string) Response {
	t.Helper()
	return s.Do(t, http.MethodGet, path, nil, headers...)
}

// Post sends body as JSON (a []byte is sent as is) and checks the response
// against api/openapi.yaml. headers are name, value pairs.
func (s *Server) Post(t testing.TB, path string, body any, headers ...string) Response {
	t.Helper()
	raw, ok := body.([]byte)
	if !ok {
		var err error
		if raw, err = json.Marshal(body); err != nil {
			t.Fatalf("encode request body: %v", err)
		}
	}
	return s.Do(t, http.MethodPost, path, raw, append([]string{"Content-Type", "application/json"}, headers...)...)
}

// Do sends a request and checks the response against api/openapi.yaml.
// headers are name, value pairs; a later pair replaces an earlier one.
func (s *Server) Do(t testing.TB, method, path string, body []byte, headers ...string) Response {
	t.Helper()
	if len(headers)%2 != 0 {
		t.Fatalf("headers must be name, value pairs: %q", headers)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, s.URL+path, reader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	for i := 0; i < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read %s %s body: %v", method, path, err)
	}
	r := Response{StatusCode: resp.StatusCode, Header: resp.Header, Body: respBody}
	checkContract(t, req, r)
	return r
}

// Logs returns every log line the server has written so far, for checking
// that logs carry IDs only (doc 08).
func (s *Server) Logs() string {
	s.logs.mu.Lock()
	defer s.logs.mu.Unlock()
	return s.logs.buf.String()
}

// CutDatabase makes the database unreachable for the rest of the test.
func (s *Server) CutDatabase() {
	s.proxy.Cut()
}

// syncBuffer is a bytes.Buffer safe for the server's concurrent writes.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}
