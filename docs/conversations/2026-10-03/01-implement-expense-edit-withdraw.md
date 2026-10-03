# Implementing #20: edit and withdraw an Expense (unattended M1 run)

Follows [00](00-implement-foreign-currency.md). Branch `feat/expense-edit-withdraw`, built on `feat/foreign-currency` (#19, PR #48).

| UTC | Who | What happened | Decided | Prompt |
|---|---|---|---|---|
| 2026-10-03 10:38:55Z | Claude | Started #20 on `feat/expense-edit-withdraw` from `feat/foreign-currency`. Read the ticket, FR-E6, FR-H1, doc 06 (state machine, `revision`, `activity_events.payload` holds field diffs), Q20, Q38, Q95 and the Settlement withdraw (#23). Seams under test (from the ticket): the HTTP seam on real Postgres; iOS ViewModels and the Data repository with JSON fixtures | — | — |
| 2026-10-03 10:43:00Z | Claude | Found the main session had written `server/internal/httpapi/expense_edit_test.go` over mine (same path, same ground); kept it, added a Closed Group test and a `fieldErrors` helper, and built the server to its diff shape | Keep main's test file; diff payload `{revision, changes: {field: {from, to}}}` | — |
| 2026-10-03 10:55:00Z | Claude | Server: contract (`PATCH /v1/expenses/{id}` with `ExpenseEdit`, `POST …/withdraw`, `revision` on Expense), sqlc `UpdateExpense`/`DeleteShares`/`SetExpenseState`, `expenses.Service.Edit`/`Withdraw`, store transactions with the Activity event; HTTP-seam tests green on Postgres. Commits b6bf4ca, 098a09a | Unchanged edit adds no revision | — |
| 2026-10-03 11:15:00Z | Claude | Red test: the app reads a pre-filled "33.30" back as "33.3", which added a revision; fixed by comparing Split entries by value (`big.Rat`). Domain `ExchangeRate.editableText`, `SplitMethod.editableInput` test-first. Commit 58cea76 | Entries compare by value | — |
| 2026-10-03 11:40:00Z | Claude | iOS test-first: Data `editExpense`/`withdrawExpense` with fixtures; `AddExpenseViewModel(editing:)` pre-fills, saves with the version, shows a conflict in place with Reload; `ExpenseDetailViewModel` Edit/Withdraw for the creator in an open Group; views, String Catalog, Q98 | Reuse Settlement withdraw strings; Reload refills the form | — |
