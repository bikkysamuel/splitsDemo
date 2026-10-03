package httpapi_test

import (
	"net/http"
	"net/url"
	"slices"
	"testing"
)

// --- Filters and pages (FR-E8) ---

// listIDs returns the IDs of every Expense the query lists, following
// next_cursor page by page with the same filters.
func (tr trip) listIDs(t *testing.T, query url.Values) []string {
	t.Helper()
	var ids []string
	for range 100 {
		resp := tr.srv.Get(t, "/v1/groups/"+tr.group.ID+"/expenses?"+query.Encode(), bearer(tr.bob.AccessToken)...)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("list ?%s = %d; want 200\n%s", query.Encode(), resp.StatusCode, resp.Body)
		}
		var page expensePage
		resp.JSON(t, &page)
		for _, e := range page.Items {
			ids = append(ids, e.ID)
		}
		if page.NextCursor == nil {
			return ids
		}
		query = cloneValues(query)
		query.Set("cursor", *page.NextCursor)
	}
	t.Fatal("more than 100 pages")
	return nil
}

func cloneValues(v url.Values) url.Values {
	out := url.Values{}
	for k, vs := range v {
		out[k] = slices.Clone(vs)
	}
	return out
}

// dated records an Expense on the given day, paid by payer and shared by
// members.
func (tr trip) dated(t *testing.T, day, category, payer string, members ...string) string {
	t.Helper()
	in := tr.input(300, payer, members...)
	in["spent_on"] = day
	in["category"] = category
	return tr.mustCreate(t, in).ID
}

func TestExpensesFilterByMemberPayingOrSharing(t *testing.T) {
	tr := newTrip(t)
	paidByGrandma := tr.dated(t, "2026-10-01", "food_drink", tr.grandma, tr.aliceID, tr.bobID)
	sharedByGrandma := tr.dated(t, "2026-10-02", "food_drink", tr.aliceID, tr.aliceID, tr.grandma)
	tr.dated(t, "2026-10-03", "food_drink", tr.aliceID, tr.aliceID, tr.bobID)

	got := tr.listIDs(t, url.Values{"member": {tr.grandma}})

	if want := []string{sharedByGrandma, paidByGrandma}; !slices.Equal(got, want) {
		t.Errorf("member=Grandma lists %v; want %v", got, want)
	}
}

func TestExpensesFilterByCategory(t *testing.T) {
	tr := newTrip(t)
	tr.dated(t, "2026-10-01", "food_drink", tr.aliceID, tr.everyone()...)
	taxi := tr.dated(t, "2026-10-02", "transport", tr.aliceID, tr.everyone()...)

	if got := tr.listIDs(t, url.Values{"category": {"transport"}}); !slices.Equal(got, []string{taxi}) {
		t.Errorf("category=transport lists %v; want [%s]", got, taxi)
	}
}

func TestExpensesFilterByAnInclusiveDateRange(t *testing.T) {
	tr := newTrip(t)
	tr.dated(t, "2026-09-30", "food_drink", tr.aliceID, tr.everyone()...)
	first := tr.dated(t, "2026-10-01", "food_drink", tr.aliceID, tr.everyone()...)
	last := tr.dated(t, "2026-10-03", "food_drink", tr.aliceID, tr.everyone()...)
	tr.dated(t, "2026-10-04", "food_drink", tr.aliceID, tr.everyone()...)

	got := tr.listIDs(t, url.Values{"from": {"2026-10-01"}, "to": {"2026-10-03"}})

	if want := []string{last, first}; !slices.Equal(got, want) {
		t.Errorf("1–3 October lists %v; want %v", got, want)
	}
	if got := tr.listIDs(t, url.Values{"from": {"2026-10-04"}}); len(got) != 1 {
		t.Errorf("from 4 October lists %v; want one", got)
	}
	if got := tr.listIDs(t, url.Values{"to": {"2026-09-30"}}); len(got) != 1 {
		t.Errorf("to 30 September lists %v; want one", got)
	}
}

func TestExpensesFilterByState(t *testing.T) {
	tr := newTrip(t)
	e := tr.mustCreate(t, tr.input(900, tr.aliceID, tr.everyone()...))
	kept := tr.dated(t, "2026-10-02", "food_drink", tr.aliceID, tr.everyone()...)
	tr.withdrawExpense(t, tr.alice.AccessToken, e.ID, e.Version)

	if got := tr.listIDs(t, url.Values{"state": {"withdrawn"}}); !slices.Equal(got, []string{e.ID}) {
		t.Errorf("state=withdrawn lists %v; want [%s]", got, e.ID)
	}
	if got := tr.listIDs(t, url.Values{"state": {"accepted"}}); !slices.Equal(got, []string{kept}) {
		t.Errorf("state=accepted lists %v; want [%s]", got, kept)
	}
}

func TestFiltersCombine(t *testing.T) {
	tr := newTrip(t)
	tr.dated(t, "2026-10-01", "transport", tr.aliceID, tr.aliceID, tr.bobID)
	want := tr.dated(t, "2026-10-02", "transport", tr.aliceID, tr.grandma)
	tr.dated(t, "2026-10-02", "food_drink", tr.aliceID, tr.grandma)

	got := tr.listIDs(t, url.Values{"member": {tr.grandma}, "category": {"transport"}, "from": {"2026-10-02"}})

	if !slices.Equal(got, []string{want}) {
		t.Errorf("combined filters list %v; want [%s]", got, want)
	}
}

// Pages of a filtered list are stable: every match exactly once, newest
// first, even with several Expenses on one day and pages of two.
func TestFilteredPagesListEveryMatchOnceInOrder(t *testing.T) {
	tr := newTrip(t)
	var want []string
	for _, day := range []string{"2026-10-01", "2026-10-01", "2026-10-01", "2026-10-02", "2026-10-02"} {
		want = append(want, tr.dated(t, day, "groceries", tr.aliceID, tr.everyone()...))
		tr.dated(t, day, "food_drink", tr.aliceID, tr.everyone()...)
	}
	// Newest first: 2 October (latest recorded first), then 1 October.
	want = []string{want[4], want[3], want[2], want[1], want[0]}

	got := tr.listIDs(t, url.Values{"category": {"groceries"}, "limit": {"2"}})

	if !slices.Equal(got, want) {
		t.Errorf("paged groceries = %v; want %v", got, want)
	}
}

func TestBadFiltersAreRefused(t *testing.T) {
	tr := newTrip(t)
	for _, query := range []string{
		"from=2026-10-02&to=2026-10-01", "category=caviar", "state=lost", "member=not-a-uuid", "from=yesterday",
	} {
		resp := tr.srv.Get(t, "/v1/groups/"+tr.group.ID+"/expenses?"+query, bearer(tr.bob.AccessToken)...)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("?%s = %d; want 400\n%s", query, resp.StatusCode, resp.Body)
		}
	}
}

// Authorization comes before the filter: outside the Group even a bad
// filter is not-found.
func TestFiltersOfAGroupIAmNotInAreNotFound(t *testing.T) {
	tr := newTrip(t)
	mallory := signUpVerified(t, tr.srv, "mallory@example.com")

	resp := tr.srv.Get(t, "/v1/groups/"+tr.group.ID+"/expenses?state=lost", bearer(mallory.AccessToken)...)

	wantProblem(t, resp, http.StatusNotFound, "not-found")
}
