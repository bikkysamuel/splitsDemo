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
       e.original_currency, e.amount_minor, e.split_method, e.state, e.version, e.created_at, g.currency
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
SELECT e.id, e.payer_id, e.category, e.note, e.spent_on, e.amount_minor, e.state, g.currency
FROM expenses e JOIN groups g ON g.id = e.group_id
WHERE e.group_id = @group_id
  AND (NOT @has_cursor::boolean OR (e.spent_on, e.id) < (@after_spent_on::date, @after_id::uuid))
ORDER BY e.spent_on DESC, e.id DESC
LIMIT @max_rows;
