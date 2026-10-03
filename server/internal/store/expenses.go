package store

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/bikkysamuel/splitsDemo/server/internal/expenses"
	"github.com/bikkysamuel/splitsDemo/server/internal/groups"
	"github.com/bikkysamuel/splitsDemo/server/internal/platform"
	"github.com/bikkysamuel/splitsDemo/server/internal/store/sqlcgen"
)

// ExpensesRepository implements expenses.Repository.
type ExpensesRepository struct{ db *DB }

// Expenses returns the Expenses repository.
func (db *DB) Expenses() *ExpensesRepository { return &ExpensesRepository{db: db} }

var _ expenses.Repository = (*ExpensesRepository)(nil)

// Create implements expenses.Repository.
func (r *ExpensesRepository) Create(ctx context.Context, e expenses.Expense) error {
	return r.db.inTx(ctx, func(q *sqlcgen.Queries) error {
		g, err := q.LockGroupShared(ctx, uuid(e.GroupID))
		if err != nil {
			return fmt.Errorf("lock group: %w", err)
		}
		if groups.State(g.State) == groups.StateClosed {
			return expenses.ErrGroupClosed
		}
		// Checked again under the lock: the service validated before it.
		if g.Currency != e.Currency {
			return &expenses.ValidationError{Fields: []expenses.FieldError{{Field: "amount/currency", Code: expenses.CodeNotGroupCurrency}}}
		}
		rate, err := numeric(e.Original.ExchangeRate)
		if err != nil {
			return err
		}
		err = q.InsertExpense(ctx, sqlcgen.InsertExpenseParams{
			ID: uuid(e.ID), GroupID: uuid(e.GroupID), CreatedBy: uuid(e.CreatedBy), PayerID: uuid(e.PayerID),
			Category: e.Category, Note: text(e.Note), SpentOn: date(e.SpentOn),
			OriginalMinor: e.Original.Amount, OriginalCurrency: e.Original.Currency, ExchangeRate: rate, AmountMinor: e.Amount,
			SplitMethod: e.Method, State: e.State, Now: timestamptz(e.CreatedAt),
		})
		if err != nil {
			return fmt.Errorf("insert expense: %w", err)
		}
		for _, s := range e.Shares {
			input, err := numeric(s.Input)
			if err != nil {
				return err
			}
			err = q.InsertShare(ctx, sqlcgen.InsertShareParams{
				ExpenseID: uuid(e.ID), MemberID: uuid(s.MemberID), ShareMinor: s.Amount, Input: input,
			})
			if err != nil {
				return fmt.Errorf("insert share: %w", err)
			}
		}
		payload := map[string]any{"amount_minor": e.Amount, "currency": e.Currency, "category": e.Category}
		if e.Original.ExchangeRate != nil {
			payload["original_minor"] = e.Original.Amount
			payload["original_currency"] = e.Original.Currency
			payload["exchange_rate"] = *e.Original.ExchangeRate
		}
		return insertEvent(ctx, q, e.GroupID, e.CreatedBy, "expense_created", "expense", e.ID, payload, e.CreatedAt)
	})
}

// ExpenseForUser implements expenses.Repository.
func (r *ExpensesRepository) ExpenseForUser(ctx context.Context, expenseID, userID platform.ID) (expenses.Expense, error) {
	q := sqlcgen.New(r.db.pool)
	row, err := q.ExpenseForUser(ctx, sqlcgen.ExpenseForUserParams{ID: uuid(expenseID), UserID: uuid(userID)})
	if isNoRows(err) {
		return expenses.Expense{}, expenses.ErrNotFound
	}
	if err != nil {
		return expenses.Expense{}, fmt.Errorf("select expense: %w", err)
	}
	shares, err := q.ExpenseShares(ctx, row.ID)
	if err != nil {
		return expenses.Expense{}, fmt.Errorf("select shares: %w", err)
	}
	e := expenses.Expense{
		ID: id(row.ID), GroupID: id(row.GroupID), PayerID: id(row.PayerID), CreatedBy: id(row.CreatedBy),
		Amount: row.AmountMinor, Currency: row.Currency, Category: row.Category, Note: optionalText(row.Note),
		Original: expenses.Original{
			Amount: row.OriginalMinor, Currency: row.OriginalCurrency, ExchangeRate: numericText(row.ExchangeRate),
		},
		SpentOn: row.SpentOn.Time, Method: row.SplitMethod, State: row.State, Version: int(row.Version),
		CreatedAt: row.CreatedAt.Time, Shares: make([]expenses.Share, len(shares)),
	}
	for i, s := range shares {
		e.Shares[i] = expenses.Share{MemberID: id(s.MemberID), Amount: s.ShareMinor, Input: numericText(s.Input)}
	}
	return e, nil
}

// List implements expenses.Repository.
func (r *ExpensesRepository) List(ctx context.Context, groupID platform.ID, after *expenses.Cursor, limit int) ([]expenses.Summary, error) {
	params := sqlcgen.ListExpensesParams{GroupID: uuid(groupID), MaxRows: int32(limit)} //nolint:gosec // ≤ 201
	if after != nil {
		params.HasCursor = true
		params.AfterSpentOn = date(after.SpentOn)
		params.AfterID = uuid(after.ID)
	}
	rows, err := sqlcgen.New(r.db.pool).ListExpenses(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("list expenses: %w", err)
	}
	items := make([]expenses.Summary, len(rows))
	for i, e := range rows {
		items[i] = expenses.Summary{
			ID: id(e.ID), PayerID: id(e.PayerID), Amount: e.AmountMinor, Currency: e.Currency,
			Category: e.Category, Note: optionalText(e.Note), SpentOn: e.SpentOn.Time, State: e.State,
		}
	}
	return items, nil
}

func text(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func optionalText(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}

func date(t time.Time) pgtype.Date { return pgtype.Date{Time: t, Valid: true} }

// numericText is a NUMERIC's exact decimal text, its scale kept ("33.30");
// NULL is nil.
func numericText(n pgtype.Numeric) *string {
	if !n.Valid || n.Int == nil {
		return nil
	}
	digits := new(big.Int).Abs(n.Int).String()
	if n.Exp >= 0 {
		digits += strings.Repeat("0", int(n.Exp))
	} else {
		scale := int(-n.Exp)
		if len(digits) <= scale {
			digits = strings.Repeat("0", scale-len(digits)+1) + digits
		}
		digits = digits[:len(digits)-scale] + "." + digits[len(digits)-scale:]
	}
	if n.Int.Sign() < 0 {
		digits = "-" + digits
	}
	return &digits
}

// numeric stores a decimal string exactly as NUMERIC (ADR-0014); nil is
// NULL.
func numeric(s *string) (pgtype.Numeric, error) {
	var n pgtype.Numeric
	if s == nil {
		return n, nil
	}
	if err := n.Scan(*s); err != nil {
		return n, fmt.Errorf("numeric %q: %w", *s, err)
	}
	return n, nil
}
