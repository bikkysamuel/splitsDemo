package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/bikkysamuel/splitsDemo/server/internal/groups"
	"github.com/bikkysamuel/splitsDemo/server/internal/platform"
	"github.com/bikkysamuel/splitsDemo/server/internal/store/sqlcgen"
)

// GroupsRepository implements groups.Repository.
type GroupsRepository struct{ db *DB }

// Groups returns the Groups repository.
func (db *DB) Groups() *GroupsRepository { return &GroupsRepository{db: db} }

var _ groups.Repository = (*GroupsRepository)(nil)

// CreateGroup implements groups.Repository.
func (r *GroupsRepository) CreateGroup(ctx context.Context, g groups.NewGroup, maxGroups int) error {
	return r.db.inTx(ctx, func(q *sqlcgen.Queries) error {
		if err := q.LockUser(ctx, uuid(g.CreatorID)); err != nil {
			return fmt.Errorf("lock user: %w", err)
		}
		n, err := q.CountActiveMemberships(ctx, uuid(g.CreatorID))
		if err != nil {
			return fmt.Errorf("count memberships: %w", err)
		}
		if n >= int64(maxGroups) {
			return groups.ErrGroupLimit
		}
		err = q.InsertGroup(ctx, sqlcgen.InsertGroupParams{
			ID: uuid(g.ID), Name: g.Name, Currency: g.Currency, Now: timestamptz(g.Now),
		})
		if err != nil {
			return fmt.Errorf("insert group: %w", err)
		}
		err = q.InsertMember(ctx, sqlcgen.InsertMemberParams{
			ID: uuid(g.MemberID), GroupID: uuid(g.ID), UserID: uuid(g.CreatorID),
			DisplayName: g.DisplayName, Role: string(groups.RoleAdmin), Now: timestamptz(g.Now),
		})
		if err != nil {
			return fmt.Errorf("insert first admin: %w", err)
		}
		return nil
	})
}

// GroupForUser implements groups.Repository.
func (r *GroupsRepository) GroupForUser(ctx context.Context, groupID, userID platform.ID) (groups.Group, error) {
	q := sqlcgen.New(r.db.pool)
	row, err := q.GroupForMember(ctx, sqlcgen.GroupForMemberParams{GroupID: uuid(groupID), UserID: uuid(userID)})
	if isNoRows(err) {
		return groups.Group{}, groups.ErrNotFound
	}
	if err != nil {
		return groups.Group{}, fmt.Errorf("select group: %w", err)
	}
	rows, err := q.GroupMembers(ctx, row.ID)
	if err != nil {
		return groups.Group{}, fmt.Errorf("select members: %w", err)
	}
	g := groups.Group{
		Summary:    groups.Summary{ID: id(row.ID), Name: row.Name, Currency: row.Currency, State: groups.State(row.State)},
		Version:    int(row.Version),
		MyMemberID: id(row.MyMemberID),
		MyRole:     groups.Role(row.MyRole),
		Members:    make([]groups.Member, len(rows)),
	}
	for i, m := range rows {
		g.Members[i] = groups.Member{
			ID: id(m.ID), UserID: optionalID(m.UserID), DisplayName: m.DisplayName,
			Role: groups.Role(m.Role), Status: groups.MemberStatus(m.Status), JoinSeq: int(m.JoinSeq),
		}
	}
	return g, nil
}

// ListGroups implements groups.Repository.
func (r *GroupsRepository) ListGroups(ctx context.Context, userID, after platform.ID, limit int) ([]groups.Summary, error) {
	rows, err := sqlcgen.New(r.db.pool).ListGroupsForUser(ctx, sqlcgen.ListGroupsForUserParams{
		UserID: uuid(userID), After: uuid(after), MaxRows: int32(limit), //nolint:gosec // limit ≤ 201
	})
	if err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}
	items := make([]groups.Summary, len(rows))
	for i, g := range rows {
		items[i] = groups.Summary{ID: id(g.ID), Name: g.Name, Currency: g.Currency, State: groups.State(g.State)}
	}
	return items, nil
}

// RenameGroup implements groups.Repository.
func (r *GroupsRepository) RenameGroup(ctx context.Context, groupID platform.ID, name string, version int, now time.Time) error {
	n, err := sqlcgen.New(r.db.pool).RenameGroup(ctx, sqlcgen.RenameGroupParams{
		Name: name, Now: timestamptz(now), ID: uuid(groupID), Version: int32(version), //nolint:gosec // a version
	})
	if err != nil {
		return fmt.Errorf("rename group: %w", err)
	}
	if n == 0 {
		return groups.ErrVersionConflict
	}
	return nil
}

func optionalID(u pgtype.UUID) *platform.ID {
	if !u.Valid {
		return nil
	}
	v := id(u)
	return &v
}
