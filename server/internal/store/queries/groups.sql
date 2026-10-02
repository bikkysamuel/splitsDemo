-- name: LockUser :exec
-- Serializes a User's Group creations, so the 200-Group limit holds.
SELECT id FROM users WHERE id = @id FOR UPDATE;

-- name: CountActiveMemberships :one
SELECT count(*) FROM members WHERE user_id = @user_id AND status = 'active';

-- name: InsertGroup :exec
INSERT INTO groups (id, name, currency, created_at, updated_at)
VALUES (@id, @name, @currency, @now, @now);

-- name: InsertMember :exec
INSERT INTO members (id, group_id, user_id, display_name, email, role, join_seq, created_at, updated_at)
VALUES (@id, @group_id, @user_id, @display_name, @email, @role,
        (SELECT coalesce(max(join_seq), 0) + 1 FROM members WHERE group_id = @group_id), @now, @now);

-- name: GroupForMember :one
-- The Group, if the User is an active Member of it, with that Member.
SELECT g.id, g.name, g.currency, g.state, g.version, m.id AS my_member_id, m.role AS my_role
FROM groups g JOIN members m ON m.group_id = g.id
WHERE g.id = @group_id AND m.user_id = @user_id AND m.status = 'active';

-- name: GroupMembers :many
SELECT id, user_id, display_name, role, status, join_seq FROM members
WHERE group_id = @group_id
ORDER BY join_seq;

-- name: ListGroupsForUser :many
SELECT g.id, g.name, g.currency, g.state
FROM groups g JOIN members m ON m.group_id = g.id
WHERE m.user_id = @user_id AND m.status = 'active' AND g.id > @after
ORDER BY g.id
LIMIT @max_rows;

-- name: RenameGroup :execrows
UPDATE groups SET name = @name, updated_at = @now, version = version + 1
WHERE id = @id AND version = @version;
