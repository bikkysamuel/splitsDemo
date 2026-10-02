package httpapi_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
)

type settlement struct {
	ID        string  `json:"id"`
	From      string  `json:"from_member_id"`
	To        string  `json:"to_member_id"`
	Amount    money   `json:"amount"`
	SettledOn string  `json:"settled_on"`
	Note      *string `json:"note"`
	CreatedBy string  `json:"created_by_member_id"`
	State     string  `json:"state"`
	Version   int     `json:"version"`
}

func (tr trip) settlementInput(from, to string, minor int64) map[string]any {
	return map[string]any{
		"from_member_id": from, "to_member_id": to,
		"amount":     map[string]any{"minor": minor, "currency": "INR"},
		"settled_on": "2026-10-02", "note": " UPI ",
	}
}

func (tr trip) settle(t *testing.T, token string, body map[string]any) (settlement, int, string) {
	t.Helper()
	resp := write(t, tr.srv, http.MethodPost, "/v1/groups/"+tr.group.ID+"/settlements", token, body)
	var s settlement
	if resp.StatusCode == http.StatusCreated {
		resp.JSON(t, &s)
	}
	return s, resp.StatusCode, resp.ProblemType(t)
}

// owed sets up: Alice paid 900 for all three, so Bob and Grandma owe 300.
func owed(t *testing.T) trip {
	t.Helper()
	tr := newTrip(t)
	tr.mustCreate(t, tr.input(900, tr.aliceID, tr.everyone()...))
	return tr
}

func TestAPartialSettlementCountsAtOnce(t *testing.T) {
	tr := owed(t)

	s, status, _ := tr.settle(t, tr.bob.AccessToken, tr.settlementInput(tr.bobID, tr.aliceID, 100))

	if status != http.StatusCreated || s.State != "accepted" || s.Amount != (money{100, "INR"}) || s.CreatedBy != tr.bobID ||
		s.Note == nil || *s.Note != "UPI" || s.SettledOn != "2026-10-02" || s.Version != 1 {
		t.Fatalf("settlement = %d %+v", status, s)
	}
	b := tr.balances(t, tr.alice.AccessToken)
	if b.of(t, tr.aliceID) != 500 || b.of(t, tr.bobID) != -200 || b.of(t, tr.grandma) != -300 {
		t.Errorf("balances = %+v; want Alice 500, Bob -200, Grandma -300", b.Balances)
	}
	var got settlement
	tr.srv.Get(t, "/v1/settlements/"+s.ID, bearer(tr.alice.AccessToken)...).JSON(t, &got)
	if got.ID != s.ID || got.Amount != s.Amount || got.State != s.State || got.Note == nil || *got.Note != "UPI" {
		t.Errorf("GET = %+v; want %+v", got, s)
	}
}

// Anyone records a Settlement (FR-S1), even one between two others,
// such as an Admin for a Placeholder.
func TestAnyMemberRecordsASettlementBetweenOthers(t *testing.T) {
	tr := owed(t)

	if _, status, _ := tr.settle(t, tr.bob.AccessToken, tr.settlementInput(tr.grandma, tr.aliceID, 300)); status != http.StatusCreated {
		t.Errorf("Bob recording Grandma → Alice = %d; want 201", status)
	}
}

// FR-S2, D16: an overpayment needs acknowledging, then saves.
func TestAnOverpaymentNeedsAcknowledging(t *testing.T) {
	tr := owed(t)
	in := tr.settlementInput(tr.bobID, tr.aliceID, 301)

	resp := write(t, tr.srv, http.MethodPost, "/v1/groups/"+tr.group.ID+"/settlements", tr.bob.AccessToken, in)

	wantProblem(t, resp, http.StatusUnprocessableEntity, "confirmation-required")
	var p struct {
		Warnings []struct{ Code string } `json:"warnings"`
	}
	resp.JSON(t, &p)
	if len(p.Warnings) != 1 || p.Warnings[0].Code != "overpayment" {
		t.Errorf("warnings = %+v; want [overpayment]", p.Warnings)
	}
	if b := tr.balances(t, tr.bob.AccessToken); b.of(t, tr.bobID) != -300 {
		t.Errorf("Bob's Balance after the warning = %d; want -300 (nothing saved)", b.of(t, tr.bobID))
	}

	in["acknowledge_warnings"] = true
	s, status, _ := tr.settle(t, tr.bob.AccessToken, in)

	if status != http.StatusCreated || s.Amount.Minor != 301 {
		t.Fatalf("acknowledged = %d %+v; want 201", status, s)
	}
	if b := tr.balances(t, tr.bob.AccessToken); b.of(t, tr.bobID) != 1 || b.of(t, tr.aliceID) != 299 {
		t.Errorf("balances = %+v; want Bob +1, Alice 299", b.Balances)
	}
}

func TestPayingSomeoneWhoIsOwedNothingWarns(t *testing.T) {
	tr := owed(t)

	if _, status, slug := tr.settle(t, tr.bob.AccessToken, tr.settlementInput(tr.bobID, tr.grandma, 50)); status != http.StatusUnprocessableEntity || slug != "confirmation-required" {
		t.Errorf("Bob → Grandma (owed nothing) = %d %s; want 422 confirmation-required", status, slug)
	}
}

func TestRecordValidatesTheInput(t *testing.T) {
	tr := owed(t)
	outsider := "0190b6c4-0000-7000-8000-00000000abcd"
	for _, tc := range []struct {
		name string
		in   map[string]any
		want string
	}{
		{"same member", tr.settlementInput(tr.bobID, tr.bobID, 10), "/to_member_id same_member"},
		{"payer not a member", tr.settlementInput(outsider, tr.aliceID, 10), "/from_member_id not_a_member"},
		{"zero", tr.settlementInput(tr.bobID, tr.aliceID, 0), "/amount/minor not_positive"},
		{"other currency", func() map[string]any {
			in := tr.settlementInput(tr.bobID, tr.aliceID, 10)
			in["amount"] = map[string]any{"minor": 10, "currency": "EUR"}
			return in
		}(), "/amount/currency not_group_currency"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp := write(t, tr.srv, http.MethodPost, "/v1/groups/"+tr.group.ID+"/settlements", tr.bob.AccessToken, tc.in)
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

// --- Withdraw (FR-S3, FR-E6) ---

func withdraw(t *testing.T, tr trip, token, id string, version int) (settlement, int, string) {
	t.Helper()
	resp := write(t, tr.srv, http.MethodPost, "/v1/settlements/"+id+"/withdraw", token, fmt.Sprintf(`{"version":%d}`, version))
	var s settlement
	if resp.StatusCode == http.StatusOK {
		resp.JSON(t, &s)
	}
	return s, resp.StatusCode, resp.ProblemType(t)
}

func TestTheCreatorWithdrawsASettlementAndItStopsCounting(t *testing.T) {
	tr := owed(t)
	s, _, _ := tr.settle(t, tr.bob.AccessToken, tr.settlementInput(tr.bobID, tr.aliceID, 300))

	w, status, _ := withdraw(t, tr, tr.bob.AccessToken, s.ID, s.Version)

	if status != http.StatusOK || w.State != "withdrawn" || w.Version != 2 {
		t.Fatalf("withdraw = %d %+v; want 200 withdrawn v2", status, w)
	}
	if b := tr.balances(t, tr.alice.AccessToken); b.of(t, tr.bobID) != -300 {
		t.Errorf("Bob's Balance = %d; want -300 again", b.of(t, tr.bobID))
	}
	var types string
	if err := connect(t, tr.srv).QueryRow(context.Background(),
		"SELECT string_agg(type, ',' ORDER BY id) FROM activity_events WHERE subject_id = $1", s.ID).Scan(&types); err != nil ||
		types != "settlement_recorded,settlement_withdrawn" {
		t.Errorf("activity events for the Settlement = %q, %v; want settlement_recorded,settlement_withdrawn", types, err)
	}
}

func TestOnlyTheCreatorWithdraws(t *testing.T) {
	tr := owed(t)
	s, _, _ := tr.settle(t, tr.bob.AccessToken, tr.settlementInput(tr.bobID, tr.aliceID, 300))

	if _, status, slug := withdraw(t, tr, tr.alice.AccessToken, s.ID, s.Version); status != http.StatusForbidden || slug != "not-creator" {
		t.Errorf("Alice withdrawing Bob's = %d %s; want 403 not-creator", status, slug)
	}
}

func TestWithdrawingTwiceOrStaleIsAConflict(t *testing.T) {
	tr := owed(t)
	s, _, _ := tr.settle(t, tr.bob.AccessToken, tr.settlementInput(tr.bobID, tr.aliceID, 300))
	withdraw(t, tr, tr.bob.AccessToken, s.ID, s.Version)

	if _, status, slug := withdraw(t, tr, tr.bob.AccessToken, s.ID, 2); status != http.StatusConflict || slug != "invalid-state" {
		t.Errorf("second withdraw = %d %s; want 409 invalid-state", status, slug)
	}
	s2, _, _ := tr.settle(t, tr.bob.AccessToken, tr.settlementInput(tr.bobID, tr.aliceID, 100))
	if _, status, slug := withdraw(t, tr, tr.bob.AccessToken, s2.ID, 7); status != http.StatusConflict || slug != "version-conflict" {
		t.Errorf("stale withdraw = %d %s; want 409 version-conflict", status, slug)
	}
}

func TestSettlementsOfAGroupIAmNotInAreNotFound(t *testing.T) {
	tr := owed(t)
	s, _, _ := tr.settle(t, tr.bob.AccessToken, tr.settlementInput(tr.bobID, tr.aliceID, 100))
	mallory := signUpVerified(t, tr.srv, "mallory@example.com")

	if _, status, _ := tr.settle(t, mallory.AccessToken, tr.settlementInput(tr.bobID, tr.aliceID, 10)); status != http.StatusNotFound {
		t.Errorf("record = %d; want 404", status)
	}
	wantProblem(t, tr.srv.Get(t, "/v1/settlements/"+s.ID, bearer(mallory.AccessToken)...), http.StatusNotFound, "not-found")
	wantProblem(t, tr.srv.Get(t, "/v1/groups/"+tr.group.ID+"/settlements", bearer(mallory.AccessToken)...), http.StatusNotFound, "not-found")
	if _, status, _ := withdraw(t, tr, mallory.AccessToken, s.ID, 1); status != http.StatusNotFound {
		t.Errorf("withdraw = %d; want 404", status)
	}
}

func TestListSettlementsNewestFirst(t *testing.T) {
	tr := owed(t)
	first, _, _ := tr.settle(t, tr.bob.AccessToken, tr.settlementInput(tr.bobID, tr.aliceID, 100))
	second, _, _ := tr.settle(t, tr.bob.AccessToken, tr.settlementInput(tr.grandma, tr.aliceID, 100))

	var page struct {
		Items []settlement `json:"items"`
	}
	tr.srv.Get(t, "/v1/groups/"+tr.group.ID+"/settlements", bearer(tr.alice.AccessToken)...).JSON(t, &page)

	if len(page.Items) != 2 || page.Items[0].ID != second.ID || page.Items[1].ID != first.ID {
		t.Errorf("items = %+v; want second then first", page.Items)
	}
}

func TestARetriedSettlementIsRecordedOnce(t *testing.T) {
	tr := owed(t)
	key := newKey()
	in := tr.settlementInput(tr.bobID, tr.aliceID, 100)
	path := "/v1/groups/" + tr.group.ID + "/settlements"

	writeWithKey(t, tr.srv, http.MethodPost, path, tr.bob.AccessToken, key, in)
	writeWithKey(t, tr.srv, http.MethodPost, path, tr.bob.AccessToken, key, in)

	if b := tr.balances(t, tr.bob.AccessToken); b.of(t, tr.bobID) != -200 {
		t.Errorf("Bob's Balance = %d; want -200 (one Settlement)", b.of(t, tr.bobID))
	}
}
