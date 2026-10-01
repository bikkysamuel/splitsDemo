package ledger

import (
	"errors"
	"fmt"
	"math"
)

// ItemState is the state of an Expense or Settlement (doc 06 state
// machines). A confirmed Settlement is Accepted.
type ItemState int

const (
	Pending ItemState = iota + 1
	Accepted
	Disputed
	WithdrawalPending
	Withdrawn
)

// CountsTowardBalances reports whether an item in this state affects
// Balances: only Accepted and WithdrawalPending items do (ADR-0009).
func (s ItemState) CountsTowardBalances() bool {
	return s == Accepted || s == WithdrawalPending
}

func (s ItemState) valid() bool {
	return s >= Pending && s <= Withdrawn
}

// Expense is what Balances need from an Expense: its state, its payer, its
// amount in the Group Currency and the Shares stored for it.
type Expense struct {
	State ItemState
	// Payer is the paying Member's join_seq.
	Payer  int
	Amount int64
	Shares []Share
}

// Share is the portion of an Expense one Member owes, in minor units.
type Share struct {
	JoinSeq int
	Amount  int64
}

// Settlement records that Member From paid Member To back, in the Group
// Currency. From and To are join_seq values.
type Settlement struct {
	State  ItemState
	From   int
	To     int
	Amount int64
}

// Balance is a Member's net position in minor units of the Group Currency.
// Positive means the Member is owed money.
type Balance struct {
	JoinSeq int
	Amount  int64
}

// ErrInconsistentLedger means the stored items break an invariant the
// database should already enforce: an unknown Member or state, Shares that
// don't sum to the amount, a non-positive amount, or totals beyond int64.
// It signals a bug or corrupt data, never bad user input.
var ErrInconsistentLedger = errors.New("inconsistent ledger")

// Balances returns each Member's Balance, in the order of members (their
// join_seq values, every Member of the Group including Former Members), by
// FR-B1: amounts paid − Shares owed + Settlements paid − Settlements
// received. Only Accepted and WithdrawalPending items count (ADR-0009), but
// every item is checked for consistency. The Balances sum to zero.
func Balances(members []int, expenses []Expense, settlements []Settlement) ([]Balance, error) {
	index := make(map[int]int, len(members))
	for i, seq := range members {
		if _, dup := index[seq]; dup {
			return nil, inconsistent("member %d is listed twice", seq)
		}
		index[seq] = i
	}
	isMember := func(seq int) bool { _, ok := index[seq]; return ok }

	for i, e := range expenses {
		if err := checkExpense(e, isMember); err != nil {
			return nil, fmt.Errorf("expense %d: %w", i, err)
		}
	}
	for i, s := range settlements {
		if err := checkSettlement(s, isMember); err != nil {
			return nil, fmt.Errorf("settlement %d: %w", i, err)
		}
	}

	// Every Balance is bounded by the total money that counts, so while
	// that running total fits in int64 no Balance can overflow.
	result := make([]Balance, len(members))
	for i, seq := range members {
		result[i].JoinSeq = seq
	}
	var counted int64
	count := func(amount int64) bool {
		var ok bool
		counted, ok = addChecked(counted, amount)
		return ok
	}
	for _, e := range expenses {
		if !e.State.CountsTowardBalances() {
			continue
		}
		if !count(e.Amount) {
			return nil, inconsistent("counted amounts overflow int64")
		}
		result[index[e.Payer]].Amount += e.Amount
		for _, sh := range e.Shares {
			result[index[sh.JoinSeq]].Amount -= sh.Amount
		}
	}
	for _, s := range settlements {
		if !s.State.CountsTowardBalances() {
			continue
		}
		if !count(s.Amount) {
			return nil, inconsistent("counted amounts overflow int64")
		}
		result[index[s.From]].Amount += s.Amount
		result[index[s.To]].Amount -= s.Amount
	}
	return result, nil
}

func checkExpense(e Expense, isMember func(int) bool) error {
	if !e.State.valid() {
		return inconsistent("unknown state %d", e.State)
	}
	if !isMember(e.Payer) {
		return inconsistent("payer %d is not a member", e.Payer)
	}
	if e.Amount <= 0 {
		return inconsistent("amount %d is not positive", e.Amount)
	}
	seen := make(map[int]bool, len(e.Shares))
	var sum int64
	for _, sh := range e.Shares {
		if !isMember(sh.JoinSeq) {
			return inconsistent("share for %d, who is not a member", sh.JoinSeq)
		}
		if seen[sh.JoinSeq] {
			return inconsistent("two shares for member %d", sh.JoinSeq)
		}
		seen[sh.JoinSeq] = true
		if sh.Amount < 0 {
			return inconsistent("share for member %d is negative", sh.JoinSeq)
		}
		var ok bool
		if sum, ok = addChecked(sum, sh.Amount); !ok {
			return inconsistent("shares overflow int64")
		}
	}
	if sum != e.Amount {
		return inconsistent("shares sum to %d, amount is %d", sum, e.Amount)
	}
	return nil
}

func checkSettlement(s Settlement, isMember func(int) bool) error {
	if !s.State.valid() {
		return inconsistent("unknown state %d", s.State)
	}
	if !isMember(s.From) || !isMember(s.To) {
		return inconsistent("from %d or to %d is not a member", s.From, s.To)
	}
	if s.From == s.To {
		return inconsistent("member %d settles with themselves", s.From)
	}
	if s.Amount <= 0 {
		return inconsistent("amount %d is not positive", s.Amount)
	}
	return nil
}

// SettleUpSuggestion is a proposed payment of Amount minor units from the
// Member with join_seq From to the Member with join_seq To.
type SettleUpSuggestion struct {
	From   int
	To     int
	Amount int64
}

// SettleUpSuggestions returns payments that bring every Balance to zero
// (FR-B2). Greedy: the Member who owes the most pays the Member owed the
// most, ties to the lowest join_seq, until nothing is owed. Each payment
// clears at least one Member and the last clears two, so there are at most
// (Members with a non-zero Balance − 1) payments. The Balances must sum to
// zero.
func SettleUpSuggestions(balances []Balance) ([]SettleUpSuggestion, error) {
	seen := make(map[int]bool, len(balances))
	// owed is the money owed to creditors, debt the money owed by debtors;
	// checking both fit in int64 means every amount below does too.
	var owed, debt int64
	var ok bool
	for _, b := range balances {
		if seen[b.JoinSeq] {
			return nil, inconsistent("member %d is listed twice", b.JoinSeq)
		}
		seen[b.JoinSeq] = true
		switch {
		case b.Amount > 0:
			owed, ok = addChecked(owed, b.Amount)
		case b.Amount == math.MinInt64:
			ok = false
		default:
			debt, ok = addChecked(debt, -b.Amount)
		}
		if !ok {
			return nil, inconsistent("money owed overflows int64")
		}
	}
	if owed != debt {
		return nil, inconsistent("balances sum to %d, not zero", owed-debt)
	}

	rest := make([]Balance, len(balances))
	copy(rest, balances)
	var suggestions []SettleUpSuggestion
	for {
		creditor, debtor := -1, -1
		for i, b := range rest {
			if b.Amount > 0 && (creditor < 0 || pickedFirst(b, b.Amount, rest[creditor], rest[creditor].Amount)) {
				creditor = i
			}
			if b.Amount < 0 && (debtor < 0 || pickedFirst(b, -b.Amount, rest[debtor], -rest[debtor].Amount)) {
				debtor = i
			}
		}
		if creditor < 0 {
			return suggestions, nil // Σ = 0, so no debtor is left either
		}
		amount := min(rest[creditor].Amount, -rest[debtor].Amount)
		suggestions = append(suggestions, SettleUpSuggestion{
			From: rest[debtor].JoinSeq, To: rest[creditor].JoinSeq, Amount: amount,
		})
		rest[creditor].Amount -= amount
		rest[debtor].Amount += amount
	}
}

// pickedFirst reports whether Member a, owing or owed aSize, is matched ahead
// of Member b, owing or owed bSize: the larger amount first, ties to the
// lower join_seq.
func pickedFirst(a Balance, aSize int64, b Balance, bSize int64) bool {
	if aSize != bSize {
		return aSize > bSize
	}
	return a.JoinSeq < b.JoinSeq
}

func addChecked(a, b int64) (int64, bool) {
	s := a + b
	if (b > 0 && s < a) || (b < 0 && s > a) {
		return 0, false
	}
	return s, true
}

func inconsistent(format string, args ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrInconsistentLedger}, args...)...)
}
