package httpapi_test

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/bikkysamuel/splitsDemo/server/internal/apptest"
)

type member struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	Status      string `json:"status"`
	Placeholder bool   `json:"placeholder"`
	JoinSeq     int    `json:"join_seq"`
	Version     int    `json:"version"`
}

type group struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Currency   string   `json:"currency"`
	State      string   `json:"state"`
	Version    int      `json:"version"`
	MyMemberID string   `json:"my_member_id"`
	Members    []member `json:"members"`
}

type groupPage struct {
	Items []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"items"`
	NextCursor *string `json:"next_cursor"`
}

// newKey returns a fresh Idempotency-Key.
func newKey() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	h := hex.EncodeToString(b)
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:]
}

// write sends a signed-in write with a fresh Idempotency-Key.
func write(t *testing.T, srv *apptest.Server, method, path, token string, body any) apptest.Response {
	t.Helper()
	return writeWithKey(t, srv, method, path, token, newKey(), body)
}

func writeWithKey(t *testing.T, srv *apptest.Server, method, path, token, key string, body any) apptest.Response {
	t.Helper()
	if method == http.MethodPost {
		return srv.Post(t, path, body, "Authorization", "Bearer "+token, "Idempotency-Key", key)
	}
	raw := []byte(fmt.Sprint(body))
	if s, ok := body.(string); ok {
		raw = []byte(s)
	}
	return srv.Do(t, method, path, raw,
		"Content-Type", "application/json", "Authorization", "Bearer "+token, "Idempotency-Key", key)
}

func createGroup(t *testing.T, srv *apptest.Server, token, name string) group {
	t.Helper()
	resp := write(t, srv, http.MethodPost, "/v1/groups", token,
		map[string]string{"name": name, "currency": "INR", "display_name": "Alice"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create group = %d; want 201\n%s", resp.StatusCode, resp.Body)
	}
	var g group
	resp.JSON(t, &g)
	return g
}

func rename(t *testing.T, srv *apptest.Server, token, groupID, name string, version int) apptest.Response {
	t.Helper()
	return write(t, srv, http.MethodPatch, "/v1/groups/"+groupID, token,
		fmt.Sprintf(`{"name":%q,"version":%d}`, name, version))
}

// --- Create (FR-G1) ---

func TestCreateGroupMakesTheCreatorItsFirstAdmin(t *testing.T) {
	srv := apptest.Start(t)
	alice := signUpVerified(t, srv, "alice@example.com")

	g := createGroup(t, srv, alice.AccessToken, "  Goa trip ")

	if g.Name != "Goa trip" || g.Currency != "INR" || g.State != "active" || g.Version != 1 {
		t.Errorf("group = %+v; want Goa trip, INR, active, version 1", g)
	}
	if len(g.Members) != 1 {
		t.Fatalf("members = %+v; want the creator only", g.Members)
	}
	m := g.Members[0]
	if m.ID != g.MyMemberID || m.DisplayName != "Alice" || m.Role != "admin" || m.Status != "active" ||
		m.Placeholder || m.JoinSeq != 1 {
		t.Errorf("creator = %+v; want Alice, admin, active, not a Placeholder, join_seq 1", m)
	}
}

func TestCreateGroupValidatesItsFields(t *testing.T) {
	srv := apptest.Start(t)
	alice := signUpVerified(t, srv, "alice@example.com")

	for _, tc := range []struct {
		name       string
		body       map[string]string
		wantErrors string
	}{
		{"empty name", map[string]string{"name": " ", "currency": "INR", "display_name": "Alice"}, "/name required"},
		{"long name", map[string]string{"name": strings.Repeat("n", 101), "currency": "INR", "display_name": "Alice"}, "/name too_long"},
		{"unknown currency", map[string]string{"name": "Trip", "currency": "ZZZ", "display_name": "Alice"}, "/currency invalid"},
		{"lowercase currency", map[string]string{"name": "Trip", "currency": "inr", "display_name": "Alice"}, "/currency invalid"},
		{"no display name", map[string]string{"name": "Trip", "currency": "INR", "display_name": ""}, "/display_name required"},
		{"long display name", map[string]string{"name": "Trip", "currency": "INR", "display_name": strings.Repeat("a", 51)}, "/display_name too_long"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp := write(t, srv, http.MethodPost, "/v1/groups", alice.AccessToken, tc.body)

			wantProblem(t, resp, http.StatusBadRequest, "validation-failed")
			var p struct {
				Errors []struct{ Field, Code string } `json:"errors"`
			}
			resp.JSON(t, &p)
			if len(p.Errors) != 1 || p.Errors[0].Field+" "+p.Errors[0].Code != tc.wantErrors {
				t.Errorf("errors = %+v; want %s", p.Errors, tc.wantErrors)
			}
		})
	}
}

func TestCreateGroupRequiresAVerifiedEmail(t *testing.T) {
	srv := apptest.Start(t)
	s := signUp(t, srv, "alice@example.com")

	resp := write(t, srv, http.MethodPost, "/v1/groups", s.AccessToken,
		map[string]string{"name": "Trip", "currency": "INR", "display_name": "Alice"})

	wantProblem(t, resp, http.StatusForbidden, "email-not-verified")
}

// NFR-R1: repeating a create with the same key replays it, creating once.
func TestCreateGroupReplaysARepeatedIdempotencyKey(t *testing.T) {
	srv := apptest.Start(t)
	alice := signUpVerified(t, srv, "alice@example.com")
	key := newKey()
	body := map[string]string{"name": "Trip", "currency": "INR", "display_name": "Alice"}

	first := writeWithKey(t, srv, http.MethodPost, "/v1/groups", alice.AccessToken, key, body)
	second := writeWithKey(t, srv, http.MethodPost, "/v1/groups", alice.AccessToken, key, body)

	if second.StatusCode != http.StatusCreated || string(second.Body) != string(first.Body) {
		t.Errorf("replay = %d %s; want the first 201 %s", second.StatusCode, second.Body, first.Body)
	}
	if second.Header.Get("Idempotent-Replayed") != "true" {
		t.Error("replay lacks Idempotent-Replayed: true")
	}
	var page groupPage
	srv.Get(t, "/v1/groups", bearer(alice.AccessToken)...).JSON(t, &page)
	if len(page.Items) != 1 {
		t.Errorf("Groups after a replayed create = %d; want 1", len(page.Items))
	}

	body["name"] = "Other"
	wantProblem(t, writeWithKey(t, srv, http.MethodPost, "/v1/groups", alice.AccessToken, key, body),
		http.StatusUnprocessableEntity, "idempotency-key-reused")
}

// FR-G2: at most 200 Groups per User.
func TestTheTwoHundredAndFirstGroupIsRefused(t *testing.T) {
	srv := apptest.Start(t)
	alice := signUpVerified(t, srv, "alice@example.com")
	for i := range 200 {
		createGroup(t, srv, alice.AccessToken, fmt.Sprintf("Group %d", i))
	}

	resp := write(t, srv, http.MethodPost, "/v1/groups", alice.AccessToken,
		map[string]string{"name": "One too many", "currency": "INR", "display_name": "Alice"})

	wantProblem(t, resp, http.StatusConflict, "group-limit-reached")
}

// --- List ---

func TestListGroupsShowsOnlyMyGroupsOldestFirst(t *testing.T) {
	srv := apptest.Start(t)
	alice := signUpVerified(t, srv, "alice@example.com")
	bob := signUpVerified(t, srv, "bob@example.com")
	first := createGroup(t, srv, alice.AccessToken, "First")
	createGroup(t, srv, bob.AccessToken, "Bob's")
	second := createGroup(t, srv, alice.AccessToken, "Second")

	var page groupPage
	srv.Get(t, "/v1/groups", bearer(alice.AccessToken)...).JSON(t, &page)

	if len(page.Items) != 2 || page.Items[0].ID != first.ID || page.Items[1].ID != second.ID {
		t.Errorf("items = %+v; want First then Second", page.Items)
	}
	if page.NextCursor != nil {
		t.Errorf("next_cursor = %q; want null on the last page", *page.NextCursor)
	}
}

func TestListGroupsPagesWithACursor(t *testing.T) {
	srv := apptest.Start(t)
	alice := signUpVerified(t, srv, "alice@example.com")
	var want []string
	for i := range 5 {
		want = append(want, createGroup(t, srv, alice.AccessToken, fmt.Sprintf("G%d", i)).ID)
	}

	var got []string
	path := "/v1/groups?limit=2"
	for pages := 0; ; pages++ {
		if pages > 5 {
			t.Fatal("too many pages")
		}
		var page groupPage
		srv.Get(t, path, bearer(alice.AccessToken)...).JSON(t, &page)
		for _, g := range page.Items {
			got = append(got, g.ID)
		}
		if page.NextCursor == nil {
			break
		}
		path = "/v1/groups?limit=2&cursor=" + *page.NextCursor
	}

	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("paged IDs = %v; want %v", got, want)
	}
}

func TestListGroupsRefusesABadCursorOrLimit(t *testing.T) {
	srv := apptest.Start(t)
	alice := signUpVerified(t, srv, "alice@example.com")

	wantProblem(t, srv.Get(t, "/v1/groups?cursor=not-a-cursor!", bearer(alice.AccessToken)...),
		http.StatusBadRequest, "invalid-cursor")
	for _, limit := range []string{"0", "201", "x"} {
		if resp := srv.Get(t, "/v1/groups?limit="+limit, bearer(alice.AccessToken)...); resp.StatusCode != http.StatusBadRequest {
			t.Errorf("limit=%s = %d; want 400", limit, resp.StatusCode)
		}
	}
}

// --- Get: 404, never 403 (doc 07) ---

func TestGetGroupReturnsItsDetailToAMember(t *testing.T) {
	srv := apptest.Start(t)
	alice := signUpVerified(t, srv, "alice@example.com")
	created := createGroup(t, srv, alice.AccessToken, "Trip")

	resp := srv.Get(t, "/v1/groups/"+created.ID, bearer(alice.AccessToken)...)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET group = %d; want 200\n%s", resp.StatusCode, resp.Body)
	}
	var g group
	resp.JSON(t, &g)
	if g.ID != created.ID || g.MyMemberID != created.MyMemberID {
		t.Errorf("group = %+v; want %+v", g, created)
	}
}

func TestAGroupIAmNotInLooksLikeOneThatDoesNotExist(t *testing.T) {
	srv := apptest.Start(t)
	alice := signUpVerified(t, srv, "alice@example.com")
	bob := signUpVerified(t, srv, "bob@example.com")
	alices := createGroup(t, srv, alice.AccessToken, "Alice's")

	notMine := srv.Get(t, "/v1/groups/"+alices.ID, bearer(bob.AccessToken)...)
	missing := srv.Get(t, "/v1/groups/0190b6c4-0000-7000-8000-00000000ffff", bearer(bob.AccessToken)...)

	wantProblem(t, notMine, http.StatusNotFound, "not-found")
	wantProblem(t, missing, http.StatusNotFound, "not-found")
	if string(notMine.Body) != string(missing.Body) {
		t.Errorf("bodies differ:\n%s\n%s", notMine.Body, missing.Body)
	}
	wantProblem(t, rename(t, srv, bob.AccessToken, alices.ID, "Mine now", 1), http.StatusNotFound, "not-found")
}

func TestGetGroupRefusesAMalformedID(t *testing.T) {
	srv := apptest.Start(t)
	alice := signUpVerified(t, srv, "alice@example.com")

	wantProblem(t, srv.Get(t, "/v1/groups/not-a-uuid", bearer(alice.AccessToken)...), http.StatusBadRequest, "invalid-request")
}

// --- Rename (FR-G3, NFR-R4) ---

func TestAnAdminRenamesTheGroup(t *testing.T) {
	srv := apptest.Start(t)
	alice := signUpVerified(t, srv, "alice@example.com")
	g := createGroup(t, srv, alice.AccessToken, "Trip")

	resp := rename(t, srv, alice.AccessToken, g.ID, " Goa 2026 ", g.Version)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("rename = %d; want 200\n%s", resp.StatusCode, resp.Body)
	}
	var renamed group
	resp.JSON(t, &renamed)
	if renamed.Name != "Goa 2026" || renamed.Version != g.Version+1 {
		t.Errorf("renamed = %+v; want Goa 2026 at version %d", renamed, g.Version+1)
	}
}

func TestRenamingWithAStaleVersionConflicts(t *testing.T) {
	srv := apptest.Start(t)
	alice := signUpVerified(t, srv, "alice@example.com")
	g := createGroup(t, srv, alice.AccessToken, "Trip")
	rename(t, srv, alice.AccessToken, g.ID, "First rename", g.Version)

	resp := rename(t, srv, alice.AccessToken, g.ID, "Second rename", g.Version)

	wantProblem(t, resp, http.StatusConflict, "version-conflict")
	var now group
	srv.Get(t, "/v1/groups/"+g.ID, bearer(alice.AccessToken)...).JSON(t, &now)
	if now.Name != "First rename" {
		t.Errorf("name = %q; want the first rename kept", now.Name)
	}
}

func TestRenameValidatesTheName(t *testing.T) {
	srv := apptest.Start(t)
	alice := signUpVerified(t, srv, "alice@example.com")
	g := createGroup(t, srv, alice.AccessToken, "Trip")

	wantProblem(t, rename(t, srv, alice.AccessToken, g.ID, "  ", g.Version), http.StatusBadRequest, "validation-failed")
}

// FR-G2 under concurrency: parallel creations can't pass the limit
// together.
func TestParallelCreationsStopAtTheLimit(t *testing.T) {
	srv := apptest.Start(t)
	alice := signUpVerified(t, srv, "alice@example.com")
	for i := range 198 {
		createGroup(t, srv, alice.AccessToken, fmt.Sprintf("Group %d", i))
	}

	statuses := make(chan int, 5)
	var wg sync.WaitGroup
	for i := range 5 {
		wg.Go(func() {
			statuses <- write(t, srv, http.MethodPost, "/v1/groups", alice.AccessToken,
				map[string]string{"name": fmt.Sprintf("Race %d", i), "currency": "INR", "display_name": "Alice"}).StatusCode
		})
	}
	wg.Wait()
	close(statuses)

	counts := map[int]int{}
	for s := range statuses {
		counts[s]++
	}
	if counts[http.StatusCreated] != 2 || counts[http.StatusConflict] != 3 {
		t.Errorf("statuses = %v; want 2 × 201 and 3 × 409", counts)
	}
}
