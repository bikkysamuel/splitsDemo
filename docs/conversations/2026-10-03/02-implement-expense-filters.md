# Implementing #21: Expense filters and cursor pagination (M1 run)

Follows [01](01-implement-expense-edit-withdraw.md). Branch `feat/expense-filters`, built on `feat/expense-edit-withdraw` (#20, PR #49).

| UTC | Who | What happened | Decided | Prompt |
|---|---|---|---|---|
| 2026-10-03 14:32:09Z | Claude | Opened PR #49 for #20 (stacked on #48) and started #21. Read the ticket, FR-E8, NFR-P3, doc 07 (filters `member`, `category`, `from`, `to`, `state`) and the existing cursor list | `member` matches payer or Share; filters AND; inclusive dates | — |
| 2026-10-03 14:45:00Z | Claude | Contract first (five query parameters), regenerated Go and Swift. HTTP-seam tests (`expense_filters_test.go`) for each filter, combined filters, stable pages of two across same-day Expenses, and bad filters: red; then `expenses.Filter` checked in the service (`ErrInvalidFilter` → `400 invalid-request`), one sqlc query with NULL-able filters. Green on real Postgres | Q99 | — |
| 2026-10-03 15:05:00Z | Claude | iOS test-first: Domain `ExpenseFilter`; Data sends the filters with the cursor; `GroupViewModel.apply(_:)`, `expenseAppeared(_:)` (infinite scroll, one load at a time, stale pages dropped); `ExpenseFilterForm` (dates behind switches, last day never before the first) and `ExpenseFilterView`; String Catalog; "Show more" removed | Q99 | — |
| 2026-10-03 14:39:28Z | Claude | `/code-review` against `feat/expense-edit-withdraw`, done in the main session (review subagents kept stalling on the stream watchdog). Spec: all three acceptance criteria met. Fixed (regression test first): the filter was checked before membership, so outside the Group a bad filter answered 400 instead of 404. Kept: the shared "Withdrawn" key from Settlements (as Q98). Docker Desktop had stopped mid-run; restarted it and Postgres | Authorization before filter checks | — |
