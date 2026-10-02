-- Settlements (doc 06, FR-S*).

-- +goose Up
CREATE TABLE settlements (
    id             uuid PRIMARY KEY,
    group_id       uuid NOT NULL REFERENCES groups (id),
    from_member_id uuid NOT NULL REFERENCES members (id),
    to_member_id   uuid NOT NULL REFERENCES members (id),
    amount_minor   bigint NOT NULL CHECK (amount_minor > 0),
    settled_on     date NOT NULL,
    note           text CHECK (length(note) BETWEEN 1 AND 500),
    created_by     uuid NOT NULL REFERENCES members (id),
    state          text NOT NULL CHECK (state IN ('pending', 'accepted', 'disputed', 'withdrawal_pending', 'withdrawn')),
    pending_since  timestamptz,
    confirmed_by   uuid REFERENCES members (id),
    created_at     timestamptz NOT NULL,
    updated_at     timestamptz NOT NULL,
    version        integer NOT NULL DEFAULT 1,
    CHECK (from_member_id <> to_member_id)
);
CREATE INDEX settlements_group_list_idx ON settlements (group_id, settled_on DESC, id DESC);
