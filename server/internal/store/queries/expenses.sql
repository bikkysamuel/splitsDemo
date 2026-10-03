-- name: LockGroupShared :one
-- Holds the Group's state steady while an Expense is written.
SELECT state, currency FROM groups WHERE id = @id FOR SHARE;

-- name: InsertExpense :exec
INSERT INTO expenses (id, group_id, created_by, payer_id, category, note, spent_on, original_minor,
                      original_currency, exchange_rate, amount_minor, split_method, state, created_at, updated_at)
VALUES (@id, @group_id, @created_by, @payer_id, @category, @note, @spent_on, @original_minor,
        @original_currency, @exchange_rate, @amount_minor, @split_method, @state, @now, @now);

-- name: InsertShare :exec
INSERT INTO expense_shares (expense_id, member_id, input, share_minor)
VALUES (@expense_id, @member_id, @input, @share_minor);

-- name: InsertActivityEvent :exec
INSERT INTO activity_events (group_id, actor_member_id, on_behalf_of_member_id, type, subject_type, subject_id,
                             payload, occurred_at)
VALUES (@group_id, @actor_member_id, @on_behalf_of_member_id, @type, @subject_type, @subject_id,
        @payload, @occurred_at);

-- name: ExpenseForUser :one
-- The Expense, if the User is an active Member of its Group.
SELECT e.id, e.group_id, e.created_by, e.payer_id, e.category, e.note, e.spent_on, e.original_minor,
       e.original_currency, e.exchange_rate, e.amount_minor, e.split_method, e.state, e.revision, e.version, e.created_at,
       g.currency
FROM expenses e
JOIN groups g ON g.id = e.group_id
JOIN members m ON m.group_id = e.group_id
WHERE e.id = @id AND m.user_id = @user_id AND m.status = 'active';

-- name: ExpenseShares :many
SELECT s.member_id, s.share_minor, s.input
FROM expense_shares s JOIN members m ON m.id = s.member_id
WHERE s.expense_id = @expense_id
ORDER BY m.join_seq;

-- name: ListExpenses :many
-- One page, newest first; after_* is the last row of the previous page.
-- A NULL filter doesn't filter (FR-E8); member matches the payer or a
-- Share.
SELECT e.id, e.payer_id, e.category, e.note, e.spent_on, e.amount_minor, e.state, g.currency
FROM expenses e JOIN groups g ON g.id = e.group_id
WHERE e.group_id = @group_id
  AND (NOT @has_cursor::boolean OR (e.spent_on, e.id) < (@after_spent_on::date, @after_id::uuid))
  AND (sqlc.narg(member)::uuid IS NULL OR e.payer_id = sqlc.narg(member)::uuid
       OR EXISTS (SELECT 1 FROM expense_shares s WHERE s.expense_id = e.id AND s.member_id = sqlc.narg(member)::uuid))
  AND (sqlc.narg(category)::text IS NULL OR e.category = sqlc.narg(category)::text)
  AND (sqlc.narg(state)::text IS NULL OR e.state = sqlc.narg(state)::text)
  AND (sqlc.narg(from_date)::date IS NULL OR e.spent_on >= sqlc.narg(from_date)::date)
  AND (sqlc.narg(to_date)::date IS NULL OR e.spent_on <= sqlc.narg(to_date)::date)
ORDER BY e.spent_on DESC, e.id DESC
LIMIT @max_rows;

-- name: UpdateExpense :one
-- A new revision of the Expense, if it is still at version and in from_state.
UPDATE expenses
SET payer_id = @payer_id, category = @category, note = @note, spent_on = @spent_on, original_minor = @original_minor,
    original_currency = @original_currency, exchange_rate = @exchange_rate, amount_minor = @amount_minor,
    split_method = @split_method, state = @state, revision = revision + 1, updated_at = @now, version = version + 1
WHERE id = @id AND version = @version AND state = @from_state
RETURNING revision, version;

-- name: DeleteShares :exec
-- Before a revision's Shares are written; the old ones are kept in its
-- `expense_edited` event.
DELETE FROM expense_shares WHERE expense_id = @expense_id;

-- name: SetExpenseState :one
UPDATE expenses SET state = @state, updated_at = @now, version = version + 1
WHERE id = @id AND version = @version AND state = @from_state
RETURNING version;
