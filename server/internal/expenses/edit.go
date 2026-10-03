package expenses

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/bikkysamuel/splitsDemo/server/internal/groups"
	"github.com/bikkysamuel/splitsDemo/server/internal/platform"
)

// Edit replaces an Expense with in, as its creator's next revision (FR-E6).
// Shares come from the same code path as Create, so an Exchange Rate
// change converts again. Sending what is saved changes nothing. In M1 the
// Expense stays Accepted (D9); with M2 an edit makes it Pending again.
func (s *Service) Edit(ctx context.Context, userID, expenseID platform.ID, version int, in Input) (Expense, error) {
	old, g, err := s.ownExpense(ctx, userID, expenseID)
	if err != nil {
		return Expense{}, err
	}
	if old.State == StateWithdrawn || old.State == StateWithdrawalPending {
		return Expense{}, ErrInvalidState
	}
	if version != old.Version {
		return Expense{}, ErrVersionConflict
	}
	c, note, err := compute(g, in)
	if err != nil {
		return Expense{}, err
	}
	e := old
	e.PayerID, e.Amount, e.Currency, e.Original = in.PayerID, c.Amount, c.Currency, c.Original
	e.Category, e.Note, e.SpentOn, e.Method = in.Category, note, dateOnly(in.SpentOn), in.Method
	e.Shares = inJoinOrder(g, c.Shares)
	changes := diff(old, e)
	if len(changes) == 0 {
		return old, nil
	}
	saved, err := s.deps.Repository.Update(ctx, e, old.State, version, changes, g.MyMemberID, s.deps.Clock.Now())
	if errors.Is(err, ErrVersionConflict) || errors.Is(err, ErrGroupClosed) {
		return Expense{}, err
	}
	if err != nil {
		return Expense{}, fmt.Errorf("expenses: edit %s: %w", expenseID, err)
	}
	return saved, nil
}

// Withdraw withdraws an Expense (its creator only; FR-E6). Nothing is
// erased: it stays listed and in the Activity History, and stops counting
// toward Balances (ADR-0009). In M1 there is no agreement yet, so it is
// Withdrawn at once; with M2 an accepted one becomes WithdrawalPending.
func (s *Service) Withdraw(ctx context.Context, userID, expenseID platform.ID, version int) (Expense, error) {
	e, g, err := s.ownExpense(ctx, userID, expenseID)
	if err != nil {
		return Expense{}, err
	}
	if e.State == StateWithdrawn {
		return Expense{}, ErrInvalidState
	}
	w, err := s.deps.Repository.Withdraw(ctx, e, e.State, version, g.MyMemberID, s.deps.Clock.Now())
	if errors.Is(err, ErrVersionConflict) || errors.Is(err, ErrGroupClosed) {
		return Expense{}, err
	}
	if err != nil {
		return Expense{}, fmt.Errorf("expenses: withdraw %s: %w", expenseID, err)
	}
	return w, nil
}

// ownExpense returns the Expense and its writable Group if the User
// created it: ErrNotFound if they can't see it, ErrGroupClosed, or
// ErrNotCreator.
func (s *Service) ownExpense(ctx context.Context, userID, expenseID platform.ID) (Expense, groups.Group, error) {
	e, err := s.Get(ctx, userID, expenseID)
	if err != nil {
		return Expense{}, groups.Group{}, err
	}
	g, err := s.writableGroup(ctx, userID, e.GroupID)
	if errors.Is(err, groups.ErrNotFound) {
		return Expense{}, groups.Group{}, ErrNotFound
	}
	if err != nil {
		return Expense{}, groups.Group{}, err
	}
	if e.CreatedBy != g.MyMemberID {
		return Expense{}, groups.Group{}, ErrNotCreator
	}
	return e, g, nil
}

// diff lists the fields that differ between two revisions, by their names
// in the Activity History (FR-H1).
func diff(old, e Expense) []Change {
	var out []Change
	add := func(field string, from, to any) {
		if from != to {
			out = append(out, Change{Field: field, From: from, To: to})
		}
	}
	add("payer_member_id", old.PayerID.String(), e.PayerID.String())
	add("amount_minor", old.Amount, e.Amount)
	add("original_minor", old.Original.Amount, e.Original.Amount)
	add("original_currency", old.Original.Currency, e.Original.Currency)
	add("exchange_rate", deref(old.Original.ExchangeRate), deref(e.Original.ExchangeRate))
	add("category", old.Category, e.Category)
	add("note", deref(old.Note), deref(e.Note))
	add("spent_on", old.SpentOn.Format("2006-01-02"), e.SpentOn.Format("2006-01-02"))
	add("split_method", old.Method, e.Method)
	if !slices.EqualFunc(old.Shares, e.Shares, sameShare) {
		out = append(out, Change{Field: "shares", From: shareParts(old.Shares), To: shareParts(e.Shares)})
	}
	return out
}

func sameShare(a, b Share) bool {
	return a.MemberID == b.MemberID && a.Amount == b.Amount && deref(a.Input) == deref(b.Input)
}

// shareParts is the Shares as the Activity History keeps them.
func shareParts(shares []Share) []map[string]any {
	out := make([]map[string]any, len(shares))
	for i, s := range shares {
		out[i] = map[string]any{"member_id": s.MemberID.String(), "share_minor": s.Amount}
		if s.Input != nil {
			out[i]["input"] = *s.Input
		}
	}
	return out
}

// deref is a comparable form of an optional string: the string, or nil.
func deref(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}
