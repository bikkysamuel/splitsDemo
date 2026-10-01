package ledger_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/bikkysamuel/splitsDemo/server/internal/ledger"
)

// expense builds an Expense paid by payer, with Shares given as join_seq →
// amount pairs: expense(ledger.Accepted, 1, 300, 1, 100, 2, 100, 3, 100).
func expense(state ledger.ItemState, payer int, amount int64, shares ...int64) ledger.Expense {
	e := ledger.Expense{State: state, Payer: payer, Amount: amount}
	for i := 0; i < len(shares); i += 2 {
		e.Shares = append(e.Shares, ledger.Share{JoinSeq: int(shares[i]), Amount: shares[i+1]})
	}
	return e
}

func settlement(state ledger.ItemState, from, to int, amount int64) ledger.Settlement {
	return ledger.Settlement{State: state, From: from, To: to, Amount: amount}
}

// Expected Balances follow FR-B1: paid − owed + Settlements paid −
// Settlements received, counting only Accepted and WithdrawalPending items.
func TestBalances(t *testing.T) {
	tests := []struct {
		name        string
		members     []int
		expenses    []ledger.Expense
		settlements []ledger.Settlement
		want        []int64
	}{
		{
			name:    "no items: everyone is settled",
			members: []int{1, 2, 3},
			want:    []int64{0, 0, 0},
		},
		{
			name:    "one payer, equal Shares: the others each owe a third",
			members: []int{1, 2, 3},
			expenses: []ledger.Expense{
				expense(ledger.Accepted, 1, 300, 1, 100, 2, 100, 3, 100),
			},
			want: []int64{200, -100, -100},
		},
		{
			name:    "one debtor: Member 3 owes both others",
			members: []int{1, 2, 3},
			expenses: []ledger.Expense{
				expense(ledger.Accepted, 1, 500, 3, 500),
				expense(ledger.Accepted, 2, 250, 3, 250),
			},
			want: []int64{500, 250, -750},
		},
		{
			name:    "a chain: 1 paid for 2, 2 paid for 3, 3 paid for 1",
			members: []int{1, 2, 3},
			expenses: []ledger.Expense{
				expense(ledger.Accepted, 1, 100, 2, 100),
				expense(ledger.Accepted, 2, 100, 3, 100),
				expense(ledger.Accepted, 3, 100, 1, 100),
			},
			want: []int64{0, 0, 0},
		},
		{
			name:    "a confirmed Settlement moves the Balance of both Members",
			members: []int{1, 2},
			expenses: []ledger.Expense{
				expense(ledger.Accepted, 1, 1000, 1, 500, 2, 500),
			},
			settlements: []ledger.Settlement{settlement(ledger.Accepted, 2, 1, 300)},
			want:        []int64{200, -200},
		},
		{
			name:    "all settled after the debtor pays back in full",
			members: []int{1, 2},
			expenses: []ledger.Expense{
				expense(ledger.Accepted, 1, 1000, 1, 500, 2, 500),
			},
			settlements: []ledger.Settlement{settlement(ledger.Accepted, 2, 1, 500)},
			want:        []int64{0, 0},
		},
		{
			name:    "Pending, Disputed and Withdrawn items are ignored",
			members: []int{1, 2},
			expenses: []ledger.Expense{
				expense(ledger.Pending, 1, 1000, 2, 1000),
				expense(ledger.Disputed, 1, 1000, 2, 1000),
				expense(ledger.Withdrawn, 1, 1000, 2, 1000),
			},
			settlements: []ledger.Settlement{
				settlement(ledger.Pending, 2, 1, 70),
				settlement(ledger.Disputed, 2, 1, 70),
				settlement(ledger.Withdrawn, 2, 1, 70),
			},
			want: []int64{0, 0},
		},
		{
			name:    "a WithdrawalPending item still counts until the withdrawal is agreed",
			members: []int{1, 2},
			expenses: []ledger.Expense{
				expense(ledger.WithdrawalPending, 1, 1000, 2, 1000),
			},
			settlements: []ledger.Settlement{settlement(ledger.WithdrawalPending, 2, 1, 400)},
			want:        []int64{600, -600},
		},
		{
			name:    "a Member with no items still gets a zero Balance, in members order",
			members: []int{4, 2, 9},
			expenses: []ledger.Expense{
				expense(ledger.Accepted, 9, 10, 4, 10),
			},
			want: []int64{-10, 0, 10},
		},
		{
			name:    "four Members pay 100, 70, 10 and 20, split equally: 50 each",
			members: []int{1, 2, 3, 4},
			expenses: []ledger.Expense{
				expense(ledger.Accepted, 1, 10000, 1, 2500, 2, 2500, 3, 2500, 4, 2500),
				expense(ledger.Accepted, 2, 7000, 1, 1750, 2, 1750, 3, 1750, 4, 1750),
				expense(ledger.Accepted, 3, 1000, 1, 250, 2, 250, 3, 250, 4, 250),
				expense(ledger.Accepted, 4, 2000, 1, 500, 2, 500, 3, 500, 4, 500),
			},
			want: []int64{5000, 2000, -4000, -3000},
		},
		{
			name:    "a payer outside the Split is owed the whole amount",
			members: []int{1, 2, 3},
			expenses: []ledger.Expense{
				expense(ledger.Accepted, 3, 101, 1, 51, 2, 50),
			},
			want: []int64{-51, -50, 101},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ledger.Balances(tt.members, tt.expenses, tt.settlements)
			if err != nil {
				t.Fatalf("Balances: %v", err)
			}
			amounts := make([]int64, len(got))
			for i, b := range got {
				if b.JoinSeq != tt.members[i] {
					t.Fatalf("Balance %d is for join_seq %d; want %d", i, b.JoinSeq, tt.members[i])
				}
				amounts[i] = b.Amount
			}
			if !slices.Equal(amounts, tt.want) {
				t.Errorf("Balances = %v; want %v", amounts, tt.want)
			}
		})
	}
}

func TestBalancesRejectsInconsistentItems(t *testing.T) {
	tests := []struct {
		name        string
		members     []int
		expenses    []ledger.Expense
		settlements []ledger.Settlement
	}{
		{name: "duplicate Member", members: []int{1, 1}},
		{
			name: "unknown state", members: []int{1, 2},
			expenses: []ledger.Expense{expense(ledger.ItemState(0), 1, 10, 2, 10)},
		},
		{
			name: "payer is not a Member", members: []int{1, 2},
			expenses: []ledger.Expense{expense(ledger.Accepted, 7, 10, 2, 10)},
		},
		{
			name: "Share for someone who is not a Member", members: []int{1, 2},
			expenses: []ledger.Expense{expense(ledger.Accepted, 1, 10, 7, 10)},
		},
		{
			name: "two Shares for one Member", members: []int{1, 2},
			expenses: []ledger.Expense{expense(ledger.Accepted, 1, 10, 2, 5, 2, 5)},
		},
		{
			name: "Shares don't sum to the amount", members: []int{1, 2},
			expenses: []ledger.Expense{expense(ledger.Accepted, 1, 10, 1, 5, 2, 4)},
		},
		{
			name: "negative Share", members: []int{1, 2},
			expenses: []ledger.Expense{expense(ledger.Accepted, 1, 10, 1, 11, 2, -1)},
		},
		{
			name: "amount not positive", members: []int{1, 2},
			expenses: []ledger.Expense{expense(ledger.Accepted, 1, 0, 2, 0)},
		},
		{
			name: "inconsistent even when it doesn't count", members: []int{1, 2},
			expenses: []ledger.Expense{expense(ledger.Pending, 1, 10, 2, 9)},
		},
		{
			name: "Settlement to oneself", members: []int{1, 2},
			settlements: []ledger.Settlement{settlement(ledger.Accepted, 1, 1, 10)},
		},
		{
			name: "Settlement from someone who is not a Member", members: []int{1, 2},
			settlements: []ledger.Settlement{settlement(ledger.Accepted, 7, 1, 10)},
		},
		{
			name: "Settlement amount not positive", members: []int{1, 2},
			settlements: []ledger.Settlement{settlement(ledger.Accepted, 2, 1, -10)},
		},
		{
			name: "Balances overflow int64", members: []int{1, 2},
			expenses: []ledger.Expense{
				expense(ledger.Accepted, 1, 1<<62, 2, 1<<62),
				expense(ledger.Accepted, 1, 1<<62, 2, 1<<62),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ledger.Balances(tt.members, tt.expenses, tt.settlements)
			if !errors.Is(err, ledger.ErrInconsistentLedger) {
				t.Fatalf("Balances error = %v; want ErrInconsistentLedger", err)
			}
		})
	}
}

func TestItemStateCountsTowardBalances(t *testing.T) {
	counts := map[ledger.ItemState]bool{
		ledger.Pending:           false,
		ledger.Accepted:          true,
		ledger.Disputed:          false,
		ledger.WithdrawalPending: true,
		ledger.Withdrawn:         false,
		ledger.ItemState(0):      false,
	}
	for state, want := range counts {
		if got := state.CountsTowardBalances(); got != want {
			t.Errorf("%v.CountsTowardBalances() = %v; want %v", state, got, want)
		}
	}
}

func balances(pairs ...int64) []ledger.Balance {
	var bs []ledger.Balance
	for i := 0; i < len(pairs); i += 2 {
		bs = append(bs, ledger.Balance{JoinSeq: int(pairs[i]), Amount: pairs[i+1]})
	}
	return bs
}

// Expected suggestions follow the greedy rule: the Member who owes the most
// pays the Member owed the most, ties to the lowest join_seq.
func TestSettleUpSuggestions(t *testing.T) {
	s := func(from, to int, amount int64) ledger.SettleUpSuggestion {
		return ledger.SettleUpSuggestion{From: from, To: to, Amount: amount}
	}
	tests := []struct {
		name     string
		balances []ledger.Balance
		want     []ledger.SettleUpSuggestion
	}{
		{name: "no Members", want: nil},
		{name: "all settled", balances: balances(1, 0, 2, 0, 3, 0), want: nil},
		{
			name:     "one debtor pays each creditor, largest first",
			balances: balances(1, 500, 2, 250, 3, -750),
			want:     []ledger.SettleUpSuggestion{s(3, 1, 500), s(3, 2, 250)},
		},
		{
			name:     "one creditor is paid by each debtor, largest first",
			balances: balances(1, -100, 2, -300, 3, 400),
			want:     []ledger.SettleUpSuggestion{s(2, 3, 300), s(1, 3, 100)},
		},
		{
			name:     "a chain collapses to one payment",
			balances: balances(1, 100, 2, 0, 3, -100),
			want:     []ledger.SettleUpSuggestion{s(3, 1, 100)},
		},
		{
			name:     "matching pairs settle directly",
			balances: balances(1, 70, 2, -30, 3, 30, 4, -70),
			want:     []ledger.SettleUpSuggestion{s(4, 1, 70), s(2, 3, 30)},
		},
		{
			name:     "ties go to the lowest join_seq, whatever the input order",
			balances: balances(5, -50, 2, 50, 3, -50, 1, 50),
			want:     []ledger.SettleUpSuggestion{s(3, 1, 50), s(5, 2, 50)},
		},
		{
			name:     "Balances 50, 20, −40, −30 settle in three payments",
			balances: balances(1, 5000, 2, 2000, 3, -4000, 4, -3000),
			want:     []ledger.SettleUpSuggestion{s(3, 1, 4000), s(4, 2, 2000), s(4, 1, 1000)},
		},
		{
			name:     "a partial payment leaves the remainder for the next match",
			balances: balances(1, 600, 2, 400, 3, -700, 4, -300),
			want:     []ledger.SettleUpSuggestion{s(3, 1, 600), s(4, 2, 300), s(3, 2, 100)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ledger.SettleUpSuggestions(tt.balances)
			if err != nil {
				t.Fatalf("SettleUpSuggestions: %v", err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("SettleUpSuggestions = %v; want %v", got, tt.want)
			}
		})
	}
}

func TestSettleUpSuggestionsRejectsInconsistentBalances(t *testing.T) {
	tests := []struct {
		name     string
		balances []ledger.Balance
	}{
		{name: "Balances don't sum to zero", balances: balances(1, 100, 2, -99)},
		{name: "duplicate Member", balances: balances(1, 100, 1, -100)},
		{name: "money owed overflows int64", balances: balances(1, 1<<62, 2, 1<<62, 3, -(1 << 62), 4, -(1 << 62))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ledger.SettleUpSuggestions(tt.balances)
			if !errors.Is(err, ledger.ErrInconsistentLedger) {
				t.Fatalf("SettleUpSuggestions error = %v; want ErrInconsistentLedger", err)
			}
		})
	}
}
