// Package balances derives a Group's Balances and Settle-up Suggestions on
// read (FR-B1, FR-B2). All the arithmetic is ledger's (ADR-0006); this
// package loads the items, authorizes, and names the Members.
package balances

import (
	"context"
	"errors"
	"fmt"

	"github.com/bikkysamuel/splitsDemo/server/internal/groups"
	"github.com/bikkysamuel/splitsDemo/server/internal/ledger"
	"github.com/bikkysamuel/splitsDemo/server/internal/platform"
)

// Items are everything Balances are computed from: every Member's
// join_seq (Former Members included), every Expense with its Shares, every
// Settlement. ledger decides which states count (ADR-0009).
type Items struct {
	Members     []int
	Expenses    []ledger.Expense
	Settlements []ledger.Settlement
}

// Repository loads a Group's ledger items.
type Repository interface {
	// Items returns the Group's items in one consistent read.
	Items(ctx context.Context, groupID platform.ID) (Items, error)
}

// GroupReader returns a Group the User is an active Member of, or
// groups.ErrNotFound. groups.Service implements it.
type GroupReader interface {
	Get(ctx context.Context, userID, groupID platform.ID) (groups.Group, error)
}

// Balance is one Member's Balance in minor units of the Group Currency;
// positive means they are owed money.
type Balance struct {
	MemberID platform.ID
	Amount   int64
}

// Suggestion is a Settle-up Suggestion: From pays To Amount.
type Suggestion struct {
	From   platform.ID
	To     platform.ID
	Amount int64
}

// Result is a Group's Balances, in joining order, and Suggestions.
type Result struct {
	Currency    string
	Balances    []Balance
	Suggestions []Suggestion
}

// Service computes Balances.
type Service struct {
	repo   Repository
	groups GroupReader
}

// NewService returns a Service.
func NewService(repo Repository, groups GroupReader) *Service {
	return &Service{repo: repo, groups: groups}
}

// ErrInconsistent wraps ledger.ErrInconsistentLedger: stored data breaks
// an invariant, a bug, never user input.
var ErrInconsistent = errors.New("balances: inconsistent ledger")

// Group returns the Balances and Suggestions of a Group the User is an
// active Member of (groups.ErrNotFound otherwise).
func (s *Service) Group(ctx context.Context, userID, groupID platform.ID) (Result, error) {
	g, err := s.groups.Get(ctx, userID, groupID)
	if err != nil {
		return Result{}, err
	}
	items, err := s.repo.Items(ctx, groupID)
	if err != nil {
		return Result{}, fmt.Errorf("balances: load items of group %s: %w", groupID, err)
	}
	balances, err := ledger.Balances(items.Members, items.Expenses, items.Settlements)
	if err != nil {
		return Result{}, fmt.Errorf("%w: group %s: %w", ErrInconsistent, groupID, err)
	}
	suggestions, err := ledger.SettleUpSuggestions(balances)
	if err != nil {
		return Result{}, fmt.Errorf("%w: group %s: %w", ErrInconsistent, groupID, err)
	}
	byJoinSeq := make(map[int]platform.ID, len(g.Members))
	for _, m := range g.Members {
		byJoinSeq[m.JoinSeq] = m.ID
	}
	r := Result{Currency: g.Currency, Balances: make([]Balance, len(balances)), Suggestions: make([]Suggestion, len(suggestions))}
	for i, b := range balances {
		r.Balances[i] = Balance{MemberID: byJoinSeq[b.JoinSeq], Amount: b.Amount}
	}
	for i, sg := range suggestions {
		r.Suggestions[i] = Suggestion{From: byJoinSeq[sg.From], To: byJoinSeq[sg.To], Amount: sg.Amount}
	}
	return r, nil
}
