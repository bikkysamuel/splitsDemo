package httpapi_test

import (
	"net/http"
	"strings"
	"testing"
)

// split builds an input with one entry per Member: member ID, then input
// ("" for none).
func (tr trip) split(minor int64, method string, entries ...string) map[string]any {
	in := tr.input(minor, tr.aliceID)
	members := []map[string]string{}
	for i := 0; i < len(entries); i += 2 {
		m := map[string]string{"member_id": entries[i]}
		if entries[i+1] != "" {
			m["input"] = entries[i+1]
		}
		members = append(members, m)
	}
	in["split"] = map[string]any{"method": method, "members": members}
	return in
}

func (tr trip) previewShares(t *testing.T, in map[string]any) []shareLine {
	t.Helper()
	resp := tr.preview(t, tr.alice.AccessToken, in)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("preview = %d; want 200\n%s", resp.StatusCode, resp.Body)
	}
	var p struct {
		Shares []shareLine `json:"shares"`
	}
	resp.JSON(t, &p)
	return p.Shares
}

func minors(lines []shareLine) []int64 {
	var out []int64
	for _, l := range lines {
		out = append(out, l.Share.Minor)
	}
	return out
}

func (tr trip) wantSplitError(t *testing.T, in map[string]any, want string) {
	t.Helper()
	resp := tr.preview(t, tr.alice.AccessToken, in)
	wantProblem(t, resp, http.StatusBadRequest, "validation-failed")
	var p struct {
		Errors []struct{ Field, Code string } `json:"errors"`
	}
	resp.JSON(t, &p)
	if len(p.Errors) != 1 || p.Errors[0].Field+" "+p.Errors[0].Code != want {
		t.Errorf("errors = %+v; want %s", p.Errors, want)
	}
	created := tr.create(t, tr.alice.AccessToken, in)
	wantProblem(t, created, http.StatusBadRequest, "validation-failed")
	var c struct {
		Errors []struct{ Field, Code string } `json:"errors"`
	}
	created.JSON(t, &c)
	if len(c.Errors) != 1 || c.Errors[0] != p.Errors[0] {
		t.Errorf("create errors = %+v; want %s like the preview", c.Errors, want)
	}
}

// --- Exact (FR-E2): minor units summing to the total ---

func TestExactSplitUsesTheAmountsAsShares(t *testing.T) {
	tr := newTrip(t)
	in := tr.split(1000, "exact", tr.aliceID, "250", tr.bobID, "700", tr.grandma, "50")

	if got := minors(tr.previewShares(t, in)); len(got) != 3 || got[0] != 250 || got[1] != 700 || got[2] != 50 {
		t.Errorf("shares = %v; want 250 700 50", got)
	}
	e := tr.mustCreate(t, in)
	if e.SplitMethod != "exact" || e.Shares[0].Share.Minor != 250 {
		t.Errorf("saved = %+v", e)
	}
}

func TestExactSplitsMustSumToTheTotal(t *testing.T) {
	tr := newTrip(t)

	tr.wantSplitError(t, tr.split(1000, "exact", tr.aliceID, "250", tr.bobID, "700"), "/split exact_sum_mismatch")
	tr.wantSplitError(t, tr.split(1000, "exact", tr.aliceID, "999.5", tr.bobID, "0.5"), "/split/members/0/input not_whole_minor_units")
}

// --- Percentage: ≤ 2 dp, exactly 100 ---

func TestPercentageSplitFollowsThePercentages(t *testing.T) {
	tr := newTrip(t)
	in := tr.split(10000, "percentage", tr.aliceID, "50", tr.bobID, "33.33", tr.grandma, "16.67")

	if got := minors(tr.previewShares(t, in)); got[0] != 5000 || got[1] != 3333 || got[2] != 1667 {
		t.Errorf("shares = %v; want 5000 3333 1667", got)
	}
	e := tr.mustCreate(t, in)
	var stored expense
	tr.srv.Get(t, "/v1/expenses/"+e.ID, bearer(tr.bob.AccessToken)...).JSON(t, &stored)
	if stored.Shares[1].Input == nil || *stored.Shares[1].Input != "33.33" {
		t.Errorf("stored input = %v; want 33.33 kept exactly", stored.Shares[1].Input)
	}
}

func TestPercentagesMustSumToExactly100WithAtMostTwoDecimals(t *testing.T) {
	tr := newTrip(t)

	tr.wantSplitError(t, tr.split(10000, "percentage", tr.aliceID, "50", tr.bobID, "49.99"), "/split percentages_not_100")
	tr.wantSplitError(t, tr.split(10000, "percentage", tr.aliceID, "50.005", tr.bobID, "49.995"), "/split/members/0/input too_many_decimals")
}

// --- Ratio: positive integers, 2:1:1 ---

func TestRatioSplitDividesByWeight(t *testing.T) {
	tr := newTrip(t)
	in := tr.split(1001, "ratio", tr.aliceID, "2", tr.bobID, "1", tr.grandma, "1")

	// 1001 × 2/4 = 500.5, × 1/4 = 250.25 twice: floors 500, 250, 250 and
	// the leftover 1 to the largest remainder (Alice, .5).
	if got := minors(tr.previewShares(t, in)); got[0] != 501 || got[1] != 250 || got[2] != 250 {
		t.Errorf("shares = %v; want 501 250 250", got)
	}
}

func TestRatioWeightsMustBeWholeNumbers(t *testing.T) {
	tr := newTrip(t)

	tr.wantSplitError(t, tr.split(1000, "ratio", tr.aliceID, "1.5", tr.bobID, "1"), "/split/members/0/input ratio_not_integer")
	tr.wantSplitError(t, tr.split(1000, "ratio", tr.aliceID, "0", tr.bobID, "1"), "/split/members/0/input input_not_positive")
}

// --- Inputs, whatever the method ---

func TestInputsAreCheckedAgainstTheMethod(t *testing.T) {
	tr := newTrip(t)

	tr.wantSplitError(t, tr.split(1000, "ratio", tr.aliceID, "2", tr.bobID, ""), "/split/members/1/input missing_input")
	tr.wantSplitError(t, tr.split(1000, "equal", tr.aliceID, "2", tr.bobID, ""), "/split/members/0/input unexpected_input")
	// At most 30 whole digits and 8 decimals, as the contract says.
	for _, bad := range []string{"1/3", "-2", "1e3", "2.", " 2", "1.123456789", strings.Repeat("1", 31)} {
		tr.wantSplitError(t, tr.split(1000, "ratio", tr.aliceID, bad, tr.bobID, "1"), "/split/members/0/input invalid")
	}
}

// The entry comes back the same from create and from a later read: its
// scale kept ("33.30"), leading zeros dropped.
func TestEntriesReadBackAsSaved(t *testing.T) {
	tr := newTrip(t)
	e := tr.mustCreate(t, tr.split(10000, "percentage", tr.aliceID, "066.70", tr.bobID, "33.30"))

	var stored expense
	tr.srv.Get(t, "/v1/expenses/"+e.ID, bearer(tr.alice.AccessToken)...).JSON(t, &stored)
	for i, want := range []string{"66.70", "33.30"} {
		if got := e.Shares[i].Input; got == nil || *got != want {
			t.Errorf("created input %d = %v; want %s", i, got, want)
		}
		if got := stored.Shares[i].Input; got == nil || *got != want {
			t.Errorf("stored input %d = %v; want %s", i, got, want)
		}
	}
}
