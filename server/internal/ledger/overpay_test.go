package ledger_test

import (
	"testing"

	"github.com/bikkysamuel/splitsDemo/server/internal/ledger"
)

// FR-S2, Q24: any positive Settlement is allowed, but paying more than the
// payer owes, or more than the receiver is owed, earns a warning.
func TestOverpays(t *testing.T) {
	// Member 1 is owed 500; 2 owes 300; 3 owes 200; 4 is settled.
	balances := []ledger.Balance{{JoinSeq: 1, Amount: 500}, {JoinSeq: 2, Amount: -300}, {JoinSeq: 3, Amount: -200}, {JoinSeq: 4, Amount: 0}}
	for _, tc := range []struct {
		name     string
		from, to int
		amount   int64
		want     bool
	}{
		{"exactly what is owed", 2, 1, 300, false},
		{"part of it", 2, 1, 100, false},
		{"more than the payer owes", 2, 1, 301, true},
		{"more than the receiver is owed", 3, 2, 1, true}, // 2 is owed nothing
		{"a settled payer", 4, 1, 1, true},
		{"the creditor paying", 1, 2, 10, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ledger.Overpays(balances, tc.from, tc.to, tc.amount)
			if err != nil || got != tc.want {
				t.Errorf("Overpays(%d → %d, %d) = %v, %v; want %v", tc.from, tc.to, tc.amount, got, err, tc.want)
			}
		})
	}
}

func TestOverpaysRefusesUnknownMembers(t *testing.T) {
	if _, err := ledger.Overpays([]ledger.Balance{{JoinSeq: 1, Amount: 0}}, 1, 9, 10); err == nil {
		t.Error("Overpays with an unknown Member = nil error; want ErrInconsistentLedger")
	}
}
