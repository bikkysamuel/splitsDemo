-- Refresh-token rotation (ADR-0011) and login throttling (FR-A4, Q34).
--
-- A Session is a family of rows: each refresh adds a row with a new token
-- pair and marks the old row replaced. Presenting a replaced refresh token
-- revokes the whole family. Existing rows each start their own family.

-- +goose Up
ALTER TABLE sessions ADD COLUMN family_id uuid;
UPDATE sessions SET family_id = id;
ALTER TABLE sessions ALTER COLUMN family_id SET NOT NULL;
CREATE INDEX sessions_family_id_idx ON sessions (family_id);

-- Failed sign-ins per account (SHA-256 of the lowercased email, so unknown
-- emails are throttled alike and no address is stored) and per client IP.
CREATE TABLE login_throttle (
    scope           text NOT NULL CHECK (scope IN ('account', 'ip')),
    key             bytea NOT NULL CHECK (length(key) = 32),
    failures        integer NOT NULL CHECK (failures > 0),
    last_failure_at timestamptz NOT NULL,
    next_allowed_at timestamptz NOT NULL,
    PRIMARY KEY (scope, key)
);
