package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
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
		_, err = q.InsertMember(ctx, sqlcgen.InsertMemberParams{
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
		g.Members[i] = member(m.ID, m.UserID, m.DisplayName, m.Role, m.Status, m.JoinSeq, m.Version)
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

// AddMember implements groups.Repository.
func (r *GroupsRepository) AddMember(ctx context.Context, m groups.NewMember, maxMembers int) (groups.Member, error) {
	var added groups.Member
	err := r.db.inTx(ctx, func(q *sqlcgen.Queries) error {
		state, err := q.LockGroup(ctx, uuid(m.GroupID))
		if err != nil {
			return fmt.Errorf("lock group: %w", err)
		}
		if groups.State(state) == groups.StateClosed {
			return groups.ErrGroupClosed
		}
		n, err := q.CountMembers(ctx, uuid(m.GroupID))
		if err != nil {
			return fmt.Errorf("count members: %w", err)
		}
		if n >= int64(maxMembers) {
			return groups.ErrMemberLimit
		}
		var taken []groups.FieldError
		nameTaken, err := q.DisplayNameExists(ctx, sqlcgen.DisplayNameExistsParams{GroupID: uuid(m.GroupID), DisplayName: m.DisplayName})
		if err != nil {
			return fmt.Errorf("check display name: %w", err)
		}
		if nameTaken {
			taken = append(taken, groups.FieldError{Field: "display_name", Code: groups.CodeTaken})
		}
		var userID pgtype.UUID
		email := pgtype.Text{}
		if m.Email != nil {
			emailTaken, err := q.MemberEmailOrUserExists(ctx, sqlcgen.MemberEmailOrUserExistsParams{GroupID: uuid(m.GroupID), Email: pgtype.Text{String: *m.Email, Valid: true}})
			if err != nil {
				return fmt.Errorf("check email: %w", err)
			}
			if emailTaken {
				taken = append(taken, groups.FieldError{Field: "email", Code: groups.CodeTaken})
			}
			userID, err = q.VerifiedUserIDByEmail(ctx, *m.Email)
			switch {
			case isNoRows(err):
				// No verified User: a Placeholder carrying the email (ADR-0017).
				email = pgtype.Text{String: *m.Email, Valid: true}
			case err != nil:
				return fmt.Errorf("find user by email: %w", err)
			}
		}
		if len(taken) > 0 {
			return &groups.ValidationError{Fields: taken}
		}
		row, err := q.InsertMember(ctx, sqlcgen.InsertMemberParams{
			ID: uuid(m.ID), GroupID: uuid(m.GroupID), UserID: userID, DisplayName: m.DisplayName,
			Email: email, Role: string(groups.RoleMember), Now: timestamptz(m.Now),
		})
		if f, ok := takenField(err); ok {
			// The database's uniqueness is the backstop for the checks above.
			return &groups.ValidationError{Fields: []groups.FieldError{{Field: f, Code: groups.CodeTaken}}}
		}
		if err != nil {
			return fmt.Errorf("insert member: %w", err)
		}
		added = member(row.ID, row.UserID, row.DisplayName, row.Role, row.Status, row.JoinSeq, row.Version)
		return nil
	})
	return added, err
}

// MakeAdmin implements groups.Repository.
func (r *GroupsRepository) MakeAdmin(ctx context.Context, groupID, memberID platform.ID, version int, now time.Time) (groups.Member, error) {
	var updated groups.Member
	err := r.db.inTx(ctx, func(q *sqlcgen.Queries) error {
		state, err := q.LockGroup(ctx, uuid(groupID))
		if err != nil {
			return fmt.Errorf("lock group: %w", err)
		}
		if groups.State(state) == groups.StateClosed {
			return groups.ErrGroupClosed
		}
		cur, err := q.MemberInGroup(ctx, sqlcgen.MemberInGroupParams{GroupID: uuid(groupID), ID: uuid(memberID)})
		if isNoRows(err) {
			return groups.ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("select member: %w", err)
		}
		if !cur.UserID.Valid || groups.MemberStatus(cur.Status) != groups.StatusActive {
			return groups.ErrMemberNotEligible
		}
		row, err := q.SetMemberRole(ctx, sqlcgen.SetMemberRoleParams{
			Role: string(groups.RoleAdmin), Now: timestamptz(now), GroupID: uuid(groupID), ID: uuid(memberID),
			Version: int32(version), //nolint:gosec // a version
		})
		if isNoRows(err) {
			return groups.ErrVersionConflict
		}
		if err != nil {
			return fmt.Errorf("set member role: %w", err)
		}
		updated = member(row.ID, row.UserID, row.DisplayName, row.Role, row.Status, row.JoinSeq, row.Version)
		return nil
	})
	return updated, err
}

func member(memberID, userID pgtype.UUID, displayName, role, status string, joinSeq, version int32) groups.Member {
	return groups.Member{
		ID: id(memberID), UserID: optionalID(userID), DisplayName: displayName,
		Role: groups.Role(role), Status: groups.MemberStatus(status), JoinSeq: int(joinSeq), Version: int(version),
	}
}

// takenField names the request field a unique violation on members is
// about.
func takenField(err error) (string, bool) {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return "", false
	}
	switch pgErr.ConstraintName {
	case "members_group_display_name_idx":
		return "display_name", true
	case "members_group_id_email_key", "members_group_id_user_id_key":
		return "email", true
	}
	return "", false
}

func optionalID(u pgtype.UUID) *platform.ID {
	if !u.Valid {
		return nil
	}
	v := id(u)
	return &v
}
