-- name: LedgerMembers :many
SELECT join_seq FROM members WHERE group_id = @group_id ORDER BY join_seq;

-- name: LedgerExpenses :many
SELECT e.id, e.state, p.join_seq AS payer_join_seq, e.amount_minor
FROM expenses e JOIN members p ON p.id = e.payer_id
WHERE e.group_id = @group_id;

-- name: LedgerShares :many
SELECT s.expense_id, m.join_seq, s.share_minor
FROM expense_shares s
JOIN expenses e ON e.id = s.expense_id
JOIN members m ON m.id = s.member_id
WHERE e.group_id = @group_id;
