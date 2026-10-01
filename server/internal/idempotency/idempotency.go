// Package idempotency makes signed-in writes safe to retry (NFR-R1): the
// first response to an Idempotency-Key is kept for 24 hours and replayed to
// every repeat of the same request by the same User.
package idempotency

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bikkysamuel/splitsDemo/server/internal/platform"
)

// Retention is how long a key and its response are kept (D14).
const Retention = 24 * time.Hour

var (
	// ErrKeyReused: the key was used for a different request.
	ErrKeyReused = errors.New("idempotency: key reused for a different request")
	// ErrInProgress: a request with the key is still running.
	ErrInProgress = errors.New("idempotency: request with this key in progress")
)

// Response is a stored response.
type Response struct {
	Status      int
	ContentType string
	Body        []byte
}

// Record is a stored key.
type Record struct {
	RequestHash []byte
	Completed   bool
	Response    Response
}

// Repository stores keys per User.
type Repository interface {
	// Claim stores key as in progress unless the User holds a live key
	// (created after staleBefore); a stale one is replaced. When a live
	// key exists it returns that record and false.
	Claim(ctx context.Context, userID, key platform.ID, requestHash []byte, now, staleBefore time.Time) (Record, bool, error)
	// Complete stores the response of an in-progress key.
	Complete(ctx context.Context, userID, key platform.ID, resp Response) error
	// Release deletes an in-progress key, so the request can be retried.
	Release(ctx context.Context, userID, key platform.ID) error
}

// Service decides whether a request runs or replays.
type Service struct {
	repo  Repository
	clock platform.Clock
}

// NewService returns a Service.
func NewService(repo Repository, clock platform.Clock) *Service {
	return &Service{repo: repo, clock: clock}
}

// Begin claims key for the request. It returns nil when the request should
// run (then call Complete or Release), or the stored response to replay. It
// returns ErrKeyReused or ErrInProgress when neither applies.
func (s *Service) Begin(ctx context.Context, userID, key platform.ID, requestHash []byte) (*Response, error) {
	now := s.clock.Now()
	rec, claimed, err := s.repo.Claim(ctx, userID, key, requestHash, now, now.Add(-Retention))
	switch {
	case err != nil:
		return nil, fmt.Errorf("idempotency: claim key: %w", err)
	case claimed:
		return nil, nil
	case !bytes.Equal(rec.RequestHash, requestHash):
		return nil, ErrKeyReused
	case !rec.Completed:
		return nil, ErrInProgress
	}
	return &rec.Response, nil
}

// Complete keeps resp for replay.
func (s *Service) Complete(ctx context.Context, userID, key platform.ID, resp Response) error {
	if err := s.repo.Complete(ctx, userID, key, resp); err != nil {
		return fmt.Errorf("idempotency: complete key: %w", err)
	}
	return nil
}

// Release forgets the key, so a retry runs the request again.
func (s *Service) Release(ctx context.Context, userID, key platform.ID) error {
	if err := s.repo.Release(ctx, userID, key); err != nil {
		return fmt.Errorf("idempotency: release key: %w", err)
	}
	return nil
}
