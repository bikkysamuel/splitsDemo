package ledger_test

import (
	"math/big"
	"testing"

	"github.com/bikkysamuel/splitsDemo/server/internal/ledger"
)

// Invariants 2 and 4 (doc 06): Balances sum to zero, and each one equals
// FR-B1 worked out independently over the Accepted and WithdrawalPending
// items only.
func FuzzBalancesFollowFRB1(f *testing.F) {
	f.Add(uint64(1), uint8(3), uint8(4))    // a small Group
	f.Add(uint64(7), uint8(1), uint8(9))    // one Member: no Settlements
	f.Add(uint64(42), uint8(49), uint8(99)) // the largest Group, many items
	f.Add(uint64(3), uint8(2), uint8(0))    // no items
	f.Fuzz(func(t *testing.T, seed uint64, n, items uint8) {
		rng := &splitmix{seed}
		members, expenses, settlements := randomLedger(t, rng, 1+int(n)%50, int(items)%100)

		got, err := ledger.Balances(members, expenses, settlements)
		if err != nil {
			t.Fatalf("Balances: %v", err)
		}

		want := make(map[int]*big.Int, len(members))
		for _, seq := range members {
			want[seq] = new(big.Int)
		}
		counts := func(s ledger.ItemState) bool { return s == ledger.Accepted || s == ledger.WithdrawalPending }
		for _, e := range expenses {
			if counts(e.State) {
				want[e.Payer].Add(want[e.Payer], big.NewInt(e.Amount))
				for _, sh := range e.Shares {
					want[sh.JoinSeq].Sub(want[sh.JoinSeq], big.NewInt(sh.Amount))
				}
			}
		}
		for _, s := range settlements {
			if counts(s.State) {
				want[s.From].Add(want[s.From], big.NewInt(s.Amount))
				want[s.To].Sub(want[s.To], big.NewInt(s.Amount))
			}
		}

		sum := new(big.Int)
		for i, b := range got {
			if b.JoinSeq != members[i] {
				t.Fatalf("Balance %d is for join_seq %d; want %d", i, b.JoinSeq, members[i])
			}
			if want[b.JoinSeq].Cmp(big.NewInt(b.Amount)) != 0 {
				t.Fatalf("Balance of join_seq %d = %d; want %s", b.JoinSeq, b.Amount, want[b.JoinSeq])
			}
			sum.Add(sum, big.NewInt(b.Amount))
		}
		if sum.Sign() != 0 {
			t.Fatalf("Σ Balances = %s; want 0", sum)
		}
	})
}

// Invariant 3 (doc 06): applying the Settle-up Suggestions brings every
// Balance to zero in at most (Members − 1) payments, each a positive amount
// from a Member who owes to a Member who is owed.
func FuzzSettleUpSuggestionsClearBalances(f *testing.F) {
	f.Add(uint64(1), uint8(3), uint8(4), false)
	f.Add(uint64(5), uint8(49), uint8(99), false)
	f.Add(uint64(9), uint8(1), uint8(0), true)
	f.Add(uint64(11), uint8(49), uint8(0), true) // arbitrary Balances, largest Group
	f.Add(uint64(13), uint8(7), uint8(0), true)
	f.Fuzz(func(t *testing.T, seed uint64, n, items uint8, arbitrary bool) {
		rng := &splitmix{seed}
		count := 1 + int(n)%50
		var bs []ledger.Balance
		if arbitrary {
			bs = randomBalances(rng, count)
		} else {
			members, expenses, settlements := randomLedger(t, rng, count, int(items)%100)
			var err error
			if bs, err = ledger.Balances(members, expenses, settlements); err != nil {
				t.Fatalf("Balances: %v", err)
			}
		}

		suggestions, err := ledger.SettleUpSuggestions(bs)
		if err != nil {
			t.Fatalf("SettleUpSuggestions(%v): %v", bs, err)
		}

		start := make(map[int]int64, len(bs))
		nonZero := 0
		for _, b := range bs {
			start[b.JoinSeq] = b.Amount
			if b.Amount != 0 {
				nonZero++
			}
		}
		// nonZero ≤ Members, so this is at least as strict as Members − 1.
		if len(suggestions) > max(0, nonZero-1) {
			t.Fatalf("%d suggestions for %d Members, %d with a non-zero Balance", len(suggestions), len(bs), nonZero)
		}
		rest := make(map[int]int64, len(bs))
		for seq, amount := range start {
			rest[seq] = amount
		}
		for _, s := range suggestions {
			if s.Amount <= 0 || s.From == s.To {
				t.Fatalf("suggestion %+v is not a positive payment between two Members", s)
			}
			if start[s.From] >= 0 || start[s.To] <= 0 {
				t.Fatalf("suggestion %+v: payer's Balance %d, payee's %d", s, start[s.From], start[s.To])
			}
			rest[s.From] += s.Amount
			rest[s.To] -= s.Amount
		}
		for seq, amount := range rest {
			if amount != 0 {
				t.Fatalf("after the suggestions join_seq %d still has %d", seq, amount)
			}
		}
	})
}

// Invariant 4 (doc 06): items in an excluded state (Pending, Disputed,
// Withdrawn) never change Balances, and neither does an Accepted item
// moving to WithdrawalPending.
func FuzzExcludedStatesNeverChangeBalances(f *testing.F) {
	f.Add(uint64(1), uint8(3), uint8(4), uint8(5))
	f.Add(uint64(2), uint8(1), uint8(0), uint8(9))
	f.Add(uint64(3), uint8(49), uint8(99), uint8(99))
	f.Add(uint64(4), uint8(2), uint8(10), uint8(0))
	f.Fuzz(func(t *testing.T, seed uint64, n, items, extra uint8) {
		rng := &splitmix{seed}
		count := 1 + int(n)%50
		members, expenses, settlements := randomLedger(t, rng, count, int(items)%100)
		before, err := ledger.Balances(members, expenses, settlements)
		if err != nil {
			t.Fatalf("Balances: %v", err)
		}

		excluded := []ledger.ItemState{ledger.Pending, ledger.Disputed, ledger.Withdrawn}
		var moreExpenses []ledger.Expense
		var moreSettlements []ledger.Settlement
		for _, e := range expenses {
			e.State = flipCounted(rng, e.State)
			moreExpenses = append(moreExpenses, e)
		}
		for _, s := range settlements {
			s.State = flipCounted(rng, s.State)
			moreSettlements = append(moreSettlements, s)
		}
		_, addE, addS := randomLedger(t, rng, count, int(extra)%100)
		for _, e := range addE {
			e.State = excluded[rng.next()%3]
			moreExpenses = insertAt(rng, moreExpenses, e)
		}
		for _, s := range addS {
			s.State = excluded[rng.next()%3]
			moreSettlements = insertAt(rng, moreSettlements, s)
		}

		after, err := ledger.Balances(members, moreExpenses, moreSettlements)
		if err != nil {
			t.Fatalf("Balances with excluded items: %v", err)
		}
		for i := range before {
			if before[i] != after[i] {
				t.Fatalf("Balance of join_seq %d went from %d to %d", before[i].JoinSeq, before[i].Amount, after[i].Amount)
			}
		}
	})
}

// randomLedger builds a Group of count Members (shuffled join_seq values)
// with items Expenses and Settlements in random states. Expenses split
// equally among a random subset of Members through ledger.Shares.
func randomLedger(t *testing.T, rng *splitmix, count, items int) ([]int, []ledger.Expense, []ledger.Settlement) {
	t.Helper()
	members := rng.perm(count)
	for i := range members {
		members[i] *= 3 // join_seq values need not be contiguous
	}
	states := []ledger.ItemState{ledger.Pending, ledger.Accepted, ledger.Disputed, ledger.WithdrawalPending, ledger.Withdrawn}
	var expenses []ledger.Expense
	var settlements []ledger.Settlement
	for range items {
		state := states[rng.next()%uint64(len(states))]
		amount := 1 + int64(rng.next()%1_000_000_000)
		if count == 1 || rng.next()%3 != 0 {
			var split []ledger.SplitMember
			for _, seq := range members {
				if rng.next()%2 == 0 {
					split = append(split, ledger.SplitMember{JoinSeq: seq})
				}
			}
			if len(split) == 0 {
				split = append(split, ledger.SplitMember{JoinSeq: members[0]})
			}
			shares, err := ledger.Shares(amount, ledger.Equal, split)
			if err != nil {
				t.Fatalf("Shares: %v", err)
			}
			e := ledger.Expense{State: state, Payer: members[rng.next()%uint64(count)], Amount: amount}
			for i, m := range split {
				e.Shares = append(e.Shares, ledger.Share{JoinSeq: m.JoinSeq, Amount: shares[i]})
			}
			expenses = append(expenses, e)
		} else {
			from := rng.next() % uint64(count)
			to := (from + 1 + rng.next()%uint64(count-1)) % uint64(count)
			settlements = append(settlements, ledger.Settlement{
				State: state, From: members[from], To: members[to], Amount: amount,
			})
		}
	}
	return members, expenses, settlements
}

// randomBalances returns count Balances summing to zero, many of them zero
// or equal, so ties are common.
func randomBalances(rng *splitmix, count int) []ledger.Balance {
	bs := make([]ledger.Balance, count)
	var sum int64
	for i, seq := range rng.perm(count) {
		bs[i].JoinSeq = seq
		if i == count-1 {
			bs[i].Amount = -sum
			break
		}
		switch rng.next() % 4 {
		case 0: // zero
		case 1:
			bs[i].Amount = 100
		case 2:
			bs[i].Amount = -100
		default:
			bs[i].Amount = int64(rng.next()%2_000_000_000_000_000) - 1_000_000_000_000_000
		}
		sum += bs[i].Amount
	}
	return bs
}

func flipCounted(rng *splitmix, s ledger.ItemState) ledger.ItemState {
	if rng.next()%2 == 0 {
		switch s {
		case ledger.Accepted:
			return ledger.WithdrawalPending
		case ledger.WithdrawalPending:
			return ledger.Accepted
		}
	}
	return s
}

func insertAt[T any](rng *splitmix, items []T, item T) []T {
	i := int(rng.next() % uint64(len(items)+1))
	items = append(items, item)
	copy(items[i+1:], items[i:])
	items[i] = item
	return items
}
