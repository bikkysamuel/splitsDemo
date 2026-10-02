-- name: LockUser :exec
-- Serializes a User's Group creations, so the 200-Group limit holds.
SELECT id FROM users WHERE id = @id FOR UPDATE;

-- name: CountActiveMemberships :one
SELECT count(*) FROM members WHERE user_id = @user_id AND status = 'active';

-- name: InsertGroup :exec
INSERT INTO groups (id, name, currency, created_at, updated_at)
VALUES (@id, @name, @currency, @now, @now);

-- name: InsertMember :one
-- Call with the Group row locked (LockGroup), so join_seq is the next one.
INSERT INTO members (id, group_id, user_id, display_name, email, role, join_seq, created_at, updated_at)
VALUES (@id, @group_id, @user_id, @display_name, @email, @role,
        (SELECT coalesce(max(join_seq), 0) + 1 FROM members WHERE group_id = @group_id), @now, @now)
RETURNING id, user_id, display_name, role, status, join_seq, version;

-- name: GroupForMember :one
-- The Group, if the User is an active Member of it, with that Member.
SELECT g.id, g.name, g.currency, g.state, g.version, m.id AS my_member_id, m.role AS my_role
FROM groups g JOIN members m ON m.group_id = g.id
WHERE g.id = @group_id AND m.user_id = @user_id AND m.status = 'active';

-- name: GroupMembers :many
SELECT id, user_id, display_name, role, status, join_seq, version FROM members
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

-- name: LockGroup :one
-- Serializes changes to a Group's Members (join_seq, the 50 limit).
SELECT state FROM groups WHERE id = @id FOR UPDATE;

-- name: CountMembers :one
SELECT count(*) FROM members WHERE group_id = @group_id;

-- name: VerifiedUserIDByEmail :one
SELECT id FROM users WHERE email = @email AND email_verified_at IS NOT NULL;

-- name: MemberEmailOrUserExists :one
-- Whether the email is already in the Group, as a Placeholder's email or as
-- the email of a Member's User.
SELECT EXISTS (
    SELECT 1 FROM members m LEFT JOIN users u ON u.id = m.user_id
    WHERE m.group_id = @group_id AND (m.email = @email OR u.email = @email)
);

-- name: DisplayNameExists :one
SELECT EXISTS (
    SELECT 1 FROM members WHERE group_id = @group_id AND lower(display_name) = lower(@display_name)
);

-- name: MemberInGroup :one
SELECT id, user_id, display_name, role, status, join_seq, version FROM members
WHERE group_id = @group_id AND id = @id;

-- name: SetMemberRole :one
UPDATE members SET role = @role, updated_at = @now, version = version + 1
WHERE group_id = @group_id AND id = @id AND version = @version
RETURNING id, user_id, display_name, role, status, join_seq, version;
