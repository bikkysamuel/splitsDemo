package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/bikkysamuel/splitsDemo/server/internal/platform"
)

// Deps are the Service's collaborators.
type Deps struct {
	Repository     Repository
	OTPSender      OTPSender
	Clock          platform.Clock
	IDs            *platform.IDGenerator
	PasswordParams PasswordParams
}

// Service runs sign-up, email verification, sign-in and token checks.
type Service struct {
	deps Deps
	// dummyHash is verified against when an email is unknown, so sign-in
	// takes as long whether or not the account exists (doc 08).
	dummyHash func() (string, error)
}

// NewService returns a Service.
func NewService(deps Deps) *Service {
	return &Service{
		deps: deps,
		dummyHash: sync.OnceValues(func() (string, error) {
			return HashPassword(rand.Text(), deps.PasswordParams)
		}),
	}
}

// SignUp creates an unverified User, sends a verification code and returns
// a Session that works for GET /v1/me only until the email is verified.
func (s *Service) SignUp(ctx context.Context, rawEmail, password string) (Session, error) {
	email, err := validateSignUp(rawEmail, password)
	if err != nil {
		return Session{}, err
	}
	hash, err := HashPassword(password, s.deps.PasswordParams)
	if err != nil {
		return Session{}, fmt.Errorf("auth: hash password: %w", err)
	}
	code, err := s.deps.OTPSender.NewCode()
	if err != nil {
		return Session{}, fmt.Errorf("auth: new code: %w", err)
	}
	// The User's ID is known only once the store has upserted the User, so
	// the store fills UserID in the code and Session (SignUpRecord).
	session, stored, err := s.newSession(platform.ID{})
	if err != nil {
		return Session{}, fmt.Errorf("auth: new session: %w", err)
	}
	user, err := s.deps.Repository.SignUp(ctx, SignUpRecord{
		UserID:       s.deps.IDs.New(),
		Email:        email,
		PasswordHash: hash,
		Now:          s.deps.Clock.Now(),
		Code:         s.newCode(platform.ID{}, PurposeVerifyEmail, code),
		Session:      stored,
	})
	if err != nil {
		return Session{}, fmt.Errorf("auth: sign up: %w", err)
	}
	if err := s.deps.OTPSender.Send(ctx, user.Email, PurposeVerifyEmail, code); err != nil {
		return Session{}, fmt.Errorf("auth: send verification code: %w", err)
	}
	session.User = user
	return session, nil
}

// VerifyEmail checks a verification code. On success the email is verified
// and a new Session is returned. Every failure is ErrInvalidCode.
func (s *Service) VerifyEmail(ctx context.Context, rawEmail, code string) (Session, error) {
	email, err := NormalizeEmail(rawEmail)
	if err != nil {
		return Session{}, &ValidationError{Fields: []FieldError{fieldError(err)}}
	}
	user, err := s.deps.Repository.UserByEmail(ctx, email)
	switch {
	case errors.Is(err, ErrNotFound):
		return Session{}, ErrInvalidCode
	case err != nil:
		return Session{}, fmt.Errorf("auth: find user: %w", err)
	case user.EmailVerified:
		return Session{}, ErrInvalidCode
	}
	live, err := s.deps.Repository.LiveCode(ctx, user.ID, PurposeVerifyEmail)
	switch {
	case errors.Is(err, ErrNotFound):
		return Session{}, ErrInvalidCode
	case err != nil:
		return Session{}, fmt.Errorf("auth: find code: %w", err)
	case !s.deps.Clock.Now().Before(live.ExpiresAt):
		return Session{}, ErrInvalidCode
	}
	// Count the attempt before comparing, so concurrent guesses can't go
	// past the limit.
	counted, err := s.deps.Repository.CountCodeAttempt(ctx, live.ID, MaxCodeAttempts)
	if err != nil {
		return Session{}, fmt.Errorf("auth: count code attempt: %w", err)
	}
	if !counted || subtle.ConstantTimeCompare(hashCode(live.ID, code), live.CodeHash) != 1 {
		return Session{}, ErrInvalidCode
	}
	session, stored, err := s.newSession(user.ID)
	if err != nil {
		return Session{}, fmt.Errorf("auth: new session: %w", err)
	}
	if err := s.deps.Repository.CompleteEmailVerification(ctx, live.ID, stored); err != nil {
		if errors.Is(err, ErrInvalidCode) {
			return Session{}, ErrInvalidCode
		}
		return Session{}, fmt.Errorf("auth: complete verification: %w", err)
	}
	session.User = User{ID: user.ID, Email: user.Email, EmailVerified: true}
	return session, nil
}

// ResendVerificationCode replaces the User's verification code and sends
// the new one. It does nothing, without error, when the email is unknown or
// already verified, so the caller can't tell the cases apart (doc 08).
func (s *Service) ResendVerificationCode(ctx context.Context, rawEmail string) error {
	email, err := NormalizeEmail(rawEmail)
	if err != nil {
		return &ValidationError{Fields: []FieldError{fieldError(err)}}
	}
	user, err := s.deps.Repository.UserByEmail(ctx, email)
	switch {
	case errors.Is(err, ErrNotFound):
		return nil
	case err != nil:
		return fmt.Errorf("auth: find user: %w", err)
	case user.EmailVerified:
		return nil
	}
	code, err := s.deps.OTPSender.NewCode()
	if err != nil {
		return fmt.Errorf("auth: new code: %w", err)
	}
	if err := s.deps.Repository.ReplaceCode(ctx, s.newCode(user.ID, PurposeVerifyEmail, code)); err != nil {
		return fmt.Errorf("auth: store code: %w", err)
	}
	if err := s.deps.OTPSender.Send(ctx, user.Email, PurposeVerifyEmail, code); err != nil {
		return fmt.Errorf("auth: send verification code: %w", err)
	}
	return nil
}

// SignIn checks the email and password and returns a new Session. An
// unverified User gets one too, and the app takes them to verification.
// clientIP is the address the request came from, for throttling. Too many
// failures return a *ThrottledError without checking the password.
func (s *Service) SignIn(ctx context.Context, rawEmail, password, clientIP string) (Session, error) {
	throttles := []throttleKey{
		{ThrottleAccount, throttleHash(strings.ToLower(strings.TrimSpace(rawEmail)))},
		{ThrottleIP, throttleHash(clientIP)},
	}
	if err := s.checkThrottles(ctx, throttles); err != nil {
		return Session{}, err
	}
	user, err := s.checkCredentials(ctx, rawEmail, password)
	if errors.Is(err, ErrInvalidCredentials) {
		if err := s.recordFailure(ctx, throttles); err != nil {
			return Session{}, err
		}
		return Session{}, ErrInvalidCredentials
	}
	if err != nil {
		return Session{}, err
	}
	if err := s.deps.Repository.ClearLoginThrottle(ctx, ThrottleAccount, throttles[0].key); err != nil {
		return Session{}, fmt.Errorf("auth: clear login throttle: %w", err)
	}
	session, stored, err := s.newSession(user.ID)
	if err != nil {
		return Session{}, fmt.Errorf("auth: new session: %w", err)
	}
	if err := s.deps.Repository.CreateSession(ctx, stored); err != nil {
		return Session{}, fmt.Errorf("auth: create session: %w", err)
	}
	session.User = user
	return session, nil
}

// checkCredentials returns the User whose email and password these are, or
// ErrInvalidCredentials. It takes as long for an unknown email (doc 08).
func (s *Service) checkCredentials(ctx context.Context, rawEmail, password string) (User, error) {
	email, err := NormalizeEmail(rawEmail)
	if err != nil {
		// A malformed email can't belong to an account.
		return User{}, ErrInvalidCredentials
	}
	user, err := s.deps.Repository.UserByEmail(ctx, email)
	if errors.Is(err, ErrNotFound) {
		dummy, err := s.dummyHash()
		if err != nil {
			return User{}, fmt.Errorf("auth: dummy hash: %w", err)
		}
		_, _ = VerifyPassword(dummy, password)
		return User{}, ErrInvalidCredentials
	}
	if err != nil {
		return User{}, fmt.Errorf("auth: find user: %w", err)
	}
	ok, err := VerifyPassword(user.PasswordHash, password)
	if err != nil {
		return User{}, fmt.Errorf("auth: verify password of user %s: %w", user.ID, err)
	}
	if !ok {
		return User{}, ErrInvalidCredentials
	}
	return user.User, nil
}

type throttleKey struct {
	scope ThrottleScope
	key   []byte
}

// throttleHash keys a throttle without storing the email or IP.
func throttleHash(v string) []byte {
	sum := sha256.Sum256([]byte(v))
	return sum[:]
}

// checkThrottles returns a *ThrottledError if any count asks the attempt to
// wait.
func (s *Service) checkThrottles(ctx context.Context, keys []throttleKey) error {
	now := s.deps.Clock.Now()
	var wait time.Duration
	for _, k := range keys {
		rec, err := s.deps.Repository.LoginThrottle(ctx, k.scope, k.key)
		if errors.Is(err, ErrNotFound) {
			continue
		}
		if err != nil {
			return fmt.Errorf("auth: read login throttle: %w", err)
		}
		if !rec.LastFailureAt.After(now.Add(-LoginFailureMemory)) {
			continue // forgotten
		}
		wait = max(wait, rec.NextAllowedAt.Sub(now))
	}
	if wait > 0 {
		return &ThrottledError{RetryAfter: wait}
	}
	return nil
}

func (s *Service) recordFailure(ctx context.Context, keys []throttleKey) error {
	now := s.deps.Clock.Now()
	for _, k := range keys {
		failures, err := s.deps.Repository.RecordLoginFailure(ctx, k.scope, k.key, now, now.Add(-LoginFailureMemory))
		if err != nil {
			return fmt.Errorf("auth: record login failure: %w", err)
		}
		if delay := LoginDelay(failures); delay > 0 {
			if err := s.deps.Repository.SetLoginNextAllowed(ctx, k.scope, k.key, now.Add(delay)); err != nil {
				return fmt.Errorf("auth: set login delay: %w", err)
			}
		}
	}
	return nil
}

// LoginDelay is how long the next attempt waits after the given number of
// consecutive failures: nothing for the first FreeLoginFailures - 1, then
// FirstLoginDelay doubling up to MaxLoginDelay.
func LoginDelay(failures int) time.Duration {
	if failures < FreeLoginFailures {
		return 0
	}
	delay := FirstLoginDelay
	for range failures - FreeLoginFailures {
		delay *= 2
		if delay >= MaxLoginDelay {
			return MaxLoginDelay
		}
	}
	return delay
}

// Refresh exchanges a refresh token for a new pair (ADR-0011). A reused
// token revokes the Session. Every failure is ErrUnauthenticated.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (Session, error) {
	if refreshToken == "" {
		return Session{}, ErrUnauthenticated
	}
	session, stored, err := s.newSession(platform.ID{})
	if err != nil {
		return Session{}, fmt.Errorf("auth: new session: %w", err)
	}
	user, err := s.deps.Repository.RotateSession(ctx, hashToken(refreshToken), s.deps.Clock.Now(), stored)
	switch {
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrUnauthenticated), errors.Is(err, ErrRefreshReused):
		return Session{}, ErrUnauthenticated
	case err != nil:
		return Session{}, fmt.Errorf("auth: rotate session: %w", err)
	}
	session.User = user
	return session, nil
}

// SignOut revokes the Principal's Session.
func (s *Service) SignOut(ctx context.Context, p Principal) error {
	if err := s.deps.Repository.RevokeSession(ctx, p.SessionID, s.deps.Clock.Now()); err != nil {
		return fmt.Errorf("auth: revoke session %s: %w", p.SessionID, err)
	}
	return nil
}

// Authenticate returns who an access token speaks for, or
// ErrUnauthenticated.
func (s *Service) Authenticate(ctx context.Context, accessToken string) (Principal, error) {
	if accessToken == "" {
		return Principal{}, ErrUnauthenticated
	}
	rec, err := s.deps.Repository.SessionByAccessHash(ctx, hashToken(accessToken))
	switch {
	case errors.Is(err, ErrNotFound):
		return Principal{}, ErrUnauthenticated
	case err != nil:
		return Principal{}, fmt.Errorf("auth: find session: %w", err)
	case rec.Revoked, !s.deps.Clock.Now().Before(rec.AccessExpiresAt):
		return Principal{}, ErrUnauthenticated
	}
	return Principal{UserID: rec.UserID, SessionID: rec.FamilyID, EmailVerified: rec.EmailVerified}, nil
}

// Me returns the signed-in User.
func (s *Service) Me(ctx context.Context, p Principal) (User, error) {
	user, err := s.deps.Repository.UserByID(ctx, p.UserID)
	if err != nil {
		return User{}, fmt.Errorf("auth: find user %s: %w", p.UserID, err)
	}
	return user, nil
}

// newCode makes the record of a one-time code for userID (hash only).
func (s *Service) newCode(userID platform.ID, purpose CodePurpose, code string) NewCode {
	now := s.deps.Clock.Now()
	id := s.deps.IDs.New()
	return NewCode{
		ID:        id,
		UserID:    userID,
		Purpose:   purpose,
		CodeHash:  hashCode(id, code),
		ExpiresAt: now.Add(CodeLifetime),
		CreatedAt: now,
	}
}

// newSession makes a token pair for userID: the Session to return and the
// record (hashes only) to store.
func (s *Service) newSession(userID platform.ID) (Session, NewSession, error) {
	access, err := newToken()
	if err != nil {
		return Session{}, NewSession{}, err
	}
	refresh, err := newToken()
	if err != nil {
		return Session{}, NewSession{}, err
	}
	now := s.deps.Clock.Now()
	id := s.deps.IDs.New()
	session := Session{
		AccessToken:      access,
		AccessExpiresAt:  now.Add(AccessTokenLifetime),
		RefreshToken:     refresh,
		RefreshExpiresAt: now.Add(RefreshTokenLifetime),
	}
	return session, NewSession{
		ID:               id,
		FamilyID:         id,
		UserID:           userID,
		AccessHash:       hashToken(access),
		AccessExpiresAt:  session.AccessExpiresAt,
		RefreshHash:      hashToken(refresh),
		RefreshExpiresAt: session.RefreshExpiresAt,
		CreatedAt:        now,
	}, nil
}

func validateSignUp(rawEmail, password string) (string, error) {
	var fields []FieldError
	email, err := NormalizeEmail(rawEmail)
	if err != nil {
		fields = append(fields, fieldError(err))
	}
	if err := CheckNewPassword(password); err != nil {
		fields = append(fields, fieldError(err))
	}
	if len(fields) > 0 {
		return "", &ValidationError{Fields: fields}
	}
	return email, nil
}

func fieldError(err error) FieldError {
	var fe FieldError
	if errors.As(err, &fe) {
		return fe
	}
	return FieldError{Field: "", Code: CodeInvalid}
}

const tokenBytes = 32

// newToken returns 32 random bytes, base64url without padding (ADR-0011).
func newToken() (string, error) {
	b := make([]byte, tokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("auth: new token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// hashToken is the SHA-256 of a token: the only form stored (ADR-0011).
func hashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

// hashCode is the stored form of a one-time code: SHA-256 over the code's
// own ID and the code, so equal codes never hash alike (ADR-0016).
func hashCode(codeID platform.ID, code string) []byte {
	h := sha256.New()
	h.Write(codeID[:])
	h.Write([]byte(code))
	return h.Sum(nil)
}
