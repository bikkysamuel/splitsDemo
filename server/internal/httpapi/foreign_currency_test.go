package httpapi_test

import (
	"net/http"
	"strings"
	"testing"
)

// abroad is an equal Split among everyone of an Expense paid in another
// currency at the given Exchange Rate ("" for none).
func (tr trip) abroad(minor int64, currency, rate string) map[string]any {
	in := tr.input(minor, tr.aliceID, tr.everyone()...)
	in["amount"] = map[string]any{"minor": minor, "currency": currency}
	if rate != "" {
		in["exchange_rate"] = rate
	}
	return in
}

type expensePreview struct {
	Amount   money       `json:"amount"`
	Original money       `json:"original_amount"`
	Rate     *string     `json:"exchange_rate"`
	Shares   []shareLine `json:"shares"`
}

// ADR-0007, ADR-0010: US$10.50 at 83.25 is ₹874.125, rounded half-up once
// to ₹874.13; the Shares divide the converted amount (87413 paise among
// three: 29137 each and the two leftover paise by join order).
func TestForeignExpenseConvertsHalfUpOnceAndSplitsTheConvertedAmount(t *testing.T) {
	tr := newTrip(t)
	in := tr.abroad(1050, "USD", "83.25")

	resp := tr.preview(t, tr.alice.AccessToken, in)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("preview = %d; want 200\n%s", resp.StatusCode, resp.Body)
	}
	var p expensePreview
	resp.JSON(t, &p)
	if p.Amount != (money{87413, "INR"}) || p.Original != (money{1050, "USD"}) || p.Rate == nil || *p.Rate != "83.25" {
		t.Errorf("preview = %+v (rate %v); want ₹874.13 from US$10.50 at 83.25", p, p.Rate)
	}
	if got := minors(p.Shares); len(got) != 3 || got[0] != 29138 || got[1] != 29138 || got[2] != 29137 || p.Shares[0].Share.Currency != "INR" {
		t.Errorf("shares = %+v; want 29138 29138 29137 INR", p.Shares)
	}

	e := tr.mustCreate(t, in)
	var stored expense
	tr.srv.Get(t, "/v1/expenses/"+e.ID, bearer(tr.bob.AccessToken)...).JSON(t, &stored)
	for name, got := range map[string]expense{"created": e, "stored": stored} {
		if got.Amount != (money{87413, "INR"}) || got.Original != (money{1050, "USD"}) || got.Rate == nil || *got.Rate != "83.25" {
			t.Errorf("%s = %+v; want ₹874.13 from US$10.50 at 83.25", name, got)
		}
	}
	// Balances are in the Group Currency: Alice paid 87413 and owes 29138.
	b := tr.balances(t, tr.alice.AccessToken)
	for _, x := range b.Balances {
		if x.MemberID == tr.aliceID && x.Balance != (money{87413 - 29138, "INR"}) {
			t.Errorf("Alice's balance = %+v; want 58275 INR", x.Balance)
		}
	}
}

func TestExpenseInTheGroupCurrencyHasNoRate(t *testing.T) {
	tr := newTrip(t)
	e := tr.mustCreate(t, tr.input(1000, tr.aliceID, tr.everyone()...))

	var stored expense
	tr.srv.Get(t, "/v1/expenses/"+e.ID, bearer(tr.alice.AccessToken)...).JSON(t, &stored)
	if stored.Original != (money{1000, "INR"}) || stored.Rate != nil {
		t.Errorf("stored = %+v (rate %v); want original ₹10.00 and no rate", stored, stored.Rate)
	}
}

// The rate is kept exactly as entered: its scale kept, leading zeros
// dropped, never through a float.
func TestExchangeRateRoundTripsExactly(t *testing.T) {
	tr := newTrip(t)
	for entered, want := range map[string]string{"83.20": "83.20", "083.5": "83.5", "0.55": "0.55", "1": "1"} {
		e := tr.mustCreate(t, tr.abroad(1000, "JPY", entered))

		var stored expense
		tr.srv.Get(t, "/v1/expenses/"+e.ID, bearer(tr.alice.AccessToken)...).JSON(t, &stored)
		for name, got := range map[string]*string{"created": e.Rate, "stored": stored.Rate} {
			if got == nil || *got != want {
				t.Errorf("%s rate for %q = %v; want %s", name, entered, got, want)
			}
		}
	}
}

func TestExchangeRateIsCheckedAgainstTheCurrencies(t *testing.T) {
	tr := newTrip(t)

	tr.wantSplitError(t, tr.abroad(1000, "USD", ""), "/exchange_rate required")
	tr.wantSplitError(t, tr.abroad(1000, "INR", "1"), "/exchange_rate not_allowed")
	for _, bad := range []string{"83.255", "0", "0.00", "-83", "8e1", "83.", ".5", " 83", strings.Repeat("1", 21)} {
		tr.wantSplitError(t, tr.abroad(1000, "USD", bad), "/exchange_rate invalid")
	}
	tr.wantSplitError(t, tr.abroad(1000, "XTS", "1"), "/amount/currency invalid")
	// US$0.01 at 0.49 is 0.49 paise: nothing to share.
	tr.wantSplitError(t, tr.abroad(1, "USD", "0.49"), "/amount/minor converts_to_zero")
	tr.wantSplitError(t, tr.abroad(9_000_000_000_000_000_000, "JPY", "99"), "/amount/minor too_large")
}

// Q97: exact entries are minor units of the Group Currency, summing to the
// converted amount.
func TestExactSplitOfAForeignExpenseSumsToTheConvertedAmount(t *testing.T) {
	tr := newTrip(t)
	in := tr.abroad(1050, "USD", "83.25")
	in["split"] = tr.split(0, "exact", tr.aliceID, "50000", tr.bobID, "37413")["split"]

	if got := minors(tr.previewShares(t, in)); len(got) != 2 || got[0] != 50000 || got[1] != 37413 {
		t.Errorf("shares = %v; want 50000 37413", got)
	}
	in["split"] = map[string]any{"method": "exact", "members": []map[string]string{
		{"member_id": tr.aliceID, "input": "525"}, {"member_id": tr.bobID, "input": "525"},
	}}
	tr.wantSplitError(t, in, "/split exact_sum_mismatch")
}
