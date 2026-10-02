package httpapi_test

import (
	"fmt"
	"net/http"
	"sort"
	"sync"
	"testing"

	"github.com/bikkysamuel/splitsDemo/server/internal/apptest"
)

func addMember(t *testing.T, srv *apptest.Server, token, groupID string, body map[string]string) apptest.Response {
	t.Helper()
	return write(t, srv, http.MethodPost, "/v1/groups/"+groupID+"/members", token, body)
}

func mustAddMember(t *testing.T, srv *apptest.Server, token, groupID string, body map[string]string) member {
	t.Helper()
	resp := addMember(t, srv, token, groupID, body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("add member %v = %d; want 201\n%s", body, resp.StatusCode, resp.Body)
	}
	var m member
	resp.JSON(t, &m)
	return m
}

func makeAdmin(t *testing.T, srv *apptest.Server, token, groupID, memberID string, version int) apptest.Response {
	t.Helper()
	return write(t, srv, http.MethodPatch, "/v1/groups/"+groupID+"/members/"+memberID, token,
		fmt.Sprintf(`{"role":"admin","version":%d}`, version))
}

func groupsOf(t *testing.T, srv *apptest.Server, token string) []string {
	t.Helper()
	var page groupPage
	srv.Get(t, "/v1/groups", bearer(token)...).JSON(t, &page)
	var ids []string
	for _, g := range page.Items {
		ids = append(ids, g.ID)
	}
	return ids
}

// --- Add (FR-M1, FR-M2, ADR-0017) ---

func TestAddingAVerifiedUsersEmailMakesThemAMemberAtOnce(t *testing.T) {
	srv := apptest.Start(t)
	alice := signUpVerified(t, srv, "alice@example.com")
	bob := signUpVerified(t, srv, "bob@example.com")
	g := createGroup(t, srv, alice.AccessToken, "Trip")

	m := mustAddMember(t, srv, alice.AccessToken, g.ID, map[string]string{"display_name": "Bob", "email": " BOB@example.com "})

	if m.Placeholder || m.Role != "member" || m.Status != "active" || m.JoinSeq != 2 || m.DisplayName != "Bob" {
		t.Errorf("member = %+v; want Bob, a linked member with join_seq 2", m)
	}
	if ids := groupsOf(t, srv, bob.AccessToken); len(ids) != 1 || ids[0] != g.ID {
		t.Errorf("Bob's Groups = %v; want [%s]", ids, g.ID)
	}
	var seen group
	srv.Get(t, "/v1/groups/"+g.ID, bearer(bob.AccessToken)...).JSON(t, &seen)
	if seen.MyMemberID != m.ID {
		t.Errorf("Bob's my_member_id = %s; want %s", seen.MyMemberID, m.ID)
	}
}

func TestAnUnknownEmailBecomesAPlaceholder(t *testing.T) {
	srv := apptest.Start(t)
	alice := signUpVerified(t, srv, "alice@example.com")
	g := createGroup(t, srv, alice.AccessToken, "Trip")

	m := mustAddMember(t, srv, alice.AccessToken, g.ID, map[string]string{"display_name": "Carol", "email": "carol@example.com"})

	if !m.Placeholder {
		t.Errorf("member = %+v; want a Placeholder", m)
	}
}

// ADR-0017: only a verified email links a User; an unverified sign-up
// stays a Placeholder until it verifies (Claim: #16).
func TestAnUnverifiedUsersEmailBecomesAPlaceholder(t *testing.T) {
	srv := apptest.Start(t)
	alice := signUpVerified(t, srv, "alice@example.com")
	dave := signUp(t, srv, "dave@example.com")
	g := createGroup(t, srv, alice.AccessToken, "Trip")

	m := mustAddMember(t, srv, alice.AccessToken, g.ID, map[string]string{"display_name": "Dave", "email": "dave@example.com"})

	if !m.Placeholder {
		t.Errorf("member = %+v; want a Placeholder", m)
	}
	wantProblem(t, srv.Get(t, "/v1/groups", bearer(dave.AccessToken)...), http.StatusForbidden, "email-not-verified")
}

func TestANameOnlyPlaceholder(t *testing.T) {
	srv := apptest.Start(t)
	alice := signUpVerified(t, srv, "alice@example.com")
	g := createGroup(t, srv, alice.AccessToken, "Trip")

	m := mustAddMember(t, srv, alice.AccessToken, g.ID, map[string]string{"display_name": "  Grandma "})

	if !m.Placeholder || m.DisplayName != "Grandma" {
		t.Errorf("member = %+v; want the Placeholder Grandma", m)
	}
}

// Any Member may add Members (FR-G3), not only Admins.
func TestAnOrdinaryMemberCanAddMembers(t *testing.T) {
	srv := apptest.Start(t)
	alice := signUpVerified(t, srv, "alice@example.com")
	bob := signUpVerified(t, srv, "bob@example.com")
	g := createGroup(t, srv, alice.AccessToken, "Trip")
	mustAddMember(t, srv, alice.AccessToken, g.ID, map[string]string{"display_name": "Bob", "email": "bob@example.com"})

	mustAddMember(t, srv, bob.AccessToken, g.ID, map[string]string{"display_name": "Erin"})
}

func TestDuplicatesAreRefused(t *testing.T) {
	srv := apptest.Start(t)
	alice := signUpVerified(t, srv, "alice@example.com")
	signUpVerified(t, srv, "bob@example.com")
	g := createGroup(t, srv, alice.AccessToken, "Trip")
	mustAddMember(t, srv, alice.AccessToken, g.ID, map[string]string{"display_name": "Bob", "email": "bob@example.com"})
	mustAddMember(t, srv, alice.AccessToken, g.ID, map[string]string{"display_name": "Carol", "email": "carol@example.com"})

	for _, tc := range []struct {
		name string
		body map[string]string
		want string
	}{
		{"display name, other case", map[string]string{"display_name": "bob"}, "/display_name taken"},
		{"a member's User email", map[string]string{"display_name": "Robert", "email": "Bob@Example.com"}, "/email taken"},
		{"the creator's own email", map[string]string{"display_name": "Me again", "email": "alice@example.com"}, "/email taken"},
		{"a placeholder's email", map[string]string{"display_name": "Caroline", "email": "carol@example.com"}, "/email taken"},
		{"malformed email", map[string]string{"display_name": "Frank", "email": "frank"}, "/email invalid"},
		{"no display name", map[string]string{"display_name": " "}, "/display_name required"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp := addMember(t, srv, alice.AccessToken, g.ID, tc.body)
			wantProblem(t, resp, http.StatusBadRequest, "validation-failed")
			var p struct {
				Errors []struct{ Field, Code string } `json:"errors"`
			}
			resp.JSON(t, &p)
			if len(p.Errors) != 1 || p.Errors[0].Field+" "+p.Errors[0].Code != tc.want {
				t.Errorf("errors = %+v; want %s", p.Errors, tc.want)
			}
		})
	}
}

// FR-G2: at most 50 Members, Placeholders included.
func TestTheFiftyFirstMemberIsRefused(t *testing.T) {
	srv := apptest.Start(t)
	alice := signUpVerified(t, srv, "alice@example.com")
	g := createGroup(t, srv, alice.AccessToken, "Trip")
	for i := range 49 {
		mustAddMember(t, srv, alice.AccessToken, g.ID, map[string]string{"display_name": fmt.Sprintf("P%d", i)})
	}

	wantProblem(t, addMember(t, srv, alice.AccessToken, g.ID, map[string]string{"display_name": "One too many"}),
		http.StatusConflict, "member-limit-reached")
}

// join_seq stays unique and gapless under concurrent adds.
func TestParallelAddsGetDistinctJoinSeqs(t *testing.T) {
	srv := apptest.Start(t)
	alice := signUpVerified(t, srv, "alice@example.com")
	g := createGroup(t, srv, alice.AccessToken, "Trip")

	seqs := make(chan int, 10)
	var wg sync.WaitGroup
	for i := range 10 {
		wg.Go(func() {
			var m member
			resp := addMember(t, srv, alice.AccessToken, g.ID, map[string]string{"display_name": fmt.Sprintf("P%d", i)})
			if resp.StatusCode == http.StatusCreated {
				resp.JSON(t, &m)
			}
			seqs <- m.JoinSeq
		})
	}
	wg.Wait()
	close(seqs)

	var got []int
	for s := range seqs {
		got = append(got, s)
	}
	sort.Ints(got)
	if fmt.Sprint(got) != "[2 3 4 5 6 7 8 9 10 11]" {
		t.Errorf("join_seqs = %v; want 2…11", got)
	}
}

func TestAddingToAGroupIAmNotInIsNotFound(t *testing.T) {
	srv := apptest.Start(t)
	alice := signUpVerified(t, srv, "alice@example.com")
	mallory := signUpVerified(t, srv, "mallory@example.com")
	g := createGroup(t, srv, alice.AccessToken, "Trip")

	wantProblem(t, addMember(t, srv, mallory.AccessToken, g.ID, map[string]string{"display_name": "Mallory"}),
		http.StatusNotFound, "not-found")
}

func TestAddMemberReplaysARepeatedIdempotencyKey(t *testing.T) {
	srv := apptest.Start(t)
	alice := signUpVerified(t, srv, "alice@example.com")
	g := createGroup(t, srv, alice.AccessToken, "Trip")
	key := newKey()
	body := map[string]string{"display_name": "Grandma"}

	first := writeWithKey(t, srv, http.MethodPost, "/v1/groups/"+g.ID+"/members", alice.AccessToken, key, body)
	second := writeWithKey(t, srv, http.MethodPost, "/v1/groups/"+g.ID+"/members", alice.AccessToken, key, body)

	if second.StatusCode != http.StatusCreated || string(second.Body) != string(first.Body) {
		t.Errorf("replay = %d %s; want the first 201", second.StatusCode, second.Body)
	}
	var now group
	srv.Get(t, "/v1/groups/"+g.ID, bearer(alice.AccessToken)...).JSON(t, &now)
	if len(now.Members) != 2 {
		t.Errorf("members = %d; want 2 (added once)", len(now.Members))
	}
}

// --- Grant Admin (FR-G3) ---

func TestAnAdminMakesAMemberAnAdmin(t *testing.T) {
	srv := apptest.Start(t)
	alice := signUpVerified(t, srv, "alice@example.com")
	bob := signUpVerified(t, srv, "bob@example.com")
	g := createGroup(t, srv, alice.AccessToken, "Trip")
	m := mustAddMember(t, srv, alice.AccessToken, g.ID, map[string]string{"display_name": "Bob", "email": "bob@example.com"})

	resp := makeAdmin(t, srv, alice.AccessToken, g.ID, m.ID, m.Version)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("make admin = %d; want 200\n%s", resp.StatusCode, resp.Body)
	}
	var updated member
	resp.JSON(t, &updated)
	if updated.Role != "admin" || updated.Version != m.Version+1 {
		t.Errorf("member = %+v; want admin at version %d", updated, m.Version+1)
	}
	// Bob can now do Admin things.
	if r := rename(t, srv, bob.AccessToken, g.ID, "Bob's trip", g.Version); r.StatusCode != http.StatusOK {
		t.Errorf("Bob renaming as Admin = %d; want 200\n%s", r.StatusCode, r.Body)
	}
}

func TestAMemberWhoIsNotAnAdminCannotGrantAdminOrRename(t *testing.T) {
	srv := apptest.Start(t)
	alice := signUpVerified(t, srv, "alice@example.com")
	bob := signUpVerified(t, srv, "bob@example.com")
	g := createGroup(t, srv, alice.AccessToken, "Trip")
	mustAddMember(t, srv, alice.AccessToken, g.ID, map[string]string{"display_name": "Bob", "email": "bob@example.com"})
	signUpVerified(t, srv, "erin@example.com")
	e := mustAddMember(t, srv, alice.AccessToken, g.ID, map[string]string{"display_name": "Erin", "email": "erin@example.com"})

	wantProblem(t, makeAdmin(t, srv, bob.AccessToken, g.ID, e.ID, e.Version), http.StatusForbidden, "admin-required")
	wantProblem(t, rename(t, srv, bob.AccessToken, g.ID, "Mine", g.Version), http.StatusForbidden, "admin-required")
}

func TestPlaceholdersCannotBeAdmins(t *testing.T) {
	srv := apptest.Start(t)
	alice := signUpVerified(t, srv, "alice@example.com")
	g := createGroup(t, srv, alice.AccessToken, "Trip")
	p := mustAddMember(t, srv, alice.AccessToken, g.ID, map[string]string{"display_name": "Grandma"})

	wantProblem(t, makeAdmin(t, srv, alice.AccessToken, g.ID, p.ID, p.Version), http.StatusConflict, "member-not-eligible")
}

func TestGrantingAdminWithAStaleVersionConflicts(t *testing.T) {
	srv := apptest.Start(t)
	alice := signUpVerified(t, srv, "alice@example.com")
	signUpVerified(t, srv, "bob@example.com")
	g := createGroup(t, srv, alice.AccessToken, "Trip")
	m := mustAddMember(t, srv, alice.AccessToken, g.ID, map[string]string{"display_name": "Bob", "email": "bob@example.com"})
	makeAdmin(t, srv, alice.AccessToken, g.ID, m.ID, m.Version)

	wantProblem(t, makeAdmin(t, srv, alice.AccessToken, g.ID, m.ID, m.Version), http.StatusConflict, "version-conflict")
}

func TestGrantingAdminToAMemberOfAnotherGroupIsNotFound(t *testing.T) {
	srv := apptest.Start(t)
	alice := signUpVerified(t, srv, "alice@example.com")
	signUpVerified(t, srv, "bob@example.com")
	mine := createGroup(t, srv, alice.AccessToken, "Mine")
	other := createGroup(t, srv, alice.AccessToken, "Other")
	m := mustAddMember(t, srv, alice.AccessToken, other.ID, map[string]string{"display_name": "Bob", "email": "bob@example.com"})

	wantProblem(t, makeAdmin(t, srv, alice.AccessToken, mine.ID, m.ID, m.Version), http.StatusNotFound, "not-found")
}
