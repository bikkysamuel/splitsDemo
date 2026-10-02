package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bikkysamuel/splitsDemo/server/internal/auth"
	"github.com/bikkysamuel/splitsDemo/server/internal/idempotency"
	"github.com/bikkysamuel/splitsDemo/server/internal/pgtest"
	"github.com/bikkysamuel/splitsDemo/server/internal/platform"
	"github.com/bikkysamuel/splitsDemo/server/internal/store"
)

// No endpoint of #10 is a signed-in write, so the Idempotency-Key rules
// (NFR-R1) are tested on the middleware itself, against real Postgres, with
// a stand-in write handler. Later write endpoints add their own replay test
// at the HTTP seam.

type idempotencyFixture struct {
	t       *testing.T
	server  *Server
	handler http.Handler
	clock   *platform.FakeClock
	users   []auth.Principal
	calls   atomic.Int32
	// status is what the stand-in handler answers.
	status atomic.Int32
}

func newIdempotencyFixture(t *testing.T) *idempotencyFixture {
	t.Helper()
	ctx := context.Background()
	db, err := store.Open(ctx, pgtest.NewSchema(t))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(db.Close)
	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	f := &idempotencyFixture{t: t, clock: platform.NewFakeClock(time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC))}
	f.status.Store(http.StatusCreated)
	ids := platform.NewIDGenerator(f.clock)
	for _, email := range []string{"alice@example.com", "bob@example.com"} {
		u, err := db.Auth().SignUp(ctx, auth.SignUpRecord{
			UserID: ids.New(), Email: email, PasswordHash: "x", Now: f.clock.Now(),
			Code: auth.NewCode{
				ID: ids.New(), Purpose: auth.PurposeVerifyEmail, CodeHash: make([]byte, 32),
				ExpiresAt: f.clock.Now(), CreatedAt: f.clock.Now(),
			},
			Session: auth.NewSession{
				ID: ids.New(), AccessHash: randomHash(ids), RefreshHash: randomHash(ids),
				AccessExpiresAt: f.clock.Now(), RefreshExpiresAt: f.clock.Now(), CreatedAt: f.clock.Now(),
			},
		})
		if err != nil {
			t.Fatalf("create user: %v", err)
		}
		f.users = append(f.users, auth.Principal{UserID: u.ID, EmailVerified: true})
	}

	f.server = &Server{mux: http.NewServeMux(), deps: Deps{
		Logger:      platform.NewLogger(t.Output()),
		Idempotency: idempotency.NewService(db.Idempotency(), f.clock),
	}}
	write := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := f.calls.Add(1)
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(int(f.status.Load()))
		_ = json.NewEncoder(w).Encode(map[string]any{"call": n, "echo": string(body)})
	})
	f.handler = f.server.idempotent(write)
	return f
}

func randomHash(ids *platform.IDGenerator) []byte {
	a, b := ids.New(), ids.New()
	return append(a[:], b[:]...)
}

// send makes a write as user (an index into f.users; -1 for anonymous).
func (f *idempotencyFixture) send(user int, method, path, key, body string) *httptest.ResponseRecorder {
	f.t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	if user >= 0 {
		req = req.WithContext(withPrincipal(req.Context(), f.users[user]))
	}
	rec := httptest.NewRecorder()
	f.handler.ServeHTTP(rec, req)
	return rec
}

func problemSlug(rec *httptest.ResponseRecorder) string {
	var p struct{ Type string }
	_ = json.Unmarshal(rec.Body.Bytes(), &p)
	return strings.TrimPrefix(p.Type, problemTypeBase)
}

const key1 = "0190b6c4-0000-7000-8000-0000000000a1"

func TestIdempotencyReplaysTheFirstResponse(t *testing.T) {
	f := newIdempotencyFixture(t)

	first := f.send(0, http.MethodPost, "/v1/groups", key1, `{"name":"Trip"}`)
	second := f.send(0, http.MethodPost, "/v1/groups", key1, `{"name":"Trip"}`)

	if f.calls.Load() != 1 {
		t.Fatalf("handler ran %d times; want once", f.calls.Load())
	}
	if second.Code != first.Code || !bytes.Equal(second.Body.Bytes(), first.Body.Bytes()) {
		t.Errorf("replay = %d %s; want the first response %d %s", second.Code, second.Body, first.Code, first.Body)
	}
	if got := second.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("replay Content-Type = %q; want application/json", got)
	}
	if got := second.Header().Get("Idempotent-Replayed"); got != "true" {
		t.Errorf("Idempotent-Replayed = %q; want true", got)
	}
}

func TestIdempotencyRequiresAKeyOnSignedInWrites(t *testing.T) {
	f := newIdempotencyFixture(t)

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		for name, key := range map[string]string{"missing": "", "not a UUID": "abc"} {
			rec := f.send(0, method, "/v1/groups", key, `{}`)
			if rec.Code != http.StatusBadRequest || problemSlug(rec) != "idempotency-key-required" {
				t.Errorf("%s with %s key = %d %s; want 400 idempotency-key-required", method, name, rec.Code, rec.Body)
			}
		}
	}
	if f.calls.Load() != 0 {
		t.Errorf("handler ran %d times; want never", f.calls.Load())
	}
}

func TestIdempotencyLeavesReadsAndAnonymousRequestsAlone(t *testing.T) {
	f := newIdempotencyFixture(t)

	f.send(0, http.MethodGet, "/v1/me", "", "")
	f.send(-1, http.MethodPost, "/v1/auth/signin", "", `{}`)
	f.send(-1, http.MethodPost, "/v1/auth/signin", "", `{}`)

	if f.calls.Load() != 3 {
		t.Errorf("handler ran %d times; want 3", f.calls.Load())
	}
}

func TestIdempotencyRefusesAKeyReusedForADifferentRequest(t *testing.T) {
	f := newIdempotencyFixture(t)
	f.send(0, http.MethodPost, "/v1/groups", key1, `{"name":"Trip"}`)

	for name, rec := range map[string]*httptest.ResponseRecorder{
		"other body":   f.send(0, http.MethodPost, "/v1/groups", key1, `{"name":"Flat"}`),
		"other path":   f.send(0, http.MethodPost, "/v1/groups/x", key1, `{"name":"Trip"}`),
		"other method": f.send(0, http.MethodPatch, "/v1/groups", key1, `{"name":"Trip"}`),
	} {
		if rec.Code != http.StatusUnprocessableEntity || problemSlug(rec) != "idempotency-key-reused" {
			t.Errorf("%s = %d %s; want 422 idempotency-key-reused", name, rec.Code, rec.Body)
		}
	}
	if f.calls.Load() != 1 {
		t.Errorf("handler ran %d times; want once", f.calls.Load())
	}
}

// Keys are scoped per User (doc 08).
func TestIdempotencyKeysArePerUser(t *testing.T) {
	f := newIdempotencyFixture(t)

	f.send(0, http.MethodPost, "/v1/groups", key1, `{"name":"Trip"}`)
	bob := f.send(1, http.MethodPost, "/v1/groups", key1, `{"name":"Trip"}`)

	if f.calls.Load() != 2 || bob.Header().Get("Idempotent-Replayed") != "" {
		t.Errorf("handler ran %d times (bob replayed: %q); want each User's request to run",
			f.calls.Load(), bob.Header().Get("Idempotent-Replayed"))
	}
}

func TestIdempotencyKeysAreKept24Hours(t *testing.T) {
	f := newIdempotencyFixture(t)
	f.send(0, http.MethodPost, "/v1/groups", key1, `{"name":"Trip"}`)

	f.clock.Advance(24*time.Hour - time.Millisecond)
	f.send(0, http.MethodPost, "/v1/groups", key1, `{"name":"Trip"}`)
	if f.calls.Load() != 1 {
		t.Fatalf("handler ran %d times within 24 h; want a replay", f.calls.Load())
	}

	f.clock.Advance(time.Millisecond)
	if rec := f.send(0, http.MethodPost, "/v1/groups", key1, `{"name":"Flat"}`); rec.Code != http.StatusCreated {
		t.Errorf("after 24 h = %d %s; want the key free again", rec.Code, rec.Body)
	}
	if f.calls.Load() != 2 {
		t.Errorf("handler ran %d times; want it to run again after 24 h", f.calls.Load())
	}
}

// A 5xx is not kept, so the client's retry runs the request again.
func TestIdempotencyForgetsServerErrors(t *testing.T) {
	f := newIdempotencyFixture(t)
	f.status.Store(http.StatusInternalServerError)
	f.send(0, http.MethodPost, "/v1/groups", key1, `{"name":"Trip"}`)

	f.status.Store(http.StatusCreated)
	rec := f.send(0, http.MethodPost, "/v1/groups", key1, `{"name":"Trip"}`)

	if rec.Code != http.StatusCreated || f.calls.Load() != 2 {
		t.Errorf("retry = %d after %d calls; want 201 from a second run", rec.Code, f.calls.Load())
	}
}

// 4xx answers are outcomes of the request, so they replay too.
func TestIdempotencyReplaysClientErrors(t *testing.T) {
	f := newIdempotencyFixture(t)
	f.status.Store(http.StatusConflict)
	f.send(0, http.MethodPost, "/v1/groups", key1, `{"name":"Trip"}`)

	f.status.Store(http.StatusCreated)
	rec := f.send(0, http.MethodPost, "/v1/groups", key1, `{"name":"Trip"}`)

	if rec.Code != http.StatusConflict || f.calls.Load() != 1 {
		t.Errorf("retry = %d after %d calls; want the stored 409", rec.Code, f.calls.Load())
	}
}

func TestIdempotencyRefusesARepeatWhileTheFirstIsRunning(t *testing.T) {
	f := newIdempotencyFixture(t)
	started, release := make(chan struct{}), make(chan struct{})
	f.handler = f.server.idempotent(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(started)
		<-release
		w.WriteHeader(http.StatusCreated)
	}))
	done := make(chan struct{})
	go func() {
		defer close(done)
		f.send(0, http.MethodPost, "/v1/groups", key1, `{"name":"Trip"}`)
	}()
	<-started

	rec := f.send(0, http.MethodPost, "/v1/groups", key1, `{"name":"Trip"}`)
	close(release)
	<-done

	if rec.Code != http.StatusConflict || problemSlug(rec) != "idempotency-key-in-progress" {
		t.Errorf("repeat while running = %d %s; want 409 idempotency-key-in-progress", rec.Code, rec.Body)
	}
}
