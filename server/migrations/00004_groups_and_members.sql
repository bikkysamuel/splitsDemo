-- Groups and Members (doc 06, FR-G*, FR-M*).

-- +goose Up
CREATE TABLE groups (
    id              uuid PRIMARY KEY,
    parent_group_id uuid REFERENCES groups (id),
    name            text NOT NULL CHECK (length(name) BETWEEN 1 AND 100),
    currency        char(3) NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    state           text NOT NULL DEFAULT 'active' CHECK (state IN ('active', 'closing', 'closed')),
    closing_since   timestamptz,
    closed_at       timestamptz,
    created_at      timestamptz NOT NULL,
    updated_at      timestamptz NOT NULL,
    version         integer NOT NULL DEFAULT 1
);

-- Sub-Groups are one level deep (FR-G5): a Parent Group can't itself have a
-- parent.
-- +goose StatementBegin
CREATE FUNCTION groups_one_level() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.parent_group_id IS NOT NULL AND EXISTS (
        SELECT 1 FROM groups WHERE id = NEW.parent_group_id AND parent_group_id IS NOT NULL
    ) THEN
        RAISE EXCEPTION 'a Sub-Group cannot have Sub-Groups (FR-G5)' USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER groups_one_level BEFORE INSERT OR UPDATE OF parent_group_id ON groups
    FOR EACH ROW EXECUTE FUNCTION groups_one_level();

-- Placeholder ⇔ user_id IS NULL. join_seq breaks rounding ties (ADR-0010).
CREATE TABLE members (
    id               uuid PRIMARY KEY,
    group_id         uuid NOT NULL REFERENCES groups (id),
    user_id          uuid REFERENCES users (id) ON DELETE SET NULL,
    display_name     text NOT NULL CHECK (length(display_name) BETWEEN 1 AND 50),
    email            citext CHECK (length(email) <= 254),
    role             text NOT NULL CHECK (role IN ('admin', 'member')),
    status           text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'former')),
    join_seq         integer NOT NULL CHECK (join_seq > 0),
    parent_member_id uuid REFERENCES members (id),
    created_at       timestamptz NOT NULL,
    updated_at       timestamptz NOT NULL,
    version          integer NOT NULL DEFAULT 1,
    UNIQUE (group_id, join_seq),
    UNIQUE (group_id, email),
    UNIQUE (group_id, user_id)
);
-- Display names are unique within a Group, ignoring case (FR-M4).
CREATE UNIQUE INDEX members_group_display_name_idx ON members (group_id, lower(display_name));
CREATE INDEX members_user_id_idx ON members (user_id) WHERE user_id IS NOT NULL;
