package store

import (
	"context"
	"fmt"
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
		err = q.InsertExpense(ctx, sqlcgen.InsertExpenseParams{
			ID: uuid(e.ID), GroupID: uuid(e.GroupID), CreatedBy: uuid(e.CreatedBy), PayerID: uuid(e.PayerID),
			Category: e.Category, Note: text(e.Note), SpentOn: date(e.SpentOn),
			OriginalMinor: e.Amount, OriginalCurrency: e.Currency, AmountMinor: e.Amount,
			SplitMethod: e.Method, State: e.State, Now: timestamptz(e.CreatedAt),
		})
		if err != nil {
			return fmt.Errorf("insert expense: %w", err)
		}
		for _, s := range e.Shares {
			err := q.InsertShare(ctx, sqlcgen.InsertShareParams{
				ExpenseID: uuid(e.ID), MemberID: uuid(s.MemberID), ShareMinor: s.Amount,
			})
			if err != nil {
				return fmt.Errorf("insert share: %w", err)
			}
		}
		return insertEvent(ctx, q, e.GroupID, e.CreatedBy, "expense_created", "expense", e.ID,
			map[string]any{"amount_minor": e.Amount, "currency": e.Currency, "category": e.Category}, e.CreatedAt)
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
		SpentOn: row.SpentOn.Time, Method: row.SplitMethod, State: row.State, Version: int(row.Version),
		CreatedAt: row.CreatedAt.Time, Shares: make([]expenses.Share, len(shares)),
	}
	for i, s := range shares {
		e.Shares[i] = expenses.Share{MemberID: id(s.MemberID), Amount: s.ShareMinor}
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
