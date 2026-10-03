package httpapi_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/bikkysamuel/splitsDemo/server/internal/apptest"
)

type money struct {
	Minor    int64  `json:"minor"`
	Currency string `json:"currency"`
}

type shareLine struct {
	MemberID string  `json:"member_id"`
	Share    money   `json:"share"`
	Input    *string `json:"input"`
}

type expense struct {
	ID          string      `json:"id"`
	GroupID     string      `json:"group_id"`
	Payer       string      `json:"payer_member_id"`
	CreatedBy   string      `json:"created_by_member_id"`
	Amount      money       `json:"amount"`
	Category    string      `json:"category"`
	Note        *string     `json:"note"`
	SpentOn     string      `json:"spent_on"`
	SplitMethod string      `json:"split_method"`
	State       string      `json:"state"`
	Version     int         `json:"version"`
	Shares      []shareLine `json:"shares"`
}

type expensePage struct {
	Items []struct {
		ID      string `json:"id"`
		SpentOn string `json:"spent_on"`
		Amount  money  `json:"amount"`
	} `json:"items"`
	NextCursor *string `json:"next_cursor"`
}

// trip is a Group of Alice (Admin, join_seq 1), Bob (linked, 2) and the
// Placeholder Grandma (3).
type trip struct {
	srv     *apptest.Server
	alice   authSession
	bob     authSession
	group   group
	aliceID string
	bobID   string
	grandma string
}

func newTrip(t *testing.T) trip {
	t.Helper()
	srv := apptest.Start(t)
	tr := trip{srv: srv, alice: signUpVerified(t, srv, "alice@example.com"), bob: signUpVerified(t, srv, "bob@example.com")}
	tr.group = createGroup(t, srv, tr.alice.AccessToken, "Goa trip")
	tr.aliceID = tr.group.MyMemberID
	tr.bobID = mustAddMember(t, srv, tr.alice.AccessToken, tr.group.ID, map[string]string{"display_name": "Bob", "email": "bob@example.com"}).ID
	tr.grandma = mustAddMember(t, srv, tr.alice.AccessToken, tr.group.ID, map[string]string{"display_name": "Grandma"}).ID
	return tr
}

func (tr trip) input(minor int64, payer string, members ...string) map[string]any {
	split := make([]map[string]string, len(members))
	for i, m := range members {
		split[i] = map[string]string{"member_id": m}
	}
	return map[string]any{
		"payer_member_id": payer,
		"amount":          map[string]any{"minor": minor, "currency": "INR"},
		"category":        "food_drink",
		"note":            "  Dinner  ",
		"spent_on":        "2026-10-01",
		"split":           map[string]any{"method": "equal", "members": split},
	}
}

func (tr trip) everyone() []string { return []string{tr.aliceID, tr.bobID, tr.grandma} }

func (tr trip) preview(t *testing.T, token string, body any) apptest.Response {
	t.Helper()
	return tr.srv.Post(t, "/v1/groups/"+tr.group.ID+"/expenses/preview", body, "Authorization", "Bearer "+token)
}

func (tr trip) create(t *testing.T, token string, body any) apptest.Response {
	t.Helper()
	return write(t, tr.srv, http.MethodPost, "/v1/groups/"+tr.group.ID+"/expenses", token, body)
}

func (tr trip) mustCreate(t *testing.T, body any) expense {
	t.Helper()
	resp := tr.create(t, tr.alice.AccessToken, body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create expense = %d; want 201\n%s", resp.StatusCode, resp.Body)
	}
	var e expense
	resp.JSON(t, &e)
	return e
}

func shareAmounts(lines []shareLine) map[string]int64 {
	m := map[string]int64{}
	for _, l := range lines {
		m[l.MemberID] = l.Share.Minor
	}
	return m
}

// --- Preview (FR-E4) ---

// ADR-0010: 100001 paise among three: floor 33333 each, and the two
// leftover paise go to the lowest join_seq on the tied remainders.
func TestPreviewSplitsEquallyWithTheLeftoverByJoinOrder(t *testing.T) {
	tr := newTrip(t)

	resp := tr.preview(t, tr.bob.AccessToken, tr.input(100001, tr.aliceID, tr.grandma, tr.bobID, tr.aliceID))

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("preview = %d; want 200\n%s", resp.StatusCode, resp.Body)
	}
	var p struct {
		Amount money       `json:"amount"`
		Shares []shareLine `json:"shares"`
	}
	resp.JSON(t, &p)
	if p.Amount != (money{100001, "INR"}) {
		t.Errorf("amount = %+v; want 100001 INR", p.Amount)
	}
	// In the Split's order: Grandma (3), Bob (2), Alice (1).
	want := []int64{33333, 33334, 33334}
	for i, l := range p.Shares {
		if l.Share.Minor != want[i] || l.Share.Currency != "INR" {
			t.Errorf("share %d = %+v; want %d INR", i, l, want[i])
		}
	}
}

// Preview and save share one code path: the saved Shares are the preview's.
func TestTheSavedSharesAreThePreviewsShares(t *testing.T) {
	tr := newTrip(t)
	in := tr.input(1000, tr.bobID, tr.everyone()...)
	var p struct {
		Shares []shareLine `json:"shares"`
	}
	tr.preview(t, tr.alice.AccessToken, in).JSON(t, &p)

	e := tr.mustCreate(t, in)

	if fmt.Sprint(shareAmounts(e.Shares)) != fmt.Sprint(shareAmounts(p.Shares)) {
		t.Errorf("saved shares %v; previewed %v", shareAmounts(e.Shares), shareAmounts(p.Shares))
	}
}

func TestPreviewSavesNothing(t *testing.T) {
	tr := newTrip(t)
	tr.preview(t, tr.alice.AccessToken, tr.input(1000, tr.aliceID, tr.everyone()...))

	var page expensePage
	tr.srv.Get(t, "/v1/groups/"+tr.group.ID+"/expenses", bearer(tr.alice.AccessToken)...).JSON(t, &page)
	if len(page.Items) != 0 {
		t.Errorf("expenses after a preview = %d; want 0", len(page.Items))
	}
}

// --- Create (FR-E1, FR-E3, D9) ---

func TestCreateRecordsAnAcceptedExpenseWithSharesInJoinOrder(t *testing.T) {
	tr := newTrip(t)

	e := tr.mustCreate(t, tr.input(100001, tr.grandma, tr.grandma, tr.bobID, tr.aliceID))

	if e.State != "accepted" || e.Payer != tr.grandma || e.CreatedBy != tr.aliceID || e.SplitMethod != "equal" ||
		e.Amount != (money{100001, "INR"}) || e.Category != "food_drink" || e.SpentOn != "2026-10-01" || e.Version != 1 {
		t.Errorf("expense = %+v", e)
	}
	if e.Note == nil || *e.Note != "Dinner" {
		t.Errorf("note = %v; want trimmed Dinner", e.Note)
	}
	if len(e.Shares) != 3 || e.Shares[0].MemberID != tr.aliceID || e.Shares[1].MemberID != tr.bobID || e.Shares[2].MemberID != tr.grandma {
		t.Errorf("shares = %+v; want Alice, Bob, Grandma (join order)", e.Shares)
	}

	var got expense
	tr.srv.Get(t, "/v1/expenses/"+e.ID, bearer(tr.bob.AccessToken)...).JSON(t, &got)
	if got.ID != e.ID || fmt.Sprint(got.Shares) != fmt.Sprint(e.Shares) {
		t.Errorf("GET expense = %+v; want %+v", got, e)
	}
}

// NFR-R2: the Expense, its Shares and its Activity History event commit
// together.
func TestCreateWritesTheActivityEventWithTheExpense(t *testing.T) {
	tr := newTrip(t)
	e := tr.mustCreate(t, tr.input(1000, tr.aliceID, tr.everyone()...))

	conn := connect(t, tr.srv)
	var n int
	err := conn.QueryRow(context.Background(),
		"SELECT count(*) FROM activity_events WHERE type = 'expense_created' AND subject_id = $1 AND actor_member_id = $2",
		e.ID, tr.aliceID).Scan(&n)
	if err != nil || n != 1 {
		t.Errorf("expense_created events = %d, %v; want 1", n, err)
	}
	var sum int64
	if err := conn.QueryRow(context.Background(), "SELECT sum(share_minor) FROM expense_shares WHERE expense_id = $1", e.ID).Scan(&sum); err != nil || sum != 1000 {
		t.Errorf("Σ shares = %d, %v; want 1000", sum, err)
	}
}

func TestCreateValidatesTheInput(t *testing.T) {
	tr := newTrip(t)
	outsider := "0190b6c4-0000-7000-8000-00000000abcd"
	for _, tc := range []struct {
		name   string
		change func(map[string]any)
		want   string
	}{
		{"zero amount", func(in map[string]any) { in["amount"] = map[string]any{"minor": 0, "currency": "INR"} }, "/amount/minor not_positive"},
		{"other currency", func(in map[string]any) { in["amount"] = map[string]any{"minor": 10, "currency": "EUR"} }, "/amount/currency not_group_currency"},
		{"unknown category", func(in map[string]any) { in["category"] = "jewellery" }, "/category invalid"},
		{"long note", func(in map[string]any) { in["note"] = strings.Repeat("n", 501) }, "/note too_long"},
		{"payer not a member", func(in map[string]any) { in["payer_member_id"] = outsider }, "/payer_member_id not_a_member"},
		{"split member not a member", func(in map[string]any) {
			in["split"] = map[string]any{"method": "equal", "members": []map[string]string{{"member_id": outsider}}}
		}, "/split/members/0/member_id not_a_member"},
		{"duplicate split member", func(in map[string]any) {
			in["split"] = map[string]any{"method": "equal", "members": []map[string]string{{"member_id": tr.aliceID}, {"member_id": tr.aliceID}}}
		}, "/split/members/1/member_id duplicate_member"},
		{"no split members", func(in map[string]any) {
			in["split"] = map[string]any{"method": "equal", "members": []map[string]string{}}
		}, "/split/members no_members"},
		{"unknown split method", func(in map[string]any) {
			in["split"] = map[string]any{"method": "thirds", "members": []map[string]string{{"member_id": tr.aliceID}}}
		}, "/split/method invalid"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := tr.input(1000, tr.aliceID, tr.everyone()...)
			tc.change(in)
			for name, resp := range map[string]apptest.Response{
				"create":  tr.create(t, tr.alice.AccessToken, in),
				"preview": tr.preview(t, tr.alice.AccessToken, in),
			} {
				wantProblem(t, resp, http.StatusBadRequest, "validation-failed")
				var p struct {
					Errors []struct{ Field, Code string } `json:"errors"`
				}
				resp.JSON(t, &p)
				if len(p.Errors) != 1 || p.Errors[0].Field+" "+p.Errors[0].Code != tc.want {
					t.Errorf("%s errors = %+v; want %s", name, p.Errors, tc.want)
				}
			}
		})
	}
}

func TestABadDateIsRefused(t *testing.T) {
	tr := newTrip(t)
	in := tr.input(1000, tr.aliceID, tr.aliceID)
	in["spent_on"] = "01/10/2026"

	if resp := tr.create(t, tr.alice.AccessToken, in); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("bad date = %d; want 400", resp.StatusCode)
	}
}

// NFR-R1: an idempotent retry never creates a second Expense.
func TestARetriedCreateNeverMakesASecondExpense(t *testing.T) {
	tr := newTrip(t)
	key := newKey()
	in := tr.input(1000, tr.aliceID, tr.everyone()...)
	path := "/v1/groups/" + tr.group.ID + "/expenses"

	first := writeWithKey(t, tr.srv, http.MethodPost, path, tr.alice.AccessToken, key, in)
	second := writeWithKey(t, tr.srv, http.MethodPost, path, tr.alice.AccessToken, key, in)

	if second.StatusCode != http.StatusCreated || string(second.Body) != string(first.Body) {
		t.Errorf("retry = %d; want the first 201 replayed", second.StatusCode)
	}
	var page expensePage
	tr.srv.Get(t, path, bearer(tr.alice.AccessToken)...).JSON(t, &page)
	if len(page.Items) != 1 {
		t.Errorf("expenses = %d; want 1", len(page.Items))
	}
}

// doc 07: 404 for non-Members, on every route.
func TestExpensesOfAGroupIAmNotInAreNotFound(t *testing.T) {
	tr := newTrip(t)
	e := tr.mustCreate(t, tr.input(1000, tr.aliceID, tr.everyone()...))
	mallory := signUpVerified(t, tr.srv, "mallory@example.com")
	in := tr.input(1000, tr.aliceID, tr.aliceID)

	wantProblem(t, tr.preview(t, mallory.AccessToken, in), http.StatusNotFound, "not-found")
	wantProblem(t, tr.create(t, mallory.AccessToken, in), http.StatusNotFound, "not-found")
	wantProblem(t, tr.srv.Get(t, "/v1/groups/"+tr.group.ID+"/expenses", bearer(mallory.AccessToken)...), http.StatusNotFound, "not-found")
	wantProblem(t, tr.srv.Get(t, "/v1/expenses/"+e.ID, bearer(mallory.AccessToken)...), http.StatusNotFound, "not-found")
	wantProblem(t, tr.srv.Get(t, "/v1/expenses/0190b6c4-0000-7000-8000-00000000eeee", bearer(tr.alice.AccessToken)...), http.StatusNotFound, "not-found")
}

// --- List ---

func TestListShowsNewestDateFirstThenNewestRecorded(t *testing.T) {
	tr := newTrip(t)
	var want []string
	for _, day := range []string{"2026-09-30", "2026-10-02", "2026-10-01", "2026-10-02"} {
		in := tr.input(1000, tr.aliceID, tr.aliceID)
		in["spent_on"] = day
		want = append(want, tr.mustCreate(t, in).ID)
	}
	// Expected: 10-02 (second), 10-02 (first), 10-01, 09-30.
	order := []string{want[3], want[1], want[2], want[0]}

	var got []string
	path := "/v1/groups/" + tr.group.ID + "/expenses?limit=3"
	for range 3 {
		var page expensePage
		tr.srv.Get(t, path, bearer(tr.bob.AccessToken)...).JSON(t, &page)
		for _, e := range page.Items {
			got = append(got, e.ID)
		}
		if page.NextCursor == nil {
			break
		}
		path = "/v1/groups/" + tr.group.ID + "/expenses?limit=3&cursor=" + *page.NextCursor
	}
	if strings.Join(got, ",") != strings.Join(order, ",") {
		t.Errorf("order = %v; want %v", got, order)
	}
}

// --- Database guards (doc 06 invariants) ---

func TestTheDatabaseGuardsTheLedgerInvariants(t *testing.T) {
	tr := newTrip(t)
	e := tr.mustCreate(t, tr.input(1000, tr.aliceID, tr.everyone()...))
	conn := connect(t, tr.srv)
	ctx := context.Background()

	if _, err := conn.Exec(ctx, "UPDATE activity_events SET type = 'x'"); err == nil {
		t.Error("updating activity_events succeeded; want append-only")
	}
	if _, err := conn.Exec(ctx, "DELETE FROM activity_events"); err == nil {
		t.Error("deleting activity_events succeeded; want append-only")
	}
	if _, err := conn.Exec(ctx, "UPDATE expense_shares SET share_minor = share_minor + 1 WHERE expense_id = $1", e.ID); err == nil {
		t.Error("breaking Σ shares = amount succeeded; want the deferred check to refuse it")
	}
	if _, err := conn.Exec(ctx, "UPDATE groups SET currency = 'EUR' WHERE id = $1", tr.group.ID); err == nil {
		t.Error("changing the Group Currency after an Expense succeeded; want it locked (D3)")
	}
}

func connect(t *testing.T, srv *apptest.Server) *pgx.Conn {
	t.Helper()
	conn, err := pgx.Connect(context.Background(), srv.DatabaseURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })
	return conn
}
