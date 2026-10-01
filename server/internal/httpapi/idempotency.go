package httpapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"net/http"

	"github.com/bikkysamuel/splitsDemo/server/internal/idempotency"
	"github.com/bikkysamuel/splitsDemo/server/internal/platform"
)

// idempotent applies the Idempotency-Key to every write by a signed-in User
// (NFR-R1): the key is required, the first response is kept 24 hours and
// replayed to repeats of the same request, and a 5xx is forgotten so the
// request can be retried. Anonymous auth endpoints are left alone: their
// responses carry tokens, which are never stored (ADR-0011).
func (s *Server) idempotent(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, signedIn := principalFrom(r.Context())
		if !signedIn || !isWrite(r.Method) {
			next.ServeHTTP(w, r)
			return
		}
		key, err := platform.ParseID(r.Header.Get("Idempotency-Key"))
		if err != nil {
			writeProblem(w, problemIdempotencyKeyRequired.problem("Send a UUID Idempotency-Key header."))
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			s.requestError(w, r, err)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))

		ctx := r.Context()
		stored, err := s.deps.Idempotency.Begin(ctx, p.UserID, key, requestHash(r, body))
		switch {
		case errors.Is(err, idempotency.ErrKeyReused):
			writeProblem(w, problemIdempotencyKeyReused.problem(""))
			return
		case errors.Is(err, idempotency.ErrInProgress):
			writeProblem(w, problemIdempotencyKeyInProgress.problem(""))
			return
		case err != nil:
			s.responseError(w, r, err)
			return
		case stored != nil:
			replay(w, *stored)
			return
		}

		// The key is ours: finish it even if the client goes away or the
		// handler panics.
		rec := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
		finishCtx := context.WithoutCancel(ctx)
		completed := false
		defer func() {
			if completed {
				return
			}
			if err := s.deps.Idempotency.Release(finishCtx, p.UserID, key); err != nil {
				s.deps.Logger.ErrorContext(finishCtx, "release idempotency key failed", "user_id", p.UserID.String())
			}
		}()
		next.ServeHTTP(rec, r)
		if rec.status >= http.StatusInternalServerError {
			return
		}
		resp := idempotency.Response{Status: rec.status, ContentType: rec.Header().Get("Content-Type"), Body: rec.body.Bytes()}
		if err := s.deps.Idempotency.Complete(finishCtx, p.UserID, key, resp); err != nil {
			s.deps.Logger.ErrorContext(finishCtx, "complete idempotency key failed", "user_id", p.UserID.String())
			return
		}
		completed = true
	})
}

func isWrite(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	}
	return false
}

// requestHash identifies a request for key reuse: method, path, query and
// body.
func requestHash(r *http.Request, body []byte) []byte {
	h := sha256.New()
	for _, part := range []string{r.Method, r.URL.Path, r.URL.RawQuery} {
		h.Write([]byte(part))
		h.Write([]byte{0})
	}
	h.Write(body)
	return h.Sum(nil)
}

func replay(w http.ResponseWriter, resp idempotency.Response) {
	if resp.ContentType != "" {
		w.Header().Set("Content-Type", resp.ContentType)
	}
	w.Header().Set("Idempotent-Replayed", "true")
	w.WriteHeader(resp.Status)
	_, _ = w.Write(resp.Body)
}

// responseRecorder passes a response through and keeps a copy.
type responseRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
	body        bytes.Buffer
}

func (r *responseRecorder) WriteHeader(status int) {
	if !r.wroteHeader {
		r.status, r.wroteHeader = status, true
	}
	r.ResponseWriter.WriteHeader(status)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.wroteHeader = true
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

func (r *responseRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }
