package store

import (
	"context"
	"fmt"
	"time"

	"github.com/bikkysamuel/splitsDemo/server/internal/auth"
	"github.com/bikkysamuel/splitsDemo/server/internal/platform"
	"github.com/bikkysamuel/splitsDemo/server/internal/store/sqlcgen"
)

// AuthRepository implements auth.Repository.
type AuthRepository struct{ db *DB }

// Auth returns the auth repository.
func (db *DB) Auth() *AuthRepository { return &AuthRepository{db: db} }

var _ auth.Repository = (*AuthRepository)(nil)

// SignUp implements auth.Repository.
func (r *AuthRepository) SignUp(ctx context.Context, rec auth.SignUpRecord) (auth.User, error) {
	var user auth.User
	err := r.db.inTx(ctx, func(q *sqlcgen.Queries) error {
		row, err := q.UpsertUnverifiedUser(ctx, sqlcgen.UpsertUnverifiedUserParams{
			ID: uuid(rec.UserID), Email: rec.Email, PasswordHash: rec.PasswordHash, Now: timestamptz(rec.Now),
		})
		if isNoRows(err) {
			return auth.ErrEmailTaken
		}
		if err != nil {
			return fmt.Errorf("upsert user: %w", err)
		}
		user = auth.User{ID: id(row.ID), Email: row.Email, EmailVerified: row.EmailVerifiedAt.Valid}
		// A replaced unverified User loses its Sessions: nobody has proved
		// they own the address yet.
		if err := q.RevokeUserSessions(ctx, sqlcgen.RevokeUserSessionsParams{Now: timestamptz(rec.Now), UserID: row.ID}); err != nil {
			return fmt.Errorf("revoke sessions: %w", err)
		}
		code := rec.Code
		code.UserID = user.ID
		if err := replaceCode(ctx, q, code); err != nil {
			return err
		}
		session := rec.Session
		session.UserID = user.ID
		return insertSession(ctx, q, session)
	})
	return user, err
}

// UserByEmail implements auth.Repository.
func (r *AuthRepository) UserByEmail(ctx context.Context, email string) (auth.UserRecord, error) {
	row, err := sqlcgen.New(r.db.pool).UserByEmail(ctx, email)
	if isNoRows(err) {
		return auth.UserRecord{}, auth.ErrNotFound
	}
	if err != nil {
		return auth.UserRecord{}, fmt.Errorf("select user by email: %w", err)
	}
	return auth.UserRecord{
		User:         auth.User{ID: id(row.ID), Email: row.Email, EmailVerified: row.EmailVerifiedAt.Valid},
		PasswordHash: row.PasswordHash,
	}, nil
}

// UserByID implements auth.Repository.
func (r *AuthRepository) UserByID(ctx context.Context, userID platform.ID) (auth.User, error) {
	row, err := sqlcgen.New(r.db.pool).UserByID(ctx, uuid(userID))
	if isNoRows(err) {
		return auth.User{}, auth.ErrNotFound
	}
	if err != nil {
		return auth.User{}, fmt.Errorf("select user by id: %w", err)
	}
	return auth.User{ID: id(row.ID), Email: row.Email, EmailVerified: row.EmailVerifiedAt.Valid}, nil
}

// CreateSession implements auth.Repository.
func (r *AuthRepository) CreateSession(ctx context.Context, s auth.NewSession) error {
	return insertSession(ctx, sqlcgen.New(r.db.pool), s)
}

// SessionByAccessHash implements auth.Repository.
func (r *AuthRepository) SessionByAccessHash(ctx context.Context, hash []byte) (auth.SessionRecord, error) {
	row, err := sqlcgen.New(r.db.pool).SessionByAccessHash(ctx, hash)
	if isNoRows(err) {
		return auth.SessionRecord{}, auth.ErrNotFound
	}
	if err != nil {
		return auth.SessionRecord{}, fmt.Errorf("select session: %w", err)
	}
	return auth.SessionRecord{
		ID:              id(row.ID),
		FamilyID:        id(row.FamilyID),
		UserID:          id(row.UserID),
		EmailVerified:   row.EmailVerifiedAt.Valid,
		AccessExpiresAt: row.AccessExpiresAt.Time,
		Revoked:         row.RevokedAt.Valid,
	}, nil
}

// ReplaceCode implements auth.Repository.
func (r *AuthRepository) ReplaceCode(ctx context.Context, c auth.NewCode) error {
	return r.db.inTx(ctx, func(q *sqlcgen.Queries) error { return replaceCode(ctx, q, c) })
}

// LiveCode implements auth.Repository.
func (r *AuthRepository) LiveCode(ctx context.Context, userID platform.ID, purpose auth.CodePurpose) (auth.CodeRecord, error) {
	row, err := sqlcgen.New(r.db.pool).LiveCode(ctx, sqlcgen.LiveCodeParams{UserID: uuid(userID), Purpose: string(purpose)})
	if isNoRows(err) {
		return auth.CodeRecord{}, auth.ErrNotFound
	}
	if err != nil {
		return auth.CodeRecord{}, fmt.Errorf("select live code: %w", err)
	}
	return auth.CodeRecord{ID: id(row.ID), CodeHash: row.CodeHash, ExpiresAt: row.ExpiresAt.Time, Attempts: int(row.Attempts)}, nil
}

// CountCodeAttempt implements auth.Repository.
func (r *AuthRepository) CountCodeAttempt(ctx context.Context, codeID platform.ID, maxAttempts int) (bool, error) {
	n, err := sqlcgen.New(r.db.pool).CountCodeAttempt(ctx, sqlcgen.CountCodeAttemptParams{
		ID: uuid(codeID), MaxAttempts: int32(maxAttempts), //nolint:gosec // a small constant
	})
	if err != nil {
		return false, fmt.Errorf("count code attempt: %w", err)
	}
	return n == 1, nil
}

// CompleteEmailVerification implements auth.Repository.
func (r *AuthRepository) CompleteEmailVerification(ctx context.Context, codeID platform.ID, s auth.NewSession) error {
	return r.db.inTx(ctx, func(q *sqlcgen.Queries) error {
		userID, err := q.ConsumeCode(ctx, sqlcgen.ConsumeCodeParams{Now: timestamptz(s.CreatedAt), ID: uuid(codeID)})
		if isNoRows(err) {
			return auth.ErrInvalidCode
		}
		if err != nil {
			return fmt.Errorf("consume code: %w", err)
		}
		if id(userID) != s.UserID {
			return fmt.Errorf("code %s belongs to another user", codeID)
		}
		if err := q.MarkEmailVerified(ctx, sqlcgen.MarkEmailVerifiedParams{Now: timestamptz(s.CreatedAt), ID: userID}); err != nil {
			return fmt.Errorf("mark email verified: %w", err)
		}
		return insertSession(ctx, q, s)
	})
}

// RotateSession implements auth.Repository.
func (r *AuthRepository) RotateSession(ctx context.Context, refreshHash []byte, now time.Time, next auth.NewSession) (auth.User, error) {
	var user auth.User
	reused := false
	err := r.db.inTx(ctx, func(q *sqlcgen.Queries) error {
		row, err := q.SessionByRefreshHashForUpdate(ctx, refreshHash)
		if isNoRows(err) {
			return auth.ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("select session by refresh hash: %w", err)
		}
		if row.ReplacedBy.Valid {
			// Theft signal (ADR-0011): end the whole Session, and keep that
			// even though the caller gets an error.
			if err := q.RevokeSessionFamily(ctx, sqlcgen.RevokeSessionFamilyParams{Now: timestamptz(now), FamilyID: row.FamilyID}); err != nil {
				return fmt.Errorf("revoke session family: %w", err)
			}
			reused = true
			return nil
		}
		if row.RevokedAt.Valid || !now.Before(row.RefreshExpiresAt.Time) {
			return auth.ErrUnauthenticated
		}
		next.FamilyID = id(row.FamilyID)
		next.UserID = id(row.UserID)
		if err := insertSession(ctx, q, next); err != nil {
			return err
		}
		if err := q.MarkSessionReplaced(ctx, sqlcgen.MarkSessionReplacedParams{
			ReplacedBy: uuid(next.ID), Now: timestamptz(now), ID: row.ID,
		}); err != nil {
			return fmt.Errorf("mark session replaced: %w", err)
		}
		user = auth.User{ID: id(row.UserID), Email: row.Email, EmailVerified: row.EmailVerifiedAt.Valid}
		return nil
	})
	if err == nil && reused {
		return auth.User{}, auth.ErrRefreshReused
	}
	return user, err
}

// RevokeSession implements auth.Repository.
func (r *AuthRepository) RevokeSession(ctx context.Context, familyID platform.ID, now time.Time) error {
	err := sqlcgen.New(r.db.pool).RevokeSessionFamily(ctx, sqlcgen.RevokeSessionFamilyParams{Now: timestamptz(now), FamilyID: uuid(familyID)})
	if err != nil {
		return fmt.Errorf("revoke session family: %w", err)
	}
	return nil
}

// UpdateLoginThrottle implements auth.Repository.
func (r *AuthRepository) UpdateLoginThrottle(ctx context.Context, k auth.ThrottleKey, update func(auth.ThrottleRecord) auth.ThrottleRecord) error {
	return r.db.inTx(ctx, func(q *sqlcgen.Queries) error {
		if err := q.EnsureLoginThrottle(ctx, sqlcgen.EnsureLoginThrottleParams{Scope: string(k.Scope), Key: k.Key}); err != nil {
			return fmt.Errorf("ensure login throttle: %w", err)
		}
		row, err := q.LoginThrottleForUpdate(ctx, sqlcgen.LoginThrottleForUpdateParams{Scope: string(k.Scope), Key: k.Key})
		if err != nil {
			return fmt.Errorf("lock login throttle: %w", err)
		}
		next := update(auth.ThrottleRecord{
			Failures: int(row.Failures), LastFailureAt: row.LastFailureAt.Time, NextAllowedAt: row.NextAllowedAt.Time,
		})
		err = q.UpdateLoginThrottle(ctx, sqlcgen.UpdateLoginThrottleParams{
			Failures:      int32(next.Failures), //nolint:gosec // a small count
			LastFailureAt: timestamptz(next.LastFailureAt),
			NextAllowedAt: timestamptz(next.NextAllowedAt),
			Scope:         string(k.Scope),
			Key:           k.Key,
		})
		if err != nil {
			return fmt.Errorf("update login throttle: %w", err)
		}
		return nil
	})
}

// ClearLoginThrottle implements auth.Repository.
func (r *AuthRepository) ClearLoginThrottle(ctx context.Context, k auth.ThrottleKey) error {
	err := sqlcgen.New(r.db.pool).ClearLoginThrottle(ctx, sqlcgen.ClearLoginThrottleParams{Scope: string(k.Scope), Key: k.Key})
	if err != nil {
		return fmt.Errorf("clear login throttle: %w", err)
	}
	return nil
}

func replaceCode(ctx context.Context, q *sqlcgen.Queries, c auth.NewCode) error {
	if err := q.DeleteUserCodes(ctx, sqlcgen.DeleteUserCodesParams{UserID: uuid(c.UserID), Purpose: string(c.Purpose)}); err != nil {
		return fmt.Errorf("delete codes: %w", err)
	}
	err := q.InsertCode(ctx, sqlcgen.InsertCodeParams{
		ID:        uuid(c.ID),
		UserID:    uuid(c.UserID),
		Purpose:   string(c.Purpose),
		CodeHash:  c.CodeHash,
		ExpiresAt: timestamptz(c.ExpiresAt),
		CreatedAt: timestamptz(c.CreatedAt),
	})
	if err != nil {
		return fmt.Errorf("insert code: %w", err)
	}
	return nil
}

func insertSession(ctx context.Context, q *sqlcgen.Queries, s auth.NewSession) error {
	err := q.InsertSession(ctx, sqlcgen.InsertSessionParams{
		ID:               uuid(s.ID),
		FamilyID:         uuid(s.FamilyID),
		UserID:           uuid(s.UserID),
		AccessHash:       s.AccessHash,
		AccessExpiresAt:  timestamptz(s.AccessExpiresAt),
		RefreshHash:      s.RefreshHash,
		RefreshExpiresAt: timestamptz(s.RefreshExpiresAt),
		CreatedAt:        timestamptz(s.CreatedAt),
	})
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}
