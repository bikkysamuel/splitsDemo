-- name: UpsertUnverifiedUser :one
-- Creates the User, or replaces the password of the unverified User with the
-- same email. Returns no row when a verified User has the email.
INSERT INTO users (id, email, password_hash, created_at, updated_at)
VALUES (@id, @email, @password_hash, @now, @now)
ON CONFLICT (email) DO UPDATE
    SET password_hash = excluded.password_hash,
        updated_at = excluded.updated_at,
        version = users.version + 1
    WHERE users.email_verified_at IS NULL
RETURNING id, email, email_verified_at;

-- name: RevokeUserSessions :exec
UPDATE sessions SET revoked_at = @now
WHERE user_id = @user_id AND revoked_at IS NULL;

-- name: DeleteUserCodes :exec
DELETE FROM one_time_codes WHERE user_id = @user_id AND purpose = @purpose;

-- name: InsertCode :exec
INSERT INTO one_time_codes (id, user_id, purpose, code_hash, expires_at, created_at)
VALUES (@id, @user_id, @purpose, @code_hash, @expires_at, @created_at);

-- name: InsertSession :exec
INSERT INTO sessions (id, user_id, access_hash, access_expires_at, refresh_hash, refresh_expires_at, created_at)
VALUES (@id, @user_id, @access_hash, @access_expires_at, @refresh_hash, @refresh_expires_at, @created_at);

-- name: UserByEmail :one
SELECT id, email, password_hash, email_verified_at FROM users WHERE email = @email;

-- name: UserByID :one
SELECT id, email, email_verified_at FROM users WHERE id = @id;

-- name: SessionByAccessHash :one
SELECT s.id, s.user_id, s.access_expires_at, s.revoked_at, u.email_verified_at
FROM sessions s JOIN users u ON u.id = s.user_id
WHERE s.access_hash = @access_hash;

-- name: LiveCode :one
SELECT id, code_hash, expires_at, attempts FROM one_time_codes
WHERE user_id = @user_id AND purpose = @purpose AND consumed_at IS NULL;

-- name: CountCodeAttempt :execrows
UPDATE one_time_codes SET attempts = attempts + 1
WHERE id = @id AND consumed_at IS NULL AND attempts < @max_attempts::integer;

-- name: ConsumeCode :one
UPDATE one_time_codes SET consumed_at = @now
WHERE id = @id AND consumed_at IS NULL
RETURNING user_id;

-- name: MarkEmailVerified :exec
UPDATE users SET email_verified_at = @now, updated_at = @now, version = version + 1
WHERE id = @id AND email_verified_at IS NULL;
