-- name: ClaimIdempotencyKey :one
-- Stores an in-progress key. A key created at or before @stale_before is past its
-- 24 h and is replaced. Returns no row when a live key exists.
INSERT INTO idempotency_keys (user_id, key, request_hash, status, created_at)
VALUES (@user_id, @key, @request_hash, 'in_progress', @now)
ON CONFLICT (user_id, key) DO UPDATE
    SET request_hash = excluded.request_hash,
        status = 'in_progress',
        response_status = NULL,
        response_content_type = NULL,
        response_body = NULL,
        created_at = excluded.created_at
    WHERE idempotency_keys.created_at <= @stale_before
RETURNING created_at;

-- name: IdempotencyKey :one
SELECT request_hash, status, response_status, response_content_type, response_body
FROM idempotency_keys WHERE user_id = @user_id AND key = @key;

-- name: CompleteIdempotencyKey :exec
UPDATE idempotency_keys
SET status = 'completed', response_status = @response_status,
    response_content_type = @response_content_type, response_body = @response_body
WHERE user_id = @user_id AND key = @key AND status = 'in_progress';

-- name: ReleaseIdempotencyKey :exec
DELETE FROM idempotency_keys WHERE user_id = @user_id AND key = @key AND status = 'in_progress';
