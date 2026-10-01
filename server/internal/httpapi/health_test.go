package httpapi_test

import (
	"net/http"
	"testing"

	"github.com/bikkysamuel/splitsDemo/server/internal/apptest"
)

func TestHealthzReportsLiveness(t *testing.T) {
	srv := apptest.Start(t)

	resp := srv.Get(t, "/healthz")

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /healthz = %d; want 200", resp.StatusCode)
	}
}

func TestReadyzReportsDatabaseReachability(t *testing.T) {
	srv := apptest.Start(t)

	if resp := srv.Get(t, "/readyz"); resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /readyz with the database up = %d; want 200", resp.StatusCode)
	}

	srv.CutDatabase()

	resp := srv.Get(t, "/readyz")
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("GET /readyz with the database unreachable = %d; want 503", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Type"); got != "application/problem+json" {
		t.Errorf("Content-Type = %q; want application/problem+json", got)
	}
}

func TestHealthzStaysUpWhenTheDatabaseIsUnreachable(t *testing.T) {
	srv := apptest.Start(t)
	srv.CutDatabase()

	if resp := srv.Get(t, "/healthz"); resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /healthz with the database unreachable = %d; want 200", resp.StatusCode)
	}
}
