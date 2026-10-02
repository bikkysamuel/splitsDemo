package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/bikkysamuel/splitsDemo/server/internal/idempotency"
	"github.com/bikkysamuel/splitsDemo/server/internal/platform"
	"github.com/bikkysamuel/splitsDemo/server/internal/store/sqlcgen"
)

// IdempotencyRepository implements idempotency.Repository.
type IdempotencyRepository struct{ db *DB }

// Idempotency returns the idempotency-key repository.
func (db *DB) Idempotency() *IdempotencyRepository { return &IdempotencyRepository{db: db} }

var _ idempotency.Repository = (*IdempotencyRepository)(nil)

// Claim implements idempotency.Repository.
func (r *IdempotencyRepository) Claim(ctx context.Context, userID, key platform.ID, requestHash []byte, now, staleBefore time.Time) (idempotency.Record, bool, error) {
	var (
		rec     idempotency.Record
		claimed bool
	)
	err := r.db.inTx(ctx, func(q *sqlcgen.Queries) error {
		_, err := q.ClaimIdempotencyKey(ctx, sqlcgen.ClaimIdempotencyKeyParams{
			UserID: uuid(userID), Key: uuid(key), RequestHash: requestHash,
			Now: timestamptz(now), StaleBefore: timestamptz(staleBefore),
		})
		if err == nil {
			claimed = true
			return nil
		}
		if !isNoRows(err) {
			return fmt.Errorf("claim key: %w", err)
		}
		row, err := q.IdempotencyKey(ctx, sqlcgen.IdempotencyKeyParams{UserID: uuid(userID), Key: uuid(key)})
		if err != nil {
			return fmt.Errorf("select key: %w", err)
		}
		rec = idempotency.Record{RequestHash: row.RequestHash, Completed: row.Status == "completed"}
		if rec.Completed {
			rec.Response = idempotency.Response{
				Status:      int(row.ResponseStatus.Int32),
				ContentType: row.ResponseContentType.String,
				Body:        row.ResponseBody,
			}
		}
		return nil
	})
	return rec, claimed, err
}

// Complete implements idempotency.Repository.
func (r *IdempotencyRepository) Complete(ctx context.Context, userID, key platform.ID, resp idempotency.Response) error {
	err := sqlcgen.New(r.db.pool).CompleteIdempotencyKey(ctx, sqlcgen.CompleteIdempotencyKeyParams{
		UserID:              uuid(userID),
		Key:                 uuid(key),
		ResponseStatus:      pgtype.Int4{Int32: int32(resp.Status), Valid: true}, //nolint:gosec // an HTTP status
		ResponseContentType: pgtype.Text{String: resp.ContentType, Valid: resp.ContentType != ""},
		ResponseBody:        resp.Body,
	})
	if err != nil {
		return fmt.Errorf("complete key: %w", err)
	}
	return nil
}

// Release implements idempotency.Repository.
func (r *IdempotencyRepository) Release(ctx context.Context, userID, key platform.ID) error {
	if err := sqlcgen.New(r.db.pool).ReleaseIdempotencyKey(ctx, sqlcgen.ReleaseIdempotencyKeyParams{UserID: uuid(userID), Key: uuid(key)}); err != nil {
		return fmt.Errorf("release key: %w", err)
	}
	return nil
}
