package httpapi_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/bikkysamuel/splitsDemo/server/internal/apptest"
)

// --- Edit and withdraw (FR-E6) ---

// edited is the body of an edit: the Expense as edited, and the version it
// changes.
func edited(in map[string]any, version int) map[string]any {
	in["version"] = version
	return in
}

func (tr trip) edit(t *testing.T, token, id string, body any) (expense, int, string) {
	t.Helper()
	resp := write(t, tr.srv, http.MethodPatch, "/v1/expenses/"+id, token, body)
	var e expense
	if resp.StatusCode == http.StatusOK {
		resp.JSON(t, &e)
	}
	return e, resp.StatusCode, resp.ProblemType(t)
}

func (tr trip) withdrawExpense(t *testing.T, token, id string, version int) (expense, int, string) {
	t.Helper()
	resp := write(t, tr.srv, http.MethodPost, "/v1/expenses/"+id+"/withdraw", token, fmt.Sprintf(`{"version":%d}`, version))
	var e expense
	if resp.StatusCode == http.StatusOK {
		resp.JSON(t, &e)
	}
	return e, resp.StatusCode, resp.ProblemType(t)
}

type activityEvent struct {
	Type    string
	Actor   string
	Payload map[string]any
}

// events returns the Activity History events about an Expense, oldest
// first.
func (tr trip) events(t *testing.T, subjectID string) []activityEvent {
	t.Helper()
	rows, err := connect(t, tr.srv).Query(context.Background(),
		"SELECT type, actor_member_id::text, payload FROM activity_events WHERE subject_id = $1 ORDER BY id", subjectID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []activityEvent
	for rows.Next() {
		var e activityEvent
		var raw []byte
		if err := rows.Scan(&e.Type, &e.Actor, &raw); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(raw, &e.Payload); err != nil {
			t.Fatal(err)
		}
		out = append(out, e)
	}
	return out
}

func TestTheCreatorEditsAnExpenseAndTheSharesAreRecomputed(t *testing.T) {
	tr := newTrip(t)
	e := tr.mustCreate(t, tr.input(900, tr.aliceID, tr.everyone()...))
	in := tr.input(1000, tr.aliceID, tr.aliceID, tr.bobID)
	in["category"] = "groceries"

	got, status, slug := tr.edit(t, tr.alice.AccessToken, e.ID, edited(in, e.Version))

	if status != http.StatusOK {
		t.Fatalf("edit = %d %s; want 200", status, slug)
	}
	if got.Amount != (money{1000, "INR"}) || got.Category != "groceries" || got.Revision != 2 || got.Version != 2 || got.State != "accepted" {
		t.Errorf("edited = %+v; want ₹10 groceries, revision 2, version 2, accepted", got)
	}
	if shares := shareAmounts(got.Shares); len(shares) != 2 || shares[tr.aliceID] != 500 || shares[tr.bobID] != 500 {
		t.Errorf("shares = %+v; want 500 each for Alice and Bob", got.Shares)
	}
	var stored expense
	tr.srv.Get(t, "/v1/expenses/"+e.ID, bearer(tr.bob.AccessToken)...).JSON(t, &stored)
	if stored.Revision != 2 || stored.Amount.Minor != 1000 || len(stored.Shares) != 2 {
		t.Errorf("stored = %+v; want the edit", stored)
	}
	if b := tr.balances(t, tr.alice.AccessToken); b.of(t, tr.grandma) != 0 || b.of(t, tr.bobID) != -500 {
		t.Errorf("Balances = %+v; want Grandma 0, Bob -500", b.Balances)
	}
}

func TestAnEditKeepsTheFieldLevelDiffInTheActivityHistory(t *testing.T) {
	tr := newTrip(t)
	e := tr.mustCreate(t, tr.input(900, tr.aliceID, tr.everyone()...))
	in := tr.input(1200, tr.aliceID, tr.everyone()...)
	in["note"] = "Lunch"

	tr.edit(t, tr.alice.AccessToken, e.ID, edited(in, e.Version))

	events := tr.events(t, e.ID)
	if len(events) != 2 || events[0].Type != "expense_created" || events[1].Type != "expense_edited" {
		t.Fatalf("events = %+v; want expense_created, expense_edited", events)
	}
	edit := events[1]
	if edit.Actor != tr.aliceID || edit.Payload["revision"] != float64(2) {
		t.Errorf("edit event = %+v; want by Alice, revision 2", edit)
	}
	changes, _ := edit.Payload["changes"].(map[string]any)
	want := map[string][2]any{
		"amount_minor":   {float64(900), float64(1200)},
		"original_minor": {float64(900), float64(1200)},
		"note":           {"Dinner", "Lunch"},
	}
	for field, fromTo := range want {
		c, _ := changes[field].(map[string]any)
		if c == nil || c["from"] != fromTo[0] || c["to"] != fromTo[1] {
			t.Errorf("changes[%s] = %v; want from %v to %v", field, changes[field], fromTo[0], fromTo[1])
		}
	}
	if _, ok := changes["shares"]; !ok {
		t.Errorf("changes = %v; want the Shares too", changes)
	}
	for _, unchanged := range []string{"category", "payer_member_id", "spent_on", "split_method", "exchange_rate"} {
		if _, ok := changes[unchanged]; ok {
			t.Errorf("changes[%s] present; want only changed fields", unchanged)
		}
	}
}

func TestSendingWhatIsSavedChangesNothing(t *testing.T) {
	tr := newTrip(t)
	in := tr.input(900, tr.aliceID, tr.everyone()...)
	e := tr.mustCreate(t, in)

	got, status, _ := tr.edit(t, tr.alice.AccessToken, e.ID, edited(tr.input(900, tr.aliceID, tr.everyone()...), e.Version))

	if status != http.StatusOK || got.Revision != 1 || got.Version != 1 {
		t.Errorf("unchanged edit = %d %+v; want 200, revision 1, version 1", status, got)
	}
	if events := tr.events(t, e.ID); len(events) != 1 {
		t.Errorf("events = %+v; want only expense_created", events)
	}
}

func TestAnExchangeRateChangeIsAnEdit(t *testing.T) {
	tr := newTrip(t)
	e := tr.mustCreate(t, tr.abroad(1050, "USD", "83.25"))

	got, status, slug := tr.edit(t, tr.alice.AccessToken, e.ID, edited(tr.abroad(1050, "USD", "84"), e.Version))

	if status != http.StatusOK || got.Rate == nil || *got.Rate != "84" || got.Amount != (money{88200, "INR"}) {
		t.Fatalf("edit = %d %s %+v; want ₹882.00 at 84", status, slug, got)
	}
	changes, _ := tr.events(t, e.ID)[1].Payload["changes"].(map[string]any)
	if c, _ := changes["exchange_rate"].(map[string]any); c == nil || c["from"] != "83.25" || c["to"] != "84" {
		t.Errorf("changes[exchange_rate] = %v; want 83.25 → 84", changes["exchange_rate"])
	}
}

func TestOnlyTheCreatorEditsOrWithdrawsAnExpense(t *testing.T) {
	tr := newTrip(t)
	e := tr.mustCreate(t, tr.input(900, tr.aliceID, tr.everyone()...))

	if _, status, slug := tr.edit(t, tr.bob.AccessToken, e.ID, edited(tr.input(1000, tr.aliceID, tr.everyone()...), e.Version)); status != http.StatusForbidden || slug != "not-creator" {
		t.Errorf("Bob editing Alice's = %d %s; want 403 not-creator", status, slug)
	}
	if _, status, slug := tr.withdrawExpense(t, tr.bob.AccessToken, e.ID, e.Version); status != http.StatusForbidden || slug != "not-creator" {
		t.Errorf("Bob withdrawing Alice's = %d %s; want 403 not-creator", status, slug)
	}
}

func TestEditingAStaleCopyIsAConflict(t *testing.T) {
	tr := newTrip(t)
	e := tr.mustCreate(t, tr.input(900, tr.aliceID, tr.everyone()...))
	tr.edit(t, tr.alice.AccessToken, e.ID, edited(tr.input(1000, tr.aliceID, tr.everyone()...), e.Version))

	if _, status, slug := tr.edit(t, tr.alice.AccessToken, e.ID, edited(tr.input(1100, tr.aliceID, tr.everyone()...), e.Version)); status != http.StatusConflict || slug != "version-conflict" {
		t.Errorf("stale edit = %d %s; want 409 version-conflict", status, slug)
	}
	if _, status, slug := tr.withdrawExpense(t, tr.alice.AccessToken, e.ID, e.Version); status != http.StatusConflict || slug != "version-conflict" {
		t.Errorf("stale withdraw = %d %s; want 409 version-conflict", status, slug)
	}
}

func TestAnEditIsValidatedLikeACreate(t *testing.T) {
	tr := newTrip(t)
	e := tr.mustCreate(t, tr.input(900, tr.aliceID, tr.everyone()...))

	resp := write(t, tr.srv, http.MethodPatch, "/v1/expenses/"+e.ID, tr.alice.AccessToken, edited(tr.abroad(1050, "USD", ""), e.Version))

	wantProblem(t, resp, http.StatusBadRequest, "validation-failed")
	if errs := fieldErrors(t, resp); errs["/exchange_rate"] != "required" {
		t.Errorf("errors = %v; want /exchange_rate required", errs)
	}
}

func TestAWithdrawnExpenseStopsCountingButStays(t *testing.T) {
	tr := newTrip(t)
	e := tr.mustCreate(t, tr.input(900, tr.aliceID, tr.everyone()...))
	tr.mustCreate(t, tr.input(300, tr.aliceID, tr.everyone()...))

	w, status, slug := tr.withdrawExpense(t, tr.alice.AccessToken, e.ID, e.Version)

	if status != http.StatusOK || w.State != "withdrawn" || w.Version != 2 || w.Revision != 1 {
		t.Fatalf("withdraw = %d %s %+v; want 200 withdrawn, version 2, revision 1", status, slug, w)
	}
	if b := tr.balances(t, tr.bob.AccessToken); b.of(t, tr.bobID) != -100 {
		t.Errorf("Bob's Balance = %d; want -100 (only the ₹3 Expense counts)", b.of(t, tr.bobID))
	}
	var page expensePage
	tr.srv.Get(t, "/v1/groups/"+tr.group.ID+"/expenses", bearer(tr.bob.AccessToken)...).JSON(t, &page)
	if len(page.Items) != 2 {
		t.Errorf("list = %+v; want both Expenses, the withdrawn one included", page.Items)
	}
	if events := tr.events(t, e.ID); len(events) != 2 || events[1].Type != "expense_withdrawn" || events[1].Actor != tr.aliceID {
		t.Errorf("events = %+v; want expense_created then expense_withdrawn by Alice", events)
	}
}

func TestAWithdrawnExpenseCanNoLongerChange(t *testing.T) {
	tr := newTrip(t)
	e := tr.mustCreate(t, tr.input(900, tr.aliceID, tr.everyone()...))
	w, _, _ := tr.withdrawExpense(t, tr.alice.AccessToken, e.ID, e.Version)

	if _, status, slug := tr.withdrawExpense(t, tr.alice.AccessToken, e.ID, w.Version); status != http.StatusConflict || slug != "invalid-state" {
		t.Errorf("second withdraw = %d %s; want 409 invalid-state", status, slug)
	}
	if _, status, slug := tr.edit(t, tr.alice.AccessToken, e.ID, edited(tr.input(1000, tr.aliceID, tr.everyone()...), w.Version)); status != http.StatusConflict || slug != "invalid-state" {
		t.Errorf("editing a withdrawn one = %d %s; want 409 invalid-state", status, slug)
	}
}

func TestEditAndWithdrawOfAnExpenseIAmNotInAreNotFound(t *testing.T) {
	tr := newTrip(t)
	e := tr.mustCreate(t, tr.input(900, tr.aliceID, tr.everyone()...))
	mallory := signUpVerified(t, tr.srv, "mallory@example.com")

	if _, status, _ := tr.edit(t, mallory.AccessToken, e.ID, edited(tr.input(1000, tr.aliceID, tr.everyone()...), e.Version)); status != http.StatusNotFound {
		t.Errorf("edit = %d; want 404", status)
	}
	if _, status, _ := tr.withdrawExpense(t, mallory.AccessToken, e.ID, e.Version); status != http.StatusNotFound {
		t.Errorf("withdraw = %d; want 404", status)
	}
}

// FR-G6: a Closed Group is read-only.
func TestAClosedGroupRefusesExpenseEditsAndWithdrawals(t *testing.T) {
	tr := newTrip(t)
	e := tr.mustCreate(t, tr.input(900, tr.aliceID, tr.everyone()...))
	if _, err := connect(t, tr.srv).Exec(context.Background(), "UPDATE groups SET state = 'closed' WHERE id = $1", tr.group.ID); err != nil {
		t.Fatalf("close the Group: %v", err)
	}

	if _, status, slug := tr.edit(t, tr.alice.AccessToken, e.ID, edited(tr.input(1000, tr.aliceID, tr.everyone()...), e.Version)); status != http.StatusConflict || slug != "group-closed" {
		t.Errorf("edit = %d %s; want 409 group-closed", status, slug)
	}
	if _, status, slug := tr.withdrawExpense(t, tr.alice.AccessToken, e.ID, e.Version); status != http.StatusConflict || slug != "group-closed" {
		t.Errorf("withdraw = %d %s; want 409 group-closed", status, slug)
	}
}

// fieldErrors maps each field error's pointer to its code.
func fieldErrors(t *testing.T, resp apptest.Response) map[string]string {
	t.Helper()
	var p struct {
		Errors []struct{ Field, Code string } `json:"errors"`
	}
	resp.JSON(t, &p)
	out := make(map[string]string, len(p.Errors))
	for _, e := range p.Errors {
		out[e.Field] = e.Code
	}
	return out
}
