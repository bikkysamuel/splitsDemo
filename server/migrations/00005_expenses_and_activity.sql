-- Expenses, their Shares and the Activity History (doc 06, FR-E*, FR-H*).

-- +goose Up
CREATE TABLE expenses (
    id                uuid PRIMARY KEY,
    group_id          uuid NOT NULL REFERENCES groups (id),
    created_by        uuid NOT NULL REFERENCES members (id),
    payer_id          uuid NOT NULL REFERENCES members (id),
    category          text NOT NULL CHECK (category IN ('food_drink', 'groceries', 'transport', 'accommodation',
                          'rent', 'utilities', 'entertainment', 'shopping', 'health', 'travel', 'other')),
    note              text CHECK (length(note) BETWEEN 1 AND 500),
    spent_on          date NOT NULL,
    original_minor    bigint NOT NULL CHECK (original_minor > 0),
    original_currency char(3) NOT NULL CHECK (original_currency ~ '^[A-Z]{3}$'),
    exchange_rate     numeric CHECK (exchange_rate > 0),
    amount_minor      bigint NOT NULL CHECK (amount_minor > 0),
    split_method      text NOT NULL CHECK (split_method IN ('equal', 'exact', 'percentage', 'ratio')),
    state             text NOT NULL CHECK (state IN ('pending', 'accepted', 'disputed', 'withdrawal_pending', 'withdrawn')),
    revision          integer NOT NULL DEFAULT 1,
    pending_since     timestamptz,
    created_at        timestamptz NOT NULL,
    updated_at        timestamptz NOT NULL,
    version           integer NOT NULL DEFAULT 1
);
-- Lists: newest first by date, then by creation (IDs are time-ordered).
CREATE INDEX expenses_group_list_idx ON expenses (group_id, spent_on DESC, id DESC);

CREATE TABLE expense_shares (
    expense_id  uuid NOT NULL REFERENCES expenses (id),
    member_id   uuid NOT NULL REFERENCES members (id),
    input       numeric,
    share_minor bigint NOT NULL CHECK (share_minor >= 0),
    PRIMARY KEY (expense_id, member_id)
);

-- Σ shares = amount (doc 06 invariant 1), checked at commit so an Expense
-- and its Shares can be written in any order inside one transaction.
-- +goose StatementBegin
CREATE FUNCTION check_expense_shares_sum(eid uuid) RETURNS void LANGUAGE plpgsql AS $$
DECLARE
    total bigint;
    shares bigint;
BEGIN
    SELECT amount_minor INTO total FROM expenses WHERE id = eid;
    IF total IS NULL THEN
        RETURN;
    END IF;
    SELECT coalesce(sum(share_minor), 0) INTO shares FROM expense_shares WHERE expense_id = eid;
    IF shares <> total THEN
        RAISE EXCEPTION 'shares of expense % sum to %, not %', eid, shares, total USING ERRCODE = 'check_violation';
    END IF;
END;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION expenses_shares_sum() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    PERFORM check_expense_shares_sum(NEW.id);
    RETURN NULL;
END;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION expense_shares_sum() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        PERFORM check_expense_shares_sum(OLD.expense_id);
    ELSE
        PERFORM check_expense_shares_sum(NEW.expense_id);
    END IF;
    RETURN NULL;
END;
$$;
-- +goose StatementEnd
CREATE CONSTRAINT TRIGGER expenses_shares_sum AFTER INSERT OR UPDATE OF amount_minor ON expenses
    DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION expenses_shares_sum();
CREATE CONSTRAINT TRIGGER expense_shares_sum AFTER INSERT OR UPDATE OR DELETE ON expense_shares
    DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION expense_shares_sum();

-- The Group Currency can't change once any Expense exists (D3).
-- +goose StatementBegin
CREATE FUNCTION groups_currency_locked() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.currency <> OLD.currency AND EXISTS (SELECT 1 FROM expenses WHERE group_id = NEW.id) THEN
        RAISE EXCEPTION 'group % has Expenses; its currency is locked (D3)', NEW.id USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER groups_currency_locked BEFORE UPDATE OF currency ON groups
    FOR EACH ROW EXECUTE FUNCTION groups_currency_locked();

-- The Activity History is append-only (D19): rows are never changed or
-- removed.
CREATE TABLE activity_events (
    id                     bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    group_id               uuid NOT NULL REFERENCES groups (id),
    actor_member_id        uuid NOT NULL REFERENCES members (id),
    on_behalf_of_member_id uuid REFERENCES members (id),
    type                   text NOT NULL,
    subject_type           text NOT NULL,
    subject_id             uuid NOT NULL,
    payload                jsonb NOT NULL DEFAULT '{}',
    occurred_at            timestamptz NOT NULL
);
CREATE INDEX activity_events_group_idx ON activity_events (group_id, id DESC);

-- +goose StatementBegin
CREATE FUNCTION activity_events_append_only() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'activity_events is append-only (D19)' USING ERRCODE = 'insufficient_privilege';
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER activity_events_no_update_delete BEFORE UPDATE OR DELETE ON activity_events
    FOR EACH ROW EXECUTE FUNCTION activity_events_append_only();
CREATE TRIGGER activity_events_no_truncate BEFORE TRUNCATE ON activity_events
    FOR EACH STATEMENT EXECUTE FUNCTION activity_events_append_only();
