-- name: InsertSettlement :exec
INSERT INTO settlements (id, group_id, from_member_id, to_member_id, amount_minor, settled_on, note, created_by,
                         state, created_at, updated_at)
VALUES (@id, @group_id, @from_member_id, @to_member_id, @amount_minor, @settled_on, @note, @created_by,
        @state, @now, @now);

-- name: SettlementForUser :one
-- The Settlement, if the User is an active Member of its Group.
SELECT s.id, s.group_id, s.from_member_id, s.to_member_id, s.amount_minor, s.settled_on, s.note, s.created_by,
       s.state, s.version, s.created_at, g.currency, m.id AS my_member_id
FROM settlements s
JOIN groups g ON g.id = s.group_id
JOIN members m ON m.group_id = s.group_id
WHERE s.id = @id AND m.user_id = @user_id AND m.status = 'active';

-- name: ListSettlements :many
SELECT s.id, s.group_id, s.from_member_id, s.to_member_id, s.amount_minor, s.settled_on, s.note, s.created_by,
       s.state, s.version, s.created_at, g.currency
FROM settlements s JOIN groups g ON g.id = s.group_id
WHERE s.group_id = @group_id
  AND (NOT @has_cursor::boolean OR (s.settled_on, s.id) < (@after_settled_on::date, @after_id::uuid))
ORDER BY s.settled_on DESC, s.id DESC
LIMIT @max_rows;

-- name: SetSettlementState :one
UPDATE settlements SET state = @state, updated_at = @now, version = version + 1
WHERE id = @id AND version = @version AND state = @from_state
RETURNING version;

-- name: LedgerSettlements :many
SELECT s.state, f.join_seq AS from_join_seq, t.join_seq AS to_join_seq, s.amount_minor
FROM settlements s
JOIN members f ON f.id = s.from_member_id
JOIN members t ON t.id = s.to_member_id
WHERE s.group_id = @group_id;
