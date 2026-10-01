package httpapi_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/bikkysamuel/splitsDemo/server/internal/apptest"
)

const (
	password = "correct horse battery"
	devCode  = "123456" // the fixed code under APP_ENV=development (ADR-0016)
)

type authSession struct {
	User struct {
		ID            string `json:"id"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
	} `json:"user"`
	AccessToken      string    `json:"access_token"`
	AccessExpiresAt  time.Time `json:"access_expires_at"`
	RefreshToken     string    `json:"refresh_token"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at"`
}

func credentials(email, pw string) map[string]string {
	return map[string]string{"email": email, "password": pw}
}

func bearer(token string) []string { return []string{"Authorization", "Bearer " + token} }

func signUp(t *testing.T, srv *apptest.Server, email string) authSession {
	t.Helper()
	resp := srv.Post(t, "/v1/auth/signup", credentials(email, password))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("sign up %s = %d; want 201\n%s", email, resp.StatusCode, resp.Body)
	}
	var s authSession
	resp.JSON(t, &s)
	return s
}

func verify(t *testing.T, srv *apptest.Server, email, code string) apptest.Response {
	t.Helper()
	return srv.Post(t, "/v1/auth/verify-email", map[string]string{"email": email, "code": code})
}

func signUpVerified(t *testing.T, srv *apptest.Server, email string) authSession {
	t.Helper()
	signUp(t, srv, email)
	resp := verify(t, srv, email, devCode)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("verify %s = %d; want 200\n%s", email, resp.StatusCode, resp.Body)
	}
	var s authSession
	resp.JSON(t, &s)
	return s
}

func wantProblem(t *testing.T, resp apptest.Response, status int, slug string) {
	t.Helper()
	if resp.StatusCode != status || resp.ProblemType(t) != slug {
		t.Fatalf("got %d %q; want %d %q\n%s", resp.StatusCode, resp.ProblemType(t), status, slug, resp.Body)
	}
	if got := resp.Header.Get("Content-Type"); got != "application/problem+json" {
		t.Errorf("Content-Type = %q; want application/problem+json", got)
	}
}

// --- Sign-up (FR-A1, FR-A3) ---

func TestSignUpCreatesAnUnverifiedUserWithASession(t *testing.T) {
	srv := apptest.Start(t)
	before := srv.Clock.Now()

	s := signUp(t, srv, "alice@example.com")

	if s.User.Email != "alice@example.com" || s.User.EmailVerified {
		t.Errorf("user = %+v; want alice@example.com, unverified", s.User)
	}
	if got, want := s.AccessExpiresAt, before.Add(15*time.Minute); !got.Equal(want) {
		t.Errorf("access_expires_at = %v; want %v (15 min)", got, want)
	}
	if got, want := s.RefreshExpiresAt, before.Add(30*24*time.Hour); !got.Equal(want) {
		t.Errorf("refresh_expires_at = %v; want %v (30 days)", got, want)
	}
	for name, token := range map[string]string{"access": s.AccessToken, "refresh": s.RefreshToken} {
		raw, err := base64.RawURLEncoding.DecodeString(token)
		if err != nil || len(raw) != 32 {
			t.Errorf("%s token %q is not 32 bytes of base64url", name, token)
		}
	}
	if s.AccessToken == s.RefreshToken {
		t.Error("access and refresh tokens are equal")
	}
}

func TestSignUpTrimsTheEmailAndTreatsCaseAsTheSameAddress(t *testing.T) {
	srv := apptest.Start(t)

	s := signUp(t, srv, "  Alice@Example.com ")
	if s.User.Email != "Alice@Example.com" {
		t.Errorf("email = %q; want it trimmed", s.User.Email)
	}
	verify(t, srv, "alice@example.com", devCode)

	resp := srv.Post(t, "/v1/auth/signup", credentials("ALICE@example.COM", password))
	wantProblem(t, resp, http.StatusConflict, "email-taken")
}

func TestSignUpValidatesEmailAndPassword(t *testing.T) {
	srv := apptest.Start(t)
	for _, tc := range []struct {
		name, email, password string
		wantErrors            []string // "field code"
	}{
		{"malformed email", "alice", password, []string{"/email invalid"}},
		{"empty email", " ", password, []string{"/email required"}},
		{"short password", "a@example.com", "too short", []string{"/password too_short"}},
		{"long password", "a@example.com", strings.Repeat("x", 129), []string{"/password too_long"}},
		{"common password", "a@example.com", "password1234", []string{"/password too_common"}},
		{"both", "alice", "short", []string{"/email invalid", "/password too_short"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp := srv.Post(t, "/v1/auth/signup", credentials(tc.email, tc.password))

			wantProblem(t, resp, http.StatusBadRequest, "validation-failed")
			var p struct {
				Errors []struct{ Field, Code string } `json:"errors"`
			}
			resp.JSON(t, &p)
			var got []string
			for _, e := range p.Errors {
				got = append(got, e.Field+" "+e.Code)
			}
			if strings.Join(got, ", ") != strings.Join(tc.wantErrors, ", ") {
				t.Errorf("errors = %v; want %v", got, tc.wantErrors)
			}
		})
	}
}

func TestSignUpRefusesARequestItCannotDecode(t *testing.T) {
	srv := apptest.Start(t)

	for name, body := range map[string][]byte{
		"not JSON":       []byte("email=alice"),
		"missing fields": []byte(`{}`),
		"wrong types":    []byte(`{"email": 1, "password": true}`),
	} {
		t.Run(name, func(t *testing.T) {
			resp := srv.Post(t, "/v1/auth/signup", body)
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("status = %d; want 400\n%s", resp.StatusCode, resp.Body)
			}
		})
	}
}

// Signing up again before verifying replaces the password: nobody has
// proved they own the address yet, so the earlier Sessions stop working.
func TestSigningUpAgainBeforeVerifyingReplacesThePasswordAndRevokesSessions(t *testing.T) {
	srv := apptest.Start(t)
	first := signUp(t, srv, "alice@example.com")

	resp := srv.Post(t, "/v1/auth/signup", credentials("alice@example.com", "another long password"))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("second sign-up = %d; want 201\n%s", resp.StatusCode, resp.Body)
	}
	var second authSession
	resp.JSON(t, &second)

	if second.User.ID != first.User.ID {
		t.Errorf("second sign-up made user %s; want the same user %s", second.User.ID, first.User.ID)
	}
	wantProblem(t, srv.Get(t, "/v1/me", bearer(first.AccessToken)...), http.StatusUnauthorized, "unauthenticated")
	if got := srv.Get(t, "/v1/me", bearer(second.AccessToken)...); got.StatusCode != http.StatusOK {
		t.Errorf("GET /v1/me with the new token = %d; want 200", got.StatusCode)
	}
	wantProblem(t, srv.Post(t, "/v1/auth/signin", credentials("alice@example.com", password)),
		http.StatusUnauthorized, "invalid-credentials")
}

// ADR-0011: only SHA-256 hashes of tokens are stored; the password is
// stored as an Argon2id hash.
func TestTokensAndPasswordsAreStoredHashed(t *testing.T) {
	srv := apptest.Start(t)
	s := signUp(t, srv, "alice@example.com")

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, srv.DatabaseURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer func() { _ = conn.Close(ctx) }()

	var accessHash, refreshHash []byte
	if err := conn.QueryRow(ctx, "SELECT access_hash, refresh_hash FROM sessions").Scan(&accessHash, &refreshHash); err != nil {
		t.Fatalf("select session: %v", err)
	}
	if a := sha256.Sum256([]byte(s.AccessToken)); string(accessHash) != string(a[:]) {
		t.Error("access_hash is not the SHA-256 of the access token")
	}
	if r := sha256.Sum256([]byte(s.RefreshToken)); string(refreshHash) != string(r[:]) {
		t.Error("refresh_hash is not the SHA-256 of the refresh token")
	}

	var passwordHash string
	if err := conn.QueryRow(ctx, "SELECT password_hash FROM users").Scan(&passwordHash); err != nil {
		t.Fatalf("select user: %v", err)
	}
	if !strings.HasPrefix(passwordHash, "$argon2id$") || strings.Contains(passwordHash, password) {
		t.Errorf("password_hash %q is not an Argon2id hash", passwordHash)
	}

	var codeHash []byte
	if err := conn.QueryRow(ctx, "SELECT code_hash FROM one_time_codes").Scan(&codeHash); err != nil {
		t.Fatalf("select code: %v", err)
	}
	if strings.Contains(string(codeHash), devCode) {
		t.Error("code_hash contains the code in clear")
	}
}

// --- Email verification (FR-A2, ADR-0016) ---

func TestVerifyEmailWithTheDevelopmentCodeVerifiesAndSignsIn(t *testing.T) {
	srv := apptest.Start(t)
	signUp(t, srv, "alice@example.com")

	resp := verify(t, srv, " ALICE@example.com", devCode)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("verify = %d; want 200\n%s", resp.StatusCode, resp.Body)
	}
	var s authSession
	resp.JSON(t, &s)
	if !s.User.EmailVerified {
		t.Error("user.email_verified = false after verification")
	}
	var me struct {
		EmailVerified bool `json:"email_verified"`
	}
	srv.Get(t, "/v1/me", bearer(s.AccessToken)...).JSON(t, &me)
	if !me.EmailVerified {
		t.Error("GET /v1/me says unverified after verification")
	}
}

func TestVerifyEmailRefusesAWrongCode(t *testing.T) {
	srv := apptest.Start(t)
	signUp(t, srv, "alice@example.com")

	wantProblem(t, verify(t, srv, "alice@example.com", "654321"), http.StatusBadRequest, "invalid-code")
}

// An unknown email answers like a wrong code, so verification doesn't
// reveal which emails have accounts (doc 08).
func TestVerifyEmailAnswersInvalidCodeForAnUnknownEmail(t *testing.T) {
	srv := apptest.Start(t)

	wantProblem(t, verify(t, srv, "nobody@example.com", devCode), http.StatusBadRequest, "invalid-code")
}

func TestVerificationCodeExpiresAfter15Minutes(t *testing.T) {
	srv := apptest.Start(t)
	signUp(t, srv, "alice@example.com")

	srv.Clock.Advance(15 * time.Minute)

	wantProblem(t, verify(t, srv, "alice@example.com", devCode), http.StatusBadRequest, "invalid-code")
}

func TestVerificationCodeWorksJustBefore15Minutes(t *testing.T) {
	srv := apptest.Start(t)
	signUp(t, srv, "alice@example.com")

	srv.Clock.Advance(15*time.Minute - time.Millisecond)

	if resp := verify(t, srv, "alice@example.com", devCode); resp.StatusCode != http.StatusOK {
		t.Fatalf("verify = %d; want 200\n%s", resp.StatusCode, resp.Body)
	}
}

func TestVerificationCodeAllowsAtMostFiveAttempts(t *testing.T) {
	srv := apptest.Start(t)
	signUp(t, srv, "alice@example.com")

	for range 5 {
		wantProblem(t, verify(t, srv, "alice@example.com", "000000"), http.StatusBadRequest, "invalid-code")
	}

	// The right code no longer works: the five attempts are used up.
	wantProblem(t, verify(t, srv, "alice@example.com", devCode), http.StatusBadRequest, "invalid-code")
}

func TestTheFifthAttemptCanStillSucceed(t *testing.T) {
	srv := apptest.Start(t)
	signUp(t, srv, "alice@example.com")
	for range 4 {
		verify(t, srv, "alice@example.com", "000000")
	}

	if resp := verify(t, srv, "alice@example.com", devCode); resp.StatusCode != http.StatusOK {
		t.Fatalf("fifth attempt = %d; want 200\n%s", resp.StatusCode, resp.Body)
	}
}

func TestVerificationCodeWorksOnce(t *testing.T) {
	srv := apptest.Start(t)
	signUpVerified(t, srv, "alice@example.com")

	wantProblem(t, verify(t, srv, "alice@example.com", devCode), http.StatusBadRequest, "invalid-code")
}

func TestVerifyEmailValidatesTheEmail(t *testing.T) {
	srv := apptest.Start(t)

	wantProblem(t, verify(t, srv, "not an email", devCode), http.StatusBadRequest, "validation-failed")
}

// --- Resend (FR-A2) ---

func TestResendReplacesAUsedUpCode(t *testing.T) {
	srv := apptest.Start(t)
	signUp(t, srv, "alice@example.com")
	for range 5 {
		verify(t, srv, "alice@example.com", "000000")
	}

	resp := srv.Post(t, "/v1/auth/verify-email/resend", map[string]string{"email": "alice@example.com"})

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("resend = %d; want 202\n%s", resp.StatusCode, resp.Body)
	}
	if resp := verify(t, srv, "alice@example.com", devCode); resp.StatusCode != http.StatusOK {
		t.Fatalf("verify with the new code = %d; want 200\n%s", resp.StatusCode, resp.Body)
	}
}

func TestResendGivesTheNewCodeAFresh15Minutes(t *testing.T) {
	srv := apptest.Start(t)
	signUp(t, srv, "alice@example.com")
	srv.Clock.Advance(14 * time.Minute)

	srv.Post(t, "/v1/auth/verify-email/resend", map[string]string{"email": "alice@example.com"})
	srv.Clock.Advance(14 * time.Minute)

	if resp := verify(t, srv, "alice@example.com", devCode); resp.StatusCode != http.StatusOK {
		t.Fatalf("verify = %d; want 200\n%s", resp.StatusCode, resp.Body)
	}
}

// doc 08: the answer is the same whether or not the email has an account.
func TestResendAnswersTheSameForUnknownAndVerifiedEmails(t *testing.T) {
	srv := apptest.Start(t)
	signUpVerified(t, srv, "verified@example.com")

	for _, email := range []string{"nobody@example.com", "verified@example.com"} {
		resp := srv.Post(t, "/v1/auth/verify-email/resend", map[string]string{"email": email})
		if resp.StatusCode != http.StatusAccepted || len(resp.Body) != 0 {
			t.Errorf("resend %s = %d %q; want 202 with no body", email, resp.StatusCode, resp.Body)
		}
	}
}

func TestResendValidatesTheEmail(t *testing.T) {
	srv := apptest.Start(t)

	resp := srv.Post(t, "/v1/auth/verify-email/resend", map[string]string{"email": "@"})

	wantProblem(t, resp, http.StatusBadRequest, "validation-failed")
}

// --- Sign-in (FR-A4) ---

func TestSignInReturnsANewSession(t *testing.T) {
	srv := apptest.Start(t)
	up := signUpVerified(t, srv, "alice@example.com")

	resp := srv.Post(t, "/v1/auth/signin", credentials(" Alice@Example.com", password))

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("sign in = %d; want 200\n%s", resp.StatusCode, resp.Body)
	}
	var s authSession
	resp.JSON(t, &s)
	if s.User.ID != up.User.ID || !s.User.EmailVerified {
		t.Errorf("user = %+v; want %s, verified", s.User, up.User.ID)
	}
	if s.AccessToken == up.AccessToken {
		t.Error("sign-in returned the earlier access token; want a new Session")
	}
	if got := srv.Get(t, "/v1/me", bearer(s.AccessToken)...); got.StatusCode != http.StatusOK {
		t.Errorf("GET /v1/me with the sign-in token = %d; want 200", got.StatusCode)
	}
}

// An unverified User can sign in and is sent to verification by the app.
func TestSignInWorksForAnUnverifiedUser(t *testing.T) {
	srv := apptest.Start(t)
	signUp(t, srv, "alice@example.com")

	resp := srv.Post(t, "/v1/auth/signin", credentials("alice@example.com", password))

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("sign in = %d; want 200\n%s", resp.StatusCode, resp.Body)
	}
	var s authSession
	resp.JSON(t, &s)
	if s.User.EmailVerified {
		t.Error("user.email_verified = true; want false")
	}
}

// doc 08: a wrong password and an unknown email look the same.
func TestSignInAnswersInvalidCredentialsForAWrongPasswordOrUnknownEmail(t *testing.T) {
	srv := apptest.Start(t)
	signUpVerified(t, srv, "alice@example.com")

	for name, body := range map[string]map[string]string{
		"wrong password":  credentials("alice@example.com", "not the password"),
		"unknown email":   credentials("nobody@example.com", password),
		"malformed email": credentials("nobody", password),
		"empty password":  credentials("alice@example.com", ""),
	} {
		t.Run(name, func(t *testing.T) {
			wantProblem(t, srv.Post(t, "/v1/auth/signin", body), http.StatusUnauthorized, "invalid-credentials")
		})
	}
}

// --- GET /v1/me and Bearer auth (FR-U1, ADR-0011) ---

func TestGetMeReturnsTheSignedInUser(t *testing.T) {
	srv := apptest.Start(t)
	s := signUpVerified(t, srv, "alice@example.com")

	resp := srv.Get(t, "/v1/me", bearer(s.AccessToken)...)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /v1/me = %d; want 200\n%s", resp.StatusCode, resp.Body)
	}
	var me struct {
		ID            string `json:"id"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
	}
	resp.JSON(t, &me)
	if me.ID != s.User.ID || me.Email != "alice@example.com" || !me.EmailVerified {
		t.Errorf("me = %+v; want %s alice@example.com verified", me, s.User.ID)
	}
}

func TestGetMeWorksForAnUnverifiedUser(t *testing.T) {
	srv := apptest.Start(t)
	s := signUp(t, srv, "alice@example.com")

	if resp := srv.Get(t, "/v1/me", bearer(s.AccessToken)...); resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /v1/me = %d; want 200\n%s", resp.StatusCode, resp.Body)
	}
}

func TestGetMeRequiresAValidAccessToken(t *testing.T) {
	srv := apptest.Start(t)
	s := signUpVerified(t, srv, "alice@example.com")

	for name, headers := range map[string][]string{
		"no header":     nil,
		"unknown token": bearer("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"),
		"not Bearer":    {"Authorization", "Basic " + s.AccessToken},
		"empty token":   {"Authorization", "Bearer "},
		"refresh token": bearer(s.RefreshToken),
	} {
		t.Run(name, func(t *testing.T) {
			resp := srv.Get(t, "/v1/me", headers...)
			wantProblem(t, resp, http.StatusUnauthorized, "unauthenticated")
			if got := resp.Header.Get("WWW-Authenticate"); got != "Bearer" {
				t.Errorf("WWW-Authenticate = %q; want Bearer", got)
			}
		})
	}
}

func TestAccessTokenExpiresAfter15Minutes(t *testing.T) {
	srv := apptest.Start(t)
	s := signUpVerified(t, srv, "alice@example.com")

	srv.Clock.Advance(15*time.Minute - time.Millisecond)
	if resp := srv.Get(t, "/v1/me", bearer(s.AccessToken)...); resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /v1/me just before expiry = %d; want 200", resp.StatusCode)
	}

	srv.Clock.Advance(time.Millisecond)
	wantProblem(t, srv.Get(t, "/v1/me", bearer(s.AccessToken)...), http.StatusUnauthorized, "unauthenticated")
}

// Every operation the contract marks bearerAuth refuses a request without
// a token, and every other one doesn't ask for one: the server's list of
// public routes matches the contract.
func TestBearerAuthMatchesTheContract(t *testing.T) {
	srv := apptest.Start(t)

	for _, op := range apptest.Operations(t) {
		t.Run(op.Method+" "+op.Path, func(t *testing.T) {
			resp := srv.Do(t, op.Method, op.Path, []byte(`{}`), "Content-Type", "application/json")
			if op.BearerAuth {
				wantProblem(t, resp, http.StatusUnauthorized, "unauthenticated")
			} else if resp.StatusCode == http.StatusUnauthorized && resp.ProblemType(t) == "unauthenticated" {
				t.Errorf("public operation answered unauthenticated")
			}
		})
	}
}

// NFR-R1: every write a signed-in User makes takes an Idempotency-Key.
func TestEveryAuthenticatedWriteDeclaresAnIdempotencyKey(t *testing.T) {
	for _, op := range apptest.Operations(t) {
		write := op.Method != http.MethodGet && op.Method != http.MethodHead
		if write && op.BearerAuth && !op.IdempotencyKey {
			t.Errorf("%s %s requires bearerAuth but does not declare the Idempotency-Key header", op.Method, op.Path)
		}
	}
}

// --- Cross-cutting ---

func TestRequestBodiesAreLimitedTo64KB(t *testing.T) {
	srv := apptest.Start(t)
	big := `{"email":"alice@example.com","password":"` + strings.Repeat("x", 64*1024) + `"}`

	wantProblem(t, srv.Post(t, "/v1/auth/signup", []byte(big)), http.StatusRequestEntityTooLarge, "request-too-large")
}

func TestEveryResponseCarriesARequestID(t *testing.T) {
	srv := apptest.Start(t)

	generated := srv.Get(t, "/healthz").Header.Get("X-Request-ID")
	if generated == "" {
		t.Fatal("no X-Request-ID on a request that sent none")
	}
	if again := srv.Get(t, "/healthz").Header.Get("X-Request-ID"); again == generated {
		t.Error("two requests got the same generated X-Request-ID")
	}

	if got := srv.Get(t, "/healthz", "X-Request-ID", "client-id_42").Header.Get("X-Request-ID"); got != "client-id_42" {
		t.Errorf("X-Request-ID = %q; want the client's client-id_42 echoed", got)
	}
	for _, bad := range []string{strings.Repeat("a", 65), "has space", "semi;colon"} {
		if got := srv.Get(t, "/healthz", "X-Request-ID", bad).Header.Get("X-Request-ID"); got == bad || got == "" {
			t.Errorf("X-Request-ID for invalid %q = %q; want a generated one", bad, got)
		}
	}
}

// doc 08: logs carry IDs only, never passwords, tokens, codes or emails.
func TestAuthFlowsLogIDsOnly(t *testing.T) {
	srv := apptest.Start(t)
	s := signUp(t, srv, "alice@example.com")
	verify(t, srv, "alice@example.com", "654321")
	verify(t, srv, "alice@example.com", devCode)
	srv.Post(t, "/v1/auth/signin", credentials("alice@example.com", "wrong password!"))
	srv.Post(t, "/v1/auth/signin", credentials("alice@example.com", password))
	srv.Get(t, "/v1/me", bearer(s.AccessToken)...)

	logs := srv.Logs()
	// Codes are short digit runs that timestamps can contain by chance, so
	// look for them as JSON strings.
	for _, secret := range []string{"alice@example.com", password, "wrong password!", `"` + devCode + `"`, `"654321"`, s.AccessToken, s.RefreshToken} {
		if strings.Contains(logs, secret) {
			t.Errorf("logs contain %s", secret)
		}
	}
	if !strings.Contains(logs, s.User.ID) {
		t.Error("logs never name the user's ID; want request logs to carry it")
	}
}
