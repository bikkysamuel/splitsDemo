// Package auth owns Users, passwords (Argon2id, ADR-0003), Sessions with
// opaque revocable tokens (ADR-0011) and one-time codes (ADR-0016).
package auth

import (
	"context"
	"errors"
	"time"

	"github.com/bikkysamuel/splitsDemo/server/internal/platform"
)

// Lifetimes (ADR-0011, ADR-0016).
const (
	AccessTokenLifetime  = 15 * time.Minute
	RefreshTokenLifetime = 30 * 24 * time.Hour
	CodeLifetime         = 15 * time.Minute
	MaxCodeAttempts      = 5
)

// Errors the service returns for expected outcomes. Anything else is a
// failure of a dependency.
var (
	// ErrEmailTaken: a verified User already has the email.
	ErrEmailTaken = errors.New("auth: email taken")
	// ErrInvalidCredentials: the email or password is wrong. Both cases look
	// the same to the caller (doc 08).
	ErrInvalidCredentials = errors.New("auth: invalid credentials")
	// ErrInvalidCode: the one-time code is wrong, expired, used up or
	// unknown, or the email has no pending verification.
	ErrInvalidCode = errors.New("auth: invalid code")
	// ErrUnauthenticated: the access token is unknown, expired or revoked.
	ErrUnauthenticated = errors.New("auth: unauthenticated")
	// ErrNotFound is returned by Repository lookups that find nothing.
	ErrNotFound = errors.New("auth: not found")
)

// User is a person with an account.
type User struct {
	ID            platform.ID
	Email         string
	EmailVerified bool
}

// Session is a newly issued Session with its tokens in clear. Only their
// SHA-256 hashes are stored.
type Session struct {
	User             User
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshExpiresAt time.Time
}

// Principal is who an access token speaks for.
type Principal struct {
	UserID        platform.ID
	SessionID     platform.ID
	EmailVerified bool
}

// CodePurpose says what a one-time code proves.
type CodePurpose string

// Code purposes, as stored in one_time_codes.purpose.
const (
	PurposeVerifyEmail   CodePurpose = "verify_email"
	PurposeResetPassword CodePurpose = "reset_password"
)

// OTPSender makes one-time codes and delivers them (ADR-0016).
type OTPSender interface {
	// NewCode returns the next code to issue: 6 digits.
	NewCode() (string, error)
	// Send delivers code to email. It is called after the code's hash is
	// stored, so a failed delivery can be retried with a resend.
	Send(ctx context.Context, email string, purpose CodePurpose, code string) error
}

// FixedCode is the code FixedCodeSender issues (ADR-0016).
const FixedCode = "123456"

// FixedCodeSender issues FixedCode for every account and sends nothing. It
// is allowed only under APP_ENV=development; the server refuses to start
// with it anywhere else (ADR-0016).
type FixedCodeSender struct{}

// NewCode returns FixedCode.
func (FixedCodeSender) NewCode() (string, error) { return FixedCode, nil }

// Send does nothing.
func (FixedCodeSender) Send(context.Context, string, CodePurpose, string) error { return nil }

// UserRecord is a stored User with its password hash.
type UserRecord struct {
	User
	PasswordHash string
}

// SessionRecord is a stored Session, found by its access-token hash.
type SessionRecord struct {
	ID              platform.ID
	UserID          platform.ID
	EmailVerified   bool
	AccessExpiresAt time.Time
	Revoked         bool
}

// NewSession is a Session to store.
type NewSession struct {
	ID               platform.ID
	UserID           platform.ID
	AccessHash       []byte
	AccessExpiresAt  time.Time
	RefreshHash      []byte
	RefreshExpiresAt time.Time
	CreatedAt        time.Time
}

// NewCode is a one-time code to store, replacing every earlier code of the
// same User and purpose.
type NewCode struct {
	ID        platform.ID
	UserID    platform.ID
	Purpose   CodePurpose
	CodeHash  []byte
	ExpiresAt time.Time
	CreatedAt time.Time
}

// CodeRecord is a stored, unconsumed one-time code.
type CodeRecord struct {
	ID        platform.ID
	CodeHash  []byte
	ExpiresAt time.Time
	Attempts  int
}

// SignUpRecord is what a sign-up stores in one transaction. UserID is used
// only if no User has the email yet; Code and Session get the stored User's
// ID.
type SignUpRecord struct {
	UserID       platform.ID
	Email        string
	PasswordHash string
	Now          time.Time
	Code         NewCode
	Session      NewSession
}

// Repository stores Users, Sessions and one-time codes. Each method is one
// transaction.
type Repository interface {
	// SignUp creates an unverified User, or, if an unverified User has the
	// email, replaces its password hash and revokes its Sessions. Either way
	// it replaces the User's verification codes and stores the new Session.
	// It returns ErrEmailTaken if a verified User has the email.
	SignUp(ctx context.Context, r SignUpRecord) (User, error)
	// UserByEmail finds a User ignoring case, or returns ErrNotFound.
	UserByEmail(ctx context.Context, email string) (UserRecord, error)
	// UserByID finds a User, or returns ErrNotFound.
	UserByID(ctx context.Context, id platform.ID) (User, error)
	// CreateSession stores a Session.
	CreateSession(ctx context.Context, s NewSession) error
	// SessionByAccessHash finds a Session by its access-token hash, or
	// returns ErrNotFound.
	SessionByAccessHash(ctx context.Context, hash []byte) (SessionRecord, error)
	// ReplaceCode deletes the User's codes for the purpose and stores c.
	ReplaceCode(ctx context.Context, c NewCode) error
	// LiveCode finds the User's unconsumed code for the purpose, or returns
	// ErrNotFound.
	LiveCode(ctx context.Context, userID platform.ID, purpose CodePurpose) (CodeRecord, error)
	// CountCodeAttempt adds one attempt to an unconsumed code that has fewer
	// than max. It reports false when the code is used up or consumed, so
	// concurrent attempts never exceed max.
	CountCodeAttempt(ctx context.Context, codeID platform.ID, max int) (bool, error)
	// CompleteEmailVerification consumes the code, marks the User's email
	// verified and stores the Session. It returns ErrInvalidCode if the code
	// was consumed meanwhile.
	CompleteEmailVerification(ctx context.Context, codeID platform.ID, s NewSession) error
}
