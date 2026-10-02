package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/bikkysamuel/splitsDemo/server/internal/balances"
	"github.com/bikkysamuel/splitsDemo/server/internal/ledger"
	"github.com/bikkysamuel/splitsDemo/server/internal/platform"
	"github.com/bikkysamuel/splitsDemo/server/internal/store/sqlcgen"
)

// BalancesRepository implements balances.Repository.
type BalancesRepository struct{ db *DB }

// Balances returns the Balances repository.
func (db *DB) Balances() *BalancesRepository { return &BalancesRepository{db: db} }

var _ balances.Repository = (*BalancesRepository)(nil)

// Items implements balances.Repository, reading every query from one
// snapshot so Shares always match their Expenses.
func (r *BalancesRepository) Items(ctx context.Context, groupID platform.ID) (balances.Items, error) {
	var items balances.Items
	err := r.db.inSnapshot(ctx, func(q *sqlcgen.Queries) error {
		seqs, err := q.LedgerMembers(ctx, uuid(groupID))
		if err != nil {
			return fmt.Errorf("select members: %w", err)
		}
		for _, s := range seqs {
			items.Members = append(items.Members, int(s))
		}
		exps, err := q.LedgerExpenses(ctx, uuid(groupID))
		if err != nil {
			return fmt.Errorf("select expenses: %w", err)
		}
		shares, err := q.LedgerShares(ctx, uuid(groupID))
		if err != nil {
			return fmt.Errorf("select shares: %w", err)
		}
		byExpense := make(map[pgtype.UUID][]ledger.Share, len(exps))
		for _, s := range shares {
			byExpense[s.ExpenseID] = append(byExpense[s.ExpenseID], ledger.Share{JoinSeq: int(s.JoinSeq), Amount: s.ShareMinor})
		}
		for _, e := range exps {
			items.Expenses = append(items.Expenses, ledger.Expense{
				State: itemState(e.State), Payer: int(e.PayerJoinSeq), Amount: e.AmountMinor, Shares: byExpense[e.ID],
			})
		}
		return nil
	})
	return items, err
}

// itemState maps a stored state; an unknown one is 0, which ledger
// refuses as inconsistent.
func itemState(s string) ledger.ItemState {
	switch s {
	case "pending":
		return ledger.Pending
	case "accepted":
		return ledger.Accepted
	case "disputed":
		return ledger.Disputed
	case "withdrawal_pending":
		return ledger.WithdrawalPending
	case "withdrawn":
		return ledger.Withdrawn
	}
	return 0
}

// inSnapshot runs fn in a read-only REPEATABLE READ transaction: every
// query sees the same snapshot.
func (db *DB) inSnapshot(ctx context.Context, fn func(q *sqlcgen.Queries) error) error {
	tx, err := db.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return fmt.Errorf("begin snapshot: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(sqlcgen.New(tx)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("end snapshot: %w", err)
	}
	return nil
}
