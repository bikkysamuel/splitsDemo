# Implementing #17: record an equal-split Expense with a live split preview (unattended M1 run)

Follows [02](02-implement-members.md). Branch `feat/expenses-equal`, built on `feat/members` (PR #43).

| UTC | Who | What happened | Decided | Prompt |
|---|---|---|---|---|
| 2026-10-02 05:30:00Z | Claude | Opened PR #43 for #15 and started #17 | — | — |
| 2026-10-02 04:10:11Z | Claude | Contract for preview, list, create and get; migration for `expenses`, `expense_shares` (deferred Σ check), `activity_events` (append-only) and the currency lock; the `expenses` service with one compute path into `ledger`; read-only POSTs skip the Idempotency-Key; HTTP-seam tests incl. the leftover by join order, preview = save, idempotent retry, 404s and the database guards; iOS `Money` (format and parse), Expenses repository, Add Expense with live preview, Expense list and detail, all test-first | See Q93 | — |
