package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bikkysamuel/splitsDemo/server/internal/groups"
	"github.com/bikkysamuel/splitsDemo/server/internal/platform"
	"github.com/bikkysamuel/splitsDemo/server/internal/settlements"
	"github.com/bikkysamuel/splitsDemo/server/internal/store/sqlcgen"
)

// SettlementsRepository implements settlements.Repository.
type SettlementsRepository struct{ db *DB }

// Settlements returns the Settlements repository.
func (db *DB) Settlements() *SettlementsRepository { return &SettlementsRepository{db: db} }

var _ settlements.Repository = (*SettlementsRepository)(nil)

// Create implements settlements.Repository.
func (r *SettlementsRepository) Create(ctx context.Context, s settlements.Settlement) error {
	return r.db.inTx(ctx, func(q *sqlcgen.Queries) error {
		g, err := q.LockGroupShared(ctx, uuid(s.GroupID))
		if err != nil {
			return fmt.Errorf("lock group: %w", err)
		}
		if groups.State(g.State) == groups.StateClosed {
			return settlements.ErrGroupClosed
		}
		if g.Currency != s.Currency {
			return &settlements.ValidationError{Fields: []settlements.FieldError{{Field: "amount/currency", Code: settlements.CodeNotGroupCurrency}}}
		}
		err = q.InsertSettlement(ctx, sqlcgen.InsertSettlementParams{
			ID: uuid(s.ID), GroupID: uuid(s.GroupID), FromMemberID: uuid(s.From), ToMemberID: uuid(s.To),
			AmountMinor: s.Amount, SettledOn: date(s.SettledOn), Note: text(s.Note), CreatedBy: uuid(s.CreatedBy),
			State: s.State, Now: timestamptz(s.CreatedAt),
		})
		if err != nil {
			return fmt.Errorf("insert settlement: %w", err)
		}
		return insertEvent(ctx, q, s.GroupID, s.CreatedBy, "settlement_recorded", "settlement", s.ID,
			map[string]any{"amount_minor": s.Amount, "currency": s.Currency}, s.CreatedAt)
	})
}

// ForUser implements settlements.Repository.
func (r *SettlementsRepository) ForUser(ctx context.Context, settlementID, userID platform.ID) (settlements.Settlement, platform.ID, error) {
	row, err := sqlcgen.New(r.db.pool).SettlementForUser(ctx, sqlcgen.SettlementForUserParams{ID: uuid(settlementID), UserID: uuid(userID)})
	if isNoRows(err) {
		return settlements.Settlement{}, platform.ID{}, settlements.ErrNotFound
	}
	if err != nil {
		return settlements.Settlement{}, platform.ID{}, fmt.Errorf("select settlement: %w", err)
	}
	return settlement(sqlcgen.ListSettlementsRow{
		ID: row.ID, GroupID: row.GroupID, FromMemberID: row.FromMemberID, ToMemberID: row.ToMemberID,
		AmountMinor: row.AmountMinor, SettledOn: row.SettledOn, Note: row.Note, CreatedBy: row.CreatedBy,
		State: row.State, Version: row.Version, CreatedAt: row.CreatedAt, Currency: row.Currency,
	}), id(row.MyMemberID), nil
}

// List implements settlements.Repository.
func (r *SettlementsRepository) List(ctx context.Context, groupID platform.ID, after *settlements.Cursor, limit int) ([]settlements.Settlement, error) {
	params := sqlcgen.ListSettlementsParams{GroupID: uuid(groupID), MaxRows: int32(limit)} //nolint:gosec // ≤ 201
	if after != nil {
		params.HasCursor = true
		params.AfterSettledOn = date(after.SettledOn)
		params.AfterID = uuid(after.ID)
	}
	rows, err := sqlcgen.New(r.db.pool).ListSettlements(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("list settlements: %w", err)
	}
	items := make([]settlements.Settlement, len(rows))
	for i, row := range rows {
		items[i] = settlement(row)
	}
	return items, nil
}

func settlement(row sqlcgen.ListSettlementsRow) settlements.Settlement {
	return settlements.Settlement{
		ID: id(row.ID), GroupID: id(row.GroupID), From: id(row.FromMemberID), To: id(row.ToMemberID),
		Amount: row.AmountMinor, Currency: row.Currency, SettledOn: row.SettledOn.Time, Note: optionalText(row.Note),
		CreatedBy: id(row.CreatedBy), State: row.State, Version: int(row.Version), CreatedAt: row.CreatedAt.Time,
	}
}

// Withdraw implements settlements.Repository.
func (r *SettlementsRepository) Withdraw(ctx context.Context, s settlements.Settlement, fromState string, version int, actor platform.ID, now time.Time) (settlements.Settlement, error) {
	err := r.db.inTx(ctx, func(q *sqlcgen.Queries) error {
		g, err := q.LockGroupShared(ctx, uuid(s.GroupID))
		if err != nil {
			return fmt.Errorf("lock group: %w", err)
		}
		if groups.State(g.State) == groups.StateClosed {
			return settlements.ErrGroupClosed
		}
		v, err := q.SetSettlementState(ctx, sqlcgen.SetSettlementStateParams{
			State: settlements.StateWithdrawn, Now: timestamptz(now), ID: uuid(s.ID),
			Version: int32(version), FromState: fromState, //nolint:gosec // a version
		})
		if isNoRows(err) {
			return settlements.ErrVersionConflict
		}
		if err != nil {
			return fmt.Errorf("set settlement state: %w", err)
		}
		s.State, s.Version = settlements.StateWithdrawn, int(v)
		return insertEvent(ctx, q, s.GroupID, actor, "settlement_withdrawn", "settlement", s.ID, map[string]any{}, now)
	})
	return s, err
}

// insertEvent appends one Activity History event (D19).
func insertEvent(ctx context.Context, q *sqlcgen.Queries, groupID, actor platform.ID, typ, subjectType string, subjectID platform.ID, payload map[string]any, at time.Time) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode activity payload: %w", err)
	}
	err = q.InsertActivityEvent(ctx, sqlcgen.InsertActivityEventParams{
		GroupID: uuid(groupID), ActorMemberID: uuid(actor), Type: typ, SubjectType: subjectType,
		SubjectID: uuid(subjectID), Payload: raw, OccurredAt: timestamptz(at),
	})
	if err != nil {
		return fmt.Errorf("insert activity event: %w", err)
	}
	return nil
}
