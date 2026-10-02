package httpapi_test

import (
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/bikkysamuel/splitsDemo/server/internal/apptest"
)

func refresh(t *testing.T, srv *apptest.Server, refreshToken string) apptest.Response {
	t.Helper()
	return srv.Post(t, "/v1/auth/refresh", map[string]string{"refresh_token": refreshToken})
}

func signOut(t *testing.T, srv *apptest.Server, accessToken, key string) apptest.Response {
	t.Helper()
	return srv.Do(t, http.MethodPost, "/v1/auth/signout", nil,
		"Authorization", "Bearer "+accessToken, "Idempotency-Key", key)
}

func signIn(t *testing.T, srv *apptest.Server, email, pw string) apptest.Response {
	t.Helper()
	return srv.Post(t, "/v1/auth/signin", credentials(email, pw))
}

const signOutKey = "0190b6c4-0000-7000-8000-0000000000b1"

// --- Refresh (ADR-0011, FR-A4) ---

func TestRefreshRotatesTheTokenPair(t *testing.T) {
	srv := apptest.Start(t)
	first := signUpVerified(t, srv, "alice@example.com")
	srv.Clock.Advance(14 * time.Minute)

	resp := refresh(t, srv, first.RefreshToken)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("refresh = %d; want 200\n%s", resp.StatusCode, resp.Body)
	}
	var next authSession
	resp.JSON(t, &next)
	if next.AccessToken == first.AccessToken || next.RefreshToken == first.RefreshToken {
		t.Error("refresh returned an old token; want a new pair")
	}
	if next.User.ID != first.User.ID || !next.User.EmailVerified {
		t.Errorf("user = %+v; want %s, verified", next.User, first.User.ID)
	}
	if want := srv.Clock.Now().Add(15 * time.Minute); !next.AccessExpiresAt.Equal(want) {
		t.Errorf("access_expires_at = %v; want %v", next.AccessExpiresAt, want)
	}
	if got := srv.Get(t, "/v1/me", bearer(next.AccessToken)...); got.StatusCode != http.StatusOK {
		t.Errorf("GET /v1/me with the new access token = %d; want 200", got.StatusCode)
	}
	// The old pair stops working at once.
	wantProblem(t, srv.Get(t, "/v1/me", bearer(first.AccessToken)...), http.StatusUnauthorized, "unauthenticated")
}

func TestRefreshWorksAfterTheAccessTokenExpired(t *testing.T) {
	srv := apptest.Start(t)
	s := signUpVerified(t, srv, "alice@example.com")
	srv.Clock.Advance(20 * time.Minute)

	if resp := refresh(t, srv, s.RefreshToken); resp.StatusCode != http.StatusOK {
		t.Fatalf("refresh = %d; want 200\n%s", resp.StatusCode, resp.Body)
	}
}

func TestRefreshTokenExpiresAfter30Days(t *testing.T) {
	srv := apptest.Start(t)
	s := signUpVerified(t, srv, "alice@example.com")

	srv.Clock.Advance(30 * 24 * time.Hour)

	wantProblem(t, refresh(t, srv, s.RefreshToken), http.StatusUnauthorized, "unauthenticated")
}

// ADR-0011: reusing an exchanged refresh token is a theft signal, so the
// whole Session ends, including the pair the thief or the owner holds now.
func TestReusingARefreshTokenRevokesTheSession(t *testing.T) {
	srv := apptest.Start(t)
	first := signUpVerified(t, srv, "alice@example.com")
	var second authSession
	refresh(t, srv, first.RefreshToken).JSON(t, &second)

	wantProblem(t, refresh(t, srv, first.RefreshToken), http.StatusUnauthorized, "unauthenticated")

	wantProblem(t, srv.Get(t, "/v1/me", bearer(second.AccessToken)...), http.StatusUnauthorized, "unauthenticated")
	wantProblem(t, refresh(t, srv, second.RefreshToken), http.StatusUnauthorized, "unauthenticated")
}

// Reuse ends that Session only: the User's other devices stay signed in.
func TestReuseLeavesTheUsersOtherSessionsAlone(t *testing.T) {
	srv := apptest.Start(t)
	phone := signUpVerified(t, srv, "alice@example.com")
	var tablet authSession
	signIn(t, srv, "alice@example.com", password).JSON(t, &tablet)
	refresh(t, srv, phone.RefreshToken)

	refresh(t, srv, phone.RefreshToken) // reuse

	if got := srv.Get(t, "/v1/me", bearer(tablet.AccessToken)...); got.StatusCode != http.StatusOK {
		t.Errorf("other Session's GET /v1/me = %d; want 200", got.StatusCode)
	}
}

func TestRefreshRefusesUnknownTokens(t *testing.T) {
	srv := apptest.Start(t)
	s := signUpVerified(t, srv, "alice@example.com")

	for name, token := range map[string]string{
		"unknown":      "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		"access token": s.AccessToken,
		"empty":        "",
	} {
		t.Run(name, func(t *testing.T) {
			wantProblem(t, refresh(t, srv, token), http.StatusUnauthorized, "unauthenticated")
		})
	}
}

func TestRefreshRefusesARequestItCannotDecode(t *testing.T) {
	srv := apptest.Start(t)

	wantProblem(t, srv.Post(t, "/v1/auth/refresh", []byte("token=x")), http.StatusBadRequest, "invalid-request")
}

// --- Sign-out (ADR-0011) ---

func TestSignOutRevokesTheSession(t *testing.T) {
	srv := apptest.Start(t)
	s := signUpVerified(t, srv, "alice@example.com")

	resp := signOut(t, srv, s.AccessToken, signOutKey)

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("sign out = %d; want 204\n%s", resp.StatusCode, resp.Body)
	}
	wantProblem(t, srv.Get(t, "/v1/me", bearer(s.AccessToken)...), http.StatusUnauthorized, "unauthenticated")
	wantProblem(t, refresh(t, srv, s.RefreshToken), http.StatusUnauthorized, "unauthenticated")
}

// Sign-out ends the whole Session, including pairs made by refreshing.
func TestSignOutAfterARefreshRevokesTheSession(t *testing.T) {
	srv := apptest.Start(t)
	first := signUpVerified(t, srv, "alice@example.com")
	var next authSession
	refresh(t, srv, first.RefreshToken).JSON(t, &next)

	signOut(t, srv, next.AccessToken, signOutKey)

	wantProblem(t, refresh(t, srv, next.RefreshToken), http.StatusUnauthorized, "unauthenticated")
}

func TestSignOutLeavesOtherSessionsAlone(t *testing.T) {
	srv := apptest.Start(t)
	phone := signUpVerified(t, srv, "alice@example.com")
	var tablet authSession
	signIn(t, srv, "alice@example.com", password).JSON(t, &tablet)

	signOut(t, srv, phone.AccessToken, signOutKey)

	if got := srv.Get(t, "/v1/me", bearer(tablet.AccessToken)...); got.StatusCode != http.StatusOK {
		t.Errorf("other Session's GET /v1/me = %d; want 200", got.StatusCode)
	}
}

func TestAnUnverifiedUserCanSignOut(t *testing.T) {
	srv := apptest.Start(t)
	s := signUp(t, srv, "alice@example.com")

	if resp := signOut(t, srv, s.AccessToken, signOutKey); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("sign out = %d; want 204\n%s", resp.StatusCode, resp.Body)
	}
}

func TestSignOutRequiresAToken(t *testing.T) {
	srv := apptest.Start(t)

	resp := srv.Do(t, http.MethodPost, "/v1/auth/signout", nil, "Idempotency-Key", signOutKey)

	wantProblem(t, resp, http.StatusUnauthorized, "unauthenticated")
}

func TestSignOutRequiresAnIdempotencyKey(t *testing.T) {
	srv := apptest.Start(t)
	s := signUpVerified(t, srv, "alice@example.com")

	resp := srv.Do(t, http.MethodPost, "/v1/auth/signout", nil, bearer(s.AccessToken)...)

	wantProblem(t, resp, http.StatusBadRequest, "idempotency-key-required")
	if got := srv.Get(t, "/v1/me", bearer(s.AccessToken)...); got.StatusCode != http.StatusOK {
		t.Errorf("GET /v1/me after a refused sign-out = %d; want 200 (still signed in)", got.StatusCode)
	}
}

// NFR-R1 at the HTTP seam. A retried sign-out never reaches the replay:
// the first one revoked the token, so the retry is unauthenticated. The
// replay rules themselves are tested on the middleware
// (idempotency_test.go).
func TestARetriedSignOutAnswersUnauthenticated(t *testing.T) {
	srv := apptest.Start(t)
	s := signUpVerified(t, srv, "alice@example.com")
	signOut(t, srv, s.AccessToken, signOutKey)

	wantProblem(t, signOut(t, srv, s.AccessToken, signOutKey), http.StatusUnauthorized, "unauthenticated")
}

// --- Login throttling (FR-A4, Q34) ---

func failSignIns(t *testing.T, srv *apptest.Server, email string, n int) {
	t.Helper()
	for i := range n {
		resp := signIn(t, srv, email, "wrong password!")
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("failure %d = %d; want 401\n%s", i+1, resp.StatusCode, resp.Body)
		}
	}
}

func wantThrottled(t *testing.T, resp apptest.Response, retryAfter int) {
	t.Helper()
	wantProblem(t, resp, http.StatusTooManyRequests, "too-many-attempts")
	if got := resp.Header.Get("Retry-After"); got != strconv.Itoa(retryAfter) {
		t.Errorf("Retry-After = %q; want %d", got, retryAfter)
	}
}

func TestFourFailuresDoNotSlowSignInDown(t *testing.T) {
	srv := apptest.Start(t)
	signUpVerified(t, srv, "alice@example.com")
	failSignIns(t, srv, "alice@example.com", 4)

	if resp := signIn(t, srv, "alice@example.com", password); resp.StatusCode != http.StatusOK {
		t.Fatalf("sign in after 4 failures = %d; want 200\n%s", resp.StatusCode, resp.Body)
	}
}

func TestTheWaitGrowsAfterFiveFailures(t *testing.T) {
	srv := apptest.Start(t)
	signUpVerified(t, srv, "alice@example.com")
	failSignIns(t, srv, "alice@example.com", 5)

	// Even the right password waits: the attempt isn't checked.
	wantThrottled(t, signIn(t, srv, "alice@example.com", password), 1)

	for _, wait := range []time.Duration{time.Second, 2 * time.Second, 4 * time.Second} {
		srv.Clock.Advance(wait)
		failSignIns(t, srv, "alice@example.com", 1)
		wantThrottled(t, signIn(t, srv, "alice@example.com", "wrong password!"), int(2*wait/time.Second))
	}
}

func TestTheWaitNeverPassesFifteenMinutes(t *testing.T) {
	srv := apptest.Start(t)
	signUpVerified(t, srv, "alice@example.com")
	failSignIns(t, srv, "alice@example.com", 5)
	for range 12 {
		srv.Clock.Advance(15 * time.Minute)
		failSignIns(t, srv, "alice@example.com", 1)
	}

	wantThrottled(t, signIn(t, srv, "alice@example.com", password), 15*60)

	// No lockout: once the wait ends, the right password works.
	srv.Clock.Advance(15 * time.Minute)
	if resp := signIn(t, srv, "alice@example.com", password); resp.StatusCode != http.StatusOK {
		t.Fatalf("sign in after the wait = %d; want 200\n%s", resp.StatusCode, resp.Body)
	}
}

func TestASuccessfulSignInClearsTheAccountsCount(t *testing.T) {
	srv := apptest.Start(t)
	signUpVerified(t, srv, "alice@example.com")
	failSignIns(t, srv, "alice@example.com", 4)
	signIn(t, srv, "alice@example.com", password)

	// From another IP, whose own count starts at zero: only the account's
	// count could slow these down.
	other := srv.FromOtherIP(t)
	failSignIns(t, other, "Alice@Example.com", 4)

	if resp := signIn(t, other, "alice@example.com", password); resp.StatusCode != http.StatusOK {
		t.Fatalf("sign in = %d; want 200 (count cleared by the earlier success)\n%s", resp.StatusCode, resp.Body)
	}
}

// The per-account count follows the account to every IP.
func TestTheAccountsCountHoldsOnEveryIP(t *testing.T) {
	srv := apptest.Start(t)
	signUpVerified(t, srv, "alice@example.com")
	failSignIns(t, srv, "alice@example.com", 5)

	wantThrottled(t, signIn(t, srv.FromOtherIP(t), "alice@example.com", password), 1)
}

func TestAnHourWithoutFailuresClearsTheCount(t *testing.T) {
	srv := apptest.Start(t)
	signUpVerified(t, srv, "alice@example.com")
	failSignIns(t, srv, "alice@example.com", 5)

	srv.Clock.Advance(time.Hour)
	failSignIns(t, srv, "alice@example.com", 4)

	if resp := signIn(t, srv, "alice@example.com", password); resp.StatusCode != http.StatusOK {
		t.Fatalf("sign in = %d; want 200 (count started over)\n%s", resp.StatusCode, resp.Body)
	}
}

// Unknown emails are throttled exactly like real ones (doc 08: no account
// enumeration).
func TestUnknownEmailsAreThrottledToo(t *testing.T) {
	srv := apptest.Start(t)
	failSignIns(t, srv, "nobody@example.com", 5)

	wantThrottled(t, signIn(t, srv, "nobody@example.com", password), 1)
}

// Per IP: guessing across many accounts from one address slows down too.
func TestFailuresFromOneIPAcrossAccountsAreThrottled(t *testing.T) {
	srv := apptest.Start(t)
	signUpVerified(t, srv, "alice@example.com")
	for i := range 5 {
		failSignIns(t, srv, fmt.Sprintf("user%d@example.com", i), 1)
	}

	wantThrottled(t, signIn(t, srv, "alice@example.com", password), 1)
	if resp := signIn(t, srv.FromOtherIP(t), "alice@example.com", password); resp.StatusCode != http.StatusOK {
		t.Errorf("sign in from another IP = %d; want 200", resp.StatusCode)
	}
}

// FR-U2: throttling is a form error on sign-in, never a session expiry; the
// signed-in Sessions keep working.
func TestThrottlingLeavesExistingSessionsAlone(t *testing.T) {
	srv := apptest.Start(t)
	s := signUpVerified(t, srv, "alice@example.com")
	failSignIns(t, srv, "alice@example.com", 5)
	wantThrottled(t, signIn(t, srv, "alice@example.com", password), 1)

	if got := srv.Get(t, "/v1/me", bearer(s.AccessToken)...); got.StatusCode != http.StatusOK {
		t.Errorf("GET /v1/me while throttled = %d; want 200", got.StatusCode)
	}
}
