# Implementing #22: Balances and Settle-up Suggestions on the Group screen and the Report tab (unattended M1 run)

Follows [03](03-implement-expenses-equal.md). Branch `feat/balances`, built on `feat/expenses-equal` (PR #44).

| UTC | Who | What happened | Decided | Prompt |
|---|---|---|---|---|
| 2026-10-02 06:20:00Z | Claude | Opened PR #44 for #17 and started #22 | — | — |
| 2026-10-02 04:19:06Z | Claude | Contract for `GET /v1/groups/{id}/balances`; the `balances` package loads ledger items from one snapshot and lets `ledger` compute Balances and Suggestions; HTTP-seam tests (settled, Σ = 0 with exact amounts and Suggestion order, only Accepted/WithdrawalPending count, 404); iOS Balances section with words and symbols, Report tab with a remembered Group picker, all test-first | See Q94 | — |
