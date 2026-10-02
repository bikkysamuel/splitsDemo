package httpapi_test

import (
	"context"
	"net/http"
	"testing"
)

type groupBalances struct {
	Balances []struct {
		MemberID string `json:"member_id"`
		Balance  money  `json:"balance"`
	} `json:"balances"`
	Suggestions []struct {
		From   string `json:"from_member_id"`
		To     string `json:"to_member_id"`
		Amount money  `json:"amount"`
	} `json:"suggestions"`
}

func (tr trip) balances(t *testing.T, token string) groupBalances {
	t.Helper()
	resp := tr.srv.Get(t, "/v1/groups/"+tr.group.ID+"/balances", bearer(token)...)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET balances = %d; want 200\n%s", resp.StatusCode, resp.Body)
	}
	var b groupBalances
	resp.JSON(t, &b)
	return b
}

func (b groupBalances) of(memberID string) int64 {
	for _, x := range b.Balances {
		if x.MemberID == memberID {
			return x.Balance.Minor
		}
	}
	return -1 << 62
}

func TestANewGroupIsSettled(t *testing.T) {
	tr := newTrip(t)

	b := tr.balances(t, tr.alice.AccessToken)

	if len(b.Balances) != 3 || len(b.Suggestions) != 0 {
		t.Fatalf("balances = %+v; want 3 zero Balances and no Suggestions", b)
	}
	for _, x := range b.Balances {
		if x.Balance != (money{0, "INR"}) {
			t.Errorf("balance %+v; want 0 INR", x)
		}
	}
}

// FR-B1, FR-B2: Alice paid 1000.01 for all three; Bob and Grandma owe her.
func TestBalancesFollowTheExpensesAndSumToZero(t *testing.T) {
	tr := newTrip(t)
	tr.mustCreate(t, tr.input(100001, tr.aliceID, tr.everyone()...))
	tr.mustCreate(t, tr.input(3000, tr.bobID, tr.bobID, tr.grandma))

	b := tr.balances(t, tr.bob.AccessToken)

	// Shares of 100001: 33334, 33334, 33333. Shares of 3000: 1500, 1500.
	want := map[string]int64{tr.aliceID: 100001 - 33334, tr.bobID: 3000 - 33334 - 1500, tr.grandma: -33333 - 1500}
	var sum int64
	for _, x := range b.Balances {
		if x.Balance.Minor != want[x.MemberID] || x.Balance.Currency != "INR" {
			t.Errorf("balance of %s = %+v; want %d INR", x.MemberID, x.Balance, want[x.MemberID])
		}
		sum += x.Balance.Minor
	}
	if sum != 0 {
		t.Errorf("Σ Balances = %d; want 0", sum)
	}
	// Bob owes 31834, Grandma 34833: Grandma (owes most) pays Alice first.
	if len(b.Suggestions) != 2 || b.Suggestions[0].From != tr.grandma || b.Suggestions[0].To != tr.aliceID ||
		b.Suggestions[0].Amount.Minor != 34833 || b.Suggestions[1].From != tr.bobID || b.Suggestions[1].Amount.Minor != 31834 {
		t.Errorf("suggestions = %+v; want Grandma→Alice 34833, Bob→Alice 31834", b.Suggestions)
	}
}

// ADR-0009: only Accepted (and WithdrawalPending) items count. Withdrawing
// comes with #20, so the state is set in the database here.
func TestWithdrawnAndPendingExpensesDoNotCount(t *testing.T) {
	tr := newTrip(t)
	counted := tr.mustCreate(t, tr.input(900, tr.aliceID, tr.everyone()...))
	withdrawn := tr.mustCreate(t, tr.input(5000, tr.bobID, tr.everyone()...))
	pending := tr.mustCreate(t, tr.input(7000, tr.grandma, tr.everyone()...))
	stillCounts := tr.mustCreate(t, tr.input(300, tr.aliceID, tr.aliceID, tr.bobID, tr.grandma))
	conn := connect(t, tr.srv)
	for id, state := range map[string]string{withdrawn.ID: "withdrawn", pending.ID: "pending", stillCounts.ID: "withdrawal_pending"} {
		if _, err := conn.Exec(context.Background(), "UPDATE expenses SET state = $1 WHERE id = $2", state, id); err != nil {
			t.Fatalf("set state: %v", err)
		}
	}
	_ = counted

	b := tr.balances(t, tr.alice.AccessToken)

	// 900 → 300 each; 300 → 100 each; both paid by Alice.
	if b.of(tr.aliceID) != 1200-400 || b.of(tr.bobID) != -400 || b.of(tr.grandma) != -400 {
		t.Errorf("balances = %+v; want Alice 800, Bob -400, Grandma -400", b.Balances)
	}
}

func TestBalancesOfAGroupIAmNotInAreNotFound(t *testing.T) {
	tr := newTrip(t)
	mallory := signUpVerified(t, tr.srv, "mallory@example.com")

	wantProblem(t, tr.srv.Get(t, "/v1/groups/"+tr.group.ID+"/balances", bearer(mallory.AccessToken)...),
		http.StatusNotFound, "not-found")
}
