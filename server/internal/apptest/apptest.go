// Package apptest starts the fully wired server against real Postgres for
// HTTP-seam tests (doc 09). Each server gets its own schema.
package apptest

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/bikkysamuel/splitsDemo/server/internal/app"
	"github.com/bikkysamuel/splitsDemo/server/internal/pgtest"
	"github.com/bikkysamuel/splitsDemo/server/internal/platform"
)

// Server is a running test server.
type Server struct {
	// URL is the base URL, without a trailing slash.
	URL string

	proxy *pgtest.Proxy
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
	a, err := app.New(ctx, cfg, platform.NewLogger(t.Output()))
	if err != nil {
		t.Fatalf("start app: %v", err)
	}
	t.Cleanup(a.Close)

	ts := httptest.NewServer(a.Handler())
	t.Cleanup(ts.Close)
	return &Server{URL: ts.URL, proxy: proxy}
}

// Response is an HTTP response with its body fully read.
type Response struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

// Get requests path on the server.
func (s *Server) Get(t testing.TB, path string) Response {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.URL+path, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read GET %s body: %v", path, err)
	}
	return Response{StatusCode: resp.StatusCode, Header: resp.Header, Body: body}
}

// CutDatabase makes the database unreachable for the rest of the test.
func (s *Server) CutDatabase() {
	s.proxy.Cut()
}
