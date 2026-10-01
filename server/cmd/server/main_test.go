package main

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/bikkysamuel/splitsDemo/server/internal/pgtest"
)

func envFrom(m map[string]string) func(string) string {
	return func(key string) string { return m[key] }
}

// ADR-0016: a fixed one-time code outside development would let anyone take
// over any account, so the server must refuse to start.
func TestServerRefusesFixedOTPOutsideDevelopment(t *testing.T) {
	for _, appEnv := range []string{"production", "staging", "Development", ""} {
		t.Run("APP_ENV="+appEnv, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := run(ctx, envFrom(map[string]string{
				"APP_ENV":      appEnv,
				"OTP_MODE":     "fixed",
				"DATABASE_URL": "postgres://unused@127.0.0.1:1/unused",
				"HTTP_ADDR":    "127.0.0.1:0",
			}), io.Discard)

			if err == nil {
				t.Fatal("run returned nil; want the server to refuse to start")
			}
			if !strings.Contains(err.Error(), "OTP_MODE") {
				t.Errorf("error %q does not name OTP_MODE", err)
			}
		})
	}
}

// NFR-O3: the startup log prints the base URL and every registered route.
func TestServerLogsBaseURLAndRoutesThenServesUntilStopped(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	env := envFrom(map[string]string{
		"APP_ENV":      "development",
		"OTP_MODE":     "fixed",
		"DATABASE_URL": pgtest.NewSchema(t),
		"HTTP_ADDR":    "127.0.0.1:0",
	})
	logR, logW := io.Pipe()
	done := make(chan error, 1)
	go func() {
		done <- run(ctx, env, logW)
		_ = logW.Close()
	}()

	started := waitForLog(t, logR, "server started")

	baseURL, _ := started["base_url"].(string)
	if !strings.HasPrefix(baseURL, "http://127.0.0.1:") {
		t.Fatalf("base_url = %q; want http://127.0.0.1:<port>", baseURL)
	}
	var routes []string
	for _, r := range started["routes"].([]any) {
		routes = append(routes, r.(string))
	}
	for _, want := range []string{"GET /healthz", "GET /readyz"} {
		if !slices.Contains(routes, want) {
			t.Errorf("routes %v missing %q", routes, want)
		}
	}

	resp, err := http.Get(baseURL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /healthz = %d; want 200", resp.StatusCode)
	}

	cancel()
	go func() { _, _ = io.Copy(io.Discard, logR) }()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("run after stop = %v; want nil", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("run did not return after its context was cancelled")
	}
}

// waitForLog reads JSON log lines until one has the given message.
func waitForLog(t *testing.T, r io.Reader, msg string) map[string]any {
	t.Helper()
	found := make(chan map[string]any, 1)
	go func() {
		sc := bufio.NewScanner(r)
		for sc.Scan() {
			var entry map[string]any
			if json.Unmarshal(sc.Bytes(), &entry) == nil && entry["msg"] == msg {
				found <- entry
				return
			}
		}
		close(found)
	}()
	select {
	case entry, ok := <-found:
		if !ok {
			t.Fatalf("log ended without %q", msg)
		}
		return entry
	case <-time.After(30 * time.Second):
		t.Fatalf("no %q log line within 30s", msg)
		return nil
	}
}
