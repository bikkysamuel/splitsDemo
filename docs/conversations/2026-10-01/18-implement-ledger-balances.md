# Implementing #9: ledger Balances and Settle-up Suggestions (/implement)

Follows [17](17-implement-ledger-shares.md). Branch `feat/ledger-balances`.

| UTC | Who | What happened | Decided | Prompt |
|---|---|---|---|---|
| 2026-10-01 14:38:00Z | User | Ran `/implement #9` | — | `/implement #9` |
| 2026-10-01 14:41:00Z | Claude | Read #9 (blocker #5 closed), ADR-0006/0009/0010, FR-B1/B2, doc 06 state machine and invariants 2–4, GLOSSARY; took the ticket's seams (table tests + fuzz on `ledger`'s public API) | — | — |
| 2026-10-01 14:46:00Z | Claude | Built `ledger.Balances` and `ledger.SettleUpSuggestions` test-first: table tests for all settled, one debtor, chains, Settlements, Pending/Disputed/Withdrawn ignored, WithdrawalPending counted, ties; inconsistent items return `ErrInconsistentLedger` | Members are identified by `join_seq`, as in Splits; greedy is largest debtor pays largest creditor, ties to the lowest `join_seq`; recorded as Q88 | — |
| 2026-10-01 14:52:00Z | Claude | Added three fuzz targets (Σ Balances = 0 against an independent FR-B1 oracle; suggestions zero every Balance in ≤ non-zero Members − 1 payments; excluded states and Accepted ↔ WithdrawalPending never change Balances) with 24 committed corpus inputs each; `scripts/check.sh server` already fuzzes every target | Probes: counting Disputed items, a lossy Settlement and overpaying a creditor each fail their target within a second | `FUZZTIME=5s scripts/check.sh server` green |
