-- Identity (doc 06): Users, Sessions (ADR-0011), one-time codes (ADR-0016),
-- and the idempotency keys of signed-in writes (NFR-R1).

-- +goose Up
CREATE TABLE users (
    id                uuid PRIMARY KEY,
    email             citext NOT NULL UNIQUE CHECK (length(email) BETWEEN 3 AND 254),
    password_hash     text NOT NULL,
    email_verified_at timestamptz,
    created_at        timestamptz NOT NULL,
    updated_at        timestamptz NOT NULL,
    version           integer NOT NULL DEFAULT 1
);

-- Only SHA-256 hashes of tokens are stored (ADR-0011).
CREATE TABLE sessions (
    id                 uuid PRIMARY KEY,
    user_id            uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    access_hash        bytea NOT NULL UNIQUE CHECK (length(access_hash) = 32),
    access_expires_at  timestamptz NOT NULL,
    refresh_hash       bytea NOT NULL UNIQUE CHECK (length(refresh_hash) = 32),
    refresh_expires_at timestamptz NOT NULL,
    replaced_by        uuid REFERENCES sessions (id),
    revoked_at         timestamptz,
    created_at         timestamptz NOT NULL
);
CREATE INDEX sessions_user_id_idx ON sessions (user_id);

CREATE TABLE one_time_codes (
    id          uuid PRIMARY KEY,
    user_id     uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    purpose     text NOT NULL CHECK (purpose IN ('verify_email', 'reset_password')),
    code_hash   bytea NOT NULL CHECK (length(code_hash) = 32),
    expires_at  timestamptz NOT NULL,
    attempts    integer NOT NULL DEFAULT 0 CHECK (attempts BETWEEN 0 AND 5),
    consumed_at timestamptz,
    created_at  timestamptz NOT NULL
);
-- At most one live code per User and purpose: a new code replaces the old.
CREATE UNIQUE INDEX one_time_codes_live_idx ON one_time_codes (user_id, purpose) WHERE consumed_at IS NULL;

-- A key is kept 24 h (D14). The response is stored once the request
-- completes; 5xx responses are not kept, so the request can be retried.
CREATE TABLE idempotency_keys (
    user_id               uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    key                   uuid NOT NULL,
    request_hash          bytea NOT NULL CHECK (length(request_hash) = 32),
    status                text NOT NULL CHECK (status IN ('in_progress', 'completed')),
    response_status       integer,
    response_content_type text,
    response_body         bytea,
    created_at            timestamptz NOT NULL,
    PRIMARY KEY (user_id, key),
    CHECK ((status = 'completed') = (response_status IS NOT NULL))
);
